package worker

import (
	"sync"

	"github.com/ManogyaDahal/DistQ/pkg/task"
)

// Registry holds task handlers keyed by task type.
// mu protects handlers.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]task.Handler
}
