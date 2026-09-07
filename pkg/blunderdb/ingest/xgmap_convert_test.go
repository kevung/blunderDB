package ingest

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/xgparser/xgparser"
)

// Moved from the legacy tests/bearoff_unit_test.go and tests/xg_import_test.go
// (root "tests" package, invisible to coverage since it built no code): both
// duplicated convertXGMoveToString (and, for bearoff_unit_test.go, formatPoint)
// as local copies instead of exercising the real ingest code, so a change here
// could drift from what it claimed to test — and xg_import_test.go's cases
// were never actually asserted (t.Logf only). This exercises the real
// convertXGMoveToString directly, with real assertions.
func TestConvertXGMoveToString(t *testing.T) {
	testCases := []struct {
		name     string
		input    [8]int32
		expected string
	}{
		{
			name:     "Simple bear-off: 6/off",
			input:    [8]int32{6, -2, -1, -1, -1, -1, -1, -1},
			expected: "6/off",
		},
		{
			name:     "Double bear-off: 4/off 3/off",
			input:    [8]int32{4, -2, 3, -2, -1, -1, -1, -1},
			expected: "4/off 3/off",
		},
		{
			name:     "Bear-off and move: 5/off 4/1",
			input:    [8]int32{5, -2, 4, 1, -1, -1, -1, -1},
			expected: "5/off 4/1",
		},
		{
			name:     "Simple move: 24/23 13/8",
			input:    [8]int32{13, 8, 24, 23, -1, -1, -1, -1},
			expected: "24/23 13/8",
		},
		{
			name:     "Bar entry: bar/23",
			input:    [8]int32{25, 23, -1, -1, -1, -1, -1, -1},
			expected: "bar/23",
		},
		// Moved from the legacy tests/xg_import_test.go: exercises mergeSlides,
		// merged from a real match transcription (anonymised — the original
		// carried real player names).
		{
			name:     "Doublet slide 11: 24/20",
			input:    [8]int32{24, 23, 23, 22, 22, 21, 21, 20},
			expected: "24/20",
		},
		{
			name:     "Grouped doublet 22: 18/16(2) 6/4(2)",
			input:    [8]int32{18, 16, 18, 16, 6, 4, 6, 4},
			expected: "18/16(2) 6/4(2)",
		},
		{
			name:     "Cannot move",
			input:    [8]int32{-1, -1, -1, -1, -1, -1, -1, -1},
			expected: "Cannot Move",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := convertXGMoveToString(tc.input)
			if result != tc.expected {
				t.Errorf("convertXGMoveToString(%v) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestXGCrawfordSentinelIsDerivedFromTheScores pins the XG half of issue #338.
//
// An XG match states no rule per game — xgparser.Game carries a game number, an
// initial score, a winner and its moves — so which game is the Crawford one is
// DERIVED from the sequence of initial scores, and the away score then says so:
// `1` in the Crawford game, `0` in every game after it (CONTEXT.md, « Away
// score »). Before the fix both said `1`, and the engine read a dead cube in
// the post-Crawford games, where the trailer doubles at the first opportunity.
func TestXGCrawfordSentinelIsDerivedFromTheScores(t *testing.T) {
	// A 7-point match: 2-2, then 6-2 (the Crawford game), then 6-3 and 6-5,
	// both post-Crawford with the same player still one point away.
	games := []xgparser.Game{
		{GameNumber: 1, InitialScore: [2]int32{2, 2}},
		{GameNumber: 2, InitialScore: [2]int32{6, 2}},
		{GameNumber: 3, InitialScore: [2]int32{6, 3}},
		{GameNumber: 4, InitialScore: [2]int32{6, 5}},
	}
	wantCrawford := []bool{false, true, false, false}
	wantAway := [][2]int{{5, 5}, {domain.Crawford, 5}, {domain.PostCrawford, 4}, {domain.PostCrawford, 2}}

	// A legal enough board for the mapper: one checker per side, everything
	// else borne off. Only the score is under test here.
	var xgPos xgparser.Position
	xgPos.Checkers[1] = 1
	xgPos.Checkers[24] = -1

	for i := range games {
		crawford := isCrawfordGame(7, i, games)
		if crawford != wantCrawford[i] {
			t.Errorf("isCrawfordGame(7, %d) = %v, want %v", i, crawford, wantCrawford[i])
		}
		pos, err := createPositionFromXG(xgPos, &games[i], 7, 1, crawford)
		if err != nil {
			t.Fatalf("game %d: createPositionFromXG: %v", i, err)
		}
		if pos.Score != wantAway[i] {
			t.Errorf("game %d (initial %v): away score %v, want %v",
				i, games[i].InitialScore, pos.Score, wantAway[i])
		}
	}

	// Money play has no Crawford game at all, whatever the scores say.
	if isCrawfordGame(0, 0, games) {
		t.Error("isCrawfordGame reports a Crawford game in money play")
	}
	// A 1-point match is its own Crawford game: its only game starts at match
	// point, and the [1 1] that follows is the cube-dead reading it wants.
	one := []xgparser.Game{{GameNumber: 1, InitialScore: [2]int32{0, 0}}}
	if !isCrawfordGame(1, 0, one) {
		t.Error("the only game of a 1-point match is not reported as Crawford")
	}
}
