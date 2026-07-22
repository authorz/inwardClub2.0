package outbox

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hibiken/asynq"
)

// TestClassifyEnqueueErr pins the AsynqEnqueuer's idempotency contract: asynq's
// "task already enqueued" signals (a redelivered dispatch colliding with the
// in-flight task by TaskID) collapse to success so the outbox row settles as
// dispatched, while any other enqueue failure propagates for the dispatcher to
// retry. This is the near-once guarantee documented in docs/outbox-dispatch.md §3.
func TestClassifyEnqueueErr(t *testing.T) {
	tests := []struct {
		name    string
		in      error
		wantNil bool
	}{
		{"success", nil, true},
		{"task id conflict is idempotent success", asynq.ErrTaskIDConflict, true},
		{"duplicate task is idempotent success", asynq.ErrDuplicateTask, true},
		{"wrapped conflict still collapses", fmt.Errorf("enqueue: %w", asynq.ErrTaskIDConflict), true},
		{"transport error propagates", errors.New("redis down"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyEnqueueErr(tt.in)
			if tt.wantNil && got != nil {
				t.Fatalf("classifyEnqueueErr(%v) = %v, want nil", tt.in, got)
			}
			if !tt.wantNil && got == nil {
				t.Fatalf("classifyEnqueueErr(%v) = nil, want error", tt.in)
			}
		})
	}
}
