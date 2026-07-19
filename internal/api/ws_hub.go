package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/config"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/ManogyaDahal/DistQ/pkg/task"
	"github.com/gorilla/websocket"
)

type Hub struct {
	client     *redisclient.Client
	cfg        *config.Config
	logger     *slog.Logger
	upgrader   websocket.Upgrader
	clients    map[*websocket.Conn]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

type StatsPayload struct {
	Timestamp   int64            `json:"timestamp"`
	Metrics     map[string]int64 `json:"metrics"`
	QueueDepths map[string]int64 `json:"queue_depths"`
	Workers     []WorkerStatus   `json:"workers"`
	DLQTasks    []TaskBrief      `json:"dlq_tasks"`
}

type WorkerStatus struct {
	ID           string `json:"id"`
	Status       string `json:"status"` // "active" or "stale"
	LastSeen     int64  `json:"last_seen"`
	OngoingTasks int64  `json:"ongoing_tasks"`
}

type TaskBrief struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Type      string    `json:"type"`
	Priority  int       `json:"priority"`
	Source    string    `json:"source,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	ErrorMsg  string    `json:"error_msg"`
}

func NewHub(client *redisclient.Client, cfg *config.Config, logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		client: client,
		cfg:    cfg,
		logger: logger.With("component", "ws_hub"),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for dev simplicity
			},
		},
		clients:    make(map[*websocket.Conn]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.logger.Info("shutting down ws hub")
			h.mu.Lock()
			for client := range h.clients {
				client.Close()
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("client connected")
			// Send initial update immediately
			if payload, err := h.collectStats(ctx); err == nil {
				_ = client.WriteJSON(payload)
			}
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				client.Close()
				delete(h.clients, client)
				h.logger.Debug("client disconnected")
			}
			h.mu.Unlock()
		case <-ticker.C:
			h.mu.RLock()
			clientCount := len(h.clients)
			h.mu.RUnlock()

			if clientCount > 0 {
				stats, err := h.collectStats(ctx)
				if err != nil {
					h.logger.Error("failed to collect stats for broadcast", "err", err)
					continue
				}

				h.mu.Lock()
				for client := range h.clients {
					if err := client.WriteJSON(stats); err != nil {
						h.logger.Debug("failed to write to client, unregistering", "err", err)
						client.Close()
						delete(h.clients, client)
					}
				}
				h.mu.Unlock()
			}
		}
	}
}

func (h *Hub) ServeWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "err", err)
		return
	}

	h.register <- conn

	// Keep-alive/read loop to detect disconnection
	go func() {
		defer func() {
			h.unregister <- conn
		}()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func (h *Hub) collectStats(ctx context.Context) (*StatsPayload, error) {
	now := time.Now().Unix()
	timeoutSecs := int64(h.cfg.HeartbeatTimeout.Seconds())

	metrics := map[string]int64{
		"ongoing_tasks":   0,
		"total_workers":   0,
		"free_workers":    0,
		"dlq_count":       0,
		"scheduled_count": 0,
		"cron_count":      0,
	}

	queueDepths := make(map[string]int64)
	workerPendingCounts := make(map[string]int64)

	for _, p := range h.cfg.PriorityLevels {
		stream := fmt.Sprintf(redisclient.KeyQueueStream, p)
		pStr := strconv.Itoa(p)

		// BUG FIX: Calculate queue depth correctly using Redis Streams consumer group semantics
		//
		// PROBLEM with old code (using XLEN):
		// - XLEN returns total entries ever added to stream (including acked/completed tasks)
		// - Once tasks are ACKED, they're removed from the pending list but stay in the stream
		// - This causes queue_depths to remain high even when all tasks are completed
		//
		// SOLUTION:
		// Queue depth should represent: "tasks still waiting to be processed"
		// This includes:
		// 1. Pending entries = tasks claimed by workers but not yet acked (XPENDING.Count)
		// 2. Undelivered entries = tasks in stream but never claimed by any worker yet
		//
		// Calculate undelivered using consumer group info:
		// - XINFO GROUPS reports the group's lag, i.e. entries that have not yet
		//   been delivered to that group.

		pendingInfo, err := h.client.Redis.XPending(ctx, stream, "workers").Result()
		pendingCount := int64(0)
		if err == nil {
			pendingCount = pendingInfo.Count
			metrics["ongoing_tasks"] += pendingCount
			for consumer, countStr := range pendingInfo.Consumers {
				workerPendingCounts[consumer] += countStr
			}
		}

		// Get undelivered entries count
		undeliveredCount := int64(0)
		groupInfo, err := h.client.Redis.XInfoGroups(ctx, stream).Result()
		if err == nil {
			for _, group := range groupInfo {
				if group.Name == "workers" {
					// Lag is -1 when Redis cannot determine it; avoid reporting an
					// inaccurate XLEN-based value in that case.
					if group.Lag >= 0 {
						undeliveredCount = group.Lag
					}
					break
				}
			}
		}

		// Queue depth = pending (being processed/retrying) + undelivered (waiting for worker)
		// This ensures completed tasks don't contribute to queue depth
		queueDepths[pStr] = pendingCount + undeliveredCount
	}

	workersList := []WorkerStatus{}
	dbWorkers, err := h.client.Redis.HGetAll(ctx, redisclient.KeyWorkers).Result()
	if err == nil {
		for id, tsStr := range dbWorkers {
			ts, err := strconv.ParseInt(tsStr, 10, 64)
			if err != nil {
				continue
			}

			status := "active"
			if now-ts > timeoutSecs {
				status = "stale"
			}

			ongoing := workerPendingCounts[id]
			if status == "active" {
				metrics["total_workers"]++
				if ongoing == 0 {
					metrics["free_workers"]++
				}
			}

			workersList = append(workersList, WorkerStatus{
				ID:           id,
				Status:       status,
				LastSeen:     ts,
				OngoingTasks: ongoing,
			})
		}
	}

	dlqCount, err := h.client.Redis.XLen(ctx, redisclient.KeyDLQ).Result()
	if err == nil {
		metrics["dlq_count"] = dlqCount
	}

	scheduledCount, err := h.client.Redis.ZCard(ctx, redisclient.KeyScheduled).Result()
	if err == nil {
		metrics["scheduled_count"] = scheduledCount
	}

	cronCount, err := h.client.Redis.HLen(ctx, redisclient.KeyCron).Result()
	if err == nil {
		metrics["cron_count"] = cronCount
	}

	dlqTasks := []TaskBrief{}
	dlqMsgs, err := h.client.Redis.XRevRangeN(ctx, redisclient.KeyDLQ, "+", "-", 50).Result()
	if err == nil {
		for _, msg := range dlqMsgs {
			t, err := decodeTask(msg.Values)
			if err != nil {
				continue
			}
			dlqTasks = append(dlqTasks, TaskBrief{
				ID:        t.ID,
				Name:      t.Name,
				Type:      t.Type,
				Priority:  t.Priority,
				Source:    t.Source,
				Status:    string(t.Status),
				CreatedAt: t.CreatedAt,
				ErrorMsg:  t.ErrorMsg,
			})
		}
	}

	return &StatsPayload{
		Timestamp:   now,
		Metrics:     metrics,
		QueueDepths: queueDepths,
		Workers:     workersList,
		DLQTasks:    dlqTasks,
	}, nil
}

func decodeTask(values map[string]any) (*task.Task, error) {
	raw, ok := values["task"]
	if !ok {
		return nil, fmt.Errorf("missing task field")
	}

	var data []byte
	switch v := raw.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		return nil, fmt.Errorf("unexpected task field type")
	}

	var t task.Task
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("unmarshal task: %w", err)
	}

	return &t, nil
}
