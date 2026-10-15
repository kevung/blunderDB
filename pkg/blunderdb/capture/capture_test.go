package capture

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func readFixture(t *testing.T, name string) Reading {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	rd, err := Decode(data)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return rd
}

// The fixtures are eXtreme Gammon boards cut out of a screen recording,
// one at its size on screen, one enlarged by half: the board is found at
// whatever scale the capture has.
func TestReadXG(t *testing.T) {
	cases := []struct {
		file  string
		board string
		dice  [2]int
		cube  domain.Cube
		roll  int
		// The enlarged capture blurs the pips of its dice; the point
		// numbers are on both, so the board's direction is never in doubt.
		doubts []string
	}{
		{"xg-dice62.jpg", "a--abBC-C--AbDa-aaAd-b--A-", [2]int{6, 2}, domain.Cube{Owner: domain.None, Value: 0}, domain.Black, nil},
		{"xg-cube2-scaled.jpg", "b-bAbBC-B--AbDa-AA-d-b----", [2]int{}, domain.Cube{Owner: domain.White, Value: 1}, -1, []string{"dice"}},
	}
	for _, c := range cases {
		rd := readFixture(t, c.file)
		if rd.Software != SoftwareXG {
			t.Errorf("%s: software %q", c.file, rd.Software)
		}
		if got := domain.EncodeXGIDBoard(&rd.Position); got != c.board {
			t.Errorf("%s: board %s, want %s (doubts %v)", c.file, got, c.board, rd.Doubts)
		}
		if c.dice != [2]int{} && rd.Position.Dice != c.dice {
			t.Errorf("%s: dice %v, want %v", c.file, rd.Position.Dice, c.dice)
		}
		if c.roll >= 0 && rd.Position.PlayerOnRoll != c.roll {
			t.Errorf("%s: on roll %d, want %d", c.file, rd.Position.PlayerOnRoll, c.roll)
		}
		if rd.Position.Cube != c.cube {
			t.Errorf("%s: cube %+v, want %+v", c.file, rd.Position.Cube, c.cube)
		}
		// Borne-off checkers are what the board lacks of fifteen.
		for col := 0; col < 2; col++ {
			n := rd.Position.Board.Bearoff[col]
			for _, p := range rd.Position.Board.Points {
				if p.Checkers > 0 && p.Color == col {
					n += p.Checkers
				}
			}
			if n != 15 {
				t.Errorf("%s: colour %d totals %d checkers", c.file, col, n)
			}
		}
		if len(rd.Unread) == 0 || rd.Unread[0] != "score" {
			t.Errorf("%s: the score must be left to the user, unread = %v", c.file, rd.Unread)
		}
		var fields []string
		for _, d := range rd.Doubts {
			fields = append(fields, d.Field)
		}
		if !slices.Equal(fields, c.doubts) {
			t.Errorf("%s: doubts %v, want %v", c.file, rd.Doubts, c.doubts)
		}
	}
}

// Cropped just under the lower row, the capture loses the point numbers:
// the board's direction is then the theme's guess, and the reading says so.
func TestReadDoubtsTheOrientationWithoutPointNumbers(t *testing.T) {
	data, err := os.ReadFile("testdata/xg-dice62.jpg")
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	cropped := img.(interface {
		SubImage(image.Rectangle) image.Image
	}).SubImage(image.Rect(b.Min.X, b.Min.Y, b.Max.X, b.Min.Y+266))
	rd, err := Read(cropped)
	if err != nil {
		t.Fatal(err)
	}
	if got := domain.EncodeXGIDBoard(&rd.Position); got != "a--abBC-C--AbDa-aaAd-b--A-" {
		t.Errorf("board %s: the theme's default direction is the right one here", got)
	}
	var fields []string
	for _, d := range rd.Doubts {
		fields = append(fields, d.Field)
	}
	if !slices.Equal(fields, []string{"orientation"}) {
		t.Errorf("doubts %v, want the orientation alone", rd.Doubts)
	}
}

func TestReadRefusesAnImageWithoutBoard(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			img.Set(x, y, color.RGBA{uint8(x / 2), uint8(y), 120, 255})
		}
	}
	if _, err := Read(img); !errors.Is(err, ErrNoBoard) {
		t.Fatalf("got %v, want ErrNoBoard", err)
	}
}

