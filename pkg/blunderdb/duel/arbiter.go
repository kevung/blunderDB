package duel

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// FormatVersion versions the draft's document, as transcript.FormatVersion
// versions a Transcription's: a change to its shape is a version of the
// document, never a DatabaseVersion migration. Version 2 adds the Cadence and
// the clock, version 3 the single game; an older draft reads as one without
// them, and an older build refuses a version 3 draft rather than play past
// its single game.
const FormatVersion = 3

// The Duel refuses, by name, a Start whose decision the rules do not leave to
// the side on roll. The rule machine reads no DecisionType; the Duel does.
const (
	// RefusedStartPendingDouble: the Start is a double waiting for its answer
	// — a cube decision with the cube already doubled in the middle, or owned
	// by the opponent of the side on roll. A Duel begins before the double.
	RefusedStartPendingDouble transcript.RefusalKind = "start_pending_double"
	// RefusedStartDecision: the Start is a cube decision that carries a roll,
	// or one the rules do not offer the side on roll (the Crawford game, a
	// dead cube, the money ceiling, a game not begun).
	RefusedStartDecision transcript.RefusalKind = "start_decision"
	// RefusedNotAwaited: the Play is not what the Decision awaits, or comes
	// from the other Side.
	RefusedNotAwaited transcript.RefusalKind = "not_awaited"
	// RefusedContributionPending: a Duel with a combined seed has no roll and
	// no Decision before every external Side has contributed.
	RefusedContributionPending transcript.RefusalKind = "contribution_pending"
	// RefusedContribution: a contribution to a Duel without a combined seed,
	// from a delegated Side, a second one from the same Side, one after the
	// first roll, or one empty or over MaxContribution bytes. Every external
	// Side contributes before the first roll, so one after it is a second.
	RefusedContribution transcript.RefusalKind = "contribution"
)

// ErrUnknownSide: a draft names a kind of Side this build cannot resolve.
var ErrUnknownSide = errors.New("unknown kind of side")

// document is what a Duel's draft holds, as JSON. The seed is not in it: the
// store keeps it in a column of its own.
type document struct {
	FormatVersion int               `json:"format_version"`
	Header        transcript.Header `json:"header"`
	Start         *domain.Position  `json:"start,omitempty"`
	// SingleGame: the Duel ends with its first game.
	SingleGame   bool                `json:"single_game,omitempty"`
	Sides        [2]SideSpec         `json:"sides"`
	DiscardAtEnd bool                `json:"discard_at_end,omitempty"`
	Fingerprint  string              `json:"fingerprint"`
	Actions      []transcript.Action `json:"actions"`
	// Rolls is the rank of the next roll drawn from the seed.
	Rolls int `json:"rolls"`
	// Dice is the roll on the board, waiting for DiceSide's play; zero when
	// none is. An opening roll keeps its drawing order: player 1's die first.
	Dice     [2]int `json:"dice,omitempty"`
	DiceSide int    `json:"dice_side,omitempty"`
	// Cadence is the Duel's clock, nil for none; Clock what it, and the
	// durations, keep between two calls (cadence.go).
	Cadence *Cadence `json:"cadence,omitempty"`
	Clock   clock    `json:"clock,omitzero"`
	// CombinedSeed: the rolls come from CombinedSeed over the sealed seed and
	// Contributions, the external Sides' in player order, and nothing is
	// rolled before each external Side has contributed.
	CombinedSeed  bool      `json:"combined_seed,omitempty"`
	Contributions [2]string `json:"contributions,omitzero"`
}

// game is a Duel in memory: its document, its seed, and the rule machine the
// document's Actions lead to, with what each Action derived.
type game struct {
	// now stamps the Decisions handed out and the Plays received.
	now   func() time.Time
	doc   document
	seed  string
	m     transcript.Machine
	infos []transcript.ActionInfo
}

// checkStart refuses what the rule machine cannot see: the Start's decision.
func checkStart(p *domain.Position) *transcript.Refusal {
	if p == nil || p.DecisionType != domain.CubeAction {
		return nil
	}
	if (p.Cube.Owner == domain.None && p.Cube.Value > 0) ||
		((p.PlayerOnRoll == domain.Black || p.PlayerOnRoll == domain.White) && p.Cube.Owner == 1-p.PlayerOnRoll) {
		return &transcript.Refusal{Kind: RefusedStartPendingDouble,
			Detail: "the Start is a double waiting for its answer; a Duel begins before the double"}
	}
	if p.Dice != [2]int{} {
		return &transcript.Refusal{Kind: RefusedStartDecision, Detail: "a cube decision is taken before the roll"}
	}
	return nil
}

