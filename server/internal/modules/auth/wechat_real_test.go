package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// newTestWeChatClient points the real client at an httptest server and pins the
// clock so token expiry is deterministic.
func newTestWeChatClient(server *httptest.Server) *WeChatHTTPClient {
	c := NewWeChatClient("appid", "secret")
	c.baseURL = server.URL
	c.http = server.Client()
	c.now = func() time.Time { return time.Unix(1_700_000_000, 0) }
	return c
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func TestWeChatCode2Session(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sns/jscode2session" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("js_code") != "login-code" || q.Get("appid") != "appid" ||
			q.Get("secret") != "secret" || q.Get("grant_type") != "authorization_code" {
			t.Errorf("unexpected query %v", q)
		}
		writeJSON(t, w, map[string]any{"openid": "openid-123", "unionid": "union-9", "session_key": "sk"})
	}))
	defer server.Close()

	sess, err := newTestWeChatClient(server).Code2Session(context.Background(), "login-code")
	if err != nil {
		t.Fatalf("Code2Session: %v", err)
	}
	if sess.OpenID != "openid-123" || sess.UnionID != "union-9" || sess.SessionKey != "sk" {
		t.Fatalf("unexpected session %+v", sess)
	}
}

func TestWeChatCode2SessionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, map[string]any{"errcode": 40029, "errmsg": "invalid code"})
	}))
	defer server.Close()

	if _, err := newTestWeChatClient(server).Code2Session(context.Background(), "bad"); err == nil {
		t.Fatal("expected error on errcode 40029")
	}
}

func TestWeChatGetPhoneNumberCachesToken(t *testing.T) {
	var tokenHits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/stable_token":
			atomic.AddInt32(&tokenHits, 1)
			writeJSON(t, w, map[string]any{"access_token": "token-abc", "expires_in": 7200})
		case "/wxa/business/getuserphonenumber":
			if got := r.URL.Query().Get("access_token"); got != "token-abc" {
				t.Errorf("wrong access_token %q", got)
			}
			var body struct {
				Code string `json:"code"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode body: %v", err)
			}
			if body.Code == "" {
				t.Error("empty phone code")
			}
			writeJSON(t, w, map[string]any{
				"errcode": 0, "errmsg": "ok",
				"phone_info": map[string]any{"phoneNumber": "+8613800138000", "purePhoneNumber": "13800138000", "countryCode": "86"},
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := newTestWeChatClient(server)
	for i := 0; i < 2; i++ {
		phone, err := c.GetPhoneNumber(context.Background(), "phone-code")
		if err != nil {
			t.Fatalf("GetPhoneNumber: %v", err)
		}
		if phone != "13800138000" {
			t.Fatalf("unexpected phone %q", phone)
		}
	}
	if got := atomic.LoadInt32(&tokenHits); got != 1 {
		t.Fatalf("expected token fetched once, got %d", got)
	}
}

func TestWeChatGetPhoneNumberRetriesOnTokenError(t *testing.T) {
	var phoneHits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/stable_token":
			writeJSON(t, w, map[string]any{"access_token": "token-abc", "expires_in": 7200})
		case "/wxa/business/getuserphonenumber":
			if atomic.AddInt32(&phoneHits, 1) == 1 {
				writeJSON(t, w, map[string]any{"errcode": 40001, "errmsg": "invalid access_token"})
				return
			}
			writeJSON(t, w, map[string]any{
				"errcode": 0, "errmsg": "ok",
				"phone_info": map[string]any{"purePhoneNumber": "13900139000"},
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	phone, err := newTestWeChatClient(server).GetPhoneNumber(context.Background(), "phone-code")
	if err != nil {
		t.Fatalf("GetPhoneNumber: %v", err)
	}
	if phone != "13900139000" {
		t.Fatalf("unexpected phone %q", phone)
	}
	if got := atomic.LoadInt32(&phoneHits); got != 2 {
		t.Fatalf("expected 2 phone calls (retry after token error), got %d", got)
	}
}

func TestWeChatGetPhoneNumberError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/cgi-bin/stable_token":
			writeJSON(t, w, map[string]any{"access_token": "token-abc", "expires_in": 7200})
		case "/wxa/business/getuserphonenumber":
			writeJSON(t, w, map[string]any{"errcode": 40003, "errmsg": "invalid openid"})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	if _, err := newTestWeChatClient(server).GetPhoneNumber(context.Background(), "phone-code"); err == nil {
		t.Fatal("expected error on non-token errcode")
	}
}
