package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// testLessonLifecycle: a Lesson keeps its Steps in the order written, then
// in the order ReorderSteps sets; a deleted Collection or Position leaves
// its Step and text (ADR-0066).
func testLessonLifecycle(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ls := s.Lessons()
	if _, err := ls.Create(ctx, "", "  ", ""); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("Create with an empty name = %v, want ErrInvalid", err)
	}
	id, err := ls.Create(ctx, "", "Primes", "Jouer contre une prime")
	if err != nil {
		t.Fatal(err)
	}
	collID, err := s.Collections().Create(ctx, "", "Exemples", "")
	if err != nil {
		t.Fatal(err)
	}
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, "", &p)
	if err != nil {
		t.Fatal(err)
	}
	s1, err := ls.AddStep(ctx, "", id, domain.LessonStep{Title: "Intro", Text: "Lisez."})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := ls.AddStep(ctx, "", id, domain.LessonStep{Title: "Série", CollectionID: collID})
	if err != nil {
		t.Fatal(err)
	}
	s3, err := ls.AddStep(ctx, "", id, domain.LessonStep{Title: "Une", PositionID: posID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ls.AddStep(ctx, "", id, domain.LessonStep{PositionID: posID + 1000}); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("AddStep to a missing position = %v, want ErrInvalid", err)
	}
	if _, err := ls.AddStep(ctx, "", id+1000, domain.LessonStep{}); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("AddStep to a missing lesson = %v, want ErrNotFound", err)
	}
	l, err := ls.Get(ctx, "", id)
	if err != nil {
		t.Fatal(err)
	}
	if got := stepIDs(l); !equalIDs(got, []int64{s1, s2, s3}) || l.StepCount != 3 {
		t.Fatalf("steps = %v (count %d), want %v", got, l.StepCount, []int64{s1, s2, s3})
	}
	if err := ls.ReorderSteps(ctx, "", id, []int64{s3, s1}); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("ReorderSteps missing a step = %v, want ErrInvalid", err)
	}
	if err := ls.ReorderSteps(ctx, "", id, []int64{s3, s1, s2}); err != nil {
		t.Fatal(err)
	}
	if err := ls.UpdateStep(ctx, "", domain.LessonStep{ID: s1, Title: "Début", Text: "Lisez bien.", PositionID: posID}); err != nil {
		t.Fatal(err)
	}
	if err := ls.Update(ctx, "", id, "Les primes", "d"); err != nil {
		t.Fatal(err)
	}
	if err := s.Collections().Delete(ctx, "", collID); err != nil {
		t.Fatal(err)
	}
	l, err = ls.Get(ctx, "", id)
	if err != nil {
		t.Fatal(err)
	}
	if got := stepIDs(l); !equalIDs(got, []int64{s3, s1, s2}) {
		t.Fatalf("reordered steps = %v", got)
	}
	if l.Name != "Les primes" || l.Steps[1].Text != "Lisez bien." || l.Steps[1].PositionID != posID {
		t.Fatalf("after updates = %+v", l)
	}
	if l.Steps[2].CollectionID != 0 || l.Steps[2].Title != "Série" {
		t.Fatalf("a deleted collection must leave its step: %+v", l.Steps[2])
	}
	if err := ls.RemoveStep(ctx, "", s3); err != nil {
		t.Fatal(err)
	}
	list, err := ls.List(ctx, "")
	if err != nil || len(list) != 1 || list[0].StepCount != 2 {
		t.Fatalf("List = %+v, %v; want one lesson of two steps", list, err)
	}
	if err := ls.Delete(ctx, "", id); err != nil {
		t.Fatal(err)
	}
	if _, err := ls.Get(ctx, "", id); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("Get after Delete = %v, want ErrNotFound", err)
	}
	if _, err := s.Positions().Load(ctx, "", posID); err != nil {
		t.Fatalf("deleting a lesson must keep its positions: %v", err)
	}
}

func stepIDs(l *domain.Lesson) []int64 {
	var out []int64
	for _, st := range l.Steps {
		out = append(out, st.ID)
	}
	return out
}

// checkLessonIsolation: another tenant neither reads nor edits this tenant's
// Lessons, nor points a Step of its own at this tenant's positions.
func checkLessonIsolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	ls := s.Lessons()
	id, err := ls.Create(ctx, a, "Primes", "")
	if err != nil {
		t.Fatal(err)
	}
	p := checkerPos()
	posID, err := s.Positions().Save(ctx, a, &p)
	if err != nil {
		t.Fatal(err)
	}
	stepID, err := ls.AddStep(ctx, a, id, domain.LessonStep{PositionID: posID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ls.Get(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Get(%s) = %v, want ErrNotFound", b, err)
	}
	if list, err := ls.List(ctx, b); err != nil || len(list) != 0 {
		t.Errorf("List(%s) = %v, %v; want none", b, list, err)
	}
	if err := ls.Update(ctx, b, id, "x", ""); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Update(%s) = %v, want ErrNotFound", b, err)
	}
	if _, err := ls.AddStep(ctx, b, id, domain.LessonStep{}); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("AddStep(%s) = %v, want ErrNotFound", b, err)
	}
	if err := ls.RemoveStep(ctx, b, stepID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("RemoveStep(%s) = %v, want ErrNotFound", b, err)
	}
	own, err := ls.Create(ctx, b, "Mine", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ls.AddStep(ctx, b, own, domain.LessonStep{PositionID: posID}); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("AddStep(%s) on %s's position = %v, want ErrInvalid", b, a, err)
	}
	if err := ls.Delete(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Delete(%s) = %v, want ErrNotFound", b, err)
	}
	if l, err := ls.Get(ctx, a, id); err != nil || len(l.Steps) != 1 {
		t.Fatalf("tenant %s's lesson after %s's attempts = %+v, %v", a, b, l, err)
	}
}
