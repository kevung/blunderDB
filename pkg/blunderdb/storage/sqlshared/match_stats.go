package sqlshared

// match_stats.go — the materialised per-match, per-seat tallies.
//
// The table is derived: every row is recomputable from the moves, so the
// contract is "missing means stale". A match always has two rows once
// computed (a seat without a counted decision still gets its row, decisions
// 0, pr NULL), and any write that can change what a match's rows summarise
// deletes them instead of patching them. Readers fill what is missing before
// they read (ensureMatchStats). One computation, one invalidation shape: a
// patch path would be a second implementation of the arithmetic that could
// drift from Compute without a test noticing.
//
// What a row depends on, and the write that must invalidate it:
//   - the match's moves (player, cube action, luck) and games: import,
//     replacement of the games, seat swap, deletion;
//   - the analysis columns of every position the match reaches (cube_error,
//     best_move_equity_error, is_forced, is_close_cube, engine, depth): any
//     analysis write that changes one of them, for every match reaching the
//     position (InvalidateMatchStatsOfPositionsSQL);
//   - the position columns countedExpr reads (decision_type, cube_value,
//     scores): repairs, which wipe the table;
//   - the library's blunder threshold: LibrarySettingsStore.Save wipes it.
// Player names are not in the table: a rename or an alias changes no row.

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// matchStatsBatch bounds the matches one computation pass covers: the IN
// lists stay under every driver's parameter limit and a batch commits on its
// own, which is what makes FillMatchStats resumable.
const matchStatsBatch = 200

// InvalidateMatchStatsOfMatchesSQL drops the rows of the matches in the IN
// list that follows it (one placeholder per id). Plain SQL for the callers
// that run their own transaction (the Database wrapper's match edits).
const InvalidateMatchStatsOfMatchesSQL = `DELETE FROM match_stats WHERE match_id IN `

// InvalidateMatchStatsOfPositionsSQL drops the rows of every match that
// reaches one of the positions in the IN list that follows it.
const InvalidateMatchStatsOfPositionsSQL = `DELETE FROM match_stats WHERE match_id IN
	(SELECT g.match_id FROM move mv JOIN game g ON g.id = mv.game_id WHERE mv.position_id IN `

// InvalidateMatchStatsOfPositionsSuffix closes InvalidateMatchStatsOfPositionsSQL's subquery.
const InvalidateMatchStatsOfPositionsSuffix = `)`

// DropOlderShapeMatchStatsSQL deletes the rows an earlier build of the same
// schema version wrote, before the error split and the Snowie parts were
// columns: they are NULL there and nowhere else. Run at every open, after
// the columns exist; the fill that follows recomputes those matches, so the
// repair is idempotent and costs one scan of the table once done.
const DropOlderShapeMatchStatsSQL = `DELETE FROM match_stats WHERE checker_moves IS NULL`

// InvalidateMatchStats drops the rows of matchIDs, in batches.
func InvalidateMatchStats(ctx context.Context, db Execer, matchIDs []int64) error {
	return execInBatches(ctx, db, matchIDs, InvalidateMatchStatsOfMatchesSQL, "")
}

// InvalidateMatchStatsOfPositions drops the rows of every match reaching one
// of positionIDs, in batches.
func InvalidateMatchStatsOfPositions(ctx context.Context, db Execer, positionIDs []int64) error {
	return execInBatches(ctx, db, positionIDs, InvalidateMatchStatsOfPositionsSQL, InvalidateMatchStatsOfPositionsSuffix)
}

// InvalidateAllMatchStats drops every row of the scope: the next read
// recomputes the whole table.
func InvalidateAllMatchStats(ctx context.Context, db Execer, scope string) error {
	tenant, args := db.TenantFilter("", scope)
	if _, err := db.Exec(ctx, `DELETE FROM match_stats WHERE `+tenant, args...); err != nil {
		return errf(db, "invalidate match stats", err)
	}
	return nil
}

func execInBatches(ctx context.Context, db Execer, ids []int64, prefix, suffix string) error {
	for start := 0; start < len(ids); start += matchStatsBatch {
		chunk := ids[start:min(start+matchStatsBatch, len(ids))]
		q := prefix + "(" + Placeholders(len(chunk)) + ")" + suffix
		if _, err := db.Exec(ctx, q, int64Args(chunk)...); err != nil {
			return errf(db, "invalidate match stats", err)
		}
	}
	return nil
}

