package domain

import "testing"

func TestAwayStoredRange(t *testing.T) {
	// Each case lists the stored away scores of one player the token admits,
	// among -1 (money), 0 (post-Crawford), 1 (Crawford), 2, 3, 4, 5 and 6.
	stored := []int{Unlimited, PostCrawford, Crawford, 2, 3, 4, 5, 6}
	cases := []struct {
		tok  string
		want []int
	}{
		{"aw<1", []int{0, 1}},
		{"aw:1", []int{0, 1}},
		{"aw<2", []int{0, 1, 2}},
		{"aw<5", []int{0, 1, 2, 3, 4, 5}},
		{"aw>0", []int{0, 1, 2, 3, 4, 5, 6}},
		{"aw>1", []int{0, 1, 2, 3, 4, 5, 6}},
		{"aw>2", []int{2, 3, 4, 5, 6}},
		{"aw>3", []int{3, 4, 5, 6}},
		{"aw:2,5", []int{2, 3, 4, 5}},
		{"aw:5,2", []int{2, 3, 4, 5}},
		{"aw:1,2", []int{0, 1, 2}},
		{"aw:4", []int{4}},
		{"aw:0", nil},
		{"aw<0", nil},
		{"aw:x", nil},
		{"aw", nil},
		{"ml:4", nil},
	}
	for _, c := range cases {
		lo, hi, hasHi, none := AwayStoredRange(c.tok)
		var got []int
		if !none {
			for _, s := range stored {
				if s >= lo && (!hasHi || s <= hi) {
					got = append(got, s)
				}
			}
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: admits %v, want %v", c.tok, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: admits %v, want %v", c.tok, got, c.want)
				break
			}
		}
	}
}
