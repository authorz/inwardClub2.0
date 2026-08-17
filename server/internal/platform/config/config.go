// Package config loads runtime configuration from .env, an optional YAML base
// file (CONFIG_FILE), and environment variables. Secrets never carry compiled
// defaults; only non-sensitive operational values do.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config is the fully-resolved application configuration shared by every
// entrypoint (api, worker, migrate, reconcile).
type Config struct {
	AppEnv   string `yaml:"appEnv"`
	LogLevel string `yaml:"logLevel"`
	HTTPAddr string `yaml:"httpAddr"`

	MySQLDSN  string `yaml:"mysqlDsn"`
	RedisAddr string `yaml:"redisAddr"`

	Business BusinessConfig `yaml:"business"`

	// UseFakeAdapters forces in-process fakes for WeChat, Qiniu, printer and the
	// offline acquirer so local dev and tests never touch external networks.
	UseFakeAdapters bool `yaml:"useFakeAdapters"`

	// Per-adapter overrides let a single external service go real while the rest
	// stay faked (e.g. real WeChat login+pay while the offline acquirer/printer
	// are still being configured). Empty/false → the adapter follows
	// UseFakeAdapters. Only WeChat login and pay have overrides today.
	WeChatLoginUseReal bool `yaml:"wechatLoginUseReal"`
	WeChatPayUseReal   bool `yaml:"wechatPayUseReal"`
	PaymentDebugMode   bool `yaml:"paymentDebugMode"`

	JWT                  JWTConfig                  `yaml:"jwt"`
	CredentialEncryption CredentialEncryptionConfig `yaml:"credentialEncryption"`
	WeChat               WeChatConfig               `yaml:"wechat"`
	Offline              OfflineConfig              `yaml:"offlineAcquirer"`
	Qiniu                QiniuConfig                `yaml:"qiniu"`
	Xpyun                XpyunConfig                `yaml:"xpyun"`
}

// JWTConfig holds signing material and token lifetimes. Audiences are fixed in
// the authn package; only the signing key and TTLs are configurable.
type JWTConfig struct {
	SigningKey string        `yaml:"signingKey"`
	Issuer     string        `yaml:"issuer"`
	AccessTTL  time.Duration `yaml:"accessTtl"`
	RefreshTTL time.Duration `yaml:"refreshTtl"`
}

// CredentialEncryptionConfig holds the RSA private key used to encrypt
// password confirmations in the two browser consoles.
type CredentialEncryptionConfig struct {
	PrivateKeyPath string `yaml:"privateKeyPath"`
}

type WeChatConfig struct {
	MiniAppID          string `yaml:"miniAppId"`
	MiniAppSecret      string `yaml:"miniAppSecret"`
	PayMchID           string `yaml:"payMchId"`
	PayMchCertSerialNo string `yaml:"payMchCertSerialNo"`
	PayPrivateKeyPath  string `yaml:"payPrivateKeyPath"`
	PayAPIV3Key        string `yaml:"payApiV3Key"`
	PayNotifyURL       string `yaml:"payNotifyUrl"`
	// PayPublicKeyPath is the WeChat Pay platform public key (public-key mode)
	// used to verify notify signatures; a full platform certificate PEM is also
	// accepted. PayPublicKeyID is its identifier (matched against Wechatpay-Serial
	// on each callback so an unknown key is rejected).
	PayPublicKeyPath string `yaml:"payPublicKeyPath"`
	PayPublicKeyID   string `yaml:"payPublicKeyId"`
}

type OfflineConfig struct {
	Provider   string `yaml:"provider"`
	MerchantID string `yaml:"merchantId"`
	APIKey     string `yaml:"apiKey"`
	NotifyURL  string `yaml:"notifyUrl"`
	// BaseURL is the acquirer HTTP API root the real client posts to (create QR,
	// refund). Required only when the real acquirer is selected (fakes off); dev
	// and tests leave it empty and use the fake.
	BaseURL string `yaml:"baseUrl"`
}

type QiniuConfig struct {
	AccessKey         string        `yaml:"accessKey"`
	SecretKey         string        `yaml:"secretKey"`
	Bucket            string        `yaml:"bucket"`
	Region            string        `yaml:"region"`
	PublicDomain      string        `yaml:"publicDomain"`
	PrivateDomain     string        `yaml:"privateDomain"`
	UploadCallbackURL string        `yaml:"uploadCallbackUrl"`
	TokenTTL          time.Duration `yaml:"tokenTtl"`
	PrivateURLTTL     time.Duration `yaml:"privateUrlTtl"`
}

