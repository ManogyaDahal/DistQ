package api

import (
	"io/fs"
	"net/http"

	"github.com/ManogyaDahal/DistQ/internal/dashboard"
)

func RegisterRoutes(mux *http.ServeMux, handlers *Handlers, hub *Hub) error {
	// SDK ENDPOINTS — task submission and status
	mux.HandleFunc("POST /tasks", handlers.TaskEnqueue)
	mux.HandleFunc("GET /tasks/{id}", handlers.TaskGet)
	mux.HandleFunc("GET /tasks", handlers.TaskList)

	// EXISTING ENDPOINTS — metrics, workers, DLQ, WebSocket
	mux.HandleFunc("GET /api/metrics", handlers.GetMetrics)
	mux.HandleFunc("GET /api/workers", handlers.GetWorkers)
	mux.HandleFunc("GET /api/dlq", handlers.GetDLQ)
	mux.HandleFunc("POST /api/dlq/retry", handlers.RetryDLQ)
	mux.HandleFunc("GET /api/ws", hub.ServeWebSocket)

	// Static dashboard files
	subFS, err := fs.Sub(dashboard.StaticFS, "static")
	if err != nil {
		return err
	}
	fileServer := http.FileServer(http.FS(subFS))
	mux.Handle("/", fileServer)

	return nil
}
