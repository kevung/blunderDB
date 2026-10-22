package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestCLIPageDocumentsEveryCommand locks the CLI reference (doc/source/cli.rst
// and its chapters doc/source/cli_*.rst, the French source of the
// translations) to handlers(): every subcommand has a section "<name> — …"
// in one of them, and every such section names an existing command. Commands
// documented elsewhere are exempted by name, with their page.
func TestCLIPageDocumentsEveryCommand(t *testing.T) {
	pages, err := filepath.Glob("doc/source/cli_*.rst")
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) == 0 {
		t.Fatal("no doc/source/cli_*.rst chapter found")
	}
	pages = append([]string{"doc/source/cli.rst"}, pages...)
	var src []byte
	for _, page := range pages {
		body, err := os.ReadFile(page)
		if err != nil {
			t.Fatalf("reading %s: %v", page, err)
		}
		src = append(src, body...)
		src = append(src, '\n')
	}

	// A command section is a heading "name — Title" underlined with dashes.
	heading := regexp.MustCompile(`(?m)^([a-z]+) — [^\n]*\n-{3,}\n`)
	documented := map[string]bool{}
	for _, m := range heading.FindAllStringSubmatch(string(src), -1) {
		documented[m[1]] = true
	}

	elsewhere := map[string]string{
		"serve":   "doc/source/mode_headless.rst",
		"migrate": "doc/source/mode_headless.rst",
		"call":    "doc/source/mode_headless.rst",
		"help":    "doc/source/cli_admin.rst (completion)",
	}
	for name, page := range elsewhere {
		body, err := os.ReadFile(strings.SplitN(page, " ", 2)[0])
		if err != nil {
			t.Fatalf("reading %s: %v", page, err)
		}
		if !strings.Contains(string(body), "blunderdb "+name) {
			t.Errorf("command %q is exempted from the command sections because %s documents it, but that page never mentions `blunderdb %s`", name, page, name)
		}
	}

	for _, name := range CommandNames() {
		if documented[name] {
			continue
		}
		if _, ok := elsewhere[name]; ok {
			continue
		}
		t.Errorf("subcommand %q has no section \"%s — …\" in doc/source/cli.rst or its cli_*.rst chapters (and no exemption naming the page that documents it)", name, name)
	}
	for name := range documented {
		if !IsCommand(name) {
			t.Errorf("the CLI pages document %q, which is not a subcommand any more", name)
		}
	}
}
