package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/config"
	"github.com/ManogyaDahal/DistQ/pkg/queue"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/ManogyaDahal/DistQ/pkg/task"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	client := redisclient.New(cfg)
	defer client.Close()

	ctx := context.Background()

	// ── Step 0: flush all stale distq:* keys so the dashboard starts clean ──
	// Previous runs leave behind stream entries, scheduled ZSET members, etc.
	// Deleting them avoids the "infinite retry flood" from unregistered task types.
	staleKeys := []string{
		redisclient.KeyScheduled,
		redisclient.KeyWorkers,
		redisclient.KeyDLQ,
		redisclient.KeyCron,
		redisclient.KeyEvents,
	}
	for _, priority := range cfg.PriorityLevels {
		staleKeys = append(staleKeys, fmt.Sprintf(redisclient.KeyQueueStream, priority))
	}
	if err := client.Redis.Del(ctx, staleKeys...).Err(); err != nil {
		logger.Warn("failed to flush stale keys (continuing anyway)", "err", err)
	} else {
		logger.Info("flushed stale distq keys", "count", len(staleKeys))
	}

	// ── Step 1: ensure consumer groups exist for every priority stream ──
	for _, priority := range cfg.PriorityLevels {
		if err := queue.EnsureConsumerGroup(ctx, client, priority); err != nil {
			logger.Error("ensure consumer group", "priority", priority, "err", err)
		}
	}
	logger.Info("consumer groups ready")

	logger.Info("seeding simulated telemetry data...")

	// ── Step 2: seed simulated worker heartbeats ──
	// NOTE: only real workers (cmd/worker) send live heartbeats; these are
	// just for the dashboard to show a realistic "Active Workers" table.
	// The heartbeat monitor in the broker will evict them after HeartbeatTimeout (30 s).
	now := time.Now().Unix()
	workers := map[string]any{
		"worker-alpha": strconv.FormatInt(now, 10),   // active — just seen
		"worker-beta":  strconv.FormatInt(now-5, 10), // active — 5 s ago
		// worker-gamma intentionally omitted; a real live worker will appear once started
	}
	if err := client.Redis.HSet(ctx, redisclient.KeyWorkers, workers).Err(); err != nil {
		logger.Error("failed to seed workers", "err", err)
	} else {
		logger.Info("seeded simulated workers", "count", len(workers))
	}

	// ── Step 3: seed tasks using REGISTERED handler types ──
	// Using demo.print and demo.sleep so the live worker can actually execute them
	// and the dashboard shows realistic processing rather than an infinite retry loop.

	// Priority-10 — fast print tasks (complete near-instantly, good for throughput demo)
	for i := 1; i <= 3; i++ {
		payload, _ := json.Marshal(map[string]any{"message": fmt.Sprintf("high-priority email #%d", i)})
		t := &task.Task{
			ID:        fmt.Sprintf("task-p10-%d", i),
			Type:      "demo.print",
			Payload:   json.RawMessage(payload),
			Priority:  10,
			Status:    task.StatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if _, err := queue.Enqueue(ctx, client, t); err != nil {
			logger.Error("enqueue priority-10 task", "err", err)
		}
	}
	logger.Info("seeded 3 demo.print tasks at priority 10")

	// Priority-5 — slow sleep tasks (stay in "ongoing" state for a few seconds,
	// so the dashboard shows non-zero ongoing_tasks and active worker utilisation)
	for i := 1; i <= 3; i++ {
		payload, _ := json.Marshal(map[string]any{
			"message": fmt.Sprintf("image processing job #%d", i),
			"seconds": 5, // each task sleeps 5 s so the dashboard can capture it mid-flight
		})
		t := &task.Task{
			ID:        fmt.Sprintf("task-p5-%d", i),
			Type:      "demo.sleep",
			Payload:   json.RawMessage(payload),
			Priority:  5,
			Status:    task.StatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if _, err := queue.Enqueue(ctx, client, t); err != nil {
			logger.Error("enqueue priority-5 task", "err", err)
		}
	}
	logger.Info("seeded 3 demo.sleep tasks at priority 5")

	// Priority-1 — intentional fail tasks that will exhaust retries and land in DLQ
	// (demonstrates the DLQ panel; MaxRetries=1 so they fail fast and only once)
	for i := 1; i <= 2; i++ {
		payload, _ := json.Marshal(map[string]any{"message": fmt.Sprintf("intentional failure #%d", i)})
		t := &task.Task{
			ID:         fmt.Sprintf("task-fail-%d", i),
			Type:       "demo.fail",
			Payload:    json.RawMessage(payload),
			Priority:   1,
			Status:     task.StatusPending,
			MaxRetries: 1, // fail once, retry once, then DLQ — avoids long retry flood
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := queue.Enqueue(ctx, client, t); err != nil {
			logger.Error("enqueue priority-1 fail task", "err", err)
		}
	}
	logger.Info("seeded 2 demo.fail tasks at priority 1 (will land in DLQ after 1 retry)")

	// ── Step 4: seed two DLQ tasks directly (pre-failed, for immediate dashboard demo) ──
	dlqTasks := []*task.Task{
		{
			ID:         "task-dlq-1",
			Type:       "demo.print",
			Payload:    json.RawMessage(`{"message":"payment settlement"}`),
			Priority:   10,
			Status:     task.StatusDead,
			MaxRetries: 3,
			RetryCount: 3,
			CreatedAt:  time.Now().Add(-10 * time.Minute),
			UpdatedAt:  time.Now().Add(-9 * time.Minute),
			ErrorMsg:   "gateway timeout: credit ledger unreachable",
		},
		{
			ID:         "task-dlq-2",
			Type:       "demo.print",
			Payload:    json.RawMessage(`{"message":"CRM sync"}`),
			Priority:   5,
			Status:     task.StatusDead,
			MaxRetries: 3,
			RetryCount: 3,
			CreatedAt:  time.Now().Add(-5 * time.Minute),
			UpdatedAt:  time.Now().Add(-4 * time.Minute),
			ErrorMsg:   "http 401: OAuth token expired",
		},
	}
	for _, t := range dlqTasks {
		if err := queue.MoveToDLQ(ctx, client, t); err != nil {
			logger.Error("failed to seed DLQ task", "task_id", t.ID, "err", err)
		}
	}
	logger.Info("seeded 2 pre-failed tasks in DLQ")

	// ── Step 5: seed a scheduled (ETA) task ──
	eta := time.Now().Add(30 * time.Second)
	scheduledTask := &task.Task{
		ID:        "task-scheduled-1",
		Type:      "demo.print",
		Payload:   json.RawMessage(`{"message":"scheduled report"}`),
		Priority:  5,
		Status:    task.StatusPending,
		ETA:       &eta,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	payload, _ := json.Marshal(scheduledTask)
	if err := client.Redis.ZAdd(ctx, redisclient.KeyScheduled, redis.Z{
		Score:  float64(eta.Unix()),
		Member: string(payload),
	}).Err(); err != nil {
		logger.Warn("failed to seed scheduled task (non-fatal)", "err", err)
	} else {
		logger.Info("seeded 1 scheduled task (ETA 30 s from now)")
	}

	// ── Step 6: seed a cron job definition ──
	cronJob := map[string]any{
		"expr":          "*/5 * * * *",
		"task_template": json.RawMessage(`{"type":"demo.print","priority":1,"payload":{"message":"periodic cleanup"}}`),
		"last_run_unix": now - 180, // last ran 3 minutes ago — due soon
	}
	cronData, _ := json.Marshal(cronJob)
	if err := client.Redis.HSet(ctx, redisclient.KeyCron, "five-min-cleanup", cronData).Err(); err != nil {
		logger.Error("failed to seed cron job", "err", err)
	} else {
		logger.Info("seeded cron job 'five-min-cleanup' (*/5 * * * *)")
	}

	// ── Step 7: simulate ongoing tasks by claiming them without acknowledging ──
	// We'll claim the demo.sleep tasks seeded in Step 3 as "worker-alpha"
	args := &redis.XReadGroupArgs{
		Group:    "workers",
		Consumer: "worker-alpha",
		Streams:  []string{fmt.Sprintf(redisclient.KeyQueueStream, 5), ">"},
		Count:    2,
		Block:    0,
	}
	if _, err := client.Redis.XReadGroup(ctx, args).Result(); err != nil {
		logger.Error("failed to claim tasks to simulate ongoing", "err", err)
	} else {
		logger.Info("claimed 2 demo.sleep tasks to simulate ongoing tasks")
	}

	logger.Info("✓ all done — open the dashboard at http://localhost:8080")
	logger.Info("  start the worker with: go run ./cmd/worker to process the seeded tasks")
}
