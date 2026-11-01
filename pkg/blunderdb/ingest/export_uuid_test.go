package ingest

import (
	"context"
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

// Opening the file names what it holds, and two openings of one export agree.
func TestExportedFileGetsItsUUIDsWhenOpened(t *testing.T) {
	path := exportOneMatch(t, domain.Match{Player1Name: "A", Player2Name: "B", CanonicalHash: "canon-y"})
	var seen [2]string
	for i := range seen {
		s, err := sqlite.Open(context.Background(), path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Migrate(context.Background()); err != nil {
			s.Close()
			t.Fatalf("Migrate: %v", err)
		}
		db, err := sql.Open("sqlite", sqlite.DSN(path))
		if err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`SELECT COALESCE(uuid, '') FROM match`).Scan(&seen[i]); err != nil {
			t.Fatal(err)
		}
		db.Close()
		s.Close()
		if i == 0 {
			// Forget again, so the second opening derives afresh.
			db, _ := sql.Open("sqlite", sqlite.DSN(path))
			if err := sqlite.ForgetObjectUUIDs(context.Background(), db); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`UPDATE match SET uuid = NULL`); err != nil {
				t.Fatal(err)
			}
			db.Close()
		}
	}
	if seen[0] == "" || seen[0] != seen[1] {
		t.Errorf("opened twice, the match holds %q then %q; want one derived identifier", seen[0], seen[1])
	}
}
