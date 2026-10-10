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
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
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
	COALESCE(m.last_visited_position,-1), COALESCE(m.comment,''), COALESCE(m.comment_author,''),
	COALESCE(m.tournament_sort_order,0),
	COALESCE(m.match_hash,''), COALESCE(m.canonical_hash,''), COALESCE(m.dice_hash,''),
	m.player1_elo, m.player2_elo, m.player1_experience, m.player2_experience,
	COALESCE(m.transcriber,''), m.has_jacoby, m.has_beaver, COALESCE(m.engine_version,''), m.video_source`

// scanMatch reconstructs a domain.Match from a row selected with
// matchSelectCols. match_date and tournament_id are nullable.
func scanMatch(sc interface{ Scan(...any) error }) (domain.Match, error) {
	var m domain.Match
	var matchDate, importDate sql.NullTime
	var tournamentID sql.NullInt64
	var elo1, elo2 sql.NullFloat64
	var exp1, exp2, jacoby, beaver sql.NullInt64
	var video sql.NullString
	if err := sc.Scan(
		&m.ID, &m.Player1Name, &m.Player2Name,
		&m.Event, &m.Location, &m.Round,
		&m.MatchLength, &matchDate, &importDate,
		&m.FilePath, &m.GameCount,
		&tournamentID, &m.TournamentName,
		&m.LastVisitedPosition, &m.Comment, &m.CommentAuthor,
		&m.TournamentSortOrder,
		&m.MatchHash, &m.CanonicalHash, &m.DiceHash,
		&elo1, &elo2, &exp1, &exp2,
		&m.Transcriber, &jacoby, &beaver, &m.EngineVersion, &video,
	); err != nil {
		return domain.Match{}, err
	}
	m.Player1Elo, m.Player2Elo = nullFloat(elo1), nullFloat(elo2)
	m.Player1Experience, m.Player2Experience = nullInt(exp1), nullInt(exp2)
	m.HasJacoby, m.HasBeaver = nullBool(jacoby), nullBool(beaver)
	if video.Valid {
		m.VideoSource = &video.String
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
	match_length, match_date, file_path, game_count, tournament_id, comment, comment_author,
	match_hash, canonical_hash, import_batch_id, dice_hash,
	player1_elo, player2_elo, player1_experience, player2_experience,
	transcriber, has_jacoby, has_beaver, engine_version, video_source
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,NULLIF(?, ''))`

func nullFloat(v sql.NullFloat64) *float64 {
	if !v.Valid {
		return nil
	}
	return &v.Float64
}

