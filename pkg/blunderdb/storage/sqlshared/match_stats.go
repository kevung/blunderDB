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
// What a row and its cells (match_stats_cells.go) depend on, and the write
// that must invalidate them:
//   - the match's moves (player, cube action, luck) and games: import,
//     replacement of the games, seat swap, deletion;
//   - the match's length (a cell's MWC when the position states none): set
//     at import and never edited; its date and tournament are joined at query time, not
//     stored, so they invalidate nothing;
//   - the analysis columns of every position the match reaches (cube_error,
//     best_move_equity_error, is_forced, is_close_cube, analysis_engine,
//     analysis_depth, best_cube_action, met_id): any analysis write that
//     changes one of them, for every match reaching the position
//     (InvalidateMatchStatsOfPositionsSQL) — the upsert clears met_id for a
//     verdict not gammonNet's (UpsertDropsMET), the MET assignment sets it;
//   - the position columns the rows and cells count by (decision_type,
//     cube_value, score_1, score_2, match_length, game_phase, game_type):
//     PositionStore.Update invalidates the matches reaching the position;
//     the phase reclassification and the Crawford repair wipe the table;
//   - the library's blunder threshold: LibrarySettingsStore.Save wipes it.
// Player names are not in the table: a rename or an alias changes no row.

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

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
// schema version wrote, or the rows an older schema left: before the error
// split, the Snowie parts and the MWC spread were columns (mwc_games and
// checker_moves are NULL there and nowhere else), or before the breakdown
// cells were written beside them (a seat with a counted decision and no
// phase cell, see match_stats_cells.go). Run at every open, after
// the columns exist; the matches it empties are then missing, recomputed by
// the next reader or the stats.matches maintenance line, so the repair is
// idempotent and costs one scan of the table once done.
const DropOlderShapeMatchStatsSQL = `DELETE FROM match_stats WHERE ` + OlderShapeMatchStatsPredicate

// OlderShapeMatchStatsPredicate selects those rows, for a probe that writes
// nothing when there is nothing to repair. Kind 1 is cellPhase.
const OlderShapeMatchStatsPredicate = `checker_moves IS NULL OR mwc_games IS NULL
	OR (decisions > 0 AND NOT EXISTS (SELECT 1 FROM match_stats_cell c
		WHERE c.match_id = match_stats.match_id AND c.seat = match_stats.seat AND c.kind = 1))`

