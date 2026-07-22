// Package diagnostics owns the admin-facing error-events feed: a durable record
// of server-side failures (5xx responses and handler-attached errors) captured
// as requests flow through the gin middleware chain and persisted to the
// error_events table (see db/migrations 00018). Events survive process restarts;
// the table is bounded by a retention cap (retentionMaxEvents) pruned on every
// write, and read newest-first with pagination.
package diagnostics

import (
	"context"
	"log/slog"
	"time"

	"github.com/inwardclub/server/internal/platform/httpx"
)

// retentionMaxEvents bounds the persisted feed: each write prunes all but the
// newest retentionMaxEvents rows so a sustained error burst cannot grow the
// table unbounded. This mirrors the previous in-memory ring-buffer bound, now
// enforced durably in the error_events table.
const retentionMaxEvents = 500

// maxMessageLen caps the stored error message so an over-long cause cannot
// overflow the message column; the excess is truncated.
const maxMessageLen = 1024

// ErrorEvent is one captured server-side failure.
type ErrorEvent struct {
	ID        int64     `json:"id"`
	RequestID string    `json:"requestId,omitempty"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// Service records and reads persisted error events.
type Service struct {
	repo Repository
	log  *slog.Logger
	now  func() time.Time
}

// NewService builds the diagnostics service over the given persistence port.
// A nil logger falls back to slog.Default so Record can always report failures.
func NewService(repo Repository, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, log: log, now: time.Now}
}

// Record persists a captured error event and prunes the feed to the retention
// cap. It is best-effort: persistence runs on the request's error path, so a
// failure to store (or prune) the diagnostic is logged and swallowed rather than
// masking the original error or breaking the already-written response.
func (s *Service) Record(ctx context.Context, requestID, method, path string, status int, message string) {
	e := ErrorEvent{
		RequestID: requestID,
		Method:    method,
		Path:      path,
		Status:    status,
		Message:   truncate(message, maxMessageLen),
		CreatedAt: s.now().UTC(),
	}
	if err := s.repo.Insert(ctx, e); err != nil {
		s.log.ErrorContext(ctx, "diagnostics: persist error event failed", slog.Any("error", err))
		return
	}
	if err := s.repo.Prune(ctx, retentionMaxEvents); err != nil {
		s.log.WarnContext(ctx, "diagnostics: prune error events failed", slog.Any("error", err))
	}
}

// List returns a page of events, newest first, plus the total count.
func (s *Service) List(ctx context.Context, page httpx.Page) ([]ErrorEvent, int64, error) {
	return s.repo.List(ctx, page.Limit(), page.Offset())
}

// truncate limits s to at most max runes, cutting on a rune boundary so a
// multibyte character is never split into invalid UTF-8.
func truncate(s string, max int) string {
	if r := []rune(s); len(r) > max {
		return string(r[:max])
	}
	return s
}
