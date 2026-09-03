package sqlite

import (
	"context"
	"fmt"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// ankiStore is sqlshared.AnkiStore (B.14, #182) plus the one method that
// stays backend-specific: Forecast's day-offset bucketing needs SQLite's
// julianday()/date() functions, which have no PostgreSQL equivalent (see
// sqlshared.AnkiStore's doc, and stats_sqlite.go's DateRange for the same
// shape of override).
type ankiStore struct{ *sqlshared.AnkiStore }

var _ storage.AnkiStore = (*ankiStore)(nil)

// Forecast projects how many cards come due over the next `days` calendar
// days, offset 0 absorbing every overdue card. (The SQLite backend is the
// single-tenant desktop store, so scope is unused.)
func (s *ankiStore) Forecast(ctx context.Context, scope string, deckID int64, days int) ([]domain.AnkiForecastDay, error) {
	switch {
	case days <= 0:
		days = 30
	case days > 365:
		days = 365
	}

	query := `SELECT MAX(0, CAST(julianday(date(due)) - julianday(date('now')) AS INTEGER)) AS day_offset, COUNT(*)
		FROM anki_card
		WHERE suspended = 0 AND date(due) < date('now', '+' || ? || ' days')`
	args := []any{days}
	if deckID != 0 {
		query += ` AND deck_id = ?`
		args = append(args, deckID)
	}
	query += ` GROUP BY day_offset`

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("sqlite: anki forecast: %w", err)
	}
	defer rows.Close()
	counts := make(map[int]int, days)
	for rows.Next() {
		var off, n int
		if err := rows.Scan(&off, &n); err != nil {
			return nil, fmt.Errorf("sqlite: anki forecast: %w", err)
		}
		counts[off] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite: anki forecast: %w", err)
	}
	return storage.BuildForecast(time.Now().UTC(), days, counts), nil
}
