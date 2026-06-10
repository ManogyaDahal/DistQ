package worker

import (
	"sync"
	"fmt"
	"errors"
	"sort"
	"strings"

	"github.com/ManogyaDahal/DistQ/pkg/task"
)

var (
	ErrEmptyTaskType = errors.New("worker: task type is required")
	ErrNilHandler = errors.New("worker: handler is nil")
)

// Registry holds task handlers keyed by task type.
// mu protects handlers.
type Registry struct {
	mu sync.RWMutex
	handlers map[string]task.Handler
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]task.Handler),
	}
}

func (r *Registry) Register(taskType string, h task.Handler) error {
	taskType = strings.TrimSpace(taskType)
	if taskType == "" {
		return ErrEmptyTaskType
	}
	if h == nil {
		return ErrNilHandler
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[taskType] = h
	return nil
}

func (r *Registry) Get(taskType string) (task.Handler, bool) {

	taskType = strings.TrimSpace(taskType)

	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[taskType]
	return h, ok
}

func (r *Registry) MustGet(taskType string) (task.Handler, error) {
	h, ok := r.Get(taskType)

	if !ok {
		return nil, &ErrUnknownTaskType{TaskType: taskType}
	}

	return h, nil
}

func (r *Registry) Unregister(taskType string) {
	taskType = strings.TrimSpace(taskType)

	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, taskType)
}

func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0 , len(r.handlers))
	for taskType := range r.handlers {
		types = append(types, taskType)
	}
	
	sort.Strings(types)
	return types
}

type ErrUnknownTaskType struct {
	TaskType string
}

func (e *ErrUnknownTaskType) Error() string {
	return fmt.Sprintf("worker: no handler registered for task type %q", e.TaskType)
}
