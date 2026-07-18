package api

import (
	"net/http"
)

func RegisterRoutes(mux *http.ServeMux, handlers *Handlers, hub *Hub) error {
	// REST API endpoints
	mux.HandleFunc("POST /api/task", handlers.SubmitTask)
	mux.HandleFunc("GET /api/task/{id}", handlers.GetTask)
	mux.HandleFunc("GET /api/metrics", handlers.GetMetrics)
	mux.HandleFunc("GET /api/workers", handlers.GetWorkers)
	mux.HandleFunc("GET /api/dlq", handlers.GetDLQ)
	mux.HandleFunc("POST /api/dlq/retry", handlers.RetryDLQ)
	mux.HandleFunc("POST /api/dlq/retry/{id}", handlers.RetryDLQ)
	mux.HandleFunc("GET /api/enqueued", handlers.GetEnqueued)
	mux.HandleFunc("GET /api/completed", handlers.GetCompleted)
	mux.HandleFunc("GET /api/scheduled", handlers.GetScheduled)
	mux.HandleFunc("GET /api/cron", handlers.GetCron)
	mux.HandleFunc("GET /api/ongoing", handlers.GetOngoing)

	// DELETE endpoints
	mux.HandleFunc("DELETE /api/scheduled/{id}", handlers.DeleteScheduled)
	mux.HandleFunc("DELETE /api/cron/{id}", handlers.DeleteCron)
	mux.HandleFunc("DELETE /api/dlq/{id}", handlers.DeleteDLQ)

	// WebSocket telemetry stream
	mux.HandleFunc("GET /api/ws", hub.ServeWebSocket)

	return nil
}
