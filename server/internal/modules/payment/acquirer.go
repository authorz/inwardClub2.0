package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/inwardclub/server/internal/platform/config"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// acquirerSignHeader carries the HMAC-SHA256 (lower-hex) signature of the raw
// request/callback body.
const acquirerSignHeader = "X-Sign"

// acquirerBodyLimit bounds how much of a response/callback body is read.
const acquirerBodyLimit = 1 << 20

// HTTPAcquirer is the real offline aggregated-collection acquirer client. It is
// selected only when USE_FAKE_ADAPTERS=false; FakeOfflineAcquirer keeps dev and
// tests offline. No concrete provider has been chosen yet, so the wire protocol
// is a single, clearly documented HMAC-signed JSON contract that a mainstream
// aggregator can be mapped onto: every outbound request body is signed with the
// shared merchant API key (HMAC-SHA256, lower-hex) in the X-Sign header, and
// every inbound callback is verified the same way. Selecting a specific provider
// then only changes the field mapping in this file — business code is untouched.
type HTTPAcquirer struct {
	cfg config.OfflineConfig

	// Injectable for tests; production defaults are set by NewHTTPAcquirer.
	baseURL string
	http    *http.Client
	now     func() time.Time
}

// NewHTTPAcquirer builds the real acquirer client from config.
func NewHTTPAcquirer(cfg config.OfflineConfig) *HTTPAcquirer {
	return &HTTPAcquirer{
		cfg:     cfg,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
		now:     time.Now,
	}
}

// acquirerCreateRequest is the create-dynamic-QR request body.
type acquirerCreateRequest struct {
	MerchantID string `json:"merchantId"`
	OutTradeNo string `json:"outTradeNo"`
	AmountCent int64  `json:"amountCent"`
	Subject    string `json:"subject"`
	ExpiresAt  int64  `json:"expiresAt"` // unix seconds
	NotifyURL  string `json:"notifyUrl"`
	Nonce      string `json:"nonce"`
	Timestamp  int64  `json:"timestamp"` // unix seconds
}

// acquirerCreateResponse is the create-dynamic-QR reply; code 0 means success.
type acquirerCreateResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		AcquirerOrderNo string `json:"acquirerOrderNo"`
		QRContent       string `json:"qrContent"`
		ExpiresAt       int64  `json:"expiresAt"` // unix seconds; optional
	} `json:"data"`
}

// acquirerRefundRequest is the refund request body.
type acquirerRefundRequest struct {
	MerchantID      string `json:"merchantId"`
	AcquirerOrderNo string `json:"acquirerOrderNo"`
	OutRefundNo     string `json:"outRefundNo"`
	AmountCent      int64  `json:"amountCent"`
	Nonce           string `json:"nonce"`
	Timestamp       int64  `json:"timestamp"`
}

// acquirerRefundResponse is the refund reply; code 0 means success.
type acquirerRefundResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		RefundNo string `json:"refundNo"`
	} `json:"data"`
}

// acquirerNotifyBody is the callback payload the acquirer POSTs to NotifyURL.
// The acquirer decides the paying channel; the server never guesses it.
type acquirerNotifyBody struct {
	OutTradeNo      string `json:"outTradeNo"`
	AcquirerOrderNo string `json:"acquirerOrderNo"`
	ExternalTradeNo string `json:"externalTradeNo"`
	Channel         string `json:"channel"` // "wechat" | "alipay"
	AmountCent      int64  `json:"amountCent"`
	Status          string `json:"status"` // "success" when paid
}

// CreateDynamicQR mints a dynamic aggregated-collection code for one order. The
// returned expiry honours the acquirer's echo when present, otherwise the
// requested expiry stands.
func (a *HTTPAcquirer) CreateDynamicQR(ctx context.Context, outTradeNo string, amountCent int64, subject string, expiresAt time.Time) (OfflineQR, error) {
	ts := a.now().Unix()
	body := acquirerCreateRequest{
		MerchantID: a.cfg.MerchantID,
		OutTradeNo: outTradeNo,
		AmountCent: amountCent,
		Subject:    subject,
		ExpiresAt:  expiresAt.Unix(),
		NotifyURL:  a.cfg.NotifyURL,
		Nonce:      acquirerNonce(outTradeNo, ts),
		Timestamp:  ts,
	}
	var resp acquirerCreateResponse
	if err := a.post(ctx, "/collect/qr", body, &resp); err != nil {
		return OfflineQR{}, err
	}
	if resp.Code != 0 {
		return OfflineQR{}, fmt.Errorf("acquirer create qr rejected: code=%d msg=%s", resp.Code, resp.Msg)
	}
	qr := OfflineQR{
		AcquirerOrderNo: resp.Data.AcquirerOrderNo,
		QRContent:       resp.Data.QRContent,
		ExpiresAt:       expiresAt,
	}
	if resp.Data.ExpiresAt > 0 {
		qr.ExpiresAt = time.Unix(resp.Data.ExpiresAt, 0).UTC()
	}
	return qr, nil
}

