package duel

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// DecisionKind is what the Arbiter awaits from a Side.
type DecisionKind string

const (
	// DecideCube: the Side on roll may double; a Play "double" or "roll".
	DecideCube DecisionKind = "cube"
	// DecideAnswer: a double waits for its answer; a Play "take" or "pass".
	DecideAnswer DecisionKind = "answer"
	// DecideMove: the dice are on the board; a Play "move".
	DecideMove DecisionKind = "move"
)

// Decision is a decision the Arbiter awaits: from which Side, of what kind, and
// the Position it is taken from — its dice set when the Side is to move. The
// Side the Decision is awaited from may also resign instead.
type Decision struct {
	Side     int             `json:"side"`
	Kind     DecisionKind    `json:"kind"`
	Position domain.Position `json:"position"`
}

// PlayKind is what a Side does.
type PlayKind string

const (
	PlayRoll   PlayKind = "roll"
	PlayDouble PlayKind = "double"
	PlayTake   PlayKind = "take"
	PlayPass   PlayKind = "pass"
	PlayMove   PlayKind = "move"
	PlayResign PlayKind = "resign"
)

// Play is one Side's answer to a Decision. Steps belong to a move, in absolute
// board indices; Level to a resignation (1 single, 2 gammon, 3 backgammon). A
// Play carries no dice: the Arbiter rolls them.
type Play struct {
	Side  int                  `json:"side"`
	Kind  PlayKind             `json:"kind"`
	Steps []domain.CheckerStep `json:"steps,omitempty"`
	Level int                  `json:"level,omitempty"`
}

// Side is one of the two players of a Duel, as the Arbiter sees it: something
// asked for a Decision. The Arbiter treats every Side alike (ADR-0072 rule 2).
//
// Decide returns ok false when the Side decides outside the Arbiter — an
// external Side, whose Play arrives later through the board, the CLI or the
// API; the Duel is then written and waits. A delegated Side (a Bot) answers at
// once, and the Arbiter goes on in the same call. A Side never sees the dice
// to come: the Decision carries the Position and the current roll only.
type Side interface {
	Decide(ctx context.Context, d Decision) (p Play, ok bool, err error)
}

// SideKind names what stands behind a Side, as the draft records it.
type SideKind string

// SideExternal is a Side whose decisions arrive from outside.
const SideExternal SideKind = "external"

// SideSpec is a Side as the draft records it: its kind, and the name the
// Match gives its player.
type SideSpec struct {
	Kind SideKind `json:"kind"`
	Name string   `json:"name,omitempty"`
}

// External is the Side whose decisions come from outside the Arbiter.
type External struct{}

func (External) Decide(context.Context, Decision) (Play, bool, error) { return Play{}, false, nil }

// SideResolver gives the Side a SideSpec stands for.
type SideResolver func(SideSpec) (Side, error)

// ExternalOnly resolves the external kind and refuses every other.
func ExternalOnly(spec SideSpec) (Side, error) {
	if spec.Kind == SideExternal {
		return External{}, nil
	}
	return nil, fmt.Errorf("side %q: %w", spec.Kind, ErrUnknownSide)
}
