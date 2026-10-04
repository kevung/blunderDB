package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"regexp"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
	sqlitedriver "modernc.org/sqlite"
)

// prunedIndexes2_31 are the indexes the 2.31.0 schema no longer declares:
// no query plan reads them (tasks/plan-grosses-bases-2026-10/SCHEMA-2-31.md),
// and the provenance backfill reads idx_analysis_provenance_pending instead.
// The two date indexes are dropped too, so that rewriting their column does
// not update them row by row; EnsureSchema rebuilds them after the chain.
var prunedIndexes2_31 = []string{
	"idx_analysis_engine",
	"idx_analysis_depth",
	"idx_analysis_creation_date",
	"idx_position_match_date",
}

// migrate_2_30_0_to_2_31_0 is the weight wave of the large-library plan
// (ADR-0071), plus additions EnsureSchema creates from the one schema after
// the chain: match_equity_table and analysis.met_id (ADR-0068),
// lesson_progress (ADR-0069) and move.error_mp, all added NULL. Scoring the
// existing moves is not part of it: that is the resumable
// MatchStore.ScoreMoves pass, run outside the open.
//
// The step rewrites the two date columns 2.30.0 stored as text into Unix
// seconds, position.state into its 28 bytes, and the action labels into
// codes. A position whose date is still NULL here (the 2.30.0 backfill not
// finished) is dated by that backfill after the chain, already as an integer.
//
// Each table is rewritten in a single pass, every conversion of that table in
// the same UPDATE: on a 15 M-row library each pass rewrites every page, so a
// pass per column multiplied the work. The action columns are retyped in
// place (retypeColumnsInteger) rather than rebuilt by DROP COLUMN, which
// rewrote the 6 GB analysis table in one transaction and left as large a WAL.
// Every conversion only touches values still in their 2.30.0 form, so an
// interrupted step resumes where it stopped.
func (d *Database) migrate_2_30_0_to_2_31_0(ctx context.Context) error {
	for _, name := range prunedIndexes2_31 {
		if _, err := d.db.ExecContext(ctx, `DROP INDEX IF EXISTS `+name); err != nil {
			return fmt.Errorf("dropping %s: %w", name, err)
		}
	}
	if _, err := d.db.ExecContext(ctx, actionLabelDDL); err != nil {
		return fmt.Errorf("creating action_label: %w", err)
	}
	conn, err := d.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Every label is registered before any column is converted, in the order
	// the codes have always been handed out: analysis, then move.
	todo := map[string][]string{}
	for _, a := range actionColumns2_31Step {
		for _, col := range a.columns {
			pending, err := actionColumnPending(ctx, conn, a.table, col)
			if err != nil {
				return err
			}
			if !pending {
				continue
			}
			todo[a.table] = append(todo[a.table], col)
			if err := registerActionLabels(ctx, conn, a.table, col); err != nil {
				return fmt.Errorf("registering the labels of %s.%s: %w", a.table, col, err)
			}
		}
	}

	var position []conversion
	if ok, err := hasColumn(ctx, conn, "position", "match_date"); err != nil {
		return err
	} else if ok {
		position = append(position, unixDateConversion("match_date"))
	}
	position = append(position, boardStateConversion)
	if err := d.convertTable(ctx, conn, "position", position, "position_2_31"); err != nil {
		return fmt.Errorf("converting position: %w", err)
	}

	for _, a := range actionColumns2_31Step {
		var convs []conversion
		if a.table == "analysis" {
			if ok, err := hasColumn(ctx, conn, "analysis", "creation_date"); err != nil {
				return err
			} else if ok {
				convs = append(convs, unixDateConversion("creation_date"))
			}
		}
		codes, err := prepareActionColumns(ctx, conn, a.table, todo[a.table])
		if err != nil {
			return fmt.Errorf("retyping the action columns of %s: %w", a.table, err)
		}
		convs = append(convs, codes...)
		if err := d.convertTable(ctx, conn, a.table, convs, a.table+"_2_31"); err != nil {
			return fmt.Errorf("converting %s: %w", a.table, err)
		}
		if err := finishActionColumns(ctx, conn, a.table, todo[a.table], d.afterActionColumnDrop); err != nil {
			return fmt.Errorf("converting %s action labels to codes: %w", a.table, err)
		}
	}
	return nil
}

// actionColumns2_31Step are the columns 2.30.0 stored as action labels, in the
// order their labels are registered.
var actionColumns2_31Step = []struct {
	table   string
	columns []string
}{
	{"analysis", []string{"best_cube_action"}},
	{"move", []string{"move_type", "cube_action"}},
}

