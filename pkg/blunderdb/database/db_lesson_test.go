package database

import (
	"path/filepath"
	"testing"
)

// A Lesson exported with its contents arrives at the recipient with its steps
// in order, pointing at the recipient's own copies of the collection and the
// position; importing the same file again creates nothing (ADR-0066).
func TestLessonTravelsThroughExportAndImport(t *testing.T) {
	isolateIdentity(t)
	coach := newTestDB(t)
	first, err := coach.SavePosition(ptr(initialPosition()))
	if err != nil {
		t.Fatal(err)
	}
	second, err := coach.SavePosition(ptr(bearoffPosition()))
	if err != nil {
		t.Fatal(err)
	}
	colID, err := coach.CreateCollection("Course", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := coach.AddPositionToCollection(colID, first); err != nil {
		t.Fatal(err)
	}
	lessonID, err := coach.CreateLesson("Débuter", "Trois étapes")
	if err != nil {
		t.Fatal(err)
	}
	for _, st := range []struct {
		title     string
		coll, pos int64
	}{{"Lire", 0, 0}, {"Courir", colID, 0}, {"Sortir", 0, second}} {
		if _, err := coach.AddLessonStep(lessonID, st.title, "texte "+st.title, st.coll, st.pos); err != nil {
			t.Fatal(err)
		}
	}
	// A Lesson not asked for stays behind.
	if _, err := coach.CreateLesson("Privée", ""); err != nil {
		t.Fatal(err)
	}

	path := exportTo(t, coach, filepath.Join(t.TempDir(), "lecon.db"), ExportOptions{
		IncludeLessons: true, LessonIDs: []int64{lessonID},
		IncludeAnalysis: true, IncludeComments: true,
		Watermark: "Cours de Jean Dupont",
	})

	student := newTestDB(t)
	for round := 0; round < 2; round++ {
		if _, err := student.CommitImportDatabase(path); err != nil {
			t.Fatalf("CommitImportDatabase (round %d): %v", round, err)
		}
	}
	lessons, err := student.ListLessons()
	if err != nil || len(lessons) != 1 || lessons[0].Name != "Débuter" || lessons[0].StepCount != 3 {
		t.Fatalf("student's lessons = %+v, %v; want Débuter with 3 steps, once", lessons, err)
	}
	l, err := student.GetLesson(lessons[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if l.Steps[0].Title != "Lire" || l.Steps[1].Title != "Courir" || l.Steps[2].Title != "Sortir" {
		t.Fatalf("step order = %+v", l.Steps)
	}
	if l.Steps[0].CollectionID != 0 || l.Steps[0].PositionID != 0 {
		t.Fatalf("a text step must stay a text step: %+v", l.Steps[0])
	}
	col, err := student.GetCollectionPositions(l.Steps[1].CollectionID)
	if err != nil || len(col) != 1 {
		t.Fatalf("the step's collection at the student = %d members, %v; want 1", len(col), err)
	}
	pos, err := student.LoadPosition(int(l.Steps[2].PositionID))
	if err != nil || pos == nil {
		t.Fatalf("the step's position at the student: %v", err)
	}
}

func ptr[T any](v T) *T { return &v }

// Deleting a match keeps a position a Lesson step shows: the step is the
// coach's work on it (positionIsHeldSQL).
func TestLessonStepHoldsItsPosition(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	matchID := importTestMatch(t, db)
	ids := getPositionIDs(t, db, 1)
	lessonID, err := db.CreateLesson("L", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.AddLessonStep(lessonID, "", "", 0, ids[0]); err != nil {
		t.Fatal(err)
	}
	if err := db.DeleteMatch(matchID); err != nil {
		t.Fatal(err)
	}
	if pos, err := db.LoadPosition(int(ids[0])); err != nil || pos == nil {
		t.Fatalf("the step's position after the match is deleted: %v", err)
	}
	l, err := db.GetLesson(lessonID)
	if err != nil || l.Steps[0].PositionID != ids[0] {
		t.Fatalf("lesson after delete = %+v, %v", l, err)
	}
}
