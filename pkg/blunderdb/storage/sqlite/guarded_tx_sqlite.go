package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

var _ storage.GuardedBeginner = (*Storage)(nil)

// BeginGuardedTx opens a transaction that holds the database's write lock from its first
// statement (BEGIN IMMEDIATE), so a read made in it cannot be outdated by another writer —
// another connection, the desktop, a `call` on the same file — before it commits. SQLite has
// one write lock per file, so the keys are not needed to tell guards apart.
func (s *Storage) BeginGuardedTx(ctx context.Context, _ ...string) (storage.Tx, error) {
	conn, err := s.sqlDB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("sqlite: guarded tx: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("sqlite: guarded tx: begin: %w", err)
	}
	return &connTx{binder: binder{db: conn}, conn: conn}, nil
}

// connTx is a transaction opened by statement on one pooled connection, which database/sql
// cannot express (its BeginTx has no IMMEDIATE). The connection returns to the pool at its end.
type connTx struct {
	binder
	conn *sql.Conn
	done bool
}

var _ storage.Tx = (*connTx)(nil)

// end runs COMMIT or ROLLBACK even when the gesture's context was cancelled: the lock must not
// outlive the request that took it.
func (t *connTx) end(stmt string) error {
	if t.done {
		return sql.ErrTxDone
	}
	t.done = true
	_, err := t.conn.ExecContext(context.Background(), stmt)
	if err != nil && stmt == "COMMIT" {
		_, _ = t.conn.ExecContext(context.Background(), "ROLLBACK")
	}
	_ = t.conn.Close()
	return err
}

// Commit makes the transaction's changes durable and releases the write lock.
func (t *connTx) Commit() error {
	if err := t.end("COMMIT"); err != nil {
		return fmt.Errorf("sqlite: commit: %w", err)
	}
	return nil
}

// Rollback discards the transaction. It is safe to call after Commit.
func (t *connTx) Rollback() error {
	if t.done {
		return nil
	}
	if err := t.end("ROLLBACK"); err != nil {
		return fmt.Errorf("sqlite: rollback: %w", err)
	}
	return nil
}
