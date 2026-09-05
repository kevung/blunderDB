// Package race computes race (bearoff) metrics shared by the GUI, the CLI
// and the serve daemon. It is pure: no SQL, no storage, no Wails.
//
// This package is the single home for per-board race computations; the
// Database wrapper and the server handlers delegate here instead of keeping
// their own copies (ADR-0009, tasks/ts-bearoff). The underlying one-sided
// bearoff distribution lives in package engine (engine.ComputeEPC); this
// package owns the board-level extraction and the typed result contract.
package race

import (
	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// Side holds the Effective Pip Count data for one player.
//
// EPC used to be defined only once every chequer was home, because the only
// one-sided table there was covered six points. A table over p points answers
// for a side whose farthest chequer stands on point ≤ p (ADR-0027 §9), so the
// condition is now the table's width and not the home board. AllInHome stays
// in the payload: the two-sided verdict, and the panel's race zone, still ask
// for a pure bearoff and nothing else.
type Side struct {
	AllInHome    bool `json:"all_in_home"`
	CheckerCount int  `json:"checker_count"`
	// Farthest is the highest point this player still occupies, 0 with no
	// chequers left. 25 stands for a chequer on the bar, which no table covers.
	Farthest int `json:"farthest"`
	// Points is the width of the table that produced EPC, 0 when none did. It
	// is what lets the panel say "exact, OS-08" instead of leaving the reader
	// to assume six.
	Points int               `json:"points,omitempty"`
	EPC    *engine.EPCResult `json:"epc,omitempty"`
}

// EPC holds both players' EPC data.
// Bottom = Black (player index 0), Top = White (player index 1).
type EPC struct {
	Bottom Side `json:"bottom"`
	Top    Side `json:"top"`
}

// ComputeEPC extracts each player's checkers from the full board and computes
// their EPC when the loaded one-sided table is wide enough to cover them.
// Board indices: WhiteBar = 0, points 1..24, BlackBar = 25; a chequer on the
// bar puts the side beyond every table.
func ComputeEPC(b *domain.Board) EPC {
	bottom, _ := computeSide(b, domain.Black)
	top, _ := computeSide(b, domain.White)
	return EPC{Bottom: bottom, Top: top}
}

// homeBoardOf returns the player's six home points, which Evaluate feeds to
// the two-sided lookup. Only meaningful when the side is AllInHome — the
// two-sided table has no wider form.
func homeBoardOf(board []int) [6]int {
	var home [6]int
	copy(home[:], board)
	return home
}

// computeSide also returns the player's board as a one-sided arrangement:
// board[i] = chequers on their (i+1)-point, 24 long. A chequer on the bar
// counts in CheckerCount and sets Farthest to 25, so no table covers the side.
func computeSide(b *domain.Board, color int) (Side, []int) {
	board := make([]int, 24)
	total, allHome, farthest := 0, true, 0
	for i := range b.Points {
		pt := b.Points[i]
		if pt.Color != color || pt.Checkers <= 0 {
			continue
		}
		total += pt.Checkers
		// Black bears off toward point 1 → its own points are 1..24 as
		// numbered; White bears off toward point 24 → its own point n is
		// board point 25-n. The two bars (i = 0 and i = 25) map to 25.
		var own int
		if color == domain.Black {
			own = i
		} else {
			own = 25 - i
		}
		if own < 1 || own > 24 {
			own = 25 // the bar: outside every one-sided table
		}
		if own > farthest {
			farthest = own
		}
		if own > 6 {
			allHome = false
		}
		if own <= 24 {
			board[own-1] += pt.Checkers
		}
	}

	side := Side{AllInHome: allHome, CheckerCount: total, Farthest: farthest}
	if total == 0 || farthest == 0 {
		return side, board
	}
	// The table answers for this side only if it is wide enough. Nothing is
	// extrapolated: a side one point too far simply has no EPC, exactly as a
	// side with a chequer on the 7-point had none before this lot.
	if width := engine.OneSidedPoints(); width >= farthest {
		if r, err := engine.ComputeEPCPoints(board[:width]); err == nil {
			side.EPC = r
			side.Points = width
		}
	}
	return side, board
}
