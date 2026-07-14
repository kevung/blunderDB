package postgres

import (
	"testing"
)

func TestParseFilterIDList(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    []int64
		wantErr bool
	}{
		{"empty", "", nil, false},
		{"single", "5", []int64{5}, false},
		{"range", "2,7", []int64{2, 3, 4, 5, 6, 7}, false},
		{"three-item comma list", "1,3,5", []int64{1, 3, 5}, false},
		{"semicolon list", "2;5;9", []int64{2, 5, 9}, false},
		{"mixed comma and semicolon", "1,3;4,5", []int64{1, 3, 4, 5}, false},
		{"invalid token", "1,abc,3", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseFilterIDList(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseFilterIDList(%q): expected error, got %v", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseFilterIDList(%q): unexpected error: %v", tc.input, err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("parseFilterIDList(%q) = %v, want %v", tc.input, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("parseFilterIDList(%q) = %v, want %v", tc.input, got, tc.want)
				}
			}
		})
	}
}
