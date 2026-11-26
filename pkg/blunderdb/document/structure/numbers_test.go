package structure

import (
	"context"
	"errors"
	"math/rand"
	"strconv"
	"testing"
)

// fontFigure is figure d of the embedded typeface f, as levels 0 to 9.
func fontFigure(f, d int) []int {
	out := make([]int, glyphSide*glyphSide)
	k := (f*10 + d) * 200
	for i := range out {
		b := digitFont[k+i/2]
		if i%2 == 0 {
			b >>= 4
		}
		out[i] = int(b & 15)
	}
	return out
}

// inkNumber draws text, figures of the embedded typeface font, h pixels
// high, centred on (cx, cy): figures printed in the image, not in a text
// layer.
func (c *canvas) inkNumber(text string, cx, cy, h float64, font int) {
	type fig struct {
		lv             []int
		x0, y0, x1, y1 int
	}
	var figs []fig
	total := 0.0
	for _, r := range text {
		lv := fontFigure(font, int(r-'0'))
		f := fig{lv: lv, x0: glyphSide, y0: glyphSide}
		for i, v := range lv {
			if v >= 5 {
				x, y := i%glyphSide, i/glyphSide
				f.x0, f.y0, f.x1, f.y1 = min(f.x0, x), min(f.y0, y), max(f.x1, x+1), max(f.y1, y+1)
			}
		}
		figs = append(figs, f)
		total += float64(f.x1-f.x0) * h / float64(f.y1-f.y0)
	}
	gap := 0.15 * h
	total += gap * float64(len(figs)-1)
	x := cx - total/2
	for _, f := range figs {
		scale := h / float64(f.y1-f.y0)
		w := float64(f.x1-f.x0) * scale
		for ty := 0; ty < int(h); ty++ {
			for tx := 0; tx < int(w); tx++ {
				sx := f.x0 + int(float64(tx)/scale)
				sy := f.y0 + int(float64(ty)/scale)
				if sx < glyphSide && sy < glyphSide && f.lv[sy*glyphSide+sx] >= 5 {
					c.set(int(x)+tx, int(cy-h/2)+ty, 0)
				}
			}
		}
		x += w + gap
	}
}

// TestNumbersReadOrAbstain: numbers drawn in each embedded typeface are read
// as drawn or not at all, never as another number.
func TestNumbersReadOrAbstain(t *testing.T) {
	r := rand.New(rand.NewSource(3))
	c := newCanvas(1400, 1100)
	var want []int
	for f := 0; f < 8; f++ {
		for k := 0; k < 6; k++ {
			v := 1 + r.Intn(167)
			want = append(want, v)
			c.inkNumber(strconv.Itoa(v), float64(80+k*220), float64(80+f*125), 22+float64(k%3)*4, f)
		}
	}
	g := normalize(fromImage(c.image()))
	nums := extractNumbers(g, box{100, 100, 100 + 13*85, 100 + 11*85})
	readNumbers([][]printedNumber{nums})
	read, wrong := 0, 0
	for _, n := range nums {
		if n.val < 0 {
			continue
		}
		read++
		ok := false
		for _, w := range want {
			if w == n.val {
				ok = true
			}
		}
		if !ok {
			wrong++
			t.Errorf("number at (%d,%d) read %d, none drawn", n.x0, n.y0, n.val)
		}
	}
	if len(nums) < len(want) || read < len(want)*3/4 {
		t.Errorf("%d numbers found, %d read, of %d drawn", len(nums), read, len(want))
	}
}

func inkBook(t *testing.T, font int) memPages {
	r := rand.New(rand.NewSource(int64(17 + font)))
	var pages []synthPage
	for i := 0; i < 8; i++ {
		o := drawOpts{p: 36, numbering: true, inkPips: true, inkFont: font}
		pages = append(pages, synthPage{pos: []synthPos{randomPos(r), randomPos(r)}, opts: []drawOpts{o, o}})
	}
	src, _ := book(t, pages, 2)
	for i := range src {
		src[i].Words = nil
	}
	return src
}

// TestPipsReadInImage: without a text layer, the pips printed in the image
// make the diagrams sure; the places are learned on the document.
func TestPipsReadInImage(t *testing.T) {
	src := inkBook(t, 0)
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Profile.PipsInImage {
		t.Fatalf("no pip place learned: %+v", res.Profile)
	}
	sure := 0
	for i, d := range res.Diagrams {
		if d.ImagePips == nil || d.TextPips != nil {
			t.Errorf("diagram %d: pips from the image %v, from the text %v", i+1, d.ImagePips, d.TextPips)
		}
		if d.State == Sure {
			sure++
		}
	}
	if sure < len(res.Diagrams)*3/4 {
		for _, d := range res.Diagrams {
			t.Log(describe(d))
		}
		t.Errorf("%d sure of %d", sure, len(res.Diagrams))
	}
}

// TestPipsTextImageDisagree: when the text layer and the image give
// different pips, the diagram is not sure and the disagreement is named.
func TestPipsTextImageDisagree(t *testing.T) {
	src := inkBook(t, 0)
	// A text layer printing wrong pips beside the first diagram, at the
	// same place as the image's.
	res0, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	d0 := res0.Diagrams[0]
	if d0.ImagePips == nil {
		t.Fatalf("first diagram: no pips read in the image: %s", describe(d0))
	}
	b := d0.Rect
	p := float64(b.Dx()) / 13
	x := float64(b.Min.X) - 1.5*p
	word := func(v int, cy float64) Word { return Word{x - 10, cy - 7, x + 10, cy + 7, strconv.Itoa(v)} }
	H := float64(b.Dy())
	src[0].Words = []Word{word(d0.ImagePips[0]+1, float64(b.Min.Y)+0.2*H), word(d0.ImagePips[1], float64(b.Max.Y)-0.2*H)}
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	d := res.Diagrams[0]
	named := false
	for _, x := range d.Doubts {
		if x.Field == "pips" && d.Printed == nil && d.TextPips != nil {
			named = true
		}
	}
	if d.State == Sure || d.CheckersSure || !named {
		t.Errorf("text and image disagree, yet %s", describe(d))
	}
}

// TestUnreadPage: a page passed over is reported and the others are read;
// a document of such pages only is refused.
func TestUnreadPage(t *testing.T) {
	src := inkBook(t, 0)
	src[1] = Page{Unread: errUnreadTest}
	res, err := Read(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unread) != 1 || res.Unread[0].Page != 1 || len(res.Diagrams) != 14 {
		t.Errorf("unread %v, %d diagrams", res.Unread, len(res.Diagrams))
	}
	if _, err := Read(context.Background(), memPages{{Unread: errUnreadTest}, {Unread: errUnreadTest}}); err == nil || !errors.Is(err, ErrNoPage) {
		t.Errorf("err = %v, want ErrNoPage", err)
	}
}

var errUnreadTest = errors.New("a page of vectors")
