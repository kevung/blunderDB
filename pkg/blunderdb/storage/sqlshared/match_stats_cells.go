package sqlshared

// match_stats_cells.go — the per-match breakdowns written beside match_stats.
//
// match_stats_cell holds the counted decisions of a match, per seat, decision
// type and table of the analysis (met_id, 0 for the built-in one), grouped by
// one dimension per kind: the phase, the plan of play, the away × away score,
// the (best, played) cube actions, the error-histogram bucket. A row carries
// the decisions, their error sum, the largest error, the blunders at the
// library's threshold and the sum of their MWC losses. match_stats_position
// lists the positions those decisions reach: the distinct-position count is
// the one figure no sum of cells gives.
//
// Both tables are written in the transaction that writes the match's
// match_stats rows, from the same join and counted predicate, and reference
// those rows ON DELETE CASCADE: every invalidation of match_stats drops them
// with it, so they share its "missing means stale" contract and its one fill
// path without an invalidation site of their own. A seat with a counted
// decision always has its phase cells, which is what lets the open tell a
// match_stats row written before the cells existed (DropOlderShapeMatchStatsSQL).
//
// A dimension whose column is NULL is stored as cellNull, so that NULL and
// 0 stay apart as the direct GROUP BY keeps them; the readers map it back.

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The kinds of match_stats_cell. Every counted decision has exactly one row
// of each kind, the cube kind excepted (cube decisions only).
const (
	cellPhase    = 1
	cellGameType = 2
	cellScore    = 3
	cellCube     = 4
	cellBucket   = 5
)

// cellNull stands for a NULL dimension column; cellText for an action
// column still holding text (”), which a label reads as no action, as it
// reads NULL, but which the direct GROUP BY keeps apart from both.
const (
	cellNull = -1
	cellText = -2
)

// cellRowsPerInsert bounds the rows of one multi-row INSERT: thirteen columns
// a row keeps the statement far below every driver's parameter limit.
const cellRowsPerInsert = 200

// errorBucket is the histogram bucket of an error, in millipoints: the lower
// bound of [0,5) [5,10) [10,25) [25,50) [50,100) [100,∞).
func errorBucket(errMP int64) int64 {
	switch {
	case errMP < 5:
		return 0
	case errMP < 10:
		return 5
	case errMP < 25:
		return 10
	case errMP < 50:
		return 25
	case errMP < 100:
		return 50
	}
	return 100
}

// decisionMWCLoss converts one decision's EMG error into the match winning
// chances it cost, NaN where the table has no value (money, a score outside
// the match). awayScore0/1 are the position's stored away scores, rawPlayer
// move.player, cubeValue the cube multiplier.
func decisionMWCLoss(errMP int64, awayScore0, awayScore1, rawPlayer, cubeValue, matchLength int) float64 {
	// XG encodes player 0 (bottom) as 1 and player 1 (top) as -1;
	// gnuBG fMove is 0 or 1.
	fMove := 0
	if rawPlayer == -1 {
		fMove = 1
	}
	// p.score_1/score_2 are away scores; ConvertEMGLossToMWCLoss expects
	// current scores (games already won). domain.PointsAway decodes the
	// Crawford sentinel first: a stored 0 is one point away, post-Crawford,
	// and subtracting it raw says "has already won" — a score the MET refuses,
	// silently dropping the row.
	currentScore0 := matchLength - domain.PointsAway(awayScore0)
	currentScore1 := matchLength - domain.PointsAway(awayScore1)
	return engine.ConvertEMGLossToMWCLoss(int(errMP), currentScore0, currentScore1, fMove, cubeValue, matchLength)
}

type cellKey struct {
	match    int64
	seat, dt int
	met      int64
	kind     int
	k1, k2   int64
}

type cellSums struct {
	decisions, blunders int64
	errMP, maxErr       int64
	mwc                 float64
	mwcDecisions        int64
	positions           int64 // distinct private positions, phase cells only
}

type cellPosition struct {
	match    int64
	seat, dt int
	met      int64
	position int64
}

