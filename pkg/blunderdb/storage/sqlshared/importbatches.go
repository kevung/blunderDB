package sqlshared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ImportBatchStore implements storage.ImportBatchStore. import_batch is a
// domain table: every statement is confined to the scope's tenant through
// Dialect.TenantFilter / TenantColumns.
//
// The report's aggregates reuse the statistics' own predicates (countedExpr,
// statsErrExpr, statsBaseJoin): a second definition of "a decision that
// counts" would be a second PR.
type ImportBatchStore struct{ DB Execer }

var _ storage.ImportBatchStore = (*ImportBatchStore)(nil)

// Begin opens a batch and returns its id.
func (s *ImportBatchStore) Begin(ctx context.Context, scope string, source, format string) (int64, error) {
	cols, args := s.DB.TenantColumns(scope)
	cols = append(cols, "source", "format", "counts")
	args = append(args, source, format, "{}")
	id, err := s.DB.Insert(ctx,
		`INSERT INTO import_batch (`+strings.Join(cols, ", ")+`) VALUES (`+Placeholders(len(cols))+`)`, args...)
	if err != nil {
		return 0, errf(s.DB, "begin import batch", err)
	}
	return id, nil
}

// Finish stamps the batch as done and stores the counts the import observed.
func (s *ImportBatchStore) Finish(ctx context.Context, scope string, batchID int64, counts domain.ImportReport) error {
	blob, err := json.Marshal(counts)
	if err != nil {
		return errf(s.DB, "finish import batch", err)
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	n, err := s.DB.Exec(ctx,
		`UPDATE import_batch SET finished_at = CURRENT_TIMESTAMP, counts = ? WHERE id = ? AND `+tenant,
		append([]any{string(blob), batchID}, targs...)...)
	if err != nil {
		return errf(s.DB, fmt.Sprintf("finish import batch %d", batchID), err)
	}
	if n == 0 {
		return fmt.Errorf("%s: finish import batch %d: %w", s.DB.Name(), batchID, storage.ErrNotFound)
	}
	return nil
}

// Load returns a batch with its stored counts, unmeasured.
func (s *ImportBatchStore) Load(ctx context.Context, scope string, batchID int64) (*domain.ImportBatch, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	row := s.DB.QueryRow(ctx,
		`SELECT id, `+s.DB.TimestampText("started_at")+`, `+s.DB.TimestampText("finished_at")+`,
		        COALESCE(source,''), COALESCE(format,''), COALESCE(counts,'{}')
		 FROM import_batch WHERE id = ? AND `+tenant,
		append([]any{batchID}, targs...)...)
	b, err := scanImportBatch(row)
	if errors.Is(err, ErrNoRows) {
		return nil, fmt.Errorf("%s: load import batch %d: %w", s.DB.Name(), batchID, storage.ErrNotFound)
	}
	if err != nil {
		return nil, errf(s.DB, fmt.Sprintf("load import batch %d", batchID), err)
	}
	return b, nil
}

// List returns the batches, most recent first.
func (s *ImportBatchStore) List(ctx context.Context, scope string, opts storage.ListOpts) ([]*domain.ImportBatch, error) {
	tenant, targs := s.DB.TenantFilter("", scope)
	limit, largs := s.DB.LimitOffset(opts.Limit, opts.Offset)
	rows, err := s.DB.Query(ctx,
		`SELECT id, `+s.DB.TimestampText("started_at")+`, `+s.DB.TimestampText("finished_at")+`,
		        COALESCE(source,''), COALESCE(format,''), COALESCE(counts,'{}')
		 FROM import_batch WHERE `+tenant+` ORDER BY id DESC`+limit,
		append(targs, largs...)...)
	if err != nil {
		return nil, errf(s.DB, "list import batches", err)
	}
	defer rows.Close()
	var out []*domain.ImportBatch
	for rows.Next() {
		b, err := scanImportBatch(rows)
		if err != nil {
			return nil, errf(s.DB, "list import batches", err)
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "list import batches", err)
	}
	return out, nil
}

func scanImportBatch(sc interface{ Scan(...any) error }) (*domain.ImportBatch, error) {
	var b domain.ImportBatch
	var counts string
	if err := sc.Scan(&b.ID, &b.StartedAt, &b.FinishedAt, &b.Source, &b.Format, &counts); err != nil {
		return nil, err
	}
	// A counts blob written by a newer version, or corrupted, must not make the
	// batch unreadable: the stored figures are a convenience, and Report
	// measures everything that can be measured anyway.
	_ = json.Unmarshal([]byte(counts), &b.Report)
	return &b, nil
}

// Report completes a batch's stored counts with what can be measured over its
// matches now. See storage.ImportBatchStore.
func (s *ImportBatchStore) Report(ctx context.Context, scope string, batchID int64, players []string) (*domain.ImportBatch, error) {
	b, err := s.Load(ctx, scope, batchID)
	if err != nil {
		return nil, err
	}
	what := fmt.Sprintf("report of import batch %d", batchID)

	if err := s.measurePositions(ctx, scope, b); err != nil {
		return nil, errf(s.DB, what, err)
	}
	if err := s.measurePerformance(ctx, scope, b, players); err != nil {
		return nil, errf(s.DB, what, err)
	}
	if err := s.measureWorst(ctx, scope, b, players); err != nil {
		return nil, errf(s.DB, what, err)
	}
	return b, nil
}

// batchPositionsFrom is the join from a batch to the positions its matches
// touched. DISTINCT because a position recurs across games and matches, and
// the report counts positions, not visits.
const batchPositionsFrom = `FROM position p
JOIN move mv ON mv.position_id = p.id
JOIN game g ON g.id = mv.game_id
JOIN match m ON m.id = g.match_id`

// measurePositions fills the two position counts the panel acts on: what the
// source tool had flagged for study, and what no engine has judged.
func (s *ImportBatchStore) measurePositions(ctx context.Context, scope string, b *domain.ImportBatch) error {
	tenant, targs := s.DB.TenantFilter("p", scope)
	args := append(append([]any{}, targs...), b.ID)
	row := s.DB.QueryRow(ctx,
		`SELECT COUNT(DISTINCT CASE WHEN `+s.DB.Bool("p.flagged", true)+` THEN p.id END),
		        COUNT(DISTINCT CASE WHEN a.position_id IS NULL THEN p.id END)
		 `+batchPositionsFrom+`
		 LEFT JOIN analysis a ON a.position_id = p.id
		 WHERE `+tenant+` AND m.import_batch_id = ?`, args...)
	return row.Scan(&b.Report.PositionsFlagged, &b.Report.PositionsWithoutAnalysis)
}

// playerClause is batchPlayerClause over every spelling the player aliases
// give each named player, as the statistics read a player.
func (s *ImportBatchStore) playerClause(ctx context.Context, scope string, players []string) (string, []any, error) {
	var names []string
	for _, p := range players {
		group, err := PlayerSpellings(ctx, s.DB, scope, p)
		if err != nil {
			return "", nil, fmt.Errorf("import batch player aliases: %w", err)
		}
		names = append(names, group...)
	}
	clause, args := batchPlayerClause(names)
	return clause, args, nil
}

// batchPlayerClause narrows a stats query to the decisions of the named
// players, seat-aware: a row counts only when one of them IS the player who
// took the decision. Empty names score both seats, and the caller says so in
// the report rather than pretending the figure is one player's.
func batchPlayerClause(players []string) (string, []any) {
	names := make([]string, 0, len(players))
	for _, n := range players {
		if n = strings.TrimSpace(n); n != "" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return "", nil
	}
	ph := strings.TrimSuffix(strings.Repeat("?,", len(names)), ",")
	clause := " AND ((m.player1_name IN (" + ph + ") AND mv.player = 1) OR (m.player2_name IN (" + ph + ") AND mv.player = -1))"
	args := make([]any, 0, 2*len(names))
	for range 2 {
		for _, n := range names {
			args = append(args, n)
		}
	}
	return clause, args
}

// measurePerformance computes the batch's own PR — the same figure the
// statistics show, over this import alone, from the same countedExpr. A batch
// that carried no analysis scores no decisions and reports a PR of zero, which
// the panel must render as "no analysis" rather than as a perfect game.
func (s *ImportBatchStore) measurePerformance(ctx context.Context, scope string, b *domain.ImportBatch, players []string) error {
	tenant, targs := s.DB.TenantFilter("p", scope)
	playerClause, playerArgs, perr := s.playerClause(ctx, scope, players)
	if perr != nil {
		return perr
	}
	args := append(append([]any{}, targs...), b.ID)
	args = append(args, playerArgs...)

	var total int64
	var decisions int
	row := s.DB.QueryRow(ctx,
		`SELECT COUNT(*), `+s.DB.Bigint("COALESCE(SUM("+statsErrExpr+"), 0)")+`
		 `+statsBaseJoin+`
		 WHERE `+tenant+` AND m.import_batch_id = ?`+playerClause+`
		   AND `+countedExpr(s.DB), args...)
	if err := row.Scan(&decisions, &total); err != nil {
		return err
	}
	b.Report.Decisions = decisions
	b.Report.PR = pr(total, decisions)
	b.Report.Player = strings.Join(players, ", ")
	return nil
}

// measureWorst lists the batch's most expensive decisions, worst first.
//
// It reads the SAME error column (statsErrExpr) and counted predicate as the
// statistics.
func (s *ImportBatchStore) measureWorst(ctx context.Context, scope string, b *domain.ImportBatch, players []string) error {
	tenant, targs := s.DB.TenantFilter("p", scope)
	playerClause, playerArgs, perr := s.playerClause(ctx, scope, players)
	if perr != nil {
		return perr
	}
	args := append(append([]any{}, targs...), b.ID)
	args = append(args, playerArgs...)
	limit, largs := s.DB.LimitOffset(domain.MaxImportBlunders, 0)
	args = append(args, largs...)

	rows, err := s.DB.Query(ctx,
		`SELECT p.id, m.id,
		        COALESCE(m.player1_name,''), COALESCE(m.player2_name,''), COALESCE(m.match_length, 0),
		        `+statsErrExpr+`, p.decision_type
		 `+statsBaseJoin+`
		 WHERE `+tenant+` AND m.import_batch_id = ?`+playerClause+`
		   AND `+countedExpr(s.DB)+`
		   AND COALESCE(`+statsErrExpr+`, 0) > 0
		 ORDER BY `+statsErrExpr+` DESC, p.id ASC`+limit, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	b.Report.WorstDecisions = nil
	for rows.Next() {
		var bl domain.ImportBlunder
		var p1, p2 string
		var length, decisionType int
		if err := rows.Scan(&bl.PositionID, &bl.MatchID, &p1, &p2, &length, &bl.ErrorMP, &decisionType); err != nil {
			return err
		}
		bl.IsCube = decisionType == 1
		bl.Label = matchLabel(p1, p2, length)
		b.Report.WorstDecisions = append(b.Report.WorstDecisions, bl)
	}
	return rows.Err()
}

// matchLabel renders a match the way the report shows it. Built here rather
// than in the panel so the CLI, the daemon and the interface all print the
// same string.
func matchLabel(player1, player2 string, length int) string {
	names := strings.TrimSpace(player1 + " — " + player2)
	if names == "—" {
		names = ""
	}
	switch {
	case names == "" && length > 0:
		return fmt.Sprintf("%d pts", length)
	case names == "":
		return ""
	case length > 0:
		return fmt.Sprintf("%s, %d pts", names, length)
	}
	return names
}

// StudyQueue — see storage.ImportBatchStore.
//
// Three queries rather than one UNION with a computed rank: the three reasons
// have different predicates and different orderings, deduplication is trivial
// in Go, and a UNION here would have to be written twice anyway because the
// two dialects order NULLs differently. Each query is bounded by the same
// limit, so the worst case reads three pages and not the batch.
func (s *ImportBatchStore) StudyQueue(ctx context.Context, scope string, batchID int64, players []string, limit int) ([]domain.StudyQueueEntry, error) {
	if batchID <= 0 {
		// queueRows reads a zero batch as "every batch", which is StudyBacklog's
		// question, not this one's.
		return nil, nil
	}
	if limit <= 0 || limit > domain.MaxStudyQueue {
		limit = domain.MaxStudyQueue
	}

	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, fmt.Errorf("study queue settings: %w", err)
	}

	var out []domain.StudyQueueEntry
	seen := map[int64]bool{}
	add := func(entries []domain.StudyQueueEntry) {
		for _, e := range entries {
			if len(out) >= limit || seen[e.PositionID] {
				continue
			}
			seen[e.PositionID] = true
			out = append(out, e)
		}
	}

	// 1. What cost something, worst first. This is what the user came for.
	// The ORDER BY repeats the select list's COALESCE rather than the bare
	// expression: PostgreSQL requires every ORDER BY expression of a SELECT
	// DISTINCT to appear in the select list, and `x` is not `COALESCE(x, 0)`
	// to it. Wrapping also settles a dialect divergence for free — SQLite
	// sorts NULLs last in DESC, PostgreSQL first — though this pass never
	// sees one, its WHERE excluding them.
	// The line is the library's error threshold (ADR-0046): "worth
	// revisiting" is precisely what an Error is.
	blunders, err := s.queueRows(ctx, scope, batchID, players, limit, domain.StudyBlunder,
		` AND COALESCE(`+statsErrExpr+`, 0) >= ?`, []any{settings.ErrorThresholdMP},
		queueCostOrder)
	if err != nil {
		return nil, err
	}
	add(blunders)

	// 2. What the SOURCE TOOL marked (ADR-0006). The user already said, in
	// another program, that this one was interesting.
	if len(out) < limit {
		flagged, err := s.queueRows(ctx, scope, batchID, players, limit, domain.StudyFlagged,
			" AND "+s.DB.Bool("p.flagged", true), nil, " ORDER BY p.id ASC")
		if err != nil {
			return nil, err
		}
		add(flagged)
	}

	// 3. The close cube decisions: nothing was lost, but the right answer was
	// not obvious, which is exactly what is worth a second look.
	if len(out) < limit {
		close, err := s.queueRows(ctx, scope, batchID, players, limit, domain.StudyClose,
			" AND p.decision_type = 1 AND "+s.DB.Bool("a.is_close_cube", true), nil, " ORDER BY p.id ASC")
		if err != nil {
			return nil, err
		}
		add(close)
	}
	return out, nil
}

// queueCostOrder is the order of the passes that rank by cost, worst first.
const queueCostOrder = ` ORDER BY COALESCE(` + statsErrExpr + `, 0) DESC, p.id ASC`

// unhandledSQL keeps the positions nothing has dealt with: no comment, no
// Anki card, in no collection, not marked studied. Correlated on p.id, whose
// values already belong to the tenant being read.
const unhandledSQL = ` AND NOT EXISTS (SELECT 1 FROM comment c WHERE c.position_id = p.id)` +
	` AND NOT EXISTS (SELECT 1 FROM anki_card k WHERE k.position_id = p.id)` +
	` AND NOT EXISTS (SELECT 1 FROM collection_position cp WHERE cp.position_id = p.id)` +
	` AND NOT EXISTS (SELECT 1 FROM study_mark sm WHERE sm.position_id = p.id)`

// StudyBacklog is the cost pass of StudyQueue over the whole library, kept to
// what is still unhandled. It is the same query as the batch queue's first
// pass with the batch filter dropped and unhandledSQL added.
func (s *ImportBatchStore) StudyBacklog(ctx context.Context, scope string, players []string, limit int) ([]domain.StudyQueueEntry, error) {
	if limit <= 0 || limit > domain.MaxStudyQueue {
		limit = domain.MaxStudyQueue
	}
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return nil, fmt.Errorf("study backlog settings: %w", err)
	}
	// queueRows keeps one row per position for this reason (a position met
	// in several matches is studied once), so the limit counts positions.
	return s.queueRows(ctx, scope, 0, players, limit, domain.StudyBacklog,
		` AND COALESCE(`+statsErrExpr+`, 0) >= ?`+unhandledSQL, []any{settings.ErrorThresholdMP}, queueCostOrder)
}

