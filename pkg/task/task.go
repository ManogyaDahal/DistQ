package task

import (
	"context"
	"encoding/json"
	"time"
)

// Task is the single source of truth for task metadata stored in Redis and
// passed between broker, worker, and API.
type Task struct {
	ID       string          // UUID, set by the broker on enqueue
	Name     string          // Optional human-readable name for the task
	Type     string          // Registered handler name, e.g. "send_email"
	Payload  json.RawMessage // Opaque bytes — handler unmarshals this
	Priority int             // Higher = dequeued first (default: 5)
	// Source indicates where the task originated from (e.g. "Go SDK", "Dashboard", "Worker").
	Source     string
	Status     TaskStatus // Current status of the task
	MaxRetries int        // Max retry attempts before DLQ (default: 3)
	RetryCount int        // How many times this task has been retried
	ETA        *time.Time // If set, task must not run before this time
	CronExpr   string     // If set, task recurs on this cron expression
	WorkerID   string     // ID of the worker currently running this task
	Queue      string     // Which priority stream this task lives in
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ErrorMsg   string // Last error message from a failed attempt
}

// TaskStatus captures the lifecycle state of a task.
type TaskStatus string

const (
	StatusPending  TaskStatus = "pending"
	StatusClaimed  TaskStatus = "claimed"
	StatusRunning  TaskStatus = "running"
	StatusDone     TaskStatus = "done"
	StatusFailed   TaskStatus = "failed"
	StatusRetrying TaskStatus = "retrying"
	StatusDead     TaskStatus = "dead"
)

// Handler is the function signature used by workers to execute tasks.
type Handler func(ctx context.Context, payload json.RawMessage) error
