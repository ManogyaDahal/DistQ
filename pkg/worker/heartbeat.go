package worker

// Package worker/heartbeat owns liveness signaling for workers and monitoring in
// the broker process.
//
// This file is intentionally comments-only scaffolding. Implementations should:
// - Send periodic heartbeats for each worker (HSET distq:workers <id> <ts>).
// - Monitor heartbeats in the broker and requeue expired tasks via XCLAIM.
// - Use context cancellation for clean shutdown.
// - Log with slog including component and worker_id where relevant.
// - Use redisclient key constants for all Redis keys.
//
// Concrete types and functions to be defined here:
// - HeartbeatSender: constructed in cmd/worker and started with context.
// - HeartbeatMonitor: constructed in cmd/broker and started with context.
