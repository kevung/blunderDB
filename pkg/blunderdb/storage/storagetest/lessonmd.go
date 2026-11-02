package storagetest

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/lessonmd"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// CheckLessonMarkdown: a Lesson written as Markdown and imported again is the
// same Lesson, an article lands as a Lesson without Steps, and a bad file
// writes nothing.
func CheckLessonMarkdown(t *testing.T, s storage.Storage, ctx context.Context, scope string) {
	t.Helper()
	lid, err := s.Lessons().Create(ctx, scope, "Bear-off", "Basics\n\n# not a title")
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range [][2]string{{"Look", "first\n\n## not a step"}, {"Words", "second"}} {
		if _, err := s.Lessons().AddStep(ctx, scope, lid, domainStep(st[0], st[1])); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := s.Lessons().List(ctx, scope)

	md, err := lessonmd.Export(ctx, s, scope, lid)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := lessonmd.Import(ctx, s, scope, md)
	if err != nil || len(ids) != 1 {
		t.Fatalf("import = %v, %v", ids, err)
	}
	a, _ := s.Lessons().Get(ctx, scope, lid)
	b, err := s.Lessons().Get(ctx, scope, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != b.Name || a.Description != b.Description || len(a.Steps) != 2 || len(b.Steps) != 2 {
		t.Fatalf("imported %+v, want %+v", b, a)
	}
	for i := range a.Steps {
		if a.Steps[i].Title != b.Steps[i].Title || a.Steps[i].Text != b.Steps[i].Text {
			t.Errorf("step %d = %+v, want %+v", i, b.Steps[i], a.Steps[i])
		}
	}

	art, err := lessonmd.Import(ctx, s, scope, "# An article\n\nJust text.\n\n# Another\n\nMore.\n")
	if err != nil || len(art) != 2 {
		t.Fatalf("articles = %v, %v", art, err)
	}
	if l, _ := s.Lessons().Get(ctx, scope, art[0]); l.Name != "An article" || l.Description != "Just text." || len(l.Steps) != 0 {
		t.Errorf("article = %+v", l)
	}

	n, _ := s.Lessons().List(ctx, scope)
	if _, err := lessonmd.Import(ctx, s, scope, "no title here\n"); err == nil {
		t.Error("a file with no title was imported")
	}
	if after, _ := s.Lessons().List(ctx, scope); len(after) != len(n) || len(n) != len(before)+3 {
		t.Errorf("lessons: %d before, %d after the import, %d after the refusal", len(before), len(n), len(after))
	}
}

// CheckLessonImportAtomic: an import cut in the middle (here the context
// cancelled by the second Step) leaves no Lesson behind, even though the
// cancelled context could not run a compensating delete.
func CheckLessonImportAtomic(t *testing.T, s storage.Storage, scope string, ctxOf func(context.Context) context.Context) {
	t.Helper()
	base, cancel := context.WithCancel(ctxOf(context.Background()))
	defer cancel()
	before, _ := s.Lessons().List(ctxOf(context.Background()), scope)
	cut := &cutAfterSteps{Storage: s, left: 2, cancel: cancel}
	_, err := lessonmd.Import(base, cut, scope, "# One\n\n## a\n\nx\n\n## b\n\ny\n\n# Two\n\n## c\n\nz\n")
	if err == nil {
		t.Fatal("an import cut in the middle reported success")
	}
	if after, _ := s.Lessons().List(ctxOf(context.Background()), scope); len(after) != len(before) {
		t.Errorf("%d lessons before the cut import, %d after: partial lessons remain", len(before), len(after))
	}
}
