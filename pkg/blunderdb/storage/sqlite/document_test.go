package sqlite_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestDocumentImport_SQLite(t *testing.T) {
	s, _ := openLib(t)
	storagetest.CheckDocumentImport(t, s, context.Background(), "")
}

func TestDocumentImportAtomic_SQLite(t *testing.T) {
	s, _ := openLib(t)
	storagetest.CheckDocumentImportAtomic(t, s, context.Background(), "")
}
