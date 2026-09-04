package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot walks up from the test's working directory to the module root, so
// the generator can be pointed at the real documentation without a chdir.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the working directory")
		}
		dir = parent
	}
}

// TestHelpBundlesAreCurrent is the reason the bundles can be committed at all:
// it re-renders them from doc/source and fails if the committed files differ,
// so documentation that moved without `make help` stops the build here instead
// of shipping an in-app help two releases behind (fiche H.6 measured exactly
// that: the whole of the 0.31.0 watermark work was missing from nine languages).
func TestHelpBundlesAreCurrent(t *testing.T) {
	stale, err := run(repoRoot(t), true)
	if err != nil {
		t.Fatalf("generating the help bundles: %v", err)
	}
	if len(stale) > 0 {
		t.Fatalf("help bundles out of date: %s\nrun: make help", strings.Join(stale, ", "))
	}
}

// TestEveryLanguageIsFullyTranslated is implied by the one above (a missing
// msgstr makes the generator error out rather than fall back to French), but it
// is worth its own failure message: it is the check that used to have no
// mechanical equivalent at all for the help.
func TestEveryLanguageIsFullyTranslated(t *testing.T) {
	root := repoRoot(t)
	g, err := newGenerator(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, lang := range languages {
		if _, err := g.bundle(lang); err != nil {
			t.Errorf("%s: %v", lang, err)
		}
	}
}

func TestInlineEscapesBeforeApplyingMarkup(t *testing.T) {
	g := &generator{refs: map[string]refTarget{}, cats: map[string]catalogue{}}
	cases := []struct{ in, want string }{
		// The help is injected with {@html}: a tag in the documentation must
		// arrive as text, and must not become one after the markup pass.
		{`<script>alert(1)</script>`, `&lt;script&gt;alert(1)&lt;/script&gt;`},
		{`<b onclick="x">`, `&lt;b onclick="x"&gt;`},
		{"a & b < c > d", "a &amp; b &lt; c &gt; d"},
		{"**gras** et *emphase*", "<strong>gras</strong> et <em>emphase</em>"},
		{"``E>x``", "<code>E&gt;x</code>"},
		{"`D` ou `D1`", "<code>D</code> ou <code>D1</code>"},
		// docutils' delimiter rules: emphasis may abut a hyphen on either side.
		{"die *TAB*-Taste", "die <em>TAB</em>-Taste"},
		// A reStructuredText backslash escape disappears...
		{`m'a,b,...\'`, `m'a,b,...'`},
		// ...including the escaped space Japanese uses to separate markup from
		// adjacent CJK text, where a real space would be wrong.
		{"検索パネル（\\ *CTRL-F*\\ ）", "検索パネル（<em>CTRL-F</em>）"},
	}
	for _, c := range cases {
		got, err := g.inline(c.in, translator{lang: sourceLang})
		if err != nil {
			t.Fatalf("inline(%q): %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("inline(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestInlineQuoteEscaping(t *testing.T) {
	// Quotes are not escaped: the renderer only ever emits text nodes and the
	// fixed tags it writes itself, never an attribute built from a source
	// string. If that ever changes, this test is where to notice.
	g := &generator{refs: map[string]refTarget{}, cats: map[string]catalogue{}}
	got, _ := g.inline(`il a dit "non"`, translator{lang: sourceLang})
	if strings.Contains(got, "<") || strings.Contains(got, ">") {
		t.Errorf("unexpected markup in %q", got)
	}
}

func TestTranslationIsNeverSilentlyFrench(t *testing.T) {
	g := &generator{refs: map[string]refTarget{}, cats: map[string]catalogue{}}
	tr := translator{lang: "de", cats: []catalogue{{"connu": "bekannt"}}}
	if _, err := g.trInline("connu", tr); err != nil {
		t.Fatalf("known string: %v", err)
	}
	if _, err := g.trInline("inconnu", tr); err == nil {
		t.Fatal("an untranslated string must be an error, not a French fallback")
	}
}

func TestParseCSVTable(t *testing.T) {
	body := []string{
		`:header: "Commande", "Action"`,
		`:widths: 10, 40`,
		``,
		`"new, ne, n", "Crée une base."`,
		`"m'a,b,...\'", "Motifs."`,
	}
	tbl, err := parseCSVTable(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.header) != 2 || tbl.header[0] != "Commande" {
		t.Fatalf("header = %#v", tbl.header)
	}
	if len(tbl.rows) != 2 {
		t.Fatalf("rows = %#v", tbl.rows)
	}
	// A comma inside a quoted cell is data, exactly as docutils reads it.
	if tbl.rows[0][0] != "new, ne, n" {
		t.Errorf("row 0 col 0 = %q", tbl.rows[0][0])
	}
	if tbl.rows[1][0] != `m'a,b,...\'` {
		t.Errorf("row 1 col 0 = %q", tbl.rows[1][0])
	}
}

func TestJSTemplateLiteralCannotBeBrokenOut(t *testing.T) {
	for _, in := range []string{"a`b", `a\b`, "a${b}c", "a\\`b"} {
		out := jsTemplateLiteral(in)
		if strings.Contains(strings.ReplaceAll(out, "\\`", ""), "`") {
			t.Errorf("jsTemplateLiteral(%q) = %q leaves a live backtick", in, out)
		}
		if strings.Contains(strings.ReplaceAll(out, `\${`, ""), "${") {
			t.Errorf("jsTemplateLiteral(%q) = %q leaves a live substitution", in, out)
		}
	}
}

func TestLoadCatalogueJoinsContinuationLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.po")
	po := "msgid \"\"\nmsgstr \"Project-Id-Version: x\\n\"\n\n" +
		"#: ../../source/x.rst:1\nmsgid \"\"\n\"une phrase \"\n\"coupée\"\nmsgstr \"\"\n\"a split \"\n\"sentence\"\n\n" +
		"msgid \"vide\"\nmsgstr \"\"\n"
	if err := os.WriteFile(path, []byte(po), 0o600); err != nil {
		t.Fatal(err)
	}
	cat, err := loadCatalogue(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cat["une phrase coupée"]; got != "a split sentence" {
		t.Errorf("continuation lines not joined: %q", got)
	}
	if _, ok := cat["vide"]; ok {
		t.Error("an empty msgstr must not be recorded as a translation")
	}
}

func TestScanTargetsResolvesSectionTitles(t *testing.T) {
	targets, err := scanTargets(filepath.Join(repoRoot(t), "doc", "source", "cmd_mode.rst"))
	if err != nil {
		t.Fatal(err)
	}
	if got := targets["cmd_filter"]; got == "" {
		t.Fatalf("cmd_filter target not resolved: %#v", targets)
	}
}
