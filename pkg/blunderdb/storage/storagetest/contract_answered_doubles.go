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
	// game records the moves of one game of a new match, each (action,
	// position) in turn, all by one player, and returns their ids. An
	// action written "Double>" is the opponent's double the next move
	// answers, as a transcription records it.
	matchOf := map[int64]int64{}
	game := func(filePath string, moves ...any) []int64 {
		m := domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 7, FilePath: filePath}
		mid, err := s.Matches().Save(ctx, "", &m)
		if err != nil {
			t.Fatalf("Save match: %v", err)
		}
		gid, err := s.Matches().CreateGame(ctx, "", &domain.Game{MatchID: mid, GameNumber: 1})
		if err != nil {
			t.Fatalf("CreateGame: %v", err)
		}
		var ids []int64
		for i := 0; i < len(moves); i += 2 {
			action, player := moves[i].(string), int32(-1)
			if action == "Double>" {
				action, player = "Double", 1
			}
			mvID, err := s.Matches().CreateMove(ctx, "", &domain.Move{GameID: gid, MoveNumber: int32(i/2 + 1), MoveType: "cube",
				PositionID: moves[i+1].(int64), Player: player, CubeAction: action})
			if err != nil {
				t.Fatalf("CreateMove: %v", err)
			}
			ids = append(ids, mvID)
			matchOf[mvID] = mid
		}
		return ids
	}
	positionOf := func(moveID int64) int64 {
		t.Helper()
		for mv, err := range s.Matches().MovesByMatch(ctx, "", matchOf[moveID]) {
			if err != nil {
				t.Fatal(err)
			}
			if mv.ID == moveID {
				return mv.PositionID
			}
		}
		t.Fatalf("move %d not found", moveID)
		return 0
	}

	// Collision: a redouble imported with its analysis, an importer's reply
	// row, a transcribed take on the redouble's row, and an imported take
	// left on the doubler's row after the double of its own game (a file
	// without a separate response).
	shared := owned(0)
	sharedID := save(shared)
	analyse(sharedID, "XG")
	reply := shared
	reply.Cube.Owner = domain.None
	replyID := save(reply)
	doubler := cubePos()
	doubler.Board.Points[12] = domain.Point{Checkers: 1, Color: domain.White}
	doublerID := save(doubler)
	redouble := game("a.xg", "Double", sharedID)[0]
	fallback := game("b.xg", "Double", sharedID, "Take", sharedID)
	transcribedTake := game("", "Double>", doublerID, "Take", sharedID)[1]
	// An XG file without its raw cube segment records a redouble taken as
	// the doubler's own take, alone on its pre-double row: the same shape
	// as a transcribed answer, but no opponent's Double precedes it.
	xgOwnedID := save(owned(4))
	xgTake := game("d.xg", "Take", xgOwnedID)[0]

	// Alone: a transcribed pass nothing else holds, analysed as a redouble.
	lone := owned(1)
	loneID := save(lone)
	analyse(loneID, "gammonNet")
	transcribedPass := game("", "Double>", doublerID, "Pass", loneID)[1]

	// An imported match corrected through a transcription keeps its file
	// but records its take the transcript's way.
	correctedID := save(owned(2))
	correctedTake := game("c.xg", "Double>", doublerID, "Take", correctedID)[1]

	// A take on a row the user brought on his own and flagged: the row
	// stays, its provenance does not follow the take.
	mine := owned(3)
	mine.IndividuallyImported, mine.Flagged = true, true
	mineID := save(mine)
	mineTake := game("", "Double>", doublerID, "Take", mineID)[1]

	moved, err := s.Matches().ReanchorAnsweredDoubles(ctx, "")
	if err != nil {
		t.Fatalf("ReanchorAnsweredDoubles: %v", err)
	}
	if moved != 4 {
		t.Fatalf("moved %d answers, want 4", moved)
	}
	if got := positionOf(transcribedTake); got != replyID {
		t.Errorf("transcribed take on %d, want the importer's reply row %d", got, replyID)
	}
	for _, m := range append([]int64{redouble, xgTake}, fallback...) {
		want := sharedID
		if m == xgTake {
			want = xgOwnedID
		}
		if got := positionOf(m); got != want {
			t.Errorf("imported move %d moved to %d, want %d", m, got, want)
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
	if got := positionOf(correctedTake); got == correctedID {
		t.Errorf("the corrected imported match's take stayed on the answerer's owned cube")
	}
	landed, err := s.Positions().Load(ctx, "", positionOf(mineTake))
	if err != nil {
		t.Fatal(err)
	}
	if landed.ID == mineID || landed.IndividuallyImported || landed.Flagged {
		t.Errorf("the take landed on %d (individually imported %v, flagged %v), want a new row without provenance",
			landed.ID, landed.IndividuallyImported, landed.Flagged)
	}
	if _, err := s.Positions().Load(ctx, "", mineID); err != nil {
		t.Errorf("the user's own row was purged: %v", err)
	}

	// The landed rows are responses to the search's take/pass filter.
	f := domain.SearchFilters{DecisionTypeFilter: true, CubeResponseFilter: "takepass"}
	f.Filter.DecisionType = domain.CubeAction
	f.Filter.PlayerOnRoll = domain.Black
	responses := map[int64]bool{}
	for pos, err := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("Find take/pass: %v", err)
		}
		responses[pos.ID] = true
	}
	for _, m := range []int64{transcribedTake, transcribedPass, correctedTake, mineTake} {
		if id := positionOf(m); !responses[id] {
			t.Errorf("the row %d move %d landed on is not a take/pass response to the search", id, m)
		}
	}

	if again, err := s.Matches().ReanchorAnsweredDoubles(ctx, ""); err != nil || again != 0 {
		t.Errorf("second run moved %d (%v), want 0", again, err)
	}
}
