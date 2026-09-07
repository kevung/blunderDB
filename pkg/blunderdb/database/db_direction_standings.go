package database

import (
	"context"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// The standings, the prizes and the close (ADR-0047 §5.6 and §3.6, issues #374, #393).
//
// The last thing a director does, and the only one the players take home. Two rules from the
// engine hold here and are not this layer's to soften: there is NO tie-break, so ties stay ties
// and share their prizes; and a ranking note is a CODE, rendered by the frontend.

// StandingRow is one line of the standings.
type StandingRow struct {
	Rank   int          `json:"rank"`
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Club   string       `json:"club,omitempty"`
	Note   tournoi.Note `json:"note"`
	Prize  float64      `json:"prize,omitempty"`
	Shared bool         `json:"shared,omitempty"`
}

// StandingsSection is one ranking: the overall one, or a section of a bracket with its own
// prizes (a consolation pays its own winners).
type StandingsSection struct {
	// Name is empty for the overall standings, and the section's identifier otherwise.
	Name string        `json:"name"`
	Kind string        `json:"kind,omitempty"`
	Rows []StandingRow `json:"rows"`
}

// StandingsView is what the standings tab shows, prizes included.
type StandingsView struct {
	Finished bool               `json:"finished"`
	Pool     float64            `json:"pool"`
	Retained float64            `json:"retained"`
	Payable  float64            `json:"payable"`
	Entrants int                `json:"entrants"`
	Sections []StandingsSection `json:"sections"`
}

// Standings replays the tournament and returns its rankings with the prizes attached.
func (d *Database) Standings(tournamentID int64) (*StandingsView, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	v := &StandingsView{
		Finished: st.Finished,
		Pool:     st.Pool(),
		Payable:  st.Distributable(),
		Entrants: len(st.Order),
	}
	v.Retained = v.Pool - v.Payable

	overall := StandingsSection{Rows: rowsFor(st, st.Ranking(), nil)}
	v.Sections = append(v.Sections, overall)

	// A section with prizes of its own ranks and pays on its own: the consolation's winner is
	// not the tournament's runner-up, and paying them out of the overall ranking would say so.
	for _, ph := range st.Phases {
		for _, sec := range ph.Sections {
			amounts := st.PrizeAmounts(sec.Name)
			if len(amounts) == 0 {
				continue
			}
			r := st.SectionRanking(sec.Name)
			if len(r) == 0 {
				continue
			}
			v.Sections = append(v.Sections, StandingsSection{
				Name: sec.Name, Kind: sec.Kind, Rows: rowsFor(st, r, amounts),
			})
		}
	}
	return v, nil
}

// rowsFor turns a ranking into rows, sharing the prizes of the places a tie occupies.
func rowsFor(st *tournoi.State, ranking []tournoi.Rank, amounts []float64) []StandingRow {
	share := map[tournoi.PlayerID]float64{}
	if len(amounts) > 0 {
		share = tournoi.Prizes(ranking, amounts)
	}
	// Who shares a rank with someone else: the view says so, because a tie is a fact of the
	// result and not a rounding artefact.
	count := map[int]int{}
	for _, r := range ranking {
		count[r.Rank]++
	}
	out := make([]StandingRow, 0, len(ranking))
	for _, r := range ranking {
		row := StandingRow{
			Rank: r.Rank, ID: string(r.Player), Name: playerNameIn(st, r.Player),
			Note: r.Note, Prize: share[r.Player], Shared: count[r.Rank] > 1,
		}
		if p := st.Players[r.Player]; p != nil {
			row.Club = p.Club
		}
		out = append(out, row)
	}
	return out
}

// StandingsCSV exports the standings as the engine writes them.
func (d *Database) StandingsCSV(tournamentID int64) (string, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return "", err
	}
	st := dir.State()
	if st == nil {
		return "", direction.ErrNoDirection
	}
	return string(st.StandingsCSV()), nil
}

// CloseDirection closes the tournament and freezes the final standings.
func (d *Database) CloseDirection(tournamentID int64) (*DirectionView, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	if err := dir.Finish(ctx, time.Now()); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}

// ReopenDirection takes a closed tournament back, because a result was wrong. The final
// standings are recomputed at the next close; the log keeps everything.
func (d *Database) ReopenDirection(tournamentID int64) (*DirectionView, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	if err := dir.Reopen(ctx, time.Now()); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}
