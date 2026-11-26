package storagetest

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/document"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// Invented positions: the opening, and the opening after 31 played 8/5 6/5.
const (
	docOpening = "XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:7:10"
	docAfter31 = "XGID=-b---BD-B---eE---c-e----B-:0:0:-1:52:0:0:0:7:10"
	docCube    = "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:2:4:0:7:10"
)

func docReleve() *document.Releve {
	return &document.Releve{
		Format:   document.Format,
		Version:  document.Version,
		Document: document.Source{Title: "Exemples inventés", Author: "Personne", Edition: "1re édition"},
		Entries: []document.Entry{
			{Rank: 1, Page: 3, Label: "Problème 1", Position: docAfter31, Checkers: document.Sure, Context: document.ContextRead},
			{Rank: 2, Page: 3, Label: "Problème 2", Position: docOpening, Checkers: document.Sure, Context: document.ContextRead},
			{Rank: 3, Page: 7, Label: "Problème 3", Position: docCube, Checkers: document.ToVerify, Context: document.ContextRead, Verified: true},
			{Rank: 4, Page: 9, Label: "Problème 4", Position: docAfter31, Checkers: document.Sure, Context: document.ContextRead},
		},
	}
}

// CheckDocumentImport: a Relevé lands in one Collection in the order of the
// document, a repeated position once with both references, the document in
// the Collection's description; importing it again writes nothing new; an
// unverified entry refuses the import before anything is written.
func CheckDocumentImport(t *testing.T, s storage.Storage, ctx context.Context, scope string) {
	t.Helper()
	r := docReleve()
	res, err := document.Import(ctx, s, scope, r, document.ImportOptions{})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if !res.CreatedCollection || res.Positions != 3 || res.Added != 3 || res.Repeated != 1 || res.NewPositions != 3 {
		t.Fatalf("first import = %+v", res)
	}
	col, err := s.Collections().Get(ctx, scope, res.CollectionID)
	if err != nil {
		t.Fatal(err)
	}
	if col.Name != "Exemples inventés" || col.Description != "Exemples inventés\nPersonne\n1re édition" {
		t.Errorf("collection = %q / %q", col.Name, col.Description)
	}
	ids, err := s.Collections().PositionIDs(ctx, scope, res.CollectionID, storage.ListOpts{})
	if err != nil || len(ids) != 3 {
		t.Fatalf("members = %v, %v", ids, err)
	}
	for i, want := range []string{docAfter31, docOpening, docCube} {
		p, err := s.Positions().Load(ctx, scope, ids[i])
		if err != nil {
			t.Fatal(err)
		}
		wantPos, _ := document.ParsePosition(want)
		if engine.ZobristHash(p) != engine.ZobristHash(&wantPos) {
			t.Errorf("member %d is not %s", i, want)
		}
		if !p.IndividuallyImported {
			t.Errorf("member %d is not marked as brought in on its own", i)
		}
	}
	text, err := s.Comments().Text(ctx, scope, ids[0])
	if err != nil || !strings.Contains(text, "Exemples inventés — p. 3, Problème 1; p. 9, Problème 4") {
		t.Errorf("comment of the repeated position = %q, %v", text, err)
	}
	if text, _ := s.Comments().Text(ctx, scope, ids[2]); !strings.Contains(text, "p. 7, Problème 3") {
		t.Errorf("comment = %q", text)
	}

	countAll := func() (int, int, int) {
		n, _ := s.Positions().Count(ctx, scope)
		var cols, comments int
		for range s.Collections().List(ctx, scope) {
			cols++
		}
		for range s.Comments().ListAll(ctx, scope, storage.ListOpts{}) {
			comments++
		}
		return n, cols, comments
	}
	p0, c0, m0 := countAll()

	again, err := document.Import(ctx, s, scope, docReleve(), document.ImportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if again.CreatedCollection || again.Added != 0 || again.NewPositions != 0 || again.CollectionID != res.CollectionID {
		t.Errorf("second import = %+v", again)
	}
	if p, c, m := countAll(); p != p0 || c != c0 || m != m0 {
		t.Errorf("a second import wrote: positions %d→%d, collections %d→%d, comments %d→%d", p0, p, c0, c, m0, m)
	}

	bad := docReleve()
	bad.Document.Title = "Autre document"
	bad.Entries = append(bad.Entries, document.Entry{Rank: 5, Page: 10, Position: "XGID=-b----E-C---eE---c-e----B-:0:0:1:62:0:0:0:7:10", Checkers: document.Uncertain, Context: document.ContextRead})
	if _, err := document.Import(ctx, s, scope, bad, document.ImportOptions{}); !errors.Is(err, document.ErrUnverified) || !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("unverified entry: err = %v", err)
	}
	if p, c, m := countAll(); p != p0 || c != c0 || m != m0 {
		t.Errorf("a refused import wrote: positions %d→%d, collections %d→%d, comments %d→%d", p0, p, c0, c, m0, m)
	}
	forced, err := document.Import(ctx, s, scope, bad, document.ImportOptions{Force: true, Collection: "Forcé"})
	if err != nil || forced.Forced != 1 || forced.Collection != "Forcé" || forced.Positions != 4 {
		t.Errorf("forced import = %+v, %v", forced, err)
	}

	// Fifteen checkers a side read with no doubt, under the document's
	// defaults: refused while the defaults are not confirmed, even when
	// complete entries are accepted; accepted as a whole once they are.
	complete := &document.Releve{Format: document.Format, Version: document.Version,
		Document: document.Source{Title: "Relevé d'un scan", Defaults: &document.Defaults{Context: document.StructureDefaults.Context}},
		Entries: []document.Entry{{Rank: 1, Page: 1, Position: "XGID=-b----E-C---eE---c-e----B-:" + document.StructureDefaults.Context,
			Checkers: document.ToVerify, Complete: true, Context: document.ContextDefault, Doubts: []capture.Doubt{document.ContextDoubt}}}}
	if _, err := document.Import(ctx, s, scope, complete, document.ImportOptions{AcceptComplete: true}); !errors.Is(err, document.ErrUnverified) {
		t.Errorf("complete entry under unconfirmed defaults: err = %v", err)
	}
	complete.Document.Defaults.Confirmed = true
	if _, err := document.Import(ctx, s, scope, complete, document.ImportOptions{}); !errors.Is(err, document.ErrUnverified) {
		t.Errorf("complete entry without AcceptComplete: err = %v", err)
	}
	if res, err := document.Import(ctx, s, scope, complete, document.ImportOptions{AcceptComplete: true}); err != nil || res.Complete != 1 || res.Forced != 0 || res.Positions != 1 {
		t.Errorf("complete import = %+v, %v", res, err)
	}

	// Checkers sure, a cube drawn and not read: the import proposes the
	// fallback rather than a bare refusal, and with it imports the position
	// as it carries its context, its reference naming the field to check.
	cube := capture.Doubt{Field: "cube", Reason: "cube drawn, value not read"}
	repli := &document.Releve{Format: document.Format, Version: document.Version,
		Document: document.Source{Title: "Relevé à replier", Defaults: &document.Defaults{Context: document.StructureDefaults.Context, Confirmed: true}},
		Entries: []document.Entry{{Rank: 1, Page: 2, Position: "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:2:4:0:7:10",
			Checkers: document.Sure, Context: document.ContextToVerify, Doubts: []capture.Doubt{document.ContextDoubt, cube}}}}
	if _, err := document.Import(ctx, s, scope, repli, document.ImportOptions{}); !errors.Is(err, document.ErrUnverified) || !strings.Contains(err.Error(), "fallback") {
		t.Errorf("context to verify without fallback: err = %v", err)
	}
	fb, err := document.Import(ctx, s, scope, repli, document.ImportOptions{Fallback: true})
	if err != nil || fb.Fallback != 1 || fb.Forced != 0 || fb.Positions != 1 {
		t.Fatalf("fallback import = %+v, %v", fb, err)
	}
	fids, err := s.Collections().PositionIDs(ctx, scope, fb.CollectionID, storage.ListOpts{})
	if err != nil || len(fids) != 1 {
		t.Fatalf("fallback members = %v, %v", fids, err)
	}
	if text, _ := s.Comments().Text(ctx, scope, fids[0]); !strings.Contains(text, "p. 2 (context not read, to check against the diagram: cube (cube drawn, value not read))") {
		t.Errorf("fallback comment = %q", text)
	}
}

