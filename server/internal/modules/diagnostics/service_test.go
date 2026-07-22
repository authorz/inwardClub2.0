package diagnostics

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	apperr "github.com/inwardclub/server/internal/platform/errors"
	"github.com/inwardclub/server/internal/platform/httpx"
)

// memRepo is an in-memory Repository for service-level tests. It stores events
// oldest-first and mirrors the SQL repo's newest-first read and keep-newest-N
// prune so retention and ordering assertions hold without a database.
type memRepo struct {
	events    []ErrorEvent
	nextID    int64
	insertErr error
	pruneErr  error
	listErr   error
}

func (r *memRepo) Insert(_ context.Context, e ErrorEvent) error {
	if r.insertErr != nil {
		return r.insertErr
	}
	r.nextID++
	e.ID = r.nextID
	r.events = append(r.events, e)
	return nil
}

func (r *memRepo) List(_ context.Context, limit, offset int) ([]ErrorEvent, int64, error) {
	if r.listErr != nil {
		return nil, 0, r.listErr
	}
	total := int64(len(r.events))
	out := make([]ErrorEvent, 0, limit)
	// events are stored oldest-first; walk backwards for newest-first, then page.
	for i := len(r.events) - 1 - offset; i >= 0 && len(out) < limit; i-- {
		out = append(out, r.events[i])
	}
	return out, total, nil
}

func (r *memRepo) Prune(_ context.Context, keep int) error {
	if r.pruneErr != nil {
		return r.pruneErr
	}
	if len(r.events) > keep {
		r.events = r.events[len(r.events)-keep:]
	}
	return nil
}

func newTestService() (*Service, *memRepo) {
	repo := &memRepo{}
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return svc, repo
}

func TestServiceListReturnsNewestFirst(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	svc.Record(ctx, "req-1", http.MethodGet, "/a", 500, "boom-1")
	svc.Record(ctx, "req-2", http.MethodGet, "/b", 500, "boom-2")

	events, total, err := svc.List(ctx, httpx.Page{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if len(events) != 2 || events[0].Message != "boom-2" || events[1].Message != "boom-1" {
		t.Fatalf("expected newest-first order, got %+v", events)
	}
}

func TestServiceRecordPrunesToRetentionCap(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()
	for i := 0; i < retentionMaxEvents+10; i++ {
		svc.Record(ctx, "", http.MethodGet, "/x", 500, "err")
	}
	_, total, err := svc.List(ctx, httpx.Page{Page: 1, PageSize: 1})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != retentionMaxEvents {
		t.Fatalf("expected total capped at %d, got %d", retentionMaxEvents, total)
	}
	if len(repo.events) != retentionMaxEvents {
		t.Fatalf("expected repo pruned to %d rows, got %d", retentionMaxEvents, len(repo.events))
	}
}

func TestServiceRecordTruncatesLongMessage(t *testing.T) {
	svc, repo := newTestService()
	long := strings.Repeat("x", maxMessageLen+50)
	svc.Record(context.Background(), "", http.MethodGet, "/x", 500, long)
	if len(repo.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(repo.events))
	}
	if got := len([]rune(repo.events[0].Message)); got != maxMessageLen {
		t.Fatalf("expected message truncated to %d runes, got %d", maxMessageLen, got)
	}
}

func TestServiceRecordBestEffortOnInsertError(t *testing.T) {
	svc, repo := newTestService()
	repo.insertErr = errors.New("db down")
	// Must not panic or propagate; the record is simply dropped.
	svc.Record(context.Background(), "", http.MethodGet, "/x", 500, "err")
	if len(repo.events) != 0 {
		t.Fatalf("expected no persisted events on insert failure, got %d", len(repo.events))
	}
}

func TestCaptureMiddlewareRecordsFailedRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := newTestService()

	r := gin.New()
	r.Use(svc.Capture())
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/boom", func(c *gin.Context) { httpx.Fail(c, apperr.Internal(nil)) })

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	req = httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	events, total, err := svc.List(context.Background(), httpx.Page{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected 1 recorded event (only the failing request), got %d", total)
	}
	if events[0].Path != "/boom" || events[0].Status != http.StatusInternalServerError {
		t.Fatalf("unexpected event: %+v", events[0])
	}
	if !strings.Contains(events[0].Message, "internal error") {
		t.Fatalf("expected message to include cause, got %q", events[0].Message)
	}
}

func TestHandlerListErrorEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := newTestService()
	svc.Record(context.Background(), "req-1", http.MethodGet, "/a", 500, "boom")
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/admin/error-events", h.ListErrorEvents)

	req := httptest.NewRequest(http.MethodGet, "/admin/error-events", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "boom") {
		t.Fatalf("expected event in response, got: %s", rec.Body.String())
	}
}

func TestHandlerListErrorEventsSurfacesRepoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, repo := newTestService()
	repo.listErr = apperr.Internal(errors.New("db down"))
	h := NewHandler(svc)

	r := gin.New()
	r.GET("/admin/error-events", h.ListErrorEvents)

	req := httptest.NewRequest(http.MethodGet, "/admin/error-events", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when the read fails, got %d: %s", rec.Code, rec.Body.String())
	}
}
