package postgres

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type matchStore struct{ db execer }

var _ storage.MatchStore = (*matchStore)(nil)

// txBeginner is the subset of *pgxpool.Pool and pgx.Tx that starts a
// transaction (a pgx.Tx opens a savepoint-backed nested transaction). The
// multi-statement match operations (SwapPlayers, MergePlayers) type-assert
// their execer to this so the writes commit atomically whether the store is
// bound to the pool or already inside a caller's transaction.
type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// nullableTime maps a zero time.Time to a SQL NULL so an unset match date is
// stored as NULL rather than the year-1 sentinel.
func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// matchSelectCols is the column list read back into a domain.Match. The
// LEFT JOIN on tournament supplies the denormalised tournament name.
const matchSelectCols = `m.id, COALESCE(m.player1_name,''), COALESCE(m.player2_name,''),
	COALESCE(m.event,''), COALESCE(m.location,''), COALESCE(m.round,''),
	COALESCE(m.match_length,0), m.match_date, m.import_date,
	COALESCE(m.file_path,''), COALESCE(m.game_count,0),
	m.tournament_id, COALESCE(t.name,''),
	COALESCE(m.last_visited_position,-1), COALESCE(m.comment,''),
	COALESCE(m.tournament_sort_order,0)`

// scanMatch reconstructs a domain.Match from a row selected with
// matchSelectCols. match_date is nullable; tournament_id is nullable.
func scanMatch(sc scanner) (domain.Match, error) {
	var m domain.Match
	var matchDate *time.Time
	var tournamentID *int64
	if err := sc.Scan(
		&m.ID, &m.Player1Name, &m.Player2Name,
		&m.Event, &m.Location, &m.Round,
		&m.MatchLength, &matchDate, &m.ImportDate,
		&m.FilePath, &m.GameCount,
		&tournamentID, &m.TournamentName,
		&m.LastVisitedPosition, &m.Comment,
		&m.TournamentSortOrder,
	); err != nil {
		return domain.Match{}, err
	}
	if matchDate != nil {
		m.MatchDate = *matchDate
	}
	m.TournamentID = tournamentID
	return m, nil
}

const matchInsertSQL = `INSERT INTO match (
	tenant_id, player1_name, player2_name, event, location, round,
	match_length, match_date, file_path, game_count, tournament_id, comment,
	match_hash, canonical_hash
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
RETURNING id, import_date`

// nullableString returns nil for an empty string so it is stored as SQL NULL,
// keeping the UNIQUE(tenant_id, canonical_hash) index from rejecting a second
// hash-less match.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Save stores a new match and returns its id, updating m.ID and m.ImportDate
// in place.
func (s *matchStore) Save(ctx context.Context, scope string, m *domain.Match) (int64, error) {
	var id int64
	var importDate time.Time
	err := s.db.QueryRow(ctx, matchInsertSQL,
		tenantID(scope), m.Player1Name, m.Player2Name, m.Event, m.Location, m.Round,
		m.MatchLength, nullableTime(m.MatchDate), m.FilePath, m.GameCount,
		m.TournamentID, m.Comment,
		nullableString(m.MatchHash), nullableString(m.CanonicalHash)).Scan(&id, &importDate)
	if err != nil {
		return 0, fmt.Errorf("postgres: save match: %w", err)
	}
	m.ID = id
	m.ImportDate = importDate
	return id, nil
}

// FindByHash returns the id of a match matching hash (preferred) or
// canonicalHash, scoped to the tenant, for duplicate detection.
func (s *matchStore) FindByHash(ctx context.Context, scope string, hash, canonicalHash string) (int64, bool, error) {
	for _, q := range []struct {
		col string
		val string
	}{
		{"match_hash", hash},
		{"canonical_hash", canonicalHash},
	} {
		if q.val == "" {
			continue
		}
		var id int64
		err := s.db.QueryRow(ctx,
			`SELECT id FROM match WHERE tenant_id = $1 AND `+q.col+` = $2 LIMIT 1`,
			tenantID(scope), q.val).Scan(&id)
		if err == nil {
			return id, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, false, fmt.Errorf("postgres: find match by %s: %w", q.col, err)
		}
	}
	return 0, false, nil
}

// matchOrderClause sorts matches by play date, falling back to import date
// when the match date is unset.
const matchOrderClause = ` ORDER BY COALESCE(m.match_date, m.import_date) DESC`

