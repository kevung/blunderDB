package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

type matchStore struct{ db execer }

var _ storage.MatchStore = (*matchStore)(nil)

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
// matchSelectCols. match_date and tournament_id are nullable.
func scanMatch(sc interface{ Scan(...any) error }) (domain.Match, error) {
	var m domain.Match
	var matchDate, importDate sql.NullTime
	var tournamentID sql.NullInt64
	if err := sc.Scan(
		&m.ID, &m.Player1Name, &m.Player2Name,
		&m.Event, &m.Location, &m.Round,
		&m.MatchLength, &matchDate, &importDate,
		&m.FilePath, &m.GameCount,
		&tournamentID, &m.TournamentName,
		&m.LastVisitedPosition, &m.Comment,
		&m.TournamentSortOrder,
	); err != nil {
		return domain.Match{}, err
	}
	if matchDate.Valid {
		m.MatchDate = matchDate.Time
	}
	if importDate.Valid {
		m.ImportDate = importDate.Time
	}
	if tournamentID.Valid {
		tid := tournamentID.Int64
		m.TournamentID = &tid
	}
	return m, nil
}

const matchInsertSQL = `INSERT INTO match (
	player1_name, player2_name, event, location, round,
	match_length, match_date, file_path, game_count, tournament_id, comment,
	match_hash, canonical_hash
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`

// nullableString returns nil for an empty string so it is stored as SQL NULL.
// This matters for canonical_hash, whose UNIQUE index would otherwise reject a
// second match with an empty (non-NULL) hash.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Save stores a new match and returns its id, updating m.ID and m.ImportDate
// in place.
func (s *matchStore) Save(ctx context.Context, scope string, m *domain.Match) (int64, error) {
	res, err := s.db.ExecContext(ctx, matchInsertSQL,
		m.Player1Name, m.Player2Name, m.Event, m.Location, m.Round,
		m.MatchLength, nullableTime(m.MatchDate), m.FilePath, m.GameCount,
		m.TournamentID, m.Comment,
		nullableString(m.MatchHash), nullableString(m.CanonicalHash))
	if err != nil {
		return 0, fmt.Errorf("sqlite: save match: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sqlite: save match id: %w", err)
	}
	var importDate sql.NullTime
	if err := s.db.QueryRowContext(ctx,
		`SELECT import_date FROM match WHERE id = ?`, id).Scan(&importDate); err == nil && importDate.Valid {
		m.ImportDate = importDate.Time
	}
	m.ID = id
	return id, nil
}

// FindByHash returns the id of a match matching hash (preferred) or
// canonicalHash, for duplicate detection.
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
		err := s.db.QueryRowContext(ctx,
			`SELECT id FROM match WHERE `+q.col+` = ? LIMIT 1`, q.val).Scan(&id)
		if err == nil {
			return id, true, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, false, fmt.Errorf("sqlite: find match by %s: %w", q.col, err)
		}
	}
	return 0, false, nil
}

// matchOrderClause sorts matches by play date, falling back to import date
// when the match date is unset. Used by LastVisited; List builds its ORDER BY
// from domain.MatchOrderByClause so the key is configurable.
const matchOrderClause = ` ORDER BY COALESCE(m.match_date, m.import_date) DESC`

