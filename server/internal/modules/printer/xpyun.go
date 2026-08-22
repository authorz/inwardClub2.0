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
	"strings"
	"time"

	"github.com/inwardclub/server/internal/platform/config"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

const (
	xpyunBaseURL   = "https://open.xpyun.net/api/openapi/xprinter"
	xpyunBodyLimit = 1 << 20
)

// XpyunCredentials are shared by every store. UKey is server-only and is never
// included in a console response.
type XpyunCredentials struct {
	User    string
	UKey    string
	BaseURL string
}

// XpyunSettingsSource is implemented by headquarters global settings.
type XpyunSettingsSource interface {
	PrinterProviderSettings(ctx context.Context) (user, ukey, baseURL string, err error)
}

type xpyunCredentialsProvider interface {
	Credentials(ctx context.Context) (XpyunCredentials, error)
}

type staticXpyunCredentials struct{ value XpyunCredentials }

func (p staticXpyunCredentials) Credentials(context.Context) (XpyunCredentials, error) {
	return p.value, nil
}

type settingsXpyunCredentials struct {
	source   XpyunSettingsSource
	fallback config.XpyunConfig
}

func (p settingsXpyunCredentials) Credentials(ctx context.Context) (XpyunCredentials, error) {
	user, ukey, baseURL, err := p.source.PrinterProviderSettings(ctx)
	if err != nil {
		return XpyunCredentials{}, err
	}
	if strings.TrimSpace(user) == "" && strings.TrimSpace(ukey) == "" {
		user, ukey = p.fallback.User, p.fallback.UKey
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = p.fallback.BaseURL
	}
	return XpyunCredentials{User: user, UKey: ukey, BaseURL: baseURL}, nil
}

// XpyunPrinter implements receipt printing and the device-management endpoints
// documented by Xpyun. Tests may override baseURL to use an httptest server.
type XpyunPrinter struct {
	credentials xpyunCredentialsProvider
	baseURL     string
	http        *http.Client
	now         func() time.Time
}

// NewXpyunPrinter builds a client backed by static legacy environment settings.
func NewXpyunPrinter(cfg config.XpyunConfig) *XpyunPrinter {
	return newXpyunPrinter(staticXpyunCredentials{value: XpyunCredentials{
		User: cfg.User, UKey: cfg.UKey, BaseURL: cfg.BaseURL,
	}})
}

// NewXpyunPrinterWithSettings builds a client that reads headquarters settings
// for every call, with legacy environment values only as a compatibility fallback.
func NewXpyunPrinterWithSettings(source XpyunSettingsSource, fallback config.XpyunConfig) *XpyunPrinter {
	return newXpyunPrinter(settingsXpyunCredentials{source: source, fallback: fallback})
}

func newXpyunPrinter(credentials xpyunCredentialsProvider) *XpyunPrinter {
	return &XpyunPrinter{
		credentials: credentials,
		http:        &http.Client{Timeout: 10 * time.Second},
		now:         time.Now,
	}
}

// Select preserves the static constructor for isolated tests and tools.
func Select(cfg config.XpyunConfig, useFake bool) CloudPrinter {
	if useFake {
		return NewFakePrinter()
	}
	return NewXpyunPrinter(cfg)
}

// SelectWithSettings is the runtime constructor used by API and worker.
func SelectWithSettings(source XpyunSettingsSource, fallback config.XpyunConfig, useFake bool) CloudPrinter {
	if useFake {
		return NewFakePrinter()
	}
	return NewXpyunPrinterWithSettings(source, fallback)
}

type xpyunResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

type xpyunBatchResult struct {
	Success []string `json:"success"`
	Fail    []string `json:"fail"`
	FailMsg []string `json:"failMsg"`
}

