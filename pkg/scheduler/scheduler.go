// Package scheduler owns the ETA and cron scheduling loops that run inside the
// broker process.
//
// Responsibilities:
//   - ETA loop: poll distq:scheduled (ZSET) every second and move tasks whose
//     ETA has elapsed into the appropriate priority stream via queue.Enqueue.
//   - Cron loop: poll distq:cron (Hash) every 30 s, parse cron expressions with
//     robfig/cron, and enqueue a new task instance when a job is due.
//
// Both loops block until ctx is cancelled and use tickers — no time.Sleep.
// Dependencies are injected; nothing is constructed globally.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	robcron "github.com/robfig/cron/v3"
	"github.com/redis/go-redis/v9"

	"github.com/ManogyaDahal/DistQ/pkg/config"
	"github.com/ManogyaDahal/DistQ/pkg/queue"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/ManogyaDahal/DistQ/pkg/task"
)

const (
	// etaTickInterval controls how often the ETA loop checks for due tasks.
	etaTickInterval = 1 * time.Second

	// cronTickInterval controls how often the cron loop evaluates job schedules.
	cronTickInterval = 30 * time.Second

	// etaBatchSize is the max number of tasks promoted per ETA tick.
	etaBatchSize = 100
)

// cronEntry is the JSON value stored inside the distq:cron hash for each job.
type cronEntry struct {
	Expr     string          `json:"expr"`              // Cron expression, e.g. "*/5 * * * *"
	Template json.RawMessage `json:"task_template"`     // Partial task JSON merged on enqueue
	LastRun  int64           `json:"last_run_unix"`     // Unix timestamp of the last enqueue
}

// RunETAScheduler promotes tasks from the distq:scheduled sorted set into their
// priority streams when their ETA timestamp has elapsed.
//
// The sorted set is keyed at redisclient.KeyScheduled. Each member is a
// JSON-encoded task.Task; the score is the ETA as a Unix timestamp (float64).
//
// This function blocks until ctx is cancelled.
func RunETAScheduler(ctx context.Context, client *redisclient.Client, cfg *config.Config, log *slog.Logger) {
	log = log.With("component", "eta_scheduler")
	ticker := time.NewTicker(etaTickInterval)
	defer ticker.Stop()

	log.Info("ETA scheduler started", "tick_interval", etaTickInterval)

	for {
		select {
		case <-ctx.Done():
			log.Info("ETA scheduler stopping")
			return
		case <-ticker.C:
			if err := promoteETATasks(ctx, client, log); err != nil {
				log.Error("ETA promotion failed", "err", err)
			}
		}
	}
}

// promoteETATasks performs one ETA promotion cycle.
func promoteETATasks(ctx context.Context, client *redisclient.Client, log *slog.Logger) error {
	now := float64(time.Now().Unix())

	// Fetch all tasks with score (ETA) <= now, up to etaBatchSize.
	members, err := client.Redis.ZRangeByScoreWithScores(ctx, redisclient.KeyScheduled, &redis.ZRangeBy{
		Min:    "0",
		Max:    fmt.Sprintf("%f", now),
		Offset: 0,
		Count:  etaBatchSize,
	}).Result()
	if err != nil {
		return fmt.Errorf("zrangebyscore on %s: %w", redisclient.KeyScheduled, err)
	}

	if len(members) == 0 {
		return nil
	}

	promoted := 0
	for _, z := range members {
		raw, ok := z.Member.(string)
		if !ok {
			log.Warn("unexpected member type in scheduled ZSET", "type", fmt.Sprintf("%T", z.Member))
			continue
		}

		var t task.Task
		if err := json.Unmarshal([]byte(raw), &t); err != nil {
			log.Error("unmarshal scheduled task", "err", err, "raw", raw)
			// Remove corrupt entry so it doesn't block the queue.
			client.Redis.ZRem(ctx, redisclient.KeyScheduled, raw)
			continue
		}

		if _, err := queue.Enqueue(ctx, client, &t); err != nil {
			log.Error("enqueue promoted task", "task_id", t.ID, "err", err)
			continue
		}

		// Remove from ZSET only after successful enqueue.
		if err := client.Redis.ZRem(ctx, redisclient.KeyScheduled, raw).Err(); err != nil {
			log.Error("zrem after enqueue", "task_id", t.ID, "err", err)
			// Not fatal — the task is in the stream; a duplicate will surface on the next tick
			// but the stream consumer will process it correctly via XACK idempotency.
		}

		promoted++
		log.Info("promoted scheduled task", "task_id", t.ID, "priority", t.Priority, "eta_score", z.Score)
	}

	if promoted > 0 {
		log.Info("ETA tick complete", "promoted", promoted)
	}

	return nil
}

