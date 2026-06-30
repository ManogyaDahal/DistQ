package api

import (
	"io/fs"
	"net/http"

	"github.com/ManogyaDahal/DistQ/internal/dashboard"
)

func RegisterRoutes(mux *http.ServeMux, handlers *Handlers, hub *Hub) error {
	// REST API endpoints
	mux.HandleFunc("POST /api/task", handlers.SubmitTask)
	mux.HandleFunc("GET /api/task/{id}", handlers.GetTask)
	mux.HandleFunc("GET /api/metrics", handlers.GetMetrics)
	mux.HandleFunc("GET /api/workers", handlers.GetWorkers)
	mux.HandleFunc("GET /api/dlq", handlers.GetDLQ)
	mux.HandleFunc("POST /api/dlq/retry", handlers.RetryDLQ)

	// WebSocket telemetry stream
	mux.HandleFunc("GET /api/ws", hub.ServeWebSocket)

	// Serve embedded static dashboard files
	subFS, err := fs.Sub(dashboard.StaticFS, "static")
	if err != nil {
		return err
	}
	fileServer := http.FileServer(http.FS(subFS))
	mux.Handle("/", fileServer)

	return nil
}
