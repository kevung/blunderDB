package structure

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// A generator of diagrams drawn the way a book draws them: a framed board,
// filled points, dark checkers filled and light ones outlined, the printed
// numbering and pip counts given as the page's text layer. Positions are
// known, so the reading is checked exactly. No book and no image of one is
// in the repository.

// synthPos is a position in each player's own numbering: bottom player
// (dark, home at the bottom right) and top player (light, home at the top
// right); index 0 is the bar.
type synthPos struct {
	bottom, top [25]int
}

func (s synthPos) board() string {
	b := []byte("--------------------------")
	for q := 1; q <= 24; q++ {
		if n := s.bottom[q]; n > 0 {
			b[q] = byte('A' + n - 1)
		}
		if n := s.top[q]; n > 0 {
			b[25-q] = byte('a' + n - 1)
		}
	}
	if s.bottom[0] > 0 {
		b[25] = byte('A' + s.bottom[0] - 1)
	}
	if s.top[0] > 0 {
		b[0] = byte('a' + s.top[0] - 1)
	}
	return string(b)
}

func pipsOfSide(s [25]int) int {
	p := 25 * s[0]
	for q := 1; q <= 24; q++ {
		p += q * s[q]
	}
	return p
}

// randomPos spreads fifteen checkers a side, at most five a point, a point
// held by one side only.
func randomPos(r *rand.Rand) synthPos {
	var s synthPos
	for _, side := range []*[25]int{&s.bottom, &s.top} {
		for n := 0; n < 15; {
			q := 1 + r.Intn(24)
			other := s.top
			if side == &s.top {
				other = s.bottom
			}
			theirs := 25 - q
			if side[q] >= 5 || other[theirs] > 0 {
				continue
			}
			side[q]++
			n++
		}
	}
	return s
}

var startPos = func() synthPos {
	var s synthPos
	for q, n := range map[int]int{24: 2, 13: 5, 8: 3, 6: 5} {
		s.bottom[q], s.top[q] = n, n
	}
	return s
}()

// canvas is a grey page under construction, 0 black to 1 white.
type canvas struct {
	w, h int
	p    []float64
}

func newCanvas(w, h int) *canvas {
	c := &canvas{w: w, h: h, p: make([]float64, w*h)}
	for i := range c.p {
		c.p[i] = 1
	}
	return c
}

func (c *canvas) set(x, y int, v float64) {
	if x >= 0 && y >= 0 && x < c.w && y < c.h {
		c.p[y*c.w+x] = v
	}
}

func (c *canvas) rect(x0, y0, x1, y1 int, v float64) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			c.set(x, y, v)
		}
	}
}

// triangle fills a point of base [x0,x1) at y=base reaching height h
// (negative: upward).
func (c *canvas) triangle(x0, x1, base, h int, v float64) {
	n := h
	if n < 0 {
		n = -n
	}
	for k := 0; k < n; k++ {
		f := float64(k) / float64(n)
		half := (1 - f) * float64(x1-x0) / 2
		cx := float64(x0+x1) / 2
		y := base + k
		if h < 0 {
			y = base - k
		}
		for x := int(math.Ceil(cx - half)); x < int(cx+half); x++ {
			c.set(x, y, v)
		}
	}
}

// disc draws a checker: a black outline, filled with fill.
func (c *canvas) disc(cx, cy, r float64, fill float64) {
	for y := int(cy - r - 2); y <= int(cy+r+2); y++ {
		for x := int(cx - r - 2); x <= int(cx+r+2); x++ {
			d := math.Hypot(float64(x)-cx, float64(y)-cy)
			switch {
			case d <= r-2:
				c.set(x, y, fill)
			case d <= r:
				c.set(x, y, 0)
			}
		}
	}
}