// buildMatchListWhere turns opts filters into a WHERE fragment (empty when no
// filter applies) and its positional args. Mirrors the Postgres builder; the
// two must stay in sync.
func buildMatchListWhere(opts storage.MatchListOpts) (whereSQL string, args []any) {
	var clauses []string
	if opts.PlayerName != "" {
		clauses = append(clauses, "(m.player1_name = ? OR m.player2_name = ?)")
		args = append(args, opts.PlayerName, opts.PlayerName)
	}
	if len(opts.TournamentIDs) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(opts.TournamentIDs)), ",")
		clauses = append(clauses, "m.tournament_id IN ("+ph+")")
		for _, id := range opts.TournamentIDs {
			args = append(args, id)
		}
	}
	// Compare on the date part (first 10 chars "YYYY-MM-DD") so an inclusive
	// DateTo (e.g. a whole-year filter "…-12-31") still matches a match
	// timestamped later that same day. substr, not date(), because match_date is
	// stored with a timezone suffix that SQLite's date() refuses to parse.
	if opts.DateFrom != "" {
		clauses = append(clauses, "substr(m.match_date,1,10) >= ?")
		args = append(args, opts.DateFrom)
	}
	if opts.DateTo != "" {
		clauses = append(clauses, "substr(m.match_date,1,10) <= ?")
		args = append(args, opts.DateTo)
	}
	if len(opts.MatchLength) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(opts.MatchLength)), ",")
		clauses = append(clauses, "m.match_length IN ("+ph+")")
		for _, ml := range opts.MatchLength {
			args = append(args, ml)
		}
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

// Get returns the match with the given id, or ErrNotFound.
func (s *matchStore) Get(ctx context.Context, scope string, id int64) (*domain.Match, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+matchSelectCols+` FROM match m
		 LEFT JOIN tournament t ON m.tournament_id = t.id
		 WHERE m.id = ?`, id)
	m, err := scanMatch(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sqlite: get match %d: %w", id, storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: get match %d: %w", id, err)
	}
	return &m, nil
}

// List streams stored matches, filtered/ordered/paginated per opts. A zero
// MatchListOpts streams every match, most recent first.
func (s *matchStore) List(ctx context.Context, scope string, opts storage.MatchListOpts) iter.Seq2[*domain.Match, error] {
	return func(yield func(*domain.Match, error) bool) {
		whereSQL, args := buildMatchListWhere(opts)
		query := `SELECT ` + matchSelectCols + ` FROM match m
			 LEFT JOIN tournament t ON m.tournament_id = t.id` + whereSQL +
			` ORDER BY ` + domain.MatchOrderByClause(opts.Sort)
		switch {
		case opts.Limit > 0:
			query += ` LIMIT ?`
			args = append(args, opts.Limit)
			if opts.Offset > 0 {
				query += ` OFFSET ?`
				args = append(args, opts.Offset)
			}
		case opts.Offset > 0:
			query += ` LIMIT -1 OFFSET ?`
			args = append(args, opts.Offset)
		}
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list matches: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			m, err := scanMatch(rows)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: list matches: %w", err))
				return
			}
			if !yield(&m, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list matches: %w", err))
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
			return fmt.Errorf("sqlite: update match %d: invalid date %q: %w", id, matchDate, err)
		}
		dateVal = t
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE match SET player1_name = ?, player2_name = ?, match_date = ? WHERE id = ?`,
		player1Name, player2Name, dateVal, id); err != nil {
		return fmt.Errorf("sqlite: update match %d: %w", id, err)
	}
	return nil
}

// UpdateComment sets the free-text comment on a match.
func (s *matchStore) UpdateComment(ctx context.Context, scope string, id int64, comment string) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE match SET comment = ? WHERE id = ?`, comment, id); err != nil {
		return fmt.Errorf("sqlite: update match %d comment: %w", id, err)
	}
	return nil
}

// positionIsHeldSQL reports whether anything still holds a position once the
// match that referenced it is gone. Deleting a match must not destroy work the
// user did on a position that merely happened to occur in it.
//
// A position is held by: another match's move; membership in a collection; an
// Anki card built from it; or having been imported individually, which says the
// user brought it in deliberately (ADR-0001).
//
// Two things deliberately do NOT hold a position, because neither is evidence
// the user did anything with it:
//   - an analysis: it arrives with the match, and every match position has one,
//     so counting it would mean never purging anything;
//   - a comment: match importers attach the source file's per-move notes as
//     comments (see ingest/xg.go), so a comment is not necessarily the user's.
//     A note the user wrote on a match position is therefore still lost when the
//     match is deleted — to keep such a position, put it in a collection or save
//     it, which marks it individually imported.
const positionIsHeldSQL = `SELECT EXISTS (SELECT 1 FROM move               WHERE position_id = ?1)
	                       OR EXISTS (SELECT 1 FROM collection_position WHERE position_id = ?1)
	                       OR EXISTS (SELECT 1 FROM anki_card           WHERE position_id = ?1)
	                       OR EXISTS (SELECT 1 FROM position            WHERE id = ?1 AND individually_imported = 1)`

