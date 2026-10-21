// Contract cases for the batched position and analysis writes a bulk copy
// uses.
package storagetest

import (
	"context"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testPositionSaveManyEqualsSaveCreated pins SaveMany against SaveCreated: ids
// and created in call order, a hash already stored (whose mark is raised, as
// SaveCreated raises it), and a hash given twice in one call.
func testPositionSaveManyEqualsSaveCreated(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	stored := provenancePos(1)
	storedID, err := s.Positions().Save(ctx, "", &stored)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	again := provenancePos(1)
	again.Flagged = true
	first, twice, other := provenancePos(2), provenancePos(2), provenancePos(3)
	ps := []*domain.Position{&first, &again, &other, &twice}
	ids, created, err := s.Positions().SaveMany(ctx, "", ps)
	if err != nil {
		t.Fatalf("SaveMany: %v", err)
	}
	if len(ids) != len(ps) || len(created) != len(ps) {
		t.Fatalf("SaveMany answered %d ids and %d created for %d positions", len(ids), len(created), len(ps))
	}
	if want := []bool{true, false, true, false}; !reflect.DeepEqual(created, want) {
		t.Errorf("created = %v, want %v", created, want)
	}
	if ids[1] != storedID {
		t.Errorf("stored hash got id %d, want %d", ids[1], storedID)
	}
	if ids[3] != ids[0] {
		t.Errorf("hash given twice got ids %d and %d", ids[0], ids[3])
	}
	if ids[0] == ids[2] || ids[0] == storedID || ids[2] == storedID {
		t.Errorf("distinct hashes share an id: %v (stored %d)", ids, storedID)
	}
	for i, p := range ps {
		if p.ID != ids[i] {
			t.Errorf("ps[%d].ID = %d, want %d", i, p.ID, ids[i])
		}
	}
	got, err := s.Positions().Load(ctx, "", storedID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !got.Flagged {
		t.Error("SaveMany did not raise the mark of a stored position")
	}
	if n, err := s.Positions().Count(ctx, ""); err != nil || n != 3 {
		t.Errorf("Count = %d, %v; want 3", n, err)
	}
	if ids, created, err := s.Positions().SaveMany(ctx, "", nil); err != nil || len(ids) != 0 || len(created) != 0 {
		t.Errorf("SaveMany(nil) = %v, %v, %v", ids, created, err)
	}
}

// testAnalysisSaveManyEqualsSave pins SaveMany against Save: a checker
// analysis, whose columns read the position's plays, and two cube analyses,
// whose played action decides the cube-response flag the search filters on.
// The positions SaveMany created are named fresh, as a bulk copy names them;
// one analysis replaces another, on a position that is not.
func testAnalysisSaveManyEqualsSave(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	checker := func() *domain.PositionAnalysis {
		e := 0.12
		return &domain.PositionAnalysis{
			AnalysisType: "XG Roller++",
			PlayedMoves:  []string{"13/11 24/23"},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{Move: "8/6 6/4", Equity: 0.50},
				{Move: "13/11 24/23", Equity: 0.38, EquityError: &e},
			}},
		}
	}
	unit, batch := statsDecisionPos(t, 0), statsDecisionPos(t, 1)
	unitID, err := s.Positions().Save(ctx, "", &unit)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Analyses().Save(ctx, "", unitID, checker()); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}

	takePos := cubePos()
	takePos.Cube = domain.Cube{Owner: -1, Value: 1}
	doublePos := cubePos()
	doublePos.Board.Points[5] = domain.Point{Checkers: 1, Color: domain.White}
	ids, created, err := s.Positions().SaveMany(ctx, "", []*domain.Position{&batch, &takePos, &doublePos})
	if err != nil {
		t.Fatalf("SaveMany positions: %v", err)
	}
	var fresh []int64
	for i, id := range ids {
		if created[i] {
			fresh = append(fresh, id)
		}
	}
	if len(fresh) != 3 {
		t.Fatalf("created = %v, want three new positions", created)
	}
	analyses := map[int64]*domain.PositionAnalysis{
		ids[0]: checker(),
		ids[1]: {PlayedCubeActions: []string{"Take"}},
		ids[2]: {PlayedCubeActions: []string{"Double"}, DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			AnalysisDepth: "3-ply", PlayerWinChances: 71.5, PlayerGammonChances: 22.1,
			OpponentWinChances: 28.5, OpponentGammonChances: 6.3,
			CubefulNoDoubleEquity: 0.61, CubefulNoDoubleError: -0.08,
			CubefulDoubleTakeEquity: 0.69, CubefulDoublePassEquity: 1,
			CubefulDoublePassError: 0.31, BestCubeAction: "Double, Take",
			WrongPassPercentage: 12.5, WrongTakePercentage: 3.5,
		}},
		unitID: checker(),
	}
	analyses[unitID].AnalysisType = "GnuBG 2-ply"
	if err := s.Analyses().SaveMany(storage.WithFreshPositions(ctx, fresh), "", analyses); err != nil {
		t.Fatalf("SaveMany analyses: %v", err)
	}
	// The scalar columns (best move error, rates, best cube action) are the
	// projection Save writes: a repair from the stored JSON changes nothing.
	if n, err := s.Analyses().RepairDenormalisedColumns(ctx, ""); err != nil || n != 0 {
		t.Errorf("RepairDenormalisedColumns after SaveMany = %d, %v; want 0 rows changed", n, err)
	}

	got, err := s.Analyses().LoadMany(ctx, "", []int64{unitID, ids[0]})
	if err != nil {
		t.Fatalf("LoadMany: %v", err)
	}
	if got[unitID] == nil || got[unitID].AnalysisType != "GnuBG 2-ply" {
		t.Errorf("SaveMany did not replace the stored analysis: %+v", got[unitID])
	}
	if err := s.Analyses().Save(ctx, "", unitID, checker()); err != nil {
		t.Fatalf("Save analysis: %v", err)
	}
	want, err := s.Analyses().Load(ctx, "", unitID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	b := got[ids[0]]
	if b == nil {
		t.Fatal("SaveMany stored no analysis for a fresh position")
	}
	b.PositionID, want.PositionID = 0, 0
	b.XGID, want.XGID = "", ""
	if !reflect.DeepEqual(b.CheckerAnalysis, want.CheckerAnalysis) || !reflect.DeepEqual(b.PlayedMoves, want.PlayedMoves) {
		t.Errorf("SaveMany stored %+v, Save stored %+v", b, want)
	}

	search := func(sub string) []int64 {
		f := domain.SearchFilters{DecisionTypeFilter: true, CubeResponseFilter: sub}
		f.Filter.DecisionType = domain.CubeAction
		f.Filter.PlayerOnRoll = takePos.PlayerOnRoll
		var found []int64
		for pos, err := range s.Search().Find(ctx, "", f, storage.ListOpts{}) {
			if err != nil {
				t.Fatalf("Find(%q): %v", sub, err)
			}
			found = append(found, pos.ID)
		}
		return found
	}
	if got := search("takepass"); len(got) != 1 || got[0] != ids[1] {
		t.Errorf("takepass filter: got %v, want [%d]", got, ids[1])
	}
	if got := search("double"); len(got) != 1 || got[0] != ids[2] {
		t.Errorf("double filter: got %v, want [%d]", got, ids[2])
	}
}
