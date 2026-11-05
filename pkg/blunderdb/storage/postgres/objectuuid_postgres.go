package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// backfillObjectUUIDs names every object held before 054 with the identifier
// derived from its natural key (sqlshared.DeriveObjectUUID, computed in SQL by
// blunderdb_derive_uuid). Once per family (ObjectUUIDFamily.MarkOf), written
// only when the pass saw every tenant's rows. One transaction per batch with
// FORCE lifted inside it, so the lock lasts one batch; resumable by
// construction, a written row no longer being NULL; a table busy past
// unforcedLockTimeout is left for the next Migrate.
func backfillObjectUUIDs(ctx context.Context, conn migrationConn) error {
	// A family whose pass is recorded is skipped; the others are named now.
	var owed []sqlshared.ObjectUUIDFamily
	for _, f := range sqlshared.ObjectUUIDFamilies {
		var done string
		err := conn.QueryRow(ctx, `SELECT value FROM metadata WHERE key = $1`, f.MarkOf()).Scan(&done)
		if err == nil {
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("postgres: read %s: %w", f.MarkOf(), err)
		}
		owed = append(owed, f)
	}
	if len(owed) == 0 {
		return nil
	}
	complete, total := true, int64(0)
	for _, f := range owed {
		var after int64
		for {
			if err := ctx.Err(); err != nil {
				return err
			}
			var n int64
			var more bool
			all, err := inUnforcedTx(ctx, conn, f.Reads, func(tx pgx.Tx) error {
				if f.Ranked {
					tag, err := tx.Exec(ctx, f.BackfillSQL(true, 0, 0))
					n = tag.RowsAffected()
					return err
				}
				var upTo *int64
				if err := tx.QueryRow(ctx, f.NextBoundSQL(after, sqlshared.ObjectUUIDBatch)).Scan(&upTo); err != nil {
					return err
				}
				if upTo == nil {
					return nil
				}
				tag, err := tx.Exec(ctx, f.BackfillSQL(true, after, *upTo))
				n, after, more = tag.RowsAffected(), *upTo, true
				return err
			})
			if errors.Is(err, errUnforcedLockTimeout) {
				slog.Warn("postgres: a table stayed locked; the uuid backfill resumes at the next start", "table", f.Table)
				return nil
			}
			if err != nil {
				return fmt.Errorf("postgres: derive the uuids of %s: %w", f.Table, err)
			}
			complete = complete && all
			total += n
			if !more {
				break
			}
		}
		// What has no natural key draws its identifier, as a new row does.
		var n int64
		all, err := inUnforcedTx(ctx, conn, []string{f.Table}, func(tx pgx.Tx) error {
			tag, err := tx.Exec(ctx, `UPDATE `+f.Table+` SET uuid = blunderdb_uuid7() WHERE uuid IS NULL`)
			n = tag.RowsAffected()
			return err
		})
		if errors.Is(err, errUnforcedLockTimeout) {
			slog.Warn("postgres: a table stayed locked; the uuid backfill resumes at the next start", "table", f.Table)
			return nil
		}
		if err != nil {
			return fmt.Errorf("postgres: draw the uuids of %s: %w", f.Table, err)
		}
		complete = complete && all
		total += n
	}
	if !complete {
		return nil
	}
	for _, f := range owed {
		if _, err := conn.Exec(ctx,
			`INSERT INTO metadata (key, value) VALUES ($1, '1') ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
			f.MarkOf()); err != nil {
			return fmt.Errorf("postgres: write %s: %w", f.MarkOf(), err)
		}
	}
	if total > 0 {
		slog.Info("gave the stored objects their uuid", "rows", total)
	}
	return nil
}
