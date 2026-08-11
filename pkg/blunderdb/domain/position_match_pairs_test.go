package domain

import "testing"

// This file tests the five P1/P2 predicate pairs in position_match.go
// side-by-side, each P2 case built as the exact board mirror of its P1 case
// (Mirror() maps point i to 25-i and flips color). Any P1/P2 asymmetry —
// e.g. an off-by-one in the point range, or a color check copy-pasted
// without flipping — would show up as the two halves of a pair disagreeing
// on mirror-equivalent boards, which is exactly the "copié-collé-inversé"
// bug class flagged by the audit.
//
// Result: no asymmetry found in any of the five pairs — every P2 range is
// the exact mirror (point range and color flipped) of its P1 counterpart.
// See fiche-11 execution notes for the one real asymmetry found nearby
// (Player1AbsolutePipCount has no Player2 counterpart at all — a deliberate
// API gap, not a mirrored predicate, so out of scope for this file).

// --- CheckerOff (Bearoff count) ---------------------------------------------

func TestMatchesPlayer1CheckerOff(t *testing.T) {
	cases := []struct {
		name   string
		off    int
		filter string
		want   bool
	}{
		{"o> above threshold", 5, "o>3", true},
		{"o> at threshold", 3, "o>3", true},
		{"o> below threshold", 2, "o>3", false},
		{"o< below threshold", 2, "o<3", true},
		{"o< above threshold", 4, "o<3", false},
		{"single value oX means oX,X", 4, "o4", true},
		{"single value oX excludes neighbours", 3, "o4", false},
		{"range o2,5 inside", 3, "o2,5", true},
		{"range o2,5 outside", 6, "o2,5", false},
		{"range reversed bounds o5,2 still works", 3, "o5,2", true},
		{"malformed value", 3, "o>x", false},
		{"unrelated prefix", 3, "z>1", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Position{Board: Board{Bearoff: [2]int{c.off, 0}}}
			if got := p.MatchesPlayer1CheckerOff(c.filter); got != c.want {
				t.Errorf("MatchesPlayer1CheckerOff(off=%d, filter=%q) = %v, want %v", c.off, c.filter, got, c.want)
			}
		})
	}
}

func TestMatchesPlayer2CheckerOff(t *testing.T) {
	// Mirror of TestMatchesPlayer1CheckerOff: same cases, uppercase filter
	// tokens, value on Bearoff[1] instead of Bearoff[0].
	cases := []struct {
		name   string
		off    int
		filter string
		want   bool
	}{
		{"O> above threshold", 5, "O>3", true},
		{"O> at threshold", 3, "O>3", true},
		{"O> below threshold", 2, "O>3", false},
		{"O< below threshold", 2, "O<3", true},
		{"O< above threshold", 4, "O<3", false},
		{"single value OX means OX,X", 4, "O4", true},
		{"single value OX excludes neighbours", 3, "O4", false},
		{"range O2,5 inside", 3, "O2,5", true},
		{"range O2,5 outside", 6, "O2,5", false},
		{"range reversed bounds O5,2 still works", 3, "O5,2", true},
		{"malformed value", 3, "O>x", false},
		{"unrelated prefix", 3, "Z>1", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Position{Board: Board{Bearoff: [2]int{0, c.off}}}
			if got := p.MatchesPlayer2CheckerOff(c.filter); got != c.want {
				t.Errorf("MatchesPlayer2CheckerOff(off=%d, filter=%q) = %v, want %v", c.off, c.filter, got, c.want)
			}
		})
	}
}

// --- BackChecker (P1: points 19-24 Black; P2: points 1-6 White) ------------

func TestMatchesBackCheckerMirrorPairs(t *testing.T) {
	cases := []struct {
		name     string
		p1Points map[int]int // Black checkers, for MatchesPlayer1BackChecker
		p2Points map[int]int // exact mirror: White checkers on 25-i
		filter1  string
		filter2  string
		want     bool
	}{
		{
			"one back checker at the edge of the zone (pt 19 / mirror pt 6)",
			map[int]int{19: 1}, map[int]int{6: -1},
			"k1", "K1", true,
		},
		{
			"one back checker at the deep edge (pt 24 / mirror pt 1)",
			map[int]int{24: 1}, map[int]int{1: -1},
			"k1", "K1", true,
		},
		{
			"checker just outside the zone (pt 18 / mirror pt 7) does not count",
			map[int]int{18: 3}, map[int]int{7: -3},
			"k1", "K1", false,
		},
		{
			"multiple back checkers, range filter",
			map[int]int{20: 2, 23: 1}, map[int]int{5: -2, 2: -1},
			"k2,3", "K2,3", true,
		},
		{
			"back checkers outside range filter",
			map[int]int{20: 4}, map[int]int{5: -4},
			"k1,2", "K1,2", false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p1 := posPts(c.p1Points)
			if got := p1.MatchesPlayer1BackChecker(c.filter1); got != c.want {
				t.Errorf("P1 %s: MatchesPlayer1BackChecker(%q) = %v, want %v", c.name, c.filter1, got, c.want)
			}
			p2 := posPts(c.p2Points)
			if got := p2.MatchesPlayer2BackChecker(c.filter2); got != c.want {
				t.Errorf("P2 (mirror) %s: MatchesPlayer2BackChecker(%q) = %v, want %v", c.name, c.filter2, got, c.want)
			}
		})
	}
}

