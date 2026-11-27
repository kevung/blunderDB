package structure

import (
	_ "embed"
	"slices"
	"sort"
	"strings"
	"sync"
)

// The caption of a diagram read in the image, on a page without a text
// layer: "Problem 66: Black to play 53." Its lines are cut into words by
// their ink alone. A word is read only as one of the few words a caption
// states its context with (a colour, "to play", "rolls"), its whole drawing
// matched against those words in common typefaces, or as a number each of
// whose figures is recognised; anything else reads "?" and holds its place.
// What is read becomes words of a text layer, judged as any text layer is
// (readDice, captionDice): a roll named and not drawn is a doubt, never a
// reading.

// wordFont holds the words of wordList in the ten typefaces of wordFonts
// (words_gen_test.go: sans serif, serif, their bold and italics), each its
// width over its
// height (a byte, in 32nds) then its ink resampled to wordW × wordH,
// quantised to levels 0 to 15, two pixels a byte (high nibble first).
// TestGenerateWords writes it.
//
//go:embed words.bin
var wordFont []byte

const (
	wordW, wordH = 48, 16
	wordScore    = 0.60 // the least correlation for a word to be read
	wordMargin   = 0.04 // the least lead over every other word
	wordAspect   = 1.3  // the most a word's proportions may differ
)

// figureFont holds the figures 0 to 9 of wordFonts' typefaces, in
// digitFont's format: the figures of a caption, often bolder or of another
// face than those around the board.
//
//go:embed figures.bin
var figureFont []byte

// captionFigures are the document's figures and figureFont's, each centred
// on its mean: a caption's figure is matched by its correlation with the
// bare glyph, which a bolder stroke than the drawing's does not mislead as
// it does the blurred figure alone.
func captionFigures(digits []template) []template {
	var out []template
	for _, t := range append(append([]template(nil), digits...), figureTemplates()...) {
		out = append(out, template{digit: t.digit, v: centred(t.v), learned: t.learned, weight: t.weight})
	}
	return out
}

const (
	captionFigureScore  = 0.70 // the least correlation for a caption's figure
	captionFigureMargin = 0.06 // its least lead over every other figure
)

// captionFigure is the figure a caption's glyph reads, or -1: the best
// correlation (figures, centred templates) reaches captionFigureScore, leads
// every other figure by captionFigureMargin, and its holes agree.
func captionFigure(figures []template, q *[glyphSide * glyphSide]uint8) int {
	m := glyphFloats(q, 1)
	holes, holeY := squareHoles(m)
	d1 := bestFigure(figureScores(figures, centred(m)), captionFigureScore, captionFigureMargin)
	if d1 < 0 || !slices.Contains(holesOf[d1], holes) {
		return -1
	}
	if (d1 == 6 || d1 == 9) && holeY >= 0 && (holeY > 0.5) != (d1 == 6) {
		return -1
	}
	return d1
}

var figureTemplates = sync.OnceValue(func() []template { return templatesOf(figureFont) })

// wordList are the words a caption is read for, as typefaces draw them;
// a word is read as its lower case.
var wordList = []string{"Black", "black", "White", "white", "Dark", "dark", "Light", "light", "to", "play", "plays", "rolls", "rolled"}

type wordTemplate struct {
	text   string
	aspect float64
	v      []float64
}

var wordTemplates = sync.OnceValue(func() []wordTemplate {
	const n = 1 + wordW*wordH/2
	var out []wordTemplate
	for k := 0; k+n <= len(wordFont); k += n {
		m := make([]float64, wordW*wordH)
		for i := range m {
			b := wordFont[k+1+i/2]
			if i%2 == 0 {
				b >>= 4
			}
			m[i] = float64(b & 15)
		}
		out = append(out, wordTemplate{strings.ToLower(wordList[(k/n)%len(wordList)]), float64(wordFont[k]) / 32, centred(m)})
	}
	return out
})

// normWord resamples a word's ink, its box w × h, to wordW × wordH.
func normWord(w, h int, ink func(x, y int) bool) []float64 {
	v := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if ink(x, y) {
				v[y*w+x] = 255
			}
		}
	}
	return resample(v, w, h, wordW, wordH, triangle, 1, true)
}

// captionGlyph is a letter, a figure or a mark of a caption's line.
type captionGlyph struct {
	x0, y0, x1, y1 int // in the page's pixels
	q              [glyphSide * glyphSide]uint8
}

