package duel

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// TimeOut is what running out of reserve does to a Duel (ADR-0073 rule 1).
type TimeOut string

const (
	// TimeContinue notes the overrun and lets the match go on: the default,
	// a training match stays whole.
	TimeContinue TimeOut = "continue"
	// TimeLoseMatch is the official sanction (USBGF, WBGF 2.1 § 4.3 (iii)):
	// the Match stops where the Duel stopped and carries the fact, no point
	// invented.
	TimeLoseMatch TimeOut = "lose_match"
)

// ErrInvalidCadence: a Cadence that sets no reserve, two, or one the session
// cannot have.
var ErrInvalidCadence = errors.New("invalid cadence")

// Cadence is the clock of a Duel: a reserve per Side and a delay free at each
// turn of the clock, never added to the reserve; no increment, which
// backgammon does not use (ADR-0073). The Arbiter keeps it, from the instants
// it stamps. A Duel without one is the default.
//
// The reserve is either Reserve, fixed for the match, or ReservePerPoint,
// times the average length still to play: ReservePerPoint × (2L − a − b) / 2
// at a score a-b of a match to L, which is how the tournament rules state it.
type Cadence struct {
	// Name is the preset the Cadence came from, "" for one set by hand.
	Name string `json:"name,omitempty"`
	// Reserve is each Side's reserve for the match, in seconds.
	Reserve int `json:"reserve,omitempty"`
	// ReservePerPoint is the reserve per point still to play, in seconds,
	// counted at the score the Duel starts from — the 2013 clock rules take
	// off the points already scored — and then the match's, as Reserve is.
	ReservePerPoint int `json:"reservePerPoint,omitempty"`
	// Delay is the simple delay, in seconds: what a turn of the clock uses
	// before the reserve runs, never carried over.
	Delay int `json:"delay"`
	// TimeOut is what running out does; "" is TimeContinue.
	TimeOut TimeOut `json:"timeOut,omitempty"`
}

// namedCadences are the Cadences a Duel offers by name.
var namedCadences = []Cadence{
	// USBGF in-person rules and WBGF 2.1 § 3.5 (v)-(vi): 2 min per point of
	// the average length left, 12 s simple delay.
	{Name: "tournament", ReservePerPoint: 120, Delay: 12},
	// Presets of a third-party clock application (hansdezwart/bgclock), its
	// author's declaration and nothing more: rapid play, no rulebook behind.
	{Name: "rapid-3+12", Reserve: 180, Delay: 12},
	{Name: "rapid-2+12", Reserve: 120, Delay: 12},
	{Name: "rapid-3+15", Reserve: 180, Delay: 15},
}

// NamedCadences returns the Cadences a Duel offers by name, the tournament
// one first.
func NamedCadences() []Cadence { return append([]Cadence(nil), namedCadences...) }

// NamedCadence returns the Cadence of that name, its TimeOut left to the
// Duel to choose.
func NamedCadence(name string) (Cadence, bool) {
	for _, c := range namedCadences {
		if c.Name == name {
			return c, true
		}
	}
	return Cadence{}, false
}

// check refuses a Cadence the session cannot be played under.
func (c Cadence) check(matchLength int) error {
	switch {
	case (c.Reserve > 0) == (c.ReservePerPoint > 0):
		return fmt.Errorf("%w: one reserve, fixed or per point", ErrInvalidCadence)
	case c.Reserve < 0 || c.ReservePerPoint < 0 || c.Delay < 0:
		return fmt.Errorf("%w: a negative time", ErrInvalidCadence)
	case c.ReservePerPoint > 0 && matchLength <= 0:
		return fmt.Errorf("%w: a reserve per point needs a match length", ErrInvalidCadence)
	case c.TimeOut != "" && c.TimeOut != TimeContinue && c.TimeOut != TimeLoseMatch:
		return fmt.Errorf("%w: time out %q", ErrInvalidCadence, c.TimeOut)
	}
	return nil
}

// reserveMS is a Side's reserve for a match started at score, in milliseconds.
func (c Cadence) reserveMS(matchLength int, score [2]int) int64 {
	if c.ReservePerPoint == 0 {
		return int64(c.Reserve) * 1000
	}
	left := int64(2*matchLength - score[0] - score[1])
	return int64(c.ReservePerPoint) * 1000 * max(left, 0) / 2
}

// String is the Cadence as a Match's origin records it.
func (c Cadence) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}

// clock is what the draft keeps of time: the awaited Decision's start, and,
// under a Cadence, the reserves. Durations are milliseconds.
type clock struct {
	// Since is when the awaited Decision was handed to its Side, or resumed;
	// zero while no Decision is timed or the Duel is suspended.
	Since time.Time `json:"since,omitzero"`
	// Spent is what the awaited Decision took before its Duel was suspended.
	Spent int64 `json:"spent,omitempty"`
	// Unknown says part of the awaited Decision's time is lost: the Duel was
	// left running when its process stopped, so how much of the gap was
	// thought is not known. Its duration is then unknown, not short.
	Unknown bool `json:"unknown,omitempty"`
	// Cube is the cube decision of the turn whose roll is on the board, for
	// the checker play to carry.
	Cube *int64 `json:"cube,omitempty"`
	// Turn is what the running turn of the clock has used: a cube decision
	// and the play after its roll are one turn, and one delay.
	Turn int64 `json:"turn,omitempty"`
	// Reserve is each Side's reserve left, set when the Duel is created.
	Reserve [2]int64 `json:"reserve,omitzero"`
	// OverTime is the player (1 or 2) whose reserve ran out first, 0 none.
	OverTime int `json:"overTime,omitempty"`
}