func nullInt(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

func nullBool(v sql.NullInt64) *bool {
	if !v.Valid {
		return nil
	}
	b := v.Int64 != 0
	return &b
}

// sourceMetadataArgs are the eight source-metadata columns, then the video
// source, in the order matchInsertSQL and ReplaceHeader list them. A nil
// pointer stays NULL, except the video source, which ReplaceHeader keeps when
// nil and clears when "" (storage.MatchStore).
func sourceMetadataArgs(m *domain.Match) []any {
	return []any{m.Player1Elo, m.Player2Elo, m.Player1Experience, m.Player2Experience,
		m.Transcriber, m.HasJacoby, m.HasBeaver, m.EngineVersion, m.VideoSource}
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

// nullableString returns nil for an empty string so it is stored as SQL NULL.
// This matters for canonical_hash, whose UNIQUE index would otherwise reject a
// second match with an empty (non-NULL) hash.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// nullableUnix binds a Unix-seconds date column (ADR-0071), 0 meaning unset.
func nullableUnix(n int64) any {
	if n == 0 {
		return nil
	}
	return n
}

// Save stores a new match and returns its id, updating m.ID and m.ImportDate
// in place.
func (s *matchStore) Save(ctx context.Context, scope string, m *domain.Match) (int64, error) {
	args := append([]any{
		m.Player1Name, m.Player2Name, m.Event, m.Location, m.Round,
		m.MatchLength, nullableTime(m.MatchDate), m.FilePath, m.GameCount,
		m.TournamentID, m.Comment, m.CommentAuthor,
		nullableString(m.MatchHash), nullableString(m.CanonicalHash), nullableID(m.ImportBatchID),
		nullableString(m.DiceHash)}, sourceMetadataArgs(m)...)
	res, err := s.db.ExecContext(ctx, matchInsertSQL, args...)
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

// matchReinstateSQL is matchInsertSQL with the id and the import date stated.
var matchReinstateSQL = strings.Replace(strings.Replace(matchInsertSQL,
	"INSERT INTO match (", "INSERT INTO match (id, import_date, ", 1),
	"VALUES (", "VALUES (?, COALESCE(?, CURRENT_TIMESTAMP), ", 1)

// Reinstate stores m under its own id and import date — see
// storage.MatchStore. AUTOINCREMENT never issues an id twice, and only an id
// it issued is taken back, so nothing collides later.
func (s *matchStore) Reinstate(ctx context.Context, scope string, m *domain.Match) error {
	if m.ID <= 0 {
		return fmt.Errorf("sqlite: reinstate match: id %d: %w", m.ID, storage.ErrInvalid)
	}
	switch ok, err := checkIssued(ctx, s.db, "match", m.ID); {
	case err != nil:
		return fmt.Errorf("sqlite: reinstate match %d: %w", m.ID, err)
	case !ok:
		return fmt.Errorf("sqlite: reinstate match: id %d was never issued: %w", m.ID, storage.ErrInvalid)
	}
	var importDate any
	if !m.ImportDate.IsZero() {
		// CURRENT_TIMESTAMP's own spelling, so the column reads as it did.
		importDate = m.ImportDate.UTC().Format("2006-01-02 15:04:05")
		if m.ImportDate.Nanosecond() != 0 {
			importDate = m.ImportDate
		}
	}
	args := append([]any{
		m.ID, importDate,
		m.Player1Name, m.Player2Name, m.Event, m.Location, m.Round,
		m.MatchLength, nullableTime(m.MatchDate), m.FilePath, m.GameCount,
		m.TournamentID, m.Comment, m.CommentAuthor,
		nullableString(m.MatchHash), nullableString(m.CanonicalHash), nullableID(m.ImportBatchID),
		nullableString(m.DiceHash)}, sourceMetadataArgs(m)...)
	if _, err := s.db.ExecContext(ctx, matchReinstateSQL, args...); err != nil {
		if isKeyViolation(err) {
			return fmt.Errorf("sqlite: reinstate match %d: %w", m.ID, storage.ErrConflict)
		}
		return fmt.Errorf("sqlite: reinstate match %d: %w", m.ID, referenced(err))
	}
	return nil
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
// person through the alias table.
func (s *matchStore) withPlayerSpellings(ctx context.Context, scope string, opts storage.MatchListOpts) (storage.MatchListOpts, error) {
	names, err := sqlshared.PlayerSpellings(ctx, binder{s.db}.shared(), scope, opts.PlayerName)
	if err != nil {
		return opts, fmt.Errorf("sqlite: match list aliases: %w", err)
	}
	opts.PlayerSpellings = names
	return opts, nil
}

// buildMatchListWhere turns opts filters into a WHERE fragment (empty when no
// filter applies) and its positional args. Mirrors the Postgres builder; the
// two must stay in sync.
func buildMatchListWhere(opts storage.MatchListOpts) (whereSQL string, args []any) {
	var clauses []string
	if names := playerFilterNames(opts); len(names) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(names)), ",")
		clauses = append(clauses, "(m.player1_name IN ("+ph+") OR m.player2_name IN ("+ph+"))")
		for range 2 {
			for _, n := range names {
				args = append(args, n)
			}
		}
	}
	if opts.PlayerNameContains != "" {
		clauses = append(clauses, `(m.player1_name LIKE ? ESCAPE '\' OR m.player2_name LIKE ? ESCAPE '\')`)
		pat := sqlshared.ContainsPattern(opts.PlayerNameContains)
		args = append(args, pat, pat)
	}
	if opts.Text != "" {
		clauses = append(clauses, `(m.player1_name LIKE ? ESCAPE '\' OR m.player2_name LIKE ? ESCAPE '\'
			OR m.event LIKE ? ESCAPE '\' OR m.location LIKE ? ESCAPE '\' OR m.round LIKE ? ESCAPE '\'
			OR t.name LIKE ? ESCAPE '\' OR substr(m.match_date,1,10) LIKE ? ESCAPE '\'
			OR CAST(m.match_length AS TEXT) LIKE ? ESCAPE '\')`)
		pat := sqlshared.ContainsPattern(opts.Text)
		for range 8 {
			args = append(args, pat)
		}
	}
	if opts.Unassigned {
		clauses = append(clauses, "m.tournament_id IS NULL")
	}
	if len(opts.TournamentIDs) > 0 {
		ph := strings.TrimSuffix(strings.Repeat("?,", len(opts.TournamentIDs)), ",")
		clauses = append(clauses, "m.tournament_id IN ("+ph+")")
		for _, id := range opts.TournamentIDs {
			args = append(args, id)
		}
	}
	if opts.IDs != nil {
		if len(opts.IDs) == 0 {
			clauses = append(clauses, "1 = 0")
		} else {
			ph := strings.TrimSuffix(strings.Repeat("?,", len(opts.IDs)), ",")
			clauses = append(clauses, "m.id IN ("+ph+")")
			for _, id := range opts.IDs {
				args = append(args, id)
			}
		}
	}
	// Compare on the date part so an inclusive DateTo matches later that day.
	// substr, not date(): match_date carries a timezone suffix date() refuses.
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
		opts, err := s.withPlayerSpellings(ctx, scope, opts)
		if err != nil {
			yield(nil, err)
			return
		}
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

// Count returns how many matches satisfy the filters of opts.
func (s *matchStore) Count(ctx context.Context, scope string, opts storage.MatchListOpts) (int, error) {
	opts, err := s.withPlayerSpellings(ctx, scope, opts)
	if err != nil {
		return 0, err
	}
	whereSQL, args := buildMatchListWhere(opts)
	var n int
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM match m
		 LEFT JOIN tournament t ON m.tournament_id = t.id`+whereSQL, args...).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("sqlite: count matches: %w", err)
	}
	return n, nil
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
	err := withTx(ctx, s.db, func(tx execer) error {
		if _, err := tx.ExecContext(ctx,
			`UPDATE match SET player1_name = ?, player2_name = ?, match_date = ? WHERE id = ?`,
			player1Name, player2Name, dateVal, id); err != nil {
			return err
		}
		return refreshMatchPositionDates(ctx, tx, id)
	})
	if err != nil {
		return fmt.Errorf("sqlite: update match %d: %w", id, err)
	}
	return nil
}

// refreshMatchPositionDates re-dates every position matchID reaches, after
// an edit of its date.
func refreshMatchPositionDates(ctx context.Context, tx execer, matchID int64) error {
	ids, err := queryInt64s(ctx, tx, matchPositionIDsSQL, matchID)
	if err != nil {
		return err
	}
	return RefreshPositionMatchDates(ctx, tx, ids)
}

// SetVideoSource attaches or detaches (empty) a match's video source.
func (s *matchStore) SetVideoSource(ctx context.Context, scope string, id int64, source string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE match SET video_source = NULLIF(?, '') WHERE id = ?`, source, id)
	if err != nil {
		return fmt.Errorf("sqlite: set match %d video source: %w", id, err)
	}
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("sqlite: set match %d video source: %w", id, err)
	} else if n == 0 {
		return fmt.Errorf("sqlite: set match %d video source: %w", id, storage.ErrNotFound)
	}
	return nil
}

