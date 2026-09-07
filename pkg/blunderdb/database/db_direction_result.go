package database

import (
	"context"
	"fmt"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"
	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
)

// Entering a result, and the room the matches are played in (ADR-0047 §5.3 and §5.5).
//
// The one rule that shapes every signature here: THE WINNER IS THE ONLY THING REQUIRED. A
// director often writes "Alice wins" and nothing else, and a result with no score is an
// ordinary result — not a half-filled form. The scores are optional, the remark is rarer still,
// and a score that contradicts the announced length is ACCEPTED with a warning rather than
// refused: during a tournament it is the director's word that stands.

// TableCell is one square of the table grid: what the director reads from two metres away.
type TableCell struct {
	Table int `json:"table"`
	// Free, Unavailable and Reserved are mutually exclusive with a running match.
	Free        bool   `json:"free"`
	Unavailable bool   `json:"unavailable"`
	Reserved    bool   `json:"reserved"`
	MatchID     string `json:"matchId,omitempty"`
	A           string `json:"a,omitempty"`
	B           string `json:"b,omitempty"`
	AName       string `json:"aName,omitempty"`
	BName       string `json:"bName,omitempty"`
	Length      int    `json:"length,omitempty"`
	// ElapsedSeconds and Slow are what makes a table that is dragging visible before a player
	// comes to complain.
	ElapsedSeconds int  `json:"elapsedSeconds,omitempty"`
	Slow           bool `json:"slow,omitempty"`
}

// TableGrid returns one cell per table of the room, in order. A room with no declared table
// count shows exactly the tables in use, since there is nothing else to draw.
func (d *Database) TableGrid(tournamentID int64) ([]TableCell, error) {
	dir, err := direction.Open(context.Background(), d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	now := time.Now()
	running := map[int]*tournoi.Match{}
	highest := 0
	for _, m := range st.Running() {
		if m.Table > 0 {
			running[m.Table] = m
			if m.Table > highest {
				highest = m.Table
			}
		}
	}
	count := st.Config.Tables.Count
	if count == 0 {
		count = highest
	}
	// A slow match is one past 1.5× its expected duration. The threshold is the one the UX
	// document fixes; it is a default, not a law of nature.
	const slowFactor = 1.5
	slow := map[tournoi.MatchID]bool{}
	for _, m := range st.SlowMatches(now, slowFactor) {
		slow[m.ID] = true
	}

	out := make([]TableCell, 0, count)
	for n := 1; n <= count; n++ {
		c := TableCell{Table: n}
		if m := running[n]; m != nil {
			c.MatchID = string(m.ID)
			c.A, c.B = string(m.A), string(m.B)
			c.AName, c.BName = playerNameIn(st, m.A), playerNameIn(st, m.B)
			c.Length = m.Length
			c.ElapsedSeconds = int(now.Sub(m.Start).Seconds())
			c.Slow = slow[m.ID]
			out = append(out, c)
			continue
		}
		switch {
		case !st.Config.Tables.AvailableFor(n, "", st.Current) && contains(st.Config.Tables.Unavailable, n):
			c.Unavailable = true
		case !st.Config.Tables.AvailableFor(n, "", st.Current):
			c.Reserved = true
		default:
			c.Free = true
		}
		out = append(out, c)
	}
	return out, nil
}

func contains(xs []int, n int) bool {
	for _, x := range xs {
		if x == n {
			return true
		}
	}
	return false
}

func playerNameIn(st *tournoi.State, id tournoi.PlayerID) string {
	if p := st.Players[id]; p != nil {
		return p.Name
	}
	return string(id)
}

// EnterResult records the result of a match. `winner` is the only thing required; scoreA and
// scoreB may both be zero, which is what a director who wrote only "Alice wins" produces.
//
// `note` is the rare remark — "ran out of time", "abandoned: …" — and travels on the event.
// A score beyond the announced length is accepted; the engine raises a standing warning and
// the director decides what to do with it.
func (d *Database) EnterResult(tournamentID int64, matchID, winner string, scoreA, scoreB int, note string) (*DirectionView, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	ev := tournoi.ResultEvent(tournoi.MatchID(matchID), tournoi.PlayerID(winner), scoreA, scoreB, time.Now())
	if note != "" {
		ev = ev.WithNote(note)
	}
	if err := dir.Apply(ctx, ev); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}

// EnterForfeit records that a player did not turn up for THIS match, without withdrawing them
// from the tournament: they go on to follow a loser's path, the consolation for instance.
func (d *Database) EnterForfeit(tournamentID int64, matchID, winner, note string) (*DirectionView, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	ev := tournoi.ForfeitEvent(tournoi.MatchID(matchID), tournoi.PlayerID(winner), time.Now())
	if note != "" {
		ev = ev.WithNote(note)
	}
	if err := dir.Apply(ctx, ev); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}

// MoveMatchToTable moves a running match to another table (noise, light, a broadcast).
func (d *Database) MoveMatchToTable(tournamentID int64, matchID string, table int) (*DirectionView, error) {
	if table <= 0 {
		return nil, fmt.Errorf("direction: table %d is not a table", table)
	}
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	if err := dir.Apply(ctx, tournoi.TableChangedEvent(tournoi.MatchID(matchID), table, time.Now())); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}

// CancelMatch removes a match launched by mistake — the wrong players, the wrong table. The
// state is recomputed; nothing is erased from the log.
func (d *Database) CancelMatch(tournamentID int64, matchID string) (*DirectionView, error) {
	ctx := context.Background()
	dir, err := direction.Open(ctx, d.DirectionStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	if err := dir.Apply(ctx, tournoi.CancelEvent(tournoi.MatchID(matchID), time.Now())); err != nil {
		return nil, err
	}
	return d.GetDirection(tournamentID)
}
