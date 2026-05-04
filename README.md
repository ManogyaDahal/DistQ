# DistQ — Distributed Task Queue & Job Scheduler

DistQ is a distributed task queue and job scheduler written in Go. It implements
message brokering, worker pool management, scheduling, retries, heartbeats, and a
REST/WebSocket API from first principles using Redis as the shared medium.

This README is a placeholder to document the repository layout and ownership. It
does not include implementation logic.

---

## Repository layout

```
distq/
├── cmd/                        # Binary entry points (thin wiring only)
│   ├── api/                    # REST API + WebSocket server entrypoint
│   ├── broker/                 # Broker entrypoint (scheduler + heartbeat monitor)
│   └── worker/                 # Worker entrypoint (pool + heartbeat sender)
├── pkg/                        # Shared packages (business logic)
│   ├── config/                 # Environment config loading and validation
│   ├── queue/                  # Redis Streams enqueue/dequeue
│   ├── redisclient/            # Redis client wrapper + key constants
│   ├── scheduler/              # ETA + cron scheduling loops
│   ├── task/                   # Task contract types
│   └── worker/                 # Worker pool, registry, retry, heartbeat
├── internal/                   # API-only internal packages
│   ├── api/                    # HTTP handlers, routes, WebSocket hub
│   └── dashboard/              # Static dashboard assets
└── test/                       # Integration and fault tests
```

---

## Package ownership

- `pkg/task`: Task wire contract and status enums.
- `pkg/redisclient`: Redis client wrapper and key naming constants.
- `pkg/queue`: Queue operations (XADD/XREADGROUP/XACK/XCLAIM).
- `pkg/scheduler`: ETA and cron scheduling loops.
- `pkg/worker`: Worker pool, registry, retry policy, and heartbeat logic.
- `pkg/config`: Config loading from environment variables.
- `internal/api`: REST handlers, routing, WebSocket hub.
- `internal/dashboard`: Static HTML/CSS/JS for monitoring UI.

---

## Notes

- The three binaries (`cmd/api`, `cmd/broker`, `cmd/worker`) are thin wiring only.
- Business logic must live under `pkg/` or `internal/` per project rules.
- See `AGENTS.md` for the full system specification and coding rules.