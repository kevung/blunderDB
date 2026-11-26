package structure

import (
	"context"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

// ctxOpts is the context drawn beside a synthetic diagram.
type ctxOpts struct {
	cube      int    // the cube's value, 0 for none
	cubePlace string // "top", "middle", "bottom"
	cubeText  bool   // the cube's value also in the text layer
	cubeBare  bool   // the cube's square drawn with a bare face
	dice      [2]int // 0 for none
	diceDark  bool
	diceText  bool     // "to play 41" printed above the diagram
	below     []string // words printed under the diagram: a move, a roll
	score     [3]int
}

// pipLayout is where a die's pips lie, in sixths of its side.
var pipLayout = map[int][][2]float64{
	1: {{3, 3}}, 2: {{1.5, 1.5}, {4.5, 4.5}}, 3: {{1.5, 1.5}, {3, 3}, {4.5, 4.5}},
	4: {{1.5, 1.5}, {4.5, 1.5}, {1.5, 4.5}, {4.5, 4.5}},
	5: {{1.5, 1.5}, {4.5, 1.5}, {3, 3}, {1.5, 4.5}, {4.5, 4.5}},
	6: {{1.5, 1.5}, {4.5, 1.5}, {1.5, 3}, {4.5, 3}, {1.5, 4.5}, {4.5, 4.5}},
}

// drawContext draws the cube, the dice and the score panel of a diagram
// drawn by draw at (ox, oy), and returns their words.
func (c *canvas) drawContext(s synthPos, ox, oy int, p float64, o ctxOpts) []Word {
	t := 3
	a := ox + t
	b := a + int(6*p)
	cc := b + int(p)
	d := cc + int(6*p)
	H := int(11 * p)
	y0, y1 := oy+t, oy+t+H
	var words []Word
	word := func(text string, cx, cy float64) {
		w := 0.25 * p * float64(len(text))
		words = append(words, Word{cx - w/2, cy - 0.2*p, cx + w/2, cy + 0.2*p, text})
	}
	if o.cube > 0 {
		side := int(0.9 * p)
		x := d + t + int(0.4*p)
		y := map[string]int{"top": y0 + 2, "middle": (y0+y1)/2 - side/2, "bottom": y1 - 2 - side}[o.cubePlace]
		c.rect(x, y, x+side, y+side, 0)
		c.rect(x+2, y+2, x+side-2, y+side-2, 1)
		cx, cy := float64(x)+float64(side)/2, float64(y)+float64(side)/2
		h := 0.42 * p
		if !o.cubeBare {
			c.inkNumber(strconv.Itoa(o.cube), cx, cy, h, 0)
		}
		if o.cubeText {
			word(strconv.Itoa(o.cube), cx, cy)
		}
	}
	if o.dice[0] > 0 {
		side := int(0.9 * p)
		for k, v := range o.dice {
			x := cc + int(1.5*p) + k*int(1.2*p)
			y := (y0+y1)/2 - side/2
			ink, paper := 0.0, 1.0
			if o.diceDark {
				ink, paper = 1, 0
			}
			c.rect(x, y, x+side, y+side, 0)
			c.rect(x+2, y+2, x+side-2, y+side-2, 1-ink)
			if o.diceDark {
				c.rect(x, y, x+side, y+side, paper)
			}
			for _, q := range pipLayout[v] {
				c.disc(float64(x)+q[0]*float64(side)/6, float64(y)+q[1]*float64(side)/6, 0.1*float64(side), ink)
			}
		}
		if o.diceText {
			word("Dark", float64(a)+2*p, float64(oy)-1.6*p)
			word("to", float64(a)+3*p, float64(oy)-1.6*p)
			word("play", float64(a)+4*p, float64(oy)-1.6*p)
			word(strconv.Itoa(o.dice[0])+strconv.Itoa(o.dice[1]), float64(a)+5*p, float64(oy)-1.6*p)
		}
	}
	for i, w := range o.below {
		word(w, float64(a)+float64(1+2*i)*p, float64(y1)+1.6*p)
	}
	if o.score[0] > 0 {
		x := float64(d+t) + 3*p
		col := func(cy float64, ws ...string) {
			for i, w := range ws {
				word(w, x+float64(i)*1.5*p, cy)
			}
		}
		col(float64(y0)+0.5*p, "score:", strconv.Itoa(o.score[2]))
		col(float64(y0)+1.5*p, "pip:", strconv.Itoa(pipsOfSide(s.top)))
		col(float64(y0+y1)/2, strconv.Itoa(o.score[0]), "point", "match")
		col(float64(y1)-1.5*p, "pip:", strconv.Itoa(pipsOfSide(s.bottom)))
		col(float64(y1)-0.5*p, "score:", strconv.Itoa(o.score[1]))
	}
	return words
}

// TestSyntheticContext: a cube read in its square and in the text, dice
// counted and confirmed by the text, and a score panel make a context read;
// a cube whose value disagrees with its place, or dice no text confirms, are
// named doubts, never read.
func TestSyntheticContext(t *testing.T) {
	r := rand.New(rand.NewSource(5))
	cases := []ctxOpts{
		{cube: 2, cubePlace: "top", cubeText: true, dice: [2]int{4, 1}, diceDark: true, diceText: true, score: [3]int{7, 1, 3}},
		{cube: 64, cubePlace: "middle", score: [3]int{5, 2, 0}},
		{cube: 8, cubePlace: "middle", score: [3]int{7, 0, 0}},
		{cube: 4, cubePlace: "bottom", dice: [2]int{6, 3}, diceDark: true, score: [3]int{5, 4, 4}},
	}
	var pages []synthPage
	var ctxs []ctxOpts
	for i := 0; i < 8; i++ {
		pages = append(pages, synthPage{pos: []synthPos{randomPos(r), randomPos(r)}, opts: []drawOpts{plainOpts(36), plainOpts(36)}})
		ctxs = append(ctxs, cases[(2*i)%4], cases[(2*i+1)%4])
	}
	src, want := book(t, pages, 3)
	// Redraw each page with its context, the boards unchanged.
	k := 0
	for i, sp := range pages {
		c := newCanvas(1275, 1650)
		var words []Word
		for j, s := range sp.pos {
			ox, oy := 300, 200+j*int(14*36.0)
			words = append(words, c.draw(s, ox, oy, sp.opts[j])...)
			words = append(words, c.drawContext(s, ox, oy, 36, ctxs[k])...)
			k++
		}
		src[i] = Page{Image: c.image(), DPI: 150, Words: words}
	}
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagrams) != len(want) {
		t.Fatalf("%d diagrams, want %d", len(res.Diagrams), len(want))
	}
	fieldsOf := func(d Diagram) map[string]bool {
		f := map[string]bool{}
		for _, x := range d.Doubts {
			f[x.Field] = true
		}
		return f
	}
	for i, d := range res.Diagrams {
		x, o, f := d.Context, ctxs[i], fieldsOf(d)
		if d.Board != want[i] {
			t.Errorf("diagram %d: board %s, want %s", i+1, d.Board, want[i])
			continue
		}
		switch i % 4 {
		case 0:
			if !x.Read || x.Cube != 2 || x.CubeOwner != -1 || x.Dice != o.dice || x.Turn != 1 || x.Score == nil || *x.Score != (Score{Length: 7, Bottom: 1, Top: 3}) {
				t.Errorf("diagram %d: context %+v (score %+v), want read: cube 2 top, 41 bottom on roll, 1-3 to 7; doubts %v", i+1, x, x.Score, d.Doubts)
			}
		case 1:
			if x.Read || !x.CubeRead || x.Cube != 1 || x.CubeOwner != 0 || x.Score == nil || *x.Score != (Score{Length: 5, Bottom: 2}) || f["cube"] {
				t.Errorf("diagram %d: context %+v, want the cube centred and the score 2-0 to 5 read; doubts %v", i+1, x, d.Doubts)
			}
		case 2:
			if x.CubeRead || !f["cube"] {
				t.Errorf("diagram %d: 8 in the middle read as %+v, want a cube doubt; doubts %v", i+1, x, d.Doubts)
			}
		case 3:
			if x.Read || x.DiceRead || !f["dice"] || x.Dice != o.dice || !x.CubeRead || x.CubeOwner != 1 || x.Cube != 4 || !f["score"] {
				t.Errorf("diagram %d: context %+v, want the cube read, dice 63 counted not read, 4-away doubted; doubts %v", i+1, x, d.Doubts)
			}
		}
	}
}

