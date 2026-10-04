package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// provenanceBatch is how many analysis rows one backfill round reads and
// writes.
const provenanceBatch = 20000

// backfillAnalysisProvenance derives analysis_engine, analysis_depth and
// creation_date (engine.AnalysisProvenance) for every analysis row whose
// analysis_engine is NULL — all of them after 032, and later the few a
// blob-only writer leaves. It runs in Go because the blob is compressed JSON
// SQL cannot read. Resumable by construction: a written row is no longer
// NULL. An undecodable blob is written as "no entry", so it is not retried on
// every Migrate.
func backfillAnalysisProvenance(ctx context.Context, db execer) error {
	var last int64
	done := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		rows, err := db.Query(ctx,
			`SELECT id, data FROM analysis WHERE analysis_engine IS NULL AND id > $1 ORDER BY id LIMIT $2`,
			last, provenanceBatch)
		if err != nil {
			return fmt.Errorf("postgres: read analyses to derive provenance: %w", err)
		}
		raw := make(map[int64][]byte, provenanceBatch)
		var ids []int64
		for rows.Next() {
			var id int64
			var data []byte
			if err := rows.Scan(&id, &data); err != nil {
				rows.Close()
				return fmt.Errorf("postgres: scan analysis to derive provenance: %w", err)
			}
			raw[id] = data
			ids = append(ids, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("postgres: read analyses to derive provenance: %w", err)
		}
		if len(ids) == 0 {
			break
		}
		decoded, failed := engine.DecodeAnalysesConcurrently(raw)
		engines := make([]string, len(ids))
		depths := make([]int64, len(ids))
		created := make([]*int64, len(ids))
		for i, id := range ids {
			engines[i], depths[i] = "", -1
			if a := decoded[id]; a != nil {
				var c int64
				engines[i], depths[i], c = engine.AnalysisProvenance(a)
				if c != 0 {
					created[i] = &c
				}
			} else if err := failed[id]; err != nil {
				slog.Warn("analysis provenance: undecodable blob", "analysis_id", id, "error", err)
			}
		}
		if _, err := db.Exec(ctx,
			`UPDATE analysis a SET analysis_engine = v.e, analysis_depth = v.d,
			        creation_date = to_timestamp(v.c)
			   FROM unnest($1::bigint[], $2::text[], $3::int[], $4::bigint[]) AS v(id, e, d, c)
			  WHERE a.id = v.id`,
			ids, engines, depths, created); err != nil {
			return fmt.Errorf("postgres: write analysis provenance: %w", err)
		}
		// Provenance is a column match_stats summarises: the matches reaching
		// these analyses are recomputed on the next read.
		if _, err := db.Exec(ctx,
			`DELETE FROM match_stats ms USING analysis a, move mv, game g
			  WHERE a.id = ANY($1::bigint[]) AND mv.position_id = a.position_id AND mv.tenant_id = a.tenant_id
			    AND g.id = mv.game_id AND ms.match_id = g.match_id AND ms.tenant_id = a.tenant_id`,
			ids); err != nil {
			return fmt.Errorf("postgres: invalidate match stats after provenance: %w", err)
		}
		last = ids[len(ids)-1]
		done += len(ids)
	}
	if done > 0 {
		slog.Info("derived the provenance of the stored analyses", "analyses", done)
	}
	return nil
}
