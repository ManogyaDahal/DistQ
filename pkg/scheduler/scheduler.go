package scheduler

// Package scheduler owns the ETA and cron scheduling loops that run inside the
// broker process.
//
// This file is intentionally comments-only scaffolding. Implementations will be
// added later, and the cmd/broker entrypoint should remain thin.
//
// Responsibilities to implement in this package:
// - ETA loop: move tasks whose ETA has elapsed from the scheduled ZSET into the
//   appropriate priority stream.
// - Cron loop: evaluate cron expressions and enqueue new task instances.
// - Respect context cancellation and use tickers (no time.Sleep polling).
//
// Dependencies to inject (do not construct globally):
// - Redis client wrapper from pkg/redisclient
// - Configuration from pkg/config
// - Structured logger (log/slog)
//
// Reminder: follow AGENTS.md rules for error wrapping, Redis key constants,
// concurrency, and logging fields.