// captionWord is a run of glyphs of a line between two spaces: each glyph,
// to read figures, and the whole word's ink, to read a word.
type captionWord struct {
	x0, y0, x1, y1 int
	glyphs         []captionGlyph
	shape          [wordW * wordH]uint8
}

// extractCaptions cuts the lines printed above and below the diagram whose
// box is b, at most four point widths out and two across, into words.
func extractCaptions(g *raster, b box) [][]captionWord {
	X0, Y0, X1, Y1 := float64(b.x0), float64(b.y0), float64(b.x1), float64(b.y1)
	p := (X1 - X0) / 13
	var lines [][]captionWord
	for _, band := range [2][2]float64{{Y0 - 4*p, Y0}, {Y1, Y1 + 4*p}} {
		rx0, rx1 := max(0, trunc(X0-2*p)), min(g.w, trunc(X1+2*p))
		ry0, ry1 := max(0, trunc(band[0])), min(g.h, trunc(band[1]))
		if rx1-rx0 < 4 || ry1-ry0 < 4 {
			continue
		}
		sub := g.crop(rx0, ry0, rx1, ry1)
		lab, comps := label8(hysteresis(sub))
		for _, ln := range textLines(comps, sub.w, sub.h, p) {
			if ws := lineWords(ln, comps, lab, sub.w, rx0, ry0); len(ws) >= 2 {
				lines = append(lines, ws)
			}
		}
	}
	return lines
}

// textLines groups the components of a letter's size into lines, the
// tallest first so that a dot or a mark joins the line of its letters.
func textLines(comps []box, w, h int, p float64) [][]int {
	var idx []int
	for i, c := range comps {
		if c.h() < 2 || c.w() < 2 || float64(c.h()) > 1.3*p || c.w() > 4*c.h()+4 {
			continue
		}
		if c.x0 == 0 || c.y0 == 0 || c.x1 == w || c.y1 == h {
			continue
		}
		idx = append(idx, i)
	}
	sort.SliceStable(idx, func(a, b int) bool { return comps[idx[a]].h() > comps[idx[b]].h() })
	type line struct {
		y0, y1 int
		ms     []int
	}
	var ls []*line
	for _, i := range idx {
		c := comps[i]
		var at *line
		for _, l := range ls {
			ov := min(l.y1, c.y1) - max(l.y0, c.y0)
			if float64(ov) >= 0.5*float64(min(c.h(), l.y1-l.y0)) {
				at = l
				break
			}
		}
		if at == nil {
			if float64(c.h()) < 0.12*p {
				continue
			}
			at = &line{y0: c.y0, y1: c.y1}
			ls = append(ls, at)
		} else if 2*c.h() >= at.y1-at.y0 {
			at.y0, at.y1 = min(at.y0, c.y0), max(at.y1, c.y1)
		}
		at.ms = append(at.ms, i)
	}
	var out [][]int
	for _, l := range ls {
		out = append(out, l.ms)
	}
	return out
}

