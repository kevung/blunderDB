package sqlite_test

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

// File-backed, so writer B runs on a connection of its own.
func TestTranscriptionInterleave_SQLite(t *testing.T) {
	s := openTempDB(t)
	storagetest.RunTranscriptionInterleave(t, s, sqlite.SharedExecer(s))
}