// nullable maps a scanned nullable integer to its stored dimension.
func nullable(v *int64) int64 {
	if v == nil {
		return cellNull
	}
	return *v
}

// actionDim maps a scanned action-code column to its stored dimension.
func actionDim(v any) int64 {
	switch x := v.(type) {
	case nil:
		return cellNull
	case int64:
		return x
	case int32:
		return int64(x)
	case int:
		return int64(x)
	}
	return cellText
}

// writeMatchStatsCells computes the cells and positions of the matches in
// the IN list ph (bound by ids) and inserts them; match_stats rows of those
// matches must already be written in the same transaction.
func writeMatchStatsCells(ctx context.Context, tx Execer, scope, ph string, ids []any, settings storage.LibrarySettings) error {
	d := tx
	pTenant, pArgs := d.TenantFilter("p", scope)
	where := ` WHERE ` + pTenant + ` AND g.match_id IN (` + ph + `) AND a.position_id IS NOT NULL AND (` + statsErrExpr + `) IS NOT NULL AND ` + countedExpr(d)
	cells := map[cellKey]*cellSums{}
	positions := map[cellPosition]int64{} // → its phase dimension
	add := func(k cellKey, errMP int64, loss float64) {
		c := cells[k]
		if c == nil {
			c = &cellSums{maxErr: errMP}
			cells[k] = c
		}
		c.decisions++
		c.errMP += errMP
		if errMP > c.maxErr {
			c.maxErr = errMP
		}
		if errMP >= int64(settings.BlunderThresholdMP) {
			c.blunders++
		}
		if !math.IsNaN(loss) {
			c.mwc += loss
			c.mwcDecisions++
		}
	}
	err := scanEach(ctx, d,
		`SELECT g.match_id, `+seatExpr+`, p.decision_type, COALESCE(a.met_id, 0), p.id, p.game_phase, p.game_type,
			p.score_1, p.score_2, a.best_cube_action, mv.cube_action, (`+statsErrExpr+`), COALESCE(mv.player, 0),
			`+cubeMultiplierExpr+`, COALESCE(p.match_length, m.match_length, 0) `+
			statsBaseJoin+where,
		append(append([]any{}, pArgs...), ids...), func(r Rows) error {
			var match, met, pos, errMP int64
			var seat, dt, rawPlayer, cubeValue, matchLength int
			var phase, gameType, score1, score2 *int64
			var best, played any
			if err := r.Scan(&match, &seat, &dt, &met, &pos, &phase, &gameType, &score1, &score2, &best, &played,
				&errMP, &rawPlayer, &cubeValue, &matchLength); err != nil {
				return err
			}
			var away0, away1 int
			if score1 != nil {
				away0 = int(*score1)
			}
			if score2 != nil {
				away1 = int(*score2)
			}
			loss := decisionMWCLoss(errMP, away0, away1, rawPlayer, cubeValue, matchLength)
			base := cellKey{match: match, seat: seat, dt: dt, met: met}
			for _, k := range []struct {
				kind   int
				k1, k2 int64
			}{
				{cellPhase, nullable(phase), 0},
				{cellGameType, nullable(gameType), 0},
				{cellScore, nullable(score1), nullable(score2)},
				{cellBucket, errorBucket(errMP), 0},
			} {
				base.kind, base.k1, base.k2 = k.kind, k.k1, k.k2
				add(base, errMP, loss)
			}
			if dt == 1 {
				base.kind, base.k1, base.k2 = cellCube, actionDim(best), actionDim(played)
				add(base, errMP, loss)
			}
			positions[cellPosition{match: match, seat: seat, dt: dt, met: met, position: pos}] = nullable(phase)
			return nil
		})
	if err != nil {
		return fmt.Errorf("match stats cells: %w", err)
	}

	shared, err := sharePositions(ctx, tx, ph, ids, positions)
	if err != nil {
		return err
	}
	for p, phase := range positions {
		if !shared[p] {
			cells[cellKey{match: p.match, seat: p.seat, dt: p.dt, met: p.met, kind: cellPhase, k1: phase}].positions++
		}
	}

	tcols, targs := tx.TenantColumns(scope)
	cellCols := append(append([]string{}, tcols...), "match_id", "seat", "decision_type", "met_id", "kind", "k1", "k2",
		"decisions", "error_mp", "max_error_mp", "blunders", "mwc_loss", "mwc_decisions", "positions")
	keys := make([]cellKey, 0, len(cells))
	for k := range cells {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return lessCell(keys[i], keys[j]) })
	if err := insertRows(ctx, tx, "match_stats_cell", cellCols, len(keys), func(i int) []any {
		k, c := keys[i], cells[keys[i]]
		return append(append([]any{}, targs...), k.match, k.seat, k.dt, k.met, k.kind, k.k1, k.k2,
			c.decisions, c.errMP, c.maxErr, c.blunders, c.mwc, c.mwcDecisions, c.positions)
	}); err != nil {
		return err
	}

	posCols := append(append([]string{}, tcols...), "match_id", "seat", "position_id", "decision_type", "met_id")
	pos := make([]cellPosition, 0, len(shared))
	for p := range shared {
		pos = append(pos, p)
	}
	sort.Slice(pos, func(i, j int) bool {
		a, b := pos[i], pos[j]
		if a.match != b.match {
			return a.match < b.match
		}
		if a.seat != b.seat {
			return a.seat < b.seat
		}
		if a.position != b.position {
			return a.position < b.position
		}
		return a.met < b.met
	})
	return insertRows(ctx, tx, "match_stats_position", posCols, len(pos), func(i int) []any {
		p := pos[i]
		return append(append([]any{}, targs...), p.match, p.seat, p.position, p.dt, p.met)
	})
}

