package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationsFS holds the forward migration files. 001 is the bootstrap baseline
// applied by bootstrap() on a fresh database; 002+ are forward migrations applied
// by migrateForward() to bring an existing database up to the current schema.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrateForward applies every forward migration (002+) that has not yet been
// recorded in schema_migrations, in numeric order. Each migration is idempotent
// (ADD COLUMN IF NOT EXISTS, CREATE INDEX IF NOT EXISTS, …), so applying one to a
// freshly bootstrapped database — whose baseline already contains the change — is
// harmless; it is simply recorded as applied so it is not replayed on later opens.
//
// The v2.7.0 baseline (001) is never replayed here.
func migrateForward(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("postgres: ensure schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("postgres: read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") || strings.HasPrefix(name, "001_") {
			continue // skip the bootstrap baseline and non-SQL files
		}
		files = append(files, name)
	}
	sort.Strings(files)

	for _, name := range files {
		version := strings.TrimSuffix(name, ".sql")

		var applied bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&applied); err != nil {
			return fmt.Errorf("postgres: check migration %s: %w", version, err)
		}
		if applied {
			continue
		}

		stmt, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("postgres: read migration %s: %w", name, err)
		}
		// Run as one batch via the simple query protocol (Exec with no bound
		// arguments), which permits the multiple semicolon-separated statements.
		if _, err := pool.Exec(ctx, string(stmt)); err != nil {
			return fmt.Errorf("postgres: apply migration %s: %w", version, err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`, version); err != nil {
			return fmt.Errorf("postgres: record migration %s: %w", version, err)
		}
	}
	return nil
}
