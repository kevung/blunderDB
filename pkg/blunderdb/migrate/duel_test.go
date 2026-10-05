package migrate

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestRunCarriesTheDuel: a migration moves the origin of a Match played
// here onto the new Match id, and the Duels in suspense with their seed.
func TestRunCarriesTheDuel(t *testing.T) {
	ctx := context.Background()
	open := func() storage.Storage {
		s, err := sqlite.Open(ctx, ":memory:", nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		return s
	}
	src, dst := open(), open()
	matchID, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 3, MatchHash: "h-duel"})
	if err != nil {
		t.Fatal(err)
	}
	if err := src.Duels().SetOrigin(ctx, "", &storage.MatchOrigin{MatchID: matchID, Start: "xgid", DiceSeed: "ab"}); err != nil {
		t.Fatal(err)
	}
	if _, err := src.Duels().Save(ctx, "", &storage.Duel{FormatVersion: "1", Document: "{}", DiceSeed: "cd"}); err != nil {
		t.Fatal(err)
	}

	rep, err := Run(ctx, src, dst, "", Options{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rep.Duels != 1 {
		t.Errorf("report counts %d duels, want 1", rep.Duels)
	}
	var newID int64
	for m, err := range dst.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err == nil && m.MatchHash == "h-duel" {
			newID = m.ID
		}
	}
	if newID == 0 {
		t.Fatal("the match did not migrate")
	}
	if o, err := dst.Duels().Origin(ctx, "", newID); err != nil || o.Start != "xgid" || o.DiceSeed != "ab" {
		t.Errorf("origin after migration = %+v, %v", o, err)
	}
	var seeds []string
	for d, err := range dst.Duels().List(ctx, "") {
		if err == nil {
			seeds = append(seeds, d.DiceSeed)
		}
	}
	if len(seeds) != 1 || seeds[0] != "cd" {
		t.Errorf("migrated duels' seeds = %v, want [cd]", seeds)
	}
}
