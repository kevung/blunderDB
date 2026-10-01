package service

import (
	"context"
	"fmt"
	"time"

	tournoi "github.com/PileOfCells/backgammon-tournoi"

	"github.com/kevung/blunderdb/pkg/blunderdb/direction"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Entering a result, and the room the matches are played in (tasks/nicomaque/fonctionnel.md §5.3 and §5.5).
//
// THE WINNER IS THE ONLY THING REQUIRED: a result without a score is ordinary. A score
// contradicting the announced length is ACCEPTED with a warning; the director's word stands.

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
	// NoTable marks a running match that has no table — paired by hand in a full room.
	// Its Table is 0; such cells come after the room's tables, one per match.
	NoTable bool `json:"noTable,omitempty"`
	// Shared flags a table holding two running matches at once — a state the log may carry from
	// before moves became swaps. Every such match gets its own cell with the same Table, the
	// extras after the room's tables, so none of them disappears from the grid.
	Shared bool `json:"shared,omitempty"`
	// Elsewhere names the sister event of the Rencontre playing on this table right now: the
	// table is not free for this one (ADR-0056).
	Elsewhere string `json:"elsewhere,omitempty"`
	// AElsewhere and BElsewhere say where a player of this match also sits right now in a
	// sister event: a manual pairing the engine would not have proposed, accepted and shown.
	AElsewhere *direction.Seat `json:"aElsewhere,omitempty"`
	BElsewhere *direction.Seat `json:"bElsewhere,omitempty"`
}

// TableGrid returns one cell per table of the room, in order. A room with no declared table
// count shows exactly the tables in use, since there is nothing else to draw.
func (d *Service) TableGrid(ctx context.Context, tournamentID int64) ([]TableCell, error) {
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	st := dir.State()
	if st == nil {
		return nil, nil
	}
	now := time.Now()
	running := map[int]*tournoi.Match{}
	var tableless, extra []*tournoi.Match
	shared := map[int]bool{}
	highest := 0
	for _, m := range st.Running() {
		if m.Table <= 0 {
			tableless = append(tableless, m)
			continue
		}
		if running[m.Table] != nil {
			shared[m.Table] = true
			extra = append(extra, m)
			continue
		}
		running[m.Table] = m
		if m.Table > highest {
			highest = m.Table
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

	room := d.roomAround(ctx, tournamentID, dir)
	elsewhere := room.tables
	seat := func(id tournoi.PlayerID) *direction.Seat {
		if s, ok := room.players[id]; ok {
			return &s
		}
		return nil
	}
	fill := func(c *TableCell, m *tournoi.Match) {
		c.MatchID = string(m.ID)
		c.A, c.B = string(m.A), string(m.B)
		c.AName, c.BName = playerNameIn(st, m.A), playerNameIn(st, m.B)
		c.Length = m.Length
		c.ElapsedSeconds = int(now.Sub(m.Start).Seconds())
		c.Slow = slow[m.ID]
		c.AElsewhere, c.BElsewhere = seat(m.A), seat(m.B)
	}

	out := make([]TableCell, 0, count+len(tableless))
	for n := 1; n <= count; n++ {
		c := TableCell{Table: n}
		if m := running[n]; m != nil {
			fill(&c, m)
			c.Shared = shared[n]
			out = append(out, c)
			continue
		}
		switch {
		case elsewhere[n] != "" && !contains(st.Config.Tables.Unavailable, n):
			c.Elsewhere = elsewhere[n]
		case !st.Config.Tables.AvailableFor(n, "", st.Current) && contains(st.Config.Tables.Unavailable, n):
			c.Unavailable = true
		case !st.Config.Tables.AvailableFor(n, "", st.Current):
			c.Reserved = true
		default:
			c.Free = true
		}
		out = append(out, c)
	}
	for _, m := range extra {
		c := TableCell{Table: m.Table, Shared: true}
		fill(&c, m)
		out = append(out, c)
	}
	// A match with no table is still a match in the room: leaving it out of the grid is how a
	// manual pairing came to be launched and seen nowhere.
	for _, m := range tableless {
		c := TableCell{NoTable: true}
		fill(&c, m)
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

// EnterResult records the result of a match. Only `winner` is required; both scores may be zero.
// `note` travels on the event. A score beyond the length raises a standing warning.
func (d *Service) EnterResult(ctx context.Context, tournamentID int64, matchID, winner string, scoreA, scoreB int, note string) (*DirectionView, error) {
	defer d.lockDirection(tournamentID)()
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
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
	return d.GetDirection(ctx, tournamentID)
}

// EnterForfeit records that a player did not turn up for THIS match, without withdrawing them
// from the tournament: they go on to follow a loser's path, the consolation for instance.
func (d *Service) EnterForfeit(ctx context.Context, tournamentID int64, matchID, winner, note string) (*DirectionView, error) {
	defer d.lockDirection(tournamentID)()
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
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
	return d.GetDirection(ctx, tournamentID)
}

// MoveMatchToTable moves a running match to another table (noise, light, a broadcast). When
// that table is taken, the two matches swap tables: two matches never share a table, and the
// same gesture puts them back. Both moves are written in one transaction.
func (d *Service) MoveMatchToTable(ctx context.Context, tournamentID int64, matchID string, table int) (*DirectionView, error) {
	defer d.lockDirection(tournamentID)()
	if table <= 0 {
		return nil, fmt.Errorf("direction: table %d is not a table", table)
	}
	err := d.directionTx(ctx, func(ctx context.Context, _ storage.Tx, store direction.Store) error {
		dir, err := direction.Open(ctx, store, tournamentID)
		if err != nil {
			return err
		}
		st := dir.State()
		if st == nil {
			return fmt.Errorf("direction: the tournament has not started")
		}
		m := st.Matches[tournoi.MatchID(matchID)]
		if m == nil || m.Status != tournoi.Running {
			return fmt.Errorf("direction: match %q is not running", matchID)
		}
		from := m.Table
		if from == table {
			return nil
		}
		o := occupant(st, table, m.ID)
		if o != nil && from <= 0 {
			// A match with no table has nowhere to send the occupant: refused rather than
			// stacking two matches on one table.
			return fmt.Errorf("direction: table %d is taken", table)
		}
		now := time.Now()
		if err := dir.Apply(ctx, tournoi.TableChangedEvent(m.ID, table, now)); err != nil {
			return err
		}
		if o != nil {
			return dir.Apply(ctx, tournoi.TableChangedEvent(o.ID, from, now))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return d.GetDirection(ctx, tournamentID)
}

// occupant is the running match at this table other than self, or nil.
func occupant(st *tournoi.State, table int, self tournoi.MatchID) *tournoi.Match {
	for _, m := range st.Running() {
		if m.Table == table && m.ID != self {
			return m
		}
	}
	return nil
}

// CancelMatch removes a match launched by mistake — the wrong players, the wrong table. The
// state is recomputed; nothing is erased from the log.
func (d *Service) CancelMatch(ctx context.Context, tournamentID int64, matchID string) (*DirectionView, error) {
	defer d.lockDirection(tournamentID)()
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return nil, err
	}
	if err := dir.Apply(ctx, tournoi.CancelEvent(tournoi.MatchID(matchID), time.Now())); err != nil {
		return nil, err
	}
	return d.GetDirection(ctx, tournamentID)
}
