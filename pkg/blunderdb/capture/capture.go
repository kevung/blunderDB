package capture

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // decoders for image.Decode
	_ "image/jpeg" // a capture pasted from a browser is often JPEG
	_ "image/png"

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

// MinFit is the fit below which no board is deemed found.
const MinFit = 0.6

// ErrNoBoard says that no known board was found in the image.
var ErrNoBoard = errors.New("no backgammon board recognised in the image")

// MaxPixels bounds the image a reading will decode: a screen capture is at
// most a few megapixels, and a small file can declare billions (a 4 KB PNG
// of 6000x6000 takes over a gigabyte), so the size is checked on the header.
const MaxPixels = 16_000_000

// ErrTooLarge says that the image declares more than MaxPixels.
var ErrTooLarge = errors.New("image too large to be a screen capture")

// Decode reads an encoded image (PNG, JPEG or GIF).
func Decode(data []byte) (Reading, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Reading{}, fmt.Errorf("decode image: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > MaxPixels/cfg.Height {
		return Reading{}, fmt.Errorf("%w: %dx%d, at most %d pixels", ErrTooLarge, cfg.Width, cfg.Height, MaxPixels)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Reading{}, fmt.Errorf("decode image: %w", err)
	}
	return Read(img)
}

// Read recognises the position a screen capture shows.
func Read(img image.Image) (Reading, error) {
	r := newRaster(img)
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
	return best, nil
}
