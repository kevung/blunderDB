//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestChangeSet_Postgres exchanges changes between two tenants of one
// database, under FORCE RLS.
func TestChangeSet_Postgres(t *testing.T) {
	s, _ := openRLS(t)
	ctx := context.Background()
	storagetest.CheckChangeSet(t, s, storage.WithTenant(ctx, 1), "1", s, storage.WithTenant(ctx, 2), "2")
}
