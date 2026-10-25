//go:build postgres

package postgres_test

import (
	"testing"

	pg "github.com/kevung/blunderdb/pkg/blunderdb/storage/postgres"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestReaderFloor_Postgres(t *testing.T) {
	s, _ := openMatchStore(t)
	storagetest.RunReaderFloor(t, s, pg.SharedExecer(s))
}
