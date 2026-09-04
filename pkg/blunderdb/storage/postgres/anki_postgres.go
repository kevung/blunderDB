package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// ankiStore is sqlshared.AnkiStore (B.14, #182) plus the one method that
// stays backend-specific: Forecast's day-offset bucketing needs PostgreSQL's
// native DATE arithmetic, which has no SQLite equivalent (see
// sqlshared.AnkiStore's doc, and stats_postgres.go's DateRange for the same
// shape of override).
type ankiStore struct{ *sqlshared.AnkiStore }

var _ storage.AnkiStore = (*ankiStore)(nil)

// ankiForecastDays clamps a requested forecast horizon into a sane range:
// non-positive falls back to 30 days, and the horizon is capped at one year.
func ankiForecastDays(days int) int {
	switch {
	case days <= 0:
		return 30
	case days > 365:
		return 365
	}
	return days
}

// Forecast projects how many cards come due over the next `days` calendar
// days, offset 0 absorbing every overdue card. deckID 0 spans the whole
// tenant.
func (s *ankiStore) Forecast(ctx context.Context, scope string, deckID int64, days int) ([]domain.AnkiForecastDay, error) {
	tenant := tenantID(scope)
	days = ankiForecastDays(days)

	query := `SELECT GREATEST(0, (CAST(due AS DATE) - CAST(now() AS DATE))) AS day_offset, COUNT(*)
		FROM anki_card
		WHERE tenant_id = ? AND suspended = FALSE AND CAST(due AS DATE) < CAST(now() AS DATE) + ?::int`
	args := []any{tenant, days}
	if deckID != 0 {
		query += ` AND deck_id = ?`
		args = append(args, deckID)
	}
	query += ` GROUP BY day_offset`

	rows, err := s.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: anki forecast: %w", err)
	}
	defer rows.Close()
	counts := make(map[int]int, days)
	for rows.Next() {
		var off, n int
		if err := rows.Scan(&off, &n); err != nil {
			return nil, fmt.Errorf("postgres: anki forecast: %w", err)
		}
		counts[off] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: anki forecast: %w", err)
	}
	return storage.BuildForecast(time.Now().UTC(), days, counts), nil
}