func (p *XpyunPrinter) Print(ctx context.Context, job Job) error {
	if strings.TrimSpace(job.DeviceSN) == "" {
		return apperr.Invalid("printer job requires a device sn")
	}
	voice := 2
	if job.Silent {
		voice = 1
	}
	var orderID string
	return p.post(ctx, "print", map[string]any{
		"sn": job.DeviceSN, "content": job.Content, "copies": 1, "voice": voice, "mode": 0,
	}, &orderID)
}

func (p *XpyunPrinter) AddPrinter(ctx context.Context, sn, name string) error {
	var result xpyunBatchResult
	if err := p.post(ctx, "addPrinters", map[string]any{
		"items": []map[string]string{{"sn": sn, "name": name}},
	}, &result); err != nil {
		return err
	}
	if !containsSN(result.Success, sn) {
		return providerResultError("添加打印机", sn, result)
	}
	return nil
}

func (p *XpyunPrinter) DeletePrinter(ctx context.Context, sn string) error {
	var result xpyunBatchResult
	if err := p.post(ctx, "delPrinters", map[string]any{"snlist": []string{sn}}, &result); err != nil {
		if providerErrorCode(err) == 1002 {
			return nil
		}
		return err
	}
	if !containsSN(result.Success, sn) {
		if batchOnlyHasCode(result, "1002") {
			return nil
		}
		return providerResultError("删除打印机", sn, result)
	}
	return nil
}

func (p *XpyunPrinter) UpdatePrinterName(ctx context.Context, sn, name string) error {
	var success bool
	if err := p.post(ctx, "updPrinter", map[string]any{"sn": sn, "name": name}, &success); err != nil {
		return err
	}
	if !success {
		return apperr.Invalid("芯烨云修改打印机信息失败")
	}
	return nil
}

func (p *XpyunPrinter) ClearQueue(ctx context.Context, sn string) error {
	var success bool
	if err := p.post(ctx, "delPrinterQueue", map[string]any{"sn": sn}, &success); err != nil {
		return err
	}
	if !success {
		return apperr.Invalid("芯烨云清空打印队列失败")
	}
	return nil
}

func (p *XpyunPrinter) SetVoice(ctx context.Context, sn string, voiceType int, volumeLevel *int) error {
	payload := map[string]any{"sn": sn, "voiceType": voiceType}
	if volumeLevel != nil {
		payload["volumeLevel"] = *volumeLevel
	}
	var success bool
	if err := p.post(ctx, "setVoiceType", payload, &success); err != nil {
		return err
	}
	if !success {
		return apperr.Invalid("芯烨云设置打印机语音失败")
	}
	return nil
}

func (p *XpyunPrinter) QueryOrderState(ctx context.Context, orderID string) (bool, error) {
	var printed bool
	err := p.post(ctx, "queryOrderState", map[string]any{"orderId": orderID}, &printed)
	return printed, err
}

func (p *XpyunPrinter) QueryOrderStatistics(ctx context.Context, sn string, date time.Time) (OrderStatistics, error) {
	var result OrderStatistics
	err := p.post(ctx, "queryOrderStatis", map[string]any{
		"sn": sn, "date": date.Format("2006-01-02"),
	}, &result)
	return result, err
}

func (p *XpyunPrinter) QueryStatuses(ctx context.Context, sns []string) (map[string]ProviderStatus, error) {
	result := make(map[string]ProviderStatus, len(sns))
	for start := 0; start < len(sns); start += 20 {
		end := start + 20
		if end > len(sns) {
			end = len(sns)
		}
		batch := sns[start:end]
		var values []int
		if err := p.post(ctx, "queryPrintersStatus", map[string]any{"snlist": batch}, &values); err != nil {
			return nil, err
		}
		if len(values) != len(batch) {
			return nil, apperr.New(apperr.CodeUnavailable, "芯烨云返回的打印机状态数量不正确")
		}
		for i, value := range values {
			result[batch[i]] = mapProviderStatus(value)
		}
	}
	return result, nil
}

