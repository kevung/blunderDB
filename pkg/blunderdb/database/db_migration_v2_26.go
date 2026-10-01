package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// migrate_2_25_0_to_2_26_0 brings every game.winner to the one encoding the
// library keeps (domain.WinnerPlayer1). The column stays; only its values
// change, by the statement PostgreSQL's 028 runs too (sqlshared.NormalizeGameWinnerSQL
// documents the rule per source). It runs in the step, not after EnsureSchema
// like a deferred backfill, so that it commits with its version stamp.
func (d *Database) migrate_2_25_0_to_2_26_0(ctx context.Context) error {
	// A library from before matches holds no game to convert.
	if ok, err := d.columnExists("game", "winner"); err != nil || !ok {
		return err
	}
	stmt := sqlshared.NormalizeGameWinnerSQL
	// A library from before import batches has none to read: the empty join
	// leaves the file name as the only record of the source, as it was then.
	if ok, err := d.columnExists("match", "import_batch_id"); err != nil {
		return err
	} else if batches, err := d.columnExists("import_batch", "format"); err != nil {
		return err
	} else if !ok || !batches {
		stmt = strings.Replace(stmt, sqlshared.WinnerBatchJoin,
			"LEFT JOIN (SELECT NULL AS id, NULL AS format) b ON 1 = 0", 1)
	}
	// The conversion is not idempotent (a normalized 1 reads as gnubg's
	// player 2), so it commits with the version stamp or not at all: an
	// interruption between the two would convert the games again on the next
	// open. The chain's own stamp after the step then rewrites the same value.
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("normalize game.winner: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE metadata SET value = '2.26.0' WHERE key = 'database_version'`); err != nil {
		return fmt.Errorf("stamp 2.26.0: %w", err)
	}
	return tx.Commit()
}