// Get returns the match with the given id, or ErrNotFound.
func (s *matchStore) Get(ctx context.Context, scope string, id int64) (*domain.Match, error) {
	row := s.db.QueryRow(ctx,
		`SELECT `+matchSelectCols+` FROM match m
		 LEFT JOIN tournament t ON m.tournament_id = t.id
		 WHERE m.id = $1 AND m.tenant_id = $2`,
		id, tenantID(scope))
	m, err := scanMatch(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("postgres: get match %d: %w", id, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: get match %d: %w", id, err)
	}
	return &m, nil
}

// List streams every stored match, most recent first.
func (s *matchStore) List(ctx context.Context, scope string) iter.Seq2[*domain.Match, error] {
	return func(yield func(*domain.Match, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT `+matchSelectCols+` FROM match m
			 LEFT JOIN tournament t ON m.tournament_id = t.id
			 WHERE m.tenant_id = $1`+matchOrderClause,
			tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list matches: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			m, err := scanMatch(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: list matches: %w", err))
				return
			}
			if !yield(&m, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list matches: %w", err))
		}
	}
}

// Update changes the editable header fields of a match. matchDate is either
// empty or a "2006-01-02" date string.
func (s *matchStore) Update(ctx context.Context, scope string, id int64, player1Name, player2Name, matchDate string) error {
	var dateVal any
	if matchDate != "" {
		t, err := time.Parse("2006-01-02", matchDate)
		if err != nil {
			return fmt.Errorf("postgres: update match %d: invalid date %q: %w", id, matchDate, err)
		}
		dateVal = t
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE match SET player1_name = $1, player2_name = $2, match_date = $3
		 WHERE id = $4 AND tenant_id = $5`,
		player1Name, player2Name, dateVal, id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: update match %d: %w", id, err)
	}
	return nil
}

// UpdateComment sets the free-text comment on a match.
func (s *matchStore) UpdateComment(ctx context.Context, scope string, id int64, comment string) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE match SET comment = $1 WHERE id = $2 AND tenant_id = $3`,
		comment, id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: update match %d comment: %w", id, err)
	}
	return nil
}

// positionIsHeldSQL reports whether anything still holds a position once the
// match that referenced it is gone. Deleting a match must not destroy work the
// user did on a position that merely happened to occur in it.
//
// A position is held by: another match's move; membership in a collection; a
// comment the user wrote; an Anki card built from it; or having been imported
// individually, which says the user brought it in deliberately (docs/adr/0001).
//
// An analysis deliberately does NOT hold a position: it arrives with the match
// rather than from the user, and every match position has one, so counting it
// would mean never purging anything.
const positionIsHeldSQL = `SELECT EXISTS (SELECT 1 FROM move               WHERE position_id = $1)
	                       OR EXISTS (SELECT 1 FROM collection_position WHERE position_id = $1)
	                       OR EXISTS (SELECT 1 FROM comment             WHERE position_id = $1)
	                       OR EXISTS (SELECT 1 FROM anki_card           WHERE position_id = $1)
	                       OR EXISTS (SELECT 1 FROM position            WHERE id = $1 AND individually_imported)`

// DeleteCascade removes a match and all of its games, moves and move analyses
// (via ON DELETE CASCADE), then deletes any position the match referenced that
// nothing else holds (see positionIsHeldSQL). The cascade and the orphan
// cleanup run in one transaction (a savepoint when the store is already inside
// a caller's tx).
func (s *matchStore) DeleteCascade(ctx context.Context, scope string, id int64) error {
	tenant := tenantID(scope)
	return s.inTx(ctx, fmt.Sprintf("delete match %d", id), func(tx pgx.Tx) error {
		// Collect the positions this match's moves reference before the
		// cascade removes those moves.
		rows, err := tx.Query(ctx,
			`SELECT DISTINCT mv.position_id
			 FROM move mv INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = $1 AND mv.tenant_id = $2 AND mv.position_id IS NOT NULL`,
			id, tenant)
		if err != nil {
			return fmt.Errorf("collect positions: %w", err)
		}
		var positionIDs []int64
		for rows.Next() {
			var pid int64
			if err := rows.Scan(&pid); err != nil {
				rows.Close()
				return fmt.Errorf("scan position id: %w", err)
			}
			positionIDs = append(positionIDs, pid)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("collect positions: %w", err)
		}

		// game/move/move_analysis cascade off the match delete.
		if _, err := tx.Exec(ctx,
			`DELETE FROM match WHERE id = $1 AND tenant_id = $2`, id, tenant); err != nil {
			return err
		}

		// Drop positions nothing holds any more; their analyses cascade off the
		// position delete.
		for _, pid := range positionIDs {
			var held bool
			if err := tx.QueryRow(ctx, positionIsHeldSQL, pid).Scan(&held); err != nil {
				return fmt.Errorf("ref check position %d: %w", pid, err)
			}
			if !held {
				if _, err := tx.Exec(ctx,
					`DELETE FROM position WHERE id = $1 AND tenant_id = $2`, pid, tenant); err != nil {
					return fmt.Errorf("delete orphan position %d: %w", pid, err)
				}
			}
		}
		return nil
	})
}

// SwapPlayers swaps player 1 and player 2 for the match: it swaps the header
// names, the per-game scores and winner, the per-move player, and the score /
// cube-owner columns of every position the match's moves reference.
func (s *matchStore) SwapPlayers(ctx context.Context, scope string, id int64) error {
	tenant := tenantID(scope)
	return s.inTx(ctx, "swap players", func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`UPDATE match SET player1_name = player2_name, player2_name = player1_name
			 WHERE id = $1 AND tenant_id = $2`, id, tenant); err != nil {
			return fmt.Errorf("swap names: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE game SET initial_score_1 = initial_score_2,
			                 initial_score_2 = initial_score_1,
			                 winner = -winner
			 WHERE match_id = $1 AND tenant_id = $2`, id, tenant); err != nil {
			return fmt.Errorf("swap game scores: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE move SET player = -player
			 WHERE tenant_id = $2
			   AND game_id IN (SELECT id FROM game WHERE match_id = $1)`, id, tenant); err != nil {
			return fmt.Errorf("swap move players: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE position SET
				score_1 = score_2, score_2 = score_1,
				cube_owner = CASE WHEN cube_owner = -1 THEN -1
				                  WHEN cube_owner IS NULL THEN NULL
				                  ELSE 1 - cube_owner END
			 WHERE tenant_id = $2 AND id IN (
				SELECT DISTINCT mv.position_id FROM move mv
				INNER JOIN game g ON mv.game_id = g.id
				WHERE g.match_id = $1 AND mv.position_id IS NOT NULL
			 )`, id, tenant); err != nil {
			return fmt.Errorf("swap position scores: %w", err)
		}
		return nil
	})
}

