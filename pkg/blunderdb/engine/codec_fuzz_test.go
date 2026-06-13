package engine

import (
	"testing"
)

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
// arbitrary bytes. The blob is read from the `analysis.data` column (raw JSON or
// zlib-compressed); the auto-detection path must never panic on garbage bytes,
// only return an error.
func FuzzDecodeAnalysisFromStorage(f *testing.F) {
	// A valid raw-JSON blob (first byte '{' → returned as-is).
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"player1WinRate":0.5}`))
	// A valid zlib-compressed blob.
	if c, err := CompressAnalysisData([]byte(`{"player1WinRate":0.5}`)); err == nil {
		f.Add(c)
	}
	f.Add([]byte(nil))
	f.Add([]byte("not json, not zlib"))
	f.Add([]byte{0x78, 0x9c, 0x00}) // truncated zlib header

	f.Fuzz(func(t *testing.T, data []byte) {
		// Contract: never panics. Both error and success are acceptable.
		_, _ = DecodeAnalysisFromStorage(data)
	})
}
