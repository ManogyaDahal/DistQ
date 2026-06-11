package redisclient

import (
	"github.com/redis/go-redis/v9"
	"github.com/ManogyaDahal/DistQ/pkg/config"
)

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

// returns New redis client
func New(cfg *config.Config) *Client{ 
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB: 0, //no database is used
	})

	return &Client{ Redis: redisClient,	}
}

//for a clean shutdown of redis client 
// might not come in use
func (c *Client) Close() error{ 
	if c == nil || c.Redis == nil { 
		return nil
	}
	return c.Redis.Close()
}
