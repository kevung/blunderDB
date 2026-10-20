package issuance

import (
	"strings"
	"testing"
)

func TestSealComment_VerifiesTheWordsItCovers(t *testing.T) {
	id, err := NewIdentity("Coach Martin")
	if err != nil {
		t.Fatal(err)
	}
	sig, err := SealComment(id, "Joue 8/4 6/4.", "Martin", 0xfeedbeef)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := CheckComment(sig, "Joue 8/4 6/4.", "Martin", 0xfeedbeef)
	if !ok || !s.Verified || s.IssuerName != "Coach Martin" || s.Fingerprint != id.Fingerprint() {
		t.Fatalf("intact comment: %+v ok=%v; want verified by %s", s, ok, id.Fingerprint())
	}

	for name, c := range map[string]struct {
		text, author string
		position     uint64
	}{
		"text altered":   {"Joue 13/7.", "Martin", 0xfeedbeef},
		"author altered": {"Joue 8/4 6/4.", "Élève", 0xfeedbeef},
		"moved":          {"Joue 8/4 6/4.", "Martin", 0xfeedbeee},
	} {
		s, ok := CheckComment(sig, c.text, c.author, c.position)
		if !ok || s.Verified || s.IssuerName != "Coach Martin" {
			t.Errorf("%s: %+v ok=%v; want signed by Coach Martin, not verified", name, s, ok)
		}
	}

	forged := strings.Replace(sig, "Coach Martin", "Coach Durand", 1)
	if s, ok := CheckComment(forged, "Joue 8/4 6/4.", "Martin", 0xfeedbeef); !ok || s.Verified {
		t.Errorf("forged issuer name: %+v ok=%v; want signed, not verified", s, ok)
	}
	if s, ok := CheckComment("{not json", "x", "", 0); !ok || s.Verified {
		t.Errorf("unreadable signature: %+v ok=%v; want signed, not verified", s, ok)
	}
	if _, ok := CheckComment("", "x", "", 0); ok {
		t.Error("an empty signature must read as unsigned")
	}
}
