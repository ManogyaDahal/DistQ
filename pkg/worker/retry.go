package worker

// This file documents retry ownership and intended behavior.
// Implementation will be added later in this package.

// Responsibilities for pkg/worker/retry:
// - Provide backoff calculation with jitter.
// - Decide whether a failed task should be retried or moved to DLQ.
// - Persist retry scheduling (ETA) and update task status.
// - Write dead tasks to the DLQ stream and publish events.

// Notes:
// - Follow AGENTS.md rules (error wrapping, no raw Redis keys, slog logging).
// - The worker pool should delegate retry handling to this package.
