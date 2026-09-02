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
// purgeOrder must stay a permutation of rlsTables (rls_postgres.go);
// TestPurgeOrderMatchesRLSTables (purge_order_test.go) fails loudly if a
// table is added to one list and not the other.
var purgeOrder = []string{
	"move_analysis", "anki_review_log", "collection_position",
	"comment", "analysis", "move", "anki_card", "game",
	"collection", "anki_deck", "match", "tournament", "position",
	"filter_library", "command_history", "search_history", "session_state",
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
// global metadata table is never touched: since schema 2.17.0 it holds no
// per-tenant row (the session moved to session_state, #156).
//
// PostgreSQL-only, like ApplyRLS/DropRLS (rls_postgres.go) — there is no
// SQLite equivalent (single-user desktop databases have no tenant to purge).
// It filters by tenant_id explicitly on every table rather than relying on
// RLS/the app.tenant_id GUC, so it purges exactly the requested tenant
// whether or not RLS (Options.EnableRLS) is enabled.
func (s *Storage) PurgeTenant(ctx context.Context, scope string) error {
	tenantID, err := storage.ParseTenant(scope)
	if err != nil {
		// Never fall through to tenant 0: before ADR-0005's 2026-09-03
		// amendment `tenant.purge "alice"` purged every named tenant at once.
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
