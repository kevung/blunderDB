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
	lid, err := src.Lessons().Create(ctx, "", "Bearoff", "d")
	if err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"one", "two"} {
		if _, err := src.Lessons().AddStep(ctx, "", lid, domain.LessonStep{Title: title, CollectionID: cid}); err != nil {
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
