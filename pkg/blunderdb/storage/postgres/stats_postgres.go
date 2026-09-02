package postgres

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// statsStore is the shared statistics store (sqlshared.StatsStore) plus the
// one method whose SQL has no backend-neutral form: DateRange's predicate on
// match_date, which is TIMESTAMPTZ here (nullableTime stores the zero time as
// NULL, so IS NOT NULL is the only guard needed).
type statsStore struct{ *sqlshared.StatsStore }

var _ storage.StatsStore = (*statsStore)(nil)

// DateRange returns the min/max match dates (YYYY-MM-DD, UTC) for the tenant.
// Both fields are empty when no matches with a date exist.
func (s *statsStore) DateRange(ctx context.Context, scope string) (storage.StatsDateRange, error) {
	var min, max string
	if err := s.DB.QueryRow(ctx,
		`SELECT COALESCE(TO_CHAR(MIN(match_date) AT TIME ZONE 'UTC','YYYY-MM-DD'),''),
		        COALESCE(TO_CHAR(MAX(match_date) AT TIME ZONE 'UTC','YYYY-MM-DD'),'')
		 FROM match WHERE tenant_id = ? AND match_date IS NOT NULL`,
		tenantID(scope),
	).Scan(&min, &max); err != nil {
		return storage.StatsDateRange{}, fmt.Errorf("postgres: stats date range: %w", err)
	}
	return storage.StatsDateRange{DateFrom: min, DateTo: max}, nil
}
