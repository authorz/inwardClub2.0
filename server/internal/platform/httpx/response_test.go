package httpx

import (
	"testing"

	apperr "github.com/inwardclub/server/internal/platform/errors"
)

func TestClientMessageKeepsChineseBusinessMessage(t *testing.T) {
	const message = "你今天已经预约座位了"
	if got := clientMessage(apperr.CodeConflict, message); got != message {
		t.Fatalf("clientMessage() = %q, want %q", got, message)
	}
}

func TestClientMessageTranslatesKnownBusinessMessage(t *testing.T) {
	got := clientMessage(apperr.CodeConflict, "capacity cannot be lower than existing seat count")
	if got != "座位数量不能小于该桌子已有的座位数" {
		t.Fatalf("clientMessage() = %q", got)
	}
}

func TestClientMessageFallsBackToChineseByCode(t *testing.T) {
	got := clientMessage(apperr.CodePermissionDenied, "subject type not permitted for this endpoint")
	if got != "没有权限执行此操作" {
		t.Fatalf("clientMessage() = %q", got)
	}
}
