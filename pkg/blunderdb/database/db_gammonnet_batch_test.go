package database

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// racePosition mirrors internal/gui/gammonnet_eval_test.go's helper: enough
// structure for a legal 0-ply search without the full opening board.
func racePosition(whitePoint, blackPoint int, onRoll int) Position {
	var p Position
	p.Board.Points[whitePoint] = domain.Point{Checkers: 15, Color: domain.White}
	p.Board.Points[blackPoint] = domain.Point{Checkers: 15, Color: domain.Black}
	p.PlayerOnRoll = onRoll
	p.Dice = [2]int{0, 0}
	p.Score = [2]int{-1, -1} // money
	p.Cube = domain.Cube{Owner: domain.None, Value: 0}
	return p
}

func newBatchTestDB(t *testing.T) *Database {
	t.Helper()
	d := NewDatabase()
	if err := d.SetupDatabase(":memory:"); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

// TestAnalyzeMissingWithGammonNetFillsOnlyTheGap covers ADR-0013's rule: a
// position that already has ANY analysis is left alone, and one with none
// gets a gammonNet analysis written.
func TestAnalyzeMissingWithGammonNetFillsOnlyTheGap(t *testing.T) {
	d := newBatchTestDB(t)

	analysed := racePosition(6, 19, domain.White)
	analysedID, err := d.SavePosition(&analysed)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	if err := d.SaveAnalysis(analysedID, PositionAnalysis{
		PositionID:            int(analysedID),
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "XG",
		DoublingCubeAnalysis:  &DoublingCubeAnalysis{AnalysisDepth: "4-ply", AnalysisEngine: "XG", BestCubeAction: "No Double"},
	}); err != nil {
		t.Fatalf("SaveAnalysis (pre-existing XG): %v", err)
	}

	unanalysed := racePosition(8, 17, domain.White)
	unanalysedID, err := d.SavePosition(&unanalysed)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}

	n, err := d.CountPositionsWithoutAnalysis()
	if err != nil {
		t.Fatalf("CountPositionsWithoutAnalysis: %v", err)
	}
	if n != 1 {
		t.Fatalf("CountPositionsWithoutAnalysis = %d, want 1 (the XG-analysed position must not count)", n)
	}

	var progressCalls [][2]int
	err = d.AnalyzeMissingWithGammonNet(context.Background(), 0, 0, 0, nil, func(done, total int) {
		progressCalls = append(progressCalls, [2]int{done, total})
	})
	if err != nil {
		t.Fatalf("AnalyzeMissingWithGammonNet: %v", err)
	}
	if len(progressCalls) != 1 || progressCalls[0] != [2]int{1, 1} {
		t.Errorf("progress calls = %v, want exactly one {1,1}", progressCalls)
	}

	// The pre-existing XG analysis is untouched.
	got, err := d.LoadAnalysis(analysedID)
	if err != nil {
		t.Fatalf("LoadAnalysis(analysed): %v", err)
	}
	if got.DoublingCubeAnalysis == nil || got.DoublingCubeAnalysis.AnalysisEngine != "XG" {
		t.Errorf("pre-existing XG analysis was overwritten: %+v", got.DoublingCubeAnalysis)
	}

	// The gap is filled, by gammonNet, at the analysis depth requested (0-ply
	// here — the test's own choice for speed, not the canonical default).
	got2, err := d.LoadAnalysis(unanalysedID)
	if err != nil {
		t.Fatalf("LoadAnalysis(unanalysed): %v", err)
	}
	if got2.DoublingCubeAnalysis == nil {
		t.Fatal("expected a cube decision written for the previously-unanalysed position")
	}
	if got2.DoublingCubeAnalysis.AnalysisEngine != "gammonNet v1.0.1" {
		t.Errorf("AnalysisEngine = %q, want gammonNet v1.0.1", got2.DoublingCubeAnalysis.AnalysisEngine)
	}
	if got2.DoublingCubeAnalysis.AnalysisDepth != "0-ply" {
		t.Errorf("AnalysisDepth = %q, want 0-ply", got2.DoublingCubeAnalysis.AnalysisDepth)
	}

	n2, err := d.CountPositionsWithoutAnalysis()
	if err != nil {
		t.Fatalf("CountPositionsWithoutAnalysis (after): %v", err)
	}
	if n2 != 0 {
		t.Errorf("CountPositionsWithoutAnalysis after the run = %d, want 0", n2)
	}
}

