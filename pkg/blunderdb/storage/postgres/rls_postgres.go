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
// tenant carried in the operation's context (storage.WithTenant). PrepareConn
// receives the Acquire call's context, so a pooled connection is scoped to the
// requesting tenant for the duration it is checked out; AfterRelease clears the
// GUC so the connection cannot leak a tenant to the next borrower. A connection
// acquired without a tenant in context is left with the GUC unset, which the
// fail-closed policies treat as "no rows" — safe for the non-tenant operations
// (the schema version) that touch only the unprotected metadata table.
// Cost: about +74 % on LoadPosition (BenchmarkLoadPosition vs
// BenchmarkLoadPositionRLS).
//
// Not SET LOCAL: it only takes effect inside a transaction, leaving the GUC
// unset for single-statement reads — turning fail-closed into fail-open, the
// one direction tenant isolation must never move in.
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

// rlsTables are the tenant-scoped tables that carry a tenant_id column —
// every table but the two pieces of database infrastructure, `metadata`
// (schema version, issuance) and `schema_migrations`, which hold no
// per-tenant data. purgeOrder (purge_postgres.go) must stay a permutation of
// this list, and both must match the tenant_id tables of the embedded
// migrations (TestPurgeOrderMatchesRLSTables); TestTenantTablesSchemaGuards
// checks that ApplyRLS polices every one of them on the live schema.
var rlsTables = []string{
	"position", "analysis", "comment", "match", "game", "move",
	"move_analysis", "tournament", "collection", "collection_position",
	"filter_library", "command_history", "search_history", "session_state",
	"library_settings", "anki_deck", "anki_card", "anki_review_log",
	"transcription",
	"training_session", "training_item",
	"import_batch", "trash",
	"direction", "direction_event", "direction_pair_member", "rencontre",
	"table_setting", "lesson", "lesson_step",
	"import_batch_file", "player_alias", "event_alias", "match_stats", "match_stats_cell", "match_stats_position",
	"lesson_progress", "match_equity_table", "action_label", "study_mark", "duel", "match_origin",
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

// beginner opens a transaction; *pgxpool.Conn, the migration connection, is
// one.
type beginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// inUnforcedTx runs fn in one transaction with FORCE ROW LEVEL SECURITY lifted
// from those of tables that carry it and that the current role owns, and put
// back before the commit. A Go-side pass run by Migrate reads on a connection
// that carries no tenant: under FORCE the fail-closed policy hides every row
// from an owner without BYPASSRLS, so the pass would find nothing and say
// nothing. ALTER TABLE is transactional, so a failed or interrupted pass
// rolls the lift back too — the table is never left unforced. A role that
// does not own a table is bound by its policy whatever FORCE says and cannot
// lift it: that table is left alone, and the pass sees only what the policy
// shows it.
func inUnforcedTx(ctx context.Context, conn beginner, tables []string, fn func(tx pgx.Tx) error) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	var lifted []string
	for _, t := range tables {
		var forced bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM pg_class WHERE oid = to_regclass($1)
			   AND relforcerowsecurity AND pg_has_role(relowner, 'USAGE'))`, t).Scan(&forced); err != nil {
			return fmt.Errorf("postgres: probe RLS on %s: %w", t, err)
		}
		if !forced {
			continue
		}
		if _, err := tx.Exec(ctx, `ALTER TABLE `+pgx.Identifier{t}.Sanitize()+` NO FORCE ROW LEVEL SECURITY`); err != nil {
			return fmt.Errorf("postgres: lift RLS on %s: %w", t, err)
		}
		lifted = append(lifted, t)
	}
	if err := fn(tx); err != nil {
		return err
	}
	for _, t := range lifted {
		if _, err := tx.Exec(ctx, `ALTER TABLE `+pgx.Identifier{t}.Sanitize()+` FORCE ROW LEVEL SECURITY`); err != nil {
			return fmt.Errorf("postgres: restore RLS on %s: %w", t, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}
