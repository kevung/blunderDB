package ingest

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/xgparser/xgparser"
)

// ErrXGHeader is returned when an XG container's header announces sizes its
// file cannot hold.
var ErrXGHeader = fmt.Errorf("%w: ingest: malformed XG header", storage.ErrInvalid)

// xgHeaderPrefix is the part of the Game Data Format header that carries the
// two sizes xgparser allocates from: magic, version, header size, thumbnail
// offset, thumbnail size — little-endian, as xgparser reads them.
const xgHeaderPrefix = 4 + 4 + 4 + 8 + 4

// checkXGHeader refuses an .xg/.xgp file whose header size or thumbnail size
// exceeds the file itself. xgparser allocates both straight from the header,
// so a 70 KB file announcing a 4 GB thumbnail takes the process down before
// any read can fail. A file too short to hold the header is left to
// xgparser, which reports it as not a game file.
func checkXGHeader(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	var hdr [xgHeaderPrefix]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return nil
	}
	size := st.Size()
	headerSize := int64(int32(binary.LittleEndian.Uint32(hdr[8:12])))
	thumbSize := int64(binary.LittleEndian.Uint32(hdr[20:24]))
	if headerSize < 0 || headerSize > size {
		return fmt.Errorf("%w: header size %d, file size %d", ErrXGHeader, headerSize, size)
	}
	if thumbSize > size {
		return fmt.Errorf("%w: thumbnail size %d, file size %d", ErrXGHeader, thumbSize, size)
	}
	return nil
}

// wrapInvalid marks a parser's refusal of its input as storage.ErrInvalid, so
// a client sees a file it must fix, not a fault to retry.
func wrapInvalid(err error) error {
	if err == nil || errors.Is(err, storage.ErrInvalid) {
		return err
	}
	return fmt.Errorf("%w: %w", storage.ErrInvalid, err)
}

// ErrParserPanic is returned when a third-party parser panicked on its input.
var ErrParserPanic = fmt.Errorf("%w: ingest: parser failed on malformed input", storage.ErrInvalid)

// guardParse runs a third-party parser and turns its panic into an error. The
// parsers read files and pastes from anywhere; bgfparser's text reader shifts
// by an unchecked cube exponent (`1 << val`) and panics on a negative one,
// which would take down the GUI or CLI with the whole import.
func guardParse[T any](parse func() (T, error)) (v T, err error) {
	defer func() {
		if r := recover(); r != nil {
			var zero T
			v, err = zero, fmt.Errorf("%w: %v", ErrParserPanic, r)
		}
	}()
	return parse()
}

// xgParseError wraps an xgparser failure for the file at path. A segment
// inflating past xgparser.MaxDecompressedSize is a decompression bomb or a
// damaged archive: the reader cannot tell which, so the message names the
// file and says both, instead of xgparser's internal wording. The cause stays
// wrapped, so errors.Is(err, xgparser.ErrDecompressionLimit) still holds.
func xgParseError(path, stage string, err error) error {
	if errors.Is(err, xgparser.ErrDecompressionLimit) {
		return fmt.Errorf("ingest: %s: fichier trop gros ou corrompu (segment décompressé au-delà de %d Mio): %w",
			filepath.Base(path), xgparser.MaxDecompressedSize>>20, wrapInvalid(err))
	}
	return fmt.Errorf("ingest: %s: %w", stage, wrapInvalid(err))
}
