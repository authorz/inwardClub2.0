package wallet

import "testing"

func TestCoinsRequired(t *testing.T) {
	tests := []struct {
		name       string
		amountCent int64
		want       int64
		wantErr    bool
	}{
		{name: "whole yuan", amountCent: 3900, want: 39},
		{name: "one yuan", amountCent: 100, want: 1},
		{name: "fractional yuan", amountCent: 3950, wantErr: true},
		{name: "zero", amountCent: 0, wantErr: true},
		{name: "negative", amountCent: -100, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CoinsRequired(tt.amountCent)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CoinsRequired(%d) error = %v, wantErr %v", tt.amountCent, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("CoinsRequired(%d) = %d, want %d", tt.amountCent, got, tt.want)
			}
		})
	}
}
