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
// beside their home row, make fifteen with those on the board: the entry is
// complete, with no supposition about the tray.
func TestTrayCompletesFifteen(t *testing.T) {
	src, want := trayBook(t, 0)
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
			t.Errorf("diagram %d: drawn and tray make fifteen, not complete: %s", i+1, describe(d))
		}
		for _, x := range d.Doubts {
			if x.Field == "checkers" {
				t.Errorf("diagram %d: doubt %+v", i+1, x)
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
