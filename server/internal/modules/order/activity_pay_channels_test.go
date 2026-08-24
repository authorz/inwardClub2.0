package order

import "testing"

func TestActivityPayMethodAllowedRequiresConfiguredCoupon(t *testing.T) {
	tests := []struct {
		name        string
		ticket      string
		activity    string
		payMethod   string
		wantAllowed bool
	}{
		{name: "activity enables coupon for ticket", ticket: `["wechat"]`, activity: `["wechat","coupon"]`, payMethod: PayMethodCoupon, wantAllowed: true},
		{name: "ticket cannot enable coupon without activity", ticket: `["wechat","coupon"]`, activity: `["wechat"]`, payMethod: PayMethodCoupon},
		{name: "empty ticket inherits activity coupon", ticket: `[]`, activity: `["wechat","coupon"]`, payMethod: PayMethodCoupon, wantAllowed: true},
		{name: "activity without coupon rejects coupon", ticket: `[]`, activity: `["wechat"]`, payMethod: PayMethodCoupon},
		{name: "legacy balance allows coin", ticket: `["balance"]`, activity: `[]`, payMethod: PayMethodCoin, wantAllowed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := activityPayMethodAllowed([]byte(tt.ticket), []byte(tt.activity), tt.payMethod)
			if err != nil {
				t.Fatal(err)
			}
			if allowed != tt.wantAllowed {
				t.Fatalf("allowed = %v, want %v", allowed, tt.wantAllowed)
			}
		})
	}
}

func TestActivityPayMethodAllowedRejectsInvalidStoredJSON(t *testing.T) {
	if _, err := activityPayMethodAllowed([]byte(`[]`), []byte(`{"bad":true}`), PayMethodCoupon); err == nil {
		t.Fatal("invalid activity pay channels should fail")
	}
	if _, err := activityPayMethodAllowed([]byte(`{"bad":true}`), []byte(`[]`), PayMethodWeChat); err == nil {
		t.Fatal("invalid ticket pay channels should fail")
	}
}
