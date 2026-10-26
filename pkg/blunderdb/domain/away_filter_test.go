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
		{"AW:4", nil},
	}
	for _, c := range cases {
		lo, hi, hasHi, none := AwayStoredRange(c.tok, "aw")
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

func TestAwayAdmitsOpponentPrefix(t *testing.T) {
	if !AwayAdmits("AW:1", "AW", PostCrawford) || !AwayAdmits("AW:1", "AW", Crawford) {
		t.Error("AW:1 must admit both one-point sentinels")
	}
	if AwayAdmits("AW:1", "AW", Unlimited) || AwayAdmits("aw:1", "AW", Crawford) {
		t.Error("money play, or a token of the other side, must admit nothing")
	}
}

func TestMatchesCrawfordFilter(t *testing.T) {
	cases := []struct {
		filter string
		score  [2]int
		want   bool
	}{
		{"", [2]int{Unlimited, Unlimited}, true},
		{CrawfordGame, [2]int{Crawford, 4}, true},
		{CrawfordGame, [2]int{5, Crawford}, true},
		{CrawfordGame, [2]int{Crawford, Crawford}, true},
		{CrawfordGame, [2]int{PostCrawford, 4}, false},
		{CrawfordGame, [2]int{Unlimited, Unlimited}, false},
		{CrawfordAfter, [2]int{PostCrawford, 4}, true},
		{CrawfordAfter, [2]int{3, PostCrawford}, true},
		{CrawfordAfter, [2]int{Crawford, 4}, false},
		{CrawfordAfter, [2]int{2, 2}, false},
		{CrawfordNone, [2]int{Crawford, PostCrawford}, false},
	}
	for _, c := range cases {
		if got := MatchesCrawfordFilter(c.filter, c.score); got != c.want {
			t.Errorf("%q on %v = %v, want %v", c.filter, c.score, got, c.want)
		}
	}
}