func int64Args(ids []int64) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}

// seatExpr maps move.player (1 / -1) to the table's seat (1 / 2).
const seatExpr = "CASE WHEN mv.player = 1 THEN 1 ELSE 2 END"

// computeMatchStatsRows is the direct calculation of the rows of matchIDs
// (the matches of the scope among them; at most matchStatsBatch). It reads
// the same join, counted predicate and error column as Compute.
func (s *StatsStore) computeMatchStatsRows(ctx context.Context, db Execer, scope string, matchIDs []int64, settings storage.LibrarySettings) ([]storage.MatchStatsRow, error) {
	d := db
	ph := Placeholders(len(matchIDs))
	ids := int64Args(matchIDs)

	mTenant, mArgs := d.TenantFilter("m", scope)
	existing := map[int64]bool{}
	if err := scanEach(ctx, d, `SELECT m.id FROM match m WHERE `+mTenant+` AND m.id IN (`+ph+`)`,
		append(mArgs, ids...), func(r Rows) error {
			var id int64
			if err := r.Scan(&id); err != nil {
				return err
			}
			existing[id] = true
			return nil
		}); err != nil {
		return nil, fmt.Errorf("match stats matches: %w", err)
	}

	type key struct {
		match int64
		seat  int
	}
	rows := map[key]*storage.MatchStatsRow{}
	for id := range existing {
		for _, seat := range []int{1, 2} {
			rows[key{id, seat}] = &storage.MatchStatsRow{MatchID: id, Seat: seat}
		}
	}

	pTenant, pArgs := d.TenantFilter("p", scope)
	analysed := ` WHERE ` + pTenant + ` AND g.match_id IN (` + ph + `) AND a.position_id IS NOT NULL AND (` + statsErrExpr + `) IS NOT NULL`
	where := analysed + ` AND ` + countedExpr(d)
	whereArgs := append(append([]any{}, pArgs...), ids...)

	if err := scanEach(ctx, d,
		`SELECT g.match_id, `+seatExpr+`, p.decision_type, `+d.Bigint(`SUM(`+statsErrExpr+`)`)+`, COUNT(*), `+
			d.Bigint(`SUM(CASE WHEN (`+statsErrExpr+`) >= ? THEN 1 ELSE 0 END)`)+`, `+
			d.Bigint(`SUM(CASE WHEN (`+statsErrExpr+`) >= ? THEN 1 ELSE 0 END)`)+` `+
			statsBaseJoin+where+` GROUP BY g.match_id, `+seatExpr+`, p.decision_type`,
		append([]any{settings.ErrorThresholdMP, settings.BlunderThresholdMP}, whereArgs...), func(r Rows) error {
			var id, sum, errs, blunders int64
			var seat, dt, n int
			if err := r.Scan(&id, &seat, &dt, &sum, &n, &errs, &blunders); err != nil {
				return err
			}
			row, ok := rows[key{id, seat}]
			if !ok {
				return nil
			}
			row.Decisions += n
			if dt == 1 {
				row.CubeDecisions += n
				row.CubeErrorMP += sum
			} else {
				row.CheckerDecisions += n
				row.CheckerErrorMP += sum
			}
			row.ErrorMP += sum
			row.Errors += int(errs)
			row.Blunders += int(blunders)
			return nil
		}); err != nil {
		return nil, fmt.Errorf("match stats decisions: %w", err)
	}

	// The Snowie rate's parts, over every analysed decision whether counted
	// or not (gnuBG formatgs.c:415-424): see computeSnowieGlobal.
	if err := scanEach(ctx, d,
		`SELECT g.match_id, `+seatExpr+`, `+d.Bigint(`SUM(`+statsErrExpr+`)`)+`, `+
			d.Bigint(`SUM(CASE WHEN p.decision_type = 0 THEN 1 ELSE 0 END)`)+` `+
			statsBaseJoin+analysed+` GROUP BY g.match_id, `+seatExpr,
		whereArgs, func(r Rows) error {
			var id, sum, moves int64
			var seat int
			if err := r.Scan(&id, &seat, &sum, &moves); err != nil {
				return err
			}
			if row, ok := rows[key{id, seat}]; ok {
				row.SnowieErrorMP, row.SnowieMoves = sum, int(moves)
			}
			return nil
		}); err != nil {
		return nil, fmt.Errorf("match stats snowie: %w", err)
	}

	// Checker decisions with a position, analysed or not: the players
	// table's Snowie denominator.
	if err := scanEach(ctx, d,
		`SELECT g.match_id, `+seatExpr+`, COUNT(*)
		 FROM move mv JOIN game g ON g.id = mv.game_id JOIN position p ON p.id = mv.position_id
		 WHERE g.match_id IN (`+ph+`) AND p.decision_type = 0
		 GROUP BY g.match_id, `+seatExpr,
		ids, func(r Rows) error {
			var id int64
			var seat, n int
			if err := r.Scan(&id, &seat, &n); err != nil {
				return err
			}
			if row, ok := rows[key{id, seat}]; ok {
				row.CheckerMoves = n
			}
			return nil
		}); err != nil {
		return nil, fmt.Errorf("match stats checker moves: %w", err)
	}

	// Provenance: the (engine, depth) most counted decisions of the seat
	// carry; ties go to the deeper analysis, then to the engine name, so the
	// pick does not depend on row order.
	type prov struct {
		engine string
		depth  int
		n      int
	}
	best := map[key]prov{}
	if err := scanEach(ctx, d,
		`SELECT g.match_id, `+seatExpr+`, COALESCE(a.analysis_engine, ''), COALESCE(a.analysis_depth, 0), COUNT(*) `+
			statsBaseJoin+where+` GROUP BY g.match_id, `+seatExpr+`, a.analysis_engine, a.analysis_depth`,
		whereArgs, func(r Rows) error {
			var id int64
			var p prov
			var seat int
			if err := r.Scan(&id, &seat, &p.engine, &p.depth, &p.n); err != nil {
				return err
			}
			k := key{id, seat}
			cur, ok := best[k]
			if !ok || p.n > cur.n || (p.n == cur.n && (p.depth > cur.depth || (p.depth == cur.depth && p.engine < cur.engine))) {
				best[k] = p
			}
			return nil
		}); err != nil {
		return nil, fmt.Errorf("match stats provenance: %w", err)
	}

	// Luck over every roll of the seat that carries it, analysed or not:
	// the denominator is the rolls measured (ADR-0010), as in PlayerTable.
	if err := scanEach(ctx, d,
		`SELECT g.match_id, `+seatExpr+`, `+d.Bigint(`SUM(mv.luck_mp)`)+`, COUNT(mv.luck_mp)
		 FROM move mv JOIN game g ON g.id = mv.game_id
		 WHERE g.match_id IN (`+ph+`) AND mv.luck_mp IS NOT NULL
		 GROUP BY g.match_id, `+seatExpr,
		ids, func(r Rows) error {
			var id, sum int64
			var seat, n int
			if err := r.Scan(&id, &seat, &sum, &n); err != nil {
				return err
			}
			if row, ok := rows[key{id, seat}]; ok {
				row.LuckMP, row.LuckRolls = sum, n
			}
			return nil
		}); err != nil {
		return nil, fmt.Errorf("match stats luck: %w", err)
	}

	out := make([]storage.MatchStatsRow, 0, len(rows))
	for k, row := range rows {
		row.PR = pr(row.ErrorMP, row.Decisions)
		if p, ok := best[k]; ok {
			row.AnalysisEngine, row.AnalysisDepth = p.engine, p.depth
		}
		out = append(out, *row)
	}
	sortMatchStatsRows(out)
	return out, nil
}

