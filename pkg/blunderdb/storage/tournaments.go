package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TournamentStore persists tournaments and their match membership.
type TournamentStore interface {
	Create(ctx context.Context, scope string, name, date, location string) (int64, error)
	Get(ctx context.Context, scope string, id int64) (*domain.Tournament, error)
	List(ctx context.Context, scope string) iter.Seq2[*domain.Tournament, error]
	Update(ctx context.Context, scope string, id int64, name, date, location string) error
	UpdateComment(ctx context.Context, scope string, id int64, comment string) error
	Delete(ctx context.Context, scope string, id int64) error

	AddMatch(ctx context.Context, scope string, tournamentID, matchID int64) error
	RemoveMatch(ctx context.Context, scope string, matchID int64) error
	SetMatchByName(ctx context.Context, scope string, matchID int64, tournamentName string) error
	ReorderMatches(ctx context.Context, scope string, tournamentID int64, matchIDs []int64) error

	// Matches streams the matches of a tournament in order.
	Matches(ctx context.Context, scope string, tournamentID int64) iter.Seq2[*domain.Match, error]

	// TournamentOf returns the tournament a match belongs to, or ErrNotFound.
	TournamentOf(ctx context.Context, scope string, matchID int64) (*domain.Tournament, error)
}
