package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
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

func (b binder) Positions() storage.PositionStore          { return &positionStore{b.db} }
func (b binder) Analyses() storage.AnalysisStore           { return &analysisStore{b.db} }
func (b binder) Matches() storage.MatchStore               { return &matchStore{b.db} }
func (b binder) Comments() storage.CommentStore            { return &commentStore{b.db} }
func (b binder) Collections() storage.CollectionStore      { return &collectionStore{b.db} }
func (b binder) Tournaments() storage.TournamentStore      { return &tournamentStore{b.db} }
func (b binder) Anki() storage.AnkiStore                   { return &ankiStore{b.db} }
func (b binder) Filters() storage.FilterStore              { return &filterStore{b.db} }
func (b binder) Session() storage.SessionStore             { return &sessionStore{b.db} }
func (b binder) Search() storage.SearchStore               { return &searchStore{b.db} }
func (b binder) SearchHistory() storage.SearchHistoryStore { return &searchHistoryStore{b.db} }
func (b binder) Stats() storage.StatsStore                 { return &statsStore{b.db} }
func (b binder) History() storage.CommandHistoryStore      { return &commandHistoryStore{b.db} }
func (b binder) Metadata() storage.MetadataStore           { return &metadataStore{b.db} }

// notImpl reports a family/method that a later P2 PR will implement.
func notImpl(family, method string) error {
	return fmt.Errorf("sqlite: %s.%s not implemented: %w", family, method, storage.ErrInternal)
}

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
