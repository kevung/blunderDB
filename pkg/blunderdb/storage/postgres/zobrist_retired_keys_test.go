// TestRetiredZobristKeysMatchTheEngine never touches a database: migration 014
// undoes a XOR that engine.ZobristHash used to apply, and SQL cannot call Go,
// so the two retired Zobrist keys are written into the .sql file as literals.
// This test reads them back and compares them with the keys the engine draws.
// Without it, a change to the Zobrist key stream would leave the PostgreSQL
// backend converting hashes with the wrong constant — silently, and only for
// the handful of positions that carry a rule flag.
package postgres

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
)

const retiredKeysMigration = "migrations/014_zobrist_without_rule_flags.sql"

func TestRetiredZobristKeysMatchTheEngine(t *testing.T) {
	src, err := migrationsFS.ReadFile(retiredKeysMigration)
	if err != nil {
		t.Fatalf("read %s: %v", retiredKeysMigration, err)
	}

	for _, tc := range []struct {
		column string
		want   uint64
	}{
		{"has_jacoby", engine.RetiredFlagDelta(1, 0)},
		{"has_beaver", engine.RetiredFlagDelta(0, 1)},
	} {
		re := regexp.MustCompile(`CASE WHEN ` + tc.column + ` THEN (-?\d+) ELSE 0 END`)
		m := re.FindSubmatch(src)
		if m == nil {
			t.Fatalf("%s: no `CASE WHEN %s THEN <literal>` in %s — did the migration change shape?",
				tc.column, tc.column, retiredKeysMigration)
		}
		got, err := strconv.ParseInt(string(m[1]), 10, 64)
		if err != nil {
			t.Fatalf("%s: literal %q is not a 64-bit integer: %v", tc.column, m[1], err)
		}
		if uint64(got) != tc.want {
			t.Errorf("%s: migration uses %d, engine.RetiredFlagDelta gives %d (%#016x) — update the literal and its comment in %s",
				tc.column, got, int64(tc.want), tc.want, retiredKeysMigration)
		}
	}
}
