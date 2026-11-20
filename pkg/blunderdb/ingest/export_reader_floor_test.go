package ingest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestExportRaisesFloorForMatchResult: the export inserts its matches outside
// the match store, so it raises the file's reader floor itself when a match
// carries its result; a file of matches without result stays readable by
// older binaries.
func TestExportRaisesFloorForMatchResult(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	exportedFloor := func(name string) string {
		t.Helper()
		outPath := filepath.Join(t.TempDir(), name)
		out, err := os.Create(outPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := exportTo(ctx, src, out, WholeTenant(FormatSQLite)); err != nil {
			out.Close()
			t.Fatalf("export: %v", err)
		}
		out.Close()
		dst, err := sqlite.Open(ctx, outPath, nil)
		if err != nil {
			t.Fatalf("reopen export: %v", err)
		}
		defer dst.Close()
		if n, err := dst.Matches().Count(ctx, "", storage.MatchListOpts{}); err != nil || n == 0 {
			t.Fatalf("exported matches = %d, %v; want some", n, err)
		}
		m, err := dst.Metadata().Load(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		return m[storage.MinReaderVersionKey]
	}

	bare := domain.Match{Player1Name: "Carol", Player2Name: "Dan", MatchLength: 5}
	if _, err := src.Matches().Save(ctx, "", &bare); err != nil {
		t.Fatal(err)
	}
	if got := exportedFloor("bare.sqlite"); got != domain.MinReaderVersion {
		t.Errorf("floor of an export without result = %q, want %q", got, domain.MinReaderVersion)
	}

	won := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 7,
		FinalScore: &[2]int32{7, 4}, Winner: domain.WinnerPlayer1}
	if _, err := src.Matches().Save(ctx, "", &won); err != nil {
		t.Fatal(err)
	}
	if got := exportedFloor("won.sqlite"); got != domain.MatchResultReaderVersion {
		t.Errorf("floor of an export carrying a result = %q, want %q", got, domain.MatchResultReaderVersion)
	}
}