// startDecision stamps the awaited Decision as handed out now, unless it
// already runs.
func (g *game) startDecision() {
	if c := &g.doc.Clock; c.Since.IsZero() {
		c.Since = g.clock()
	}
}

// setReserves gives each Side its reserve for the match, from the score the
// Duel starts at.
func (g *game) setReserves() {
	if cad := g.doc.Cadence; cad != nil {
		r := cad.reserveMS(g.doc.Header.MatchLength, g.m.Score())
		g.doc.Clock.Reserve = [2]int64{r, r}
	}
}

// elapsed is what the awaited Decision has taken at instant at, and whether
// that is all of it.
func (g *game) elapsed(at time.Time) (int64, bool) {
	c := g.doc.Clock
	if c.Since.IsZero() || at.IsZero() {
		return c.Spent, false
	}
	return c.Spent + max(at.Sub(c.Since).Milliseconds(), 0), !c.Unknown
}

// charge is what a Decision of ms takes from the reserve: the part of the
// clock's turn beyond the delay that this Decision adds.
func (g *game) charge(ms int64) int64 {
	return chargeMS(int64(g.doc.Cadence.Delay)*1000, g.doc.Clock.Turn, ms)
}

// chargeMS is what a Decision of ms takes from the reserve in a clock's turn
// that had used turn, under a simple delay.
func chargeMS(delay, turn, ms int64) int64 {
	return max(turn+ms-delay, 0) - max(turn-delay, 0)
}

// overTime reports whether the awaited Side's reserve cannot pay for ms.
func (g *game) overTime(side int, ms int64) bool {
	return g.doc.Cadence != nil && g.charge(ms) > g.doc.Clock.Reserve[side]
}

// noteOverTime records that side ran out, if no one did before.
func (g *game) noteOverTime(side int) {
	if g.doc.Clock.OverTime == 0 {
		g.doc.Clock.OverTime = side + 1
	}
}

// timeLost reports whether running out has ended the Duel.
func (g *game) timeLost() bool {
	return g.doc.Cadence != nil && g.doc.Cadence.TimeOut == TimeLoseMatch && g.doc.Clock.OverTime != 0
}

// account closes a Decision of ms that side took and ended with a play of
// kind: the reserve pays, and the clock's turn goes on only from a cube
// decision to the play after its roll.
func (g *game) account(side int, ms int64, kind PlayKind) {
	c := &g.doc.Clock
	if g.doc.Cadence != nil {
		if g.overTime(side, ms) {
			c.Reserve[side] = 0
			g.noteOverTime(side)
		} else {
			c.Reserve[side] -= g.charge(ms)
		}
	}
	c.Turn += ms
	if kind != PlayRoll {
		c.Turn = 0
	}
	c.Since, c.Spent, c.Unknown = time.Time{}, 0, false
}

// suspend stops the clocks at now: what the awaited Decision took is kept.
func (g *game) suspend(now time.Time) {
	c := &g.doc.Clock
	if !c.Since.IsZero() {
		c.Spent += max(now.Sub(c.Since).Milliseconds(), 0)
	}
	c.Since = time.Time{}
}

// resume restarts the clocks at now. A Decision still running although the
// row said suspense — a draft from before is_open, when the open Duel lived in
// memory, or one a migration carried open — has its gap uncounted and its
// duration unknown. An open Duel is never resumed: its clocks run on.
func (g *game) resume(now time.Time) {
	c := &g.doc.Clock
	if g.awaiting() == nil {
		return
	}
	if !c.Since.IsZero() {
		c.Unknown = true
	}
	c.Since = now
}

// flag checks the awaited Side's reserve at now and notes it run out. It
// reports whether that changed the draft.
func (g *game) flag(now time.Time) bool {
	d := g.awaiting()
	if d == nil || g.doc.Cadence == nil || g.doc.Clock.OverTime != 0 {
		return false
	}
	ms, _ := g.elapsed(now)
	if !g.overTime(d.Side, ms) {
		return false
	}
	g.noteOverTime(d.Side)
	return true
}

// ClockState is a Duel's clock as a caller sees it, under a Cadence. A
// running Side's reserve left at instant t is Reserve[side] minus the charge
// of Spent + (t − Awaiting.Since) on a turn that had used Turn.
type ClockState struct {
	Cadence Cadence `json:"cadence"`
	// Reserve is each Side's reserve in milliseconds before the awaited
	// Decision; Turn what the clock's turn used before it, Spent what it took
	// before a suspension.
	Reserve  [2]int64 `json:"reserve"`
	Turn     int64    `json:"turn"`
	Spent    int64    `json:"spent"`
	OverTime int      `json:"overTime,omitempty"`
}
