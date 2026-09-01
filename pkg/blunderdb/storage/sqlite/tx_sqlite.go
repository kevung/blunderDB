package sqlite

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// txImpl is a SQLite transaction. Via the embedded binder (bound to *sql.Tx)
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
// opened on the handle. It exists for the legacy Database wrapper's few
// remaining hand-written transactional paths (the native .db importer), so
// they can write positions through PositionStore.Save — the one place the
// Zobrist hash and the scalar search columns are computed — instead of
// re-implementing the insert with a raw INSERT that leaves those columns
// NULL. Commit and Rollback act on tx itself; a caller that keeps driving tx
// directly should simply ignore them.
func WrapTx(tx *sql.Tx) storage.Tx {
	return &txImpl{binder: binder{db: tx}, tx: tx}
}
