package postgres

import (
	"context"
	"fmt"
)

// purgeOrder lists every tenant-scoped table PurgeTenant deletes from,
// children before parents, so no ON DELETE CASCADE/SET NULL action needs to
// fire during the purge itself (every referenced row for this tenant is
// already gone by the time its parent's row is deleted) — explicit rather
// than relying on cascade ordering.
//
// purgeOrder must stay a permutation of rlsTables (rls_postgres.go);
// TestPurgeOrderMatchesRLSTables (purge_postgres_test.go) fails loudly if a
// table is added to one list and not the other.
var purgeOrder = []string{
	"move_analysis", "anki_review_log", "collection_position",
	"comment", "analysis", "move", "anki_card", "game",
	"collection", "anki_deck", "match", "tournament", "position",
	"filter_library", "command_history", "search_history",
}

// PurgeTenant permanently deletes every row belonging to tenantID across all
// tenant-scoped tables (purgeOrder). It is idempotent — purging a tenant with
// no data, or purging twice, succeeds with zero rows affected. Runs in a
// single transaction: either every table is purged or none is.
//
// PostgreSQL-only, like ApplyRLS/DropRLS (rls_postgres.go) — there is no
// SQLite equivalent (single-user desktop databases have no tenant to purge).
// It filters by tenant_id explicitly on every table rather than relying on
// RLS/the app.tenant_id GUC, so it purges exactly the requested tenant
// whether or not RLS (Options.EnableRLS) is enabled.
func (s *Storage) PurgeTenant(ctx context.Context, tenantID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: purge tenant %d: begin: %w", tenantID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op after a successful Commit

	for _, t := range purgeOrder {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE tenant_id = $1`, t), tenantID); err != nil {
			return fmt.Errorf("postgres: purge tenant %d: %s: %w", tenantID, t, err)
		}
	}
	return tx.Commit(ctx)
}