// MergePlayers rewrites every occurrence of the given player names (in both
// the player1 and player2 columns) to a single canonical name.
func (s *matchStore) MergePlayers(ctx context.Context, scope string, names []string, canonical string) error {
	if canonical == "" {
		return fmt.Errorf("postgres: merge players: canonical name must not be empty: %w", storage.ErrInternal)
	}
	if len(names) == 0 {
		return fmt.Errorf("postgres: merge players: no names to merge: %w", storage.ErrInternal)
	}
	tenant := tenantID(scope)
	return s.inTx(ctx, "merge players", func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`UPDATE match SET player1_name = $1
			 WHERE tenant_id = $2 AND player1_name = ANY($3)`,
			canonical, tenant, names); err != nil {
			return fmt.Errorf("rename player1: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE match SET player2_name = $1
			 WHERE tenant_id = $2 AND player2_name = ANY($3)`,
			canonical, tenant, names); err != nil {
			return fmt.Errorf("rename player2: %w", err)
		}
		return nil
	})
}

// inTx runs fn inside a transaction started from the store's execer. When the
// store is already bound to a transaction the pgx.Tx opens a savepoint, so the
// operation is atomic in either binding.
func (s *matchStore) inTx(ctx context.Context, what string, fn func(pgx.Tx) error) error {
	b, ok := s.db.(txBeginner)
	if !ok {
		return fmt.Errorf("postgres: %s: execer cannot begin a transaction: %w", what, storage.ErrInternal)
	}
	tx, err := b.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: %s: begin: %w", what, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(tx); err != nil {
		return fmt.Errorf("postgres: %s: %w", what, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: %s: commit: %w", what, err)
	}
	return nil
}

// SetLastVisitedPosition records the last position index viewed in a match.
func (s *matchStore) SetLastVisitedPosition(ctx context.Context, scope string, id int64, positionIndex int) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE match SET last_visited_position = $1 WHERE id = $2 AND tenant_id = $3`,
		positionIndex, id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: set last visited position for match %d: %w", id, err)
	}
	return nil
}

// LastVisited returns the most recently visited match, falling back to the
// most recent match when none has been visited, or ErrNotFound when the
// scope holds no matches.
func (s *matchStore) LastVisited(ctx context.Context, scope string) (*domain.Match, error) {
	tenant := tenantID(scope)
	row := s.db.QueryRow(ctx,
		`SELECT `+matchSelectCols+` FROM match m
		 LEFT JOIN tournament t ON m.tournament_id = t.id
		 WHERE m.tenant_id = $1 AND m.last_visited_position >= 0
		 ORDER BY m.import_date DESC LIMIT 1`, tenant)
	m, err := scanMatch(row)
	if err == nil {
		return &m, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("postgres: last visited match: %w", err)
	}

	row = s.db.QueryRow(ctx,
		`SELECT `+matchSelectCols+` FROM match m
		 LEFT JOIN tournament t ON m.tournament_id = t.id
		 WHERE m.tenant_id = $1`+matchOrderClause+` LIMIT 1`, tenant)
	m, err = scanMatch(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("postgres: last visited match: %w", storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: last visited match: %w", err)
	}
	return &m, nil
}

const gameInsertSQL = `INSERT INTO game (
	tenant_id, match_id, game_number, initial_score_1, initial_score_2,
	winner, points_won, move_count
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`

// CreateGame stores a new game and returns its id, updating g.ID in place.
func (s *matchStore) CreateGame(ctx context.Context, scope string, g *domain.Game) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, gameInsertSQL,
		tenantID(scope), g.MatchID, g.GameNumber,
		g.InitialScore[0], g.InitialScore[1],
		g.Winner, g.PointsWon, g.MoveCount).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("postgres: create game: %w", err)
	}
	g.ID = id
	return id, nil
}

