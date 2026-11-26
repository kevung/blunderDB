package arkqr

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // a photographed sheet
	_ "image/png"  // a scanned sheet
	"io"
	"regexp"
	"strings"

	"github.com/makiuchi-d/gozxing"
	multiqr "github.com/makiuchi-d/gozxing/multi/qrcode"
	qrgen "github.com/skip2/go-qrcode"
)

// Modules returns the frame's QR code, error correction level M, as rows of
// dark (true) and light modules, its quiet zone included.
func Modules(f Frame) ([][]bool, error) {
	q, err := qrgen.New(f.Text(), qrgen.Medium)
	if err != nil {
		return nil, fmt.Errorf("arkqr: QR code of frame %d: %w", f.Index, err)
	}
	return q.Bitmap(), nil
}

// ScanImage reads every archive code an image shows: a scan or a photograph
// of one or several sheets, PNG or JPEG. A code that is not an archive's is
// skipped; finding none is not an error.
func ScanImage(r io.Reader) ([]Frame, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("arkqr: image: %w", err)
	}
	return scan(img)
}

func scan(img image.Image) ([]Frame, error) {
	seen := map[string]bool{}
	var out []Frame
	read := func(img image.Image) {
		bmp, err := gozxing.NewBinaryBitmapFromImage(img)
		if err != nil {
			return
		}
		hints := map[gozxing.DecodeHintType]any{gozxing.DecodeHintType_TRY_HARDER: true}
		res, err := multiqr.NewQRCodeMultiReader().DecodeMultiple(bmp, hints)
		if err != nil {
			return
		}
		for _, r := range res {
			if seen[r.GetText()] {
				continue
			}
			seen[r.GetText()] = true
			if f, err := ParseText(r.GetText()); err == nil {
				out = append(out, f)
			}
		}
	}
	read(img)
	// The finder-pattern search gives up on a crowded sheet; read again the
	// quadrants and the overlapping halves, where fewer codes compete.
	sub, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return out, nil
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	for _, frac := range [][4]int{
		{0, 0, 2, 2}, {1, 0, 2, 2}, {0, 1, 2, 2}, {1, 1, 2, 2},
		{0, 0, 1, 2}, {0, 1, 1, 2}, {0, 0, 2, 1}, {1, 0, 2, 1},
	} {
		// x, y in halves; the window spans 1/frac[2] of the width, 1/frac[3] of the height.
		x0 := b.Min.X + frac[0]*w/2
		y0 := b.Min.Y + frac[1]*h/2
		r := image.Rect(x0, y0, x0+w/frac[2], y0+h/frac[3]).Inset(-w / 20).Intersect(b)
		read(sub.SubImage(r))
	}
	return out, nil
}

// payloadRe finds the frames a page embeds, base64 one per line.
var payloadRe = regexp.MustCompile(`(?s)<script type="application/vnd\.blunderdb\.archive"[^>]*>(.*?)</script>`)

// ErrNoPayload is an HTML file that is not an archive page.
var ErrNoPayload = errors.New("arkqr: no archive in this page")

// ReadPage returns the frames an archive page embeds, without reading a code.
func ReadPage(page []byte) ([]Frame, error) {
	var out []Frame
	found := false
	for _, m := range payloadRe.FindAllSubmatch(page, -1) {
		found = true
		for _, line := range strings.Fields(string(m[1])) {
			b, err := base64.StdEncoding.DecodeString(line)
			if err != nil {
				return nil, fmt.Errorf("%w: %w", ErrFrame, err)
			}
			f, err := ParseFrame(b)
			if err != nil {
				return nil, err
			}
			out = append(out, f)
		}
	}
	if !found {
		return nil, ErrNoPayload
	}
	return out, nil
}

// IsPage reports whether an HTML text is an archive page.
func IsPage(page []byte) bool { return payloadRe.Match(page) }

// svgPath draws the dark modules as one path, a rectangle per run.
func svgPath(mods [][]bool) string {
	var b bytes.Buffer
	for y, row := range mods {
		for x := 0; x < len(row); {
			if !row[x] {
				x++
				continue
			}
			e := x
			for e < len(row) && row[e] {
				e++
			}
			fmt.Fprintf(&b, "M%d %dh%dv1h-%dz", x, y, e-x, e-x)
			x = e
		}
	}
	return b.String()
}
