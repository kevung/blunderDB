package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestDocsNeverCiteAFutureVersion enforces one half of the documentation
// rule in CLAUDE.md ("the documentation describes the published version, in
// the present tense, and nothing else") that no phrase-level grep can catch:
// a version NUMBER above the one conf.py declares. Two "depuis la 0.37.0"
// shipped in a 0.36.0 site (tasks/critique-doc-2026-09, persona 7 #2) —
// the release skill's grep looks for tell-tale words, and a number is not a
// word. Only application versions (0.x.y) are compared: the schema version
// (2.x.y) and third-party versions live on their own scales.
func TestDocsNeverCiteAFutureVersion(t *testing.T) {
	root := repoRoot(t)

	conf, err := os.ReadFile(filepath.Join(root, "doc", "source", "conf.py"))
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^release\s*=\s*'(\d+)\.(\d+)\.(\d+)'`).FindSubmatch(conf)
	if m == nil {
		t.Fatal("doc/source/conf.py: no `release = 'X.Y.Z'` line")
	}
	release := [3]int{atoi(m[1]), atoi(m[2]), atoi(m[3])}

	pages, err := filepath.Glob(filepath.Join(root, "doc", "source", "*.rst"))
	if err != nil {
		t.Fatal(err)
	}
	cite := regexp.MustCompile(`\b0\.(\d+)\.(\d+)\b`)
	for _, page := range pages {
		body, err := os.ReadFile(page)
		if err != nil {
			t.Fatal(err)
		}
		for lineNo, line := range strings.Split(string(body), "\n") {
			for _, c := range cite.FindAllStringSubmatch(line, -1) {
				v := [3]int{0, atoi([]byte(c[1])), atoi([]byte(c[2]))}
				if newer(v, release) {
					t.Errorf("%s:%d cites version %s, above the published %d.%d.%d — the docs describe what ships; say it in the present tense, without a number",
						filepath.Base(page), lineNo+1, c[0], release[0], release[1], release[2])
				}
			}
		}
	}
}

func newer(a, b [3]int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] > b[i]
		}
	}
	return false
}

func atoi(b []byte) int {
	n, err := strconv.Atoi(string(b))
	if err != nil {
		panic(err)
	}
	return n
}
