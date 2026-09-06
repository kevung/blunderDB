package sqlshared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// ImportBatchStore implements storage.ImportBatchStore. import_batch is a
// domain table: every statement is confined to the scope's tenant through
// Dialect.TenantFilter / TenantColumns.
//
// The SQL is the same on both backends, which is why it is here: the report's
// aggregates reuse the statistics' own predicates (countedExpr, statsErrExpr,
// statsBaseJoin) rather than restating what counts as a decision — a second
// definition of "a decision that counts" would be a second PR.
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
	playerClause, playerArgs := batchPlayerClause(players)
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
// It reads the SAME error column the statistics read (statsErrExpr) and the
// same counted predicate, so the worst decision of an import is the worst
// decision of the statistics restricted to it — and not a fifth definition of
// "expensive".
func (s *ImportBatchStore) measureWorst(ctx context.Context, scope string, b *domain.ImportBatch, players []string) error {
	tenant, targs := s.DB.TenantFilter("p", scope)
	playerClause, playerArgs := batchPlayerClause(players)
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
	if limit <= 0 || limit > domain.MaxStudyQueue {
		limit = domain.MaxStudyQueue
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
	blunders, err := s.queueRows(ctx, scope, batchID, players, limit, domain.StudyBlunder,
		` AND COALESCE(`+statsErrExpr+`, 0) >= ?`, []any{domain.StudyBlunderThresholdMP},
		` ORDER BY COALESCE(`+statsErrExpr+`, 0) DESC, p.id ASC`)
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

// queueRows runs one of the queue's three passes. extraWhere and extraArgs are
// what makes a pass its own; everything else — the batch, the player filter,
// the counted-decision predicate, the label — is shared, so the three passes
// cannot disagree about what belongs to the batch.
func (s *ImportBatchStore) queueRows(ctx context.Context, scope string, batchID int64, players []string, limit int,
	reason domain.StudyQueueReason, extraWhere string, extraArgs []any, orderBy string) ([]domain.StudyQueueEntry, error) {
	tenant, targs := s.DB.TenantFilter("p", scope)
	playerClause, playerArgs := batchPlayerClause(players)
	args := append(append([]any{}, targs...), batchID)
	args = append(args, playerArgs...)
	args = append(args, extraArgs...)
	limitSQL, largs := s.DB.LimitOffset(limit, 0)
	args = append(args, largs...)

	rows, err := s.DB.Query(ctx,
		`SELECT DISTINCT p.id, m.id,
		        COALESCE(m.player1_name,''), COALESCE(m.player2_name,''), COALESCE(m.match_length, 0),
		        COALESCE(`+statsErrExpr+`, 0), p.decision_type
		 `+statsBaseJoin+`
		 WHERE `+tenant+` AND m.import_batch_id = ?`+playerClause+`
		   AND `+countedExpr(s.DB)+extraWhere+orderBy+limitSQL, args...)
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
		if reason != domain.StudyBlunder {
			e.ErrorMP = 0
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