// TestAnalyzeMissingWithGammonNetResumeIsIdempotent: a cancelled run leaves
// already-written positions alone, and a second call only ever sees what is
// still missing — the resume mechanism ADR-0013 asks for, with no journal.
func TestAnalyzeMissingWithGammonNetResumeIsIdempotent(t *testing.T) {
	d := newBatchTestDB(t)

	var ids []int64
	for i, pt := range []int{8, 17, 20} {
		pos := racePosition(pt, pt+2, domain.White)
		id, err := d.SavePosition(&pos)
		if err != nil {
			t.Fatalf("SavePosition %d: %v", i, err)
		}
		ids = append(ids, id)
	}

	ctx, cancel := context.WithCancel(context.Background())
	seen := 0
	err := d.AnalyzeMissingWithGammonNet(ctx, 0, 0, 0, func() {
		seen++
		if seen == 2 {
			cancel() // abort after the first position is written, before the second starts
		}
	}, nil)
	if err == nil {
		t.Fatal("expected context.Canceled from the aborted run")
	}

	n, err := d.CountPositionsWithoutAnalysis()
	if err != nil {
		t.Fatalf("CountPositionsWithoutAnalysis: %v", err)
	}
	if n == 0 || n == len(ids) {
		t.Fatalf("CountPositionsWithoutAnalysis after cancel = %d, want strictly between 0 and %d", n, len(ids))
	}

	// Resume: a fresh call with no cancellation finishes the rest.
	if err := d.AnalyzeMissingWithGammonNet(context.Background(), 0, 0, 0, nil, nil); err != nil {
		t.Fatalf("resume AnalyzeMissingWithGammonNet: %v", err)
	}
	n2, err := d.CountPositionsWithoutAnalysis()
	if err != nil {
		t.Fatalf("CountPositionsWithoutAnalysis (final): %v", err)
	}
	if n2 != 0 {
		t.Errorf("CountPositionsWithoutAnalysis after resume = %d, want 0", n2)
	}
}

// TestAnalyzeMissingWithGammonNetYieldGatesEachPosition is the other half of
// #129's preemption criterion (internal/gui's TestWaitForInteractiveEvaluation*
// covers the actual signal the GUI wires in): a yield that blocks must
// genuinely stall the loop before the next position, not just be called and
// ignored — "un lot qui ne cède pas est exactement la panne qu'on prétend
// éviter".
func TestAnalyzeMissingWithGammonNetYieldGatesEachPosition(t *testing.T) {
	d := newBatchTestDB(t)

	for i, pt := range []int{8, 17} {
		pos := racePosition(pt, pt+2, domain.White)
		if _, err := d.SavePosition(&pos); err != nil {
			t.Fatalf("SavePosition %d: %v", i, err)
		}
	}

	release := make(chan struct{})
	var processedBeforeRelease int
	done := make(chan error, 1)

	go func() {
		calls := 0
		done <- d.AnalyzeMissingWithGammonNet(context.Background(), 0, 0, 0, func() {
			calls++
			if calls == 2 {
				<-release // the second position's yield blocks until told to proceed
			}
		}, nil)
	}()

	// While the second position's yield is blocked, only the first must have
	// been written — the loop must not have raced ahead of its own gate.
	time.Sleep(100 * time.Millisecond)
	n, err := d.CountPositionsWithoutAnalysis()
	if err != nil {
		t.Fatalf("CountPositionsWithoutAnalysis (mid-block): %v", err)
	}
	processedBeforeRelease = 2 - n
	if processedBeforeRelease != 1 {
		t.Fatalf("positions processed while yield was blocked = %d, want exactly 1 (the yield did not gate the second position)", processedBeforeRelease)
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatalf("AnalyzeMissingWithGammonNet: %v", err)
	}
	if n2, _ := d.CountPositionsWithoutAnalysis(); n2 != 0 {
		t.Errorf("CountPositionsWithoutAnalysis after release = %d, want 0", n2)
	}
}
