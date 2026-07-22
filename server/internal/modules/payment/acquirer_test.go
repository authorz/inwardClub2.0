package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

const testAcquirerKey = "acq-secret-key"

func hmacHex(key string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func newTestAcquirer(t *testing.T, handler http.HandlerFunc) *HTTPAcquirer {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	a := NewHTTPAcquirer(config.OfflineConfig{
		Provider:   "generic",
		MerchantID: "M123",
		APIKey:     testAcquirerKey,
		NotifyURL:  "https://api.example.com/internal/payments/offline-acquirer/notify",
	})
	a.baseURL = srv.URL
	return a
}

func TestHTTPAcquirerCreateDynamicQR(t *testing.T) {
	a := newTestAcquirer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/collect/qr" {
			t.Errorf("path = %q, want /collect/qr", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if got, want := r.Header.Get(acquirerSignHeader), hmacHex(testAcquirerKey, body); got != want {
			t.Errorf("signature = %q, want %q", got, want)
		}
		var req acquirerCreateRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.MerchantID != "M123" || req.OutTradeNo != "PO-1" || req.AmountCent != 1500 {
			t.Errorf("unexpected request mapping: %+v", req)
		}
		if req.NotifyURL == "" || req.Nonce == "" || req.Timestamp == 0 {
			t.Errorf("missing signed fields: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"msg":  "ok",
			"data": map[string]any{
				"acquirerOrderNo": "ACQ-9",
				"qrContent":       "https://pay.example.com/qr/abc",
			},
		})
	})

	expires := time.Now().Add(10 * time.Minute).UTC()
	qr, err := a.CreateDynamicQR(context.Background(), "PO-1", 1500, "咖啡", expires)
	if err != nil {
		t.Fatalf("CreateDynamicQR: %v", err)
	}
	if qr.AcquirerOrderNo != "ACQ-9" || qr.QRContent != "https://pay.example.com/qr/abc" {
		t.Errorf("qr mapping = %+v", qr)
	}
	if !qr.ExpiresAt.Equal(expires) {
		t.Errorf("ExpiresAt = %v, want requested %v", qr.ExpiresAt, expires)
	}
}

func TestHTTPAcquirerCreateRejectsNonZeroCode(t *testing.T) {
	a := newTestAcquirer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 40001, "msg": "merchant blocked"})
	})
	_, err := a.CreateDynamicQR(context.Background(), "PO-2", 100, "x", time.Now())
	if err == nil || !strings.Contains(err.Error(), "merchant blocked") {
		t.Fatalf("want rejection carrying provider msg, got %v", err)
	}
}

func TestHTTPAcquirerVerifyNotification(t *testing.T) {
	a := NewHTTPAcquirer(config.OfflineConfig{APIKey: testAcquirerKey})

	body := []byte(`{"outTradeNo":"PO-1","acquirerOrderNo":"ACQ-9","externalTradeNo":"WX-7","channel":"wechat","amountCent":1500,"status":"SUCCESS"}`)
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(string(body)))
	req.Header.Set(acquirerSignHeader, hmacHex(testAcquirerKey, body))

	n, err := a.VerifyNotification(req)
	if err != nil {
		t.Fatalf("VerifyNotification: %v", err)
	}
	if n.OutTradeNo != "PO-1" || n.AcquirerOrderNo != "ACQ-9" || n.ExternalTradeNo != "WX-7" {
		t.Errorf("notification mapping = %+v", n)
	}
	if n.Channel != "wechat" || n.AmountCent != 1500 || !n.Success {
		t.Errorf("notification mapping = %+v", n)
	}
}

func TestHTTPAcquirerVerifyNotificationRejectsBadSignature(t *testing.T) {
	a := NewHTTPAcquirer(config.OfflineConfig{APIKey: testAcquirerKey})

	body := []byte(`{"outTradeNo":"PO-1","status":"SUCCESS"}`)
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(string(body)))
	req.Header.Set(acquirerSignHeader, "deadbeef")

	_, err := a.VerifyNotification(req)
	if err == nil || apperr.From(err).Code != apperr.CodeUnauthenticated {
		t.Fatalf("want UNAUTHENTICATED, got %v", err)
	}
}

func TestHTTPAcquirerRefund(t *testing.T) {
	a := newTestAcquirer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/refund" {
			t.Errorf("path = %q, want /refund", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if got, want := r.Header.Get(acquirerSignHeader), hmacHex(testAcquirerKey, body); got != want {
			t.Errorf("signature = %q, want %q", got, want)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0, "msg": "ok", "data": map[string]any{"refundNo": "RF-42"},
		})
	})
	refundNo, err := a.Refund(context.Background(), "ACQ-9", "RF-1", 1500)
	if err != nil {
		t.Fatalf("Refund: %v", err)
	}
	if refundNo != "RF-42" {
		t.Errorf("refundNo = %q, want RF-42", refundNo)
	}
}
