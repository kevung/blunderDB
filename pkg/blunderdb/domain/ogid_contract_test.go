package domain

import (
	"encoding/json"
	"os"
	"testing"
)

// The OGID contract (issue #260, fiche I.4).
//
// Every case pairs an OGID with the XGID of the SAME physical position, and
// this test asserts that the two decoders agree. Pinning the new reader
// against the old one — rather than against a table of expectations somebody
// typed out — is what makes the corpus a contract instead of a transcription
// of one person's reading of a specification.
//
// The strings themselves come from the reference implementation (AnkiGammon
// 1.8.1), over positions dumped from testdata/test.xg and testdata/test.mat.
// That provenance is the point: the fiche forbade writing this reader before
// real samples existed, because the format it originally named turned out not
// to exist at all.

type ogidCorpus struct {
	Cases []struct {
		XGID string `json:"xgid"`
		OGID string `json:"ogid"`
	} `json:"cases"`
}

func TestDecodeOGIDMatchesDecodeXGID(t *testing.T) {
	raw, err := os.ReadFile("testdata/ogid_corpus.json")
	if err != nil {
		raw, err = os.ReadFile("../../../testdata/ogid_corpus.json")
	}
	if err != nil {
		t.Fatalf("reading the corpus: %v", err)
	}
	var corpus ogidCorpus
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatalf("parsing the corpus: %v", err)
	}
	if len(corpus.Cases) == 0 {
		t.Fatal("the corpus is empty")
	}

	for _, c := range corpus.Cases {
		want, err := DecodeXGID(c.XGID)
		if err != nil {
			t.Fatalf("DecodeXGID(%s): %v", c.XGID, err)
		}
		got, err := DecodeOGID(c.OGID)
		if err != nil {
			t.Fatalf("DecodeOGID(%s): %v", c.OGID, err)
		}

		if got.Board.Points != want.Board.Points {
			t.Errorf("board differs\n OGID %s\n XGID %s\n  got %v\n want %v",
				c.OGID, c.XGID, got.Board.Points, want.Board.Points)
			continue
		}
		if got.Board.Bearoff != want.Board.Bearoff {
			t.Errorf("bearoff differs for %s: got %v, want %v", c.OGID, got.Board.Bearoff, want.Board.Bearoff)
		}
		if got.Cube != want.Cube {
			t.Errorf("cube differs for %s: got %+v, want %+v", c.OGID, got.Cube, want.Cube)
		}
		if got.Dice != want.Dice {
			t.Errorf("dice differ for %s: got %v, want %v", c.OGID, got.Dice, want.Dice)
		}
		if got.PlayerOnRoll != want.PlayerOnRoll {
			t.Errorf("player on roll differs for %s: got %d, want %d", c.OGID, got.PlayerOnRoll, want.PlayerOnRoll)
		}
		if got.Score != want.Score {
			t.Errorf("score differs for %s: got %v, want %v", c.OGID, got.Score, want.Score)
		}
		if got.DecisionType != want.DecisionType {
			t.Errorf("decision type differs for %s: got %d, want %d", c.OGID, got.DecisionType, want.DecisionType)
		}
	}
}

// A malformed OGID is refused with a named error, never decoded into a
// plausible-looking board — the failure mode a lax reader produces is a
// position nobody can trace back to anything.
func TestDecodeOGIDRefusesMalformed(t *testing.T) {
	for name, s := range map[string]string{
		"empty":            "",
		"two fields":       "11ccccc:66666",
		"short cube":       "11ccccc:66666:N0",
		"bad character":    "11ccc!c:66666888dddddoo:N0N",
		"too many on one":  "1111111111111111:66666888dddddoo:N0N",
		"colours collided": "1111:1111:N0N",
	} {
		if _, err := DecodeOGID(s); err == nil {
			t.Errorf("%s: expected a refusal for %q", name, s)
		}
	}
}

// The router must never have to ask which format a paste is in.
func TestLooksLikeOGID(t *testing.T) {
	yes := []string{
		"OGID=11ccccchhhjjjjj:66666888dddddoo:N0N::B::0:0:7:",
		"11ccccchhhjjjjj:66666888dddddoo:N0N:51:B::0:0:7:",
		"  ogid=11ccccchhhjjjjj:66666888dddddoo:W2N  ",
	}
	for _, s := range yes {
		if !LooksLikeOGID(s) {
			t.Errorf("LooksLikeOGID(%q) = false, want true", s)
		}
	}
	no := []string{
		"",
		"XGID=-b----E-C---eE---c-e----B-:0:0:1:51:0:0:0:7:0",
		"-b----E-C---eE---c-e----B-:0:0:1:51:0:0:0:7:0", // an XGID's third field is a number
		"just some prose about a position",
	}
	for _, s := range no {
		if LooksLikeOGID(s) {
			t.Errorf("LooksLikeOGID(%q) = true, want false", s)
		}
	}
}
