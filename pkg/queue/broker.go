// pkg/queue/broker.go
package queue

import (
	"context"

	"github.com/ManogyaDahal/DistQ/pkg/models"
)

type Broker interface {
	Enqueue(ctx context.Context, task models.Task) (string, error)
	GetTask(ctx context.Context, id string) (*models.Task, error)
	ListPending(ctx context.Context, limit int) ([]models.Task, error)
}
