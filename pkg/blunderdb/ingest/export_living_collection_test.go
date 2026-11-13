package ingest

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
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

// A living query that names rows by number is rewritten onto the file's
// numbers; one naming a row the file does not hold leaves frozen, holding what
// it selects at the source — never widened by a dropped token, never aimed at
// another row by a kept one.
func TestExportRenumbersOrFreezesALivingQueryNamingRows(t *testing.T) {
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
	matchID, err := src.Matches().Save(ctx, "", &domain.Match{Player1Name: "A", Player2Name: "B", MatchLength: 5})
	if err != nil {
		t.Fatal(err)
	}
	gameID, err := src.Matches().CreateGame(ctx, "", &domain.Game{MatchID: matchID, GameNumber: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := src.Matches().CreateMove(ctx, "", &domain.Move{GameID: gameID, MoveNumber: 1, PositionID: ids[2],
		MoveType: "checker", Player: 1, Dice: [2]int32{4, 2}, CheckerMove: "8/4 6/4"}); err != nil {
		t.Fatal(err)
	}
	diceOf := func(st storage.Storage, id int64) [2]int {
		p, err := st.Positions().Load(ctx, "", id)
		if err != nil {
			t.Fatalf("position %d: %v", id, err)
		}
		return p.Dice
	}

	cases := []struct {
		name       string
		query      string
		wantLiving bool
	}{
		// Only the last position leaves: it is numbered 1 in the file, not 3.
		{"position id", fmt.Sprintf("id%d", ids[2]), true},
		{"position id range", fmt.Sprintf("id%d,%d", ids[2], ids[2]+1), true},
		{"match not exported", fmt.Sprintf("ma%d", matchID), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			live, err := storage.CreateCollection(ctx, src, "", tc.name, "", tc.query)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = src.Collections().Delete(ctx, "", live) })
			path := filepath.Join(t.TempDir(), "export.db")
			sel := Selection{CollectionIDs: []int64{live}, CollectionPositions: true}
			if _, err := ExportSQLite(ctx, src, "", path, ExportOptions{Selection: sel}); err != nil {
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
			if len(got) != 1 {
				t.Fatalf("exported %d collections, want 1", len(got))
			}
			if living := got[0].FilterQuery != ""; living != tc.wantLiving {
				t.Fatalf("exported query %q, want living=%v", got[0].FilterQuery, tc.wantLiving)
			}
			ev, err := storage.EvaluateCollection(ctx, dst, "", got[0].ID, 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(ev.PositionIDs) != 1 || diceOf(dst, ev.PositionIDs[0]) != diceOf(src, ids[2]) {
				t.Errorf("exported collection (query %q) shows %v, want the position the source query selects", got[0].FilterQuery, ev.PositionIDs)
			}
		})
	}
}

func TestCarriedLivingQueryRenumbersALikeTarget(t *testing.T) {
	got, ok := carriedLivingQuery("like42<12", map[int64]int64{42: 7}, nil, nil)
	if f, _ := searchquery.Parse(got); !ok || f.LikeTargetID != 7 || f.LikeMaxDistance != 12 {
		t.Errorf("rewritten to %q (ok %v), want a like on 7 within 12", got, ok)
	}
	if _, ok := carriedLivingQuery("id42", map[int64]int64{}, nil, nil); ok {
		t.Error("an id list left empty must freeze the collection, not select everything")
	}
	if _, ok := carriedLivingQuery("like42", map[int64]int64{}, nil, nil); ok {
		t.Error("a like target the file does not hold must freeze the collection")
	}
	if got, ok := carriedLivingQuery("s n<1", nil, nil, nil); !ok || got != "s n<1" {
		t.Errorf("a query naming no row travels as written, got %q", got)
	}
}

// A living query the source refuses to evaluate (`like` outside a ranked
// search) leaves frozen by export and by transfer alike: one rule, so that a
// collection never arrives living through one path and frozen through the
// other.
func TestExportAndTransferFreezeALivingQueryTheSourceRefuses(t *testing.T) {
	ctx := context.Background()
	srcPath := filepath.Join(t.TempDir(), "club.db")
	src, err := sqlite.Open(ctx, srcPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := domain.InitializePosition()
	p.Dice = [2]int{3, 1}
	pid, err := src.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	cid, err := src.Collections().Create(ctx, "", "Proches", "")
	if err != nil {
		t.Fatal(err)
	}
	// Written below the validation, as a collection stored before the
	// refusal would be.
	if err := src.Collections().SetFilterQuery(ctx, "", cid, fmt.Sprintf("like%d", pid)); err != nil {
		t.Fatal(err)
	}
	if !livingRefused(cid, fmt.Sprintf("like%d", pid)) {
		t.Fatal("the source must refuse a like query outside a ranked search")
	}

	frozenIn := func(t *testing.T, st storage.Storage) {
		t.Helper()
		var got []storage.Collection
		for c, err := range st.Collections().List(ctx, "") {
			if err != nil {
				t.Fatal(err)
			}
			got = append(got, *c)
		}
		if len(got) != 1 || got[0].Name != "Proches" {
			t.Fatalf("collections = %+v, want only Proches", got)
		}
		if got[0].FilterQuery != "" {
			t.Errorf("arrived living with %q, want frozen", got[0].FilterQuery)
		}
		for m, err := range st.Collections().Members(ctx, "", got[0].ID) {
			if err != nil {
				t.Fatal(err)
			}
			t.Errorf("frozen collection holds %d, want nothing", m.PositionID)
		}
	}

	t.Run("export", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "export.db")
		sel := Selection{CollectionIDs: []int64{cid}, CollectionPositions: true}
		if _, err := ExportSQLite(ctx, src, "", path, ExportOptions{Selection: sel}); err != nil {
			t.Fatal(err)
		}
		dst, err := sqlite.Open(ctx, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer dst.Close()
		frozenIn(t, dst)
	})
	src.Close()

	t.Run("transfer", func(t *testing.T) {
		target := openMemTarget(t)
		sum, err := DBImporter{S: target}.Import(ctx, "", Source{Path: srcPath}, nil)
		if err != nil {
			t.Fatal(err)
		}
		frozenIn(t, target)
		if len(sum.LivingCollectionsFrozen) != 1 {
			t.Errorf("livingCollectionsFrozen = %v, want [Proches]", sum.LivingCollectionsFrozen)
		}
	})
}
