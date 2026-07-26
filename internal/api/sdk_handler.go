package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/client"
)

// SDKDemoResult is deliberately presentation-friendly while its operations
// remain entirely backed by the public Go client SDK.
type SDKDemoResult struct {
	Methods     []string                `json:"methods"`
	TaskIDs     []string                `json:"task_ids,omitempty"`
	Task        *client.TaskStatus      `json:"task,omitempty"`
	Metrics     *client.MetricsResponse `json:"metrics,omitempty"`
	Workers     []client.WorkerStatus   `json:"workers,omitempty"`
	Submitted   int                     `json:"submitted"`
	Retries     int                     `json:"retries"`
	DLQCount    int64                   `json:"dlq_count"`
	ExecutionMS int64                   `json:"execution_ms"`
	LatencyMS   int64                   `json:"latency_ms"`
	Message     string                  `json:"message"`
}

func (h *Handlers) RunSDKDemo(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	baseURL := "http://" + r.Host
	sdk := client.New(baseURL)
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	result, err := h.runSDKAction(ctx, sdk, r.PathValue("action"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "SDK demo failed", err)
		return
	}
	result.ExecutionMS = time.Since(started).Milliseconds()
	result.LatencyMS = result.ExecutionMS
	if result.Metrics != nil {
		result.DLQCount = result.Metrics.Metrics["dlq_count"]
	}
	h.writeJSON(w, http.StatusOK, result)
}

// GetSDKTask keeps lifecycle polling inside the Go SDK boundary.
func (h *Handlers) GetSDKTask(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	sdk := client.New("http://" + r.Host)
	status, err := sdk.GetTask(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "SDK GetTask failed", err)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]any{"task": status, "api_response_ms": time.Since(started).Milliseconds()})
}

func (h *Handlers) runSDKAction(ctx context.Context, sdk *client.Client, action string) (*SDKDemoResult, error) {
	result := &SDKDemoResult{}
	metrics := func() error {
		m, err := sdk.GetMetrics(ctx)
		if err == nil {
			result.Metrics = m
			result.Methods = append(result.Methods, "GetMetrics()")
		}
		return err
	}
	submit := func(kind string, priority int, eta *time.Time) error {
		payload, _ := json.Marshal(map[string]any{"message": "Hello from SDK", "demo": action})
		res, err := sdk.SubmitTask(ctx, client.SubmitTaskRequest{
			Type: kind, Payload: payload, Priority: priority, ETA: eta, Source: "Go SDK Playground",
		})
		if err != nil {
			return err
		}
		result.Methods = append(result.Methods, "SubmitTask()")
		result.TaskIDs = append(result.TaskIDs, res.ID)
		result.Submitted++
		return nil
	}

	switch action {
	case "connection":
		if err := metrics(); err != nil {
			return nil, err
		}
		result.Message = "Connection successful — DistQ API is reachable through the Go SDK."
	case "submit":
		if err := submit("demo.sleep", 5, nil); err != nil {
			return nil, err
		}
		status, err := sdk.GetTask(ctx, result.TaskIDs[0])
		if err == nil {
			result.Task = status
			result.Methods = append(result.Methods, "GetTask()")
		}
		workers, err := sdk.GetWorkers(ctx)
		if err == nil {
			result.Workers = workers
			result.Methods = append(result.Methods, "GetWorkers()")
		}
		_ = metrics()
		result.Message = "Demo task submitted through SubmitTask()."
	case "priority":
		for _, batch := range []struct{ n, priority int }{{10, 10}, {20, 5}, {10, 1}} {
			for i := 0; i < batch.n; i++ {
				if err := submit("demo.sleep", batch.priority, nil); err != nil {
					return nil, err
				}
			}
		}
		_ = metrics()
		result.Message = "40 priority tasks submitted: 10 high, 20 normal, 10 low. Open Queue Dashboard to observe processing."
	case "scheduled":
		now := time.Now()
		first, second := now.Add(30*time.Second), now.Add(60*time.Second)
		if err := submit("demo.sleep", 5, &first); err != nil {
			return nil, err
		}
		if err := submit("demo.sleep", 5, &second); err != nil {
			return nil, err
		}
		_ = metrics()
		result.Message = "Two SDK scheduled tasks created for 30 seconds and 60 seconds from now."
	case "failure":
		maxRetries := 1 // Keep the presentation demo short while still showing a retry.
		payload, _ := json.Marshal(map[string]any{"message": "Intentional SDK failure"})
		failure, err := sdk.SubmitTask(ctx, client.SubmitTaskRequest{
			Type:       "demo.fail",
			Payload:    payload,
			Priority:   5,
			MaxRetries: &maxRetries,
			Source:     "Go SDK Playground",
		})
		if err != nil {
			return nil, err
		}
		result.Methods = append(result.Methods, "SubmitTask()")
		result.TaskIDs = append(result.TaskIDs, failure.ID)
		result.Submitted++
		for i := 0; i < 10; i++ {
			status, err := sdk.GetTask(ctx, failure.ID)
			if err == nil {
				result.Task = status
				result.Methods = append(result.Methods, "GetTask()")
				if status.Status == "dead" || status.Status == "failed" {
					break
				}
			}
			time.Sleep(time.Second)
		}
		_ = metrics()
		if result.Task != nil && result.Task.Status == "dead" {
			result.Message = "Failure demo complete: the task retried once and was moved to the DLQ."
		} else {
			result.Message = "Failure task submitted; retry status is being synchronized with the dashboard."
		}
	case "stress":
		for i := 0; i < 100; i++ {
			priority := []int{10, 5, 1}[i%3]
			kind := "demo.sleep"
			if i%15 == 0 {
				kind = "demo.fail"
			}
			if err := submit(kind, priority, nil); err != nil {
				return nil, err
			}
		}
		_ = metrics()
		result.Message = "100/100 mixed tasks submitted through the Go SDK."
	case "complete":
		if err := metrics(); err != nil {
			return nil, err
		}
		if err := submit("demo.sleep", 5, nil); err != nil {
			return nil, err
		}
		for i := 0; i < 5; i++ {
			status, err := sdk.GetTask(ctx, result.TaskIDs[0])
			if err != nil {
				break
			}
			result.Task = status
			result.Methods = append(result.Methods, "GetTask()")
			if status.Status == "done" || status.Status == "failed" || status.Status == "dead" {
				break
			}
			time.Sleep(time.Second)
		}
		_ = metrics()
		workers, err := sdk.GetWorkers(ctx)
		if err == nil {
			result.Workers = workers
			result.Methods = append(result.Methods, "GetWorkers()")
		}
		result.Message = "Complete SDK demo finished: connected, submitted, monitored, and retrieved live system data."
	default:
		return nil, fmt.Errorf("unknown SDK action %q", action)
	}
	return result, nil
}
