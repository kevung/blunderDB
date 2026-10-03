package service

import (
	"context"
	"encoding/csv"
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The season ranking (ADR-0061): the places of several finished tournaments turned into points
// by a configurable scale and summed per person, with an optional club Elo replayed over every
// match they played. It is derived at every call and never stored, like the standings.

// DefaultSeasonPoints is the scale used when a query gives none: 1st place 25 points, then 18,
// 15, 12, 10, 8, 6, 4, 2, 1, nothing beyond. A club states its own with SeasonQuery.Points.
var DefaultSeasonPoints = []float64{25, 18, 15, 12, 10, 8, 6, 4, 2, 1}

// EloStart is the club rating every person begins the season with.
const EloStart = 1500.0

// SeasonQuery selects the tournaments of a season and how they score. A Rencontre, a period, or
// both (the Rencontre's events within the period); neither is every directed tournament.
type SeasonQuery struct {
	RencontreID int64 `json:"rencontreId,omitempty"`
	// From and To bound Tournament.Date, inclusive, as YYYY-MM-DD; empty is unbounded.
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	// Points is the scale by place: Points[0] for the winner. Empty means DefaultSeasonPoints.
	Points []float64 `json:"points,omitempty"`
	// Participation is added for every finished tournament a person is ranked in.
	Participation float64 `json:"participation,omitempty"`
	// Elo replays a club rating over the season's matches (ADR-0061 §3).
	Elo bool `json:"elo,omitempty"`
}

// SeasonEvent is one tournament of the season. An unfinished one is listed and scores nothing:
// its places are not yet facts.
type SeasonEvent struct {
	TournamentID int64  `json:"tournamentId"`
	Name         string `json:"name"`
	Date         string `json:"date"`
	Entrants     int    `json:"entrants"`
	Finished     bool   `json:"finished"`
}

// SeasonRow is one person of the season ranking. Places and Points are aligned on
// SeasonView.Events: 0 is "did not play it".
type SeasonRow struct {
	Rank    int       `json:"rank"`
	Name    string    `json:"name"`
	Club    string    `json:"club,omitempty"`
	Total   float64   `json:"total"`
	Played  int       `json:"played"`
	Best    int       `json:"best"`
	Wins    int       `json:"wins"`
	Losses  int       `json:"losses"`
	Places  []int     `json:"places"`
	Points  []float64 `json:"points"`
	Elo     float64   `json:"elo,omitempty"`
	Matches int       `json:"matches,omitempty"`
}

// SeasonView is the season ranking, with the scale it was computed with.
type SeasonView struct {
	Events        []SeasonEvent `json:"events"`
	Rows          []SeasonRow   `json:"rows"`
	Points        []float64     `json:"points"`
	Participation float64       `json:"participation"`
	Elo           bool          `json:"elo"`
}

// personKey identifies a person across tournaments: a player's id is the tournament's own, so
// the name, folded and trimmed, is what two events share (as for a Rencontre's busy players).
func personKey(name string) string { return strings.ToLower(strings.Join(strings.Fields(name), " ")) }

// placePoints is what a place is worth when `tied` people share it: the mean of the places they
// occupy, so a tie neither creates nor destroys points.
func placePoints(scale []float64, rank, tied int) float64 {
	sum := 0.0
	for p := rank; p < rank+tied; p++ {
		if p-1 < len(scale) {
			sum += scale[p-1]
		}
	}
	return sum / float64(tied)
}

// eloUpdate is the FIBS formula (ADR-0061 §3): the expected score of the higher-rated player
// in an n-point match, and K = 4·√n.
func eloUpdate(winner, loser float64, n int) (float64, float64) {
	if n < 1 {
		n = 1
	}
	sn := math.Sqrt(float64(n))
	pWin := 1 / (1 + math.Pow(10, -(winner-loser)*sn/2000))
	delta := 4 * sn * (1 - pWin)
	return winner + delta, loser - delta
}

// SeasonRanking aggregates the selected tournaments into one ranking.
func (d *Service) SeasonRanking(ctx context.Context, q SeasonQuery) (*SeasonView, error) {
	if q.From != "" && q.To != "" && q.From > q.To {
		return nil, direction.Refusef("season: from %s is after to %s", q.From, q.To)
	}
	scale := q.Points
	if len(scale) == 0 {
		scale = DefaultSeasonPoints
	}
	ids, err := d.seasonTournaments(ctx, q.RencontreID)
	if err != nil {
		return nil, err
	}

	type played struct {
		ev SeasonEvent
		st *tournoi.State
	}
	var events []played
	for _, id := range ids {
		t, err := d.st.Tournaments().Get(ctx, d.scope, id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				continue
			}
			return nil, err
		}
		if (q.From != "" && t.Date < q.From) || (q.To != "" && t.Date > q.To) {
			continue
		}
		dir, err := direction.Open(ctx, d.dirStore(), id)
		if err != nil {
			if errors.Is(err, direction.ErrNoDirection) {
				continue
			}
			return nil, err
		}
		st := dir.State()
		if st == nil {
			continue
		}
		events = append(events, played{
			ev: SeasonEvent{TournamentID: id, Name: t.Name, Date: t.Date, Entrants: len(st.Order), Finished: st.Finished},
			st: st,
		})
	}
	// Chronological: the Elo is replayed in the order the season was played.
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].ev.Date != events[j].ev.Date {
			return events[i].ev.Date < events[j].ev.Date
		}
		return events[i].ev.TournamentID < events[j].ev.TournamentID
	})

	v := &SeasonView{Events: []SeasonEvent{}, Rows: []SeasonRow{}, Points: scale, Participation: q.Participation, Elo: q.Elo}
	rows := map[string]*SeasonRow{}
	row := func(name, club string) *SeasonRow {
		k := personKey(name)
		r := rows[k]
		if r == nil {
			r = &SeasonRow{Name: name, Places: make([]int, len(events)), Points: make([]float64, len(events)), Elo: EloStart}
			rows[k] = r
		}
		if club != "" {
			r.Club = club
		}
		return r
	}
	for i, e := range events {
		v.Events = append(v.Events, e.ev)
		if !e.ev.Finished {
			continue
		}
		st := e.st
		ranking := st.Ranking()
		tied := map[int]int{}
		for _, rk := range ranking {
			tied[rk.Rank]++
		}
		for _, rk := range ranking {
			p := st.Players[rk.Player]
			club := ""
			if p != nil {
				club = p.Club
			}
			r := row(playerNameIn(st, rk.Player), club)
			pts := placePoints(scale, rk.Rank, tied[rk.Rank]) + q.Participation
			r.Places[i], r.Points[i] = rk.Rank, pts
			r.Total += pts
			r.Played++
			if r.Best == 0 || rk.Rank < r.Best {
				r.Best = rk.Rank
			}
			for _, ph := range st.Phases {
				r.Wins += ph.Wins[rk.Player]
				r.Losses += ph.Losses[rk.Player]
			}
		}
		if q.Elo {
			for _, mid := range seasonMatchOrder(st) {
				m := st.Matches[mid]
				if m.Forfeit || m.Winner == "" {
					continue
				}
				loserID := m.A
				if m.Winner == m.A {
					loserID = m.B
				}
				w := row(playerNameIn(st, m.Winner), "")
				l := row(playerNameIn(st, loserID), "")
				w.Elo, l.Elo = eloUpdate(w.Elo, l.Elo, m.Length)
				w.Matches++
				l.Matches++
			}
		}
	}

	for _, r := range rows {
		if r.Played == 0 {
			continue
		}
		if !q.Elo {
			r.Elo = 0
		} else {
			r.Elo = math.Round(r.Elo*10) / 10
		}
		v.Rows = append(v.Rows, *r)
	}
	// Points first, then the better best place, then the name: a total order, so two calls
	// rank alike. People level on points and best place share their rank.
	sort.Slice(v.Rows, func(i, j int) bool {
		a, b := v.Rows[i], v.Rows[j]
		if a.Total != b.Total {
			return a.Total > b.Total
		}
		if a.Best != b.Best {
			return a.Best < b.Best
		}
		return personKey(a.Name) < personKey(b.Name)
	})
	for i := range v.Rows {
		if i > 0 && v.Rows[i].Total == v.Rows[i-1].Total && v.Rows[i].Best == v.Rows[i-1].Best {
			v.Rows[i].Rank = v.Rows[i-1].Rank
		} else {
			v.Rows[i].Rank = i + 1
		}
	}
	return v, nil
}

