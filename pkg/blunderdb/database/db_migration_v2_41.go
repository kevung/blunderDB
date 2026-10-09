package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// answeredDoublesPendingKey is the metadata key that says the transcribed
// answers to a double still have to be moved. It is written before the
// version is stamped and deleted once the move has committed, so an open
// interrupted in between resumes it on the next one.
const answeredDoublesPendingKey = "answered_doubles_pending"

// migrate_2_40_0_to_2_41_0 moves every take or pass a transcribed or duelled
// match recorded on the answerer's own redouble position (the turned cube
// owned by the answerer) onto the response position, the turned cube held by
// no one, as importers record it, and empties gammonNet's stale verdicts on
// those positions (ADR-0083). The step only raises the pending key: the
// move itself runs in runMigrationChain once EnsureSchema has created every
// column the rescore of the moved moves reads.
func (d *Database) migrate_2_40_0_to_2_41_0(ctx context.Context) error {
	present, err := d.columnExists("move", "id")
	if err != nil || !present {
		return err
	}
	_, err = d.db.ExecContext(ctx, `INSERT OR REPLACE INTO metadata (key, value) VALUES (?, '1')`, answeredDoublesPendingKey)
	return err
}

// finishAnsweredDoubles, when the pending key is raised, drops gammonNet's
// verdicts on take/pass positions (scored as the answerer's centred-cube
// decision before gammonNet knew a response) so the next analysis fills them
// as the doubler's, then moves the answers, and clears the key after both
// committed. The drop runs first so a moved answer is rescored against what
// its new row will keep. Both are idempotent, so a resumed open repeats them
// safely. ReanchorAnsweredDoubles drops the
// match_stats rows of every match it touched, and the statistics of a match
// holding a take on the doubler's own row, once converted at half its cube,
// go too; FillMatchStats recomputes them.
func (d *Database) finishAnsweredDoubles(ctx context.Context) error {
	var v string
	err := d.db.QueryRowContext(ctx, `SELECT value FROM metadata WHERE key = ?`, answeredDoublesPendingKey).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	dropped, err := sqlite.DropGammonNetResponseAnalyses(ctx, d.db)
	if err != nil {
		return fmt.Errorf("dropping gammonNet's verdicts on take/pass positions: %w", err)
	}
	if dropped > 0 {
		slog.Info("dropped gammonNet's verdicts on take/pass positions for reanalysis", "analyses", dropped)
	}
	n, err := sqlite.New(d.db).Matches().ReanchorAnsweredDoubles(ctx, "")
	if err != nil {
		return fmt.Errorf("moving the transcribed answers to a double: %w", err)
	}
	if n > 0 {
		slog.Info("moved the transcribed answers to a double onto the ownerless cube", "moves", n)
	}
	if _, err := d.db.ExecContext(ctx, sqlshared.DropOwnedCubeAnswerMatchStatsSQL); err != nil {
		return fmt.Errorf("dropping the statistics of matches with a take on the doubler's row: %w", err)
	}
	_, err = d.db.ExecContext(ctx, `DELETE FROM metadata WHERE key = ?`, answeredDoublesPendingKey)
	return err
}
