package models

import (
	"errors"
	"time"
)

type TaskStatus string

const (
	StatusPending TaskStatus = "pending"
	StatusRunning TaskStatus = "running"
	StatusSuccess TaskStatus = "success"
	StatusFailed  TaskStatus = "failed"
	StatusDead    TaskStatus = "dead"
)

type Task struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Payload    map[string]any `json:"payload"`
	Status     TaskStatus     `json:"status"`
	Priority   int            `json:"priority"`
	RetryCount int            `json:"retry_count"`
	CreatedAt  time.Time      `json:"created_at"`
}

type EnqueueRequest struct {
	Type     string         `json:"type"`
	Payload  map[string]any `json:"payload"`
	Priority int            `json:"priority"`
}

type EnqueueResponse struct {
	ID     string     `json:"id"`
	Status TaskStatus `json:"status"`
}

var ErrTaskNotFound = errors.New("task not found")
