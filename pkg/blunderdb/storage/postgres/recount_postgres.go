package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

// recountTables are the tables recountDecisions reads or writes, every
// tenant's rows of which it must see.
var recountTables = []string{
	"analysis", "position", "move", "game", "action_label",
	"match_stats", "match_stats_cell", "match_stats_position",
}

// recountDecisions recomputes the analysis columns 042 changed the rules of
// (is_forced, is_close_cube, best_move_equity_error) and the per-move columns
// 044 added, for every tenant, then drops every match_stats row (cells and
// positions cascade): the DELETE of 042 and 043 ran on a connection carrying
// no tenant, which FORCEd row-level security shows no row. One of
// runGoBackfills' passes, so it runs once per library.
func recountDecisions(ctx context.Context, conn beginner) (bool, error) {
	return inUnforcedTx(ctx, conn, recountTables, func(tx pgx.Tx) error {
		n, err := repairDenormalisedColumns(ctx, tx, nil)
		if err != nil {
			return fmt.Errorf("postgres: decision recount: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM match_stats`); err != nil {
			return fmt.Errorf("postgres: decision recount: invalidate match stats: %w", err)
		}
		if n > 0 {
			slog.Info("recounted the decisions of the stored analyses and moves", "rows", n)
		}
		return nil
	})
}
