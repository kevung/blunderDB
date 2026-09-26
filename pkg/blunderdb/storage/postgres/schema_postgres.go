package postgres

import (
	"context"
	_ "embed"
	"fmt"
)

// schemaSQL is the full v2.7.0 DDL for a fresh PostgreSQL database. It is run
// as one batch via the simple query protocol (Exec with no bound arguments),
// which permits the multiple semicolon-separated statements.
//
//go:embed migrations/001_initial_v2_7_0.sql
var schemaSQL string

// bootstrap creates the v2.7.0 schema on a fresh database. It assumes an
// empty database; the DDL uses CREATE TABLE/INDEX IF NOT EXISTS so a re-run
// is harmless.
//
// It deliberately does NOT record database_version: Migrate writes that once,
// after bootstrap and every forward migration have run to completion (see
// setDatabaseVersion in postgres.go), so an interrupted chain never leaves a
// version that disagrees with the schema.
func bootstrap(ctx context.Context, db execer) error {
	if _, err := db.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("postgres: bootstrap schema: %w", err)
	}
	return nil
}