func sortMatchStatsRows(rows []storage.MatchStatsRow) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].MatchID != rows[j].MatchID {
			return rows[i].MatchID < rows[j].MatchID
		}
		return rows[i].Seat < rows[j].Seat
	})
}

// scanEach runs a query and hands every row to fn.
func scanEach(ctx context.Context, db Execer, query string, args []any, fn func(Rows) error) error {
	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := fn(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// refreshMatchStatsBatch recomputes and replaces the rows of one batch of
// matches in a single transaction.
func (s *StatsStore) refreshMatchStatsBatch(ctx context.Context, scope string, matchIDs []int64, settings storage.LibrarySettings) error {
	return s.DB.Transact(ctx, func(tx Execer) error {
		rows, err := s.computeMatchStatsRows(ctx, tx, scope, matchIDs, settings)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, InvalidateMatchStatsOfMatchesSQL+"("+Placeholders(len(matchIDs))+")", int64Args(matchIDs)...); err != nil {
			return fmt.Errorf("match stats delete: %w", err)
		}
		tcols, targs := tx.TenantColumns(scope)
		cols := append(append([]string{}, tcols...), "match_id", "seat", "decisions", "checker_decisions", "cube_decisions",
			"error_mp", "pr", "luck_mp", "luck_rolls", "blunders", "analysis_engine", "analysis_depth",
			"checker_error_mp", "cube_error_mp", "errors", "snowie_error_mp", "snowie_moves", "checker_moves")
		insert := `INSERT INTO match_stats (` + strings.Join(cols, ", ") + `) VALUES (` + Placeholders(len(cols)) + `)`
		for _, r := range rows {
			var prv, eng, depth any
			if r.Decisions > 0 {
				prv = r.PR
			}
			if r.AnalysisEngine != "" {
				eng = r.AnalysisEngine
			}
			if r.AnalysisDepth != 0 {
				depth = r.AnalysisDepth
			}
			args := append(append([]any{}, targs...), r.MatchID, r.Seat, r.Decisions, r.CheckerDecisions, r.CubeDecisions,
				r.ErrorMP, prv, r.LuckMP, r.LuckRolls, r.Blunders, eng, depth,
				r.CheckerErrorMP, r.CubeErrorMP, r.Errors, r.SnowieErrorMP, r.SnowieMoves, r.CheckerMoves)
			if _, err := tx.Exec(ctx, insert, args...); err != nil {
				return fmt.Errorf("match stats insert: %w", err)
			}
		}
		return nil
	})
}

// RefreshMatchStats recomputes the rows of matchIDs now — what an import
// calls on the match it has just written.
func (s *StatsStore) RefreshMatchStats(ctx context.Context, scope string, matchIDs []int64) error {
	if len(matchIDs) == 0 {
		return nil
	}
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return fmt.Errorf("match stats settings: %w", err)
	}
	for start := 0; start < len(matchIDs); start += matchStatsBatch {
		if err := s.refreshMatchStatsBatch(ctx, scope, matchIDs[start:min(start+matchStatsBatch, len(matchIDs))], settings); err != nil {
			return err
		}
	}
	return nil
}

