package worker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
)

// RedisHeartbeatStore implements HeartbeatStore using the distq:workers hash.
//
// Each worker's heartbeat is stored as:
//
//	HSET distq:workers <workerID> <unix_timestamp>|<concurrency>
//
// The concurrency field records how many goroutine slots the worker was
// configured with, so the dashboard can render "X / Y slots" instead of
// a bare active-task count.
//
// The format is backward-compatible: if only a plain timestamp is present
// (legacy entry) concurrency defaults to 0 (shown as "unknown" in the UI).
type RedisHeartbeatStore struct {
	client *redisclient.Client
}

// NewRedisHeartbeatStore returns a HeartbeatStore backed by the distq:workers
// Redis hash.
func NewRedisHeartbeatStore(client *redisclient.Client) *RedisHeartbeatStore {
	return &RedisHeartbeatStore{client: client}
}

// Beat writes or updates the heartbeat timestamp and concurrency for workerID.
func (s *RedisHeartbeatStore) Beat(ctx context.Context, workerID string, at time.Time, concurrency int) error {
	value := fmt.Sprintf("%d|%d", at.Unix(), concurrency)
	if err := s.client.Redis.HSet(ctx, redisclient.KeyWorkers, workerID, value).Err(); err != nil {
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
	for id, val := range raw {
		ts := parseTimestamp(val)
		if ts == 0 {
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

// parseTimestamp extracts the unix timestamp from either the legacy plain
// integer format or the new "timestamp|concurrency" format.
func parseTimestamp(val string) int64 {
	part := val
	if idx := strings.Index(val, "|"); idx != -1 {
		part = val[:idx]
	}
	ts, _ := strconv.ParseInt(part, 10, 64)
	return ts
}
