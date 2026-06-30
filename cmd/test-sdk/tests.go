package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
)

func RunAllTests(ctx context.Context, sdk *redisclient.SDKClient) {

	fmt.Println("\n========== DistQ SDK Integration Tests ==========")

	// Test 1
	fmt.Println("\n[TEST 1] Normal Print Task")

	t1, err := sdk.Enqueue(ctx, "demo.print", map[string]any{
		"message": "Hello DistQ",
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Task ID:", t1.ID)

	// Test 2
	fmt.Println("\n[TEST 2] High Priority Task")

	t2, err := sdk.EnqueuePriority(ctx, "demo.print", 10, map[string]any{
		"message": "High Priority",
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Task ID:", t2.ID)

	// Test 3
	fmt.Println("\n[TEST 3] Low Priority Task")

	t3, err := sdk.EnqueuePriority(ctx, "demo.print", 1, map[string]any{
		"message": "Low Priority",
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Task ID:", t3.ID)

	// Test 4
	fmt.Println("\n[TEST 4] Sleep Task")

	t4, err := sdk.Enqueue(ctx, "demo.sleep", map[string]any{
		"message":  "Sleeping...",
		"duration": "15s",
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Task ID:", t4.ID)

	// Test 5
	fmt.Println("\n[TEST 5] Scheduled Task")

	eta := time.Now().Add(30 * time.Second)

	t5, err := sdk.EnqueueScheduled(
		ctx,
		"demo.print",
		5,
		eta,
		map[string]any{
			"message": "Run After 30 Seconds",
		},
	)
	if err != nil {
		fmt.Println("Scheduled enqueue failed:", err)
	} else {
		fmt.Println("Task ID:", t5.ID)
	}

	// Test 6
	fmt.Println("\n[TEST 6] Failure Task")

	t6, err := sdk.Enqueue(ctx, "demo.fail", map[string]any{
		"error": "Intentional Failure",
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Task ID:", t6.ID)

	// Test 7
	fmt.Println("\n[TEST 7] Queue Load")

	for i := 1; i <= 20; i++ {
		_, _ = sdk.Enqueue(ctx, "demo.sleep", map[string]any{
			"message":  fmt.Sprintf("Bulk Task %d", i),
			"duration": "5s",
		})
	}

	fmt.Println("20 tasks submitted.")

	// Test 8
	fmt.Println("\n[TEST 8] Status Check")

	time.Sleep(2 * time.Second)

	status, err := sdk.Status(ctx, t1.ID)
	if err != nil {
		fmt.Println("Status check failed:", err)
	} else {
		fmt.Printf("%+v\n", *status)
	}

	// Test 9
	fmt.Println("\n[TEST 9] Invalid Task")

	_, err = sdk.Status(ctx, "fake-task")
	if err != nil {
		fmt.Println("Expected:", err)
	}

	fmt.Println("\n===================================")
	fmt.Println("All tests submitted successfully.")
	fmt.Println("Check the dashboard for results.")
	fmt.Println("===================================")
}
