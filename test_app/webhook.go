package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StartWebhookReceiver starts a local HTTP server to receive webhook callbacks from DistQ.
func StartWebhookReceiver(ctx context.Context, port string) error {
	mux := http.NewServeMux()

	// Handler 1: Simulate a successful order creation webhook
	mux.HandleFunc("POST /webhook/order-created", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		fmt.Printf(" [WEBHOOK RECEIVED] POST /webhook/order-created\n")
		fmt.Printf("   ├─ Headers: User-Agent=%s\n", r.Header.Get("User-Agent"))
		fmt.Printf("   └─ Body: %s\n", string(body))

		// Simulate actual work with a random delay between 10 and 15 seconds
		delay := time.Duration(10000 + (time.Now().UnixNano() % 5000)) * time.Millisecond
		fmt.Printf("   └─ Simulating work for %v...\n", delay)
		time.Sleep(delay)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Handler 2: Simulate a notification webhook
	mux.HandleFunc("POST /webhook/notification", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		fmt.Printf(" [WEBHOOK RECEIVED] POST /webhook/notification\n")
		fmt.Printf("   └─ Body: %s\n", string(body))

		// Simulate actual work with a random delay between 10 and 15 seconds
		delay := time.Duration(10000 + (time.Now().UnixNano() % 5000)) * time.Millisecond
		fmt.Printf("   └─ Simulating work for %v...\n", delay)
		time.Sleep(delay)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Handler 3: Intentionally failing webhook to demonstrate retry and DLQ logic
	mux.HandleFunc("POST /webhook/failing-endpoint", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf(" [WEBHOOK RECEIVED] POST /webhook/failing-endpoint -> Intentionally returning 500\n")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal server error"}`))
	})

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Starting Webhook Receiver on port %s...\n", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("webhook receiver failed: %w", err)
	}

	return nil
}
