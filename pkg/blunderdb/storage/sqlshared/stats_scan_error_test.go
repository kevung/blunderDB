package sqlshared

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestComputeCorruptedRowReturnsError: stats with a corrupted row give an
// error, not a false PR. Each subtest fails the single row of one of
// Compute's Query calls, which must yield a non-nil error and a nil result —
// never a partial StatsResult.
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
// s.DB.QueryRow(...).Scan(...)` sites (Snowie ER global): a query failure
// must surface, not leave a zero value. QueryRow call 1 is the Totals
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

// TestMatchBadgesCorruptedRowReturnsError: a scan error must be reported,
// never silently drop one match's badge.
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
