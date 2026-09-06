package domain

import "testing"

func TestAnalysisDepthRank_OrdersPliesNumerically(t *testing.T) {
	// The bug this guards: "2-ply" >= "10-ply" is true as strings.
	if AnalysisDepthRank("2-ply") >= AnalysisDepthRank("10-ply") {
		t.Fatal("2-ply must rank below 10-ply")
	}
	if AnalysisDepthRank("0-ply") >= AnalysisDepthRank("1-ply") {
		t.Fatal("0-ply must rank below 1-ply")
	}
}

func TestAnalysisDepthRank_XGCodesAboveEveryPly(t *testing.T) {
	deepest := AnalysisDepthRank("10-ply")
	for _, label := range []string{"Book", "XG Roller", "XG Roller++", "Rollout", "rollout 1296"} {
		if AnalysisDepthRank(label) <= deepest {
			t.Errorf("%q must rank above 10-ply", label)
		}
	}
	if AnalysisDepthRank("XG Roller++") <= AnalysisDepthRank("XG Roller") {
		t.Error("XG Roller++ must rank above XG Roller")
	}
	if AnalysisDepthRank("XG Roller") <= AnalysisDepthRank("Book") {
		t.Error("XG Roller must rank above Book")
	}
}

func TestAnalysisDepthRank_UnknownLabelsLoseToAnyKnownOne(t *testing.T) {
	for _, label := range []string{"", "  ", "n/a", "-ply", "x-ply"} {
		if got := AnalysisDepthRank(label); got != -1 {
			t.Errorf("AnalysisDepthRank(%q) = %d, want -1", label, got)
		}
	}
	if AnalysisDepthRank("") >= AnalysisDepthRank("0-ply") {
		t.Error("an unlabelled analysis must lose to 0-ply")
	}
	// A bare gnuBG integer is a ply count the parser could not name.
	if AnalysisDepthRank("3") != 3 {
		t.Errorf("bare integer: got %d, want 3", AnalysisDepthRank("3"))
	}
}
