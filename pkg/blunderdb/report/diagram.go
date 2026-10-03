package report

import (
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

const (
	diagramW  = 520
	diagramH  = 380
	margin    = 14
	barW      = 36
	pointW    = (diagramW - 2*margin - barW) / 12
	pointH    = 150
	checkerR  = pointW / 2
	maxStacks = 5
)

// Diagram draws a position as a standalone SVG document with the default
// palette: the points numbered from the on-roll player's side, the bar, the
// cube and the dice. The desktop app draws its own diagrams in the palette the
// player chose; this one is for the places without a screen.
func Diagram(p *domain.Position) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, diagramW, diagramH, diagramW, diagramH)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#e8dcc0"/>`, diagramW, diagramH)
	fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" fill="none" stroke="#5a4a32" stroke-width="3"/>`, margin, margin, diagramW-2*margin, diagramH-2*margin)

	flip := p.PlayerOnRoll == domain.White
	// x of the left edge of the n-th slot of a half (0..11), the bar skipped.
	slotX := func(i int) int {
		x := margin + i*pointW
		if i >= 6 {
			x += barW
		}
		return x
	}
	// slot returns the column and whether the point is on the top row.
	slot := func(point int) (int, bool) {
		if flip {
			point = 25 - point
		}
		if point >= 13 {
			return point - 13, true
		}
		return 12 - point, false
	}
	for point := 1; point <= 24; point++ {
		col, top := slot(point)
		x := slotX(col)
		fill := "#8a5a3c"
		if (col+btoi(top))%2 == 0 {
			fill = "#c9a46a"
		}
		if top {
			fmt.Fprintf(&b, `<polygon points="%d,%d %d,%d %d,%d" fill="%s"/>`, x, margin, x+pointW, margin, x+pointW/2, margin+pointH, fill)
		} else {
			fmt.Fprintf(&b, `<polygon points="%d,%d %d,%d %d,%d" fill="%s"/>`, x, diagramH-margin, x+pointW, diagramH-margin, x+pointW/2, diagramH-margin-pointH, fill)
		}
		pt := p.Board.Points[point]
		for k := 0; k < pt.Checkers && k < maxStacks; k++ {
			cy := margin + checkerR + k*2*checkerR
			if !top {
				cy = diagramH - margin - checkerR - k*2*checkerR
			}
			b.WriteString(checker(x+pointW/2, cy, pt.Color))
			if k == maxStacks-1 && pt.Checkers > maxStacks {
				fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="13" text-anchor="middle" fill="%s">%d</text>`, x+pointW/2, cy+5, textOn(pt.Color), pt.Checkers)
			}
		}
	}

	// The bar: Points[0] and Points[25].
	barX := margin + 6*pointW + barW/2
	for i, idx := range []int{0, 25} {
		pt := p.Board.Points[idx]
		for k := 0; k < pt.Checkers && k < 3; k++ {
			cy := diagramH/2 - checkerR*(1+2*k) - 4
			if i == 1 {
				cy = diagramH/2 + checkerR*(1+2*k) + 4
			}
			b.WriteString(checker(barX, cy, pt.Color))
		}
	}

	// Cube, on its owner's side (centre when unowned).
	if p.Cube.Value >= 0 && p.Cube.Value < 30 {
		v := 1 << p.Cube.Value // the position stores the exponent
		cy := diagramH / 2
		switch p.Cube.Owner {
		case domain.Black:
			cy = diagramH - margin - 18
		case domain.White:
			cy = margin + 18
		}
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="28" height="28" rx="4" fill="#fff" stroke="#222"/><text x="%d" y="%d" font-size="15" font-weight="bold" text-anchor="middle" fill="#222">%d</text>`, barX-14, cy-14, barX, cy+5, v)
	}

	// Dice, on the right half.
	if p.Dice[0] > 0 && p.Dice[1] > 0 {
		for i, d := range p.Dice {
			x := slotX(8) + i*40
			fmt.Fprintf(&b, `<rect x="%d" y="%d" width="30" height="30" rx="5" fill="#fff" stroke="#222"/><text x="%d" y="%d" font-size="18" font-weight="bold" text-anchor="middle" fill="#222">%d</text>`, x, diagramH/2-15, x+15, diagramH/2+6, d)
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func checker(cx, cy, color int) string {
	fill, stroke := "#f4f4f0", "#444"
	if color == domain.Black {
		fill, stroke = "#222", "#000"
	}
	return fmt.Sprintf(`<circle cx="%d" cy="%d" r="%d" fill="%s" stroke="%s"/>`, cx, cy, checkerR-1, fill, stroke)
}

func textOn(color int) string {
	if color == domain.Black {
		return "#fff"
	}
	return "#222"
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