// newGame checks the session and the Start and returns the Duel at its Start,
// before the Arbiter has played anything.
func newGame(doc document, seed string) (*game, error) {
	if err := checkStart(doc.Start); err != nil {
		return nil, err
	}
	m, err := transcript.NewMachine(doc.Header, doc.Start)
	if err != nil {
		return nil, err
	}
	if doc.Start != nil && doc.Start.DecisionType == domain.CubeAction && !m.CubeAvailable() {
		return nil, &transcript.Refusal{Kind: RefusedStartDecision,
			Detail: "the rules offer the side on roll no cube decision here"}
	}
	doc.Header = m.Header()
	return &game{doc: doc, seed: seed, m: m}, nil
}

// loadGame replays a draft's Actions. Every one of them was admitted when it
// was played, so a refusal here is a damaged draft.
func loadGame(doc document, seed string) (*game, error) {
	actions := doc.Actions
	doc.Actions = nil
	m, err := transcript.NewMachine(doc.Header, doc.Start)
	if err != nil {
		return nil, fmt.Errorf("duel draft: %w", err)
	}
	g := &game{doc: doc, seed: seed, m: m}
	for i, a := range actions {
		if err := g.apply(a); err != nil {
			return nil, fmt.Errorf("duel draft, action %d: %w", i, err)
		}
	}
	return g, nil
}

// apply plays one Action through the machine and records it.
func (g *game) apply(a transcript.Action) error {
	next, info, err := g.m.Apply(a)
	if err != nil {
		return err
	}
	g.m = next
	g.infos = append(g.infos, info)
	g.doc.Actions = append(g.doc.Actions, a)
	return nil
}

// finished reports whether the Duel's play is over: the match won — a money
// session only once forfeited — or, for a single game, that game ended.
func (g *game) finished() bool {
	if over, _ := g.m.Finished(); over {
		return true
	}
	if !g.doc.SingleGame {
		return false
	}
	games := g.m.Games()
	return len(games) > 0 && games[0].Finished
}

// forfeitedBy is the player (1 or 2) whose forfeit ended the match, 0 none.
func (g *game) forfeitedBy() int {
	if n := len(g.doc.Actions); n > 0 && g.doc.Actions[n-1].Kind == transcript.KindForfeit {
		return g.doc.Actions[n-1].Side + 1
	}
	return 0
}

// forfeit has side give the match up (transcript.KindForfeit): at any moment,
// the Decision awaited or not, since it answers none. The roll on the board,
// if any, is never played.
func (g *game) forfeit(side int) error {
	if side != domain.Black && side != domain.White {
		return &transcript.Refusal{Kind: RefusedNotAwaited, Detail: fmt.Sprintf("the side is %d, not player 1 or 2", side)}
	}
	if err := g.apply(transcript.Action{Side: side, Kind: transcript.KindForfeit}); err != nil {
		return err
	}
	g.doc.Dice, g.doc.DiceSide = [2]int{}, 0
	return nil
}

// awaiting is the Decision a Side owes, or nil when the Arbiter acts next —
// a roll to draw — or the match is over.
func (g *game) awaiting() *Decision {
	if g.finished() || g.timeLost() || g.contributionsPending() {
		return nil
	}
	n := g.m.Next()
	switch {
	case n.Expects == transcript.KindTake:
		return &Decision{Side: n.Side, Kind: DecideAnswer, Position: n.Position}
	case g.doc.Dice != [2]int{}:
		pos := n.Position
		pos.PlayerOnRoll = g.doc.DiceSide
		pos.Dice = [2]int{max(g.doc.Dice[0], g.doc.Dice[1]), min(g.doc.Dice[0], g.doc.Dice[1])}
		pos.DecisionType = domain.CheckerAction
		return &Decision{Side: g.doc.DiceSide, Kind: DecideMove, Position: pos}
	case g.m.CubeAvailable():
		return &Decision{Side: n.Side, Kind: DecideCube, Position: n.Position}
	}
	return nil
}