// UpsertDropsMET reports whether the analysis upsert clears a stored met_id:
// it keeps the column only for a gammonNet verdict, and met_id is a
// dimension of the match_stats cells.
func UpsertDropsMET(storedMET bool, newEngine string) bool {
	return storedMET && !strings.HasPrefix(newEngine, "gammonNet")
}

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

	// Checker decisions with a position, analysed or not, but an unscored
	// play: the players table's Snowie denominator.
	if err := scanEach(ctx, d,
		`SELECT g.match_id, `+seatExpr+`, COUNT(*)
		 FROM move mv JOIN game g ON g.id = mv.game_id JOIN position p ON p.id = mv.position_id
		 WHERE g.match_id IN (`+ph+`) AND p.decision_type = 0 AND NOT `+UnscoredPlaySQL("p", "mv")+`
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

	// The MWC loss of the counted decisions, per seat and game, converted as
	// MatchBadges converts them: what the badges and the players table's
	// MWC7 read back instead of the decisions. Summed in move order, so a
	// recomputation (a seat swap, a refill) gives the same float to the bit.
	games := map[int64]*gameLosses{}
	if err := scanEach(ctx, d,
		`SELECT g.match_id, `+seatExpr+`, mv.game_id, `+statsErrExpr+`, COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), `+
			cubeMultiplierExpr+`, COALESCE(p.match_length, m.match_length, 0) `+statsBaseJoin+where+
			` ORDER BY g.match_id, mv.game_id, mv.id`,
		whereArgs, func(r Rows) error {
			var id, game, errMP int64
			var seat, away0, away1, cube, length int
			if err := r.Scan(&id, &seat, &game, &errMP, &away0, &away1, &cube, &length); err != nil {
				return err
			}
			gl := games[id]
			if gl == nil {
				gl = newGameLosses()
				games[id] = gl
			}
			gl.add(seat-1, game, decisionMWCLoss(errMP, away0, away1, cube, length))
			return nil
		}); err != nil {
		return nil, fmt.Errorf("match stats MWC: %w", err)
	}

	out := make([]storage.MatchStatsRow, 0, len(rows))
	for k, row := range rows {
		row.PR = pr(row.ErrorMP, row.Decisions)
		if gl := games[k.match]; gl != nil {
			row.MWCGames = len(gl.games)
			if seat := k.seat - 1; gl.priced[seat] {
				row.MWCPriced, row.MWCLoss = true, gl.total[seat]
				_, row.MWCGameSS = gl.spread(seat)
			}
		}
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
	err := s.DB.Transact(ctx, func(tx Execer) error {
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
			"checker_error_mp", "cube_error_mp", "errors", "snowie_error_mp", "snowie_moves", "checker_moves",
			"mwc_loss", "mwc_games", "mwc_game_ss")
		if err := insertRows(ctx, tx, "match_stats", cols, len(rows), func(i int) []any {
			r := rows[i]
			var prv, eng, depth, mwc, mwcSS any
			if r.Decisions > 0 {
				prv = r.PR
			}
			if r.MWCPriced {
				mwc, mwcSS = r.MWCLoss, r.MWCGameSS
			}
			if r.AnalysisEngine != "" {
				eng = r.AnalysisEngine
			}
			if r.AnalysisDepth != 0 {
				depth = r.AnalysisDepth
			}
			return append(append([]any{}, targs...), r.MatchID, r.Seat, r.Decisions, r.CheckerDecisions, r.CubeDecisions,
				r.ErrorMP, prv, r.LuckMP, r.LuckRolls, r.Blunders, eng, depth,
				r.CheckerErrorMP, r.CubeErrorMP, r.Errors, r.SnowieErrorMP, r.SnowieMoves, r.CheckerMoves,
				mwc, r.MWCGames, mwcSS)
		}); err != nil {
			return err
		}
		return writeMatchStatsCells(ctx, tx, scope, Placeholders(len(matchIDs)), int64Args(matchIDs), settings)
	})
	if LockContention(s.DB, err) && !errors.Is(err, storage.ErrBusy) {
		return fmt.Errorf("%w: match stats: %w", storage.ErrBusy, err)
	}
	return err
}

// readTransacter is implemented by an Execer whose plain transaction holds
// the write lock from its start (SQLite): TransactRead opens one that does
// not, for a transaction that only reads.
type readTransacter interface {
	TransactRead(ctx context.Context, fn func(Execer) error) error
}

// transactRead runs fn in a transaction that only reads.
func transactRead(ctx context.Context, db Execer, fn func(Execer) error) error {
	if r, ok := db.(readTransacter); ok {
		return r.TransactRead(ctx, fn)
	}
	return db.Transact(ctx, fn)
}

// fillKeyer is implemented by an Execer bound to a pool: FillKey names the
// table a fill of the scope writes, so that two fills of it can be told to
// be one. A transaction answers no.
type fillKeyer interface {
	FillKey(scope string) (any, bool)
}

// matchStatsFills holds the fills in flight, one per key.
var matchStatsFills = fillGate{running: map[any]chan struct{}{}}

// fillGate lets one fill of a scope run at a time. Concurrent readers of a
// cold library would each start the whole fill, rewrite the same matches and
// fight over the write lock; a reader arriving during a fill waits for it to
// end, then runs its own, which finds nothing missing — or picks up where a
// fill cut by its caller's cancellation stopped.
type fillGate struct {
	mu      sync.Mutex
	running map[any]chan struct{}
}

func (g *fillGate) do(ctx context.Context, key any, fn func() (int, error)) (int, error) {
	for {
		g.mu.Lock()
		done, busy := g.running[key]
		if !busy {
			done = make(chan struct{})
			g.running[key] = done
			g.mu.Unlock()
			break
		}
		g.mu.Unlock()
		select {
		case <-done:
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	defer func() {
		g.mu.Lock()
		close(g.running[key])
		delete(g.running, key)
		g.mu.Unlock()
	}()
	return fn()
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

// lockContender is implemented by an Execer that can tell a write lost to
// another writer from a real failure (SQLite's SQLITE_BUSY).
type lockContender interface {
	LockContention(err error) bool
}

// LockContention reports whether err is db's backend losing the write lock
// to another writer. A backend that cannot say answers no.
func LockContention(db Execer, err error) bool {
	c, ok := db.(lockContender)
	return ok && err != nil && c.LockContention(err)
}

// MissingMatchStats — see storage.StatsStore.
func (s *StatsStore) MissingMatchStats(ctx context.Context, scope string) ([]int64, error) {
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
	if k, ok := s.DB.(fillKeyer); ok {
		if key, ok := k.FillKey(scope); ok {
			// Every statistics read comes here: a filled library answers
			// from this probe without queueing behind the other readers.
			if ids, err := s.MissingMatchStats(ctx, scope); err != nil || len(ids) == 0 {
				return 0, err
			}
			return matchStatsFills.do(ctx, key, func() (int, error) { return s.fillMatchStats(ctx, scope, progress) })
		}
	}
	return s.fillMatchStats(ctx, scope, progress)
}

func (s *StatsStore) fillMatchStats(ctx context.Context, scope string, progress func(done, total int)) (int, error) {
	ids, err := s.MissingMatchStats(ctx, scope)
	if err != nil || len(ids) == 0 {
		return 0, err
	}
	beforeMatchStatsFill(len(ids))
	// A reader that cannot write must not answer from an incomplete table:
	// the figures would quietly leave out the matches it lacks.
	if RefusesWrites(ctx, s.DB) {
		return 0, fmt.Errorf("%w: %d match(es) have no statistics yet and the database is open read-only; open it for writing and run the stats.matches maintenance once", ErrMatchStatsIncomplete, len(ids))
	}
	settings, err := librarySettings(ctx, s.DB, scope)
	if err != nil {
		return 0, fmt.Errorf("match stats settings: %w", err)
	}
	// A batch can drop the rows of a match filled earlier whose private
	// position it reaches (sharePositions); the next round recomputes those
	// with the position shared, which drops nothing more, so the loop ends.
	filled := 0
	for len(ids) > 0 {
		for start := 0; start < len(ids); start += matchStatsBatch {
			if err := ctx.Err(); err != nil {
				return filled + start, err
			}
			end := min(start+matchStatsBatch, len(ids))
			if err := s.refreshMatchStatsBatch(ctx, scope, ids[start:end], settings); err != nil {
				return filled + start, err
			}
			if progress != nil {
				progress(end, len(ids))
			}
		}
		filled += len(ids)
		if ids, err = s.MissingMatchStats(ctx, scope); err != nil {
			return filled, err
		}
	}
	return filled, nil
}

// matchStatsReadable reports whether the scope's match_stats rows and their
// cells hold every match, so a read of them gives the figures the decisions
// would. A writer fills what is missing and answers yes. A reader that
// cannot write answers yes only when nothing is missing and no row is of an
// older shape, which the open would have repaired: a library filled once by
// a writer is then read from the table, read-only or not. That answer holds
// for the snapshot it was read in, so a reader asks it inside the
// transaction that then reads the table (withReadSnapshot): a writer
// invalidating a match between the two would otherwise drop it from the sums.
func (s *StatsStore) matchStatsReadable(ctx context.Context, scope string) (bool, error) {
	if !RefusesWrites(ctx, s.DB) {
		_, err := s.FillMatchStats(ctx, scope, nil)
		return err == nil, err
	}
	ids, err := s.MissingMatchStats(ctx, scope)
	if err != nil || len(ids) > 0 {
		return false, err
	}
	var olderShape bool
	if err := s.DB.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM match_stats WHERE `+OlderShapeMatchStatsPredicate+`)`).Scan(&olderShape); err != nil {
		return false, errf(s.DB, "match stats shape", err)
	}
	afterMatchStatsProbe()
	return !olderShape, nil
}

