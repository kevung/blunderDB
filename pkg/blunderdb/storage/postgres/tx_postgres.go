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
