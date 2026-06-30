package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/models"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	taskpkg "github.com/ManogyaDahal/DistQ/pkg/task"
	"github.com/redis/go-redis/v9"
)

// TaskEnqueue handles POST /tasks — submit a new task to the queue
func (h *Handlers) TaskEnqueue(w http.ResponseWriter, r *http.Request) {
	var req models.EnqueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid json", err)
		return
	}
	if strings.TrimSpace(req.Type) == "" {
		h.writeError(w, http.StatusBadRequest, "task type is required", nil)
		return
	}

	now := time.Now().UTC()
	isScheduled := req.ETA != nil && req.ETA.After(now)

	// Generate task ID and populate fields
	apiTask := models.Task{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Type:      req.Type,
		Payload:   req.Payload,
		Priority:  req.Priority,
		Status:    models.StatusPending,
		ETA:       req.ETA,
		CreatedAt: now,
	}
	if isScheduled {
		apiTask.Status = models.StatusScheduled
	}

	// Store task metadata in Redis hash
	data, err := json.Marshal(apiTask)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to marshal task", err)
		return
	}

	metaKey := fmt.Sprintf(redisclient.KeyTaskMeta, apiTask.ID)
	if err := h.client.Redis.HSet(r.Context(), metaKey, "data", string(data)).Err(); err != nil {
		h.writeError(w, http.StatusInternalServerError, "failed to store task", err)
		return
	}

	if isScheduled {
		scheduledAt := req.ETA.UTC()
		scheduledAtPtr := scheduledAt
		payload, err := json.Marshal(taskpkg.Task{
			ID:         apiTask.ID,
			Type:       apiTask.Type,
			Payload:    toRawMessage(apiTask.Payload),
			Priority:   apiTask.Priority,
			Status:     taskpkg.StatusPending,
			MaxRetries: 3,
			ETA:        &scheduledAtPtr,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
		if err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to marshal scheduled task", err)
			return
		}

		if err := h.client.Redis.ZAdd(r.Context(), redisclient.KeyScheduled, redis.Z{
			Score:  float64(scheduledAt.Unix()),
			Member: string(payload),
		}).Err(); err != nil {
			h.writeError(w, http.StatusInternalServerError, "failed to schedule task", err)
			return
		}
	} else {
		// Add to priority stream for workers to pick up
		streamKey := fmt.Sprintf(redisclient.KeyQueueStream, apiTask.Priority)
		taskJSON, err := json.Marshal(apiTask)
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
	}

	h.logger.Info("task enqueued", "task_id", apiTask.ID, "type", apiTask.Type)

	respStatus := models.StatusPending
	if isScheduled {
		respStatus = models.StatusScheduled
	}

	h.writeJSON(w, http.StatusOK, models.EnqueueResponse{
		ID:     apiTask.ID,
		Status: respStatus,
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

func toRawMessage(payload map[string]any) json.RawMessage {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return raw
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
