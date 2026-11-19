//go:build postgres

package postgres_test

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestCheckerSources_PostgresRLS keeps each tenant's checker sources apart
// with row-level security in force, and the restricted role raises the
// reader floor when it writes the first blob carrying sources.
func TestCheckerSources_PostgresRLS(t *testing.T) {
	dsn := startPostgres(t)
	s := restrictedRLSStore(t, dsn, "rls_sources")
	defer s.Close()
	storagetest.CheckCheckerSourcesUnderRLS(t, s, "301", "302")
}
