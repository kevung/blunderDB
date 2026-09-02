package race

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// isolateSources points the resolver at an empty temp dir so tests never pick
// up a real downloaded database from the developer's XDG data dir.
func isolateSources(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	SetDataDir(dir)
	SetExternalPath("")
	t.Cleanup(func() {
		SetDataDir("")
		SetExternalPath("")
		// Resolve()'s process-lifetime cache may still hold a handle open on
		// a file inside dir (it is only closed the *next* time Resolve() is
		// called). Drop it now — same package, so cachedDB is reachable
		// directly — so t.TempDir()'s own RemoveAll cleanup (registered
		// before this one, so it runs after, per t.Cleanup's LIFO order)
		// does not race an open file handle on Windows.
		srcMu.Lock()
		if cachedDB != nil && cachedDB != embeddedDB {
			cachedDB.Close()
		}
		cachedDB, cachedStamp = nil, ""
		srcMu.Unlock()
	})
	return dir
}

func clearBoard() domain.Board {
	var b domain.Board
	for i := range b.Points {
		b.Points[i] = domain.Point{Color: -1}
	}
	return b
}

func put(b *domain.Board, point, color, n int) {
	b.Points[point] = domain.Point{Color: color, Checkers: n}
}

func TestEvaluate_ExactRegime(t *testing.T) {
	isolateSources(t)
	var pos domain.Position
	pos.Board = clearBoard()
	put(&pos.Board, 1, domain.Black, 2)
	put(&pos.Board, 3, domain.Black, 2)
	put(&pos.Board, 22, domain.White, 3)
	put(&pos.Board, 24, domain.White, 1)
	pos.PlayerOnRoll = domain.Black
	pos.Cube.Owner = domain.None

	r := Evaluate(&pos)
	if r.Race == nil {
		t.Fatal("pure bearoff must produce a race zone")
	}
	if r.Race.Regime != RegimeExact || r.Race.SourceCheckers != 6 {
		t.Fatalf("want exact regime on TS-06-06, got %+v", r.Race)
	}
	if r.Race.Money == nil || r.Race.Money.CubeState != CubeCentered || r.Race.Money.Verdict == "" {
		t.Fatalf("centered cube must carry a verdict, got %+v", r.Race.Money)
	}
	if r.Race.OnRoll != domain.Black {
		t.Fatalf("on roll: got %d", r.Race.OnRoll)
	}
	if r.Race.WinProb <= 0 || r.Race.WinProb >= 1 {
		t.Fatalf("degenerate win prob %v", r.Race.WinProb)
	}
	if r.Race.Sigma != 0 || r.Race.P99 != 0 {
		t.Fatalf("exact regime must not carry error bounds: %+v", r.Race)
	}

	// Cube against: no verdict.
	pos.Cube.Owner = domain.White
	r = Evaluate(&pos)
	if r.Race.Money.CubeState != CubeAgainst || r.Race.Money.Verdict != "" {
		t.Fatalf("cube against: got %+v", r.Race.Money)
	}
	// Cube owned by the on-roll player.
	pos.Cube.Owner = domain.Black
	r = Evaluate(&pos)
	if r.Race.Money.CubeState != CubeOwned {
		t.Fatalf("cube owned: got %+v", r.Race.Money)
	}

	// White on roll: p flips to White's viewpoint.
	pos.Cube.Owner = domain.None
	pos.PlayerOnRoll = domain.White
	r2 := Evaluate(&pos)
	if r2.Race.OnRoll != domain.White {
		t.Fatalf("on roll: got %d", r2.Race.OnRoll)
	}
}

func TestEvaluate_EstimatedRegime(t *testing.T) {
	isolateSources(t)
	var pos domain.Position
	pos.Board = clearBoard()
	// 8 checkers for Black: outside the embedded TS-06-06 domain.
	put(&pos.Board, 1, domain.Black, 3)
	put(&pos.Board, 2, domain.Black, 3)
	put(&pos.Board, 5, domain.Black, 2)
	put(&pos.Board, 20, domain.White, 4)
	pos.PlayerOnRoll = domain.Black
	pos.Cube.Owner = domain.None

	r := Evaluate(&pos)
	if r.Race == nil || r.Race.Regime != RegimeEstimated {
		t.Fatalf("want estimated regime, got %+v", r.Race)
	}
	if r.Race.Money != nil {
		t.Fatal("the cube verdict must never be estimated (ADR-0009)")
	}
	if r.Race.Sigma <= 0 || r.Race.P99 <= 0 {
		t.Fatalf("estimated regime must expose its error bounds, got %+v", r.Race)
	}
	if r.Race.WinProb <= 0 || r.Race.WinProb >= 1 {
		t.Fatalf("degenerate win prob %v", r.Race.WinProb)
	}
}

