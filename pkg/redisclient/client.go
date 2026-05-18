package redisclient

import "github.com/redis/go-redis/v9"

// Client wraps the shared go-redis client instance used across the system.
// All Redis access should flow through this package so key naming stays centralized.
type Client struct {
	Redis *redis.Client
}

// Redis key name constants. Use these everywhere instead of raw strings.
const (
	KeyQueueStream = "distq:queue:%d" // %d = priority level
	KeyScheduled   = "distq:scheduled"
	KeyTaskMeta    = "distq:task:%s" // %s = task ID
	KeyWorkers     = "distq:workers"
	KeyDLQ         = "distq:dlq"
	KeyCron        = "distq:cron"
	KeyEvents      = "distq:events"
)
