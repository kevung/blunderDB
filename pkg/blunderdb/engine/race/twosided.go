package race

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// Two-sided bearoff databases (gnubg .bd, header "gnubg-TS-PP-CC-F") store,
// for every ordered pair of one-sided positions, four uint16 equities for the
// player on roll, money game, gammonless, scale u/32767.5 − 1:
//
//	plane 0 — cubeless equity (u/65535 is the win probability),
//	plane 1 — cubeful no-double continuation equity, player on roll owns the cube,
//	plane 2 — idem, cube centered,
//	plane 3 — cubeful equity, opponent owns the cube (no decision to make).
//
// Planes 1–2 are continuation values (the "ND" of the cube decision), NOT the
// decision-tree optimum; the money verdict is reconstructed from them and
// plane 3 (see CubeVerdict). Semantics pinned against gnubg cfevaluate on 160
// fixture states in testdata/money_fixtures.json (exact to quantisation).

//go:embed gnubg_ts0.bd
var embeddedTS0 []byte

// TwoSided reads one gnubg two-sided bearoff database through an io.ReaderAt;
// lookups are 8-byte reads, the file is never loaded into memory.
type TwoSided struct {
	r         io.ReaderAt
	nPoints   int
	nCheckers int
	nPos      int // C(nPoints+nCheckers, nPoints)
	origin    string
}

const tsHeaderSize = 40

// newTwoSided parses the 40-byte header and validates the advertised size.
func newTwoSided(r io.ReaderAt, size int64, origin string) (*TwoSided, error) {
	var hdr [tsHeaderSize]byte
	if _, err := r.ReadAt(hdr[:], 0); err != nil {
		return nil, fmt.Errorf("%s: read header: %w", origin, err)
	}
	// "gnubg-TS-06-11-1" + 'x' padding + '\n'.
	fields := strings.Split(strings.TrimRight(string(hdr[:]), "x\n\x00"), "-")
	if len(fields) != 5 || fields[0] != "gnubg" || fields[1] != "TS" {
		return nil, fmt.Errorf("%s: not a gnubg two-sided bearoff database", origin)
	}
	nPoints, err1 := strconv.Atoi(fields[2])
	nCheckers, err2 := strconv.Atoi(fields[3])
	if err1 != nil || err2 != nil || nPoints != 6 || nCheckers < 1 || nCheckers > 15 {
		return nil, fmt.Errorf("%s: unsupported domain TS-%s-%s", origin, fields[2], fields[3])
	}
	if !strings.HasPrefix(fields[4], "1") {
		return nil, fmt.Errorf("%s: cubeless-only two-sided database (fCubeful=0)", origin)
	}
	nPos := combination(nPoints+nCheckers, nPoints)
	want := int64(tsHeaderSize) + int64(nPos)*int64(nPos)*8
	if size >= 0 && size != want {
		return nil, fmt.Errorf("%s: size %d does not match TS-06-%02d (want %d)", origin, size, nCheckers, want)
	}
	return &TwoSided{r: r, nPoints: nPoints, nCheckers: nCheckers, nPos: nPos, origin: origin}, nil
}

// OpenTwoSided opens a two-sided database file. The returned handle keeps the
// file open; it is intended to live for the process (sources are resolved and
// cached by Resolve).
func OpenTwoSided(path string) (*TwoSided, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	ts, err := newTwoSided(f, st.Size(), path)
	if err != nil {
		f.Close()
		return nil, err
	}
	return ts, nil
}

var (
	embeddedOnce sync.Once
	embeddedDB   *TwoSided
)

// EmbeddedTwoSided returns the built-in TS-06-06 database (gnubg's default
// gnubg_ts0.bd). It never fails: the file is compiled in.
func EmbeddedTwoSided() *TwoSided {
	embeddedOnce.Do(func() {
		ts, err := newTwoSided(bytes.NewReader(embeddedTS0), int64(len(embeddedTS0)), "embedded gnubg_ts0.bd")
		if err != nil {
			panic(fmt.Sprintf("embedded gnubg_ts0.bd invalid: %v", err))
		}
		embeddedDB = ts
	})
	return embeddedDB
}

// Origin describes where this database came from (path or "embedded …").
func (t *TwoSided) Origin() string { return t.origin }

// Close releases the underlying file handle opened by OpenTwoSided. It is a
// safe no-op for the embedded database (backed by a bytes.Reader, which does
// not implement io.Closer). Windows in particular keeps a removed or
// replaced file's directory entry alive — and refuses to reopen the path —
// until every handle on it is closed, so any code that opens a TwoSided
// outside the long-lived process cache in source.go (a test, or a one-shot
// tool) must Close it once done.
func (t *TwoSided) Close() error {
	if c, ok := t.r.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// Checkers returns the per-player checker capacity of the database domain.
func (t *TwoSided) Checkers() int { return t.nCheckers }

// Covers reports whether a pair of home boards is inside this database's
// domain.
func (t *TwoSided) Covers(us, them [6]int) bool {
	return sum(us[:]) <= t.nCheckers && sum(them[:]) <= t.nCheckers
}

// Entry holds the four equities of one lookup, player on roll's viewpoint.
type Entry struct {
	WinProb    float64 // plane 0 as a probability
	Cubeless   float64 // plane 0 as an equity
	OwnedND    float64 // plane 1
	CenteredND float64 // plane 2
	Against    float64 // plane 3
}

// Lookup reads the entry for (player on roll, opponent) home boards.
func (t *TwoSided) Lookup(us, them [6]int) (Entry, error) {
	if !t.Covers(us, them) {
		return Entry{}, fmt.Errorf("%s: position outside TS-06-%02d domain", t.origin, t.nCheckers)
	}
	iu := engine.BearoffIndex(us, t.nPoints, t.nCheckers)
	it := engine.BearoffIndex(them, t.nPoints, t.nCheckers)
	off := int64(tsHeaderSize) + (int64(iu)*int64(t.nPos)+int64(it))*8
	var b [8]byte
	if _, err := t.r.ReadAt(b[:], off); err != nil {
		return Entry{}, fmt.Errorf("%s: read entry: %w", t.origin, err)
	}
	eq := func(k int) float64 {
		return float64(binary.LittleEndian.Uint16(b[2*k:]))/32767.5 - 1.0
	}
	return Entry{
		WinProb:    float64(binary.LittleEndian.Uint16(b[0:])) / 65535.0,
		Cubeless:   eq(0),
		OwnedND:    eq(1),
		CenteredND: eq(2),
		Against:    eq(3),
	}, nil
}

func sum(xs []int) int {
	s := 0
	for _, x := range xs {
		s += x
	}
	return s
}

// combination computes C(n, k) using iterative multiplication.
func combination(n, k int) int {
	if k > n {
		return 0
	}
	if k > n-k {
		k = n - k
	}
	result := 1
	for i := 0; i < k; i++ {
		result = result * (n - i) / (i + 1)
	}
	return result
}
