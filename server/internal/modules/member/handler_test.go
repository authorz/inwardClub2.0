package member

import "testing"

func TestOptionalStoreID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{name: "omitted", value: "", want: 0},
		{name: "whitespace", value: "  ", want: 0},
		{name: "store", value: "7", want: 7},
		{name: "zero", value: "0", wantErr: true},
		{name: "negative", value: "-1", wantErr: true},
		{name: "invalid", value: "abc", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := optionalStoreID(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("optionalStoreID(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("optionalStoreID(%q) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}
