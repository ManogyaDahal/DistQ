package api

// This file defines HTTP route registration for the API server.
// It is intentionally a placeholder scaffold without logic.
// Implementations will wire handlers and middleware here.
//
// Ownership:
// - internal/api owns HTTP handlers, routes, and the WebSocket hub.
// - cmd/api should remain thin and call into this package.
//
// TODOs for future implementation:
// - Register REST endpoints: POST /tasks, GET /tasks/:id, DELETE /tasks/:id,
//   GET /dlq, GET /workers, GET /metrics.
// - Register WebSocket endpoint: GET /ws.
// - Attach logging and recovery middleware as needed.
