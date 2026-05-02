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

	notifv1 "github.com/yourorg/monorepo/gen/go/private/notification"
	pkgdb "github.com/yourorg/monorepo/pkg/database"
	"github.com/yourorg/monorepo/pkg/logging"
	pkgmigrate "github.com/yourorg/monorepo/pkg/migrate"
	"github.com/yourorg/monorepo/pkg/metrics"
	"github.com/yourorg/monorepo/pkg/middleware"
	"github.com/yourorg/monorepo/services/notification-service/internal/config"
	"github.com/yourorg/monorepo/services/notification-service/internal/fcm"
	"github.com/yourorg/monorepo/services/notification-service/internal/repository"
	"github.com/yourorg/monorepo/services/notification-service/internal/service"
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

	logger.Info("starting notification-service",
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

		// Run database migrations.
		if err := pkgmigrate.Up(db, cfg.Database.Driver, "migrations"); err != nil {
			logger.Fatal("failed to run database migrations", zap.Error(err))
		}
		logger.Info("database migrations applied")
	}

	// Initialize FCM sender
	fcmSender, err := fcm.NewSender(
		context.Background(),
		cfg.Firebase.CredentialsFile,
		cfg.Firebase.ProjectID,
	)
	if err != nil {
		logger.Fatal("failed to initialise FCM sender", zap.Error(err))
	}
	logger.Info("FCM sender initialised")

	// Create repository and service
	repo := repository.NewRepository(db)
	notifService := service.NewNotificationService(repo, fcmSender, logger.Logger)

	// Initialize metrics
	var appMetrics *metrics.Metrics
	var metricsServer *http.Server
	if cfg.Metrics.Enabled {
		appMetrics = metrics.New(metrics.Config{
			Namespace: cfg.Service.Name,
		})
		logger.Info("metrics initialized")

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
	notifv1.RegisterNotificationServiceServer(grpcServer, notifService)
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		logger.Fatal("failed to listen on gRPC port", zap.Error(err))
	}

	logger.Info("gRPC server listening", zap.Int("port", cfg.GRPC.Port))

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