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
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
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
	COALESCE(m.last_visited_position,-1), COALESCE(m.comment,''), COALESCE(m.comment_author,''),
	COALESCE(m.tournament_sort_order,0),
	COALESCE(m.match_hash,''), COALESCE(m.canonical_hash,''), COALESCE(m.dice_hash,''),
	m.player1_elo, m.player2_elo, m.player1_experience, m.player2_experience,
	COALESCE(m.transcriber,''), m.has_jacoby, m.has_beaver, COALESCE(m.engine_version,'')`

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
		&m.LastVisitedPosition, &m.Comment, &m.CommentAuthor,
		&m.TournamentSortOrder,
		&m.MatchHash, &m.CanonicalHash, &m.DiceHash,
		&m.Player1Elo, &m.Player2Elo, &m.Player1Experience, &m.Player2Experience,
		&m.Transcriber, &m.HasJacoby, &m.HasBeaver, &m.EngineVersion,
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
	match_length, match_date, file_path, game_count, tournament_id, comment, comment_author,
	match_hash, canonical_hash, import_batch_id, dice_hash,
	player1_elo, player2_elo, player1_experience, player2_experience,
	transcriber, has_jacoby, has_beaver, engine_version
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
	$17,$18,$19,$20,$21,$22,$23,$24,$25)
RETURNING id, import_date`

// sourceMetadataArgs are the eight source-metadata columns in the order
// matchInsertSQL and ReplaceHeader list them. A nil pointer stays NULL.
func sourceMetadataArgs(m *domain.Match) []any {
	return []any{m.Player1Elo, m.Player2Elo, m.Player1Experience, m.Player2Experience,
		m.Transcriber, m.HasJacoby, m.HasBeaver, m.EngineVersion}
}

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

// nullableUnix binds a Unix-seconds date (ADR-0071), 0 meaning unset, to a
// TIMESTAMPTZ column: PostgreSQL already stores that type as an integer
// count in UTC.
func nullableUnix(n int64) any {
	if n == 0 {
		return nil
	}
	return time.Unix(n, 0).UTC()
}

