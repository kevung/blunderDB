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
