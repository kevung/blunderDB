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

	// Three matches whose seat totals are equal: the match units agree, so
	// there is no band, whereas the seats would spread.
	u = eloUnits{}
	for m := int64(1); m <= 3; m++ {
		u.add(m, 0, 0, 7, 0, 0.05*float64(m))
		u.add(m, 0, 1, 7, 0, 0.3-0.05*float64(m))
	}
	res = storage.StatsResult{}
	u.fill(&res)
	e := res.MWC7
	if e.HasInterval || e.Matches != 6 || math.Abs(e.Loss-0.15) > 1e-12 {
		t.Errorf("three matches of equal totals: %+v", e)
	}
}
