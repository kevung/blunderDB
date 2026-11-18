package parser

import (
	"errors"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// TestParsePositionReadsGNUbgID: the three ways a GNUbg identifier is pasted
// reach the same decoder, and the response names the format read.
func TestParsePositionReadsGNUbgID(t *testing.T) {
	const id = "4HPwATDgc/ABMA:cIkFAAAAAAAE"
	want, err := domain.DecodeGNUbgID(id)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{
		id,
		"GNU Backgammon  Position ID: 4HPwATDgc/ABMA\n                 Match ID   : cIkFAAAAAAAE\n",
		"set gnubgid " + id,
	} {
		res, err := ParsePosition(text)
		if err != nil {
			t.Fatalf("ParsePosition(%q): %v", text, err)
		}
		if res.Format != FormatGNUbgID || res.Position != want {
			t.Errorf("ParsePosition(%q) = %s %+v", text, res.Format, res.Position)
		}
		if res.Analysis == nil || res.Analysis.AnalysisType != "" {
			t.Errorf("a GNUbg ID carries no analysis, got %+v", res.Analysis)
		}
	}

	res, err := ParsePosition("4HPwATDgc/ABMA")
	if err != nil || res.Format != FormatGNUbgID || res.Position.PlayerOnRoll != domain.Black {
		t.Errorf("Position ID alone = %v %+v", err, res)
	}
}

func TestParsePositionNamesTheFormat(t *testing.T) {
	for text, format := range map[string]string{
		"XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:0:10": FormatXGID,
		"11ccccchhhjjjjj:66666888dddddoo:N0N:51:W::0:0:7:":    FormatOGID,
	} {
		res, err := ParsePosition(text)
		if err != nil {
			t.Fatalf("ParsePosition(%q): %v", text, err)
		}
		if res.Format != format {
			t.Errorf("ParsePosition(%q).Format = %q, want %q", text, res.Format, format)
		}
	}
}

func TestParsePositionRefusesWithoutIdentifier(t *testing.T) {
	for _, text := range []string{"Bonjour, une question de videau", "//////////////"} {
		if _, err := ParsePosition(text); !errors.Is(err, ErrNoPosition) {
			t.Errorf("ParsePosition(%q) = %v, want ErrNoPosition", text, err)
		}
	}
}
