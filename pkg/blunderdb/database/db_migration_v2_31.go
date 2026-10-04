package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
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
// lesson_progress (ADR-0069), study_mark (the study queue's explicit "seen"
// gesture) and move.error_mp, all added NULL or empty. Scoring the
// existing moves is not part of it: that is the resumable
// MatchStore.ScoreMoves pass, run outside the open.
//
// The step rewrites the two date columns 2.30.0 stored as text into Unix
// seconds. A position whose date is still NULL here (the 2.30.0 backfill not
// finished) is dated by that backfill after the chain, already as an integer.
func (d *Database) migrate_2_30_0_to_2_31_0(ctx context.Context) error {
	for _, name := range prunedIndexes2_31 {
		if _, err := d.db.ExecContext(ctx, `DROP INDEX IF EXISTS `+name); err != nil {
			return fmt.Errorf("dropping %s: %w", name, err)
		}
	}
	conversions := []struct{ table, column, phase string }{
		{"position", "match_date", "position_match_date_unix"},
		{"analysis", "creation_date", "analysis_creation_date_unix"},
	}
	for _, c := range conversions {
		if err := d.unixDateColumn(ctx, c.table, c.column, c.phase); err != nil {
			return fmt.Errorf("converting %s.%s to Unix seconds: %w", c.table, c.column, err)
		}
	}
	if err := d.batchByID(ctx, "position", binaryStateSQL, "position_state_binary"); err != nil {
		return fmt.Errorf("converting position.state to binary: %w", err)
	}
	if _, err := d.db.ExecContext(ctx, actionLabelDDL); err != nil {
		return fmt.Errorf("creating action_label: %w", err)
	}
	actions := []struct {
		table   string
		columns []string
	}{
		{"analysis", []string{"best_cube_action"}},
		{"move", []string{"move_type", "cube_action"}},
	}
	for _, a := range actions {
		if err := d.actionCodeColumns(ctx, a.table, a.columns); err != nil {
			return fmt.Errorf("converting %s action labels to codes: %w", a.table, err)
		}
	}
	return nil
}

// actionLabelDDL is the action_label table of the 2.31.0 schema, which the
// step fills before EnsureSchema runs.
const actionLabelDDL = `CREATE TABLE IF NOT EXISTS action_label (
	code  INTEGER PRIMARY KEY,
	label TEXT NOT NULL UNIQUE
)`

// actionCodeColumns rewrites the text action labels of table.columns as
// action codes (domain.ActionCode). The labels the fixed list lacks are
// registered first; each column is then rebuilt as an INTEGER column — a text
// column would store the codes back as text — filled in id ranges, the text
// one dropped and the new one renamed. A column already INTEGER, or absent,
// is left alone.
func (d *Database) actionCodeColumns(ctx context.Context, table string, columns []string) error {
	var todo []string
	for _, col := range columns {
		var typ string
		err := d.db.QueryRowContext(ctx,
			`SELECT type FROM pragma_table_info(?) WHERE name = ?`, table, col).Scan(&typ)
		if err == sql.ErrNoRows || strings.EqualFold(typ, "INTEGER") {
			continue
		}
		if err != nil {
			return err
		}
		todo = append(todo, col)
	}
	if len(todo) == 0 {
		return nil
	}
	set := make([]string, len(todo))
	for i, col := range todo {
		if _, err := d.db.ExecContext(ctx, `INSERT INTO action_label (code, label)
			SELECT (SELECT COALESCE(MAX(code), ?) FROM action_label) + ROW_NUMBER() OVER (ORDER BY v), v
			FROM (SELECT DISTINCT `+col+` AS v FROM `+table+`
			      WHERE `+col+` IS NOT NULL AND `+col+` NOT IN `+sqlshared.FixedActionLabelsSQL()+`
			        AND `+col+` NOT IN (SELECT label FROM action_label))`,
			domain.FirstRegisteredActionCode-1); err != nil {
			return fmt.Errorf("registering the labels of %s: %w", col, err)
		}
		if _, err := d.db.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN `+col+`_code INTEGER`); err != nil {
			return err
		}
		set[i] = col + `_code = ` + sqlshared.ActionLabelCodesSQL(col)
	}
	update := `UPDATE ` + table + ` SET ` + strings.Join(set, ", ") + ` WHERE id >= ? AND id < ?`
	if err := d.batchByID(ctx, table, update, table+"_action_codes"); err != nil {
		return err
	}
	for _, col := range todo {
		for _, stmt := range []string{
			`ALTER TABLE ` + table + ` DROP COLUMN ` + col,
			`ALTER TABLE ` + table + ` RENAME COLUMN ` + col + `_code TO ` + col,
		} {
			if _, err := d.db.ExecContext(ctx, stmt); err != nil {
				return err
			}
		}
	}
	return nil
}

// binaryStateSQL rewrites a compact-array state ("[0,1,-3,…]", 28 values)
// as the 28 signed bytes of engine.EncodeBoardState, in SQL so the step does
// not round-trip 15 M rows through Go. A legacy full-Position JSON state
// ('{') is left as it is: the decoder still reads it.
var binaryStateSQL = func() string {
	var format, args strings.Builder
	for i := range engine.BoardStateLen {
		format.WriteString("%02X")
		fmt.Fprintf(&args, ", (CAST(json_extract(state, '$[%d]') AS INTEGER) + 256) %% 256", i)
	}
	return `UPDATE position SET state = unhex(printf('` + format.String() + `'` + args.String() + `))
		WHERE id >= ? AND id < ? AND typeof(state) = 'text' AND substr(state, 1, 1) = '['
		  AND json_array_length(state) = ` + fmt.Sprint(engine.BoardStateLen)
}()

// unixDateBatch is how many ids one conversion transaction covers.
const unixDateBatch = 100000

// unixDateColumn rewrites the text values of table.column as Unix seconds, in
// id ranges so that the progress callback moves on a 15 M-row table. A
// column the table does not have yet (a library older than 2.30.0, whose
// columns EnsureSchema adds after the chain) has nothing to convert.
func (d *Database) unixDateColumn(ctx context.Context, table, column, phase string) error {
	var present int
	if err := d.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, table, column).Scan(&present); err != nil {
		return err
	}
	if present == 0 {
		return nil
	}
	update := `UPDATE ` + table + ` SET ` + column + ` = ` + sqlite.UnixFromMatchDateSQL(column) +
		` WHERE id >= ? AND id < ? AND typeof(` + column + `) = 'text'`
	return d.batchByID(ctx, table, update, phase)
}

// batchByID runs update — which takes an id range [?, ?) — over the whole of
// table, one transaction per unixDateBatch ids, reporting progress as phase.
func (d *Database) batchByID(ctx context.Context, table, update, phase string) error {
	var maxID int64
	if err := d.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM `+table).Scan(&maxID); err != nil {
		return err
	}
	for next := int64(0); next <= maxID; next += unixDateBatch {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := next + unixDateBatch
		if err := d.inTx(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, update, next, end)
			return err
		}); err != nil {
			return err
		}
		done := end
		if done > maxID {
			done = maxID
		}
		d.emitMigrationProgress(phase, int(done), int(maxID))
	}
	return nil
}
