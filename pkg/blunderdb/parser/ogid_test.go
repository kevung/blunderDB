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
// without a code path of their own. The corpus pairs each OGID with the XGID
// of the same physical position, so the assertion is that the two readers
// agree.
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

// A pasted OGID puts on roll the player HedgeHog means: the opponent of the
// colour field, or the doubler at a pending double. Both strings are from a
// match HedgeHog exported.
func TestParsePositionOGIDPlayerOnRoll(t *testing.T) {
	for text, want := range map[string]int{
		"11ccccchhhjjjjj:66666888dddddoo:N0N:46:B:IB:0:0:3:0":    domain.White,
		"17ccccghhhjjjjj:66666888dddddoo:N0N:46:W:R:0:0:3:1":     domain.Black,
		"OGID=cccccgghhhjjjjj:112233666777dll:N0O::B:D:0:0:3:11": domain.Black,
		"ccccchhhjjjllmm:24455566699ddkk:N0O::W:D:1:1:3:10":      domain.White,
	} {
		res, err := ParsePosition(text)
		if err != nil {
			t.Fatalf("ParsePosition(%q): %v", text, err)
		}
		if res.Position.PlayerOnRoll != want {
			t.Errorf("%s: on roll %d, want %d", text, res.Position.PlayerOnRoll, want)
		}
	}
}
