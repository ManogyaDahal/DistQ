// Package main wires the worker binary.
// This entrypoint should stay thin and only assemble dependencies,
// then delegate all behavior to packages under pkg/.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/ManogyaDahal/DistQ/internal/handlers"
	"github.com/ManogyaDahal/DistQ/pkg/config"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/ManogyaDahal/DistQ/pkg/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	redisClient := redisclient.New(cfg)
	defer redisClient.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log := logger.With("component", "worker")

	// Generate a unique worker ID based on hostname and process ID
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "worker"
	}
	workerID := fmt.Sprintf("%s-%d", hostname, os.Getpid())
	log = log.With("worker_id", workerID)

	log.Info("starting worker process")

	// Build a worker registry and register demo handlers
	registry := worker.NewRegistry()
	if err := handlers.RegisterDemoHandlers(registry, log); err != nil {
		log.Error("failed to register demo handlers", "err", err)
		os.Exit(1)
	}
	log.Info("registered handlers", "types", registry.Types())

	// Start heartbeat sender
	heartbeatStore := worker.NewRedisHeartbeatStore(redisClient)
	sender, err := worker.NewHeartbeatSender(workerID, heartbeatStore,
		worker.WithHeartbeatSenderInterval(cfg.HeartbeatInterval),
		worker.WithHeartbeatSenderLogger(log),
	)
	if err != nil {
		log.Error("failed to create heartbeat sender", "err", err)
		os.Exit(1)
	}

	go func() {
		if err := sender.Run(ctx); err != nil && err != context.Canceled {
			log.Error("heartbeat sender stopped with error", "err", err)
		}
	}()
	log.Info("heartbeat sender started")

	// Initialize queue adapter (which also handles retry/DLQ storage)
	// We pass HeartbeatTimeout as minIdle so stale/abandoned messages in PEL can be reclaimed.
	queueAdapter := worker.NewRedisQueueAdapter(redisClient, cfg.PriorityLevels, cfg.HeartbeatTimeout)

	// Initialize retry handler
	retryHandler, err := worker.NewRetryHandler(queueAdapter,
		worker.WithRetryMaxAttempts(cfg.MaxRetries),
		worker.WithRetryLogger(log),
	)
	if err != nil {
		log.Error("failed to create retry handler", "err", err)
		os.Exit(1)
	}

	// Initialize and run the worker pool
	pool, err := worker.NewPool(workerID, cfg.WorkerConcurrency, queueAdapter, registry,
		worker.WithFailureHandler(retryHandler),
		worker.WithLogger(log),
	)
	if err != nil {
		log.Error("failed to create worker pool", "err", err)
		os.Exit(1)
	}

	log.Info("worker pool running", "concurrency", cfg.WorkerConcurrency)
	if err := pool.Run(ctx); err != nil && err != context.Canceled {
		log.Error("worker pool exited with error", "err", err)
		os.Exit(1)
	}

	log.Info("worker shut down gracefully")
}
