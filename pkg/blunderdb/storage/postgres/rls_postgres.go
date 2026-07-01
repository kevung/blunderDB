package postgres

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// configureRLSPool installs pool hooks that bind the `app.tenant_id` GUC to the
// tenant carried in the operation's context (storage.WithTenant). BeforeAcquire
// receives the Acquire call's context, so a pooled connection is scoped to the
// requesting tenant for the duration it is checked out; AfterRelease clears the
// GUC so the connection cannot leak a tenant to the next borrower. A connection
// acquired without a tenant in context is left with the GUC unset, which the
// fail-closed policies treat as "no rows" — safe for the non-tenant operations
// (metadata/version) that touch only the unprotected metadata table.
func configureRLSPool(cfg *pgxpool.Config) {
	cfg.PrepareConn = func(ctx context.Context, conn *pgx.Conn) (bool, error) {
		if tenant, ok := storage.TenantFromContext(ctx); ok {
			if _, err := conn.Exec(ctx,
				`SELECT set_config('app.tenant_id', $1, false)`,
				strconv.FormatInt(tenant, 10)); err != nil {
				return false, err // discard the connection rather than leak an unset GUC
			}
		}
		return true, nil
	}
	cfg.AfterRelease = func(conn *pgx.Conn) bool {
		if _, err := conn.Exec(context.Background(), `RESET app.tenant_id`); err != nil {
			return false // drop a connection we could not reset
		}
		return true
	}
}

// rlsTables are the tenant-scoped domain tables that carry a tenant_id column.
// The global `metadata` table is intentionally excluded (it holds the schema
// version and is not tenant-scoped).
var rlsTables = []string{
	"position", "analysis", "comment", "match", "game", "move",
	"move_analysis", "tournament", "collection", "collection_position",
	"filter_library", "command_history", "search_history",
	"anki_deck", "anki_card", "anki_review_log",
}

// ApplyRLS installs (idempotently) Row-Level Security on every tenant-scoped
// table: it enables and FORCEs RLS — so even the table owner is subject — and
// creates a fail-closed `tenant_isolation` policy that restricts rows to the
// tenant in the `app.tenant_id` GUC. `current_setting(..., true)` returns NULL
// when the GUC is unset, so a connection without a tenant sees no rows and
// cannot insert.
//
// Enforcement also requires the connection to carry the GUC: open the Storage
// with Options.EnableRLS and propagate the tenant via storage.WithTenant.
// Application-level tenant filtering stays in place either way — RLS is a
// second layer, not a replacement.
func (s *Storage) ApplyRLS(ctx context.Context) error {
	return forEachRLSTable(ctx, s.pool, func(t string) []string {
		// NULLIF maps an unset/reset GUC (custom GUCs reset to '' , not NULL) to
		// NULL, so the comparison yields no rows instead of an `''::bigint` cast
		// error — fail-closed for a connection without a tenant.
		policyPred := "tenant_id = NULLIF(current_setting('app.tenant_id', true), '')::bigint"
		return []string{
			fmt.Sprintf(`ALTER TABLE %s ENABLE ROW LEVEL SECURITY`, t),
			fmt.Sprintf(`ALTER TABLE %s FORCE ROW LEVEL SECURITY`, t),
			fmt.Sprintf(`DROP POLICY IF EXISTS tenant_isolation ON %s`, t),
			fmt.Sprintf(`CREATE POLICY tenant_isolation ON %s USING (%s) WITH CHECK (%s)`,
				t, policyPred, policyPred),
		}
	})
}

// DropRLS removes the tenant_isolation policy and disables RLS on every
// tenant-scoped table. Idempotent.
func (s *Storage) DropRLS(ctx context.Context) error {
	return forEachRLSTable(ctx, s.pool, func(t string) []string {
		return []string{
			fmt.Sprintf(`DROP POLICY IF EXISTS tenant_isolation ON %s`, t),
			fmt.Sprintf(`ALTER TABLE %s NO FORCE ROW LEVEL SECURITY`, t),
			fmt.Sprintf(`ALTER TABLE %s DISABLE ROW LEVEL SECURITY`, t),
		}
	})
}

func forEachRLSTable(ctx context.Context, pool *pgxpool.Pool, stmts func(table string) []string) error {
	for _, t := range rlsTables {
		for _, stmt := range stmts(t) {
			if _, err := pool.Exec(ctx, stmt); err != nil {
				return fmt.Errorf("postgres: RLS on %s: %w", t, err)
			}
		}
	}
	return nil
}
