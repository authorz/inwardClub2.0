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

func TestParseLegacyReceiptTaskID(t *testing.T) {
	paymentID, deviceID, ok := parseLegacyReceiptTaskID("payment:45:printer:4:print-receipt")
	if !ok || paymentID != 45 || deviceID != 4 {
		t.Fatalf("parsed payment=%d device=%d ok=%t", paymentID, deviceID, ok)
	}
	for _, invalid := range []string{"", "payment:0:printer:4:print-receipt", "coupon:45:printer:4"} {
		if _, _, ok := parseLegacyReceiptTaskID(invalid); ok {
			t.Fatalf("expected %q to be rejected", invalid)
		}
	}
}
