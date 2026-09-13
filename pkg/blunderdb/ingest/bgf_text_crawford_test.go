package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestBGFTextPositionWritesTheCrawfordSentinel is #360 on the BGBlitz position
// file, the one importer that reads a position through an XGID it does not
// decode itself: bgfparser reads the scores, never field 7, so every position
// one point from victory came in as the Crawford game.
//
// The fixture is a real BGBlitz export at 6-3 in a 7-point match, and it says
// "not the Crawford game" twice, independently: its XGID's field 7 is 0, and
// the Crawford bit of its gnubg Match-ID (QYnoAGAAGAAE) is 0. So O, one point
// away, is at the post-Crawford sentinel 0 (CONTEXT.md, « Away score »). The
// same text with field 7 set to 1 stays at the Crawford sentinel 1.
func TestBGFTextPositionWritesTheCrawfordSentinel(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "bgf_positions", "01_checkerPosition_FR.txt")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	const postCrawford = ":1:-1:1:21:3:6:0:7:10"
	if !strings.Contains(string(raw), postCrawford) {
		t.Fatalf("fixture no longer carries the XGID tail %q; the case proves nothing", postCrawford)
	}

	for _, c := range []struct {
		name, text string
		want       [2]int
	}{
		{"after the Crawford game", string(raw), [2]int{4, domain.PostCrawford}},
		{"in the Crawford game", strings.Replace(string(raw), postCrawford, ":1:-1:1:21:3:6:1:7:10", 1), [2]int{4, domain.Crawford}},
	} {
		t.Run(c.name, func(t *testing.T) {
			graphs, err := MapBGFTextPositionText(c.text)
			if err != nil {
				t.Fatalf("MapBGFTextPositionText: %v", err)
			}
			if len(graphs) != 1 {
				t.Fatalf("got %d position graphs, want 1", len(graphs))
			}
			if got := graphs[0].Position.Score; got != c.want {
				t.Errorf("away score = %v, want %v", got, c.want)
			}
		})
	}
}
