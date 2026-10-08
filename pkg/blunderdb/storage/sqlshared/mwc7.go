package sqlshared

// mwc7.go — the 7-point MWC loss (ADR-0075) of the stats reads. Every
// reader sums the MWC losses decisionMWCLoss already gives, per (match, seat)
// and, for one match, per game; the formula and the pooling are the domain's.

import (
	"cmp"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// gameLosses accumulates one match's MWC losses per seat and game. Every game
// the match's counted decisions reach is a unit of the resampling, for both
// seats: a game in which one player lost nothing is a game of zero loss.
type gameLosses struct {
	games  map[int64]struct{}
	bySeat [2]map[int64]float64
	total  [2]float64
	// priced says the seat has at least one decision the conversion values:
	// a seat whose every loss is NaN has no MWC7, as the statistics, which
	// skip it, also say.
	priced [2]bool
}

func newGameLosses() *gameLosses {
	return &gameLosses{games: map[int64]struct{}{}, bySeat: [2]map[int64]float64{{}, {}}}
}

// add records a decision of seat (0 for player 1) in game; a NaN loss (no
// value at that score) records the game only.
func (g *gameLosses) add(seat int, game int64, loss float64) {
	g.games[game] = struct{}{}
	if loss != loss {
		return
	}
	g.bySeat[seat][game] += loss
	g.total[seat] += loss
	g.priced[seat] = true
}

// elo is seat's MWC7 over a match of matchLength points.
func (g *gameLosses) elo(seat, matchLength int) domain.MWC7 {
	if !g.priced[seat] {
		return domain.MWC7{}
	}
	ids := make([]int64, 0, len(g.games))
	for id := range g.games {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	losses := make([]float64, len(ids))
	for i, id := range ids {
		losses[i] = g.bySeat[seat][id]
	}
	return domain.MatchMWC7(g.total[seat], matchLength, losses)
}

// eloUnit is one player's (one seat's) MWC loss over one match.
type eloUnit struct {
	match, tournament int64
	seat, length      int
	loss              float64
	// byType splits loss into checker (0) and cube (1) decisions.
	byType [2]float64
}

type eloUnitKey struct {
	match int64
	seat  int
}

// eloUnits gathers the units of a stats selection as its decisions or cells
// arrive.
type eloUnits map[eloUnitKey]*eloUnit

func (u eloUnits) add(match, tournament int64, seat, matchLength, decisionType int, loss float64) {
	k := eloUnitKey{match, seat}
	e := u[k]
	if e == nil {
		e = &eloUnit{match: match, tournament: tournament, seat: seat, length: matchLength}
		u[k] = e
	}
	e.loss += loss
	if decisionType == 0 {
		e.byType[0] += loss
	} else {
		e.byType[1] += loss
	}
}

// fill pools the units into the selection's MWC7 and back-fills the
// per-tournament and per-match rows, in key order so the sums are the same
// on every run.
func (u eloUnits) fill(result *storage.StatsResult) {
	keys := make([]eloUnitKey, 0, len(u))
	for k := range u {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b eloUnitKey) int {
		return cmp.Or(cmp.Compare(a.match, b.match), cmp.Compare(a.seat, b.seat))
	})
	var global domain.MWC7Pool
	var byType [2]domain.MWC7Pool
	byMatch := map[int64]*domain.MWC7Pool{}
	byTournament := map[int64]*domain.MWC7Pool{}
	pool := func(m map[int64]*domain.MWC7Pool, id int64) *domain.MWC7Pool {
		p := m[id]
		if p == nil {
			p = &domain.MWC7Pool{}
			m[id] = p
		}
		return p
	}
	// The seats of one match are one unit of the resampling (ADR-0078): they
	// play the same positions. Keys are sorted by match, so a match's seats
	// are consecutive.
	for i := 0; i < len(keys); {
		first := u[keys[i]]
		var loss float64
		var byTypeLoss [2]float64
		n := 0
		for ; i < len(keys) && keys[i].match == first.match; i++ {
			e := u[keys[i]]
			loss += e.loss
			byTypeLoss[0] += e.byType[0]
			byTypeLoss[1] += e.byType[1]
			n++
		}
		global.AddMatch(loss, n, first.length)
		byType[0].AddMatch(byTypeLoss[0], n, first.length)
		byType[1].AddMatch(byTypeLoss[1], n, first.length)
		pool(byMatch, first.match).AddMatch(loss, n, first.length)
		if first.tournament != 0 {
			pool(byTournament, first.tournament).AddMatch(loss, n, first.length)
		}
	}
	result.MWC7 = global.Result()
	result.MWC7Checker = byType[0].Result()
	result.MWC7Cube = byType[1].Result()
	for i, ms := range result.PerMatch {
		if p := byMatch[ms.ID]; p != nil {
			result.PerMatch[i].MWC7 = p.Result()
		}
	}
	for i, ts := range result.PerTournament {
		if p := byTournament[ts.ID]; p != nil {
			result.PerTournament[i].MWC7 = p.Result()
		}
	}
}
