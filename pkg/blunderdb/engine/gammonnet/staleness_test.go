package gammonnet

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestIsStaleAnalysisNilIsNotStale: a position with no analysis at all is
// not this predicate's concern (it belongs to the gap-fill sweep, not the
// stale one) — nil must never be reported stale.
func TestIsStaleAnalysisNilIsNotStale(t *testing.T) {
	if IsStaleAnalysis(nil, "2-ply") {
		t.Error("IsStaleAnalysis(nil, ...) = true, want false")
	}
}

// TestIsStaleAnalysisCurrentIsNotStale: an entry at the running EngineVersion
// and the target depth is up to date.
func TestIsStaleAnalysisCurrentIsNotStale(t *testing.T) {
	a := &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			AnalysisEngine: EngineVersion,
			AnalysisDepth:  "2-ply",
		},
	}
	if IsStaleAnalysis(a, "2-ply") {
		t.Error("a current gammonNet entry at the target depth was reported stale")
	}
}

// TestIsStaleAnalysisOlderEngineVersionIsStale covers the version half of the
// predicate: an older gammonNet tag at the SAME depth is stale.
func TestIsStaleAnalysisOlderEngineVersionIsStale(t *testing.T) {
	a := &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			AnalysisEngine: "gammonNet v1.0.1",
			AnalysisDepth:  "2-ply",
		},
	}
	if !IsStaleAnalysis(a, "2-ply") {
		t.Error("an older EngineVersion at the target depth was not reported stale")
	}
}

// TestIsStaleAnalysisDifferentDepthIsStale: raising the canonical depth must
// mark a same-EngineVersion row stale on the depth mismatch alone.
func TestIsStaleAnalysisDifferentDepthIsStale(t *testing.T) {
	a := &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			AnalysisEngine: EngineVersion,
			AnalysisDepth:  "0-ply",
		},
	}
	if !IsStaleAnalysis(a, "2-ply") {
		t.Error("a same-EngineVersion entry at a different depth than targetDepth was not reported stale")
	}
	if IsStaleAnalysis(a, "0-ply") {
		t.Error("a same-EngineVersion entry at the SAME depth as targetDepth was wrongly reported stale")
	}
}

// TestIsStaleAnalysisForeignEngineIsNeverStale: a position carrying any
// non-gammonNet entry is never stale, whatever its gammonNet entries say
// (ADR-0013).
func TestIsStaleAnalysisForeignEngineIsNeverStale(t *testing.T) {
	a := &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
			AnalysisEngine: "XG",
			AnalysisDepth:  "4-ply",
		},
	}
	if IsStaleAnalysis(a, "2-ply") {
		t.Error("an XG-analysed position was reported stale by the gammonNet staleness predicate")
	}
}

// TestIsStaleAnalysisMixedCheckerMoveEntriesAreForeignProtected: a
// CheckerAnalysis with a mix of gammonNet and imported moves (a position
// analysed once by XG and once, on a different move, by gammonNet) must
// also fall under the foreign-engine exception — "allOurs" requires EVERY
// entry to be gammonNet's own.
func TestIsStaleAnalysisMixedCheckerMoveEntriesAreForeignProtected(t *testing.T) {
	a := &domain.PositionAnalysis{
		CheckerAnalysis: &domain.CheckerAnalysis{
			Moves: []domain.CheckerMove{
				{AnalysisEngine: "gammonNet v1.0.1", AnalysisDepth: "0-ply"},
				{AnalysisEngine: "XG", AnalysisDepth: "4-ply"},
			},
		},
	}
	if IsStaleAnalysis(a, "2-ply") {
		t.Error("a mix of gammonNet and XG checker-move entries was reported stale")
	}
}

// TestIsStaleAnalysisV170IsStaleUnderV181 pins the adoption of gammonNet
// v1.8.1 and its float16 weights: every analysis labelled v1.7.0 is stale at
// every depth, although no chosen move differs in the gold corpora, while a
// v1.8.1 entry at the target depth is current.
func TestIsStaleAnalysisV170IsStaleUnderV181(t *testing.T) {
	if EngineVersion != "gammonNet v1.8.1" {
		t.Fatalf("EngineVersion = %q, want gammonNet v1.8.1", EngineVersion)
	}
	for _, depth := range []string{"0-ply", "1-ply", "2-ply", "3-ply", "4-ply"} {
		old := &domain.PositionAnalysis{
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisEngine: "gammonNet v1.7.0", AnalysisDepth: depth},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{AnalysisEngine: "gammonNet v1.7.0", AnalysisDepth: depth},
			}},
		}
		if !IsStaleAnalysis(old, depth) {
			t.Errorf("a gammonNet v1.7.0 analysis at %s was not reported stale", depth)
		}
		current := &domain.PositionAnalysis{
			DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{AnalysisEngine: "gammonNet v1.8.1", AnalysisDepth: depth},
			CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{
				{AnalysisEngine: "gammonNet v1.8.1", AnalysisDepth: depth},
			}},
		}
		if IsStaleAnalysis(current, depth) {
			t.Errorf("a gammonNet v1.8.1 analysis at %s was reported stale", depth)
		}
	}
}
