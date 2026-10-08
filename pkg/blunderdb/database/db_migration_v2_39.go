package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// migrate_2_38_0_to_2_39_0 gives every move the error of its own decision:
// move.decision_error_mp and move.is_close_cube, the analysis's projections
// scored against the play this match made rather than the first one the
// position saw (sqlshared/played_decisions.go), which the statistics now
// read. They are computed here for every stored move; the analysis columns
// are left as they are. A library too old to carry what the pass reads has
// it deferred until EnsureSchema has run (pendingPlayedDecisions).
func (d *Database) migrate_2_38_0_to_2_39_0(ctx context.Context) error {
	present, err := d.columnExists("move", "id")
	if err != nil || !present {
		return err
	}
	for _, col := range []string{"decision_error_mp INTEGER", "is_close_cube INTEGER NOT NULL DEFAULT 0"} {
		if err := d.addColumn("move", col); err != nil {
			return err
		}
	}
	for _, c := range [][2]string{{"analysis", "is_close_cube"}, {"analysis", "is_forced"}, {"position", "player_on_roll"},
		{"position", "state"}, {"position", "decision_type"}, {"move", "cube_action"}, {"match_stats", "match_id"}} {
		present, err := d.columnExists(c[0], c[1])
		if err != nil {
			return err
		}
		if !present {
			d.pendingPlayedDecisions = true
			return nil
		}
	}
	return d.recountPlayedDecisions(ctx)
}

// recountPlayedDecisions scores every move of the library by its position's
// analysis. It drops the match_stats rows of what changed; FillMatchStats
// recomputes them.
func (d *Database) recountPlayedDecisions(ctx context.Context) error {
	n, err := sqlite.RescorePlayedDecisions(ctx, d.db)
	if err != nil {
		return fmt.Errorf("scoring the stored moves: %w", err)
	}
	slog.Info("scored the decisions of the stored moves", "moves", n)
	return nil
}
