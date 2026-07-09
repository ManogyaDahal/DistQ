package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/ManogyaDahal/DistQ/pkg/config"
	"github.com/ManogyaDahal/DistQ/pkg/queue"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/ManogyaDahal/DistQ/pkg/task"
)

type Handlers struct {
	client *redisclient.Client
	hub    *Hub
	cfg    *config.Config
	logger *slog.Logger
}

func NewHandlers(client *redisclient.Client, hub *Hub, cfg *config.Config, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{
		client: client,
		hub:    hub,
		cfg:    cfg,
		logger: logger.With("component", "api_handlers"),
	}
}

func (h *Handlers) GetMetrics(w http.ResponseWriter, r *http.Request) {
	stats, err := h.hub.collectStats(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to collect stats", err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"metrics":      stats.Metrics,
		"queue_depths": stats.QueueDepths,
		"timestamp":    stats.Timestamp,
	})
}

func (h *Handlers) GetWorkers(w http.ResponseWriter, r *http.Request) {
	stats, err := h.hub.collectStats(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to collect stats", err)
		return
	}

	h.writeJSON(w, http.StatusOK, stats.Workers)
}

func (h *Handlers) GetDLQ(w http.ResponseWriter, r *http.Request) {
	stats, err := h.hub.collectStats(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to collect stats", err)
		return
	}

	h.writeJSON(w, http.StatusOK, stats.DLQTasks)
}

func (h *Handlers) RetryDLQ(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	ctx := r.Context()
	messages, err := h.client.Redis.XRange(ctx, redisclient.KeyDLQ, "-", "+").Result()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to read DLQ stream", err)
		return
	}

	type ReprocessResult struct {
		ID      string `json:"id"`
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	results := []ReprocessResult{}
	processedCount := 0

	for _, msg := range messages {
		t, err := decodeTask(msg.Values)
		if err != nil {
			h.logger.Error("failed to decode DLQ task", "msg_id", msg.ID, "err", err)
			if taskID == "" || (t != nil && t.ID == taskID) {
				results = append(results, ReprocessResult{ID: msg.ID, Success: false, Error: "decode failed"})
			}
			continue
		}

		if taskID != "" && t.ID != taskID {
			continue
		}

		processedCount++
		t.RetryCount = 0
		t.Status = task.StatusPending
		t.UpdatedAt = time.Now().UTC()
		t.ErrorMsg = ""

		_, err = queue.Enqueue(ctx, h.client, t)
		if err != nil {
			h.logger.Error("failed to re-enqueue task", "task_id", t.ID, "err", err)
			results = append(results, ReprocessResult{ID: t.ID, Success: false, Error: err.Error()})
			continue
		}

		if err := h.client.Redis.XDel(ctx, redisclient.KeyDLQ, msg.ID).Err(); err != nil {
			h.logger.Error("failed to delete message from DLQ stream", "msg_id", msg.ID, "err", err)
			results = append(results, ReprocessResult{ID: t.ID, Success: true, Error: "enqueue success, delete failed: " + err.Error()})
		} else {
			results = append(results, ReprocessResult{ID: t.ID, Success: true})
		}
	}

	if taskID != "" && processedCount == 0 {
		h.writeError(w, http.StatusNotFound, fmt.Sprintf("task %s not found in DLQ", taskID), nil)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{
		"processed_count": processedCount,
		"results":         results,
	})
}

func (h *Handlers) GetCompleted(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var cursor uint64
	var keys []string

	for {
		var err error
		var scanKeys []string
		scanKeys, cursor, err = h.client.Redis.Scan(ctx, cursor, "distq:task:*", 100).Result()
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to scan task keys", err)
			return
		}
		keys = append(keys, scanKeys...)
		if cursor == 0 {
			break
		}
	}

	completedTasks := []task.Task{}
	if len(keys) > 0 {
		pipe := h.client.Redis.Pipeline()
		var cmds []*redis.StringCmd
		for _, key := range keys {
			cmds = append(cmds, pipe.HGet(ctx, key, "data"))
		}
		_, _ = pipe.Exec(ctx)

		for _, cmd := range cmds {
			data, err := cmd.Result()
			if err != nil {
				continue
			}
			var t task.Task
			if err := json.Unmarshal([]byte(data), &t); err == nil {
				if t.Status == task.StatusDone {
					completedTasks = append(completedTasks, t)
				}
			}
		}
	}

	h.writeJSON(w, http.StatusOK, completedTasks)
}

