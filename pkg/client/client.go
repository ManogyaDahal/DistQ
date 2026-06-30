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
	ID         string     `json:"ID"`
	Type       string     `json:"Type"`
	Payload    any        `json:"Payload"` // Leaving as any to allow reading general output if needed
	Priority   int        `json:"Priority"`
	Status     string     `json:"Status"`
	MaxRetries int        `json:"MaxRetries"`
	RetryCount int        `json:"RetryCount"`
	WorkerID   string     `json:"WorkerID,omitempty"`
	Queue      string     `json:"Queue,omitempty"`
	CreatedAt  time.Time  `json:"CreatedAt"`
	UpdatedAt  time.Time  `json:"UpdatedAt"`
	ErrorMsg   string     `json:"ErrorMsg,omitempty"`
	ETA        *time.Time `json:"ETA,omitempty"`
	CronExpr   string     `json:"CronExpr,omitempty"`
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
