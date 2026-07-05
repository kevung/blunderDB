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
	"filter_library", "command_history", "search_history",
}

// PurgeTenant permanently deletes every row belonging to scope across all
// tenant-scoped tables (purgeOrder) and its session state. It is idempotent —
// purging a tenant with no data, or purging twice, succeeds with zero rows
// affected. Runs in a single transaction: either everything is purged or
// nothing is.
//
// scope is the same opaque tenant identifier the rest of this package takes
// (X-Tenant-ID header value / storage.ParseTenant's input), not an
// already-converted tenant_id — consistent with every other Store method in
// this repo. PurgeTenant derives the numeric tenant_id internally for the
// tenant_id-scoped tables, and uses scope directly (via sessionScopedKey) for
// the metadata-backed session rows, which have no tenant_id column at all.
//
// PostgreSQL-only, like ApplyRLS/DropRLS (rls_postgres.go) — there is no
// SQLite equivalent (single-user desktop databases have no tenant to purge).
// It filters by tenant_id explicitly on every table rather than relying on
// RLS/the app.tenant_id GUC, so it purges exactly the requested tenant
// whether or not RLS (Options.EnableRLS) is enabled.
func (s *Storage) PurgeTenant(ctx context.Context, scope string) error {
	tenantID := storage.ParseTenant(scope)

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

	// Session state (P4, session_postgres.go) has no tenant_id column — it is
	// namespaced by prefixing the same scope string onto a handful of fixed
	// metadata keys (sessionScopedKey). Purge those rows too, otherwise a
	// decommissioned tenant's session crumbs (last search, last position,
	// open views, ...) linger forever, contradicting "purge deletes
	// everything". Reusing sessionKeys/sessionScopedKey from
	// session_postgres.go (rather than re-deriving the "<scope>:" prefix
	// here) keeps this in lockstep with Save/Load/Clear, and matching the
	// exact scoped key set (rather than a LIKE prefix pattern) means an
	// unusual scope value containing SQL LIKE wildcards (%, _) can't cause
	// over-matching. The unscoped global schema-version row in metadata is
	// never in this set, so it is never touched.
	scopedSessionKeys := make([]string, len(sessionKeys))
	for i, k := range sessionKeys {
		scopedSessionKeys[i] = sessionScopedKey(scope, k)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM metadata WHERE key = ANY($1)`, scopedSessionKeys); err != nil {
		return fmt.Errorf("postgres: purge tenant %q: metadata: %w", scope, err)
	}

	return tx.Commit(ctx)
}
