package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAdminServiceListChannelSettingsDefaults(t *testing.T) {
	svc := NewAdminService(&memStoreRepo{})
	views := svc.ListChannelSettings(context.Background())
	if len(views) != 2 {
		t.Fatalf("expected 2 default channels, got %d", len(views))
	}
	if views[0].Channel != "offline" || !views[0].Enabled {
		t.Fatalf("expected offline enabled first (sorted), got %+v", views[0])
	}
	if views[1].Channel != "wechat" || !views[1].Enabled {
		t.Fatalf("expected wechat enabled second, got %+v", views[1])
	}
}

func TestAdminServiceUpdateChannelSettingsRejectsUnknownChannel(t *testing.T) {
	svc := NewAdminService(&memStoreRepo{})
	_, err := svc.UpdateChannelSettings(context.Background(), UpdateChannelSettingsRequest{
		Channels: []ChannelSetting{{Channel: "alipay", Enabled: false}},
	})
	if err == nil {
		t.Fatal("expected an error for an unknown channel")
	}
}

func TestAdminServiceUpdateChannelSettingsDisablesChannel(t *testing.T) {
	svc := NewAdminService(&memStoreRepo{})
	views, err := svc.UpdateChannelSettings(context.Background(), UpdateChannelSettingsRequest{
		Channels: []ChannelSetting{{Channel: "wechat", Enabled: false}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, v := range views {
		if v.Channel == "wechat" && v.Enabled {
			t.Fatalf("expected wechat disabled, got %+v", v)
		}
	}
	// Re-listing must reflect the persisted (in-memory) change.
	again := svc.ListChannelSettings(context.Background())
	for _, v := range again {
		if v.Channel == "wechat" && v.Enabled {
			t.Fatalf("expected wechat still disabled on re-list, got %+v", v)
		}
	}
}

func TestAdminHandlerPaymentChannelSettingsRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewAdminService(&memStoreRepo{})
	h := NewAdminHandler(svc)

	r := gin.New()
	r.GET("/admin/payment-channel-settings", h.PaymentChannelSettings)
	r.PUT("/admin/payment-channel-settings", h.UpdatePaymentChannelSettings)

	req := httptest.NewRequest(http.MethodGet, "/admin/payment-channel-settings", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "wechat") {
		t.Fatalf("expected wechat channel in response, got: %s", rec.Body.String())
	}

	body, _ := json.Marshal(UpdateChannelSettingsRequest{
		Channels: []ChannelSetting{{Channel: "offline", Enabled: false}},
	})
	req = httptest.NewRequest(http.MethodPut, "/admin/payment-channel-settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data []ChannelSetting `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	for _, v := range resp.Data {
		if v.Channel == "offline" && v.Enabled {
			t.Fatalf("expected offline disabled after update, got %+v", v)
		}
	}
}
