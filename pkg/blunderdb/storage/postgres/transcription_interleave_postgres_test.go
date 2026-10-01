//go:build postgres

package postgres_test

import (
	"testing"

	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// The pool hands writer B a connection of its own.
func TestTranscriptionInterleave_Postgres(t *testing.T) {
	s, _ := openMatchStore(t)
	storagetest.RunTranscriptionInterleave(t, s, pg.SharedExecer(s))
}
