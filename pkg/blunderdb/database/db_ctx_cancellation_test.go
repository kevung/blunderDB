package database

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// The Ctx variants of search, stats and export must honour a context
// cancelled before the call. Cancelling up front keeps the tests
// deterministic: database/sql dispatches nothing once ctx.Err() != nil.

func TestLoadPositionsByFiltersCoreCtxRespectsCancellation(t *testing.T) {
	t.Parallel()
	db := newTestDBWithXG(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := db.LoadPositionsByFiltersCoreCtx(ctx, SearchFilters{Filter: emptyFilter()}, storage.ListOpts{})
	if err == nil {
		t.Fatal("LoadPositionsByFiltersCoreCtx: want an error on an already-cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("LoadPositionsByFiltersCoreCtx: err = %v, want it to wrap context.Canceled", err)
	}
}

func TestComputeStatsCtxRespectsCancellation(t *testing.T) {
	t.Parallel()
	db := newTestDBWithXG(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := db.ComputeStatsCtx(ctx, StatsFilter{DecisionType: -1})
	if err == nil {
		t.Fatal("ComputeStatsCtx: want an error on an already-cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("ComputeStatsCtx: err = %v, want it to wrap context.Canceled", err)
	}
}

func TestExportDatabaseCtxRespectsCancellation(t *testing.T) {
	isolateIdentity(t)
	db := newTestDBWithXG(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	path := filepath.Join(t.TempDir(), "cancelled.db")
	err := db.ExportDatabaseCtx(ctx, ExportOptions{ExportPath: path, AllPositions: true, Metadata: map[string]string{}})
	if err == nil {
		t.Fatal("ExportDatabaseCtx: want an error on an already-cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("ExportDatabaseCtx: err = %v, want it to wrap context.Canceled", err)
	}
}

// The context.Background() convenience methods must still work exactly as
// before: they are what the GUI calls, unchanged.
func TestLoadPositionsByFiltersCoreStillWorksWithoutContext(t *testing.T) {
	t.Parallel()
	db := newTestDBWithXG(t)

	positions, _, err := db.LoadPositionsByFiltersCore(SearchFilters{Filter: emptyFilter()}, storage.ListOpts{})
	if err != nil {
		t.Fatalf("LoadPositionsByFiltersCore: %v", err)
	}
	if len(positions) == 0 {
		t.Fatal("LoadPositionsByFiltersCore: want at least one position from the XG fixture")
	}
}
