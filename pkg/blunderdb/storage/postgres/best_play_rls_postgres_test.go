//go:build postgres

package postgres_test

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestBestPlay_PostgresRLS: a match's moves name the best play from their own
// tenant's analysis only, with row-level security in force.
func TestBestPlay_PostgresRLS(t *testing.T) {
	dsn := startPostgres(t)
	s := restrictedRLSStore(t, dsn, "rls_best_play")
	defer s.Close()
	storagetest.CheckBestPlayTenantIsolation(t, s, "321", "322")
}
