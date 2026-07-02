package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/task"
)

const (
	DefaultMaxRetries = 3
	DefaultBaseDelay  = time.Second
	DefaultMaxDelay   = 30 * time.Second
)

type RetryStore interface {
	ScheduleRetry(ctx context.Context, t *task.Task) error
	MoveToDLQ(ctx context.Context, t *task.Task) error
}

type RetryHandler struct {
	store      RetryStore
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	logger     *slog.Logger
}

type RetryOption func(*RetryHandler)

func WithRetryMaxAttempts(maxRetries int) RetryOption {
	return func(h *RetryHandler) {
		if maxRetries > 0 {
			h.maxRetries = maxRetries
		}
	}
}

func WithRetryBaseDelay(baseDelay time.Duration) RetryOption {
	return func(h *RetryHandler) {
		if baseDelay > 0 {
			h.baseDelay = baseDelay
		}
	}
}

func WithRetryMaxDelay(maxDelay time.Duration) RetryOption {
	return func(h *RetryHandler) {
		if maxDelay > 0 {
			h.maxDelay = maxDelay
		}
	}
}

func WithRetryLogger(logger *slog.Logger) RetryOption {
	return func(h *RetryHandler) {
		if logger != nil {
			h.logger = logger
		}
	}
}

func NewRetryHandler(store RetryStore, opts ...RetryOption) (*RetryHandler, error) {
	if store == nil {
		return nil, errors.New("worker: retry store is required")
	}

	h := &RetryHandler{
		store:      store,
		maxRetries: DefaultMaxRetries,
		baseDelay:  DefaultBaseDelay,
		maxDelay:   DefaultMaxDelay,
		logger:     slog.Default(),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h, nil
}

func (h *RetryHandler) HandleFailure(ctx context.Context, t *task.Task, cause error) error {
	if t == nil {
		return errors.New("worker: failed task is nil")
	}

	if cause == nil {
		cause = errors.New("unknown task failure")
	}

	now := time.Now().UTC()
	t.ErrorMsg = cause.Error()
	t.UpdatedAt = now

	maxRetries := t.MaxRetries
	if maxRetries <= 0 {
		maxRetries = h.maxRetries
	}

	// No retries left: permanently move task to dead-letter queue.
	if t.RetryCount >= maxRetries {
		t.Status = task.StatusDead
		t.ETA = nil
		t.UpdatedAt = time.Now().UTC()

		if err := h.store.MoveToDLQ(ctx, t); err != nil {
			return fmt.Errorf("worker: move task %q to dlq: %w", t.ID, err)
		}

		h.logger.Warn(
			"task moved to dlq",
			slog.String("task_id", t.ID),
			slog.String("task_type", t.Type),
			slog.Int("retry_count", t.RetryCount),
			slog.String("error", cause.Error()),
		)

		return nil
	}

	// Retries remain: schedule task for later.
	t.RetryCount++
	t.Status = task.StatusRetrying

	retryAt := now.Add(h.backoff(t.RetryCount))
	t.ETA = &retryAt
	t.UpdatedAt = time.Now().UTC()

	if err := h.store.ScheduleRetry(ctx, t); err != nil {
		return fmt.Errorf("worker: schedule retry for task %q: %w", t.ID, err)
	}

	h.logger.Info(
		"task scheduled for retry",
		slog.String("task_id", t.ID),
		slog.String("task_type", t.Type),
		slog.Int("retry_count", t.RetryCount),
		slog.Time("retry_at", retryAt),
		slog.String("error", cause.Error()),
	)

	return nil
}

func (h *RetryHandler) backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	delay := h.baseDelay

	for i := 1; i < attempt; i++ {
		delay *= 2

		if delay >= h.maxDelay {
			delay = h.maxDelay
			break
		}
	}

	jitterLimit := delay / 2
	if jitterLimit <= 0 {
		return delay
	}

	return delay + time.Duration(rand.Int63n(int64(jitterLimit)))
}
