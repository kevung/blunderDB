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

// Side holds the Effective Pip Count data for one player. EPC is only
// defined once every checker is in the player's home board (bearing-off
// zone); otherwise EPC is nil but CheckerCount is still meaningful.
type Side struct {
	AllInHome    bool              `json:"all_in_home"`
	CheckerCount int               `json:"checker_count"`
	EPC          *engine.EPCResult `json:"epc,omitempty"`
}

// EPC holds both players' EPC data.
// Bottom = Black (player index 0), Top = White (player index 1).
type EPC struct {
	Bottom Side `json:"bottom"`
	Top    Side `json:"top"`
}

// ComputeEPC extracts each player's home-board checkers from the full board
// and computes their EPC when the whole army is home. Board indices:
// WhiteBar = 0, points 1..24, BlackBar = 25; both bars count against
// AllInHome.
func ComputeEPC(b *domain.Board) EPC {
	return EPC{
		Bottom: computeSide(b, domain.Black),
		Top:    computeSide(b, domain.White),
	}
}

func computeSide(b *domain.Board, color int) Side {
	var home [6]int // home[i] = checkers on this player's (i+1)-point
	total, allHome := 0, true
	for i := range b.Points {
		pt := b.Points[i]
		if pt.Color != color || pt.Checkers <= 0 {
			continue
		}
		total += pt.Checkers
		// Black bears off toward point 1 → home points are 1..6.
		// White bears off toward point 24 → home points are 24..19.
		// Bar checkers (i = 0 or 25) always fall outside the 1..6 range.
		var homeIdx int
		if color == domain.Black {
			homeIdx = i
		} else {
			homeIdx = 25 - i
		}
		if homeIdx < 1 || homeIdx > 6 {
			allHome = false
			continue
		}
		home[homeIdx-1] += pt.Checkers
	}
	side := Side{AllInHome: allHome, CheckerCount: total}
	if allHome && total > 0 {
		if r, err := engine.ComputeEPC(home); err == nil {
			side.EPC = r
		}
	}
	return side
}
