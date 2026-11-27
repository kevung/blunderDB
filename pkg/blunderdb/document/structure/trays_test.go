package structure

import (
	"context"
	"math/rand"
	"strings"
	"testing"
)

// endgamePos is a random position whose players have borne off some
// checkers: bottom loses bOff, top loses tOff, from their highest points.
func endgamePos(r *rand.Rand, bOff, tOff int) synthPos {
	s := randomPos(r)
	for _, x := range []struct {
		side *[25]int
		n    int
	}{{&s.bottom, bOff}, {&s.top, tOff}} {
		for q := 24; q >= 1 && x.n > 0; q-- {
			for x.side[q] > 0 && x.n > 0 {
				x.side[q]--
				x.n--
			}
		}
	}
	return s
}

func trayBook(t *testing.T, extra int) (memPages, []string) {
	return trayBookWith(t, drawOpts{p: 36, numbering: true, tray: true}, extra)
}

func trayBookWith(t *testing.T, base drawOpts, extra int) (memPages, []string) {
	r := rand.New(rand.NewSource(5))
	var pages []synthPage
	for i := 0; i < 5; i++ {
		o := base
		o2 := o
		o2.trayExtra = extra
		pages = append(pages, synthPage{pos: []synthPos{endgamePos(r, 1+i, 3), endgamePos(r, 0, 2+i)}, opts: []drawOpts{o, o2}})
	}
	// Fifteen a side on the first pages, so the checker is learned as usual.
	r2 := rand.New(rand.NewSource(6))
	for i := 0; i < 3; i++ {
		o := drawOpts{p: 36, numbering: true, tray: true}
		pages = append([]synthPage{{pos: []synthPos{randomPos(r2), randomPos(r2)}, opts: []drawOpts{o, o}}}, pages...)
	}
	return book(t, pages, 3)
}

// TestTrayCompletesFifteen: the checkers borne off, drawn as bars in a tray
// beside their home row, make fifteen with those on the board. With printed
// pips that agree, the entry is complete and nothing is named about the tray;
// without them, the tray alone does not complete it and its count is named.
func TestTrayCompletesFifteen(t *testing.T) {
	for _, pips := range []bool{true, false} {
		src, want := trayBookWith(t, drawOpts{p: 36, numbering: true, tray: true, pips: pips}, 0)
		res, err := Read(context.Background(), src)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Diagrams) != len(want) {
			t.Fatalf("pips %v: %d diagrams, want %d", pips, len(res.Diagrams), len(want))
		}
		for i, d := range res.Diagrams {
			if d.Board != want[i] && d.Board != swapCase(want[i]) {
				t.Errorf("pips %v, diagram %d: want %s, got %s", pips, i+1, want[i], describe(d))
			}
			if i < 6 {
				continue
			}
			named := false
			for _, x := range d.Doubts {
				if strings.Contains(x.Reason, "counted in the tray, no printed pips confirm them") {
					named = true
				} else if x.Field == "checkers" {
					t.Errorf("pips %v, diagram %d: doubt %+v", pips, i+1, x)
				}
			}
			if pips && (!d.Complete || !d.CheckersSure || named) {
				t.Errorf("pips agree, diagram %d: drawn and tray make fifteen, not complete and sure: %s", i+1, describe(d))
			}
			if !pips && (d.Complete || d.CheckersSure || !named) {
				t.Errorf("no pips, diagram %d: complete by the tray alone, or the tray not named: %s", i+1, describe(d))
			}
		}
	}
}

// racingAhead is a position whose bottom player has borne off two checkers
// and whose pips, both sides together, are well below those of the board
// read the other way: without printed pips the direction of play assumed is
// the drawn one, and the bottom tray is that player's.
func racingAhead(r *rand.Rand) synthPos {
	for {
		s := endgamePos(r, 2, 0)
		var pips, n int
		for q := 1; q <= 24; q++ {
			pips += q * (s.bottom[q] + s.top[q])
			n += s.bottom[q] + s.top[q]
		}
		if 2*pips+40 < 25*n {
			return s
		}
	}
}

