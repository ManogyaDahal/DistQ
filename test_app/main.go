package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/client"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("       DistQ Independent Test Application")
	fmt.Println("==================================================")

	// 1. Load config
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize DistQ Client SDK
	distqClient := client.New(cfg.DistQAPIURL)

	// 3. Health Check (ensure DistQ API is running)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, err = distqClient.GetMetrics(ctx)
	cancel()
	if err != nil {
		fmt.Printf("\n[ERROR] Could not connect to DistQ API at %s\n", cfg.DistQAPIURL)
		fmt.Printf("Reason: %v\n", err)
		fmt.Printf("Please ensure the DistQ API server is running (go run ./cmd/api) before starting this test app.\n")
		os.Exit(1)
	}
	fmt.Printf("Successfully connected to DistQ API.\n\n")

	// Set up main context for graceful shutdown
	mainCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Optionally timeout after DEMO_DURATION
	if cfg.DemoDuration > 0 {
		var timeoutCancel context.CancelFunc
		mainCtx, timeoutCancel = context.WithTimeout(mainCtx, cfg.DemoDuration)
		defer timeoutCancel()
	}

	// 4. Start Webhook Receiver
	go func() {
		if err := StartWebhookReceiver(mainCtx, cfg.WebhookPort); err != nil {
			fmt.Printf("[ERROR] Webhook receiver stopped: %v\n", err)
		}
	}()
	time.Sleep(500 * time.Millisecond) // Give the server a moment to start

	// 5. Initialize Producer
	producer := NewProducer(distqClient, cfg)

	// 6. Submit a Cron Job upfront
	producer.SubmitCronJob(mainCtx)

	// 7. Start random task generation
	go producer.Start(mainCtx)

	// 8. Wait for shutdown
	fmt.Printf("\nTest application is running. Press Ctrl+C to stop.\n\n")
	<-mainCtx.Done()

	fmt.Println("\nShutting down test application gracefully...")
	time.Sleep(1 * time.Second) // allow some time for graceful shutdown of webhook server
	fmt.Println("Done.")
}
