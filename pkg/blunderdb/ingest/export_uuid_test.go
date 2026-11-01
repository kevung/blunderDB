package ingest

import (
	"database/sql"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// An export carries no identifier it invented: its rows hold no uuid and the
// file no mark, so the recipient's first open derives them from the natural
// keys, and one export is the same bytes whenever it is made.
func TestExportLeavesUUIDsToTheRecipient(t *testing.T) {
	path := exportOneMatch(t, domain.Match{Player1Name: "A", Player2Name: "B", CanonicalHash: "canon-x"})
	db, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var rows, withUUID, marks int
	if err := db.QueryRow(`SELECT COUNT(*), COUNT(uuid) FROM match`).Scan(&rows, &withUUID); err != nil {
		t.Fatal(err)
	}
	if rows == 0 || withUUID != 0 {
		t.Errorf("exported matches = %d, with a uuid = %d; want some, none with a uuid", rows, withUUID)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM metadata WHERE key = ?`, sqlshared.ObjectUUIDKey).Scan(&marks); err != nil || marks != 0 {
		t.Errorf("the export carries the uuid mark (%d, %v); the recipient's open must derive the uuids", marks, err)
	}
}
