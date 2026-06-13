package ingest

import (
	"context"
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Shared test helpers for the import tests. These used to live in xg_test.go
// alongside the legacy↔ingest parity tests; once those parity tests were
// retired (the ingest path is now the single implementation, exercised end to
// end by the database package's absolute-value import tests), the helpers moved
// here so the surviving native-.db importer tests in db_test.go keep working.
// Nothing here imports the database package, which is what lets the database
// package import ingest for the GUI/CLI import delegation.

// positionKey is a stable, id-independent identity for a position, used to
// match positions across two stores.
func positionKey(p *domain.Position) string {
	c := *p
	c.ID = 0
	b, _ := json.Marshal(c)
	return string(b)
}

// normaliseAnalysisForCompare zeroes the fields that legitimately differ
// between two import runs (timestamps and the row-local position id) so the
// rest can be compared field-by-field.
func normaliseAnalysisForCompare(a *domain.PositionAnalysis) *domain.PositionAnalysis {
	if a == nil {
		return nil
	}
	c := *a
	c.PositionID = 0
	c.CreationDate = time.Time{}
	c.LastModifiedDate = time.Time{}
	return &c
}

// posData is the raw stored data for one position: its analysis (nil if none)
// and the sorted set of its comment texts.
type posData struct {
	analysis *domain.PositionAnalysis
	comments []string
}

// rawAnalyses reads every position with its raw stored analysis and comments.
func rawAnalyses(t *testing.T, ctx context.Context, s storage.Storage) map[string]*posData {
	t.Helper()
	var positions []*domain.Position
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatalf("Positions().List: %v", err)
		}
		pc := *p
		positions = append(positions, &pc)
	}
	out := make(map[string]*posData, len(positions))
	for _, p := range positions {
		d := &posData{}
		if a, err := s.Analyses().Load(ctx, "", p.ID); err == nil {
			d.analysis = a
		}
		for c, err := range s.Comments().ByPosition(ctx, "", p.ID) {
			if err != nil {
				t.Fatalf("Comments().ByPosition: %v", err)
			}
			d.comments = append(d.comments, c.Text)
		}
		sort.Strings(d.comments)
		out[positionKey(p)] = d
	}
	return out
}
