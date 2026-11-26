//go:build postgres

package postgres_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestDocumentImport_Postgres: the import of a Relevé under FORCE RLS.
func TestDocumentImport_Postgres(t *testing.T) {
	s, _ := openRLS(t)
	ctx := storage.WithTenant(context.Background(), 1)
	storagetest.CheckDocumentImport(t, s, ctx, "1")
	storagetest.CheckDocumentImportAtomic(t, s, ctx, "1")
}
