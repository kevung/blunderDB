package document

import (
	"slices"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/structure"
)

func reviewReleve(t *testing.T) *Releve {
	t.Helper()
	pos := "XGID=-b----E-C---eE---c-e----B-:" + StructureDefaults.Context
	cube := capture.Doubt{Field: "cube", Reason: "drawn"}
	r := &Releve{Format: Format, Version: Version, Document: Source{Title: "T", Defaults: structureDefaults(structure.Profile{DirectionVoted: true})}, Entries: []Entry{
		{Rank: 1, Page: 1, Position: pos, Checkers: Sure, Context: ContextDefault, Doubts: []capture.Doubt{ContextDoubt}},
		{Rank: 2, Page: 3, Label: "Problem 4", Position: pos, Checkers: Sure, Context: ContextToVerify, Doubts: []capture.Doubt{ContextDoubt, cube}},
		{Rank: 3, Page: 4, Position: pos, Checkers: ToVerify, Context: ContextToVerify, Doubts: []capture.Doubt{ContextDoubt, cube}},
	}}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	return r
}

// Refused is the list Import applies: the ranks it names are the ones the
// import refuses, and an import with those options passes when it is empty.
func TestRefusedMatchesImport(t *testing.T) {
	r := reviewReleve(t)
	if got := Refused(r, ImportOptions{}); !slices.Equal(got, []int{1, 2, 3}) {
		t.Fatalf("unconfirmed defaults: %v", got)
	}
	if err := ConfirmDefaults(r); err != nil {
		t.Fatal(err)
	}
	if got := Refused(r, ImportOptions{}); !slices.Equal(got, []int{2, 3}) {
		t.Fatalf("confirmed defaults: %v", got)
	}
	if got := Refused(r, ImportOptions{Fallback: true}); !slices.Equal(got, []int{3}) {
		t.Fatalf("fallback: %v", got)
	}
	if got := Refused(r, ImportOptions{Fallback: true, AcceptComplete: true}); !slices.Equal(got, []int{3}) {
		t.Fatalf("fallback and complete: %v", got)
	}
}

func TestReviewItemsAndFallback(t *testing.T) {
	r := reviewReleve(t)
	res := Review(r, ImportOptions{})
	if len(res.Items) != 3 || len(res.FallbackRanks) != 0 {
		t.Fatalf("unconfirmed defaults offer no fallback: %+v", res)
	}
	if err := ConfirmDefaults(r); err != nil {
		t.Fatal(err)
	}
	res = Review(r, ImportOptions{Fallback: true})
	if !slices.Equal(res.FallbackRanks, []int{2}) || !slices.Equal(res.Refused, []int{3}) {
		t.Fatalf("confirmed: %+v", res)
	}
	var it ReviewItem
	for _, x := range res.Items {
		if x.Rank == 2 {
			it = x
		}
	}
	if !it.Fallback || it.Held || it.Reference != "p. 3, Problem 4" || !slices.Equal(it.Fields, []string{"cube"}) || !strings.Contains(it.Note, "cube (drawn)") {
		t.Fatalf("item: %+v", it)
	}
}

func TestConfirmDefaultsConfirmsDirectionToo(t *testing.T) {
	r := reviewReleve(t)
	r.Document.Defaults.Direction = assumedDirection("right")
	if err := ConfirmDefaults(r); err != nil {
		t.Fatal(err)
	}
	if !r.Document.Defaults.Confirmed || !r.Document.Defaults.Direction.Confirmed {
		t.Errorf("defaults %+v direction %+v", r.Document.Defaults, r.Document.Defaults.Direction)
	}
	r.Document.Defaults = nil
	if err := ConfirmDefaults(r); err == nil {
		t.Errorf("confirming absent defaults: no error")
	}
}
