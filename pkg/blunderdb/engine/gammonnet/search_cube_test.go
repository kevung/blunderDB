// SPDX-License-Identifier: MIT

package gammonnet

import (
	"math"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// The cube in the search (ADR-0023): three things carry the port, and each
// has here the test that would give it away.

// openingAt builds the opening position with the given dice, at a raw
// blunderDB score (mover's away first; -1/-1 is money; 0 is 1-away
// post-Crawford, 1 is Crawford — the sentinel MatchStateFromPosition reads).
func openingAt(t *testing.T, d1, d2, awayMover, awayOpponent int) domain.Position {
	t.Helper()
	pos := domain.InitializePosition()
	pos.Dice = [2]int{d1, d2}
	pos.PlayerOnRoll = domain.Black
	pos.Score = [2]int{awayMover, awayOpponent}
	pos.Cube = domain.Cube{Owner: domain.None, Value: 0}
	return pos
}

// The whole point, on the position that raised the question: with the
// opening 6-4 at 4-away/2-away the trailer plays 8/2 6/2 — the gammon-go
// play, gnubg's cubeful choice (+0.350, 24/18 13/9 at +0.307) — and only
// because the leaves are priced with the cube. Cubeless, the very same search
// prefers 24/18 13/9 (gnubg cubeless: −0.088 against −0.094 for 8/2 6/2),
// because without the double the trailer's gammons are worth less than the
// leader's. Both sides of the contrast are pinned, and so is the sign of the
// equity: the old cubeless number was −0.09, the cubeful one sits around
// +0.3 (upstream use_cube +0.294, gnubg +0.350), and a port that mixed the two
// scales would land somewhere no engine does.
func TestTheCubeMakesTheTrailerPlayForTheGammon(t *testing.T) {
	pos := openingAt(t, 6, 4, 4, 2)

	res, err := EvaluatePosition(pos, 2, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Moves) == 0 || !samePlay(res.Moves[0].Move, "8/2 6/2") {
		t.Fatalf("cubeful 2-ply at 4-away/2-away: best move %q, want 8/2 6/2 (top: %v)", firstMove(res), topMoves(res, 3))
	}
	if e := res.Moves[0].Equity; e < 0.15 || e > 0.45 {
		t.Errorf("cubeful 2-ply equity of 8/2 6/2 at 4-away/2-away = %.3f, want roughly +0.3 (gnubg +0.350, upstream +0.294)", e)
	}

	// The same search, cubeless.
	cfg, _, err := ConfigForPosition(&pos, 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	cfg.UseCube = false
	s, err := NewSearcher(cfg)
	if err != nil {
		t.Fatal(err)
	}
	gnPos, _ := FromDomain(&pos)
	best, ok, err := s.WithWorkers(8).BestPlay(&gnPos, 6, 4)
	if err != nil || !ok {
		t.Fatalf("cubeless search: ok=%v err=%v", ok, err)
	}
	notations := notationIndex(domain.LegalMoves(&pos), pos.PlayerOnRoll)
	if got := notationForCandidate(&best, notations); !samePlay(got, "24/18 13/9") {
		t.Errorf("cubeless 2-ply at 4-away/2-away: best move %q, want 24/18 13/9", got)
	}
}

// samePlay compares two notations as the sets of hops they are: the search
// and the reference name the same play, but nothing orders the hops the same
// way ("6/2 8/2" is "8/2 6/2").
func samePlay(a, b string) bool {
	x, y := strings.Fields(a), strings.Fields(b)
	sort.Strings(x)
	sort.Strings(y)
	return len(x) == len(y) && slices.Equal(x, y)
}

func firstMove(r EvalResult) string {
	if len(r.Moves) == 0 {
		return ""
	}
	return r.Moves[0].Move
}

func topMoves(r EvalResult, n int) []string {
	var out []string
	for i := 0; i < len(r.Moves) && i < n; i++ {
		out = append(out, r.Moves[i].Move)
	}
	return out
}

// In the Crawford game there is no cube, so UseCube must change nothing:
// same move, same equity to the bit, at 1 ply where the leaves are actually
// consulted. This is the port-side twin of gammonNet's
// test_crawford_search_with_use_cube_is_the_cubeless_search.
func TestCrawfordSearchWithCubeIsTheCubelessSearch(t *testing.T) {
	for _, away := range [][2]int{{4, 1}, {1, 4}} {
		state := MatchState{AwayOnRoll: away[0], AwayOpponent: away[1], Cube: 1, Crawford: true}
		cubeless := DefaultConfig(1)
		cubeless.UseMatch, cubeless.Match = true, state
		cubeful := cubeless
		cubeful.UseCube, cubeful.CubeOwner, cubeful.CubeX = true, CubeCentred, DefaultEfficiency(CubeCentred)

		a, _ := NewSearcher(cubeless)
		b, _ := NewSearcher(cubeful)
		for _, roll := range [][2]int{{6, 4}, {3, 1}, {5, 2}} {
			pos := openingAt(t, roll[0], roll[1], away[0], away[1])
			gnPos, _ := FromDomain(&pos)
			ca, oka, _ := a.BestPlay(&gnPos, roll[0], roll[1])
			cb, okb, _ := b.BestPlay(&gnPos, roll[0], roll[1])
			if oka != okb || ca.Play.Result != cb.Play.Result || ca.Equity != cb.Equity {
				t.Errorf("Crawford %d/%d roll %v: cubeless (%v, %.9f) vs use_cube (%v, %.9f) differ",
					away[0], away[1], roll, ca.Play.Result, ca.Equity, cb.Play.Result, cb.Equity)
			}
		}
	}
}

// The linearity identity at a MATCH score and at depth: the match valuation
// is linear in the distribution, so matchEquity(Probs(pos)) must equal the
// equity the scalar recursion assigns to pos — with the state swapped at
// every ply on both walks. Before ADR-0023 probsAt reused the ROOT's state at
// every level, which this identity misses at 1 ply (the inner call is a leaf)
// and at any symmetric score, and catches at 2 ply at 4-away/2-away.
func TestProbsMatchEquityMatchesPositionEquity(t *testing.T) {
	state := MatchState{AwayOnRoll: 4, AwayOpponent: 2, Cube: 1}
	cfg := DefaultConfig(2)
	cfg.UseMatch, cfg.Match = true, state
	s, err := NewSearcher(cfg)
	if err != nil {
		t.Fatal(err)
	}
	s = s.WithWorkers(8)

	pos := openingAt(t, 0, 0, 4, 2)
	gnPos, _ := FromDomain(&pos)
	probs, ok := s.Probs(&gnPos)
	if !ok {
		t.Fatal("Probs refused")
	}
	fromProbs, ok := matchEquity(state, &probs)
	if !ok {
		t.Fatal("matchEquity refused")
	}
	want, ok := s.positionEquity(&gnPos, 2, 0, &state, CubeCentred)
	if !ok {
		t.Fatal("positionEquity refused")
	}
	if d := math.Abs(fromProbs - want); d > 1e-5 {
		t.Errorf("2-ply at 4-away/2-away: matchEquity(Probs) = %.6f, positionEquity = %.6f (Δ %.2e): the probability walk is not swapping the score", fromProbs, want, d)
	}
}
