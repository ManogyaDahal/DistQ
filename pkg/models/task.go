package models

import (
	"errors"
	"time"
)

//so  this file was actually built for the language of the entire DistQ system like before the taks
//sent to the broker and worker what the task look like what the properties it should have it is defined

type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusScheduled TaskStatus = "scheduled"
	StatusRunning   TaskStatus = "running"
	StatusSuccess   TaskStatus = "success"
	StatusFailed    TaskStatus = "failed"
	StatusDead      TaskStatus = "dead"
)

type Task struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Payload    map[string]any `json:"payload"`
	Status     TaskStatus     `json:"status"`
	Priority   int            `json:"priority"`
	RetryCount int            `json:"retry_count"`
	ETA        *time.Time     `json:"eta,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// so this EnqueueResquest is basically what a user sends when creating a task.
type EnqueueRequest struct {
	Type     string         `json:"type"`
	Payload  map[string]any `json:"payload"`
	Priority int            `json:"priority"`
	ETA      *time.Time     `json:"eta,omitempty"`
}

// and this was after client send the task and in the response what to give to it .
type EnqueueResponse struct {
	ID     string     `json:"id"`
	Status TaskStatus `json:"status"`
}

var ErrTaskNotFound = errors.New("task not found")