// A few kilobytes of PNG can declare a canvas of a gigabyte: the header is
// checked before anything is allocated. The server refuses beyond what a
// reading holds; the desktop only beyond what no screen produces.
func TestDecodeRefusesAnOversizedImage(t *testing.T) {
	pngOf := func(w, h int) []byte {
		var buf bytes.Buffer
		img := image.NewPaletted(image.Rect(0, 0, w, h), color.Palette{color.White, color.Black})
		if err := png.Encode(&buf, img); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	if _, err := DecodeBounded(pngOf(4000, 2200)); !errors.Is(err, ErrTooLarge) {
		t.Errorf("bounded: got %v, want ErrTooLarge", err)
	}
	if _, err := Decode(pngOf(8000, 8000)); !errors.Is(err, ErrTooLarge) {
		t.Errorf("desktop: got %v, want ErrTooLarge", err)
	}
}

// A capture larger than a 4K screen — a 5K display, two screens — is shrunk
// before it is read, and its board still reported in its own pixels.
func TestReadShrinksALargeCapture(t *testing.T) {
	data, err := os.ReadFile("testdata/xg-dice62.jpg")
	if err != nil {
		t.Fatal(err)
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	small, err := Read(src)
	if err != nil {
		t.Fatal(err)
	}
	// The bound is lowered so that an enlargement is past it: the shrinking
	// is the same at any size, and a real 5K capture costs minutes under -race.
	defer func(v int) { readPixels = v }(readPixels)
	cases := []struct {
		k, readPixels int
	}{
		{4, 300_000}, // 1416x1096 shrunk by 3: a 1.33x scale
		// 1062x822 shrunk by 2, a 1.5x scale: every other pixel of the
		// shrunk board blends two of the original, which once left the
		// board placed a few pixels small (the checkers of two points and
		// the dice missed) and the dice's spots counted short.
		{3, 250_000},
	}
	for _, c := range cases {
		readPixels = c.readPixels
		b := src.Bounds()
		big := image.NewRGBA(image.Rect(0, 0, b.Dx()*c.k, b.Dy()*c.k))
		for y := 0; y < big.Rect.Dy(); y++ {
			for x := 0; x < big.Rect.Dx(); x++ {
				big.Set(x, y, src.At(b.Min.X+x/c.k, b.Min.Y+y/c.k))
			}
		}
		rd, err := Read(big)
		if err != nil {
			t.Fatal(err)
		}
		if got := domain.EncodeXGIDBoard(&rd.Position); got != "a--abBC-C--AbDa-aaAd-b--A-" {
			t.Errorf("x%d: board %s (doubts %v)", c.k, got, rd.Doubts)
		}
		if rd.Position.Dice != [2]int{6, 2} || len(rd.Doubts) != 0 {
			t.Errorf("x%d: dice %v, doubts %v", c.k, rd.Position.Dice, rd.Doubts)
		}
		// Located afresh at another scale, the board may move by a pixel or
		// two of the original: within 2 % of its width.
		want := small.Board.Min.Mul(c.k)
		tol := rd.Board.Dx() / 50
		if d := rd.Board.Min.Sub(want); d.X*d.X+d.Y*d.Y > tol*tol {
			t.Errorf("x%d: board found at %v, want near %v", c.k, rd.Board, want)
		}
	}
}

// A die whose spots were miscounted is unread, never misread: every face is
// symmetric about its centre, which holds a spot exactly when the count is odd.
func TestPointSymmetricRejectsAMiscountedDie(t *testing.T) {
	six := []image.Point{{0, 0}, {0, 5}, {0, 10}, {8, 0}, {8, 5}, {8, 10}}
	cases := []struct {
		name  string
		spots []image.Point
		want  bool
	}{
		{"six", six, true},
		{"five", []image.Point{{0, 0}, {0, 10}, {4, 5}, {8, 0}, {8, 10}}, true},
		{"one", []image.Point{{4, 5}}, true},
		{"six less one", six[1:], false},
		{"six less two of a column", six[2:], false},
		{"four plus a stray", []image.Point{{0, 0}, {0, 10}, {8, 0}, {8, 10}, {2, 5}}, false},
	}
	for _, c := range cases {
		if got := pointSymmetric(c.spots, 1.5); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}
