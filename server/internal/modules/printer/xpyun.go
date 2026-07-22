package printer

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/inwardclub/server/internal/platform/config"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

// xpyunBaseURL is the Xpyun (芯烨云) cloud-printer open-API root. It is fixed for
// the service; tests override the unexported baseURL field to point at a local
// server.
const xpyunBaseURL = "https://open.xpyun.net/api/openapi/xprinter"

// xpyunBodyLimit bounds how much of a response body is read.
const xpyunBodyLimit = 1 << 20

// XpyunPrinter is the real cloud-printer adapter (Xpyun open API), selected only
// when USE_FAKE_ADAPTERS=false; FakePrinter keeps dev and tests offline. Account
// auth is the user + user key from config; each request is signed
// SHA1(user + userKey + timestamp), which Xpyun's protocol mandates.
type XpyunPrinter struct {
	cfg config.XpyunConfig

	// Injectable for tests; production defaults are set by NewXpyunPrinter.
	baseURL string
	http    *http.Client
	now     func() time.Time
}

// NewXpyunPrinter builds the real printer client from config.
func NewXpyunPrinter(cfg config.XpyunConfig) *XpyunPrinter {
	return &XpyunPrinter{
		cfg:     cfg,
		baseURL: xpyunBaseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
		now:     time.Now,
	}
}

// Select returns the real Xpyun printer when fakes are disabled, otherwise the
// in-process fake. It is the config-driven seam the worker consumes so
// print:receipt jobs run against the real provider in production and stay
// offline in dev/tests.
func Select(cfg config.XpyunConfig, useFake bool) Printer {
	if useFake {
		return NewFakePrinter()
	}
	return NewXpyunPrinter(cfg)
}

// xpyunPrintRequest is the /print request body. Content carries the fully
// rendered receipt (with Xpyun markup); Job.Template is resolved into Content
// upstream, so the print call itself takes no template. Voice is the buzzer
// volume and Mode 0 is a normal single-slip print.
type xpyunPrintRequest struct {
	User      string `json:"user"`
	Timestamp string `json:"timestamp"`
	Sign      string `json:"sign"`
	SN        string `json:"sn"`
	Content   string `json:"content"`
	Copies    int    `json:"copies"`
	Voice     int    `json:"voice"`
	Mode      int    `json:"mode"`
}

// xpyunResponse is the shared Xpyun reply envelope; code 0 means accepted.
type xpyunResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// Print submits one receipt to the device SN through the Xpyun open API.
func (p *XpyunPrinter) Print(ctx context.Context, job Job) error {
	if job.DeviceSN == "" {
		return apperr.Invalid("printer job requires a device sn")
	}
	ts := strconv.FormatInt(p.now().Unix(), 10)
	reqBody := xpyunPrintRequest{
		User:      p.cfg.User,
		Timestamp: ts,
		Sign:      xpyunSign(p.cfg.User, p.cfg.UKey, ts),
		SN:        job.DeviceSN,
		Content:   job.Content,
		Copies:    1,
		Voice:     2,
		Mode:      0,
	}
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/print", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, xpyunBodyLimit))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("xpyun print: unexpected status %d: %s", resp.StatusCode, string(respBody))
	}
	var out xpyunResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return fmt.Errorf("xpyun print: decode response: %w", err)
	}
	if out.Code != 0 {
		return fmt.Errorf("xpyun print rejected: code=%d msg=%s", out.Code, out.Msg)
	}
	return nil
}

// xpyunSign is the lower-hex SHA1 of user+userKey+timestamp, per Xpyun's spec.
func xpyunSign(user, ukey, ts string) string {
	sum := sha1.Sum([]byte(user + ukey + ts))
	return hex.EncodeToString(sum[:])
}

// compile-time assertion: the real client satisfies the printer interface.
var _ Printer = (*XpyunPrinter)(nil)
