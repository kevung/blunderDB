package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// migrate_2_36_0_to_2_37_0 recomputes the counted-decision columns of every
// analysis under the rules XG counts a Performance Rating by: is_forced from
// the position's legal plays (engine.IsForcedChecker), is_close_cube from the
// no-double's distance to min(double/take, double/pass)
// (engine.ComputeIsCloseCube), and best_move_equity_error NULL for a played
// move no candidate names. No column changes shape; what they mean does, and
// rows written under the old rules would keep counting the old way. The other
// projected columns are left as they are: their rules did not change.
//
// A library too old to carry the columns the legal plays are read from
// reaches this step before EnsureSchema adds them; its rows keep the values
// the earlier steps derived, and `blunderdb repair` recomputes them.
func (d *Database) migrate_2_36_0_to_2_37_0(ctx context.Context) error {
	for _, c := range [][2]string{{"analysis", "is_close_cube"}, {"analysis", "is_forced"}, {"position", "player_on_roll"}, {"position", "state"}, {"move", "cube_action"}} {
		present, err := d.columnExists(c[0], c[1])
		if err != nil || !present {
			return err
		}
	}
	type row struct {
		id             int64
		data           []byte
		mvMove, mvCube sql.NullString
		state          []byte
		por, d1, d2    *int64
		forced, close  sql.NullInt64
		moveErr        sql.NullInt64
	}
	changed := 0
	var lastID int64
	for {
		page, err := func() ([]row, error) {
			var page []row
			rows, err := d.db.QueryContext(ctx,
				`SELECT a.id, a.data, a.is_forced, a.is_close_cube, a.best_move_equity_error,
			        (SELECT mv.checker_move FROM move mv WHERE mv.position_id = a.position_id AND COALESCE(mv.checker_move, '') <> '' ORDER BY mv.id LIMIT 1),
			        (SELECT `+sqlshared.ActionLabelSQL("mv.cube_action")+` FROM move mv WHERE mv.position_id = a.position_id AND `+sqlshared.ActionNotEmptySQL("mv.cube_action")+` ORDER BY mv.id LIMIT 1),
			        `+sqlshared.LegalPlaysColumns+`
			 FROM analysis a LEFT JOIN position p ON p.id = a.position_id
			 WHERE a.id > ? ORDER BY a.id LIMIT 500`, lastID)
			if err != nil {
				return nil, fmt.Errorf("recount decisions: read analyses: %w", err)
			}
			defer rows.Close()
			for rows.Next() {
				var r row
				if err := rows.Scan(&r.id, &r.data, &r.forced, &r.close, &r.moveErr, &r.mvMove, &r.mvCube, &r.state, &r.por, &r.d1, &r.d2); err != nil {
					return nil, fmt.Errorf("recount decisions: scan: %w", err)
				}
				page = append(page, r)
			}
			if err := rows.Err(); err != nil {
				return nil, fmt.Errorf("recount decisions: rows: %w", err)
			}
			return page, nil
		}()
		if err != nil {
			return err
		}
		if len(page) == 0 {
			break
		}
		lastID = page[len(page)-1].id
		for _, r := range page {
			a, err := engine.DecodeAnalysisFromStorage(r.data)
			if err != nil {
				continue // an unreadable blob keeps its columns: they are all that is left
			}
			playedMove, playedCube := engine.PlayedActionsFor(a.PlayedMoves, a.PlayedCubeActions,
				[]string{r.mvMove.String}, []string{r.mvCube.String})
			c := engine.PopulateAnalysisColumns(&a, playedMove, playedCube, sqlshared.LegalPlays(r.state, r.por, r.d1, r.d2))
			moveErr := r.moveErr
			if c.BestMoveUnscored {
				moveErr = sql.NullInt64{}
			}
			if c.IsForced == r.forced.Int64 && c.IsCloseCube == r.close.Int64 && moveErr == r.moveErr {
				continue
			}
			if _, err := d.db.ExecContext(ctx, `UPDATE analysis SET is_forced = ?, is_close_cube = ?, best_move_equity_error = ? WHERE id = ?`,
				c.IsForced, c.IsCloseCube, moveErr, r.id); err != nil {
				return fmt.Errorf("recount decisions: update %d: %w", r.id, err)
			}
			changed++
		}
	}
	if changed > 0 {
		present, err := d.columnExists("match_stats", "match_id")
		if err != nil {
			return err
		}
		if present {
			// The rows summarise the columns just rewritten.
			if _, err := d.db.ExecContext(ctx, `DELETE FROM match_stats`); err != nil {
				return fmt.Errorf("recount decisions: invalidate match stats: %w", err)
			}
		}
	}
	slog.Info("recounted the decisions of the stored analyses", "analyses", changed)
	return nil
}
