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
//     is owned by that side's player. A cube drawn alone with a bare face
//     is centred where the document draws it so beside three diagrams and
//     never prints a value at that place: an owned cube shows its value,
//     and neither its square nor the text layer prints one. A cube not found is
//     not centred: the field is then left to the document's defaults.
//   - the dice: two dice of one colour whose pips are counted in the image,
//     confirmed by the text layer beside the diagram: a roll named ("to play
//     41", "rolls a 65") or a move written out whose steps are those dice
//     ("2/0 2/1"). The count alone, however clear, is one source; the
//     player on roll by the dice's colour (the class of checkers they match)
//     and their half of the board (the right half for the bottom player),
//     which must agree. A diagram that draws no dice and whose caption
//     names them ("Black to play 53", in the text layer or read in the
//     image) is given that roll and player, each a named doubt: one source.
//   - the score: a panel of the text layer giving the match length and two
//     scores, each assigned to a player by the pip count printed beside it,
//     which must be the pips read for that player; equal scores need no
//     assignment.
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
	reScoreTag = regexp.MustCompile(`(?i)^score[:;]?$`)
	// reGlued is a tag glued to its number by an OCR layer: "pip:9".
	reGlued    = regexp.MustCompile(`(?i)^(score|pips?)[:;]([0-9]{1,3})$`)
	rePlayTag  = regexp.MustCompile(`(?i)^(plays?|rolls?|rolled)$`)
	reDiceWord = regexp.MustCompile(`^\(?([1-6])-?([1-6])\)?[.,:;?!]?$`)
	rePoint    = regexp.MustCompile(`(?i)^(point|pt\.?|points)$`)
	reMatch    = regexp.MustCompile(`(?i)^match`)
	reArticle  = regexp.MustCompile(`(?i)^an?$`)
	// reMoveWord is one part of a move's notation: "13/7", "bar/22*",
	// "6/off", "13/9/5", "8/5(2)".
	reMoveWord = regexp.MustCompile(`(?i)^\(?((?:bar|\d{1,2})(?:/(?:\d{1,2}|off)\*?)+)(?:\(([1-4])\))?\)?[.,:;]?$`)
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
// a cube's value read in a square beside the field, and every number
// printed inside a cube drawn, read or not, to name why its value is not.
func cubeFigures(rd *boardReading) []printedNumber {
	var out []printedNumber
	for _, n := range rd.numbers {
		if !rd.inCubeZone(n.cx(), n.cy()) {
			continue
		}
		if (n.boxed && cubeValues[n.val]) || squareOf(rd.cubes, n) >= 0 {
			out = append(out, n)
		}
	}
	return out
}

// squareOf is the index of the square holding the number's centre, or -1.
func squareOf(sqs []square, n printedNumber) int {
	for i, s := range sqs {
		if s.x0 <= n.cx() && n.cx() <= s.x1 && s.y0 <= n.cy() && n.cy() <= s.y1 {
			return i
		}
	}
	return -1
}

// unreadCube says why the value of a cube drawn, its face not bare, is not
// read: the figures found in it, or their absence.
func unreadCube(rd *boardReading, k int) string {
	for _, n := range rd.cubeFigs {
		if squareOf(rd.cubes, n) != k {
			continue
		}
		if n.val < 0 {
			return "the figures printed in it are not recognised"
		}
		if !cubeValues[n.val] {
			return fmt.Sprintf("%d is read in it, not a cube's value", n.val)
		}
	}
	return "no figure of the document's is found in it (printed light on dark, or smaller)"
}

// readContext reads the diagram's context and names what it leaves unread.
func readContext(rd *boardReading, d *Diagram, words []Word, prof Profile, env contextEnv) {
	var doubts []capture.Doubt
	add := func(field, family, format string, a ...any) {
		doubts = append(doubts, capture.Doubt{Field: field, Reason: fmt.Sprintf(format, a...)})
		if family != "" {
			d.Families = append(d.Families, family)
		}
	}
	c := &d.Context
	readCube(rd, d, words, prof, env, c, add)
	readDice(rd, words, prof, env, c, add)
	readScore(rd, d, words, env, c, add)
	if len(rd.dice) == 0 && c.Turn == 0 && c.CubeRead && c.CubeOwner == -1 {
		c.Turn = -1
		add("turn", "cube owned", "the cube is owned by the top player and no dice are drawn: taken as the top player's cube decision, not read")
	}
	c.Read = c.CubeRead && c.DiceRead && c.TurnRead && c.Score != nil && len(doubts) == 0
	if len(doubts) > 0 && d.State == Confirmed {
		d.State = ToVerify
	}
	d.Doubts = append(d.Doubts, doubts...)
}

