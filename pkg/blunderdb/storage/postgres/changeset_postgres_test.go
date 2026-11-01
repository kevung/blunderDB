//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestChangeSet_Postgres exchanges changes between two tenants of one
// database, under FORCE RLS.
func TestChangeSet_Postgres(t *testing.T) {
	s, _ := openRLS(t)
	ctx := context.Background()
	storagetest.CheckChangeSet(t, s, storage.WithTenant(ctx, 1), "1", s, storage.WithTenant(ctx, 2), "2")
}

// TestChangeSetPreJournal_Postgres: a tenant's rows with no journal entry, as
// an upgrade leaves them, under FORCE RLS.
func TestChangeSetPreJournal_Postgres(t *testing.T) {
	ctx := context.Background()
	s, conn, ownerDSN := openAsRLSOwner(t)
	if err := s.ApplyRLS(ctx); err != nil {
		t.Fatal(err)
	}
	rls, err := pg.Open(ctx, ownerDSN, &storage.Options{EnableRLS: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rls.Close() })
	storagetest.CheckChangeSetPreJournal(t, rls, storage.WithTenant(ctx, 1), "1", rls, storage.WithTenant(ctx, 2), "2", func() {
		asTenant(t, conn, "1", func() {
			if _, err := conn.Exec(ctx, `DELETE FROM change_log`); err != nil {
				t.Fatal(err)
			}
		})
	})
}
