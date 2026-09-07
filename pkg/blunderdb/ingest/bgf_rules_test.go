package ingest

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestBGFRulesReachThePosition (issue #171): until ADR-0028 the BGF mapper
// dropped the session's optional rules on the floor — the repository's only
// TODO — and every imported position claimed Jacoby and beaver were off. They
// now travel from the top of the file (useJacoby/useBeaver) to the position's
// columns, and only in money play, where they mean something.
func TestBGFRulesReachThePosition(t *testing.T) {
	gameData := map[string]interface{}{"scoreGreen": 0, "scoreRed": 0}
	var board [28]int
	board[0] = 2
	board[23] = -2

	for _, tc := range []struct {
		name           string
		matchLen       int
		rules          bgfRules
		jacoby, beaver int
	}{
		{"money, both rules", 0, bgfRules{jacoby: true, beaver: true}, 1, 1},
		{"money, jacoby only", 0, bgfRules{jacoby: true}, 1, 0},
		{"money, beaver only", 0, bgfRules{beaver: true}, 0, 1},
		{"money, neither", 0, bgfRules{}, 0, 0},
		// At a match score the two rules do not apply, whatever the file says.
		{"match play", 7, bgfRules{jacoby: true, beaver: true}, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pos, err := createPositionFromBGF(board, gameData, tc.matchLen, 1, -1, tc.rules)
			if err != nil {
				t.Fatalf("createPositionFromBGF: %v", err)
			}
			if pos.HasJacoby != tc.jacoby || pos.HasBeaver != tc.beaver {
				t.Errorf("HasJacoby=%d HasBeaver=%d, want %d and %d",
					pos.HasJacoby, pos.HasBeaver, tc.jacoby, tc.beaver)
			}
		})
	}
}

// TestBGFCrawfordSentinelReachesTheAwayScore is issue #338 on the BGF side.
//
// BGBlitz states "isCrawford" per game and the mapper already read it into
// bgfRules; what it never reached was the SCORE. The away score carries the
// Crawford rule inside the number (CONTEXT.md, « Away score »), so the same
// 6-2 gives `1` in the Crawford game and `0` after it — where the cube is live
// again and the trailer doubles at the first opportunity.
func TestBGFCrawfordSentinelReachesTheAwayScore(t *testing.T) {
	var board [28]int
	board[0] = 2
	board[23] = -2

	for _, tc := range []struct {
		name       string
		matchLen   int
		green, red int
		crawford   bool
		want       [2]int
	}{
		{"the Crawford game", 7, 6, 2, true, [2]int{domain.Crawford, 5}},
		{"the game after it", 7, 6, 2, false, [2]int{domain.PostCrawford, 5}},
		{"double match point, post-Crawford", 7, 6, 6, false, [2]int{domain.PostCrawford, domain.PostCrawford}},
		{"money play knows no Crawford", 0, 3, 1, false, [2]int{domain.Unlimited, domain.Unlimited}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gameData := map[string]interface{}{"scoreGreen": tc.green, "scoreRed": tc.red}
			pos, err := createPositionFromBGF(board, gameData, tc.matchLen, 1, -1, bgfRules{crawford: tc.crawford})
			if err != nil {
				t.Fatalf("createPositionFromBGF: %v", err)
			}
			if pos.Score != tc.want {
				t.Errorf("away score %v, want %v", pos.Score, tc.want)
			}
		})
	}
}
