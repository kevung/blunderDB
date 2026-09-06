package domain

import (
	"strconv"
	"strings"
)

// AnalysisDepthRank orders the free-text depth labels the importers write
// into CheckerMove.AnalysisDepth and CubeDecision.AnalysisDepth, so that
// "the deeper analysis wins" on a merge is decided numerically. The labels
// come from three sources with one shape each: "N-ply" (XG, gnuBG, BGF and
// gammonNet), "Book" / "XG Roller" / "XG Roller++" (XG's codes 998–1002,
// see ingest.translateAnalysisDepth), and a bare integer for a gnuBG depth
// the parser did not recognise. Comparing the strings themselves put
// "2-ply" above "10-ply" (tasks/critique-doc-2026-09, lot 2), which is what
// this function replaces.
//
// The rank is monotonic in strength: plies first, then XG's book and
// rollouts (which XG itself numbers above every ply), then an explicit
// rollout label. An empty or unknown label ranks lowest, so a labelled
// analysis always wins over an unlabelled one and two unknown labels tie —
// and on a tie the caller keeps its own preference (the incoming move).
func AnalysisDepthRank(label string) int {
	s := strings.TrimSpace(label)
	if s == "" {
		return -1
	}
	if n, ok := strings.CutSuffix(s, "-ply"); ok {
		if v, err := strconv.Atoi(n); err == nil && v >= 0 {
			return v
		}
		return -1
	}
	switch strings.ToLower(s) {
	case "book":
		return 100
	case "xg roller":
		return 101
	case "xg roller++":
		return 102
	}
	if strings.Contains(strings.ToLower(s), "rollout") {
		return 200
	}
	if v, err := strconv.Atoi(s); err == nil && v >= 0 {
		return v
	}
	return -1
}
