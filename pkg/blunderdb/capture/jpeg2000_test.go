package capture

import (
	"os"
	"testing"
)

// An image brought by a request is never read in JPEG 2000: that decoder is
// not registered for the process, and only PNG, JPEG and GIF pass.
func TestDecodeRefusesJPEG2000(t *testing.T) {
	data, err := os.ReadFile("../document/jpx/testdata/lossy97.jp2")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeBounded(data); err == nil {
		t.Error("a JPEG 2000 image is read as a position image")
	}
}
