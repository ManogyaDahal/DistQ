package api

import (
	"context"
	"sync"
	"time"

	"github.com/ManogyaDahal/DistQ/pkg/models"
	"github.com/ManogyaDahal/DistQ/pkg/queue"
)

type StubBroker struct {
	mu    sync.RWMutex
	tasks map[string]models.Task
	order []string
}

func NewStubBroker() queue.Broker {
	return &StubBroker{
		tasks: make(map[string]models.Task),
		order: make([]string, 0),
	}
}

func (b *StubBroker) Enqueue(ctx context.Context, task models.Task) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if task.ID == "" {
		task.ID = time.Now().Format("20060102150405")
	}
	task.Status = models.StatusPending
	task.CreatedAt = time.Now()

	b.tasks[task.ID] = task
	b.order = append(b.order, task.ID)

	return task.ID, nil
}

func (b *StubBroker) GetTask(ctx context.Context, id string) (*models.Task, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	t, ok := b.tasks[id]
	if !ok {
		return nil, models.ErrTaskNotFound
	}
	return &t, nil
}

func (b *StubBroker) ListPending(ctx context.Context, limit int) ([]models.Task, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	out := make([]models.Task, 0, limit)
	for _, id := range b.order {
		if len(out) >= limit {
			break
		}
		t := b.tasks[id]
		if t.Status == models.StatusPending {
			out = append(out, t)
		}
	}
	return out, nil
}
