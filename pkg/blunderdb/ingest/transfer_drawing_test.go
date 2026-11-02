package ingest

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

func drawOn(t *testing.T, s *sqlite.Storage, p *domain.Position, marks []domain.DrawingMark) {
	t.Helper()
	if err := s.PositionDrawings().SetDrawing(context.Background(), "", p.ID, marks); err != nil {
		t.Fatal(err)
	}
}

func drawingOf(t *testing.T, s storage.Storage, zobrist uint64) []domain.DrawingMark {
	t.Helper()
	ctx := context.Background()
	id, found, err := s.Positions().Exists(ctx, "", zobrist)
	if err != nil || !found {
		t.Fatalf("position %d not found: %v", zobrist, err)
	}
	marks, err := s.(storage.DrawingProvider).PositionDrawings().Drawing(ctx, "", id)
	if err != nil {
		t.Fatal(err)
	}
	return marks
}

func firstTwo(t *testing.T, s storage.Storage) (a, b *domain.Position) {
	t.Helper()
	var ps []*domain.Position
	for p, err := range s.Positions().List(context.Background(), "", storage.ListOpts{Limit: 2}) {
		if err != nil {
			t.Fatal(err)
		}
		pc := *p
		ps = append(ps, &pc)
	}
	if len(ps) < 2 {
		t.Fatal("fixture has fewer than two positions")
	}
	return ps[0], ps[1]
}

var (
	redArrow  = []domain.DrawingMark{{Kind: domain.DrawingArrow, From: 13, To: 7, Color: "red"}}
	blueBoxes = []domain.DrawingMark{{Kind: domain.DrawingSquare, From: 6, To: 6, Color: "blue"}, {Kind: domain.DrawingSquare, From: 8, To: 8, Color: "green"}}
)

// A drawing leaves with its position, in an exports.sqlite file and in a
// transfer; a drawing the destination already has on that position stays.
func TestTransfer_CarriesDrawing(t *testing.T) {
	ctx := context.Background()
	a := seedLibrary(t, "match_with_comment.xg", "from A")
	p1, p2 := firstTwo(t, a)
	drawOn(t, a, p1, redArrow)
	drawOn(t, a, p2, blueBoxes)
	z1, z2 := engine.ZobristHash(p1), engine.ZobristHash(p2)

	// Export to a file: the drawing is in it.
	file := filepath.Join(t.TempDir(), "out.db")
	if _, err := ExportSQLite(ctx, a, "", file, WholeTenant(FormatSQLite)); err != nil {
		t.Fatal(err)
	}
	out, err := sqlite.Open(ctx, file, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if got := drawingOf(t, out, z1); !reflect.DeepEqual(got, redArrow) {
		t.Errorf("exported drawing of p1 = %+v, want %+v", got, redArrow)
	}

	// Transfer into a library that holds the same positions: p2 already
	// carries a drawing of its own, p1 none.
	b := seedLibrary(t, "match_with_comment.xg", "from B")
	b1, b2 := firstTwo(t, b)
	own := []domain.DrawingMark{{Kind: domain.DrawingArrow, From: 24, To: 18, Color: "yellow"}}
	drawOn(t, b, b2, own)
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, WholeLibraryOptions(), nil); err != nil {
		t.Fatal(err)
	}
	if engine.ZobristHash(b1) != z1 || engine.ZobristHash(b2) != z2 {
		t.Skip("the two seeds do not list the same positions first")
	}
	if got := drawingOf(t, b, z1); !reflect.DeepEqual(got, redArrow) {
		t.Errorf("transferred drawing of p1 = %+v, want %+v", got, redArrow)
	}
	if got := drawingOf(t, b, z2); !reflect.DeepEqual(got, own) {
		t.Errorf("a transfer overwrote the destination's own drawing: %+v, want %+v", got, own)
	}
	// Replaying changes nothing.
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, WholeLibraryOptions(), nil); err != nil {
		t.Fatal(err)
	}
	if got := drawingOf(t, b, z1); !reflect.DeepEqual(got, redArrow) {
		t.Errorf("after a replay p1 = %+v", got)
	}
}
