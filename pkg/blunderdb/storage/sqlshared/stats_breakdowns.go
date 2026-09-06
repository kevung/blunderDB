package sqlshared

import (
	"context"
	"fmt"
	"sort"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The three breakdowns of issue #266 (fiche I.10): the same PR the statistics
// already show, sliced by game phase, by tag, and by away × away score.
//
// Every one of them reuses countedExpr and statsErrExpr — what counts as a
// decision and what an error costs are stated once, in stats.go, and a
// breakdown that restated either would be a second PR wearing the same name.
// What each pass adds is a GROUP BY, or, for tags, a second query.

// computePerPhase splits the selection by the position's derived phase
// (ADR-0035). A database whose phases have never been computed reports
// everything as "unknown", which is honest: the column says so, and
// `blunderdb repair` is what fills it.
func (s *StatsStore) computePerPhase(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	rows, err := s.DB.Query(ctx,
		`SELECT COALESCE(p.game_phase, 0), `+d.Bigint(`SUM(`+statsErrExpr+`)`)+`, COUNT(*), `+
			d.Bigint(`SUM(CASE WHEN `+statsErrExpr+` >= ? THEN 1 ELSE 0 END)`)+` `+
			statsBaseJoin+q.whereSQL+` GROUP BY p.game_phase`,
		append([]any{blunderThresholdMP}, q.baseArgs...)...)
	if err != nil {
		return fmt.Errorf("per-phase query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var phase int
		var sumErr int64
		var ps storage.PhaseStats
		if err := rows.Scan(&phase, &sumErr, &ps.NumDecisions, &ps.BlunderCount); err != nil {
			return fmt.Errorf("per-phase scan: %w", err)
		}
		ps.Phase = domain.GamePhase(phase).String()
		ps.PR = pr(sumErr, ps.NumDecisions)
		result.PerPhase = append(result.PerPhase, ps)
	}
	return rows.Err()
}

// computePerScore fills the away × away matrix.
//
// p.score_1 and p.score_2 are AWAY scores of the NORMALISED position, so
// score_1 is always the player on roll's — the one taking the decision. The
// cell is therefore (score_1, score_2) with no seat arithmetic, which is also
// why it does not depend on which player the filter selected.
//
// Money play (match_length 0, both scores 0) lands in the (0,0) cell and is
// left there rather than dropped: "my PR at money" is a real question, and it
// is the one cell of the matrix whose meaning is not an away score.
//
// Crawford is NOT a dimension here, because it is not stored on a position.
// Its practical effect is small — a Crawford game has no cube decision at
// all — but the omission is real and the documentation says so rather than
// letting the matrix imply a distinction it does not make.
func (s *StatsStore) computePerScore(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	d := s.DB
	rows, err := s.DB.Query(ctx,
		`SELECT COALESCE(p.score_1, 0), COALESCE(p.score_2, 0), `+
			d.Bigint(`SUM(`+statsErrExpr+`)`)+`, COUNT(*), `+
			d.Bigint(`SUM(CASE WHEN `+statsErrExpr+` >= ? THEN 1 ELSE 0 END)`)+` `+
			statsBaseJoin+q.whereSQL+` GROUP BY p.score_1, p.score_2`,
		append([]any{blunderThresholdMP}, q.baseArgs...)...)
	if err != nil {
		return fmt.Errorf("per-score query: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sumErr int64
		var cell storage.ScoreCellStats
		if err := rows.Scan(&cell.MoverAway, &cell.OpponentAway, &sumErr,
			&cell.NumDecisions, &cell.BlunderCount); err != nil {
			return fmt.Errorf("per-score scan: %w", err)
		}
		cell.PR = pr(sumErr, cell.NumDecisions)
		result.PerScore = append(result.PerScore, cell)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	sort.Slice(result.PerScore, func(i, j int) bool {
		a, b := result.PerScore[i], result.PerScore[j]
		if a.MoverAway != b.MoverAway {
			return a.MoverAway < b.MoverAway
		}
		return a.OpponentAway < b.OpponentAway
	})
	return nil
}

// computePerTag splits the selection by the tags in the positions' comments.
//
// A tag lives in prose, not in a column (see domain.ExtractTags), so this
// cannot be a GROUP BY. It is two queries instead of one join, deliberately:
// joining `comment` into statsBaseJoin would multiply a decision by the number
// of comments on its position and inflate every count. The first query is the
// decisions, the second the tags per position, and the two meet in Go.
//
// A position carrying two tags contributes to both rows. The rows therefore do
// not sum to the total, which is what a label means as opposed to a partition.
func (s *StatsStore) computePerTag(ctx context.Context, q statsQuery, result *storage.StatsResult) error {
	type decision struct {
		positionID int64
		errMP      int64
	}
	var decisions []decision
	positions := map[int64]bool{}

	rows, err := s.DB.Query(ctx,
		`SELECT p.id, COALESCE(`+statsErrExpr+`, 0) `+statsBaseJoin+q.whereSQL, q.baseArgs...)
	if err != nil {
		return fmt.Errorf("per-tag decisions query: %w", err)
	}
	if err := func() error {
		defer rows.Close()
		for rows.Next() {
			var d decision
			if err := rows.Scan(&d.positionID, &d.errMP); err != nil {
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
	for _, d := range decisions {
		for _, tag := range tagsByPosition[d.positionID] {
			t := byTag[tag]
			if t == nil {
				t = &tally{}
				byTag[tag] = t
			}
			t.sumErr += d.errMP
			t.count++
			if d.errMP >= blunderThresholdMP {
				t.blunders++
			}
		}
	}
	for tag, t := range byTag {
		result.PerTag = append(result.PerTag, storage.TagStats{
			Tag: tag, PR: pr(t.sumErr, t.count),
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
