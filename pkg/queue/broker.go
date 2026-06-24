package queue

import (
	"context"

	"github.com/ManogyaDahal/DistQ/pkg/models"
)

// so in this broker.go i made basically defined a broker as an interface its basically a contract of behaviour
// what it basically does like anything struct or may be which have these 3 method it will be a broker and can do furhter task in it
// so the reason why i import ctx is basically it give function about cancelation and timestamps which mean when
// user goes away or cancel the app the system get notified that user left and if the user response is late like 5 sec or 3 sec then
// the system make it to status dead and dont wast resources further more
type Broker interface {
	Enqueue(ctx context.Context, task models.Task) (string, error)
	GetTask(ctx context.Context, id string) (*models.Task, error)
	ListPending(ctx context.Context, limit int) ([]models.Task, error)
}
