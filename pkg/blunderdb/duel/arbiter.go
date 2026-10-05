package duel

import (
	"context"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// FormatVersion versions the draft's document, as transcript.FormatVersion
// versions a Transcription's: a change to its shape is a version of the
// document, never a DatabaseVersion migration.
const FormatVersion = 1

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
)

// ErrUnknownSide: a draft names a kind of Side this build cannot resolve.
var ErrUnknownSide = errors.New("unknown kind of side")

// document is what a Duel's draft holds, as JSON. The seed is not in it: the
// store keeps it in a column of its own.
type document struct {
	FormatVersion int                 `json:"format_version"`
	Header        transcript.Header   `json:"header"`
	Start         *domain.Position    `json:"start,omitempty"`
	Sides         [2]SideSpec         `json:"sides"`
	DiscardAtEnd  bool                `json:"discard_at_end,omitempty"`
	Fingerprint   string              `json:"fingerprint"`
	Actions       []transcript.Action `json:"actions"`
	// Rolls is the rank of the next roll drawn from the seed.
	Rolls int `json:"rolls"`
	// Dice is the roll on the board, waiting for DiceSide's play; zero when
	// none is. An opening roll keeps its drawing order: player 1's die first.
	Dice     [2]int `json:"dice,omitempty"`
	DiceSide int    `json:"dice_side,omitempty"`
}

// game is a Duel in memory: its document, its seed, and the rule machine the
// document's Actions lead to, with what each Action derived.
type game struct {
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

// finished reports whether the match is won. A money session never is.
func (g *game) finished() bool {
	over, _ := g.m.Finished()
	return over
}

// awaiting is the Decision a Side owes, or nil when the Arbiter acts next —
// a roll to draw — or the match is over.
func (g *game) awaiting() *Decision {
	if g.finished() {
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
	for {
		d, err := Roll(g.seed, g.doc.Rolls)
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

// play applies a Side's Play to the Decision it answers.
func (g *game) play(p Play) error {
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
		if err := g.apply(transcript.Action{Side: p.Side, Kind: transcript.KindResign, Level: p.Level}); err != nil {
			return err
		}
		// The game is over: a roll left on the board belongs to it.
		g.doc.Dice, g.doc.DiceSide = [2]int{}, 0
		return nil
	case PlayRoll:
		if d.Kind != DecideCube {
			return wrong()
		}
		return g.roll()
	case PlayDouble:
		if d.Kind != DecideCube {
			return wrong()
		}
		return g.apply(transcript.Action{Side: p.Side, Kind: transcript.KindDouble})
	case PlayTake, PlayPass:
		if d.Kind != DecideAnswer {
			return wrong()
		}
		kind := transcript.KindTake
		if p.Kind == PlayPass {
			kind = transcript.KindPass
		}
		return g.apply(transcript.Action{Side: p.Side, Kind: kind})
	case PlayMove:
		if d.Kind != DecideMove {
			return wrong()
		}
		kind := transcript.KindChecker
		if len(p.Steps) == 0 {
			kind = transcript.KindDance
		}
		if err := g.apply(transcript.Action{Side: p.Side, Kind: kind, Dice: g.doc.Dice, Steps: p.Steps}); err != nil {
			return err
		}
		g.doc.Dice, g.doc.DiceSide = [2]int{}, 0
		return nil
	}
	return &transcript.Refusal{Kind: RefusedNotAwaited, Detail: fmt.Sprintf("%q is no play", p.Kind)}
}

// settle plays what is not a decision — the roll when the cube is not
// available, the dance, the only play — and asks each Side for its Decision
// until one decides outside the Arbiter or the match is over (ADR-0072 rule 9).
func (g *game) settle(ctx context.Context, sides [2]Side) error {
	for !g.finished() {
		if err := ctx.Err(); err != nil {
			return err
		}
		d := g.awaiting()
		if d == nil {
			if err := g.roll(); err != nil {
				return err
			}
			continue
		}
		if d.Kind == DecideMove {
			switch plays := domain.LegalMoves(&d.Position); len(plays) {
			case 0:
				if err := g.play(Play{Side: d.Side, Kind: PlayMove}); err != nil {
					return err
				}
				continue
			case 1:
				if err := g.play(Play{Side: d.Side, Kind: PlayMove, Steps: plays[0].Steps}); err != nil {
					return err
				}
				continue
			}
		}
		p, ok, err := sides[d.Side].Decide(ctx, *d)
		if err != nil || !ok {
			return err
		}
		if err := g.play(p); err != nil {
			return fmt.Errorf("player %d's side: %w", d.Side+1, err)
		}
	}
	return nil
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
