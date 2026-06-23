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
	"github.com/ManogyaDahal/DistQ/pkg/scheduler"
	"github.com/ManogyaDahal/DistQ/pkg/worker"
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

	// Start ETA scheduler: promotes tasks from distq:scheduled into priority streams.
	go scheduler.RunETAScheduler(ctx, redis, cfg, log)
	log.Info("ETA scheduler started")

	// Start Cron scheduler: re-enqueues recurring tasks on their cron expressions.
	go scheduler.RunCronScheduler(ctx, redis, cfg, log)
	log.Info("cron scheduler started")

	// Start heartbeat monitor: detects dead workers and reclaims their in-flight tasks.
	monitor := worker.NewHeartbeatMonitor(redis, cfg, log)
	go monitor.Run(ctx)

	log.Info("broker started")
	<-ctx.Done()
	log.Info("broker shutting down")
}
