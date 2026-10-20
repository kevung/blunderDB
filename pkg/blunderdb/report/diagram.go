package report

import (
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

const (
	margin    = 14
	barW      = 36
	cubeCol   = 40 // left strip holding the cube
	pointW    = 38
	checkerR  = pointW / 2
	maxStacks = 5
	pointH    = maxStacks * 2 * checkerR // a full stack reaches the middle gap
	midGap    = 44                       // the dice sit here, clear of every stack
	headerH   = 44
	numH      = 16
	boardH    = 2*pointH + midGap
	boardW    = 12*pointW + barW
	diagramW  = 2*margin + cubeCol + boardW
	diagramH  = margin + headerH + 2*numH + boardH + margin
)

// Diagram draws a position as a standalone SVG document with the default
// palette: a header (the player on roll, ringed, and each side's score as
// points away), the points numbered from the on-roll player's side, the bar,
// the cube in its own strip and the dice in the gap between the stacks. The
// desktop app draws its own diagrams in the palette the player chose; this
// one is for the places without a screen.
func Diagram(p *domain.Position) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, diagramW, diagramH, diagramW, diagramH)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="#e8dcc0"/>`, diagramW, diagramH)

	boardX := margin + cubeCol
	boardTop := margin + headerH + numH
	boardBot := boardTop + boardH
	midY := boardTop + boardH/2
	fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" fill="none" stroke="#5a4a32" stroke-width="3"/>`, boardX, boardTop, boardW, boardH)

	// Header: Black's and White's score, the one on roll ringed.
	hy := margin + headerH/2
	for i, side := range []int{domain.Black, domain.White} {
		cx := boardX + 20 + i*110
		b.WriteString(checker(cx, hy, side))
		if side == p.PlayerOnRoll {
			fmt.Fprintf(&b, `<circle cx="%d" cy="%d" r="%d" fill="none" stroke="#c0392b" stroke-width="3"/>`, cx, hy, checkerR+1)
		}
		fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="15" font-weight="bold" fill="#222">%s</text>`, cx+checkerR+4, hy+5, scoreText(p.Score[side]))
	}

	flip := p.PlayerOnRoll == domain.White
	slotX := func(i int) int {
		x := boardX + i*pointW
		if i >= 6 {
			x += barW
		}
		return x
	}
	// slot returns the column, whether the point is on the top row, and the
	// number the player on roll gives it.
	slot := func(point int) (int, bool, int) {
		if flip {
			point = 25 - point
		}
		if point >= 13 {
			return point - 13, true, point
		}
		return 12 - point, false, point
	}
	for point := 1; point <= 24; point++ {
		col, top, label := slot(point)
		x := slotX(col)
		fill := "#8a5a3c"
		if (col+btoi(top))%2 == 0 {
			fill = "#c9a46a"
		}
		if top {
			fmt.Fprintf(&b, `<polygon points="%d,%d %d,%d %d,%d" fill="%s"/>`, x, boardTop, x+pointW, boardTop, x+pointW/2, boardTop+pointH, fill)
			fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="11" text-anchor="middle" fill="#5a4a32">%d</text>`, x+pointW/2, boardTop-4, label)
		} else {
			fmt.Fprintf(&b, `<polygon points="%d,%d %d,%d %d,%d" fill="%s"/>`, x, boardBot, x+pointW, boardBot, x+pointW/2, boardBot-pointH, fill)
			fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="11" text-anchor="middle" fill="#5a4a32">%d</text>`, x+pointW/2, boardBot+13, label)
		}
		pt := p.Board.Points[point]
		for k := 0; k < pt.Checkers && k < maxStacks; k++ {
			cy := boardTop + checkerR + k*2*checkerR
			if !top {
				cy = boardBot - checkerR - k*2*checkerR
			}
			b.WriteString(checker(x+pointW/2, cy, pt.Color))
			if k == maxStacks-1 && pt.Checkers > maxStacks {
				fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="13" text-anchor="middle" fill="%s">%d</text>`, x+pointW/2, cy+5, textOn(pt.Color), pt.Checkers)
			}
		}
	}

	// The bar: Points[0] and Points[25], stacked outward from the middle gap.
	barX := boardX + 6*pointW + barW/2
	for i, idx := range []int{0, 25} {
		pt := p.Board.Points[idx]
		for k := 0; k < pt.Checkers && k < 3; k++ {
			cy := midY - midGap/2 - checkerR*(1+2*k)
			if i == 1 {
				cy = midY + midGap/2 + checkerR*(1+2*k)
			}
			b.WriteString(checker(barX, cy, pt.Color))
		}
	}

	// Cube, in the left strip: at the owner's end, in the middle when unowned.
	if p.Cube.Value >= 0 && p.Cube.Value < 30 {
		v := 1 << p.Cube.Value // the position stores the exponent
		cy := midY
		switch p.Cube.Owner {
		case domain.Black:
			cy = boardBot - 18
		case domain.White:
			cy = boardTop + 18
		}
		cx := margin + cubeCol/2
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="28" height="28" rx="4" fill="#fff" stroke="#222"/><text x="%d" y="%d" font-size="15" font-weight="bold" text-anchor="middle" fill="#222">%d</text>`, cx-14, cy-14, cx, cy+5, v)
	}

	// Dice, in the middle gap of the right half.
	if p.Dice[0] > 0 && p.Dice[1] > 0 {
		for i, d := range p.Dice {
			x := slotX(8) + i*40
			fmt.Fprintf(&b, `<rect x="%d" y="%d" width="30" height="30" rx="5" fill="#fff" stroke="#222"/><text x="%d" y="%d" font-size="18" font-weight="bold" text-anchor="middle" fill="#222">%d</text>`, x, midY-15, x+15, midY+6, d)
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

// scoreText writes a side's score the way match players do: points away,
// an infinity sign at money play.
func scoreText(away int) string {
	if away == domain.Unlimited {
		return "∞"
	}
	return fmt.Sprintf("-%d", domain.PointsAway(away))
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