type matchSeat struct {
	match int64
	seat  int
}

// sharePositions decides which of the batch's counted positions are shared:
// referenced by a move of another match or seat. The others are private —
// every move reaching them belongs to that one seat — and are only counted
// in their phase cell; the shared ones get a match_stats_position row, so a
// distinct count over those rows plus the private counts is the number of
// distinct positions of any selection.
//
// A private position turns shared when another match's moves reach it.
// Before this batch, a position no filled seat lists as shared was private
// to at most one filled seat (the induction this function keeps); when the
// batch reaches such a position from outside that seat, the seat's figures
// are stale and its match's match_stats rows are dropped, for the fill to
// recompute — this time with the position shared. A position already listed
// as shared needs nothing: nobody holds it private.
func sharePositions(ctx context.Context, tx Execer, ph string, ids []any, positions map[cellPosition]int64) (map[cellPosition]bool, error) {
	shared := map[cellPosition]bool{}
	if len(positions) == 0 {
		return shared, nil
	}
	// The batch's own references, per position and seat.
	own := map[int64]map[matchSeat]int64{}
	if err := scanEach(ctx, tx,
		`SELECT mv.position_id, g.match_id, `+seatExpr+`, COUNT(*) FROM move mv JOIN game g ON g.id = mv.game_id
		 WHERE g.match_id IN (`+ph+`) AND mv.position_id IS NOT NULL GROUP BY mv.position_id, g.match_id, `+seatExpr,
		ids, func(r Rows) error {
			var pos, n int64
			var ms matchSeat
			if err := r.Scan(&pos, &ms.match, &ms.seat, &n); err != nil {
				return err
			}
			if own[pos] == nil {
				own[pos] = map[matchSeat]int64{}
			}
			own[pos][ms] = n
			return nil
		}); err != nil {
		return nil, fmt.Errorf("match stats position references: %w", err)
	}
	distinct := map[int64]bool{}
	for p := range positions {
		distinct[p.position] = true
	}
	list := make([]int64, 0, len(distinct))
	for p := range distinct {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool { return list[i] < list[j] })

	// Every reference, from the move index alone.
	total := map[int64]int64{}
	if err := eachChunk(ctx, tx, list, `SELECT position_id, COUNT(*) FROM move WHERE position_id IN `, ` GROUP BY position_id`, func(r Rows) error {
		var pos, n int64
		if err := r.Scan(&pos, &n); err != nil {
			return err
		}
		total[pos] = n
		return nil
	}); err != nil {
		return nil, fmt.Errorf("match stats position total references: %w", err)
	}

	var outside []int64
	for _, p := range list {
		var inBatch int64
		for _, n := range own[p] {
			inBatch += n
		}
		if total[p] > inBatch {
			outside = append(outside, p)
		}
	}
	for p := range positions {
		if own[p.position][matchSeat{p.match, p.seat}] < total[p.position] {
			shared[p] = true
		}
	}
	if len(outside) == 0 {
		return shared, nil
	}

	listed := map[int64]bool{}
	if err := eachChunk(ctx, tx, outside, `SELECT DISTINCT position_id FROM match_stats_position WHERE position_id IN `, ``, func(r Rows) error {
		var pos int64
		if err := r.Scan(&pos); err != nil {
			return err
		}
		listed[pos] = true
		return nil
	}); err != nil {
		return nil, fmt.Errorf("match stats listed positions: %w", err)
	}
	var unlisted []int64
	for _, p := range outside {
		if !listed[p] {
			unlisted = append(unlisted, p)
		}
	}
	batch := map[int64]bool{}
	for _, id := range ids {
		batch[id.(int64)] = true
	}
	stale := map[int64]bool{}
	if err := eachChunk(ctx, tx, unlisted,
		`SELECT DISTINCT g.match_id FROM move mv JOIN game g ON g.id = mv.game_id
		 JOIN match_stats ms ON ms.match_id = g.match_id AND ms.seat = 1 WHERE mv.position_id IN `, ``, func(r Rows) error {
			var id int64
			if err := r.Scan(&id); err != nil {
				return err
			}
			if !batch[id] {
				stale[id] = true
			}
			return nil
		}); err != nil {
		return nil, fmt.Errorf("match stats stale seats: %w", err)
	}
	staleIDs := make([]int64, 0, len(stale))
	for id := range stale {
		staleIDs = append(staleIDs, id)
	}
	sort.Slice(staleIDs, func(i, j int) bool { return staleIDs[i] < staleIDs[j] })
	return shared, InvalidateMatchStats(ctx, tx, staleIDs)
}

