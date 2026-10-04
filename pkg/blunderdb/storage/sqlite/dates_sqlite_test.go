package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// UnixFromMatchDateSQL must land on the instant Go computes for every text
// form a match date can take, whatever its zone.
func TestUnixFromMatchDateSQL(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	paris := time.FixedZone("CET", 3600)
	ny := time.FixedZone("EST", -5*3600-30*60)
	for _, tm := range []time.Time{
		time.Date(2019, 12, 7, 13, 24, 33, 0, time.UTC),
		time.Date(2024, 1, 1, 0, 30, 0, 0, paris),
		time.Date(2023, 6, 30, 23, 59, 59, 123456789, ny),
		time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC),
	} {
		var got int64
		if err := db.QueryRowContext(context.Background(), `SELECT `+UnixFromMatchDateSQL("?1"), tm.String()).Scan(&got); err != nil {
			t.Fatalf("%s: %v", tm, err)
		}
		if got != tm.Unix() {
			t.Errorf("%q → %d, want %d", tm.String(), got, tm.Unix())
		}
	}
	for text, want := range map[string]int64{
		"2019-12-07 13:24:33":  1575725073,
		"2019-12-07T13:24:33Z": 1575725073,
		"2019-12-07":           1575676800,
	} {
		var got int64
		if err := db.QueryRow(`SELECT `+UnixFromMatchDateSQL("?1"), text).Scan(&got); err != nil || got != want {
			t.Errorf("%q → %d, %v; want %d", text, got, err, want)
		}
	}
}