type drawOpts struct {
	pips       bool    // print the pip counts in the text layer
	cubeTop    bool    // draw a cube on the top side, beside the field
	cubeBare   bool    // its face bare; otherwise a mark no figure reads
	p          float64 // point width in pixels
	numbering  bool
	hidePipTop bool
	inkPips    bool // draw the pip counts in the image, figures of inkFont
	inkFont    int
	// inkPipsFar puts the counts that many point widths right of the
	// board instead of left of it, shifted across by inkPipsShift points;
	// inkPipsLetter writes a W before the top count, a B before the bottom
	// one.
	inkPipsFar, inkPipsShift float64
	inkPipsLetter            bool
	// tray draws a box beside the field for each row, its borne-off
	// checkers as thin bars; trayExtra bars more than borne off.
	tray      bool
	trayExtra int
	// trayExtraTop adds bars to the top row's tray alone.
	trayExtraTop int
	// trayCube draws a cube in the top tray's box; trayHollow draws the
	// borne-off checkers as hollow rectangles.
	trayCube, trayHollow bool
	// trayStrokes draws, in the top tray's box, two hairlines across it
	// (a cube's or a label's outline), and in the bottom one three bars
	// floating halfway up, off the tray's outer end.
	trayStrokes bool
	// trayShape draws, in place of the bottom tray's bars, a stray shape that
	// the bars' reader counts as two: "bracket" (an open "[") or "dotted" (a
	// rectangle with dotted sides).
	trayShape string
	// darkFill is the dark checkers' grey, 0.2 when unset.
	darkFill float64
}

