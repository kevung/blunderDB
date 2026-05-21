package sqlite_test

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestContract_SQLite runs the backend-agnostic storage contract suite against
// a fresh in-memory SQLite database. Open bootstraps the v2.7.0 schema.
func TestContract_SQLite(t *testing.T) {
	storagetest.RunContractTests(t, func() storage.Storage {
		s, err := sqlite.Open(context.Background(), ":memory:", nil)
		if err != nil {
			t.Fatalf("sqlite.Open: %v", err)
		}
		return s
	})
}
