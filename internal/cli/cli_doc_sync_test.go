package cli

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestCLIPageDocumentsEveryCommand locks doc/source/cli.rst (the French
// source of the translations) to handlers(): every subcommand has a section
// "<name> — …", and every such section names an existing command. Commands
// documented elsewhere are exempted by name, with their page.
func TestCLIPageDocumentsEveryCommand(t *testing.T) {
	src, err := os.ReadFile("doc/source/cli.rst")
	if err != nil {
		t.Fatalf("reading doc/source/cli.rst: %v", err)
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
		"help":    "doc/source/cli.rst (Syntaxe générale)",
	}
	for name, page := range elsewhere {
		body, err := os.ReadFile(strings.SplitN(page, " ", 2)[0])
		if err != nil {
			t.Fatalf("reading %s: %v", page, err)
		}
		if !strings.Contains(string(body), "blunderdb "+name) {
			t.Errorf("command %q is exempted from cli.rst because %s documents it, but that page never mentions `blunderdb %s`", name, page, name)
		}
	}

	for _, name := range CommandNames() {
		if documented[name] {
			continue
		}
		if _, ok := elsewhere[name]; ok {
			continue
		}
		t.Errorf("subcommand %q has no section \"%s — …\" in doc/source/cli.rst (and no exemption naming the page that documents it)", name, name)
	}
	for name := range documented {
		if !IsCommand(name) {
			t.Errorf("doc/source/cli.rst documents %q, which is not a subcommand any more", name)
		}
	}
}