type XpyunConfig struct {
	User string `yaml:"user"`
	UKey string `yaml:"ukey"`
}

// BusinessConfig configures the "business day" reference used by day-bucketed
// features (the daily sign-in calendar day and the weekly/monthly leaderboard
// windows). TZ fixes the wall-clock zone the business day is measured in;
// NowRFC3339 optionally pins the current instant for dev/test/acceptance so the
// business day never drifts with the host clock. NowRFC3339 MUST be empty in
// production, where the clock falls through to the real host time.
type BusinessConfig struct {
	TZ         string `yaml:"tz"`
	NowRFC3339 string `yaml:"nowRfc3339"`
}

// Clock resolves the configured business day: a fixed location plus a source of
// "now". Now applies the business zone, so callers get the wall-clock day in
// that zone regardless of the host zone. In production now is time.Now; when
// BUSINESS_NOW_RFC3339 is set the clock is pinned to that instant.
type Clock struct {
	loc *time.Location
	now func() time.Time
}

// Now returns the current business instant, expressed in the business zone.
func (c *Clock) Now() time.Time { return c.now().In(c.loc) }

// Location returns the business zone.
func (c *Clock) Location() *time.Location { return c.loc }

// BusinessClock builds the business-day clock from configuration. It resolves
// the IANA zone (falling back to a fixed UTC+8 "CST" when the tz database is
// unavailable, matching the historical sign-in behaviour) and pins "now" when
// BUSINESS_NOW_RFC3339 is set.
func (c *Config) BusinessClock() (*Clock, error) {
	loc := loadBusinessLocation(c.Business.TZ)
	now := time.Now
	if s := c.Business.NowRFC3339; s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("parse BUSINESS_NOW_RFC3339: %w", err)
		}
		now = func() time.Time { return t }
	}
	return &Clock{loc: loc, now: now}, nil
}

func loadBusinessLocation(name string) *time.Location {
	if name == "" {
		name = "Asia/Shanghai"
	}
	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}
	return time.FixedZone("CST", 8*3600)
}

// Load resolves configuration: .env, YAML base (if CONFIG_FILE is set), then
// environment overrides. godotenv never overwrites an already-exported value,
// so process-level settings retain the highest priority.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	cfg := defaults()

	if path := os.Getenv("CONFIG_FILE"); path != "" {
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}
		if err := yaml.Unmarshal(raw, cfg); err != nil {
			return nil, fmt.Errorf("parse config file: %w", err)
		}
	}

	applyEnv(cfg)
	return cfg, nil
}

func defaults() *Config {
	return &Config{
		AppEnv:          "development",
		LogLevel:        "info",
		HTTPAddr:        ":8081",
		UseFakeAdapters: true,
		Business: BusinessConfig{
			TZ: "Asia/Shanghai",
		},
		JWT: JWTConfig{
			Issuer:     "inwardclub",
			AccessTTL:  2 * time.Hour,
			RefreshTTL: 30 * 24 * time.Hour,
		},
		Qiniu: QiniuConfig{
			TokenTTL:      10 * time.Minute,
			PrivateURLTTL: 5 * time.Minute,
		},
	}
}

