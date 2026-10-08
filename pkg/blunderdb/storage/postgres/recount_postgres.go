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

// recountPagesPerBatch is how many pages of analyses one transaction of
// recountDecisions recomputes: it bounds how long the ACCESS EXCLUSIVE locks
// of the unforced transaction are held.
const recountPagesPerBatch = 20

// recountDecisions recomputes the analysis columns 042 changed the rules of
// (is_forced, is_close_cube, best_move_equity_error) and the per-move columns
// 044 added, for every tenant, and drops the match_stats rows (cells and
// positions cascade) that summarise what changed. It also drops them all in
// its first batch: the DELETE of 042 and 043 ran on a connection carrying no
// tenant, which FORCEd row-level security shows no row. One of
// runGoBackfills' passes, so it runs once per library.
//
// One transaction per batch, FORCE lifted inside it, as the provenance pass:
// a batch written is a batch kept, and the next start resumes past it by
// recomputing from the beginning (an up-to-date row is not rewritten).
func recountDecisions(ctx context.Context, conn beginner) (bool, error) {
	var last int64
	total := 0
	complete := true
	for first := true; ; first = false {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		after, n := last, 0
		all, err := inUnforcedTx(ctx, conn, recountTables, func(tx pgx.Tx) error {
			if first {
				if _, err := tx.Exec(ctx, `DELETE FROM match_stats`); err != nil {
					return fmt.Errorf("postgres: decision recount: invalidate match stats: %w", err)
				}
			}
			var err error
			n, last, err = repairDenormalisedPages(ctx, tx, nil, after, recountPagesPerBatch)
			if err != nil {
				return fmt.Errorf("postgres: decision recount: %w", err)
			}
			if n > 0 && !first {
				if _, err := tx.Exec(ctx, `DELETE FROM match_stats`); err != nil {
					return fmt.Errorf("postgres: decision recount: invalidate match stats: %w", err)
				}
			}
			return nil
		})
		if err != nil {
			return false, err
		}
		complete = complete && all
		total += n
		if last == after {
			break
		}
		slog.Info("recounting the decisions of the stored analyses and moves", "through_analysis", last, "rows", total)
	}
	if total > 0 {
		slog.Info("recounted the decisions of the stored analyses and moves", "rows", total)
	}
	return complete, nil
}
