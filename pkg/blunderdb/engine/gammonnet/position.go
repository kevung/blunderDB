// SPDX-License-Identifier: MIT

// Package gammonnet is a Go port of the gammonNet evaluator's network,
// feature encoding, search and cube model (https://github.com/kevung/gammonNet,
// MIT).
//
// It ports four gammonNet C modules (ADR-0011): gn_encoding (the 196-feature
// perspective encoding), gn_infer (the MLP forward pass and BGNN weight
// format), gn_search (the expectiminimax over the 21 rolls) and gn_cube (the
// Janowski cube model). The match equity table is blunderDB's own
// (engine.GnuBGGetME), not a re-ported gn_met.c; see cube.go.
//
// # The boundary
//
// A domain.Position is converted to a Position once, on entry; nothing inside
// allocates per evaluation or calls back into domain.
//
// # Attribution
//
// The weights are `strehl-prob5-512-512-256-128`, by Alexander Strehl
// (alexstrehl/backgammon-ai-engine, MIT, pinned commit b2750df), redistributed
// by gammonNet v1.0.1. "gammonNet" names the configuration, not the weights.
// LICENSE.gammonNet and NOTICE.gammonNet travel with this package.
package gammonnet

import (
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Player identifiers, matching gammonNet's convention.
//
// BEWARE: they are the INVERSE of domain's (Black = 0, White = 1); every
// conversion is written out field by field for that reason.
const (
	White = 0
	Black = 1
)

const (
	// NumPoints is the number of points on the board.
	NumPoints = 24
	// NumCheckers is each player's checker count.
	NumCheckers = 15
)

// Position is gammonNet's board convention, reproduced exactly.
//
//	Points[i] is a SIGNED checker count: positive is that many WHITE checkers,
//	negative that many BLACK checkers, zero empty. A point never holds both.
//
//	Index i denotes point (i+1) for WHITE and point (24-i) for BLACK.
//	Equivalently WHITE bears off towards index 0 and BLACK towards index 23,
//	so index 0 is WHITE's ace point and BLACK's 24 point.
//
//	Turn is the player who acts next.
//
// A mistake in this convention does not crash: it yields plausible, wrong
// probabilities.
type Position struct {
	Points [NumPoints]int8
	Bar    [2]uint8
	Off    [2]uint8
	Turn   uint8
}

// Valid reports whether p is structurally sound: fifteen checkers a side, no
// point over capacity, a known player on turn. The evaluator refuses an invalid
// position rather than approximating one.
func (p *Position) Valid() bool {
	if p.Turn != White && p.Turn != Black {
		return false
	}
	for _, n := range p.Points {
		if n > NumCheckers || n < -NumCheckers {
			return false
		}
	}
	if p.Bar[White] > NumCheckers || p.Bar[Black] > NumCheckers {
		return false
	}
	if p.Off[White] > NumCheckers || p.Off[Black] > NumCheckers {
		return false
	}
	return p.checkerCount(White) == NumCheckers && p.checkerCount(Black) == NumCheckers
}

func (p *Position) checkerCount(player uint8) int {
	total := int(p.Bar[player]) + int(p.Off[player])
	for _, n := range p.Points {
		switch {
		case player == White && n > 0:
			total += int(n)
		case player == Black && n < 0:
			total += int(-n)
		}
	}
	return total
}

// FromDomain converts a blunderDB position into the evaluator's representation.
// Both axes differ: gammonNet index i is domain point 24-i (domain's White
// ace point is 24), and player identifiers are swapped (see White/Black).
// An illegal board is an error, never a silently wrong position.
func FromDomain(p *domain.Position) (Position, error) {
	var out Position

	for i := 0; i < NumPoints; i++ {
		pt := p.Board.Points[NumPoints-i] // gammonNet index i ↔ domain point 24-i
		if pt.Checkers == 0 {
			continue
		}
		if pt.Checkers < 0 || pt.Checkers > NumCheckers {
			return Position{}, fmt.Errorf("gammonnet: point %d holds %d checkers", NumPoints-i, pt.Checkers)
		}
		switch pt.Color {
		case domain.White:
			out.Points[i] = int8(pt.Checkers)
		case domain.Black:
			out.Points[i] = int8(-pt.Checkers)
		default:
			return Position{}, fmt.Errorf("gammonnet: point %d has unknown colour %d", NumPoints-i, pt.Color)
		}
	}

	out.Bar[White] = uint8(p.Board.Points[domain.WhiteBar].Checkers)
	out.Bar[Black] = uint8(p.Board.Points[domain.BlackBar].Checkers)
	out.Off[White] = uint8(p.Board.Bearoff[domain.White])
	out.Off[Black] = uint8(p.Board.Bearoff[domain.Black])

	switch p.PlayerOnRoll {
	case domain.White:
		out.Turn = White
	case domain.Black:
		out.Turn = Black
	default:
		return Position{}, fmt.Errorf("gammonnet: no player on roll (%d)", p.PlayerOnRoll)
	}

	if !out.Valid() {
		return Position{}, fmt.Errorf("gammonnet: position is not structurally valid")
	}
	return out, nil
}

// isOver reports whether a player has borne off all fifteen checkers.
func (p *Position) isOver() bool {
	return p.Off[White] == NumCheckers || p.Off[Black] == NumCheckers
}

// winner returns White, Black, or -1 when the game is not over.
func (p *Position) winner() int {
	switch {
	case p.Off[White] == NumCheckers:
		return White
	case p.Off[Black] == NumCheckers:
		return Black
	}
	return -1
}

// swapTurn flips whose turn it is. Checkers are untouched.
func (p *Position) swapTurn() {
	if p.Turn == White {
		p.Turn = Black
	} else {
		p.Turn = White
	}
}

// gameValue is the stake of a finished game: 1 plain, 2 gammon, 3 backgammon.
// It returns -1 when the game is not over.
//
// The order of the tests is the rule: a loser who has borne off ANY checker
// loses a plain game, whatever else is true. Only then does a checker on the
// bar, or one still sitting in the winner's home board, make it a backgammon.
func gameValue(p *Position) int {
	w := p.winner()
	if w < 0 {
		return -1
	}
	loser := Black
	if w == Black {
		loser = White
	}
	if p.Off[loser] > 0 {
		return 1
	}
	if p.Bar[loser] > 0 {
		return 3
	}
	low := homeLow(uint8(w))
	for i := low; i < low+6; i++ {
		n := p.Points[i]
		if (loser == Black && n < 0) || (loser == White && n > 0) {
			return 3
		}
	}
	return 2
}

// terminalEquity is the money value of a finished position from its own
// Turn's view — in practice the loser's, so negative, but written
// symmetrically.
func terminalEquity(p *Position) float64 {
	stake := gameValue(p)
	if stake < 0 {
		return 0
	}
	if int(p.Turn) == p.winner() {
		return float64(stake)
	}
	return -float64(stake)
}

// terminalValue is terminalEquity's match counterpart, 2×MWC−1 from p.Turn's
// view, falling back to terminalEquity when state is nil — gn_search.c's
// terminal_value. An invalid state values as 0, as the C does.
func terminalValue(p *Position, state *MatchState) float64 {
	if state == nil {
		return terminalEquity(p)
	}
	stake := gameValue(p)
	if stake < 0 {
		return 0
	}
	onRollWins := int(p.Turn) == p.winner()
	mwc, ok := metAfter(*state, stake*state.Cube, onRollWins)
	if !ok || mwc < 0 {
		return 0
	}
	return 2*mwc - 1
}
