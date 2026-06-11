package api

import (
	"encoding/json"
	"net/http"

	"github.com/ManogyaDahal/DistQ/pkg/models"
	"github.com/ManogyaDahal/DistQ/pkg/queue"
)

type Handler struct {
	Broker queue.Broker
}

func NewHandler(broker queue.Broker) *Handler {
	return &Handler{Broker: broker}
}

// POST /tasks
func (h *Handler) Enqueue(w http.ResponseWriter, r *http.Request) {
	var req models.EnqueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	task := models.Task{
		Type:     req.Type,
		Payload:  req.Payload,
		Priority: req.Priority,
	}

	id, err := h.Broker.Enqueue(r.Context(), task)
	if err != nil {
		http.Error(w, `{"error":"enqueue failed"}`, http.StatusInternalServerError)
		return
	}

	resp := models.EnqueueResponse{
		ID:     id,
		Status: models.StatusPending,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GET /tasks/:id
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, `{"error":"missing id"}`, http.StatusBadRequest)
		return
	}

	task, err := h.Broker.GetTask(r.Context(), id)
	if err == models.ErrTaskNotFound {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

// GET /tasks
func (h *Handler) ListPending(w http.ResponseWriter, r *http.Request) {
	tasks, _ := h.Broker.ListPending(r.Context(), 20)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}
