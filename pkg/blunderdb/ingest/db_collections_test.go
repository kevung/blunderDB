package ingest

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func TestDBImportCollections_SQLite(t *testing.T) {
	target, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	checkDBImportCollections(t, target, "")
}

// checkDBImportCollections: a native .db import brings its collections along,
// members remapped onto the target's ids. A collection merges into the
// target's one of the same name — members already there keep their place, none
// is doubled — and a second import of the same file changes nothing.
func checkDBImportCollections(t *testing.T, target storage.Storage, scope string) {
	t.Helper()
	ctx := context.Background()

	opening := domain.InitializePosition()
	reply := domain.InitializePosition()
	reply.Dice = [2]int{6, 5}
	own := domain.InitializePosition()
	own.Dice = [2]int{4, 2}

	srcPath := filepath.Join(t.TempDir(), "club.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	so, _ := src.Positions().Save(ctx, "", &opening)
	sr, _ := src.Positions().Save(ctx, "", &reply)
	cOpen, _ := src.Collections().Create(ctx, "", "Ouvertures", "du club")
	if err := src.Collections().AddPositions(ctx, "", cOpen, []int64{so, sr}); err != nil {
		t.Fatal(err)
	}
	cLive, _ := src.Collections().Create(ctx, "", "Gaffes", "")
	if err := src.Collections().SetFilterQuery(ctx, "", cLive, "e>0.1"); err != nil {
		t.Fatal(err)
	}
	src.Close()

	// The target already holds its own "Ouvertures", with the reply and a
	// position the source never had.
	tOwn, err := target.Positions().Save(ctx, scope, &own)
	if err != nil {
		t.Fatal(err)
	}
	tReply, err := target.Positions().Save(ctx, scope, &reply)
	if err != nil {
		t.Fatal(err)
	}
	tColl, err := target.Collections().Create(ctx, scope, "Ouvertures", "la mienne")
	if err != nil {
		t.Fatal(err)
	}
	if err := target.Collections().AddPositions(ctx, scope, tColl, []int64{tReply, tOwn}); err != nil {
		t.Fatal(err)
	}

	for pass := 1; pass <= 2; pass++ {
		sum, err := DBImporter{S: target}.Import(ctx, scope, Source{Format: FormatNativeDB, Path: srcPath}, nil)
		if err != nil {
			t.Fatalf("pass %d: import: %v", pass, err)
		}
		if sum.Collections != 2 {
			t.Errorf("pass %d: Summary.Collections = %d, want 2", pass, sum.Collections)
		}

		colls := map[string]storage.Collection{}
		for c, err := range target.Collections().List(ctx, scope) {
			if err != nil {
				t.Fatal(err)
			}
			colls[c.Name] = *c
		}
		if len(colls) != 2 {
			t.Fatalf("pass %d: target collections = %v, want Ouvertures and Gaffes", pass, colls)
		}
		if got := colls["Ouvertures"]; got.ID != tColl || got.Description != "la mienne" {
			t.Errorf("pass %d: Ouvertures = %+v; want the target's own, untouched", pass, got)
		}
		if got := colls["Gaffes"]; got.FilterQuery != "e>0.1" {
			t.Errorf("pass %d: Gaffes query = %q, want the source's", pass, got.FilterQuery)
		}

		var members []int64
		for m, err := range target.Collections().Members(ctx, scope, tColl) {
			if err != nil {
				t.Fatal(err)
			}
			members = append(members, m.PositionID)
		}
		tOpening := members[len(members)-1]
		if want := []int64{tReply, tOwn, tOpening}; !reflect.DeepEqual(members, want) || tOpening == tReply || tOpening == tOwn {
			t.Errorf("pass %d: Ouvertures members = %v, want the target's two then the opening", pass, members)
		}
	}
}
