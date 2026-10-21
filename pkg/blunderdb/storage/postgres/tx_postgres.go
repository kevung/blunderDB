package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// txImpl is a PostgreSQL transaction. Via the embedded binder (bound to a
// pgx.Tx) every store operation reached through it runs inside the
// transaction.
//
// pgx's Commit/Rollback take a context.Context, but the storage.Tx interface
// (modelled on database/sql) does not. The context passed to BeginTx is
// retained here so Commit/Rollback can honour its deadline/cancellation.
type txImpl struct {
	binder
	tx  pgx.Tx
	ctx context.Context
}

var _ storage.Tx = (*txImpl)(nil)
var _ storage.StatsRefresher = (*txImpl)(nil)

// statsTables are the tables a library copy fills and RefreshStats analyses.
const statsTables = `position, analysis, match, game, move`

// statsDeniedSQL counts the statsTables the current role may not ANALYZE:
// it must own them (or be a member of their owner), be superuser, or, from
// PostgreSQL 17, hold MAINTAIN on them. ANALYZE refused only warns and does
// nothing, so the right is checked first. CASE keeps PostgreSQL 16 from
// evaluating a privilege name it does not know.
const statsDeniedSQL = `SELECT count(*) FROM pg_class c
	WHERE c.oid = ANY (ARRAY['position', 'analysis', 'match', 'game', 'move']::regclass[])
	  AND NOT pg_has_role(c.relowner, 'MEMBER')
	  AND NOT (SELECT rolsuper FROM pg_roles WHERE rolname = current_user)
	  AND NOT CASE WHEN current_setting('server_version_num')::int >= 170000
	               THEN has_table_privilege(c.oid, 'MAINTAIN') ELSE false END`

// RefreshStats analyses the tables a library copy fills. Without it, a
// migration's move inserts fire the multi_played trigger with a plan drawn
// up for empty tables: a BitmapAnd that reads the whole tenant_id index of
// move on every insert, quadratic over the copy.
//
// ANALYZE inside a transaction holds SHARE UPDATE EXCLUSIVE on each table
// until it ends: a second migration (another tenant) waits for this one to
// commit, autovacuum skips the tables and CREATE INDEX on them waits; reads
// and writes of the daemon are not blocked. A role without the right to
// ANALYZE is refused here rather than left to the quadratic plan.
func (t *txImpl) RefreshStats(ctx context.Context) error {
	var denied int
	if err := t.tx.QueryRow(ctx, statsDeniedSQL).Scan(&denied); err != nil {
		return fmt.Errorf("postgres: refresh stats: check rights: %w", err)
	}
	if denied > 0 {
		return fmt.Errorf("postgres: refresh stats: the role may not ANALYZE %s (it must own them, or hold MAINTAIN from PostgreSQL 17)",
			statsTables)
	}
	if _, err := t.tx.Exec(ctx, `ANALYZE `+statsTables); err != nil {
		return fmt.Errorf("postgres: refresh stats: %w", err)
	}
	return nil
}

// Commit makes the transaction's changes durable.
func (t *txImpl) Commit() error {
	if err := t.tx.Commit(t.ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}

// Rollback discards the transaction. It is safe to call after Commit.
func (t *txImpl) Rollback() error {
	err := t.tx.Rollback(t.ctx)
	if err == nil || errors.Is(err, pgx.ErrTxClosed) {
		return nil
	}
	return fmt.Errorf("postgres: rollback: %w", err)
}
