package ingest

import "testing"

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
