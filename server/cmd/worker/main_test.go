package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/hibiken/asynq"

	"github.com/inwardclub/server/internal/modules/printer"
	"github.com/inwardclub/server/internal/modules/reporting"
)

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestPrintHandlerDispatchesJob(t *testing.T) {
	fake := printer.NewFakePrinter()
	handler := printHandler(quietLogger(), fake)

	payload, _ := json.Marshal(printer.Job{DeviceSN: "SN-9", Template: "order", Content: "hello"})
	task := asynq.NewTask(TaskPrint, payload)

	if err := handler(context.Background(), task); err != nil {
		t.Fatalf("printHandler: %v", err)
	}
	if fake.Count() != 1 {
		t.Fatalf("recorded jobs = %d, want 1", fake.Count())
	}
	if got := fake.Jobs[0]; got.DeviceSN != "SN-9" || got.Content != "hello" {
		t.Errorf("recorded job = %+v", got)
	}
}

func TestPrintHandlerDropsUndecodablePayload(t *testing.T) {
	fake := printer.NewFakePrinter()
	handler := printHandler(quietLogger(), fake)

	task := asynq.NewTask(TaskPrint, []byte("not json"))
	// A payload that can never decode is dropped (nil) so asynq does not retry it.
	if err := handler(context.Background(), task); err != nil {
		t.Fatalf("want nil for undecodable payload, got %v", err)
	}
	if fake.Count() != 0 {
		t.Errorf("recorded jobs = %d, want 0", fake.Count())
	}
}

type printRecorderStub struct {
	started []int64
	printed []int64
	failed  []int64
}

func (r *printRecorderStub) StartAttempt(_ context.Context, id int64) error {
	r.started = append(r.started, id)
	return nil
}

func (r *printRecorderStub) MarkPrinted(_ context.Context, id int64) error {
	r.printed = append(r.printed, id)
	return nil
}

func (r *printRecorderStub) MarkFailed(_ context.Context, id int64, _ error) error {
	r.failed = append(r.failed, id)
	return nil
}

type failingPrinter struct{ err error }

func (p failingPrinter) Print(context.Context, printer.Job) error { return p.err }

func TestPrintHandlerRecordsSuccessfulAttempt(t *testing.T) {
	fake := printer.NewFakePrinter()
	recorder := &printRecorderStub{}
	handler := printHandler(quietLogger(), fake, recorder)
	payload, _ := json.Marshal(printer.Job{ID: 42, DeviceSN: "SN-42", Content: "receipt"})

	if err := handler(context.Background(), asynq.NewTask(TaskPrint, payload)); err != nil {
		t.Fatalf("printHandler: %v", err)
	}
	if len(recorder.started) != 1 || recorder.started[0] != 42 {
		t.Fatalf("started = %v, want [42]", recorder.started)
	}
	if len(recorder.printed) != 1 || recorder.printed[0] != 42 {
		t.Fatalf("printed = %v, want [42]", recorder.printed)
	}
	if len(recorder.failed) != 0 {
		t.Fatalf("failed = %v, want none", recorder.failed)
	}
}

func TestPrintHandlerRecordsFailedAttempt(t *testing.T) {
	printErr := errors.New("provider unavailable")
	recorder := &printRecorderStub{}
	handler := printHandler(quietLogger(), failingPrinter{err: printErr}, recorder)
	payload, _ := json.Marshal(printer.Job{ID: 77, DeviceSN: "SN-77", Content: "receipt"})

	err := handler(context.Background(), asynq.NewTask(TaskPrint, payload))
	if !errors.Is(err, printErr) {
		t.Fatalf("error = %v, want %v", err, printErr)
	}
	if len(recorder.started) != 1 || recorder.started[0] != 77 {
		t.Fatalf("started = %v, want [77]", recorder.started)
	}
	if len(recorder.failed) != 1 || recorder.failed[0] != 77 {
		t.Fatalf("failed = %v, want [77]", recorder.failed)
	}
	if len(recorder.printed) != 0 {
		t.Fatalf("printed = %v, want none", recorder.printed)
	}
}

// stubRollupRepo is a reporting.RollupRepository that records the request and
// returns canned output, so the handler wiring can be exercised without a DB.
type stubRollupRepo struct {
	got  reporting.RollupRequest
	res  reporting.RollupResult
	err  error
	runs int
}

func (s *stubRollupRepo) RollupDaily(_ context.Context, req reporting.RollupRequest) (reporting.RollupResult, error) {
	s.got = req
	s.runs++
	return s.res, s.err
}