// TestSyntheticContextBareCubeAndRolls: a cube's bare face, wherever it is
// drawn, is the centred cube, read; dice counted in the image are read when
// a roll named beside the diagram ("rolls a 52") or a move written out whose
// every step is one die agrees with them. A move whose step spends two
// dice confirms nothing; one of other dice is a doubt, never a reading.
func TestSyntheticContextBareCubeAndRolls(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	cases := []ctxOpts{
		{cube: 64, cubePlace: "top", cubeBare: true, dice: [2]int{6, 3}, diceDark: true, below: []string{"13/7", "10/7"}},
		{cube: 64, cubePlace: "bottom", cubeBare: true, dice: [2]int{5, 2}, diceDark: true, below: []string{"Dark", "rolls", "a", "52."}},
		{dice: [2]int{4, 4}, diceDark: true, below: []string{"8/4(2)", "6/2(2)"}},
		{dice: [2]int{6, 1}, diceDark: true, below: []string{"13/6"}},
		{dice: [2]int{3, 1}, diceDark: true, below: []string{"6/4", "5/4"}},
		{dice: [2]int{2, 1}, diceDark: true, below: []string{"2/0", "2/1;"}},
	}
	var pages []synthPage
	var ctxs []ctxOpts
	for i := 0; i < 6; i++ {
		pages = append(pages, synthPage{pos: []synthPos{randomPos(r), randomPos(r)}, opts: []drawOpts{plainOpts(36), plainOpts(36)}})
		ctxs = append(ctxs, cases[(2*i)%6], cases[(2*i+1)%6])
	}
	src, want := book(t, pages, 3)
	k := 0
	for i, sp := range pages {
		c := newCanvas(1275, 1650)
		var words []Word
		for j, s := range sp.pos {
			ox, oy := 300, 200+j*int(14*36.0)
			words = append(words, c.draw(s, ox, oy, sp.opts[j])...)
			words = append(words, c.drawContext(s, ox, oy, 36, ctxs[k])...)
			k++
		}
		src[i] = Page{Image: c.image(), DPI: 150, Words: words}
	}
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagrams) != len(want) {
		t.Fatalf("%d diagrams, want %d", len(res.Diagrams), len(want))
	}
	for i, d := range res.Diagrams {
		x, o := d.Context, ctxs[i]
		reasons := map[string]string{}
		for _, y := range d.Doubts {
			reasons[y.Field] += y.Reason
		}
		if d.Board != want[i] {
			t.Errorf("diagram %d: board %s, want %s", i+1, d.Board, want[i])
			continue
		}
		if o.cubeBare && (!x.CubeRead || x.Cube != 1 || x.CubeOwner != 0 || reasons["cube"] != "") {
			t.Errorf("diagram %d: a bare cube on the %s side read as %+v, want centred; doubts %v", i+1, o.cubePlace, x, d.Doubts)
		}
		if x.Dice != o.dice {
			t.Errorf("diagram %d: dice %v counted, want %v", i+1, x.Dice, o.dice)
			continue
		}
		switch i % 6 {
		case 0, 1, 2, 5:
			if !x.DiceRead || reasons["dice"] != "" {
				t.Errorf("diagram %d: dice %v with %v beside not read; doubts %v", i+1, o.dice, o.below, d.Doubts)
			}
		case 3:
			// The move of the diagram above, 4-4, is near enough to be
			// read too: a doubt either way, never a reading.
			if x.DiceRead || reasons["dice"] == "" {
				t.Errorf("diagram %d: dice %v confirmed by a step of two dice; doubts %v", i+1, o.dice, d.Doubts)
			}
		case 4:
			if x.DiceRead || !strings.Contains(reasons["dice"], "the text says otherwise") {
				t.Errorf("diagram %d: dice %v read against a 2-1 move; doubts %v", i+1, o.dice, d.Doubts)
			}
		}
	}
}
