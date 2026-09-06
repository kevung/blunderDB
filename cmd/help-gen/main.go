// Command help-gen builds the in-app help bundles (frontend/src/i18n/help/*.js)
// from the documentation that already states the same facts.
//
// Three tabs — the manual, the keyboard shortcuts and the command line — are
// rendered from doc/source/manuel.rst, doc/source/raccourcis.rst and
// doc/source/cmd_mode.rst, in the nine documentation languages, through the
// gettext catalogues under doc/source/locale. Only the About tab is
// hand-written HTML, under frontend/src/i18n/help/prose/<lang>.html, and is
// copied through verbatim; see the ADR "the in-app help is generated from the
// documentation" for why it alone is not derived from the .rst.
//
// The manual tab was a hand-written digest until it had drifted eight sections
// behind manuel.rst — the trash, the tags, the micro-drills, the cube matrix
// among them — in nine languages at once, with nothing to catch it. Generating
// it is what makes that drift impossible rather than merely noticed.
//
// Usage:
//
//	go run ./cmd/help-gen          # rewrite the nine bundles (make help)
//	go run ./cmd/help-gen -check   # exit non-zero if any bundle is stale
//
// TestHelpBundlesAreCurrent runs the -check path, so `go test ./...` fails when
// the documentation moved and the bundles were not regenerated.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const sourceLang = "fr"

// languages is the documentation's own language list (French source plus the
// eight gettext catalogues), and must stay equal to LOCALES in
// frontend/src/i18n/index.js — i18n.locales.test.js checks the bundle set.
var languages = []string{"de", "el", "en", "es", "fi", "fr", "it", "ja", "ru"}

// tabs maps each generated help tab to the .rst document it is rendered from.
var tabs = []struct{ tab, doc string }{
	{"manual", "manuel"},
	{"shortcuts", "raccourcis"},
	{"commands", "cmd_mode"},
}

type refTarget struct{ doc, title string }

type generator struct {
	root string
	// refs resolves a `.. _label:` to the document and French title of the
	// section it precedes, so a :ref: role can render as readable text.
	refs map[string]refTarget
	// cats caches catalogues by "<lang>/<doc>".
	cats map[string]catalogue
}

func main() {
	check := flag.Bool("check", false, "verify the committed bundles are up to date instead of rewriting them")
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	stale, err := run(*root, *check)
	if err != nil {
		fmt.Fprintln(os.Stderr, "help-gen:", err)
		os.Exit(1)
	}
	if len(stale) > 0 {
		fmt.Fprintf(os.Stderr, "help-gen: %d help bundle(s) out of date: %s\n", len(stale), strings.Join(stale, ", "))
		fmt.Fprintln(os.Stderr, "run: make help")
		os.Exit(1)
	}
}

// run generates every bundle. In check mode nothing is written and the names of
// the bundles that differ from what the documentation says are returned.
func run(root string, check bool) ([]string, error) {
	g, err := newGenerator(root)
	if err != nil {
		return nil, err
	}
	var stale []string
	for _, lang := range languages {
		want, err := g.bundle(lang)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", lang, err)
		}
		path := filepath.Join(root, "frontend", "src", "i18n", "help", lang+".js")
		if check {
			got, err := os.ReadFile(path)
			if err != nil || string(got) != want {
				stale = append(stale, lang+".js")
			}
			continue
		}
		if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
			return nil, err
		}
	}
	return stale, nil
}

func newGenerator(root string) (*generator, error) {
	g := &generator{root: root, refs: map[string]refTarget{}, cats: map[string]catalogue{}}
	// Cross-references may point into any document, so every source file
	// contributes its link targets.
	sources, err := filepath.Glob(filepath.Join(root, "doc", "source", "*.rst"))
	if err != nil {
		return nil, err
	}
	for _, path := range sources {
		targets, err := scanTargets(path)
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(filepath.Base(path), ".rst")
		for label, title := range targets {
			g.refs[label] = refTarget{doc: name, title: title}
		}
	}
	return g, nil
}

