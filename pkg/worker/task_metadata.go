package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ManogyaDahal/DistQ/pkg/models"
	"github.com/ManogyaDahal/DistQ/pkg/redisclient"
	"github.com/ManogyaDahal/DistQ/pkg/task"
)

// updateTaskMetadata copies worker task state into the API metadata hash.
//
// It first reads the task originally saved by POST /tasks so API-only data
// such as CreatedAt and Payload are preserved.
func (a *RedisQueueAdapter) updateTaskMetadata(ctx context.Context, t *task.Task) error {
	if t == nil {
		return fmt.Errorf("adapter: cannot update metadata for nil task")
	}

	metaKey := fmt.Sprintf(redisclient.KeyTaskMeta, t.ID)

	// Read the original task metadata saved by TaskEnqueue.
	existingData, err := a.client.Redis.HGet(ctx, metaKey, "data").Result()
	if err != nil {
		return fmt.Errorf("adapter: read existing task metadata: %w", err)
	}

	var existing models.Task
	if err := json.Unmarshal([]byte(existingData), &existing); err != nil {
		return fmt.Errorf("adapter: decode existing task metadata: %w", err)
	}

	// Preserve the original creation time. Worker stream messages may not
	// include CreatedAt, so t.CreatedAt can be the zero time.
	createdAt := existing.CreatedAt
	if !t.CreatedAt.IsZero() {
		createdAt = t.CreatedAt
	}

	// Preserve the original payload from API metadata. The worker has a
	// json.RawMessage payload, while the API model uses map[string]any.
	apiTask := models.Task{
		ID:         t.ID,
		Type:       t.Type,
		Payload:    existing.Payload,
		Priority:   t.Priority,
		Status:     convertTaskStatus(t.Status),
		RetryCount: t.RetryCount,
		ETA:        t.ETA,
		CreatedAt:  createdAt,
		UpdatedAt:  t.UpdatedAt,
		WorkerID:   t.WorkerID,
		ErrorMsg:   t.ErrorMsg,
	}

	data, err := json.Marshal(apiTask)
	if err != nil {
		return fmt.Errorf("adapter: marshal task metadata: %w", err)
	}

	if err := a.client.Redis.HSet(ctx, metaKey, "data", string(data)).Err(); err != nil {
		return fmt.Errorf("adapter: update task metadata: %w", err)
	}

	return nil

}

// convertTaskStatus converts worker internal status to API-visible status.
func convertTaskStatus(status task.TaskStatus) models.TaskStatus {
	switch status {
	case task.StatusPending:
		return models.StatusPending
	case task.StatusClaimed, task.StatusRunning:
		return models.StatusRunning
	case task.StatusDone:
		return models.StatusSuccess
	case task.StatusFailed:
		return models.StatusFailed
	case task.StatusRetrying:
		return models.StatusScheduled
	case task.StatusDead:
		return models.StatusDead
	default:
		return models.StatusPending
	}
}