func applyEnv(cfg *Config) {
	setStr(&cfg.AppEnv, "APP_ENV")
	setStr(&cfg.LogLevel, "LOG_LEVEL")
	setStr(&cfg.HTTPAddr, "HTTP_ADDR")
	setStr(&cfg.MySQLDSN, "MYSQL_DSN")
	setStr(&cfg.RedisAddr, "REDIS_ADDR")
	setBool(&cfg.UseFakeAdapters, "USE_FAKE_ADAPTERS")
	setBool(&cfg.WeChatLoginUseReal, "WECHAT_LOGIN_USE_REAL")
	setBool(&cfg.WeChatPayUseReal, "WECHAT_PAY_USE_REAL")
	setBool(&cfg.PaymentDebugMode, "PAYMENT_DEBUG_MODE")

	setStr(&cfg.Business.TZ, "BUSINESS_TZ")
	setStr(&cfg.Business.NowRFC3339, "BUSINESS_NOW_RFC3339")

	setStr(&cfg.JWT.SigningKey, "JWT_SIGNING_KEY")
	setStr(&cfg.JWT.Issuer, "JWT_ISSUER")
	setDur(&cfg.JWT.AccessTTL, "JWT_ACCESS_TTL")
	setDur(&cfg.JWT.RefreshTTL, "JWT_REFRESH_TTL")
	setStr(&cfg.CredentialEncryption.PrivateKeyPath, "CREDENTIAL_ENCRYPTION_PRIVATE_KEY_PATH")

	setStr(&cfg.WeChat.MiniAppID, "WECHAT_MINI_APP_ID")
	setStr(&cfg.WeChat.MiniAppSecret, "WECHAT_MINI_APP_SECRET")
	setStr(&cfg.WeChat.PayMchID, "WECHAT_PAY_MCH_ID")
	setStr(&cfg.WeChat.PayMchCertSerialNo, "WECHAT_PAY_MCH_CERT_SERIAL_NO")
	setStr(&cfg.WeChat.PayPrivateKeyPath, "WECHAT_PAY_PRIVATE_KEY_PATH")
	setStr(&cfg.WeChat.PayAPIV3Key, "WECHAT_PAY_API_V3_KEY")
	setStr(&cfg.WeChat.PayNotifyURL, "WECHAT_PAY_NOTIFY_URL")
	setStr(&cfg.WeChat.PayPublicKeyPath, "WECHAT_PAY_PUBLIC_KEY_PATH")
	setStr(&cfg.WeChat.PayPublicKeyID, "WECHAT_PAY_PUBLIC_KEY_ID")

	setStr(&cfg.Offline.Provider, "OFFLINE_ACQUIRER_PROVIDER")
	setStr(&cfg.Offline.MerchantID, "OFFLINE_ACQUIRER_MERCHANT_ID")
	setStr(&cfg.Offline.APIKey, "OFFLINE_ACQUIRER_API_KEY")
	setStr(&cfg.Offline.NotifyURL, "OFFLINE_ACQUIRER_NOTIFY_URL")
	setStr(&cfg.Offline.BaseURL, "OFFLINE_ACQUIRER_BASE_URL")

	setStr(&cfg.Qiniu.AccessKey, "QINIU_ACCESS_KEY")
	setStr(&cfg.Qiniu.SecretKey, "QINIU_SECRET_KEY")
	setStr(&cfg.Qiniu.Bucket, "QINIU_BUCKET")
	setStr(&cfg.Qiniu.Region, "QINIU_REGION")
	setStr(&cfg.Qiniu.PublicDomain, "QINIU_PUBLIC_DOMAIN")
	setStr(&cfg.Qiniu.PrivateDomain, "QINIU_PRIVATE_DOMAIN")
	setStr(&cfg.Qiniu.UploadCallbackURL, "QINIU_UPLOAD_CALLBACK_URL")
	setDurSeconds(&cfg.Qiniu.TokenTTL, "QINIU_TOKEN_TTL_SECONDS")
	setDurSeconds(&cfg.Qiniu.PrivateURLTTL, "QINIU_PRIVATE_URL_TTL_SECONDS")

	setStr(&cfg.Xpyun.User, "XPYUN_USER")
	setStr(&cfg.Xpyun.UKey, "XPYUN_UKEY")
}

// WeChatLoginReal reports whether the real WeChat login client should be used:
// either fakes are globally off, or the login adapter is explicitly overridden.
func (c *Config) WeChatLoginReal() bool {
	return !c.UseFakeAdapters || c.WeChatLoginUseReal
}

// WeChatPayReal reports whether the real WeChat Pay gateway should be used.
func (c *Config) WeChatPayReal() bool {
	return !c.UseFakeAdapters || c.WeChatPayUseReal
}

// WeChatPayAmountOverrideCent returns the real channel amount used in payment
// debug mode. Business orders and entitlements keep their original amounts.
// Production always returns zero so a stray debug flag cannot discount a live
// payment.
func (c *Config) WeChatPayAmountOverrideCent() int64 {
	if c.PaymentDebugMode && c.AppEnv != "production" {
		return 1
	}
	return 0
}

