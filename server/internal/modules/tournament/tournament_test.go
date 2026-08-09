package tournament

import (
	"context"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/platform/httpx"
)

type fakeRepository struct {
	created Input
	current []Event
}

func (f *fakeRepository) List(context.Context, *int64, Filter, httpx.Page) ([]Event, int64, error) {
	return nil, 0, nil
}
func (f *fakeRepository) ListCurrent(context.Context, int64, time.Time) ([]Event, error) {
	return f.current, nil
}
func (f *fakeRepository) Get(context.Context, *int64, int64, bool) (Event, error) {
	return Event{}, nil
}
func (f *fakeRepository) Create(_ context.Context, _ *int64, in Input, now time.Time) (Event, error) {
	f.created = in
	return Event{ID: 1, StoreID: in.StoreID, Title: in.Title, Status: in.Status, StartAt: in.StartAt, EndAt: in.EndAt, CreatedAt: now, UpdatedAt: now}, nil
}
func (f *fakeRepository) Update(context.Context, *int64, int64, Input, time.Time) (Event, error) {
	return Event{}, nil
}
func (f *fakeRepository) Delete(context.Context, *int64, int64) error { return nil }

type fakeAssets struct{}

func (fakeAssets) PublicURLByID(context.Context, int64) (string, error) {
	return "https://assets.example/event.jpg", nil
}

func TestCreateDefaultsToPublishedAndSanitizesRichText(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo, fakeAssets{})
	start := time.Now().Add(time.Hour)
	end := start.Add(time.Hour)
	event, err := svc.Create(context.Background(), nil, Input{
		StoreID: 1, Title: " 德州赛事 ", Content: `<p>规则</p><script>alert(1)</script>`, StartAt: &start, EndAt: &end,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if event.Status != "published" || repo.created.Status != "published" {
		t.Fatalf("status = %q, want published", event.Status)
	}
	if repo.created.Title != "德州赛事" {
		t.Fatalf("title = %q", repo.created.Title)
	}
	if repo.created.Content == `<p>规则</p><script>alert(1)</script>` {
		t.Fatal("rich content was not sanitized")
	}
}

func TestCreateRejectsInvalidTimeRange(t *testing.T) {
	svc := NewService(&fakeRepository{}, fakeAssets{})
	start := time.Now().Add(time.Hour)
	end := start.Add(-time.Minute)
	if _, err := svc.Create(context.Background(), nil, Input{StoreID: 1, Title: "赛事", StartAt: &start, EndAt: &end}); err == nil {
		t.Fatal("Create() error = nil, want invalid time range")
	}
}

func TestListCurrentDecoratesBannerURL(t *testing.T) {
	assetID := int64(9)
	repo := &fakeRepository{current: []Event{{ID: 1, StoreID: 2, Title: "赛事", AssetID: &assetID}}}
	events, err := NewService(repo, fakeAssets{}).ListCurrent(context.Background(), 2)
	if err != nil {
		t.Fatalf("ListCurrent() error = %v", err)
	}
	if len(events) != 1 || events[0].ImageURL == "" {
		t.Fatalf("events = %#v", events)
	}
}
