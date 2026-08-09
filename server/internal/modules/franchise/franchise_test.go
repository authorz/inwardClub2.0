package franchise

import (
	"context"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/platform/httpx"
)

type memoryRepo struct {
	created         CreateInquiryRequest
	createdMemberID *int64
	updatedID       int64
	updatedStatus   string
}

func (r *memoryRepo) Create(_ context.Context, memberID *int64, req CreateInquiryRequest, now time.Time) (Inquiry, error) {
	r.created = req
	r.createdMemberID = memberID
	return Inquiry{ID: 1, MemberID: memberID, ContactName: req.ContactName, Phone: req.Phone, ExpectedRegion: req.ExpectedRegion, Source: req.Source, CreatedAt: now}, nil
}

func (r *memoryRepo) List(context.Context, httpx.Page, ListFilter) ([]Inquiry, int64, error) {
	return nil, 0, nil
}

func (r *memoryRepo) UpdateStatus(_ context.Context, id int64, status string, _ time.Time) error {
	r.updatedID = id
	r.updatedStatus = status
	return nil
}

type memorySources []string

func (s memorySources) FranchiseInquirySources(context.Context) ([]string, error) { return s, nil }
func (s memorySources) FranchiseHotline(context.Context) (string, error)          { return "400-888-8888", nil }

func TestCreateInquiryValidatesConfiguredSource(t *testing.T) {
	repo := &memoryRepo{}
	svc := NewService(repo, memorySources{"美团", "微信小程序"})
	if _, err := svc.Create(context.Background(), nil, CreateInquiryRequest{
		ContactName: "张先生", Phone: "13888888888", ExpectedRegion: "重庆市渝中区", Source: "抖音",
	}); err == nil {
		t.Fatal("expected invalid source error")
	}
	memberID := int64(9)
	got, err := svc.Create(context.Background(), &memberID, CreateInquiryRequest{
		ContactName: " 张先生 ", Phone: " 13888888888 ", ExpectedRegion: " 重庆市渝中区 ", Source: " 美团 ",
	})
	if err != nil {
		t.Fatalf("create inquiry: %v", err)
	}
	if got.ID != 1 || repo.created.ContactName != "张先生" || repo.created.Source != "美团" {
		t.Fatalf("unexpected created inquiry: %#v", repo.created)
	}
	if repo.createdMemberID == nil || *repo.createdMemberID != memberID {
		t.Fatalf("member attribution was not persisted: %#v", repo.createdMemberID)
	}
}

func TestCreateInquiryRejectsInvalidPhone(t *testing.T) {
	svc := NewService(&memoryRepo{}, memorySources{"美团"})
	if _, err := svc.Create(context.Background(), nil, CreateInquiryRequest{
		ContactName: "张先生", Phone: "not-a-phone", ExpectedRegion: "重庆", Source: "美团",
	}); err == nil {
		t.Fatal("expected invalid phone error")
	}
}

func TestCreateInquiryRejectsUnsafeText(t *testing.T) {
	svc := NewService(&memoryRepo{}, memorySources{"美团"})
	for _, req := range []CreateInquiryRequest{
		{ContactName: `<img src=x onerror=alert(1)>`, Phone: "13888888888", ExpectedRegion: "重庆", Source: "美团"},
		{ContactName: "张先生", Phone: "13888888888", ExpectedRegion: `<script>alert(1)</script>`, Source: "美团"},
	} {
		if _, err := svc.Create(context.Background(), nil, req); err == nil {
			t.Fatalf("unsafe inquiry was accepted: %#v", req)
		}
	}
}

func TestUpdateInquiryStatus(t *testing.T) {
	repo := &memoryRepo{}
	svc := NewService(repo, memorySources{"美团"})
	if err := svc.UpdateStatus(context.Background(), 8, StatusProcessed); err != nil {
		t.Fatalf("update status: %v", err)
	}
	if repo.updatedID != 8 || repo.updatedStatus != StatusProcessed {
		t.Fatalf("unexpected status update: id=%d status=%q", repo.updatedID, repo.updatedStatus)
	}
	if err := svc.UpdateStatus(context.Background(), 8, "done"); err == nil {
		t.Fatal("expected invalid status error")
	}
}