func (p *XpyunPrinter) post(ctx context.Context, path string, payload map[string]any, target any) error {
	credentials, err := p.credentials.Credentials(ctx)
	if err != nil {
		return err
	}
	credentials.User = strings.TrimSpace(credentials.User)
	credentials.UKey = strings.TrimSpace(credentials.UKey)
	if credentials.User == "" || credentials.UKey == "" {
		return apperr.Invalid("请先在总后台全局设置中配置打印机开发者账号和密钥")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(credentials.BaseURL), "/")
	if p.baseURL != "" {
		baseURL = strings.TrimRight(p.baseURL, "/")
	}
	if baseURL == "" {
		baseURL = xpyunBaseURL
	}
	timestamp := strconv.FormatInt(p.now().Unix(), 10)
	payload["user"] = credentials.User
	payload["timestamp"] = timestamp
	payload["sign"] = xpyunSign(credentials.User, credentials.UKey, timestamp)
	raw, err := json.Marshal(payload)
	if err != nil {
		return apperr.Internal(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/"+path, bytes.NewReader(raw))
	if err != nil {
		return apperr.Internal(err)
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	resp, err := p.http.Do(req)
	if err != nil {
		return apperr.New(apperr.CodeUnavailable, "芯烨云接口暂不可用").WithCause(err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, xpyunBodyLimit))
	if err != nil {
		return apperr.New(apperr.CodeUnavailable, "读取芯烨云接口响应失败").WithCause(err)
	}
	if resp.StatusCode != http.StatusOK {
		return apperr.New(apperr.CodeUnavailable, "芯烨云接口响应异常").WithCause(
			fmt.Errorf("status=%d body=%s", resp.StatusCode, string(responseBody)),
		)
	}
	var out xpyunResponse
	if err := json.Unmarshal(responseBody, &out); err != nil {
		return apperr.New(apperr.CodeUnavailable, "芯烨云接口响应格式不正确").WithCause(err)
	}
	if out.Code != 0 {
		return apperr.Invalid(fmt.Sprintf("芯烨云接口拒绝请求：%s（%d）", out.Msg, out.Code)).
			WithDetails(map[string]any{"providerCode": out.Code})
	}
	if target == nil || len(out.Data) == 0 || string(out.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(out.Data, target); err != nil {
		return apperr.New(apperr.CodeUnavailable, "芯烨云接口数据格式不正确").WithCause(err)
	}
	return nil
}

func providerResultError(action, sn string, result xpyunBatchResult) error {
	detail := strings.Join(result.FailMsg, "、")
	if detail == "" {
		detail = strings.Join(result.Fail, "、")
	}
	if detail == "" {
		detail = sn
	}
	return apperr.Invalid(fmt.Sprintf("芯烨云%s失败：%s", action, detail))
}

func providerErrorCode(err error) int {
	details, ok := apperr.From(err).Details.(map[string]any)
	if !ok {
		return 0
	}
	code, _ := details["providerCode"].(int)
	return code
}

func batchOnlyHasCode(result xpyunBatchResult, code string) bool {
	if len(result.FailMsg) == 0 {
		return false
	}
	for _, message := range result.FailMsg {
		if !strings.HasSuffix(message, ":"+code) {
			return false
		}
	}
	return true
}

func containsSN(items []string, sn string) bool {
	for _, item := range items {
		if item == sn {
			return true
		}
	}
	return false
}

func mapProviderStatus(value int) ProviderStatus {
	switch value {
	case 0:
		return ProviderStatusOffline
	case 1:
		return ProviderStatusOnline
	case 2:
		return ProviderStatusAbnormal
	default:
		return ProviderStatusUnknown
	}
}

func xpyunSign(user, ukey, timestamp string) string {
	sum := sha1.Sum([]byte(user + ukey + timestamp))
	return hex.EncodeToString(sum[:])
}

var _ CloudPrinter = (*XpyunPrinter)(nil)
