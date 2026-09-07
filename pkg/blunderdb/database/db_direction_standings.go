package database

import (
	"context"
	"encoding/csv"
	"strconv"
	"strings"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/PileOfCells/backgammon-tournoi/render"
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

	// The overall ranking has a scale of its own, under the key `all`. Passing nil here paid
	// nobody at all when a director had put their whole prize fund on the general standings —
	// the commonest configuration there is (found by the arithmetic test of #393).
	overall := StandingsSection{Rows: rowsFor(st, st.Ranking(), st.PrizeAmounts(tournoi.PrizeSectionAll))}
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

// StandingsCSV exports the standings in the USER'S LANGUAGE.
//
// The engine writes a CSV of its own, and it writes it in French — headings and ranking notes
// alike. That is fine for its own console and wrong here: this file is pasted into a director's
// accounts, and blunderDB speaks nine languages. So the CSV is built from the same catalogue the
// display page and the pairing sheet use (issue #386), and a ranking note goes through the same
// labeler rather than through the engine's `String()`.
//
// The separator stays a semicolon, which is what shipped and what a French spreadsheet opens
// without being asked.
func (d *Database) StandingsCSV(tournamentID int64) (string, error) {
	v, err := d.Standings(tournamentID)
	if err != nil {
		return "", err
	}
	if v == nil {
		return "", direction.ErrNoDirection
	}
	cat, _ := d.directionStrings()
	l := direction.NewLabeler(cat, nil)
	var b strings.Builder
	w := csv.NewWriter(&b)
	w.Comma = ';'
	head := []string{
		l.Term(render.TermPhase, 0), l.Term(render.TermRank, 0), "id",
		l.Term(render.TermPlayer, 0), l.Term(render.TermClub, 0),
		l.Term(render.TermState, 0), l.Term(render.TermPrize, 0),
	}
	if err := w.Write(head); err != nil {
		return "", err
	}
	for _, sec := range v.Sections {
		name := l.SectionName(sec.Name)
		for _, r := range sec.Rows {
			prize := ""
			if r.Prize != 0 {
				prize = strconv.FormatFloat(r.Prize, 'f', 2, 64)
			}
			row := []string{name, strconv.Itoa(r.Rank), r.ID, r.Name, r.Club, l.Note(r.Note), prize}
			if err := w.Write(row); err != nil {
				return "", err
			}
		}
	}
	w.Flush()
	return b.String(), w.Error()
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