// draw puts a diagram on the canvas with its top-left corner at (ox, oy)
// and returns its words.
func (c *canvas) draw(s synthPos, ox, oy int, o drawOpts) []Word {
	p := o.p
	half := int(6 * p)
	bar := int(p)
	t := 3
	H := int(11 * p)
	a := ox + t
	b := a + half
	cc := b + bar
	d := cc + half
	y0, y1 := oy+t, oy+t+H
	// frame and bar
	c.rect(ox, oy, d+t, oy+t, 0)
	c.rect(ox, y1, d+t, y1+t, 0)
	c.rect(ox, oy, a, y1+t, 0)
	c.rect(d, oy, d+t, y1+t, 0)
	c.rect(b, oy, b+2, y1+t, 0)
	c.rect(cc-2, oy, cc, y1+t, 0)
	col := func(j int) (int, int) {
		base := a
		if j >= 6 {
			base = cc
		}
		x0 := base + int(float64(j%6)*p)
		return x0, x0 + int(p)
	}
	th := int(0.42 * float64(H))
	for j := 0; j < 12; j++ {
		x0, x1 := col(j)
		v := 0.45
		if j%2 == 1 {
			v = 0.7
		}
		c.triangle(x0+1, x1-1, y0, th, v)
		c.triangle(x0+1, x1-1, y1-1, -th, 1.2-v)
	}
	r := 0.48 * p
	step := 2 * r
	df := 0.2
	if o.darkFill > 0 {
		df = o.darkFill
	}
	stackAt := func(top bool, j, n int, fill float64) {
		x0, x1 := col(j)
		cx := float64(x0+x1) / 2
		for k := 0; k < n; k++ {
			cy := float64(y0) + r + float64(k)*step
			if !top {
				cy = float64(y1) - r - float64(k)*step
			}
			c.disc(cx, cy, r, fill)
		}
	}
	for q := 1; q <= 24; q++ {
		if n := s.bottom[q]; n > 0 {
			if q <= 12 {
				stackAt(false, 12-q, n, df)
			} else {
				stackAt(true, q-13, n, df)
			}
		}
		if n := s.top[q]; n > 0 {
			if q <= 12 {
				stackAt(true, 12-q, n, 1)
			} else {
				stackAt(false, q-13, n, 1)
			}
		}
	}
	if o.tray {
		x0, x1 := d+t+int(0.4*p), d+t+int(1.2*p)
		mid := (y0 + y1) / 2
		for row, off := range []int{15 - s.top[0], 15 - s.bottom[0]} {
			for q := 1; q <= 24; q++ {
				if row == 0 {
					off -= s.top[q]
				} else {
					off -= s.bottom[q]
				}
			}
			off += o.trayExtra
			if row == 0 {
				off += o.trayExtraTop
			}
			ya, yb := y0, mid-int(0.5*p)
			if row == 1 {
				ya, yb = mid+int(0.5*p), y1
			}
			c.rect(x0, ya, x1, yb, 0)
			c.rect(x0+2, ya+2, x1-2, yb-2, 1)
			if o.trayHollow {
				for k := 0; k < off; k++ {
					y := yb - 12 - k*12
					if row == 0 {
						y = ya + 4 + k*12
					}
					c.rect(x0+4, y, x1-4, y+8, 0)
					c.rect(x0+6, y+2, x1-6, y+6, 1)
				}
				continue
			}
			if o.trayShape != "" && row == 1 {
				ys, ye := yb-17, yb-4
				c.rect(x0+4, ys, x1-4, ys+3, 0)
				c.rect(x0+4, ye-3, x1-4, ye, 0)
				for y := ys + 3; y < ye-3; y++ {
					switch {
					case o.trayShape == "bracket":
						c.rect(x0+4, y, x0+7, y+1, 0)
					case (y-ys)%3 == 1:
						c.rect(x0+4, y, x0+7, y+1, 0)
						c.rect(x1-7, y, x1-4, y+1, 0)
					}
				}
				continue
			}
			for k := 0; k < off; k++ {
				y := yb - 6 - k*7
				if row == 0 {
					y = ya + 4 + k*7
				}
				c.rect(x0+4, y, x1-4, y+3, 0)
			}
			if o.trayStrokes {
				if row == 0 {
					for _, y := range []int{ya + int(0.5*p), ya + int(1.3*p)} {
						c.rect(x0+2, y, x1-2, y+1, 0)
					}
				} else {
					for k := 0; k < 3; k++ {
						y := (ya+yb)/2 + k*7
						c.rect(x0+4, y, x1-4, y+3, 0)
					}
				}
			}
			if o.trayCube && row == 0 {
				side := int(0.9 * p)
				cx := (x0+x1)/2 - side/2
				cy := yb - side - 6
				c.rect(cx, cy, cx+side, cy+side, 0)
				c.rect(cx+2, cy+2, cx+side-2, cy+side-2, 1)
			}
		}
	}
	bx := float64(b+cc) / 2
	for k := 0; k < s.bottom[0]; k++ {
		c.disc(bx, float64(y0)+0.3*float64(H)+float64(k)*step, r, df)
	}
	for k := 0; k < s.top[0]; k++ {
		c.disc(bx, float64(y1)-0.3*float64(H)-float64(k)*step, r, 1)
	}
	if o.cubeTop {
		side := int(0.9 * p)
		x := d + t + int(0.4*p)
		c.rect(x, y0+2, x+side, y0+2+side, 0)
		c.rect(x+2, y0+4, x+side-2, y0+side, 1)
		if !o.cubeBare {
			c.rect(x+side/2-2, y0+2+side/4, x+side/2+2, y0+2+3*side/4, 0)
		}
	}
	var words []Word
	digitW := 0.25 * p
	word := func(text string, cx, cy float64) {
		w := digitW * float64(len(text))
		words = append(words, Word{cx - w/2, cy - 0.2*p, cx + w/2, cy + 0.2*p, text})
	}
	if o.numbering {
		for j := 0; j < 12; j++ {
			x0, x1 := col(j)
			cx := float64(x0+x1) / 2
			word(strconv.Itoa(13+j), cx, float64(oy)-0.5*p)
			word(strconv.Itoa(12-j), cx, float64(y1+t)+0.5*p)
		}
	}
	if o.inkPips {
		h := 0.55 * p
		x := float64(ox) - 1.5*p
		if o.inkPipsFar > 0 {
			x = float64(d+t) + (o.inkPipsFar+o.inkPipsShift)*p
		}
		for k, side := range [2][25]int{s.top, s.bottom} {
			y := float64(y0) + 0.2*float64(H)
			if k == 1 {
				y = float64(y1) - 0.2*float64(H)
			}
			l, _ := c.inkNumber(strconv.Itoa(pipsOfSide(side)), x, y, h, o.inkFont)
			if o.inkPipsLetter {
				c.inkLetter("WB"[k], l-0.6*h, y, h)
			}
		}
	}
	if o.pips {
		if !o.hidePipTop {
			word(strconv.Itoa(pipsOfSide(s.top)), float64(ox)-1.5*p, float64(y0)+0.2*float64(H))
		}
		word(strconv.Itoa(pipsOfSide(s.bottom)), float64(ox)-1.5*p, float64(y1)-0.2*float64(H))
	}
	return words
}