// TestTrayStrayShapeNotComplete: thirteen checkers drawn and, in their tray,
// a shape the bars' reader counts as two — an open "[" or a rectangle with
// dotted sides. Without printed pips the entry is never complete nor sure
// and the tray's count is named; with pips that agree it is complete.
func TestTrayStrayShapeNotComplete(t *testing.T) {
	for _, shape := range []string{"bracket", "dotted"} {
		for _, pips := range []bool{false, true} {
			r := rand.New(rand.NewSource(9))
			var pages []synthPage
			r2 := rand.New(rand.NewSource(6))
			for i := 0; i < 3; i++ {
				o := drawOpts{p: 36, numbering: true, tray: true, pips: pips}
				pages = append(pages, synthPage{pos: []synthPos{randomPos(r2), randomPos(r2)}, opts: []drawOpts{o, o}})
			}
			for i := 0; i < 4; i++ {
				o := drawOpts{p: 36, numbering: true, tray: true, pips: pips, trayShape: shape}
				pages = append(pages, synthPage{pos: []synthPos{racingAhead(r), racingAhead(r)}, opts: []drawOpts{o, o}})
			}
			src, _ := book(t, pages, 9)
			res, err := Read(context.Background(), src)
			if err != nil {
				t.Fatal(err)
			}
			counted := 0
			for i, d := range res.Diagrams {
				if i < 6 {
					continue
				}
				named := false
				for _, x := range d.Doubts {
					if strings.Contains(x.Reason, "2 borne-off checkers counted in the tray, no printed pips confirm them") {
						named = true
					}
				}
				if !pips && (d.Complete || d.CheckersSure) {
					t.Errorf("%s, no pips, diagram %d: complete or sure by a stray shape: %s", shape, i+1, describe(d))
				}
				if !pips && named {
					counted++
				}
				if pips && !d.Complete {
					t.Errorf("%s, pips agree, diagram %d: not complete: %s", shape, i+1, describe(d))
				}
			}
			if !pips && counted < 6 {
				t.Errorf("%s, no pips: %d diagrams name the tray's count, want at least 6 of 8", shape, counted)
			}
		}
	}
}

// TestTrayDisagrees: a tray holding one bar too many is named, never taken
// for the borne-off checkers.
func TestTrayDisagrees(t *testing.T) {
	src, _ := trayBook(t, 1)
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	named := 0
	for i, d := range res.Diagrams {
		if i < 6 || i%2 == 0 {
			continue
		}
		if d.Complete {
			t.Errorf("diagram %d: complete with a tray that does not make fifteen: %s", i+1, describe(d))
		}
		for _, x := range d.Doubts {
			if strings.Contains(x.Reason, "in its tray: they do not make fifteen") {
				named++
			}
		}
	}
	if named < 5 {
		t.Errorf("%d diagrams name the tray's disagreement, want at least 5", named)
	}
}

// TestHollowClasses: a document whose "dark" checkers are as light as a
// shaded point draws no filled checker; every diagram is refused with that
// cause, none counted as a position.
func TestHollowClasses(t *testing.T) {
	r := rand.New(rand.NewSource(9))
	var pages []synthPage
	for i := 0; i < 4; i++ {
		o := drawOpts{p: 36, numbering: true, pips: true, darkFill: 0.75}
		pages = append(pages, synthPage{pos: []synthPos{randomPos(r), randomPos(r)}, opts: []drawOpts{o, o}})
	}
	src, _ := book(t, pages, 4)
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	for i, d := range res.Diagrams {
		if d.State != Uncertain || d.Board != "" || len(d.Doubts) == 0 || d.Doubts[0].Reason != HollowReason {
			t.Errorf("diagram %d: %s", i+1, describe(d))
		}
	}
}

