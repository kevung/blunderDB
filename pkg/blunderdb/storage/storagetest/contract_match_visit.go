package storagetest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testMatchLastVisitedFollowsTheVisit: the last visited match is the one
// visited last, whatever the order the matches were imported in.
func testMatchLastVisitedFollowsTheVisit(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	a, _ := statsFixtureMatch(t, s, 0, "Ann", "Abe")
	b, _ := statsFixtureMatch(t, s, 4, "Bea", "Bob")
	visit := func(id int64) {
		t.Helper()
		if err := s.Matches().SetLastVisitedPosition(ctx, "", id, 1); err != nil {
			t.Fatalf("SetLastVisitedPosition(%d): %v", id, err)
		}
		time.Sleep(3 * time.Millisecond) // the stamp of a visit has millisecond grain on SQLite
	}
	last := func(want int64, why string) {
		t.Helper()
		m, err := s.Matches().LastVisited(ctx, "")
		if err != nil {
			t.Fatalf("LastVisited: %v", err)
		}
		if m.ID != want {
			t.Errorf("%s: last visited = match %d, want %d", why, m.ID, want)
		}
	}
	visit(b)
	visit(a)
	last(a, "B visited, then A")
	visit(b)
	last(b, "A visited, then B again")
}

// testMatchLastVisitedFallbackIsTheListHead: with no match visited,
// LastVisited answers the match List shows first — an undated match by its
// import date, a tie on the date by the higher id — and ErrNotFound on an
// empty scope.
func testMatchLastVisitedFallbackIsTheListHead(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()
	if _, err := ms.LastVisited(ctx, ""); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("LastVisited on an empty scope: %v, want ErrNotFound", err)
	}
	save := func(m domain.Match) int64 {
		t.Helper()
		id, err := ms.Save(ctx, "", &m)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	day := time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)
	save(domain.Match{Player1Name: "Ann", Player2Name: "Abe", MatchLength: 5, MatchDate: day})
	undated := save(domain.Match{Player1Name: "Bea", Player2Name: "Bob", MatchLength: 5})
	head := func(why string) int64 {
		t.Helper()
		m, err := ms.LastVisited(ctx, "")
		if err != nil {
			t.Fatalf("LastVisited: %v", err)
		}
		for first, err := range ms.List(ctx, "", storage.MatchListOpts{Limit: 1}) {
			if err != nil {
				t.Fatal(err)
			}
			if first.ID != m.ID {
				t.Errorf("%s: LastVisited = %d, List head = %d", why, m.ID, first.ID)
			}
		}
		return m.ID
	}
	if got := head("undated, imported now"); got != undated {
		t.Errorf("LastVisited = %d, want the undated match imported now (%d)", got, undated)
	}

	later := day.AddDate(200, 0, 0)
	a := save(domain.Match{Player1Name: "Cy", Player2Name: "Cal", MatchLength: 5, MatchDate: later})
	b := save(domain.Match{Player1Name: "Di", Player2Name: "Dan", MatchLength: 5, MatchDate: later})
	if got := head("tie on the date"); got != max(a, b) {
		t.Errorf("LastVisited = %d, want the higher id of the tie (%d)", got, max(a, b))
	}
}
