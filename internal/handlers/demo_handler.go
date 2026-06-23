package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	Message string `json:"message"`
	Seconds int    `json:"seconds"`
}

type FailPayload struct {
	Message string `json:"message"`
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

		if data.Seconds <= 0 {
			data.Seconds = 1
		}

		logger.Info(
			"demo sleep task started",
			slog.String("message", data.Message),
			slog.Int("seconds", data.Seconds),
		)

		timer := time.NewTimer(time.Duration(data.Seconds) * time.Second)
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

		if data.Message == "" {
			data.Message = "intentional demo failure"
		}

		logger.Warn(
			"demo fail task intentionally failed",
			slog.String("message", data.Message),
		)

		return fmt.Errorf("demo fail handler: %s", data.Message)
	}
}