// inkLetter draws a capital W or B, h pixels high, its right at right and
// its middle at cy: a B of two stacked boxes, shaped like an 8, a W of four
// slanted strokes. It gives the letter's left.
func (c *canvas) inkLetter(l byte, right, cy, h float64) float64 {
	sw := max(2, int(0.12*h))
	w := 0.7 * h
	if l == 'W' {
		w = 1.1 * h
	}
	cx := right - w/2
	x0, y0 := int(cx-w/2), int(cy-h/2)
	x1, y1, ym := int(cx+w/2), int(cy+h/2), int(cy)
	switch l {
	case 'B':
		c.rect(x0, y0, x0+sw, y1, 0)
		for _, y := range []int{y0, ym - sw/2, y1 - sw} {
			c.rect(x0, y, x1-sw, y+sw, 0)
		}
		c.rect(x1-sw, y0+sw, x1, ym-sw/2, 0)
		c.rect(x1-sw, ym+sw/2, x1, y1-sw, 0)
	case 'W':
		xs := []float64{cx - w/2, cx - w/4, cx, cx + w/4, cx + w/2}
		for k := 0; k < 4; k++ {
			ya, yb := float64(y0), float64(y1)
			if k%2 == 1 {
				ya, yb = yb, ya
			}
			for ty := 0; ty < int(h); ty++ {
				f := float64(ty) / h
				x := xs[k] + f*(xs[k+1]-xs[k])
				y := ya + f*(yb-ya)
				c.rect(int(x)-sw/2, int(y), int(x)-sw/2+sw, int(y)+1, 0)
			}
		}
	}
	return cx - w/2
}

func (c *canvas) noise(r *rand.Rand, amp float64) {
	for i := range c.p {
		c.p[i] = min(max(c.p[i]+amp*(r.Float64()*2-1), 0), 1)
	}
}

// rotated turns the canvas by deg degrees about its centre (nearest pixel),
// filling with white: a page scanned askew.
func (c *canvas) rotated(deg float64) *canvas {
	out := newCanvas(c.w, c.h)
	a := deg * math.Pi / 180
	cs, sn := math.Cos(a), math.Sin(a)
	cx, cy := float64(c.w)/2, float64(c.h)/2
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			sx := int(math.Round(cs*dx - sn*dy + cx))
			sy := int(math.Round(sn*dx + cs*dy + cy))
			if sx >= 0 && sy >= 0 && sx < c.w && sy < c.h {
				out.p[y*c.w+x] = c.p[sy*c.w+sx]
			}
		}
	}
	return out
}

func (c *canvas) image() *image.Gray {
	g := image.NewGray(image.Rect(0, 0, c.w, c.h))
	for i, v := range c.p {
		g.Pix[i] = uint8(math.Round(v * 255))
	}
	return g
}

// memPages are pages drawn in memory.
type memPages []Page

func (m memPages) Len() int                 { return len(m) }
func (m memPages) Page(i int) (Page, error) { return m[i], nil }

type synthPage struct {
	pos    []synthPos
	opts   []drawOpts
	spread bool // two pages side by side, one diagram on each
	skew   float64
	noise  float64
	// inkGrey prints the page with a faded ink: black becomes this grey,
	// white stays white, every grey between scaled alike.
	inkGrey float64
}

// book draws the pages and returns them with the boards expected, in the
// order a reader meets them.
func book(t *testing.T, pages []synthPage, seed int64) (memPages, []string) {
	t.Helper()
	r := rand.New(rand.NewSource(seed))
	var out memPages
	var want []string
	for _, sp := range pages {
		W, H := 1275, 1650
		if sp.spread {
			W = 2550
		}
		c := newCanvas(W, H)
		var words []Word
		for i, s := range sp.pos {
			o := sp.opts[i]
			ox, oy := 300, 200+i*int(14*o.p)
			if sp.spread {
				ox = 300 + i*1275
				oy = 300
			}
			words = append(words, c.draw(s, ox, oy, o)...)
			want = append(want, s.board())
		}
		if sp.skew != 0 {
			c = c.rotated(sp.skew)
		}
		if sp.noise > 0 {
			c.noise(r, sp.noise)
		}
		if sp.inkGrey > 0 {
			for k, v := range c.p {
				c.p[k] = sp.inkGrey + (1-sp.inkGrey)*v
			}
		}
		out = append(out, Page{Image: c.image(), DPI: 150, Words: words})
	}
	return out, want
}

func plainOpts(p float64) drawOpts { return drawOpts{pips: true, numbering: true, p: p} }

func swapCase(s string) string {
	b := []byte(s)
	for i, c := range b {
		switch {
		case c >= 'a' && c <= 'z':
			b[i] = c - 'a' + 'A'
		case c >= 'A' && c <= 'Z':
			b[i] = c - 'A' + 'a'
		}
	}
	return string(b)
}

func describe(d Diagram) string {
	var s []string
	for _, x := range d.Doubts {
		s = append(s, fmt.Sprintf("%s %d: %s", x.Field, x.Point, x.Reason))
	}
	return fmt.Sprintf("state %d board %s checkers %v pips %v printed %v doubts %v", d.State, d.Board, d.Checkers, d.Pips, d.Printed, s)
}

