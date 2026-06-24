package redisclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/models"
)

// SDKClient wraps the DistQ HTTP API for Go developers.
// This is NOT the Redis client. It calls the REST API endpoints.
type SDKClient struct {
	BaseURL string
	HTTP    *http.Client
}

// NewSDK creates a client that talks to the DistQ API server.
func NewSDK(baseURL string) *SDKClient {
	return &SDKClient{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Enqueue submits a new task to the queue via the API.
func (c *SDKClient) Enqueue(ctx context.Context, taskType string, payload map[string]any) (*models.EnqueueResponse, error) {
	req := models.EnqueueRequest{
		Type:    taskType,
		Payload: payload,
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/tasks", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("enqueue failed: %d", resp.StatusCode)
	}

	var out models.EnqueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Status checks the current state of a task by its ID.
func (c *SDKClient) Status(ctx context.Context, id string) (*models.Task, error) {
	httpReq, _ := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/tasks/"+id, nil)

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, models.ErrTaskNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status failed: %d", resp.StatusCode)
	}

	var out models.Task
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
