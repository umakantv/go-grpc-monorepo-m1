package main

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	auth "github.com/yourorg/monorepo/gen/go/private/auth"
	pkgdb "github.com/yourorg/monorepo/pkg/database"
	"github.com/yourorg/monorepo/pkg/logging"
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
	logger := logging.New(cfg.Logging.Level, cfg.Logging.Format)
	defer logger.Sync()

	logger.Info("starting auth-service",
		zap.String("service", cfg.Service.Name),
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

	// Create repository and service
	repo := repository.NewRepository(db)
	authService := service.NewAuthService(repo, &cfg.JWT, logger.Logger)

	// Create gRPC server with interceptors
	// Note: No HTTP gateway for internal services
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.ChainUnaryInterceptors(
			middleware.RecoveryInterceptor(logger.Logger),
			middleware.LoggerInterceptor(logger.Logger),
		)),
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

	logger.Info("server stopped")
}
