package postgres

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"math"

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
	COALESCE(m.tournament_sort_order,0),
	COALESCE(m.match_hash,''), COALESCE(m.canonical_hash,'')`

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
		&m.MatchHash, &m.CanonicalHash,
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
	match_hash, canonical_hash, import_batch_id
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
RETURNING id, import_date`

// nullableID returns nil for a zero id so it is stored as SQL NULL — which is
// what a foreign key with ON DELETE SET NULL expects, and what "this match came
// in with no batch" means.
func nullableID(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

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
		nullableString(m.MatchHash), nullableString(m.CanonicalHash), nullableID(m.ImportBatchID)).Scan(&id, &importDate)
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
// when the match date is unset. Used by LastVisited; List builds its ORDER BY
// from domain.MatchOrderByClause so the key is configurable.
const matchOrderClause = ` ORDER BY COALESCE(m.match_date, m.import_date) DESC`

// buildMatchListWhere appends opts filters to the tenant scope. next is the next
// free placeholder number ($1 is the tenant), so the returned SQL starts with
// " AND …". Mirrors the SQLite builder; the two must stay in sync.
func buildMatchListWhere(opts storage.MatchListOpts, next int) (whereSQL string, args []any) {
	var clauses []string
	if opts.PlayerName != "" {
		clauses = append(clauses, fmt.Sprintf("(m.player1_name = $%d OR m.player2_name = $%d)", next, next+1))
		args = append(args, opts.PlayerName, opts.PlayerName)
		next += 2
	}
	if len(opts.TournamentIDs) > 0 {
		ph := make([]string, len(opts.TournamentIDs))
		for i, id := range opts.TournamentIDs {
			ph[i] = fmt.Sprintf("$%d", next)
			args = append(args, id)
			next++
		}
		clauses = append(clauses, "m.tournament_id IN ("+strings.Join(ph, ",")+")")
	}
	// Compare on the date part so an inclusive DateTo (e.g. a whole-year filter
	// "…-12-31") still matches a match timestamped later that same day.
	if opts.DateFrom != "" {
		clauses = append(clauses, fmt.Sprintf("m.match_date::date >= $%d", next))
		args = append(args, opts.DateFrom)
		next++
	}
	if opts.DateTo != "" {
		clauses = append(clauses, fmt.Sprintf("m.match_date::date <= $%d", next))
		args = append(args, opts.DateTo)
		next++
	}
	if len(opts.MatchLength) > 0 {
		ph := make([]string, len(opts.MatchLength))
		for i, ml := range opts.MatchLength {
			ph[i] = fmt.Sprintf("$%d", next)
			args = append(args, ml)
			next++
		}
		clauses = append(clauses, "m.match_length IN ("+strings.Join(ph, ",")+")")
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return " AND " + strings.Join(clauses, " AND "), args
}

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

// List streams stored matches, filtered/ordered/paginated per opts. A zero
// MatchListOpts streams every match, most recent first.
func (s *matchStore) List(ctx context.Context, scope string, opts storage.MatchListOpts) iter.Seq2[*domain.Match, error] {
	return func(yield func(*domain.Match, error) bool) {
		args := []any{tenantID(scope)}
		whereSQL, filterArgs := buildMatchListWhere(opts, len(args)+1)
		args = append(args, filterArgs...)
		query := `SELECT ` + matchSelectCols + ` FROM match m
			 LEFT JOIN tournament t ON m.tournament_id = t.id
			 WHERE m.tenant_id = $1` + whereSQL +
			` ORDER BY ` + domain.MatchOrderByClause(opts.Sort)
		if opts.Limit > 0 {
			args = append(args, opts.Limit)
			query += fmt.Sprintf(" LIMIT $%d", len(args))
		}
		if opts.Offset > 0 {
			args = append(args, opts.Offset)
			query += fmt.Sprintf(" OFFSET $%d", len(args))
		}
		rows, err := s.db.Query(ctx, query, args...)
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
// A position is held by: another match's move; membership in a collection; an
// Anki card built from it; a comment the USER wrote on it (#263); having been
// imported individually, which says the user brought it in deliberately
// (docs/adr/0001); or the study mark the source tool carried, since deleting a
// match must not delete the very positions the `fl` filter exists to surface
// (docs/adr/0006).
//
// Two things deliberately do NOT hold a position, because neither is evidence
// the user did anything with it:
//   - an analysis: it arrives with the match, and every match position has one,
//     so counting it would mean never purging anything;
//   - a comment that is not the user's: match importers attach the source
//     file's per-move notes as comments (see ingest/xg.go), and until 2.19.0
//     nothing told them apart from a note the user typed — so no comment held a
//     position at all, and a note the user had written was lost with the match.
//     A comment now carries its origin; only origin = 'user' holds. An imported
//     note, or one written before the column existed ('unknown'), still does
//     not, which leaves the rows of every older database judged as they always
//     were.
//
// Phrased as a WHERE-clause fragment correlated against the outer `position`
// row rather than a standalone query — see deleteOrphanedPositions, which
// embeds it directly into a set-based DELETE instead of running it once per
// candidate position.
const positionIsHeldSQL = `EXISTS (SELECT 1 FROM move               WHERE position_id = position.id AND tenant_id = position.tenant_id)
	                       OR EXISTS (SELECT 1 FROM collection_position WHERE position_id = position.id AND tenant_id = position.tenant_id)
	                       OR EXISTS (SELECT 1 FROM anki_card           WHERE position_id = position.id AND tenant_id = position.tenant_id)
	                       OR EXISTS (SELECT 1 FROM comment             WHERE position_id = position.id AND tenant_id = position.tenant_id AND origin = 'user')
	                       OR position.individually_imported
	                       OR position.flagged`

// deleteOrphanedPositions removes every position in ids (within tenant) that
// positionIsHeldSQL says nothing holds any more, as a single set-based DELETE
// rather than one EXISTS round-trip per id.
func deleteOrphanedPositions(ctx context.Context, tx execer, tenant int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids)+1)
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	args[len(ids)] = tenant
	query := fmt.Sprintf(`DELETE FROM position WHERE id IN (%s) AND tenant_id = $%d AND NOT (%s)`,
		strings.Join(placeholders, ","), len(ids)+1, positionIsHeldSQL)
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete orphaned positions: %w", err)
	}
	return nil
}

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

		if err := deleteOrphanedPositions(ctx, tx, tenant, positionIDs); err != nil {
			return err
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
		// Positions swap by copy-on-write, NOT in place (#107): a position is
		// deduplicated by Zobrist and may be shared with other matches, and its
		// score/cube are part of that hash. For each position this match uses, save
		// a swapped copy (Save recomputes the Zobrist and dedups) and repoint this
		// match's moves to it; the original stays intact for whoever else holds it.
		rows, err := tx.Query(ctx,
			`SELECT DISTINCT mv.position_id FROM move mv
			 INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = $1 AND mv.tenant_id = $2 AND mv.position_id IS NOT NULL`,
			id, tenant)
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
		// Every swap position loaded in one round trip (LoadByIDs) instead of
		// one Load per position (B.11, #179): a match's moves can reference
		// this position library's biggest table, and this swap used to query
		// it once per distinct position. Save (below) still runs once per
		// position — it is what recomputes each one's Zobrist hash and dedups
		// it against the rest of the library, and that decision is
		// irreducibly per-position.
		loaded, err := ps.LoadByIDs(ctx, scope, posIDs)
		if err != nil {
			return fmt.Errorf("load swap positions: %w", err)
		}
		byID := make(map[int64]*domain.Position, len(loaded))
		for i := range loaded {
			byID[loaded[i].ID] = &loaded[i]
		}
		// Positions this swap repointed away from: each is a delete candidate
		// (mirrors the orphan cleanup of DeleteCascade), collected here and
		// checked in one set-based DELETE after the loop rather than one
		// EXISTS round-trip per position — the repoints must all land first
		// anyway, since an orphan check run mid-loop could not see a later
		// position's repoint.
		var swappedAway []int64
		for _, pid := range posIDs {
			pos, ok := byID[pid]
			if !ok {
				return fmt.Errorf("load swap position %d: %w", pid, storage.ErrNotFound)
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
			if _, err := tx.Exec(ctx,
				`UPDATE move SET position_id = $1
				 WHERE position_id = $2 AND tenant_id = $3
				   AND game_id IN (SELECT id FROM game WHERE match_id = $4)`,
				newID, pid, tenant, id); err != nil {
				return fmt.Errorf("repoint swapped move: %w", err)
			}
			swappedAway = append(swappedAway, pid)
		}
		if err := deleteOrphanedPositions(ctx, tx, tenant, swappedAway); err != nil {
			return fmt.Errorf("swap orphan cleanup: %w", err)
		}
		return nil
	})
}

