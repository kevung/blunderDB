package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A trigger written by an older body is replaced when the journal is ensured
// again: IF NOT EXISTS would keep the old one for good.
func TestEnsureChangeLog_ReplacesATriggerBody(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lib.db")
	s, err := sqlite.Open(ctx, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	body := func() string {
		var sqlText string
		if err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE name = 'change_log_collection_u'`).Scan(&sqlText); err != nil {
			t.Fatal(err)
		}
		return sqlText
	}
	want := body()
	for _, q := range []string{
		`DROP TRIGGER change_log_collection_u`,
		`CREATE TRIGGER change_log_collection_u AFTER UPDATE ON collection BEGIN SELECT 'old'; END`,
	} {
		if _, err := db.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := sqlite.EnsureChangeLog(ctx, db); err != nil {
		t.Fatal(err)
	}
	if got := body(); got != want || strings.Contains(got, "SELECT 'old'") {
		t.Errorf("the old body stayed:\n%s", got)
	}
}
