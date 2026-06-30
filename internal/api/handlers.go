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
	for _, msg := range messages {
		t, err := decodeTask(msg.Values)
		if err != nil {
			h.logger.Error("failed to decode DLQ task", "msg_id", msg.ID, "err", err)
			results = append(results, ReprocessResult{ID: msg.ID, Success: false, Error: "decode failed"})
			continue
		}

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

	h.writeJSON(w, http.StatusOK, map[string]any{
		"processed_count": len(messages),
		"results":         results,
	})
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
			"kind":      "cron",
			"cron_expr": req.CronExpr,
			"status":    "registered",
		})
		return
	}

	// Build the task.
	t := &task.Task{
		ID:         taskID,
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
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}
