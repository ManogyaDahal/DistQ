package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
)

func RunAllTests(ctx context.Context, sdk *redisclient.SDKClient) {

	fmt.Println("DistQ Stress Test")

	fmt.Println("\nSubmitting 10 High Priority Tasks...")

	for i := 1; i <= 10; i++ {

		_, err := sdk.EnqueuePriority(
			ctx,
			"demo.sleep",
			10,
			map[string]any{
				"message":  fmt.Sprintf("HIGH-%02d", i),
				"duration": "15s",
			},
		)

		if err != nil {
			fmt.Println(err)
		}
	}

	fmt.Println("Submitting 80 Normal Priority Tasks...")

	for i := 1; i <= 80; i++ {

		_, err := sdk.EnqueuePriority(
			ctx,
			"demo.sleep",
			5,
			map[string]any{
				"message":  fmt.Sprintf("NORMAL-%02d", i),
				"duration": "5s",
			},
		)

		if err != nil {
			fmt.Println(err)
		}
	}

	fmt.Println("Submitting 10 Low Priority Tasks...")

	for i := 1; i <= 10; i++ {

		_, err := sdk.EnqueuePriority(
			ctx,
			"demo.sleep",
			1,
			map[string]any{
				"message":  fmt.Sprintf("LOW-%02d", i),
				"duration": "15s",
			},
		)

		if err != nil {
			fmt.Println(err)
		}
	}

	fmt.Println("Submitting 3 Failure Tasks...")

	for i := 1; i <= 3; i++ {

		_, err := sdk.Enqueue(
			ctx,
			"demo.fail",
			map[string]any{
				"error": fmt.Sprintf("Failure %d", i),
			},
		)

		if err != nil {
			fmt.Println(err)
		}
	}

	fmt.Println("Submitting Scheduled Tasks...")

	eta1 := time.Now().Add(30 * time.Second)
	eta2 := time.Now().Add(60 * time.Second)

	_, _ = sdk.EnqueueScheduled(
		ctx,
		"demo.print",
		5,
		eta1,
		map[string]any{
			"message": "Scheduled Task 1",
		},
	)

	_, _ = sdk.EnqueueScheduled(
		ctx,
		"demo.print",
		5,
		eta2,
		map[string]any{
			"message": "Scheduled Task 2",
		},
	)

	fmt.Println()
	fmt.Println("Stress Test Submitted")

	fmt.Println("High Priority : 10")
	fmt.Println("Normal Priority : 80")
	fmt.Println("Low Priority : 10")
	fmt.Println("Failure Tasks : 3")
	fmt.Println("Scheduled Tasks : 2")
	fmt.Println("Total Tasks : 105")

	fmt.Println()
	fmt.Println("Open the dashboard and watch:")
	fmt.Println("- Queue depths")
	fmt.Println("- Running workers")
	fmt.Println("- Ongoing tasks")
	fmt.Println("- Scheduled tasks")
	fmt.Println("- Retry attempts")
	fmt.Println("- Dead-letter queue")
}
