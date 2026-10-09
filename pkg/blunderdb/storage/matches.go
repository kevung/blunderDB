package storage

import (
	"context"
	"iter"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// MatchListOpts filters, orders and paginates a match List query. Zero values
// mean "no filter" / "default order" / "no limit" / "from the start", so a zero
// MatchListOpts reproduces the historical stream-everything-by-date behaviour.
// MatchDice is one match as its dice describe it (MatchStore.DiceSequences).
// Games holds, per game in order, the dice of its checker moves; Initial is
// the score at the start of each game, in seat order.
type MatchDice struct {
	ID       int64
	Player1  string
	Player2  string
	Length   int
	DiceHash string
	Initial  [][2]int
	Games    [][][2]int
}

type MatchListOpts struct {
	// PlayerName keeps only matches where this name, or any spelling the
	// player aliases give the same person, is player 1 or player 2 — as the
	// statistics and the position search read a player. It is the
	// match-level "my matches" filter — distinct from StatsFilter.PlayerName,
	// which selects a player's decisions by joining moves.
	PlayerName string
	// PlayerSpellings is PlayerName's group of spellings, filled by the store
	// from the alias table; a caller leaves it empty.
	PlayerSpellings []string
	// PlayerNameContains keeps matches where either player's name contains
	// this text, case-insensitively (ASCII only on SQLite, whose LIKE folds
	// nothing else), with % and _ taken literally. It serves a search box.
	PlayerNameContains string
	// Text keeps matches where this text appears, case-insensitively, in a
	// player name, the event, location, round, tournament name or date: the
	// match panel's search box. % and _ are taken literally.
	Text string
	// Unassigned keeps only matches that belong to no tournament.
	Unassigned    bool
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

	// Reinstate stores m again under its own id m.ID and its import date
	// m.ImportDate (zero means now), for a match deleted earlier and put back
	// from the trash: what pointed at the match by id finds it again. The id
	// must be one the store issued — ErrInvalid otherwise — and free:
	// ErrConflict when a match of any scope holds it. Everything else is
	// written as Save writes it.
	Reinstate(ctx context.Context, scope string, m *domain.Match) error
	// FindByHash looks up an existing match for duplicate detection. It returns
	// the id of a match whose match_hash equals hash (same-format duplicate) or
	// whose canonical_hash equals canonicalHash (cross-format duplicate),
	// preferring an exact match_hash match. found is false when neither is
	// present. Empty arguments are ignored.
	FindByHash(ctx context.Context, scope string, hash, canonicalHash string) (id int64, found bool, err error)

	// ListByDiceHash returns the matches whose dice_hash equals hash, by id:
	// the candidates for "the same match under other names". Empty hash, no
	// match.
	ListByDiceHash(ctx context.Context, scope string, hash string) ([]domain.Match, error)

	// SetDiceHash stores a match's dice_hash, computed after the fact for a
	// match imported before the column existed.
	SetDiceHash(ctx context.Context, scope string, id int64, hash string) error

	// DiceSequences streams, by match id, every match with the dice of its
	// checker moves per game in game and move order — what ingest needs to
	// compute dice_hash and compare matches by their dice.
	DiceSequences(ctx context.Context, scope string) iter.Seq2[MatchDice, error]

	// Get returns the match with the given id, or ErrNotFound.
	Get(ctx context.Context, scope string, id int64) (*domain.Match, error)

	// List streams stored matches, filtered, ordered and paginated per opts. A
	// zero MatchListOpts streams every match, most recent first (the historical
	// behaviour).
	List(ctx context.Context, scope string, opts MatchListOpts) iter.Seq2[*domain.Match, error]

	// Count returns how many matches satisfy the filters of opts (Sort, Limit
	// and Offset are ignored): the total behind a paginated List.
	Count(ctx context.Context, scope string, opts MatchListOpts) (int, error)

	// Update changes the editable header fields of a match.
	Update(ctx context.Context, scope string, id int64, player1Name, player2Name, matchDate string) error

	// UpdateComment sets the free-text comment on a match.
	UpdateComment(ctx context.Context, scope string, id int64, comment string) error

	// SetVideoSource attaches the video a match was transcribed from
	// (ADR-0082), or detaches it when source is empty. ErrNotFound when no
	// match has this id.
	SetVideoSource(ctx context.Context, scope string, id int64, source string) error

	// ReplaceHeader rewrites the header columns of an existing match in place,
	// from m: the two names, the event, location and round, the length, the
	// date, the hashes, the game count and the source metadata (Elo,
	// experience, transcriber, session rules, engine version). The id, the import date, the
	// tournament, the comment and the last-visited position are NOT touched —
	// they are what a replacement exists to preserve (ADR-0045 §2), and none of
	// them is a property of the transcript being re-saved. The video source
	// (ADR-0082) is kept when m.VideoSource is nil, cleared when it is "", and
	// replaced otherwise. A header built by transcript.BuildPlayed — a
	// Transcription's save, a Duel's — always states it, "" included, so a
	// video detached in the draft leaves the Match; nil is for a writer that
	// knows nothing of the video, which leaves it where it was.
	//
	// Update is the user's edit of a match's identity (two names and a date);
	// this is the writer re-stating what the match now contains.
	ReplaceHeader(ctx context.Context, scope string, id int64, m *domain.Match) error

	// DeleteCascade removes a match and all of its games, moves and analyses.
	// The implementation runs the whole multi-table cascade atomically; when
	// reached through a Tx it joins that transaction.
	DeleteCascade(ctx context.Context, scope string, id int64) error

	// DeleteGames removes every game of a match — and, by cascade, its moves
	// and move analyses — while LEAVING the match row itself. It returns the
	// ids of the positions those moves referenced, the candidates for the
	// orphan purge.
	//
	// It does NOT purge them: the caller rewrites the match's games first, so a
	// position both versions share is still held (and keeps its id and
	// analysis). PurgeOrphanPositions is the second half.
	DeleteGames(ctx context.Context, scope string, matchID int64) (positionIDs []int64, err error)

	// PurgeOrphanPositions deletes each of the given positions that nothing
	// holds any more, by the same retention predicate DeleteCascade applies.
	// Positions still held — by another move, a collection, an Anki card, a
	// comment the user wrote, an individual import or a study mark — are left
	// alone, and so are ids that no longer exist.
	PurgeOrphanPositions(ctx context.Context, scope string, positionIDs []int64) error

	// ReanchorAnsweredDoubles moves every take or pass recorded the
	// transcript's way — the turned cube owned by the answerer, on a row no
	// double of its game stands on — onto
	// the position importers record it on: the same, the cube held by no one
	// (sqlshared.AnsweredOwnedCubeMovesSQL). Copy-on-write, as SwapPlayers: the
	// row it leaves keeps its id and analysis for whatever else holds it — a
	// redouble on the same board — and is purged when nothing does, its stale
	// analysis with it; its provenance flags stay behind. The moved moves are
	// rescored by their new row. Invalidates the stats of the matches it
	// touches and returns how many moves moved.
	ReanchorAnsweredDoubles(ctx context.Context, scope string) (int, error)

	// SwapPlayers swaps player 1 and player 2 for the match (and mirrors the
	// stored positions accordingly).
	SwapPlayers(ctx context.Context, scope string, id int64) error

	// MergePlayers makes every other given player name an alias of the
	// canonical name; the stored matches keep the names their files wrote,
	// and every reader that groups players resolves them through the aliases.
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
	// then move number, in one query; callers regroup by Move.GameID.
	MovesByMatch(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.Move, error]

	// MovePositions streams the positions of a match together with their
	// game/move context.
	MovePositions(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.MatchMovePosition, error]

	// MovesByPositions returns the moves recorded against each of the given
	// positions — what was actually played there, in every match that reached
	// it — keyed by position id, in id order.
	MovesByPositions(ctx context.Context, scope string, positionIDs []int64) (map[int64][]*domain.Move, error)

	// ScoreMoves writes move.error_mp (domain.Move.ErrorMP) for at most limit
	// unscored moves past the id after whose position carries an analysis,
	// in id order, and returns the id of the last move it examined (0 when
	// none is left) and how many it scored. A play the analysis cannot score
	// stays NULL. It is the resumable pass that scores a library older than
	// the column: the caller loops from 0 on the id it gets back, and a pass
	// interrupted and restarted skips what it already wrote.
	ScoreMoves(ctx context.Context, scope string, after int64, limit int) (next int64, scored int, err error)

	// RescorePositionMoves rewrites move.error_mp, and the decision error the
	// statistics count (move.decision_error_mp, move.is_close_cube), for every
	// move played from the position, from its current analysis (NULL when it
	// has none, or when a play is absent from it): an analysis written or
	// replaced, or moves pointed at another position, make them stale. The
	// matches whose counted errors change lose their match_stats rows.
	RescorePositionMoves(ctx context.Context, scope string, positionID int64) error

	// CreateMoveAnalysis stores an evaluation row attached to a move and
	// returns its id, updating ma.ID in place.
	CreateMoveAnalysis(ctx context.Context, scope string, ma *domain.MoveAnalysis) (int64, error)

	// MoveAnalysesByMatch streams every move analysis of a match, by game,
	// move, then id.
	MoveAnalysesByMatch(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.MoveAnalysis, error]
}

// DiceRow is one row of DiceSequencesSQL; GameID and the dice are 0 where
// the LEFT JOINs found nothing.
type DiceRow struct {
	MatchID, GameID        int64
	Player1, Player2, Hash string
	Length                 int
	Score1, Score2, D1, D2 int
}

// DiceFolder folds the ordered rows of DiceSequencesSQL into MatchDice, one
// per match: Add returns the previous match once a row of the next arrives,
// Flush the last one.
type DiceFolder struct {
	cur      *MatchDice
	lastGame int64
}

// Add folds r in, and returns the match it completed, if any.
func (f *DiceFolder) Add(r DiceRow) (MatchDice, bool) {
	var done MatchDice
	completed := false
	if f.cur != nil && f.cur.ID != r.MatchID {
		done, completed = *f.cur, true
		f.cur = nil
	}
	if f.cur == nil {
		f.cur = &MatchDice{ID: r.MatchID, Player1: r.Player1, Player2: r.Player2, Length: r.Length, DiceHash: r.Hash}
		f.lastGame = 0
	}
	if r.GameID != 0 && r.GameID != f.lastGame {
		f.cur.Games = append(f.cur.Games, nil)
		f.cur.Initial = append(f.cur.Initial, [2]int{r.Score1, r.Score2})
		f.lastGame = r.GameID
	}
	if r.GameID != 0 && r.D1 > 0 {
		last := len(f.cur.Games) - 1
		f.cur.Games[last] = append(f.cur.Games[last], [2]int{r.D1, r.D2})
	}
	return done, completed
}

// Flush returns the match still being folded, if any.
func (f *DiceFolder) Flush() (MatchDice, bool) {
	if f.cur == nil {
		return MatchDice{}, false
	}
	done := *f.cur
	f.cur = nil
	return done, true
}

// ScoreBatchSize bounds the moves one ScoreMoves call examines.
const ScoreBatchSize = 5000

// ScoreAllMoves runs the MatchStore.ScoreMoves pass over the whole scope and
// returns how many moves it scored. before, when non-nil, runs around each
// batch (the GUI's wrapper takes its lock there); it returns the function that
// ends the batch. progress, when non-nil, receives the running total.
func ScoreAllMoves(ctx context.Context, store MatchStore, scope string, before func() func(), progress func(scored int)) (int, error) {
	var next int64
	total := 0
	for {
		done := func() {}
		if before != nil {
			done = before()
		}
		n, k, err := store.ScoreMoves(ctx, scope, next, ScoreBatchSize)
		done()
		if err != nil {
			return total, err
		}
		total += k
		if progress != nil {
			progress(total)
		}
		if n == 0 {
			return total, nil
		}
		next = n
	}
}
