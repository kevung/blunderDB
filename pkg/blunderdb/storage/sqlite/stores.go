package sqlite

import (
	"context"
	"database/sql"
	"strings"

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
