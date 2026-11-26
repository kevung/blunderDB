package structure

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
)

// Context is what a diagram prints of its context — cube, dice, player on
// roll, score — field by field. A value is given only when it was read; it
// is certified ("Read") only when cross-checked:
//
//   - the cube: its value read in its square (figures of the document) or
//     in the text layer, where two sources must agree, and coherent with its
//     place — 64 in the middle is centred, 2 to 64 at the top or the bottom
//     is owned by that side's player. A cube not found is not centred: the
//     field is then left to the document's defaults.
//   - the dice: two dice of one colour whose pips are counted in the image,
//     confirmed by the text layer ("to play 41"); the player on roll by the
//     dice's colour (the class of checkers they match) and their half of the
//     board (the right half for the bottom player), which must agree.
//   - the score: a panel of the text layer giving the match length and two
//     scores, each assigned to a player by the pip count printed beside it,
//     which must be the pips read for that player.
//
// Whatever is drawn or printed and not certified is a named doubt.
type Context struct {
	Cube      int // the cube's value; 0 when not read
	CubeOwner int // 0 centred, 1 the bottom player, -1 the top player
	CubeRead  bool
	Dice      [2]int // the dice counted in the image; zero when not counted
	DiceRead  bool
	Turn      int // 1 the bottom player, -1 the top one; 0 when not known
	TurnRead  bool
	Score     *Score // nil when no score is read
	// Read is every field read and cross-checked: the cube, the dice, the
	// player on roll and the score.
	Read bool
}

// Score is the score a diagram's panel prints.
type Score struct {
	Length      int // the match length
	Bottom, Top int // each player's score
	Crawford    bool
}

var cubeValues = map[int]bool{2: true, 4: true, 8: true, 16: true, 32: true, 64: true}

var (
	reScoreTag = regexp.MustCompile(`(?i)^score:?$`)
	rePlayTag  = regexp.MustCompile(`(?i)^(plays?|rolls?|rolled)$`)
	reDiceWord = regexp.MustCompile(`^\(?([1-6])-?([1-6])\)?[.,:;]?$`)
	rePoint    = regexp.MustCompile(`(?i)^(point|pt\.?|points)$`)
	reMatch    = regexp.MustCompile(`(?i)^match`)
)

// fieldPoint is the width of one point in the page's pixels.
func (rd *boardReading) fieldPoint() float64 {
	f := rd.field
	return ((f[1] - f[0]) + (f[3] - f[2])) / 12
}

// inCubeZone tells whether (x, y), in the page's pixels, lies where a cube
// is drawn: at the board's height, on the bar or beside the field, at most
// two points and a half out.
func (rd *boardReading) inCubeZone(x, y float64) bool {
	f := rd.field
	p := rd.fieldPoint()
	if p <= 0 || y < f[4] || y > f[5] || x < f[0]-2.5*p || x > f[3]+2.5*p {
		return false
	}
	return x < f[0] || x > f[3] || (f[1] <= x && x <= f[2])
}

// placeOf is a cube's place by its height: the top or bottom third of the
// field, or its middle.
func (rd *boardReading) placeOf(y float64) string {
	f := rd.field
	h := f[5] - f[4]
	switch {
	case y < f[4]+0.3*h:
		return "top"
	case y > f[5]-0.3*h:
		return "bottom"
	}
	return "middle"
}

// cubeFigures keeps, once the document's figures are read, the numbers of
// a cube's value read in a square beside the field.
func cubeFigures(rd *boardReading) []printedNumber {
	var out []printedNumber
	for _, n := range rd.numbers {
		if n.boxed && cubeValues[n.val] && rd.inCubeZone(n.cx(), n.cy()) {
			out = append(out, n)
		}
	}
	return out
}

// readContext reads the diagram's context and names what it leaves unread.
func readContext(rd *boardReading, d *Diagram, words []Word, prof Profile) {
	var doubts []capture.Doubt
	add := func(field, family, format string, a ...any) {
		doubts = append(doubts, capture.Doubt{Field: field, Reason: fmt.Sprintf(format, a...)})
		if family != "" {
			d.Families = append(d.Families, family)
		}
	}
	c := &d.Context
	readCube(rd, d, words, prof, c, add)
	readDice(rd, words, prof, c, add)
	readScore(rd, d, words, c, add)
	if len(rd.dice) == 0 && c.CubeRead && c.CubeOwner == -1 {
		c.Turn = -1
		add("turn", "cube owned", "the cube is owned by the top player and no dice are drawn: taken as the top player's cube decision, not read")
	}
	c.Read = c.CubeRead && c.DiceRead && c.TurnRead && c.Score != nil && len(doubts) == 0
	if len(doubts) > 0 && d.State == Confirmed {
		d.State = ToVerify
	}
	d.Doubts = append(d.Doubts, doubts...)
}