// actionLabelDDL is the action_label table of the 2.31.0 schema, which the
// step fills before EnsureSchema runs.
const actionLabelDDL = `CREATE TABLE IF NOT EXISTS action_label (
	code  INTEGER PRIMARY KEY,
	label TEXT NOT NULL UNIQUE
)`

// conversion is one column of a table pass: its new value, and the
// condition under which a row still holds the 2.30.0 form.
type conversion struct {
	column  string
	value   string
	pending string
}

func unixDateConversion(column string) conversion {
	return conversion{
		column:  column,
		value:   sqlite.UnixFromMatchDateSQL(column),
		pending: `typeof(` + column + `) = 'text'`,
	}
}

// boardStateConversion rewrites a compact-array state ("[0,1,-3,…]", 28
// values) as the 28 signed bytes of engine.EncodeBoardState. The Go function
// boardStateBlobFunc reads the common form in one scan; what it declines
// falls back to the SQL expression, 28 json_extract calls that define the
// result. A legacy full-Position JSON state ('{') is left as it is: the
// decoder still reads it.
var boardStateConversion = conversion{
	column: "state",
	value: `COALESCE(` + boardStateBlobFunc + `(state),
		CASE WHEN json_array_length(state) = ` + fmt.Sprint(engine.BoardStateLen) + `
		THEN ` + boardStateSQL + ` ELSE state END)`,
	pending: `typeof(state) = 'text' AND substr(state, 1, 1) = '['`,
}

// boardStateSQL is the 28 bytes of a compact-array state, in SQL.
var boardStateSQL = func() string {
	var format, args strings.Builder
	for i := range engine.BoardStateLen {
		format.WriteString("%02X")
		fmt.Fprintf(&args, ", (CAST(json_extract(state, '$[%d]') AS INTEGER) + 256) %% 256", i)
	}
	return `unhex(printf('` + format.String() + `'` + args.String() + `))`
}()

// boardStateBlobFunc is the SQL name of boardStateBlob.
const boardStateBlobFunc = "blunderdb_board_state_blob"

func init() {
	sqlitedriver.MustRegisterDeterministicScalarFunction(boardStateBlobFunc, 1,
		func(_ *sqlitedriver.FunctionContext, args []driver.Value) (driver.Value, error) {
			var s []byte
			switch v := args[0].(type) {
			case string:
				s = []byte(v)
			case []byte:
				s = v
			default:
				return nil, nil
			}
			if b, ok := boardStateBlob(s); ok {
				return b, nil
			}
			return nil, nil
		})
}

// boardStateBlob reads a strict JSON array of engine.BoardStateLen integers
// and returns the bytes the SQL expression of boardStateConversion yields for
// it, (v + 256) % 256 each. It declines (false) anything else — another
// length, a non-integer, a value below -256 (where SQL's truncating % and a
// byte part ways), JSON5 — and the SQL expression decides.
func boardStateBlob(s []byte) ([]byte, bool) {
	out := make([]byte, 0, engine.BoardStateLen)
	i := 0
	skip := func() {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
			i++
		}
	}
	skip()
	if i >= len(s) || s[i] != '[' {
		return nil, false
	}
	i++
	for {
		skip()
		neg := false
		if i < len(s) && s[i] == '-' {
			neg = true
			i++
		}
		start := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		digits := s[start:i]
		if len(digits) == 0 || len(digits) > 18 || (len(digits) > 1 && digits[0] == '0') {
			return nil, false
		}
		var v int64
		for _, c := range digits {
			v = v*10 + int64(c-'0')
		}
		if neg {
			v = -v
		}
		if v < -256 || len(out) == engine.BoardStateLen {
			return nil, false
		}
		out = append(out, byte((v+256)%256))
		skip()
		if i >= len(s) {
			return nil, false
		}
		if s[i] == ']' {
			i++
			break
		}
		if s[i] != ',' {
			return nil, false
		}
		i++
	}
	skip()
	if i != len(s) || len(out) != engine.BoardStateLen {
		return nil, false
	}
	return out, true
}

// retypedMarker is the metadata key that says table.col was retyped in place
// and may still hold labels: written in the transaction of the schema edit,
// deleted once the table pass is over. A column declared INTEGER without it
// was never a label column of this step and is left alone.
func retypedMarker(table, col string) string { return "migrate_2_31_retyped:" + table + "." + col }

