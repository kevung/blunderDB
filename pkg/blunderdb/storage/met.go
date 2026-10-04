package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// MatchEquityTableStore keeps the library's match equity tables (ADR-0068).
// None current means the built-in Kazaross-XG2.
type MatchEquityTableStore interface {
	// Save stores the table and returns its id. A table whose digest the
	// scope already holds is not stored twice: Save returns the existing id
	// and leaves its name. An empty name, digest or source is ErrInvalid.
	Save(ctx context.Context, scope string, met domain.MatchEquityTable) (int64, error)
	// List returns the tables by name, without their source.
	List(ctx context.Context, scope string) ([]*domain.MatchEquityTable, error)
	// Current returns the current table with its source, or nil when the
	// library uses the built-in one.
	Current(ctx context.Context, scope string) (*domain.MatchEquityTable, error)
	// SetCurrent makes the table id current, or the built-in one when id is
	// 0; ErrNotFound for an unknown id.
	SetCurrent(ctx context.Context, scope string, id int64) error
	// TagAnalyses records that the analyses of positionIDs were computed
	// with the table metID, or the built-in one when metID is 0. Imported
	// analyses are never tagged: their files do not name a table.
	TagAnalyses(ctx context.Context, scope string, metID int64, positionIDs []int64) error
	// Load returns the table id with its source; ErrNotFound for an unknown
	// id.
	Load(ctx context.Context, scope string, id int64) (*domain.MatchEquityTable, error)
	// OfAnalyses returns the table of each of positionIDs' analyses that was
	// valued with an imported table; the others are absent.
	OfAnalyses(ctx context.Context, scope string, positionIDs []int64) (map[int64]int64, error)
	// OfAnalysis returns the table the position's analysis was computed
	// with: 0 for the built-in one, or when the position has no analysis.
	OfAnalysis(ctx context.Context, scope string, positionID int64) (int64, error)
}
