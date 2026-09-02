package sqlite

// Unit coverage for the SQLITE_BUSY retry path used by positionStore.Save
// (P5, Windows retest): busy_timeout alone is not always enough under a
// heavy burst of concurrent writers, so Save also retries a bounded number
// of times with a short backoff. This file is `package sqlite` (not
// `_test`) because retryOnBusy and isBusyErr are unexported.

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

// openBusyDB opens a connection to path with busy_timeout(0): any write that
// contends with another connection's lock fails immediately with a real
// SQLITE_BUSY, instead of waiting out perConnPragmas' 10s — what the retry
// tests below need to stay fast and deterministic.
func openBusyDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(0)")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestIsBusyErr_RecognisesARealSQLITE_BUSY holds a write transaction open on
// one connection and confirms a second connection's contending write both
// fails and is classified as busy by isBusyErr — pinning the detection
// against the actual driver error shape, not a hand-built stand-in.
func TestIsBusyErr_RecognisesARealSQLITE_BUSY(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "busy.db")

	owner, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { owner.Close() })
	if err := owner.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	tx, err := owner.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO position (zobrist_hash, decision_type, state) VALUES (?, ?, ?)`, int64(1), 0, ""); err != nil {
		t.Fatalf("hold the write lock: %v", err)
	}

	contender := openBusyDB(t, path)
	_, err = contender.ExecContext(ctx, `INSERT INTO position (zobrist_hash, decision_type, state) VALUES (?, ?, ?)`, int64(2), 0, "")
	if err == nil {
		t.Fatal("a contending write against an open write transaction must fail")
	}
	if !isBusyErr(err) {
		t.Fatalf("isBusyErr(%v) = false, want true", err)
	}
}

// TestRetryOnBusy_RecoversOnceTheLockClears drives retryOnBusy with a
// contending write that keeps failing with a genuine SQLITE_BUSY until a
// concurrent goroutine releases the holding transaction — the same shape as
// two pooled connections racing for the write lock in TestConcurrentWrites,
// just isolated to the retry helper.
func TestRetryOnBusy_RecoversOnceTheLockClears(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "busy.db")

	owner, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { owner.Close() })
	if err := owner.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	tx, err := owner.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO position (zobrist_hash, decision_type, state) VALUES (?, ?, ?)`, int64(1), 0, ""); err != nil {
		t.Fatalf("hold the write lock: %v", err)
	}

	released := make(chan struct{})
	go func() {
		time.Sleep(60 * time.Millisecond)
		_ = tx.Commit()
		close(released)
	}()

	contender := openBusyDB(t, path)
	attempts := 0
	err = retryOnBusy(func() error {
		attempts++
		_, execErr := contender.ExecContext(ctx, `INSERT INTO position (zobrist_hash, decision_type, state) VALUES (?, ?, ?)`, int64(2), 0, "")
		return execErr
	})
	<-released
	if err != nil {
		t.Fatalf("retryOnBusy did not recover once the lock cleared: %v", err)
	}
	if attempts < 2 {
		t.Fatalf("attempts = %d, want at least 2 (the lock was held past the first try)", attempts)
	}
}

// TestRetryOnBusy_GivesUpAfterTheBudget confirms the retry loop is bounded:
// a lock that never clears must not hang the caller forever.
func TestRetryOnBusy_GivesUpAfterTheBudget(t *testing.T) {
	calls := 0
	stubBusyErr := func() error {
		// Reuse a real SQLITE_BUSY, produced the same way as the other
		// tests above, rather than hand-building the driver's unexported
		// error type.
		ctx := context.Background()
		path := filepath.Join(t.TempDir(), "busy.db")
		owner, err := Open(ctx, path, nil)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer owner.Close()
		if err := owner.Migrate(ctx); err != nil {
			t.Fatalf("Migrate: %v", err)
		}
		tx, err := owner.sqlDB.BeginTx(ctx, nil)
		if err != nil {
			t.Fatalf("BeginTx: %v", err)
		}
		defer tx.Rollback()
		if _, err := tx.ExecContext(ctx, `INSERT INTO position (zobrist_hash, decision_type, state) VALUES (?, ?, ?)`, int64(1), 0, ""); err != nil {
			t.Fatalf("hold the write lock: %v", err)
		}
		contender := openBusyDB(t, path)
		_, execErr := contender.ExecContext(ctx, `INSERT INTO position (zobrist_hash, decision_type, state) VALUES (?, ?, ?)`, int64(2), 0, "")
		return execErr
	}

	err := retryOnBusy(func() error {
		calls++
		return stubBusyErr()
	})
	if err == nil {
		t.Fatal("a lock that never clears must still surface an error")
	}
	if !isBusyErr(err) {
		t.Fatalf("final error should still be a busy error, got %v", err)
	}
	if calls != busyRetryAttempts {
		t.Fatalf("calls = %d, want exactly %d (the bounded budget)", calls, busyRetryAttempts)
	}
}
