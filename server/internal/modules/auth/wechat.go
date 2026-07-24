// Package auth implements the three independent authentication flows: mini
// program WeChat login (member/staff), admin console and store console password
// login. Each mints tokens for its own audience only.
package auth

import (
	"context"
	"crypto/sha1"
)

// WeChatSession is the result of exchanging a login code with WeChat.
type WeChatSession struct {
	OpenID     string
	UnionID    string
	SessionKey string
}

// WeChatClient exchanges a mini-program login code for a session. Business code
// depends on this interface; the real client and the fake both satisfy it.
type WeChatClient interface {
	Code2Session(ctx context.Context, code string) (WeChatSession, error)
	// GetPhoneNumber resolves a phone number from an encrypted mini-program
	// phone code. Returned only for the auditable phone-binding flow.
	GetPhoneNumber(ctx context.Context, phoneCode string) (string, error)
}

// FakeWeChatClient is a deterministic offline client for dev/tests.
//
// The real WeChat issues a NEW js_code on every wx.login(), while a member's
// openid stays stable. Deriving the openid from the code (the old behaviour)
// therefore made every DevTools login look like a brand-new user, so the
// login flow could never be exercised there. To keep DevTools usable we pin
// the fake openid to a single stable value; set DEV_FAKE_OPENID to simulate a
// different member. Tests that need per-code isolation pass a distinct
// DEV_FAKE_OPENID (or the value derived below when it is empty and code is set).
type FakeWeChatClient struct{ openID string }

const defaultFakeOpenID = "fake_openid_dev"

// NewFakeWeChatClient builds the fake WeChat client. A non-empty openID pins the
// session identity (from DEV_FAKE_OPENID); empty falls back to a stable default.
func NewFakeWeChatClient(openID string) *FakeWeChatClient {
	if openID == "" {
		openID = defaultFakeOpenID
	}
	return &FakeWeChatClient{openID: openID}
}

// Code2Session returns a deterministic fake session with a STABLE openid,
// independent of the per-login code, so DevTools can log the same user in twice.
func (f *FakeWeChatClient) Code2Session(_ context.Context, _ string) (WeChatSession, error) {
	return WeChatSession{
		OpenID:     f.openID,
		SessionKey: "fake_session_key",
	}, nil
}

// GetPhoneNumber returns a deterministic fake phone number.
func (f *FakeWeChatClient) GetPhoneNumber(_ context.Context, phoneCode string) (string, error) {
	sum := sha1.Sum([]byte("phone:" + phoneCode))
	// Produce a stable 11-digit CN-style number for the fake.
	n := int(sum[0])%9 + 1
	return "1" + itoa11(n, sum[:]), nil
}

func itoa11(lead int, seed []byte) string {
	digits := []byte{byte('0' + lead)}
	for len(digits) < 10 {
		digits = append(digits, byte('0'+int(seed[len(digits)%len(seed)])%10))
	}
	return string(digits)
}
