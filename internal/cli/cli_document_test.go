package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if !strings.Contains(out, "Unverified: rank 2") {
		t.Errorf("check of an unverified entry:\n%s", out)
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