// TestParseRollupRequestEmpty: no payload means a full recompute (all bounds nil).
func TestParseRollupRequestEmpty(t *testing.T) {
	for _, payload := range [][]byte{nil, {}, []byte("{}")} {
		req, err := parseRollupRequest(payload)
		if err != nil {
			t.Fatalf("payload %q: %v", payload, err)
		}
		if req.From != nil || req.To != nil || req.StoreID != nil {
			t.Fatalf("payload %q: expected unbounded, got %+v", payload, req)
		}
	}
}

// TestParseRollupRequestDate: a single date pins From == To to that day.
func TestParseRollupRequestDate(t *testing.T) {
	req, err := parseRollupRequest([]byte(`{"date":"2026-07-17"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	if req.From == nil || !req.From.Equal(want) || req.To == nil || !req.To.Equal(want) {
		t.Fatalf("expected From==To==%v, got %+v", want, req)
	}
}

// TestParseRollupRequestRangeAndStore: from/to and storeId pin a scoped window.
func TestParseRollupRequestRangeAndStore(t *testing.T) {
	req, err := parseRollupRequest([]byte(`{"from":"2026-07-01","to":"2026-07-18","storeId":42}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	if req.From == nil || !req.From.Equal(from) || req.To == nil || !req.To.Equal(to) {
		t.Fatalf("window not parsed: %+v", req)
	}
	if req.StoreID == nil || *req.StoreID != 42 {
		t.Fatalf("store not parsed: %+v", req)
	}
}

// TestParseRollupRequestDateWins: an explicit date overrides any from/to.
func TestParseRollupRequestDateWins(t *testing.T) {
	req, err := parseRollupRequest([]byte(`{"date":"2026-07-17","from":"2026-01-01","to":"2026-12-31"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := time.Date(2026, 7, 17, 0, 0, 0, 0, time.UTC)
	if req.From == nil || !req.From.Equal(want) || req.To == nil || !req.To.Equal(want) {
		t.Fatalf("expected date to override range, got %+v", req)
	}
}

// TestParseRollupRequestInvalid: malformed date or json is a hard error so the
// worker drops the task instead of recomputing the wrong window.
func TestParseRollupRequestInvalid(t *testing.T) {
	if _, err := parseRollupRequest([]byte(`{"date":"07/17/2026"}`)); err == nil {
		t.Fatal("expected error for malformed date")
	}
	if _, err := parseRollupRequest([]byte(`{bad json`)); err == nil {
		t.Fatal("expected error for malformed json")
	}
}

// TestRollupHandlerFullRecompute: the scheduled/startup task (empty payload)
// drives an unbounded recompute through the pipeline.
func TestRollupHandlerFullRecompute(t *testing.T) {
	repo := &stubRollupRepo{res: reporting.RollupResult{RevenueRows: 2, ReservationRows: 1}}
	handler := rollupHandler(quietLogger(), reporting.NewRollupService(repo))

	if err := handler(context.Background(), asynq.NewTask(TaskReportRollup, nil)); err != nil {
		t.Fatalf("rollupHandler: %v", err)
	}
	if repo.runs != 1 {
		t.Fatalf("pipeline runs = %d, want 1", repo.runs)
	}
	if repo.got.From != nil || repo.got.To != nil || repo.got.StoreID != nil {
		t.Fatalf("expected full recompute, got %+v", repo.got)
	}
}

// TestRollupHandlerDropsBadPayload: an undecodable payload is dropped with
// asynq.SkipRetry (no retry) and the pipeline is never invoked.
func TestRollupHandlerDropsBadPayload(t *testing.T) {
	repo := &stubRollupRepo{}
	handler := rollupHandler(quietLogger(), reporting.NewRollupService(repo))

	err := handler(context.Background(), asynq.NewTask(TaskReportRollup, []byte("{bad")))
	if !errors.Is(err, asynq.SkipRetry) {
		t.Fatalf("expected SkipRetry, got %v", err)
	}
	if repo.runs != 0 {
		t.Fatalf("pipeline runs = %d, want 0", repo.runs)
	}
}

// TestRollupHandlerPropagatesError: a pipeline error is returned so asynq retries.
func TestRollupHandlerPropagatesError(t *testing.T) {
	repo := &stubRollupRepo{err: errors.New("db down")}
	handler := rollupHandler(quietLogger(), reporting.NewRollupService(repo))

	if err := handler(context.Background(), asynq.NewTask(TaskReportRollup, nil)); err == nil {
		t.Fatal("expected propagated error")
	}
}
