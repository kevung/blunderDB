package gui

import (
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	_ "modernc.org/sqlite"
)

// realNames are the people the embedded demo must never name again (issue
// #162): the six players of the database shipped before 0.36.0, plus the
// people the fixtures scripts/build-demo-db.sh now imports are named after.
// Each token is matched as a whole word, case-insensitively, in every TEXT
// column of every table.
var realNames = []string{
	// the six names of the previous demo.db.gz
	"unger", "harmand", "friebe", "jacobi", "huyck", "larsen",
	// the people behind the current sources, disguised by scripts/demodb
	"caslou", "maxence", "tachi",
}

// openDemoRaw decompresses the embedded demo and opens it on a plain
// database/sql handle: the Database wrapper migrates at open, which would
// hide a stale database_version.
func openDemoRaw(t *testing.T) *sql.DB {
	t.Helper()
	path, err := NewApp(nil).PrepareDemoDatabase()
	if err != nil {
		t.Fatalf("PrepareDemoDatabase: %v", err)
	}
	t.Cleanup(func() { os.Remove(path) })
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("opening demo db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestDemoDatabaseIsCurrent(t *testing.T) {
	db := openDemoRaw(t)
	var version string
	if err := db.QueryRow(`SELECT value FROM metadata WHERE key = 'database_version'`).Scan(&version); err != nil {
		t.Fatalf("reading database_version: %v", err)
	}
	if version != domain.DatabaseVersion {
		t.Errorf("demo.db.gz carries database_version %s, want %s: run scripts/build-demo-db.sh", version, domain.DatabaseVersion)
	}
}

func TestDemoDatabaseNamesNobodyReal(t *testing.T) {
	db := openDemoRaw(t)
	forbidden := regexp.MustCompile(`(?i)\b(` + strings.Join(realNames, "|") + `)\b`)
	for _, table := range tableNames(t, db) {
		for _, column := range textColumns(t, db, table) {
			if value, hit := firstHit(t, db, table, column, forbidden); hit != "" {
				t.Errorf("%s.%s names %q: %q", table, column, hit, value)
			}
		}
	}
}

func tableNames(t *testing.T, db *sql.DB) []string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return names
}

// firstHit returns the first value of table.column the pattern matches, and
// the word it matched. Table and column names come from the schema itself.
func firstHit(t *testing.T, db *sql.DB, table, column string, pattern *regexp.Regexp) (value, hit string) {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf(`SELECT "%s" FROM "%s" WHERE "%s" IS NOT NULL`, column, table, column))
	if err != nil {
		t.Fatalf("%s.%s: %v", table, column, err)
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&value); err != nil {
			t.Fatalf("%s.%s: %v", table, column, err)
		}
		if hit = pattern.FindString(value); hit != "" {
			return value, hit
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("%s.%s: %v", table, column, err)
	}
	return "", ""
}

// textColumns lists the columns declared as text: the analysis blob is
// declared JSON and holds compressed bytes that a word regexp could match by
// accident.
func textColumns(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, table))
	if err != nil {
		t.Fatalf("table_info(%s): %v", table, err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var (
			cid, notNull, pk int
			name, ctype      string
			dflt             sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("table_info(%s): %v", table, err)
		}
		if strings.Contains(strings.ToUpper(ctype), "TEXT") {
			columns = append(columns, name)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("table_info(%s): %v", table, err)
	}
	return columns
}

// TestDemoDatabaseShowsEveryPanel guards the content the guided tours and the
// manual promise: matches in a tournament, collections, comments, an Anki
// deck with a review journal, and analyses from more than one engine.
func TestDemoDatabaseShowsEveryPanel(t *testing.T) {
	db := openDemoRaw(t)
	for _, tc := range []struct {
		what  string
		query string
		min   int
	}{
		{"matches", `SELECT count(*) FROM match`, 3},
		{"tournaments", `SELECT count(*) FROM tournament`, 1},
		{"matches in a tournament", `SELECT count(*) FROM match WHERE tournament_id IS NOT NULL`, 2},
		{"collections", `SELECT count(*) FROM collection`, 3},
		{"positions in collections", `SELECT count(*) FROM collection_position`, 40},
		{"comments", `SELECT count(*) FROM comment WHERE text LIKE '%#blunder%' OR text LIKE '%#cube%'`, 10},
		{"anki decks", `SELECT count(*) FROM anki_deck`, 1},
		{"anki cards", `SELECT count(*) FROM anki_card`, 20},
		{"anki reviews", `SELECT count(*) FROM anki_review_log`, 20},
		{"analysed positions", `SELECT count(*) FROM analysis`, 400},
	} {
		var n int
		if err := db.QueryRow(tc.query).Scan(&n); err != nil {
			t.Fatalf("%s: %v", tc.what, err)
		}
		if n < tc.min {
			t.Errorf("%s: %d, want at least %d", tc.what, n, tc.min)
		}
	}
}