// eachChunk runs prefix (ids) suffix over ids, byPositionsChunk at a time.
func eachChunk(ctx context.Context, db Execer, ids []int64, prefix, suffix string, fn func(Rows) error) error {
	for start := 0; start < len(ids); start += byPositionsChunk {
		chunk := ids[start:min(start+byPositionsChunk, len(ids))]
		if err := scanEach(ctx, db, prefix+"("+Placeholders(len(chunk))+")"+suffix, int64Args(chunk), fn); err != nil {
			return err
		}
	}
	return nil
}

func lessCell(a, b cellKey) bool {
	for _, c := range [][2]int64{
		{a.match, b.match}, {int64(a.seat), int64(b.seat)}, {int64(a.dt), int64(b.dt)}, {a.met, b.met},
		{int64(a.kind), int64(b.kind)}, {a.k1, b.k1}, {a.k2, b.k2},
	} {
		if c[0] != c[1] {
			return c[0] < c[1]
		}
	}
	return false
}

// insertRows inserts n rows into table, cellRowsPerInsert per statement.
func insertRows(ctx context.Context, tx Execer, table string, cols []string, n int, row func(i int) []any) error {
	one := "(" + Placeholders(len(cols)) + ")"
	head := `INSERT INTO ` + table + ` (` + strings.Join(cols, ", ") + `) VALUES `
	for start := 0; start < n; start += cellRowsPerInsert {
		end := min(start+cellRowsPerInsert, n)
		args := make([]any, 0, (end-start)*len(cols))
		for i := start; i < end; i++ {
			args = append(args, row(i)...)
		}
		q := head + strings.TrimSuffix(strings.Repeat(one+",", end-start), ",")
		if _, err := tx.Exec(ctx, q, args...); err != nil {
			return fmt.Errorf("%s insert: %w", table, err)
		}
	}
	return nil
}
