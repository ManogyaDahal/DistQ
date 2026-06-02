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
