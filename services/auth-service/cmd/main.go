package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	auth "github.com/yourorg/monorepo/gen/go/private/auth"
	pkgdb "github.com/yourorg/monorepo/pkg/database"
	"github.com/yourorg/monorepo/pkg/logging"
	"github.com/yourorg/monorepo/pkg/metrics"
	"github.com/yourorg/monorepo/pkg/middleware"
	"github.com/yourorg/monorepo/services/auth-service/internal/config"
	"github.com/yourorg/monorepo/services/auth-service/internal/repository"
	"github.com/yourorg/monorepo/services/auth-service/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load configuration
	cfg, err := config.Load(".", "config")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := logging.New(cfg.Logging.Level, cfg.Logging.Format, cfg.Service.Name)
	defer logger.Sync()

	logger.Info("starting auth-service",
		zap.String("env", cfg.Service.Env),
	)

	// Initialize database connection (if enabled)
	var db *sql.DB
	if cfg.Database.Enabled {
		db, err = pkgdb.New(pkgdb.Config{
			Driver:   cfg.Database.Driver,
			Host:     cfg.Database.Host,
			Port:     cfg.Database.Port,
			Name:     cfg.Database.Name,
			User:     cfg.Database.User,
			Password: cfg.Database.Password,
			SSLMode:  cfg.Database.SSLMode,
			Path:     cfg.Database.Path,
		})
		if err != nil {
			logger.Fatal("failed to connect to database", zap.Error(err))
		}
		defer db.Close()
		logger.Info("connected to database")
	}

	// Create repository and service
	repo := repository.NewRepository(db)
	authService := service.NewAuthService(repo, &cfg.JWT, logger.Logger)

	// Initialize metrics
	var appMetrics *metrics.Metrics
	var metricsServer *http.Server
	if cfg.Metrics.Enabled {
		appMetrics = metrics.New(metrics.Config{
			Namespace: cfg.Service.Name,
		})
		logger.Info("metrics initialized")

		// Start metrics HTTP server
		metricsServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.Metrics.Port),
			Handler: appMetrics.Handler(),
		}
		go func() {
			logger.Info("metrics server listening", zap.Int("port", cfg.Metrics.Port))
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatal("metrics server error", zap.Error(err))
			}
		}()
	}

	// Create gRPC server with interceptors
	// Note: No HTTP gateway for internal services
	var grpcInterceptors []grpc.UnaryServerInterceptor
	if appMetrics != nil {
		grpcInterceptors = append(grpcInterceptors, appMetrics.GRPCInterceptor())
	}
	grpcInterceptors = append(grpcInterceptors,
		middleware.RecoveryInterceptor(logger),
		middleware.LoggerInterceptor(logger),
	)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.ChainUnaryInterceptors(grpcInterceptors...)),
	)

	// Register services
	auth.RegisterAuthServiceServer(grpcServer, authService)
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		logger.Fatal("failed to listen on gRPC port", zap.Error(err))
	}

	logger.Info("gRPC server listening", zap.Int("port", cfg.GRPC.Port))

	// Handle graceful shutdown in a goroutine
	go func() {
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown
	grpcServer.GracefulStop()

	if metricsServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("metrics server shutdown error", zap.Error(err))
		}
	}

	logger.Info("server stopped")
}
