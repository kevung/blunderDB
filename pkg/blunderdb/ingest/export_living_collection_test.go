package ingest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A LIVING collection leaves living: its query travels, and with its positions
// asked for, the positions the query selects now travel too, so that the
// recipient's evaluation of the query finds them in the file.
func TestExportKeepsALivingCollectionLiving(t *testing.T) {
	ctx := context.Background()
	src, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()

	var ids []int64
	for _, dice := range [][2]int{{3, 1}, {6, 5}, {4, 2}} {
		p := domain.InitializePosition()
		p.Dice = dice
		id, err := src.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	// Nothing was ever played: the query selects every position.
	const query = "s n<1"
	live, err := storage.CreateCollection(ctx, src, "", "Vivante", "toutes", query)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name          string
		sel           Selection
		wantPositions int
	}{
		{"with its positions", Selection{CollectionIDs: []int64{live}, CollectionPositions: true}, len(ids)},
		{"alone", Selection{PositionIDs: ids[:1], CollectionIDs: []int64{live}}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "export.db")
			if _, err := ExportSQLite(ctx, src, "", path, ExportOptions{Selection: tc.sel}); err != nil {
				t.Fatal(err)
			}
			dst, err := sqlite.Open(ctx, path, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer dst.Close()

			var got []storage.Collection
			for c, err := range dst.Collections().List(ctx, "") {
				if err != nil {
					t.Fatal(err)
				}
				got = append(got, *c)
			}
			if len(got) != 1 || got[0].FilterQuery != query || got[0].Description != "toutes" {
				t.Fatalf("exported collections %+v, want one living on %q", got, query)
			}
			ev, err := storage.EvaluateCollection(ctx, dst, "", got[0].ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			if !ev.Living || ev.Total != tc.wantPositions {
				t.Errorf("exported collection evaluates to %d positions (living %v), want %d living", ev.Total, ev.Living, tc.wantPositions)
			}
		})
	}
}
