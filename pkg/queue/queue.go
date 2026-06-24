// Package queue owns all enqueue/dequeue operations against Redis Streams.
// This is a placeholder file used to establish package ownership and structure.
//
// Responsibilities to implement here:
// - Enqueue: serialize task.Task to JSON and XADD into the correct priority stream.
// - Dequeue: XREADGROUP COUNT 1 to claim a task for a worker.
// - Ack: XACK on success.
// - Claim: XCLAIM for reassigning timed-out tasks.
// - Ensure consumer group "workers" exists per priority level.
//
// Dependencies (by design):
// - pkg/task for Task definitions.
// - pkg/redisclient for Redis client wrapper and key constants.
//
// All logic here should follow the rules in AGENTS.md (errors wrapped with context,
// no raw Redis keys, and COUNT 1 on XREADGROUP).
package queue

import ( 
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings" 
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/ManogyaDahal/DistQ/pkg/task"
	"github.com/redis/go-redis/v9"
)

const (
	consumerGroup = "workers"
	taskField = "task"
)

// EnsureConsumerGroup creates the consumer group for a given priority stream
func EnsureConsumerGroup(ctx context.Context, client *redisclient.Client, priority int) error {
	if client == nil || client.Redis == nil { 
		return errors.New("Redis client id nil")
	}

	stream := fmt.Sprintf(redisclient.KeyQueueStream, priority)
	err := client.Redis.XGroupCreateMkStream(ctx, stream, consumerGroup, "0").Err()

	if err != nil { 
		if strings.Contains(err.Error(), "BUSYGROUP"){
			return nil
		}
		return fmt.Errorf("ensure consumer group for %s: %w", stream, err)
	}

	return nil
}

// Enqueue serializes a task and appends it to the priority stream.
// It also persists task metadata to distq:task:<id> for dashboard and retry lookups.
func Enqueue(ctx context.Context, client *redisclient.Client, t *task.Task) (string, error) {
	if client == nil || client.Redis == nil {
		return "", errors.New("redis client is nil")
	}
	if t == nil {
		return "", errors.New("task is nil")
	}

	stream := fmt.Sprintf(redisclient.KeyQueueStream, t.Priority)

	payload, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("marshal task: %w", err)
	}

	id, err := client.Redis.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: 10000, // approximate trim — prevents unbounded stream growth
		Approx: true,
		Values: map[string]any{taskField: payload},
	}).Result()
	if err != nil {
		return "", fmt.Errorf("xadd to %s: %w", stream, err)
	}

	// Persist task metadata so the dashboard and retry logic can look it up by ID.
	metaKey := fmt.Sprintf(redisclient.KeyTaskMeta, t.ID)
	if err := client.Redis.HSet(ctx, metaKey, "data", payload).Err(); err != nil {
		// Non-fatal: stream entry is already written; log but don't fail the enqueue.
		_ = err // callers should check their own logger; queue is the authoritative store
	}

	return id, nil
}

// MoveToDLQ appends a dead task to the dead-letter queue stream.
// Call this when a task has exhausted MaxRetries.
func MoveToDLQ(ctx context.Context, client *redisclient.Client, t *task.Task) error {
	if client == nil || client.Redis == nil {
		return errors.New("redis client is nil")
	}
	if t == nil {
		return errors.New("task is nil")
	}

	payload, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("marshal task for DLQ: %w", err)
	}

	_, err = client.Redis.XAdd(ctx, &redis.XAddArgs{
		Stream: redisclient.KeyDLQ,
		Values: map[string]any{taskField: payload},
	}).Result()
	if err != nil {
		return fmt.Errorf("xadd to DLQ: %w", err)
	}

	return nil
}

