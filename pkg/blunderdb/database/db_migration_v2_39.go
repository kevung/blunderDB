package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// playedDecisionsPendingKey is the metadata key that says the moves still
// have to be scored. It is written before the version is stamped and deleted
// once the pass has run to the end, so an open interrupted in between (the
// pass commits chunk by chunk) is resumed by the next one.
const playedDecisionsPendingKey = "played_decisions_pending"

// rescorePlayedDecisions is the pass itself, a variable so a test can
// interrupt it.
var rescorePlayedDecisions = sqlite.RescorePlayedDecisions

// migrate_2_38_0_to_2_39_0 gives every move the error of its own decision:
// move.decision_error_mp and move.is_close_cube, the analysis's projections
// scored against the play this match made rather than the first one the
// position saw (sqlshared/played_decisions.go), which the statistics now
// read. The step adds the columns and raises the pending key; the scoring
// itself runs in runMigrationChain once EnsureSchema has created everything
// it reads, which a library this old may still lack.
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
	_, err = d.db.ExecContext(ctx, `INSERT OR REPLACE INTO metadata (key, value) VALUES (?, '1')`, playedDecisionsPendingKey)
	return err
}

// finishPlayedDecisions scores the moves when the pending key is raised and
// clears it after the pass succeeded.
func (d *Database) finishPlayedDecisions(ctx context.Context) error {
	var v string
	err := d.db.QueryRowContext(ctx, `SELECT value FROM metadata WHERE key = ?`, playedDecisionsPendingKey).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := d.recountPlayedDecisions(ctx); err != nil {
		return err
	}
	_, err = d.db.ExecContext(ctx, `DELETE FROM metadata WHERE key = ?`, playedDecisionsPendingKey)
	return err
}

// recountPlayedDecisions scores every move of the library by its position's
// analysis. It drops the match_stats rows of what changed; FillMatchStats
// recomputes them.
func (d *Database) recountPlayedDecisions(ctx context.Context) error {
	n, err := rescorePlayedDecisions(ctx, d.db)
	if err != nil {
		return fmt.Errorf("scoring the stored moves: %w", err)
	}
	slog.Info("scored the decisions of the stored moves", "moves", n)
	return nil
}
