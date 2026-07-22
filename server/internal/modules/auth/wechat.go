// Package auth implements the three independent authentication flows: mini
// program WeChat login (member/staff), admin console and store console password
// login. Each mints tokens for its own audience only.
package auth

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
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

// FakeWeChatClient is a deterministic offline client for dev/tests. The openid
// is derived from the code so the same code maps to the same member.
type FakeWeChatClient struct{}

// NewFakeWeChatClient builds the fake WeChat client.
func NewFakeWeChatClient() *FakeWeChatClient { return &FakeWeChatClient{} }

// Code2Session returns a deterministic fake session.
func (f *FakeWeChatClient) Code2Session(_ context.Context, code string) (WeChatSession, error) {
	sum := sha1.Sum([]byte("openid:" + code))
	return WeChatSession{
		OpenID:     "fake_openid_" + hex.EncodeToString(sum[:8]),
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
