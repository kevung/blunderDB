package sqlite_test

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestBlunderCountCache_SQLite(t *testing.T) {
	s := openTempDB(t)
	storagetest.RunBlunderCountCache(t, s, sqlite.SharedExecer(s))
}
