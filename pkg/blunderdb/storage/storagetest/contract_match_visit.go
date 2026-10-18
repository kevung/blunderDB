package storagetest

import (
	"context"
	"testing"
	"time"

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
