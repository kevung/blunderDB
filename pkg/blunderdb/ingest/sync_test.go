package ingest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// On a backend whose identifiers can appear out of order, a mark is not
// trusted until SyncSettleDelay has passed: the exports made meanwhile start
// from the last trusted mark, so a row committed late by a concurrent import
// is sent by the next export instead of being lost under the mark.
func TestSyncCursorUnorderedOverlapsUntilSettled(t *testing.T) {
	t0 := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	var c SyncCursor

	since, c := c.start(t0)
	if since != (SyncMark{}) {
		t.Fatalf("first export starts at %+v, want zero", since)
	}
	// The export at t0 read positions up to 10; an import still running
	// holds 7, invisible to it.
	c = c.advance(SyncMark{Position: 10}, t0, false)

	since, c = c.start(t0.Add(time.Minute))
	if since.Position != 0 {
		t.Fatalf("a minute later the export starts at %d, want 0: position 7 may still appear", since.Position)
	}
	c = c.advance(SyncMark{Position: 12}, t0.Add(time.Minute), false)

	since, c = c.start(t0.Add(SyncSettleDelay))
	if since.Position != 10 {
		t.Fatalf("after the delay the export starts at %d, want 10", since.Position)
	}
	if len(c.Pending) != 1 || c.Pending[0].Mark.Position != 12 {
		t.Fatalf("pending after settling = %+v, want the mark 12 alone", c.Pending)
	}
	since, _ = c.start(t0.Add(SyncSettleDelay + time.Minute))
	if since.Position != 12 {
		t.Fatalf("start = %d, want 12", since.Position)
	}
}

// On an ordered backend the mark is trusted at once.
func TestSyncCursorOrderedAdvancesAtOnce(t *testing.T) {
	now := time.Now()
	c := SyncCursor{}.advance(SyncMark{Position: 10}, now, true)
	if since, _ := c.start(now); since.Position != 10 {
		t.Fatalf("start = %d, want 10", since.Position)
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
