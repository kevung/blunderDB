package database

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
)

// tinySettings keeps the engine's part under a second.
func tinySettings() rollout.Settings {
	s := rollout.Fast()
	s.MaxGames, s.MinGames, s.Truncation, s.Candidates = 36, 36, 2, 2
	return s
}

// A stored rollout sits beside the imported analysis; saving that analysis
// again — an import, the GUI's own save — keeps it; the batch skips the
// position on the next run.
func TestRolloutPosition_StoresBesideAndSurvivesSaveAnalysis(t *testing.T) {
	d := newTestDB(t)
	p := domain.InitializePosition()
	id, err := d.SavePosition(&p)
	if err != nil {
		t.Fatal(err)
	}
	imported := PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
		{Move: "8/5 6/5", Equity: 0.1, AnalysisEngine: "XG", AnalysisDepth: "XG Roller++"},
	}}}
	if err := d.SaveAnalysis(id, imported); err != nil {
		t.Fatal(err)
	}

	s := tinySettings()
	res, err := d.RolloutPosition(context.Background(), id, s, nil, true, nil)
	if err != nil || res == nil {
		t.Fatalf("RolloutPosition: %v", err)
	}
	if err := d.SaveAnalysis(id, imported); err != nil {
		t.Fatal(err)
	}
	a, err := d.LoadAnalysis(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Rollouts) != 1 || a.Rollouts[0].Signature != s.Signature() {
		t.Fatalf("rollouts after a re-save: %+v", a.Rollouts)
	}
	for _, m := range a.CheckerAnalysis.Moves {
		if m.AnalysisEngine != "XG" {
			t.Errorf("a rollout entry entered the imported analysis: %+v", m)
		}
	}

	todo, err := d.PositionsToRollout(context.Background(), domain.SearchFilters{}, s)
	if err != nil || len(todo) != 0 {
		t.Errorf("PositionsToRollout after the rollout: %d, %v", len(todo), err)
	}
	sum, err := d.RolloutFiltered(context.Background(), domain.SearchFilters{}, s, nil)
	if err != nil || sum.Total != 0 {
		t.Errorf("RolloutFiltered reran a done position: %+v, %v", sum, err)
	}
}

// Wails binds every exported method of *Database to the webview. A method
// taking a context is bound but uncallable there, like the other batch entry
// points; one taking a Result would let the webview store a rollout it made up.
func TestDatabaseBindsNoRolloutResultWriter(t *testing.T) {
	resultType := reflect.TypeOf((*rollout.Result)(nil))
	typ := reflect.TypeOf(&Database{})
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		for j := 1; j < m.Type.NumIn(); j++ {
			if m.Type.In(j) == resultType {
				t.Errorf("(*Database).%s is bound to the webview but takes a %s", m.Name, resultType)
			}
		}
	}
}

// A rollout, or a batch, started on one file writes nothing once another file
// has been opened: its positions are those of the previous library.
func TestRolloutWriteRefusedAfterTheDatabaseChanged(t *testing.T) {
	d := newTestDB(t)
	p := domain.InitializePosition()
	id, err := d.SavePosition(&p)
	if err != nil {
		t.Fatal(err)
	}
	s := tinySettings()
	ctx := context.Background()
	res, err := d.RolloutPosition(ctx, id, s, nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	gen := d.currentGeneration()
	todo, err := d.PositionsToRollout(ctx, domain.SearchFilters{}, s)
	if err != nil || len(todo) != 1 {
		t.Fatalf("PositionsToRollout: %d, %v", len(todo), err)
	}
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	if err := d.storeRollout(gen, id, res); !errors.Is(err, ErrDatabaseChanged) {
		t.Errorf("storeRollout after a switch: %v, want ErrDatabaseChanged", err)
	}
	sum, err := d.rolloutPositionsAt(ctx, gen, todo, s, nil)
	if err != nil || sum.RolledOut != 0 || sum.Failed != 1 {
		t.Errorf("batch after a switch: %+v, %v; want one failed write", sum, err)
	}
	if err := d.saveAnalysisAt(gen, id, PositionAnalysis{}, 0); !errors.Is(err, ErrDatabaseChanged) {
		t.Errorf("saveAnalysisAt after a switch: %v, want ErrDatabaseChanged", err)
	}
}

// A plan read before the file changed writes nothing into the next one.
func TestRolloutPlanRefusedAfterTheDatabaseChanged(t *testing.T) {
	d := newTestDB(t)
	p := domain.InitializePosition()
	id, err := d.SavePosition(&p)
	if err != nil {
		t.Fatal(err)
	}
	s := tinySettings()
	ctx := context.Background()
	byIDs, err := d.PlanRolloutIDs(ctx, []int64{id}, s)
	if err != nil || len(byIDs.Positions) != 1 {
		t.Fatalf("PlanRolloutIDs: %v, %v", byIDs, err)
	}
	byQuery, err := d.PlanRollout(ctx, domain.SearchFilters{}, s)
	if err != nil || len(byQuery.Positions) != 1 {
		t.Fatalf("PlanRollout: %v, %v", byQuery, err)
	}
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	for name, plan := range map[string]*RolloutPlan{"ids": byIDs, "query": byQuery} {
		sum, err := d.RunRolloutPlan(ctx, plan, s, nil)
		if err != nil || sum.RolledOut != 0 || sum.Failed != 1 {
			t.Errorf("%s plan after a switch: %+v, %v; want one failed write", name, sum, err)
		}
	}
}

// The hook registered with SetBeforeSwitch runs before every switch, outside
// the lock: a job it waits for can still take the lock to finish its write.
func TestBeforeSwitchRunsOutsideTheLock(t *testing.T) {
	d := newTestDB(t)
	calls := 0
	d.SetBeforeSwitch(func() {
		calls++
		d.mu.Lock()
		d.mu.Unlock() //nolint:staticcheck // probing that mu is free
	})
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("beforeSwitch ran %d times, want 2", calls)
	}
}

