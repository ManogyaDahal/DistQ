package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/models"
	"github.com/ManogyaDahal/DistQ/pkg/queue"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/ManogyaDahal/DistQ/pkg/task"
	"github.com/redis/go-redis/v9"
)

// RedisQueueAdapter bridges pkg/queue operations with pool.Queue and retry.RetryStore.
type RedisQueueAdapter struct {
	client     *redisclient.Client
	priorities []int
	minIdle    time.Duration

	mu       sync.Mutex
	msgIDs   map[string]string
	msgPrios map[string]int
}

func NewRedisQueueAdapter(
	client *redisclient.Client,
	priorities []int,
	minIdle time.Duration,
) *RedisQueueAdapter {
	return &RedisQueueAdapter{
		client:     client,
		priorities: priorities,
		minIdle:    minIdle,
		msgIDs:     make(map[string]string),
		msgPrios:   make(map[string]int),
	}
}

func (a *RedisQueueAdapter) Dequeue(
	ctx context.Context,
	workerID string,
) (*task.Task, error) {
	if a.client == nil || a.client.Redis == nil {
		return nil, errors.New("adapter: redis client is nil")
	}

	for _, priority := range a.priorities {
		ids, err := queue.PendingIDs(ctx, a.client, priority, a.minIdle, 1)
		if err == nil && len(ids) > 0 {
			tasks, err := queue.Claim(ctx, a.client, priority, a.minIdle, workerID, ids)
			if err == nil && len(tasks) > 0 && tasks[0] != nil {
				t := tasks[0]

				a.mu.Lock()
				a.msgIDs[t.ID] = ids[0]
				a.msgPrios[t.ID] = priority
				a.mu.Unlock()

				return t, nil
			}
		}
	}

	for _, priority := range a.priorities {
		t, msgID, err := queue.Dequeue(ctx, a.client, priority, workerID)
		if err != nil {
			return nil, err
		}

		if t != nil {
			a.mu.Lock()
			a.msgIDs[t.ID] = msgID
			a.msgPrios[t.ID] = priority
			a.mu.Unlock()

			return t, nil
		}
	}

	return nil, ErrNoTask
}

func (a *RedisQueueAdapter) Ack(ctx context.Context, t *task.Task) error {
	if t == nil {
		return errors.New("adapter: cannot ack nil task")
	}

	// Final SDK/API metadata save before Redis stream ACK.
	if err := a.UpdateTaskMetadata(ctx, t); err != nil {
		return err
	}

	a.mu.Lock()
	msgID, ok := a.msgIDs[t.ID]
	priority, okPrio := a.msgPrios[t.ID]

	if ok {
		delete(a.msgIDs, t.ID)
		delete(a.msgPrios, t.ID)
	}
	a.mu.Unlock()

	if !ok || !okPrio {
		return fmt.Errorf(
			"adapter: no stream message ID tracked for task %q",
			t.ID,
		)
	}

	return queue.Ack(ctx, a.client, priority, msgID)
}

// UpdateMeta is required by the dashboard queue flow.
func (a *RedisQueueAdapter) UpdateMeta(
	ctx context.Context,
	t *task.Task,
) error {
	return queue.UpdateTaskMeta(ctx, a.client, t)
}

func (a *RedisQueueAdapter) ScheduleRetry(
	ctx context.Context,
	t *task.Task,
) error {
	if t == nil {
		return errors.New("adapter: cannot schedule retry for nil task")
	}

	if t.ETA == nil {
		return errors.New("adapter: task ETA is nil for retry scheduling")
	}

	if err := a.UpdateMeta(ctx, t); err != nil {
		return err
	}

	if err := a.UpdateTaskMetadata(ctx, t); err != nil {
		return err
	}

	payload, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("adapter: marshal task for retry: %w", err)
	}

	if err := a.client.Redis.ZAdd(ctx, redisclient.KeyScheduled, redis.Z{
		Score:  float64(t.ETA.Unix()),
		Member: string(payload),
	}).Err(); err != nil {
		return fmt.Errorf("adapter: zadd scheduled retry: %w", err)
	}

	return a.ackOriginal(ctx, t.ID)
}

func (a *RedisQueueAdapter) MoveToDLQ(
	ctx context.Context,
	t *task.Task,
) error {
	if t == nil {
		return errors.New("adapter: cannot move nil task to DLQ")
	}

	if err := a.UpdateMeta(ctx, t); err != nil {
		return err
	}

	if err := a.UpdateTaskMetadata(ctx, t); err != nil {
		return err
	}

	if err := queue.MoveToDLQ(ctx, a.client, t); err != nil {
		return fmt.Errorf("adapter: move task to DLQ: %w", err)
	}

	return a.ackOriginal(ctx, t.ID)
}

// UpdateTaskMetadata keeps SDK GET /tasks/{id} data synchronized with worker state.
func (a *RedisQueueAdapter) UpdateTaskMetadata(
	ctx context.Context,
	t *task.Task,
) error {
	if t == nil {
		return errors.New("adapter: cannot update metadata for nil task")
	}

	apiTask := models.Task{
		ID:         t.ID,
		Type:       t.Type,
		Priority:   t.Priority,
		Status:     toModelStatus(t.Status),
		ETA:        t.ETA,
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
		WorkerID:   t.WorkerID,
		RetryCount: t.RetryCount,
		ErrorMsg:   t.ErrorMsg,
	}

	if len(t.Payload) > 0 {
		if err := json.Unmarshal(t.Payload, &apiTask.Payload); err != nil {
			return fmt.Errorf("adapter: decode task payload: %w", err)
		}
	}

	data, err := json.Marshal(apiTask)
	if err != nil {
		return fmt.Errorf("adapter: marshal task metadata: %w", err)
	}

	metaKey := fmt.Sprintf(redisclient.KeyTaskMeta, t.ID)

	if err := a.client.Redis.HSet(
		ctx,
		metaKey,
		"data",
		string(data),
	).Err(); err != nil {
		return fmt.Errorf("adapter: save task metadata: %w", err)
	}

	return nil
}

func (a *RedisQueueAdapter) ackOriginal(
	ctx context.Context,
	taskID string,
) error {
	a.mu.Lock()
	msgID, ok := a.msgIDs[taskID]
	priority, okPrio := a.msgPrios[taskID]

	if ok {
		delete(a.msgIDs, taskID)
		delete(a.msgPrios, taskID)
	}
	a.mu.Unlock()

	if ok && okPrio {
		return queue.Ack(ctx, a.client, priority, msgID)
	}

	return nil
}

func toModelStatus(status task.TaskStatus) models.TaskStatus {
	switch status {
	case task.StatusPending:
		return models.StatusPending
	case task.StatusClaimed, task.StatusRunning:
		return models.StatusRunning
	case task.StatusDone:
		return models.StatusSuccess
	case task.StatusRetrying:
		return models.StatusScheduled
	case task.StatusFailed:
		return models.StatusFailed
	case task.StatusDead:
		return models.StatusDead
	default:
		return models.StatusPending
	}
}