// TestSyntheticBook reads a small book: two diagrams on most pages, one page
// askew, noise, a spread of two pages; every diagram sure and exact.
func TestSyntheticBook(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	var pages []synthPage
	pages = append(pages, synthPage{pos: []synthPos{startPos, randomPos(r)}, opts: []drawOpts{plainOpts(36), plainOpts(36)}})
	for i := 0; i < 6; i++ {
		sp := synthPage{pos: []synthPos{randomPos(r), randomPos(r)}, opts: []drawOpts{plainOpts(36), plainOpts(36)}}
		switch i {
		case 2:
			sp.skew = 0.8
		case 3:
			sp.noise = 0.06
		case 4:
			sp.skew, sp.noise = -0.6, 0.04
		}
		pages = append(pages, sp)
	}
	pages = append(pages, synthPage{pos: []synthPos{randomPos(r), randomPos(r)}, opts: []drawOpts{plainOpts(40), plainOpts(40)}, spread: true})
	src, want := book(t, pages, 1)
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagrams) != len(want) {
		t.Fatalf("%d diagrams, want %d", len(res.Diagrams), len(want))
	}
	for i, d := range res.Diagrams {
		if d.Board != want[i] || d.State != Confirmed {
			t.Errorf("diagram %d (page %d): want %s sure, got %s", i+1, d.Page+1, want[i], describe(d))
		}
		if !d.Complete {
			t.Errorf("diagram %d: fifteen a side read without doubt, not flagged complete", i+1)
		}
	}
	if !res.Profile.DirectionVoted || res.Profile.HomeSide != 'R' || !res.Profile.BottomIsDark || res.Profile.Numbering != "BR" {
		t.Errorf("profile %+v", res.Profile)
	}
}

// TestSyntheticUnconfirmed: without printed pips nothing is sure; a drawn
// cube off the centre and a missing pip count are named doubts; the
// direction of play, assumed, is named.
func TestSyntheticUnconfirmed(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	var pages []synthPage
	for i := 0; i < 4; i++ {
		o := drawOpts{p: 36, numbering: true}
		o2 := o
		if i == 0 {
			o2.cubeTop = true
		}
		pages = append(pages, synthPage{pos: []synthPos{randomPos(r), randomPos(r)}, opts: []drawOpts{o, o2}})
	}
	src, want := book(t, pages, 2)
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagrams) != len(want) {
		t.Fatalf("%d diagrams, want %d", len(res.Diagrams), len(want))
	}
	for i, d := range res.Diagrams {
		if d.State == Confirmed {
			t.Errorf("diagram %d sure without printed pips: %s", i+1, describe(d))
		}
		// The direction of play is a supposition here: the board may come
		// out with the players exchanged, which the doubt says.
		if d.Board != want[i] && d.Board != swapCase(want[i]) {
			t.Errorf("diagram %d: want %s, got %s", i+1, want[i], describe(d))
		}
		fields := map[string]bool{}
		for _, x := range d.Doubts {
			fields[x.Field] = true
		}
		if !fields["pips"] || !fields["orientation"] {
			t.Errorf("diagram %d: the missing pips and the assumed direction are not named: %s", i+1, describe(d))
		}
		if i == 1 && !fields["cube"] {
			t.Errorf("diagram 2: a cube drawn on the top side is not named: %s", describe(d))
		}
	}
}

// TestSyntheticDeducedStack: a stack hidden in part (drawn shorter than the
// position says) is restored from the printed pips, and is sure only when
// its side then counts fifteen.
func TestSyntheticDeducedStack(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	var pages []synthPage
	for i := 0; i < 6; i++ {
		a, b := randomPos(r), randomPos(r)
		pages = append(pages, synthPage{pos: []synthPos{a, b}, opts: []drawOpts{plainOpts(36), plainOpts(36)}})
	}
	// The last diagram: the bottom player's 6 point holds five; draw only
	// four, print the pips of five.
	last := &pages[len(pages)-1]
	full := synthPos{}
	full.bottom[6], full.bottom[8], full.bottom[13], full.bottom[24] = 5, 3, 5, 2
	full.top = startPos.top
	drawn := full
	drawn.bottom[6] = 4
	last.pos[1] = drawn
	src, _ := book(t, pages, 3)
	// The text layer says five on the 6 point: replace the printed bottom
	// count.
	pg := src[len(src)-1]
	ws := pg.Words
	for i := len(ws) - 1; i >= 0; i-- {
		if ws[i].Text == strconv.Itoa(pipsOfSide(drawn.bottom)) {
			ws[i].Text = strconv.Itoa(pipsOfSide(full.bottom))
			break
		}
	}
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	d := res.Diagrams[len(res.Diagrams)-1]
	if d.Board != full.board() || d.State != Confirmed {
		t.Errorf("want %s sure (deduced), got %s", full.board(), describe(d))
	}
	if !strings.Contains(strings.Join(d.Families, ","), "short high stack") {
		t.Errorf("families %v", d.Families)
	}
}

