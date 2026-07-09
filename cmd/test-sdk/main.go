package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/client"
)

func main() {
	ctx := context.Background()

	sdk := client.New("http://localhost:8080")

	payload, err := json.Marshal(map[string]any{
		"message":  "new-client-sdk-test",
		"duration": "5s",
	})
	if err != nil {
		log.Fatal(err)
	}

	result, err := sdk.SubmitTask(ctx, client.SubmitTaskRequest{
		Type:     "demo.sleep",
		Payload:  payload,
		Priority: 10,
	})
	if err != nil {
		log.Fatal("submit failed:", err)
	}

	fmt.Printf("submitted: %+v\n", result)

	time.Sleep(7 * time.Second)

	task, err := sdk.GetTask(ctx, result.ID)
	if err != nil {
		log.Fatal("get task failed:", err)
	}

	fmt.Printf("task after worker processing: %+v\n", task)

	metrics, err := sdk.GetMetrics(ctx)
	if err != nil {
		log.Fatal("metrics failed:", err)
	}

	fmt.Printf("metrics: %+v\n", metrics)
}
