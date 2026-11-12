package ingest

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// analysesByHash reads every analysis of s, keyed by its position's Zobrist hash.
func analysesByHash(t *testing.T, s storage.Storage) map[uint64]domain.PositionAnalysis {
	t.Helper()
	ctx := context.Background()
	var ids []int64
	hashOf := map[int64]uint64{}
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
		hashOf[p.ID] = engine.ZobristHash(p)
	}
	loaded, err := s.Analyses().LoadMany(ctx, "", ids)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uint64]domain.PositionAnalysis{}
	for id, a := range loaded {
		out[hashOf[id]] = *a
	}
	return out
}

// withoutProvenance is a with what names its match, and its row's own ids and
// dates, cleared.
func withoutProvenance(a domain.PositionAnalysis) domain.PositionAnalysis {
	a.PositionID = 0
	a.Player1, a.Player2 = "", ""
	a.PlayedMove, a.PlayedCubeAction = "", ""
	a.PlayedMoves, a.PlayedCubeActions = nil, nil
	a.CreationDate, a.LastModifiedDate = time.Time{}, time.Time{}
	return a
}

// TestTransfer_OmitProvenance: a collection copied with OmitProvenance arrives
// with its analyses whole but nothing naming the match they were played in —
// no player, no played move or cube action, no match. Without it the copy
// carries them as before, and the source is never touched.
func TestTransfer_OmitProvenance(t *testing.T) {
	ctx := context.Background()
	src := seedLibrary(t, "test.xg", "lesson")
	var col int64
	for c, err := range src.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		col = c.ID
	}
	nameAnalyses(t, src, "Alice", "Bob")
	before := analysesByHash(t, src)

	copyCollection := func(omit bool, whole bool) (*sqlite.Storage, TransferReport) {
		dst, err := sqlite.Open(ctx, ":memory:", nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { dst.Close() })
		opts := WholeLibraryOptions()
		if !whole {
			opts.WholeLibrary = false
			opts.CollectionIDs = []int64{col}
		}
		opts.OmitProvenance = omit
		rep, err := Transfer(ctx, TransferEnd{S: src}, TransferEnd{S: dst}, opts, nil)
		if err != nil {
			t.Fatalf("transfer (omit provenance %v): %v", omit, err)
		}
		return dst, rep
	}

	kept, _ := copyCollection(false, false)
	withNames := analysesByHash(t, kept)
	named, played := 0, 0
	for _, a := range withNames {
		if a.Player1 == "Alice" && a.Player2 == "Bob" {
			named++
		}
		if len(a.PlayedMoves) > 0 || len(a.PlayedCubeActions) > 0 {
			played++
		}
	}
	if named != len(withNames) || played == 0 {
		t.Fatalf("by default the copy carries the players and the played moves: %d of %d analyses named, %d with played moves", named, len(withNames), played)
	}

	bare, _ := copyCollection(true, false)
	got := analysesByHash(t, bare)
	if len(got) == 0 || len(got) != len(withNames) {
		t.Fatalf("%d analyses arrived without provenance, %d with it", len(got), len(withNames))
	}
	for h, a := range got {
		if a.Player1 != "" || a.Player2 != "" || a.PlayedMove != "" || a.PlayedCubeAction != "" ||
			len(a.PlayedMoves) > 0 || len(a.PlayedCubeActions) > 0 {
			t.Errorf("position %x carries provenance: players %q/%q, played %q %v, cube %q %v",
				h, a.Player1, a.Player2, a.PlayedMove, a.PlayedMoves, a.PlayedCubeAction, a.PlayedCubeActions)
		}
		if a.CheckerAnalysis == nil && a.DoublingCubeAnalysis == nil {
			t.Errorf("position %x arrived without its evaluation", h)
		}
		if want := withoutProvenance(withNames[h]); !reflect.DeepEqual(withoutProvenance(a), want) {
			t.Errorf("position %x: the analysis differs beyond its provenance:\n got %+v\nwant %+v", h, withoutProvenance(a), want)
		}
	}

	whole, rep := copyCollection(true, true)
	if rep.Exported.Positions == 0 || rep.Exported.Matches != 0 || rep.Exported.Tournaments != 0 ||
		rep.Exported.Transcriptions != 0 || rep.Exported.Aliases != 0 {
		t.Errorf("a whole library without provenance read %+v, want positions and no match, tournament, transcription nor alias", rep.Exported)
	}
	for _, err := range whole.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		t.Error("the destination holds a match")
		break
	}
	for h, a := range analysesByHash(t, whole) {
		if a.Player1 != "" || a.Player2 != "" || len(a.PlayedMoves) > 0 || len(a.PlayedCubeActions) > 0 {
			t.Errorf("position %x of the whole library carries provenance", h)
			break
		}
	}

	if after := analysesByHash(t, src); !reflect.DeepEqual(after, before) {
		t.Error("a transfer changed the source's analyses")
	}
}

// nameAnalyses writes the players of a match into every analysis of s, as an
// import that knows them does.
func nameAnalyses(t *testing.T, s storage.Storage, p1, p2 string) {
	t.Helper()
	ctx := context.Background()
	var ids []int64
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	loaded, err := s.Analyses().LoadMany(ctx, "", ids)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range loaded {
		a.Player1, a.Player2 = p1, p2
	}
	if err := s.Analyses().SaveMany(ctx, "", loaded); err != nil {
		t.Fatal(err)
	}
}

// TestTransfer_OmitProvenanceRefusesNamedMatches: a match or a tournament named
// by id cannot leave without the names it is made of.
func TestTransfer_OmitProvenanceRefusesNamedMatches(t *testing.T) {
	ctx := context.Background()
	src := seedLibrary(t, "test.xg", "lesson")
	var match int64
	for m, err := range src.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		match = m.ID
	}
	dst, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	opts := WholeLibraryOptions()
	opts.MatchIDs = []int64{match}
	opts.OmitProvenance = true
	if _, err := Transfer(ctx, TransferEnd{S: src}, TransferEnd{S: dst}, opts, nil); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("a named match without provenance: %v, want ErrInvalid", err)
	}
}
