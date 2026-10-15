package capture

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // decoders for image.Decode
	_ "image/jpeg" // a capture pasted from a browser is often JPEG
	_ "image/png"
	"slices"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// Software names the program whose board a capture shows.
type Software string

const (
	SoftwareXG Software = "xg"
)

// Doubt is one thing the reading could not settle. The correction step shows
// it on the board; nothing doubtful is ever settled by a guess the user does
// not see.
type Doubt struct {
	// Field is what is in doubt: "point", "bar", "dice", "cube", "turn",
	// "orientation" or "checkers" (a colour's total is not 15 on the board
	// and off it).
	Field string `json:"field"`
	// Point is the blunderDB index (1..24, 0 and 25 for the bars) when Field
	// is "point" or "bar".
	Point int `json:"point"`
	// Reason is a short English phrase for logs and the CLI.
	Reason string `json:"reason"`
}

// Reading is what Read made of a capture.
type Reading struct {
	Position domain.Position `json:"position"`
	Software Software        `json:"software"`
	// Fit is the share of the board's landmarks that agree with the theme,
	// 0..1; below MinFit the capture is refused.
	Fit float64 `json:"fit"`
	// Board is where the playing area was found, in the capture's pixels.
	Board image.Rectangle `json:"board"`
	// Unread lists what a capture never shows and the user must set: the
	// score, and the turn when no dice are on the board.
	Unread []string `json:"unread"`
	Doubts []Doubt  `json:"doubts"`
}

// SetScore settles the score a capture never shows: two Away scores, player
// 1's first, or {-1, -1} for money play.
func (rd *Reading) SetScore(score [2]int) {
	rd.Position.Score = score
	rd.Unread = slices.DeleteFunc(rd.Unread, func(f string) bool { return f == "score" })
}

// SetTurn settles the player on roll (domain.Black or domain.White). Dice on
// the board already name it by their colour; the user's word still wins.
func (rd *Reading) SetTurn(player int) {
	rd.Position.PlayerOnRoll = player
	rd.Unread = slices.DeleteFunc(rd.Unread, func(f string) bool { return f == "turn" })
}

// MinFit is the fit below which no board is deemed found.
const MinFit = 0.6

// ErrNoBoard says that no known board was found in the image.
var ErrNoBoard = errors.New("no backgammon board recognised in the image")

// MaxPixels bounds the image a reading works on. A reading holds about 35
// bytes per pixel at its peak — its RGB copy and three summed-area tables of
// int64 — so the bound is a whole 4K screen (3840x2160), about 290 MB. Read
// shrinks a larger image below it; a board drawn at a readable size stays
// readable at half or a third of a 5K or two-screen capture.
const MaxPixels = 3840 * 2160

// readPixels is the size past which Read shrinks its input; a variable so that
// a test can exercise the shrinking on a small image.
var readPixels = MaxPixels

// MaxDecodePixels bounds what Decode accepts at all, checked on the header
// since a small file can declare billions (a 4 KB PNG of 6000x6000): two 6K
// screens side by side take 41 Mpx, and the decoded image weighs 4 to 8
// bytes per pixel before it is shrunk.
const MaxDecodePixels = 50_000_000

// ErrTooLarge says that the image declares more pixels than the reader takes.
var ErrTooLarge = errors.New("image too large")

// Decode reads an encoded image (PNG, JPEG or GIF), shrinking one larger
// than MaxPixels before reading it.
func Decode(data []byte) (Reading, error) {
	return decode(data, MaxDecodePixels)
}

// DecodeBounded is Decode for a shared server: it refuses an image larger
// than MaxPixels instead of decoding and shrinking it, so that no request
// holds more than one reading's memory.
func DecodeBounded(data []byte) (Reading, error) {
	return decode(data, MaxPixels)
}

func decode(data []byte, limit int) (Reading, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Reading{}, fmt.Errorf("decode image: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > limit/cfg.Height {
		return Reading{}, fmt.Errorf("%w: %dx%d, this reader takes at most %d pixels", ErrTooLarge, cfg.Width, cfg.Height, limit)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Reading{}, fmt.Errorf("decode image: %w", err)
	}
	return Read(img)
}

// shrinkFactor is the smallest integer f that brings a w×h image within
// readPixels (MaxPixels unless a test lowers it) once each f×f block is averaged into one pixel.
func shrinkFactor(w, h int) int {
	f := 1
	for ((w+f-1)/f)*((h+f-1)/f) > readPixels {
		f++
	}
	return f
}

// Read recognises the position a screen capture shows.
func Read(img image.Image) (Reading, error) {
	f := shrinkFactor(img.Bounds().Dx(), img.Bounds().Dy())
	r := newRaster(img, f)
	var best Reading
	bestFit := -1.0
	for _, t := range themes {
		g, fit := t.locate(r)
		if fit > bestFit {
			bestFit = fit
			best = t.read(r, g)
			best.Fit = fit
		}
	}
	if bestFit < MinFit {
		return Reading{}, ErrNoBoard
	}
	// The board is reported in the capture's own pixels, not the shrunk ones.
	best.Board = image.Rectangle{Min: best.Board.Min.Mul(f), Max: best.Board.Max.Mul(f)}
	return best, nil
}
