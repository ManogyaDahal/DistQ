package models

import (
	"errors"
	"time"
)

// TaskStatus represents the task state visible through the API.
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusScheduled TaskStatus = "scheduled"
	StatusRunning   TaskStatus = "running"
	StatusSuccess   TaskStatus = "success"
	StatusFailed    TaskStatus = "failed"
	StatusDead      TaskStatus = "dead"
)

// Task is the API-facing task metadata stored in Redis.
//
// This is what GET /tasks/{id} returns to SDK users and dashboard clients.
type Task struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Payload    map[string]any `json:"payload"`
	Status     TaskStatus     `json:"status"`
	Priority   int            `json:"priority"`
	RetryCount int            `json:"retry_count"`
	ETA        *time.Time     `json:"eta,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	WorkerID   string         `json:"worker_id,omitempty"`
	ErrorMsg   string         `json:"error_msg,omitempty"`
}

// EnqueueRequest is sent by SDK/API users when creating a task.
type EnqueueRequest struct {
	Type     string         `json:"type"`
	Payload  map[string]any `json:"payload"`
	Priority int            `json:"priority"`
	ETA      *time.Time     `json:"eta,omitempty"`
}

// EnqueueResponse is returned after a task is accepted.
type EnqueueResponse struct {
	ID     string     `json:"id"`
	Status TaskStatus `json:"status"`
}

var ErrTaskNotFound = errors.New("task not found")
