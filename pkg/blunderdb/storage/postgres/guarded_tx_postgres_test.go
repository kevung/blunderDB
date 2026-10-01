//go:build postgres

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestBeginGuardedTx_Serialises: two handles on one database, as two processes are; the second
// guard on a key waits until the first transaction holding it commits.
func TestBeginGuardedTx_Serialises(t *testing.T) {
	dsn := purgeTestDB(t)
	open := func() *Storage {
		s, err := Open(context.Background(), dsn, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
		return s
	}
	a, b := open(), open()
	ctx := context.Background()
	first, err := a.BeginGuardedTx(ctx, "direction|1|rencontre|7")
	if err != nil {
		t.Fatal(err)
	}
	second := make(chan storage.Tx, 1)
	go func() {
		tx, err := b.BeginGuardedTx(ctx, "direction|1|rencontre|7")
		if err != nil {
			t.Error(err)
		}
		second <- tx
	}()
	select {
	case tx := <-second:
		if tx != nil {
			_ = tx.Rollback()
		}
		t.Fatal("the second guard was granted while the first instance held it")
	case <-time.After(300 * time.Millisecond):
	}
	if err := first.Commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case tx := <-second:
		if tx != nil {
			_ = tx.Rollback()
		}
	case <-time.After(8 * time.Second):
		t.Fatal("the second guard still waits after the first committed")
	}
}