// lineWords cuts a line into glyphs (components one above the other make
// one: the dot of an i, a colon) and the glyphs into words at every gap of
// a fifth of the line's median glyph height; marks (a period, a colon, a
// comma) end a word and are dropped.
func lineWords(ms []int, comps []box, lab []int32, w, ox, oy int) []captionWord {
	sort.Slice(ms, func(a, b int) bool { return comps[ms[a]].x0 < comps[ms[b]].x0 })
	type glyph struct {
		b     box
		parts []int
	}
	var gs []glyph
	for _, i := range ms {
		c := comps[i]
		if n := len(gs); n > 0 {
			l := &gs[n-1]
			ov := min(l.b.x1, c.x1) - max(l.b.x0, c.x0)
			vo := min(l.b.y1, c.y1) - max(l.b.y0, c.y0)
			if 2*ov >= min(l.b.w(), c.w()) && 5*vo <= min(l.b.h(), c.h()) {
				l.b = box{min(l.b.x0, c.x0), min(l.b.y0, c.y0), max(l.b.x1, c.x1), max(l.b.y1, c.y1)}
				l.parts = append(l.parts, i)
				continue
			}
		}
		gs = append(gs, glyph{c, []int{i}})
	}
	ref := 0
	var hs []float64
	for _, g := range gs {
		ref = max(ref, g.b.h())
		hs = append(hs, float64(g.b.h()))
	}
	gap := 0.2 * median(hs)
	mark := func(g glyph) bool {
		for _, i := range g.parts {
			if c := comps[i]; 10*c.h() >= 4*ref || 10*c.w() >= 3*ref {
				return false
			}
		}
		return true
	}
	var out []captionWord
	var cur *captionWord
	in := map[int32]bool{}
	end := func() {
		if cur != nil {
			x0, y0 := cur.x0-ox, cur.y0-oy
			v := normWord(cur.x1-cur.x0, cur.y1-cur.y0, func(x, y int) bool { return in[lab[(y0+y)*w+x0+x]] })
			for k, x := range v {
				cur.shape[k] = uint8(x)
			}
			out = append(out, *cur)
			cur = nil
		}
		clear(in)
	}
	last := -1
	for _, g := range gs {
		if mark(g) {
			end()
			continue
		}
		if cur != nil && float64(g.b.x0-last) >= gap {
			end()
		}
		mine := map[int32]bool{}
		for _, i := range g.parts {
			mine[int32(i+1)], in[int32(i+1)] = true, true
		}
		b := g.b
		v := normSquare(b.w(), b.h(), func(x, y int) bool { return mine[lab[(b.y0+y)*w+b.x0+x]] })
		cg := captionGlyph{x0: b.x0 + ox, y0: b.y0 + oy, x1: b.x1 + ox, y1: b.y1 + oy}
		for k, x := range v {
			cg.q[k] = uint8(x)
		}
		if cur == nil {
			cur = &captionWord{x0: cg.x0, y0: cg.y0, x1: cg.x1, y1: cg.y1}
		}
		cur.x0, cur.y0, cur.x1, cur.y1 = min(cur.x0, cg.x0), min(cur.y0, cg.y0), max(cur.x1, cg.x1), max(cur.y1, cg.y1)
		cur.glyphs = append(cur.glyphs, cg)
		last = b.x1
	}
	end()
	return out
}

// captionText is the word as read: the word of wordList whose drawing its
// ink matches best, of like proportions, ahead of every other word; else its
// figures when each is read (captionFigure).
// A word that is neither reads "?": it holds its place in the line, so that
// the words around it are never taken for neighbours.
func captionText(w captionWord, figures []template) string {
	if text, s := wordRead(w); s >= wordScore {
		return text
	}
	fig := make([]byte, 0, len(w.glyphs))
	for k := range w.glyphs {
		d := captionFigure(figures, &w.glyphs[k].q)
		if d < 0 {
			return "?"
		}
		fig = append(fig, byte('0'+d))
	}
	return string(fig)
}

// wordRead is the word of wordList the ink matches best and its lead
// counted in: its correlation, less what falls short of wordMargin ahead of
// the best other word.
func wordRead(w captionWord) (string, float64) {
	aspect := float64(w.x1-w.x0) / float64(max(1, w.y1-w.y0))
	m := make([]float64, len(w.shape))
	for i, x := range w.shape {
		m[i] = float64(x)
	}
	u := centred(m)
	best := map[string]float64{}
	for _, t := range wordTemplates() {
		if aspect > t.aspect*wordAspect || aspect*wordAspect < t.aspect {
			continue
		}
		s := 0.0
		for i, x := range t.v {
			s += x * u[i]
		}
		if b, ok := best[t.text]; !ok || s > b {
			best[t.text] = s
		}
	}
	top, s1, s2 := "", -1.0, -1.0
	for t, s := range best {
		switch {
		case s > s1:
			top, s1, s2 = t, s, s1
		case s > s2:
			s2 = s
		}
	}
	if s1-s2 < wordMargin {
		return top, s1 - (wordMargin - (s1 - s2))
	}
	return top, s1
}

// captionWords are the words read in the captions found about a page's
// diagrams, line by line, each once (a line found about two diagrams is
// one line).
func captionWords(lines [][][]captionWord, figures []template) []Word {
	var out []Word
	seen := map[[4]int]bool{}
	for _, ls := range lines {
		for _, ln := range ls {
			for _, w := range ln {
				k := [4]int{w.x0, w.y0, w.x1, w.y1}
				if seen[k] {
					continue
				}
				seen[k] = true
				out = append(out, Word{X0: float64(w.x0), Y0: float64(w.y0), X1: float64(w.x1), Y1: float64(w.y1), Text: captionText(w, figures)})
			}
		}
	}
	return out
}

// colourOf is the class of checkers a caption's colour word names: 1 the
// darker, -1 the lighter, 0 none.
func colourOf(s string) int {
	switch strings.ToLower(strings.Trim(s, ".,:;()")) {
	case "black", "dark":
		return 1
	case "white", "light":
		return -1
	}
	return 0
}