// matchStatsReadableFor is matchStatsReadable for a read of matchIDs alone:
// a writer fills the rows those matches lack, not the whole library's, so a
// page of the match list costs that page. A reader that cannot write answers
// yes only when none of them lacks its rows and the table holds no row of an
// older shape. No ids is the whole scope.
func (s *StatsStore) matchStatsReadableFor(ctx context.Context, scope string, matchIDs []int64) (bool, error) {
	if len(matchIDs) == 0 {
		return s.matchStatsReadable(ctx, scope)
	}
	missing, err := s.missingMatchStatsAmong(ctx, scope, matchIDs)
	if err != nil {
		return false, err
	}
	if RefusesWrites(ctx, s.DB) {
		if len(missing) > 0 {
			return false, nil
		}
		var olderShape bool
		if err := s.DB.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM match_stats WHERE `+OlderShapeMatchStatsPredicate+`)`).Scan(&olderShape); err != nil {
			return false, errf(s.DB, "match stats shape", err)
		}
		afterMatchStatsProbe()
		return !olderShape, nil
	}
	// A refresh can drop the rows of a match it shares a position with
	// (sharePositions); the next round recomputes them, which drops nothing
	// more, as in FillMatchStats.
	for len(missing) > 0 {
		if err := s.RefreshMatchStats(ctx, scope, missing); err != nil {
			return false, err
		}
		if missing, err = s.missingMatchStatsAmong(ctx, scope, matchIDs); err != nil {
			return false, err
		}
	}
	return true, nil
}

// missingMatchStatsAmong is MissingMatchStats restricted to matchIDs.
func (s *StatsStore) missingMatchStatsAmong(ctx context.Context, scope string, matchIDs []int64) ([]int64, error) {
	tenant, targs := s.DB.TenantFilter("m", scope)
	var ids []int64
	for start := 0; start < len(matchIDs); start += matchStatsBatch {
		chunk := matchIDs[start:min(start+matchStatsBatch, len(matchIDs))]
		err := scanEach(ctx, s.DB,
			`SELECT m.id FROM match m WHERE `+tenant+` AND m.id IN (`+Placeholders(len(chunk))+`)`+
				` AND NOT EXISTS (SELECT 1 FROM match_stats ms WHERE ms.match_id = m.id) ORDER BY m.id`,
			append(append([]any{}, targs...), int64Args(chunk)...), func(r Rows) error {
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
	}
	return ids, nil
}

// beforeMatchStatsFill runs when a fill finds matches to compute; a test
// counts the fills a burst of readers starts.
var beforeMatchStatsFill = func(missing int) {}

// afterMatchStatsProbe runs between a reader's completeness probe and its
// read; a test slips a writer's invalidation in there.
var afterMatchStatsProbe = func() {}

// withReadSnapshot runs fn on a store whose reads share one snapshot when
// the connection refuses writes, so matchStatsReadable's answer covers the
// reads that follow it. A writer runs fn as is: it fills the table first,
// which a read transaction could not hold.
func (s *StatsStore) withReadSnapshot(ctx context.Context, fn func(*StatsStore) error) error {
	if !RefusesWrites(ctx, s.DB) {
		return fn(s)
	}
	return transactRead(ctx, s.DB, func(tx Execer) error { return fn(&StatsStore{DB: tx}) })
}

// inReadSnapshot is withReadSnapshot for a reader returning a value.
func inReadSnapshot[T any](ctx context.Context, s *StatsStore, fn func(*StatsStore) (T, error)) (T, error) {
	var out T
	err := s.withReadSnapshot(ctx, func(s *StatsStore) error {
		var err error
		out, err = fn(s)
		return err
	})
	return out, err
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
		ms.checker_error_mp, ms.cube_error_mp, ms.errors, ms.snowie_error_mp, ms.snowie_moves, ms.checker_moves,
		COALESCE(ms.mwc_loss, 0), CASE WHEN ms.mwc_loss IS NULL THEN 0 ELSE 1 END, COALESCE(ms.mwc_games, 0), COALESCE(ms.mwc_game_ss, 0)
		FROM match_stats ms WHERE ` + tenant
	var out []storage.MatchStatsRow
	collect := func(query string, qargs []any) error {
		return scanEach(ctx, s.DB, query, qargs, func(r Rows) error {
			var row storage.MatchStatsRow
			var priced int
			if err := r.Scan(&row.MatchID, &row.Seat, &row.Decisions, &row.CheckerDecisions, &row.CubeDecisions, &row.ErrorMP,
				&row.PR, &row.LuckMP, &row.LuckRolls, &row.Blunders, &row.AnalysisEngine, &row.AnalysisDepth,
				&row.CheckerErrorMP, &row.CubeErrorMP, &row.Errors, &row.SnowieErrorMP, &row.SnowieMoves, &row.CheckerMoves,
				&row.MWCLoss, &priced, &row.MWCGames, &row.MWCGameSS); err != nil {
				return err
			}
			row.MWCPriced = priced == 1
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
	filter, err := s.withPlayerAliases(ctx, scope, filter)
	if err != nil {
		return nil, fmt.Errorf("match series aliases: %w", err)
	}
	var res storage.StatsResult
	err = s.withReadSnapshot(ctx, func(s *StatsStore) error {
		readable := false
		if fromMatchStats(filter) {
			var err error
			if readable, err = s.matchStatsReadable(ctx, scope); err != nil {
				return err
			}
		}
		if readable {
			if err := s.perMatchFromTable(ctx, statsQuery{scope: scope, filter: filter}, &res); err != nil {
				return errf(s.DB, "match series", err)
			}
			return nil
		}
		// A provenance filter, or a reader that cannot fill the table:
		// Compute's direct per-match pass alone, over the decisions.
		whereSQL, baseArgs := s.buildStatsWhereClause(scope, filter)
		q := statsQuery{scope: scope, filter: filter, whereSQL: whereSQL, baseArgs: baseArgs, join: statsBaseJoin}
		if err := s.computePerMatch(ctx, q, &res); err != nil {
			return errf(s.DB, "match series", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res.PerMatch, nil
}
