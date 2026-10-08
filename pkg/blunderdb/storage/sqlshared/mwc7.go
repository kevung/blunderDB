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
}

// elo is seat's MWC7 over a match of matchLength points.
func (g *gameLosses) elo(seat, matchLength int) domain.MWC7 {
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
}

type eloUnitKey struct {
	match int64
	seat  int
}

// eloUnits gathers the units of a stats selection as its decisions or cells
// arrive.
type eloUnits map[eloUnitKey]*eloUnit

func (u eloUnits) add(match, tournament int64, seat, matchLength int, loss float64) {
	k := eloUnitKey{match, seat}
	e := u[k]
	if e == nil {
		e = &eloUnit{match: match, tournament: tournament, seat: seat, length: matchLength}
		u[k] = e
	}
	e.loss += loss
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
	for _, k := range keys {
		e := u[k]
		global.AddLoss(e.loss, e.length)
		pool(byMatch, e.match).AddLoss(e.loss, e.length)
		if e.tournament != 0 {
			pool(byTournament, e.tournament).AddLoss(e.loss, e.length)
		}
	}
	result.MWC7 = global.Result()
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
