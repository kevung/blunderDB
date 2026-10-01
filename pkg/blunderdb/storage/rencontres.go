package storage

import (
	"context"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// RencontreStore persists the Rencontres and which Tournament plays in which
// (ADR-0056). A Tournament is in at most one Rencontre; deleting a Rencontre
// detaches its Tournaments and never deletes one.
type RencontreStore interface {
	Create(ctx context.Context, scope string, r domain.Rencontre) (int64, error)
	// Get returns the Rencontre with its member Tournaments, its table
	// settings (by number) and the rooms of its events, or ErrNotFound.
	Get(ctx context.Context, scope string, id int64) (*domain.Rencontre, error)
	// List returns every Rencontre with its members, newest first.
	List(ctx context.Context, scope string) ([]*domain.Rencontre, error)
	// Update rewrites name, dates, tables and output folder.
	Update(ctx context.Context, scope string, r domain.Rencontre) error
	Delete(ctx context.Context, scope string, id int64) error
	// Attach puts a Tournament in a Rencontre; rencontreID 0 detaches it and
	// clears its rooms, a fact of the membership (ADR-0058 §5).
	Attach(ctx context.Context, scope string, tournamentID, rencontreID int64) error
	// Of returns the Rencontre a Tournament plays in, 0 when none.
	Of(ctx context.Context, scope string, tournamentID int64) (int64, error)

	// SetTableSettings replaces the whole list of the Rencontre's table
	// properties (ADR-0058). A number ≤ 0 or given twice is ErrInvalid and
	// writes nothing; a missing Rencontre is ErrNotFound.
	SetTableSettings(ctx context.Context, scope string, rencontreID int64, settings []domain.TableSetting) error
	// SetEventRooms sets the rooms a Tournament may play in within its
	// Rencontre; nil or empty means every table. ErrNotFound when the
	// Tournament does not exist.
	SetEventRooms(ctx context.Context, scope string, tournamentID int64, rooms []string) error
	// TournamentTableSettings returns the table properties a Tournament owns
	// to run on its own, by number. They are kept, and ignored, while it
	// plays in a Rencontre (ADR-0058 §3).
	TournamentTableSettings(ctx context.Context, scope string, tournamentID int64) ([]domain.TableSetting, error)
	// SetTournamentTableSettings is SetTableSettings for a Tournament's own
	// tables.
	SetTournamentTableSettings(ctx context.Context, scope string, tournamentID int64, settings []domain.TableSetting) error
}