// MergePlayers rewrites every occurrence of the given player names (in both
// the player1 and player2 columns) to a single canonical name.
func (s *matchStore) MergePlayers(ctx context.Context, scope string, names []string, canonical string) error {
	if canonical == "" {
		return fmt.Errorf("postgres: merge players: canonical name must not be empty: %w", storage.ErrInvalid)
	}
	if len(names) == 0 {
		return fmt.Errorf("postgres: merge players: no names to merge: %w", storage.ErrInvalid)
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

// moveSelectCols reads a domain.Move (scanMove is its counterpart);
// moveSelectColsMV is the same list qualified with the alias mv for joins.
const moveSelectCols = `id, game_id, COALESCE(move_number,0), COALESCE(move_type,''),
	position_id, COALESCE(player,0), COALESCE(dice_1,0), COALESCE(dice_2,0),
	COALESCE(checker_move,''), COALESCE(cube_action,''), luck_mp`

const moveSelectColsMV = `mv.id, mv.game_id, COALESCE(mv.move_number,0), COALESCE(mv.move_type,''),
	mv.position_id, COALESCE(mv.player,0), COALESCE(mv.dice_1,0), COALESCE(mv.dice_2,0),
	COALESCE(mv.checker_move,''), COALESCE(mv.cube_action,''), mv.luck_mp`

func scanMove(sc scanner) (domain.Move, error) {
	var mv domain.Move
	var d1, d2 int32
	var positionID *int64
	var luckMP *int32
	if err := sc.Scan(&mv.ID, &mv.GameID, &mv.MoveNumber, &mv.MoveType,
		&positionID, &mv.Player, &d1, &d2, &mv.CheckerMove, &mv.CubeAction, &luckMP); err != nil {
		return domain.Move{}, err
	}
	mv.Dice = [2]int32{d1, d2}
	if positionID != nil {
		mv.PositionID = *positionID
	}
	mv.LuckMP = luckMP
	return mv, nil
}

// MovesByPositions — see storage.MatchStore.
func (s *matchStore) MovesByPositions(ctx context.Context, scope string, positionIDs []int64) (map[int64][]*domain.Move, error) {
	out := make(map[int64][]*domain.Move)
	if len(positionIDs) == 0 {
		return out, nil
	}
	rows, err := s.db.Query(ctx,
		`SELECT `+moveSelectCols+` FROM move WHERE position_id = ANY($1) AND tenant_id = $2 ORDER BY id`,
		positionIDs, tenantID(scope))
	if err != nil {
		return nil, fmt.Errorf("postgres: moves by positions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		mv, err := scanMove(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: moves by positions: %w", err)
		}
		out[mv.PositionID] = append(out[mv.PositionID], &mv)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: moves by positions: %w", err)
	}
	return out, nil
}

// CreateMoveAnalysis — see storage.MatchStore. The rate and equity columns
// are BIGINT here (schema 001), so the values are stored rounded: nothing
// writes fractional move analyses any more, and a copy of an old SQLite file
// carries them only as far as this rounding.
func (s *matchStore) CreateMoveAnalysis(ctx context.Context, scope string, ma *domain.MoveAnalysis) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO move_analysis (tenant_id, move_id, analysis_type, depth, equity, equity_error,
		    win_rate, gammon_rate, backgammon_rate, opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`,
		tenantID(scope), ma.MoveID, ma.AnalysisType, ma.Depth,
		int64(math.Round(ma.Equity)), int64(math.Round(ma.EquityError)),
		int64(math.Round(ma.WinRate)), int64(math.Round(ma.GammonRate)), int64(math.Round(ma.BackgammonRate)),
		int64(math.Round(ma.OpponentWinRate)), int64(math.Round(ma.OpponentGammonRate)), int64(math.Round(ma.OpponentBackgammonRate))).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("postgres: create move analysis: %w", err)
	}
	ma.ID = id
	return id, nil
}

// MoveAnalysesByMatch — see storage.MatchStore.
func (s *matchStore) MoveAnalysesByMatch(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.MoveAnalysis, error] {
	return func(yield func(*domain.MoveAnalysis, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT ma.id, ma.move_id, COALESCE(ma.analysis_type,''), COALESCE(ma.depth,''),
			        COALESCE(ma.equity,0), COALESCE(ma.equity_error,0),
			        COALESCE(ma.win_rate,0), COALESCE(ma.gammon_rate,0), COALESCE(ma.backgammon_rate,0),
			        COALESCE(ma.opponent_win_rate,0), COALESCE(ma.opponent_gammon_rate,0), COALESCE(ma.opponent_backgammon_rate,0)
			 FROM move_analysis ma
			 INNER JOIN move mv ON ma.move_id = mv.id
			 INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = $1 AND ma.tenant_id = $2
			 ORDER BY g.game_number, mv.move_number, ma.id`,
			matchID, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list move analyses: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var ma domain.MoveAnalysis
			var eq, eqErr, win, gam, bg, oWin, oGam, oBg int64
			if err := rows.Scan(&ma.ID, &ma.MoveID, &ma.AnalysisType, &ma.Depth,
				&eq, &eqErr, &win, &gam, &bg, &oWin, &oGam, &oBg); err != nil {
				yield(nil, fmt.Errorf("postgres: list move analyses: %w", err))
				return
			}
			ma.Equity, ma.EquityError = float64(eq), float64(eqErr)
			ma.WinRate, ma.GammonRate, ma.BackgammonRate = float64(win), float64(gam), float64(bg)
			ma.OpponentWinRate, ma.OpponentGammonRate, ma.OpponentBackgammonRate = float64(oWin), float64(oGam), float64(oBg)
			if !yield(&ma, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("postgres: list move analyses: %w", err))
		}
	}
}

