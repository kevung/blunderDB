// Package nomigrator_test isolates a single scenario: storage/sqlite.Migrate
// on a non-fresh, outdated database when no migrator has been registered.
//
// Its own package so this test binary never imports package database, whose
// init() registers the migration chain (sqlite/migrate_hook.go): the sibling
// sqlite_test package imports it. This is cmd/serve's situation without the
// blank import of package database.
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
