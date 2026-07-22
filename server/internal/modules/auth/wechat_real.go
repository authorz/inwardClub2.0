package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	// defaultWeChatBaseURL is the WeChat mini-program open API host. Tests point
	// baseURL at an httptest server; production uses this constant.
	defaultWeChatBaseURL = "https://api.weixin.qq.com"
	// tokenRefreshMargin refreshes the cached app access token a little before it
	// actually expires so an in-flight phone request never races WeChat's expiry.
	tokenRefreshMargin = 5 * time.Minute
	// weChatHTTPTimeout bounds each WeChat call so a stalled upstream never blocks
	// a login or phone-binding request indefinitely.
	weChatHTTPTimeout = 10 * time.Second
)

// WeChatHTTPClient is the real WeChat mini-program client. It is used only when
// USE_FAKE_ADAPTERS=false; local dev and every test default to FakeWeChatClient.
//
// It implements WeChatClient with two upstream calls:
//   - Code2Session  -> GET  /sns/jscode2session            (login exchange)
//   - GetPhoneNumber -> POST /wxa/business/getuserphonenumber (phone binding)
//
// getuserphonenumber requires an app access token, fetched from the multi-
// instance-safe /cgi-bin/stable_token endpoint and cached in-process until
// shortly before it expires.
type WeChatHTTPClient struct {
	appID     string
	appSecret string
	baseURL   string
	http      *http.Client
	now       func() time.Time

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewWeChatClient builds the real WeChat client from the mini-program app id and
// secret (validated as required in config when USE_FAKE_ADAPTERS=false).
func NewWeChatClient(appID, appSecret string) *WeChatHTTPClient {
	return &WeChatHTTPClient{
		appID:     appID,
		appSecret: appSecret,
		baseURL:   defaultWeChatBaseURL,
		http:      &http.Client{Timeout: weChatHTTPTimeout},
		now:       time.Now,
	}
}

type code2SessionResponse struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// Code2Session exchanges a mini-program login code for the member's openid,
// unionid and session key via WeChat jscode2session.
func (c *WeChatHTTPClient) Code2Session(ctx context.Context, code string) (WeChatSession, error) {
	q := url.Values{}
	q.Set("appid", c.appID)
	q.Set("secret", c.appSecret)
	q.Set("js_code", code)
	q.Set("grant_type", "authorization_code")
	endpoint := c.baseURL + "/sns/jscode2session?" + q.Encode()

	var resp code2SessionResponse
	if err := c.getJSON(ctx, endpoint, &resp); err != nil {
		return WeChatSession{}, err
	}
	if resp.ErrCode != 0 {
		return WeChatSession{}, weChatAPIError("jscode2session", resp.ErrCode, resp.ErrMsg)
	}
	if resp.OpenID == "" {
		return WeChatSession{}, fmt.Errorf("wechat jscode2session returned empty openid")
	}
	return WeChatSession{
		OpenID:     resp.OpenID,
		UnionID:    resp.UnionID,
		SessionKey: resp.SessionKey,
	}, nil
}

type getPhoneResponse struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	PhoneInfo struct {
		PhoneNumber     string `json:"phoneNumber"`
		PurePhoneNumber string `json:"purePhoneNumber"`
		CountryCode     string `json:"countryCode"`
	} `json:"phone_info"`
}

// GetPhoneNumber resolves a phone number from a mini-program phone code via
// WeChat getuserphonenumber. It uses a cached app access token and, if WeChat
// reports the token invalid, force-refreshes the token once and retries.
func (c *WeChatHTTPClient) GetPhoneNumber(ctx context.Context, phoneCode string) (string, error) {
	token, err := c.accessTokenValue(ctx, false)
	if err != nil {
		return "", err
	}
	resp, err := c.callGetPhone(ctx, token, phoneCode)
	if err != nil {
		return "", err
	}
	if isTokenError(resp.ErrCode) {
		// The cached token was invalidated out of band; refresh once and retry.
		if token, err = c.accessTokenValue(ctx, true); err != nil {
			return "", err
		}
		if resp, err = c.callGetPhone(ctx, token, phoneCode); err != nil {
			return "", err
		}
	}
	if resp.ErrCode != 0 {
		return "", weChatAPIError("getuserphonenumber", resp.ErrCode, resp.ErrMsg)
	}
	// Prefer the country-code-free number so it matches the 11-digit CN format the
	// member store and masking assume; fall back to phoneNumber if absent.
	phone := resp.PhoneInfo.PurePhoneNumber
	if phone == "" {
		phone = resp.PhoneInfo.PhoneNumber
	}
	if phone == "" {
		return "", fmt.Errorf("wechat getuserphonenumber returned empty phone")
	}
	return phone, nil
}

func (c *WeChatHTTPClient) callGetPhone(ctx context.Context, token, phoneCode string) (getPhoneResponse, error) {
	endpoint := c.baseURL + "/wxa/business/getuserphonenumber?access_token=" + url.QueryEscape(token)
	var resp getPhoneResponse
	if err := c.postJSON(ctx, endpoint, map[string]string{"code": phoneCode}, &resp); err != nil {
		return getPhoneResponse{}, err
	}
	return resp, nil
}

type stableTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

// accessTokenValue returns a valid app access token, fetching a fresh one when
// the cache is empty/expired or forceRefresh is set. The lock is held across the
// fetch so concurrent callers reuse a single token rather than each fetching.
func (c *WeChatHTTPClient) accessTokenValue(ctx context.Context, forceRefresh bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !forceRefresh && c.accessToken != "" && c.now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}
	endpoint := c.baseURL + "/cgi-bin/stable_token"
	body := map[string]any{
		"grant_type":    "client_credential",
		"appid":         c.appID,
		"secret":        c.appSecret,
		"force_refresh": forceRefresh,
	}
	var resp stableTokenResponse
	if err := c.postJSON(ctx, endpoint, body, &resp); err != nil {
		return "", err
	}
	if resp.ErrCode != 0 {
		return "", weChatAPIError("stable_token", resp.ErrCode, resp.ErrMsg)
	}
	if resp.AccessToken == "" {
		return "", fmt.Errorf("wechat stable_token returned empty access_token")
	}
	c.accessToken = resp.AccessToken
	c.tokenExpiry = c.now().Add(time.Duration(resp.ExpiresIn)*time.Second - tokenRefreshMargin)
	return c.accessToken, nil
}

func (c *WeChatHTTPClient) getJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *WeChatHTTPClient) postJSON(ctx context.Context, endpoint string, payload, out any) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

// do executes the request and decodes a JSON body. WeChat returns HTTP 200 with
// an errcode field for logical failures, so the decoded struct carries the
// application-level error; only transport/non-200 failures surface here.
func (c *WeChatHTTPClient) do(req *http.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wechat api %s: unexpected status %d", req.URL.Path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("wechat api %s: decode response: %w", req.URL.Path, err)
	}
	return nil
}

// isTokenError reports whether an errcode signals an invalid/expired access
// token, which warrants a single forced refresh and retry.
func isTokenError(code int) bool {
	switch code {
	case 40001, 40014, 42001:
		return true
	default:
		return false
	}
}

func weChatAPIError(op string, code int, msg string) error {
	return fmt.Errorf("wechat %s failed: errcode=%d errmsg=%q", op, code, msg)
}