// roll puts the next roll on the board: the Start's own when it carries one,
// otherwise the seed's at the next rank. A game's opening roll is drawn again
// on a double, and gives the play to the higher die.
func (g *game) roll() error {
	n := g.m.Next()
	if n.Position.Dice != [2]int{} {
		g.doc.Dice, g.doc.DiceSide = n.Position.Dice, n.Side
		return nil
	}
	seed := g.seed
	if g.doc.CombinedSeed {
		var err error
		if seed, err = CombinedSeed(g.seed, g.doc.Contributions); err != nil {
			return err
		}
	}
	for {
		d, err := Roll(seed, g.doc.Rolls)
		if err != nil {
			return err
		}
		g.doc.Rolls++
		if !n.GameStart {
			g.doc.Dice, g.doc.DiceSide = d, n.Side
			return nil
		}
		if d[0] != d[1] {
			g.doc.Dice, g.doc.DiceSide = d, domain.Black
			if d[1] > d[0] {
				g.doc.DiceSide = domain.White
			}
			return nil
		}
	}
}

// receive takes a Side's Play at the instant the Arbiter stamped on it: the
// Decision's duration, the reserve's charge, then the Play. A Side out of
// time under TimeLoseMatch has lost the match before its Play counts.
func (g *game) receive(p Play) error {
	if g.contributionsPending() {
		return &transcript.Refusal{Kind: RefusedContributionPending,
			Detail: "the Duel awaits the external Sides' contributions to its seed before any play"}
	}
	d := g.awaiting()
	ms, known := g.elapsed(p.At)
	if d != nil && p.Side == d.Side && g.overTime(d.Side, ms) && g.doc.Cadence.TimeOut == TimeLoseMatch {
		g.noteOverTime(d.Side)
		return nil
	}
	var dur *int64
	if known {
		dur = &ms
	}
	if err := g.play(p, dur); err != nil {
		return err
	}
	g.account(p.Side, ms, p.Kind)
	return nil
}

// play applies a Side's Play to the Decision it answers; ms is the time the
// Decision took, nil when unknown or when the Arbiter played it alone.
func (g *game) play(p Play, ms *int64) error {
	d := g.awaiting()
	if d == nil {
		return &transcript.Refusal{Kind: RefusedNotAwaited, Detail: "no decision is awaited"}
	}
	if p.Side != d.Side {
		return &transcript.Refusal{Kind: RefusedNotAwaited, Detail: fmt.Sprintf("player %d's decision is awaited", d.Side+1)}
	}
	wrong := func() error {
		return &transcript.Refusal{Kind: RefusedNotAwaited, Detail: fmt.Sprintf("a %s decision is awaited, not %q", d.Kind, p.Kind)}
	}
	switch p.Kind {
	case PlayResign:
		if err := g.apply(transcript.Action{Side: p.Side, Kind: transcript.KindResign, Level: p.Level, DecisionMS: ms}); err != nil {
			return err
		}
		g.doc.Clock.Cube = nil
		// The game is over: a roll left on the board belongs to it.
		g.doc.Dice, g.doc.DiceSide = [2]int{}, 0
		return nil
	case PlayRoll:
		if d.Kind != DecideCube {
			return wrong()
		}
		if err := g.roll(); err != nil {
			return err
		}
		g.doc.Clock.Cube = ms
		return nil
	case PlayDouble:
		if d.Kind != DecideCube {
			return wrong()
		}
		return g.apply(transcript.Action{Side: p.Side, Kind: transcript.KindDouble, DecisionMS: ms})
	case PlayTake, PlayPass:
		if d.Kind != DecideAnswer {
			return wrong()
		}
		kind := transcript.KindTake
		if p.Kind == PlayPass {
			kind = transcript.KindPass
		}
		return g.apply(transcript.Action{Side: p.Side, Kind: kind, DecisionMS: ms})
	case PlayMove:
		if d.Kind != DecideMove {
			return wrong()
		}
		kind := transcript.KindChecker
		if len(p.Steps) == 0 {
			kind = transcript.KindDance
		}
		if err := g.apply(transcript.Action{Side: p.Side, Kind: kind, Dice: g.doc.Dice, Steps: p.Steps,
			DecisionMS: ms, CubeDecisionMS: g.doc.Clock.Cube}); err != nil {
			return err
		}
		g.doc.Dice, g.doc.DiceSide = [2]int{}, 0
		g.doc.Clock.Cube = nil
		return nil
	}
	return &transcript.Refusal{Kind: RefusedNotAwaited, Detail: fmt.Sprintf("%q is no play", p.Kind)}
}

