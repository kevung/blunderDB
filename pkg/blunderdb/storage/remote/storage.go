package remote

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Storage is the remote backend. It holds no connection of its own: every
// method is one HTTP request through Options.HTTPClient.
type Storage struct {
	c *client
}

var _ storage.Storage = (*Storage)(nil)

// Open returns a Storage over the daemon at o.BaseURL. It does not contact
// the daemon; the first method call does.
func Open(o Options) (*Storage, error) {
	c, err := newClient(o)
	if err != nil {
		return nil, err
	}
	return &Storage{c: c}, nil
}

// One value type per family, each holding the client; the methods with a
// /v1 route are written by hand beside them, the rest in unsupported_gen.go.
type (
	positionStore         struct{ c *client }
	analysisStore         struct{ c *client }
	matchStore            struct{ c *client }
	commentStore          struct{ c *client }
	collectionStore       struct{ c *client }
	tournamentStore       struct{ c *client }
	rencontreStore        struct{ c *client }
	lessonStore           struct{ c *client }
	matchEquityTableStore struct{ c *client }
	directionStore        struct{ c *client }
	ankiStore             struct{ c *client }
	filterStore           struct{ c *client }
	sessionStore          struct{ c *client }
	searchStore           struct{ c *client }
	searchHistoryStore    struct{ c *client }
	statsStore            struct{ c *client }
	commandHistoryStore   struct{ c *client }
	metadataStore         struct{ c *client }
	importBatchStore      struct{ c *client }
	trashStore            struct{ c *client }
	librarySettingsStore  struct{ c *client }
	transcriptionStore    struct{ c *client }
	duelStore             struct{ c *client }
	trainingStore         struct{ c *client }
	aliasStore            struct{ c *client }
)

func (s *Storage) Positions() storage.PositionStore { return positionStore{s.c} }
func (s *Storage) Analyses() storage.AnalysisStore  { return analysisStore{s.c} }
func (s *Storage) Matches() storage.MatchStore      { return matchStore{s.c} }
func (s *Storage) Comments() storage.CommentStore   { return commentStore{s.c} }
func (s *Storage) Collections() storage.CollectionStore {
	return collectionStore{s.c}
}
func (s *Storage) Tournaments() storage.TournamentStore { return tournamentStore{s.c} }
func (s *Storage) Rencontres() storage.RencontreStore   { return rencontreStore{s.c} }
func (s *Storage) Lessons() storage.LessonStore         { return lessonStore{s.c} }
func (s *Storage) MatchEquityTables() storage.MatchEquityTableStore {
	return matchEquityTableStore{s.c}
}
func (s *Storage) Directions() storage.DirectionStore        { return directionStore{s.c} }
func (s *Storage) Anki() storage.AnkiStore                   { return ankiStore{s.c} }
func (s *Storage) Filters() storage.FilterStore              { return filterStore{s.c} }
func (s *Storage) Session() storage.SessionStore             { return sessionStore{s.c} }
func (s *Storage) Search() storage.SearchStore               { return searchStore{s.c} }
func (s *Storage) SearchHistory() storage.SearchHistoryStore { return searchHistoryStore{s.c} }
func (s *Storage) Stats() storage.StatsStore                 { return statsStore{s.c} }
func (s *Storage) History() storage.CommandHistoryStore      { return commandHistoryStore{s.c} }
func (s *Storage) Metadata() storage.MetadataStore           { return metadataStore{s.c} }
func (s *Storage) ImportBatches() storage.ImportBatchStore   { return importBatchStore{s.c} }
func (s *Storage) Trash() storage.TrashStore                 { return trashStore{s.c} }
func (s *Storage) LibrarySettings() storage.LibrarySettingsStore {
	return librarySettingsStore{s.c}
}
func (s *Storage) Transcriptions() storage.TranscriptionStore { return transcriptionStore{s.c} }
func (s *Storage) Duels() storage.DuelStore                   { return duelStore{s.c} }
func (s *Storage) Training() storage.TrainingStore            { return trainingStore{s.c} }
func (s *Storage) Aliases() storage.AliasStore                { return aliasStore{s.c} }

// BeginTx is refused: /v1 has no transaction that spans calls, and a Tx that
// applied each call at once would make Rollback a lie.
func (s *Storage) BeginTx(context.Context) (storage.Tx, error) {
	return nil, unsupported("Storage.BeginTx")
}

// Close releases nothing: the backend holds no connection beyond the HTTP
// client's pool, which belongs to the caller.
func (s *Storage) Close() error { return nil }

type versionResp struct {
	Version string `json:"version"`
}

// Version reports the schema version of the daemon's database.
func (s *Storage) Version(ctx context.Context) (string, error) {
	var r versionResp
	err := s.c.call(ctx, "", "metadata.version", nil, &r)
	return r.Version, err
}

// Migrate cannot migrate: the daemon migrates its own database when it
// starts. It checks the schema is the one this binary speaks and refuses
// otherwise, so a caller never goes on against a schema it does not know.
func (s *Storage) Migrate(ctx context.Context) error {
	v, err := s.Version(ctx)
	if err != nil {
		return err
	}
	if v != domain.DatabaseVersion {
		return fmt.Errorf("%w: remote: the daemon's schema is %s, this client speaks %s", storage.ErrConflict, v, domain.DatabaseVersion)
	}
	return nil
}
