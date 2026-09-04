package sqlite

import (
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// dsnPragmas returns the _pragma values DSN encoded, decoded.
func dsnPragmas(t *testing.T, dsn string) []string {
	t.Helper()
	i := strings.LastIndex(dsn, "?")
	if i < 0 {
		t.Fatalf("DSN %q carries no query", dsn)
	}
	q, err := url.ParseQuery(dsn[i+1:])
	if err != nil {
		t.Fatalf("parse query of %q: %v", dsn, err)
	}
	return q["_pragma"]
}

func TestDSN_MemoryKeepsNameAndSkipsWAL(t *testing.T) {
	dsn := DSN(":memory:")
	if !strings.HasPrefix(dsn, ":memory:?") {
		t.Fatalf("DSN(:memory:) = %q, want the bare name followed by the query", dsn)
	}
	pragmas := strings.Join(dsnPragmas(t, dsn), " ")
	if strings.Contains(pragmas, "journal_mode") {
		t.Errorf("in-memory DSN must not request WAL: %q", pragmas)
	}
	for _, want := range []string{"busy_timeout(10000)", "foreign_keys(ON)"} {
		if !strings.Contains(pragmas, want) {
			t.Errorf("DSN(:memory:) lacks %s: %q", want, pragmas)
		}
	}
}

// A plain path is handed to the driver untouched: spaces, accents and
// backslashes are all legal SQLite file names and must not be escaped
// (the driver does not percent-decode the path part of a DSN).
func TestDSN_PlainPathIsPassedVerbatim(t *testing.T) {
	for _, path := range []string{
		"/home/kévin/Mes bases/base 2026.db",
		`C:\Users\Kévin\Documents\base 2026.db`,
		"relative.db",
	} {
		dsn := DSN(path)
		if !strings.HasPrefix(dsn, path+"?") {
			t.Errorf("DSN(%q) = %q, want the path verbatim before the query", path, dsn)
		}
		pragmas := strings.Join(dsnPragmas(t, dsn), " ")
		for _, want := range []string{"busy_timeout(10000)", "foreign_keys(ON)", "journal_mode(WAL)"} {
			if !strings.Contains(pragmas, want) {
				t.Errorf("DSN(%q) lacks %s: %q", path, want, pragmas)
			}
		}
	}
}

func TestDSN_CallerURIKeepsItsQuery(t *testing.T) {
	dsn := DSN("file:base.db?mode=ro")
	if !strings.HasPrefix(dsn, "file:base.db?mode=ro&_pragma=") {
		t.Fatalf("DSN(file:…?mode=ro) = %q, want the caller's query kept and ours appended", dsn)
	}
	if got := DSN("file:base.db"); !strings.HasPrefix(got, "file:base.db?_pragma=") {
		t.Fatalf("DSN(file:base.db) = %q", got)
	}
}

// A bare path containing '?' would be split by the driver at that '?' and a
// DIFFERENT file opened ("quoi?.db" became "quoi"); DSN must rewrite it as a
// percent-encoded file: URI. The test opens the database for real and checks
// the file lands at the exact path asked for.
//
// Unix-only, and deliberately so rather than by accident: Windows forbids '?'
// in a file name, so the scenario this guards cannot arise there — DSN's own
// doc comment says as much. Creating the file is what fails on Windows, not
// DSN, which is why the skip is here and not a t.Fatal (E.4 banned SILENT
// skips; this one names the reason). The half that DOES arise on every
// platform — accents, spaces and '%' — is covered by the test below, which
// runs on Windows too.
func TestDSN_PathWithQuestionMarkOpensTheRightFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows forbids '?' in a file name: the split-at-'?' scenario cannot arise")
	}

	dir := filepath.Join(t.TempDir(), "dossier é")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "quoi? 100%.db")

	dsn := DSN(path)
	if !strings.HasPrefix(dsn, "file:///") {
		t.Fatalf("DSN(%q) = %q, want a file: URI", path, dsn)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE t(x)`); err != nil {
		t.Fatalf("create through %q: %v", dsn, err)
	}
	if _, err := os.Stat(path); err != nil {
		entries, _ := os.ReadDir(dir)
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("database not created at %q (dir holds %q): %v", path, names, err)
	}
	var fk int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != 1 {
		t.Errorf("PRAGMA foreign_keys = %d (err %v), want 1: the query was lost", fk, err)
	}
}

// The everyday Windows path — "C:\\Users\\José\\Parties 100%.db" — has no '?'
// and so never takes the file: URI branch: DSN hands the accented path to the
// driver verbatim, and the driver hands it to SQLite verbatim. That contract
// is what this checks, on every platform, by opening the database for real.
// Nothing else in the suite opens a database through an accented path.
func TestDSN_AccentedPathOpensTheRightFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dossier é")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "parties 100%.db")

	db, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE t(x)`); err != nil {
		t.Fatalf("create through %q: %v", DSN(path), err)
	}
	if _, err := os.Stat(path); err != nil {
		entries, _ := os.ReadDir(dir)
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("database not created at %q (dir holds %q): %v", path, names, err)
	}
	var fk int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != 1 {
		t.Errorf("PRAGMA foreign_keys = %d (err %v), want 1: the query was lost", fk, err)
	}
}
