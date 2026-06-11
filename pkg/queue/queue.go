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

// Enqueue searilizes a task and appends it to the priority streams
func Enqueue(ctx context.Context, client *redisclient.Client, t *task.Task) (string, error){ 
	if client == nil || client.Redis == nil { 
		return "", errors.New("Empty redis client")
	}
	
	if t == nil { 
		return "", errors.New("task is nil")
	}
	
	stream := fmt.Sprintf(redisclient.KeyQueueStream, t.Priority)

	payload, err := json.Marshal(t)
	if err != nil {
		return "", err
	}

	id, err := client.Redis.XAdd(ctx, &redis.XAddArgs{
		Stream:  stream,
		Values: map[string]any{taskField: payload},
	}).Result()
	if err != nil { 
		return "", err
	}

	return id, nil
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
		Block: 0, // blocks until message or context cancellation
	}).Result()

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
