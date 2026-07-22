// Package printer abstracts cloud receipt printing (Xpyun). Business code enqueues
// print jobs through the outbox; the worker calls Printer. Phase-1 ships the
// interface and a fake that records jobs in memory.
package printer

import (
	"context"
	"sync"
)

// Job is a single print request.
type Job struct {
	DeviceSN string
	Template string
	Content  string
}

// Printer abstracts a cloud printing provider.
type Printer interface {
	Print(ctx context.Context, job Job) error
}

// FakePrinter records print jobs in memory for dev/tests.
type FakePrinter struct {
	mu   sync.Mutex
	Jobs []Job
}

// NewFakePrinter builds the fake printer.
func NewFakePrinter() *FakePrinter { return &FakePrinter{} }

// Print records the job.
func (f *FakePrinter) Print(_ context.Context, job Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Jobs = append(f.Jobs, job)
	return nil
}

// Count returns the number of recorded jobs (test helper).
func (f *FakePrinter) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Jobs)
}