// --- CheckerInZone (P1: points 0-12 Black; P2: points 13-25 White) ---------

func TestMatchesCheckerInZoneMirrorPairs(t *testing.T) {
	cases := []struct {
		name     string
		p1Points map[int]int
		p2Points map[int]int
		filter1  string
		filter2  string
		want     bool
	}{
		{
			"checker on the low edge of the zone (pt 0 / mirror pt 25)",
			map[int]int{0: 1}, map[int]int{25: -1},
			"z1", "Z1", true,
		},
		{
			"checker on the high edge of the zone (pt 12 / mirror pt 13)",
			map[int]int{12: 1}, map[int]int{13: -1},
			"z1", "Z1", true,
		},
		{
			"checker just outside the zone (pt 13 / mirror pt 12) does not count",
			map[int]int{13: 5}, map[int]int{12: -5},
			"z1", "Z1", false,
		},
		{
			"range filter matches",
			map[int]int{3: 2, 8: 1}, map[int]int{22: -2, 17: -1},
			"z2,3", "Z2,3", true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p1 := posPts(c.p1Points)
			if got := p1.MatchesPlayer1CheckerInZone(c.filter1); got != c.want {
				t.Errorf("P1 %s: MatchesPlayer1CheckerInZone(%q) = %v, want %v", c.name, c.filter1, got, c.want)
			}
			p2 := posPts(c.p2Points)
			if got := p2.MatchesPlayer2CheckerInZone(c.filter2); got != c.want {
				t.Errorf("P2 (mirror) %s: MatchesPlayer2CheckerInZone(%q) = %v, want %v", c.name, c.filter2, got, c.want)
			}
		})
	}
}

// --- OutfieldBlot (P1: points 7-18 Black, count==1; P2: same range White) --

func TestMatchesOutfieldBlotMirrorPairs(t *testing.T) {
	cases := []struct {
		name     string
		p1Points map[int]int
		p2Points map[int]int
		filter1  string
		filter2  string
		want     bool
	}{
		{
			"single blot at the low edge (pt 7), self-mirrors to pt 18",
			map[int]int{7: 1}, map[int]int{18: -1},
			"bo1,1", "BO1,1", true,
		},
		{
			"single blot at the high edge (pt 18), self-mirrors to pt 7",
			map[int]int{18: 1}, map[int]int{7: -1},
			"bo1,1", "BO1,1", true,
		},
		{
			"a made point (2 checkers) is not a blot",
			map[int]int{10: 2}, map[int]int{15: -2},
			"bo1,1", "BO1,1", false,
		},
		{
			"blot just outside the outfield (pt 6 / mirror pt 19) does not count",
			map[int]int{6: 1}, map[int]int{19: -1},
			"bo1,1", "BO1,1", false,
		},
		{
			"two blots, range filter",
			map[int]int{9: 1, 14: 1}, map[int]int{16: -1, 11: -1},
			"bo2,2", "BO2,2", true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p1 := posPts(c.p1Points)
			if got := p1.MatchesPlayer1OutfieldBlot(c.filter1); got != c.want {
				t.Errorf("P1 %s: MatchesPlayer1OutfieldBlot(%q) = %v, want %v", c.name, c.filter1, got, c.want)
			}
			p2 := posPts(c.p2Points)
			if got := p2.MatchesPlayer2OutfieldBlot(c.filter2); got != c.want {
				t.Errorf("P2 (mirror) %s: MatchesPlayer2OutfieldBlot(%q) = %v, want %v", c.name, c.filter2, got, c.want)
			}
		})
	}
}

// --- JanBlot (P1: points 1-6 Black, count==1; P2: points 19-24 White) ------

func TestMatchesJanBlotMirrorPairs(t *testing.T) {
	cases := []struct {
		name     string
		p1Points map[int]int
		p2Points map[int]int
		filter1  string
		filter2  string
		want     bool
	}{
		{
			"blot at the low edge (pt 1 / mirror pt 24)",
			map[int]int{1: 1}, map[int]int{24: -1},
			"bj1,1", "BJ1,1", true,
		},
		{
			"blot at the high edge (pt 6 / mirror pt 19)",
			map[int]int{6: 1}, map[int]int{19: -1},
			"bj1,1", "BJ1,1", true,
		},
		{
			"a made point is not a blot",
			map[int]int{3: 2}, map[int]int{22: -2},
			"bj1,1", "BJ1,1", false,
		},
		{
			"blot just outside the jan (pt 7 / mirror pt 18) does not count",
			map[int]int{7: 1}, map[int]int{18: -1},
			"bj1,1", "BJ1,1", false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p1 := posPts(c.p1Points)
			if got := p1.MatchesPlayer1JanBlot(c.filter1); got != c.want {
				t.Errorf("P1 %s: MatchesPlayer1JanBlot(%q) = %v, want %v", c.name, c.filter1, got, c.want)
			}
			p2 := posPts(c.p2Points)
			if got := p2.MatchesPlayer2JanBlot(c.filter2); got != c.want {
				t.Errorf("P2 (mirror) %s: MatchesPlayer2JanBlot(%q) = %v, want %v", c.name, c.filter2, got, c.want)
			}
		})
	}
}
