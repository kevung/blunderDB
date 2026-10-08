package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// answeredDoublesPendingKey is the metadata key that says the transcribed
// answers to a double still have to be moved. It is written before the
// version is stamped and deleted once the move has committed, so an open
// interrupted in between resumes it on the next one.
const answeredDoublesPendingKey = "answered_doubles_pending"

// migrate_2_39_0_to_2_40_0 moves every take or pass a transcribed or duelled
// match recorded on the answerer's own redouble position (the turned cube
// owned by the answerer) onto the response position, the turned cube held by
// no one, as importers record it. The step only raises the pending key: the
// move itself runs in runMigrationChain once EnsureSchema has created every
// column the rescore of the moved moves reads.
func (d *Database) migrate_2_39_0_to_2_40_0(ctx context.Context) error {
	present, err := d.columnExists("move", "id")
	if err != nil || !present {
		return err
	}
	_, err = d.db.ExecContext(ctx, `INSERT OR REPLACE INTO metadata (key, value) VALUES (?, '1')`, answeredDoublesPendingKey)
	return err
}

// finishAnsweredDoubles moves the answers when the pending key is raised and
// clears it after the move committed. ReanchorAnsweredDoubles drops the
// match_stats rows of every match it touched; FillMatchStats recomputes them.
func (d *Database) finishAnsweredDoubles(ctx context.Context) error {
	var v string
	err := d.db.QueryRowContext(ctx, `SELECT value FROM metadata WHERE key = ?`, answeredDoublesPendingKey).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	n, err := sqlite.New(d.db).Matches().ReanchorAnsweredDoubles(ctx, "")
	if err != nil {
		return fmt.Errorf("moving the transcribed answers to a double: %w", err)
	}
	if n > 0 {
		slog.Info("moved the transcribed answers to a double onto the ownerless cube", "moves", n)
	}
	_, err = d.db.ExecContext(ctx, `DELETE FROM metadata WHERE key = ?`, answeredDoublesPendingKey)
	return err
}
