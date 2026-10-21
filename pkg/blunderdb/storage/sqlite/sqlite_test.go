package sqlite_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestContract_SQLite runs the backend-agnostic storage contract suite against
// a fresh in-memory SQLite database. Open bootstraps the v2.7.0 schema.
func TestContract_SQLite(t *testing.T) {
	storagetest.RunContractTests(t, func() storage.Storage {
		s, err := sqlite.Open(context.Background(), ":memory:", nil)
		if err != nil {
			t.Fatalf("sqlite.Open: %v", err)
		}
		return s
	})
}

// TestReadAcross_SQLite: SQLite holds one tenant, so its read set is that
// tenant alone — the daemon's "1" — and a read across it is a plain read,
// tagged. There is no outsider to leave out: the header widens nothing here.
func TestReadAcross_SQLite(t *testing.T) {
	storagetest.RunReadAcrossTests(t, func() storage.Storage {
		s, err := sqlite.Open(context.Background(), ":memory:", nil)
		if err != nil {
			t.Fatalf("sqlite.Open: %v", err)
		}
		return s
	}, []string{"1"}, "")
}

// TestElsewhere_SQLite holds the Elsewhere read on separate files, the shape
// the CLI reads.
func TestElsewhere_SQLite(t *testing.T) {
	storagetest.RunElsewhereTests(t, func() storage.Storage {
		s, err := sqlite.Open(context.Background(), ":memory:", nil)
		if err != nil {
			t.Fatalf("sqlite.Open: %v", err)
		}
		return s
	})
}

// TestOpenReadOnly_WritesNothing: a file opened read-only refuses a write and
// keeps its bytes, whatever its name holds. SQLite may still create the
// -wal/-shm sidecars a WAL file is read through; they hold no content.
func TestOpenReadOnly_WritesNothing(t *testing.T) {
	ctx := context.Background()
	name := "club ?#%.db"
	if runtime.GOOS == "windows" {
		name = "club #%.db" // Windows forbids '?' in a file name
	}
	path := filepath.Join(t.TempDir(), name)
	rw, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	rw.Close()
	for _, side := range []string{"-wal", "-shm"} {
		os.Remove(path + side)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ro, err := sqlite.OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	if _, err := ro.Version(ctx); err != nil {
		t.Fatalf("Version: %v", err)
	}
	p := domain.InitializePosition()
	if _, err := ro.Positions().Save(ctx, "", &p); err == nil {
		t.Error("a read-only store accepted a write")
	}
	ro.Close()
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("reading the file read-only changed it")
	}
}

// A database in a folder that cannot be written is read all the same, and no
// -wal/-shm file appears beside it.
func TestOpenReadOnly_ReadOnlyFolder(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "club.db")
	rw, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	rw.Close()
	for _, side := range []string{"-wal", "-shm"} {
		os.Remove(path + side)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0o755) }()
	if f, err := os.Create(filepath.Join(dir, "probe")); err == nil {
		f.Close()
		t.Skip("the folder stays writable (running as root)")
	}
	ro, err := sqlite.OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatalf("OpenReadOnly in a read-only folder: %v", err)
	}
	defer ro.Close()
	if _, err := ro.Version(ctx); err != nil {
		t.Fatalf("Version: %v", err)
	}
	for _, side := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(path + side); err == nil {
			t.Errorf("%s was created", side)
		}
	}
}

// A mode=ro Storage says it refuses writes, so the player table falls back to
// the direct computation instead of trying to fill match_stats; a writable one
// still says it accepts them.
func TestOpenReadOnly_RefusesWrites(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "club.db")
	rw, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if sqlshared.RefusesWrites(ctx, sqlite.SharedExecer(rw)) {
		t.Error("a writable store reports refusing writes")
	}
	rw.Close()
	ro, err := sqlite.OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatalf("OpenReadOnly: %v", err)
	}
	defer ro.Close()
	if !sqlshared.RefusesWrites(ctx, sqlite.SharedExecer(ro)) {
		t.Error("a read-only store does not report refusing writes")
	}
	if _, err := ro.Stats().PlayerTable(ctx, "", storage.StatsFilter{}); err != nil {
		t.Errorf("PlayerTable on a read-only store: %v", err)
	}
}
