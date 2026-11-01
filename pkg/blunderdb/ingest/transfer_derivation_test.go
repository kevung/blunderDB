package ingest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// derivationOf is the derivation of the position with that Zobrist hash and
// the Zobrist hash of its origin (0 when it derives from nothing).
func derivationOf(t *testing.T, s storage.Storage, zobrist uint64) (originZ uint64, move string) {
	t.Helper()
	ctx := context.Background()
	id, found, err := s.Positions().Exists(ctx, "", zobrist)
	if err != nil || !found {
		t.Fatalf("position %d not found: %v", zobrist, err)
	}
	d, err := s.(storage.DerivationProvider).PositionDerivations().DerivedFrom(ctx, "", id)
	if err != nil {
		t.Fatal(err)
	}
	if d == nil {
		return 0, ""
	}
	loaded, err := s.Positions().LoadByIDs(ctx, "", []int64{d.FromID})
	if err != nil || len(loaded) != 1 {
		t.Fatalf("origin %d: %v", d.FromID, err)
	}
	return engine.ZobristHash(&loaded[0]), d.Move
}

// A variant leaves with its link to the origin, in an exports.sqlite file and
// in a transfer, under the ids of the destination; a link the destination
// already has stays; a link whose origin is not carried is dropped.
func TestTransfer_CarriesDerivation(t *testing.T) {
	ctx := context.Background()
	a := seedLibrary(t, "match_with_comment.xg", "from A")
	origin, variant := firstTwo(t, a)
	zo, zv := engine.ZobristHash(origin), engine.ZobristHash(variant)
	if err := a.PositionDerivations().SetDerivedFrom(ctx, "", variant.ID, &domain.Derivation{FromID: origin.ID, Move: "13/7 24/23"}); err != nil {
		t.Fatal(err)
	}

	file := filepath.Join(t.TempDir(), "out.db")
	if _, err := ExportSQLite(ctx, a, "", file, WholeTenant(FormatSQLite)); err != nil {
		t.Fatal(err)
	}
	out, err := sqlite.Open(ctx, file, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if oz, mv := derivationOf(t, out, zv); oz != zo || mv != "13/7 24/23" {
		t.Errorf("exported derivation = (%d, %q), want (%d, %q)", oz, mv, zo, "13/7 24/23")
	}

	// A selection that leaves the origin out carries no link, not a dangling one.
	only := filepath.Join(t.TempDir(), "only.db")
	sel := ExportOptions{Format: FormatSQLite, Selection: Selection{Positions: []*domain.Position{variant}}}
	if _, err := ExportSQLite(ctx, a, "", only, sel); err != nil {
		t.Fatal(err)
	}
	solo, err := sqlite.Open(ctx, only, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer solo.Close()
	if oz, _ := derivationOf(t, solo, zv); oz != 0 {
		t.Errorf("an export without the origin kept a link to %d", oz)
	}

	b := seedLibrary(t, "match_with_comment.xg", "from B")
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: b}, WholeLibraryOptions(), nil); err != nil {
		t.Fatal(err)
	}
	if oz, mv := derivationOf(t, b, zv); oz != zo || mv != "13/7 24/23" {
		t.Errorf("transferred derivation = (%d, %q), want (%d, %q)", oz, mv, zo, "13/7 24/23")
	}
	// The destination's own origin for that variant is not overwritten.
	third := seedLibrary(t, "match_with_comment.xg", "from C")
	var ps []*domain.Position
	for p, err := range third.Positions().List(ctx, "", storage.ListOpts{Limit: 3}) {
		if err != nil {
			t.Fatal(err)
		}
		pc := *p
		ps = append(ps, &pc)
	}
	if len(ps) < 3 {
		t.Skip("fixture has fewer than three positions")
	}
	if err := third.PositionDerivations().SetDerivedFrom(ctx, "", ps[1].ID, &domain.Derivation{FromID: ps[2].ID, Move: "own"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Transfer(ctx, TransferEnd{S: a}, TransferEnd{S: third}, WholeLibraryOptions(), nil); err != nil {
		t.Fatal(err)
	}
	if oz, mv := derivationOf(t, third, engine.ZobristHash(ps[1])); oz != engine.ZobristHash(ps[2]) || mv != "own" {
		t.Errorf("a transfer overwrote the destination's own derivation: (%d, %q)", oz, mv)
	}
}
