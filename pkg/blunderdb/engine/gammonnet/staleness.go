package gammonnet

import (
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// engineVersionPrefix identifies an AnalysisEngine string as gammonNet's
// own, whatever tag it names — "gammonNet v1.2.1" starts with it exactly
// like "gammonNet v1.3.0" will.
const engineVersionPrefix = "gammonNet "

// IsStaleAnalysis reports whether a's every entry is gammonNet's own and at
// least one entry is either older than the running build's EngineVersion or
// not at targetDepth (the exact string DepthLabel produces for the depth a
// caller is about to re-run at). This is the ONE staleness predicate a
// re-analysis sweep uses — the gammonNet batch on *database.Database
// (db_gammonnet_batch.go) and the serve daemon's
// /v1/gammonnet.sweepStale both call it rather than keeping their own copy
// (#191, CLI/GUI/server parity: the logic lives where it can be shared, not
// forked per mode).
//
// A position that also carries an XG, GNUbg or BGBlitz entry is never
// reported stale here, whatever its gammonNet entries say — ADR-0013
// protects an imported analysis unconditionally, and a re-analysis sweep
// only ever touches this package's own past output (ADR-0016's narrow
// exception).
//
// targetDepth entering the predicate is what makes moving the canonical
// depth up — 0-ply to 2-ply, say — actually mark existing rows stale:
// EngineVersion alone never changes just because the depth a caller asks
// for did, so a depth-only bump used to leave every already-analysed
// position looking perfectly current (#191).
func IsStaleAnalysis(a *domain.PositionAnalysis, targetDepth string) bool {
	if a == nil {
		return false
	}
	allOurs, anyStale, sawAny := true, false, false
	check := func(engine, depth string) {
		sawAny = true
		if !strings.HasPrefix(engine, engineVersionPrefix) {
			allOurs = false
			return
		}
		if engine != EngineVersion || depth != targetDepth {
			anyStale = true
		}
	}
	if a.CheckerAnalysis != nil {
		for _, m := range a.CheckerAnalysis.Moves {
			check(m.AnalysisEngine, m.AnalysisDepth)
		}
	}
	if a.DoublingCubeAnalysis != nil {
		check(a.DoublingCubeAnalysis.AnalysisEngine, a.DoublingCubeAnalysis.AnalysisDepth)
	}
	return sawAny && allOurs && anyStale
}
