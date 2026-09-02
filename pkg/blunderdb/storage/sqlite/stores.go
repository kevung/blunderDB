package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	sqlitedriver "modernc.org/sqlite"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// execer is satisfied by both *sql.DB (autocommit) and *sql.Tx
// (transactional), so every store method is written once and works in either
// mode.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// binder provides the 14 per-family accessors over an execer. Storage embeds
// it bound to a *sql.DB; txImpl embeds it bound to a *sql.Tx.
type binder struct {
	db execer
}

func (b binder) Positions() storage.PositionStore     { return &positionStore{b.db} }
func (b binder) Analyses() storage.AnalysisStore      { return &analysisStore{b.db} }
func (b binder) Matches() storage.MatchStore          { return &matchStore{b.db} }
func (b binder) Comments() storage.CommentStore       { return &sqlshared.CommentStore{DB: b.shared()} }
func (b binder) Collections() storage.CollectionStore { return &collectionStore{b.db} }
func (b binder) Tournaments() storage.TournamentStore { return &tournamentStore{b.db} }
func (b binder) Anki() storage.AnkiStore              { return &ankiStore{b.db} }
func (b binder) Filters() storage.FilterStore         { return &sqlshared.FilterStore{DB: b.shared()} }
func (b binder) Session() storage.SessionStore        { return &sqlshared.SessionStore{DB: b.shared()} }
func (b binder) Search() storage.SearchStore          { return &sqlshared.SearchStore{DB: b.shared()} }
func (b binder) SearchHistory() storage.SearchHistoryStore {
	return &sqlshared.SearchHistoryStore{DB: b.shared()}
}
func (b binder) Stats() storage.StatsStore { return &statsStore{&sqlshared.StatsStore{DB: b.shared()}} }
func (b binder) History() storage.CommandHistoryStore {
	return &sqlshared.CommandHistoryStore{DB: b.shared()}
}
func (b binder) Metadata() storage.MetadataStore { return &sqlshared.MetadataStore{DB: b.shared()} }

// withTx runs fn atomically over db. When db is a *sql.DB it opens a
// transaction and commits (or rolls back) around fn; when db is already a
// *sql.Tx — the store is reached through a caller's transaction — fn runs
// directly and that outer transaction provides atomicity.
func withTx(ctx context.Context, db execer, fn func(execer) error) error {
	sqlDB, ok := db.(*sql.DB)
	if !ok {
		return fn(db)
	}
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// queryInt64s runs a single-column query and returns every value, with the
// cursor closed before it returns. Most stores read a list of ids and then
// write on the same connection: the cursor has to be gone by then, and a
// defer in the caller would keep it open across those writes.
func queryInt64s(ctx context.Context, db execer, query string, args ...any) ([]int64, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// sqliteBusyCode is SQLite's SQLITE_BUSY result code (the writer lock could
// not be acquired within busy_timeout). https://www.sqlite.org/rescode.html#busy
const sqliteBusyCode = 5

// isBusyErr reports whether err is (or wraps) a SQLITE_BUSY error from the
// modernc.org/sqlite driver.
func isBusyErr(err error) bool {
	var sqliteErr *sqlitedriver.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == sqliteBusyCode
}

// busyRetryAttempts bounds retryOnBusy: enough to ride out a burst of
// contending writers without turning a real deadlock or a stuck lock into a
// long hang.
const busyRetryAttempts = 5

// busyBackoff is a short, bounded exponential backoff between busy retries:
// 10ms, 20ms, 40ms, 80ms, capped at 160ms.
func busyBackoff(attempt int) time.Duration {
	d := 10 * time.Millisecond << attempt
	if d > 160*time.Millisecond {
		d = 160 * time.Millisecond
	}
	return d
}

// retryOnBusy runs fn, retrying up to busyRetryAttempts times with
// busyBackoff between attempts whenever fn fails with SQLITE_BUSY. Even at a
// 10s busy_timeout (perConnPragmas) a pooled connection can occasionally
// still lose the race for the write lock under a heavy burst of concurrent
// writers — more often on Windows, where file locking is measurably slower
// than POSIX advisory locks. fn must be safe to call more than once: it is
// not wrapped in a transaction retryOnBusy could roll back, so every
// statement it runs needs to already be idempotent on its own.
func retryOnBusy(fn func() error) error {
	var err error
	for attempt := 0; attempt < busyRetryAttempts; attempt++ {
		err = fn()
		if err == nil || !isBusyErr(err) {
			return err
		}
		time.Sleep(busyBackoff(attempt))
	}
	return err
}

// forEachIn runs `prefix (?,?,…) suffix` over ids in chunks that stay under
// SQLite's parameter limit (32766 by default — a real selection runs to tens
// of thousands of positions) and hands every row to scan. The chunk size
// leaves room for the arguments a caller's prefix may carry.
func forEachIn(ctx context.Context, db execer, ids []int64, prefix, suffix string, scan func(*sql.Rows) error) error {
	const chunk = 900
	for start := 0; start < len(ids); start += chunk {
		batch := ids[start:min(start+chunk, len(ids))]
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		args := make([]any, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		if err := func() error {
			rows, err := db.QueryContext(ctx, prefix+`(`+placeholders+`)`+suffix, args...)
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				if err := scan(rows); err != nil {
					return err
				}
			}
			return rows.Err()
		}(); err != nil {
			return err
		}
	}
	return nil
}