func readCube(rd *boardReading, d *Diagram, words []Word, prof Profile, c *Context, add func(string, string, string, ...any)) {
	type found struct {
		val   int
		place string
		src   string
	}
	var fs []found
	for _, n := range rd.cubeFigs {
		fs = append(fs, found{n.val, rd.placeOf(n.cy()), "image"})
	}
	for i, w := range words {
		if !reNumber.MatchString(w.Text) {
			continue
		}
		v, _ := strconv.Atoi(w.Text)
		x, y := (w.X0+w.X1)/2, (w.Y0+w.Y1)/2
		if !cubeValues[v] || !rd.inCubeZone(x, y) {
			continue
		}
		// A pip count is not a cube: tagged, printed on the document's
		// pip side outside the board, or the pips printed here.
		if i > 0 && (rePipTag.MatchString(words[i-1].Text) || reScoreTag.MatchString(words[i-1].Text)) {
			continue
		}
		if (prof.PipPlace == 'G' && x < float64(rd.bbox.x0)) || (prof.PipPlace == 'D' && x > float64(rd.bbox.x1)) {
			continue
		}
		if p := d.Printed; p != nil && (p[0] == v || p[1] == v) {
			continue
		}
		fs = append(fs, found{v, rd.placeOf(y), "text"})
	}
	drawn := map[string]bool{}
	for _, s := range rd.cubes {
		drawn[s.place] = true
	}
	if len(fs) == 0 {
		for _, pl := range []string{"top", "middle", "bottom"} {
			if drawn[pl] && pl != "middle" {
				add("cube", "cube drawn", "a cube is drawn on the %s side: its value is not read, the position says centred", pl)
			}
		}
		return
	}
	for _, f := range fs[1:] {
		if f.val != fs[0].val || f.place != fs[0].place {
			add("cube", "cube drawn", "the cube is read %d (%s, %s) and %d (%s, %s): the readings disagree", fs[0].val, fs[0].place, fs[0].src, f.val, f.place, f.src)
			return
		}
	}
	v, pl := fs[0].val, fs[0].place
	for s := range drawn {
		if s != pl {
			add("cube", "cube drawn", "a cube is drawn on the %s side and %d is read on the %s side", s, v, pl)
			return
		}
	}
	switch {
	case pl == "middle" && v == 64:
		c.Cube, c.CubeOwner, c.CubeRead = 1, 0, true
	case pl == "top":
		c.Cube, c.CubeOwner, c.CubeRead = v, -1, true
	case pl == "bottom":
		c.Cube, c.CubeOwner, c.CubeRead = v, 1, true
	default:
		add("cube", "cube drawn", "the cube reads %d in the middle: a value and a place that disagree", v)
	}
}

func readDice(rd *boardReading, words []Word, prof Profile, c *Context, add func(string, string, string, ...any)) {
	if len(rd.dice) == 0 {
		return
	}
	if len(rd.dice) != 2 {
		add("dice", "dice drawn", "%d dice drawn: not read, the position says no dice", len(rd.dice))
		return
	}
	a, b := rd.dice[0], rd.dice[1]
	if a.pips == 0 || b.pips == 0 || a.dark != b.dark {
		add("dice", "dice drawn", "2 dice drawn and not counted clearly: the position says no dice")
		return
	}
	c.Dice = [2]int{a.pips, b.pips}
	// The player on roll: the colour of the dice, and their half.
	byColour := 1
	if a.dark != prof.BottomIsDark {
		byColour = -1
	}
	mid := (rd.field[1] + rd.field[2]) / 2
	right := a.x0 > mid && b.x0 > mid
	left := a.x1 < mid && b.x1 < mid
	c.Turn = byColour
	switch {
	case (right && byColour == 1) || (left && byColour == -1):
		c.TurnRead = true
	default:
		add("turn", "dice drawn", "the dice's colour and their half of the board do not tell the same player on roll")
	}
	// The dice: confirmed by the text layer beside the diagram.
	X0, Y0, X1, Y1 := float64(rd.bbox.x0), float64(rd.bbox.y0), float64(rd.bbox.x1), float64(rd.bbox.y1)
	p := rd.fieldPoint()
	var said [][2]int
	for i := 1; i < len(words); i++ {
		w := words[i]
		cx, cy := (w.X0+w.X1)/2, (w.Y0+w.Y1)/2
		if cx < X0-8*p || cx > X1+8*p || cy < Y0-4*p || cy > Y1+4*p {
			continue
		}
		if m := reDiceWord.FindStringSubmatch(w.Text); m != nil && rePlayTag.MatchString(words[i-1].Text) {
			x, _ := strconv.Atoi(m[1])
			y, _ := strconv.Atoi(m[2])
			said = append(said, [2]int{x, y})
		}
	}
	same := func(s [2]int) bool {
		return s == c.Dice || s == [2]int{c.Dice[1], c.Dice[0]}
	}
	switch {
	case len(said) == 1 && same(said[0]):
		c.DiceRead = true
	case len(said) == 0:
		add("dice", "dice drawn", "dice %d-%d counted in the image, no text confirms them", c.Dice[0], c.Dice[1])
	default:
		add("dice", "dice drawn", "dice %d-%d counted in the image, the text says otherwise", c.Dice[0], c.Dice[1])
	}
}

