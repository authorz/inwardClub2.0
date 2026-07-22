package outbox

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

// memRow is one persisted outbox_events row for the in-memory Store fake. The
// fake mirrors SQLStore.Dispatch's contract (claim due+pending rows, apply the
// Result, always bump attempts) so the Dispatcher's decisions are tested without
// a live MySQL, exactly as the payment settlement spine does for its repository.
type memRow struct {
	ev           Event
	status       string
	dispatchedAt time.Time
}

type memStore struct{ rows []*memRow }

func (s *memStore) add(id int64, topic, idem string, attempts int, availableAt time.Time) *memRow {
	r := &memRow{
		ev:     Event{ID: id, Topic: topic, IdemKey: idem, Attempts: attempts, AvailableAt: availableAt, Payload: []byte(`{}`)},
		status: StatusPending,
	}
	s.rows = append(s.rows, r)
	return r
}

func (s *memStore) Dispatch(_ context.Context, limit int, now time.Time, handle func(Event) Result) (int, error) {
	processed := 0
	for _, r := range s.rows {
		if processed >= limit {
			break
		}
		if r.status != StatusPending || r.ev.AvailableAt.After(now) {
			continue
		}
		res := handle(r.ev)
		r.ev.Attempts++
		switch res.Status {
		case StatusDispatched:
			r.status = StatusDispatched
			r.dispatchedAt = now
		case StatusFailed:
			r.status = StatusFailed
			r.ev.AvailableAt = res.AvailableAt
		case StatusPending:
			r.ev.AvailableAt = res.AvailableAt
		}
		processed++
	}
	return processed, nil
}

// recordingEnqueuer captures every enqueued event and can be told to fail for a
// given topic.
type recordingEnqueuer struct {
	got  []Event
	fail map[string]error
}

func (e *recordingEnqueuer) Enqueue(_ context.Context, ev Event) error {
	e.got = append(e.got, ev)
	if err := e.fail[ev.Topic]; err != nil {
		return err
	}
	return nil
}

func testDispatcher(store Store, enq Enqueuer) *Dispatcher {
	d := NewDispatcher(store, enq, slog.New(slog.NewTextHandler(io.Discard, nil)))
	d.batchSize = 2
	d.maxAttempts = 3
	d.baseBackoff = 10 * time.Second
	d.maxBackoff = time.Minute
	return d
}

func TestRunOnceDispatchesDuePendingEvents(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	store := &memStore{}
	store.add(1, "payment:post-process", "payment:1:post-process", 0, now.Add(-time.Second))
	store.add(2, "print:receipt", "", 0, now)

	enq := &recordingEnqueuer{}
	processed, dispatched, err := testDispatcher(store, enq).RunOnce(context.Background(), now)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if processed != 2 || dispatched != 2 {
		t.Fatalf("processed=%d dispatched=%d, want 2/2", processed, dispatched)
	}
	if len(enq.got) != 2 {
		t.Fatalf("enqueued %d events, want 2", len(enq.got))
	}
	// Topic, payload and idem key must reach the enqueuer verbatim.
	if enq.got[0].Topic != "payment:post-process" || enq.got[0].IdemKey != "payment:1:post-process" {
		t.Fatalf("event 0 not passed through: %+v", enq.got[0])
	}
	if string(enq.got[0].Payload) != `{}` {
		t.Fatalf("payload not passed through: %s", enq.got[0].Payload)
	}
	for _, r := range store.rows {
		if r.status != StatusDispatched {
			t.Fatalf("row %d status=%q, want dispatched", r.ev.ID, r.status)
		}
		if r.dispatchedAt != now {
			t.Fatalf("row %d dispatched_at=%v, want %v", r.ev.ID, r.dispatchedAt, now)
		}
	}
}

func TestRunOnceSkipsEventsNotYetDue(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	store := &memStore{}
	store.add(1, "print:receipt", "", 0, now.Add(time.Hour)) // scheduled in the future

	enq := &recordingEnqueuer{}
	processed, dispatched, err := testDispatcher(store, enq).RunOnce(context.Background(), now)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if processed != 0 || dispatched != 0 {
		t.Fatalf("processed=%d dispatched=%d, want 0/0", processed, dispatched)
	}
	if len(enq.got) != 0 {
		t.Fatalf("enqueued %d events, want 0", len(enq.got))
	}
	if store.rows[0].status != StatusPending {
		t.Fatalf("future row status=%q, want pending", store.rows[0].status)
	}
}

func TestRunOnceRetriesWithBackoffOnEnqueueError(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	store := &memStore{}
	row := store.add(1, "print:receipt", "", 0, now)

	enq := &recordingEnqueuer{fail: map[string]error{"print:receipt": errors.New("redis down")}}
	d := testDispatcher(store, enq)
	processed, dispatched, err := d.RunOnce(context.Background(), now)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if processed != 1 || dispatched != 0 {
		t.Fatalf("processed=%d dispatched=%d, want 1/0", processed, dispatched)
	}
	if row.status != StatusPending {
		t.Fatalf("status=%q, want pending (retry)", row.status)
	}
	if row.ev.Attempts != 1 {
		t.Fatalf("attempts=%d, want 1", row.ev.Attempts)
	}
	// attempt 1 → baseBackoff (10s), so the row is deferred to now+10s.
	if want := now.Add(10 * time.Second); !row.ev.AvailableAt.Equal(want) {
		t.Fatalf("available_at=%v, want %v", row.ev.AvailableAt, want)
	}
}

func TestRunOnceFailsAfterMaxAttempts(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	store := &memStore{}
	// maxAttempts is 3; this row has already failed twice, so the next failure is
	// terminal.
	row := store.add(1, "print:receipt", "", 2, now)

	enq := &recordingEnqueuer{fail: map[string]error{"print:receipt": errors.New("still down")}}
	d := testDispatcher(store, enq)
	if _, _, err := d.RunOnce(context.Background(), now); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if row.status != StatusFailed {
		t.Fatalf("status=%q, want failed", row.status)
	}
	if row.ev.Attempts != 3 {
		t.Fatalf("attempts=%d, want 3", row.ev.Attempts)
	}
}

func TestDrainClearsBacklogAcrossBatches(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	store := &memStore{}
	for i := int64(1); i <= 5; i++ { // batchSize is 2 → needs 3 batches
		store.add(i, "print:receipt", "", 0, now)
	}
	enq := &recordingEnqueuer{}
	d := testDispatcher(store, enq)
	d.now = func() time.Time { return now }

	d.drain(context.Background())

	if len(enq.got) != 5 {
		t.Fatalf("enqueued %d events, want 5", len(enq.got))
	}
	for _, r := range store.rows {
		if r.status != StatusDispatched {
			t.Fatalf("row %d status=%q, want dispatched", r.ev.ID, r.status)
		}
	}
}
