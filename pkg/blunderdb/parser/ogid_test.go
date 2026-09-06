package parser

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestParsePositionReadsOGID pins the claim the package doc makes: an OGID
// reaches the database through the SAME entry point as an XGID, so the
// clipboard, `blunderdb import` and /v1/positions.parseText all accept one
// without a code path of their own (#260).
//
// The corpus pairs each OGID with the XGID of the same physical position, so
// the assertion is that the two readers agree — a fact, not a transcription of
// someone's reading of the format.
func TestParsePositionReadsOGID(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "ogid_corpus.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	var corpus struct {
		Cases []struct {
			XGID string `json:"xgid"`
			OGID string `json:"ogid"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatalf("decode corpus: %v", err)
	}
	if len(corpus.Cases) == 0 {
		t.Fatal("empty corpus")
	}

	for _, c := range corpus.Cases {
		want, err := domain.DecodeXGID(strings.TrimPrefix(c.XGID, "XGID="))
		if err != nil {
			t.Fatalf("reference XGID %q: %v", c.XGID, err)
		}
		// Both spellings, because both occur in the wild: the identifier alone,
		// and the identifier introduced by its name.
		for _, text := range []string{c.OGID, "OGID=" + c.OGID, "voici la position\nOGID=" + c.OGID + "\n"} {
			res, err := ParsePosition(text)
			if err != nil {
				t.Fatalf("ParsePosition(%q): %v", text, err)
			}
			if res.Position.Board != want.Board {
				t.Errorf("board mismatch for %q", text)
			}
			if res.Position.Cube != want.Cube || res.Position.PlayerOnRoll != want.PlayerOnRoll {
				t.Errorf("cube/turn mismatch for %q", text)
			}
			if res.Analysis == nil || res.Analysis.AnalysisType != "" {
				t.Errorf("an OGID carries no analysis, got %+v", res.Analysis)
			}
		}
	}
}

// TestParsePositionStillRefusesNonIdentifiers guards the other half: reading
// OGID must not turn every colon-separated line into a position.
func TestParsePositionStillRefusesNonIdentifiers(t *testing.T) {
	for _, text := range []string{
		"bonjour",
		"12:34:56",
		"a:b:c",
		"Position-ID: 4HPwATDgc/ABMA",
	} {
		if _, err := ParsePosition(text); err == nil {
			t.Errorf("ParsePosition(%q) accepted a non-identifier", text)
		}
	}
}
