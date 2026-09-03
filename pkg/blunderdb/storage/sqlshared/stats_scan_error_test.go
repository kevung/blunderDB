package sqlshared

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestComputeCorruptedRowReturnsError is B.6's (#174) acceptance test on
// StatsStore.Compute: "stats with a corrupted row → an error, not a false
// PR." Compute runs nine Query calls in sequence (PR by decision_type, PR
// per tournament, PR per match, cube action breakdown, cube direction
// matrix, error histogram, top blunders, rolling PR, the MWC pass); before
// B.6 every one of them wrapped its row loop in an immediately-invoked
// closure whose `return` on a Scan error only exited the closure, and the
// surrounding code went on to read rows.Err() — which a Scan error never
// sets — and returned a result computed from whatever rows were scanned
// before the bad one, with a nil error. A caller had no way to tell that
// result apart from a database that genuinely only had that many rows.
//
// Each subtest fails exactly one of the nine Query calls' single row, and
// every one of them must now make Compute return a non-nil error and a nil
// result — never a partial StatsResult.
func TestComputeCorruptedRowReturnsError(t *testing.T) {
	labels := []string{
		"PR by decision_type",
		"PR per tournament",
		"PR per match",
		"cube action breakdown",
		"cube direction matrix",
		"error histogram",
		"top blunders",
		"rolling PR",
		"MWC pass",
	}

	for i, label := range labels {
		callToFail := i + 1
		t.Run(label, func(t *testing.T) {
			f := &fakeExecer{queryCallToFail: callToFail}
			store := &StatsStore{DB: f}

			result, err := store.Compute(context.Background(), "t", storage.StatsFilter{DecisionType: -1})
			if err == nil {
				t.Fatalf("Compute with a corrupted row in %q query = (result=%+v, nil error); want a non-nil error", label, result)
			}
			if result != nil {
				t.Errorf("Compute returned a non-nil result alongside its error: %+v", result)
			}
		})
	}
}

// TestComputeSnowieGlobalQueryFailureReturnsError covers the two `_ =
// s.DB.QueryRow(...).Scan(...)` sites (stats.go, Snowie ER global): a query
// failure there used to be discarded outright (the `_ =`), leaving
// SnowieGlobal computed from whatever zero value the Go variable already
// held instead of surfacing the failure. QueryRow call 1 is the Totals
// query (which every subtest needs to succeed to get this far); calls 2 and
// 3 are the Snowie numerator and denominator.
func TestComputeSnowieGlobalQueryFailureReturnsError(t *testing.T) {
	for _, call := range []int{2, 3} {
		f := &fakeExecer{queryRowCallToFail: call}
		store := &StatsStore{DB: f}
		result, err := store.Compute(context.Background(), "t", storage.StatsFilter{DecisionType: -1})
		if err == nil {
			t.Errorf("Compute with Snowie global QueryRow call %d failing = (result=%+v, nil error); want a non-nil error", call, result)
		}
		if result != nil {
			t.Errorf("Compute returned a non-nil result alongside its error (call %d): %+v", call, result)
		}
	}
}

// TestMatchDetailCorruptedRowReturnsError covers the two vulnerable sites in
// MatchDetail: the main per-decision loop (a plain `continue` on Scan error,
// not even inside a closure) and the Snowie ER sub-pass's closure.
func TestMatchDetailCorruptedRowReturnsError(t *testing.T) {
	cases := []struct {
		name string
		call int
	}{
		{"main per-decision loop", 1},
		{"Snowie ER sub-pass", 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &fakeExecer{queryCallToFail: c.call}
			store := &StatsStore{DB: f}
			result, err := store.MatchDetail(context.Background(), "t", 1)
			if err == nil {
				t.Fatalf("MatchDetail with a corrupted row in %q = (result=%+v, nil error); want a non-nil error", c.name, result)
			}
			if result != nil {
				t.Errorf("MatchDetail returned a non-nil result alongside its error: %+v", result)
			}
		})
	}
}

// TestMatchBadgesCorruptedRowReturnsError: the scan error used to be a bare
// `continue`, silently dropping one match's badge rather than reporting a
// database that could not be trusted.
func TestMatchBadgesCorruptedRowReturnsError(t *testing.T) {
	f := &fakeExecer{queryCallToFail: 1}
	store := &StatsStore{DB: f}
	result, err := store.MatchBadges(context.Background(), "t", nil)
	if err == nil {
		t.Fatalf("MatchBadges with a corrupted row = (result=%+v, nil error); want a non-nil error", result)
	}
	if result != nil {
		t.Errorf("MatchBadges returned a non-nil result alongside its error: %+v", result)
	}
}

// TestTournamentBadgesCorruptedRowReturnsError mirrors MatchBadges.
func TestTournamentBadgesCorruptedRowReturnsError(t *testing.T) {
	f := &fakeExecer{queryCallToFail: 1}
	store := &StatsStore{DB: f}
	result, err := store.TournamentBadges(context.Background(), "t")
	if err == nil {
		t.Fatalf("TournamentBadges with a corrupted row = (result=%+v, nil error); want a non-nil error", result)
	}
	if result != nil {
		t.Errorf("TournamentBadges returned a non-nil result alongside its error: %+v", result)
	}
}

// TestPlayerTableCorruptedRowReturnsError covers PlayerTable's four Query
// calls (decisions, Snowie numerator, luck, matches), each previously a
// closure that turned a Scan error into a silent `continue`.
func TestPlayerTableCorruptedRowReturnsError(t *testing.T) {
	labels := []string{"decisions", "snowie", "luck", "matches"}
	for i, label := range labels {
		callToFail := i + 1
		t.Run(label, func(t *testing.T) {
			f := &fakeExecer{queryCallToFail: callToFail}
			store := &StatsStore{DB: f}
			result, err := store.PlayerTable(context.Background(), "t", storage.StatsFilter{DecisionType: -1})
			if err == nil {
				t.Fatalf("PlayerTable with a corrupted row in %q = (result=%+v, nil error); want a non-nil error", label, result)
			}
			if result != nil {
				t.Errorf("PlayerTable returned a non-nil result alongside its error: %+v", result)
			}
		})
	}
}
