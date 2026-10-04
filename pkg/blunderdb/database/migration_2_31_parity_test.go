package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// write2_30Library writes at path a library in the 2.30.0 shape a BMAB-sized
// one has: text dates and action labels, compact-array boards, no
// action_label table, provenance never derived, half its blobs legacy JSON,
// the match-date backfill half done and match_stats empty, one board still
// in the legacy full-Position JSON form. With swap, the action columns are
// declared `TEXT DEFAULT NULL`, a declaration retypeColumnsInteger declines,
// so the step converts them through its ADD/DROP/RENAME path.
func write2_30Library(t *testing.T, path string, swap bool) {
	t.Helper()
	d := NewDatabase()
	if err := d.SetupDatabase(path); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	if _, err := d.ImportXGMatch(filepath.Join("testdata", "test.xg")); err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	boards := boardsByID(t, d.db)
	legacyBoard := slices.Min(slices.Collect(maps.Keys(boards)))
	for id, b := range boards {
		state := engine.EncodeBoardCompact(b)
		if id == legacyBoard {
			js, err := json.Marshal(domain.Position{Board: b})
			if err != nil {
				t.Fatal(err)
			}
			state = string(js)
		} else if id%3 == 0 {
			// The same board with JSON whitespace: still a compact array.
			state = strings.ReplaceAll(state, ",", ", ")
		}
		if _, err := d.db.Exec(`UPDATE position SET state = ? WHERE id = ?`, state, id); err != nil {
			t.Fatal(err)
		}
	}
	legacy := map[int64][]byte{}
	if err := d.forEachRow(`SELECT id, data FROM analysis WHERE id % 2 = 0`, nil, func(r *sql.Rows) {
		var id int64
		var data []byte
		if r.Scan(&id, &data) == nil {
			legacy[id] = data
		}
	}); err != nil {
		t.Fatal(err)
	}
	for id, data := range legacy {
		js, err := engine.DecompressAnalysisData(data)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := d.db.Exec(`UPDATE analysis SET data = ? WHERE id = ?`, js, id); err != nil {
			t.Fatal(err)
		}
	}
	stmts := []string{
		`ALTER TABLE move DROP COLUMN error_mp`,
		`DROP INDEX idx_analysis_met`,
		`ALTER TABLE analysis DROP COLUMN met_id`,
		`DROP TABLE lesson_progress`,
		`DROP TABLE match_equity_table`,
		`DROP INDEX idx_analysis_provenance_pending`,
		`UPDATE position SET match_date = (SELECT MIN(m.match_date) FROM move mv
		   JOIN game g ON g.id = mv.game_id JOIN match m ON m.id = g.match_id
		  WHERE mv.position_id = position.id)`,
		`UPDATE position SET match_date = NULL WHERE id > (SELECT MAX(id) / 2 FROM position)`,
		`INSERT INTO metadata (key, value) SELECT '` + matchDateBackfillKey + `', MAX(id) / 2 + 1 FROM position`,
		`UPDATE analysis SET creation_date = datetime(creation_date, 'unixepoch')`,
		`UPDATE analysis SET analysis_engine = NULL, analysis_depth = NULL`,
		`DELETE FROM match_stats`,
		`CREATE INDEX idx_analysis_engine ON analysis(analysis_engine)`,
		`CREATE INDEX idx_analysis_depth ON analysis(analysis_depth)`,
		`UPDATE metadata SET value = '2.30.0' WHERE key = 'database_version'`,
	}
	textDecl := `TEXT`
	if swap {
		textDecl = `TEXT DEFAULT NULL`
	}
	for _, c := range actionColumns2_31 {
		stmts = append(stmts,
			`ALTER TABLE `+c.table+` ADD COLUMN `+c.column+`_text `+textDecl,
			`UPDATE `+c.table+` SET `+c.column+`_text = `+sqlshared.ActionLabelSQL(c.column),
			`ALTER TABLE `+c.table+` DROP COLUMN `+c.column,
			`ALTER TABLE `+c.table+` RENAME COLUMN `+c.column+`_text TO `+c.column)
	}
	stmts = append(stmts,
		`DROP TABLE action_label`,
		`UPDATE move SET cube_action = 'Unknown(-1)' WHERE id = (SELECT MIN(id) FROM move WHERE move_type = 'cube')`,
		`UPDATE move SET cube_action = NULL WHERE id = (SELECT MAX(id) FROM move WHERE move_type = 'cube')`,
		`UPDATE analysis SET best_cube_action = 'Doppel, Annahme' WHERE id = (SELECT MIN(id) FROM analysis)`,
		// A label that reads as a number must stay a label.
		`UPDATE analysis SET best_cube_action = '42' WHERE id = (SELECT MAX(id) FROM analysis)`,
	)
	for _, stmt := range stmts {
		if _, err := d.db.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	in, err := os.Open(from)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(to)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(out, in); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

// migrateByReference runs the 2.30.0 → 2.31.0 step as first written, and
// its provenance backfill, then opens the library so the chain finishes the
// rest exactly as it does for the current step.
func migrateByReference(t *testing.T, path string) {
	t.Helper()
	ctx := context.Background()
	d := NewDatabase()
	var err error
	if d.db, err = sql.Open("sqlite", sqlite.DSN(path)); err != nil {
		t.Fatal(err)
	}
	if err := d.referenceMigrate230To231(ctx); err != nil {
		t.Fatalf("reference step: %v", err)
	}
	if err := d.referenceBackfillAnalysisProvenance(ctx); err != nil {
		t.Fatalf("reference provenance: %v", err)
	}
	if _, err := d.db.Exec(`UPDATE metadata SET value = '2.31.0' WHERE key = 'database_version'`); err != nil {
		t.Fatal(err)
	}
	if err := d.db.Close(); err != nil {
		t.Fatal(err)
	}
	open := NewDatabase()
	if err := open.OpenDatabase(path); err != nil {
		t.Fatalf("open after the reference step: %v", err)
	}
	if err := open.Close(); err != nil {
		t.Fatal(err)
	}
}

// libraryContent is every cell of every table, keyed by table, rowid and
// column name (column order differs between a retyped and a rebuilt
// column), with each table's declared column types and the index DDL.
func libraryContent(t *testing.T, path string) map[string]string {
	t.Helper()
	db, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	out := map[string]string{}
	var tables []string
	eachRow(t, db, `SELECT type, name, COALESCE(sql, '') FROM sqlite_master ORDER BY name`, func(r *sql.Rows) {
		var typ, name, ddl string
		if err := r.Scan(&typ, &name, &ddl); err != nil {
			t.Fatal(err)
		}
		if typ == "table" {
			tables = append(tables, name)
		} else {
			out["schema "+typ+" "+name] = ddl
		}
	})
	for _, table := range tables {
		var cols []string
		eachRow(t, db, `SELECT name, type FROM pragma_table_info('`+table+`')`, func(r *sql.Rows) {
			var name, typ string
			if err := r.Scan(&name, &typ); err != nil {
				t.Fatal(err)
			}
			cols = append(cols, name)
			out["column "+table+"."+name] = typ
		})
		slices.Sort(cols)
		quoted := make([]string, len(cols))
		for i, c := range cols {
			quoted[i] = `quote("` + c + `")`
		}
		rowid := "rowid"
		if strings.HasPrefix(table, "sqlite_stat") {
			rowid = "0"
		}
		vals := make([]sql.NullString, len(cols)+1)
		ptrs := make([]any, len(vals))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		n := 0
		eachRow(t, db, `SELECT `+rowid+`, `+strings.Join(quoted, ", ")+` FROM "`+table+`"`, func(r *sql.Rows) {
			if err := r.Scan(ptrs...); err != nil {
				t.Fatal(err)
			}
			key := vals[0].String
			if rowid == "0" {
				key = fmt.Sprint(n)
			}
			for i, c := range cols {
				if table == "match_stats" && c == "computed_at" {
					continue // when the open ran, not what it computed
				}
				out[table+"#"+key+"."+c] = vals[i+1].String
			}
			n++
		})
	}
	return out
}

// TestMigrate_2_31_MatchesReference holds the single-pass, retype-in-place
// step, its ADD/DROP/RENAME fallback, and the pipelined provenance backfill to
// the step as first written: the same library must come out cell for cell.
func TestMigrate_2_31_MatchesReference(t *testing.T) {
	t.Parallel()
	for _, swap := range []bool{false, true} {
		t.Run(fmt.Sprintf("swap=%v", swap), func(t *testing.T) {
			t.Parallel()
			matchesReference(t, swap)
		})
	}
}

func matchesReference(t *testing.T, swap bool) {
	dir := tempDir(t)
	source := filepath.Join(dir, "v2300.db")
	write2_30Library(t, source, swap)
	ref, cur := filepath.Join(dir, "reference.db"), filepath.Join(dir, "current.db")
	copyFile(t, source, ref)
	copyFile(t, source, cur)

	migrateByReference(t, ref)
	d := NewDatabase()
	if err := d.OpenDatabase(cur); err != nil {
		t.Fatalf("open v2.30.0 library: %v", err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	want, got := libraryContent(t, ref), libraryContent(t, cur)
	if len(want) < 1000 {
		t.Fatalf("reference content has %d cells; the fixture is too small to compare", len(want))
	}
	for _, k := range slices.Sorted(maps.Keys(want)) {
		if got[k] != want[k] {
			t.Errorf("%s = %q, want %q", k, got[k], want[k])
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("%s = %q, absent from the reference", k, got[k])
		}
	}
	for _, k := range []string{"column analysis.best_cube_action", "column move.cube_action"} {
		if got[k] != "INTEGER" {
			t.Errorf("%s declared %q, want INTEGER", k, got[k])
		}
	}
}

// TestRetypeColumnsInteger: a column declared plainly TEXT is retyped, and
// every connection, the editing one included, then stores integers as
// integers; a declaration it cannot edit safely is refused untouched.
func TestRetypeColumnsInteger(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	path := filepath.Join(tempDir(t), "retype.db")
	db, err := sql.Open("sqlite", sqlite.DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, a TEXT, \"b\" text,c TEXT NOT NULL DEFAULT '', at TEXT);" +
		"INSERT INTO t (a, b, c, at) VALUES ('x', 'y', 'z', 'w')"); err != nil {
		t.Fatal(err)
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if ok, err := retypeColumnsInteger(ctx, conn, "t", []string{"c"}, nil); err != nil || ok {
		t.Fatalf("retype of a constrained column = %v, %v; want refused", ok, err)
	}
	if ok, err := retypeColumnsInteger(ctx, conn, "t", []string{"a", "b"}, nil); err != nil || !ok {
		t.Fatalf("retype = %v, %v; want done", ok, err)
	}
	other, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	for i, q := range []*sql.Conn{conn, other} {
		if _, err := q.ExecContext(ctx, `UPDATE t SET a = '7', b = 8`); err != nil {
			t.Fatal(err)
		}
		var ta, tb, tat, tc string
		if err := q.QueryRowContext(ctx, `SELECT typeof(a), typeof(b), typeof(c), (SELECT type FROM pragma_table_info('t') WHERE name = 'at') FROM t`).Scan(&ta, &tb, &tc, &tat); err != nil {
			t.Fatal(err)
		}
		if ta != "integer" || tb != "integer" || tc != "text" || tat != "TEXT" {
			t.Errorf("connection %d: typeof a, b, c = %s, %s, %s, at declared %s; want integer, integer, text, TEXT", i, ta, tb, tc, tat)
		}
	}
	var check string
	if err := db.QueryRow(`PRAGMA integrity_check`).Scan(&check); err != nil || check != "ok" {
		t.Fatalf("integrity_check = %q, %v", check, err)
	}
}

// TestBoardStateBlob holds the Go reading of a compact board to the SQL
// expression it stands in for, on the forms it accepts and declines.
func TestBoardStateBlob(t *testing.T) {
	t.Parallel()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	board := func(sep string, vals ...string) string {
		v := make([]string, 28)
		for i := range v {
			v[i] = fmt.Sprint(i%7 - 3)
		}
		copy(v, vals)
		return "[" + strings.Join(v, sep) + "]"
	}
	cases := []string{
		board(","), board(", "), " " + board(",\n") + "\t", board(",", "-256"), board(",", "255", "1000"),
		board(",", "-0"), board(",", "-257"), board(",", "01"), board(",", "1.5"), board(",", "1e2"),
		board(",", "999999999999999999999"), "[1,2,3]", board(",") + ",", "[" + board(",")[1:len(board(","))-1] + ",4]",
		"[]", "{\"board\":1}", "", "[1,2,]",
	}
	for _, c := range cases {
		var want []byte
		wantErr := db.QueryRow(`SELECT CASE WHEN json_array_length(state) = 28 THEN `+boardStateSQL+
			` END FROM (SELECT ? AS state)`, c).Scan(&want)
		blob, ok := boardStateBlob([]byte(c))
		if !ok {
			continue
		}
		if wantErr != nil {
			t.Errorf("%q: accepted by Go, SQL fails: %v", c, wantErr)
			continue
		}
		if string(blob) != string(want) {
			t.Errorf("%q: Go %x, SQL %x", c, blob, want)
		}
	}
	for _, c := range []string{board(","), board(", "), board(",", "-256")} {
		if _, ok := boardStateBlob([]byte(c)); !ok {
			t.Errorf("%q declined; want the Go path", c)
		}
	}
	for _, c := range []string{board(",", "-257"), board(",", "01"), board(",", "1.5"), "[1,2,3]", "[1,2,]"} {
		if _, ok := boardStateBlob([]byte(c)); ok {
			t.Errorf("%q accepted; want the SQL path", c)
		}
	}
}

// TestMigrate_2_31_ResumesAfterCancel cancels the open inside each phase of
// the crossing, several transactions into its table pass, then opens again:
// the library must come out as if the first open had never been cut. On the
// ADD/DROP/RENAME path it also cuts between a DROP and its RENAME.
func TestMigrate_2_31_ResumesAfterCancel(t *testing.T) {
	t.Parallel()
	for _, swap := range []bool{false, true} {
		t.Run(fmt.Sprintf("swap=%v", swap), func(t *testing.T) {
			t.Parallel()
			resumesAfterCancel(t, swap)
		})
	}
}

func resumesAfterCancel(t *testing.T, swap bool) {
	dir := tempDir(t)
	source := filepath.Join(dir, "v2300.db")
	write2_30Library(t, source, swap)
	ref := filepath.Join(dir, "reference.db")
	copyFile(t, source, ref)
	migrateByReference(t, ref)
	want := libraryContent(t, ref)

	// The table passes are cut on their second batch; the backfills, one
	// batch on this library, on their first; the swap on its first DROP.
	cuts := map[string]int{"position_2_31": 2, "analysis_2_31": 2, "move_2_31": 2,
		"position_match_date": 1, "analysis_provenance": 1, "match_stats": 1}
	if swap {
		cuts[swapCut] = 1
	}
	for _, cut := range slices.Sorted(maps.Keys(cuts)) {
		t.Run(cut, func(t *testing.T) {
			path := filepath.Join(dir, cut+".db")
			copyFile(t, source, path)
			d := NewDatabase()
			d.convertBatchSize = 7
			seen := 0
			if cut == swapCut {
				d.afterActionColumnDrop = func() error {
					seen++
					return errSwapCut
				}
			}
			d.SetMigrationProgress(func(phase string, done, total int) {
				if phase == cut {
					if seen++; seen == cuts[cut] {
						d.CancelImport()
					}
				}
			})
			err := d.OpenDatabase(path)
			_ = d.Close()
			if seen < cuts[cut] {
				t.Fatalf("phase %s reported %d times; want it cut on report %d", cut, seen, cuts[cut])
			}
			if err == nil {
				t.Fatalf("open cut in %s succeeded", cut)
			}

			again := NewDatabase()
			if err := again.OpenDatabase(path); err != nil {
				t.Fatalf("open after the cut: %v", err)
			}
			if err := again.Close(); err != nil {
				t.Fatal(err)
			}
			got := libraryContent(t, path)
			for _, k := range slices.Sorted(maps.Keys(want)) {
				if got[k] != want[k] {
					t.Errorf("%s = %q, want %q", k, got[k], want[k])
				}
			}
			if len(got) != len(want) {
				t.Errorf("%d cells, the reference %d", len(got), len(want))
			}
		})
	}
}

// swapCut names the cut between the DROP and the RENAME of an action-column
// swap, which no progress phase reports.
const swapCut = "action_swap"

var errSwapCut = errors.New("cut between DROP and RENAME")

func eachRow(t *testing.T, db *sql.DB, q string, each func(*sql.Rows)) {
	t.Helper()
	rows, err := db.Query(q)
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	defer rows.Close()
	for rows.Next() {
		each(rows)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}
