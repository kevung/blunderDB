package ingest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// fakeFences stands for a backend's running transactions: a token names
// the transactions open when it was taken, cleared once all have ended.
type fakeFences struct{ open map[string]bool }

func (f fakeFences) cleared(token string) (bool, error) { return !f.open[token], nil }

// exportOnce runs one incremental export's cursor arithmetic: check the
// fences, read from Settled, take the fence of the mark just read.
func exportOnce(t *testing.T, c SyncCursor, f fakeFences, read SyncMark, token string) (SyncMark, SyncCursor) {
	t.Helper()
	n, err := c.clearedPrefix(f.cleared)
	if err != nil {
		t.Fatal(err)
	}
	return c.Settled, c.advance(n, read, token, time.Now(), false, true)
}

// On a backend whose identifiers can appear out of order, a mark is trusted
// only through an export that read after every transaction running at the
// mark's read had ended — however long that takes. A transaction kept open
// across many exports keeps them all starting below its identifiers.
func TestSyncCursorUnorderedWaitsForItsFence(t *testing.T) {
	f := fakeFences{open: map[string]bool{}}
	var c SyncCursor

	// The export E1 read positions up to 10; an import still running holds
	// 7, invisible to it, and outlives any delay.
	f.open["e1"] = true
	since, c := exportOnce(t, c, f, SyncMark{Position: 10}, "e1")
	if since != (SyncMark{}) {
		t.Fatalf("first export starts at %+v, want zero", since)
	}
	for i := range 3 {
		since, c = exportOnce(t, c, f, SyncMark{Position: 12}, "e2")
		if since.Position != 0 {
			t.Fatalf("export %d while the import runs starts at %d, want 0: position 7 may still appear", i, since.Position)
		}
	}
	// The import commits position 7. The next export still starts at 0 —
	// it is the one that carries 7 — and only then is 10 trusted.
	f.open["e1"] = false
	since, c = exportOnce(t, c, f, SyncMark{Position: 12}, "e3")
	if since.Position != 0 {
		t.Fatalf("the export after the commit starts at %d, want 0", since.Position)
	}
	if c.Settled.Position != 12 {
		t.Fatalf("settled after the cleared export = %d, want 12 (every cleared mark folds in)", c.Settled.Position)
	}
	if len(c.Pending) != 1 || c.Pending[0].Fence != "e3" {
		t.Fatalf("pending = %+v, want the last mark alone", c.Pending)
	}
}

// A newer mark whose fence cleared is not trusted ahead of an older one
// whose fence has not: it lies above the older one's late rows.
func TestSyncCursorClearsInOrder(t *testing.T) {
	f := fakeFences{open: map[string]bool{"old": true}}
	c := SyncCursor{Pending: []PendingMark{
		{Mark: SyncMark{Position: 10}, Fence: "old"},
		{Mark: SyncMark{Position: 20}, Fence: "new"},
	}}
	since, c := exportOnce(t, c, f, SyncMark{Position: 30}, "e")
	if since.Position != 0 || c.Settled.Position != 0 || len(c.Pending) != 3 {
		t.Fatalf("start %d, cursor %+v: nothing may settle behind an open fence", since.Position, c)
	}
}

// On an ordered backend the mark is trusted at once; on a backend that can
// name no fence it never is, and every export starts from the last
// trusted mark.
func TestSyncCursorOrderedAndUnfenced(t *testing.T) {
	now := time.Now()
	if c := (SyncCursor{}).advance(0, SyncMark{Position: 10}, "", now, true, false); c.Settled.Position != 10 {
		t.Fatalf("ordered: settled = %d, want 10", c.Settled.Position)
	}
	c := SyncCursor{}.advance(0, SyncMark{Position: 10}, "", now, false, false)
	if c.Settled.Position != 0 || len(c.Pending) != 0 {
		t.Fatalf("unfenced: cursor = %+v, want the zero cursor", c)
	}
}

// An export that skipped a row leaves its cursor where it was: the mark
// would otherwise pass a row that never left.
func TestIncrementalExportWithSkippedRowsHoldsTheCursor(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	p := domain.InitializePosition()
	if _, err := src.Positions().Save(ctx, "", &p); err != nil {
		t.Fatal(err)
	}

	opts, err := OptionsFor(ctx, src, "", domain.ExportOptions{Incremental: true, SyncCursor: "x", IncludeAnalysis: true})
	if err != nil {
		t.Fatal(err)
	}
	// A tournament that no longer exists: the export skips it.
	opts.Selection.TournamentIDs, opts.Selection.TournamentMatches = []int64{999}, false
	report, err := ExportSQLite(ctx, src, "", filepath.Join(t.TempDir(), "a.db"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if report.Skipped == 0 || !report.SyncHeld {
		t.Fatalf("report = %+v, want a skipped row and the cursor held", report)
	}
	cur, err := LoadSyncCursor(ctx, src, "", "x")
	if err != nil {
		t.Fatal(err)
	}
	if cur.Settled != (SyncMark{}) || len(cur.Pending) != 0 {
		t.Fatalf("cursor advanced past a skipped export: %+v", cur)
	}
}
