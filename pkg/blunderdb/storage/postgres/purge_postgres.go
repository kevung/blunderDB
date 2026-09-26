package postgres

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// purgeOrder lists every tenant-scoped table PurgeTenant deletes from,
// children before parents, so no ON DELETE CASCADE/SET NULL action needs to
// fire during the purge itself (every referenced row for this tenant is
// already gone by the time its parent's row is deleted) — explicit rather
// than relying on cascade ordering.
//
// purgeOrder must list exactly the tables the migrations create with a
// tenant_id column, like rlsTables (rls_postgres.go):
// TestPurgeOrderMatchesRLSTables (purge_order_test.go) reads that set from the
// embedded migrations, and TestPurgeTenant/TestTenantTablesSchemaGuards
// (purge_postgres_test.go) from the live schema — row counts after a purge and
// foreign-key order. Comparing the two lists only with each other would miss
// a table both forget.
var purgeOrder = []string{
	"move_analysis", "anki_review_log", "collection_position", "training_item",
	"training_session", "direction_event", "direction",
	"comment", "analysis", "move", "anki_card", "game",
	"collection", "anki_deck", "transcription", "match", "import_batch",
	"tournament", "position",
	"filter_library", "command_history", "search_history", "session_state",
	"library_settings", "trash",
}

// PurgeTenant permanently deletes every row belonging to scope across all
// tenant-scoped tables (purgeOrder), session state included. It is
// idempotent — purging a tenant with no data, or purging twice, succeeds with
// zero rows affected. Runs in a single transaction: either everything is
// purged or nothing is.
//
// scope is the same opaque tenant identifier the rest of this package takes
// (X-Tenant-ID header value / storage.ParseTenant's input), not an
// already-converted tenant_id — consistent with every other Store method in
// this repo. PurgeTenant derives the numeric tenant_id internally. The
// global metadata table is never touched: it holds no per-tenant row.
//
// PostgreSQL-only, like ApplyRLS/DropRLS (rls_postgres.go) — there is no
// SQLite equivalent (single-user desktop databases have no tenant to purge).
// It filters by tenant_id explicitly on every table rather than relying on
// RLS/the app.tenant_id GUC, so it purges exactly the requested tenant
// whether or not RLS (Options.EnableRLS) is enabled.
func (s *Storage) PurgeTenant(ctx context.Context, scope string) error {
	tenantID, err := storage.ParseTenant(scope)
	if err != nil {
		// Never fall through to tenant 0, which would purge the wrong tenant
		// (ADR-0005).
		return fmt.Errorf("postgres: purge tenant: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: purge tenant %q: begin: %w", scope, err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	for _, t := range purgeOrder {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE tenant_id = $1`, t), tenantID); err != nil {
			return fmt.Errorf("postgres: purge tenant %q: %s: %w", scope, t, err)
		}
	}

	return tx.Commit(ctx)
}
