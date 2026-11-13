package database

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/reach/reachtest"
)

// TestRankCountsThePlaysOfARealMatch holds the ranking to the oracle on the
// plies of a real match: each checker play's position and the next one in
// its game, then the one after, ranked through the Database the three modes
// share.
func TestRankCountsThePlaysOfARealMatch(t *testing.T) {
	if testing.Short() {
		t.Skip("the oracle enumerates every play of every roll")
	}
	db := newTestDBWithXG(t)
	matches, err := db.GetAllMatches()
	if err != nil || len(matches) == 0 {
		t.Fatalf("GetAllMatches: %d, %v", len(matches), err)
	}
	moves, err := db.GetMatchMovePositions(matches[0].ID)
	if err != nil {
		t.Fatalf("GetMatchMovePositions: %v", err)
	}
	var plies []domain.MatchMovePosition
	for _, m := range moves {
		if m.MoveType == "checker" {
			plies = append(plies, m)
		}
	}
	ones, checked := 0, 0
	for i := 0; i+2 < len(plies) && checked < 40; i += 3 {
		a := plies[i]
		ranked, err := db.RankPositionsByFilters(domain.SearchFilters{LikeTargetID: a.Position.ID, LikeMaxDistance: 2}, 100000)
		if err != nil {
			t.Fatalf("Rank: %v", err)
		}
		got := map[int64]int{}
		for _, n := range ranked {
			got[n.Position.ID] = n.Distance
		}
		for gap := 1; gap <= 2; gap++ {
			b := plies[i+gap]
			if b.GameID != a.GameID {
				continue
			}
			want, ok := reachtest.Distance(&a.Position, &b.Position, 2)
			if !ok || want > gap {
				t.Fatalf("plies %d and %d of one game are %d/%v plays apart by the oracle, want at most %d", i, i+gap, want, ok, gap)
			}
			d, found := got[b.Position.ID]
			if b.Position.ID == a.Position.ID {
				continue
			}
			if !found || d != want {
				t.Fatalf("ply %d ranked at %d/%v from ply %d, the oracle says %d", i+gap, d, found, i, want)
			}
			if want == 1 {
				ones++
			}
			checked++
		}
	}
	if checked < 20 || ones == 0 {
		t.Fatalf("only %d pairs checked, %d at one play", checked, ones)
	}
}
