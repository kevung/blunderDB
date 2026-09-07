package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// MatchListOpts filters, orders and paginates a match List query. Zero values
// mean "no filter" / "default order" / "no limit" / "from the start", so a zero
// MatchListOpts reproduces the historical stream-everything-by-date behaviour.
type MatchListOpts struct {
	// PlayerName keeps only matches where this exact name is player 1 or
	// player 2. It is the match-level "my matches" filter — distinct from
	// StatsFilter.PlayerName, which selects a player's decisions by joining moves.
	PlayerName    string
	TournamentIDs []int64
	DateFrom      string // ISO "YYYY-MM-DD", inclusive
	DateTo        string // ISO "YYYY-MM-DD", inclusive
	MatchLength   []int
	// Sort is a key understood by domain.MatchOrderByClause ("" = most recent
	// first). PR/MWC are not match columns (they are computed badges), so they
	// are not sortable here.
	Sort   string
	Limit  int
	Offset int
}

// MatchStore persists matches and their games, moves and move analyses.
type MatchStore interface {
	// Save stores a new match and returns its id. m.MatchHash and
	// m.CanonicalHash, when non-empty, are persisted for duplicate detection.
	Save(ctx context.Context, scope string, m *domain.Match) (int64, error)

	// FindByHash looks up an existing match for duplicate detection. It returns
	// the id of a match whose match_hash equals hash (same-format duplicate) or
	// whose canonical_hash equals canonicalHash (cross-format duplicate),
	// preferring an exact match_hash match. found is false when neither is
	// present. Empty arguments are ignored.
	FindByHash(ctx context.Context, scope string, hash, canonicalHash string) (id int64, found bool, err error)

	// Get returns the match with the given id, or ErrNotFound.
	Get(ctx context.Context, scope string, id int64) (*domain.Match, error)

	// List streams stored matches, filtered, ordered and paginated per opts. A
	// zero MatchListOpts streams every match, most recent first (the historical
	// behaviour).
	List(ctx context.Context, scope string, opts MatchListOpts) iter.Seq2[*domain.Match, error]

	// Update changes the editable header fields of a match.
	Update(ctx context.Context, scope string, id int64, player1Name, player2Name, matchDate string) error

	// UpdateComment sets the free-text comment on a match.
	UpdateComment(ctx context.Context, scope string, id int64, comment string) error

	// ReplaceHeader rewrites the header columns of an existing match in place,
	// from m: the two names, the event, location and round, the length, the
	// date, the two hashes and the game count. The id, the import date, the
	// tournament, the comment and the last-visited position are NOT touched —
	// they are what a replacement exists to preserve (ADR-0045 §2), and none of
	// them is a property of the transcript being re-saved.
	//
	// Update is the user's edit of a match's identity (two names and a date);
	// this is the writer re-stating what the match now contains.
	ReplaceHeader(ctx context.Context, scope string, id int64, m *domain.Match) error

	// DeleteCascade removes a match and all of its games, moves and analyses.
	// The implementation runs the whole multi-table cascade atomically; when
	// reached through a Tx it joins that transaction (D2).
	DeleteCascade(ctx context.Context, scope string, id int64) error

	// DeleteGames removes every game of a match — and, by cascade, its moves
	// and move analyses — while LEAVING the match row itself. It returns the
	// ids of the positions those moves referenced, the candidates for the
	// orphan purge.
	//
	// It does NOT purge them: the caller rewrites the match's games first, so
	// that a position both versions share is still held when the predicate is
	// finally asked. Purging here would delete every position of the match that
	// nothing else holds — analysis included — and the rewrite would put it back
	// under a new id, which is exactly what a replacement must not do.
	// PurgeOrphanPositions is the second half.
	DeleteGames(ctx context.Context, scope string, matchID int64) (positionIDs []int64, err error)

	// PurgeOrphanPositions deletes each of the given positions that nothing
	// holds any more, by the same retention predicate DeleteCascade applies.
	// Positions still held — by another move, a collection, an Anki card, a
	// comment the user wrote, an individual import or a study mark — are left
	// alone, and so are ids that no longer exist.
	PurgeOrphanPositions(ctx context.Context, scope string, positionIDs []int64) error

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

	// MovesByMatch streams every move of a match in one pass, ordered by game
	// then move number. It lets a caller render a whole match detail without the
	// 1+N round-trips of calling Moves once per game (each move carries GameID so
	// the caller regroups by game). Mirrors MovePositions' match-scoped shape.
	MovesByMatch(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.Move, error]

	// MovePositions streams the positions of a match together with their
	// game/move context.
	MovePositions(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.MatchMovePosition, error]

	// MovesByPositions returns the moves recorded against each of the given
	// positions — what was actually played there, in every match that reached
	// it — keyed by position id, in id order.
	MovesByPositions(ctx context.Context, scope string, positionIDs []int64) (map[int64][]*domain.Move, error)

	// CreateMoveAnalysis stores an evaluation row attached to a move and
	// returns its id, updating ma.ID in place.
	CreateMoveAnalysis(ctx context.Context, scope string, ma *domain.MoveAnalysis) (int64, error)

	// MoveAnalysesByMatch streams every move analysis of a match, by game,
	// move, then id.
	MoveAnalysesByMatch(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.MoveAnalysis, error]
}
