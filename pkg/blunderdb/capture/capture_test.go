package capture

import (
	"errors"
	"image"
	"image/color"
	"os"
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
	}{
		{"xg-dice62.jpg", "a--abBC-C--AbDa-aaAd-b--A-", [2]int{6, 2}, domain.Cube{Owner: domain.None, Value: 0}, domain.Black},
		{"xg-cube2-scaled.jpg", "b-bAbBC-B--AbDa-AA-d-b----", [2]int{}, domain.Cube{Owner: domain.White, Value: 1}, -1},
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
