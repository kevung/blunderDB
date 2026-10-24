// Contract cases for the 2.31.0 schema: the library's match equity tables
// (ADR-0068), the student's progress through a Lesson (ADR-0069), and the
// resumable pass that stores each move's error.
// The tables that run them live in contract.go and tenant_isolation.go.
package storagetest

import (
	"context"
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func testMETLifecycle(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	mt := s.MatchEquityTables()
	if cur, err := mt.Current(ctx, ""); err != nil || cur != nil {
		t.Fatalf("Current on a new library = %+v, %v; want the built-in (nil)", cur, err)
	}
	if _, err := mt.Save(ctx, "", domain.MatchEquityTable{Name: "x", Digest: "d"}); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("Save without a source = %v, want ErrInvalid", err)
	}
	a, err := mt.Save(ctx, "", domain.MatchEquityTable{Name: "Rockwell-Kazaross", Digest: "aaa", Source: "<met/>"})
	if err != nil {
		t.Fatal(err)
	}
	again, err := mt.Save(ctx, "", domain.MatchEquityTable{Name: "renamed copy", Digest: "aaa", Source: "<met/>"})
	if err != nil || again != a {
		t.Fatalf("Save of a digest already held = %d, %v; want %d", again, err, a)
	}
	b, err := mt.Save(ctx, "", domain.MatchEquityTable{Name: "Jacobs-Trice", Digest: "bbb", Source: "<met>b</met>"})
	if err != nil {
		t.Fatal(err)
	}
	if err := mt.SetCurrent(ctx, "", a); err != nil {
		t.Fatal(err)
	}
	if err := mt.SetCurrent(ctx, "", b); err != nil {
		t.Fatal(err)
	}
	cur, err := mt.Current(ctx, "")
	if err != nil || cur == nil || cur.ID != b || cur.Source != "<met>b</met>" || !cur.Current {
		t.Fatalf("Current after SetCurrent(%d) = %+v, %v", b, cur, err)
	}
	list, err := mt.List(ctx, "")
	if err != nil || len(list) != 2 {
		t.Fatalf("List = %+v, %v; want two tables", list, err)
	}
	currents := 0
	for _, m := range list {
		if m.Source != "" {
			t.Errorf("List carries the source of %q", m.Name)
		}
		if m.Current {
			currents++
		}
	}
	if currents != 1 || list[0].Name != "Jacobs-Trice" || list[1].Name != "Rockwell-Kazaross" {
		t.Errorf("List = %+v; want by name with exactly one current", list)
	}
	if err := mt.SetCurrent(ctx, "", b+1000); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetCurrent(unknown) = %v, want ErrNotFound", err)
	}
	if err := mt.SetCurrent(ctx, "", 0); err != nil {
		t.Fatal(err)
	}
	if cur, err := mt.Current(ctx, ""); err != nil || cur != nil {
		t.Errorf("Current after SetCurrent(0) = %+v, %v; want the built-in", cur, err)
	}
}

func testLessonProgress(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ls := s.Lessons()
	id, err := ls.Create(ctx, "", "Primes", "")
	if err != nil {
		t.Fatal(err)
	}
	s1, err := ls.AddStep(ctx, "", id, domain.LessonStep{Title: "Lire"})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := ls.AddStep(ctx, "", id, domain.LessonStep{Title: "Jouer"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ls.Get(ctx, "", id); err != nil {
		t.Fatal(err)
	}
	if done, err := ls.DoneSteps(ctx, "", id); err != nil || len(done) != 0 {
		t.Fatalf("DoneSteps after reading the Lesson = %v, %v; want none (ADR-0007)", done, err)
	}
	if err := ls.SetStepDone(ctx, "", s1, true); err != nil {
		t.Fatal(err)
	}
	first, err := ls.DoneSteps(ctx, "", id)
	if err != nil || len(first) != 1 || first[s1] == "" {
		t.Fatalf("DoneSteps after the gesture = %v, %v; want step %d dated", first, err, s1)
	}
	if err := ls.SetStepDone(ctx, "", s1, true); err != nil {
		t.Fatal(err)
	}
	if again, _ := ls.DoneSteps(ctx, "", id); again[s1] != first[s1] {
		t.Errorf("a second gesture moved the date from %q to %q", first[s1], again[s1])
	}
	if err := ls.SetStepDone(ctx, "", s1, false); err != nil {
		t.Fatal(err)
	}
	if err := ls.SetStepDone(ctx, "", s2, true); err != nil {
		t.Fatal(err)
	}
	if done, _ := ls.DoneSteps(ctx, "", id); len(done) != 1 || done[s2] == "" {
		t.Errorf("DoneSteps after withdrawing %d and marking %d = %v", s1, s2, done)
	}
	if err := ls.RemoveStep(ctx, "", s2); err != nil {
		t.Fatal(err)
	}
	if done, _ := ls.DoneSteps(ctx, "", id); len(done) != 0 {
		t.Errorf("DoneSteps after removing the done Step = %v; want none", done)
	}
	if err := ls.SetStepDone(ctx, "", s2, true); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetStepDone(removed step) = %v, want ErrNotFound", err)
	}
	if _, err := ls.DoneSteps(ctx, "", id+1000); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("DoneSteps(unknown lesson) = %v, want ErrNotFound", err)
	}
}

// checkSchema231Isolation: another tenant neither sees nor makes current this
// tenant's tables, marks its Steps done.
func checkSchema231Isolation(t *testing.T, ctx context.Context, s storage.Storage, a, b string) {
	mt := s.MatchEquityTables()
	id, err := mt.Save(ctx, a, domain.MatchEquityTable{Name: "T", Digest: "same", Source: "<met/>"})
	if err != nil {
		t.Fatal(err)
	}
	if list, err := mt.List(ctx, b); err != nil || len(list) != 0 {
		t.Errorf("List(%s) = %v, %v; want none", b, list, err)
	}
	if err := mt.SetCurrent(ctx, b, id); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetCurrent(%s) on %s's table = %v, want ErrNotFound", b, a, err)
	}
	own, err := mt.Save(ctx, b, domain.MatchEquityTable{Name: "T", Digest: "same", Source: "<met/>"})
	if err != nil || own == id {
		t.Errorf("Save(%s) of the same digest = %d, %v; want a row of its own", b, own, err)
	}
	if err := mt.SetCurrent(ctx, a, id); err != nil {
		t.Fatal(err)
	}
	if cur, err := mt.Current(ctx, b); err != nil || cur != nil {
		t.Errorf("Current(%s) = %+v, %v; want the built-in", b, cur, err)
	}

	ls := s.Lessons()
	lessonID, err := ls.Create(ctx, a, "Primes", "")
	if err != nil {
		t.Fatal(err)
	}
	stepID, err := ls.AddStep(ctx, a, lessonID, domain.LessonStep{Title: "Lire"})
	if err != nil {
		t.Fatal(err)
	}
	if err := ls.SetStepDone(ctx, b, stepID, true); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("SetStepDone(%s) on %s's step = %v, want ErrNotFound", b, a, err)
	}
	if _, err := ls.DoneSteps(ctx, b, lessonID); !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("DoneSteps(%s) on %s's lesson = %v, want ErrNotFound", b, a, err)
	}
}
