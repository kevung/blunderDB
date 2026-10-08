package transcript

import (
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// RefusalKind names why a [Machine] refuses a session, a Start or an Action. The
// kinds an Action shares with a Replay carry the same value as the [InconsistencyKind]
// a Replay would mark it with: where the Replay keeps the Action and signals it, the
// Machine refuses it (ADR-0072 rule 3).
type RefusalKind string

const (
	// RefusedSession: the header asks for rules a match is not played under here —
	// a length outside 1 to 25, a beaver, Jacoby in match play, a cube ceiling other
	// than 64 at money.
	RefusedSession RefusalKind = "session"

	// The Start is a Position the rules do not allow (ADR-0072 rule 12). It is
	// refused by name, never adjusted.
	RefusedStartTurn  RefusalKind = "start_turn"
	RefusedStartBoard RefusalKind = "start_board"
	RefusedStartCube  RefusalKind = "start_cube"
	RefusedStartScore RefusalKind = "start_score"
	RefusedStartDice  RefusalKind = "start_dice"

	// RefusedOutOfTurn: the Action comes from the side no Action is awaited from, or
	// answers nothing, or comes when no game is running.
	RefusedOutOfTurn RefusalKind = "out_of_turn"
	// RefusedRoll: the dice are no roll (a die outside 1 to 6), or differ from the
	// roll the Start already carries.
	RefusedRoll RefusalKind = "roll"
	// RefusedRecord: an unrecorded play or a declared score — devices of a record,
	// which no player plays.
	RefusedRecord RefusalKind = "record"

	RefusedIllegalMove      = RefusalKind(IllegalMove)
	RefusedDoubleTurn       = RefusalKind(DoubleTurn)
	RefusedImpossibleCube   = RefusalKind(ImpossibleCube)
	RefusedPastEnd          = RefusalKind(PastEnd)
	RefusedInconsistentDice = RefusalKind(InconsistentDice)
)

// Refusal is a reasoned refusal: its Kind for the program, its Detail for the person.
type Refusal struct {
	Kind   RefusalKind `json:"kind"`
	Detail string      `json:"detail"`
}

func (r *Refusal) Error() string { return string(r.Kind) + ": " + r.Detail }

func refuse(kind RefusalKind, format string, args ...any) *Refusal {
	return &Refusal{Kind: kind, Detail: fmt.Sprintf(format, args...)}
}

// maxMatchLength is the longest match the equity table covers (ADR-0072 rule 12).
const maxMatchLength = 25

// moneyMaxCube is the money session's cube ceiling as a log2 exponent: 64.
const moneyMaxCube = 6

// Machine is the rule machine of a match: a state, and an Action that gives the next
// state or a [Refusal]. It walks the same core a Replay walks; the Replay marks what
// the core finds and goes on, the Machine refuses it and stays where it was.
//
// A Machine is a value: Apply returns a new one and never changes its receiver, so a
// caller keeps any earlier state it wants to come back to.
type Machine struct {
	s state
	// applied is the number of Actions the Machine has accepted.
	applied int
	// roll is the roll the Start carries, waiting for the play that uses it; zero
	// once played or when the Start carries none.
	roll [2]int
}

// NewMachine checks the session's rules and the Start, and returns the Machine at
// the Start. A nil start is the default one: the opening position at the start of
// the match, the opening roll to come.
//
// The Start is a Position: its board, cube, side on roll, away score — Crawford
// sentinel included — and its dice, when it carries a roll. Only the first game
// begins there; the next ones begin at the opening position. An opening board with a
// centred cube and no roll is a game not yet begun: the opening roll decides who
// plays first, at the Start's score. A Position's DecisionType is not read: the side
// on roll is to roll, or to double first.
func NewMachine(h Header, start *domain.Position) (Machine, error) {
	if err := checkSession(&h); err != nil {
		return Machine{}, err
	}
	st := newState(h)
	st.jacobyScores = true
	m := Machine{s: *st}
	if start == nil {
		return m, nil
	}
	if err := m.begin(*start); err != nil {
		return Machine{}, err
	}
	return m, nil
}

// checkSession refuses the rules a match is not played under, and fills in the money
// session's ceiling, which is 64 whatever the header leaves unsaid.
func checkSession(h *Header) *Refusal {
	switch {
	case h.MatchLength < 0 || h.MatchLength > maxMatchLength:
		return refuse(RefusedSession, "a match is played to 1 to %d points, not %d", maxMatchLength, h.MatchLength)
	case h.Beaver:
		return refuse(RefusedSession, "the beaver is not played")
	case h.MatchLength > 0 && h.Jacoby:
		return refuse(RefusedSession, "the Jacoby rule belongs to money play")
	case h.MatchLength > 0 && h.MaxCube != 0:
		return refuse(RefusedSession, "a match has no cube ceiling")
	case h.MatchLength == 0 && h.MaxCube != 0 && h.MaxCube != moneyMaxCube:
		return refuse(RefusedSession, "the money cube is capped at 64, not %d", 1<<h.MaxCube)
	}
	if h.MatchLength == 0 {
		h.MaxCube = moneyMaxCube
	}
	return nil
}

// begin sets the Machine at a Start it has checked.
func (m *Machine) begin(p domain.Position) *Refusal {
	if p.PlayerOnRoll != domain.Black && p.PlayerOnRoll != domain.White {
		return refuse(RefusedStartTurn, "the side on roll is %d, not player 1 or 2", p.PlayerOnRoll)
	}
	board, err := startBoard(p.Board)
	if err != nil {
		return err
	}
	points, crawford, crawfordPlayed, err := m.startScore(p.Score)
	if err != nil {
		return err
	}
	if err := m.startCube(p.Cube, crawford); err != nil {
		return err
	}
	opening := board == InitialBoard() && p.Cube == centredCube()
	if err := startDice(p.Dice, opening); err != nil {
		return err
	}

	s := &m.s
	s.points = points
	s.crawfordPlayed = crawfordPlayed
	if opening && p.Dice == [2]int{} {
		// The game has not begun: ensureGame derives its Crawford mention from the
		// score, as it does for every game after the first.
		return nil
	}
	s.games = append(s.games, GameInfo{
		Number:       1,
		InitialScore: points,
		DerivedScore: points,
		Winner:       -1,
		Crawford:     crawford,
		First:        -1,
		Last:         -1,
	})
	s.crawfordPlayed = crawfordPlayed || crawford
	s.board, s.cube, s.turn = board, p.Cube, p.PlayerOnRoll
	s.gameActive = true
	m.roll = p.Dice
	return nil
}

// startBoard checks that a board holds fifteen checkers a side, each where its colour
// may stand, with neither side borne off. An empty point is written uncoloured, which
// is a matter of spelling, not of rules.
func startBoard(b domain.Board) (domain.Board, *Refusal) {
	var count [2]int
	for i := range b.Points {
		pt := &b.Points[i]
		switch {
		case pt.Checkers < 0:
			return b, refuse(RefusedStartBoard, "point %d holds %d checkers", i, pt.Checkers)
		case pt.Checkers == 0:
			pt.Color = domain.None
			continue
		case pt.Color != domain.Black && pt.Color != domain.White:
			return b, refuse(RefusedStartBoard, "the checkers on point %d belong to nobody", i)
		case i == domain.BlackBar && pt.Color != domain.Black,
			i == domain.WhiteBar && pt.Color != domain.White:
			return b, refuse(RefusedStartBoard, "player %d's checkers are on the other side's bar", pt.Color+1)
		}
		count[pt.Color] += pt.Checkers
	}
	for side := range count {
		off := b.Bearoff[side]
		switch {
		case off < 0:
			return b, refuse(RefusedStartBoard, "player %d has borne off %d checkers", side+1, off)
		case off >= domain.CheckersPerPlayer:
			return b, refuse(RefusedStartBoard, "player %d has borne off every checker: the game is over", side+1)
		case count[side]+off != domain.CheckersPerPlayer:
			return b, refuse(RefusedStartBoard, "player %d has %d checkers, not %d",
				side+1, count[side]+off, domain.CheckersPerPlayer)
		}
	}
	return b, nil
}

// startScore reads the away score of a Start into the points each side holds, and
// whether the Start's game is the Crawford game or comes after it. The two
// sentinels say it (CONTEXT.md "Away score"): 1 is the Crawford game, 0 one point
// away after it. In a 1-point match the only score is 1-away each, Crawford by the
// rule's letter, as a Replay derives it.
func (m *Machine) startScore(away [2]int) (points [2]int, crawford, played bool, err *Refusal) {
	L := m.s.header.MatchLength
	if L <= 0 {
		if away != [2]int{domain.Unlimited, domain.Unlimited} {
			return points, false, false, refuse(RefusedStartScore, "a money session has no score")
		}
		return points, false, false, nil
	}
	for side, a := range away {
		if a < 0 || a > L {
			return points, false, false, refuse(RefusedStartScore, "player %d is %d away in a %d-point match", side+1, a, L)
		}
	}
	if L == 1 {
		if away != [2]int{1, 1} {
			return points, false, false, refuse(RefusedStartScore, "a 1-point match is played at 1-away each")
		}
		return points, true, false, nil
	}
	sentinel := func(a int) bool { return a == domain.Crawford || a == domain.PostCrawford }
	if sentinel(away[0]) && sentinel(away[1]) && away != [2]int{domain.PostCrawford, domain.PostCrawford} {
		return points, false, false, refuse(RefusedStartScore,
			"the Crawford game is the first a side plays one point away: it cannot find the other side there")
	}
	for side, a := range away {
		points[side] = L - a
		if a == domain.PostCrawford {
			points[side] = L - 1
			played = true
		}
		if a == domain.Crawford {
			crawford = true
		}
	}
	return points, crawford, played, nil
}

// startCube checks the cube of a Start: a centred cube is worth 1, a turned one has
// an owner, the money ceiling holds, and the Crawford game has no cube.
func (m *Machine) startCube(c domain.Cube, crawford bool) *Refusal {
	switch {
	case c.Owner != domain.None && c.Owner != domain.Black && c.Owner != domain.White:
		return refuse(RefusedStartCube, "the cube's owner is %d, not player 1, player 2 or nobody", c.Owner)
	case c.Value < 0:
		return refuse(RefusedStartCube, "the cube's exponent is %d", c.Value)
	case c.Value == 0 && c.Owner != domain.None:
		return refuse(RefusedStartCube, "a cube at 1 is owned by nobody")
	case c.Value > 0 && c.Owner == domain.None:
		return refuse(RefusedStartCube, "a cube at %d has an owner", cubeValue(c))
	case m.s.header.MaxCube > 0 && c.Value > m.s.header.MaxCube:
		return refuse(RefusedStartCube, "the cube is capped at %d", 1<<m.s.header.MaxCube)
	case c.Value > maxMatchLength:
		return refuse(RefusedStartCube, "the cube's exponent is %d", c.Value)
	case crawford && c.Value != 0:
		return refuse(RefusedStartCube, "the cube is dead in the Crawford game")
	}
	return nil
}

// startDice checks the roll a Start carries: none, or two dice. On the opening board
// it is the opening roll, which no double can be.
func startDice(d [2]int, opening bool) *Refusal {
	if d == [2]int{} {
		return nil
	}
	if !isDie(d[0]) || !isDie(d[1]) {
		return refuse(RefusedStartDice, "%d%d is not a roll", d[0], d[1])
	}
	if opening && d[0] == d[1] {
		return refuse(RefusedStartDice, "no opening roll is a double")
	}
	return nil
}

func isDie(v int) bool { return v >= 1 && v <= 6 }

// Apply plays one Action. It returns the Machine after it and what it derives about
// it, or a [*Refusal] and the receiver unchanged.
func (m Machine) Apply(a Action) (Machine, ActionInfo, error) {
	if err := m.admit(a); err != nil {
		return m, ActionInfo{}, err
	}
	next := m
	next.s = m.s.clone()
	info := next.s.step(m.applied, a)
	if len(info.Inconsistencies) > 0 {
		inc := info.Inconsistencies[0]
		return m, ActionInfo{}, &Refusal{Kind: RefusalKind(inc.Kind), Detail: inc.Detail}
	}
	next.applied++
	next.roll = [2]int{}
	return next, info, nil
}

// admit holds what a Replay cannot see from one Action and the state alone: who is
// awaited, and what a record writes but nobody plays. The rest the core finds.
func (m Machine) admit(a Action) *Refusal {
	s := &m.s
	switch {
	case s.matchOver():
		return refuse(RefusedPastEnd, "the match is already won")
	case a.Kind == KindUnrecorded:
		return refuse(RefusedRecord, "a play is played, not left unrecorded")
	case a.Score != nil:
		return refuse(RefusedRecord, "the score is the one the games give")
	case a.Side != domain.Black && a.Side != domain.White:
		return refuse(RefusedOutOfTurn, "the side is %d, not player 1 or 2", a.Side)
	}
	if a.Kind == KindChecker || a.Kind == KindDance {
		if !isDie(a.Dice[0]) || !isDie(a.Dice[1]) {
			return refuse(RefusedRoll, "%d%d is not a roll", a.Dice[0], a.Dice[1])
		}
		if m.roll != [2]int{} && !sameRoll(m.roll, a.Dice) {
			return refuse(RefusedRoll, "the roll is %d%d, not %d%d", m.roll[0], m.roll[1], a.Dice[0], a.Dice[1])
		}
	}
	if a.Kind == KindForfeit {
		// A match is given up at any moment, by either side, between games too.
		return nil
	}
	if !s.gameActive {
		// The game's first Action: the opening roll names its side, which the core
		// checks; a cube action there the core refuses too. A resignation resigns
		// a game, and none is running.
		if a.Kind == KindResign {
			return refuse(RefusedOutOfTurn, "no game is running")
		}
		return nil
	}
	if a.Kind == KindResign {
		return nil
	}
	if s.pendingDouble >= 0 {
		if a.Kind != KindTake && a.Kind != KindPass {
			return refuse(RefusedOutOfTurn, "player %d's double waits for its answer", s.pendingDouble+1)
		}
		if a.Side == s.pendingDouble {
			return refuse(RefusedOutOfTurn, "player %d does not answer their own double", a.Side+1)
		}
		return nil
	}
	if a.Kind == KindDouble && m.roll != [2]int{} {
		return refuse(RefusedImpossibleCube, "the dice are already rolled")
	}
	if a.Side != s.turn && (a.Kind == KindChecker || a.Kind == KindDance || a.Kind == KindDouble) {
		return refuse(RefusedOutOfTurn, "player %d is on roll", s.turn+1)
	}
	return nil
}

// Next describes the Action the Machine awaits. When the Start carries a roll, the
// Position carries it too: the dice are already on the board, nobody rolls them.
func (m Machine) Next() Next {
	n := m.s.next(nil)
	if m.roll != [2]int{} {
		n.Position.Dice = m.roll
	}
	return n
}

// Score is the points each side holds.
func (m Machine) Score() [2]int { return m.s.points }

// Games is what the Machine derives about each game so far.
func (m Machine) Games() []GameInfo { return append([]GameInfo(nil), m.s.games...) }

// Finished reports whether the match is won, and by whom. A money session is
// only when a side forfeited it.
func (m Machine) Finished() (bool, int) {
	if !m.s.matchOver() {
		return false, -1
	}
	return true, m.s.winner()
}

// Header is the session the Machine plays under, its money ceiling filled in.
func (m Machine) Header() Header { return m.s.header }

// CubeAvailable reports whether the side the Machine awaits a roll from may double
// first — whether its turn begins with a decision. It is false when no roll is
// awaited (no game running, a double waiting for its answer, the Start's roll already
// on the board), in the Crawford game, when the opponent owns the cube, at the money
// ceiling, and when the cube is dead at the score: the doubler already wins the match
// with the cube as it stands, so doubling can only give the opponent the cube.
func (m Machine) CubeAvailable() bool {
	s := &m.s
	if !s.gameActive || s.matchOver() || s.pendingDouble >= 0 || m.roll != [2]int{} {
		return false
	}
	return s.mayDouble(s.turn)
}

// mayDouble reports whether side may double before its roll in the game being
// played: not in the Crawford game, not when the opponent owns the cube, not at the
// money ceiling, and not when the cube is dead at the score.
func (s *state) mayDouble(side int) bool {
	if side != domain.Black && side != domain.White {
		return false
	}
	if n := len(s.games); n > 0 && s.games[n-1].Crawford {
		return false
	}
	if s.cube.Owner != domain.None && s.cube.Owner != side {
		return false
	}
	if s.header.MaxCube > 0 && s.cube.Value >= s.header.MaxCube {
		return false
	}
	if L := s.header.MatchLength; L > 0 && s.points[side]+cubeValue(s.cube) >= L {
		return false
	}
	return true
}