// actionColumnPending reports whether table.col still has labels to convert:
// declared other than INTEGER, or retyped by an interrupted run of the step.
func actionColumnPending(ctx context.Context, conn *sql.Conn, table, col string) (bool, error) {
	typ, err := columnType(ctx, conn, table, col)
	if err != nil || typ == "" {
		return false, err
	}
	if !strings.EqualFold(typ, "INTEGER") {
		return true, nil
	}
	var n int
	err = conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM metadata WHERE key = ?`, retypedMarker(table, col)).Scan(&n)
	return n > 0, err
}

// registerActionLabels gives a code to every label of table.col the fixed
// list lacks. Only text values are labels: a resumed step finds the codes it
// already wrote in the same column.
func registerActionLabels(ctx context.Context, conn *sql.Conn, table, col string) error {
	_, err := conn.ExecContext(ctx, `INSERT INTO action_label (code, label)
		SELECT (SELECT COALESCE(MAX(code), ?) FROM action_label) + ROW_NUMBER() OVER (ORDER BY v), v
		FROM (SELECT DISTINCT CAST(`+col+` AS TEXT) AS v FROM `+table+`
		      WHERE typeof(`+col+`) = 'text' AND CAST(`+col+` AS TEXT) NOT IN `+sqlshared.FixedActionLabelsSQL()+`
		        AND CAST(`+col+` AS TEXT) NOT IN (SELECT label FROM action_label))`,
		domain.FirstRegisteredActionCode-1)
	return err
}

// prepareActionColumns makes each action column of table able to hold its
// codes and returns the conversions that write them. A TEXT column is retyped
// INTEGER in place (retypeColumnsInteger); when its declaration is not the
// plain form that can be edited safely, an INTEGER <col>_code column is added
// instead, filled by the same pass and swapped in by finishActionColumns.
func prepareActionColumns(ctx context.Context, conn *sql.Conn, table string, columns []string) ([]conversion, error) {
	var convs, retype []string
	var swap []string
	for _, col := range columns {
		typ, err := columnType(ctx, conn, table, col)
		if err != nil {
			return nil, err
		}
		switch {
		case typ == "":
			continue
		case strings.EqualFold(typ, "INTEGER"):
			// Retyped by an interrupted run (its marker says so): the
			// labels its pass did not reach remain to convert.
			convs = append(convs, col)
		default:
			retype = append(retype, col)
		}
	}
	if len(retype) > 0 {
		done, err := retypeColumnsInteger(ctx, conn, table, retype, func(tx *sql.Tx) error {
			for _, col := range retype {
				if _, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO metadata (key, value) VALUES (?, '1')`,
					retypedMarker(table, col)); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if done {
			convs = append(convs, retype...)
		} else {
			swap = retype
		}
	}
	var out []conversion
	for _, col := range convs {
		out = append(out, conversion{
			column:  col,
			value:   sqlshared.ActionLabelCodesSQL(`CAST(` + col + ` AS TEXT)`),
			pending: `typeof(` + col + `) = 'text'`,
		})
	}
	for _, col := range swap {
		if ok, err := hasColumn(ctx, conn, table, col+"_code"); err != nil {
			return nil, err
		} else if !ok {
			if _, err := conn.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN `+col+`_code INTEGER`); err != nil {
				return nil, err
			}
		}
		out = append(out, conversion{
			column:  col + `_code`,
			value:   sqlshared.ActionLabelCodesSQL(col),
			pending: col + `_code IS NULL AND ` + col + ` IS NOT NULL`,
		})
	}
	return out, nil
}

// finishActionColumns closes the conversion of table's action columns once
// the table pass is over: the in-place ones lose their marker, the
// <col>_code columns prepareActionColumns had to add are swapped in. The
// DROP and the RENAME of a swap share one transaction: committed apart, a
// cut between them would leave the codes under <col>_code with nothing left
// to say the column still has to be renamed. afterDrop, nil outside tests,
// runs between the two.
func finishActionColumns(ctx context.Context, conn *sql.Conn, table string, columns []string, afterDrop func() error) error {
	for _, col := range columns {
		if _, err := conn.ExecContext(ctx, `DELETE FROM metadata WHERE key = ?`, retypedMarker(table, col)); err != nil {
			return err
		}
		ok, err := hasColumn(ctx, conn, table, col+"_code")
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := swapActionColumn(ctx, conn, table, col, afterDrop); err != nil {
			return err
		}
	}
	return nil
}

func swapActionColumn(ctx context.Context, conn *sql.Conn, table, col string, afterDrop func() error) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `ALTER TABLE `+table+` DROP COLUMN `+col); err != nil {
		return err
	}
	if afterDrop != nil {
		if err := afterDrop(); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE `+table+` RENAME COLUMN `+col+`_code TO `+col); err != nil {
		return err
	}
	return tx.Commit()
}

