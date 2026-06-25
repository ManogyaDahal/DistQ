package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/models"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/redis/go-redis/v9"
)

// TaskEnqueue handles POST /tasks — submit a new task to the queue
func (h *Handlers) TaskEnqueue(w http.ResponseWriter, r *http.Request) {
	var req models.EnqueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid json", err)
		return
	}

	// Generate task ID and populate fields
	task := models.Task{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      req.Type,
		Payload:   req.Payload,
		Priority:  req.Priority,
		Status:    models.StatusPending,
		CreatedAt: time.Now(),
	}

	// Store task metadata in Redis hash
	data, err := json.Marshal(task)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to marshal task", err)
		return
	}

	metaKey := fmt.Sprintf(redisclient.KeyTaskMeta, task.ID)
	if err := h.client.Redis.HSet(r.Context(), metaKey, "data", string(data)).Err(); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to store task", err)
		return
	}

	// Add to priority stream for workers to pick up
	streamKey := fmt.Sprintf(redisclient.KeyQueueStream, task.Priority)
	taskJSON, err := json.Marshal(task)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to marshal task for stream", err)
		return
	}

	if err := h.client.Redis.XAdd(r.Context(), &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]any{"task": string(taskJSON)},
	}).Err(); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to enqueue to stream", err)
		return
	}

	h.logger.Info("task enqueued", "task_id", task.ID, "type", task.Type)

	h.writeJSON(w, http.StatusOK, models.EnqueueResponse{
		ID:     task.ID,
		Status: models.StatusPending,
	})
}

// TaskGet handles GET /tasks/{id} — check status of one task
func (h *Handlers) TaskGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		h.writeError(w, http.StatusBadRequest, "missing task id", nil)
		return
	}

	metaKey := fmt.Sprintf(redisclient.KeyTaskMeta, id)
	data, err := h.client.Redis.HGet(r.Context(), metaKey, "data").Result()
	if err == redis.Nil {
		h.writeError(w, http.StatusNotFound, "task not found", nil)
		return
	}
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to read task", err)
		return
	}

	var task models.Task
	if err := json.Unmarshal([]byte(data), &task); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to decode task", err)
		return
	}

	h.writeJSON(w, http.StatusOK, task)
}

// TaskList handles GET /tasks — list pending tasks
func (h *Handlers) TaskList(w http.ResponseWriter, r *http.Request) {
	if h.hub == nil || h.hub.cfg == nil {
		h.writeError(w, http.StatusInternalServerError, "dashboard config unavailable", nil)
		return
	}

	tasks := make([]models.Task, 0)
	for _, priority := range h.hub.cfg.PriorityLevels {
		streamKey := fmt.Sprintf(redisclient.KeyQueueStream, priority)
		messages, err := h.client.Redis.XRange(r.Context(), streamKey, "-", "+").Result()
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to read queue", err)
			return
		}

		for _, msg := range messages {
			raw, ok := msg.Values["task"]
			if !ok {
				continue
			}

			var data []byte
			switch v := raw.(type) {
			case string:
				data = []byte(v)
			case []byte:
				data = v
			default:
				continue
			}

			var item models.Task
			if err := json.Unmarshal(data, &item); err != nil {
				continue
			}

			// Preserve the source priority stream if it was not encoded.
			if item.Priority == 0 {
				item.Priority = priority
			}
			tasks = append(tasks, item)
		}
	}

	h.writeJSON(w, http.StatusOK, tasks)
}
