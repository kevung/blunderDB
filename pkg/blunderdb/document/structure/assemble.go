package structure

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
)

// A corner of the board, as row and side: "BL" is bottom left. A player's
// home is the corner of its points 1 to 6.
var opposite = map[string]string{"BR": "TR", "BL": "TL", "TR": "BR", "TL": "BL"}

// corners in the order hypotheses are tried; the first wins a tie.
var corners = []string{"BR", "BL", "TR", "TL"}

// pointNumber is the number, for a player whose home is home, of column j
// (0..11 left to right) of the top or bottom row.
func pointNumber(top bool, j int, home string) int {
	jj := 11 - j
	if home[1] == 'R' {
		jj = j
	}
	if (top && home[0] == 'T') || (!top && home[0] == 'B') {
		return 12 - jj
	}
	return 13 + jj
}

// classes are the document's two checker classes, learned on the inner grey
// of every checker it draws: 0 dark, 1 light.
type classes struct {
	c0, c1, t0, t1, split float64
	ok                    bool
}

func (c classes) isChecker(m float64) bool {
	return !c.ok || math.Abs(m-c.c0) <= c.t0 || math.Abs(m-c.c1) <= c.t1
}

func (c classes) isDark(m float64) bool { return c.ok && math.Abs(m-c.c0) <= c.t0 }

func (c classes) of(m float64) int {
	if m < c.split {
		return 0
	}
	return 1
}

// learnClasses splits the checkers in two by 2-means, then makes the classes
// robust (median, and four median absolute deviations, at least 0.08): a
// "checker" far from both is not one.
func learnClasses(ms []float64) classes {
	if len(ms) == 0 {
		return classes{split: 0.5}
	}
	c0, c1 := ms[0], ms[0]
	for _, m := range ms {
		c0, c1 = min(c0, m), max(c1, m)
	}
	split := func() ([]float64, []float64) {
		var a, b []float64
		for _, m := range ms {
			if math.Abs(m-c0) > math.Abs(m-c1) {
				b = append(b, m)
			} else {
				a = append(a, m)
			}
		}
		return a, b
	}
	mean := func(v []float64) float64 {
		s := 0.0
		for _, x := range v {
			s += x
		}
		return s / float64(len(v))
	}
	for i := 0; i < 30; i++ {
		a, b := split()
		if len(a) > 0 {
			c0 = mean(a)
		}
		if len(b) > 0 {
			c1 = mean(b)
		}
	}
	var a, b []float64
	for i := 0; i < 5; i++ {
		a, b = split()
		if len(a) > 0 {
			c0 = median(a)
		}
		if len(b) > 0 {
			c1 = median(b)
		}
	}
	tol := func(v []float64, c float64) float64 {
		if len(v) == 0 {
			return 0.08
		}
		d := make([]float64, len(v))
		for i, x := range v {
			d[i] = math.Abs(x - c)
		}
		return max(4*median(d)*1.4826, 0.08)
	}
	return classes{c0: c0, c1: c1, t0: tol(a, c0), t1: tol(b, c1), split: (c0 + c1) / 2, ok: true}
}

var (
	reNumber = regexp.MustCompile(`^[0-9]{1,3}$`)
	rePipTag = regexp.MustCompile(`(?i)^(pips?:?|pipcount:?)$`)
)

type sideNum struct {
	v  int
	cy float64
}

type numWord struct {
	v   int
	fx  float64
	top bool
}

