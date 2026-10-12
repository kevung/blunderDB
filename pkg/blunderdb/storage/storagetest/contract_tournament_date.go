// Contract cases for a tournament's effective date: the entered date, else
// the day of its earliest match, derived on read (ADR-0007).
// The table that runs them lives in contract.go.
package storagetest

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

var isoDay = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// datedMatchIn saves a match played on day and attaches it to tournament tID.
func datedMatchIn(t *testing.T, s storage.Storage, tID int64, day time.Time) {
	t.Helper()
	ctx := context.Background()
	m := domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 5, MatchDate: day}
	id, err := s.Matches().Save(ctx, "", &m)
	if err != nil {
		t.Fatalf("Save match: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", tID, id); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}
}

func testTournamentEffectiveDate(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	create := func(name, date string) int64 {
		t.Helper()
		id, err := s.Tournaments().Create(ctx, "", name, date, "")
		if err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
		return id
	}
	// Created in the reverse of the expected order, so the order cannot
	// come from creation.
	empty := create("Empty", "")
	derived := create("Derived", "")
	dated := create("Dated", "2025-03-01")
	datedMatchIn(t, s, derived, time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC))
	datedMatchIn(t, s, derived, time.Date(2024, 5, 10, 0, 0, 0, 0, time.UTC))
	// An entered date wins over an earlier match.
	datedMatchIn(t, s, dated, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))

	want := map[int64][2]string{ // id → {Date, EffectiveDate}
		dated:   {"2025-03-01", "2025-03-01"},
		derived: {"", "2024-05-10"},
		empty:   {"", ""},
	}
	for id, w := range want {
		got, err := s.Tournaments().Get(ctx, "", id)
		if err != nil {
			t.Fatalf("Get %d: %v", id, err)
		}
		if got.Date != w[0] || got.EffectiveDate != w[1] {
			t.Errorf("Get %s: Date %q EffectiveDate %q, want %q %q", got.Name, got.Date, got.EffectiveDate, w[0], w[1])
		}
	}
	var order []int64
	for tour, err := range s.Tournaments().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if w, ok := want[tour.ID]; ok {
			if tour.Date != w[0] || tour.EffectiveDate != w[1] {
				t.Errorf("List %s: Date %q EffectiveDate %q, want %q %q", tour.Name, tour.Date, tour.EffectiveDate, w[0], w[1])
			}
			order = append(order, tour.ID)
		}
	}
	if len(order) != 3 || order[0] != dated || order[1] != derived || order[2] != empty {
		t.Errorf("List order %v, want most recent effective date first, undated last: %v", order, []int64{dated, derived, empty})
	}
	// Reading wrote nothing back.
	if got, _ := s.Tournaments().Get(ctx, "", derived); got != nil && got.Date != "" {
		t.Errorf("derived date stored: %q", got.Date)
	}
}

// testStatsProgressionDates pins the Progression series: every PerMatch date
// is a bare day, whichever path computes it, and PerTournament runs in the
// chronological order of the effective dates.
func testStatsProgressionDates(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	matchA, _ := statsFixtureMatch(t, s, 0, "Alice", "Bob")
	matchB, _ := statsFixtureMatch(t, s, 2, "Alice", "Carol")
	// The undated tournament is created first: its fixture match is dated
	// later than the dated tournament's, so it still sorts after it.
	undated, err := s.Tournaments().Create(ctx, "", "Undated", "", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	dated, err := s.Tournaments().Create(ctx, "", "Dated", "2025-01-01", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", undated, matchA); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}
	if err := s.Tournaments().AddMatch(ctx, "", dated, matchB); err != nil {
		t.Fatalf("AddMatch: %v", err)
	}

	for _, f := range []storage.StatsFilter{
		{DecisionType: -1},
		{DecisionType: 0, PlayerName: "Alice"},
		{DecisionType: -1, MinAnalysisDepth: 2},
		{DecisionType: -1, PlayerName: "Alice", AnalysisEngine: "XG"},
	} {
		res, err := s.Stats().Compute(ctx, "", f)
		if err != nil {
			t.Fatalf("Compute %+v: %v", f, err)
		}
		series, err := s.Stats().MatchSeries(ctx, "", f)
		if err != nil {
			t.Fatalf("MatchSeries %+v: %v", f, err)
		}
		for _, m := range append(res.PerMatch, series...) {
			if !isoDay.MatchString(m.Date) {
				t.Errorf("filter %+v: match %d date %q, want YYYY-MM-DD", f, m.ID, m.Date)
			}
		}
		if len(res.PerTournament) == 0 {
			continue
		}
		var got []string
		for _, ts := range res.PerTournament {
			got = append(got, ts.Date+" "+ts.Name)
		}
		if len(got) != 2 || got[0] != "2025-01-01 Dated" || got[1] != "2025-06-01 Undated" {
			t.Errorf("filter %+v: PerTournament %v, want [2025-01-01 Dated 2025-06-01 Undated]", f, got)
		}
	}
}
