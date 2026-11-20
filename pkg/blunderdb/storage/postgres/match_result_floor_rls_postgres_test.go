//go:build postgres

package postgres_test

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// TestMatchResultFloor_PostgresRLS: the restricted role raises the reader
// floor when it stores the first match carrying its result.
func TestMatchResultFloor_PostgresRLS(t *testing.T) {
	dsn := startPostgres(t)
	s := restrictedRLSStore(t, dsn, "rls_result")
	defer s.Close()
	storagetest.CheckMatchResultFloorUnderRLS(t, s, "311")
}