// Validate checks that required runtime fields are present. Fake adapters relax
// external-service requirements so local dev works without real credentials.
func (c *Config) Validate() error {
	if c.MySQLDSN == "" {
		return fmt.Errorf("MYSQL_DSN is required")
	}
	if c.JWT.SigningKey == "" {
		return fmt.Errorf("JWT_SIGNING_KEY is required")
	}
	if c.AppEnv == "production" && strings.TrimSpace(c.CredentialEncryption.PrivateKeyPath) == "" {
		return fmt.Errorf("CREDENTIAL_ENCRYPTION_PRIVATE_KEY_PATH is required in production")
	}
	// WeChat login/pay validate independently so one can go real while the
	// offline acquirer / printer stay faked (they gate on UseFakeAdapters).
	if c.WeChatLoginReal() {
		if err := c.WeChat.validateLogin(); err != nil {
			return err
		}
	}
	if c.WeChatPayReal() {
		if err := c.WeChat.validatePay(); err != nil {
			return err
		}
	}
	if !c.UseFakeAdapters {
		if c.Qiniu.AccessKey == "" || c.Qiniu.SecretKey == "" || c.Qiniu.Bucket == "" {
			return fmt.Errorf("qiniu credentials required when USE_FAKE_ADAPTERS=false")
		}
		if err := c.Offline.validate(); err != nil {
			return err
		}
		if err := c.Xpyun.validate(); err != nil {
			return err
		}
	}
	return nil
}

// validate ensures the offline aggregated-collection acquirer credentials the
// real HTTP client needs are present.
func (o OfflineConfig) validate() error {
	required := []struct {
		env string
		val string
	}{
		{"OFFLINE_ACQUIRER_PROVIDER", o.Provider},
		{"OFFLINE_ACQUIRER_MERCHANT_ID", o.MerchantID},
		{"OFFLINE_ACQUIRER_API_KEY", o.APIKey},
		{"OFFLINE_ACQUIRER_BASE_URL", o.BaseURL},
		{"OFFLINE_ACQUIRER_NOTIFY_URL", o.NotifyURL},
	}
	for _, r := range required {
		if r.val == "" {
			return fmt.Errorf("%s is required when USE_FAKE_ADAPTERS=false", r.env)
		}
	}
	return nil
}

// validate ensures the Xpyun cloud-printer credentials the real client needs are
// present.
func (x XpyunConfig) validate() error {
	if x.User == "" {
		return fmt.Errorf("XPYUN_USER is required when USE_FAKE_ADAPTERS=false")
	}
	if x.UKey == "" {
		return fmt.Errorf("XPYUN_UKEY is required when USE_FAKE_ADAPTERS=false")
	}
	return nil
}

// validateLogin ensures the mini-program login credentials used by the real
// WeChat login + phone-resolver client (jscode2session / getuserphonenumber)
// are present.
func (w WeChatConfig) validateLogin() error {
	if w.MiniAppID == "" {
		return fmt.Errorf("WECHAT_MINI_APP_ID is required when USE_FAKE_ADAPTERS=false")
	}
	if w.MiniAppSecret == "" {
		return fmt.Errorf("WECHAT_MINI_APP_SECRET is required when USE_FAKE_ADAPTERS=false")
	}
	return nil
}

// validatePay ensures the WeChat Pay merchant credentials needed by the real
// gateway are present. The APIv3 key must be exactly 32 bytes (AES-256-GCM).
func (w WeChatConfig) validatePay() error {
	required := []struct {
		env string
		val string
	}{
		{"WECHAT_MINI_APP_ID", w.MiniAppID},
		{"WECHAT_PAY_MCH_ID", w.PayMchID},
		{"WECHAT_PAY_MCH_CERT_SERIAL_NO", w.PayMchCertSerialNo},
		{"WECHAT_PAY_PRIVATE_KEY_PATH", w.PayPrivateKeyPath},
		{"WECHAT_PAY_API_V3_KEY", w.PayAPIV3Key},
		{"WECHAT_PAY_NOTIFY_URL", w.PayNotifyURL},
		{"WECHAT_PAY_PUBLIC_KEY_PATH", w.PayPublicKeyPath},
		{"WECHAT_PAY_PUBLIC_KEY_ID", w.PayPublicKeyID},
	}
	for _, r := range required {
		if r.val == "" {
			return fmt.Errorf("%s is required when USE_FAKE_ADAPTERS=false", r.env)
		}
	}
	if len(w.PayAPIV3Key) != 32 {
		return fmt.Errorf("WECHAT_PAY_API_V3_KEY must be 32 bytes")
	}
	return nil
}

func setStr(dst *string, key string) {
	if v, ok := os.LookupEnv(key); ok {
		*dst = v
	}
}

func setBool(dst *bool, key string) {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			*dst = b
		}
	}
}

func setDur(dst *time.Duration, key string) {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			*dst = d
		}
	}
}

func setDurSeconds(dst *time.Duration, key string) {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			*dst = time.Duration(n) * time.Second
		}
	}
}
