# DistQ — Distributed Task Queue & Job Scheduler

DistQ is a distributed task queue and job scheduler written in Go. It implements
message brokering, worker pool management, scheduling, retries, heartbeats, and a
REST/WebSocket API from first principles using Redis as the shared medium.

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
