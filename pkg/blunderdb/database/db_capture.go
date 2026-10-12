package database

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
)

// ReadCaptureFile reads the position a screen capture shows (PNG, JPEG or
// GIF). Pure: nothing is stored; the reading goes to the board for correction
// and is saved, once the user agrees with it, through SaveIndividualPosition.
func (d *Database) ReadCaptureFile(path string) (capture.Reading, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return capture.Reading{}, err
	}
	return capture.Decode(data)
}

// ReadCaptureData is ReadCaptureFile for an image the GUI holds in memory —
// pasted from the clipboard — base64-encoded as Wails carries bytes.
func (d *Database) ReadCaptureData(b64 string) (capture.Reading, error) {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return capture.Reading{}, fmt.Errorf("bad image data: %w", err)
	}
	return capture.Decode(data)
}
