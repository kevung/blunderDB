// TestPurgeOrderMatchesRLSTables never touches a database — it is a pure
// in-memory comparison of two package-level slices — unlike TestPurgeTenant
// and friends (purge_postgres_test.go, tagged `//go:build postgres`, which
// provision a real PostgreSQL via testcontainers-go). It used to live in that
// tagged file, so `go test ./...` (no `-tags postgres`, the only CI job in
// .github/workflows/build.yml) never ran it. It lives in its own untagged
// file, `package postgres` (white-box) like its old home, because it needs
// direct access to the two unexported package-level variables it guards
// (purgeOrder in purge_postgres.go, rlsTables in rls_postgres.go).
package postgres

import (
	"slices"
	"testing"
)

// TestPurgeOrderMatchesRLSTables guards against purgeOrder (purge_postgres.go)
// and rlsTables (rls_postgres.go) silently drifting apart: sorting both and
// comparing catches a table added to one list but not the other, in either
// direction. This test needs no database.
func TestPurgeOrderMatchesRLSTables(t *testing.T) {
	got := slices.Clone(purgeOrder)
	want := slices.Clone(rlsTables)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("purgeOrder and rlsTables have drifted apart:\n purgeOrder (sorted) = %v\n rlsTables  (sorted) = %v", got, want)
	}
}
