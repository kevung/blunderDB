package database

import (
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// preAutoincrementComment is the comment table as 2.45.0 and earlier declared
// it: identifiers are rowids, reused once the newest row is deleted.
const preAutoincrementComment = `CREATE TABLE comment (
		id INTEGER PRIMARY KEY,
		position_id INTEGER,
		text TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		modified_at DATETIME,
		origin TEXT NOT NULL DEFAULT 'unknown',
		author TEXT NOT NULL DEFAULT '',
		FOREIGN KEY(position_id) REFERENCES position(id) ON DELETE CASCADE
	)`

// copyDemoDatabase writes the sample library the GUI embeds to path: a real
// library, with comments of every origin and the triggers a release carries.
func copyDemoDatabase(t *testing.T, path string) {
	t.Helper()
	f, err := os.Open(filepath.Join("internal", "gui", "demo.db.gz"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, zr); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

// commentShape is everything the rebuild of comment could lose: its rows with
// every column, its columns, its foreign keys, and every index and trigger of
// the library (a trigger on another table may name comment).
type commentShape struct {
	Rows, Columns, ForeignKeys, Schema []string
}

func readCommentShape(t *testing.T, db *sql.DB) commentShape {
	t.Helper()
	strs := func(query string) []string {
		rows, err := db.Query(query)
		if err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		var out []string
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			parts := make([]string, len(vals))
			for i, v := range vals {
				if b, ok := v.([]byte); ok {
					v = string(b)
				}
				parts[i] = fmt.Sprintf("%s=%v", cols[i], v)
			}
			out = append(out, strings.Join(parts, "|"))
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		return out
	}
	// Whitespace in a statement is layout, not schema: an index rebuilt from
	// the reference statement keeps its meaning, not its spacing.
	schema := strs(`SELECT type, name, tbl_name, sql FROM sqlite_master WHERE type IN ('index', 'trigger') AND sql IS NOT NULL ORDER BY type, name`)
	for i, s := range schema {
		schema[i] = strings.Join(strings.Fields(s), " ")
	}
	return commentShape{
		Rows:        strs(`SELECT * FROM comment ORDER BY id`),
		Columns:     strs(`SELECT name, type, "notnull", dflt_value, pk FROM pragma_table_info('comment') ORDER BY cid`),
		ForeignKeys: strs(`SELECT "table", "from", "to", on_delete FROM pragma_foreign_key_list('comment')`),
		Schema:      schema,
	}
}

// TestMigrate_2_45_0_to_2_46_0_CommentRebuildKeepsEverything runs the
// migration on the demo library with its comment table put back in the
// 2.45.0 shape, and checks that the rebuild to AUTOINCREMENT keeps every row
// with every column, the columns, the foreign key and its cascade, every index
// and trigger of the library — and that an identifier no longer serves twice.
func TestMigrate_2_45_0_to_2_46_0_CommentRebuildKeepsEverything(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(tempDir(t), "demo_v2450.db")
	copyDemoDatabase(t, dbPath)

	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	raw.SetMaxOpenConns(1)
	for _, stmt := range []string{
		`PRAGMA foreign_keys = OFF`,
		`ALTER TABLE comment RENAME TO comment_new_shape`,
		preAutoincrementComment,
		`INSERT INTO comment SELECT id, position_id, text, created_at, modified_at, origin, author FROM comment_new_shape`,
		`DROP TABLE comment_new_shape`,
		`CREATE INDEX IF NOT EXISTS idx_comment_position ON comment(position_id, origin)`,
		`UPDATE metadata SET value = '2.45.0' WHERE key = 'database_version'`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	before := readCommentShape(t, raw)
	if err := raw.Close(); err != nil {
		t.Fatal(err)
	}
	if len(before.Rows) == 0 {
		t.Fatal("the demo library holds no comment: the test would prove nothing")
	}

	d := NewDatabase()
	if err := d.OpenDatabase(dbPath); err != nil {
		t.Fatalf("open the 2.45.0 demo: %v", err)
	}
	closeOnCleanup(t, d)
	if v, err := d.CheckDatabaseVersion(); err != nil || v != DatabaseVersion {
		t.Fatalf("version after migration = %q, %v; want %q", v, err, DatabaseVersion)
	}
	var ddl string
	if err := d.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'comment'`).Scan(&ddl); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ddl, "AUTOINCREMENT") {
		t.Fatalf("comment after migration is not AUTOINCREMENT:\n%s", ddl)
	}
	after := readCommentShape(t, d.db)
	if !reflect.DeepEqual(before.Rows, after.Rows) {
		t.Errorf("comment rows changed:\nbefore %v\nafter  %v", before.Rows, after.Rows)
	}
	if !reflect.DeepEqual(before.Columns, after.Columns) {
		t.Errorf("comment columns changed:\nbefore %v\nafter  %v", before.Columns, after.Columns)
	}
	if !reflect.DeepEqual(before.ForeignKeys, after.ForeignKeys) {
		t.Errorf("comment foreign keys changed:\nbefore %v\nafter  %v", before.ForeignKeys, after.ForeignKeys)
	}
	if !reflect.DeepEqual(before.Schema, after.Schema) {
		t.Errorf("indexes or triggers changed:\nbefore %v\nafter  %v", before.Schema, after.Schema)
	}
	var orphans int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check('comment')`).Scan(&orphans); err != nil || orphans != 0 {
		t.Fatalf("foreign key check on comment = %d, %v; want 0", orphans, err)
	}

	var maxID, pid int64
	if err := d.db.QueryRow(`SELECT MAX(id) FROM comment`).Scan(&maxID); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT position_id FROM comment WHERE id = ?`, maxID).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := d.db.Exec(`DELETE FROM comment WHERE id = ?`, maxID); err != nil {
		t.Fatal(err)
	}
	res, err := d.db.Exec(`INSERT INTO comment (position_id, text) VALUES (?, 'new')`, pid)
	if err != nil {
		t.Fatal(err)
	}
	if id, _ := res.LastInsertId(); id <= maxID {
		t.Fatalf("new comment id = %d after deleting the newest (%d): an identifier served twice", id, maxID)
	}

	// The cascade survives the rebuild: deleting the position takes its comments.
	if _, err := d.db.Exec(`DELETE FROM position WHERE id = ?`, pid); err != nil {
		t.Fatal(err)
	}
	var left int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM comment WHERE position_id = ?`, pid).Scan(&left); err != nil || left != 0 {
		t.Fatalf("comments of a deleted position = %d, %v; want 0 (cascade lost)", left, err)
	}
}
