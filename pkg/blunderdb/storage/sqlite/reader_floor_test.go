package sqlite_test

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestReaderFloor_SQLite(t *testing.T) {
	s := openTempDB(t)
	storagetest.RunReaderFloor(t, s, sqlite.SharedExecer(s))
}
