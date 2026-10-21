package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// holdWriteLock takes the file's write lock on its own connection and
// releases it after d.
func holdWriteLock(t *testing.T, path string, d time.Duration) {
	t.Helper()
	other, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { other.Close() })
	conn, err := other.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(context.Background(), `INSERT OR REPLACE INTO metadata (key, value) VALUES ('lock_test', 'x')`); err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(d)
		_, _ = conn.ExecContext(context.Background(), "COMMIT")
		conn.Close()
	}()
}

// TestWriteTransactions_WaitForTheLock: a write transaction that reads before
// it writes, begun while another connection holds the write lock, waits for
// it (busy_timeout) instead of failing — through the shared stores'
// Transact and through Storage.BeginTx alike.
func TestWriteTransactions_WaitForTheLock(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "a.db")
	st, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	readThenWrite := func(db execer) error {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metadata`).Scan(&n); err != nil {
			return err
		}
		_, err := db.ExecContext(ctx, `INSERT OR REPLACE INTO metadata (key, value) VALUES ('after', 'y')`)
		return err
	}

	holdWriteLock(t, path, 200*time.Millisecond)
	if err := shared(st.binder).Transact(ctx, func(tx sqlshared.Execer) error {
		return readThenWrite(tx.(shared).db)
	}); err != nil {
		t.Errorf("Transact: %v", err)
	}

	holdWriteLock(t, path, 200*time.Millisecond)
	tx, err := st.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	if err := readThenWrite(tx.(*txImpl).tx); err != nil {
		t.Errorf("BeginTx: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

// TestDeferredWriteOnAStaleSnapshotIsBusy: a deferred transaction that read
// before another writer committed cannot write — SQLITE_BUSY_SNAPSHOT, an
// extended code — and that failure reaches the caller as storage.ErrBusy.
func TestDeferredWriteOnAStaleSnapshotIsBusy(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "a.db")
	st, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	err = withDeferredTx(ctx, st.sqlDB, func(db execer) error {
		var n int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM metadata`).Scan(&n); err != nil {
			return err
		}
		holdWriteLock(t, path, 0) // commits a write outdating this snapshot
		time.Sleep(100 * time.Millisecond)
		_, err := db.ExecContext(ctx, `INSERT OR REPLACE INTO metadata (key, value) VALUES ('stale', 'z')`)
		return err
	})
	if !errors.Is(err, storage.ErrBusy) {
		t.Errorf("stale snapshot write: %v, want storage.ErrBusy", err)
	}
	if !isBusyErr(err) {
		t.Errorf("isBusyErr(%v) = false for an extended busy code", err)
	}
}
