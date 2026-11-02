// Contract cases for what the cascade takes off a deleted position besides
// its analysis and notes: its label, its drawing and its derivation links.
package storagetest

import (
	"context"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/trash"
)

// positionMarks is what markPosition laid on a position.
type positionMarks struct {
	publicID string
	drawing  []domain.DrawingMark
	origin   int64
	variant  int64
}

// markPosition gives pos a public id, a drawing, an origin and a variant.
func markPosition(t *testing.T, s storage.Storage, pos, origin, variant int64) positionMarks {
	t.Helper()
	ctx := context.Background()
	m := positionMarks{
		publicID: "pub-trash-1",
		drawing: []domain.DrawingMark{
			{Kind: domain.DrawingArrow, From: 13, To: 7, Color: "red"},
			{Kind: domain.DrawingSquare, From: 6, To: 6, Color: "green"},
		},
		origin: origin, variant: variant,
	}
	if err := s.(storage.PublicIDProvider).PublicIDs().SetPublicID(ctx, "", pos, m.publicID); err != nil {
		t.Fatal(err)
	}
	if err := s.(storage.DrawingProvider).PositionDrawings().SetDrawing(ctx, "", pos, m.drawing); err != nil {
		t.Fatal(err)
	}
	d := s.(storage.DerivationProvider).PositionDerivations()
	if err := d.SetDerivedFrom(ctx, "", pos, &domain.Derivation{FromID: origin, Move: "13/7"}); err != nil {
		t.Fatal(err)
	}
	if err := d.SetDerivedFrom(ctx, "", variant, &domain.Derivation{FromID: pos, Move: "6/1"}); err != nil {
		t.Fatal(err)
	}
	return m
}

// check says the position has all its marks back.
func (m positionMarks) check(t *testing.T, s storage.Storage, pos int64) {
	t.Helper()
	ctx := context.Background()
	if got, err := s.(storage.PublicIDProvider).PublicIDs().PublicID(ctx, "", pos); err != nil || got != m.publicID {
		t.Errorf("public id after restore = %q, %v; want %q", got, err, m.publicID)
	}
	if got, err := s.(storage.DrawingProvider).PositionDrawings().Drawing(ctx, "", pos); err != nil || !reflect.DeepEqual(got, m.drawing) {
		t.Errorf("drawing after restore = %+v, %v; want %+v", got, err, m.drawing)
	}
	d := s.(storage.DerivationProvider).PositionDerivations()
	if got, err := d.DerivedFrom(ctx, "", pos); err != nil || got == nil || got.FromID != m.origin || got.Move != "13/7" {
		t.Errorf("origin after restore = %+v, %v; want position %d by 13/7", got, err, m.origin)
	}
	if got, err := d.Variants(ctx, "", pos); err != nil || !reflect.DeepEqual(got, []int64{m.variant}) {
		t.Errorf("variants after restore = %v, %v; want [%d]", got, err, m.variant)
	}
	if got, err := d.DerivedFrom(ctx, "", m.variant); err != nil || got == nil || got.Move != "6/1" {
		t.Errorf("the variant's move after restore = %+v, %v; want 6/1", got, err)
	}
}

// testTrashPositionKeepsMarks: trashing a position then restoring it gives it
// back its label, its drawing and both directions of its derivation. A
// drawing or a label written since the delete is newer work and stays.
func testTrashPositionKeepsMarks(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	save := func(n int) int64 {
		p := provenancePos(n)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	pos, origin, variant := save(41), save(42), save(43)
	marks := markPosition(t, s, pos, origin, variant)

	entry, err := trash.Position(ctx, s, "", pos)
	if err != nil {
		t.Fatalf("trash.Position: %v", err)
	}
	if got, err := s.(storage.DerivationProvider).PositionDerivations().DerivedFrom(ctx, "", variant); err != nil || got != nil {
		t.Fatalf("the variant keeps a link to a deleted position: %+v, %v", got, err)
	}
	res, err := trash.Restore(ctx, s, "", entry)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if res.ID != pos {
		t.Fatalf("restored as %d, want its own id %d", res.ID, pos)
	}
	marks.check(t, s, pos)

	// Second trip: while the position waits in the trash its variant is
	// derived from another one; restoring must not take that link back.
	entry, err = trash.Position(ctx, s, "", pos)
	if err != nil {
		t.Fatal(err)
	}
	other := save(44)
	d := s.(storage.DerivationProvider).PositionDerivations()
	if err := d.SetDerivedFrom(ctx, "", variant, &domain.Derivation{FromID: other, Move: "newer"}); err != nil {
		t.Fatal(err)
	}
	if _, err = trash.Restore(ctx, s, "", entry); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.DerivedFrom(ctx, "", variant); got == nil || got.FromID != other {
		t.Errorf("a link written since the delete was overwritten: %+v", got)
	}
}
