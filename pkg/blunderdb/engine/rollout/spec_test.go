package rollout

import (
	"strings"
	"testing"
)

func TestParseSpec(t *testing.T) {
	std := Standard()
	std.Ply = 1
	free := Fast()
	free.MaxGames, free.MinGames, free.Truncation = 648, 108, 0
	cases := map[string]Settings{
		"":                             Fast(),
		"rapide":                       Fast(),
		"standard,ply=1":               std,
		"libre games=648 truncation=0": free,
		"games=648;truncation=0":       free,
	}
	for spec, want := range cases {
		got, err := ParseSpec(spec)
		if err != nil || got != want {
			t.Errorf("ParseSpec(%q) = %+v, %v; want %+v", spec, got, err, want)
		}
	}
	for _, bad := range []string{"turbo", "games=100", "ply=1,standard", "depth=3", "jsd=x"} {
		if _, err := ParseSpec(bad); err == nil {
			t.Errorf("ParseSpec(%q) accepted", bad)
		}
	}
}

// Two rollouts that roll different plays are two Configurations: their
// Signatures differ, so storing one never drops the other. The order the
// plays are named in is not part of it.
func TestSignatureNamesTheCandidates(t *testing.T) {
	five, three := Fast(), Fast()
	three.Candidates = 3
	if five.Signature() == three.Signature() {
		t.Error("5 and 3 best candidates share a Signature")
	}
	named := five.SignatureFor([]string{"13/10 6/5", "8/5 6/5"})
	if named == five.Signature() {
		t.Error("named plays share the Signature of the best candidates")
	}
	if named != five.SignatureFor([]string{"8/5 6/5", "13/10 6/5", "8/5 6/5"}) {
		t.Error("the order or repetition of named plays changed the Signature")
	}
	if named == five.SignatureFor([]string{"8/5 6/5", "24/21 13/11"}) {
		t.Error("two different sets of named plays share a Signature")
	}
	if !strings.HasPrefix(five.Signature(), EngineVersion+";") {
		t.Errorf("Signature %q no longer starts with the EngineVersion", five.Signature())
	}
}