func (h *Handlers) GetEnqueued(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var cursor uint64
	var keys []string

	for {
		var err error
		var scanKeys []string
		scanKeys, cursor, err = h.client.Redis.Scan(ctx, cursor, "distq:task:*", 100).Result()
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to scan task keys", err)
			return
		}
		keys = append(keys, scanKeys...)
		if cursor == 0 {
			break
		}
	}

	enqueuedTasks := []task.Task{}
	if len(keys) > 0 {
		pipe := h.client.Redis.Pipeline()
		var cmds []*redis.StringCmd
		for _, key := range keys {
			cmds = append(cmds, pipe.HGet(ctx, key, "data"))
		}
		_, _ = pipe.Exec(ctx)

		for _, cmd := range cmds {
			data, err := cmd.Result()
			if err != nil {
				continue
			}
			var t task.Task
			if err := json.Unmarshal([]byte(data), &t); err == nil {
				// Actively enqueued tasks are pending and have no future ETA
				if t.Status == task.StatusPending && t.ETA == nil {
					enqueuedTasks = append(enqueuedTasks, t)
				}
			}
		}
	}

	h.writeJSON(w, http.StatusOK, enqueuedTasks)
}

func (h *Handlers) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		h.writeError(w, http.StatusBadRequest, "task id is required", nil)
		return
	}

	metaKey := fmt.Sprintf(redisclient.KeyTaskMeta, taskID)
	data, err := h.client.Redis.HGet(r.Context(), metaKey, "data").Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			h.writeError(w, http.StatusNotFound, "task not found", nil)
			return
		}
		h.writeError(w, http.StatusInternalServerError, "failed to get task", err)
		return
	}

	var t task.Task
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to parse task metadata", err)
		return
	}

	h.writeJSON(w, http.StatusOK, t)
}

func (h *Handlers) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode json response", "err", err)
	}
}

func (h *Handlers) writeError(w http.ResponseWriter, status int, msg string, err error) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	h.writeJSON(w, status, map[string]any{
		"error":   msg,
		"details": errStr,
	})
}

// SubmitTaskRequest is the JSON body expected by the POST /api/task endpoint.
type SubmitTaskRequest struct {
	Name       string          `json:"name,omitempty"`
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Priority   int             `json:"priority"`
	MaxRetries *int            `json:"max_retries,omitempty"`
	ETA        *time.Time      `json:"eta,omitempty"`
	CronExpr   string          `json:"cron_expr,omitempty"`
}

