package database

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

func displayedVerdict(t *testing.T, pos Position) PositionAnalysis {
	t.Helper()
	res, err := gammonnet.EvaluatePosition(pos, 0, 0, 0)
	if err != nil {
		t.Fatalf("EvaluatePosition: %v", err)
	}
	v := PositionAnalysis{DoublingCubeAnalysis: res.Cube}
	if len(res.Moves) > 0 {
		v.CheckerAnalysis = &CheckerAnalysis{Moves: res.Moves}
	}
	return v
}

// A scratch board saved before carries an XGID and a blank cube block: it is
// unevaluated, receives the verdict, keeps its XGID, and a second save keeps
// the first verdict.
func TestSaveDisplayedEvaluationFillsAScratchBoardOnce(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)

	pos := racePosition(6, 19, 0)
	id, err := d.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	if err := d.SaveAnalysis(id, PositionAnalysis{XGID: "XGID=scratch", DoublingCubeAnalysis: &DoublingCubeAnalysis{}}); err != nil {
		t.Fatalf("SaveAnalysis (scratch): %v", err)
	}

	written, err := d.SaveDisplayedEvaluation(id, displayedVerdict(t, pos), 0)
	if err != nil || written != EvaluationWritten {
		t.Fatalf("SaveDisplayedEvaluation = %v, %v; want true, nil", written, err)
	}
	got, err := d.LoadAnalysis(id)
	if err != nil {
		t.Fatalf("LoadAnalysis: %v", err)
	}
	if got.DoublingCubeAnalysis == nil || got.DoublingCubeAnalysis.AnalysisEngine != gammonnet.EngineVersion || got.DoublingCubeAnalysis.AnalysisDepth != "0-ply" {
		t.Fatalf("stored cube = %+v, want gammonNet 0-ply", got.DoublingCubeAnalysis)
	}
	if got.XGID != "XGID=scratch" {
		t.Errorf("XGID = %q, the scratch board's must be kept", got.XGID)
	}

	again, err := d.SaveDisplayedEvaluation(id, displayedVerdict(t, pos), 0)
	if err != nil || again != EvaluationKept {
		t.Errorf("second SaveDisplayedEvaluation = %v, %v; want false, nil", again, err)
	}
}

// An imported analysis is never replaced by the board's verdict.
func TestSaveDisplayedEvaluationKeepsAnExistingAnalysis(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)

	pos := racePosition(8, 17, 0)
	id, err := d.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	if err := d.SaveAnalysis(id, PositionAnalysis{
		AnalysisType:          "DoublingCube",
		AnalysisEngineVersion: "XG",
		DoublingCubeAnalysis:  &DoublingCubeAnalysis{AnalysisDepth: "4-ply", AnalysisEngine: "XG", BestCubeAction: "No Double"},
	}); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}

	written, err := d.SaveDisplayedEvaluation(id, displayedVerdict(t, pos), 0)
	if err != nil || written != EvaluationKept {
		t.Fatalf("SaveDisplayedEvaluation = %v, %v; want false, nil", written, err)
	}
	got, err := d.LoadAnalysis(id)
	if err != nil {
		t.Fatalf("LoadAnalysis: %v", err)
	}
	if got.DoublingCubeAnalysis == nil || got.DoublingCubeAnalysis.AnalysisEngine != "XG" || len(got.AllCubeAnalyses) > 1 {
		t.Errorf("the XG analysis was touched: %+v / %d cube entries", got.DoublingCubeAnalysis, len(got.AllCubeAnalyses))
	}
}

// With dice set the verdict is the move list, stored as a checker analysis.
func TestSaveDisplayedEvaluationStoresMoves(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)

	pos := racePosition(6, 19, 0)
	pos.Dice = [2]int{6, 5}
	id, err := d.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	verdict := displayedVerdict(t, pos)
	if verdict.CheckerAnalysis == nil {
		t.Fatal("expected moves for a position with dice")
	}
	written, err := d.SaveDisplayedEvaluation(id, verdict, 0)
	if err != nil || written != EvaluationWritten {
		t.Fatalf("SaveDisplayedEvaluation = %v, %v; want true, nil", written, err)
	}
	got, err := d.LoadAnalysis(id)
	if err != nil {
		t.Fatalf("LoadAnalysis: %v", err)
	}
	if got.AnalysisType != "CheckerMove" || got.CheckerAnalysis == nil || len(got.CheckerAnalysis.Moves) == 0 {
		t.Fatalf("stored = %s with %v, want a checker analysis", got.AnalysisType, got.CheckerAnalysis)
	}
	if e := got.CheckerAnalysis.Moves[0].AnalysisEngine; e != gammonnet.EngineVersion {
		t.Errorf("move engine = %q, want %q", e, gammonnet.EngineVersion)
	}
}

func TestSaveDisplayedEvaluationRefusesAnEmptyVerdict(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)
	if _, err := d.SaveDisplayedEvaluation(1, PositionAnalysis{}, 0); err == nil {
		t.Error("an empty verdict must be refused")
	}
}

// A verdict valued with another table than the library's current one is not
// written: its label would name a table that did not value it.
func TestSaveDisplayedEvaluationRefusesAnotherTable(t *testing.T) {
	t.Parallel()
	d := newBatchTestDB(t)

	pos := racePosition(6, 19, 0)
	id, err := d.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	got, err := d.SaveDisplayedEvaluation(id, displayedVerdict(t, pos), 42)
	if err != nil || got != EvaluationMETChanged {
		t.Fatalf("SaveDisplayedEvaluation = %v, %v; want metChanged, nil", got, err)
	}
	if a, err := d.LoadAnalysis(id); err == nil && analysisHasEvaluation(a) {
		t.Errorf("a verdict was written under another table: %+v", a)
	}
}