// DeleteCascade removes a match and all of its games, moves and move analyses
// (via ON DELETE CASCADE), then deletes any position the match referenced that
// nothing else holds (see positionIsHeldSQL). The cascade and the orphan
// cleanup run atomically.
func (s *matchStore) DeleteCascade(ctx context.Context, scope string, id int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		rows, err := tx.QueryContext(ctx,
			`SELECT DISTINCT mv.position_id
			 FROM move mv INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = ? AND mv.position_id IS NOT NULL`, id)
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
		if _, err := tx.ExecContext(ctx, `DELETE FROM match WHERE id = ?`, id); err != nil {
			return err
		}

		for _, pid := range positionIDs {
			var held bool
			if err := tx.QueryRowContext(ctx, positionIsHeldSQL, pid).Scan(&held); err != nil {
				return fmt.Errorf("ref check position %d: %w", pid, err)
			}
			if !held {
				if _, err := tx.ExecContext(ctx, `DELETE FROM position WHERE id = ?`, pid); err != nil {
					return fmt.Errorf("delete orphan position %d: %w", pid, err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: delete match %d: %w", id, err)
	}
	return nil
}

// SwapPlayers swaps player 1 and player 2 for the match: it swaps the header
// names, the per-game scores and winner, the per-move player, and the score /
// cube-owner columns of every position the match's moves reference.
func (s *matchStore) SwapPlayers(ctx context.Context, scope string, id int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		if _, err := tx.ExecContext(ctx,
			`UPDATE match SET player1_name = player2_name, player2_name = player1_name
			 WHERE id = ?`, id); err != nil {
			return fmt.Errorf("swap names: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE game SET initial_score_1 = initial_score_2,
			                 initial_score_2 = initial_score_1,
			                 winner = -winner
			 WHERE match_id = ?`, id); err != nil {
			return fmt.Errorf("swap game scores: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE move SET player = -player
			 WHERE game_id IN (SELECT id FROM game WHERE match_id = ?)`, id); err != nil {
			return fmt.Errorf("swap move players: %w", err)
		}
		// Positions swap by copy-on-write, NOT in place (#107): a position is
		// deduplicated by Zobrist and may be shared with other matches, and its
		// score/cube are part of that hash. For each position this match uses, save
		// a swapped copy (Save recomputes the Zobrist and dedups) and repoint this
		// match's moves to it; the original stays intact for whoever else holds it.
		rows, err := tx.QueryContext(ctx,
			`SELECT DISTINCT mv.position_id FROM move mv
			 INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = ? AND mv.position_id IS NOT NULL`, id)
		if err != nil {
			return fmt.Errorf("collect swap positions: %w", err)
		}
		var posIDs []int64
		for rows.Next() {
			var pid int64
			if err := rows.Scan(&pid); err != nil {
				rows.Close()
				return fmt.Errorf("scan swap position id: %w", err)
			}
			posIDs = append(posIDs, pid)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("collect swap positions: %w", err)
		}

		ps := &positionStore{db: tx}
		for _, pid := range posIDs {
			pos, err := ps.Load(ctx, scope, pid)
			if err != nil {
				return fmt.Errorf("load swap position %d: %w", pid, err)
			}
			pos.Score[0], pos.Score[1] = pos.Score[1], pos.Score[0]
			if pos.Cube.Owner != domain.None {
				pos.Cube.Owner = 1 - pos.Cube.Owner
			}
			newID, err := ps.Save(ctx, scope, pos)
			if err != nil {
				return fmt.Errorf("save swapped position: %w", err)
			}
			if newID == pid {
				continue // swap is a no-op for this position (self-mirrored)
			}
			if _, err := tx.ExecContext(ctx,
				`UPDATE move SET position_id = ?
				 WHERE position_id = ? AND game_id IN (SELECT id FROM game WHERE match_id = ?)`,
				newID, pid, id); err != nil {
				return fmt.Errorf("repoint swapped move: %w", err)
			}
			// Drop the original if this match was its last holder (mirrors the
			// orphan cleanup of DeleteCascade).
			var held bool
			if err := tx.QueryRowContext(ctx, positionIsHeldSQL, pid).Scan(&held); err != nil {
				return fmt.Errorf("swap orphan check: %w", err)
			}
			if !held {
				if _, err := tx.ExecContext(ctx, `DELETE FROM position WHERE id = ?`, pid); err != nil {
					return fmt.Errorf("delete swap orphan: %w", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: swap players for match %d: %w", id, err)
	}
	return nil
}

// MergePlayers rewrites every occurrence of the given player names (in both
// the player1 and player2 columns) to a single canonical name.
func (s *matchStore) MergePlayers(ctx context.Context, scope string, names []string, canonical string) error {
	if canonical == "" {
		return fmt.Errorf("sqlite: merge players: canonical name must not be empty: %w", storage.ErrInternal)
	}
	if len(names) == 0 {
		return fmt.Errorf("sqlite: merge players: no names to merge: %w", storage.ErrInternal)
	}
	placeholders := make([]string, len(names))
	args := make([]any, 0, len(names)+1)
	args = append(args, canonical)
	for i, n := range names {
		placeholders[i] = "?"
		args = append(args, n)
	}
	in := ""
	for i, p := range placeholders {
		if i > 0 {
			in += ", "
		}
		in += p
	}
	err := withTx(ctx, s.db, func(tx execer) error {
		if _, err := tx.ExecContext(ctx,
			`UPDATE match SET player1_name = ? WHERE player1_name IN (`+in+`)`, args...); err != nil {
			return fmt.Errorf("rename player1: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE match SET player2_name = ? WHERE player2_name IN (`+in+`)`, args...); err != nil {
			return fmt.Errorf("rename player2: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: merge players: %w", err)
	}
	return nil
}

// SetLastVisitedPosition records the last position index viewed in a match.
func (s *matchStore) SetLastVisitedPosition(ctx context.Context, scope string, id int64, positionIndex int) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE match SET last_visited_position = ? WHERE id = ?`, positionIndex, id); err != nil {
		return fmt.Errorf("sqlite: set last visited position for match %d: %w", id, err)
	}
	return nil
}

// LastVisited returns the most recently visited match, falling back to the
// most recent match when none has been visited, or ErrNotFound when no match
// is stored.
func (s *matchStore) LastVisited(ctx context.Context, scope string) (*domain.Match, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+matchSelectCols+` FROM match m
		 LEFT JOIN tournament t ON m.tournament_id = t.id
		 WHERE m.last_visited_position >= 0
		 ORDER BY m.import_date DESC LIMIT 1`)
	m, err := scanMatch(row)
	if err == nil {
		return &m, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sqlite: last visited match: %w", err)
	}

	row = s.db.QueryRowContext(ctx,
		`SELECT `+matchSelectCols+` FROM match m
		 LEFT JOIN tournament t ON m.tournament_id = t.id`+matchOrderClause+` LIMIT 1`)
	m, err = scanMatch(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("sqlite: last visited match: %w", storage.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("sqlite: last visited match: %w", err)
	}
	return &m, nil
}

const gameInsertSQL = `INSERT INTO game (
	match_id, game_number, initial_score_1, initial_score_2,
	winner, points_won, move_count
) VALUES (?,?,?,?,?,?,?)`

// CreateGame stores a new game and returns its id, updating g.ID in place.
func (s *matchStore) CreateGame(ctx context.Context, scope string, g *domain.Game) (int64, error) {
	res, err := s.db.ExecContext(ctx, gameInsertSQL,
		g.MatchID, g.GameNumber, g.InitialScore[0], g.InitialScore[1],
		g.Winner, g.PointsWon, g.MoveCount)
	if err != nil {
		return 0, fmt.Errorf("sqlite: create game: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sqlite: create game id: %w", err)
	}
	g.ID = id
	return id, nil
}

// Games streams the games of a match ordered by game number.
func (s *matchStore) Games(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.Game, error] {
	return func(yield func(*domain.Game, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT id, COALESCE(match_id,0), COALESCE(game_number,0),
			        COALESCE(initial_score_1,0), COALESCE(initial_score_2,0),
			        COALESCE(winner,0), COALESCE(points_won,0), COALESCE(move_count,0)
			 FROM game WHERE match_id = ? ORDER BY game_number`, matchID)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list games: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var g domain.Game
			var s1, s2 int32
			if err := rows.Scan(&g.ID, &g.MatchID, &g.GameNumber,
				&s1, &s2, &g.Winner, &g.PointsWon, &g.MoveCount); err != nil {
				yield(nil, fmt.Errorf("sqlite: list games: %w", err))
				return
			}
			g.InitialScore = [2]int32{s1, s2}
			if !yield(&g, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list games: %w", err))
		}
	}
}

const moveInsertSQL = `INSERT INTO move (
	game_id, move_number, move_type, position_id, player,
	dice_1, dice_2, checker_move, cube_action
) VALUES (?,?,?,?,?,?,?,?,?)`

// CreateMove stores a new move and returns its id, updating mv.ID in place. A
// zero PositionID is stored as NULL (no associated position).
func (s *matchStore) CreateMove(ctx context.Context, scope string, mv *domain.Move) (int64, error) {
	var positionID any
	if mv.PositionID != 0 {
		positionID = mv.PositionID
	}
	res, err := s.db.ExecContext(ctx, moveInsertSQL,
		mv.GameID, mv.MoveNumber, mv.MoveType, positionID, mv.Player,
		mv.Dice[0], mv.Dice[1], mv.CheckerMove, mv.CubeAction)
	if err != nil {
		return 0, fmt.Errorf("sqlite: create move: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sqlite: create move id: %w", err)
	}
	mv.ID = id
	return id, nil
}

// Moves streams the moves of a game ordered by move number.
func (s *matchStore) Moves(ctx context.Context, scope string, gameID int64) iter.Seq2[*domain.Move, error] {
	return func(yield func(*domain.Move, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT id, COALESCE(game_id,0), COALESCE(move_number,0), COALESCE(move_type,''),
			        position_id, COALESCE(player,0),
			        COALESCE(dice_1,0), COALESCE(dice_2,0),
			        COALESCE(checker_move,''), COALESCE(cube_action,'')
			 FROM move WHERE game_id = ? ORDER BY move_number`, gameID)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list moves: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var mv domain.Move
			var d1, d2 int32
			var positionID sql.NullInt64
			if err := rows.Scan(&mv.ID, &mv.GameID, &mv.MoveNumber, &mv.MoveType,
				&positionID, &mv.Player, &d1, &d2, &mv.CheckerMove, &mv.CubeAction); err != nil {
				yield(nil, fmt.Errorf("sqlite: list moves: %w", err))
				return
			}
			mv.Dice = [2]int32{d1, d2}
			if positionID.Valid {
				mv.PositionID = positionID.Int64
			}
			if !yield(&mv, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list moves: %w", err))
		}
	}
}

// MovesByMatch streams every move of a match in chronological order (by game,
// then move). One query instead of Games + a Moves call per game: callers
// regroup by Move.GameID. Single-tenant store, so scope is ignored (as in Moves).
func (s *matchStore) MovesByMatch(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.Move, error] {
	return func(yield func(*domain.Move, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT mv.id, COALESCE(mv.game_id,0), COALESCE(mv.move_number,0), COALESCE(mv.move_type,''),
			        mv.position_id, COALESCE(mv.player,0),
			        COALESCE(mv.dice_1,0), COALESCE(mv.dice_2,0),
			        COALESCE(mv.checker_move,''), COALESCE(mv.cube_action,'')
			 FROM move mv INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = ?
			 ORDER BY g.game_number, mv.move_number`, matchID)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list moves by match: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var mv domain.Move
			var d1, d2 int32
			var positionID sql.NullInt64
			if err := rows.Scan(&mv.ID, &mv.GameID, &mv.MoveNumber, &mv.MoveType,
				&positionID, &mv.Player, &d1, &d2, &mv.CheckerMove, &mv.CubeAction); err != nil {
				yield(nil, fmt.Errorf("sqlite: list moves by match: %w", err))
				return
			}
			mv.Dice = [2]int32{d1, d2}
			if positionID.Valid {
				mv.PositionID = positionID.Int64
			}
			if !yield(&mv, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list moves by match: %w", err))
		}
	}
}

// xgPlayerToBlunderDB maps the XG move-player encoding (1 / -1) stored in the
// move table to the blunderDB encoding (0 = player 1, 1 = player 2). GnuBG
// imports are stored already converted to the XG encoding.
func xgPlayerToBlunderDB(player int32) int32 {
	if player == 1 {
		return 0
	}
	return 1
}

// MovePositions streams every position of a match in chronological order,
// each carrying its game / move context.
func (s *matchStore) MovePositions(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.MatchMovePosition, error] {
	return func(yield func(*domain.MatchMovePosition, error) bool) {
		var player1Name, player2Name string
		err := s.db.QueryRowContext(ctx,
			`SELECT COALESCE(player1_name,''), COALESCE(player2_name,'')
			 FROM match WHERE id = ?`, matchID).Scan(&player1Name, &player2Name)
		if errors.Is(err, sql.ErrNoRows) {
			yield(nil, fmt.Errorf("sqlite: move positions for match %d: %w", matchID, storage.ErrNotFound))
			return
		}
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: move positions for match %d: %w", matchID, err))
			return
		}

		rows, err := s.db.QueryContext(ctx,
			`SELECT mv.id, COALESCE(mv.game_id,0), COALESCE(g.game_number,0), COALESCE(mv.move_number,0),
			        COALESCE(mv.move_type,''), COALESCE(mv.player,0), mv.position_id,
			        p.state, p.decision_type, p.player_on_roll, p.dice_1, p.dice_2,
			        p.cube_value, p.cube_owner, p.score_1, p.score_2,
			        p.has_jacoby, p.has_beaver,
			        COALESCE(mv.checker_move,''), COALESCE(mv.cube_action,'')
			 FROM move mv
			 INNER JOIN game g ON mv.game_id = g.id
			 INNER JOIN position p ON mv.position_id = p.id
			 WHERE g.match_id = ?
			 ORDER BY g.game_number, mv.move_number`, matchID)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: move positions for match %d: %w", matchID, err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var moveID, gameID, positionID int64
			var gameNumber, moveNumber, player int32
			var moveType, state, checkerMove, cubeAction string
			var dt, por, d1, d2, cv, co, s1, s2, hj, hb sql.NullInt64
			if err := rows.Scan(&moveID, &gameID, &gameNumber, &moveNumber,
				&moveType, &player, &positionID,
				&state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb,
				&checkerMove, &cubeAction); err != nil {
				yield(nil, fmt.Errorf("sqlite: move positions for match %d: %w", matchID, err))
				return
			}
			position := engine.ReconstructPosition(positionID, state,
				int(dt.Int64), int(por.Int64), int(d1.Int64), int(d2.Int64),
				int(cv.Int64), int(co.Int64), int(s1.Int64), int(s2.Int64),
				int(hj.Int64), int(hb.Int64))
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
			yield(nil, fmt.Errorf("sqlite: move positions for match %d: %w", matchID, err))
		}
	}
}
