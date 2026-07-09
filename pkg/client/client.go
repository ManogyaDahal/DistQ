package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a wrapper around the DistQ HTTP API.
// It allows external applications to submit tasks, query task status,
// and retrieve system metrics without having to construct HTTP requests manually.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// ClientOption allows customizing the Client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client (useful for custom timeouts or TLS).
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// New creates a new DistQ Client pointing to the given baseURL (e.g. "http://localhost:8080").
func New(baseURL string, opts ...ClientOption) *Client {
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SubmitTaskRequest represents the payload required to submit a task.
type SubmitTaskRequest struct {
	Type       string          `json:"type"`
	Payload    json.RawMessage `json:"payload"`
	Priority   int             `json:"priority,omitempty"`
	MaxRetries *int            `json:"max_retries,omitempty"`
	ETA        *time.Time      `json:"eta,omitempty"`
	CronExpr   string          `json:"cron_expr,omitempty"`
}

// SubmitTaskResponse is the JSON response from the POST /api/task endpoint.
type SubmitTaskResponse struct {
	ID       string     `json:"id"`
	Kind     string     `json:"kind"`
	Status   string     `json:"status"`
	Priority int        `json:"priority"`
	Queue    string     `json:"queue,omitempty"`
	ETA      *time.Time `json:"eta,omitempty"`
	CronExpr string     `json:"cron_expr,omitempty"`
}

// TaskStatus represents the current state of a task as fetched from GET /api/task/{id}.
type TaskStatus struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Payload    any        `json:"payload"`
	Priority   int        `json:"priority"`
	Status     string     `json:"status"`
	MaxRetries int        `json:"max_retries"`
	RetryCount int        `json:"retry_count"`
	WorkerID   string     `json:"worker_id,omitempty"`
	Queue      string     `json:"queue,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ErrorMsg   string     `json:"error_msg,omitempty"`
	ETA        *time.Time `json:"eta,omitempty"`
	CronExpr   string     `json:"cron_expr,omitempty"`
}

// UnmarshalJSON accepts both the worker-native JSON shape used by /api/task
// (CreatedAt, RetryCount, etc.) and the API-facing snake_case metadata shape.
func (t *TaskStatus) UnmarshalJSON(data []byte) error {
	type taskStatusAlias TaskStatus
	var base taskStatusAlias
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}

	*t = TaskStatus(base)

	var camel struct {
		MaxRetriesCamel *int       `json:"MaxRetries"`
		RetryCountCamel *int       `json:"RetryCount"`
		WorkerIDCamel   string     `json:"WorkerID"`
		CreatedAtCamel  *time.Time `json:"CreatedAt"`
		UpdatedAtCamel  *time.Time `json:"UpdatedAt"`
		ErrorMsgCamel   string     `json:"ErrorMsg"`
		CronExprCamel   string     `json:"CronExpr"`
	}
	if err := json.Unmarshal(data, &camel); err != nil {
		return err
	}

	if camel.MaxRetriesCamel != nil {
		t.MaxRetries = *camel.MaxRetriesCamel
	}
	if camel.RetryCountCamel != nil {
		t.RetryCount = *camel.RetryCountCamel
	}
	if camel.WorkerIDCamel != "" {
		t.WorkerID = camel.WorkerIDCamel
	}
	if camel.CreatedAtCamel != nil {
		t.CreatedAt = *camel.CreatedAtCamel
	}
	if camel.UpdatedAtCamel != nil {
		t.UpdatedAt = *camel.UpdatedAtCamel
	}
	if camel.ErrorMsgCamel != "" {
		t.ErrorMsg = camel.ErrorMsgCamel
	}
	if camel.CronExprCamel != "" {
		t.CronExpr = camel.CronExprCamel
	}

	return nil
}

// SubmitTask sends a new task to the DistQ API.
func (c *Client) SubmitTask(ctx context.Context, req SubmitTaskRequest) (*SubmitTaskResponse, error) {
	url := fmt.Sprintf("%s/api/task", c.baseURL)

	bodyData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("distq client: failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("distq client: failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("distq client: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var result SubmitTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("distq client: failed to decode response: %w", err)
	}

	return &result, nil
}

// GetTask fetches the status of an existing task by its ID.
func (c *Client) GetTask(ctx context.Context, taskID string) (*TaskStatus, error) {
	url := fmt.Sprintf("%s/api/task/%s", c.baseURL, taskID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("distq client: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("distq client: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp)
	}

	var result TaskStatus
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("distq client: failed to decode response: %w", err)
	}

	return &result, nil
}

// helper to parse error messages from the DistQ API
func parseErrorResponse(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   string `json:"error"`
		Details string `json:"details,omitempty"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		if errResp.Details != "" {
			return fmt.Errorf("distq api error (HTTP %d): %s - %s", resp.StatusCode, errResp.Error, errResp.Details)
		}
		return fmt.Errorf("distq api error (HTTP %d): %s", resp.StatusCode, errResp.Error)
	}
	return fmt.Errorf("distq api error: HTTP %d - %s", resp.StatusCode, string(body))
}

func (c *Client) getJSON(ctx context.Context, endpoint string, result any) error {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("distq client: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("distq client: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseErrorResponse(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("distq client: failed to decode response: %w", err)
	}

	return nil
}

func (c *Client) postJSON(ctx context.Context, endpoint string, result any) error {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("distq client: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("distq client: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return parseErrorResponse(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("distq client: failed to decode response: %w", err)
	}

	return nil
}

type MetricsResponse struct {
	Metrics     map[string]int64 `json:"metrics"`
	QueueDepths map[string]int64 `json:"queue_depths"`
	Timestamp   int64            `json:"timestamp"`
}

type WorkerStatus struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	LastSeen     int64  `json:"last_seen"`
	OngoingTasks int64  `json:"ongoing_tasks"`
}

// GetMetrics returns general system metrics and queue depths.
func (c *Client) GetMetrics(ctx context.Context) (*MetricsResponse, error) {
	var res MetricsResponse
	err := c.getJSON(ctx, "/api/metrics", &res)
	return &res, err
}

// GetWorkers returns the list of registered workers and their status.
func (c *Client) GetWorkers(ctx context.Context) ([]WorkerStatus, error) {
	var res []WorkerStatus
	err := c.getJSON(ctx, "/api/workers", &res)
	return res, err
}

// GetScheduled returns all scheduled (ETA) tasks.
func (c *Client) GetScheduled(ctx context.Context) ([]map[string]any, error) {
	var res []map[string]any
	err := c.getJSON(ctx, "/api/scheduled", &res)
	return res, err
}

// GetCron returns all registered cron jobs.
func (c *Client) GetCron(ctx context.Context) ([]map[string]any, error) {
	var res []map[string]any
	err := c.getJSON(ctx, "/api/cron", &res)
	return res, err
}

// GetOngoing returns detailed information about tasks currently being processed.
func (c *Client) GetOngoing(ctx context.Context) ([]map[string]any, error) {
	var res []map[string]any
	err := c.getJSON(ctx, "/api/ongoing", &res)
	return res, err
}

// GetDLQ returns the dead-letter queue tasks.
func (c *Client) GetDLQ(ctx context.Context) ([]map[string]any, error) {
	var res []map[string]any
	err := c.getJSON(ctx, "/api/dlq", &res)
	return res, err
}

// RetryDLQ triggers a retry of all tasks currently in the DLQ.
func (c *Client) RetryDLQ(ctx context.Context) (map[string]any, error) {
	var res map[string]any
	err := c.postJSON(ctx, "/api/dlq/retry", &res)
	return res, err
}
