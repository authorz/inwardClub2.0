// Package payment owns unified payment, refund and offline aggregated collection.
// External providers are reached only through the interfaces below so no third-
// party SDK leaks into business code. Phase-1 ships the interfaces + fakes and
// the WeChat notify / offline notify callback skeletons.
package payment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// WeChatPrepay is the JSAPI payment parameters returned to the mini program.
type WeChatPrepay struct {
	PrepayID  string `json:"prepayId"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
	Timestamp string `json:"timeStamp"`
}

// WeChatNotification is the verified, decrypted content of a WeChat pay notify.
type WeChatNotification struct {
	OutTradeNo    string
	TransactionID string
	AmountCent    int64
	Success       bool
}

// WeChatPayGateway abstracts WeChat Pay order creation, cancellation, notify
// verification and refunds. JSAPI serves the mini program; Native serves the
// store console's dynamic collection code.
type WeChatPayGateway interface {
	CreateJSAPIPrepay(ctx context.Context, outTradeNo string, amountCent int64, openID, description string) (WeChatPrepay, error)
	CreateNativeOrder(ctx context.Context, outTradeNo string, amountCent int64, description string, expiresAt time.Time) (string, error)
	CloseOrder(ctx context.Context, outTradeNo string) error
	VerifyNotification(req *http.Request) (WeChatNotification, error)
	Refund(ctx context.Context, outTradeNo, outRefundNo string, amountCent, totalCent int64) (string, error)
}

// OfflineQR is the dynamic aggregated collection code from the acquirer.
type OfflineQR struct {
	AcquirerOrderNo string
	QRContent       string
	ExpiresAt       time.Time
}

// OfflineNotification is the verified content of an acquirer callback. The
// channel is wechat or alipay; the acquirer decides, the server never guesses.
type OfflineNotification struct {
	AcquirerOrderNo string
	OutTradeNo      string
	ExternalTradeNo string
	Channel         string // "wechat" | "alipay"
	AmountCent      int64
	Success         bool
}

// OfflineAcquirer abstracts the (not-yet-selected) aggregated collection
// provider. A dynamic code is bound to one store/amount/purpose/expiry.
type OfflineAcquirer interface {
	CreateDynamicQR(ctx context.Context, outTradeNo string, amountCent int64, subject string, expiresAt time.Time) (OfflineQR, error)
	VerifyNotification(req *http.Request) (OfflineNotification, error)
	Refund(ctx context.Context, acquirerOrderNo, outRefundNo string, amountCent int64) (string, error)
}

// FakeWeChatPayGateway is an offline gateway for dev/tests.
type FakeWeChatPayGateway struct{}

// NewFakeWeChatPayGateway builds the fake gateway.
func NewFakeWeChatPayGateway() *FakeWeChatPayGateway { return &FakeWeChatPayGateway{} }

func (f *FakeWeChatPayGateway) CreateJSAPIPrepay(_ context.Context, outTradeNo string, _ int64, _, _ string) (WeChatPrepay, error) {
	return WeChatPrepay{
		PrepayID:  "fake_prepay_" + short(outTradeNo),
		NonceStr:  short(outTradeNo + "nonce"),
		Package:   "prepay_id=fake_prepay_" + short(outTradeNo),
		SignType:  "RSA",
		PaySign:   "fake_sign",
		Timestamp: "0",
	}, nil
}

func (f *FakeWeChatPayGateway) CreateNativeOrder(_ context.Context, outTradeNo string, _ int64, _ string, _ time.Time) (string, error) {
	return "weixin://wxpay/bizpayurl/up?pr=fake_" + short(outTradeNo), nil
}

func (f *FakeWeChatPayGateway) CloseOrder(_ context.Context, _ string) error { return nil }

// fakeWeChatNotifyBody is the simplified smoke-test payload accepted by the fake
// WeChat gateway. It is never used against a real WeChat callback.
type fakeWeChatNotifyBody struct {
	OutTradeNo    string `json:"outTradeNo"`
	TransactionID string `json:"transactionId"`
	AmountCent    int64  `json:"amountCent"`
	Success       bool   `json:"success"`
}

// VerifyNotification reads an optional simplified JSON body so internal notify
// endpoints can be exercised at the HTTP level in dev/acceptance smoke tests.
// An empty body preserves the original zero-value behaviour.
func (f *FakeWeChatPayGateway) VerifyNotification(req *http.Request) (WeChatNotification, error) {
	body, ok, err := readFakeBody(req)
	if err != nil {
		return WeChatNotification{}, err
	}
	if !ok {
		return WeChatNotification{}, nil
	}
	var b fakeWeChatNotifyBody
	if err := json.Unmarshal(body, &b); err != nil {
		return WeChatNotification{}, err
	}
	return WeChatNotification{
		OutTradeNo:    b.OutTradeNo,
		TransactionID: b.TransactionID,
		AmountCent:    b.AmountCent,
		Success:       b.Success,
	}, nil
}

func (f *FakeWeChatPayGateway) Refund(_ context.Context, _, outRefundNo string, _, _ int64) (string, error) {
	return "fake_refund_" + short(outRefundNo), nil
}

// FakeOfflineAcquirer is an offline acquirer for dev/tests.
type FakeOfflineAcquirer struct{}

// NewFakeOfflineAcquirer builds the fake acquirer.
func NewFakeOfflineAcquirer() *FakeOfflineAcquirer { return &FakeOfflineAcquirer{} }

func (f *FakeOfflineAcquirer) CreateDynamicQR(_ context.Context, outTradeNo string, _ int64, _ string, expiresAt time.Time) (OfflineQR, error) {
	return OfflineQR{
		AcquirerOrderNo: "fake_acq_" + short(outTradeNo),
		QRContent:       "https://pay.fake.example.com/qr/" + short(outTradeNo),
		ExpiresAt:       expiresAt,
	}, nil
}

// fakeOfflineNotifyBody is the simplified smoke-test payload accepted by the
// fake offline acquirer. It is never used against a real acquirer callback.
type fakeOfflineNotifyBody struct {
	OutTradeNo      string `json:"outTradeNo"`
	AcquirerOrderNo string `json:"acquirerOrderNo"`
	ExternalTradeNo string `json:"externalTradeNo"`
	Channel         string `json:"channel"`
	AmountCent      int64  `json:"amountCent"`
	Success         bool   `json:"success"`
}

// VerifyNotification reads an optional simplified JSON body so internal notify
// endpoints can be exercised at the HTTP level in dev/acceptance smoke tests.
// An empty body preserves the original zero-value behaviour.
func (f *FakeOfflineAcquirer) VerifyNotification(req *http.Request) (OfflineNotification, error) {
	body, ok, err := readFakeBody(req)
	if err != nil {
		return OfflineNotification{}, err
	}
	if !ok {
		return OfflineNotification{}, nil
	}
	var b fakeOfflineNotifyBody
	if err := json.Unmarshal(body, &b); err != nil {
		return OfflineNotification{}, err
	}
	return OfflineNotification{
		AcquirerOrderNo: b.AcquirerOrderNo,
		OutTradeNo:      b.OutTradeNo,
		ExternalTradeNo: b.ExternalTradeNo,
		Channel:         b.Channel,
		AmountCent:      b.AmountCent,
		Success:         b.Success,
	}, nil
}

func (f *FakeOfflineAcquirer) Refund(_ context.Context, acquirerOrderNo, _ string, _ int64) (string, error) {
	return "fake_acq_refund_" + short(acquirerOrderNo), nil
}

// readFakeBody drains an optional request body for the fakes. It returns
// ok=false when there is no body (nil request/body or empty content), so the
// caller can keep the original zero-value behaviour and existing tests that
// pass a nil request still pass.
func readFakeBody(req *http.Request) ([]byte, bool, error) {
	if req == nil || req.Body == nil {
		return nil, false, nil
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, false, err
	}
	if len(body) == 0 {
		return nil, false, nil
	}
	return body, true, nil
}

func short(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:4])
}