// ErrMatchStatsIncomplete reports a match_stats table missing rows that a
// read-only connection cannot fill.
var ErrMatchStatsIncomplete = errors.New("match statistics incomplete")

// writeRefuser is implemented by an Execer whose connection may refuse writes
// (SQLite's read-only fallback sets query_only).
type writeRefuser interface {
	RefusesWrites(ctx context.Context) bool
}

// RefusesWrites reports whether db's connection refuses writes. A backend
// that cannot say is taken to accept them.
func RefusesWrites(ctx context.Context, db Execer) bool {
	w, ok := db.(writeRefuser)
	return ok && w.RefusesWrites(ctx)
}

// missingMatchStats lists the matches of the scope without rows.
func (s *StatsStore) missingMatchStats(ctx context.Context, scope string) ([]int64, error) {
	tenant, args := s.DB.TenantFilter("m", scope)
	var ids []int64
	err := scanEach(ctx, s.DB,
		`SELECT m.id FROM match m WHERE `+tenant+
			` AND NOT EXISTS (SELECT 1 FROM match_stats ms WHERE ms.match_id = m.id) ORDER BY m.id`,
		args, func(r Rows) error {
			var id int64
			if err := r.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
			return nil
		})
	if err != nil {
		return nil, errf(s.DB, "missing match stats", err)
	}
	return ids, nil
}

