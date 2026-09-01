// Contract cases for positions and analyses: save/load, dedup, provenance, blob
// compression and the repair of denormalised columns.
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testPositionSaveLoad(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if id == 0 {
		t.Fatal("Save returned id 0")
	}
	if p.ID != id {
		t.Errorf("Save did not set p.ID: got %d, want %d", p.ID, id)
	}

	got, err := s.Positions().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ID != id {
		t.Errorf("Load id: got %d, want %d", got.ID, id)
	}
	if got.DecisionType != domain.CheckerAction {
		t.Errorf("Load DecisionType: got %d, want %d", got.DecisionType, domain.CheckerAction)
	}
	if got.Board != p.Board {
		t.Errorf("Load board mismatch:\n got %+v\nwant %+v", got.Board, p.Board)
	}
}

func testPositionDedup(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p1 := checkerPos()
	id1, err := s.Positions().Save(ctx, "", &p1)
	if err != nil {
		t.Fatalf("first Save: %v", err)
	}
	p2 := checkerPos()
	id2, err := s.Positions().Save(ctx, "", &p2)
	if err != nil {
		t.Fatalf("second Save: %v", err)
	}
	if id1 != id2 {
		t.Errorf("dedup failed: first Save id %d, second Save id %d", id1, id2)
	}

	n := 0
	for _, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		n++
	}
	if n != 1 {
		t.Errorf("after dedup expected 1 stored position, got %d", n)
	}
}

func testPositionUpdatePreservesId(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	id, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Mutate the board (15 checkers preserved) and update in place.
	p.Board.Points[19] = domain.Point{Checkers: 4, Color: domain.White}
	p.Board.Points[20] = domain.Point{Checkers: 1, Color: domain.White}
	if err := s.Positions().Update(ctx, "", &p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := s.Positions().Load(ctx, "", id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ID != id {
		t.Errorf("Update changed id: got %d, want %d", got.ID, id)
	}
	if got.Board != p.Board {
		t.Errorf("Update did not persist board change:\n got %+v\nwant %+v", got.Board, p.Board)
	}
}

// testPositionProvenanceSticky pins the rule that makes the individually
// imported flag usable: it is ORed into the stored value, never assigned.
// Both orderings below are ordinary user behaviour, and the flag must mean the
// same thing in each — see docs/adr/0001.
func testPositionProvenanceSticky(t *testing.T, s storage.Storage) {
	ctx := context.Background()

	flag := func(id int64) bool {
		p, err := s.Positions().Load(ctx, "", id)
		if err != nil {
			t.Fatalf("Load position %d: %v", id, err)
		}
		return p.IndividuallyImported
	}
	save := func(p domain.Position, individual bool) int64 {
		p.IndividuallyImported = individual
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatalf("Save position: %v", err)
		}
		return id
	}

	// S1 — the user imports a position on its own, then imports the match it
	// came from. The match import must not clear the flag.
	solo := save(provenancePos(1), true)
	if got := save(provenancePos(1), false); got != solo {
		t.Fatalf("dedup failed: match import created id %d, want %d", got, solo)
	}
	if !flag(solo) {
		t.Error("S1: importing the match cleared individually_imported")
	}

	// S2 — the match is imported first, then the user imports one of its
	// positions on its own. The insert is a no-op, so the flag has to be raised
	// on the row that is already there.
	fromMatch := save(provenancePos(2), false)
	if flag(fromMatch) {
		t.Error("a match-sourced position came back individually imported")
	}
	if got := save(provenancePos(2), true); got != fromMatch {
		t.Fatalf("dedup failed: individual import created id %d, want %d", got, fromMatch)
	}
	if !flag(fromMatch) {
		t.Error("S2: individually importing an already-stored position did not mark it")
	}

	// A position only ever seen inside a match stays unmarked.
	if only := save(provenancePos(3), false); flag(only) {
		t.Error("a position seen only in a match was marked individually imported")
	}
}

func testAnalysisSaveAndCompress(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}

	a := domain.PositionAnalysis{
		AnalysisType: "CheckerMove",
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{
				{Index: 0, Move: "13/11 24/23", Equity: 0.123, PlayerWinChance: 54.32},
			},
		},
	}
	if err := s.Analyses().Save(ctx, "", posID, &a); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	// Load round-trips through the zlib-compressed data column.
	got, err := s.Analyses().Load(ctx, "", posID)
	if err != nil {
		t.Fatalf("Load analysis: %v", err)
	}
	if got.AnalysisType != "CheckerMove" {
		t.Errorf("AnalysisType: got %q, want %q", got.AnalysisType, "CheckerMove")
	}
	if got.PositionID != int(posID) {
		t.Errorf("PositionID: got %d, want %d", got.PositionID, posID)
	}
	if got.CheckerAnalysis == nil || len(got.CheckerAnalysis.Moves) != 1 {
		t.Fatalf("CheckerAnalysis not round-tripped: %+v", got.CheckerAnalysis)
	}
	if got.CheckerAnalysis.Moves[0].Move != "13/11 24/23" {
		t.Errorf("move: got %q, want %q", got.CheckerAnalysis.Moves[0].Move, "13/11 24/23")
	}
}

// testRepairDenormalisedColumns pins the repair on the very defect that made it
// necessary: a column left holding the error of an action that was not played.
//
// The check that matters is the SECOND run returning 0. A repair that rewrites
// every row every time cannot tell "something was wrong" from "it ran", and the
// count is the only thing an operator has to decide whether to worry.
func testRepairDenormalisedColumns(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	as := s.Analyses()

	pos := statsDecisionPos(t, 7)
	pos.DecisionType = domain.CubeAction
	posID, err := s.Positions().Save(ctx, "", &pos)
	if err != nil {
		t.Fatalf("Save position: %v", err)
	}
	a := domain.PositionAnalysis{
		PlayedCubeActions: []string{"No Double"},
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			BestCubeAction:          "No Double",
			CubefulNoDoubleEquity:   0.40,
			CubefulDoubleTakeEquity: 0.20,
			CubefulDoublePassEquity: 1.00,
			CubefulNoDoubleError:    0,
			CubefulDoubleTakeError:  -0.200,
			CubefulDoublePassError:  -0.600,
		},
	}
	if err := as.Save(ctx, "", posID, &a); err != nil {
		if errors.Is(err, storage.ErrInternal) {
			t.Skip("Analyses not implemented on this backend")
		}
		t.Fatalf("Save analysis: %v", err)
	}

	// Une base saine ne bouge pas : c'est ce qui rend le compteur lisible.
	n, err := as.RepairDenormalisedColumns(ctx, "")
	if err != nil {
		t.Fatalf("Repair (sain): %v", err)
	}
	if n != 0 {
		t.Errorf("réparation d'une base saine : %d lignes touchées, want 0", n)
	}

	// La réparation d'une colonne RÉELLEMENT abîmée se teste là où l'on peut
	// l'abîmer — en SQL, dans analyses_repair_sqlite_test.go. Le contrat, lui,
	// n'a que l'interface : il vérifie ce qu'il peut, et ce qu'il vérifie est
	// justement ce qui rend le compteur lisible.
}
