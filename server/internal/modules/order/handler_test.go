package order

import (
	"testing"

	"github.com/inwardclub/server/internal/platform/authn"
	apperr "github.com/inwardclub/server/internal/platform/errors"
)

func TestPreMemberActivityOrderOnlyAllowsWeChat(t *testing.T) {
	if err := validateActivityPayMethodForSubject(authn.SubjectPreMember, PayMethodWeChat); err != nil {
		t.Fatalf("pre-member WeChat payment should be allowed: %v", err)
	}
	if code := apperr.From(validateActivityPayMethodForSubject(authn.SubjectPreMember, PayMethodCoin)).Code; code != apperr.CodePermissionDenied {
		t.Fatalf("pre-member coin payment: got %s, want %s", code, apperr.CodePermissionDenied)
	}
	if err := validateActivityPayMethodForSubject(authn.SubjectMember, PayMethodCoin); err != nil {
		t.Fatalf("completed member coin payment should remain allowed: %v", err)
	}
}
