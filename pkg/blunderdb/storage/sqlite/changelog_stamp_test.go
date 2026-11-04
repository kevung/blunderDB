package sqlite

import (
	"database/sql"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The date after a date is counted by SQLite's own calendar on text: it has to
// land on the very next millisecond, or two writes would tie after all or a
// peer's comparison would see a gap the clock never made.
func TestStampAfterIsTheNextMillisecond(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	stmt, err := db.Prepare(`SELECT ` + stampAfter("?"))
	if err != nil {
		t.Fatal(err)
	}
	defer stmt.Close()
	// Far ahead of any clock, so the expression shows the step and not now.
	start := time.Date(2999, 12, 31, 23, 59, 59, 0, time.UTC)
	for i := range 200_000 {
		at := start.Add(time.Duration(i*7+i%13) * time.Millisecond)
		var got string
		if err := stmt.QueryRow(at.Format(storage.ChangedAtLayout)).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if want := at.Add(time.Millisecond).Format(storage.ChangedAtLayout); got != want {
			t.Fatalf("after %s: %s, want %s", at.Format(storage.ChangedAtLayout), got, want)
		}
	}
}
