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
	tests := map[string]string{
		"capacity cannot be lower than existing seat count": "座位数量不能小于该桌子已有的座位数",
		"seat is already reserved":                          "该座位已被预约，请选择其他座位",
		"seat is not available":                             "该座位当前不可预约，请选择其他座位",
		"reservation cannot be cancelled":                   "该预约已取消或不存在",
	}
	for message, want := range tests {
		if got := clientMessage(apperr.CodeConflict, message); got != want {
			t.Fatalf("clientMessage(%q) = %q, want %q", message, got, want)
		}
	}
}

func TestClientMessageFallsBackToChineseByCode(t *testing.T) {
	got := clientMessage(apperr.CodePermissionDenied, "subject type not permitted for this endpoint")
	if got != "没有权限执行此操作" {
		t.Fatalf("clientMessage() = %q", got)
	}
}
