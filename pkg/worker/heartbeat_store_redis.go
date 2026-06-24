package worker

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
)

// RedisHeartbeatStore implements HeartbeatStore using the distq:workers hash.
//
// Each worker's heartbeat is stored as:
//
//	HSET distq:workers <workerID> <unix_timestamp>
//
// The dashboard (ws_hub.go) reads this same hash with HGetAll and parses
// each value as a unix timestamp with strconv.ParseInt, so this format
// MUST remain consistent.
type RedisHeartbeatStore struct {
	client *redisclient.Client
}

// NewRedisHeartbeatStore returns a HeartbeatStore backed by the distq:workers
// Redis hash.
func NewRedisHeartbeatStore(client *redisclient.Client) *RedisHeartbeatStore {
	return &RedisHeartbeatStore{client: client}
}

// Beat writes or updates the heartbeat timestamp for workerID.
func (s *RedisHeartbeatStore) Beat(ctx context.Context, workerID string, at time.Time) error {
	ts := strconv.FormatInt(at.Unix(), 10)
	if err := s.client.Redis.HSet(ctx, redisclient.KeyWorkers, workerID, ts).Err(); err != nil {
		return fmt.Errorf("hset worker heartbeat: %w", err)
	}
	return nil
}

// List returns every registered worker and the time of their last heartbeat.
func (s *RedisHeartbeatStore) List(ctx context.Context) (map[string]time.Time, error) {
	raw, err := s.client.Redis.HGetAll(ctx, redisclient.KeyWorkers).Result()
	if err != nil {
		return nil, fmt.Errorf("hgetall %s: %w", redisclient.KeyWorkers, err)
	}

	result := make(map[string]time.Time, len(raw))
	for id, tsStr := range raw {
		ts, err := strconv.ParseInt(tsStr, 10, 64)
		if err != nil {
			continue // skip corrupt entries
		}
		result[id] = time.Unix(ts, 0).UTC()
	}
	return result, nil
}

// Remove deletes a worker from the heartbeat hash.
func (s *RedisHeartbeatStore) Remove(ctx context.Context, workerID string) error {
	if err := s.client.Redis.HDel(ctx, redisclient.KeyWorkers, workerID).Err(); err != nil {
		return fmt.Errorf("hdel worker heartbeat: %w", err)
	}
	return nil
}
