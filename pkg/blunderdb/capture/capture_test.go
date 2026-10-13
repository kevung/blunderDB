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
// checked before anything is allocated.
func TestDecodeRefusesAnOversizedImage(t *testing.T) {
	var buf bytes.Buffer
	img := image.NewPaletted(image.Rect(0, 0, 5000, 5000), color.Palette{color.White, color.Black})
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(buf.Bytes()); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("got %v, want ErrTooLarge", err)
	}
}
