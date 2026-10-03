package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// analysedByHash maps the Zobrist hash of every analysed position of st to
// its id and analysis, so two libraries can be compared position by position.
func analysedByHash(t *testing.T, ctx context.Context, st storage.Storage) (map[uint64]int64, map[int64]*domain.PositionAnalysis) {
	t.Helper()
	ids, err := st.Positions().ListIDs(ctx, "", storage.ListOpts{})
	if err != nil {
		t.Fatal(err)
	}
	positions, err := st.Positions().LoadByIDs(ctx, "", ids)
	if err != nil {
		t.Fatal(err)
	}
	analyses, err := st.Analyses().LoadMany(ctx, "", ids)
	if err != nil {
		t.Fatal(err)
	}
	byHash := map[uint64]int64{}
	for i := range positions {
		if analyses[positions[i].ID] != nil {
			byHash[engine.ZobristHash(&positions[i])] = positions[i].ID
		}
	}
	return byHash, analyses
}

// A re-import keeps the rollouts on both sides: those the target wrote, and
// those the source brought to a position the target had already analysed.
func TestDBImportKeepsRollouts(t *testing.T) {
	ctx := context.Background()
	srcPath := makeSourceDB(t)
	target, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	imp := DBImporter{S: target}
	if _, err := imp.Import(ctx, "", Source{Format: FormatNativeDB, Path: srcPath}, nil); err != nil {
		t.Fatal(err)
	}

	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	srcIDs, srcAnalyses := analysedByHash(t, ctx, src)
	tgtIDs, _ := analysedByHash(t, ctx, target)
	var hashes []uint64
	for h := range srcIDs {
		if _, ok := tgtIDs[h]; ok {
			hashes = append(hashes, h)
		}
		if len(hashes) == 2 {
			break
		}
	}
	if len(hashes) < 2 {
		src.Close()
		t.Fatal("fixture has fewer than two analysed positions")
	}
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	brought := srcAnalyses[srcIDs[hashes[0]]]
	brought.AttachRollout(domain.RolloutAnalysis{Signature: "source", Games: 216, Date: at})
	if err := src.Analyses().Save(ctx, "", srcIDs[hashes[0]], brought); err != nil {
		t.Fatal(err)
	}
	src.Close()
	rolloutOnly := &domain.PositionAnalysis{}
	rolloutOnly.AttachRollout(domain.RolloutAnalysis{Signature: "target", Games: 216, Date: at})
	if err := target.Analyses().Save(ctx, "", tgtIDs[hashes[1]], rolloutOnly); err != nil {
		t.Fatal(err)
	}

	if _, err := imp.Import(ctx, "", Source{Format: FormatNativeDB, Path: srcPath}, nil); err != nil {
		t.Fatal(err)
	}
	got, err := target.Analyses().Load(ctx, "", tgtIDs[hashes[0]])
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasRollout("source") || !got.HasPrimary() {
		t.Errorf("analysed position: rollouts %+v, primary %v — want the source rollout beside its analysis", got.Rollouts, got.HasPrimary())
	}
	got, err = target.Analyses().Load(ctx, "", tgtIDs[hashes[1]])
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasRollout("target") || !got.HasPrimary() {
		t.Errorf("rollout-only position: rollouts %+v, primary %v — want the imported analysis beside its rollout", got.Rollouts, got.HasPrimary())
	}
}
