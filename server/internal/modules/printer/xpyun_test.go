package printer

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/platform/config"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

func newTestXpyun(t *testing.T, handler http.HandlerFunc) *XpyunPrinter {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	p := NewXpyunPrinter(config.XpyunConfig{User: "acct", UKey: "ukey-123"})
	p.baseURL = srv.URL
	p.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	return p
}

func TestXpyunPrintSignsAndSucceeds(t *testing.T) {
	p := newTestXpyun(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/print" {
			t.Errorf("path = %q, want /print", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req xpyunPrintRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.SN != "SN-001" || req.Content != "receipt-body" {
			t.Errorf("request mapping = %+v", req)
		}
		if want := xpyunSign(req.User, "ukey-123", req.Timestamp); req.Sign != want {
			t.Errorf("sign = %q, want %q", req.Sign, want)
		}
		if req.Timestamp != "1700000000" {
			t.Errorf("timestamp = %q, want 1700000000", req.Timestamp)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok", "data": "print-order-1"})
	})

	err := p.Print(context.Background(), Job{DeviceSN: "SN-001", Template: "order", Content: "receipt-body"})
	if err != nil {
		t.Fatalf("Print: %v", err)
	}
}

func TestXpyunPrintRejectedByProvider(t *testing.T) {
	p := newTestXpyun(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 1002, "msg": "printer offline"})
	})
	err := p.Print(context.Background(), Job{DeviceSN: "SN-001", Content: "x"})
	if err == nil || !strings.Contains(err.Error(), "printer offline") {
		t.Fatalf("want rejection carrying provider msg, got %v", err)
	}
}

func TestXpyunPrintRequiresDeviceSN(t *testing.T) {
	p := NewXpyunPrinter(config.XpyunConfig{User: "acct", UKey: "ukey-123"})
	err := p.Print(context.Background(), Job{Content: "x"})
	if err == nil || apperr.From(err).Code != apperr.CodeInvalidArgument {
		t.Fatalf("want INVALID_ARGUMENT, got %v", err)
	}
}

func TestSelectHonoursFakeFlag(t *testing.T) {
	cfg := config.XpyunConfig{User: "acct", UKey: "ukey-123"}
	if _, ok := Select(cfg, true).(*FakePrinter); !ok {
		t.Errorf("Select(useFake=true) should return *FakePrinter")
	}
	if _, ok := Select(cfg, false).(*XpyunPrinter); !ok {
		t.Errorf("Select(useFake=false) should return *XpyunPrinter")
	}
}
