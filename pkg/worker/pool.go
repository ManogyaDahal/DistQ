package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/task"
)

var ErrNoTask = errors.New("worker: no task available")

const MaxWorkerConcurrency = 4

type Queue interface {
	Dequeue(ctx context.Context, workerID string) (*task.Task, error)
	Ack(ctx context.Context, t *task.Task) error
	UpdateMeta(ctx context.Context, t *task.Task) error
}

type FailureHandler interface {
	HandleFailure(ctx context.Context, t *task.Task, cause error) error
}

// MetadataUpdater is implemented by RedisQueueAdapter.
// It lets the pool save SDK/API task status changes to Redis.
type MetadataUpdater interface {
	UpdateTaskMetadata(ctx context.Context, t *task.Task) error
}

type Pool struct {
	workerID       string
	concurrency    int
	queue          Queue
	registry       *Registry
	failureHandler FailureHandler
	logger         *slog.Logger
	idleBackoff    time.Duration
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
		if d > 0 {
			p.idleBackoff = d
		}
	}
}

func NewPool(
	workerID string,
	concurrency int,
	queue Queue,
	registry *Registry,
	opts ...PoolOption,
) (*Pool, error) {
	if workerID == "" {
		return nil, errors.New("worker: workerID is required")
	}
	if concurrency < 1 {
		return nil, errors.New("worker: concurrency must be at least 1")
	}
	if concurrency > MaxWorkerConcurrency {
		return nil, fmt.Errorf(
			"worker: concurrency cannot exceed %d",
			MaxWorkerConcurrency,
		)
	}
	if queue == nil {
		return nil, errors.New("worker: queue is required")
	}
	if registry == nil {
		return nil, errors.New("worker: registry is required")
	}

	p := &Pool{
		workerID:    workerID,
		concurrency: concurrency,
		queue:       queue,
		registry:    registry,
		logger:      slog.Default(),
		idleBackoff: 500 * time.Millisecond,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p, nil
}

func (p *Pool) Run(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, p.concurrency)

	for i := 0; i < p.concurrency; i++ {
		wg.Add(1)

		go func(slot int) {
			defer wg.Done()

			if err := p.work(runCtx, slot); err != nil {
				errCh <- err
				cancel()
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
	}

	return ctx.Err()
}

func (p *Pool) work(ctx context.Context, slot int) error {
	logger := p.logger.With(
		slog.String("component", "worker_pool"),
		slog.String("worker_id", p.workerID),
		slog.Int("slot", slot),
	)

	slotConsumer := fmt.Sprintf("%s-slot-%d", p.workerID, slot)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		t, err := p.queue.Dequeue(ctx, slotConsumer)

		switch {
		case errors.Is(err, ErrNoTask):
			if waitErr := wait(ctx, p.idleBackoff); waitErr != nil {
				return waitErr
			}
			continue

		case err != nil:
			return fmt.Errorf("worker: dequeue task: %w", err)

		case t == nil:
			if waitErr := wait(ctx, p.idleBackoff); waitErr != nil {
				return waitErr
			}
			continue
		}

		if err := p.execute(ctx, t, logger); err != nil {
			return err
		}
	}
}

func (p *Pool) execute(
	ctx context.Context,
	t *task.Task,
	logger *slog.Logger,
) error {
	handler, ok := p.registry.Get(t.Type)
	if !ok {
		return p.handleFailure(
			ctx,
			t,
			&ErrUnknownTaskType{TaskType: t.Type},
			logger,
		)
	}

	// Task starts running.
	t.Status = task.StatusRunning
	t.WorkerID = p.workerID
	t.UpdatedAt = time.Now().UTC()

	// Keep dashboard queue metadata updated.
	if err := p.queue.UpdateMeta(ctx, t); err != nil {
		return fmt.Errorf(
			"worker: update dashboard running metadata for task %q: %w",
			t.ID,
			err,
		)
	}

	// Keep SDK/API GET /tasks/{id} metadata updated.
	if err := p.updateMetadata(ctx, t); err != nil {
		return fmt.Errorf(
			"worker: update SDK running metadata for task %q: %w",
			t.ID,
			err,
		)
	}

	if err := handler(ctx, t.Payload); err != nil {
		return p.handleFailure(
			ctx,
			t,
			fmt.Errorf("worker: execute task %q: %w", t.ID, err),
			logger,
		)
	}

	// Task completed successfully.
	t.Status = task.StatusDone
	t.ErrorMsg = ""
	t.UpdatedAt = time.Now().UTC()

	if err := p.queue.UpdateMeta(ctx, t); err != nil {
		return fmt.Errorf(
			"worker: update dashboard success metadata for task %q: %w",
			t.ID,
			err,
		)
	}

	if err := p.updateMetadata(ctx, t); err != nil {
		return fmt.Errorf(
			"worker: update SDK success metadata for task %q: %w",
			t.ID,
			err,
		)
	}

	if err := p.queue.Ack(ctx, t); err != nil {
		return fmt.Errorf("worker: ack task %q: %w", t.ID, err)
	}

	logger.Info(
		"task completed",
		slog.String("task_id", t.ID),
		slog.String("task_type", t.Type),
	)

	return nil
}

func (p *Pool) handleFailure(
	ctx context.Context,
	t *task.Task,
	cause error,
	logger *slog.Logger,
) error {
	t.Status = task.StatusFailed
	t.ErrorMsg = cause.Error()
	t.UpdatedAt = time.Now().UTC()

	if err := p.queue.UpdateMeta(ctx, t); err != nil {
		return fmt.Errorf(
			"worker: update dashboard failed metadata for task %q: %w",
			t.ID,
			err,
		)
	}

	if err := p.updateMetadata(ctx, t); err != nil {
		return fmt.Errorf(
			"worker: update SDK failed metadata for task %q: %w",
			t.ID,
			err,
		)
	}

	if p.failureHandler == nil {
		logger.Error(
			"task failed without failure handler",
			slog.String("task_id", t.ID),
			slog.String("task_type", t.Type),
			slog.String("error", cause.Error()),
		)
		return nil
	}

	if err := p.failureHandler.HandleFailure(ctx, t, cause); err != nil {
		return fmt.Errorf("worker: handle failed task %q: %w", t.ID, err)
	}

	logger.Warn(
		"task failed",
		slog.String("task_id", t.ID),
		slog.String("task_type", t.Type),
		slog.String("error", cause.Error()),
	)

	return nil
}

// updateMetadata only runs when the adapter supports SDK/API metadata persistence.
func (p *Pool) updateMetadata(ctx context.Context, t *task.Task) error {
	updater, ok := p.queue.(MetadataUpdater)
	if !ok {
		return nil
	}

	return updater.UpdateTaskMetadata(ctx, t)
}

func wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
