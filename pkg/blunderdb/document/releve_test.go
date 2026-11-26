package document

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/capture"
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
		"trailing data":   func(s string) string { return s + "{}" },
		"wrong format":    func(s string) string { return strings.Replace(s, "blunderdb-releve", "autre", 1) },
		"newer version":   func(s string) string { return strings.Replace(s, `"version": 2`, `"version": 3`, 1) },
		"no title":        func(s string) string { return strings.Replace(s, `"title": "Exemples inventés"`, `"title": " "`, 1) },
		"bad sha256":      func(s string) string { return strings.Replace(s, "e3b0c442", "zz", 1) },
		"bad home":        func(s string) string { return strings.Replace(s, `"home": "right"`, `"home": "up"`, 1) },
		"rank gap":        func(s string) string { return strings.Replace(s, `"rank": 3`, `"rank": 5`, 1) },
		"page backwards":  func(s string) string { return strings.Replace(s, `"page": 3`, `"page": 1`, 1) },
		"bad checkers":    func(s string) string { return strings.Replace(s, `"checkers": "to-verify"`, `"checkers": "maybe"`, 1) },
		"bad context":     func(s string) string { return strings.Replace(s, `"context": "read"`, `"context": "maybe"`, 1) },
		"no context":      func(s string) string { return strings.Replace(s, `"context": "read",`, ``, 1) },
		"sure with doubt": func(s string) string { return strings.Replace(s, `"checkers": "to-verify"`, `"checkers": "sure"`, 1) },
		"complete with doubt": func(s string) string {
			return strings.Replace(s, `"checkers": "to-verify",`, `"checkers": "to-verify", "complete": true,`, 1)
		},
		"state in version 2": func(s string) string {
			return strings.Replace(s, `"context": "read",`, `"context": "read", "state": "sure",`, 1)
		},
		"default without defaults": func(s string) string {
			return strings.Replace(s, `"context": "read",`, `"context": "default",`, 1)
		},
		"defaults in version 1": func(s string) string {
			return strings.Replace(strings.Replace(s, `"version": 2`, `"version": 1`, 1), `"profile": {`, `"defaults": {"context": "0:0:1:00:0:0:0:0:10", "confirmed": false}, "profile": {`, 1)
		},
		"bad defaults": func(s string) string {
			return strings.Replace(s, `"profile": {`, `"defaults": {"context": "0:0:1:00:0:0", "confirmed": false}, "profile": {`, 1)
		},
		"rect off the page": func(s string) string { return strings.Replace(s, `"w": 0.8, "h": 0.4 }`, `"w": 0.95, "h": 0.4 }`, 1) },
		"16 checkers": func(s string) string {
			return strings.Replace(s, "-b----E-C---eE---c-e----B-:0:0:1:31", "-b----E-C---eE---c-e----C-:0:0:1:31", 1)
		},
		"short XGID": func(s string) string {
			return strings.Replace(s, ":1:31:0:0:0:7:10\"", ":1:31:0:0:0:7\"", 1)
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
	r.Entries[3].Checkers = Uncertain
	if got := Unverified(r, false); len(got) != 2 || got[0] != 2 || got[1] != 4 {
		t.Errorf("unverified = %v", got)
	}
}

