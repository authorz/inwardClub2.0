package outbox

import (
	"context"
	"log/slog"
	"time"
)

// Dispatcher drains persisted outbox_events into the asynq queue. It is the
// relay half of the transactional outbox: business writes append pending events
// via Write inside their transaction; the Dispatcher polls the committed rows
// and enqueues each one as an asynq task keyed by its topic. Running it in the
// worker process is what turns a stored event into an actually-executed effect.
type Dispatcher struct {
	store Store
	enq   Enqueuer
	log   *slog.Logger

	batchSize   int
	maxAttempts int
	baseBackoff time.Duration
	maxBackoff  time.Duration
	interval    time.Duration
	now         func() time.Time
}

// Store claims a batch of due, pending events and applies each event's Result
// atomically with the claim, so a crash mid-batch leaves the untouched rows
// pending for redelivery. Implementations must claim rows in a way that is safe
// for concurrent dispatchers (e.g. SELECT ... FOR UPDATE SKIP LOCKED). It
// returns the number of events processed in the batch.
type Store interface {
	Dispatch(ctx context.Context, limit int, now time.Time, handle func(Event) Result) (int, error)
}

// Enqueuer hands a single event to the async task backend. A nil error means the
// task is in the queue (or was already there); implementations should treat a
// duplicate/already-enqueued signal as success so redelivery stays idempotent.
type Enqueuer interface {
	Enqueue(ctx context.Context, ev Event) error
}

// Result is the terminal state the Dispatcher wants written back for a claimed
// event. Status is one of the outbox status constants; AvailableAt and LastError
// are honoured only for the pending (retry) and failed transitions.
type Result struct {
	Status      string
	AvailableAt time.Time
	LastError   string
}

// Dispatcher tuning defaults. These are deliberately conservative: the relay
// holds the claim's row locks while it enqueues, so a modest batch keeps each
// transaction short even when Redis is slow.
const (
	defaultBatchSize   = 100
	defaultMaxAttempts = 10
	defaultBaseBackoff = 5 * time.Second
	defaultMaxBackoff  = 5 * time.Minute
	defaultInterval    = 2 * time.Second

	// lastErrorMax matches outbox_events.last_error's VARCHAR(512).
	lastErrorMax = 512
)

// NewDispatcher builds a Dispatcher with production defaults. The worker owns the
// store (a *SQLStore), the enqueuer (a *AsynqEnqueuer) and the process lifetime.
func NewDispatcher(store Store, enq Enqueuer, log *slog.Logger) *Dispatcher {
	return &Dispatcher{
		store:       store,
		enq:         enq,
		log:         log,
		batchSize:   defaultBatchSize,
		maxAttempts: defaultMaxAttempts,
		baseBackoff: defaultBaseBackoff,
		maxBackoff:  defaultMaxBackoff,
		interval:    defaultInterval,
		now:         func() time.Time { return time.Now().UTC() },
	}
}

// Run polls until ctx is cancelled. It drains on startup and on every tick so a
// burst of events clears promptly instead of trickling out one interval apart.
func (d *Dispatcher) Run(ctx context.Context) {
	d.log.Info("outbox dispatcher started", "interval", d.interval.String(), "batch", d.batchSize)
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	d.drain(ctx)
	for {
		select {
		case <-ctx.Done():
			d.log.Info("outbox dispatcher stopped")
			return
		case <-ticker.C:
			d.drain(ctx)
		}
	}
}

// drain repeatedly dispatches full batches until the queue is empty (a batch
// came back smaller than the limit) or a cycle errors.
func (d *Dispatcher) drain(ctx context.Context) {
	for ctx.Err() == nil {
		processed, dispatched, err := d.RunOnce(ctx, d.now())
		if err != nil {
			d.log.Error("outbox dispatch cycle failed", "error", err)
			return
		}
		if dispatched > 0 {
			d.log.Info("outbox events dispatched", "count", dispatched)
		}
		if processed < d.batchSize {
			return
		}
	}
}

// RunOnce claims and dispatches a single batch of due events. It returns the
// number of events processed and the subset that were successfully enqueued.
func (d *Dispatcher) RunOnce(ctx context.Context, now time.Time) (processed, dispatched int, err error) {
	processed, err = d.store.Dispatch(ctx, d.batchSize, now, func(ev Event) Result {
		switch e := d.enq.Enqueue(ctx, ev); {
		case e == nil:
			dispatched++
			return Result{Status: StatusDispatched}
		default:
			attempts := ev.Attempts + 1
			if attempts >= d.maxAttempts {
				d.log.Error("outbox event exhausted retries",
					"id", ev.ID, "topic", ev.Topic, "attempts", attempts, "error", e)
				return Result{Status: StatusFailed, LastError: truncateError(e)}
			}
			d.log.Warn("outbox enqueue failed; will retry",
				"id", ev.ID, "topic", ev.Topic, "attempts", attempts, "error", e)
			return Result{
				Status:      StatusPending,
				AvailableAt: now.Add(d.backoff(attempts)),
				LastError:   truncateError(e),
			}
		}
	})
	return processed, dispatched, err
}

// backoff grows exponentially with the attempt count and is capped at maxBackoff.
func (d *Dispatcher) backoff(attempts int) time.Duration {
	wait := d.baseBackoff
	for i := 1; i < attempts && wait < d.maxBackoff; i++ {
		wait *= 2
	}
	if wait > d.maxBackoff {
		return d.maxBackoff
	}
	return wait
}

func truncateError(err error) string {
	s := err.Error()
	if len(s) > lastErrorMax {
		return s[:lastErrorMax]
	}
	return s
}
