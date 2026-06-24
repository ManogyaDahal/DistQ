package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ManogyaDahal/DistQ/pkg/config"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/ManogyaDahal/DistQ/pkg/task"
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
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	Priority   int       `json:"priority"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	ErrorMsg   string    `json:"error_msg"`
	Payload    string    `json:"payload,omitempty"`
	MaxRetries int       `json:"max_retries"`
	RetryCount int       `json:"retry_count"`
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

		// Use consumer-group lag instead of XLen. XLen counts every message
		// ever written including already-ACK'd ones — Redis streams don't
		// auto-delete. Pending = delivered to workers but not yet ACK'd;
		// Lag = enqueued but not yet delivered to any consumer.
		depth := int64(0)
		groups, groupErr := h.client.Redis.XInfoGroups(ctx, stream).Result()
		if groupErr == nil {
			for _, g := range groups {
				if g.Name == "workers" {
					depth = g.Pending + g.Lag
					break
				}
			}
		}
		queueDepths[pStr] = depth

		pendingInfo, err := h.client.Redis.XPending(ctx, stream, "workers").Result()
		if err == nil {
			metrics["ongoing_tasks"] += pendingInfo.Count
			for consumer, count := range pendingInfo.Consumers {
				// Consumers are named "workerID-slot-N"; strip the slot suffix
				// to aggregate back to the base workerID used in the heartbeat hash.
				baseID := consumer
				if idx := strings.LastIndex(consumer, "-slot-"); idx != -1 {
					baseID = consumer[:idx]
				}
				workerPendingCounts[baseID] += count
			}
		}
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
				ID:         t.ID,
				Type:       t.Type,
				Priority:   t.Priority,
				Status:     string(t.Status),
				CreatedAt:  t.CreatedAt,
				ErrorMsg:   t.ErrorMsg,
				Payload:    string(t.Payload),
				MaxRetries: t.MaxRetries,
				RetryCount: t.RetryCount,
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
