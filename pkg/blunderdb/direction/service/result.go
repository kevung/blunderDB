package service

import (
	"context"
	"fmt"
	"slices"
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
	// Name, Room and AssignedTo are the table's properties (ADR-0058): the name shown beside
	// the number, the room it stands in, the persons it is kept for.
	Name       string   `json:"name,omitempty"`
	Room       string   `json:"room,omitempty"`
	AssignedTo []string `json:"assignedTo,omitempty"`
	// OutsideRooms marks a table outside the rooms the event may play in: never proposed to
	// it, and refused to its gestures.
	OutsideRooms bool `json:"outsideRooms,omitempty"`
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
	room, err := d.roomAround(ctx, tournamentID, dir)
	if err != nil {
		return nil, err
	}
	return gridOf(dir, room, time.Now()), nil
}

// gridOf draws the table grid of a replayed Direction in the room given.
func gridOf(dir *direction.Direction, room sisterRoom, now time.Time) []TableCell {
	st := dir.State()
	if st == nil {
		return nil
	}
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
	named := func(n int) TableCell {
		c := TableCell{Table: n, OutsideRooms: !room.plan.Allowed(n)}
		if s, ok := room.plan.Setting(n); ok {
			c.Name, c.Room, c.AssignedTo = s.Name, s.Room, s.AssignedTo
		}
		return c
	}
	for n := 1; n <= count; n++ {
		c := named(n)
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
		case slices.Contains(room.plan.Closed(), n) && room.plan.Allowed(n):
			// Reserved, or kept for someone: never proposed, placed by hand (ADR-0058 §7, §8).
			c.Reserved = true
		default:
			c.Free = true
		}
		out = append(out, c)
	}
	for _, m := range extra {
		c := named(m.Table)
		c.Shared = true
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
	return out
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
	if err := d.enterResult(ctx, tournamentID, matchID, winner, scoreA, scoreB, note); err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
}

