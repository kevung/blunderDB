package database

import "testing"

func TestLoadPositionView_MatchesSeparateCalls(t *testing.T) {
	t.Parallel()
	db := trashDB(t)

	pos := InitializePosition()
	pos.Dice = [2]int{6, 5}
	id, err := db.SavePosition(&pos)
	if err != nil {
		t.Fatalf("SavePosition: %v", err)
	}

	view, err := db.LoadPositionView(id)
	if err != nil {
		t.Fatalf("LoadPositionView (bare): %v", err)
	}
	if view.Analysis != nil || view.Comment != "" {
		t.Fatalf("bare position: got %+v", view)
	}

	if err := db.SaveAnalysis(id, PositionAnalysis{AnalysisType: "CheckerMove"}); err != nil {
		t.Fatalf("SaveAnalysis: %v", err)
	}
	if err := db.SaveComment(id, "à revoir"); err != nil {
		t.Fatalf("SaveComment: %v", err)
	}
	view, err = db.LoadPositionView(id)
	if err != nil {
		t.Fatalf("LoadPositionView: %v", err)
	}
	want, _ := db.LoadComment(id)
	if view.Comment != want {
		t.Errorf("comment = %q, LoadComment = %q", view.Comment, want)
	}
	if view.Analysis == nil || view.Analysis.AnalysisType != "CheckerMove" {
		t.Errorf("analysis = %+v", view.Analysis)
	}
}
