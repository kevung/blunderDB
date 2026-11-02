// Contract case for the derivation of a position: one origin and the move
// played from it, replaced whole, never a loop, gone with either position.
package storagetest

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testPositionDerivations(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	prov, ok := s.(storage.DerivationProvider)
	if !ok {
		t.Fatal("this backend keeps no derivations")
	}
	d := prov.PositionDerivations()
	save := func(seed int) int64 {
		p := provenancePos(seed)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	root, v1, v2 := save(3), save(4), save(5)

	if got, err := d.DerivedFrom(ctx, "", v1); err != nil || got != nil {
		t.Fatalf("a fresh position derives from nothing: %v, %v", got, err)
	}
	if _, err := d.DerivedFrom(ctx, "", 987654); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("DerivedFrom of a missing position = %v, want ErrNotFound", err)
	}

	want := domain.Derivation{FromID: root, Move: "13/7 24/23"}
	if err := d.SetDerivedFrom(ctx, "", v1, &want); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.DerivedFrom(ctx, "", v1); got == nil || *got != want {
		t.Errorf("DerivedFrom = %+v, want %+v", got, want)
	}
	if err := d.SetDerivedFrom(ctx, "", v2, &domain.Derivation{FromID: root}); err != nil {
		t.Fatal(err)
	}

	all, err := d.Derivations(ctx, "", []int64{root, v1, v2, 987654})
	if err != nil || len(all) != 2 || all[v1] != want || all[v2].FromID != root {
		t.Errorf("Derivations = %+v, %v: want v1 and v2 only", all, err)
	}
	if got, err := d.Variants(ctx, "", root); err != nil || !reflect.DeepEqual(got, []int64{v1, v2}) {
		t.Errorf("Variants(root) = %v, %v; want [%d %d]", got, err, v1, v2)
	}
	if got, err := d.Variants(ctx, "", v1); err != nil || len(got) != 0 {
		t.Errorf("Variants(v1) = %v, %v; want none", got, err)
	}
	if _, err := d.Variants(ctx, "", 987654); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Variants of a missing origin = %v, want ErrNotFound", err)
	}

	long := make([]byte, domain.MaxDerivationMove+1)
	for i := range long {
		long[i] = 'x'
	}
	bad := map[string]*domain.Derivation{
		"itself":         {FromID: v1},
		"no origin":      {FromID: 0},
		"missing origin": {FromID: 987654},
		"loop":           {FromID: v1}, // set on root below: root <- v1 <- root
		"long move":      {FromID: root, Move: string(long)},
	}
	if err := d.SetDerivedFrom(ctx, "", root, bad["loop"]); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("a loop: SetDerivedFrom = %v, want ErrInvalid", err)
	}
	delete(bad, "loop")
	for name, dv := range bad {
		if err := d.SetDerivedFrom(ctx, "", v1, dv); !errors.Is(err, storage.ErrInvalid) {
			t.Errorf("%s: SetDerivedFrom = %v, want ErrInvalid", name, err)
		}
	}
	if got, _ := d.DerivedFrom(ctx, "", v1); got == nil || *got != want {
		t.Errorf("a refused derivation changed the previous one: %+v", got)
	}
	if got, _ := d.DerivedFrom(ctx, "", root); got != nil {
		t.Errorf("a refused loop left a derivation on root: %+v", got)
	}
	if err := d.SetDerivedFrom(ctx, "", 987654, &want); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetDerivedFrom on a missing position = %v, want ErrNotFound", err)
	}

	// Replaced by a new origin, then cleared.
	if err := d.SetDerivedFrom(ctx, "", v1, &domain.Derivation{FromID: v2}); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.DerivedFrom(ctx, "", v1); got == nil || got.FromID != v2 || got.Move != "" {
		t.Errorf("SetDerivedFrom replaces, got %+v", got)
	}
	if err := d.SetDerivedFrom(ctx, "", v1, nil); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.DerivedFrom(ctx, "", v1); got != nil {
		t.Errorf("nil clears the derivation, got %+v", got)
	}

	// Deleting the origin drops the link, not the variant; deleting the variant
	// drops its link.
	if err := d.SetDerivedFrom(ctx, "", v1, &want); err != nil {
		t.Fatal(err)
	}
	if err := s.Positions().Delete(ctx, "", root); err != nil {
		t.Fatal(err)
	}
	if got, err := d.DerivedFrom(ctx, "", v2); err != nil || got != nil {
		t.Errorf("a deleted origin leaves a link: %+v, %v (the variant must stay)", got, err)
	}
	if err := s.Positions().Delete(ctx, "", v2); err != nil {
		t.Fatal(err)
	}
	if all, _ := d.Derivations(ctx, "", []int64{v2}); len(all) != 0 {
		t.Errorf("a deleted variant keeps its link: %+v", all)
	}
}

// testPositionDerivationsRingRace: N writers each make a position derive from
// its neighbour, closing a ring only if every one of them is accepted. Each
// write looks fine against the others' past, so only serialising the writes
// refuses one.
func testPositionDerivationsRingRace(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	d := s.(storage.DerivationProvider).PositionDerivations()
	const n = 4
	ids := make([]int64, n)
	for i := range ids {
		p := provenancePos(60 + i)
		id, err := s.Positions().Save(ctx, "", &p)
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = id
	}
	for round := 0; round < 5; round++ {
		for _, id := range ids {
			if err := d.SetDerivedFrom(ctx, "", id, nil); err != nil {
				t.Fatal(err)
			}
		}
		errs := make([]error, n)
		var wg sync.WaitGroup
		for i := range ids {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs[i] = d.SetDerivedFrom(ctx, "", ids[i], &domain.Derivation{FromID: ids[(i+1)%n]})
			}()
		}
		wg.Wait()
		accepted := 0
		for i, err := range errs {
			switch {
			case err == nil:
				accepted++
			case !errors.Is(err, storage.ErrInvalid):
				t.Fatalf("round %d: write %d failed otherwise than by refusal: %v", round, i, err)
			}
		}
		if accepted == n {
			t.Fatalf("round %d: all %d writes were accepted: the positions derive from each other in a ring", round, n)
		}
	}
}