// SubmitTask handles POST /api/task — the primary endpoint for external
// applications to submit tasks into the queue.
func (h *Handlers) SubmitTask(w http.ResponseWriter, r *http.Request) {
	var req SubmitTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON body", err)
		return
	}

	if strings.TrimSpace(req.Type) == "" {
		h.writeError(w, http.StatusBadRequest, "task type is required", nil)
		return
	}

	// Default priority.
	if req.Priority <= 0 {
		req.Priority = 5
	}

	// Validate priority against configured levels.
	validPriority := false
	for _, p := range h.cfg.PriorityLevels {
		if req.Priority == p {
			validPriority = true
			break
		}
	}
	if !validPriority {
		h.writeError(w, http.StatusBadRequest,
			fmt.Sprintf("invalid priority %d; must be one of %v", req.Priority, h.cfg.PriorityLevels), nil)
		return
	}

	// Determine max retries: use request value if provided, otherwise config default.
	maxRetries := h.cfg.MaxRetries
	if req.MaxRetries != nil {
		maxRetries = *req.MaxRetries
	}

	taskID := generateID()
	now := time.Now().UTC()
	ctx := r.Context()

	// ── Case 1: Cron task ──────────────────────────────────────────────────
	if req.CronExpr != "" {
		template, err := json.Marshal(map[string]any{
			"name":        req.Name,
			"type":        req.Type,
			"priority":    req.Priority,
			"payload":     req.Payload,
			"max_retries": maxRetries,
		})
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to marshal cron template", err)
			return
		}

		cronEntry := map[string]any{
			"expr":          req.CronExpr,
			"task_template": json.RawMessage(template),
			"last_run_unix": now.Unix(),
		}
		cronData, err := json.Marshal(cronEntry)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to marshal cron entry", err)
			return
		}

		jobID := fmt.Sprintf("job-%s", taskID)
		if err := h.client.Redis.HSet(ctx, redisclient.KeyCron, jobID, cronData).Err(); err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to store cron job", err)
			return
		}

		h.logger.Info("cron job registered",
			slog.String("job_id", jobID),
			slog.String("task_type", req.Type),
			slog.String("cron_expr", req.CronExpr),
		)

		h.writeJSON(w, http.StatusCreated, map[string]any{
			"id":        jobID,
			"name":      req.Name,
			"kind":      "cron",
			"cron_expr": req.CronExpr,
			"status":    "registered",
		})
		return
	}

	// Build the task.
	t := &task.Task{
		ID:         taskID,
		Name:       req.Name,
		Type:       req.Type,
		Payload:    req.Payload,
		Priority:   req.Priority,
		Status:     task.StatusPending,
		MaxRetries: maxRetries,
		ETA:        req.ETA,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// ── Case 2: Scheduled (ETA) task ───────────────────────────────────────
	if req.ETA != nil {
		payload, err := json.Marshal(t)
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to marshal task", err)
			return
		}

		if err := h.client.Redis.ZAdd(ctx, redisclient.KeyScheduled, redis.Z{
			Score:  float64(req.ETA.Unix()),
			Member: string(payload),
		}).Err(); err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to schedule task", err)
			return
		}

		h.logger.Info("task scheduled",
			slog.String("task_id", taskID),
			slog.String("task_type", req.Type),
			slog.Time("eta", *req.ETA),
		)

		h.writeJSON(w, http.StatusCreated, map[string]any{
			"id":       taskID,
			"name":     req.Name,
			"kind":     "scheduled",
			"status":   string(t.Status),
			"eta":      req.ETA,
			"priority": req.Priority,
		})
		return
	}

	// ── Case 3: Immediate task ──────────────────────────────────────────────
	streamID, err := queue.Enqueue(ctx, h.client, t)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to enqueue task", err)
		return
	}

	h.logger.Info("task submitted",
		slog.String("task_id", taskID),
		slog.String("task_type", req.Type),
		slog.Int("priority", req.Priority),
		slog.String("stream_id", streamID),
	)

	h.writeJSON(w, http.StatusCreated, map[string]any{
		"id":        taskID,
		"name":      req.Name,
		"kind":      "immediate",
		"status":    string(t.Status),
		"priority":  req.Priority,
		"stream_id": streamID,
		"queue":     fmt.Sprintf(redisclient.KeyQueueStream, req.Priority),
	})
}

// generateID returns a new random UUID v4 string without external dependencies.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	hstr := hex.EncodeToString(b)
	return hstr[0:8] + "-" + hstr[8:12] + "-" + hstr[12:16] + "-" + hstr[16:20] + "-" + hstr[20:32]
}

func (h *Handlers) GetScheduled(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	results, err := h.client.Redis.ZRangeWithScores(ctx, redisclient.KeyScheduled, 0, -1).Result()
	if err != nil && err != redis.Nil {
		h.writeError(w, http.StatusInternalServerError, "failed to fetch scheduled tasks", err)
		return
	}

	tasks := []map[string]any{}
	for _, z := range results {
		var t task.Task
		if err := json.Unmarshal([]byte(z.Member.(string)), &t); err == nil {
			tasks = append(tasks, map[string]any{
				"task": t,
				"eta":  z.Score,
			})
		}
	}
	h.writeJSON(w, http.StatusOK, tasks)
}

func (h *Handlers) GetCron(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cronJobs, err := h.client.Redis.HGetAll(ctx, redisclient.KeyCron).Result()
	if err != nil && err != redis.Nil {
		h.writeError(w, http.StatusInternalServerError, "failed to fetch cron jobs", err)
		return
	}

	jobs := []map[string]any{}
	for id, data := range cronJobs {
		var job map[string]any
		if err := json.Unmarshal([]byte(data), &job); err == nil {
			job["id"] = id
			jobs = append(jobs, job)
		}
	}
	h.writeJSON(w, http.StatusOK, jobs)
}

