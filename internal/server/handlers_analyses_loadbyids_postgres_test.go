//go:build postgres

// Needs Docker (testcontainers), like handlers_tenant_test.go:
//
//	go test -tags postgres ./internal/server/... -run TestAnalysesLoadByIDsTenant -v
package server

import (
	"testing"
)

// TestAnalysesLoadByIDsTenantIsolation asks one tenant for a list mixing its
// own analysed positions with another tenant's. Only its own come back: an id
// is a global row id on PostgreSQL, so knowing another tenant's id must never
// be enough to read its analysis.
func TestAnalysesLoadByIDsTenantIsolation(t *testing.T) {
	ts := newPostgresTestServer(t)

	mineA := saveAnalysedPosition(t, ts, "1", 1, true)
	mineB := saveAnalysedPosition(t, ts, "1", 2, true)
	theirsA := saveAnalysedPosition(t, ts, "2", 1, true)
	theirsB := saveAnalysedPosition(t, ts, "2", 2, true)

	got := loadAnalysesByIDs(t, ts, "1", []int64{theirsA, mineA, theirsB, mineB})
	want := []int64{mineA, mineB}
	if len(got) != len(want) {
		t.Fatalf("tenant 1 read %d analyses %+v, want only its own positions %v", len(got), got, want)
	}
	for i, a := range got {
		if int64(a.PositionID) != want[i] {
			t.Errorf("tenant 1 analyses.loadByIds[%d] = position %d, want %d", i, a.PositionID, want[i])
		}
	}

	other := loadAnalysesByIDs(t, ts, "2", []int64{mineA, mineB})
	if len(other) != 0 {
		t.Fatalf("tenant 2 read tenant 1's analyses: %+v", other)
	}
}
