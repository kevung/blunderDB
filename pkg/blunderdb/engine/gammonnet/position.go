// Package gammonnet is a Go port of the gammonNet evaluator's network and
// feature encoding (https://github.com/kevung/gammonNet, MIT).
//
// It ports two of gammonNet's C modules and nothing else: gn_encoding (the
// 196-feature perspective encoding) and gn_infer (the MLP forward pass and the
// BGNN weight format). Legal-move generation, the match equity table and the
// endgame databases already exist in this repository and are not duplicated
// here — see ADR-0011.
//
// # The boundary
//
// The representation boundary sits at this package's edge, never inside its
// loop. A domain.Position is converted to a Position once, on entry; from there
// on everything stays in this package's narrow representation. Nothing in here
// allocates per evaluation, and nothing in here calls back into domain.
//
// # Attribution
//
// The weights are `strehl-prob5-512-512-256-128`, the work of Alexander Strehl
// (alexstrehl/backgammon-ai-engine, MIT, pinned commit b2750df), redistributed
// by gammonNet v1.0.1. A network keeps its name as long as its weights do not
// change: neither the search around it, nor a quantisation, nor this port makes
// it a new network. "gammonNet" names the configuration, not the weights.
// LICENSE.gammonNet and NOTICE.gammonNet travel with this package.
package gammonnet

import (
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Player identifiers, matching gammonNet's convention.
//
// BEWARE: they are the INVERSE of this repository's. gammonNet has
// GN_WHITE = 0 and GN_BLACK = 1; domain has Black = 0 and White = 1. Every
// conversion below is written out field by field for that reason — the two
// structures must never be assumed to agree.
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
// A mistake in this convention does not crash. It produces five perfectly
// plausible probabilities that are wrong, and it contaminates every measurement
// taken afterwards without ever looking broken.
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
//
// The geometry differs on both axes and neither difference is visible at a
// glance, so both are written out here rather than assumed:
//
//   - Indices are reversed. domain numbers points 1..24 with Black moving
//     high→low (its home is 1..6) and White low→high (its home is 19..24), so
//     White's ace point is domain index 24. gammonNet's index 0 IS White's ace
//     point. Hence gammonNet index i is domain point 24-i.
//   - Player identifiers are swapped, as documented on White/Black above.
//
// It returns an error rather than a silently wrong position when the input does
// not describe a legal board.
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