const moveInsertSQL = `INSERT INTO move (
	tenant_id, game_id, move_number, move_type, position_id, player,
	dice_1, dice_2, checker_move, cube_action, luck_mp
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`

// CreateMove stores a new move and returns its id, updating mv.ID in place. A
// zero PositionID is stored as NULL (no associated position).
func (s *matchStore) CreateMove(ctx context.Context, scope string, mv *domain.Move) (int64, error) {
	var positionID any
	if mv.PositionID != 0 {
		positionID = mv.PositionID
	}
	var luckMP any
	if mv.LuckMP != nil {
		luckMP = *mv.LuckMP
	}
	var id int64
	err := s.db.QueryRow(ctx, moveInsertSQL,
		tenantID(scope), mv.GameID, mv.MoveNumber, mv.MoveType, positionID, mv.Player,
		mv.Dice[0], mv.Dice[1], mv.CheckerMove, mv.CubeAction, luckMP).Scan(&id)
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
			`SELECT `+moveSelectCols+` FROM move WHERE game_id = $1 AND tenant_id = $2
			 ORDER BY move_number`,
			gameID, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list moves: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			mv, err := scanMove(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: list moves: %w", err))
				return
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
			`SELECT `+moveSelectColsMV+`
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
			mv, err := scanMove(rows)
			if err != nil {
				yield(nil, fmt.Errorf("postgres: list moves by match: %w", err))
				return
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
