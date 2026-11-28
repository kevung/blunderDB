package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestBestMoveBatchReplaysAConflictBoundedly(t *testing.T) {
	saved := bestMoveRetryBackoff
	bestMoveRetryBackoff = 0
	t.Cleanup(func() { bestMoveRetryBackoff = saved })
	conflict := func(code string) error {
		return fmt.Errorf("postgres: write analysis best move: %w", &pgconn.PgError{Code: code})
	}
	cases := []struct {
		name      string
		errs      []error
		wantCalls int
		wantErr   bool
	}{
		{"deadlock then success", []error{conflict("40P01"), nil}, 2, false},
		{"serialization then success", []error{conflict("40001"), conflict("40001"), nil}, 3, false},
		{"other error is not replayed", []error{conflict("23505")}, 1, true},
		{"plain error is not replayed", []error{errors.New("boom")}, 1, true},
		{"gives up past the bound", nil, bestMoveRetries + 1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			calls := 0
			n, _, err := retryBestMoveBatch(context.Background(), func() (int, bool, error) {
				calls++
				if c.errs == nil {
					return 0, false, conflict("40P01")
				}
				if e := c.errs[calls-1]; e != nil {
					return 0, false, e
				}
				return 7, true, nil
			})
			if calls != c.wantCalls || (err != nil) != c.wantErr || (!c.wantErr && n != 7) {
				t.Fatalf("calls=%d n=%d err=%v, want calls=%d err=%v", calls, n, err, c.wantCalls, c.wantErr)
			}
		})
	}
}