// Save stores a new match and returns its id, updating m.ID and m.ImportDate
// in place.
func (s *matchStore) Save(ctx context.Context, scope string, m *domain.Match) (int64, error) {
	var id int64
	var importDate time.Time
	args := append([]any{
		tenantID(scope), m.Player1Name, m.Player2Name, m.Event, m.Location, m.Round,
		m.MatchLength, nullableTime(m.MatchDate), m.FilePath, m.GameCount,
		m.TournamentID, m.Comment, m.CommentAuthor,
		nullableString(m.MatchHash), nullableString(m.CanonicalHash), nullableID(m.ImportBatchID),
		nullableString(m.DiceHash)}, sourceMetadataArgs(m)...)
	err := s.db.QueryRow(ctx, matchInsertSQL, args...).Scan(&id, &importDate)
	if err != nil {
		return 0, fmt.Errorf("postgres: save match: %w", referenced(err))
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
// playerFilterNames are the names opts.PlayerName stands for: its spellings
// when the store resolved them, the name alone otherwise.
func playerFilterNames(opts storage.MatchListOpts) []string {
	if opts.PlayerName == "" {
		return nil
	}
	if len(opts.PlayerSpellings) > 0 {
		return opts.PlayerSpellings
	}
	return []string{opts.PlayerName}
}

// withPlayerSpellings resolves opts.PlayerName to every spelling of the
// person through the tenant's alias table.
func (s *matchStore) withPlayerSpellings(ctx context.Context, scope string, opts storage.MatchListOpts) (storage.MatchListOpts, error) {
	names, err := sqlshared.PlayerSpellings(ctx, binder{s.db}.shared(), scope, opts.PlayerName)
	if err != nil {
		return opts, fmt.Errorf("postgres: match list aliases: %w", err)
	}
	opts.PlayerSpellings = names
	return opts, nil
}

func buildMatchListWhere(opts storage.MatchListOpts, next int) (whereSQL string, args []any) {
	var clauses []string
	if names := playerFilterNames(opts); len(names) > 0 {
		clauses = append(clauses, fmt.Sprintf("(m.player1_name = ANY($%d) OR m.player2_name = ANY($%d))", next, next))
		args = append(args, names)
		next++
	}
	if opts.PlayerNameContains != "" {
		clauses = append(clauses, fmt.Sprintf(`(m.player1_name ILIKE $%d ESCAPE '\' OR m.player2_name ILIKE $%d ESCAPE '\')`, next, next+1))
		pat := sqlshared.ContainsPattern(opts.PlayerNameContains)
		args = append(args, pat, pat)
		next += 2
	}
	if opts.Text != "" {
		clauses = append(clauses, fmt.Sprintf(`(m.player1_name ILIKE $%[1]d ESCAPE '\' OR m.player2_name ILIKE $%[1]d ESCAPE '\'
			OR m.event ILIKE $%[1]d ESCAPE '\' OR m.location ILIKE $%[1]d ESCAPE '\' OR m.round ILIKE $%[1]d ESCAPE '\'
			OR t.name ILIKE $%[1]d ESCAPE '\' OR CAST(m.match_date::date AS TEXT) ILIKE $%[1]d ESCAPE '\'
			OR CAST(m.match_length AS TEXT) ILIKE $%[1]d ESCAPE '\')`, next))
		args = append(args, sqlshared.ContainsPattern(opts.Text))
		next++
	}
	if opts.Unassigned {
		clauses = append(clauses, "m.tournament_id IS NULL")
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

// Count returns how many matches satisfy the filters of opts.
func (s *matchStore) Count(ctx context.Context, scope string, opts storage.MatchListOpts) (int, error) {
	opts, err := s.withPlayerSpellings(ctx, scope, opts)
	if err != nil {
		return 0, err
	}
	args := []any{tenantID(scope)}
	whereSQL, filterArgs := buildMatchListWhere(opts, len(args)+1)
	args = append(args, filterArgs...)
	var n int
	err = s.db.QueryRow(ctx, `SELECT COUNT(*) FROM match m
		 LEFT JOIN tournament t ON m.tournament_id = t.id
		 WHERE m.tenant_id = $1`+whereSQL, args...).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("postgres: count matches: %w", err)
	}
	return n, nil
}

// List streams stored matches, filtered/ordered/paginated per opts. A zero
// MatchListOpts streams every match, most recent first.
func (s *matchStore) List(ctx context.Context, scope string, opts storage.MatchListOpts) iter.Seq2[*domain.Match, error] {
	return func(yield func(*domain.Match, error) bool) {
		opts, err := s.withPlayerSpellings(ctx, scope, opts)
		if err != nil {
			yield(nil, err)
			return
		}
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
	tenant := tenantID(scope)
	return s.inTx(ctx, fmt.Sprintf("update match %d", id), func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`UPDATE match SET player1_name = $1, player2_name = $2, match_date = $3
			 WHERE id = $4 AND tenant_id = $5`,
			player1Name, player2Name, dateVal, id, tenant); err != nil {
			return err
		}
		return refreshMatchPositionDates(ctx, tx, tenant, id)
	})
}

// ReplaceHeader rewrites a match's header columns in place — see
// storage.MatchStore. The hashes go in through nullableString for the same
// reason Save does it: their UNIQUE index counts two empty strings as a
// duplicate, where two NULLs are two unknowns.
func (s *matchStore) ReplaceHeader(ctx context.Context, scope string, id int64, m *domain.Match) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE match SET player1_name = $1, player2_name = $2, event = $3, location = $4,
		                  round = $5, match_length = $6, match_date = $7, game_count = $8,
		                  match_hash = $9, canonical_hash = $10, dice_hash = $11,
		                  player1_elo = $12, player2_elo = $13, player1_experience = $14,
		                  player2_experience = $15, transcriber = $16, has_jacoby = $17,
		                  has_beaver = $18, engine_version = $19
		 WHERE id = $20 AND tenant_id = $21`,
		append(append([]any{m.Player1Name, m.Player2Name, m.Event, m.Location,
			m.Round, m.MatchLength, nullableTime(m.MatchDate), m.GameCount,
			nullableString(m.MatchHash), nullableString(m.CanonicalHash), nullableString(m.DiceHash)},
			sourceMetadataArgs(m)...), id, tenantID(scope))...)
	if err != nil {
		return fmt.Errorf("postgres: replace match %d header: %w", id, err)
	}
	// A replacement names its match: none there means the caller holds a
	// stale id, which must not pass for a rewrite.
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("postgres: replace match %d header: %w", id, storage.ErrNotFound)
	}
	if err := refreshMatchPositionDates(ctx, s.db, tenantID(scope), id); err != nil {
		return fmt.Errorf("postgres: replace match %d header: %w", id, err)
	}
	return nil
}

// UpdateComment sets the free-text comment on a match, signed by the
// context's comment author (storage.WithCommentAuthor).
func (s *matchStore) UpdateComment(ctx context.Context, scope string, id int64, comment string) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE match SET comment = $1, comment_author = $2 WHERE id = $3 AND tenant_id = $4`,
		comment, storage.CommentAuthorFromContext(ctx), id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: update match %d comment: %w", id, err)
	}
	return nil
}

