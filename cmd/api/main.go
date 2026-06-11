package main

import (
	"log"
	"net/http"

	"github.com/ManogyaDahal/DistQ/internal/api"
)

func main() {
	broker := api.NewStubBroker()
	handler := api.NewHandler(broker)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /tasks", handler.Enqueue)
	mux.HandleFunc("GET /tasks/{id}", handler.GetStatus)
	mux.HandleFunc("GET /tasks", handler.ListPending)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message":"DistQ API is running","endpoints":["POST /tasks","GET /tasks/:id","GET /tasks"]}`))
	})
	log.Println("DistQ API running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
