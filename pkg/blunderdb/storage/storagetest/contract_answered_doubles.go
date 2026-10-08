package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testMatchReanchorAnsweredDoubles: a transcribed take recorded with the
// turned cube owned by the answerer shares its row with that player's own
// redouble decision. Reanchoring moves the take — and only it — onto the
// cube held by no one, merging into an importer's row when one exists, and
// purges a row nothing else holds, its stale analysis with it.
func testMatchReanchorAnsweredDoubles(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	owned := func(variant int) domain.Position {
		p := cubePos()
		p.Board.Points[5+variant] = domain.Point{Checkers: 1, Color: domain.Black}
		p.PlayerOnRoll = domain.Black
		p.Cube = domain.Cube{Owner: domain.Black, Value: 1}
		return p
	}
	save := func(p domain.Position) int64 {
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		return id
	}
	analyse := func(id int64, engine string) {
		a := &domain.PositionAnalysis{AnalysisType: "DoublingCube", AnalysisEngineVersion: engine,
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisDepth: "0-ply", AnalysisEngine: engine, BestCubeAction: "No Double"}}
		if err := s.Analyses().Save(ctx, "", id, a); err != nil {
			t.Fatalf("Save analysis: %v", err)
		}
	}
	move := func(filePath, action string, posID int64) (matchID, moveID int64) {
		m := domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 7, FilePath: filePath}
		mid, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		gid, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: mid, GameNumber: 1})
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		mvID, err := s.Matches().CreateMove(ctx, "", &domain.Move{GameID: gid, MoveNumber: 1, MoveType: "cube", PositionID: posID, Player: -1, CubeAction: action})
		if err != nil {
			t.Fatalf("CreateMove: %v", err)
		}
		return mid, mvID
	}
	positionOf := func(matchID int64) int64 {
		for mv, err := range s.Matches().MovesByMatch(ctx, "", matchID) {
			if err != nil {
				t.Fatal(err)
			}
			return mv.PositionID
		}
		t.Fatalf("match %d has no move", matchID)
		return 0
	}

	// Collision: a redouble imported with its analysis, an importer's reply
	// row, a transcribed take on the redouble's row, and an imported take
	// left on the doubler's row (a file without a separate response).
	shared := owned(0)
	sharedID := save(shared)
	analyse(sharedID, "XG")
	reply := shared
	reply.Cube.Owner = domain.None
	replyID := save(reply)
	redouble, _ := move("a.xg", "Double", sharedID)
	fallback, _ := move("b.xg", "Take", sharedID)
	transcribedTake, _ := move("", "Take", sharedID)

	// Alone: a transcribed pass nothing else holds, analysed as a redouble.
	lone := owned(1)
	loneID := save(lone)
	analyse(loneID, "gammonNet")
	transcribedPass, _ := move("", "Pass", loneID)

	moved, err := s.Matches().ReanchorAnsweredDoubles(ctx, "")
	if err != nil {
		t.Fatalf("ReanchorAnsweredDoubles: %v", err)
	}
	if moved != 2 {
		t.Fatalf("moved %d answers, want 2", moved)
	}
	if got := positionOf(transcribedTake); got != replyID {
		t.Errorf("transcribed take on %d, want the importer's reply row %d", got, replyID)
	}
	for _, m := range []int64{redouble, fallback} {
		if got := positionOf(m); got != sharedID {
			t.Errorf("imported move of match %d moved to %d, want %d", m, got, sharedID)
		}
	}
	if a, err := s.Analyses().Load(ctx, "", sharedID); err != nil || a.AnalysisEngineVersion != "XG" {
		t.Errorf("the redouble's analysis: %v, %v", a, err)
	}
	if _, err := s.Positions().Load(ctx, "", loneID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("the row only the pass held survives: %v", err)
	}
	newID := positionOf(transcribedPass)
	p, err := s.Positions().Load(ctx, "", newID)
	if err != nil {
		t.Fatal(err)
	}
	if p.Cube.Owner != domain.None || p.Cube.Value != 1 {
		t.Errorf("the pass stands on cube %+v, want value 1 held by no one", p.Cube)
	}
	if _, err := s.Analyses().Load(ctx, "", newID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("the pass kept the redouble's analysis: %v", err)
	}

	if again, err := s.Matches().ReanchorAnsweredDoubles(ctx, ""); err != nil || again != 0 {
		t.Errorf("second run moved %d (%v), want 0", again, err)
	}
}