// settle plays what is not a decision — the roll when the cube is not
// available, the dance, the only play — and asks each Side for its Decision
// until one decides outside the Arbiter or the match is over (ADR-0072 rule 9).
func (g *game) settle(ctx context.Context, sides [2]Side) error {
	for !g.finished() && !g.timeLost() && !g.contributionsPending() {
		if err := ctx.Err(); err != nil {
			return err
		}
		d := g.awaiting()
		if d == nil {
			if resigned, err := g.resignUnasked(ctx, sides); err != nil || resigned {
				if err != nil {
					return err
				}
				continue
			}
			if err := g.roll(); err != nil {
				return err
			}
			continue
		}
		if d.Kind == DecideMove {
			var forced *Play
			switch plays := domain.LegalMoves(&d.Position); len(plays) {
			case 0:
				forced = &Play{Side: d.Side, Kind: PlayMove}
			case 1:
				forced = &Play{Side: d.Side, Kind: PlayMove, Steps: plays[0].Steps}
			}
			if forced != nil {
				// No decision: no duration, and the play ends the clock's
				// turn as a played one would.
				if err := g.play(*forced, nil); err != nil {
					return err
				}
				g.doc.Clock.Turn = 0
				continue
			}
		}
		g.startDecision()
		d.Since = g.doc.Clock.Since
		p, ok, err := sides[d.Side].Decide(ctx, *d)
		if err != nil || !ok {
			return err
		}
		p.At = g.clock()
		if err := g.receive(p); err != nil {
			return fmt.Errorf("player %d's side: %w", d.Side+1, err)
		}
	}
	return nil
}

// pendingContributions lists the external Sides a combined seed still awaits
// a contribution from; none without a combined seed.
func (g *game) pendingContributions() []int {
	if !g.doc.CombinedSeed {
		return nil
	}
	var out []int
	for i, s := range g.doc.Sides {
		if s.Kind == SideExternal && g.doc.Contributions[i] == "" {
			out = append(out, i)
		}
	}
	return out
}

func (g *game) contributionsPending() bool { return len(g.pendingContributions()) > 0 }

// contribute records an external Side's contribution to the combined seed:
// once per Side, before the first roll.
func (g *game) contribute(side int, c string) error {
	refuse := func(detail string) error {
		return &transcript.Refusal{Kind: RefusedContribution, Detail: detail}
	}
	switch {
	case !g.doc.CombinedSeed:
		return refuse("the Duel was not created with a combined seed")
	case side != domain.Black && side != domain.White:
		return refuse(fmt.Sprintf("the side is %d, not player 1 or 2", side))
	case g.doc.Sides[side].Kind != SideExternal:
		return refuse("only an external Side contributes")
	case g.doc.Contributions[side] != "":
		return refuse("this Side has already contributed")
	case c == "" || len(c) > MaxContribution || !utf8.ValidString(c):
		return refuse(fmt.Sprintf("a contribution is 1 to %d bytes of text", MaxContribution))
	}
	g.doc.Contributions[side] = c
	return nil
}

// resignUnasked asks the Side on roll, before a roll no Decision is posed on,
// whether it resigns — only a Resigner, which reads it exactly; an external
// Side is never asked. No decision was posed, so none is timed.
func (g *game) resignUnasked(ctx context.Context, sides [2]Side) (bool, error) {
	n := g.m.Next()
	if n.GameStart || n.Expects != transcript.KindChecker || n.Position.Dice != [2]int{} {
		return false, nil
	}
	r, ok := sides[n.Side].(Resigner)
	if !ok {
		return false, nil
	}
	level, err := r.ResignBeforeRoll(ctx, n.Position)
	if err != nil || level == 0 {
		return false, err
	}
	if err := g.apply(transcript.Action{Side: n.Side, Kind: transcript.KindResign, Level: level}); err != nil {
		return false, err
	}
	g.doc.Clock.Cube, g.doc.Clock.Turn = nil, 0
	return true, nil
}

// clock is the game's now, the zero time when it has none.
func (g *game) clock() time.Time {
	if g.now == nil {
		return time.Time{}
	}
	return g.now()
}

// parts is the Match the Duel has become so far. A game begun but not played
// in is left out: it holds no Move, and writing it would only say a game
// began. An unfinished game keeps no winner; nothing is invented for it.
func (g *game) parts() transcript.Parts {
	games := g.m.Games()
	for len(games) > 0 {
		last := games[len(games)-1]
		if last.Finished || last.First >= 0 {
			break
		}
		games = games[:len(games)-1]
	}
	return transcript.BuildPlayed(g.doc.Header, games, g.infos, false)
}