// TestFadedInkNotHollow: a scan whose ink has faded to grey prints its
// filled checkers paler than mid-grey, yet still darker than halfway between
// its ink and its paper: they are filled checkers, read as such, not a
// document without them.
func TestFadedInkNotHollow(t *testing.T) {
	r := rand.New(rand.NewSource(9))
	var pages []synthPage
	for i := 0; i < 4; i++ {
		o := drawOpts{p: 36, numbering: true, pips: true, darkFill: 0.25}
		pages = append(pages, synthPage{pos: []synthPos{randomPos(r), randomPos(r)}, opts: []drawOpts{o, o}, inkGrey: 0.4})
	}
	src, want := book(t, pages, 4)
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagrams) != len(want) {
		t.Fatalf("%d diagrams, want %d", len(res.Diagrams), len(want))
	}
	for i, d := range res.Diagrams {
		for _, x := range d.Doubts {
			if x.Reason == HollowReason {
				t.Fatalf("diagram %d: faded filled checkers taken for hollow ones: %s", i+1, describe(d))
			}
		}
		if d.Board != want[i] && d.Board != swapCase(want[i]) {
			t.Errorf("diagram %d: want %s, got %s", i+1, want[i], describe(d))
		}
	}
}

// TestTrayStrokesNotBars: beside camps of fifteen, hairlines drawn across a
// tray and bars floating off its outer end are not checkers borne off: no
// tray is read, no doubt is raised by it, and the entries stay complete.
func TestTrayStrokesNotBars(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	var pages []synthPage
	for i := 0; i < 5; i++ {
		o := drawOpts{p: 36, numbering: true, tray: true, trayStrokes: true}
		pages = append(pages, synthPage{pos: []synthPos{randomPos(r), randomPos(r)}, opts: []drawOpts{o, o}})
	}
	src, want := book(t, pages, 7)
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Diagrams) != len(want) {
		t.Fatalf("%d diagrams, want %d", len(res.Diagrams), len(want))
	}
	for i, d := range res.Diagrams {
		if d.Board != want[i] && d.Board != swapCase(want[i]) {
			t.Errorf("diagram %d: want %s, got %s", i+1, want[i], describe(d))
		}
		if !d.Complete {
			t.Errorf("diagram %d: strokes in a tray taken for checkers: %s", i+1, describe(d))
		}
	}
}

// TestTrayBesideFifteenNamed: a camp of fifteen drawn whose tray holds bars
// is a contradiction even though the camp alone makes fifteen: that camp is
// named, the other is not, and no entry is complete.
func TestTrayBesideFifteenNamed(t *testing.T) {
	r := rand.New(rand.NewSource(8))
	var pages []synthPage
	for i := 0; i < 5; i++ {
		o := drawOpts{p: 36, numbering: true, tray: true, trayExtraTop: 2}
		pages = append(pages, synthPage{pos: []synthPos{randomPos(r), randomPos(r)}, opts: []drawOpts{o, o}})
	}
	src, _ := book(t, pages, 8)
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	named := 0
	for i, d := range res.Diagrams {
		if d.Complete {
			t.Errorf("diagram %d: complete beside a tray of two bars: %s", i+1, describe(d))
		}
		for _, x := range d.Doubts {
			if !strings.Contains(x.Reason, "in its tray") {
				continue
			}
			if !strings.HasPrefix(x.Reason, "the top player") || !strings.Contains(x.Reason, "15 checkers drawn and 2 in its tray") {
				t.Errorf("diagram %d: wrong camp named: %s", i+1, x.Reason)
			}
			named++
		}
	}
	if named < 8 {
		t.Errorf("%d diagrams name the top camp's tray, want at least 8 of %d", named, len(res.Diagrams))
	}
}

