package jpx

import (
	"context"
	"errors"
	"fmt"
	"image"

	"github.com/kevung/blunderdb/pkg/blunderdb/document/jpx/internal/j2k"
)

// MaxPixels is the largest image, in pixels of its reference grid.
const MaxPixels = j2k.MaxPixels

var (
	// ErrUnsupported: a valid JPEG 2000 image this reader does not take.
	ErrUnsupported = errors.New("jpeg 2000: unsupported")
	// ErrMalformed: bytes that are not a JPEG 2000 image.
	ErrMalformed = errors.New("jpeg 2000: malformed")
	// ErrBudget: the pixels a document may decode in JPEG 2000 are spent.
	ErrBudget = errors.New("jpeg 2000: pixel budget of the document exhausted")
)

func unsupported(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrUnsupported}, a...)...)
}

func malformed(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrMalformed}, a...)...)
}

// Info is what the headers of an image say.
type Info struct {
	Width, Height int
	Components    int
	// Tiles is the number of tiles; Reversible is true when the first
	// coding style names the 5/3 transform.
	Tiles      int
	Reversible bool
}

// Budget bounds the pixels a document decodes in JPEG 2000, whatever its
// caller's deadline. The zero value allows nothing; a nil *Budget allows
// everything.
type Budget struct{ Left int64 }

// Decode parses data, spends its pixels from budget, and decodes it,
// checking ctx as it goes. width and height, when positive, are the
// dimensions the container announces: another reference grid is refused.
func Decode(ctx context.Context, data []byte, width, height int, budget *Budget) (image.Image, Info, error) {
	im, info, err := parse(data)
	if err != nil {
		return nil, info, err
	}
	if (width > 0 && width != info.Width) || (height > 0 && height != info.Height) {
		return nil, info, unsupported("the image is %d×%d, its container says %d×%d", info.Width, info.Height, width, height)
	}
	if budget != nil {
		n := int64(info.Width) * int64(info.Height)
		if n > budget.Left {
			return nil, info, ErrBudget
		}
		budget.Left -= n
	}
	img, err := decodeSafe(ctx, im)
	return img, info, err
}

// Inspect parses the headers of data and refuses what the package doc lists.
func Inspect(data []byte) (Info, error) {
	_, info, err := parse(data)
	return info, err
}

func parse(data []byte) (im *j2k.Image, info Info, err error) {
	defer func() {
		if e := recover(); e != nil {
			im, err = nil, malformed("%v", e)
		}
	}()
	im, err = j2k.Parse(data)
	if err != nil {
		return nil, info, classify(err)
	}
	info = Info{Width: im.Width(), Height: im.Height(), Components: im.Components(), Tiles: im.Tiles(), Reversible: im.Reversible()}
	return im, info, nil
}

func decodeSafe(ctx context.Context, im *j2k.Image) (img image.Image, err error) {
	defer func() {
		if e := recover(); e != nil {
			img, err = nil, malformed("%v", e)
		}
	}()
	img, err = im.Decode(ctx)
	if err != nil {
		return nil, classify(err)
	}
	return img, nil
}

func classify(err error) error {
	var u j2k.UnsupportedError
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.As(err, &u):
		return unsupported("%v", err)
	default:
		return malformed("%v", err)
	}
}