func (d *Service) enterResult(ctx context.Context, tournamentID int64, matchID, winner string, scoreA, scoreB int, note string) (err error) {
	d, release, err := d.lockDirection(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	ev := tournoi.ResultEvent(tournoi.MatchID(matchID), tournoi.PlayerID(winner), scoreA, scoreB, time.Now())
	if note != "" {
		ev = ev.WithNote(note)
	}
	if err := dir.Apply(ctx, ev); err != nil {
		return err
	}
	return nil
}

// EnterForfeit records that a player did not turn up for THIS match, without withdrawing them
// from the tournament: they go on to follow a loser's path, the consolation for instance.
func (d *Service) EnterForfeit(ctx context.Context, tournamentID int64, matchID, winner, note string) (*DirectionView, error) {
	if err := d.enterForfeit(ctx, tournamentID, matchID, winner, note); err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
}

func (d *Service) enterForfeit(ctx context.Context, tournamentID int64, matchID, winner, note string) (err error) {
	d, release, err := d.lockDirection(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	ev := tournoi.ForfeitEvent(tournoi.MatchID(matchID), tournoi.PlayerID(winner), time.Now())
	if note != "" {
		ev = ev.WithNote(note)
	}
	if err := dir.Apply(ctx, ev); err != nil {
		return err
	}
	return nil
}

// MoveMatchToTable moves a running match to another table (noise, light, a broadcast). When
// that table is taken, the two matches swap tables: two matches never share a table, and the
// same gesture puts them back. Both moves are written in one transaction.
//
// In a Rencontre the occupant may belong to a sister event: the swap then writes one table
// change in each of the two logs, in the same transaction, under the room's lock — each log
// still replays alone, and the room never holds two matches on one table because of a move.
func (d *Service) MoveMatchToTable(ctx context.Context, tournamentID int64, matchID string, table int) (*DirectionView, error) {
	if table <= 0 {
		return nil, direction.Refusef("direction: table %d is not a table", table)
	}
	touched, err := d.moveMatch(ctx, tournamentID, matchID, table)
	if err != nil {
		return nil, err
	}
	// The display page of every Direction the move wrote in — the sister's too, whose director
	// may not be looking at it — is rewritten here, so the GUI, the CLI and any client get it
	// alike, once the lock is released: a slow folder must not hold the room's other gestures.
	d.writePages(context.WithoutCancel(ctx), tournamentID, touched)
	return d.viewAfter(ctx, tournamentID)
}

// moveMatch writes the move and, for a swap with a sister event, names that sister in touched.
func (d *Service) moveMatch(ctx context.Context, tournamentID int64, matchID string, table int) (touched int64, err error) {
	// The room's lock as soon as the event is in a Rencontre, and its members read under it: a
	// swap may write a sister's log, and the membership must not change in between.
	// The pages are written by the caller, the sister's with them, rather than on release.
	d, release, shared, err := d.lockGesture(ctx, gestureTarget{tournamentID: tournamentID}, false)
	if err != nil {
		return 0, err
	}
	defer release(&err)
	var sisters []int64
	if shared {
		sisters = d.sistersOf(ctx, tournamentID)
	}
	err = d.directionTx(ctx, func(ctx context.Context, _ storage.Tx, store direction.Store) error {
		dir, err := direction.Open(ctx, store, tournamentID)
		if err != nil {
			return err
		}
		if err := d.tableBeyond(ctx, dir, tournamentID, table); err != nil {
			return err
		}
		if err := d.refuseOutside(ctx, dir, tournamentID, table); err != nil {
			return err
		}
		st := dir.State()
		if st == nil {
			return direction.Refusef("direction: the tournament has not started")
		}
		m := st.Matches[tournoi.MatchID(matchID)]
		if m == nil || m.Status != tournoi.Running {
			return direction.Refusef("direction: match %q is not running", matchID)
		}
		from := m.Table
		if from == table {
			return nil
		}
		if contains(st.Config.Tables.Unavailable, table) {
			// A table out of service takes no match, whether by a move or as the far end of a
			// swap: the rule is here so the grid, the card and the CLI refuse in one voice.
			return direction.Refusef("direction: table %d is out of service", table)
		}
		o := occupant(st, table, m.ID)
		var sister *direction.Direction
		var so *tournoi.Match
		if o == nil {
			if sister, so, err = sisterOccupant(ctx, store, sisters, table); err != nil {
				return err
			}
		}
		if (o != nil || so != nil) && from <= 0 {
			// A match with no table has nowhere to send the occupant, whatever its event:
			// refused rather than stacking two matches on one table.
			return direction.Refusef("direction: table %d is taken", table)
		}
		// The occupant goes back to the moved match's table: that end of the swap must be in
		// service too, in the occupant's own configuration.
		if o != nil && contains(st.Config.Tables.Unavailable, from) {
			return direction.Refusef("direction: table %d is out of service", from)
		}
		if so != nil && contains(sister.State().Config.Tables.Unavailable, from) {
			return direction.Refusef("direction: table %d is out of service", from)
		}
		if so != nil {
			// The sister's match lands on the moved match's table: that end of the swap must
			// be in the sister's rooms too.
			if err := d.refuseOutside(ctx, sister, sister.Record().TournamentID, from); err != nil {
				return err
			}
		}
		now := time.Now()
		if err := dir.Apply(ctx, tournoi.TableChangedEvent(m.ID, table, now)); err != nil {
			return err
		}
		switch {
		case o != nil:
			return dir.Apply(ctx, tournoi.TableChangedEvent(o.ID, from, now))
		case so != nil:
			touched = sister.Record().TournamentID
			return sister.Apply(ctx, tournoi.TableChangedEvent(so.ID, from, now))
		}
		return nil
	})
	return touched, err
}

// sistersOf lists the other events of the Rencontre a Tournament plays in; none outside one.
func (d *Service) sistersOf(ctx context.Context, tournamentID int64) []int64 {
	rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID)
	if err != nil || rid == 0 {
		return nil
	}
	r, err := d.st.Rencontres().Get(ctx, d.scope, rid)
	if err != nil {
		return nil
	}
	var out []int64
	for _, tid := range r.TournamentIDs {
		if tid != tournamentID {
			out = append(out, tid)
		}
	}
	return out
}

// sisterOccupant finds the match of a sister event running at this table, with its Direction
// opened on the transaction's store so the swap writes through it; nil when the table is free
// of them.
func sisterOccupant(ctx context.Context, store direction.Store, sisters []int64, table int) (*direction.Direction, *tournoi.Match, error) {
	for _, tid := range sisters {
		// A sister that does not replay may be sitting on this very table: a write that cannot
		// see the room is refused rather than risking two matches on one table.
		dir, err := direction.Open(ctx, store, tid)
		if err != nil {
			return nil, nil, fmt.Errorf("direction: event %d of the room does not replay: %w", tid, err)
		}
		st := dir.State()
		if st == nil || st.Finished {
			continue
		}
		if o := occupant(st, table, ""); o != nil {
			return dir, o, nil
		}
	}
	return nil, nil, nil
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
	if err := d.cancelMatch(ctx, tournamentID, matchID); err != nil {
		return nil, err
	}
	return d.viewAfter(ctx, tournamentID)
}

func (d *Service) cancelMatch(ctx context.Context, tournamentID int64, matchID string) (err error) {
	d, release, err := d.lockDirection(ctx, tournamentID)
	if err != nil {
		return err
	}
	defer release(&err)
	dir, err := direction.Open(ctx, d.dirStore(), tournamentID)
	if err != nil {
		return err
	}
	if err := dir.Apply(ctx, tournoi.CancelEvent(tournoi.MatchID(matchID), time.Now())); err != nil {
		return err
	}
	return nil
}

// tableBeyond refuses a table past the last one — the room's in a Rencontre, the configuration's
// otherwise: a match seated there would have no cell in the grid and vanish from it. A count of
// 0 bounds nothing.
func (d *Service) tableBeyond(ctx context.Context, dir *direction.Direction, tournamentID int64, table int) error {
	limit := 0
	if cfg, err := dir.Config(); err == nil {
		limit = cfg.Tables.Count
	}
	if rid, err := d.st.Rencontres().Of(ctx, d.scope, tournamentID); err == nil && rid != 0 {
		if r, err := d.st.Rencontres().Get(ctx, d.scope, rid); err == nil && r.Tables > 0 {
			limit = r.Tables
		}
	}
	if limit > 0 && table > limit {
		return direction.Refusef("direction: there is no table %d, the last is %d", table, limit)
	}
	return nil
}
