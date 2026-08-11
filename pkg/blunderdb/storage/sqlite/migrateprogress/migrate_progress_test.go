// Package migrateprogress_test covers a single seam: storage.Options.
// MigrationProgress, captured by sqlite.Open and forwarded by
// (*sqlite.Storage).Migrate to the registered Migrator.
//
// It lives in its own directory/package for the same reason
// storage/sqlite/nomigrator does: it calls sqlite.RegisterMigrator itself
// (with a fake migrator that only records its progress calls, no real
// schema work), and the sibling storage/sqlite_test package already
// registers the real legacy chain via bench_test.go's import of package
// database — whichever registration runs last wins for the whole test
// binary (see migrate_hook.go's doc comment), so sharing a binary with that
// import would make this test's own registration order-dependent and
// fragile, and would clobber the real migrator for any other test in that
// binary that came to depend on it.
package migrateprogress_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

type progressCall struct {
	phase       string
	done, total int
}

func TestMigrate_ForwardsMigrationProgressToRegisteredMigrator(t *testing.T) {
	sqlite.RegisterMigrator(func(ctx context.Context, db *sql.DB, progress func(phase string, done, total int)) error {
		if progress == nil {
			t.Fatal("registered migrator received a nil progress callback")
		}
		progress("position", 1, 2)
		progress("position", 2, 2)
		return nil
	})

	ctx := context.Background()
	var calls []progressCall
	opts := &storage.Options{
		MigrationProgress: func(phase string, done, total int) {
			calls = append(calls, progressCall{phase, done, total})
		},
	}
	s, err := sqlite.Open(ctx, ":memory:", opts)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer s.Close()

	// Open() already bootstrapped the metadata table (fresh DB); back-date
	// the version so Migrate treats it as non-fresh and outdated, taking the
	// registered-migrator branch rather than the already-current no-op.
	if err := s.Metadata().SetVersion(ctx, "", "2.0.0"); err != nil {
		t.Fatalf("SetVersion: %v", err)
	}

	if err := s.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	want := []progressCall{{"position", 1, 2}, {"position", 2, 2}}
	if !reflect.DeepEqual(calls, want) {
		t.Errorf("progress calls = %#v, want %#v", calls, want)
	}
}