func readCube(rd *boardReading, d *Diagram, words []Word, prof Profile, env contextEnv, c *Context, add func(string, string, string, ...any)) {
	type found struct {
		val   int
		place string
		src   string
	}
	var fs []found
	for _, n := range rd.cubeFigs {
		if n.boxed && cubeValues[n.val] {
			fs = append(fs, found{n.val, rd.placeOf(n.cy()), "image"})
		}
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
		// A bare face, alone, is the centred cube: an owned cube always
		// shows its value, and neither the square nor the text layer
		// prints one. One bare square could be a figure the scan lost, so
		// it is read only where the document draws it so, at that place,
		// beside at least two other diagrams.
		if len(rd.cubes) == 1 && rd.cubes[0].blank {
			k := rd.squareKey(rd.cubes[0])
			if n := env.valued[k]; n > 0 {
				add("cube", "cube drawn", "a cube with a bare face is drawn on the %s side, where %d other diagrams print a value: its figure may be lost, not read; the position says centred", rd.cubes[0].place, n)
				return
			}
			if env.bare[k] >= 3 {
				c.Cube, c.CubeOwner, c.CubeRead = 1, 0, true
				return
			}
			add("cube", "cube drawn", "a cube with a bare face is drawn on the %s side, a drawing the document repeats beside %d other diagrams only: not read, the position says centred", rd.cubes[0].place, max(0, env.bare[k]-1))
			return
		}
		bare := 0
		for _, s := range rd.cubes {
			if s.blank {
				bare++
			}
		}
		if bare > 0 {
			add("cube", "cube drawn", "%d cubes are drawn, %d with a bare face: not read, the position says centred", len(rd.cubes), bare)
			return
		}
		said := map[string]bool{}
		for k, s := range rd.cubes {
			if s.place != "middle" && !said[s.place] {
				said[s.place] = true
				add("cube", "cube drawn", "a cube is drawn on the %s side: its value is not read, %s; the position says centred", s.place, unreadCube(rd, k))
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

func readDice(rd *boardReading, words []Word, prof Profile, env contextEnv, c *Context, add func(string, string, string, ...any)) {
	near := rd.nearWord(env)
	if len(rd.dice) == 0 {
		captionDice(words, near, prof, c, add)
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
	said := textRolls(words, near)
	same := func(s [2]int) bool {
		return s == c.Dice || s == [2]int{c.Dice[1], c.Dice[0]}
	}
	// One citation agreeing suffices when the text names one roll; when it
	// names several, the counted one must be cited twice.
	agree, distinct := 0, map[[2]int]bool{}
	for _, s := range said {
		if same(s) {
			agree++
		}
		if s[0] > s[1] {
			s = [2]int{s[1], s[0]}
		}
		distinct[s] = true
	}
	switch {
	case agree >= 2 || (agree == 1 && len(distinct) == 1):
		c.DiceRead = true
	case len(said) == 0:
		add("dice", "dice drawn", "dice %d-%d counted in the image, no text confirms them", c.Dice[0], c.Dice[1])
	case agree == 1:
		add("dice", "dice drawn", "dice %d-%d counted in the image, cited once beside other rolls: not confirmed", c.Dice[0], c.Dice[1])
	default:
		add("dice", "dice drawn", "dice %d-%d counted in the image, the text says otherwise", c.Dice[0], c.Dice[1])
	}
}

// nearWord tells whether a word is printed beside the diagram: in its box
// widened by eight point widths across and four along, and nearer its field
// than any other diagram's of the page.
func (rd *boardReading) nearWord(env contextEnv) func(Word) bool {
	X0, Y0, X1, Y1 := float64(rd.bbox.x0), float64(rd.bbox.y0), float64(rd.bbox.x1), float64(rd.bbox.y1)
	p := rd.fieldPoint()
	return func(w Word) bool {
		cx, cy := (w.X0+w.X1)/2, (w.Y0+w.Y1)/2
		if cx < X0-8*p || cx > X1+8*p || cy < Y0-4*p || cy > Y1+4*p {
			return false
		}
		return !env.nearerOther(rd, cx, cy)
	}
}

// captionDice reads the roll a caption gives a diagram that draws no dice —
// "Black to play 53", "White rolls 41": the dice and the player on roll,
// by the colour named. The caption is then the only source of either: both
// are given and named doubts, never read. Captions beside the diagram that
// disagree give nothing but the doubt.
func captionDice(words []Word, near func(Word) bool, prof Profile, c *Context, add func(string, string, string, ...any)) {
	said := captionRolls(words, near)
	if len(said) == 0 {
		return
	}
	first := said[0]
	for _, s := range said[1:] {
		if s.colour != first.colour || (s.roll != first.roll && s.roll != [2]int{first.roll[1], first.roll[0]}) {
			add("dice", "dice in caption", "the text beside the diagram names %d different rolls and no dice are drawn: not read", len(said))
			return
		}
	}
	c.Turn = first.colour
	if !prof.BottomIsDark {
		c.Turn = -first.colour
	}
	if first.roll[0] == 0 {
		add("dice", "dice in caption", "a roll is named beside the diagram and not read, none drawn")
	} else {
		c.Dice = first.roll
		add("dice", "dice in caption", "dice %d-%d named beside the diagram, none drawn: one source, not confirmed", c.Dice[0], c.Dice[1])
	}
	add("turn", "dice in caption", "the player on roll named beside the diagram by colour only: one source, not confirmed")
}

// captionRoll is a roll a caption names, with the class of checkers on roll
// (1 the darker, -1 the lighter).
type captionRoll struct {
	roll   [2]int
	colour int
}

// captionRolls are the rolls named with their player beside a diagram: a
// colour word opening a caption, then "to play", "plays", "rolls" or
// "rolled" (an article allowed), then the roll ending its line — zero when the word in its place is not read
// ("?", in the image).
func captionRolls(words []Word, near func(Word) bool) []captionRoll {
	var out []captionRoll
	for i := 0; i+2 < len(words); i++ {
		col := colourOf(words[i].Text)
		if col == 0 || !near(words[i]) {
			continue
		}
		// A caption states the roll: its colour word opens the line or
		// follows a number or the end of a sentence, and its roll ends
		// the line; "if White rolls 15 and plays" is no caption.
		if i > 0 && sameLine(words[i-1], words[i]) && !opensCaption(words[i-1].Text) {
			continue
		}
		j := i + 1
		if strings.EqualFold(words[j].Text, "to") {
			if !strings.EqualFold(strings.Trim(words[j+1].Text, ".,:;"), "play") {
				continue
			}
			j++
		} else if !rePlayTag.MatchString(words[j].Text) {
			continue
		}
		j++
		if j < len(words) && reArticle.MatchString(words[j].Text) {
			j++
		}
		if j >= len(words) || !near(words[j]) {
			continue
		}
		if j+1 < len(words) && sameLine(words[j], words[j+1]) {
			continue
		}
		m := reDiceWord.FindStringSubmatch(words[j].Text)
		switch {
		case m != nil:
			x, _ := strconv.Atoi(m[1])
			y, _ := strconv.Atoi(m[2])
			out = append(out, captionRoll{[2]int{x, y}, col})
		case words[j].Text == "?":
			out = append(out, captionRoll{colour: col})
		}
	}
	return out
}

// sameLine tells whether two words stand on one line: their heights
// overlap by half the smaller.
func sameLine(a, b Word) bool {
	return min(a.Y1, b.Y1)-max(a.Y0, b.Y0) >= 0.5*min(a.Y1-a.Y0, b.Y1-b.Y0)
}

// opensCaption tells whether a caption may follow the word on its line: a
// number ("Problem 66:"), a sentence's end, or a word of the image not read
// ("?": a caption's number in a figure the reader does not know).
func opensCaption(s string) bool {
	return s == "?" || reNumber.MatchString(strings.TrimRight(s, ".:")) || strings.HasSuffix(s, ":") || strings.HasSuffix(s, ".")
}

// textRolls are the rolls the text layer names beside a diagram (words for
// which near holds), each a second source for the dice counted in the
// image: a roll tagged by a verb ("to play 41", "rolls a 65"), or a move
// written out whose every step is one die ("2/0 2/1" is 2-1, "8/5(2) 6/3(2)"
// is 3-3; a checker borne off spends the die of its point). A move with a
// step of two dice tells no roll and is passed over. A checker borne off
// with a larger die makes the text disagree with the image: a doubt, never
// a reading.
func textRolls(words []Word, near func(Word) bool) [][2]int {
	var out [][2]int
	for i := 1; i < len(words); i++ {
		w := words[i]
		if !near(w) {
			continue
		}
		m := reDiceWord.FindStringSubmatch(w.Text)
		if m == nil {
			continue
		}
		tag := i - 1
		if reArticle.MatchString(words[tag].Text) && tag > 0 {
			tag--
		}
		if rePlayTag.MatchString(words[tag].Text) {
			x, _ := strconv.Atoi(m[1])
			y, _ := strconv.Atoi(m[2])
			out = append(out, [2]int{x, y})
		}
	}
	for i := 0; i < len(words); {
		j := i
		var steps []int
		exact := true
		for ; j < len(words) && near(words[j]); j++ {
			m := reMoveWord.FindStringSubmatch(words[j].Text)
			if m == nil {
				break
			}
			times := 1
			if m[2] != "" {
				times, _ = strconv.Atoi(m[2])
			}
			pts := strings.Split(strings.ReplaceAll(strings.ToLower(m[1]), "*", ""), "/")
			for k := 1; k < len(pts); k++ {
				from, to := pointOf(pts[k-1]), pointOf(pts[k])
				d := from - to
				if d < 1 || d > 6 {
					exact = false // a step of two dice
				}
				for range times {
					steps = append(steps, d)
				}
			}
		}
		if j == i {
			i++
			continue
		}
		if r, ok := rollOf(steps); ok && exact {
			out = append(out, r)
		}
		i = j
	}
	return out
}

// pointOf is a point of a move's notation: 25 the bar, 0 off.
func pointOf(s string) int {
	switch s {
	case "bar":
		return 25
	case "off":
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}

// rollOf is the roll a move's steps spend: two different dice once each,
// or one die two to four times (a double, possibly not played in full).
func rollOf(steps []int) ([2]int, bool) {
	switch {
	case len(steps) == 2 && steps[0] != steps[1]:
		return [2]int{steps[0], steps[1]}, true
	case len(steps) >= 2 && len(steps) <= 4:
		for _, d := range steps {
			if d != steps[0] {
				return [2]int{}, false
			}
		}
		return [2]int{steps[0], steps[0]}, true
	}
	return [2]int{}, false
}

// readScore reads the panel a diagram's text layer prints beside it — "score:
// 2  pip: 156  7 point match  pip: 150  score: 4" — each score assigned to
// a player by the pips printed nearest to it. Equal scores need no
// assignment: the panel is read whole without its pips. The words are taken
// as an OCR layer gives them: a tag glued to its number ("pip:9"), a colon
// read as a semicolon, "point" cut in two.
func readScore(rd *boardReading, d *Diagram, words []Word, env contextEnv, c *Context, add func(string, string, string, ...any)) {
	words = splitTagged(words)
	X0, Y0, X1, Y1 := float64(rd.bbox.x0), float64(rd.bbox.y0), float64(rd.bbox.x1), float64(rd.bbox.y1)
	p := rd.fieldPoint()
	type tagged struct{ v, cy float64 }
	var scores, pips []tagged
	var lengths []int
	crawford := false
	for i, w := range words {
		cx, cy := (w.X0+w.X1)/2, (w.Y0+w.Y1)/2
		if cx < X0-8*p || cx > X1+8*p || cy < Y0-2*p || cy > Y1+2*p || env.nearerOther(rd, cx, cy) {
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
			if n, ok := matchLength(words[i:]); ok {
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
	equal := len(scores) == 2 && scores[0].v == scores[1].v
	if len(scores) != 2 || len(lengths) != 1 || (len(pips) != 2 && !equal) {
		add("score", "score", "a score is printed beside the diagram and not read whole (%d scores, %d match lengths, %d pip counts)", len(scores), len(lengths), len(pips))
		return
	}
	s := Score{Length: lengths[0], Crawford: crawford}
	var set [2]bool
	if equal {
		s.Bottom, s.Top, set = int(scores[0].v), int(scores[0].v), [2]bool{true, true}
	}
	for _, sc := range scores {
		if equal {
			break
		}
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

// splitTagged cuts a tag glued to its number ("pip:9") in two words, the
// box shared by its letters' count.
func splitTagged(words []Word) []Word {
	var out []Word
	for i, w := range words {
		m := reGlued.FindStringSubmatch(w.Text)
		if m == nil {
			if out != nil {
				out = append(out, w)
			}
			continue
		}
		if out == nil {
			out = append([]Word(nil), words[:i]...)
		}
		cut := w.X0 + (w.X1-w.X0)*float64(len(m[1])+1)/float64(len(w.Text))
		out = append(out, Word{X0: w.X0, Y0: w.Y0, X1: cut, Y1: w.Y1, Text: m[1] + ":"}, Word{X0: cut, Y0: w.Y0, X1: w.X1, Y1: w.Y1, Text: m[2]})
	}
	if out == nil {
		return words
	}
	return out
}

// matchLength reads a match length at the head of ws: a number, then
// "point" (or "pt", "points") in one word or cut in two, then "match".
func matchLength(ws []Word) (int, bool) {
	if len(ws) < 3 || !reNumber.MatchString(ws[0].Text) {
		return 0, false
	}
	n, _ := strconv.Atoi(ws[0].Text)
	pt := ""
	for k := 1; k < len(ws) && k <= 2; k++ {
		pt += strings.ToLower(ws[k].Text)
		if rePoint.MatchString(pt) && k+1 < len(ws) && reMatch.MatchString(ws[k+1].Text) {
			return n, true
		}
	}
	return 0, false
}

// nearerOther tells whether (x, y) lies nearer the field of another diagram
// of the page than rd's: what is printed there is that diagram's.
func (env contextEnv) nearerOther(rd *boardReading, x, y float64) bool {
	own := rd.fieldDistance(x, y)
	for _, o := range env.others {
		if o.fieldDistance(x, y) < own {
			return true
		}
	}
	return false
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

// contextEnv is what the context of one diagram is read against: the other
// diagrams of its page, which claim the words and squares nearer to them,
// how many diagrams of the document draw a lone bare cube at each place
// (squareKey), and how many draw there a cube whose face is not bare and
// not read 64: a value printed where a bare face would say centred makes
// that bare face a figure the scan may have lost.
type contextEnv struct {
	others []*boardReading
	bare   map[string]int
	valued map[string]int
}

// fieldDistance is the distance from (x, y), in the page's pixels, to the
// diagram's field; 0 inside it.
func (rd *boardReading) fieldDistance(x, y float64) float64 {
	f := rd.field
	dx := max(f[0]-x, 0, x-f[3])
	dy := max(f[4]-y, 0, y-f[5])
	return math.Hypot(dx, dy)
}

// squareKey is where a cube is drawn: its height and its side of the field.
func (rd *boardReading) squareKey(s square) string {
	cx := (s.x0 + s.x1) / 2
	side := "bar"
	switch {
	case cx < rd.field[0]:
		side = "left"
	case cx > rd.field[3]:
		side = "right"
	}
	return s.place + " " + side
}

// keepSquares judges the squares found about each diagram, before any is
// read: a square nearer another diagram of its page is that diagram's, and
// one whose aspect, the stretch the document's checkers measure taken out,
// is not square (0.75 to 1.33) is no cube nor die. stretch is the checkers'
// width over their height, 1 when they are round.
func keepSquares(rds []*boardReading, stretch float64) {
	byPage := map[int][]*boardReading{}
	for _, rd := range rds {
		byPage[rd.page] = append(byPage[rd.page], rd)
	}
	keep := func(rd *boardReading, sqs []square) []square {
		var out []square
		for _, s := range sqs {
			if a := s.aspect / stretch; a < 0.75 || a > 1.33 {
				continue
			}
			cx, cy := (s.x0+s.x1)/2, (s.y0+s.y1)/2
			own, mine := rd.fieldDistance(cx, cy), true
			for _, o := range byPage[rd.page] {
				if o != rd && o.fieldDistance(cx, cy) < own {
					mine = false
				}
			}
			if mine {
				out = append(out, s)
			}
		}
		return out
	}
	for _, rd := range rds {
		rd.cubes, rd.dice = keep(rd, rd.cubes), keep(rd, rd.dice)
	}
}

// readsCentred tells whether the diagram's k-th cube reads centred: 64
// read in it, in the middle. A 64 at the top or the bottom is an owned cube
// showing its value, which a bare face at that place must not be taken for.
func readsCentred(rd *boardReading, k int) bool {
	if rd.cubes[k].place != "middle" {
		return false
	}
	for _, n := range rd.cubeFigs {
		if n.val == 64 && squareOf(rd.cubes, n) == k {
			return true
		}
	}
	return false
}

// contextEnvs gives each diagram its contextEnv.
func contextEnvs(rds []*boardReading) []contextEnv {
	bare, valued := map[string]int{}, map[string]int{}
	for _, rd := range rds {
		if len(rd.cubes) == 1 && rd.cubes[0].blank {
			bare[rd.squareKey(rd.cubes[0])]++
		}
		seen := map[string]bool{}
		for i, s := range rd.cubes {
			k := rd.squareKey(s)
			if s.blank || seen[k] || readsCentred(rd, i) {
				continue
			}
			seen[k] = true
			valued[k]++
		}
	}
	out := make([]contextEnv, len(rds))
	for i, rd := range rds {
		out[i].bare, out[i].valued = bare, valued
		for _, o := range rds {
			if o != rd && o.page == rd.page {
				out[i].others = append(out[i].others, o)
			}
		}
	}
	return out
}
