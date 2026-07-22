package payment

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/inwardclub/server/internal/platform/config"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

const testAPIV3Key = "01234567890123456789012345678901" // exactly 32 bytes

// fixedNow pins the client clock so notify timestamp-skew checks are deterministic.
func fixedNow() time.Time { return time.Unix(1_700_000_000, 0).UTC() }

func mustKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return key
}

// testClient builds a real client wired to a test server, signing outbound
// requests with merchant and verifying notifies with platform.
func testClient(t *testing.T, baseURL string, merchant, platform *rsa.PrivateKey) *WeChatPayClient {
	t.Helper()
	return &WeChatPayClient{
		appID:     "wxappid",
		mchID:     "1600000000",
		serialNo:  "MCH_SERIAL_NO",
		apiV3Key:  []byte(testAPIV3Key),
		notifyURL: "https://example.com/internal/payments/wechat/notify",
		privKey:   merchant,
		pubKey:    &platform.PublicKey,
		pubKeyID:  "PUB_KEY_ID_TEST",
		baseURL:   baseURL,
		http:      http.DefaultClient,
		now:       fixedNow,
	}
}

func TestCreateJSAPIPrepaySignsRequestAndPaySign(t *testing.T) {
	merchant := mustKey(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/pay/transactions/jsapi" || r.Method != http.MethodPost {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		body, _ := readAll(r)
		// The server independently re-verifies the request signature with the
		// merchant public key, proving the Authorization header is correct.
		if err := verifyAuthHeader(r, "/v3/pay/transactions/jsapi", body, &merchant.PublicKey); err != nil {
			t.Errorf("verify request signature: %v", err)
		}
		var got jsapiPrepayRequest
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if got.OutTradeNo != "PO123" || got.Amount.Total != 4599 || got.Payer.OpenID != "openid-1" {
			t.Errorf("unexpected request body: %+v", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"prepay_id":"wx_prepay_abc"}`))
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, merchant, mustKey(t))
	prepay, err := c.CreateJSAPIPrepay(context.Background(), "PO123", 4599, "openid-1", "InwardClub order")
	if err != nil {
		t.Fatalf("CreateJSAPIPrepay: %v", err)
	}
	if prepay.Package != "prepay_id=wx_prepay_abc" || prepay.SignType != "RSA" {
		t.Fatalf("unexpected prepay: %+v", prepay)
	}
	// The paySign must verify against the merchant key over the documented
	// wx.requestPayment message: appId\ntimeStamp\nnonceStr\npackage\n.
	msg := c.appID + "\n" + prepay.Timestamp + "\n" + prepay.NonceStr + "\n" + prepay.Package + "\n"
	if err := verifyRSA(&merchant.PublicKey, msg, prepay.PaySign); err != nil {
		t.Fatalf("paySign does not verify: %v", err)
	}
}

func TestCreateJSAPIPrepayNon2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"PARAM_ERROR","message":"bad amount"}`))
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, mustKey(t), mustKey(t))
	if _, err := c.CreateJSAPIPrepay(context.Background(), "PO1", 1, "o", "d"); err == nil {
		t.Fatal("expected error on non-2xx response")
	}
}

func TestRefundReturnsRefundID(t *testing.T) {
	merchant := mustKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := readAll(r)
		if err := verifyAuthHeader(r, "/v3/refund/domestic/refunds", body, &merchant.PublicKey); err != nil {
			t.Errorf("verify request signature: %v", err)
		}
		var got refundRequest
		_ = json.Unmarshal(body, &got)
		if got.OutTradeNo != "PO9" || got.Amount.Refund != 100 || got.Amount.Total != 200 {
			t.Errorf("unexpected refund body: %+v", got)
		}
		_, _ = w.Write([]byte(`{"refund_id":"wx_refund_1","status":"PROCESSING"}`))
	}))
	defer srv.Close()

	c := testClient(t, srv.URL, merchant, mustKey(t))
	id, err := c.Refund(context.Background(), "PO9", "RF9", 100, 200)
	if err != nil {
		t.Fatalf("Refund: %v", err)
	}
	if id != "wx_refund_1" {
		t.Fatalf("refund id: got %q", id)
	}
}

