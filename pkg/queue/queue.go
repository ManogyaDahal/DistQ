package queue

// Package queue owns all enqueue/dequeue operations against Redis Streams.
// This is a placeholder file used to establish package ownership and structure.
//
// Responsibilities to implement here:
// - Enqueue: serialize task.Task to JSON and XADD into the correct priority stream.
// - Dequeue: XREADGROUP COUNT 1 to claim a task for a worker.
// - Ack: XACK on success.
// - Claim: XCLAIM for reassigning timed-out tasks.
// - Ensure consumer group "workers" exists per priority level.
//
// Dependencies (by design):
// - pkg/task for Task definitions.
// - pkg/redisclient for Redis client wrapper and key constants.
//
// All logic here should follow the rules in AGENTS.md (errors wrapped with context,
// no raw Redis keys, and COUNT 1 on XREADGROUP).