// seasonTournaments lists the candidate tournaments: a Rencontre's members, or every directed
// tournament of the scope.
func (d *Service) seasonTournaments(ctx context.Context, rencontreID int64) ([]int64, error) {
	if rencontreID != 0 {
		r, err := d.st.Rencontres().Get(ctx, d.scope, rencontreID)
		if err != nil {
			return nil, err
		}
		return r.TournamentIDs, nil
	}
	recs, err := d.dirStore().ListDirections(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(recs))
	for _, r := range recs {
		ids = append(ids, r.TournamentID)
	}
	return ids, nil
}

// seasonMatchOrder is the tournament's finished matches by end time, then creation order.
func seasonMatchOrder(st *tournoi.State) []tournoi.MatchID {
	ids := make([]tournoi.MatchID, 0, len(st.MatchOrder))
	for _, id := range st.MatchOrder {
		if m := st.Matches[id]; m != nil && m.Status == tournoi.Finished {
			ids = append(ids, id)
		}
	}
	sort.SliceStable(ids, func(i, j int) bool { return st.Matches[ids[i]].End.Before(st.Matches[ids[j]].End) })
	return ids
}

// SeasonCSV exports the season ranking: one line per person, then one column of points per
// tournament. Semicolon-separated, like the standings, for a French spreadsheet.
func (d *Service) SeasonCSV(ctx context.Context, q SeasonQuery) (string, error) {
	v, err := d.SeasonRanking(ctx, q)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	w := csv.NewWriter(&b)
	w.Comma = ';'
	head := []string{"rank", "name", "club", "total", "played", "best", "wins", "losses"}
	if v.Elo {
		head = append(head, "elo", "matches")
	}
	for _, e := range v.Events {
		head = append(head, e.Date+" "+e.Name)
	}
	if err := w.Write(head); err != nil {
		return "", err
	}
	num := func(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
	for _, r := range v.Rows {
		line := []string{strconv.Itoa(r.Rank), r.Name, r.Club, num(r.Total), strconv.Itoa(r.Played),
			strconv.Itoa(r.Best), strconv.Itoa(r.Wins), strconv.Itoa(r.Losses)}
		if v.Elo {
			line = append(line, num(r.Elo), strconv.Itoa(r.Matches))
		}
		for i := range v.Events {
			cell := ""
			if r.Places[i] != 0 {
				cell = num(r.Points[i])
			}
			line = append(line, cell)
		}
		if err := w.Write(line); err != nil {
			return "", err
		}
	}
	w.Flush()
	return b.String(), w.Error()
}
