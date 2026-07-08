package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/worker"
)

const (
	// WebhookTask is the built-in task type that POSTs a payload to a client-supplied URL.
	// Clients never need to modify the worker binary — they just host an HTTP endpoint.
	WebhookTask = "webhook"

	// webhookDefaultMethod is the HTTP method used when the client does not specify one.
	webhookDefaultMethod = http.MethodPost

	// webhookMaxTimeout is the maximum per-task timeout a client may request.
	webhookMaxTimeout = 60 * time.Second

	// webhookDefaultTimeout is used when the client does not specify a timeout.
	webhookDefaultTimeout = 30 * time.Second
)

// WebhookPayload is the JSON structure a client must place in the task's Payload field
// when submitting a task of type "webhook".
//
// Example (sent via SDK or raw curl):
//
//	{
//	  "url":             "https://your-app.com/tasks/callback",
//	  "method":          "POST",
//	  "headers":         { "X-Secret": "token123" },
//	  "body":            { "user_id": 42, "action": "send_email" },
//	  "timeout_seconds": 10
//	}
type WebhookPayload struct {
	// URL is the HTTP(S) endpoint DistQ will call. Required.
	URL string `json:"url"`

	// Method is the HTTP method to use (GET, POST, PUT, PATCH, DELETE).
	// Defaults to POST if omitted or empty.
	Method string `json:"method,omitempty"`

	// Headers contains any custom HTTP headers to include in the request.
	// A "Content-Type: application/json" and "User-Agent: DistQ-Webhook/1.0"
	// header are always added, but can be overridden here.
	Headers map[string]string `json:"headers,omitempty"`

	// Body is an arbitrary JSON value that will be forwarded as the request body.
	// Ignored for methods that do not carry a body (e.g. GET, HEAD).
	Body json.RawMessage `json:"body,omitempty"`

	// TimeoutSeconds caps how long DistQ will wait for the endpoint to respond.
	// Maximum is 60 s; defaults to 30 s when zero or negative.
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
}

// RegisterWebhookHandler registers the built-in "webhook" task type on the provided Registry.
// Call this once during worker startup, alongside RegisterDemoHandlers or any other handlers.
func RegisterWebhookHandler(registry *worker.Registry, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}
	return registry.Register(WebhookTask, webhookHandler(logger))
}

// webhookHandler returns the task.Handler function for the "webhook" task type.
func webhookHandler(logger *slog.Logger) func(ctx context.Context, payload json.RawMessage) error {
	return func(ctx context.Context, payload json.RawMessage) error {
		var wp WebhookPayload
		if err := json.Unmarshal(payload, &wp); err != nil {
			return fmt.Errorf("webhook handler: decode payload: %w", err)
		}

		// --- Validate URL ---
		if strings.TrimSpace(wp.URL) == "" {
			return fmt.Errorf("webhook handler: payload missing required field \"url\"")
		}
		parsed, err := url.ParseRequestURI(wp.URL)
		if err != nil {
			return fmt.Errorf("webhook handler: invalid url %q: %w", wp.URL, err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("webhook handler: url scheme must be http or https, got %q", parsed.Scheme)
		}

		// --- Resolve method ---
		method := strings.ToUpper(strings.TrimSpace(wp.Method))
		if method == "" {
			method = webhookDefaultMethod
		}

		// --- Build body ---
		var bodyReader *bytes.Reader
		hasBody := method != http.MethodGet && method != http.MethodHead && len(wp.Body) > 0
		if hasBody {
			bodyReader = bytes.NewReader(wp.Body)
		} else {
			bodyReader = bytes.NewReader(nil)
		}

		// --- Resolve timeout (per-task, capped at max) ---
		timeout := time.Duration(wp.TimeoutSeconds) * time.Second
		switch {
		case timeout <= 0:
			timeout = webhookDefaultTimeout
		case timeout > webhookMaxTimeout:
			timeout = webhookMaxTimeout
		}

		// Honour the parent context but also cap by the per-task timeout.
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// --- Build HTTP request ---
		req, err := http.NewRequestWithContext(callCtx, method, wp.URL, bodyReader)
		if err != nil {
			return fmt.Errorf("webhook handler: build request: %w", err)
		}

		// Default headers — client headers can override these.
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "DistQ-Webhook/1.0")
		for k, v := range wp.Headers {
			req.Header.Set(k, v)
		}

		// --- Execute ---
		start := time.Now()
		resp, err := http.DefaultClient.Do(req)
		elapsed := time.Since(start)

		if err != nil {
			return fmt.Errorf("webhook handler: call %s %s: %w", method, wp.URL, err)
		}
		defer resp.Body.Close()

		logger.Info(
			"webhook called",
			slog.String("url", wp.URL),
			slog.String("method", method),
			slog.Int("status", resp.StatusCode),
			slog.Duration("elapsed", elapsed),
		)

		// Treat any non-2xx as a failure — the retry/DLQ logic will handle the rest.
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf(
				"webhook handler: endpoint returned non-2xx status %d for %s %s",
				resp.StatusCode, method, wp.URL,
			)
		}

		return nil
	}
}
