package ingest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func readTestdata(t *testing.T, parts ...string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{"..", "..", "..", "testdata"}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// A truncated .mat must be refused as a whole: the games before the cut parse
// cleanly, so without the completeness check they would be imported silently.
// Several cuts per file, so a single lucky cut on a detectable boundary cannot
// make the test pass.
func TestMATTruncatedIsRefused(t *testing.T) {
	for _, tc := range []struct {
		file  string
		games int
	}{
		{"test.mat", 7},
		{"charlot1-charlot2_7p_2025-11-08-2305.mat", 4},
	} {
		data := readTestdata(t, tc.file)

		t.Run(tc.file+"/complete", func(t *testing.T) {
			g, err := MapGnuBGText(string(data))
			if err != nil {
				t.Fatalf("complete file refused: %v", err)
			}
			if len(g.Games) != tc.games {
				t.Fatalf("games = %d, want %d", len(g.Games), tc.games)
			}
		})

		for _, cut := range []int{len(data) / 4, len(data) / 2, 3 * len(data) / 4, len(data) - 40} {
			t.Run(fmt.Sprintf("%s@%d", tc.file, cut), func(t *testing.T) {
				g, err := MapGnuBGText(string(data[:cut]))
				assertIncompleteMatch(t, g, err)

				// The file path (Desktop import) goes through the same check.
				path := filepath.Join(t.TempDir(), "cut.mat")
				if err := os.WriteFile(path, data[:cut], 0o600); err != nil {
					t.Fatal(err)
				}
				g, err = MapGnuBG(path)
				assertIncompleteMatch(t, g, err)
			})
		}
	}
}

func assertIncompleteMatch(t *testing.T, g *MatchGraph, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("truncated match accepted (%d games mapped)", len(g.Games))
	}
	// A cut inside a score line is already a parse error; only a cut inside
	// the moves reaches the completeness check, and must be named by it.
	if !errors.Is(err, ErrIncompleteMatch) && !strings.Contains(err.Error(), "parse gnubg") {
		t.Fatalf("want ErrIncompleteMatch or a parse error, got %v", err)
	}
}

// bgfparser's text parser never fails: unrelated text yields an empty
// Position. The mapping must refuse it rather than store an empty board,
// while every real BGBlitz text export still maps.
func TestBGFTextWithoutPositionIsRefused(t *testing.T) {
	for _, content := range []string{"", "bonjour", "ceci ne contient ni match ni position\n"} {
		t.Run(fmt.Sprintf("%q", content), func(t *testing.T) {
			g, err := MapBGFTextPositionText(content)
			if err == nil {
				t.Fatalf("unrelated text accepted: %+v", g)
			}
			if !errors.Is(err, ErrNoPosition) || !errors.Is(err, storage.ErrInvalid) {
				t.Fatalf("want ErrNoPosition wrapping storage.ErrInvalid, got %v", err)
			}
		})
	}

	t.Run("file path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "hello.txt")
		if err := os.WriteFile(path, []byte("bonjour\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := MapBGFTextPosition(path); !errors.Is(err, ErrNoPosition) {
			t.Fatalf("want ErrNoPosition, got %v", err)
		}
	})

	fixtures, err := filepath.Glob(filepath.Join("..", "..", "..", "testdata", "bgf_positions", "*.txt"))
	if err != nil || len(fixtures) == 0 {
		t.Fatalf("no BGF text fixture found (%v)", err)
	}
	for _, f := range fixtures {
		t.Run(filepath.Base(f), func(t *testing.T) {
			graphs, err := MapBGFTextPosition(f)
			if err != nil {
				t.Fatalf("real export refused: %v", err)
			}
			if len(graphs) != 1 || graphs[0].Position == nil {
				t.Fatalf("want one position, got %d", len(graphs))
			}
		})
	}
}
