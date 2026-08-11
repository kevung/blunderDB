// Package nomigrator_test isolates a single scenario: storage/sqlite.Migrate
// on a non-fresh, outdated database when no migrator has been registered.
//
// It lives in its own directory/package (rather than alongside the other
// storage/sqlite tests) specifically so this test binary never imports
// package database — not even transitively. Package database's init()
// registers the legacy migration chain (see
// pkg/blunderdb/database/migrate_hook.go and
// pkg/blunderdb/storage/sqlite/migrate_hook.go); the sibling
// storage/sqlite_test package already imports database from bench_test.go,
// which would make registeredMigrator non-nil for the whole test binary and
// defeat this test. This is exactly the situation cmd/serve is in when it
// builds without the blank import of package database.
package nomigrator_test

import (
	"context"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func TestMigrate_NonFreshWithoutRegisteredMigrator(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer s.Close()

	// Open() already bootstrapped the metadata table (so the DB is no longer
	// "fresh"); back-date the recorded version to simulate a pre-existing
	// user database older than the current schema.
	if err := s.Metadata().SetVersion(ctx, "", "2.0.0"); err != nil {
		t.Fatalf("SetVersion: %v", err)
	}

	err = s.Migrate(ctx)
	if err == nil {
		t.Fatal("Migrate: expected an error for a non-fresh, outdated database with no registered migrator, got nil")
	}
	if !strings.Contains(err.Error(), "no migrator is registered") {
		t.Errorf("Migrate error = %q, want it to mention the missing migrator", err.Error())
	}
}

func TestMigrate_NonFreshAlreadyCurrent(t *testing.T) {
	ctx := context.Background()
	s, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	defer s.Close()

	if err := s.Migrate(ctx); err != nil {
		t.Errorf("Migrate on an already-current, non-fresh database: %v", err)
	}
}
