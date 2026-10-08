package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

// recountMarker is the metadata row 044 writes to ask for recountPlayedDecisions;
// metadata carries no row-level security, so the probe sees it whatever the
// role.
const recountMarker = "recount_played_decisions"

// recountTables are the tables the recount reads or writes, every tenant's
// rows of which it must see.
var recountTables = []string{
	"analysis", "position", "move", "game", "action_label",
	"match_stats", "match_stats_cell", "match_stats_position",
}

// recountLockTimeout bounds the wait for the ACCESS EXCLUSIVE lock lifting
// FORCE takes: a Migrate behind a long reader fails and is retried rather
// than queueing every other session behind it.
const recountLockTimeout = "30s"

// recountPlayedDecisions recomputes, once, the analysis columns 042 changed
// the rules of (is_forced, is_close_cube, best_move_equity_error) and the
// per-move columns 044 added, for every tenant, then drops every match_stats
// row (cells and positions cascade): the DELETE of 042 and 043 ran on a
// connection carrying no tenant, which FORCEd row-level security shows no
// row. It runs only while 044's marker row exists, and deletes it in the same
// transaction as its writes: an interrupted pass leaves the marker and is run
// again whole, a finished one never runs again.
func recountPlayedDecisions(ctx context.Context, conn beginner, db execer) error {
	var pending bool
	if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM metadata WHERE key = $1)`, recountMarker).Scan(&pending); err != nil {
		return fmt.Errorf("postgres: probe decision recount: %w", err)
	}
	if !pending {
		return nil
	}
	return inUnforcedTx(ctx, conn, recountTables, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SET LOCAL lock_timeout = '`+recountLockTimeout+`'`); err != nil {
			return fmt.Errorf("postgres: decision recount: %w", err)
		}
		n, err := repairDenormalisedColumns(ctx, tx, nil)
		if err != nil {
			return fmt.Errorf("postgres: decision recount: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM match_stats`); err != nil {
			return fmt.Errorf("postgres: decision recount: invalidate match stats: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM metadata WHERE key = $1`, recountMarker); err != nil {
			return fmt.Errorf("postgres: decision recount: clear marker: %w", err)
		}
		slog.Info("recounted the decisions of the stored analyses and moves", "rows", n)
		return nil
	})
}
