// Package printer abstracts cloud receipt printing (Xpyun). Business code enqueues
// print jobs through the outbox; the worker calls Printer. Phase-1 ships the
// interface and a fake that records jobs in memory.
package printer

import (
	"context"
	"sync"
	"time"
)

// Job is a single print request.
type Job struct {
	ID       int64
	DeviceSN string
	Template string
	Content  string
}

// Printer abstracts a cloud printing provider.
type Printer interface {
	Print(ctx context.Context, job Job) error
}

// ProviderStatus is the live connectivity state reported by Xpyun.
type ProviderStatus string

const (
	ProviderStatusOffline      ProviderStatus = "offline"
	ProviderStatusOnline       ProviderStatus = "online"
	ProviderStatusAbnormal     ProviderStatus = "abnormal"
	ProviderStatusUnknown      ProviderStatus = "unknown"
	ProviderStatusUnconfigured ProviderStatus = "unconfigured"
)

// OrderStatistics is Xpyun's printed/waiting count for one device and day.
type OrderStatistics struct {
	Printed int `json:"printed"`
	Waiting int `json:"waiting"`
}

// CloudPrinter covers the Xpyun operations used by printing and console device
// management. Device registration and removal are deliberately provider-first:
// the service persists local state only after these calls succeed.
type CloudPrinter interface {
	Printer
	AddPrinter(ctx context.Context, sn, name string) error
	DeletePrinter(ctx context.Context, sn string) error
	UpdatePrinterName(ctx context.Context, sn, name string) error
	ClearQueue(ctx context.Context, sn string) error
	SetVoice(ctx context.Context, sn string, voiceType int, volumeLevel *int) error
	QueryOrderState(ctx context.Context, orderID string) (bool, error)
	QueryOrderStatistics(ctx context.Context, sn string, date time.Time) (OrderStatistics, error)
	QueryStatuses(ctx context.Context, sns []string) (map[string]ProviderStatus, error)
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

func (f *FakePrinter) AddPrinter(context.Context, string, string) error { return nil }
func (f *FakePrinter) DeletePrinter(context.Context, string) error      { return nil }
func (f *FakePrinter) UpdatePrinterName(context.Context, string, string) error {
	return nil
}
func (f *FakePrinter) ClearQueue(context.Context, string) error { return nil }
func (f *FakePrinter) SetVoice(context.Context, string, int, *int) error {
	return nil
}
func (f *FakePrinter) QueryOrderState(context.Context, string) (bool, error) { return true, nil }
func (f *FakePrinter) QueryOrderStatistics(context.Context, string, time.Time) (OrderStatistics, error) {
	return OrderStatistics{}, nil
}
func (f *FakePrinter) QueryStatuses(_ context.Context, sns []string) (map[string]ProviderStatus, error) {
	statuses := make(map[string]ProviderStatus, len(sns))
	for _, sn := range sns {
		statuses[sn] = ProviderStatusOnline
	}
	return statuses, nil
}

// Count returns the number of recorded jobs (test helper).
func (f *FakePrinter) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Jobs)
}

var _ CloudPrinter = (*FakePrinter)(nil)
