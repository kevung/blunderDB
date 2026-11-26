package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
	"github.com/kevung/blunderdb/pkg/blunderdb/document"
)

const exampleReleve = "pkg/blunderdb/document/testdata/releve-exemple.json"

// document check validates without a database; document import fills one
// Collection in the document's order and a second import adds nothing.
func TestCLI_DocumentCheckAndImport(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	out := captureOut(t, cli, func() {
		if err := cli.Run([]string{"document", "check", exampleReleve}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "3 distinct, 1 repeated") || !strings.Contains(out, "Unverified: none") {
		t.Errorf("check:\n%s", out)
	}

	type result struct {
		CollectionID      int64 `json:"collectionId"`
		CreatedCollection bool  `json:"createdCollection"`
		Added             int   `json:"added"`
	}
	imp := func(args ...string) (result, error) {
		var r result
		var err error
		out := captureOut(t, cli, func() {
			err = cli.Run(append([]string{"document", "import", exampleReleve, "--db", dbPath, "--format", "json"}, args...))
		})
		if err == nil {
			if jerr := json.Unmarshal([]byte(out), &r); jerr != nil {
				t.Fatalf("%v in %s", jerr, out)
			}
		}
		return r, err
	}
	first, err := imp()
	if err != nil || !first.CreatedCollection || first.Added != 3 {
		t.Fatalf("first import = %+v, %v", first, err)
	}
	second, err := imp()
	if err != nil || second.CreatedCollection || second.Added != 0 || second.CollectionID != first.CollectionID {
		t.Errorf("second import = %+v, %v", second, err)
	}
	cols, err := cli.db.GetAllCollections()
	if err != nil || len(cols) != 1 || cols[0].Name != "Exemples inventés" || cols[0].PositionCount != 3 {
		t.Errorf("collections = %+v, %v", cols, err)
	}

	data, _ := os.ReadFile(exampleReleve)
	unverified := filepath.Join(t.TempDir(), "unverified.json")
	edited := strings.Replace(string(data), `"verified": true`, `"verified": false`, 1)
	if err := os.WriteFile(unverified, []byte(edited), 0o600); err != nil {
		t.Fatal(err)
	}
	out = captureOut(t, cli, func() { _ = cli.Run([]string{"document", "check", unverified}) })
	if !strings.Contains(out, "Unverified: rank 2") || !strings.Contains(out, "Review:\n  rank 2, p. ") {
		t.Errorf("check of an unverified entry:\n%s", out)
	}
	out = captureOut(t, cli, func() { _ = cli.Run([]string{"document", "check", "--format", "json", unverified}) })
	if !strings.Contains(out, `"review": [`) || !strings.Contains(out, `"rank": 2`) || !strings.Contains(out, `"held": true`) {
		t.Errorf("json check lists no review:\n%s", out)
	}
	if err := cli.Run([]string{"document", "import", "--db", dbPath, "--collection", "Autre", unverified}); err == nil {
		t.Error("an unverified entry was imported without --force")
	}
	if cols, _ := cli.db.GetAllCollections(); len(cols) != 1 {
		t.Errorf("a refused import wrote a collection: %+v", cols)
	}
	if err := cli.Run([]string{"document", "import", "--db", dbPath, "--collection", "Autre", "--force", unverified}); err != nil {
		t.Errorf("forced import: %v", err)
	}
}

// An entry a scan's reader confirmed by its checkers is counted apart and
// refuses the import until a person verifies its context.
func TestCLI_DocumentContextNotRead(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	r, err := document.Decode(mustRead(t, exampleReleve))
	if err != nil {
		t.Fatal(err)
	}
	r.Entries[0].Context, r.Entries[0].Doubts = document.ContextToVerify, []capture.Doubt{document.ContextDoubt}
	if !r.CheckersConfirmed(r.Entries[0]) {
		t.Fatal("the entry is not taken as checkers confirmed")
	}
	path := filepath.Join(t.TempDir(), "scan.json")
	write := func() {
		data, err := r.Encode()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	out := captureOut(t, cli, func() { _ = cli.Run([]string{"document", "check", path}) })
	if !strings.Contains(out, "sure 2, to verify 2") || !strings.Contains(out, "Checkers:   1 confirmed") || !strings.Contains(out, "Unverified: rank 1") {
		t.Errorf("check:\n%s", out)
	}
	err = cli.Run([]string{"document", "import", "--db", dbPath, path})
	if !errors.Is(err, document.ErrUnverified) || !strings.Contains(err.Error(), "cube, dice, player on roll and score") || !strings.Contains(err.Error(), `"verified": true`) {
		t.Errorf("import of an unread context: %v", err)
	}
	if cols, _ := cli.db.GetAllCollections(); len(cols) != 0 {
		t.Errorf("a refused import wrote a collection: %+v", cols)
	}
	r.Entries[0].Verified = true
	write()
	if err := cli.Run([]string{"document", "import", "--db", dbPath, path}); err != nil {
		t.Errorf("import once verified: %v", err)
	}
}

// Under the document's defaults, nothing is sure until they are confirmed;
// then an entry of 15 + 15 checkers read with no doubt is imported with
// --accept-complete, and only with it.
func TestCLI_DocumentDefaultsAndComplete(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	pos := "XGID=-b----E-C---eE---c-e----B-:" + document.StructureDefaults.Context
	d := document.StructureDefaults
	r := &document.Releve{Format: document.Format, Version: document.Version, Document: document.Source{Title: "Scan", Defaults: &d},
		Entries: []document.Entry{
			{Rank: 1, Page: 1, Position: pos, Checkers: document.Sure, Context: document.ContextDefault, Doubts: []capture.Doubt{document.ContextDoubt}},
			{Rank: 2, Page: 2, Position: "XGID=-b---BD-B---eE---c-e----B-:" + document.StructureDefaults.Context, Checkers: document.ToVerify, Complete: true,
				Context: document.ContextDefault, Doubts: []capture.Doubt{document.ContextDoubt}},
		}}
	path := filepath.Join(t.TempDir(), "scan.json")
	write := func() {
		data, err := r.Encode()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	out := captureOut(t, cli, func() { _ = cli.Run([]string{"document", "check", path}) })
	if !strings.Contains(out, "sure 0, to verify 2") || !strings.Contains(out, "not confirmed") || !strings.Contains(out, "Unverified: rank 1, 2 (import refused without --force)") {
		t.Errorf("check under unconfirmed defaults:\n%s", out)
	}
	if err := cli.Run([]string{"document", "import", "--db", dbPath, "--accept-complete", path}); !errors.Is(err, document.ErrUnverified) || !strings.Contains(err.Error(), "confirm the document's defaults") {
		t.Errorf("import under unconfirmed defaults: %v", err)
	}
	d.Confirmed = true
	write()
	out = captureOut(t, cli, func() { _ = cli.Run([]string{"document", "check", path}) })
	if !strings.Contains(out, "sure 1, to verify 1") || !strings.Contains(out, "Complete:   1") || !strings.Contains(out, "Unverified: rank 2 (import refused without --accept-complete or --force)") {
		t.Errorf("check under confirmed defaults:\n%s", out)
	}
	if err := cli.Run([]string{"document", "import", "--db", dbPath, path}); !errors.Is(err, document.ErrUnverified) || !strings.Contains(err.Error(), "accept complete") {
		t.Errorf("import of a complete entry without --accept-complete: %v", err)
	}
	out = captureOut(t, cli, func() {
		if err := cli.Run([]string{"document", "import", "--db", dbPath, "--accept-complete", path}); err != nil {
			t.Errorf("import with --accept-complete: %v", err)
		}
	})
	if !strings.Contains(out, "Complete: 1 entries") {
		t.Errorf("import with --accept-complete:\n%s", out)
	}
}

// A match book's defaults learned its score as printed: an entry under
// them carries its own score and one confirmation certifies it; an entry
// whose score is not read is proposed the fallback, imported with
// --fallback.
func TestCLI_DocumentLearnedDefaultsAndFallback(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	d := document.StructureDefaults
	d.Printed = []string{"score"}
	r := &document.Releve{Format: document.Format, Version: document.Version, Document: document.Source{Title: "Match book", Defaults: &d},
		Entries: []document.Entry{
			{Rank: 1, Page: 1, Position: "XGID=-b----E-C---eE---c-e----B-:0:0:1:00:2:4:0:7:10", Checkers: document.Sure, Context: document.ContextDefault, Doubts: []capture.Doubt{document.ContextDoubt}},
			{Rank: 2, Page: 2, Position: "XGID=-b---BD-B---eE---c-e----B-:" + document.StructureDefaults.Context, Checkers: document.Sure,
				Context: document.ContextToVerify, Doubts: []capture.Doubt{document.ContextDoubt, document.MissingScoreDoubt}},
			{Rank: 3, Page: 3, Position: "XGID=-b---BC-C---eE---c-e----B-:" + document.StructureDefaults.Context, Checkers: document.ToVerify, Complete: true,
				Context: document.ContextToVerify, Doubts: []capture.Doubt{document.ContextDoubt, document.NoPipsDoubt, document.MissingScoreDoubt}},
		}}
	path := filepath.Join(t.TempDir(), "match.json")
	write := func() {
		data, err := r.Encode()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	d.Confirmed = true
	write()
	out := captureOut(t, cli, func() { _ = cli.Run([]string{"document", "check", path}) })
	if !strings.Contains(out, "sure 1, to verify 2") || !strings.Contains(out, "Printed:    score") ||
		!strings.Contains(out, "Fallback:   rank 2 imported with --fallback under the context its position carries (rank 2, 3 with --fallback --accept-complete)") {
		t.Errorf("check:\n%s", out)
	}
	if err := cli.Run([]string{"document", "import", "--db", dbPath, path}); !errors.Is(err, document.ErrUnverified) || !strings.Contains(err.Error(), "(fallback)") {
		t.Errorf("import without --fallback: %v", err)
	}
	out = captureOut(t, cli, func() {
		if err := cli.Run([]string{"document", "import", "--db", dbPath, "--fallback", "--accept-complete", path}); err != nil {
			t.Errorf("import with --fallback --accept-complete: %v", err)
		}
	})
	if !strings.Contains(out, "Fallback: 2 entries") {
		t.Errorf("import with --fallback:\n%s", out)
	}
}

// From the page images of a book without printed pips to the database:
// analyze writes complete entries under unconfirmed defaults; once the
// person confirms them and verifies the diagram whose cube is drawn,
// --accept-complete imports the rest as a whole, and only with it.
func TestCLI_DocumentCompleteEndToEnd(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	path := filepath.Join(t.TempDir(), "releve.json")
	out := captureOut(t, cli, func() {
		if err := cli.Run([]string{"document", "analyze", "pkg/blunderdb/document/testdata/sans-pips", "-o", path, "--title", "Sans pips"}); err != nil {
			t.Errorf("analyze: %v", err)
		}
	})
	if !strings.Contains(out, "default 7, to verify 1") {
		t.Errorf("analyze:\n%s", out)
	}
	r, err := document.Decode(mustRead(t, path))
	if err != nil {
		t.Fatal(err)
	}
	write := func() {
		data, err := r.Encode()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if r.Document.Defaults.Direction == nil || r.Document.Defaults.Direction.Confirmed {
		t.Fatalf("no unconfirmed direction of play: %+v", r.Document.Defaults)
	}
	// The context confirmed, the direction assumed is not: refused.
	r.Document.Defaults.Confirmed = true
	r.Entries[1].Verified = true
	write()
	out = captureOut(t, cli, func() { _ = cli.Run([]string{"document", "check", path}) })
	if !strings.Contains(out, "Direction:  direction of play assumed") || !strings.Contains(out, "7 more held back until the direction of play is confirmed") {
		t.Errorf("check with the direction unconfirmed:\n%s", out)
	}
	err = cli.Run([]string{"document", "import", "--db", dbPath, "--accept-complete", path})
	if !errors.Is(err, document.ErrUnverified) || !strings.Contains(err.Error(), "confirm it once") {
		t.Errorf("import with the direction unconfirmed: %v", err)
	}
	if cols, _ := cli.db.GetAllCollections(); len(cols) != 0 {
		t.Errorf("a refused import wrote a collection: %+v", cols)
	}
	// The doubts stripped by hand, the document still says the direction
	// unconfirmed: check does not call the entries certified, import refuses.
	for i := range r.Entries {
		r.Entries[i].Doubts = slices.DeleteFunc(r.Entries[i].Doubts, func(d capture.Doubt) bool { return d == document.AssumedDirectionDoubt })
	}
	write()
	out = captureOut(t, cli, func() { _ = cli.Run([]string{"document", "check", path}) })
	if strings.Contains(out, "certified context (imported") || !strings.Contains(out, "7 more held back until the direction of play is confirmed") {
		t.Errorf("check with the doubts stripped, the direction unconfirmed:\n%s", out)
	}
	if err := cli.Run([]string{"document", "import", "--db", dbPath, "--accept-complete", path}); !errors.Is(err, document.ErrUnverified) {
		t.Errorf("import with the doubts stripped, the direction unconfirmed: %v", err)
	}
	r.Document.Defaults.Direction.Confirmed = true
	write()
	if err := cli.Run([]string{"document", "import", "--db", dbPath, path}); !errors.Is(err, document.ErrUnverified) || !strings.Contains(err.Error(), "accept complete") {
		t.Errorf("import without --accept-complete: %v", err)
	}
	out = captureOut(t, cli, func() {
		if err := cli.Run([]string{"document", "import", "--db", dbPath, "--accept-complete", path}); err != nil {
			t.Errorf("import with --accept-complete: %v", err)
		}
	})
	if !strings.Contains(out, "Complete: 7 entries") || strings.Contains(out, "Forced") {
		t.Errorf("import with --accept-complete:\n%s", out)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// document analyze reads a PDF's diagrams into a relevé that document check
// and document import take, and refuses a PDF it cannot read by its glyphs.
func TestCLI_DocumentAnalyze(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	releve := filepath.Join(t.TempDir(), "releve.json")
	out := captureOut(t, cli, func() {
		if err := cli.Run([]string{"document", "analyze", "pkg/blunderdb/document/testdata/xg-diagrammes.pdf", "-o", releve, "--title", "Invented"}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "Entries:    6 (sure 3, to verify 2, uncertain 1)") {
		t.Errorf("analyze:\n%s", out)
	}
	data, err := os.ReadFile(releve)
	if err != nil || !strings.Contains(string(data), `"title": "Invented"`) {
		t.Fatalf("relevé %v:\n%s", err, data)
	}
	if err := cli.Run([]string{"document", "import", releve, "--db", dbPath}); err == nil {
		t.Errorf("an unverified relevé was imported without --force")
	}
	if err := cli.Run([]string{"document", "analyze", exampleReleve, "-o", releve}); err == nil {
		t.Errorf("a JSON file was analysed as a PDF")
	}
}
