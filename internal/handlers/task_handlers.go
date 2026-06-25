package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/worker"
)

const (
	DemoPrintTask = "demo.print"
	DemoSleepTask = "demo.sleep"
	DemoFailTask  = "demo.fail"
)

type PrintPayload struct {
	Message string `json:"message"`
}

type SleepPayload struct {
	Message  string `json:"message"`
	Seconds  int    `json:"seconds"`
	Duration string `json:"duration"`
}

type FailPayload struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

func RegisterDemoHandlers(registry *worker.Registry, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	if err := registry.Register(DemoPrintTask, PrintHandler(logger)); err != nil {
		return err
	}

	if err := registry.Register(DemoSleepTask, SleepHandler(logger)); err != nil {
		return err
	}

	if err := registry.Register(DemoFailTask, FailHandler(logger)); err != nil {
		return err
	}

	return nil
}

func PrintHandler(logger *slog.Logger) func(ctx context.Context, payload json.RawMessage) error {
	return func(ctx context.Context, payload json.RawMessage) error {
		var data PrintPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return fmt.Errorf("demo print handler: decode payload: %w", err)
		}

		logger.Info(
			"demo print task executed",
			slog.String("message", data.Message),
		)

		return nil
	}
}

func SleepHandler(logger *slog.Logger) func(ctx context.Context, payload json.RawMessage) error {
	return func(ctx context.Context, payload json.RawMessage) error {
		var data SleepPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return fmt.Errorf("demo sleep handler: decode payload: %w", err)
		}

		delay := time.Second
		if strings.TrimSpace(data.Duration) != "" {
			parsed, err := time.ParseDuration(strings.TrimSpace(data.Duration))
			if err != nil {
				return fmt.Errorf("demo sleep handler: parse duration %q: %w", data.Duration, err)
			}
			delay = parsed
		} else if data.Seconds > 0 {
			delay = time.Duration(data.Seconds) * time.Second
		}

		logger.Info(
			"demo sleep task started",
			slog.String("message", data.Message),
			slog.Duration("duration", delay),
		)

		timer := time.NewTimer(delay)
		defer timer.Stop()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}

		logger.Info(
			"demo sleep task finished",
			slog.String("message", data.Message),
		)

		return nil
	}
}

func FailHandler(logger *slog.Logger) func(ctx context.Context, payload json.RawMessage) error {
	return func(ctx context.Context, payload json.RawMessage) error {
		var data FailPayload
		if err := json.Unmarshal(payload, &data); err != nil {
			return fmt.Errorf("demo fail handler: decode payload: %w", err)
		}

		message := strings.TrimSpace(data.Error)
		if message == "" {
			message = strings.TrimSpace(data.Message)
		}
		if message == "" {
			message = "intentional demo failure"
		}

		logger.Warn(
			"demo fail task intentionally failed",
			slog.String("message", message),
		)

		return fmt.Errorf("demo fail handler: %s", message)
	}
}