// retypeColumnsInteger changes the declared type of table.columns from TEXT
// to INTEGER by editing the CREATE TABLE text, the procedure SQLite documents
// for schema changes that leave the stored records valid
// (https://sqlite.org/lang_altertable.html#otheralter): every record stays as
// it is, only the affinity applied to the values written next changes. It
// edits nothing and reports false unless each column is declared exactly
// `<name> TEXT` followed by a comma or the closing parenthesis.
//
// The schema cookie is bumped so that every connection, this one included,
// reloads the schema; the caller checks the new type on conn before writing.
func retypeColumnsInteger(ctx context.Context, conn *sql.Conn, table string, columns []string, also func(*sql.Tx) error) (bool, error) {
	var ddl string
	if err := conn.QueryRowContext(ctx,
		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&ddl); err != nil {
		return false, err
	}
	for _, col := range columns {
		re := regexp.MustCompile("(?i)([(,]\\s*[\"`\\[]?" + regexp.QuoteMeta(col) + "[\"`\\]]?\\s+)TEXT(\\s*[,)])")
		if len(re.FindAllStringIndex(ddl, -1)) != 1 {
			return false, nil
		}
		ddl = re.ReplaceAllString(ddl, "${1}INTEGER${2}")
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var version int64
	if err := tx.QueryRowContext(ctx, `PRAGMA schema_version`).Scan(&version); err != nil {
		return false, err
	}
	for _, stmt := range []string{
		`PRAGMA writable_schema = ON`,
		`UPDATE sqlite_master SET sql = ` + sqlQuote(ddl) + ` WHERE type = 'table' AND name = ` + sqlQuote(table),
		fmt.Sprintf(`PRAGMA schema_version = %d`, version+1),
		`PRAGMA writable_schema = OFF`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return false, fmt.Errorf("%s: %w", stmt, err)
		}
	}
	if also != nil {
		if err := also(tx); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	for _, col := range columns {
		typ, err := columnType(ctx, conn, table, col)
		if err != nil {
			return false, err
		}
		if !strings.EqualFold(typ, "INTEGER") {
			return false, fmt.Errorf("%s.%s still declared %q after the schema edit", table, col, typ)
		}
	}
	return true, nil
}

func sqlQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }

func columnType(ctx context.Context, conn *sql.Conn, table, col string) (string, error) {
	var typ string
	err := conn.QueryRowContext(ctx,
		`SELECT type FROM pragma_table_info(?) WHERE name = ?`, table, col).Scan(&typ)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return typ, err
}

func hasColumn(ctx context.Context, conn *sql.Conn, table, col string) (bool, error) {
	typ, err := columnType(ctx, conn, table, col)
	if err != nil {
		return false, err
	}
	if typ != "" {
		return true, nil
	}
	// A column declared without a type has an empty type too.
	var n int
	err = conn.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, col).Scan(&n)
	return n > 0, err
}

// convertBatch is how many ids one conversion transaction covers: large
// enough to amortise the commit, small enough that the WAL a batch leaves
// stays in the tens of megabytes and is recycled by the next checkpoint.
const convertBatch = 50000

// convertTable applies convs to table in one pass, in id ranges, each range
// one transaction on conn; a row none of the conversions has left to do is
// not rewritten. The WAL is truncated at the end: the pass rewrote the whole
// table, and the file would otherwise keep its high-water mark.
func (d *Database) convertTable(ctx context.Context, conn *sql.Conn, table string, convs []conversion, phase string) error {
	if len(convs) == 0 {
		return nil
	}
	set := make([]string, len(convs))
	pending := make([]string, len(convs))
	for i, c := range convs {
		set[i] = c.column + ` = CASE WHEN ` + c.pending + ` THEN ` + c.value + ` ELSE ` + c.column + ` END`
		pending[i] = `(` + c.pending + `)`
	}
	update := `UPDATE ` + table + ` SET ` + strings.Join(set, ", ") +
		` WHERE id >= ? AND id < ? AND (` + strings.Join(pending, " OR ") + `)`
	var maxID int64
	if err := conn.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM `+table).Scan(&maxID); err != nil {
		return err
	}
	batch := int64(convertBatch)
	if d.convertBatchSize > 0 {
		batch = d.convertBatchSize
	}
	for next := int64(0); next <= maxID; next += batch {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := next + batch
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, update, next, end); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		done := end
		if done > maxID {
			done = maxID
		}
		d.emitMigrationProgress(phase, int(done), int(maxID))
	}
	return checkpointWAL(ctx, conn)
}

// checkpointWAL copies the WAL back into the database and truncates it, so
// that a migration's disk peak is one phase's WAL rather than the sum of
// them. A checkpoint a reader keeps from completing is not an error: the WAL
// is merely left as it is.
func checkpointWAL(ctx context.Context, q interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}) error {
	_, err := q.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}
