package api

// This file is a placeholder for the WebSocket hub implementation.
//
// Ownership: internal/api
//
// Responsibilities to implement here:
// - Subscribe to Redis Pub/Sub channel (distq:events).
// - Broadcast JSON events to all connected WebSocket clients.
// - Track connected clients and handle join/leave lifecycle.
// - Use context cancellation for clean shutdown.
// - Log with slog including component="api".
//
// NOTE: Keep business logic in this package; cmd/api should only wire dependencies.
