// TestWinnerMigrationSQLIsShared never touches a database: `package postgres`
// for the unexported migrationsFS.
package postgres

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"
)

// TestWinnerMigrationSQLIsShared: 028 runs the very statement SQLite's
// 2.25.0 → 2.26.0 step runs, so both backends convert game.winner by one rule.
func TestWinnerMigrationSQLIsShared(t *testing.T) {
	b, err := migrationsFS.ReadFile("migrations/028_game_winner_encoding.sql")
	if err != nil {
		t.Fatalf("read 028: %v", err)
	}
	s := string(b)
	const begin, end = "-- BEGIN NormalizeGameWinnerSQL\n", ";\n-- END NormalizeGameWinnerSQL"
	i, j := strings.Index(s, begin), strings.Index(s, end)
	if i < 0 || j < i {
		t.Fatal("028 lacks the BEGIN/END NormalizeGameWinnerSQL markers")
	}
	if got := s[i+len(begin) : j]; got != sqlshared.NormalizeGameWinnerSQL {
		t.Errorf("028's statement differs from sqlshared.NormalizeGameWinnerSQL:\n%s", got)
	}
}
