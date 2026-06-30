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

// SDKClient wraps the DistQ REST API.
//
// This SDK allows Go developers to submit tasks,
// schedule tasks and query task status without
// directly interacting with Redis.
type SDKClient struct {
	BaseURL string
	HTTP    *http.Client
}

// NewSDK creates a new SDK client.
func NewSDK(baseURL string) *SDKClient {
	return &SDKClient{
		BaseURL: baseURL,
		HTTP: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

///////////////////////////////////////////////////////////////
// BASIC ENQUEUE
///////////////////////////////////////////////////////////////

// Enqueue submits a normal task.
//
// Default priority = 5.
func (c *SDKClient) Enqueue(
	ctx context.Context,
	taskType string,
	payload map[string]any,
) (*models.EnqueueResponse, error) {

	req := models.EnqueueRequest{
		Type:     taskType,
		Payload:  payload,
		Priority: 5,
	}

	return c.EnqueueRequest(ctx, req)
}

///////////////////////////////////////////////////////////////
// ADVANCED ENQUEUE
///////////////////////////////////////////////////////////////

// EnqueueRequest submits a fully customized request.
//
// Supports:
//
// # Priority
//
// # Scheduled ETA
//
// Custom Payload
func (c *SDKClient) EnqueueRequest(
	ctx context.Context,
	req models.EnqueueRequest,
) (*models.EnqueueResponse, error) {

	if req.Priority == 0 {
		req.Priority = 5
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/tasks",
		bytes.NewReader(body),
	)

	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf(
			"enqueue failed (%d)",
			resp.StatusCode,
		)

	}

	var out models.EnqueueResponse

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {

		return nil, err

	}

	return &out, nil

}

///////////////////////////////////////////////////////////////
// PRIORITY TASK
///////////////////////////////////////////////////////////////

// EnqueuePriority creates a task
// with the given priority.
func (c *SDKClient) EnqueuePriority(
	ctx context.Context,
	taskType string,
	priority int,
	payload map[string]any,
) (*models.EnqueueResponse, error) {

	req := models.EnqueueRequest{
		Type:     taskType,
		Priority: priority,
		Payload:  payload,
	}

	return c.EnqueueRequest(ctx, req)

}

///////////////////////////////////////////////////////////////
// SCHEDULED TASK
///////////////////////////////////////////////////////////////

// EnqueueScheduled schedules
// a task for future execution.
func (c *SDKClient) EnqueueScheduled(
	ctx context.Context,
	taskType string,
	priority int,
	eta time.Time,
	payload map[string]any,
) (*models.EnqueueResponse, error) {

	req := models.EnqueueRequest{
		Type:     taskType,
		Priority: priority,
		Payload:  payload,
		ETA:      &eta,
	}

	return c.EnqueueRequest(ctx, req)

}

///////////////////////////////////////////////////////////////
// STATUS
///////////////////////////////////////////////////////////////

func (c *SDKClient) Status(
	ctx context.Context,
	id string,
) (*models.Task, error) {

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.BaseURL+"/tasks/"+id,
		nil,
	)

	if err != nil {

		return nil, err

	}

	resp, err := c.HTTP.Do(httpReq)

	if err != nil {

		return nil, err

	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {

		return nil, models.ErrTaskNotFound

	}

	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf(
			"status failed (%d)",
			resp.StatusCode,
		)

	}

	var out models.Task

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {

		return nil, err

	}

	return &out, nil

}

///////////////////////////////////////////////////////////////
// WAIT UNTIL FINISHED
///////////////////////////////////////////////////////////////

// Wait blocks until the task reaches
// success/failed/dead or timeout.
func (c *SDKClient) Wait(
	ctx context.Context,
	id string,
	timeout time.Duration,
) (*models.Task, error) {

	deadline := time.Now().Add(timeout)

	for {

		task, err := c.Status(ctx, id)

		if err != nil {

			return nil, err

		}

		switch task.Status {

		case models.StatusSuccess,
			models.StatusFailed,
			models.StatusDead:

			return task, nil

		}

		if time.Now().After(deadline) {

			return task,
				fmt.Errorf("timeout waiting for task")

		}

		time.Sleep(time.Second)

	}

}
