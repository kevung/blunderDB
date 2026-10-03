package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// txImpl is a SQLite transaction. Via the embedded binder (bound to a stmtTx)
// every store operation reached through it runs inside the transaction.
type txImpl struct {
	binder
	tx *sql.Tx
}

var _ storage.Tx = (*txImpl)(nil)

// Commit makes the transaction's changes durable.
func (t *txImpl) Commit() error {
	if err := t.tx.Commit(); err != nil {
		return fmt.Errorf("sqlite: commit: %w", err)
	}
	return nil
}

// Rollback discards the transaction. It is safe to call after Commit.
func (t *txImpl) Rollback() error {
	err := t.tx.Rollback()
	if err == nil || errors.Is(err, sql.ErrTxDone) {
		return nil
	}
	return fmt.Errorf("sqlite: rollback: %w", err)
}

// WrapTx exposes the store families over a transaction the caller has already
// opened on the handle, so the Database wrapper's hand-written transactional
// paths (the native .db importer) write positions through PositionStore.Save,
// the one place the Zobrist hash and scalar search columns are computed.
// Commit and Rollback act on tx itself; a caller driving tx directly ignores
// them.
func WrapTx(tx *sql.Tx) storage.Tx {
	return newTxImpl(tx)
}

func newTxImpl(tx *sql.Tx) *txImpl {
	return &txImpl{binder: binder{db: &stmtTx{Tx: tx}}, tx: tx}
}

// hotStatements are the statements an import runs once per position or per
// move. Inside a transaction each is prepared on first use and reused until
// the transaction ends, so parsing and planning are paid once per statement
// rather than once per row. Statements built at run time (IN lists, search
// filters) are not cached: their text varies, so a cache would only grow.
var hotStatements = map[string]bool{
	positionInsertSQL:          true,
	positionIDByHashSQL:        true,
	markFlaggedSQL:             true,
	markIndividualSQL:          true,
	analysisMergeSelectSQL:     true,
	analysisUpsertSQL:          true,
	cubeResponseSQL:            true,
	playedActionsSQL:           true,
	moveInsertSQL:              true,
	positionMatchDateOnMoveSQL: true,
	// A fresh position's first analysis: the match_stats rows of the
	// matches reaching it (none, on an import) are dropped.
	invalidateMatchStatsOfPositionSQL: true,
}

// stmtTx is the execer of a transaction: a *sql.Tx whose hotStatements are
// prepared once. database/sql closes a transaction's prepared statements at
// Commit or Rollback, so the set lives and dies with the transaction.
type stmtTx struct {
	*sql.Tx
	mu    sync.Mutex
	stmts map[string]*sql.Stmt
}

// prepared returns the transaction's statement for query, or nil when query
// is not a hot statement or cannot be prepared — the caller then runs it
// unprepared, which reports any error the usual way.
func (t *stmtTx) prepared(ctx context.Context, query string) *sql.Stmt {
	if !hotStatements[query] {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if st, ok := t.stmts[query]; ok {
		return st
	}
	st, err := t.Tx.PrepareContext(ctx, query) //nolint:sqlclosecheck // closed by the transaction's Commit or Rollback
	if err != nil {
		return nil
	}
	if t.stmts == nil {
		t.stmts = make(map[string]*sql.Stmt)
	}
	t.stmts[query] = st
	return st
}

func (t *stmtTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if st := t.prepared(ctx, query); st != nil { //nolint:sqlclosecheck // closed with the transaction
		return st.ExecContext(ctx, args...)
	}
	return t.Tx.ExecContext(ctx, query, args...)
}

func (t *stmtTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if st := t.prepared(ctx, query); st != nil { //nolint:sqlclosecheck // closed with the transaction
		return st.QueryContext(ctx, args...)
	}
	return t.Tx.QueryContext(ctx, query, args...)
}

func (t *stmtTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if st := t.prepared(ctx, query); st != nil { //nolint:sqlclosecheck // closed with the transaction
		return st.QueryRowContext(ctx, args...)
	}
	return t.Tx.QueryRowContext(ctx, query, args...)
}