// FillMatchStats — see storage.StatsStore.
func (s *StatsStore) FillMatchStats(ctx context.Context, scope string, progress func(done, total int)) (int, error) {
	ids, err := s.missingMatchStats(ctx, scope)
	if err != nil || len(ids) == 0 {
		return 0, err
	}
	// A reader that cannot write must not answer from an incomplete table:
	// the figures would quietly leave out the matches it lacks.
	if RefusesWrites(ctx, s.DB) {
		return 0, fmt.Errorf("%w: %d match(es) have no statistics yet and the database is open read-only; open it for writing once", ErrMatchStatsIncomplete, len(ids))
	}
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return 0, fmt.Errorf("match stats settings: %w", err)
	}
	for start := 0; start < len(ids); start += matchStatsBatch {
		if err := ctx.Err(); err != nil {
			return start, err
		}
		end := min(start+matchStatsBatch, len(ids))
		if err := s.refreshMatchStatsBatch(ctx, scope, ids[start:end], settings); err != nil {
			return start, err
		}
		if progress != nil {
			progress(end, len(ids))
		}
	}
	return len(ids), nil
}

// RebuildMatchStats — see storage.StatsStore.
func (s *StatsStore) RebuildMatchStats(ctx context.Context, scope string, progress func(done, total int)) (int, error) {
	if err := InvalidateAllMatchStats(ctx, s.DB, scope); err != nil {
		return 0, err
	}
	return s.FillMatchStats(ctx, scope, progress)
}

// MatchStats — see storage.StatsStore.
func (s *StatsStore) MatchStats(ctx context.Context, scope string, matchIDs []int64) ([]storage.MatchStatsRow, error) {
	if _, err := s.FillMatchStats(ctx, scope, nil); err != nil {
		return nil, err
	}
	tenant, args := s.DB.TenantFilter("ms", scope)
	q := `SELECT ms.match_id, ms.seat, ms.decisions, ms.checker_decisions, ms.cube_decisions, ms.error_mp,
		COALESCE(ms.pr, 0), ms.luck_mp, ms.luck_rolls, ms.blunders, COALESCE(ms.analysis_engine, ''), COALESCE(ms.analysis_depth, 0),
		ms.checker_error_mp, ms.cube_error_mp, ms.errors, ms.snowie_error_mp, ms.snowie_moves, ms.checker_moves
		FROM match_stats ms WHERE ` + tenant
	var out []storage.MatchStatsRow
	collect := func(query string, qargs []any) error {
		return scanEach(ctx, s.DB, query, qargs, func(r Rows) error {
			var row storage.MatchStatsRow
			if err := r.Scan(&row.MatchID, &row.Seat, &row.Decisions, &row.CheckerDecisions, &row.CubeDecisions, &row.ErrorMP,
				&row.PR, &row.LuckMP, &row.LuckRolls, &row.Blunders, &row.AnalysisEngine, &row.AnalysisDepth,
				&row.CheckerErrorMP, &row.CubeErrorMP, &row.Errors, &row.SnowieErrorMP, &row.SnowieMoves, &row.CheckerMoves); err != nil {
				return err
			}
			out = append(out, row)
			return nil
		})
	}
	if len(matchIDs) == 0 {
		if err := collect(q, args); err != nil {
			return nil, errf(s.DB, "match stats", err)
		}
	} else {
		for start := 0; start < len(matchIDs); start += matchStatsBatch {
			chunk := matchIDs[start:min(start+matchStatsBatch, len(matchIDs))]
			if err := collect(q+` AND ms.match_id IN (`+Placeholders(len(chunk))+`)`, append(append([]any{}, args...), int64Args(chunk)...)); err != nil {
				return nil, errf(s.DB, "match stats", err)
			}
		}
	}
	sortMatchStatsRows(out)
	return out, nil
}

// MatchSeries — see storage.StatsStore.
func (s *StatsStore) MatchSeries(ctx context.Context, scope string, filter storage.StatsFilter) ([]storage.MatchStats, error) {
	if !fromMatchStats(filter) {
		return nil, fmt.Errorf("match series with a provenance filter: %w", storage.ErrInvalid)
	}
	if _, err := s.FillMatchStats(ctx, scope, nil); err != nil {
		return nil, err
	}
	var res storage.StatsResult
	if err := s.perMatchFromTable(ctx, statsQuery{scope: scope, filter: filter}, &res); err != nil {
		return nil, errf(s.DB, "match series", err)
	}
	return res.PerMatch, nil
}
