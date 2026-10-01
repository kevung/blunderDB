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
// documents the rule per source). It runs in the step, not after
// EnsureSchema like a deferred backfill: the version is stamped as soon as the
// step returns, and a conversion left for later would never be retried.
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
	if _, err := d.db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("normalize game.winner: %w", err)
	}
	return nil
}
