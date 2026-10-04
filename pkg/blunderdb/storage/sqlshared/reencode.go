package sqlshared

import (
	"context"
	"log/slog"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

// ReencodeAnalyses implements storage.AnalysisStore.ReencodeAnalyses for both
// backends. The first byte of a binary blob is compared in SQL (substr of a
// BLOB or a bytea is a byte string on both), so a restarted pass reads again
// only the rows it has not rewritten.
func ReencodeAnalyses(ctx context.Context, db Execer, scope string, after int64, limit int) (int64, int, error) {
	if limit <= 0 {
		limit = 1
	}
	tenant, targs := db.TenantFilter("a", scope)
	args := append(targs, engine.BinaryBlobPrefix(), after, limit)
	rows, err := db.Query(ctx, `SELECT a.position_id, a.data FROM analysis a
		WHERE `+tenant+` AND a.data IS NOT NULL AND substr(a.data, 1, 1) <> ? AND a.position_id > ?
		ORDER BY a.position_id LIMIT ?`, args...)
	if err != nil {
		return 0, 0, errf(db, "reencode analyses", err)
	}
	type row struct {
		id   int64
		data []byte
	}
	var page []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.data); err != nil {
			rows.Close()
			return 0, 0, errf(db, "reencode analyses", err)
		}
		page = append(page, r)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, 0, errf(db, "reencode analyses", err)
	}
	rows.Close()
	if len(page) == 0 {
		return 0, 0, nil
	}

	blobs := make([][]byte, len(page))
	for i, r := range page {
		blobs[i] = r.data
	}
	fresh, encErrs := engine.RecompressAnalysesConcurrently(blobs)

	rewritten := 0
	err = db.Transact(ctx, func(tx Execer) error {
		for i, r := range page {
			if !engine.NeedsRecompression(r.data) {
				continue
			}
			if encErrs[i] != nil {
				// Left as it is: its bytes are the only trace of that analysis.
				slog.Warn("reencode analyses: blob does not decode, left as is",
					"position_id", r.id, "err", encErrs[i])
				continue
			}
			utenant, uargs := tx.TenantFilter("", scope)
			args := append([]any{fresh[i]}, uargs...)
			if _, err := tx.Exec(ctx, `UPDATE analysis SET data = ? WHERE `+utenant+` AND position_id = ?`,
				append(args, r.id)...); err != nil {
				return err
			}
			rewritten++
		}
		return nil
	})
	if err != nil {
		return 0, 0, errf(db, "reencode analyses", err)
	}
	return page[len(page)-1].id, rewritten, nil
}
