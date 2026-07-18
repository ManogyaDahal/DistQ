package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	DistQAPIURL     string
	WebhookHost     string
	WebhookPort     string
	TaskIntervalMin time.Duration
	TaskIntervalMax time.Duration
	DemoDuration    time.Duration
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func LoadConfig() (*AppConfig, error) {
	// Best effort to load .env file
	_ = godotenv.Load()

	minMs, _ := strconv.Atoi(getEnvOrDefault("TASK_INTERVAL_MIN_MS", "2000"))
	maxMs, _ := strconv.Atoi(getEnvOrDefault("TASK_INTERVAL_MAX_MS", "5000"))
	durationSec, _ := strconv.Atoi(getEnvOrDefault("DEMO_DURATION_SECONDS", "120"))

	cfg := &AppConfig{
		DistQAPIURL:     getEnvOrDefault("DISTQ_API_URL", "http://localhost:8080"),
		WebhookHost:     getEnvOrDefault("WEBHOOK_HOST", "http://localhost:9090"),
		WebhookPort:     getEnvOrDefault("WEBHOOK_PORT", "9090"),
		TaskIntervalMin: time.Duration(minMs) * time.Millisecond,
		TaskIntervalMax: time.Duration(maxMs) * time.Millisecond,
		DemoDuration:    time.Duration(durationSec) * time.Second,
	}

	fmt.Printf("Loaded Config:\n")
	fmt.Printf(" - DistQ API URL: %s\n", cfg.DistQAPIURL)
	fmt.Printf(" - Webhook Receiver: %s\n", cfg.WebhookHost)
	fmt.Printf(" - Task Interval: %v - %v\n", cfg.TaskIntervalMin, cfg.TaskIntervalMax)
	fmt.Printf(" - Demo Duration: %v\n\n", cfg.DemoDuration)

	return cfg, nil
}
