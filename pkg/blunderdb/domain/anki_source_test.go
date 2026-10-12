package domain

import (
	"slices"
	"testing"
)

// A search deck's ids are read from its JSON document, whose command may
// hold commas; a legacy comma-separated list is still read whole.
func TestSearchSourceIDs(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   []int64
	}{
		{`{"command":"s id3,4","position":"{}","ids":[3,4]}`, []int64{3, 4}},
		{`{"ids":[7],"library":true}`, []int64{7}},
		{`{"command":"s e>0.1"}`, nil},
		{"5, 6,x,8", []int64{5, 6, 8}},
		{"", nil},
	} {
		if got := SearchSourceIDs(tc.source); !slices.Equal(got, tc.want) {
			t.Errorf("SearchSourceIDs(%q) = %v, want %v", tc.source, got, tc.want)
		}
	}
}