// ReplaceHeader rewrites a match's header columns in place — see
// storage.MatchStore. The hashes go in through nullableString for the same
// reason Save does it: their UNIQUE index counts two empty strings as a
// duplicate, where two NULLs are two unknowns.
func (s *matchStore) ReplaceHeader(ctx context.Context, scope string, id int64, m *domain.Match) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE match SET player1_name = ?, player2_name = ?, event = ?, location = ?,
		                  round = ?, match_length = ?, match_date = ?, game_count = ?,
		                  match_hash = ?, canonical_hash = ?, dice_hash = ?,
		                  player1_elo = ?, player2_elo = ?, player1_experience = ?, player2_experience = ?,
		                  transcriber = ?, has_jacoby = ?, has_beaver = ?, engine_version = ?,
		                  video_source = NULLIF(COALESCE(?, video_source), '')
		 WHERE id = ?`,
		append(append([]any{m.Player1Name, m.Player2Name, m.Event, m.Location,
			m.Round, m.MatchLength, nullableTime(m.MatchDate), m.GameCount,
			nullableString(m.MatchHash), nullableString(m.CanonicalHash), nullableString(m.DiceHash)},
			sourceMetadataArgs(m)...), id)...)
	if err != nil {
		return fmt.Errorf("sqlite: replace match %d header: %w", id, err)
	}
	// A replacement names its match: none there means the caller holds a
	// stale id, which must not pass for a rewrite.
	if n, err := res.RowsAffected(); err != nil {
		return fmt.Errorf("sqlite: replace match %d header: %w", id, err)
	} else if n == 0 {
		return fmt.Errorf("sqlite: replace match %d header: %w", id, storage.ErrNotFound)
	}
	if err := refreshMatchPositionDates(ctx, s.db, id); err != nil {
		return fmt.Errorf("sqlite: replace match %d header: %w", id, err)
	}
	return nil
}

// UpdateComment sets the free-text comment on a match, signed by the
// context's comment author (storage.WithCommentAuthor).
func (s *matchStore) UpdateComment(ctx context.Context, scope string, id int64, comment string) error {
	if _, err := s.db.ExecContext(ctx,
		`UPDATE match SET comment = ?, comment_author = ? WHERE id = ?`,
		comment, storage.CommentAuthorFromContext(ctx), id); err != nil {
		return fmt.Errorf("sqlite: update match %d comment: %w", id, err)
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
const positionIsHeldSQL = `EXISTS (SELECT 1 FROM move               WHERE position_id = position.id)
	                       OR EXISTS (SELECT 1 FROM collection_position WHERE position_id = position.id)
	                       OR EXISTS (SELECT 1 FROM anki_card           WHERE position_id = position.id)
	                       OR EXISTS (SELECT 1 FROM lesson_step         WHERE position_id = position.id)
	                       OR EXISTS (SELECT 1 FROM comment             WHERE position_id = position.id AND origin = 'user')
	                       OR position.individually_imported = 1
	                       OR position.flagged = 1`

// deleteOrphanedPositions removes every position in ids that positionIsHeldSQL
// says nothing holds any more, as a single set-based DELETE rather than one
// EXISTS round-trip per id.
func deleteOrphanedPositions(ctx context.Context, tx execer, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	query := fmt.Sprintf(`DELETE FROM position WHERE id IN (%s) AND NOT (%s)`, placeholders, positionIsHeldSQL)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("delete orphaned positions: %w", err)
	}
	return nil
}

// DeleteCascade removes a match and all of its games, moves and move analyses
// (via ON DELETE CASCADE), then deletes any position the match referenced that
// nothing else holds (see positionIsHeldSQL). The cascade and the orphan
// cleanup run atomically.
func (s *matchStore) DeleteCascade(ctx context.Context, scope string, id int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		positionIDs, err := queryInt64s(ctx, tx,
			`SELECT DISTINCT mv.position_id
			 FROM move mv INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = ? AND mv.position_id IS NOT NULL`, id)
		if err != nil {
			return fmt.Errorf("collect positions: %w", err)
		}

		if _, err := tx.ExecContext(ctx, sqlshared.InvalidateMatchStatsOfMatchesSQL+"(?)", id); err != nil {
			return fmt.Errorf("invalidate match stats: %w", err)
		}
		// game/move/move_analysis cascade off the match delete.
		if _, err := tx.ExecContext(ctx, `DELETE FROM match WHERE id = ?`, id); err != nil {
			return err
		}

		if err := deleteOrphanedPositions(ctx, tx, positionIDs); err != nil {
			return err
		}
		return RefreshPositionMatchDates(ctx, tx, positionIDs)
	})
	if err != nil {
		return fmt.Errorf("sqlite: delete match %d: %w", id, err)
	}
	return nil
}

// DeleteGames removes a match's games (moves and move analyses cascade off
// them) and returns the positions they referenced, without touching the match
// row — see storage.MatchStore. The collection query is DeleteCascade's, run
// before the delete for the same reason: the moves that name the positions are
// about to be gone.
func (s *matchStore) DeleteGames(ctx context.Context, scope string, matchID int64) ([]int64, error) {
	var positionIDs []int64
	err := withTx(ctx, s.db, func(tx execer) error {
		ids, err := queryInt64s(ctx, tx,
			`SELECT DISTINCT mv.position_id
			 FROM move mv INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = ? AND mv.position_id IS NOT NULL`, matchID)
		if err != nil {
			return fmt.Errorf("collect positions: %w", err)
		}
		positionIDs = ids
		if _, err := tx.ExecContext(ctx, sqlshared.InvalidateMatchStatsOfMatchesSQL+"(?)", matchID); err != nil {
			return fmt.Errorf("invalidate match stats: %w", err)
		}
		// move/move_analysis cascade off the game delete.
		if _, err := tx.ExecContext(ctx, `DELETE FROM game WHERE match_id = ?`, matchID); err != nil {
			return err
		}
		return RefreshPositionMatchDates(ctx, tx, ids)
	})
	if err != nil {
		return nil, fmt.Errorf("sqlite: delete games of match %d: %w", matchID, err)
	}
	return positionIDs, nil
}

// PurgeOrphanPositions drops the positions in ids that positionIsHeldSQL no
// longer holds — see storage.MatchStore. It is deleteOrphanedPositions, the
// half of DeleteCascade a replacement needs on its own, once the new rows are
// written.
func (s *matchStore) PurgeOrphanPositions(ctx context.Context, scope string, ids []int64) error {
	if err := deleteOrphanedPositions(ctx, s.db, ids); err != nil {
		return fmt.Errorf("sqlite: purge orphan positions: %w", err)
	}
	return nil
}

// SwapPlayers swaps player 1 and player 2 for the match: it swaps the header
// names, the per-game scores and winner, the per-move player, and the score /
// cube-owner columns of every position the match's moves reference.
func (s *matchStore) SwapPlayers(ctx context.Context, scope string, id int64) error {
	err := withTx(ctx, s.db, func(tx execer) error {
		// The seats trade places: each row now describes the other player.
		if _, err := tx.ExecContext(ctx, sqlshared.InvalidateMatchStatsOfMatchesSQL+"(?)", id); err != nil {
			return fmt.Errorf("invalidate match stats: %w", err)
		}
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
		// Positions swap by copy-on-write, NOT in place: a position may be
		// shared with other matches and its score/cube are part of the Zobrist
		// hash. Save a swapped copy (Save dedups) and repoint this match's moves.
		posIDs, err := queryInt64s(ctx, tx,
			`SELECT DISTINCT mv.position_id FROM move mv
			 INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = ? AND mv.position_id IS NOT NULL`, id)
		if err != nil {
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
		var swappedAway, repointed []int64
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
			if _, err := tx.ExecContext(ctx,
				`UPDATE move SET position_id = ?
				 WHERE position_id = ? AND game_id IN (SELECT id FROM game WHERE match_id = ?)`,
				newID, pid, id); err != nil {
				return fmt.Errorf("repoint swapped move: %w", err)
			}
			swappedAway = append(swappedAway, pid)
			repointed = append(repointed, newID)
		}
		// The repointed moves are scored by their new position's analysis.
		if _, err := sqlshared.RescorePlayedDecisionsOf(ctx, binder{tx}.shared(), repointed); err != nil {
			return fmt.Errorf("rescore swapped moves: %w", err)
		}
		if err := deleteOrphanedPositions(ctx, tx, swappedAway); err != nil {
			return fmt.Errorf("swap orphan cleanup: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("sqlite: swap players for match %d: %w", id, err)
	}
	return nil
}

// MergePlayers records every other name as an alias of canonical
// (storage.AliasStore.Set): the matches keep the names their files wrote, and
// the stats, the players table, the search and every later import read them
// as canonical.
func (s *matchStore) MergePlayers(ctx context.Context, scope string, names []string, canonical string) error {
	return sqlshared.MergeAliases(ctx, shared(binder{s.db}), scope, storage.AliasPlayer, names, canonical)
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
	// The match gains moves: its rows no longer cover all of them.
	if _, err := s.db.ExecContext(ctx, sqlshared.InvalidateMatchStatsOfMatchesSQL+"(?)", g.MatchID); err != nil {
		return 0, fmt.Errorf("sqlite: invalidate match stats: %w", err)
	}
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

// moveSelectCols reads a domain.Move; scanMove is its counterpart.
var moveSelectCols = moveColsOf("")

// moveColsOf renders moveSelectCols with every column prefixed by p ("mv."
// for a join).
func moveColsOf(p string) string {
	return p + `id, COALESCE(` + p + `game_id,0), COALESCE(` + p + `move_number,0), ` + sqlshared.ActionLabelOrEmptySQL(p+"move_type") + `,
	` + p + `position_id, COALESCE(` + p + `player,0), COALESCE(` + p + `dice_1,0), COALESCE(` + p + `dice_2,0),
	COALESCE(` + p + `checker_move,''), ` + sqlshared.ActionLabelOrEmptySQL(p+"cube_action") + `, ` + p + `luck_mp,
	` + p + `decision_ms, ` + p + `cube_decision_ms, ` + p + `roll_tick_ms, ` + p + `tick_ms`
}

func scanMove(sc interface{ Scan(...any) error }) (domain.Move, error) {
	var mv domain.Move
	var d1, d2 int32
	var positionID sql.NullInt64
	var luckMP sql.NullInt32
	var decision, cube, rollTick, tick sql.NullInt64
	if err := sc.Scan(&mv.ID, &mv.GameID, &mv.MoveNumber, &mv.MoveType,
		&positionID, &mv.Player, &d1, &d2, &mv.CheckerMove, &mv.CubeAction, &luckMP,
		&decision, &cube, &rollTick, &tick); err != nil {
		return domain.Move{}, err
	}
	mv.DecisionMS, mv.CubeDecisionMS = sqlshared.NullableMS(decision), sqlshared.NullableMS(cube)
	mv.RollTickMS, mv.TickMS = sqlshared.NullableMS(rollTick), sqlshared.NullableMS(tick)
	mv.Dice = [2]int32{d1, d2}
	if positionID.Valid {
		mv.PositionID = positionID.Int64
	}
	if luckMP.Valid {
		v := luckMP.Int32
		mv.LuckMP = &v
	}
	return mv, nil
}

// scanScoredMove reads moveSelectCols followed by the Position's analysis
// blob, and scores the play from it.
func scanScoredMove(rows *sql.Rows, scorer sqlshared.PlayScorer) (domain.Move, error) {
	var mv domain.Move
	var d1, d2 int32
	var positionID sql.NullInt64
	var luckMP sql.NullInt32
	var data []byte
	var decision, cube, rollTick, tick sql.NullInt64
	if err := rows.Scan(&mv.ID, &mv.GameID, &mv.MoveNumber, &mv.MoveType,
		&positionID, &mv.Player, &d1, &d2, &mv.CheckerMove, &mv.CubeAction, &luckMP,
		&decision, &cube, &rollTick, &tick, &data); err != nil {
		return domain.Move{}, err
	}
	mv.DecisionMS, mv.CubeDecisionMS = sqlshared.NullableMS(decision), sqlshared.NullableMS(cube)
	mv.RollTickMS, mv.TickMS = sqlshared.NullableMS(rollTick), sqlshared.NullableMS(tick)
	mv.Dice = [2]int32{d1, d2}
	if positionID.Valid {
		mv.PositionID = positionID.Int64
	}
	if luckMP.Valid {
		v := luckMP.Int32
		mv.LuckMP = &v
	}
	scorer.Score(&mv, data)
	return mv, nil
}

const moveInsertSQL = `INSERT INTO move (
	game_id, move_number, move_type, position_id, player,
	dice_1, dice_2, checker_move, cube_action, luck_mp,
	decision_ms, cube_decision_ms, decision_error_mp, is_close_cube,
	roll_tick_ms, tick_ms
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

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
	moveType, err := actionCode(ctx, s.db, mv.MoveType)
	if err != nil {
		return 0, fmt.Errorf("sqlite: create move: %w", err)
	}
	cubeAction, err := actionCode(ctx, s.db, mv.CubeAction)
	if err != nil {
		return 0, fmt.Errorf("sqlite: create move: %w", err)
	}
	decisionErr, closeCube, err := sqlshared.PlayedDecisionArgs(ctx, binder{s.db}.shared(), mv.PositionID, mv.CheckerMove, mv.CubeAction)
	if err != nil {
		return 0, fmt.Errorf("sqlite: create move: %w", err)
	}
	res, err := s.db.ExecContext(ctx, moveInsertSQL,
		mv.GameID, mv.MoveNumber, moveType, positionID, mv.Player,
		mv.Dice[0], mv.Dice[1], mv.CheckerMove, cubeAction, luckMP,
		sqlshared.MSArg(mv.DecisionMS), sqlshared.MSArg(mv.CubeDecisionMS), decisionErr, closeCube,
		sqlshared.MSArg(mv.RollTickMS), sqlshared.MSArg(mv.TickMS))
	if err != nil {
		return 0, fmt.Errorf("sqlite: create move: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sqlite: create move id: %w", err)
	}
	mv.ID = id
	if positionID != nil {
		if _, err := s.db.ExecContext(ctx, positionMatchDateOnMoveSQL, mv.GameID, positionID); err != nil {
			return 0, fmt.Errorf("sqlite: date position of move: %w", err)
		}
	}
	return id, nil
}

