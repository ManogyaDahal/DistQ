package api

import (
	"net/http"
	"os"
	"path/filepath"
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

	// WebSocket telemetry stream
	mux.HandleFunc("GET /api/ws", hub.ServeWebSocket)

	// Serve React dashboard build
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	distPath := filepath.Join(workDir, "dashboard", "dist")
	
	// Check if dist folder exists
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		// Fallback to current directory + dashboard/dist if we're not running from project root
		distPath = "dashboard/dist"
	}

	fileServer := http.FileServer(http.Dir(distPath))
	mux.Handle("/", fileServer)

	return nil
}
