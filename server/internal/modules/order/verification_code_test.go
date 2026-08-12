package order

import (
	"regexp"
	"testing"
)

func TestNewCodeReturnsSixDigits(t *testing.T) {
	pattern := regexp.MustCompile(`^[0-9]{6}$`)
	for i := 0; i < 100; i++ {
		code, err := newCode()
		if err != nil {
			t.Fatalf("newCode: %v", err)
		}
		if !pattern.MatchString(code) {
			t.Fatalf("newCode() = %q, want exactly six digits", code)
		}
	}
}
