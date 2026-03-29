package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/gotrack/config"
	grpcHandler "github.com/gotrack/internal/api/grpc"
	"github.com/gotrack/internal/api/rest"
	"github.com/gotrack/internal/repository"
	"github.com/gotrack/internal/service"
	"github.com/gotrack/internal/validator"
	"github.com/gotrack/pkg/logger"
	"github.com/gotrack/pkg/middleware"
	pb "github.com/gotrack/proto"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	log := logger.New(cfg.LogLevel)
	defer log.Sync() //nolint

	// Database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})
	defer rdb.Close()

	// Layers
	repo := repository.NewEventRepository(pool, rdb)
	v := validator.New()
	svc := service.NewEventService(repo, v, log)

	// REST server
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Logger(log))

	r.GET("/health", rest.NewHandler(svc).HealthCheck)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := r.Group("/api/v1", middleware.AuthMiddleware(repo))
	{
		h := rest.NewHandler(svc)
		api.POST("/events", h.IngestEvents)
		api.GET("/events", h.QueryEvents)
	}

	httpSrv := &http.Server{
		Addr:    ":" + cfg.Server.HTTPPort,
		Handler: r,
	}

	// gRPC server
	grpcSrv := grpc.NewServer()
	pb.RegisterEventServiceServer(grpcSrv, grpcHandler.NewHandler(svc, log))

	lis, err := net.Listen("tcp", ":"+cfg.Server.GRPCPort)
	if err != nil {
		log.Fatal("failed to listen for gRPC", zap.Error(err))
	}

	// Start servers
	go func() {
		log.Info("HTTP server starting", zap.String("port", cfg.Server.HTTPPort))
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	go func() {
		log.Info("gRPC server starting", zap.String("port", cfg.Server.GRPCPort))
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatal("gRPC server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down servers...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// gRPC graceful stop with timeout
	grpcDone := make(chan struct{})
	go func() {
		grpcSrv.GracefulStop()
		close(grpcDone)
	}()
	select {
	case <-grpcDone:
	case <-shutdownCtx.Done():
		grpcSrv.Stop()
	}

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP shutdown error", zap.Error(err))
	}
	log.Info("servers stopped")
}
