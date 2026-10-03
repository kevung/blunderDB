package storage

import (
	"math/rand/v2"
	"reflect"
	"testing"
)

func studyFixture() *RecurringErrors {
	return &RecurringErrors{Groups: []RecurringErrorGroup{
		{Theme: "a", PositionIDs: []int64{1, 2}},
		{Theme: "b", PositionIDs: []int64{2, 3}},
		{Theme: "c", PositionIDs: []int64{4}},
		{Theme: "d", PositionIDs: []int64{5}},
	}}
}

func TestStudyPositionIDs_WorstThreeAreDeduplicated(t *testing.T) {
	got := studyFixture().StudyPositionIDs(0)
	if want := []int64{1, 2, 3, 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("worst groups = %v, want %v", got, want)
	}
}

func TestStudyPositionIDs_OneGroupByRank(t *testing.T) {
	r := studyFixture()
	if got, want := r.StudyPositionIDs(4), []int64{5}; !reflect.DeepEqual(got, want) {
		t.Fatalf("rank 4 = %v, want %v", got, want)
	}
	if got := r.StudyPositionIDs(5); len(got) != 0 {
		t.Fatalf("rank past the end = %v, want nothing", got)
	}
	var none *RecurringErrors
	if got := none.StudyPositionIDs(0); got != nil {
		t.Fatalf("nil result = %v", got)
	}
}

func TestDrawStudyQuiz_BoundedDistinctAndReproducible(t *testing.T) {
	ids := []int64{1, 2, 3, 4, 5, 6, 7, 8}
	a := DrawStudyQuiz(ids, 3, rand.New(rand.NewPCG(1, 2)))
	b := DrawStudyQuiz(ids, 3, rand.New(rand.NewPCG(1, 2)))
	if len(a) != 3 || !reflect.DeepEqual(a, b) {
		t.Fatalf("draws = %v and %v", a, b)
	}
	seen := map[int64]bool{}
	for _, id := range a {
		if seen[id] {
			t.Fatalf("duplicate %d in %v", id, a)
		}
		seen[id] = true
	}
	if got := DrawStudyQuiz(ids, 20, nil); len(got) != len(ids) {
		t.Fatalf("fewer ids than the size: got %d, want %d", len(got), len(ids))
	}
	if !reflect.DeepEqual(ids, []int64{1, 2, 3, 4, 5, 6, 7, 8}) {
		t.Fatal("the input was shuffled in place")
	}
}
