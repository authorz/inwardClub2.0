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
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/inwardclub/server/internal/platform/config"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// wechatAPIBaseURL is the WeChat Pay v3 production host. Overridden in tests.
const wechatAPIBaseURL = "https://api.mch.weixin.qq.com"

// notifyMaxSkew bounds how far a notify timestamp may drift from now before the
// callback is rejected as a possible replay (WeChat's recommended window).
const notifyMaxSkew = 5 * time.Minute

// WeChatPayClient is the single wrapper around the WeChat Pay v3 REST API. It is
// used only when USE_FAKE_ADAPTERS=false; tests and local dev use
// FakeWeChatPayGateway. All request signing, response-signature verification and
// resource decryption live here so no crypto detail leaks into business code.
type WeChatPayClient struct {
	appID     string
	mchID     string
	serialNo  string          // merchant certificate serial (signs outbound requests)
	apiV3Key  []byte          // AES-256-GCM key for notify resource decryption
	notifyURL string          // where WeChat posts payment callbacks
	privKey   *rsa.PrivateKey // merchant API private key (signs outbound requests)

	// pubKey verifies inbound notify signatures. pubKeyID is matched against the
	// Wechatpay-Serial header so a callback signed by an unknown key is rejected.
	pubKey   *rsa.PublicKey
	pubKeyID string

	// Injectable for tests; production defaults are set by NewWeChatPayClient.
	baseURL string
	http    *http.Client
	now     func() time.Time
}

// NewWeChatPayClient builds the real gateway from config, loading the merchant
// private key and the platform public key from disk. It returns an error (rather
// than panicking) so bootstrap can surface misconfiguration at startup.
func NewWeChatPayClient(cfg config.WeChatConfig) (*WeChatPayClient, error) {
	if len(cfg.PayAPIV3Key) != 32 {
		return nil, fmt.Errorf("wechat pay apiV3 key must be 32 bytes, got %d", len(cfg.PayAPIV3Key))
	}
	priv, err := loadRSAPrivateKey(cfg.PayPrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load wechat pay private key: %w", err)
	}
	pub, err := loadRSAPublicKey(cfg.PayPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load wechat pay public key: %w", err)
	}
	return &WeChatPayClient{
		appID:     cfg.MiniAppID,
		mchID:     cfg.PayMchID,
		serialNo:  cfg.PayMchCertSerialNo,
		apiV3Key:  []byte(cfg.PayAPIV3Key),
		notifyURL: cfg.PayNotifyURL,
		privKey:   priv,
		pubKey:    pub,
		pubKeyID:  cfg.PayPublicKeyID,
		baseURL:   wechatAPIBaseURL,
		http:      &http.Client{Timeout: 10 * time.Second},
		now:       time.Now,
	}, nil
}

// jsapiPrepayRequest / nativeOrderRequest / closeOrderRequest / refundRequest
// mirror the v3 request bodies. Only the fields used by this project are
// populated.
type jsapiPrepayRequest struct {
	AppID       string     `json:"appid"`
	MchID       string     `json:"mchid"`
	Description string     `json:"description"`
	OutTradeNo  string     `json:"out_trade_no"`
	NotifyURL   string     `json:"notify_url"`
	Amount      amountBody `json:"amount"`
	Payer       payerBody  `json:"payer"`
}

type amountBody struct {
	Total    int64  `json:"total"`
	Currency string `json:"currency"`
}

type payerBody struct {
	OpenID string `json:"openid"`
}

type nativeOrderRequest struct {
	AppID       string     `json:"appid"`
	MchID       string     `json:"mchid"`
	Description string     `json:"description"`
	OutTradeNo  string     `json:"out_trade_no"`
	TimeExpire  string     `json:"time_expire"`
	NotifyURL   string     `json:"notify_url"`
	Amount      amountBody `json:"amount"`
}

type closeOrderRequest struct {
	MchID string `json:"mchid"`
}

type refundRequest struct {
	OutTradeNo  string           `json:"out_trade_no"`
	OutRefundNo string           `json:"out_refund_no"`
	Amount      refundAmountBody `json:"amount"`
}

type refundAmountBody struct {
	Refund   int64  `json:"refund"`
	Total    int64  `json:"total"`
	Currency string `json:"currency"`
}

