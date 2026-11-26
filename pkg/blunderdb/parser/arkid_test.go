package parser

import (
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func TestParsePositionReadsAnArkID(t *testing.T) {
	want, err := domain.DecodeArkID("[34]65:x2(ySQmLhfA)S3Qm3L3hf3")
	if err != nil {
		t.Fatal(err)
	}
	for _, in := range []string{
		"[34]65:x2(ySQmLhfA)S3Qm3L3hf3",
		"ArkID=[34]65:x2(ymhf)m3hf3(SQLA)S3QL3\n",
		"look at this\n  [34]65:x2(ySQmLhfA)S3Qm3L3hf3  \nthanks",
	} {
		res, err := ParsePosition(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if res.Format != FormatArkID || res.Position != want {
			t.Errorf("%q: format %q, position differs", in, res.Format)
		}
	}
}

func TestParsePositionRefusesABadArkID(t *testing.T) {
	if _, err := ParsePosition("[34]65:x2(ySQ)S3"); err == nil {
		// 2 checkers a side only is a legal board; the point is that the
		// decoder, not the router, judges.
		t.Log("accepted a sparse board")
	}
	if _, err := ParsePosition("[34]65:f3"); err == nil {
		t.Error("a point of three checkers without a gate was accepted")
	}
}