// positionMatchDateOnMoveSQL lowers position.match_date to the date of the
// match a new move ties the position to. It compares with the stored value
// rather than taking the MIN over the position's moves: an opening position
// is reached by thousands of moves, and an import would pay that scan on
// every one of them.
var positionMatchDateOnMoveSQL = `UPDATE position SET match_date = d.md
	FROM (SELECT ` + UnixFromMatchDateSQL("m.match_date") + ` AS md FROM game g JOIN match m ON m.id = g.match_id WHERE g.id = ?) AS d
	WHERE position.id = ? AND d.md IS NOT NULL
	  AND (position.match_date IS NULL OR position.match_date > d.md)`

// positionMatchDateRefreshSQL recomputes position.match_date from every match
// that still reaches the position: the slow, exact form, for the rare edits
// that can raise the date (a match deleted, its date changed, its games
// replaced). Takes the IN list of ids.
var positionMatchDateRefreshSQL = `UPDATE position SET match_date =
	(SELECT MIN(` + UnixFromMatchDateSQL("m.match_date") + `) FROM move mv
	   JOIN game g ON g.id = mv.game_id
	   JOIN match m ON m.id = g.match_id
	  WHERE mv.position_id = position.id)
	WHERE id IN `

// RefreshPositionMatchDates recomputes position.match_date for ids, in
// batches; see positionMatchDateRefreshSQL. Exported for the Database
// wrapper, whose match edits run their own SQL.
func RefreshPositionMatchDates(ctx context.Context, db interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}, ids []int64) error {
	const batch = 500
	for start := 0; start < len(ids); start += batch {
		chunk := ids[start:min(start+batch, len(ids))]
		args := make([]any, len(chunk))
		for i, id := range chunk {
			args[i] = id
		}
		q := positionMatchDateRefreshSQL + "(" + strings.TrimSuffix(strings.Repeat("?,", len(chunk)), ",") + ")"
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			return fmt.Errorf("refresh position match dates: %w", err)
		}
	}
	return nil
}