// CreateJSAPIPrepay creates a JSAPI transaction and returns the parameters the
// mini program passes to wx.requestPayment (with the RSA paySign computed here).
func (c *WeChatPayClient) CreateJSAPIPrepay(ctx context.Context, outTradeNo string, amountCent int64, openID, description string) (WeChatPrepay, error) {
	body := jsapiPrepayRequest{
		AppID:       c.appID,
		MchID:       c.mchID,
		Description: description,
		OutTradeNo:  outTradeNo,
		NotifyURL:   c.notifyURL,
		Amount:      amountBody{Total: amountCent, Currency: "CNY"},
		Payer:       payerBody{OpenID: openID},
	}
	respBody, err := c.doRequest(ctx, http.MethodPost, "/v3/pay/transactions/jsapi", body)
	if err != nil {
		return WeChatPrepay{}, err
	}
	var out struct {
		PrepayID string `json:"prepay_id"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return WeChatPrepay{}, apperr.Internal(fmt.Errorf("decode jsapi response: %w", err))
	}
	if out.PrepayID == "" {
		return WeChatPrepay{}, apperr.Internal(fmt.Errorf("jsapi response missing prepay_id"))
	}

	ts := strconv.FormatInt(c.now().Unix(), 10)
	nonce, err := randomNonce()
	if err != nil {
		return WeChatPrepay{}, apperr.Internal(err)
	}
	pkg := "prepay_id=" + out.PrepayID
	// wx.requestPayment sign message: appId\ntimeStamp\nnonceStr\npackage\n.
	paySign, err := c.sign(c.appID + "\n" + ts + "\n" + nonce + "\n" + pkg + "\n")
	if err != nil {
		return WeChatPrepay{}, apperr.Internal(err)
	}
	return WeChatPrepay{
		PrepayID:  out.PrepayID,
		NonceStr:  nonce,
		Package:   pkg,
		SignType:  "RSA",
		PaySign:   paySign,
		Timestamp: ts,
	}, nil
}

// CreateNativeOrder creates a fixed-amount Native transaction. WeChat returns
// code_url, which the store console converts into a QR image for the customer.
func (c *WeChatPayClient) CreateNativeOrder(ctx context.Context, outTradeNo string, amountCent int64, description string, expiresAt time.Time) (string, error) {
	body := nativeOrderRequest{
		AppID:       c.appID,
		MchID:       c.mchID,
		Description: description,
		OutTradeNo:  outTradeNo,
		TimeExpire:  expiresAt.Format(time.RFC3339),
		NotifyURL:   c.notifyURL,
		Amount:      amountBody{Total: amountCent, Currency: "CNY"},
	}
	respBody, err := c.doRequest(ctx, http.MethodPost, "/v3/pay/transactions/native", body)
	if err != nil {
		return "", err
	}
	var out struct {
		CodeURL string `json:"code_url"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", apperr.Internal(fmt.Errorf("decode native response: %w", err))
	}
	if out.CodeURL == "" {
		return "", apperr.Internal(fmt.Errorf("native response missing code_url"))
	}
	return out.CodeURL, nil
}

// CloseOrder closes an unpaid WeChat order by merchant order number.
func (c *WeChatPayClient) CloseOrder(ctx context.Context, outTradeNo string) error {
	path := "/v3/pay/transactions/out-trade-no/" + outTradeNo + "/close"
	_, err := c.doRequest(ctx, http.MethodPost, path, closeOrderRequest{MchID: c.mchID})
	return err
}

// Refund submits a domestic refund and returns the WeChat refund id.
func (c *WeChatPayClient) Refund(ctx context.Context, outTradeNo, outRefundNo string, amountCent, totalCent int64) (string, error) {
	body := refundRequest{
		OutTradeNo:  outTradeNo,
		OutRefundNo: outRefundNo,
		Amount:      refundAmountBody{Refund: amountCent, Total: totalCent, Currency: "CNY"},
	}
	respBody, err := c.doRequest(ctx, http.MethodPost, "/v3/refund/domestic/refunds", body)
	if err != nil {
		return "", err
	}
	var out struct {
		RefundID string `json:"refund_id"`
	}
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", apperr.Internal(fmt.Errorf("decode refund response: %w", err))
	}
	if out.RefundID == "" {
		return "", apperr.Internal(fmt.Errorf("refund response missing refund_id"))
	}
	return out.RefundID, nil
}

// notifyEnvelope / notifyResource model the notify body and its encrypted
// resource (AEAD_AES_256_GCM).
type notifyEnvelope struct {
	EventType string `json:"event_type"`
	Resource  struct {
		Algorithm      string `json:"algorithm"`
		Ciphertext     string `json:"ciphertext"`
		AssociatedData string `json:"associated_data"`
		Nonce          string `json:"nonce"`
	} `json:"resource"`
}

type notifyResource struct {
	OutTradeNo    string `json:"out_trade_no"`
	TransactionID string `json:"transaction_id"`
	TradeState    string `json:"trade_state"`
	Amount        struct {
		Total int64 `json:"total"`
	} `json:"amount"`
}

