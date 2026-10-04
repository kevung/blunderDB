package database

// The 2.30.0 → 2.31.0 step as it was first written: one SQL pass per column,
// action columns rebuilt by ADD/DROP/RENAME. It is the oracle the current
// step is held to (TestMigrate_2_31_MatchesReference): same content, cell
// for cell.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// referenceMigrate_2_30_0_to_2_31_0 is the weight wave of the large-library plan
// (ADR-0071), plus additions EnsureSchema creates from the one schema after
// the chain: match_equity_table and analysis.met_id (ADR-0068),
// lesson_progress (ADR-0069) and move.error_mp, all added NULL. Scoring the
// existing moves is not part of it: that is the resumable
// MatchStore.ScoreMoves pass, run outside the open.
//
// The step rewrites the two date columns 2.30.0 stored as text into Unix
// seconds. A position whose date is still NULL here (the 2.30.0 backfill not
// finished) is dated by that backfill after the chain, already as an integer.
func (d *Database) referenceMigrate_2_30_0_to_2_31_0(ctx context.Context) error {
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
		if err := d.referenceUnixDateColumn(ctx, c.table, c.column, c.phase); err != nil {
			return fmt.Errorf("converting %s.%s to Unix seconds: %w", c.table, c.column, err)
		}
	}
	if err := d.referenceBatchByID(ctx, "position", referenceBinaryStateSQL, "position_state_binary"); err != nil {
		return fmt.Errorf("converting position.state to binary: %w", err)
	}
	if _, err := d.db.ExecContext(ctx, referenceActionLabelDDL); err != nil {
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
		if err := d.referenceActionCodeColumns(ctx, a.table, a.columns); err != nil {
			return fmt.Errorf("converting %s action labels to codes: %w", a.table, err)
		}
	}
	return nil
}

// referenceActionLabelDDL is the action_label table of the 2.31.0 schema, which the
// step fills before EnsureSchema runs.
const referenceActionLabelDDL = `CREATE TABLE IF NOT EXISTS action_label (
	code  INTEGER PRIMARY KEY,
	label TEXT NOT NULL UNIQUE
)`

// referenceActionCodeColumns rewrites the text action labels of table.columns as
// action codes (domain.ActionCode). The labels the fixed list lacks are
// registered first; each column is then rebuilt as an INTEGER column — a text
// column would store the codes back as text — filled in id ranges, the text
// one dropped and the new one renamed. A column already INTEGER, or absent,
// is left alone.
func (d *Database) referenceActionCodeColumns(ctx context.Context, table string, columns []string) error {
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
	if err := d.referenceBatchByID(ctx, table, update, table+"_action_codes"); err != nil {
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

// referenceBinaryStateSQL rewrites a compact-array state ("[0,1,-3,…]", 28 values)
// as the 28 signed bytes of engine.EncodeBoardState, in SQL so the step does
// not round-trip 15 M rows through Go. A legacy full-Position JSON state
// ('{') is left as it is: the decoder still reads it.
var referenceBinaryStateSQL = func() string {
	var format, args strings.Builder
	for i := range engine.BoardStateLen {
		format.WriteString("%02X")
		fmt.Fprintf(&args, ", (CAST(json_extract(state, '$[%d]') AS INTEGER) + 256) %% 256", i)
	}
	return `UPDATE position SET state = unhex(printf('` + format.String() + `'` + args.String() + `))
		WHERE id >= ? AND id < ? AND typeof(state) = 'text' AND substr(state, 1, 1) = '['
		  AND json_array_length(state) = ` + fmt.Sprint(engine.BoardStateLen)
}()

// referenceUnixDateBatch is how many ids one conversion transaction covers.
const referenceUnixDateBatch = 100000

// referenceUnixDateColumn rewrites the text values of table.column as Unix seconds, in
// id ranges so that the progress callback moves on a 15 M-row table. A
// column the table does not have yet (a library older than 2.30.0, whose
// columns EnsureSchema adds after the chain) has nothing to convert.
func (d *Database) referenceUnixDateColumn(ctx context.Context, table, column, phase string) error {
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
	return d.referenceBatchByID(ctx, table, update, phase)
}

// referenceBatchByID runs update — which takes an id range [?, ?) — over the whole of
// table, one transaction per referenceUnixDateBatch ids, reporting progress as phase.
func (d *Database) referenceBatchByID(ctx context.Context, table, update, phase string) error {
	var maxID int64
	if err := d.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM `+table).Scan(&maxID); err != nil {
		return err
	}
	for next := int64(0); next <= maxID; next += referenceUnixDateBatch {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := next + referenceUnixDateBatch
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

// referenceBackfillAnalysisProvenance derives analysis_engine, analysis_depth and
// creation_date (engine.AnalysisProvenance) for every analysis row whose
// analysis_engine is NULL: every row on the open crossing 2.30.0, and on
// later opens the few a blob-only writer left. The probe is one step of
// idx_analysis_provenance_pending, so an open with nothing to do pays nothing. Resumable
// by construction: a written row is no longer NULL.
func (d *Database) referenceBackfillAnalysisProvenance(ctx context.Context) error {
	var probe int
	err := d.db.QueryRowContext(ctx, `SELECT 1 FROM analysis WHERE analysis_engine IS NULL LIMIT 1`).Scan(&probe)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	var total int
	if err := d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM analysis WHERE analysis_engine IS NULL`).Scan(&total); err != nil {
		return err
	}
	// The provenance written here is a column match_stats summarises; on the
	// crossing itself the table is still empty and nothing needs dropping.
	var statsProbe int
	haveStats := d.db.QueryRowContext(ctx, `SELECT 1 FROM match_stats LIMIT 1`).Scan(&statsProbe) == nil
	done := 0
	var last int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		raw, ids, err := d.nullProvenanceBatch(ctx, last)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}
		decoded, failed := engine.DecodeAnalysesConcurrently(raw)
		err = d.inTx(ctx, func(tx *sql.Tx) error {
			stmt, err := tx.PrepareContext(ctx,
				`UPDATE analysis SET analysis_engine = ?, analysis_depth = ?, creation_date = ? WHERE id = ?`)
			if err != nil {
				return err
			}
			defer stmt.Close()
			for _, id := range ids {
				// An undecodable blob is written as "no entry" so that it is not
				// retried on every open; `blunderdb verify` still reports it.
				eng, depth, created := "", int64(-1), int64(0)
				if a := decoded[id]; a != nil {
					eng, depth, created = engine.AnalysisProvenance(a)
				} else if err := failed[id]; err != nil {
					slog.Warn("analysis provenance: undecodable blob", "analysis_id", id, "error", err)
				}
				var createdVal any
				if created != 0 {
					createdVal = created
				}
				if _, err := stmt.ExecContext(ctx, eng, depth, createdVal, id); err != nil {
					return err
				}
			}
			if haveStats {
				args := make([]any, len(ids))
				for i, id := range ids {
					args[i] = id
				}
				if _, err := tx.ExecContext(ctx, `DELETE FROM match_stats WHERE match_id IN
					(SELECT g.match_id FROM analysis a JOIN move mv ON mv.position_id = a.position_id
					   JOIN game g ON g.id = mv.game_id WHERE a.id IN (`+sqlshared.Placeholders(len(ids))+`))`, args...); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		last = ids[len(ids)-1]
		done += len(ids)
		d.emitMigrationProgress("analysis_provenance", done, total)
	}
	if done > 0 {
		slog.Info("derived the provenance of the stored analyses", "analyses", done)
	}
	return nil
}
