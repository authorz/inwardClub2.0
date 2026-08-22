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

type xpyunSettingsStub struct {
	user, key, baseURL string
}

func (s xpyunSettingsStub) PrinterProviderSettings(context.Context) (string, string, string, error) {
	return s.user, s.key, s.baseURL, nil
}

func TestXpyunPrintSignsAndSucceeds(t *testing.T) {
	p := newTestXpyun(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/print" {
			t.Errorf("path = %q, want /print", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req struct {
			User      string `json:"user"`
			Timestamp string `json:"timestamp"`
			Sign      string `json:"sign"`
			SN        string `json:"sn"`
			Content   string `json:"content"`
			Voice     int    `json:"voice"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.SN != "SN-001" || req.Content != "receipt-body" {
			t.Errorf("request mapping = %+v", req)
		}
		if req.Voice != 2 {
			t.Errorf("voice = %d, want 2", req.Voice)
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

func TestXpyunPrintUsesSilentVoiceMode(t *testing.T) {
	p := newTestXpyun(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Voice int `json:"voice"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Voice != 1 {
			t.Errorf("voice = %d, want silent mode 1", req.Voice)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok", "data": "print-order-1"})
	})
	if err := p.Print(context.Background(), Job{DeviceSN: "SN-001", Content: "test", Silent: true}); err != nil {
		t.Fatalf("Print: %v", err)
	}
}

func TestXpyunSetVoiceMapsProviderParameters(t *testing.T) {
	p := newTestXpyun(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/setVoiceType" {
			t.Errorf("path = %q, want /setVoiceType", r.URL.Path)
		}
		var req struct {
			SN          string `json:"sn"`
			VoiceType   int    `json:"voiceType"`
			VolumeLevel int    `json:"volumeLevel"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.SN != "SN-001" || req.VoiceType != 4 || req.VolumeLevel != 3 {
			t.Errorf("voice request = %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok", "data": true})
	})
	volume := 3
	if err := p.SetVoice(context.Background(), "SN-001", 4, &volume); err != nil {
		t.Fatalf("SetVoice: %v", err)
	}
}

func TestXpyunAddPrinterRequiresProviderSuccessList(t *testing.T) {
	p := newTestXpyun(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/addPrinters" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json;charset=UTF-8" {
			t.Errorf("content type = %q", got)
		}
		var req struct {
			Items []struct {
				SN   string `json:"sn"`
				Name string `json:"name"`
			} `json:"items"`
			User      string `json:"user"`
			Timestamp string `json:"timestamp"`
			Sign      string `json:"sign"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Items) != 1 || req.Items[0].SN != "SN-001" || req.Items[0].Name != "SN-001" {
			t.Fatalf("items = %#v", req.Items)
		}
		if req.Sign != xpyunSign(req.User, "ukey-123", req.Timestamp) {
			t.Fatal("invalid signature")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "msg": "ok", "data": map[string]any{"success": []string{"SN-001"}},
		})
	})
	if err := p.AddPrinter(context.Background(), "SN-001", "SN-001"); err != nil {
		t.Fatalf("AddPrinter: %v", err)
	}
}

func TestXpyunAddPrinterRejectsPartialFailure(t *testing.T) {
	p := newTestXpyun(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "msg": "ok", "data": map[string]any{
				"success": []string{}, "fail": []string{"SN-001"}, "failMsg": []string{"SN-001:1011"},
			},
		})
	})
	if err := p.AddPrinter(context.Background(), "SN-001", "SN-001"); err == nil || !strings.Contains(err.Error(), "1011") {
		t.Fatalf("expected provider failure, got %v", err)
	}
}

func TestXpyunQueryStatusesMapsProviderValues(t *testing.T) {
	p := newTestXpyun(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok", "data": []int{0, 1, 2}})
	})
	statuses, err := p.QueryStatuses(context.Background(), []string{"A", "B", "C"})
	if err != nil {
		t.Fatalf("QueryStatuses: %v", err)
	}
	if statuses["A"] != ProviderStatusOffline || statuses["B"] != ProviderStatusOnline || statuses["C"] != ProviderStatusAbnormal {
		t.Fatalf("statuses = %#v", statuses)
	}
}

func TestXpyunUsesSharedGlobalSettings(t *testing.T) {
	var requestedUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		requestedUser, _ = payload["user"].(string)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "msg": "ok", "data": []int{1}})
	}))
	defer srv.Close()
	p := NewXpyunPrinterWithSettings(xpyunSettingsStub{
		user: "global-account", key: "global-key", baseURL: srv.URL,
	}, config.XpyunConfig{User: "legacy-account", UKey: "legacy-key"})
	if _, err := p.QueryStatuses(context.Background(), []string{"SN-1"}); err != nil {
		t.Fatalf("QueryStatuses: %v", err)
	}
	if requestedUser != "global-account" {
		t.Fatalf("requested user = %q", requestedUser)
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