// TestReadBounds: an image too large, too small or at an absurd resolution
// is refused by name; noise and blank pages are no diagram, and never a
// panic.
func TestReadBounds(t *testing.T) {
	for _, pg := range []Page{
		{Image: image.NewGray(image.Rect(0, 0, 10, 10)), DPI: 150},
		{Image: image.NewGray(image.Rect(0, 0, 20000, 6000)), DPI: 150},
		{Image: image.NewGray(image.Rect(0, 0, 500, 500)), DPI: 5},
		{DPI: 150},
	} {
		if _, err := Read(context.Background(), memPages{pg}); !errors.Is(err, ErrPage) {
			t.Errorf("%v: err %v, want ErrPage", pg.Image, err)
		}
	}
	r := rand.New(rand.NewSource(1))
	noisy := image.NewGray(image.Rect(0, 0, 600, 800))
	for i := range noisy.Pix {
		noisy.Pix[i] = uint8(r.Intn(256))
	}
	blank := image.NewRGBA(image.Rect(0, 0, 600, 800))
	for i := range blank.Pix {
		blank.Pix[i] = 255
	}
	_ = color.White
	for _, img := range []image.Image{noisy, blank} {
		if _, err := Read(context.Background(), memPages{{Image: img, DPI: 150}}); !errors.Is(err, ErrNoDiagram) {
			t.Errorf("err %v, want ErrNoDiagram", err)
		}
	}
}

// unconfirmedBook is the book of TestSyntheticUnconfirmed: no pip count
// printed, the direction of play assumed, a cube drawn on the top side of
// the second diagram.
func unconfirmedBook(t *testing.T) (memPages, []string) {
	r := rand.New(rand.NewSource(11))
	var pages []synthPage
	for i := 0; i < 4; i++ {
		o := drawOpts{p: 36, numbering: true}
		o2 := o
		if i == 0 {
			o2.cubeTop = true
		}
		pages = append(pages, synthPage{pos: []synthPos{randomPos(r), randomPos(r)}, opts: []drawOpts{o, o2}})
	}
	return book(t, pages, 2)
}

// TestSyntheticCompleteWithoutPips: a reading of fifteen checkers a side
// with no printed pips is complete, and its only doubts beside a drawn cube
// are the absent pips and the assumed direction, by their fixed reasons — the
// ones package document accepts on a complete entry.
func TestSyntheticCompleteWithoutPips(t *testing.T) {
	src, _ := unconfirmedBook(t)
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	for i, d := range res.Diagrams {
		if !d.Complete || d.CheckersSure {
			t.Errorf("diagram %d: complete %v, checkers sure %v: %s", i+1, d.Complete, d.CheckersSure, describe(d))
		}
		for _, x := range d.Doubts {
			ok := (x.Field == "pips" && x.Reason == NoPipsReason) || (x.Field == "orientation" && x.Reason == AssumedDirectionReason) || x.Field == "cube"
			if !ok {
				t.Errorf("diagram %d: doubt %+v", i+1, x)
			}
		}
	}
}

// TestWriteSyntheticNoPips writes the unconfirmed book as page images into
// package document's testdata, for its end-to-end tests; run it with
// BLUNDERDB_WRITE_SYNTH=1 after changing the drawing.
func TestWriteSyntheticNoPips(t *testing.T) {
	if os.Getenv("BLUNDERDB_WRITE_SYNTH") == "" {
		t.Skip("set BLUNDERDB_WRITE_SYNTH=1 to write the page images")
	}
	src, _ := unconfirmedBook(t)
	dir := filepath.Join("..", "testdata", "sans-pips")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i, p := range src {
		var b bytes.Buffer
		if err := png.Encode(&b, p.Image); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("p%03d.png", i+1)), b.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
