// Contract case for the drawing of a position: arrows and squares in a closed
// set of colours, replaced whole, gone with their position.
package storagetest

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testPositionDrawings(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	prov, ok := s.(storage.DrawingProvider)
	if !ok {
		t.Fatal("this backend keeps no drawings")
	}
	d := prov.PositionDrawings()
	pa, pb := provenancePos(3), provenancePos(4)
	a, err := s.Positions().Save(ctx, "", &pa)
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Positions().Save(ctx, "", &pb)
	if err != nil {
		t.Fatal(err)
	}

	if got, err := d.Drawing(ctx, "", a); err != nil || len(got) != 0 {
		t.Fatalf("a fresh position has no drawing: %v, %v", got, err)
	}
	if _, err := d.Drawing(ctx, "", 987654); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("drawing of a missing position = %v, want ErrNotFound", err)
	}

	three := []domain.DrawingMark{
		{Kind: domain.DrawingArrow, From: 13, To: 7, Color: "red"},
		{Kind: domain.DrawingSquare, From: 6, To: 6, Color: "green"},
		{Kind: domain.DrawingArrow, From: 24, To: 0, Color: "blue"},
	}
	if err := d.SetDrawing(ctx, "", a, three); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.Drawing(ctx, "", a); !reflect.DeepEqual(got, three) {
		t.Errorf("drawing = %+v, want %+v in the order drawn", got, three)
	}
	// A square names one point: its To is its From, whatever was sent.
	if err := d.SetDrawing(ctx, "", b, []domain.DrawingMark{{Kind: domain.DrawingSquare, From: 8, To: 3, Color: "yellow"}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.Drawing(ctx, "", b); len(got) != 1 || got[0].To != 8 {
		t.Errorf("a square is on one point, got %+v", got)
	}

	all, err := d.Drawings(ctx, "", []int64{a, b, 987654})
	if err != nil || len(all) != 2 || len(all[a]) != 3 || len(all[b]) != 1 {
		t.Errorf("Drawings = %+v, %v: want both positions, none for the missing one", all, err)
	}

	bad := map[string][]domain.DrawingMark{
		"colour":    {{Kind: domain.DrawingSquare, From: 1, To: 1, Color: "pink"}},
		"kind":      {{Kind: "circle", From: 1, To: 1, Color: "red"}},
		"point":     {{Kind: domain.DrawingSquare, From: 26, To: 26, Color: "red"}},
		"self":      {{Kind: domain.DrawingArrow, From: 5, To: 5, Color: "red"}},
		"too many":  make([]domain.DrawingMark, domain.MaxDrawingMarks+1),
		"half good": {three[0], {Kind: "x"}},
	}
	for i := range bad["too many"] {
		bad["too many"][i] = domain.DrawingMark{Kind: domain.DrawingSquare, From: 1, To: 1, Color: "red"}
	}
	for name, marks := range bad {
		if err := d.SetDrawing(ctx, "", a, marks); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("%s: SetDrawing = %v, want ErrInvalid", name, err)
		}
	}
	if got, _ := d.Drawing(ctx, "", a); !reflect.DeepEqual(got, three) {
		t.Errorf("a refused drawing changed the previous one: %+v", got)
	}
	if err := d.SetDrawing(ctx, "", 987654, three); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetDrawing on a missing position = %v, want ErrNotFound", err)
	}

	if err := d.SetDrawing(ctx, "", a, three[:1]); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.Drawing(ctx, "", a); len(got) != 1 {
		t.Errorf("SetDrawing replaces, got %+v", got)
	}
	if err := d.SetDrawing(ctx, "", a, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.Drawing(ctx, "", a); len(got) != 0 {
		t.Errorf("no marks clears the drawing, got %+v", got)
	}

	if err := s.Positions().Delete(ctx, "", b); err != nil {
		t.Fatal(err)
	}
	if all, _ := d.Drawings(ctx, "", []int64{b}); len(all) != 0 {
		t.Errorf("a deleted position keeps its drawing: %+v", all)
	}
}
