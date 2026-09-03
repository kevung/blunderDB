package engine

import (
	"bytes"
	"compress/zlib"
	"testing"
)

// zlibCompressForFuzzSeed builds a legacy-format seed for
// FuzzDecodeAnalysisFromStorage: every 2.x release before #180 wrote analysis
// blobs this way (CompressAnalysisData used compress/zlib directly), and
// that format must still decode, so the fuzz corpus needs a genuine example
// of it independent of the current (zstd) CompressAnalysisData.
func zlibCompressForFuzzSeed(jsonData []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(jsonData); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// FuzzDecodeBoardCompact exercises the compact-board decoder. The compact state
// string is read back from the SQLite/Postgres `position.state` column; a
// corrupt or hand-edited row must decode without panicking. When the input is
// itself the output of EncodeBoardCompact, decoding must round-trip exactly.
func FuzzDecodeBoardCompact(f *testing.F) {
	seeds := []string{
		"",
		"[]",
		"[0]",
		"null",
		"not json",
		"[1,2,3]",
		"[0,0,0,0,0,0,2,0,0,0,0,-5,0,-3,0,0,0,5,-5,0,0,0,3,0,5,0,0,0]",
		"[999999999999,-999999999999]",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		b := DecodeBoardCompact(s)
		// Re-encoding a decoded board and decoding again must be stable: the
		// codec is the storage round-trip, so it has to be idempotent.
		again := DecodeBoardCompact(EncodeBoardCompact(b))
		if again != b {
			t.Fatalf("DecodeBoardCompact not idempotent for %q:\n first=%+v\nsecond=%+v", s, b, again)
		}
	})
}

// FuzzDecodeAnalysisFromStorage exercises the analysis blob decoder against
// arbitrary bytes. The blob is read from the `analysis.data` column (raw
// JSON, legacy zlib, or current zstd — see the format-detection doc comment
// on DecompressAnalysisData); the auto-detection path must never panic on
// garbage bytes, only return an error. One seed per format (#180): a real
// blob written by each codec this package has ever produced, plus a
// decompression bomb for each of the two compressed formats.
func FuzzDecodeAnalysisFromStorage(f *testing.F) {
	// A valid raw-JSON blob (first byte '{' → returned as-is).
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"player1WinRate":0.5}`))
	// A valid zstd blob — the current codec (CompressAnalysisData).
	if c, err := CompressAnalysisData([]byte(`{"player1WinRate":0.5}`)); err == nil {
		f.Add(c)
	}
	// A valid legacy zlib blob — every 2.x release before this one wrote this
	// format, and it must decode forever (see the package doc comment).
	f.Add(zlibCompressForFuzzSeed([]byte(`{"player1WinRate":0.5}`)))
	f.Add([]byte(nil))
	f.Add([]byte("not json, not zlib, not zstd"))
	f.Add([]byte{0x78, 0x9c, 0x00})             // truncated zlib header
	f.Add([]byte{0x28, 0xB5, 0x2F, 0xFD, 0x00}) // truncated zstd header
	// A decompression bomb per compressed format: a small payload that
	// inflates to several times MaxAnalysisBytes. The decoder must refuse
	// both, not allocate for either.
	f.Add(zlibZeroBomb(64 << 20))
	f.Add(zstdZeroBomb(64 << 20))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Contract: never panics. Both error and success are acceptable.
		_, _ = DecodeAnalysisFromStorage(data)
	})
}
