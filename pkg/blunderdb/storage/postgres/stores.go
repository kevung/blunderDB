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

// errSeq2 returns an iterator that yields a single (nil, err) pair.
func errSeq2[T any](err error) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) { yield(nil, err) }
}
