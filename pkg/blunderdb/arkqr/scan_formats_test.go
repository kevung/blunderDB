package arkqr

import (
	"bytes"
	"image"
	"io"
	"sync/atomic"
	"testing"
)

var trapDecoded atomic.Bool

// registerTrap registers, under JPEG 2000's own signature, a decoder that
// records being called: whatever another package of the process registers,
// a sheet brought by a request must not reach it.
func registerTrap() []byte {
	sig := "\x00\x00\x00\x0cjP  \r\n\x87\n"
	image.RegisterFormat("jp2", sig, func(io.Reader) (image.Image, error) {
		trapDecoded.Store(true)
		return image.NewGray(image.Rect(0, 0, 1, 1)), nil
	}, func(io.Reader) (image.Config, error) {
		return image.Config{ColorModel: image.NewGray(image.Rect(0, 0, 1, 1)).ColorModel(), Width: 1, Height: 1}, nil
	})
	return []byte(sig + "rest of a JP2 file")
}

func TestScanImageOnlyPNGAndJPEG(t *testing.T) {
	data := registerTrap()
	if _, err := ScanImage(bytes.NewReader(data)); err == nil {
		t.Error("a JP2 file is scanned")
	}
	if trapDecoded.Load() {
		t.Error("a registered JPEG 2000 decoder is reached")
	}
}
