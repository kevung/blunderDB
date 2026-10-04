package database

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// TestCommitImportDatabase_Collections: the desktop import of a .db follows
// the daemon's rule (ingest.MergeCollections) — merge by name, members
// remapped and appended without duplicates, a living target left alone, a
// second import of the same file changing nothing.
func TestCommitImportDatabase_Collections(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	opening := domain.InitializePosition()
	reply := domain.InitializePosition()
	reply.Dice = [2]int{6, 5}
	own := domain.InitializePosition()
	own.Dice = [2]int{4, 2}

	srcPath := filepath.Join(dir, "club.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	so, _ := src.Positions().Save(ctx, "", &opening)
	sr, _ := src.Positions().Save(ctx, "", &reply)
	cOpen, _ := src.Collections().Create(ctx, "", "Ouvertures", "")
	_ = src.Collections().AddPositions(ctx, "", cOpen, []int64{so, sr})
	cLive, _ := src.Collections().Create(ctx, "", "Gaffes", "")
	_ = src.Collections().SetFilterQuery(ctx, "", cLive, "e>0.1")
	cPlain, _ := src.Collections().Create(ctx, "", "Vivante", "")
	_ = src.Collections().AddPositions(ctx, "", cPlain, []int64{so})
	src.Close()

	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(dir, "mine.db")); err != nil {
		t.Fatal(err)
	}
	// Windows cannot remove a database file still open when TempDir cleans up.
	t.Cleanup(func() { _ = d.Close() })
	tOwn, _ := d.SavePosition(&own)
	tReply, _ := d.SavePosition(&reply)
	tColl, _ := d.CreateCollection("Ouvertures", "")
	if err := d.AddPositionsToCollection(tColl, []int64{tReply, tOwn}); err != nil {
		t.Fatal(err)
	}
	tLive, _ := d.CreateCollection("Vivante", "")
	if err := d.SetCollectionFilter(tLive, "s>0"); err != nil {
		t.Fatal(err)
	}

	for pass, want := range []int{2, 0} {
		res, err := d.CommitImportDatabase(srcPath)
		if err != nil {
			t.Fatalf("pass %d: %v", pass+1, err)
		}
		if res["collections"] != want {
			t.Errorf("pass %d: collections = %v, want %d", pass+1, res["collections"], want)
		}
		if !reflect.DeepEqual(res["livingCollectionsSkipped"], []string{"Vivante"}) {
			t.Errorf("pass %d: livingCollectionsSkipped = %v", pass+1, res["livingCollectionsSkipped"])
		}
		colls, _ := d.GetAllCollections()
		if len(colls) != 3 {
			t.Fatalf("pass %d: %d collections, want Ouvertures, Vivante, Gaffes", pass+1, len(colls))
		}
		for _, c := range colls {
			if c.Name == "Gaffes" && c.FilterQuery != "e>0.1" {
				t.Errorf("pass %d: Gaffes query = %q", pass+1, c.FilterQuery)
			}
		}
		ps, _ := d.GetCollectionPositions(tColl)
		var ids []int64
		for _, p := range ps {
			ids = append(ids, p.ID)
		}
		if len(ids) != 3 || ids[0] != tReply || ids[1] != tOwn || ids[2] == tReply || ids[2] == tOwn {
			t.Errorf("pass %d: Ouvertures = %v, want %d, %d then the opening", pass+1, ids, tReply, tOwn)
		}
	}
}
