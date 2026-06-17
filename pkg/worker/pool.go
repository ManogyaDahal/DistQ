package worker

// Package worker hosts the worker pool implementation.
//
// This file is intentionally comments-only scaffolding.
//
// Ownership:
// - pkg/worker/pool is responsible for the worker goroutine pool.
// - It coordinates dequeueing tasks, executing handlers, and acknowledgements.
//
// Implementation notes (to be added later):
// - Use context-aware goroutines and a WaitGroup.
// - Use pkg/queue for Redis Streams operations.
// - Use pkg/worker/registry for handler lookup.
// - Use pkg/worker/retry for retry/DLQ routing.
// - Respect cancellation and ensure graceful shutdown.

import (
	"fmt"
	"sync"
	"context"
	"errors"
	"time"
	"log/slog"

	"github.com/ManogyaDahal/DistQ/pkg/task"
)

var ErrNoTask = errors.New("worker: no task available")

type Queue interface {
	Dequeue(ctx context.Context, workerID string) (*task.Task, error)
	Ack(ctx context.Context, workerID string, taskID string) error
}

type FailureHandler interface {
	HandleFailure(ctx context.Context, t *task.Task, cause error) error
}

type Pool struct {
	workerID string
	concurrency int
	queue Queue
	registry *Registry
	failureHandler FailureHandler
	logger *slog.Logger
	idleBackoff time.Duration
}

type PoolOption func(*Pool)

func WithFailureHandler(handler FailureHandler) PoolOption {
	return func(p *Pool) {
		p.failureHandler = handler
	}
}

func WithLogger(logger *slog.Logger) PoolOption {
	return func(p *Pool) {
		if logger != nil {
			p.logger = logger
		}
	}
}

func WithIdleBackoff(d time.Duration) PoolOption {
	return func(p *Pool) {
		
	}
}