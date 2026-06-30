package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ManogyaDahal/DistQ/pkg/queue"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/ManogyaDahal/DistQ/pkg/task"
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

// NewRedisQueueAdapter creates a new RedisQueueAdapter.
func NewRedisQueueAdapter(client *redisclient.Client, priorities []int, minIdle time.Duration) *RedisQueueAdapter {
	return &RedisQueueAdapter{
		client:     client,
		priorities: priorities,
		minIdle:    minIdle,
		msgIDs:     make(map[string]string),
		msgPrios:   make(map[string]int),
	}
}

// Dequeue claims an existing idle/stale task from the priority streams or pulls a new one.
func (a *RedisQueueAdapter) Dequeue(ctx context.Context, workerID string) (*task.Task, error) {
	if a.client == nil || a.client.Redis == nil {
		return nil, errors.New("adapter: redis client is nil")
	}

	// 1. Try to reclaim stale tasks from the streams (highest priority first).
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

	// 2. Pull a new task from the highest priority stream available.
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

// Ack acknowledges a successfully processed task and removes it from tracking.
func (a *RedisQueueAdapter) Ack(ctx context.Context, t *task.Task) error {
	if t == nil {
		return errors.New("adapter: cannot ack nil task")
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
		return fmt.Errorf("adapter: no stream message ID tracked for task %q", t.ID)
	}

	return queue.Ack(ctx, a.client, priority, msgID)
}

// UpdateMeta delegates to queue.UpdateTaskMeta to persist the task status.
func (a *RedisQueueAdapter) UpdateMeta(ctx context.Context, t *task.Task) error {
	return queue.UpdateTaskMeta(ctx, a.client, t)
}

// ScheduleRetry puts a failed task back into the scheduled queue (ZSET) and ACKs the original stream message.
func (a *RedisQueueAdapter) ScheduleRetry(ctx context.Context, t *task.Task) error {
	if t == nil {
		return errors.New("adapter: cannot schedule retry for nil task")
	}
	if t.ETA == nil {
		return errors.New("adapter: task ETA is nil for retry scheduling")
	}

	payload, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("adapter: marshal task for retry: %w", err)
	}

	err = a.client.Redis.ZAdd(ctx, redisclient.KeyScheduled, redis.Z{
		Score:  float64(t.ETA.Unix()),
		Member: payload,
	}).Err()
	if err != nil {
		return fmt.Errorf("adapter: zadd scheduled retry: %w", err)
	}

	// ACK the original message since it has been rescheduled as a new delayed item.
	_ = a.ackOriginal(ctx, t.ID)

	return a.UpdateMeta(ctx, t)
}

// MoveToDLQ pushes a failed task that exceeded max retries into the DLQ and ACKs the original stream message.
func (a *RedisQueueAdapter) MoveToDLQ(ctx context.Context, t *task.Task) error {
	if t == nil {
		return errors.New("adapter: cannot move nil task to DLQ")
	}

	if err := queue.MoveToDLQ(ctx, a.client, t); err != nil {
		return fmt.Errorf("adapter: move task to DLQ: %w", err)
	}

	// ACK the original message since it has been stored in DLQ.
	_ = a.ackOriginal(ctx, t.ID)

	return a.UpdateMeta(ctx, t)
}

// ackOriginal acknowledges and removes the tracking of a message.
func (a *RedisQueueAdapter) ackOriginal(ctx context.Context, taskID string) error {
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
