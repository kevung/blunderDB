package postgres

import (
	"context"
	"fmt"
	"iter"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// execer is the query surface shared by *pgxpool.Pool (autocommit) and pgx.Tx
// (transactional), so every store method is written once and works in either
// mode.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// binder provides the 14 per-family accessors over an execer. Storage embeds
// it bound to a *pgxpool.Pool; txImpl embeds it bound to a pgx.Tx.
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

// notImpl reports a family/method that a later P3 PR will implement.
func notImpl(family, method string) error {
	return fmt.Errorf("postgres: %s.%s not implemented: %w", family, method, storage.ErrInternal)
}

// withTx runs fn inside a transaction started from db. The pgx.Tx is passed to
// fn as an execer; when db is already a transaction the pgx.Tx is a
// savepoint-backed nested transaction, so fn is atomic in either binding.
func withTx(ctx context.Context, db execer, fn func(execer) error) error {
	b, ok := db.(txBeginner)
	if !ok {
		return fmt.Errorf("postgres: execer cannot begin a transaction: %w", storage.ErrInternal)
	}
	tx, err := b.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit: %w", err)
	}
	return nil
}

// errSeq2 returns an iterator that yields a single (nil, err) pair.
func errSeq2[T any](err error) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) { yield(nil, err) }
}