// PendingIDs returns up to count stream message IDs that have been idle for at
// least minIdle in the given priority stream. Used by the heartbeat monitor to
// find in-flight tasks belonging to crashed workers.
func PendingIDs(ctx context.Context, client *redisclient.Client, priority int, minIdle time.Duration, count int64) ([]string, error) {
	if client == nil || client.Redis == nil {
		return nil, errors.New("redis client is nil")
	}

	stream := fmt.Sprintf(redisclient.KeyQueueStream, priority)

	pending, err := client.Redis.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: stream,
		Group:  consumerGroup,
		Idle:   minIdle,
		Start:  "-",
		End:    "+",
		Count:  count,
	}).Result()
	if err != nil {
		if strings.Contains(err.Error(), "NOGROUP") {
			if gcErr := EnsureConsumerGroup(ctx, client, priority); gcErr == nil {
				pending, err = client.Redis.XPendingExt(ctx, &redis.XPendingExtArgs{
					Stream: stream,
					Group:  consumerGroup,
					Idle:   minIdle,
					Start:  "-",
					End:    "+",
					Count:  count,
				}).Result()
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("xpending on %s: %w", stream, err)
	}

	ids := make([]string, 0, len(pending))
	for _, p := range pending {
		ids = append(ids, p.ID)
	}
	return ids, nil
}

// Dequeue claims a task for a woker using XREADGROUP COUNT 1
func Dequeue(ctx context.Context, client *redisclient.Client, priority int, consumer string) (*task.Task, string, error){
	if client == nil || client.Redis == nil { 
		return nil, "", errors.New("Empty redis client")
	}

	stream := fmt.Sprintf(redisclient.KeyQueueStream, priority)

	streams, err := client.Redis.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:  consumerGroup,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count: 1,
		Block: 50 * time.Millisecond, // check quickly and proceed to next priority if empty
	}).Result()

	if err != nil { 
		if strings.Contains(err.Error(), "NOGROUP") {
			if gcErr := EnsureConsumerGroup(ctx, client, priority); gcErr == nil {
				streams, err = client.Redis.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:  consumerGroup,
					Consumer: consumer,
					Streams:  []string{stream, ">"},
					Count: 1,
					Block: 50 * time.Millisecond,
				}).Result()
			}
		}
	}

	if err != nil { 
		if errors.Is(err, redis.Nil){ 
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("xreadgroup from %s: %w", stream, err)
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 { 
		return nil, "", nil
	}

	msg := streams[0].Messages[0]
	t, err := decodeTask(msg.Values)
	if err != nil { 
		return nil, "", fmt.Errorf("decode task %s: %w", stream, err)
	}

	return t, msg.ID, nil
}

// used for decoding the encoded task with various values
func decodeTask(values map[string]any) (*task.Task, error){ 
	raw, ok := values[taskField]	
	if !ok{
		return nil, fmt.Errorf("missing %q field", taskField)
	}

	var data []byte
	switch v := raw.(type){ 
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, fmt.Errorf("unexpected %q type: %T", taskField, raw)
	}

	var t task.Task
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}

	return &t, nil
}

// Ack acknowledges successful processing of a message.
func Ack(ctx context.Context, client *redisclient.Client, priority int, streamID string) error {
	if client == nil || client.Redis == nil {
		return errors.New("redis client is nil")
	}
	if streamID == "" {
		return errors.New("stream ID is empty")
	}

	stream := fmt.Sprintf(redisclient.KeyQueueStream, priority)

	_, err := client.Redis.XAck(ctx, stream, consumerGroup, streamID).Result()
	if err != nil {
		if strings.Contains(err.Error(), "NOGROUP") {
			if gcErr := EnsureConsumerGroup(ctx, client, priority); gcErr == nil {
				_, err = client.Redis.XAck(ctx, stream, consumerGroup, streamID).Result()
			}
		}
	}
	if err != nil {
		return fmt.Errorf("xack on %s: %w", stream, err)
	}

	return nil
}

// Claim reassigns timed-out tasks to a consumer.
func Claim(ctx context.Context, client *redisclient.Client, priority int, minIdle time.Duration, consumer string, streamIDs []string) ([]*task.Task, error) {
	if client == nil || client.Redis == nil {
		return nil, errors.New("redis client is nil")
	}
	if len(streamIDs) == 0 {
		return nil, nil
	}

	stream := fmt.Sprintf(redisclient.KeyQueueStream, priority)

	messages, err := client.Redis.XClaim(ctx, &redis.XClaimArgs{
		Stream:   stream,
		Group:    consumerGroup,
		Consumer: consumer,
		MinIdle:  minIdle,
		Messages: streamIDs,
	}).Result()
	if err != nil {
		if strings.Contains(err.Error(), "NOGROUP") {
			if gcErr := EnsureConsumerGroup(ctx, client, priority); gcErr == nil {
				messages, err = client.Redis.XClaim(ctx, &redis.XClaimArgs{
					Stream:   stream,
					Group:    consumerGroup,
					Consumer: consumer,
					MinIdle:  minIdle,
					Messages: streamIDs,
				}).Result()
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("xclaim on %s: %w", stream, err)
	}

	tasks := make([]*task.Task, 0, len(messages))
	for _, msg := range messages {
		t, err := decodeTask(msg.Values)
		if err != nil {
			return nil, fmt.Errorf("decode claimed task from %s: %w", stream, err)
		}
		tasks = append(tasks, t)
	}

	return tasks, nil
}