func (s *ImportBatchStore) SetStudied(ctx context.Context, scope string, positionID int64, studied bool) error {
	ok, err := rowExists(ctx, s.DB, scope, "position", positionID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s: position %d: %w", s.DB.Name(), positionID, storage.ErrNotFound)
	}
	return s.DB.Transact(ctx, func(tx Execer) error {
		if !studied {
			tenant, targs := tx.TenantFilter("", scope)
			if _, err := tx.Exec(ctx, `DELETE FROM study_mark WHERE position_id = ? AND `+tenant,
				append([]any{positionID}, targs...)...); err != nil {
				return errf(tx, "withdraw study mark", err)
			}
			return nil
		}
		cols, args := tx.TenantColumns(scope)
		cols = append(cols, "position_id", "marked_at")
		args = append(args, positionID, time.Now().Unix())
		if _, err := tx.Exec(ctx, `INSERT INTO study_mark (`+strings.Join(cols, ", ")+`) VALUES (`+
			Placeholders(len(cols))+`) ON CONFLICT DO NOTHING`, args...); err != nil {
			return errf(tx, "mark position studied", err)
		}
		return nil
	})
}

// queueRows runs one of the queue's three passes. extraWhere and extraArgs are
// what makes a pass its own; everything else — the batch, the player filter,
// the counted-decision predicate, the label — is shared, so the three passes
// cannot disagree about what belongs to the batch.
func (s *ImportBatchStore) queueRows(ctx context.Context, scope string, batchID int64, players []string, limit int,
	reason domain.StudyQueueReason, extraWhere string, extraArgs []any, orderBy string) ([]domain.StudyQueueEntry, error) {
	tenant, targs := s.DB.TenantFilter("p", scope)
	playerClause, playerArgs, perr := s.playerClause(ctx, scope, players)
	if perr != nil {
		return nil, perr
	}
	batchClause := ""
	args := append([]any{}, targs...)
	if batchID > 0 {
		batchClause = " AND m.import_batch_id = ?"
		args = append(args, batchID)
	}
	args = append(args, playerArgs...)
	args = append(args, extraArgs...)
	limitSQL, largs := s.DB.LimitOffset(limit, 0)
	args = append(args, largs...)

	query := `SELECT DISTINCT p.id, m.id,
		        COALESCE(m.player1_name,''), COALESCE(m.player2_name,''), COALESCE(m.match_length, 0),
		        COALESCE(` + statsErrExpr + `, 0), p.decision_type
		 ` + statsBaseJoin + `
		 WHERE ` + tenant + batchClause + playerClause + `
		   AND ` + countedExpr(s.DB) + extraWhere + orderBy + limitSQL
	if reason == domain.StudyBacklog {
		// Across the whole library a position met in several matches joins
		// once per match; the window keeps its first match in SQL, so LIMIT
		// counts positions. The cost is the position's own, the same on
		// every row it keeps.
		query = `SELECT pid, mid, p1, p2, len, cost, dt FROM (
		   SELECT p.id AS pid, m.id AS mid,
		          COALESCE(m.player1_name,'') AS p1, COALESCE(m.player2_name,'') AS p2,
		          COALESCE(m.match_length, 0) AS len,
		          COALESCE(` + statsErrExpr + `, 0) AS cost, p.decision_type AS dt,
		          ROW_NUMBER() OVER (PARTITION BY p.id ORDER BY m.id) AS rn
		   ` + statsBaseJoin + `
		   WHERE ` + tenant + batchClause + playerClause + `
		     AND ` + countedExpr(s.DB) + extraWhere + `
		 ) q WHERE rn = 1 ORDER BY cost DESC, pid ASC` + limitSQL
	}
	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.StudyQueueEntry
	for rows.Next() {
		var e domain.StudyQueueEntry
		var p1, p2 string
		var length, decisionType int
		if err := rows.Scan(&e.PositionID, &e.MatchID, &p1, &p2, &length, &e.ErrorMP, &decisionType); err != nil {
			return nil, err
		}
		e.Reason = reason
		e.IsCube = decisionType == 1
		e.Label = matchLabel(p1, p2, length)
		// Only a blunder's cost means anything: a flagged position may have
		// been played perfectly, and showing it a "0" beside a cost would read
		// as a measurement rather than as an absence.
		if reason != domain.StudyBlunder && reason != domain.StudyBacklog {
			e.ErrorMP = 0
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// journalChunk bounds the rows of one INSERT, well under both backends'
// parameter limits.
const journalChunk = 100

// RecordFiles appends the files to the batch's journal.
func (s *ImportBatchStore) RecordFiles(ctx context.Context, scope string, batchID int64, files []domain.ImportFileEntry) error {
	if len(files) == 0 {
		return nil
	}
	tcols, targs := s.DB.TenantColumns(scope)
	cols := append(append([]string{}, tcols...), "batch_id", "path", "size", "mtime", "sha256", "outcome", "match_id", "error")
	one := "(" + strings.TrimSuffix(strings.Repeat("?, ", len(tcols)+3), ", ") + ", " + s.DB.TimestampArg() + ", ?, ?, ?, ?)"
	for start := 0; start < len(files); start += journalChunk {
		chunk := files[start:min(start+journalChunk, len(files))]
		var args []any
		rows := make([]string, len(chunk))
		for i, f := range chunk {
			rows[i] = one
			args = append(args, targs...)
			args = append(args, batchID, f.Path, f.Size)
			if f.MTime == "" {
				args = append(args, nil)
			} else {
				args = append(args, f.MTime+"+00")
			}
			var match any
			if f.MatchID != 0 {
				match = f.MatchID
			}
			args = append(args, f.SHA256, f.Outcome, match, f.Error)
		}
		if _, err := s.DB.Exec(ctx,
			`INSERT INTO import_batch_file (`+strings.Join(cols, ", ")+`) VALUES `+strings.Join(rows, ", "), args...); err != nil {
			return errf(s.DB, fmt.Sprintf("record files of import batch %d", batchID), err)
		}
	}
	return nil
}

// Files returns the batch's journal.
func (s *ImportBatchStore) Files(ctx context.Context, scope string, batchID int64) ([]domain.ImportFileEntry, error) {
	if _, err := s.Load(ctx, scope, batchID); err != nil {
		return nil, err
	}
	tenant, targs := s.DB.TenantFilter("", scope)
	rows, err := s.DB.Query(ctx,
		`SELECT path, size, `+s.DB.TimestampText("mtime")+`, sha256, outcome, COALESCE(match_id, 0), error
		 FROM import_batch_file WHERE batch_id = ? AND `+tenant+` ORDER BY id`,
		append([]any{batchID}, targs...)...)
	if err != nil {
		return nil, errf(s.DB, "list files of import batch", err)
	}
	defer rows.Close()
	var out []domain.ImportFileEntry
	for rows.Next() {
		var f domain.ImportFileEntry
		if err := rows.Scan(&f.Path, &f.Size, &f.MTime, &f.SHA256, &f.Outcome, &f.MatchID, &f.Error); err != nil {
			return nil, errf(s.DB, "list files of import batch", err)
		}
		// SQLite hands back what was written ("... +00"), PostgreSQL what it
		// renders: both reduce to the same second.
		if len(f.MTime) > 19 {
			f.MTime = f.MTime[:19]
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, errf(s.DB, "list files of import batch", err)
	}
	return out, nil
}
