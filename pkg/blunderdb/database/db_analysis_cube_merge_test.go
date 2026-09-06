package database

import (
	"testing"
)

// A second engine's cube analysis must not push the first one out (#269).
//
// This used to happen: SaveAnalysis kept the incoming cube analysis whenever
// it was non-nil, so importing a GNUbg file over an XG one silently dropped
// XG's cube verdict — while the same position's CHECKER moves accumulated
// from both. ADR-0013's own text describes the behaviour this test now
// enforces ("merges engines inside it, every entry tagged with its own
// AnalysisEngine"); ingest/merge.go had it, this wrapper did not.
func TestSaveAnalysisKeepsEveryEnginesCubeAnalysis(t *testing.T) {
	db := newTestDB(t)
	pos := InitializePosition()
	pos.DecisionType = CubeAction
	id, err := db.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}

	xg := PositionAnalysis{
		AnalysisType: "DoublingCube",
		DoublingCubeAnalysis: &DoublingCubeAnalysis{
			AnalysisEngine: "XG", AnalysisDepth: "4-ply",
			BestCubeAction:          "Double, Take",
			CubefulNoDoubleEquity:   0.40,
			CubefulDoubleTakeEquity: 0.55,
			CubefulDoublePassEquity: 1.00,
		},
	}
	if err := db.SaveAnalysis(id, xg); err != nil {
		t.Fatalf("SaveAnalysis (XG): %v", err)
	}

	gnubg := PositionAnalysis{
		AnalysisType: "DoublingCube",
		DoublingCubeAnalysis: &DoublingCubeAnalysis{
			AnalysisEngine: "GNUbg", AnalysisDepth: "2-ply",
			BestCubeAction:          "No Double",
			CubefulNoDoubleEquity:   0.60,
			CubefulDoubleTakeEquity: 0.45,
			CubefulDoublePassEquity: 1.00,
		},
	}
	if err := db.SaveAnalysis(id, gnubg); err != nil {
		t.Fatalf("SaveAnalysis (GNUbg): %v", err)
	}

	got, err := db.LoadAnalysis(id)
	if err != nil {
		t.Fatalf("LoadAnalysis: %v", err)
	}
	engines := map[string]string{}
	for _, ca := range got.AllCubeAnalyses {
		engines[ca.AnalysisEngine] = ca.BestCubeAction
	}
	if len(got.AllCubeAnalyses) != 2 {
		t.Fatalf("cube analyses kept: %d (%v), want both engines", len(got.AllCubeAnalyses), engines)
	}
	if engines["XG"] != "Double, Take" {
		t.Errorf("XG's verdict was lost or changed: %q", engines["XG"])
	}
	if engines["GNUbg"] != "No Double" {
		t.Errorf("GNUbg's verdict was lost or changed: %q", engines["GNUbg"])
	}
	// The engine that just spoke is the primary one, because it is what the
	// derived scalar columns read.
	if got.DoublingCubeAnalysis == nil || got.DoublingCubeAnalysis.AnalysisEngine != "GNUbg" {
		t.Errorf("primary cube analysis = %+v, want GNUbg's", got.DoublingCubeAnalysis)
	}

	// Re-saving the SAME engine updates its entry rather than adding a third.
	xg.DoublingCubeAnalysis.BestCubeAction = "Too good to double, pass"
	if err := db.SaveAnalysis(id, xg); err != nil {
		t.Fatalf("SaveAnalysis (XG again): %v", err)
	}
	got, err = db.LoadAnalysis(id)
	if err != nil {
		t.Fatalf("LoadAnalysis: %v", err)
	}
	if len(got.AllCubeAnalyses) != 2 {
		t.Fatalf("re-saving one engine gave %d entries, want 2", len(got.AllCubeAnalyses))
	}
	for _, ca := range got.AllCubeAnalyses {
		if ca.AnalysisEngine == "XG" && ca.BestCubeAction != "Too good to double, pass" {
			t.Errorf("XG's entry was not updated: %q", ca.BestCubeAction)
		}
	}
}

// One engine carries no set: an array of one would be noise in every blob that
// held it, and AllCubeAnalyses is read as "several" everywhere.
func TestSaveAnalysisLeavesASingleEngineWithoutASet(t *testing.T) {
	db := newTestDB(t)
	pos := InitializePosition()
	pos.DecisionType = CubeAction
	id, err := db.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}
	if err := db.SaveAnalysis(id, PositionAnalysis{
		AnalysisType:         "DoublingCube",
		DoublingCubeAnalysis: &DoublingCubeAnalysis{AnalysisEngine: "XG", BestCubeAction: "No Double"},
	}); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}
	got, err := db.LoadAnalysis(id)
	if err != nil {
		t.Fatalf("LoadAnalysis: %v", err)
	}
	if len(got.AllCubeAnalyses) != 0 {
		t.Errorf("a single engine produced a set of %d", len(got.AllCubeAnalyses))
	}
	if got.DoublingCubeAnalysis == nil {
		t.Fatal("the only cube analysis was lost")
	}
}
