package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/client"
)

func main() {
	// 1. Start a local webhook receiver
	go func() {
		http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("Webhook received! Sleeping for 10 seconds...")
			time.Sleep(10 * time.Second)
			fmt.Println("Webhook task complete!")
			w.WriteHeader(http.StatusOK)
		})
		fmt.Println("Starting Webhook Server on :9091")
		http.ListenAndServe(":9091", nil)
	}()

	// 2. Initialize DistQ Client
	c := client.New("http://localhost:8080")

	// 3. Randomly submit tasks
	for {
		interval := time.Duration(rand.Intn(4000)+1000) * time.Millisecond
		time.Sleep(interval)

		fmt.Println("Submitting new webhook task...")
		_, err := c.SubmitWebhook(context.Background(), client.WebhookRequest{
			URL: "http://host.docker.internal:9091/webhook",
		})

		if err != nil {
			fmt.Printf("Failed to enqueue: %v\n", err)
		} else {
			fmt.Println("Successfully enqueued task!")
		}
	}
}
