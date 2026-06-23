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

	logger.Info("seeding simulated telemetry data into Redis...")

	// 1. Seed active and stale workers
	now := time.Now().Unix()
	workers := map[string]any{
		"worker-alpha": strconv.FormatInt(now, 10),                     // active
		"worker-beta":  strconv.FormatInt(now-2, 10),                   // active
		"worker-gamma": strconv.FormatInt(now-45, 10),                  // stale (> 30s)
	}
	if err := client.Redis.HSet(ctx, redisclient.KeyWorkers, workers).Err(); err != nil {
		logger.Error("failed to seed workers", "err", err)
	} else {
		logger.Info("seeded simulated workers")
	}

	// 2. Seed some active streams (queue depths)
	// We will enqueue tasks to priority 10 and 5
	for i := 1; i <= 3; i++ {
		t := &task.Task{
			ID:         fmt.Sprintf("task-p10-%d", i),
			Type:       "send_email",
			Payload:    json.RawMessage(`{"to":"user@example.com"}`),
			Priority:   10,
			Status:     task.StatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := queue.Enqueue(ctx, client, t); err != nil {
			logger.Error("failed to enqueue priority 10 task", "err", err)
		}
	}

	for i := 1; i <= 5; i++ {
		t := &task.Task{
			ID:         fmt.Sprintf("task-p5-%d", i),
			Type:       "process_image",
			Payload:    json.RawMessage(`{"img_id":"img-456"}`),
			Priority:   5,
			Status:     task.StatusPending,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if _, err := queue.Enqueue(ctx, client, t); err != nil {
			logger.Error("failed to enqueue priority 5 task", "err", err)
		}
	}
	logger.Info("seeded tasks into priority streams")

	// 3. Seed some tasks directly to the DLQ
	dlqTasks := []*task.Task{
		{
			ID:         "task-dlq-1",
			Type:       "payment_settle",
			Payload:    json.RawMessage(`{"txn_id":"txn-881"}`),
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
			Type:       "sync_crm",
			Payload:    json.RawMessage(`{"customer_id":"cust-22"}`),
			Priority:   5,
			Status:     task.StatusDead,
			MaxRetries: 3,
			RetryCount: 3,
			CreatedAt:  time.Now().Add(-5 * time.Minute),
			UpdatedAt:  time.Now().Add(-4 * time.Minute),
			ErrorMsg:   "http 401: invalid Salesforce OAuth token",
		},
	}

	for _, t := range dlqTasks {
		if err := queue.MoveToDLQ(ctx, client, t); err != nil {
			logger.Error("failed to enqueue DLQ task", "err", err)
		}
	}
	logger.Info("seeded DLQ tasks")

	// 4. Seed a cron job definition
	cronJob := map[string]any{
		"expr":          "*/5 * * * *",
		"task_template": json.RawMessage(`{"type":"db_cleanup","priority":1}`),
		"last_run_unix": now - 180,
	}
	cronData, _ := json.Marshal(cronJob)
	if err := client.Redis.HSet(ctx, redisclient.KeyCron, "hourly-cleanup", cronData).Err(); err != nil {
		logger.Error("failed to seed cron job", "err", err)
	} else {
		logger.Info("seeded cron job")
	}

	// 5. Seed some fake pending tasks in PEL for worker-alpha
	// We claim a task in priority 10 to worker-alpha
	stream10 := fmt.Sprintf(redisclient.KeyQueueStream, 10)
	err = queue.EnsureConsumerGroup(ctx, client, 10)
	if err == nil {
		// Read a message and assign to worker-alpha
		msgs, err := client.Redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    "workers",
			Consumer: "worker-alpha",
			Streams:  []string{stream10, "0"},
			Count:    1,
		}).Result()
		if err == nil && len(msgs) > 0 && len(msgs[0].Messages) > 0 {
			logger.Info("assigned a pending task to worker-alpha via consumer group claim")
		}
	}

	logger.Info("simulated data seeded successfully. Open the dashboard at http://localhost:8080 to see it!")
}