// matchPositionIDsSQL lists the positions a match's moves reach.
const matchPositionIDsSQL = `SELECT DISTINCT mv.position_id
	FROM move mv INNER JOIN game g ON mv.game_id = g.id
	WHERE g.match_id = ? AND mv.position_id IS NOT NULL`

// Moves streams the moves of a game ordered by move number.
func (s *matchStore) Moves(ctx context.Context, scope string, gameID int64) iter.Seq2[*domain.Move, error] {
	return func(yield func(*domain.Move, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT `+moveColsOf("mv.")+`, a.data
			 FROM move mv LEFT JOIN analysis a ON a.position_id = mv.position_id
			 WHERE mv.game_id = ? ORDER BY mv.move_number`, gameID)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list moves: %w", err))
			return
		}
		defer rows.Close()
		scorer := sqlshared.PlayScorer{}
		for rows.Next() {
			mv, err := scanScoredMove(rows, scorer)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: list moves: %w", err))
				return
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
			`SELECT `+moveColsOf("mv.")+`, a.data
			 FROM move mv INNER JOIN game g ON mv.game_id = g.id
			 LEFT JOIN analysis a ON a.position_id = mv.position_id
			 WHERE g.match_id = ?
			 ORDER BY g.game_number, mv.move_number`, matchID)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list moves by match: %w", err))
			return
		}
		defer rows.Close()
		scorer := sqlshared.PlayScorer{}
		for rows.Next() {
			mv, err := scanScoredMove(rows, scorer)
			if err != nil {
				yield(nil, fmt.Errorf("sqlite: list moves by match: %w", err))
				return
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

// MovesByPositions — see storage.MatchStore.
func (s *matchStore) MovesByPositions(ctx context.Context, scope string, positionIDs []int64) (map[int64][]*domain.Move, error) {
	out := make(map[int64][]*domain.Move)
	err := forEachIn(ctx, s.db, positionIDs,
		`SELECT `+moveSelectCols+` FROM move WHERE position_id IN `, ` ORDER BY id`,
		func(rows *sql.Rows) error {
			mv, err := scanMove(rows)
			if err != nil {
				return err
			}
			out[mv.PositionID] = append(out[mv.PositionID], &mv)
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("sqlite: moves by positions: %w", err)
	}
	return out, nil
}

const moveAnalysisCols = `analysis_type, depth, equity, equity_error, win_rate, gammon_rate, backgammon_rate,
	opponent_win_rate, opponent_gammon_rate, opponent_backgammon_rate`

// CreateMoveAnalysis — see storage.MatchStore.
func (s *matchStore) CreateMoveAnalysis(ctx context.Context, scope string, ma *domain.MoveAnalysis) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO move_analysis (move_id, `+moveAnalysisCols+`) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		ma.MoveID, ma.AnalysisType, ma.Depth, ma.Equity, ma.EquityError,
		ma.WinRate, ma.GammonRate, ma.BackgammonRate,
		ma.OpponentWinRate, ma.OpponentGammonRate, ma.OpponentBackgammonRate)
	if err != nil {
		return 0, fmt.Errorf("sqlite: create move analysis: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("sqlite: create move analysis id: %w", err)
	}
	ma.ID = id
	return id, nil
}

// MoveAnalysesByMatch — see storage.MatchStore.
func (s *matchStore) MoveAnalysesByMatch(ctx context.Context, scope string, matchID int64) iter.Seq2[*domain.MoveAnalysis, error] {
	return func(yield func(*domain.MoveAnalysis, error) bool) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT ma.id, ma.move_id, `+qualify(moveAnalysisCols, "ma")+`
			 FROM move_analysis ma
			 INNER JOIN move mv ON ma.move_id = mv.id
			 INNER JOIN game g ON mv.game_id = g.id
			 WHERE g.match_id = ?
			 ORDER BY g.game_number, mv.move_number, ma.id`, matchID)
		if err != nil {
			yield(nil, fmt.Errorf("sqlite: list move analyses: %w", err))
			return
		}
		defer rows.Close()
		for rows.Next() {
			var ma domain.MoveAnalysis
			var analysisType, depth sql.NullString
			var eq, eqErr, win, gam, bg, oWin, oGam, oBg sql.NullFloat64
			if err := rows.Scan(&ma.ID, &ma.MoveID, &analysisType, &depth,
				&eq, &eqErr, &win, &gam, &bg, &oWin, &oGam, &oBg); err != nil {
				yield(nil, fmt.Errorf("sqlite: list move analyses: %w", err))
				return
			}
			ma.AnalysisType, ma.Depth = analysisType.String, depth.String
			ma.Equity, ma.EquityError = eq.Float64, eqErr.Float64
			ma.WinRate, ma.GammonRate, ma.BackgammonRate = win.Float64, gam.Float64, bg.Float64
			ma.OpponentWinRate, ma.OpponentGammonRate, ma.OpponentBackgammonRate = oWin.Float64, oGam.Float64, oBg.Float64
			if !yield(&ma, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(nil, fmt.Errorf("sqlite: list move analyses: %w", err))
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
			        `+sqlshared.ActionLabelOrEmptySQL("mv.move_type")+`, COALESCE(mv.player,0), mv.position_id,
			        p.state, p.decision_type, p.player_on_roll, p.dice_1, p.dice_2,
			        p.cube_value, p.cube_owner, p.score_1, p.score_2,
			        p.has_jacoby, p.has_beaver, p.max_cube,
			        COALESCE(mv.checker_move,''), `+sqlshared.ActionLabelOrEmptySQL("mv.cube_action")+`,
			        mv.decision_ms, mv.cube_decision_ms, mv.roll_tick_ms, mv.tick_ms
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
			var dt, por, d1, d2, cv, co, s1, s2, hj, hb, mc, decisionMS, cubeDecisionMS, rollTick, tick sql.NullInt64
			if err := rows.Scan(&moveID, &gameID, &gameNumber, &moveNumber,
				&moveType, &player, &positionID,
				&state, &dt, &por, &d1, &d2, &cv, &co, &s1, &s2, &hj, &hb, &mc,
				&checkerMove, &cubeAction, &decisionMS, &cubeDecisionMS, &rollTick, &tick); err != nil {
				yield(nil, fmt.Errorf("sqlite: move positions for match %d: %w", matchID, err))
				return
			}
			position := engine.ReconstructPosition(positionID, state,
				int(dt.Int64), int(por.Int64), int(d1.Int64), int(d2.Int64),
				int(cv.Int64), int(co.Int64), int(s1.Int64), int(s2.Int64),
				int(hj.Int64), int(hb.Int64))
			position.MaxCube = int(mc.Int64)
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

				DecisionMS:     sqlshared.NullableMS(decisionMS),
				CubeDecisionMS: sqlshared.NullableMS(cubeDecisionMS),
				RollTickMS:     sqlshared.NullableMS(rollTick),
				TickMS:         sqlshared.NullableMS(tick),
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

// ListByDiceHash returns the matches sharing dice_hash — see storage.MatchStore.
func (s *matchStore) ListByDiceHash(ctx context.Context, scope string, hash string) ([]domain.Match, error) {
	if hash == "" {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+matchSelectCols+` FROM match m
		 LEFT JOIN tournament t ON m.tournament_id = t.id
		 WHERE m.dice_hash = ? ORDER BY m.id`, hash)
	if err != nil {
		return nil, fmt.Errorf("sqlite: list matches by dice hash: %w", err)
	}
	defer rows.Close()
	var out []domain.Match
	for rows.Next() {
		m, err := scanMatch(rows)
		if err != nil {
			return nil, fmt.Errorf("sqlite: list matches by dice hash: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SetDiceHash stores a match's dice_hash — see storage.MatchStore.
func (s *matchStore) SetDiceHash(ctx context.Context, scope string, id int64, hash string) error {
	if _, err := s.db.ExecContext(ctx, `UPDATE match SET dice_hash = ? WHERE id = ?`, nullableString(hash), id); err != nil {
		return fmt.Errorf("sqlite: set match %d dice hash: %w", id, err)
	}
	return nil
}

// DiceSequences streams every match with its dice — see storage.MatchStore.
func (s *matchStore) DiceSequences(ctx context.Context, scope string) iter.Seq2[storage.MatchDice, error] {
	return func(yield func(storage.MatchDice, error) bool) {
		rows, err := s.db.QueryContext(ctx, fmt.Sprintf(sqlshared.DiceSequencesSQL, "1 = 1"))
		if err != nil {
			yield(storage.MatchDice{}, fmt.Errorf("sqlite: match dice: %w", err))
			return
		}
		defer rows.Close()
		var f storage.DiceFolder
		for rows.Next() {
			var r storage.DiceRow
			if err := rows.Scan(&r.MatchID, &r.Player1, &r.Player2, &r.Length, &r.Hash,
				&r.GameID, &r.Score1, &r.Score2, &r.D1, &r.D2); err != nil {
				yield(storage.MatchDice{}, fmt.Errorf("sqlite: match dice: %w", err))
				return
			}
			if m, ok := f.Add(r); ok && !yield(m, nil) {
				return
			}
		}
		if err := rows.Err(); err != nil {
			yield(storage.MatchDice{}, fmt.Errorf("sqlite: match dice: %w", err))
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

// qualify prefixes every column of a plain column list (moveSelectCols has its own moveColsOf) with alias —
// the list is written unqualified so it can serve single-table queries too.
func qualify(cols, alias string) string {
	var out []string
	depth, start := 0, 0
	emit := func(c string) {
		c = strings.TrimSpace(c)
		if strings.HasPrefix(c, "COALESCE(") {
			c = "COALESCE(" + alias + "." + c[len("COALESCE("):]
		} else {
			c = alias + "." + c
		}
		out = append(out, c)
	}
	for i, r := range cols {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				emit(cols[start:i])
				start = i + 1
			}
		}
	}
	emit(cols[start:])
	return strings.Join(out, ", ")
}
