package outbox

import (
	"context"
	"errors"

	"github.com/hibiken/asynq"
)

// AsynqEnqueuer enqueues outbox events as asynq tasks. The event's topic is the
// asynq task type, so a topic must match a task type registered by the worker
// (see cmd/worker). The event payload is carried verbatim as the task payload.
type AsynqEnqueuer struct {
	client   *asynq.Client
	maxRetry int
}

// NewAsynqEnqueuer wraps an asynq client. maxRetry is asynq's per-task retry
// budget once the task is running (independent of the outbox's own dispatch
// retries, which only cover getting the task into the queue).
func NewAsynqEnqueuer(client *asynq.Client, maxRetry int) *AsynqEnqueuer {
	return &AsynqEnqueuer{client: client, maxRetry: maxRetry}
}

// Enqueue submits the event. When the event carries an idempotency key it is used
// as the asynq task ID, so a redelivered dispatch (e.g. after a crash between
// enqueue and commit) collides with the in-flight task instead of duplicating
// it; that collision is reported back as success so the outbox row still settles.
func (e *AsynqEnqueuer) Enqueue(ctx context.Context, ev Event) error {
	opts := []asynq.Option{asynq.MaxRetry(e.maxRetry)}
	if ev.IdemKey != "" {
		opts = append(opts, asynq.TaskID(ev.IdemKey))
	}
	_, err := e.client.EnqueueContext(ctx, asynq.NewTask(ev.Topic, ev.Payload), opts...)
	return classifyEnqueueErr(err)
}

// classifyEnqueueErr collapses asynq's "already enqueued" signals to success: a
// redelivered dispatch (e.g. a crash between enqueue and the outbox commit) finds
// the prior same-ID task still in the queue and must settle the row rather than
// retry forever. Any other error is returned so the dispatcher retries the enqueue.
func classifyEnqueueErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, asynq.ErrTaskIDConflict), errors.Is(err, asynq.ErrDuplicateTask):
		return nil
	default:
		return err
	}
}
