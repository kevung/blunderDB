package database

import (
	"math"
	"path/filepath"
	"testing"
)

func TestScoreCheck(t *testing.T) {
	matchFile := filepath.Join(
		"testdata",
		"2024-08-10-Aachen-1x11pt-1x7pt-2x7ptDoubleConsultation",
		"double",
		"Squire-Jørgensen-Harmand-Unger 7 point match 18-08-2024.xg",
	)
	db := newTestDB(t)
	matchID, err := db.ImportXGMatch(matchFile)
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	rows, err := db.db.Query(
		`SELECT g.game_number, mv.player,
		       COALESCE(p.score_1, 0), COALESCE(p.score_2, 0),
		       COALESCE(m.match_length, 0), COALESCE(p.cube_value, 0),
		       (`+statsErrExpr+`) as err_mp
		FROM position p
		JOIN analysis a ON a.position_id = p.id
		JOIN move mv ON mv.position_id = p.id
		JOIN game g ON g.id = mv.game_id
		JOIN match m ON m.id = g.match_id
		WHERE g.match_id = ?
		  AND a.position_id IS NOT NULL
		  AND (`+statsErrExpr+`) IS NOT NULL
		ORDER BY g.game_number, mv.move_number
		LIMIT 30`, matchID)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	t.Log("game|player|away1|away2|ml|cv|errMP|mwcLoss%")
	for rows.Next() {
		var gameNum, player, s1, s2, ml, cv int
		var errMP int64
		if err2 := rows.Scan(&gameNum, &player, &s1, &s2, &ml, &cv, &errMP); err2 != nil {
			t.Logf("  scan error: %v", err2)
			continue
		}
		fMove := 0
		if player == -1 {
			fMove = 1
		}
		cur0, cur1 := ml-s1, ml-s2
		mwc := ConvertEMGLossToMWCLoss(int(errMP), cur0, cur1, fMove, 1<<cv, ml)
		if math.IsNaN(mwc) {
			mwc = 0
		}
		t.Logf("  g=%d p=%+d s1=%d s2=%d ml=%d cv=%d e=%d mwc=%.3f%%",
			gameNum, player, s1, s2, ml, cv, errMP, mwc*100)
	}
}
