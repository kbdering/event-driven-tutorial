package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kbdering/waiterservice/internal/config"
	"github.com/kbdering/waiterservice/internal/service"
	"github.com/kbdering/waiterservice/metrics"
	"github.com/kbdering/waiterservice/pkg/logging"
	"github.com/kbdering/waiterservice/pkg/tracing"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	if err := logging.InitLogger(); err != nil {
		panic(err)
	}
	defer logging.Sync()

	// Initialize OpenTelemetry
	tracing.InitOpenTelemetry()

	// Initialize metrics server
	metricsServer := metrics.NewMetricsServer(9090)
	go func() {
		if err := metricsServer.Start(); err != nil {
			logging.GetLogger().Error("Failed to initialise metrics server")
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	waiterService := service.NewWaiterService(cfg)
	if err := waiterService.Setup(ctx); err != nil {
		panic(err)
	}

	go func() {
		<-sigChan
		metricsServer.Stop(ctx)
		cancel()
	}()

	if err := waiterService.Run(ctx); err != nil {
		panic(err)
	}
}