// Games streams the games of a match ordered by game number.
func (s *matchStore) Games(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.Game, error] {
	return func(yield func(*domain.Game, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT id, match_id, COALESCE(game_number,0),
			        COALESCE(initial_score_1,0), COALESCE(initial_score_2,0),
			        COALESCE(winner,0), COALESCE(points_won,0), COALESCE(move_count,0)
			 FROM game WHERE match_id = $1 AND tenant_id = $2
			 ORDER BY game_number`,
			matchID, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list games: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var g domain.Game
			var s1, s2 int32
			if err := rows.Scan(&g.ID, &g.MatchID, &g.GameNumber,
				&s1, &s2, &g.Winner, &g.PointsWon, &g.MoveCount); err != nil {
				yield(nil, fmt.Errorf("postgres: list games: %w", err))
				return
			}
			g.InitialScore = [2]int32{s1, s2}
			if !yield(&g, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list games: %w", err))
		}
	}
}

const moveInsertSQL = `INSERT INTO move (
	tenant_id, game_id, move_number, move_type, position_id, player,
	dice_1, dice_2, checker_move, cube_action
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`

// CreateMove stores a new move and returns its id, updating mv.ID in place. A
// zero PositionID is stored as NULL (no associated position).
func (s *matchStore) CreateMove(ctx context.Context, scope string, mv *domain.Move) (int64, error) {
	var positionID any
	if mv.PositionID != 0 {
		positionID = mv.PositionID
	}
	var id int64
	err := s.db.QueryRow(ctx, moveInsertSQL,
		tenantID(scope), mv.GameID, mv.MoveNumber, mv.MoveType, positionID, mv.Player,
		mv.Dice[0], mv.Dice[1], mv.CheckerMove, mv.CubeAction).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("postgres: create move: %w", err)
	}
	mv.ID = id
	return id, nil
}

// Moves streams the moves of a game ordered by move number.
func (s *matchStore) Moves(ctx context.Context, scope string, gameID int64) iter.Seq2[*domain.Move, error] {
	return func(yield func(*domain.Move, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT id, game_id, COALESCE(move_number,0), COALESCE(move_type,''),
			        position_id, COALESCE(player,0),
			        COALESCE(dice_1,0), COALESCE(dice_2,0),
			        COALESCE(checker_move,''), COALESCE(cube_action,'')
			 FROM move WHERE game_id = $1 AND tenant_id = $2
			 ORDER BY move_number`,
			gameID, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list moves: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var mv domain.Move
			var d1, d2 int32
			var positionID *int64
			if err := rows.Scan(&mv.ID, &mv.GameID, &mv.MoveNumber, &mv.MoveType,
				&positionID, &mv.Player, &d1, &d2, &mv.CheckerMove, &mv.CubeAction); err != nil {
				yield(nil, fmt.Errorf("postgres: list moves: %w", err))
				return
			}
			mv.Dice = [2]int32{d1, d2}
			if positionID != nil {
				mv.PositionID = *positionID
			}
			if !yield(&mv, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list moves: %w", err))
		}
	}
}

// MovesByMatch streams every move of a match in chronological order (by game,
// then move). One query instead of Games + a Moves call per game: callers
// regroup by Move.GameID. Mirrors Moves' columns and MovePositions' match join.
func (s *matchStore) MovesByMatch(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.Move, error] {
	return func(yield func(*domain.Move, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT mv.id, mv.game_id, COALESCE(mv.move_number,0), COALESCE(mv.move_type,''),
			        mv.position_id, COALESCE(mv.player,0),
			        COALESCE(mv.dice_1,0), COALESCE(mv.dice_2,0),
			        COALESCE(mv.checker_move,''), COALESCE(mv.cube_action,'')
			 FROM move mv INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = $1 AND mv.tenant_id = $2
			 ORDER BY g.game_number, mv.move_number`,
			matchID, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list moves by match: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var mv domain.Move
			var d1, d2 int32
			var positionID *int64
			if err := rows.Scan(&mv.ID, &mv.GameID, &mv.MoveNumber, &mv.MoveType,
				&positionID, &mv.Player, &d1, &d2, &mv.CheckerMove, &mv.CubeAction); err != nil {
				yield(nil, fmt.Errorf("postgres: list moves by match: %w", err))
				return
			}
			mv.Dice = [2]int32{d1, d2}
			if positionID != nil {
				mv.PositionID = *positionID
			}
			if !yield(&mv, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list moves by match: %w", err))
		}
	}
}

// xgPlayerToBlunderDB maps the XG move-player encoding (1 / -1) stored in the
// move table to the blunderDB encoding (0 = player 1, 1 = player 2). GnuBG
// imports are stored already converted to the XG encoding, so this one mapping
// covers every move source.
func xgPlayerToBlunderDB(player int32) int32 {
	if player == 1 {
		return 0
	}
	return 1
}

// MovePositions streams every position of a match in chronological order,
// each carrying its game / move context. Positions are returned as stored
// (from the player-on-roll point of view).
func (s *matchStore) MovePositions(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.MatchMovePosition, error] {
	return func(yield func(*domain.MatchMovePosition, error) bool) {
		tenant := tenantID(scope)

		var player1Name, player2Name string
		err := s.db.QueryRow(ctx,
			`SELECT COALESCE(player1_name,''), COALESCE(player2_name,'')
			 FROM match WHERE id = $1 AND tenant_id = $2`,
			matchID, tenant).Scan(&player1Name, &player2Name)
		if errors.Is(err, pgx.ErrNoRows) {
			yield(nil, fmt.Errorf("postgres: move positions for match %d: %w", matchID, storage.ErrNotFound))
			return
		}
		if err != nil {
			yield(nil, fmt.Errorf("postgres: move positions for match %d: %w", matchID, err))
			return
		}

		rows, err := s.db.Query(ctx,
			`SELECT mv.id, mv.game_id, COALESCE(g.game_number,0), COALESCE(mv.move_number,0),
			        COALESCE(mv.move_type,''), COALESCE(mv.player,0), mv.position_id,
			        p.state, p.decision_type, p.player_on_roll, p.dice_1, p.dice_2,
			        p.cube_value, p.cube_owner, p.score_1, p.score_2,
			        p.has_jacoby, p.has_beaver,
			        COALESCE(mv.checker_move,''), COALESCE(mv.cube_action,'')
			 FROM move mv
			 INNER JOIN game g ON mv.game_id = g.id
			 INNER JOIN position p ON mv.position_id = p.id
			 WHERE g.match_id = $1 AND mv.tenant_id = $2
			 ORDER BY g.game_number, mv.move_number`,
			matchID, tenant)
		if err != nil {
			yield(nil, fmt.Errorf("postgres: move positions for match %d: %w", matchID, err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var moveID, gameID, positionID int64
			var gameNumber, moveNumber, player int32
			var moveType, state, checkerMove, cubeAction string
			var dt, por, d1, d2, cv, co, s1, s2 *int64
			var hj, hb *bool
			if err := rows.Scan(&moveID, &gameID, &gameNumber, &moveNumber,
				&moveType, &player, &positionID,
				&state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb,
				&checkerMove, &cubeAction); err != nil {
				yield(nil, fmt.Errorf("postgres: move positions for match %d: %w", matchID, err))
				return
			}
			position := engine.ReconstructPosition(positionID, state,
				derefInt(dt), derefInt(por), derefInt(d1), derefInt(d2),
				derefInt(cv), derefInt(co), derefInt(s1), derefInt(s2),
				boolToIntPtr(hj), boolToIntPtr(hb))
			mp := domain.MatchMovePosition{
				Position:     position,
				MoveID:       moveID,
				GameID:       gameID,
				GameNumber:   gameNumber,
				MoveNumber:   moveNumber,
				MoveType:     moveType,
				PlayerOnRoll: xgPlayerToBlunderDB(player),
				Player1Name:  player1Name,
				Player2Name:  player2Name,
				CheckerMove:  checkerMove,
				CubeAction:   cubeAction,
			}
			if !yield(&mp, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: move positions for match %d: %w", matchID, err))
		}
	}
}
