package duel

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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
	// Since is when the Arbiter handed the Decision to the Side: the instant
	// a Cadence starts the Side's clock from (ADR-0073). On a resumed Duel it
	// is the instant of the resumption, the clocks having stood still.
	Since time.Time `json:"since"`
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
	// At is when the Arbiter received the Play — the instant a Cadence
	// stops the Side's clock at. The Arbiter stamps it; a Side's own value
	// is overwritten.
	At time.Time `json:"-"`
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

// Resigner is a Side that can tell, before a roll the cube decision does not
// precede, whether it resigns and for how much (0: it plays on). The Arbiter
// asks it there; it asks no other Side, so nothing changes for them.
type Resigner interface {
	ResignBeforeRoll(ctx context.Context, pos domain.Position) (level int, err error)
}

// SideKind names what stands behind a Side, as the draft records it.
type SideKind string

const (
	// SideExternal is a Side whose decisions arrive from outside.
	SideExternal SideKind = "external"
	// SideBot is a Side delegated to a Bot, at its Level.
	SideBot SideKind = "bot"
)

// SideSpec is a Side as the draft records it: its kind, and the name the
// Match gives its player. A Bot's Level is a named level of gammonNet; its
// name is its Configuration's (BotName), whatever was asked. An external Side
// may declare the Bot that plays behind it (Declared); without a name of its
// own, it is then named as a delegated Side would be.
type SideSpec struct {
	Kind     SideKind     `json:"kind"`
	Name     string       `json:"name,omitempty"`
	Level    string       `json:"level,omitempty"`
	Declared *DeclaredBot `json:"declared,omitempty"`
}

// DeclaredBot is a Bot an external Side says plays behind it: a client that
// runs gammonNet itself — in a webview, a script, an agent. The Arbiter
// records it in the Match's origin as declared, never attested: it sees
// nothing of what plays behind an external Side and authenticates no one
// (ADR-0005), and it treats the Side as any external one.
type DeclaredBot struct {
	// Configuration is the Bot's Configuration, a level name such as
	// "normal"; Engine the gammonNet tag it was built from.
	Configuration string `json:"configuration"`
	Engine        string `json:"engine"`
}

// maxDeclared bounds each text of a declaration, in bytes.
const maxDeclared = 64

// check refuses a declaration with an empty or oversized text.
func (b DeclaredBot) check() error {
	for _, f := range [...]struct{ name, v string }{{"configuration", b.Configuration}, {"engine", b.Engine}} {
		if strings.TrimSpace(f.v) == "" || len(f.v) > maxDeclared || !utf8.ValidString(f.v) {
			return fmt.Errorf("declared bot: %s is 1 to %d bytes of text: %w", f.name, maxDeclared, storage.ErrInvalid)
		}
	}
	return nil
}

// External is the Side whose decisions come from outside the Arbiter.
type External struct{}

func (External) Decide(context.Context, Decision) (Play, bool, error) { return Play{}, false, nil }

// SideResolver gives the Side a SideSpec stands for.
type SideResolver func(SideSpec) (Side, error)

// Resolve resolves the external kind and the Bot, and refuses every other.
func Resolve(spec SideSpec) (Side, error) {
	if spec.Kind == SideBot {
		return NewBot(spec.Level)
	}
	return ExternalOnly(spec)
}

// ExternalOnly resolves the external kind and refuses every other.
func ExternalOnly(spec SideSpec) (Side, error) {
	if spec.Kind == SideExternal {
		return External{}, nil
	}
	return nil, fmt.Errorf("side %q: %w", spec.Kind, ErrUnknownSide)
}
