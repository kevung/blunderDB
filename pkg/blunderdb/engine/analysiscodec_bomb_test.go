package engine

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"testing"
	"time"
)

// zlibZeroBomb hand-assembles a zlib stream that inflates to n zero bytes,
// without running a compressor: one fixed-Huffman deflate block holding a
// literal 0 followed by <length 258, distance 1> back-references. That is 13
// bits per 258 bytes, so ten gigabytes nominal cost about sixty megabytes to
// build in a few milliseconds — where compressing ten gigabytes of zeros for
// real takes tens of seconds. The adler32 trailer of all-zero data is
// computed in closed form.
func zlibZeroBomb(n int64) []byte {
	if n < 1 {
		panic("a bomb needs at least the literal byte")
	}
	repeats := (n - 1) / 258
	tail := (n - 1) % 258 // zeros still owed after the full-length copies

	var out []byte
	var acc uint64 // pending bits, LSB first
	var nacc uint
	// Deflate packs Huffman codes MSB-first into an LSB-first bit stream, and
	// header/extra fields LSB-first; putBits handles the latter, putCode the
	// former.
	putBits := func(v uint64, width uint) {
		acc |= v << nacc
		nacc += width
		for nacc >= 8 {
			out = append(out, byte(acc))
			acc >>= 8
			nacc -= 8
		}
	}
	putCode := func(code uint64, width uint) {
		var rev uint64
		for i := uint(0); i < width; i++ {
			rev = rev<<1 | (code>>i)&1
		}
		putBits(rev, width)
	}
	lengthCode := func(length int64) { // fixed Huffman, lengths 3..258
		switch {
		case length == 258:
			putCode(0b11000101, 8) // symbol 285
		case length <= 10: // symbols 257..264, no extra bits
			putCode(uint64(257+length-3-256), 7) // symbols 257..264 follow end-of-block (256 = code 0)
		default:
			panic("zlibZeroBomb only emits lengths 3..10 and 258")
		}
		putCode(0, 5) // distance code 0 = distance 1
	}

	out = append(out, 0x78, 0x01) // zlib header: deflate, 32 KiB window, no dict
	putBits(1, 1)                 // BFINAL
	putBits(1, 2)                 // BTYPE = fixed Huffman
	putCode(0b00110000, 8)        // literal 0

	// 13 bits per repeat: eight repeats are exactly 13 bytes, so the body is a
	// 13-byte pattern repeated — bytes.Repeat instead of thirty million calls.
	full := repeats / 8
	if full > 0 {
		saved, savedAcc, savedN := out, acc, nacc
		out, acc, nacc = nil, 0, 0
		for i := 0; i < 8; i++ {
			lengthCode(258)
		}
		pattern := out
		if nacc != 0 || len(pattern) != 13 {
			panic("zlibZeroBomb: eight repeats must pack to 13 whole bytes")
		}
		out, acc, nacc = saved, savedAcc, savedN
		// The pattern is byte-aligned only if the stream is; flush the header
		// bits first by padding with repeats until aligned.
		for nacc != 0 {
			lengthCode(258)
			repeats--
			full = repeats / 8
		}
		out = append(out, bytes.Repeat(pattern, int(full))...)
		repeats -= full * 8
	}
	for ; repeats > 0; repeats-- {
		lengthCode(258)
	}
	for tail >= 3 {
		l := min(tail, 10)
		lengthCode(l)
		tail -= l
	}
	for ; tail > 0; tail-- {
		putCode(0b00110000, 8)
	}
	putCode(0, 7) // end of block (symbol 256)
	if nacc > 0 {
		putBits(0, 8-nacc)
	}
	// adler32 of n zero bytes: a stays 1, b accumulates a once per byte.
	a := uint32(1)
	b := uint32(n % 65521)
	adler := b<<16 | a
	return append(out, byte(adler>>24), byte(adler>>16), byte(adler>>8), byte(adler))
}

func TestZlibZeroBombIsAValidStream(t *testing.T) {
	for _, n := range []int64{1, 2, 3, 258, 259, 300, 2064, 2065, 5000, 1 << 20} {
		r, err := zlib.NewReader(bytes.NewReader(zlibZeroBomb(n)))
		if err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		got, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}
		if int64(len(got)) != n || bytes.ContainsFunc(got, func(r rune) bool { return r != 0 }) {
			t.Fatalf("n=%d: inflated to %d bytes, expected %d zeros", n, len(got), n)
		}
	}
}

func TestDecompressAnalysisDataRefusesADecompressionBomb(t *testing.T) {
	const nominal = 10_000_000_000 // 10 GB
	bomb := zlibZeroBomb(nominal)
	start := time.Now()
	_, err := DecompressAnalysisData(bomb)
	elapsed := time.Since(start)
	if !errors.Is(err, ErrAnalysisTooLarge) {
		t.Fatalf("expected ErrAnalysisTooLarge, got %v", err)
	}
	t.Logf("refused in %s", elapsed)
	// About 25 ms natively: the cap stops inflation at 16 MiB. The race
	// detector alone pushes it past a second, which says nothing about the cap.
	if elapsed > time.Second && !raceEnabled {
		t.Fatalf("refusing a %d-byte bomb took %s, expected under a second", int64(nominal), elapsed)
	}
	if _, err := DecodeAnalysisFromStorage(bomb); !errors.Is(err, ErrAnalysisTooLarge) {
		t.Fatalf("DecodeAnalysisFromStorage must surface the refusal, got %v", err)
	}
}

func TestDecompressAnalysisDataAcceptsUpToTheCap(t *testing.T) {
	atCap := zlibZeroBomb(MaxAnalysisBytes)
	out, err := DecompressAnalysisData(atCap)
	if err != nil || len(out) != MaxAnalysisBytes {
		t.Fatalf("a blob of exactly the cap must inflate (%d bytes, %v)", len(out), err)
	}
	if _, err := DecompressAnalysisData(zlibZeroBomb(MaxAnalysisBytes + 1)); !errors.Is(err, ErrAnalysisTooLarge) {
		t.Fatalf("one byte past the cap must be refused, got %v", err)
	}
}
