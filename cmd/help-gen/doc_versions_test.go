package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestDocsNeverCiteAFutureVersion: no application version (0.x.y) above
// conf.py's may appear in the docs (CLAUDE.md, Documentation); the release
// skill's grep only looks for words.
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
