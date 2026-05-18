package api

// Package api owns HTTP handler functions for the REST API.
// This file is intentionally a placeholder with documentation only.
// Implementations will be added later per AGENTS.md rules.
//
// Handler ownership:
// - Submit task: POST /tasks
// - Get task status: GET /tasks/:id
// - Cancel task: DELETE /tasks/:id
// - List DLQ: GET /dlq
// - List workers: GET /workers
// - Metrics: GET /metrics
//
// Notes for implementation:
// - Use pkg/task for task definitions and status.
// - Use pkg/queue for enqueue/dequeue operations.
// - Use pkg/redisclient for Redis key constants.
// - Use slog for structured logging with component fields.
// - No business logic in cmd/api; keep wiring thin in main.go.
