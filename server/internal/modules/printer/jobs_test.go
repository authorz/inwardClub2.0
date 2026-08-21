package printer

import (
	"strings"
	"testing"
)

func TestTruncatePrintJobErrorPreservesRuneBoundary(t *testing.T) {
	message := strings.Repeat("错", maxPrintJobErrorLength+10)
	got := truncatePrintJobError(message)
	if len([]rune(got)) != maxPrintJobErrorLength {
		t.Fatalf("rune length = %d, want %d", len([]rune(got)), maxPrintJobErrorLength)
	}
}
