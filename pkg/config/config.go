package config

import "time"

// Config holds all runtime configuration loaded from environment variables.
// Loading and validation are implemented elsewhere in this package.
type Config struct {
	RedisAddr         string        // REDIS_ADDR, default "localhost:6379"
	RedisPassword     string        // REDIS_PASSWORD, default ""
	WorkerConcurrency int           // WORKER_CONCURRENCY, default 4
	HeartbeatInterval time.Duration // HEARTBEAT_INTERVAL, default 5s
	HeartbeatTimeout  time.Duration // HEARTBEAT_TIMEOUT, default 30s
	PriorityLevels    []int         // PRIORITY_LEVELS, default [10,5,1]
	APIPort           string        // API_PORT, default "8080"
	MaxRetries        int           // MAX_RETRIES, default 3
	LogLevel          string        // LOG_LEVEL, default "info"
}

// returns the default configuration of the task queue
func Load() (*Config, error) {
	return &Config{
		RedisAddr:         "localhost:6379",
		RedisPassword:     "",
		WorkerConcurrency: 4,
		HeartbeatInterval: 5 * time.Second,
		HeartbeatTimeout:  30 * time.Second,
		PriorityLevels:    []int{10, 5, 1},
		APIPort:           "8080",
		MaxRetries:        3,
		LogLevel:          "info",
	}, nil
}