func TestEvaluate_OutsideDomain(t *testing.T) {
	isolateSources(t)
	var pos domain.Position
	pos.Board = clearBoard()
	put(&pos.Board, 1, domain.Black, 2)
	put(&pos.Board, 13, domain.Black, 1) // straggler: not a pure bearoff
	put(&pos.Board, 24, domain.White, 2)
	pos.PlayerOnRoll = domain.Black
	if r := Evaluate(&pos); r.Race != nil {
		t.Fatalf("no race zone outside pure bearoff, got %+v", r.Race)
	}
	// A side with every checker borne off: game over, no race zone.
	pos.Board = clearBoard()
	put(&pos.Board, 1, domain.Black, 2)
	if r := Evaluate(&pos); r.Race != nil {
		t.Fatalf("no race zone once a side has borne off everything, got %+v", r.Race)
	}
}

// writeSyntheticTS fabricates a structurally valid TS-06-NN file (zeroed
// entries) via a sparse truncate, for resolution tests only.
func writeSyntheticTS(t *testing.T, path string, nCheckers int) {
	t.Helper()
	nPos := combination(6+nCheckers, 6)
	hdr := make([]byte, tsHeaderSize)
	for i := range hdr {
		hdr[i] = 'x'
	}
	copy(hdr, []byte("gnubg-TS-06-"+twoDigits(nCheckers)+"-1"))
	hdr[tsHeaderSize-1] = '\n'
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(hdr); err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(int64(tsHeaderSize) + int64(nPos)*int64(nPos)*8); err != nil {
		t.Fatal(err)
	}
}

func twoDigits(n int) string {
	return string([]byte{byte('0' + n/10), byte('0' + n%10)})
}

func TestResolve_WidestDomainWins(t *testing.T) {
	dir := isolateSources(t)

	if got := Resolve(); got.Checkers() != 6 {
		t.Fatalf("floor must be the embedded TS-06-06, got %d", got.Checkers())
	}

	// A wider external file wins over the embedded one.
	ext := filepath.Join(dir, "wide.bd")
	writeSyntheticTS(t, ext, 8)
	SetExternalPath(ext)
	if got := Resolve(); got.Checkers() != 8 {
		t.Fatalf("external TS-06-08 must win, got TS-06-%02d", got.Checkers())
	}

	// A wider downloaded file wins over both.
	writeSyntheticTS(t, filepath.Join(dir, DownloadedFileName), 9)
	if got := Resolve(); got.Checkers() != 9 {
		t.Fatalf("downloaded TS-06-09 must win, got TS-06-%02d", got.Checkers())
	}

	// A NARROWER external file must lose to the embedded database.
	SetExternalPath("")
	os.Remove(filepath.Join(dir, DownloadedFileName))
	narrow := filepath.Join(dir, "narrow.bd")
	writeSyntheticTS(t, narrow, 3)
	SetExternalPath(narrow)
	if got := Resolve(); got.Checkers() != 6 {
		t.Fatalf("embedded TS-06-06 must beat a TS-06-03, got TS-06-%02d", got.Checkers())
	}

	// An invalid candidate is skipped, never fatal.
	bad := filepath.Join(dir, "bad.bd")
	if err := os.WriteFile(bad, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	SetExternalPath(bad)
	if got := Resolve(); got.Checkers() != 6 {
		t.Fatalf("invalid external must fall back to embedded, got TS-06-%02d", got.Checkers())
	}
}

// The entry read through a synthetic zeroed file decodes to u=0 everywhere:
// p=0, equities −1. This pins the offset arithmetic (any header misalignment
// would surface as garbage).
func TestLookup_ZeroedFileDecodesToFloor(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "z.bd")
	writeSyntheticTS(t, p, 2)
	ts, err := OpenTwoSided(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ts.Close() })
	e, err := ts.Lookup([6]int{1, 1, 0, 0, 0, 0}, [6]int{0, 2, 0, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	if e.WinProb != 0 || e.Cubeless != -1 || e.Against != -1 {
		t.Fatalf("zeroed entries must decode to the scale floor, got %+v", e)
	}
	if !ts.Covers([6]int{2, 0, 0, 0, 0, 0}, [6]int{0, 0, 0, 0, 0, 2}) ||
		ts.Covers([6]int{2, 1, 0, 0, 0, 0}, [6]int{0, 0, 0, 0, 0, 2}) {
		t.Fatal("Covers must enforce the per-player checker capacity")
	}
}