// A version 1 Relevé is still read, as version 2: a sure entry has its
// checkers sure and its context read, any other its context to verify.
func TestDecodeVersion1(t *testing.T) {
	data, err := os.ReadFile("testdata/releve-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if r.Version != Version || r.Entries[0].Checkers != Sure || r.Entries[0].Context != ContextRead || r.Entries[0].State != "" ||
		r.Entries[1].Checkers != ToVerify || r.Entries[1].Context != ContextToVerify {
		t.Fatalf("upgraded %+v", r.Entries)
	}
	if s := Summarize(r); s.Sure != 3 || s.ToVerify != 1 || s.Verified != 2 || len(s.Unverified) != 0 {
		t.Errorf("summary = %+v", s)
	}
	out, err := r.Encode()
	if err != nil || !strings.Contains(string(out), `"version": 2`) || strings.Contains(string(out), `"state"`) {
		t.Errorf("a version 1 relevé is written as version 2: %v\n%s", err, out)
	}
}

// A version 1 entry a scan's reader left with ContextDoubt comes under the
// structure reader's defaults, unconfirmed: still not sure.
func TestUpgradeStructureEntry(t *testing.T) {
	pos := "XGID=-b----E-C---eE---c-e----B-:" + StructureDefaults.Context
	cube := capture.Doubt{Field: "cube", Reason: "drawn"}
	r := &Releve{Format: Format, Version: 1, Document: Source{Title: "T"}, Entries: []Entry{
		{Rank: 1, Page: 1, Position: pos, State: ToVerify, Doubts: []capture.Doubt{ContextDoubt}},
		{Rank: 2, Page: 1, Position: pos, State: ToVerify, Doubts: []capture.Doubt{ContextDoubt, cube}},
	}}
	u := r.Upgrade()
	if err := u.Validate(); err != nil {
		t.Fatal(err)
	}
	if u.Document.Defaults == nil || u.Document.Defaults.Confirmed || u.Entries[0].Checkers != Sure || u.Entries[0].Context != ContextDefault ||
		u.Entries[1].Context != ContextToVerify || u.StateOf(u.Entries[0]) != ToVerify {
		t.Fatalf("upgraded %+v %+v", u.Document.Defaults, u.Entries)
	}
	if r.Version != 1 || r.Entries[0].State != ToVerify {
		t.Errorf("Upgrade changed its receiver")
	}
}

// An entry is sure only when its checkers are and its context is certified:
// read, or under defaults a person confirmed. Confirming the defaults leaves
// an entry whose cube was drawn to verify.
func TestContextRule(t *testing.T) {
	pos := "XGID=-b----E-C---eE---c-e----B-:" + StructureDefaults.Context
	cube := capture.Doubt{Field: "cube", Reason: "drawn"}
	r := &Releve{Format: Format, Version: Version, Document: Source{Title: "T", Defaults: structureDefaults()}, Entries: []Entry{
		{Rank: 1, Page: 1, Position: pos, Checkers: Sure, Context: ContextDefault, Doubts: []capture.Doubt{ContextDoubt}},
		{Rank: 2, Page: 1, Position: pos, Checkers: Sure, Context: ContextToVerify, Doubts: []capture.Doubt{ContextDoubt, cube}},
		{Rank: 3, Page: 1, Position: pos, Checkers: ToVerify, Complete: true, Context: ContextDefault, Doubts: []capture.Doubt{ContextDoubt}},
		{Rank: 4, Page: 1, Position: pos, Checkers: ToVerify, Complete: true, Context: ContextToVerify, Doubts: []capture.Doubt{ContextDoubt, cube}},
		{Rank: 5, Page: 1, Position: pos, Checkers: ToVerify, Context: ContextDefault, Doubts: []capture.Doubt{ContextDoubt, {Field: "pips", Reason: "none"}}},
		{Rank: 6, Page: 1, Position: "XGID=-b----E-C---eE---c-e----B-:0:0:1:31:0:0:0:7:10", Checkers: Sure, Context: ContextRead},
	}}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	states := func() (s []State) {
		for _, e := range r.Entries {
			s = append(s, r.StateOf(e))
		}
		return s
	}
	if got := states(); !slices.Equal(got, []State{ToVerify, ToVerify, ToVerify, ToVerify, ToVerify, Sure}) {
		t.Errorf("unconfirmed defaults: states %v", got)
	}
	if got := Unverified(r, true); !slices.Equal(got, []int{1, 2, 3, 4, 5}) {
		t.Errorf("unconfirmed defaults, complete accepted: unverified %v", got)
	}
	r.Document.Defaults.Confirmed = true
	if got := states(); !slices.Equal(got, []State{Sure, ToVerify, ToVerify, ToVerify, ToVerify, Sure}) {
		t.Errorf("confirmed defaults: states %v", got)
	}
	if got := Unverified(r, false); !slices.Equal(got, []int{2, 3, 4, 5}) {
		t.Errorf("confirmed defaults: unverified %v", got)
	}
	if got := Unverified(r, true); !slices.Equal(got, []int{2, 4, 5}) {
		t.Errorf("confirmed defaults, complete accepted: unverified %v", got)
	}
	s := Summarize(r)
	if s.Sure != 2 || s.Complete != 1 || s.CheckersConfirmed != 1 || s.ContextDefault != 3 || s.ContextToVerify != 2 || s.ContextRead != 1 {
		t.Errorf("summary %+v", s)
	}
	// The hint never offers complete entries whose context is not certified.
	r.Document.Defaults.Confirmed = false
	if h := refusalHint(r, []int{3, 4}, false); strings.Contains(h, "accept complete") || !strings.Contains(h, "confirm the document's defaults") {
		t.Errorf("unconfirmed defaults: hint %q", h)
	}
	if h := refusalHint(r, []int{4}, false); strings.Contains(h, "accept complete") {
		t.Errorf("context to verify: hint %q", h)
	}
	r.Document.Defaults.Confirmed = true
	if h := refusalHint(r, []int{3}, false); !strings.Contains(h, "accept complete") {
		t.Errorf("confirmed defaults: hint %q", h)
	}
	// Under the defaults, the position carries them.
	r.Entries[0].Position = "XGID=-b----E-C---eE---c-e----B-:1:1:1:00:0:0:0:0:10"
	if err := r.Validate(); !errors.Is(err, ErrInvalid) {
		t.Errorf("an entry under the defaults with another cube: err = %v", err)
	}
}
