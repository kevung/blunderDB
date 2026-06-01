package database

// Verifies P2 PR6: a pre-existing (older-version) database opened through the
// storage backend (sqlite.Open + Storage.Migrate) is upgraded to the current
// schema by the migration chain registered via the storage hook — the same
// chain the GUI/CLI wrapper runs in OpenDatabase. This is the headless path
// used by `serve` / `migrate` / `call`.

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func TestStorageMigratesLegacyDatabase(t *testing.T) {
	ctx := context.Background()
	// Start versions the synthetic createOldDatabase fixture builds faithfully.
	// "1.0.0" already traverses every step of the chain (1.0→2.8), so the
	// intermediate 2.x steps are covered transitively; the later start points
	// exercise the tail directly. (The fixture cannot represent an exact 2.0.0
	// schema — it omits columns the real 1.9→2.0 backfill adds — so 2.0.0 is
	// not a valid synthetic start point.)
	for _, version := range []string{"1.0.0", "1.6.0", "2.5.0", "2.7.0"} {
		t.Run(version, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "old.db")
			createOldDatabase(t, path, version)

			st, err := sqlite.Open(ctx, path, nil)
			if err != nil {
				t.Fatalf("sqlite.Open: %v", err)
			}
			defer st.Close()

			if err := st.Migrate(ctx); err != nil {
				t.Fatalf("Migrate from %s: %v", version, err)
			}

			got, err := st.Version(ctx)
			if err != nil {
				t.Fatalf("Version: %v", err)
			}
			// Reaching the current version proves the whole chain ran: each step
			// bumps database_version only after its DDL/data work succeeds.
			if got != DatabaseVersion {
				t.Errorf("version after migrate: got %s, want %s", got, DatabaseVersion)
			}

			// The migrated database must be usable through the storage layer:
			// the SearchHistory store selects the exclude_position column added
			// by the final 2.7→2.8 step. Draining the iterator runs the query.
			for _, err := range st.SearchHistory().List(ctx, "") {
				if err != nil {
					t.Errorf("SearchHistory().List after migrate from %s: %v", version, err)
					break
				}
			}
		})
	}
}

// TestStorageMigrateCurrentIsNoop confirms migrating an already-current database
// through the storage path is a safe idempotent no-op.
func TestStorageMigrateCurrentIsNoop(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "current.db")

	// Bootstrap a current database via the wrapper, then migrate via storage.
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	d.Close()

	st, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate (current): %v", err)
	}
	got, err := st.Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if got != DatabaseVersion {
		t.Errorf("version: got %s, want %s", got, DatabaseVersion)
	}
}
