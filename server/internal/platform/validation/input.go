// Package validation contains the shared boundary validation used by HTTP-facing
// services. Values are rejected instead of silently rewritten so an attacker
// cannot submit one value and have a different value persisted.
package validation

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	dangerousTextPattern = regexp.MustCompile(`(?i)(?:[<>]|&(?:lt|gt|#0*60|#0*62|#x0*3c|#x0*3e);|javascript\s*:|data\s*:\s*text/html|(?:^|[\s"'])on(?:abort|blur|change|click|error|focus|input|load|mouseover|submit)\s*=)`)
	phonePattern         = regexp.MustCompile(`^1[3-9][0-9]{9}$`)
	inviteCodePattern    = regexp.MustCompile(`^[A-Za-z0-9]{4,32}$`)
	verificationPattern  = regexp.MustCompile(`^[0-9]{6}$`)
)

// TextOptions describes a plain-text field. Plain text deliberately rejects
// markup and event-handler syntax; HTML belongs only in explicitly sanitized
// rich-text fields.
type TextOptions struct {
	Label         string
	MinRunes      int
	MaxRunes      int
	AllowEmpty    bool
	AllowNewlines bool
}

// PlainText trims and validates a user-controlled plain-text value.
func PlainText(value string, opts TextOptions) (string, error) {
	value = strings.TrimSpace(value)
	label := opts.Label
	if label == "" {
		label = "内容"
	}
	if value == "" {
		if opts.AllowEmpty {
			return "", nil
		}
		return "", fmt.Errorf("请填写%s", label)
	}
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("%s包含无效字符", label)
	}
	for _, r := range value {
		if unicode.IsControl(r) && !(opts.AllowNewlines && (r == '\n' || r == '\r' || r == '\t')) {
			return "", fmt.Errorf("%s包含不允许的控制字符", label)
		}
	}
	if dangerousTextPattern.MatchString(value) {
		return "", fmt.Errorf("%s包含不安全内容", label)
	}
	count := utf8.RuneCountInString(value)
	if opts.MinRunes > 0 && count < opts.MinRunes {
		return "", fmt.Errorf("%s至少需要%d个字", label, opts.MinRunes)
	}
	if opts.MaxRunes > 0 && count > opts.MaxRunes {
		return "", fmt.Errorf("%s不能超过%d个字", label, opts.MaxRunes)
	}
	return value, nil
}

// Phone validates the 11-digit mainland mobile number requested by the
// franchise consultation form.
func Phone(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !phonePattern.MatchString(value) {
		return "", fmt.Errorf("请输入正确的11位手机号")
	}
	return value, nil
}

// InviteCode validates both current numeric codes and legacy alphanumeric ones.
func InviteCode(value string, allowEmpty bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" && allowEmpty {
		return "", nil
	}
	if !inviteCodePattern.MatchString(value) {
		return "", fmt.Errorf("邀请码格式不正确")
	}
	return value, nil
}

// VerificationCode validates the six-digit ticket code used by both manual
// entry and QR scanning.
func VerificationCode(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !verificationPattern.MatchString(value) {
		return "", fmt.Errorf("核销码格式不正确")
	}
	return value, nil
}

// OpaqueToken bounds tokens supplied by external SDKs without assuming their
// internal alphabet. Control characters are never valid in these credentials.
func OpaqueToken(label, value string, maxBytes int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s不能为空", label)
	}
	if maxBytes > 0 && len(value) > maxBytes {
		return "", fmt.Errorf("%s格式不正确", label)
	}
	if dangerousTextPattern.MatchString(value) {
		return "", fmt.Errorf("%s格式不正确", label)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("%s格式不正确", label)
		}
	}
	return value, nil
}

// HTTPURL accepts only absolute HTTP(S) URLs. It is suitable for persisted
// image links and prevents javascript:/data: schemes reaching image renderers.
func HTTPURL(label, value string, allowEmpty bool) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" && allowEmpty {
		return "", nil
	}
	if len(value) > 2048 {
		return "", fmt.Errorf("%s地址过长", label)
	}
	u, err := url.Parse(value)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", fmt.Errorf("%s地址格式不正确", label)
	}
	return u.String(), nil
}
