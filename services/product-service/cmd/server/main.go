package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/config"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/logger"
	"github.com/ilramdhan/simple-store-go-grpc/services/product-service/internal/server"
)

func main() {
	// Parse config path dynamically (Production ready)
	configPath := flag.String("config", "internal/config/default.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		panic("Failed to load config: " + err.Error())
	}

	logger.InitLogger(cfg.Log.Level, cfg.App.Name)

	// Production Grade Context with Cancel on OS Signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := server.NewApp(ctx, cfg)
	if err != nil {
		logger.FromContext(ctx).Error("Failed to initialize app", "error", err)
		os.Exit(1)
	}

	logger.FromContext(ctx).Info("Starting application...")
	
	// Run will block until the app is done or an error occurs
	if err := app.Run(ctx); err != nil {
		logger.FromContext(ctx).Error("App terminated with error", "error", err)
		os.Exit(1)
	}
}