func TestVerifyNotificationSuccess(t *testing.T) {
	platform := mustKey(t)
	c := testClient(t, "", mustKey(t), platform)

	req := buildNotify(t, platform, "PUB_KEY_ID_TEST", strconv.FormatInt(fixedNow().Unix(), 10), notifyResource{
		OutTradeNo:    "PO777",
		TransactionID: "4200001",
		TradeState:    "SUCCESS",
		Amount: struct {
			Total int64 `json:"total"`
		}{Total: 8800},
	})

	n, err := c.VerifyNotification(req)
	if err != nil {
		t.Fatalf("VerifyNotification: %v", err)
	}
	if !n.Success || n.OutTradeNo != "PO777" || n.TransactionID != "4200001" || n.AmountCent != 8800 {
		t.Fatalf("unexpected notification: %+v", n)
	}
}

func TestVerifyNotificationNonSuccessTradeState(t *testing.T) {
	platform := mustKey(t)
	c := testClient(t, "", mustKey(t), platform)
	req := buildNotify(t, platform, "PUB_KEY_ID_TEST", strconv.FormatInt(fixedNow().Unix(), 10), notifyResource{
		OutTradeNo: "PO1", TradeState: "CLOSED",
	})
	n, err := c.VerifyNotification(req)
	if err != nil {
		t.Fatalf("VerifyNotification: %v", err)
	}
	if n.Success {
		t.Fatal("CLOSED trade_state must not be Success")
	}
}

func TestVerifyNotificationRejectsTamperedBody(t *testing.T) {
	platform := mustKey(t)
	c := testClient(t, "", mustKey(t), platform)
	req := buildNotify(t, platform, "PUB_KEY_ID_TEST", strconv.FormatInt(fixedNow().Unix(), 10), notifyResource{OutTradeNo: "PO1", TradeState: "SUCCESS"})
	// Tamper the body after signing.
	req.Body = readCloser([]byte(`{"event_type":"TRANSACTION.SUCCESS","resource":{}}`))

	_, err := c.VerifyNotification(req)
	assertUnauthenticated(t, err)
}

func TestVerifyNotificationRejectsStaleTimestamp(t *testing.T) {
	platform := mustKey(t)
	c := testClient(t, "", mustKey(t), platform)
	stale := strconv.FormatInt(fixedNow().Add(-10*time.Minute).Unix(), 10)
	req := buildNotify(t, platform, "PUB_KEY_ID_TEST", stale, notifyResource{OutTradeNo: "PO1", TradeState: "SUCCESS"})

	_, err := c.VerifyNotification(req)
	assertUnauthenticated(t, err)
}

func TestVerifyNotificationRejectsUnknownSerial(t *testing.T) {
	platform := mustKey(t)
	c := testClient(t, "", mustKey(t), platform)
	req := buildNotify(t, platform, "SOME_OTHER_KEY_ID", strconv.FormatInt(fixedNow().Unix(), 10), notifyResource{OutTradeNo: "PO1", TradeState: "SUCCESS"})

	_, err := c.VerifyNotification(req)
	assertUnauthenticated(t, err)
}

func TestVerifyNotificationRejectsWrongPlatformKey(t *testing.T) {
	// Client trusts one platform key; the notify is signed by a different one.
	c := testClient(t, "", mustKey(t), mustKey(t))
	attacker := mustKey(t)
	req := buildNotify(t, attacker, "PUB_KEY_ID_TEST", strconv.FormatInt(fixedNow().Unix(), 10), notifyResource{OutTradeNo: "PO1", TradeState: "SUCCESS"})

	_, err := c.VerifyNotification(req)
	assertUnauthenticated(t, err)
}

func TestNewWeChatPayClientLoadsKeysFromDisk(t *testing.T) {
	dir := t.TempDir()
	merchant := mustKey(t)
	platform := mustKey(t)
	privPath := filepath.Join(dir, "apiclient_key.pem")
	pubPath := filepath.Join(dir, "pub_key.pem")
	writePKCS8(t, privPath, merchant)
	writePKIX(t, pubPath, &platform.PublicKey)

	cfg := config.WeChatConfig{
		MiniAppID:          "wxappid",
		PayMchID:           "1600000000",
		PayMchCertSerialNo: "MCH_SERIAL_NO",
		PayPrivateKeyPath:  privPath,
		PayAPIV3Key:        testAPIV3Key,
		PayNotifyURL:       "https://example.com/notify",
		PayPublicKeyPath:   pubPath,
		PayPublicKeyID:     "PUB_KEY_ID_TEST",
	}
	c, err := NewWeChatPayClient(cfg)
	if err != nil {
		t.Fatalf("NewWeChatPayClient: %v", err)
	}
	if c.privKey == nil || c.pubKey == nil || c.baseURL != wechatAPIBaseURL {
		t.Fatal("client not fully initialised")
	}
}

