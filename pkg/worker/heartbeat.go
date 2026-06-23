package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

const (
	DefaultHeartbeatInterval = 5 * time.Second
	DefaultHeartbeatTimeout  = 30 * time.Second
)

type HeartbeatStore interface {
	Beat(ctx context.Context, workerID string, at time.Time) error
	List(ctx context.Context) (map[string]time.Time, error)
	Remove(ctx context.Context, workerID string) error
}

type HeartbeatSender struct {
	workerID string
	store    HeartbeatStore
	interval time.Duration
	logger   *slog.Logger
}

type HeartbeatSenderOption func(*HeartbeatSender)

func WithHeartbeatSenderInterval(interval time.Duration) HeartbeatSenderOption {
	return func(s *HeartbeatSender) {
		if interval > 0 {
			s.interval = interval
		}
	}
}

func WithHeartbeatSenderLogger(logger *slog.Logger) HeartbeatSenderOption {
	return func(s *HeartbeatSender) {
		if logger != nil {
			s.logger = logger
		}
	}
}

func NewHeartbeatSender(workerID string, store HeartbeatStore, opts ...HeartbeatSenderOption) (*HeartbeatSender, error) {
	if workerID == "" {
		return nil, errors.New("worker: heartbeat worker ID is required")
	}
	if store == nil {
		return nil, errors.New("worker: heartbeat store is required")
	}

	s := &HeartbeatSender{
		workerID: workerID,
		store:    store,
		interval: DefaultHeartbeatInterval,
		logger:   slog.Default(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

func (s *HeartbeatSender) Run(ctx context.Context) error {
	logger := s.logger.With(
		slog.String("component", "heartbeat_sender"),
		slog.String("worker_id", s.workerID),
	)

	if err := s.beat(ctx, logger); err != nil {
		return err
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.beat(ctx, logger); err != nil {
				return err
			}
		}
	}
}

func (s *HeartbeatSender) beat(ctx context.Context, logger *slog.Logger) error {
	now := time.Now().UTC()
	if err := s.store.Beat(ctx, s.workerID, now); err != nil {
		return fmt.Errorf("worker: send heartbeat for %q: %w", s.workerID, err)
	}

	logger.Debug("heartbeat sent", slog.Time("at", now))
	return nil
}

type HeartbeatMonitor struct {
	store    HeartbeatStore
	timeout  time.Duration
	interval time.Duration
	logger   *slog.Logger
}

type HeartbeatMonitorOption func(*HeartbeatMonitor)

func WithHeartbeatMonitorTimeout(timeout time.Duration) HeartbeatMonitorOption {
	return func(m *HeartbeatMonitor) {
		if timeout > 0 {
			m.timeout = timeout
		}
	}
}

func WithHeartbeatMonitorInterval(interval time.Duration) HeartbeatMonitorOption {
	return func(m *HeartbeatMonitor) {
		if interval > 0 {
			m.interval = interval
		}
	}
}

func WithHeartbeatMonitorLogger(logger *slog.Logger) HeartbeatMonitorOption {
	return func(m *HeartbeatMonitor) {
		if logger != nil {
			m.logger = logger
		}
	}
}

func NewHeartbeatMonitor(store HeartbeatStore, opts ...HeartbeatMonitorOption) (*HeartbeatMonitor, error) {
	if store == nil {
		return nil, errors.New("worker: heartbeat store is required")
	}

	m := &HeartbeatMonitor{
		store:    store,
		timeout:  DefaultHeartbeatTimeout,
		interval: DefaultHeartbeatInterval,
		logger:   slog.Default(),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

func (m *HeartbeatMonitor) Run(ctx context.Context) error {
	if err := m.Check(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := m.Check(ctx); err != nil {
				return err
			}
		}
	}
}

func (m *HeartbeatMonitor) Check(ctx context.Context) error {
	workers, err := m.store.List(ctx)
	if err != nil {
		return fmt.Errorf("worker: list heartbeats: %w", err)
	}

	now := time.Now().UTC()
	for workerID, lastBeat := range workers {
		if now.Sub(lastBeat) <= m.timeout {
			continue
		}

		if err := m.store.Remove(ctx, workerID); err != nil {
			return fmt.Errorf("worker: remove expired heartbeat for %q: %w", workerID, err)
		}

		m.logger.Warn(
			"worker heartbeat expired",
			slog.String("component", "heartbeat_monitor"),
			slog.String("worker_id", workerID),
			slog.Time("last_beat", lastBeat),
			slog.Duration("timeout", m.timeout),
		)
	}

	return nil
}
