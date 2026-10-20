package database

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/ingest"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestIncrementalSync_Converges: two libraries that imported different matches
// converge after one incremental export each way, and an export with nothing
// new since the last one carries nothing. The recipient records no cursor:
// only the producer's metadata names one (ADR-0007, ADR-0096).
func TestIncrementalSync_Converges(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	toB, toA := filepath.Join(dir, "to-b"), filepath.Join(dir, "to-a")

	a, b := openLibrary(t, filepath.Join(dir, "a.db")), openLibrary(t, filepath.Join(dir, "b.db"))
	if _, err := a.ImportXGMatch("testdata/test.xg"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.ImportGnuBGMatch("testdata/gnubg_selfplay_drops_7p.mat"); err != nil {
		t.Fatal(err)
	}
	extra := domain.InitializePosition()
	extra.Dice = [2]int{6, 5}
	pid, err := b.SavePosition(&extra)
	if err != nil {
		t.Fatal(err)
	}
	cid, err := b.store.Collections().Create(ctx, "", "Révisions", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := b.store.Collections().AddPosition(ctx, "", cid, pid); err != nil {
		t.Fatal(err)
	}

	first := syncExport(t, a, toB, "a1.db")
	if first.Positions == 0 || first.Matches != 1 {
		t.Fatalf("first export of A = %+v, want its match and positions", first)
	}
	if again := syncExport(t, a, toB, "a2.db"); !emptyExport(again) {
		t.Fatalf("second export of A with nothing new = %+v, want empty", again)
	}

	if _, err := b.CommitImportDatabase(filepath.Join(toB, "a1.db")); err != nil {
		t.Fatal(err)
	}
	if _, err := a.CommitImportDatabase(syncExport(t, b, toA, "b1.db").Path); err != nil {
		t.Fatal(err)
	}

	if za, zb := zobrists(t, a), zobrists(t, b); !slices.Equal(za, zb) {
		t.Fatalf("positions differ after the round trip: A %d, B %d", len(za), len(zb))
	}
	if ma, mb := matchHashes(t, a), matchHashes(t, b); !slices.Equal(ma, mb) || len(ma) != 2 {
		t.Fatalf("matches after the round trip: A %v, B %v", ma, mb)
	}
	if !hasCollection(t, a, "Révisions") {
		t.Fatal("B's collection did not reach A")
	}

	// What A received from B goes back once — it is new to A's cursor — and
	// adds nothing to B; after that both sides have nothing to send.
	if _, err := b.CommitImportDatabase(syncExport(t, a, toB, "a3.db").Path); err != nil {
		t.Fatal(err)
	}
	if r := syncExport(t, a, toB, "a4.db"); !emptyExport(r) {
		t.Errorf("A after convergence = %+v, want empty", r)
	}
	if r := syncExport(t, b, toA, "b2.db"); !emptyExport(r) {
		t.Errorf("B after convergence = %+v, want empty", r)
	}

	// The cursor is the producer's, per destination: B's own cursor names only
	// the folder B exports to, never the one it received from.
	meta, err := b.store.Metadata().Load(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	for k := range meta {
		if k == "sync_cursor:"+toB {
			t.Errorf("the recipient recorded the producer's cursor %q", k)
		}
	}
	if _, ok := meta["sync_cursor:"+toA]; !ok {
		t.Errorf("B's own cursor is missing: %v", slices.Collect(maps.Keys(meta)))
	}
}

func openLibrary(t *testing.T, path string) *Database {
	t.Helper()
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func syncExport(t *testing.T, d *Database, folder, name string) ingest.ExportReport {
	t.Helper()
	opts := ingest.WholeLibraryOptions()
	opts.Incremental = true
	opts.ExportPath = filepath.Join(folder, name)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := d.ExportDatabase(opts)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func emptyExport(r ingest.ExportReport) bool {
	return r.Positions == 0 && r.Matches == 0 && r.Collections == 0 && r.Tournaments == 0 && r.Lessons == 0
}

func zobrists(t *testing.T, d *Database) []uint64 {
	t.Helper()
	var out []uint64
	for p, err := range d.store.Positions().List(context.Background(), "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, engine.ZobristHash(p))
	}
	slices.Sort(out)
	return out
}

func matchHashes(t *testing.T, d *Database) []string {
	t.Helper()
	var out []string
	for m, err := range d.store.Matches().List(context.Background(), "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, m.MatchHash)
	}
	slices.Sort(out)
	return out
}

func hasCollection(t *testing.T, d *Database, name string) bool {
	t.Helper()
	for c, err := range d.store.Collections().List(context.Background(), "") {
		if err != nil {
			t.Fatal(err)
		}
		if c.Name == name {
			return true
		}
	}
	return false
}
