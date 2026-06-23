package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/queue"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/ManogyaDahal/DistQ/pkg/task"
)

type Handlers struct {
	client *redisclient.Client
	hub    *Hub
	logger *slog.Logger
}

func NewHandlers(client *redisclient.Client, hub *Hub, logger *slog.Logger) *Handlers {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handlers{
		client: client,
		hub:    hub,
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
