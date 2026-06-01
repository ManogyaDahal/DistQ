package worker

import (
	"sync"
	"fmt"

	"github.com/ManogyaDahal/DistQ/pkg/task"
)

// Registry holds task handlers keyed by task type.
// mu protects handlers.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]task.Handler
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]task.Handler),
	}
}

func (r *Registry) Register(taskType string, h task.Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[taskType] = h
}

func (r *Registry) Get(taskType string) (task.Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[taskType]
	return h, ok
}

type ErrUnknownTaskType struct {
	TaskType string
}

func (e *ErrUnknownTaskType) Error() string {
	return fmt.Sprintf("worker: no handler registered for task type %q", e.TaskType)
}
