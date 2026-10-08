package sqlshared

import (
	"context"
	"fmt"
	"sort"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The breakdowns: the same PR the statistics already show, sliced by game
// phase, plan of play, tag, and away × away score. Each reuses countedExpr and
// statsErrExpr (stated once, in stats.go) and adds only a GROUP BY, or, for
// tags, a second query.

// computePerPhase splits the selection by the position's derived phase
// (ADR-0035). A database whose phases have never been computed reports
// everything as "unknown", which is honest: the column says so, and
// `blunderdb repair` is what fills it.
func (s *StatsStore) computePerPhase(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	rows, err := s.breakdownRows(ctx, q, "p.game_phase")
	if err != nil {
		return fmt.Errorf("per-phase query: %w", err)
	}
	for _, r := range rows {
		result.PerPhase = append(result.PerPhase, storage.PhaseStats{
			Phase: domain.GamePhase(r.k1).String(), PR: pr(r.sumErr, int(r.n)), PRInterval: r.interval,
			NumDecisions: int(r.n), BlunderCount: int(r.blunders),
		})
	}
	return nil
}

// computePerGameType splits the selection by the position's derived plan of
// play. Same shape as computePerPhase.
func (s *StatsStore) computePerGameType(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	rows, err := s.breakdownRows(ctx, q, "p.game_type")
	if err != nil {
		return fmt.Errorf("per-game-type query: %w", err)
	}
	for _, r := range rows {
		result.PerGameType = append(result.PerGameType, storage.GameTypeStats{
			GameType: domain.GameType(r.k1).String(), PR: pr(r.sumErr, int(r.n)), PRInterval: r.interval,
			NumDecisions: int(r.n), BlunderCount: int(r.blunders),
		})
	}
	return nil
}

// computePerScore fills the away × away matrix.
//
// p.score_1 and p.score_2 are AWAY scores of the NORMALISED position, so
// score_1 is always the player on roll's — the one taking the decision. The
// cell is therefore (score_1, score_2) with no seat arithmetic, which is also
// why it does not depend on which player the filter selected.
//
// Money play (both scores at the -1 sentinel) has its own cell, Money, with
// both aways at 0: "my PR at money" is a real question. Post-Crawford is read
// as one away (domain.PointsAway), as the biases read it, so the cells of
// Crawford and post-Crawford at the same score are one.
func (s *StatsStore) computePerScore(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	rows, err := s.breakdownRowsBy(ctx, q, scoreKey, "p.score_1", "p.score_2")
	if err != nil {
		return fmt.Errorf("per-score query: %w", err)
	}
	for _, r := range rows {
		result.PerScore = append(result.PerScore, scoreCell(r.k1, r.k2, r.nulls == 0, storage.ScoreCellStats{
			PR: pr(r.sumErr, int(r.n)), PRInterval: r.interval,
			NumDecisions: int(r.n), BlunderCount: int(r.blunders),
		}))
	}
	sortScoreCells(result.PerScore)
	return nil
}

// scoreKey is the cell a stored pair of away scores is counted under: money
// (both negative) as (-1, -1), otherwise each score through domain.PointsAway.
// A key with a NULL column is left as it is, apart from the others.
func scoreKey(k breakdownKey) breakdownKey {
	if k.nulls != 0 {
		return k
	}
	return breakdownKey{k1: scoreAway(k.k1, k.k2), k2: scoreAway(k.k2, k.k1)}
}

// scoreAway is away decoded, or the money sentinel when both scores are.
func scoreAway(away, other int) int {
	if away < 0 && other < 0 {
		return -1
	}
	return domain.PointsAway(away)
}

// scoreCell completes c with the key it is counted under: decoded is false
// for a key kept apart (NULL), shown as stored.
func scoreCell(k1, k2 int, decoded bool, c storage.ScoreCellStats) storage.ScoreCellStats {
	if decoded && k1 < 0 && k2 < 0 {
		c.Money = true
		return c
	}
	c.MoverAway, c.OpponentAway = k1, k2
	return c
}

// sortScoreCells orders the away × away matrix, money first, then by mover's
// away, then opponent's.
func sortScoreCells(cells []storage.ScoreCellStats) {
	sort.SliceStable(cells, func(i, j int) bool {
		a, b := cells[i], cells[j]
		if a.Money != b.Money {
			return a.Money
		}
		if a.MoverAway != b.MoverAway {
			return a.MoverAway < b.MoverAway
		}
		return a.OpponentAway < b.OpponentAway
	})
}

// computePerTag splits the selection by the tags in the positions' comments.
//
// A tag lives in prose, not in a column (see domain.ExtractTags), so this
// cannot be a GROUP BY. Two queries rather than one join, deliberately:
// joining `comment` would multiply a decision by its position's comment count.
// A position carrying two tags contributes to both rows, so the rows do not
// sum to the total.
func (s *StatsStore) computePerTag(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	type decision struct {
		positionID int64
		errMP      int64
		match      int64
	}
	var decisions []decision
	positions := map[int64]bool{}

	// Only a commented position can carry a tag: reading the others would
	// hold every decision of the selection in memory for nothing.
	commented, cArgs := s.DB.TenantFilter("c", q.scope)
	rows, err := s.DB.Query(ctx,
		`SELECT p.id, COALESCE(`+statsErrExpr+`, 0), m.id `+q.join+q.whereSQL+
			` AND p.id IN (SELECT c.position_id FROM comment c WHERE `+commented+` AND c.text != '')`,
		append(append([]any{}, q.baseArgs...), cArgs...)...)
	if err != nil {
		return fmt.Errorf("per-tag decisions query: %w", err)
	}
	if err := func() error {
		defer rows.Close()
		for rows.Next() {
			var d decision
			if err := rows.Scan(&d.positionID, &d.errMP, &d.match); err != nil {
				return err
			}
			decisions = append(decisions, d)
			positions[d.positionID] = true
		}
		return rows.Err()
	}(); err != nil {
		return fmt.Errorf("per-tag decisions scan: %w", err)
	}
	if len(positions) == 0 {
		return nil
	}

	tagsByPosition, err := s.tagsOfPositions(ctx, q.scope, positions)
	if err != nil {
		return err
	}
	if len(tagsByPosition) == 0 {
		return nil
	}

	type tally struct {
		sumErr   int64
		count    int
		blunders int
	}
	byTag := map[string]*tally{}
	units := matchUnits[string]{}
	for _, d := range decisions {
		for _, tag := range tagsByPosition[d.positionID] {
			t := byTag[tag]
			if t == nil {
				t = &tally{}
				byTag[tag] = t
			}
			t.sumErr += d.errMP
			t.count++
			units.add(tag, d.match, d.errMP, 1)
			if d.errMP >= int64(q.settings.BlunderThresholdMP) {
				t.blunders++
			}
		}
	}
	for tag, t := range byTag {
		result.PerTag = append(result.PerTag, storage.TagStats{
			Tag: tag, PR: pr(t.sumErr, t.count), PRInterval: units.interval(tag),
			NumDecisions: t.count, BlunderCount: t.blunders,
		})
	}
	// Most decisions first, then alphabetically: a total order, so two runs
	// on the same data return the same rows in the same places.
	sort.Slice(result.PerTag, func(i, j int) bool {
		a, b := result.PerTag[i], result.PerTag[j]
		if a.NumDecisions != b.NumDecisions {
			return a.NumDecisions > b.NumDecisions
		}
		return a.Tag < b.Tag
	})
	return nil
}

// tagsOfPositions reads the comments of the given positions and returns their
// tags, keyed by position id.
func (s *StatsStore) tagsOfPositions(ctx context.Context, scope string, positions map[int64]bool) (map[int64][]string, error) {
	ids := make([]int64, 0, len(positions))
	for id := range positions {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	tenant, targs := s.DB.TenantFilter("", scope)
	out := map[int64][]string{}
	for start := 0; start < len(ids); start += byPositionsChunk {
		batch := ids[start:min(start+byPositionsChunk, len(ids))]
		args := make([]any, 0, len(batch)+len(targs))
		for _, id := range batch {
			args = append(args, id)
		}
		args = append(args, targs...)
		rows, err := s.DB.Query(ctx,
			`SELECT position_id, COALESCE(text,'') FROM comment
			 WHERE position_id IN (`+Placeholders(len(batch))+`) AND `+tenant+` AND text != ''`,
			args...)
		if err != nil {
			return nil, fmt.Errorf("per-tag comments query: %w", err)
		}
		if err := func() error {
			defer rows.Close()
			for rows.Next() {
				var id int64
				var text string
				if err := rows.Scan(&id, &text); err != nil {
					return err
				}
				for _, tag := range domain.ExtractTags(text) {
					out[id] = appendUnique(out[id], tag)
				}
			}
			return rows.Err()
		}(); err != nil {
			return nil, fmt.Errorf("per-tag comments scan: %w", err)
		}
	}
	return out, nil
}

// appendUnique adds tag unless the slice already has it — two comments on the
// same position may both carry it, and the position holds it once.
func appendUnique(tags []string, tag string) []string {
	for _, t := range tags {
		if t == tag {
			return tags
		}
	}
	return append(tags, tag)
}
