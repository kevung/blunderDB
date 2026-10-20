package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// A sync fence names the transactions running at a moment, so an
// incremental export can tell later that none of them can still commit a
// row under the mark it read then (ingest.SyncFence, ADR-0096). Two records
// cover them:
//
//   - xmax, the first transaction id not yet assigned: every transaction
//     that has written anything holds an id below it, whatever its role,
//     and they have all ended once the oldest running id (the snapshot's
//     xmin) reaches it;
//   - the (pid, xact_start) of every transaction open in this database,
//     which also covers one that drew a sequence value and has not yet
//     written its row, so holds no id.
//
// The check is database-wide and cluster-wide for ids: a long transaction
// elsewhere only delays a mark, never loses a row.
type syncFence struct {
	Xmax string        `json:"xmax"`
	Open []openTxnMark `json:"open,omitempty"`
}

type openTxnMark struct {
	PID   int32     `json:"pid"`
	Start time.Time `json:"start"`
}

// SyncFence names every transaction running now.
func (s *Storage) SyncFence(ctx context.Context) (string, error) {
	var f syncFence
	if err := s.pool.QueryRow(ctx, `SELECT pg_snapshot_xmax(pg_current_snapshot())::text`).Scan(&f.Xmax); err != nil {
		return "", fmt.Errorf("postgres: read snapshot: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT pid, xact_start FROM pg_stat_activity
		WHERE datname = current_database() AND pid <> pg_backend_pid() AND xact_start IS NOT NULL`)
	if err != nil {
		return "", fmt.Errorf("postgres: read open transactions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var m openTxnMark
		if err := rows.Scan(&m.PID, &m.Start); err != nil {
			return "", err
		}
		f.Open = append(f.Open, m)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	raw, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// SyncFenceCleared reports whether every transaction the token names has
// ended. A token that cannot be read is not cleared: the mark behind it
// stays pending, an overlap rather than a hole.
func (s *Storage) SyncFenceCleared(ctx context.Context, token string) (bool, error) {
	var f syncFence
	if err := json.Unmarshal([]byte(token), &f); err != nil || f.Xmax == "" {
		return false, nil
	}
	pids := make([]int32, len(f.Open))
	starts := make([]time.Time, len(f.Open))
	for i, m := range f.Open {
		pids[i], starts[i] = m.PID, m.Start
	}
	var cleared bool
	err := s.pool.QueryRow(ctx, `
		SELECT pg_snapshot_xmin(pg_current_snapshot()) >= $1::xid8
		   AND NOT EXISTS (
		       SELECT 1 FROM pg_stat_activity a
		       JOIN unnest($2::int4[], $3::timestamptz[]) AS f(pid, start)
		         ON a.pid = f.pid AND a.xact_start = f.start)`,
		f.Xmax, pids, starts).Scan(&cleared)
	if err != nil {
		return false, fmt.Errorf("postgres: check sync fence: %w", err)
	}
	return cleared, nil
}
