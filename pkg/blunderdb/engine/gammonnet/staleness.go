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
// not at targetDepth (the exact DepthLabel string, so a depth change alone
// marks rows stale). It is the one staleness predicate every re-analysis
// sweep uses (db_gammonnet_batch.go, /v1/gammonnet.sweepStale).
//
// A position that also carries an XG, GNUbg or BGBlitz entry is never stale:
// ADR-0013 protects an imported analysis unconditionally.
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

// IsOurAnalysis reports whether every entry of a was written by gammonNet,
// whatever version — IsStaleAnalysis's first half, which the comparison sweep
// also needs.
//
// An analysis with no entry at all is nobody's and answers false.
func IsOurAnalysis(a *domain.PositionAnalysis) bool {
	if a == nil {
		return false
	}
	ours, sawAny := true, false
	check := func(engine string) {
		sawAny = true
		if !strings.HasPrefix(engine, engineVersionPrefix) {
			ours = false
		}
	}
	if a.CheckerAnalysis != nil {
		for _, m := range a.CheckerAnalysis.Moves {
			check(m.AnalysisEngine)
		}
	}
	if a.DoublingCubeAnalysis != nil {
		check(a.DoublingCubeAnalysis.AnalysisEngine)
	}
	return sawAny && ours
}
