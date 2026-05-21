// Package storagetest provides a backend-agnostic contract test suite for
// implementations of storage.Storage. Each backend (SQLite now, PostgreSQL
// later) runs the same suite against a fresh instance:
//
//	func TestContract_SQLite(t *testing.T) {
//	    storagetest.RunContractTests(t, func() storage.Storage {
//	        s, _ := sqlite.Open(context.Background(), ":memory:", nil)
//	        return s
//	    })
//	}
//
// It lives in a regular (non-_test.go) file so it can be imported by the
// backend packages' test binaries — an exported helper in a _test.go file
// would not be visible across packages.
//
// The subtests below are skeletons: they are filled in alongside each
// family's SQLite implementation (P2 PRs 2-6).
package storagetest

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// RunContractTests exercises a storage.Storage implementation. factory must
// return a fresh, empty, migrated Storage on every call.
func RunContractTests(t *testing.T, factory func() storage.Storage) {
	t.Helper()

	cases := []struct {
		name string
		fn   func(t *testing.T, s storage.Storage)
	}{
		{"Position/Save+Load", nil},
		{"Position/DedupByZobrist", nil},
		{"Position/UpdatePreservesId", nil},
		{"Analysis/SaveAndCompress", nil},
		{"Match/CreateGameMoveCascade", nil},
		{"Match/DeleteCascade", nil},
		{"Tournament/AddRemoveMatch", nil},
		{"Collection/MoveBetweenCollections", nil},
		{"Anki/ReviewUpdatesScheduling", nil},
		{"Filter/SaveAndList", nil},
		{"Session/SaveLoadEmpty", nil},
		{"Search/FilterByDecisionType", nil},
		{"Stats/AggregateCounts", nil},
		{"Tx/RollbackUndoes", nil},
		{"Tx/CommitPersists", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.fn == nil {
				t.Skip("contract case pending storage implementation (P2 PRs 2-6)")
			}
			s := factory()
			defer s.Close()
			tc.fn(t, s)
		})
	}
}
