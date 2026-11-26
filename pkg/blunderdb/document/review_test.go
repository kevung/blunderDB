package document

import (
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/document/structure"
)

func TestReviewAndConfirmDefaults(t *testing.T) {
	pos := "XGID=-b----E-C---eE---c-e----B-:" + StructureDefaults.Context
	cube := capture.Doubt{Field: "cube", Reason: "drawn"}
	r := &Releve{Format: Format, Version: Version, Document: Source{Title: "T", Defaults: structureDefaults(structure.Profile{DirectionVoted: true})}, Entries: []Entry{
		{Rank: 1, Page: 1, Position: pos, Checkers: Sure, Context: ContextDefault, Doubts: []capture.Doubt{ContextDoubt}},
		{Rank: 2, Page: 3, Label: "Problem 4", Position: pos, Checkers: Sure, Context: ContextToVerify, Doubts: []capture.Doubt{ContextDoubt, cube}},
	}}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := Review(r); len(got) != 2 {
		t.Fatalf("unconfirmed defaults: %d items, want 2", len(got))
	}
	if err := ConfirmDefaults(r, true, false); err != nil {
		t.Fatal(err)
	}
	items := Review(r)
	if len(items) != 1 || items[0].Rank != 2 || items[0].Reference != "p. 3, Problem 4" || !strings.Contains(items[0].Note, "cube (drawn)") {
		t.Fatalf("after confirmation: %+v", items)
	}
	if err := ConfirmDefaults(r, false, true); err == nil {
		t.Errorf("confirming a direction the relevé does not carry: no error")
	}
	r.Document.Defaults = nil
	if err := ConfirmDefaults(r, true, false); err == nil {
		t.Errorf("confirming absent defaults: no error")
	}
}