// Folding a duplicate position into the one the index holds keeps that
// one's analysis and moves the duplicate's rollouts beside it.
func TestMergePositionIntoKeepsTheDuplicatesRollouts(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	keep := domain.InitializePosition()
	keepID, err := d.SavePosition(&keep)
	if err != nil {
		t.Fatal(err)
	}
	dup := domain.InitializePosition()
	dup.Cube.Value = 1
	dup.Score = [2]int{3, 5}
	dupID, err := d.SavePosition(&dup)
	if err != nil || dupID == keepID {
		t.Fatalf("SavePosition dup: %d, %v", dupID, err)
	}
	imported := PositionAnalysis{AnalysisType: "CheckerMove", CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
		{Move: "8/5 6/5", Equity: 0.1, AnalysisEngine: "XG", AnalysisDepth: "XG Roller++"},
	}}}
	if err := d.SaveAnalysis(keepID, imported); err != nil {
		t.Fatal(err)
	}
	s := tinySettings()
	if _, err := d.RolloutPosition(ctx, dupID, s, nil, true, nil); err != nil {
		t.Fatal(err)
	}

	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := mergePositionInto(ctx, tx, keepID, dupID); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	a, err := d.LoadAnalysis(keepID)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Rollouts) != 1 {
		t.Errorf("rollouts on the kept position: %+v", a.Rollouts)
	}
	if a.CheckerAnalysis == nil || a.CheckerAnalysis.Moves[0].AnalysisEngine != "XG" {
		t.Errorf("the kept analysis did not win: %+v", a.CheckerAnalysis)
	}
}

// A list on screen is rolled out as listed: in its order, each position once,
// unknown ids dropped, and those already carrying this rollout skipped.
func TestPositionsToRolloutIDsFollowsTheList(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	s := tinySettings()
	ids := make([]int64, 0, 2)
	for _, pos := range []domain.Position{domain.InitializePosition(), domain.InitializePosition()} {
		pos.Dice = [2]int{3, 1}
		if len(ids) == 1 {
			pos.Dice = [2]int{6, 4}
		}
		id, err := d.SavePosition(&pos)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	if ids[0] == ids[1] {
		t.Skip("the two positions hash alike")
	}
	got, err := d.PositionsToRolloutIDs(ctx, []int64{ids[1], 987654, ids[0], ids[1]}, s)
	if err != nil || len(got) != 2 || got[0].ID != ids[1] || got[1].ID != ids[0] {
		t.Fatalf("PositionsToRolloutIDs: %v, %v", got, err)
	}
	if _, err := d.RolloutPosition(ctx, ids[1], s, nil, true, nil); err != nil {
		t.Fatal(err)
	}
	got, err = d.PositionsToRolloutIDs(ctx, []int64{ids[1], ids[0]}, s)
	if err != nil || len(got) != 1 || got[0].ID != ids[0] {
		t.Fatalf("after one rollout: %v, %v", got, err)
	}
}

// openReadOnlyPair opens path twice: the first instance holds the write lock,
// the second falls back to read-only, as when two blunderDB windows share a
// library. It returns the second, with a saved position of the first.
func openReadOnlyPair(t *testing.T) (*Database, int64) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "shared.db")
	writer := NewDatabase()
	if err := writer.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = writer.Close() })
	p := domain.InitializePosition()
	id, err := writer.SavePosition(&p)
	if err != nil {
		t.Fatal(err)
	}
	reader := NewDatabase()
	if err := reader.OpenDatabase(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reader.Close() })
	if !reader.IsReadOnly() {
		t.Fatal("second instance should be read-only")
	}
	return reader, id
}

// A rollout that would store is refused on a read-only database before the
// engine runs, not after its games on a failed write; one that stores
// nothing still runs.
func TestRolloutRefusedOnReadOnlyDatabase(t *testing.T) {
	d, id := openReadOnlyPair(t)
	s := tinySettings()
	ctx := context.Background()
	progressed := false
	if _, err := d.RolloutPosition(ctx, id, s, nil, true, func(rollout.Progress) { progressed = true }); !errors.Is(err, ErrReadOnly) {
		t.Errorf("RolloutPosition store: err = %v, want ErrReadOnly", err)
	}
	if progressed {
		t.Error("the engine ran before the refusal")
	}
	if _, err := d.PlanRollout(ctx, SearchFilters{}, s); !errors.Is(err, ErrReadOnly) {
		t.Errorf("PlanRollout: err = %v, want ErrReadOnly", err)
	}
	if _, err := d.PlanRolloutIDs(ctx, []int64{id}, s); !errors.Is(err, ErrReadOnly) {
		t.Errorf("PlanRolloutIDs: err = %v, want ErrReadOnly", err)
	}
	if _, err := d.RolloutFiltered(ctx, SearchFilters{}, s, nil); !errors.Is(err, ErrReadOnly) {
		t.Errorf("RolloutFiltered: err = %v, want ErrReadOnly", err)
	}
	if _, err := d.RolloutPositions(ctx, []Position{{ID: id}}, s, nil); !errors.Is(err, ErrReadOnly) {
		t.Errorf("RolloutPositions: err = %v, want ErrReadOnly", err)
	}
	if _, err := d.RolloutPosition(ctx, id, s, nil, false, nil); err != nil {
		t.Errorf("RolloutPosition without store: %v", err)
	}
}
