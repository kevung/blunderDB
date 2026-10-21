// Command help-gen builds the in-app help bundles (frontend/public/help/*.bin)
// from the documentation that already states the same facts.
//
// The manual, shortcuts and commands tabs are rendered from manuel.rst (and the
// chapters its toctree lists), raccourcis.rst and cmd_mode.rst through the
// gettext catalogues, in nine languages; only the About tab is hand-written HTML (prose/<lang>.html),
// copied verbatim (ADR-0034).
//
// A bundle is a gzip-compressed JSON object {manual, shortcuts, commands, about}:
// the help is ~4 MB of HTML for nine languages, and the binary embeds
// frontend/dist uncompressed. The WebView fetches one language on demand and
// inflates it with DecompressionStream (frontend/src/i18n/help/index.js). The
// extension is not .gz so that no server or WebView treats it as a transfer
// encoding and inflates it behind the loader's back.
//
// Usage:
//
//	go run ./cmd/help-gen          # rewrite the nine bundles (make help)
//	go run ./cmd/help-gen -check   # exit non-zero if any bundle is stale
//
// TestHelpBundlesAreCurrent runs -check, so `go test ./...` catches staleness.
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const sourceLang = "fr"

// languages must equal LOCALES in frontend/src/i18n/index.js
// (i18n.locales.test.js checks).
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
		path := filepath.Join(root, "frontend", "public", "help", lang+".bin")
		if check {
			// The compressed bytes depend on the Go release; the content is what must match.
			got, err := os.ReadFile(path)
			if err != nil {
				stale = append(stale, lang+".bin")
				continue
			}
			if plain, err := gunzip(got); err != nil || string(plain) != want {
				stale = append(stale, lang+".bin")
			}
			continue
		}
		packed, err := gzipBytes([]byte(want))
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, packed, 0o600); err != nil {
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

// renderDoc renders one .rst document through its own catalogue.
func (g *generator) renderDoc(name, lang string) (string, error) {
	doc, err := parseRST(filepath.Join(g.root, "doc", "source", name+".rst"))
	if err != nil {
		return "", err
	}
	tr := translator{lang: lang}
	if lang != sourceLang {
		cat, err := g.catalogueFor(lang, name)
		if err != nil {
			return "", err
		}
		tr.cats = []catalogue{cat}
	}
	return g.renderTab(doc, tr)
}

// bundle renders the JSON document (uncompressed) for one language.
func (g *generator) bundle(lang string) (string, error) {
	prose, err := g.prose(lang)
	if err != nil {
		return "", err
	}
	rendered := map[string]string{}
	for _, t := range tabs {
		html, err := g.renderDoc(t.doc, lang)
		if err != nil {
			return "", fmt.Errorf("%s: %w", t.doc, err)
		}
		rendered[t.tab] = html
	}

	out := map[string]string{}
	for _, tab := range []string{"manual", "shortcuts", "commands", "about"} {
		body, ok := rendered[tab]
		if !ok {
			body, ok = prose[tab]
			if !ok {
				return "", fmt.Errorf("prose/%s.html has no %q section", lang, tab)
			}
		}
		out[tab] = body
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return "", err
	}
	return b.String(), nil
}

func gzipBytes(plain []byte) ([]byte, error) {
	var b bytes.Buffer
	zw, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := zw.Write(plain); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func gunzip(packed []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(packed))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	var b bytes.Buffer
	if _, err := b.ReadFrom(zr); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

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
