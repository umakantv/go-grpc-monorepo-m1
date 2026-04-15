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

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/yourorg/monorepo/clients/auth-client"
	userv1 "github.com/yourorg/monorepo/gen/go/public/user"
	"github.com/yourorg/monorepo/pkg/config"
	pkgdb "github.com/yourorg/monorepo/pkg/database"
	"github.com/yourorg/monorepo/pkg/logging"
	"github.com/yourorg/monorepo/pkg/middleware"
	"github.com/yourorg/monorepo/services/user-api/internal/repository"
	"github.com/yourorg/monorepo/services/user-api/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	logger.Info("starting user-api service",
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
		})
		if err != nil {
			logger.Fatal("failed to connect to database", zap.Error(err))
		}
		defer db.Close()
		logger.Info("connected to database")
	}

	// Initialize auth service client
	var authClient *authclient.Client
	authAddr := os.Getenv("AUTH_SERVICE_ADDR")
	if authAddr == "" {
		authAddr = "localhost:9091"
	}
	authClient, err = authclient.New(authAddr)
	if err != nil {
		logger.Warn("failed to connect to auth service", zap.Error(err))
		// Continue anyway - auth might not be needed for all operations
	}
	if authClient != nil {
		defer authClient.Close()
	}

	// Create repository and service
	repo := repository.NewRepository(db)
	userService := service.NewUserService(repo, authClient, logger.Logger)

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.ChainUnaryInterceptors(
			middleware.RecoveryInterceptor(logger),
			middleware.LoggerInterceptor(logger),
		)),
	)

	// Register services
	userv1.RegisterUserServiceServer(grpcServer, userService)
	reflection.Register(grpcServer)

	// Start gRPC server
	grpcListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		logger.Fatal("failed to listen on gRPC port", zap.Error(err))
	}

	go func() {
		logger.Info("gRPC server listening", zap.Int("port", cfg.GRPC.Port))
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	// Start HTTP gateway (if enabled)
	var httpServer *http.Server
	if cfg.HTTP.Enabled {
		ctx := context.Background()
		mux := runtime.NewServeMux()

		// Register gRPC gateway
		opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		err := userv1.RegisterUserServiceHandlerFromEndpoint(ctx, mux, fmt.Sprintf("localhost:%d", cfg.GRPC.Port), opts)
		if err != nil {
			logger.Fatal("failed to register gRPC gateway", zap.Error(err))
		}

		httpServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.HTTP.Port),
			Handler: mux,
		}

		go func() {
			logger.Info("HTTP gateway listening", zap.Int("port", cfg.HTTP.Port))
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatal("HTTP server error", zap.Error(err))
			}
		}()
	}

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown
	grpcServer.GracefulStop()

	if httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("HTTP server shutdown error", zap.Error(err))
		}
	}

	logger.Info("server stopped")
}
