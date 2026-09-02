package sqlite

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// statsStore is the shared statistics store (sqlshared.StatsStore) plus the
// one method whose SQL has no backend-neutral form: DateRange's predicate on
// match_date, which is TEXT here and carries a year-1 sentinel for "unset".
type statsStore struct{ *sqlshared.StatsStore }

var _ storage.StatsStore = (*statsStore)(nil)

// DateRange returns the minimum and maximum match dates present in the
// database. Both fields are empty when no matches with a date exist.
func (s *statsStore) DateRange(ctx context.Context, scope string) (storage.StatsDateRange, error) {
	var min, max string
	if err := s.DB.QueryRow(ctx,
		`SELECT COALESCE(MIN(SUBSTR(match_date,1,10)),''), COALESCE(MAX(SUBSTR(match_date,1,10)),'')
		 FROM match
		 WHERE match_date IS NOT NULL AND match_date != '' AND match_date != '0001-01-01T00:00:00Z'`,
	).Scan(&min, &max); err != nil {
		return storage.StatsDateRange{}, fmt.Errorf("sqlite: stats date range: %w", err)
	}
	return storage.StatsDateRange{DateFrom: min, DateTo: max}, nil
}