func TestNewWeChatPayClientRejectsBadAPIV3KeyLength(t *testing.T) {
	_, err := NewWeChatPayClient(config.WeChatConfig{PayAPIV3Key: "too-short"})
	if err == nil {
		t.Fatal("expected error for wrong apiV3 key length")
	}
}

// --- helpers ---

func readAll(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)
	return buf.Bytes(), err
}

func readCloser(b []byte) *readCloserT { return &readCloserT{bytes.NewReader(b)} }

type readCloserT struct{ *bytes.Reader }

func (readCloserT) Close() error { return nil }

func verifyRSA(pub *rsa.PublicKey, message, signatureB64 string) error {
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig)
}

// verifyAuthHeader re-derives and verifies the WECHATPAY2-SHA256-RSA2048 request
// signature the same way WeChat's servers would.
func verifyAuthHeader(r *http.Request, path string, body []byte, pub *rsa.PublicKey) error {
	raw := r.Header.Get("Authorization")
	fields := parseAuthToken(strings.TrimPrefix(raw, "WECHATPAY2-SHA256-RSA2048 "))
	msg := r.Method + "\n" + path + "\n" + fields["timestamp"] + "\n" + fields["nonce_str"] + "\n" + string(body) + "\n"
	return verifyRSA(pub, msg, fields["signature"])
}

func parseAuthToken(token string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(token, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		out[kv[0]] = strings.Trim(kv[1], `"`)
	}
	return out
}

// buildNotify produces a signed + AES-256-GCM-encrypted WeChat notify request.
func buildNotify(t *testing.T, platform *rsa.PrivateKey, serial, timestamp string, res notifyResource) *http.Request {
	t.Helper()
	plain, _ := json.Marshal(res)
	block, err := aes.NewCipher([]byte(testAPIV3Key))
	if err != nil {
		t.Fatalf("aes: %v", err)
	}
	gcm, _ := cipher.NewGCM(block)
	nonce := []byte("abcdef123456") // 12-byte GCM nonce
	ad := "transaction"
	ct := gcm.Seal(nil, nonce, plain, []byte(ad))

	var env notifyEnvelope
	env.EventType = "TRANSACTION.SUCCESS"
	env.Resource.Algorithm = "AEAD_AES_256_GCM"
	env.Resource.Ciphertext = base64.StdEncoding.EncodeToString(ct)
	env.Resource.AssociatedData = ad
	env.Resource.Nonce = string(nonce)
	body, _ := json.Marshal(env)

	nonceStr := "notifynonce"
	msg := timestamp + "\n" + nonceStr + "\n" + string(body) + "\n"
	digest := sha256.Sum256([]byte(msg))
	sig, err := rsa.SignPKCS1v15(rand.Reader, platform, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign notify: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/internal/payments/wechat/notify", bytes.NewReader(body))
	req.Header.Set("Wechatpay-Timestamp", timestamp)
	req.Header.Set("Wechatpay-Nonce", nonceStr)
	req.Header.Set("Wechatpay-Signature", base64.StdEncoding.EncodeToString(sig))
	req.Header.Set("Wechatpay-Serial", serial)
	return req
}

func assertUnauthenticated(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.From(err).Code != apperr.CodeUnauthenticated {
		t.Fatalf("want Unauthenticated, got %v", err)
	}
}

func writePKCS8(t *testing.T, path string, key *rsa.PrivateKey) {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	writePEM(t, path, "PRIVATE KEY", der)
}

func writePKIX(t *testing.T, path string, pub *rsa.PublicKey) {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal pkix: %v", err)
	}
	writePEM(t, path, "PUBLIC KEY", der)
}

func writePEM(t *testing.T, path, typ string, der []byte) {
	t.Helper()
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0o600); err != nil {
		t.Fatalf("write pem: %v", err)
	}
}
