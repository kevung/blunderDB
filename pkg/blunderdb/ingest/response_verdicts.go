package ingest

import (
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// responseVerdictsRightFrom is the first schema version whose gammonNet
// verdicts on take/pass positions are the doubler's decision (ADR-0083).
var responseVerdictsRightFrom = [3]int{2, 41, 0}

// StaleResponseVerdict reports whether a, the analysis of pos read from a
// source database at schema sourceVersion, is a gammonNet verdict on a
// take/pass position that scored the answerer's centred-cube decision. Such
// a verdict judges the wrong decision and its label does not tell it from a
// right one, so an import leaves it behind, as opening the source would drop
// it; the next analysis gives the doubler's. A version that does not parse
// is taken as current: nothing is dropped on a guess.
func StaleResponseVerdict(sourceVersion string, pos *domain.Position, a *domain.PositionAnalysis) bool {
	if a == nil || !domain.IsResponsePosition(pos) || !versionBefore(sourceVersion, responseVerdictsRightFrom) {
		return false
	}
	label, _, _ := engine.AnalysisProvenance(a)
	return strings.HasPrefix(label, "gammonNet")
}

// versionBefore reports whether the schema version v ("MAJOR.MINOR.PATCH")
// is older than ref; false when v does not parse.
func versionBefore(v string, ref [3]int) bool {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return false
		}
		if n != ref[i] {
			return n < ref[i]
		}
	}
	return false
}