// RunCronScheduler evaluates registered cron expressions stored in distq:cron and
// enqueues a new task instance when a job is due.
//
// The distq:cron hash has:
//
//	field  = a unique job name / ID string
//	value  = JSON-encoded cronEntry
//
// On each tick, all entries are fetched with HGETALL, their next-fire time is
// computed using robfig/cron, and a task is enqueued if the job is overdue.
// last_run_unix is updated in the hash after a successful enqueue.
//
// This function blocks until ctx is cancelled.
func RunCronScheduler(ctx context.Context, client *redisclient.Client, cfg *config.Config, log *slog.Logger) {
	log = log.With("component", "cron_scheduler")
	ticker := time.NewTicker(cronTickInterval)
	defer ticker.Stop()

	log.Info("cron scheduler started", "tick_interval", cronTickInterval)

	for {
		select {
		case <-ctx.Done():
			log.Info("cron scheduler stopping")
			return
		case <-ticker.C:
			if err := evaluateCronJobs(ctx, client, cfg, log); err != nil {
				log.Error("cron evaluation failed", "err", err)
			}
		}
	}
}

// evaluateCronJobs performs one cron evaluation cycle.
func evaluateCronJobs(ctx context.Context, client *redisclient.Client, cfg *config.Config, log *slog.Logger) error {
	entries, err := client.Redis.HGetAll(ctx, redisclient.KeyCron).Result()
	if err != nil {
		return fmt.Errorf("hgetall %s: %w", redisclient.KeyCron, err)
	}

	if len(entries) == 0 {
		return nil
	}

	now := time.Now()
	parser := robcron.NewParser(
		robcron.Minute | robcron.Hour | robcron.Dom | robcron.Month | robcron.Dow,
	)

	for jobID, raw := range entries {
		var entry cronEntry
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			log.Error("unmarshal cron entry", "job_id", jobID, "err", err)
			continue
		}

		sched, err := parser.Parse(entry.Expr)
		if err != nil {
			log.Error("parse cron expression", "job_id", jobID, "expr", entry.Expr, "err", err)
			continue
		}

		// Determine whether the job should have fired since the last run.
		lastRun := time.Unix(entry.LastRun, 0)
		nextFire := sched.Next(lastRun)

		if nextFire.After(now) {
			// Not yet due.
			continue
		}

		// Build the task from the stored template, stamping fields required by the broker.
		t, err := buildCronTask(entry, jobID)
		if err != nil {
			log.Error("build cron task", "job_id", jobID, "err", err)
			continue
		}

		if _, err := queue.Enqueue(ctx, client, t); err != nil {
			log.Error("enqueue cron task", "job_id", jobID, "task_id", t.ID, "err", err)
			continue
		}

		// Update last_run so the next tick does not re-fire.
		entry.LastRun = now.Unix()
		updated, err := json.Marshal(entry)
		if err != nil {
			log.Error("marshal updated cron entry", "job_id", jobID, "err", err)
			continue
		}
		if err := client.Redis.HSet(ctx, redisclient.KeyCron, jobID, updated).Err(); err != nil {
			log.Error("hset cron entry after enqueue", "job_id", jobID, "err", err)
		}

		log.Info("cron task enqueued", "job_id", jobID, "task_id", t.ID, "expr", entry.Expr, "next_fire", sched.Next(now))
	}

	return nil
}

// buildCronTask constructs a task.Task from a cron entry template, filling in
// the CronExpr, Status, and timestamps. The caller is responsible for ensuring
// the template contains at least a valid Type and Priority.
func buildCronTask(entry cronEntry, jobID string) (*task.Task, error) {
	var t task.Task
	if len(entry.Template) > 0 {
		if err := json.Unmarshal(entry.Template, &t); err != nil {
			return nil, fmt.Errorf("unmarshal task template for job %q: %w", jobID, err)
		}
	}

	now := time.Now().UTC()
	t.CronExpr = entry.Expr
	t.Status = task.StatusPending
	t.CreatedAt = now
	t.UpdatedAt = now

	// Use job ID as a stable task type if none is set in the template.
	if t.Type == "" {
		t.Type = jobID
	}
	// Apply system default priority if template didn't set one.
	if t.Priority == 0 {
		t.Priority = 5
	}
	// Generate a unique ID for this instance.
	if t.ID == "" {
		t.ID = fmt.Sprintf("cron-%s-%d", jobID, now.UnixNano())
	}

	return &t, nil
}
