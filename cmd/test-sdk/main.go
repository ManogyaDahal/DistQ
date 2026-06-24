package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
)

func main() {
	// Create SDK client pointing to your API server
	c := redisclient.NewSDK("http://localhost:8081")

	ctx := context.Background()

	// Test 1: Enqueue a task
	fmt.Println("=== Test 1: Enqueue ===")
	resp, err := c.Enqueue(ctx, "send_email", map[string]any{
		"to":      "user@test.com",
		"subject": "Hello from SDK",
	})
	if err != nil {
		log.Fatal("Enqueue failed:", err)
	}
	fmt.Printf("Enqueued: ID=%s, Status=%s\n", resp.ID, resp.Status)

	// Test 2: Check status immediately
	fmt.Println("\n=== Test 2: Status ===")
	task, err := c.Status(ctx, resp.ID)
	if err != nil {
		log.Fatal("Status failed:", err)
	}
	fmt.Printf("Task: Type=%s, Status=%s, Priority=%d\n", task.Type, task.Status, task.Priority)

	// Test 3: Check non-existent task (should fail)
	fmt.Println("\n=== Test 3: Not Found ===")
	_, err = c.Status(ctx, "fake-id-123")
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	} else {
		fmt.Println("ERROR: Should have failed!")
	}

	fmt.Println("\n=== All tests passed ===")
}