// catalogueFor loads (once) the catalogue of one document in one language.
func (g *generator) catalogueFor(lang, doc string) (catalogue, error) {
	key := lang + "/" + doc
	if c, ok := g.cats[key]; ok {
		return c, nil
	}
	c, err := loadCatalogue(filepath.Join(g.root, "doc", "source", "locale", lang, "LC_MESSAGES", doc+".po"))
	if err != nil {
		return nil, err
	}
	g.cats[key] = c
	return c, nil
}

// refTitle renders a `:ref:` label as the title of the section it points at.
func (g *generator) refTitle(label string, tr translator) (string, error) {
	target, ok := g.refs[strings.TrimSpace(label)]
	if !ok {
		return "", fmt.Errorf("unknown :ref: target %q", label)
	}
	if tr.lang == sourceLang {
		return target.title, nil
	}
	cat, err := g.catalogueFor(tr.lang, target.doc)
	if err != nil {
		return "", err
	}
	title, ok := cat[target.title]
	if !ok {
		return "", fmt.Errorf("no %s translation for section title %q (target of :ref:`%s`)", tr.lang, target.title, label)
	}
	return title, nil
}

// bundle renders the complete JavaScript module for one language.
func (g *generator) bundle(lang string) (string, error) {
	prose, err := g.prose(lang)
	if err != nil {
		return "", err
	}
	rendered := map[string]string{}
	for _, t := range tabs {
		doc, err := parseRST(filepath.Join(g.root, "doc", "source", t.doc+".rst"))
		if err != nil {
			return "", err
		}
		tr := translator{lang: lang}
		if lang != sourceLang {
			cat, err := g.catalogueFor(lang, t.doc)
			if err != nil {
				return "", err
			}
			tr.cats = []catalogue{cat}
		}
		html, err := g.renderTab(doc, tr)
		if err != nil {
			return "", fmt.Errorf("%s: %w", t.doc, err)
		}
		rendered[t.tab] = html
	}

	var b strings.Builder
	b.WriteString(header)
	for _, tab := range []string{"manual", "shortcuts", "commands", "about"} {
		body, ok := rendered[tab]
		if !ok {
			body, ok = prose[tab]
			if !ok {
				return "", fmt.Errorf("prose/%s.html has no %q section", lang, tab)
			}
		}
		fmt.Fprintf(&b, "    %s: `\n%s`,\n", tab, jsTemplateLiteral(body))
	}
	out := strings.TrimSuffix(b.String(), ",\n") + "\n};\n"
	return out, nil
}

const header = `// GENERATED FILE — do not edit by hand, and do not translate it here.
//
// Produced by ` + "`go run ./cmd/help-gen`" + ` (make help) from:
//   - doc/source/manuel.rst      → the "manual" tab
//   - doc/source/raccourcis.rst  → the "shortcuts" tab
//   - doc/source/cmd_mode.rst    → the "commands" tab
//   - doc/source/locale/<lang>/LC_MESSAGES/*.po for the eight translations
//   - frontend/src/i18n/help/prose/<lang>.html → the "about" tab
//
// Fix the documentation (and its .po catalogues), or the prose fragment, then
// run ` + "`make help`" + `. TestHelpBundlesAreCurrent fails if this file is stale.
export default {
`

// prose reads the hand-written fragments, split on `<!-- tab: name -->`.
func (g *generator) prose(lang string) (map[string]string, error) {
	path := filepath.Join(g.root, "frontend", "src", "i18n", "help", "prose", lang+".html")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	var name string
	var body []string
	flush := func() {
		if name != "" {
			out[name] = strings.Trim(strings.Join(body, "\n"), "\n") + "\n"
		}
		body = nil
	}
	for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "<!-- tab:") {
			flush()
			name = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(line), "<!-- tab:"), "-->"))
			continue
		}
		body = append(body, line)
	}
	flush()
	return out, nil
}

// jsTemplateLiteral makes a fragment safe to embed in a backtick-quoted
// JavaScript template literal: a backslash, a backtick or a `${` would end it
// or start a substitution.
func jsTemplateLiteral(s string) string {
	r := strings.NewReplacer(`\`, `\\`, "`", "\\`", "${", "\\${")
	return r.Replace(s)
}
