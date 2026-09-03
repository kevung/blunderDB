package sqlshared

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// These are the direct unit tests B.15 (#183) asks for on the table that
// replaced the six near-identical win/gammon/backgammon-rate blocks inside
// find's per-row Go filter — the six had no direct test of their own before
// this refactor, only the integration-level search equivalence suite in
// pkg/blunderdb/database.

func TestRateFilterChecksAllInactiveByDefault(t *testing.T) {
	checks := rateFilterChecks(domain.SearchFilters{})
	if matchesRateFilters(checks, nil) != true {
		t.Error("no filter active: want true even with a nil analysis")
	}
}

func TestMatchesRateFiltersNilAnalysisFailsAnActiveFilter(t *testing.T) {
	checks := rateFilterChecks(domain.SearchFilters{WinRateFilter: "w>0.5"})
	if matchesRateFilters(checks, nil) {
		t.Error("active filter against a nil analysis: want false")
	}
}

func TestMatchesRateFiltersAnalysisWithNeitherSourceFails(t *testing.T) {
	checks := rateFilterChecks(domain.SearchFilters{WinRateFilter: "w>0.5"})
	if matchesRateFilters(checks, &domain.PositionAnalysis{}) {
		t.Error("analysis with no cube analysis and no checker moves: want false")
	}
}

// TestRateFilterChecksReadFromCubeAnalysis pins each of the six checks
// against domain.DoublingCubeAnalysis, the primary source (a cube decision
// has no checker moves).
func TestRateFilterChecksReadFromCubeAnalysis(t *testing.T) {
	ana := &domain.PositionAnalysis{DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{
		PlayerWinChances:          0.60,
		PlayerGammonChances:       0.20,
		PlayerBackgammonChances:   0.02,
		OpponentWinChances:        0.40,
		OpponentGammonChances:     0.10,
		OpponentBackgammonChances: 0.01,
	}}

	tests := []struct {
		name string
		f    domain.SearchFilters
		want bool
	}{
		{"win in range", domain.SearchFilters{WinRateFilter: "w>0.5"}, true},
		{"win out of range", domain.SearchFilters{WinRateFilter: "w>0.7"}, false},
		{"gammon in range", domain.SearchFilters{GammonRateFilter: "g>0.1"}, true},
		{"gammon out of range", domain.SearchFilters{GammonRateFilter: "g>0.3"}, false},
		{"backgammon in range", domain.SearchFilters{BackgammonRateFilter: "b>0.01"}, true},
		{"backgammon out of range", domain.SearchFilters{BackgammonRateFilter: "b>0.05"}, false},
		{"opponent win in range", domain.SearchFilters{Player2WinRateFilter: "W>0.3"}, true},
		{"opponent win out of range", domain.SearchFilters{Player2WinRateFilter: "W>0.5"}, false},
		{"opponent gammon in range", domain.SearchFilters{Player2GammonRateFilter: "G>0.05"}, true},
		{"opponent gammon out of range", domain.SearchFilters{Player2GammonRateFilter: "G>0.2"}, false},
		{"opponent backgammon in range", domain.SearchFilters{Player2BackgammonRateFilter: "B>0.005"}, true},
		{"opponent backgammon out of range", domain.SearchFilters{Player2BackgammonRateFilter: "B>0.02"}, false},
		{"several at once, all satisfied", domain.SearchFilters{WinRateFilter: "w>0.5", GammonRateFilter: "g>0.1"}, true},
		{"several at once, one fails", domain.SearchFilters{WinRateFilter: "w>0.5", GammonRateFilter: "g>0.3"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checks := rateFilterChecks(tc.f)
			if got := matchesRateFilters(checks, ana); got != tc.want {
				t.Errorf("matchesRateFilters(%+v) = %v, want %v", tc.f, got, tc.want)
			}
		})
	}
}

// TestRateFilterChecksFallBackToCheckerMove pins the fallback source: a
// checker decision has no DoublingCubeAnalysis, so each rate reads the best
// (first) checker move instead.
func TestRateFilterChecksFallBackToCheckerMove(t *testing.T) {
	ana := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{
		PlayerWinChance:          0.60,
		PlayerGammonChance:       0.20,
		PlayerBackgammonChance:   0.02,
		OpponentWinChance:        0.40,
		OpponentGammonChance:     0.10,
		OpponentBackgammonChance: 0.01,
	}}}}

	tests := []struct {
		name string
		f    domain.SearchFilters
		want bool
	}{
		{"win", domain.SearchFilters{WinRateFilter: "w>0.5"}, true},
		{"gammon", domain.SearchFilters{GammonRateFilter: "g>0.1"}, true},
		{"backgammon", domain.SearchFilters{BackgammonRateFilter: "b>0.01"}, true},
		{"opponent win", domain.SearchFilters{Player2WinRateFilter: "W>0.3"}, true},
		{"opponent gammon", domain.SearchFilters{Player2GammonRateFilter: "G>0.05"}, true},
		{"opponent backgammon", domain.SearchFilters{Player2BackgammonRateFilter: "B>0.005"}, true},
		{"win, out of range", domain.SearchFilters{WinRateFilter: "w>0.7"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checks := rateFilterChecks(tc.f)
			if got := matchesRateFilters(checks, ana); got != tc.want {
				t.Errorf("matchesRateFilters(%+v) = %v, want %v", tc.f, got, tc.want)
			}
		})
	}
}

// TestRateFilterChecksCubeAnalysisWinsOverCheckerMove: a position carrying
// both (uncommon, but AllCubeAnalyses/CheckerAnalysis are independently
// optional) reads the cube analysis, matching the six original blocks'
// if/else-if precedence.
func TestRateFilterChecksCubeAnalysisWinsOverCheckerMove(t *testing.T) {
	ana := &domain.PositionAnalysis{
		DoublingCubeAnalysis: &domain.DoublingCubeAnalysis{PlayerWinChances: 0.9},
		CheckerAnalysis:      &domain.CheckerAnalysis{Moves: []domain.CheckerMove{{PlayerWinChance: 0.1}}},
	}
	checks := rateFilterChecks(domain.SearchFilters{WinRateFilter: "w>0.5"})
	if !matchesRateFilters(checks, ana) {
		t.Error("want the cube analysis's 0.9 to win, satisfying w>0.5")
	}
}

// TestRateFilterChecksEmptyCheckerMovesFails: a CheckerAnalysis with an
// empty Moves slice (e.g. a decoded analysis with nothing to say) is the
// same as having no checker analysis at all.
func TestRateFilterChecksEmptyCheckerMovesFails(t *testing.T) {
	ana := &domain.PositionAnalysis{CheckerAnalysis: &domain.CheckerAnalysis{}}
	checks := rateFilterChecks(domain.SearchFilters{WinRateFilter: "w>0.5"})
	if matchesRateFilters(checks, ana) {
		t.Error("empty Moves slice: want false, nothing to read a rate from")
	}
}
