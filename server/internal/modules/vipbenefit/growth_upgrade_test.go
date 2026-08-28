package vipbenefit

import "testing"

func TestTierUpgradeNeeded(t *testing.T) {
	level2 := 2
	level3 := 3
	tests := []struct {
		name         string
		currentLevel *int
		targetLevel  int
		want         bool
	}{
		{name: "unranked member", currentLevel: nil, targetLevel: 1, want: true},
		{name: "higher target", currentLevel: &level2, targetLevel: 3, want: true},
		{name: "same target", currentLevel: &level3, targetLevel: 3, want: false},
		{name: "lower target", currentLevel: &level3, targetLevel: 2, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tierUpgradeNeeded(tt.currentLevel, tt.targetLevel); got != tt.want {
				t.Fatalf("tierUpgradeNeeded() = %v, want %v", got, tt.want)
			}
		})
	}
}