// CheckDocumentImportAtomic: an import that fails after its positions were
// saved leaves neither position, Collection nor comment behind.
func CheckDocumentImportAtomic(t *testing.T, s storage.Storage, ctx context.Context, scope string) {
	t.Helper()
	n0, _ := s.Positions().Count(ctx, scope)
	var c0 int
	for range s.Collections().List(ctx, scope) {
		c0++
	}
	r := docReleve()
	r.Document.Title = "Import interrompu"
	r.Entries[0].Position = "XGID=-b----E-C---eE---c-e----B-:0:0:1:11:0:0:0:7:10"
	_, err := document.Import(ctx, &failMembership{Storage: s}, scope, r, document.ImportOptions{})
	if err == nil {
		t.Fatal("an import whose membership write failed reported success")
	}
	n1, _ := s.Positions().Count(ctx, scope)
	var c1 int
	for range s.Collections().List(ctx, scope) {
		c1++
	}
	if n1 != n0 || c1 != c0 {
		t.Errorf("a failed import left positions %d→%d, collections %d→%d", n0, n1, c0, c1)
	}
}

// failMembership is a Storage whose transactions refuse to add positions to a
// Collection, the last write an import makes before its comments.
type failMembership struct{ storage.Storage }

func (f *failMembership) BeginTx(ctx context.Context) (storage.Tx, error) {
	tx, err := f.Storage.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	return &failMembershipTx{Tx: tx}, nil
}

type failMembershipTx struct{ storage.Tx }

func (t *failMembershipTx) Collections() storage.CollectionStore {
	return &failAddPositions{CollectionStore: t.Tx.Collections()}
}

type failAddPositions struct{ storage.CollectionStore }

func (failAddPositions) AddPositions(context.Context, string, int64, []int64) error {
	return errors.New("membership refused")
}
