package duel

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/gammonnet"
)

// ErrUnknownLevel: a Bot is asked to play at a level gammonNet does not name.
var ErrUnknownLevel = errors.New("unknown bot level")

// BotLevels are the levels a Bot plays at: gammonNet's named levels, those of
// the analysis, at full strength (ADR-0072 rule 7). No weakened level.
var BotLevels = []string{"instant", "normal", "thorough"}

// BotName is the name a Bot's Side carries in the Match: the Configuration
// it plays with, the one that will analyse the match.
func BotName(level string) string { return "gammonNet " + level }

// Bot is a delegated Side: gammonNet's stateless playing policy at a named
// level (ADR-0072 rule 6). It answers every Decision at once, so the Arbiter
// goes on in the same call.
type Bot struct {
	level string
}

// NewBot returns the Bot playing at level, refusing a level gammonNet does
// not name rather than defaulting.
func NewBot(level string) (*Bot, error) {
	for _, l := range BotLevels {
		if l == level {
			return &Bot{level: level}, nil
		}
	}
	return nil, fmt.Errorf("bot level %q: %w", level, ErrUnknownLevel)
}

// policies holds idle policies: one owns a 5 MB searcher and is not
// goroutine-safe, so each Decide borrows one rather than building it.
var policies sync.Pool

func borrowPolicy() (*gammonnet.Policy, error) {
	if p, ok := policies.Get().(*gammonnet.Policy); ok {
		return p, nil
	}
	return gammonnet.NewPolicy()
}

// Decide plays the policy's Action for d.
func (b *Bot) Decide(ctx context.Context, d Decision) (Play, bool, error) {
	if err := ctx.Err(); err != nil {
		return Play{}, false, err
	}
	pos := d.Position
	var pending gammonnet.Pending
	switch d.Kind {
	case DecideMove:
		pending = gammonnet.PendingMove
	case DecideCube:
		pending = gammonnet.PendingCube
	case DecideAnswer:
		// The policy reads a double from the doubler's side, on roll before
		// the turn; the Decision is the taker's, with the cube turned.
		pending = gammonnet.PendingTake
		pos.PlayerOnRoll = 1 - d.Side
		pos.Cube.Value--
		pos.Cube.Owner = domain.None
		if pos.Cube.Value > 0 {
			pos.Cube.Owner = pos.PlayerOnRoll
		}
	default:
		return Play{}, false, fmt.Errorf("bot: no policy for a %q decision", d.Kind)
	}
	in, err := gammonnet.PolicyDecisionFromDomain(&pos, pending, 0)
	if err != nil {
		return Play{}, false, fmt.Errorf("bot: %w", err)
	}
	policy, err := borrowPolicy()
	if err != nil {
		return Play{}, false, err
	}
	action, err := policy.Decide(b.level, &in)
	policies.Put(policy)
	if err != nil {
		return Play{}, false, fmt.Errorf("bot: %w", err)
	}

	play := Play{Side: d.Side}
	switch action.Kind {
	case gammonnet.ActionMove:
		play.Kind = PlayMove
		if play.Steps, err = gammonnet.PolicyPlaySteps(&pos, &action.Play); err != nil {
			return Play{}, false, fmt.Errorf("bot: %w", err)
		}
	case gammonnet.ActionRoll:
		play.Kind = PlayRoll
	case gammonnet.ActionDouble:
		play.Kind = PlayDouble
	case gammonnet.ActionResign:
		play.Kind, play.Level = PlayResign, action.ResignValue
	case gammonnet.ActionTake:
		play.Kind = PlayTake
	case gammonnet.ActionPass:
		play.Kind = PlayPass
	default:
		return Play{}, false, fmt.Errorf("bot: action %d answers no %q decision", action.Kind, d.Kind)
	}
	return play, true, nil
}
