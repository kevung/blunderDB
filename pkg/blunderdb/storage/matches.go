package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// MatchStore persists matches and their games, moves and move analyses.
type MatchStore interface {
	// Save stores a new match and returns its id.
	Save(ctx context.Context, scope string, m *domain.Match) (int64, error)

	// Get returns the match with the given id, or ErrNotFound.
	Get(ctx context.Context, scope string, id int64) (*domain.Match, error)

	// List streams every stored match.
	List(ctx context.Context, scope string) iter.Seq2[*domain.Match, error]

	// Update changes the editable header fields of a match.
	Update(ctx context.Context, scope string, id int64, player1Name, player2Name, matchDate string) error

	// UpdateComment sets the free-text comment on a match.
	UpdateComment(ctx context.Context, scope string, id int64, comment string) error

	// DeleteCascade removes a match and all of its games, moves and analyses.
	// It must run inside tx so the multi-table cascade is atomic.
	DeleteCascade(ctx context.Context, tx Tx, scope string, id int64) error

	// SwapPlayers swaps player 1 and player 2 for the match (and mirrors the
	// stored positions accordingly).
	SwapPlayers(ctx context.Context, scope string, id int64) error

	// MergePlayers rewrites every occurrence of the given player names to a
	// single canonical name.
	MergePlayers(ctx context.Context, scope string, names []string, canonical string) error

	// SetLastVisitedPosition records the last position index viewed in a match.
	SetLastVisitedPosition(ctx context.Context, scope string, id int64, positionIndex int) error

	// LastVisited returns the most recently visited match, or ErrNotFound.
	LastVisited(ctx context.Context, scope string) (*domain.Match, error)

	// CreateGame stores a new game and returns its id.
	CreateGame(ctx context.Context, scope string, g *domain.Game) (int64, error)

	// Games streams the games of a match in order.
	Games(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.Game, error]

	// CreateMove stores a new move and returns its id.
	CreateMove(ctx context.Context, scope string, mv *domain.Move) (int64, error)

	// Moves streams the moves of a game in order.
	Moves(ctx context.Context, scope string, gameID int64) iter.Seq2[*domain.Move, error]

	// MovePositions streams the positions of a match together with their
	// game/move context.
	MovePositions(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.MatchMovePosition, error]
}