// positionIsHeldSQL reports whether anything still holds a position once the
// match that referenced it is gone: deleting a match must not destroy the
// user's work on a position that merely occurred in it. The predicate is
// stated three times — database/db_match.go, storage/sqlite, storage/postgres —
// and must stay identical in all three.
//
// A position is held by: another match's move; a collection membership; an
// Anki card; a Lesson step (ADR-0066); a comment with origin = 'user'; the
// individually_imported mark
// (ADR-0001); or the source tool's study mark (ADR-0006).
//
// Deliberately NOT held by an analysis (every match position has one, so
// nothing would ever be purged), nor by an imported or 'unknown'-origin
// comment (the source file's per-move notes, see ingest/xg.go).
//
// A WHERE fragment correlated against the outer `position` row, embedded in
// deleteOrphanedPositions' set-based DELETE.
const positionIsHeldSQL = `EXISTS (SELECT 1 FROM move               WHERE position_id = position.id AND tenant_id = position.tenant_id)
	                       OR EXISTS (SELECT 1 FROM collection_position WHERE position_id = position.id AND tenant_id = position.tenant_id)
	                       OR EXISTS (SELECT 1 FROM anki_card           WHERE position_id = position.id AND tenant_id = position.tenant_id)
	                       OR EXISTS (SELECT 1 FROM lesson_step         WHERE position_id = position.id AND tenant_id = position.tenant_id)
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

		if _, err := tx.Exec(ctx, `DELETE FROM match_stats WHERE match_id = $1 AND tenant_id = $2`, id, tenant); err != nil {
			return fmt.Errorf("invalidate match stats: %w", err)
		}
		// game/move/move_analysis cascade off the match delete.
		if _, err := tx.Exec(ctx,
			`DELETE FROM match WHERE id = $1 AND tenant_id = $2`, id, tenant); err != nil {
			return err
		}

		if err := deleteOrphanedPositions(ctx, tx, tenant, positionIDs); err != nil {
			return err
		}
		return refreshPositionMatchDates(ctx, tx, tenant, positionIDs)
	})
}

// DeleteGames removes a match's games (moves and move analyses cascade off
// them) and returns the positions they referenced, without touching the match
// row — see storage.MatchStore. The collection query is DeleteCascade's, run
// before the delete for the same reason: the moves that name the positions are
// about to be gone.
func (s *matchStore) DeleteGames(ctx context.Context, scope string, matchID int64) ([]int64, error) {
	tenant := tenantID(scope)
	var positionIDs []int64
	err := s.inTx(ctx, fmt.Sprintf("delete games of match %d", matchID), func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT DISTINCT mv.position_id
			 FROM move mv INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = $1 AND mv.tenant_id = $2 AND mv.position_id IS NOT NULL`,
			matchID, tenant)
		if err != nil {
			return fmt.Errorf("collect positions: %w", err)
		}
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

		if _, err := tx.Exec(ctx, `DELETE FROM match_stats WHERE match_id = $1 AND tenant_id = $2`, matchID, tenant); err != nil {
			return fmt.Errorf("invalidate match stats: %w", err)
		}
		// move/move_analysis cascade off the game delete.
		if _, err := tx.Exec(ctx,
			`DELETE FROM game WHERE match_id = $1 AND tenant_id = $2`, matchID, tenant); err != nil {
			return err
		}
		return refreshPositionMatchDates(ctx, tx, tenant, positionIDs)
	})
	if err != nil {
		return nil, err
	}
	return positionIDs, nil
}