// VerifyNotification verifies the callback signature against the shared API key,
// then maps the body to an OfflineNotification. A bad signature is
// Unauthenticated so the handler returns a non-2xx and the acquirer retries.
func (a *HTTPAcquirer) VerifyNotification(req *http.Request) (OfflineNotification, error) {
	if req == nil || req.Body == nil {
		return OfflineNotification{}, apperr.Invalid("empty acquirer notify request")
	}
	body, err := io.ReadAll(io.LimitReader(req.Body, acquirerBodyLimit))
	if err != nil {
		return OfflineNotification{}, apperr.Invalid("cannot read acquirer callback body")
	}
	if !a.verify(body, req.Header.Get(acquirerSignHeader)) {
		return OfflineNotification{}, apperr.Unauthenticated("invalid acquirer callback signature")
	}
	var b acquirerNotifyBody
	if err := json.Unmarshal(body, &b); err != nil {
		return OfflineNotification{}, apperr.Invalid("invalid acquirer callback body")
	}
	return OfflineNotification{
		AcquirerOrderNo: b.AcquirerOrderNo,
		OutTradeNo:      b.OutTradeNo,
		ExternalTradeNo: b.ExternalTradeNo,
		Channel:         b.Channel,
		AmountCent:      b.AmountCent,
		Success:         strings.EqualFold(b.Status, "success"),
	}, nil
}

// Refund submits a refund against a settled collection and returns the
// acquirer's refund id.
func (a *HTTPAcquirer) Refund(ctx context.Context, acquirerOrderNo, outRefundNo string, amountCent int64) (string, error) {
	ts := a.now().Unix()
	body := acquirerRefundRequest{
		MerchantID:      a.cfg.MerchantID,
		AcquirerOrderNo: acquirerOrderNo,
		OutRefundNo:     outRefundNo,
		AmountCent:      amountCent,
		Nonce:           acquirerNonce(outRefundNo, ts),
		Timestamp:       ts,
	}
	var resp acquirerRefundResponse
	if err := a.post(ctx, "/refund", body, &resp); err != nil {
		return "", err
	}
	if resp.Code != 0 {
		return "", fmt.Errorf("acquirer refund rejected: code=%d msg=%s", resp.Code, resp.Msg)
	}
	return resp.Data.RefundNo, nil
}

// post marshals body, signs it, POSTs to baseURL+path and decodes the JSON reply.
func (a *HTTPAcquirer) post(ctx context.Context, path string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(acquirerSignHeader, a.sign(raw))

	resp, err := a.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, acquirerBodyLimit))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("acquirer %s: unexpected status %d: %s", path, resp.StatusCode, string(respBody))
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("acquirer %s: decode response: %w", path, err)
	}
	return nil
}

// sign is the lower-hex HMAC-SHA256 of body under the shared API key.
func (a *HTTPAcquirer) sign(body []byte) string {
	mac := hmac.New(sha256.New, []byte(a.cfg.APIKey))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// verify constant-time compares the expected signature against the header value.
func (a *HTTPAcquirer) verify(body []byte, got string) bool {
	return hmac.Equal([]byte(a.sign(body)), []byte(got))
}

// acquirerNonce derives a deterministic per-request nonce from a unique seed
// (the order/refund number) and the request timestamp.
func acquirerNonce(seed string, ts int64) string {
	return short(seed + strconv.FormatInt(ts, 10))
}

// compile-time assertion: the real client satisfies the acquirer interface.
var _ OfflineAcquirer = (*HTTPAcquirer)(nil)