// readScore reads the panel a diagram's text layer prints beside it — "score:
// 2  pip: 156  7 point match  pip: 150  score: 4" — each score assigned to
// a player by the pips printed nearest to it.
func readScore(rd *boardReading, d *Diagram, words []Word, c *Context, add func(string, string, string, ...any)) {
	X0, Y0, X1, Y1 := float64(rd.bbox.x0), float64(rd.bbox.y0), float64(rd.bbox.x1), float64(rd.bbox.y1)
	p := rd.fieldPoint()
	type tagged struct{ v, cy float64 }
	var scores, pips []tagged
	var lengths []int
	crawford := false
	for i, w := range words {
		cx, cy := (w.X0+w.X1)/2, (w.Y0+w.Y1)/2
		if cx < X0-8*p || cx > X1+8*p || cy < Y0-2*p || cy > Y1+2*p {
			continue
		}
		if strings.EqualFold(strings.Trim(w.Text, ".,:;()"), "crawford") {
			crawford = true
		}
		if i+1 >= len(words) {
			continue
		}
		nx := words[i+1]
		v, err := strconv.Atoi(nx.Text)
		if err != nil || !reNumber.MatchString(nx.Text) {
			if reNumber.MatchString(w.Text) && rePoint.MatchString(nx.Text) && i+2 < len(words) && reMatch.MatchString(words[i+2].Text) {
				n, _ := strconv.Atoi(w.Text)
				lengths = append(lengths, n)
			}
			continue
		}
		ny := (nx.Y0 + nx.Y1) / 2
		switch {
		case reScoreTag.MatchString(w.Text):
			scores = append(scores, tagged{float64(v), ny})
		case rePipTag.MatchString(w.Text):
			pips = append(pips, tagged{float64(v), ny})
		}
	}
	if len(scores) == 0 && len(lengths) == 0 {
		return
	}
	if len(scores) != 2 || len(lengths) != 1 || len(pips) != 2 {
		add("score", "score", "a score is printed beside the diagram and not read whole (%d scores, %d match lengths, %d pip counts)", len(scores), len(lengths), len(pips))
		return
	}
	s := Score{Length: lengths[0], Crawford: crawford}
	var set [2]bool
	for _, sc := range scores {
		near := pips[0]
		if math.Abs(pips[1].cy-sc.cy) < math.Abs(pips[0].cy-sc.cy) {
			near = pips[1]
		}
		switch {
		case d.Pips[0] != d.Pips[1] && int(near.v) == d.Pips[0]:
			s.Bottom, set[0] = int(sc.v), true
		case d.Pips[0] != d.Pips[1] && int(near.v) == d.Pips[1]:
			s.Top, set[1] = int(sc.v), true
		}
	}
	if !set[0] || !set[1] {
		add("score", "score", "the score is printed, but its pips do not tell which player has which score")
		return
	}
	L := s.Length
	if L < 1 || L > 64 || s.Bottom >= L || s.Top >= L {
		add("score", "score", "printed score %d-%d to %d is not a match score", s.Bottom, s.Top, L)
		return
	}
	if s.Crawford && s.Bottom != L-1 && s.Top != L-1 {
		add("score", "score", "a Crawford game printed with neither player 1-away")
		return
	}
	c.Score = &s
	if !s.Crawford && (s.Bottom == L-1 || s.Top == L-1) {
		add("score", "score", "a player 1-away: the Crawford game and the games after it are not told apart")
	}
}

// missingCubes: in a document that draws its cube — one found beside at
// least a third of its diagrams — a diagram where none is found is not
// centred by default: its cube was missed, which is a named doubt.
func missingCubes(rds []*boardReading, ds []Diagram) {
	found := func(i int) bool { return len(rds[i].cubes) > 0 || ds[i].Context.CubeRead }
	n := 0
	for i := range ds {
		if found(i) {
			n++
		}
	}
	if len(ds) == 0 || 3*n < len(ds) {
		return
	}
	for i := range ds {
		if found(i) {
			continue
		}
		d := &ds[i]
		d.Doubts = append(d.Doubts, capture.Doubt{Field: "cube", Reason: fmt.Sprintf("the document draws its cube (found beside %d diagrams of %d), none is found here: it is not taken as centred", n, len(ds))})
		d.Families = append(d.Families, "cube missed")
		d.Context.Read = false
		if d.State == Confirmed {
			d.State = ToVerify
		}
	}
}
