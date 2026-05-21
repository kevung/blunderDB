package postgres

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// schemaSQL is the full v2.7.0 DDL for a fresh PostgreSQL database. It is run
// as one batch via the simple query protocol (Exec with no bound arguments),
// which permits the multiple semicolon-separated statements.
//
//go:embed migrations/001_initial_v2_7_0.sql
var schemaSQL string

// bootstrap creates the v2.7.0 schema on a fresh database and records the
// schema version. It assumes an empty database; the DDL uses
// CREATE TABLE/INDEX IF NOT EXISTS so a re-run is harmless.
func bootstrap(ctx context.Context, db execer) error {
	if _, err := db.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("postgres: bootstrap schema: %w", err)
	}
	if _, err := db.Exec(ctx,
		`INSERT INTO metadata (key, value) VALUES ('database_version', $1)
		 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		domain.DatabaseVersion); err != nil {
		return fmt.Errorf("postgres: bootstrap version: %w", err)
	}
	return nil
}
