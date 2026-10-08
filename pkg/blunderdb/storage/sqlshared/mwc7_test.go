package sqlshared

import (
	"math"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The two seats of a match play the same positions: without a player filter
// they are one unit of the resampling, so a single match has no interval
// however its seats differ, and the pooled tournament neither.
func TestEloUnitsResampleMatchesNotSeats(t *testing.T) {
	u := eloUnits{}
	u.add(1, 9, 0, 7, 0, 0.15)
	u.add(1, 9, 1, 7, 0, 0.15)
	res := storage.StatsResult{PerTournament: []storage.TournamentStats{{ID: 9}}}
	u.fill(&res)
	if !res.MWC7.Available || res.MWC7.HasInterval || res.MWC7.Matches != 2 {
		t.Errorf("one match, two seats: %+v", res.MWC7)
	}
	if got := res.PerTournament[0].MWC7; got.HasInterval {
		t.Errorf("tournament of one match: %+v", got)
	}

	// Two matches whose seat totals are equal: the match units agree, so the
	// interval collapses on the value, whereas the seats would spread.
	u = eloUnits{}
	u.add(1, 0, 0, 7, 0, 0.05)
	u.add(1, 0, 1, 7, 0, 0.25)
	u.add(2, 0, 0, 7, 0, 0.25)
	u.add(2, 0, 1, 7, 0, 0.05)
	res = storage.StatsResult{}
	u.fill(&res)
	e := res.MWC7
	if !e.HasInterval || math.Abs(e.High-e.Low) > 1e-12 || math.Abs(e.Loss-0.15) > 1e-12 {
		t.Errorf("two matches of equal totals: %+v", e)
	}
}
