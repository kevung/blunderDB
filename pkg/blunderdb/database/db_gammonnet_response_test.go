package database

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// A transcribed take or pass is stored with the cube already turned and the
// answerer on roll: evaluated as it stands, that is the answerer's redouble.
// The batch must score it as the reply to the doubler's cube decision — the
// same error, and the same MWC loss, as a direct analysis of that decision.
func TestAnalyzeMissingWithGammonNetScoresTakesAndPassesAsReplies(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)

	text, err := os.ReadFile(filepath.Join("testdata", "gnubg_selfplay_drops_7p.mat"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := transcript.FromMAT(string(text))
	if err != nil {
		t.Fatalf("FromMAT: %v", err)
	}
	saved, err := d.MaterializeTranscription(doc.Header, doc.Actions)
	if err != nil {
		t.Fatalf("MaterializeTranscription: %v", err)
	}
	if _, err := d.AnalyzeMissingWithGammonNet(context.Background(), 0, 0, 0, 0, nil, nil); err != nil {
		t.Fatalf("AnalyzeMissingWithGammonNet: %v", err)
	}

	type moveRow struct {
		gameID, positionID int64
		player             int
		action             string
		cubeErrMP          *int64
	}
	rows, err := d.db.Query(`
		SELECT mv.game_id, mv.position_id, mv.player, `+sqlshared.ActionLabelOrEmptySQL("mv.cube_action")+`, a.cube_error
		  FROM move mv JOIN game g ON g.id = mv.game_id
		  LEFT JOIN analysis a ON a.position_id = mv.position_id
		 WHERE g.match_id = ? ORDER BY mv.id`, saved.MatchID)
	if err != nil {
		t.Fatal(err)
	}
	var moves []moveRow
	for rows.Next() {
		var r moveRow
		if err := rows.Scan(&r.gameID, &r.positionID, &r.player, &r.action, &r.cubeErrMP); err != nil {
			t.Fatal(err)
		}
		moves = append(moves, r)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}

	matchLength := doc.Header.MatchLength
	var takes, passes int
	var wantMWC [2]float64
	for i, mv := range moves {
		if !engine.IsResponseCubeAction(mv.action) {
			continue
		}
		if i == 0 || moves[i-1].gameID != mv.gameID || engine.CanonicalCubeAction(moves[i-1].action) != engine.CubeDouble {
			t.Fatalf("move %d (%s) does not follow a double", i, mv.action)
		}
		loaded, err := d.LoadPositionsByIDs([]int64{moves[i-1].positionID, mv.positionID})
		if err != nil || len(loaded) != 2 {
			t.Fatalf("LoadPositionsByIDs: %v (%d positions)", err, len(loaded))
		}
		doubler, answer := loaded[0], loaded[1]
		if doubler.ID != moves[i-1].positionID {
			doubler, answer = answer, doubler
		}
		if !gammonnet.IsResponsePosition(&answer) {
			t.Fatalf("move %d (%s): stored with cube %+v, want the turned cube held by no one", i, mv.action, answer.Cube)
		}
		// Positions are stored with the player on roll as 0.
		rebuilt := gammonnet.DoublerPosition(answer)
		rebuilt = rebuilt.NormalizeForStorage()
		if !sameCheckers(rebuilt.Board, doubler.Board) || rebuilt.Cube != doubler.Cube || rebuilt.PlayerOnRoll != doubler.PlayerOnRoll || rebuilt.Score != doubler.Score {
			t.Fatalf("move %d: the rebuilt doubler position differs from the double's", i)
		}

		direct, err := gammonnet.EvaluatePositionWithMET(nil, doubler, nil, 0, 0, 0)
		if err != nil || direct.Cube == nil {
			t.Fatalf("direct evaluation of the double: %v", err)
		}
		want, ok := engine.CubeActionError(direct.Cube, mv.action)
		if !ok {
			t.Fatalf("CubeActionError(%q) not ok", mv.action)
		}
		if mv.cubeErrMP == nil {
			t.Fatalf("move %d (%s): no cube_error stored", i, mv.action)
		}
		if got := float64(*mv.cubeErrMP) / 1000; math.Abs(got-want) > 0.0015 {
			t.Errorf("move %d (%s): stored error %.4f, direct analysis %.4f", i, mv.action, got, want)
		}

		fMove := 0
		if mv.player == -1 {
			fMove = 1
		}
		score0 := matchLength - domain.PointsAway(answer.Score[0])
		score1 := matchLength - domain.PointsAway(answer.Score[1])
		wantMWC[fMove] += engine.ConvertEMGLossToMWCLoss(int(*mv.cubeErrMP), score0, score1, fMove, 1<<doubler.Cube.Value, matchLength)

		if engine.CanonicalCubeAction(mv.action) == engine.CubeTake {
			takes++
		} else {
			passes++
		}
	}
	if takes == 0 || passes == 0 {
		t.Fatalf("the fixture yields %d takes and %d passes, want both", takes, passes)
	}

	stats, err := d.GetMatchDetailStats(saved.MatchID)
	if err != nil {
		t.Fatalf("GetMatchDetailStats: %v", err)
	}
	for side, got := range []float64{stats.Player1.TakeMWCLoss, stats.Player2.TakeMWCLoss} {
		if math.Abs(got-wantMWC[side]) > 1e-6 {
			t.Errorf("player %d take MWC loss %.6f, direct analysis %.6f", side+1, got, wantMWC[side])
		}
	}
}

// sameCheckers compares two boards by their checkers: an empty point's colour
// carries nothing.
func sameCheckers(a, b domain.Board) bool {
	if a.Bearoff != b.Bearoff {
		return false
	}
	for i := range a.Points {
		pa, pb := a.Points[i], b.Points[i]
		if pa.Checkers != pb.Checkers || (pa.Checkers > 0 && pa.Color != pb.Color) {
			return false
		}
	}
	return true
}
