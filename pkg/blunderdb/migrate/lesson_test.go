package migrate

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// A Lesson travels with its Steps, in order, and a Step's Collection follows
// the remap instead of keeping the source's id.
func TestRunCarriesLessons(t *testing.T) {
	ctx := context.Background()
	open := func() storage.Storage {
		s, err := sqlite.Open(ctx, ":memory:", nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		if err := s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		return s
	}
	src, dst := open(), open()
	// An extra collection in the destination-to-be shifts the ids.
	if _, err := src.Collections().Create(ctx, "", "first", ""); err != nil {
		t.Fatal(err)
	}
	cid, err := src.Collections().Create(ctx, "", "second", "")
	if err != nil {
		t.Fatal(err)
	}
	// Positions saved in the destination first would shift ids if the remap were skipped.
	for i := 0; i < 2; i++ {
		p := domain.InitializePosition()
		p.DecisionType = domain.CheckerAction
		p.Dice = [2]int{3 + i, 1}
		if _, err := src.Positions().Save(ctx, "", &p); err != nil {
			t.Fatal(err)
		}
	}
	posList := []int64{}
	for p, err := range src.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		posList = append(posList, p.ID)
	}
	lid, err := src.Lessons().Create(ctx, "", "Bearoff", "d")
	if err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"one", "two"} {
		if _, err := src.Lessons().AddStep(ctx, "", lid, domain.LessonStep{Title: title, CollectionID: cid, PositionID: posList[len(posList)-1]}); err != nil {
			t.Fatal(err)
		}
	}

	rep, err := Run(ctx, src, dst, "", Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Lessons != 1 {
		t.Fatalf("report counts %d lessons, want 1", rep.Lessons)
	}
	ls, err := dst.Lessons().List(ctx, "")
	if err != nil || len(ls) != 1 {
		t.Fatalf("destination lessons: %v, %v", ls, err)
	}
	l, err := dst.Lessons().Get(ctx, "", ls[0].ID)
	if err != nil || len(l.Steps) != 2 || l.Steps[0].Title != "one" || l.Steps[1].Title != "two" {
		t.Fatalf("steps not carried in order: %+v, %v", l, err)
	}
	c, err := dst.Collections().Get(ctx, "", l.Steps[0].CollectionID)
	if err != nil || c.Name != "second" {
		t.Fatalf("step points at %+v (%v), want the migrated collection %q", c, err, "second")
	}
}

func TestRunCarriesLessonStepPosition(t *testing.T) {
	ctx := context.Background()
	open := func() storage.Storage {
		s, err := sqlite.Open(ctx, ":memory:", nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { s.Close() })
		if err := s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		return s
	}
	src, dst := open(), open()
	a := domain.InitializePosition()
	a.DecisionType = domain.CheckerAction
	a.Dice = [2]int{2, 1}
	b := a
	b.Dice = [2]int{6, 5}
	if _, err := src.Positions().Save(ctx, "", &a); err != nil {
		t.Fatal(err)
	}
	bid, err := src.Positions().Save(ctx, "", &b)
	if err != nil {
		t.Fatal(err)
	}
	lid, _ := src.Lessons().Create(ctx, "", "L", "")
	if _, err := src.Lessons().AddStep(ctx, "", lid, domain.LessonStep{Title: "s", PositionID: bid}); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(ctx, src, dst, "", Options{}); err != nil {
		t.Fatal(err)
	}
	ls, _ := dst.Lessons().List(ctx, "")
	l, err := dst.Lessons().Get(ctx, "", ls[0].ID)
	if err != nil || len(l.Steps) != 1 || l.Steps[0].PositionID == 0 {
		t.Fatalf("step lost its position: %+v, %v", l, err)
	}
	got, err := dst.Positions().Load(ctx, "", l.Steps[0].PositionID)
	if err != nil || got.Dice != [2]int{6, 5} {
		t.Fatalf("step points at %+v (%v), want the 6-5 position", got, err)
	}
}