func (h *Handlers) GetOngoing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ongoingTasks := []map[string]any{}

	for _, p := range h.cfg.PriorityLevels {
		stream := fmt.Sprintf(redisclient.KeyQueueStream, p)

		pendingInfo, err := h.client.Redis.XPending(ctx, stream, "workers").Result()
		if err != nil || pendingInfo.Count == 0 {
			continue
		}

		for consumer := range pendingInfo.Consumers {
			pendingMsgs, err := h.client.Redis.XPendingExt(ctx, &redis.XPendingExtArgs{
				Stream:   stream,
				Group:    "workers",
				Start:    "-",
				End:      "+",
				Count:    1000,
				Consumer: consumer,
			}).Result()
			if err != nil {
				continue
			}

			pipe := h.client.Redis.Pipeline()
			var cmds []*redis.XMessageSliceCmd
			for _, pMsg := range pendingMsgs {
				cmds = append(cmds, pipe.XRange(ctx, stream, pMsg.ID, pMsg.ID))
			}
			_, _ = pipe.Exec(ctx)

			for i, cmd := range cmds {
				msgs, err := cmd.Result()
				if err != nil || len(msgs) == 0 {
					continue
				}
				
				t, err := decodeTask(msgs[0].Values)
				if err == nil {
					ongoingTasks = append(ongoingTasks, map[string]any{
						"task":      t,
						"worker_id": consumer,
						"stream_id": pendingMsgs[i].ID,
						"idle_ms":   pendingMsgs[i].Idle.Milliseconds(),
						"retries":   pendingMsgs[i].RetryCount,
					})
				}
			}
		}
	}
	h.writeJSON(w, http.StatusOK, ongoingTasks)
}

func (h *Handlers) DeleteScheduled(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		h.writeError(w, http.StatusBadRequest, "task id is required", nil)
		return
	}

	ctx := r.Context()
	results, err := h.client.Redis.ZRange(ctx, redisclient.KeyScheduled, 0, -1).Result()
	if err != nil && err != redis.Nil {
		h.writeError(w, http.StatusInternalServerError, "failed to fetch scheduled tasks", err)
		return
	}

	for _, member := range results {
		var t task.Task
		if err := json.Unmarshal([]byte(member), &t); err == nil && t.ID == taskID {
			if err := h.client.Redis.ZRem(ctx, redisclient.KeyScheduled, member).Err(); err != nil {
				h.writeError(w, http.StatusInternalServerError, "failed to delete scheduled task", err)
				return
			}
			h.writeJSON(w, http.StatusOK, map[string]any{"success": true})
			return
		}
	}

	h.writeError(w, http.StatusNotFound, "scheduled task not found", nil)
}

func (h *Handlers) DeleteCron(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("id")
	if jobID == "" {
		h.writeError(w, http.StatusBadRequest, "job id is required", nil)
		return
	}

	ctx := r.Context()
	deleted, err := h.client.Redis.HDel(ctx, redisclient.KeyCron, jobID).Result()
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to delete cron job", err)
		return
	}

	if deleted == 0 {
		h.writeError(w, http.StatusNotFound, "cron job not found", nil)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *Handlers) DeleteDLQ(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		h.writeError(w, http.StatusBadRequest, "task id is required", nil)
		return
	}

	ctx := r.Context()
	messages, err := h.client.Redis.XRange(ctx, redisclient.KeyDLQ, "-", "+").Result()
	if err != nil && err != redis.Nil {
		h.writeError(w, http.StatusInternalServerError, "failed to read DLQ stream", err)
		return
	}

	for _, msg := range messages {
		t, err := decodeTask(msg.Values)
		if err == nil && t != nil && t.ID == taskID {
			if err := h.client.Redis.XDel(ctx, redisclient.KeyDLQ, msg.ID).Err(); err != nil {
				h.writeError(w, http.StatusInternalServerError, "failed to delete task from DLQ", err)
				return
			}
			h.writeJSON(w, http.StatusOK, map[string]any{"success": true})
			return
		}
	}

	h.writeError(w, http.StatusNotFound, "task not found in DLQ", nil)
}