// TestTrayNotMistaken: a cube drawn in a tray's box, and checkers borne off
// drawn as hollow rectangles, are not counted as bars: the tray is not read
// and no entry is complete by it.
func TestTrayNotMistaken(t *testing.T) {
	for _, o := range []drawOpts{
		{p: 36, numbering: true, tray: true, trayCube: true},
		{p: 36, numbering: true, tray: true, trayHollow: true},
	} {
		src, _ := trayBookWith(t, o, 0)
		res, err := Read(context.Background(), src)
		if err != nil {
			t.Fatal(err)
		}
		for i, d := range res.Diagrams {
			if i < 6 {
				continue
			}
			for _, x := range d.Doubts {
				if strings.Contains(x.Reason, "in its tray") {
					t.Errorf("cube %v hollow %v, diagram %d: %+v", o.trayCube, o.trayHollow, i+1, x)
				}
			}
			if o.trayHollow && d.Complete {
				t.Errorf("hollow, diagram %d: complete by a tray of hollow rectangles: %s", i+1, describe(d))
			}
		}
	}
}

// trayMask is a field one point 36 pixels wide whose bottom row has a tray
// on its right: the field's edge and the box's outer stroke, both stopping
// at the board's bottom border.
func trayMask() (*mask, frame) {
	m := newMask(700, 700)
	f := frame{a: 100, b: 316, c: 356, d: 572, y0: 100, y1: 600}
	fillMask(m, 572, 80, 576, 604)
	fillMask(m, 612, 80, 616, 604)
	fillMask(m, 100, 600, 616, 604)
	return m, f
}

func fillMask(m *mask, x0, y0, x1, y1 int) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			m.p[y*m.w+x] = true
		}
	}
}

// TestTraysInsetRectangle: a hollow rectangle drawn inset in a tray,
// against its outer end, has two edges like two bars; its sides stand
// inside the band, away from its strokes, and still make it a rectangle,
// never two checkers borne off.
func TestTraysInsetRectangle(t *testing.T) {
	m, f := trayMask()
	fillMask(m, 583, 585, 606, 588)
	fillMask(m, 583, 595, 606, 598)
	fillMask(m, 583, 588, 586, 595)
	fillMask(m, 603, 588, 606, 595)
	if got := trays(m, 0, 0, f, nil, 0, 0); got != [2]int{} {
		t.Fatalf("inset hollow rectangle read as bars: %v", got)
	}
}

// TestTraysCubeOverBars: regular bars against the tray's outer end are read,
// unless a cube found by the squares overlaps the band: then the tray is not
// read at all.
func TestTraysCubeOverBars(t *testing.T) {
	m, f := trayMask()
	for k := 0; k < 3; k++ {
		fillMask(m, 579, 591-7*k, 610, 595-7*k)
	}
	if got := trays(m, 0, 0, f, nil, 0, 0); got != [2]int{0, 3} {
		t.Fatalf("three bars read %v, want [0 3]", got)
	}
	cube := square{place: "bottom", x0: 578, y0: 560, x1: 611, y1: 593}
	if got := trays(m, 0, 0, f, []square{cube}, 0, 0); got != [2]int{} {
		t.Fatalf("tray under a cube read %v", got)
	}
}

// TestTrayDiagramsVoteForPipPlaces: in a document whose every diagram has
// checkers borne off in its trays, the pips printed in the image are still
// located — a borne-off checker counts no pip, so such diagrams vote for the
// pips' places — and, confirmed by them, the diagrams are complete.
func TestTrayDiagramsVoteForPipPlaces(t *testing.T) {
	r := rand.New(rand.NewSource(23))
	var pages []synthPage
	for i := 0; i < 8; i++ {
		o := drawOpts{p: 36, numbering: true, tray: true, inkPips: true}
		pages = append(pages, synthPage{pos: []synthPos{endgamePos(r, 2, 1), endgamePos(r, 1, 3)}, opts: []drawOpts{o, o}})
	}
	src, _ := book(t, pages, 5)
	for i := range src {
		src[i].Words = nil
	}
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Profile.PipsInImage {
		t.Fatalf("no pip place learned on diagrams completed by their trays: %+v", res.Profile)
	}
	complete := 0
	for _, d := range res.Diagrams {
		if d.Complete {
			complete++
		}
	}
	if complete < len(res.Diagrams)*3/4 {
		for _, d := range res.Diagrams {
			t.Log(describe(d))
		}
		t.Errorf("%d complete of %d", complete, len(res.Diagrams))
	}
}
