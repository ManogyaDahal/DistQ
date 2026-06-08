// Package main wires the broker binary.
//
// Responsibilities (wiring only):
// - Load configuration from pkg/config.
// - Create Redis client from pkg/redisclient.
// - Initialize queue consumer groups for each priority level.
// - Start scheduler loops from pkg/scheduler.
// - Start heartbeat monitor from pkg/worker/heartbeat.
// - Block until shutdown with context cancellation.
//
// No business logic should live here; delegate to pkg/* packages.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ManogyaDahal/DistQ/pkg/config"
	"github.com/ManogyaDahal/DistQ/pkg/queue"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	redis := redisclient.New(cfg)
	defer redis.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log := logger.With("component", "broker")

	// Ensure consumer groups for each priority level
	for _, priority := range cfg.PriorityLevels {
		if err := queue.EnsureConsumerGroup(ctx, redis, priority); err != nil {
			log.Error("ensure consumer group", "priority", priority, "err", err)
			os.Exit(1)
		}
	}
	log.Info("consumer groups initialized", "priorities", cfg.PriorityLevels)

	// Start scheduler after created 
	// Start heartbeat monitor

	log.Info("broker started")
	<-ctx.Done()
	log.Info("broker shutting down")
}
