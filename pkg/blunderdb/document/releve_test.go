package document

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func readExample(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/releve-exemple.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDecodeExample(t *testing.T) {
	r, err := Decode(readExample(t))
	if err != nil {
		t.Fatal(err)
	}
	if r.Document.Title != "Exemples inventés" || len(r.Entries) != 4 || r.Document.Profile.Home != "right" {
		t.Fatalf("decoded %+v", r)
	}
	if got := r.Entries[0].Reference(); got != "p. 1, Exemple 1" {
		t.Errorf("reference = %q", got)
	}
	s := Summarize(r)
	if s.Sure != 3 || s.ToVerify != 1 || s.Verified != 2 || s.Positions != 3 || s.Repeated != 1 || len(s.Unverified) != 0 {
		t.Errorf("summary = %+v", s)
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	r, err := Decode(readExample(t))
	if err != nil {
		t.Fatal(err)
	}
	out, err := r.Encode()
	if err != nil {
		t.Fatal(err)
	}
	back, err := Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := back.Encode()
	if string(again) != string(out) {
		t.Errorf("round trip changed the relevé:\n%s\n---\n%s", out, again)
	}
}

func TestDecodeRefusals(t *testing.T) {
	base := string(readExample(t))
	cases := map[string]func(string) string{
		"unknown field": func(s string) string {
			return strings.Replace(s, `"verified": false`, `"verified": false, "comment": "x"`, 1)
		},
		"trailing data":     func(s string) string { return s + "{}" },
		"wrong format":      func(s string) string { return strings.Replace(s, "blunderdb-releve", "autre", 1) },
		"newer version":     func(s string) string { return strings.Replace(s, `"version": 1`, `"version": 2`, 1) },
		"no title":          func(s string) string { return strings.Replace(s, `"title": "Exemples inventés"`, `"title": " "`, 1) },
		"bad sha256":        func(s string) string { return strings.Replace(s, "e3b0c442", "zz", 1) },
		"bad home":          func(s string) string { return strings.Replace(s, `"home": "right"`, `"home": "up"`, 1) },
		"rank gap":          func(s string) string { return strings.Replace(s, `"rank": 3`, `"rank": 5`, 1) },
		"page backwards":    func(s string) string { return strings.Replace(s, `"page": 3`, `"page": 1`, 1) },
		"bad state":         func(s string) string { return strings.Replace(s, `"state": "to-verify"`, `"state": "maybe"`, 1) },
		"sure with doubt":   func(s string) string { return strings.Replace(s, `"state": "to-verify"`, `"state": "sure"`, 1) },
		"rect off the page": func(s string) string { return strings.Replace(s, `"w": 0.8, "h": 0.4 }`, `"w": 0.95, "h": 0.4 }`, 1) },
		"16 checkers": func(s string) string {
			return strings.Replace(s, "-b----E-C---eE---c-e----B-:0:0:1:31", "-b----E-C---eE---c-e----C-:0:0:1:31", 1)
		},
		"short XGID": func(s string) string {
			return strings.Replace(s, ":0:0:0:7:10\",\n      \"state\": \"sure\",\n      \"rect\"", ":0:0:0:7\",\n      \"state\": \"sure\",\n      \"rect\"", 1)
		},
		"bad dice":          func(s string) string { return strings.Replace(s, ":1:31:", ":1:71:", 1) },
		"score past length": func(s string) string { return strings.Replace(s, ":00:2:4:0:7:", ":00:2:7:0:7:", 1) },
		"X on O's bar": func(s string) string {
			return strings.Replace(s, "-b----E-C---eE---c-e----B-:0:0:1:31", "Ab----E-C---eE---c-e----A-:0:0:1:31", 1)
		},
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			src := edit(base)
			if src == base {
				t.Fatal("the edit changed nothing")
			}
			_, err := Decode([]byte(src))
			if !errors.Is(err, ErrInvalid) || !errors.Is(err, storage.ErrInvalid) {
				t.Errorf("err = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestEmptyReleveRefused(t *testing.T) {
	r := &Releve{Format: Format, Version: Version, Document: Source{Title: "T"}}
	if _, err := r.Encode(); !errors.Is(err, ErrInvalid) {
		t.Errorf("err = %v", err)
	}
}

func TestUnverified(t *testing.T) {
	r, _ := Decode(readExample(t))
	r.Entries[1].Verified = false
	r.Entries[3].State = Uncertain
	if got := Unverified(r); len(got) != 2 || got[0] != 2 || got[1] != 4 {
		t.Errorf("unverified = %v", got)
	}
}
