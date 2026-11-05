package sqlite_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/storagetest"
)

func TestChangeSetLinks_SQLite(t *testing.T) {
	ctx := context.Background()
	storagetest.CheckChangeSetLinks(t, func(t *testing.T) *storagetest.ChangeSetPair {
		a, _ := openLib(t)
		b, _ := openLib(t)
		return storagetest.NewChangeSetPair(t, a, ctx, "", b, ctx, "")
	})
}

func TestChangeSetCardTwinBeforeTheJournal_SQLite(t *testing.T) {
	ctx := context.Background()
	storagetest.CheckChangeSetCardTwinBeforeTheJournal(t, func(t *testing.T) (*storagetest.ChangeSetPair, func()) {
		a, _ := openLib(t)
		b, path := openLib(t)
		return storagetest.NewChangeSetPair(t, a, ctx, "", b, ctx, ""), func() {
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if _, err := db.ExecContext(ctx, `DELETE FROM change_log WHERE family = 'anki_card'`); err != nil {
				t.Fatal(err)
			}
		}
	})
}