// PurgeOrphanPositions drops the positions in ids that positionIsHeldSQL no
// longer holds — see storage.MatchStore. It is deleteOrphanedPositions, the
// half of DeleteCascade a replacement needs on its own, once the new rows are
// written.
func (s *matchStore) PurgeOrphanPositions(ctx context.Context, scope string, ids []int64) error {
	if err := deleteOrphanedPositions(ctx, s.db, tenantID(scope), ids); err != nil {
		return fmt.Errorf("postgres: purge orphan positions: %w", err)
	}
	return nil
}

// SwapPlayers swaps player 1 and player 2 for the match: it swaps the header
// names, the per-game scores and winner, the per-move player, and the score /
// cube-owner columns of every position the match's moves reference.
func (s *matchStore) SwapPlayers(ctx context.Context, scope string, id int64) error {
	tenant := tenantID(scope)
	return s.inTx(ctx, "swap players", func(tx pgx.Tx) error {
		// The seats trade places: each row now describes the other player.
		if _, err := tx.Exec(ctx, `DELETE FROM match_stats WHERE match_id = $1 AND tenant_id = $2`, id, tenant); err != nil {
			return fmt.Errorf("invalidate match stats: %w", err)
		}
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
		// Positions swap by copy-on-write, NOT in place: a position may be
		// shared with other matches and its score/cube are part of the Zobrist
		// hash. Save a swapped copy (Save dedups) and repoint this match's moves.
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
		// Loaded in one round trip; Save below stays per-position, since the
		// Zobrist dedup decision is.
		loaded, err := ps.LoadByIDs(ctx, scope, posIDs)
		if err != nil {
			return fmt.Errorf("load swap positions: %w", err)
		}
		byID := make(map[int64]*domain.Position, len(loaded))
		for i := range loaded {
			byID[loaded[i].ID] = &loaded[i]
		}
		// Positions repointed away from are orphan candidates, checked in one
		// DELETE after the loop: a mid-loop check could not see later repoints.
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

// MergePlayers records every other name as an alias of canonical
// (storage.AliasStore.Set): the matches keep the names their files wrote, and
// the stats, the players table, the search and every later import read them
// as canonical.
func (s *matchStore) MergePlayers(ctx context.Context, scope string, names []string, canonical string) error {
	return sqlshared.MergeAliases(ctx, shared(binder{s.db}), scope, storage.AliasPlayer, names, canonical)
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
		return 0, fmt.Errorf("postgres: create game: %w", referenced(err))
	}
	g.ID = id
	// The match gains moves: its rows no longer cover all of them.
	if _, err := s.db.Exec(ctx, `DELETE FROM match_stats WHERE match_id = $1 AND tenant_id = $2`, g.MatchID, tenantID(scope)); err != nil {
		return 0, fmt.Errorf("postgres: invalidate match stats: %w", err)
	}
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
var moveSelectCols = `id, game_id, COALESCE(move_number,0), ` + sqlshared.TenantActionLabelOrEmptySQL("move.move_type") + `,
	position_id, COALESCE(player,0), COALESCE(dice_1,0), COALESCE(dice_2,0),
	COALESCE(checker_move,''), ` + sqlshared.TenantActionLabelOrEmptySQL("move.cube_action") + `, luck_mp,
	decision_ms, cube_decision_ms`

var moveSelectColsMV = `mv.id, mv.game_id, COALESCE(mv.move_number,0), ` + sqlshared.TenantActionLabelOrEmptySQL("mv.move_type") + `,
	mv.position_id, COALESCE(mv.player,0), COALESCE(mv.dice_1,0), COALESCE(mv.dice_2,0),
	COALESCE(mv.checker_move,''), ` + sqlshared.TenantActionLabelOrEmptySQL("mv.cube_action") + `, mv.luck_mp,
	mv.decision_ms, mv.cube_decision_ms`

func scanMove(sc scanner) (domain.Move, error) {
	var mv domain.Move
	var d1, d2 int32
	var positionID *int64
	var luckMP *int32
	if err := sc.Scan(&mv.ID, &mv.GameID, &mv.MoveNumber, &mv.MoveType,
		&positionID, &mv.Player, &d1, &d2, &mv.CheckerMove, &mv.CubeAction, &luckMP,
		&mv.DecisionMS, &mv.CubeDecisionMS); err != nil {
		return domain.Move{}, err
	}
	mv.Dice = [2]int32{d1, d2}
	if positionID != nil {
		mv.PositionID = *positionID
	}
	mv.LuckMP = luckMP
	return mv, nil
}

// scanScoredMove reads moveSelectColsMV followed by the Position's analysis
// blob, and scores the play from it.
func scanScoredMove(rows pgx.Rows, scorer sqlshared.PlayScorer) (domain.Move, error) {
	var mv domain.Move
	var d1, d2 int32
	var positionID *int64
	var data []byte
	if err := rows.Scan(&mv.ID, &mv.GameID, &mv.MoveNumber, &mv.MoveType,
		&positionID, &mv.Player, &d1, &d2, &mv.CheckerMove, &mv.CubeAction, &mv.LuckMP,
		&mv.DecisionMS, &mv.CubeDecisionMS, &data); err != nil {
		return domain.Move{}, err
	}
	mv.Dice = [2]int32{d1, d2}
	if positionID != nil {
		mv.PositionID = *positionID
	}
	scorer.Score(&mv, data)
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
		return 0, fmt.Errorf("postgres: create move analysis: %w", referenced(err))
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
	dice_1, dice_2, checker_move, cube_action, luck_mp,
	decision_ms, cube_decision_ms
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING id`

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
	moveType, err := actionCode(ctx, s.db, tenantID(scope), mv.MoveType)
	if err != nil {
		return 0, fmt.Errorf("postgres: create move: %w", err)
	}
	cubeAction, err := actionCode(ctx, s.db, tenantID(scope), mv.CubeAction)
	if err != nil {
		return 0, fmt.Errorf("postgres: create move: %w", err)
	}
	var id int64
	err = s.db.QueryRow(ctx, moveInsertSQL,
		tenantID(scope), mv.GameID, mv.MoveNumber, moveType, positionID, mv.Player,
		mv.Dice[0], mv.Dice[1], mv.CheckerMove, cubeAction, luckMP,
		sqlshared.MSArg(mv.DecisionMS), sqlshared.MSArg(mv.CubeDecisionMS)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("postgres: create move: %w", referenced(err))
	}
	mv.ID = id
	if positionID != nil {
		if _, err := s.db.Exec(ctx, positionMatchDateOnMoveSQL, mv.GameID, positionID, tenantID(scope)); err != nil {
			return 0, fmt.Errorf("postgres: date position of move: %w", err)
		}
	}
	return id, nil
}

// positionMatchDateOnMoveSQL lowers position.match_date to the date of the
// match a new move ties the position to; see the SQLite twin for why it
// compares instead of taking the MIN over the position's moves.
const positionMatchDateOnMoveSQL = `UPDATE position SET match_date = d.md
	FROM (SELECT m.match_date AS md FROM game g
	        JOIN match m ON m.id = g.match_id AND m.tenant_id = g.tenant_id
	       WHERE g.id = $1 AND g.tenant_id = $3) AS d
	WHERE position.id = $2 AND position.tenant_id = $3 AND d.md IS NOT NULL
	  AND (position.match_date IS NULL OR position.match_date > d.md)`

// refreshPositionMatchDates recomputes position.match_date for ids from every
// match that still reaches them: the exact form, for the rare edits that can
// raise the date (a match deleted, its date changed, its games replaced).
func refreshPositionMatchDates(ctx context.Context, db execer, tenant int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := db.Exec(ctx, `UPDATE position SET match_date =
		(SELECT MIN(m.match_date) FROM move mv
		   JOIN game g  ON g.id = mv.game_id AND g.tenant_id = mv.tenant_id
		   JOIN match m ON m.id = g.match_id AND m.tenant_id = g.tenant_id
		  WHERE mv.position_id = position.id AND mv.tenant_id = position.tenant_id)
		WHERE tenant_id = $1 AND id = ANY($2)`, tenant, ids)
	if err != nil {
		return fmt.Errorf("refresh position match dates: %w", err)
	}
	return nil
}

// refreshMatchPositionDates re-dates every position matchID reaches.
func refreshMatchPositionDates(ctx context.Context, db execer, tenant, matchID int64) error {
	_, err := db.Exec(ctx, `UPDATE position SET match_date =
		(SELECT MIN(m.match_date) FROM move mv
		   JOIN game g  ON g.id = mv.game_id AND g.tenant_id = mv.tenant_id
		   JOIN match m ON m.id = g.match_id AND m.tenant_id = g.tenant_id
		  WHERE mv.position_id = position.id AND mv.tenant_id = position.tenant_id)
		WHERE tenant_id = $1 AND id IN (SELECT mv.position_id FROM move mv
		   JOIN game g ON g.id = mv.game_id AND g.tenant_id = mv.tenant_id
		  WHERE g.match_id = $2 AND mv.tenant_id = $1 AND mv.position_id IS NOT NULL)`, tenant, matchID)
	if err != nil {
		return fmt.Errorf("refresh position match dates: %w", err)
	}
	return nil
}

// Moves streams the moves of a game ordered by move number.
func (s *matchStore) Moves(ctx context.Context, scope string, gameID int64) iter.Seq2[*domain.Move, error] {
	return func(yield func(*domain.Move, error) bool) {
		rows, err := s.db.Query(ctx,
			`SELECT `+moveSelectColsMV+`, a.data
			 FROM move mv LEFT JOIN analysis a
			   ON a.position_id = mv.position_id AND a.tenant_id = mv.tenant_id
			 WHERE mv.game_id = $1 AND mv.tenant_id = $2
			 ORDER BY mv.move_number`,
			gameID, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list moves: %w", err))
			return
		}
		defer rows.Close()
		scorer := sqlshared.PlayScorer{}
		for rows.Next() {
			mv, err := scanScoredMove(rows, scorer)
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
			`SELECT `+moveSelectColsMV+`, a.data
			 FROM move mv INNER JOIN game g ON mv.game_id = g.id
			 LEFT JOIN analysis a
			   ON a.position_id = mv.position_id AND a.tenant_id = mv.tenant_id
			 WHERE g.match_id = $1 AND mv.tenant_id = $2
			 ORDER BY g.game_number, mv.move_number`,
			matchID, tenantID(scope))
		if err != nil {
			yield(nil, fmt.Errorf("postgres: list moves by match: %w", err))
			return
		}
		defer rows.Close()
		scorer := sqlshared.PlayScorer{}
		for rows.Next() {
			mv, err := scanScoredMove(rows, scorer)
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
			        `+sqlshared.TenantActionLabelOrEmptySQL("mv.move_type")+`, COALESCE(mv.player,0), mv.position_id,
			        p.state, p.decision_type, p.player_on_roll, p.dice_1, p.dice_2,
			        p.cube_value, p.cube_owner, p.score_1, p.score_2,
			        p.has_jacoby, p.has_beaver, p.max_cube,
			        COALESCE(mv.checker_move,''), `+sqlshared.TenantActionLabelOrEmptySQL("mv.cube_action")+`,
			        mv.decision_ms, mv.cube_decision_ms
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
			var moveType, checkerMove, cubeAction string
			var state []byte
			var dt, por, d1, d2, cv, co, s1, s2 *int64
			var hj, hb *bool
			var mc, decisionMS, cubeDecisionMS *int64
			if err := rows.Scan(&moveID, &gameID, &gameNumber, &moveNumber,
				&moveType, &player, &positionID,
				&state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb, &mc,
				&checkerMove, &cubeAction, &decisionMS, &cubeDecisionMS); err != nil {
				yield(nil, fmt.Errorf("postgres: move positions for match %d: %w", matchID, err))
				return
			}
			position := engine.ReconstructPosition(positionID, string(state),
				derefInt(dt), derefInt(por), derefInt(d1), derefInt(d2),
				derefInt(cv), derefInt(co), derefInt(s1), derefInt(s2),
				boolToIntPtr(hj), boolToIntPtr(hb))
			position.MaxCube = derefInt(mc)
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

				DecisionMS:     decisionMS,
				CubeDecisionMS: cubeDecisionMS,
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

// ListByDiceHash returns the tenant's matches sharing dice_hash — see
// storage.MatchStore.
func (s *matchStore) ListByDiceHash(ctx context.Context, scope string, hash string) ([]domain.Match, error) {
	if hash == "" {
		return nil, nil
	}
	rows, err := s.db.Query(ctx, `SELECT `+matchSelectCols+` FROM match m
		 LEFT JOIN tournament t ON m.tournament_id = t.id
		 WHERE m.tenant_id = $1 AND m.dice_hash = $2 ORDER BY m.id`, tenantID(scope), hash)
	if err != nil {
		return nil, fmt.Errorf("postgres: list matches by dice hash: %w", err)
	}
	defer rows.Close()
	var out []domain.Match
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: list matches by dice hash: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SetDiceHash stores a match's dice_hash — see storage.MatchStore.
func (s *matchStore) SetDiceHash(ctx context.Context, scope string, id int64, hash string) error {
	if _, err := s.db.Exec(ctx, `UPDATE match SET dice_hash = $1 WHERE id = $2 AND tenant_id = $3`,
		nullableString(hash), id, tenantID(scope)); err != nil {
		return fmt.Errorf("postgres: set match %d dice hash: %w", id, err)
	}
	return nil
}

// DiceSequences streams the tenant's matches with their dice — see
// storage.MatchStore.
func (s *matchStore) DiceSequences(ctx context.Context, scope string) iter.Seq2[storage.MatchDice, error] {
	return func(yield func(storage.MatchDice, error) bool) {
		rows, err := s.db.Query(ctx, fmt.Sprintf(sqlshared.DiceSequencesSQL, "m.tenant_id = $1"), tenantID(scope))
		if err != nil {
			yield(storage.MatchDice{}, fmt.Errorf("postgres: match dice: %w", err))
			return
		}
		defer rows.Close()
		var f storage.DiceFolder
		for rows.Next() {
			var r storage.DiceRow
			if err := rows.Scan(&r.MatchID, &r.Player1, &r.Player2, &r.Length, &r.Hash,
				&r.GameID, &r.Score1, &r.Score2, &r.D1, &r.D2); err != nil {
				yield(storage.MatchDice{}, fmt.Errorf("postgres: match dice: %w", err))
				return
			}
			if m, ok := f.Add(r); ok && !yield(m, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(storage.MatchDice{}, fmt.Errorf("postgres: match dice: %w", err))
			return
		}
		if m, ok := f.Flush(); ok {
			yield(m, nil)
		}
	}
}

// ScoreMoves — see storage.MatchStore.
func (s *matchStore) ScoreMoves(ctx context.Context, scope string, after int64, limit int) (int64, int, error) {
	return sqlshared.ScoreMoves(ctx, binder{s.db}.shared(), scope, after, limit)
}

// RescorePositionMoves — see storage.MatchStore.
func (s *matchStore) RescorePositionMoves(ctx context.Context, scope string, positionID int64) error {
	return sqlshared.RescorePositionMoves(ctx, binder{s.db}.shared(), scope, positionID)
}