// textFeatures finds, in the page's text layer, the numbering printed above
// and below the board and the candidate pip counts: numbers beside it at its
// height (left G, right D), numbers tagged "pips" (L), numbers inside the
// numbering row at the bar (B).
func textFeatures(b box, words []Word) ([]numWord, map[byte][]sideNum) {
	X0, Y0, X1, Y1 := float64(b.x0), float64(b.y0), float64(b.x1), float64(b.y1)
	W := X1 - X0
	p := W / 13
	side := map[byte][]sideNum{}
	for i := 0; i+1 < len(words); i++ {
		nx := words[i+1]
		if rePipTag.MatchString(words[i].Text) && reNumber.MatchString(nx.Text) {
			cy, cx := (nx.Y0+nx.Y1)/2, (nx.X0+nx.X1)/2
			if Y0-2*p <= cy && cy <= Y1+2*p && X0-8*p <= cx && cx <= X1+8*p {
				v, _ := strconv.Atoi(nx.Text)
				side['L'] = append(side['L'], sideNum{v, cy})
			}
		}
	}
	var num []numWord
	for _, w := range words {
		if !reNumber.MatchString(w.Text) {
			continue
		}
		cx, cy := (w.X0+w.X1)/2, (w.Y0+w.Y1)/2
		v, _ := strconv.Atoi(w.Text)
		switch {
		case X0 <= cx && cx <= X1 && ((Y0-1.2*p <= cy && cy < Y0+0.1*p) || (Y1-0.1*p < cy && cy <= Y1+1.2*p)):
			num = append(num, numWord{v, (cx - X0) / W, cy < (Y0+Y1)/2})
		case Y0-0.3*p <= cy && cy <= Y1+0.3*p && v >= 1 && v <= 375:
			if X0-3.5*p <= cx && cx < X0 {
				side['G'] = append(side['G'], sideNum{v, cy})
			} else if X1 < cx && cx <= X1+3.5*p {
				side['D'] = append(side['D'], sideNum{v, cy})
			}
		}
	}
	// Pips printed inside the numbering row, at the bar: a number of the
	// middle of the row that follows neither neighbour ("18 52 19").
	for _, top := range []bool{true, false} {
		var row []numWord
		for _, t := range num {
			if t.top == top {
				row = append(row, t)
			}
		}
		sort.SliceStable(row, func(i, j int) bool { return row[i].fx < row[j].fx })
		for i, t := range row {
			if t.fx < 0.3 || t.fx > 0.7 {
				continue
			}
			var nb []int
			for _, d := range []int{-1, 1} {
				if i+d >= 0 && i+d < len(row) {
					nb = append(nb, row[i+d].v)
				}
			}
			follows := false
			for _, v := range nb {
				if abs(t.v-v) == 1 {
					follows = true
				}
			}
			if len(nb) == 0 || follows {
				continue
			}
			if t.v > 24 || (len(nb) == 2 && abs(nb[0]-nb[1]) == 1) {
				y := Y1
				if top {
					y = Y0
				}
				side['B'] = append(side['B'], sideNum{t.v, y})
				for k, u := range num {
					if u == t {
						num = append(num[:k:k], num[k+1:]...)
						break
					}
				}
			}
		}
	}
	return num, side
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// stack is a pile read on one side of a column: its class and height.
type stack struct{ class, n int }

// count is how a stack was counted other than checker by checker.
type count struct {
	kind  string // "length" or "share"
	side  int
	from  int
	to    int
	other int
}

// readColumn turns one whole column (top stack, bottom stack) into stacks.
// It counts the checkers seen one by one, extended by the length of a dark
// stack (dark checkers touching have no edge between them), and shares a
// chain running from top to bottom at its one change of class.
func readColumn(ct, cb column, cl classes, N int) (out [2]*stack, doubt string, marks [2]bool, how []count) {
	var res [2][]int
	for i, col := range []column{ct, cb} {
		ds := col.discs
		for len(ds) > 0 && !cl.isChecker(ds[len(ds)-1].mean) {
			ds = ds[:len(ds)-1]
		}
		for _, x := range ds {
			if !cl.isChecker(x.mean) {
				return out, "off classes", marks, nil
			}
		}
		for _, x := range ds {
			res[i] = append(res[i], cl.of(x.mean))
			if x.marked {
				marks[i] = true
			}
		}
	}
	for i, col := range []column{ct, cb} {
		if marks[i] {
			continue
		}
		dark := true
		for _, c := range res[i] {
			if c != 0 {
				dark = false
			}
		}
		if !dark {
			continue
		}
		run := 0
		for run < len(col.slotMeans) && cl.isDark(col.slotMeans[run]) {
			run++
		}
		other := len(res[1-i])
		if run > len(res[i]) && run < len(col.slotMeans) && run+other <= N {
			how = append(how, count{"length", i, len(res[i]), run, 0})
			res[i] = make([]int, run)
		}
	}
	nt, nb := len(res[0]), len(res[1])
	if nt+nb > N {
		cut := func(c []int) int {
			ch := -1
			for k := 1; k < len(c); k++ {
				if c[k] != c[k-1] {
					if ch >= 0 {
						return -1
					}
					ch = k
				}
			}
			return ch
		}
		i, j := cut(res[0]), cut(res[1])
		if i < 0 || j < 0 || i+j != nt || nt != nb {
			return out, "full column", marks, how
		}
		out[0], out[1] = &stack{res[0][0], i}, &stack{res[1][0], j}
		how = append(how, count{"share", 0, i, j, 0})
		return out, "", marks, how
	}
	for s := 0; s < 2; s++ {
		c := res[s]
		if len(c) == 0 {
			continue
		}
		for _, x := range c {
			if x != c[0] {
				return [2]*stack{}, "mixed classes", marks, how
			}
		}
		out[s] = &stack{c[0], len(c)}
	}
	return out, "", marks, how
}

// side of a reading's doubts, before they are named for the relevé.
type rawDoubt struct {
	kind  string // "off classes", "full column", "mixed classes", "marked", "bar marked"
	j     int    // column, or -1
	class int
	point int // the class's own point number for "marked"
}

type assembled struct {
	pts    [2]map[int]int // per class: point → checkers
	bar    [2]int
	doubts []rawDoubt
	marked [][2]int // (class, point)
	how    []count
	howJ   []int
}

func assemble(rd *boardReading, cl classes, home0 string) assembled {
	a := assembled{pts: [2]map[int]int{{}, {}}}
	homes := [2]string{home0, opposite[home0]}
	for j := 0; j < 12; j++ {
		ct, cb := rd.cols[j], rd.cols[12+j]
		out, d, marks, how := readColumn(ct, cb, cl, rd.n)
		for range how {
			a.howJ = append(a.howJ, j)
		}
		a.how = append(a.how, how...)
		if d != "" {
			a.doubts = append(a.doubts, rawDoubt{kind: d, j: j})
			continue
		}
		for s, top := range []bool{true, false} {
			if out[s] == nil || out[s].n == 0 {
				continue
			}
			k, n := out[s].class, out[s].n
			pt := pointNumber(top, j, homes[k])
			if marks[s] {
				a.doubts = append(a.doubts, rawDoubt{kind: "marked", j: j, class: k, point: pt})
				a.marked = append(a.marked, [2]int{k, pt})
			}
			a.pts[k][pt] = n
		}
	}
	for _, x := range rd.bar {
		if !cl.isChecker(x.mean) {
			continue
		}
		k := cl.of(x.mean)
		a.bar[k]++
		if x.std > 0.25 {
			a.doubts = append(a.doubts, rawDoubt{kind: "bar marked", j: -1, class: k})
		}
	}
	return a
}

func pipsOf(pts [2]map[int]int, bar [2]int, k int) int {
	s := 25 * bar[k]
	for p, n := range pts[k] {
		s += p * n
	}
	return s
}

func checkersOf(pts [2]map[int]int, bar [2]int, k int) int {
	s := bar[k]
	for _, n := range pts[k] {
		s += n
	}
	return s
}

func sorted2(a, b int) [2]int {
	if a > b {
		return [2]int{b, a}
	}
	return [2]int{a, b}
}

// solveByPips uses the printed pips as a constraint: one side alone
// disagrees, short by d; the one stack that can explain it (the marked one,
// else one of at least three checkers seen: only a high stack can hide some)
// gets d/point more. Two equations: the pips (the deduction) and fifteen
// checkers (the confirmation, true only when nothing is borne off).
func solveByPips(a assembled, printed *[2]int, order string) (pts [2]map[int]int, class, point, add int, fifteen, ok bool) {
	if (order != "top=class0" && order != "top=class1") || printed == nil {
		return
	}
	tgt := *printed
	if order == "top=class1" {
		tgt = [2]int{printed[1], printed[0]}
	}
	var bad []int
	for k := 0; k < 2; k++ {
		if pipsOf(a.pts, a.bar, k) != tgt[k] {
			bad = append(bad, k)
		}
	}
	if len(bad) != 1 {
		return
	}
	k := bad[0]
	d := tgt[k] - pipsOf(a.pts, a.bar, k)
	if d <= 0 {
		return
	}
	var cands []int
	for _, m := range a.marked {
		if m[0] == k {
			cands = append(cands, m[1])
		}
	}
	if len(cands) == 0 {
		for pt, h := range a.pts[k] {
			if h >= 3 {
				cands = append(cands, pt)
			}
		}
	}
	n0 := checkersOf(a.pts, a.bar, k)
	sols := 0
	for _, pt := range cands {
		if d%pt == 0 && n0+d/pt <= 15 {
			sols++
			point, add = pt, d/pt
		}
	}
	if sols != 1 {
		return
	}
	pts = [2]map[int]int{{}, {}}
	for c := 0; c < 2; c++ {
		for p, n := range a.pts[c] {
			pts[c][p] = n
		}
	}
	pts[k][point] += add
	return pts, k, point, add, n0+add == 15, true
}

// boardString is the XGID board of the reading: the class whose home is in
// the bottom row is the XGID's upper-case player. False when a point holds
// more checkers than a letter can say.
func boardString(pts [2]map[int]int, bar [2]int, upper int) (string, bool) {
	lower := 1 - upper
	s := []byte("--------------------------")
	for p, n := range pts[upper] {
		if n > 15 {
			return "", false
		}
		s[p] = byte('A' + n - 1)
	}
	for p, n := range pts[lower] {
		if n > 15 {
			return "", false
		}
		s[25-p] = byte('a' + n - 1)
	}
	if bar[upper] > 15 || bar[lower] > 15 {
		return "", false
	}
	if bar[upper] > 0 {
		s[25] = byte('A' + bar[upper] - 1)
	}
	if bar[lower] > 0 {
		s[0] = byte('a' + bar[lower] - 1)
	}
	return string(s), true
}

// relevé learns what the whole document shares (classes, numbering, pip
// place, direction of play) and turns each raw reading into a diagram.
// releve judges every diagram with the document's classes, numbering,
// printed pips (text layer, image) and direction of play; home, when set,
// imposes the corner of class 0's home instead of voting it.
func releve(rds []*boardReading, words map[int][]Word, img []*[2]int, home string) ([]Diagram, Profile) {
	var means []float64
	for _, rd := range rds {
		for _, c := range rd.cols {
			for _, x := range c.discs {
				means = append(means, x.mean)
			}
		}
		for _, x := range rd.bar {
			means = append(means, x.mean)
		}
	}
	cl := learnClasses(means)
	numbering := map[string]map[[2]int]int{}
	for _, h := range corners {
		numbering[h] = map[[2]int]int{}
		for t, top := range []bool{true, false} {
			for j := 0; j < 12; j++ {
				numbering[h][[2]int{t, j}] = pointNumber(top, j, h)
			}
		}
	}
	type feat struct {
		num  []numWord
		side map[byte][]sideNum
		pips *[2]int
		read pipsRead
	}
	fs := make([]feat, len(rds))
	numVotes := map[string]int{}
	var numOrder []string
	vote := func(m map[string]int, order *[]string, k string) {
		if _, ok := m[k]; !ok {
			*order = append(*order, k)
		}
		m[k]++
	}
	sideCount := map[byte]int{}
	for i, rd := range rds {
		w, ok := words[rd.page]
		if !ok {
			fs[i].side = map[byte][]sideNum{}
		} else {
			fs[i].num, fs[i].side = textFeatures(rd.bbox, w)
		}
		for _, t := range fs[i].num {
			j := trunc(t.fx * 13)
			if j >= 6 {
				j--
			}
			j = min(max(j, 0), 11)
			ti := 1
			if t.top {
				ti = 0
			}
			for _, h := range corners {
				if numbering[h][[2]int{ti, j}] == t.v {
					vote(numVotes, &numOrder, h)
				}
			}
		}
		for _, s := range []byte("GDLB") {
			n := len(fs[i].side[s])
			if (s == 'L' || s == 'B') && n == 2 || (s == 'G' || s == 'D') && n >= 2 {
				sideCount[s]++
			}
		}
	}
	// Without a text layer: the shape of the numbering (one glyph for 1-9,
	// two for 10-24).
	if len(numVotes) == 0 {
		for _, rd := range rds {
			var ones [2]int
			for r := 0; r < 2; r++ {
				for _, v := range rd.digits[r] {
					if v == 1 {
						ones[r]++
					}
				}
			}
			if max(ones[0], ones[1]) < 6 || min(ones[0], ones[1]) > 3 {
				continue
			}
			r := 1
			if ones[0] > ones[1] {
				r = 0
			}
			left, right := 0, 0
			for j, v := range rd.digits[r] {
				if v == 1 {
					if j < 6 {
						left++
					} else {
						right++
					}
				}
			}
			if left == right {
				continue
			}
			h := "B"
			if r == 0 {
				h = "T"
			}
			if left > right {
				h += "L"
			} else {
				h += "R"
			}
			vote(numVotes, &numOrder, h)
		}
	}
	numberingCorner := ""
	if len(numVotes) > 0 {
		best, total := "", 0
		for _, h := range numOrder {
			total += numVotes[h]
			if best == "" || numVotes[h] > numVotes[best] {
				best = h
			}
		}
		if float64(numVotes[best]) >= 0.6*float64(total) && numVotes[best] >= 20 {
			numberingCorner = best
		}
	}
	var pipSide byte
	if len(sideCount) > 0 {
		if float64(sideCount['L']) >= 0.5*float64(len(rds)) {
			pipSide = 'L'
		} else {
			pipSide = 'G'
			for _, s := range []byte("DB") {
				if sideCount[s] > sideCount[pipSide] {
					pipSide = s
				}
			}
		}
		if sideCount[pipSide] == 0 {
			pipSide = 0
		}
	}
	for i := range fs {
		var m *[2]int
		if img != nil {
			m = img[i]
		}
		fs[i].pips, fs[i].read = printedPips(fs[i].side[pipSide], pipSide != 0, m)
	}
	// Direction of play: the corner of class 0's home.
	hyps := corners
	if numberingCorner != "" {
		hyps = []string{numberingCorner, opposite[numberingCorner]}
	}
	if home != "" {
		hyps = []string{home}
	}
	votes := map[string]int{}
	pipSum := map[string]float64{}
	totalVotes := 0
	for i, rd := range rds {
		for _, h := range hyps {
			a := assemble(rd, cl, h)
			pc := [2]int{pipsOf(a.pts, a.bar, 0), pipsOf(a.pts, a.bar, 1)}
			n := checkersOf(a.pts, a.bar, 0) + checkersOf(a.pts, a.bar, 1)
			pipSum[h] += float64(pc[0]+pc[1]) / float64(max(n, 1))
			if pr := fs[i].pips; pr != nil && sorted2(pc[0], pc[1]) == sorted2(pr[0], pr[1]) {
				votes[h]++
				totalVotes++
			}
		}
	}
	home0, assumed := hyps[0], false
	if totalVotes >= 10 {
		for _, h := range hyps {
			if votes[h] > votes[home0] {
				home0 = h
			}
		}
	} else {
		assumed = true
		for _, h := range hyps {
			if pipSum[h] < pipSum[home0] {
				home0 = h
			}
		}
	}
	// Which printed count (higher, lower) belongs to which class: a vote.
	orderVotes := map[string]int{}
	var orderOrder []string
	for i, rd := range rds {
		pr := fs[i].pips
		if pr == nil {
			continue
		}
		a := assemble(rd, cl, home0)
		pc := [2]int{pipsOf(a.pts, a.bar, 0), pipsOf(a.pts, a.bar, 1)}
		if sorted2(pc[0], pc[1]) == sorted2(pr[0], pr[1]) && pc[0] != pc[1] {
			if pr[0] == pc[0] {
				vote(orderVotes, &orderOrder, "top=class0")
			} else {
				vote(orderVotes, &orderOrder, "top=class1")
			}
		}
	}
	order := ""
	if len(orderVotes) > 0 {
		best, total := "", 0
		for _, k := range orderOrder {
			total += orderVotes[k]
			if best == "" || orderVotes[k] > orderVotes[best] {
				best = k
			}
		}
		if float64(orderVotes[best]) >= 0.95*float64(total) && orderVotes[best] >= 10 {
			order = best
		}
	}
	upper := 1
	if home0[0] == 'B' {
		upper = 0
	}
	prof := Profile{
		Classes:        [2]float64{cl.c0, cl.c1},
		HomeSide:       home0[1],
		BottomIsDark:   upper == 0,
		DirectionVoted: !assumed,
		Numbering:      numberingCorner,
		PipPlace:       pipSide,
	}
	diagrams := make([]Diagram, 0, len(rds))
	for i, rd := range rds {
		diagrams = append(diagrams, judge(rd, cl, home0, upper, order, assumed, fs[i].pips, fs[i].read))
	}
	return diagrams, prof
}

// printedPips are a diagram's printed pips, from the text layer (the
// numbers at the document's pip place, top then bottom) and from the image:
// when both give a count and they differ, neither is taken.
func printedPips(side []sideNum, placed bool, image *[2]int) (*[2]int, pipsRead) {
	read := pipsRead{image: image}
	if placed {
		c := append([]sideNum(nil), side...)
		sort.SliceStable(c, func(a, b int) bool { return c[a].cy < c[b].cy })
		if len(c) >= 2 {
			read.text = &[2]int{c[0].v, c[len(c)-1].v}
		}
	}
	t, m := read.text, read.image
	switch {
	case t != nil && m != nil && *t != *m:
		return nil, read
	case t != nil:
		return t, read
	}
	return m, read
}

// pipsRead are a diagram's printed pips (upper number, lower number) as
// the text layer and the image give them, nil where one gives none.
type pipsRead struct{ text, image *[2]int }

func colourName(k int) string {
	if k == 0 {
		return "dark"
	}
	return "light"
}

// judge gives one diagram its board, state and named doubts. Confirmed only
// when the printed pips agree on both sides, no side has more than fifteen
// checkers and nothing is in doubt; a stack deduced from the pips is sure
// only when its side then has fifteen checkers.
func judge(rd *boardReading, cl classes, home0 string, upper int, order string, assumed bool, printed *[2]int, read pipsRead) Diagram {
	a := assemble(rd, cl, home0)
	homeUpper := "B" + home0[1:]
	idx := func(top bool, j int) int { return pointNumber(top, j, homeUpper) }
	classIdx := func(k, pt int) int {
		if k == upper {
			return pt
		}
		return 25 - pt
	}
	var doubts []capture.Doubt
	var families []string
	add := func(field string, point int, family, format string, args ...any) {
		doubts = append(doubts, capture.Doubt{Field: field, Point: point, Reason: fmt.Sprintf(format, args...)})
		if family != "" {
			families = append(families, family)
		}
	}
	onlyMarked := true
	for _, d := range a.doubts {
		if d.kind != "marked" {
			onlyMarked = false
		}
	}
	pts, bar := a.pts, a.bar
	var solved bool
	var sClass, sPoint, sAdd int
	var fifteen bool
	if onlyMarked {
		if p, k, pt, ad, f, ok := solveByPips(a, printed, order); ok {
			pts, solved, sClass, sPoint, sAdd, fifteen = p, true, k, pt, ad, f
		}
	}
	for _, d := range a.doubts {
		switch d.kind {
		case "off classes":
			add("point", idx(true, d.j), "off classes", "column of points %d and %d: a disc of neither checker class (faded or stained?)", idx(true, d.j), idx(false, d.j))
		case "full column":
			add("point", idx(true, d.j), "top/bottom mixed", "column of points %d and %d: one chain from top to bottom, not shared at a single change of class", idx(true, d.j), idx(false, d.j))
		case "mixed classes":
			add("point", idx(true, d.j), "top/bottom mixed", "column of points %d and %d: both classes in one stack", idx(true, d.j), idx(false, d.j))
		case "marked":
			if !solved {
				add("point", classIdx(d.class, d.point), "figure", "a checker bears a figure: the stack's height is not read")
			}
		case "bar marked":
			bi := 0
			if d.class == upper {
				bi = 25
			}
			add("bar", bi, "bar", "a checker on the bar bears a mark")
		}
	}
	if solved && !fifteen {
		add("point", classIdx(sClass, sPoint), "", "the stack's height is deduced from the printed pips (%d more), and its side does not then have fifteen checkers", sAdd)
	}
	if solved {
		if len(a.marked) > 0 {
			families = append(families, "figure")
		} else {
			families = append(families, "short high stack")
		}
	}
	if min(rd.rows[0], rd.rows[1]) == 0 {
		add("board", 0, "half board", "half board: a row without any point")
	}
	n := [2]int{checkersOf(pts, bar, 0), checkersOf(pts, bar, 1)}
	pc := [2]int{pipsOf(pts, bar, 0), pipsOf(pts, bar, 1)}
	var okPips bool
	switch {
	case printed == nil:
	case order == "top=class0":
		okPips = pc == *printed
	case order == "top=class1":
		okPips = pc == [2]int{printed[1], printed[0]}
	default:
		okPips = sorted2(pc[0], pc[1]) == sorted2(printed[0], printed[1])
	}
	pieceDoubts := len(doubts)
	state := ToVerify
	if max(n[0], n[1]) > 15 {
		state = Uncertain
	} else if okPips && pieceDoubts == 0 && (!solved || fifteen) {
		state = Confirmed
	}
	d := Diagram{
		Page:      rd.page,
		Rect:      image2Rect(rd.bbox),
		PageSize:  [2]int{rd.pageW, rd.pageH},
		Checkers:  [2]int{n[upper], n[1-upper]},
		Pips:      [2]int{pc[upper], pc[1-upper]},
		Printed:   printed,
		TextPips:  read.text,
		ImagePips: read.image,
		Complete:  n == [2]int{15, 15} && pieceDoubts == 0 && !solved,
	}
	// What is drawn and not read: never invented, named so a sure reading
	// is not taken for more than its checkers.
	// The cube, dice, player on roll and score are read after the
	// judgement (readContext): what is drawn and not read is named there.
	if !okPips {
		if t, m := read.text, read.image; printed == nil && t != nil && m != nil {
			add("pips", 0, "pips text/image", "printed pips read %d and %d in the text layer, %d and %d in the image: the two disagree", t[0], t[1], m[0], m[1])
		} else if printed == nil {
			add("pips", 0, "", NoPipsReason)
		} else {
			add("pips", 0, "", "pips read %d (bottom) and %d (top), printed %d and %d", pc[upper], pc[1-upper], printed[0], printed[1])
		}
		if assumed {
			add("orientation", 0, "", AssumedDirectionReason)
		}
		for _, k := range []int{upper, 1 - upper} {
			if n[k] < 15 {
				add("checkers", 0, "", "the %s player (%s) has %d checkers drawn: the other %d are taken as borne off, the tray is not read", map[bool]string{true: "bottom", false: "top"}[k == upper], colourName(k), n[k], 15-n[k])
			} else if n[k] > 15 {
				add("checkers", 0, "", "the %s player (%s) has %d checkers drawn, more than fifteen", map[bool]string{true: "bottom", false: "top"}[k == upper], colourName(k), n[k])
			}
		}
	}
	if state == Confirmed && len(doubts) > 0 {
		state = ToVerify
	}
	// How stacks were counted, when it is not one by one: on an entry to
	// verify, a lead for the person who checks it.
	for i, h := range a.how {
		fam := map[string]string{"length": "joined", "share": "top/bottom joined"}[h.kind]
		families = append(families, fam)
		if state != Confirmed {
			j := a.howJ[i]
			if h.kind == "length" {
				doubts = append(doubts, capture.Doubt{Field: "point", Point: idx(h.side == 0, j), Reason: fmt.Sprintf("a dark stack counted by its length: %d checkers, %d seen one by one", h.to, h.from)})
			} else {
				doubts = append(doubts, capture.Doubt{Field: "point", Point: idx(true, j), Reason: fmt.Sprintf("column of points %d and %d: one chain shared at its change of class, %d above and %d below", idx(true, j), idx(false, j), h.from, h.to)})
			}
		}
	}
	if state != Uncertain {
		if b, ok := boardString(pts, bar, upper); ok {
			d.Board = b
		}
	}
	d.State, d.Doubts, d.Families = state, doubts, families
	d.CheckersSure = max(n[0], n[1]) <= 15 && okPips && pieceDoubts == 0 && (!solved || fifteen)
	return d
}
