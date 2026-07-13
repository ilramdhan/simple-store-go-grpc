package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	productv1 "github.com/ilramdhan/simple-store-go-grpc/gen/go/product/v1"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/config"
	grpchandler "github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/handler/grpc"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/logger"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/repository/postgres"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/usecase"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/interceptor"
)

type App struct {
	cfg        *config.Config
	dbPool     *pgxpool.Pool
	grpcServer *grpc.Server
	httpServer *http.Server
}

func NewApp(ctx context.Context, cfg *config.Config) (*App, error) {
	// 1. Setup DB
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		return nil, err
	}
	poolCfg.MaxConns = cfg.Database.MaxConns
	poolCfg.MinConns = cfg.Database.MinConns

	dbPool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}

	// 2. Setup Layers
	repo := postgres.NewProductRepository(dbPool)
	uc := usecase.NewProductUsecase(repo)
	handler := grpchandler.NewHandler(uc)

	// 3. Setup gRPC Server with Interceptors (Phase 13, 16)
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			interceptor.UnaryRecovery(),
			interceptor.UnaryRequestID(),
			interceptor.UnaryLogging(),
			interceptor.UnaryAuth(cfg.App.JWTSecret), // Phase 14: Authentication
		),
	)
	productv1.RegisterProductServiceServer(grpcServer, handler)

	// Phase 15: Health Checks & Readiness
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	
	// Readiness probe goroutine
	go func() {
		for {
			time.Sleep(5 * time.Second)
			if err := dbPool.Ping(context.Background()); err != nil {
				healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
				logger.FromContext(context.Background()).Warn("Database ping failed, health status set to NOT_SERVING", "error", err)
			} else {
				healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
			}
		}
	}()

	// 4. Setup gRPC-Gateway
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	_ = productv1.RegisterProductServiceHandlerFromEndpoint(
		ctx, mux, fmt.Sprintf("localhost:%d", cfg.Server.GRPCPort), opts,
	)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler: mux,
	}

	return &App{
		cfg:        cfg,
		dbPool:     dbPool,
		grpcServer: grpcServer,
		httpServer: httpServer,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	g, gCtx := errgroup.WithContext(ctx)

	// 1. Run gRPC Server
	g.Go(func() error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.cfg.Server.GRPCPort))
		if err != nil {
			return err
		}
		logger.FromContext(gCtx).Info("gRPC server starting", "port", a.cfg.Server.GRPCPort)
		if err := a.grpcServer.Serve(lis); err != nil {
			return fmt.Errorf("gRPC server failed: %w", err)
		}
		return nil
	})

	// 2. Run HTTP Gateway
	g.Go(func() error {
		logger.FromContext(gCtx).Info("HTTP Gateway starting", "port", a.cfg.Server.HTTPPort)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP gateway failed: %w", err)
		}
		return nil
	})

	// 3. Graceful Shutdown Listener
	g.Go(func() error {
		<-gCtx.Done()
		return a.shutdown()
	})

	return g.Wait()
}

func (a *App) shutdown() error {
	logger.FromContext(context.Background()).Info("Shutting down servers gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
		logger.FromContext(context.Background()).Error("HTTP server shutdown failed", "error", err)
	}

	// Shutdown gRPC server gracefully
	stopped := make(chan struct{})
	go func() {
		a.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-shutdownCtx.Done():
		logger.FromContext(context.Background()).Warn("gRPC GracefulStop timed out, forcing stop")
		a.grpcServer.Stop()
	case <-stopped:
		logger.FromContext(context.Background()).Info("gRPC server stopped gracefully")
	}

	// Close DB pool
	a.dbPool.Close()
	logger.FromContext(context.Background()).Info("Shutdown complete")
	return nil
}
