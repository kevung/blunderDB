package structure

import (
	"image"
	"image/draw"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// captionPage draws text, in a typeface the reader holds no drawing of, on a
// white page under the box {100, 100, 750, 400}: a board 650 pixels wide.
func captionPage(t *testing.T, ttf []byte, text string) (*raster, box) {
	t.Helper()
	f, err := opentype.Parse(ttf)
	if err != nil {
		t.Fatal(err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: 30, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewGray(image.Rect(0, 0, 900, 600))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)
	d := font.Drawer{Dst: img, Src: image.Black, Face: face, Dot: fixed.P(200, 460)}
	d.DrawString(text)
	return normalize(fromImage(img)), box{100, 100, 750, 400}
}

// TestCaptionInImage: a caption drawn under a diagram is read word by word —
// the colour, "to play", the roll — and a word of no caption's reads "?".
func TestCaptionInImage(t *testing.T) {
	for _, c := range []struct {
		ttf  []byte
		name string
	}{{goregular.TTF, "regular"}, {gobold.TTF, "bold"}} {
		g, b := captionPage(t, c.ttf, "Problem 12: Black to play 54.")
		var got []string
		for _, w := range captionWords([][][]captionWord{extractCaptions(g, b)}, captionFigures(nil)) {
			got = append(got, w.Text)
		}
		if s := strings.Join(got, " "); s != "? 12 black to play 54" {
			t.Errorf("%s: caption read %q", c.name, s)
		}
	}
}

// TestCaptionDice: a caption gives a diagram that draws no dice its roll and
// player on roll, both doubts; a roll not read, or two rolls, give the doubt
// alone.
func TestCaptionDice(t *testing.T) {
	ws := func(ts ...string) []Word {
		var out []Word
		for i, x := range ts {
			out = append(out, Word{X0: float64(150 + 40*i), X1: float64(185 + 40*i), Y0: 680, Y1: 700, Text: x})
		}
		return out
	}
	read := func(words []Word, bottomDark bool) (Context, map[string]string) {
		rd := handBoard(0, 0)
		var d Diagram
		readContext(rd, &d, words, Profile{BottomIsDark: bottomDark}, contextEnv{})
		return d.Context, doubtsOn(d)
	}
	x, why := read(ws("Problem", "12:", "Black", "to", "play", "54."), true)
	if x.Dice != [2]int{5, 4} || x.DiceRead || x.Turn != 1 || x.TurnRead || x.Read || !strings.Contains(why["dice"], "one source") || why["turn"] == "" {
		t.Errorf("caption 54 to Black, Black at the bottom: %+v, doubts %v", x, why)
	}
	x, _ = read(ws("White", "rolls", "a", "31"), true)
	if x.Dice != [2]int{3, 1} || x.Turn != -1 {
		t.Errorf("caption 31 to White, Black at the bottom: %+v", x)
	}
	x, why = read(ws("Black", "to", "play", "?"), false)
	if x.Dice != [2]int{} || x.Turn != -1 || !strings.Contains(why["dice"], "not read") {
		t.Errorf("roll not read: %+v, doubts %v", x, why)
	}
	two := append(ws("Black", "to", "play", "54"), ws("White", "to", "play", "21")...)
	for i := 4; i < 8; i++ {
		two[i].Y0, two[i].Y1 = 710, 730
	}
	x, why = read(two, true)
	if x.Dice != [2]int{} || x.Turn != 0 || !strings.Contains(why["dice"], "different rolls") {
		t.Errorf("two captions: %+v, doubts %v", x, why)
	}
	// Prose is no caption: the colour word inside a sentence, the roll
	// before the line's end.
	for _, prose := range [][]string{{"if", "White", "rolls", "15"}, {"White", "rolls", "15", "and", "plays"}} {
		x, why = read(ws(prose...), true)
		if x.Dice != [2]int{} || x.Turn != 0 || len(why) != 0 {
			t.Errorf("%q read as a caption: %+v, doubts %v", prose, x, why)
		}
	}
	// A caption nearer another diagram of the page is that diagram's.
	rd, other := handBoard(0, 0), handBoard(0, 700)
	var d Diagram
	far := ws("Black", "to", "play", "54")
	for i := range far {
		far[i].Y0, far[i].Y1 = 745, 760
	}
	readContext(rd, &d, far, Profile{BottomIsDark: true}, contextEnv{others: []*boardReading{other}})
	if d.Context.Dice != [2]int{} || len(d.Doubts) != 0 {
		t.Errorf("another diagram's caption read: %+v, doubts %v", d.Context, d.Doubts)
	}
}

// TestScoreAsOCR: a score panel read as an OCR layer gives it — a tag glued
// to its number, a colon read as a semicolon, "point" cut in two — and equal
// scores read without the pips that assign them.
func TestScoreAsOCR(t *testing.T) {
	col := func(lines ...[]string) []Word {
		var out []Word
		for i, l := range lines {
			for k, x := range l {
				out = append(out, Word{X0: float64(780 + 30*k), X1: float64(805 + 30*k), Y0: float64(120 + 60*i), Y1: float64(135 + 60*i), Text: x})
			}
		}
		return out
	}
	read := func(words []Word) (*Score, string) {
		rd := handBoard(0, 0)
		d := Diagram{Pips: [2]int{150, 160}}
		readContext(rd, &d, words, Profile{}, contextEnv{})
		return d.Context.Score, doubtsOn(d)["score"]
	}
	s, why := read(col([]string{"score;", "2"}, []string{"pip:160"}, []string{"7", "po", "int", "match"}, []string{"pip:", "150"}, []string{"score:", "4"}))
	if s == nil || *s != (Score{Length: 7, Bottom: 4, Top: 2}) || why != "" {
		t.Errorf("OCR panel: %+v, doubt %q", s, why)
	}
	s, why = read(col([]string{"score:", "0"}, []string{"pip:", "160"}, []string{"7", "point", "match"}, []string{"score:", "0"}))
	if s == nil || *s != (Score{Length: 7}) || why != "" {
		t.Errorf("equal scores, one pip count of the diagram's: %+v, doubt %q", s, why)
	}
	// A pip count neither player has: the panel may be another's, or a
	// figure misread (3 read 8 makes 8-8 of 3-8).
	s, why = read(col([]string{"score:", "8"}, []string{"pip:", "89"}, []string{"7", "point", "match"}, []string{"pip:", "150"}, []string{"score:", "8"}))
	if s != nil || !strings.Contains(why, "neither player has") {
		t.Errorf("equal scores beside a foreign pip count: %+v, doubt %q", s, why)
	}
	s, why = read(col([]string{"score:", "0"}, []string{"7", "point", "match"}, []string{"score:", "0"}))
	if s != nil || !strings.Contains(why, "one source") {
		t.Errorf("equal scores with no pip count: %+v, doubt %q", s, why)
	}
	s, why = read(col([]string{"score:", "1"}, []string{"pip:", "89"}, []string{"7", "point", "match"}, []string{"score:", "0"}))
	if s != nil || !strings.Contains(why, "not read whole") {
		t.Errorf("unequal scores, one pip count: %+v, doubt %q", s, why)
	}
}
