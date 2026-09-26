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
	// Get returns the Rencontre with its member Tournaments, or ErrNotFound.
	Get(ctx context.Context, scope string, id int64) (*domain.Rencontre, error)
	// List returns every Rencontre with its members, newest first.
	List(ctx context.Context, scope string) ([]*domain.Rencontre, error)
	// Update rewrites name, dates, tables and output folder.
	Update(ctx context.Context, scope string, r domain.Rencontre) error
	Delete(ctx context.Context, scope string, id int64) error
	// Attach puts a Tournament in a Rencontre; rencontreID 0 detaches it.
	Attach(ctx context.Context, scope string, tournamentID, rencontreID int64) error
	// Of returns the Rencontre a Tournament plays in, 0 when none.
	Of(ctx context.Context, scope string, tournamentID int64) (int64, error)
}
