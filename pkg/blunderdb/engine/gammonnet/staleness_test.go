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

// TestIsStaleAnalysisDifferentDepthIsStale is #191's actual fix: raising the
// canonical depth (0-ply to 2-ply, say) must mark a same-EngineVersion row
// stale on the depth mismatch alone — before this, EngineVersion never
// changed just because the target depth did, so a depth-only bump left
// every already-analysed position looking perfectly current.
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

// TestIsStaleAnalysisForeignEngineIsNeverStale is ADR-0016's narrow
// exception to ADR-0013: a position that carries any non-gammonNet entry
// (XG, GNUbg, BGBlitz) is never reported stale by this predicate, whatever
// its gammonNet entries look like — an imported analysis is protected
// unconditionally, and a re-analysis sweep must never touch it.
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