// VerifyNotification verifies the callback signature against the configured
// platform public key, guards against replay via the timestamp window, then
// decrypts the resource and maps it to a WeChatNotification. A verification
// failure is Unauthenticated so the handler returns a non-2xx and WeChat retries.
func (c *WeChatPayClient) VerifyNotification(req *http.Request) (WeChatNotification, error) {
	if req == nil || req.Body == nil {
		return WeChatNotification{}, apperr.Invalid("empty wechat notify request")
	}
	ts := req.Header.Get("Wechatpay-Timestamp")
	nonce := req.Header.Get("Wechatpay-Nonce")
	sig := req.Header.Get("Wechatpay-Signature")
	serial := req.Header.Get("Wechatpay-Serial")
	if ts == "" || nonce == "" || sig == "" {
		return WeChatNotification{}, apperr.Unauthenticated("missing wechat notify signature headers")
	}
	if c.pubKeyID != "" && serial != "" && serial != c.pubKeyID {
		return WeChatNotification{}, apperr.Unauthenticated("unknown wechat pay signing serial")
	}

	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return WeChatNotification{}, apperr.Unauthenticated("invalid wechat notify timestamp")
	}
	if skew := c.now().Unix() - tsInt; skew > int64(notifyMaxSkew.Seconds()) || skew < -int64(notifyMaxSkew.Seconds()) {
		return WeChatNotification{}, apperr.Unauthenticated("stale wechat notify timestamp")
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return WeChatNotification{}, apperr.Internal(err)
	}
	// Signed message: timestamp\nnonce\nbody\n.
	if err := c.verify(ts+"\n"+nonce+"\n"+string(body)+"\n", sig); err != nil {
		return WeChatNotification{}, apperr.Unauthenticated("invalid wechat notify signature")
	}

	var env notifyEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return WeChatNotification{}, apperr.Invalid("invalid wechat notify body")
	}
	plain, err := c.decryptResource(env.Resource.Nonce, env.Resource.AssociatedData, env.Resource.Ciphertext)
	if err != nil {
		return WeChatNotification{}, apperr.Unauthenticated("cannot decrypt wechat notify resource")
	}
	var res notifyResource
	if err := json.Unmarshal(plain, &res); err != nil {
		return WeChatNotification{}, apperr.Invalid("invalid wechat notify resource")
	}
	return WeChatNotification{
		OutTradeNo:    res.OutTradeNo,
		TransactionID: res.TransactionID,
		AmountCent:    res.Amount.Total,
		Success:       res.TradeState == "SUCCESS",
	}, nil
}

// doRequest signs and sends a v3 request, returning the raw success body. A
// non-2xx response is mapped to an application error carrying WeChat's code.
func (c *WeChatPayClient) doRequest(ctx context.Context, method, path string, reqBody any) ([]byte, error) {
	var payload []byte
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		payload = b
	}

	auth, err := c.authorization(method, path, payload)
	if err != nil {
		return nil, apperr.Internal(err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, apperr.Internal(err)
	}
	httpReq.Header.Set("Authorization", auth)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "inwardclub-server")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respBody, &e)
		return nil, apperr.Internal(fmt.Errorf("wechat pay %s %s: http %d code=%s message=%s", method, path, resp.StatusCode, e.Code, e.Message))
	}
	return respBody, nil
}

// authorization builds the WECHATPAY2-SHA256-RSA2048 Authorization header value.
func (c *WeChatPayClient) authorization(method, path string, body []byte) (string, error) {
	nonce, err := randomNonce()
	if err != nil {
		return "", err
	}
	ts := strconv.FormatInt(c.now().Unix(), 10)
	// Signed message: method\nurl\ntimestamp\nnonce\nbody\n.
	sig, err := c.sign(method + "\n" + path + "\n" + ts + "\n" + nonce + "\n" + string(body) + "\n")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",timestamp="%s",serial_no="%s",signature="%s"`,
		c.mchID, nonce, ts, c.serialNo, sig,
	), nil
}

// sign computes a base64 SHA256withRSA (RSASSA-PKCS1-v1_5) signature.
func (c *WeChatPayClient) sign(message string) (string, error) {
	digest := sha256.Sum256([]byte(message))
	sig, err := rsa.SignPKCS1v15(rand.Reader, c.privKey, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// verify checks a base64 SHA256withRSA signature against the platform public key.
func (c *WeChatPayClient) verify(message, signatureB64 string) error {
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(c.pubKey, crypto.SHA256, digest[:], sig)
}

// decryptResource opens the AEAD_AES_256_GCM resource envelope with the apiV3 key.
func (c *WeChatPayClient) decryptResource(nonce, associatedData, ciphertextB64 string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(c.apiV3Key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, []byte(nonce), ciphertext, []byte(associatedData))
}

// randomNonce returns a 32-char hex nonce for request/pay signing.
func randomNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// loadRSAPrivateKey reads a PEM private key (PKCS#8 or PKCS#1).
func loadRSAPrivateKey(path string) (*rsa.PrivateKey, error) {
	block, err := readPEM(path)
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA")
	}
	return key, nil
}

// loadRSAPublicKey reads a PEM RSA public key (PKIX) or a certificate PEM,
// supporting both WeChat Pay public-key mode and platform-certificate mode.
func loadRSAPublicKey(path string) (*rsa.PublicKey, error) {
	block, err := readPEM(path)
	if err != nil {
		return nil, err
	}
	if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		if key, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			return key, nil
		}
		return nil, fmt.Errorf("certificate public key is not RSA")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	key, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA")
	}
	return key, nil
}

func readPEM(path string) (*pem.Block, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil {
		return nil, fmt.Errorf("no PEM block in %s", path)
	}
	return block, nil
}

// compile-time assertion: the real client satisfies the gateway interface.
var _ WeChatPayGateway = (*WeChatPayClient)(nil)
