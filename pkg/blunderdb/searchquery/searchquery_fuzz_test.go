package searchquery

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

// FuzzParse holds the search grammar to its promise on arbitrary input: a
// command typed in the search bar, sent to search.query or passed to the CLI
// never panics the reader, and whatever it reads formats back into a command
// that reads to the same filters, from the first turn. The seeds are the
// shared corpus, so the fuzzer starts from every token the grammar knows.
//
// TestMain has moved to the repository root, so a crash input lands in
// testdata/fuzz/FuzzParse there.
func FuzzParse(f *testing.F) {
	raw, err := os.ReadFile("testdata/search_query_corpus.json")
	if err != nil {
		f.Fatalf("read corpus: %v", err)
	}
	var doc struct {
		Cases []struct {
			Command string `json:"command"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		f.Fatalf("parse corpus: %v", err)
	}
	for _, c := range doc.Cases {
		f.Add(c.Command)
	}
	f.Add(`s t"unterminated`)
	f.Add(`s #l'ouverture #don't #it's ma1;2 id0;`)
	f.Add(`ss c"a b" p>-1 p<99999999999999999999 n3,2 like7<2* E9 ##`)

	f.Fuzz(func(t *testing.T, command string) {
		_ = Tokenize(command)
		first, _ := Parse(command)
		formatted := Format(first)
		second, _ := Parse(formatted)
		if diff := fieldDiff(first, second); diff != "" {
			t.Fatalf("round trip changed the filters\n  command   %q\n  formatted %q\n%s", command, formatted, diff)
		}
	})
}

// fieldDiff names the SearchFilters fields that differ, so a failure prints
// the few that matter rather than two whole boards.
func fieldDiff(a, b any) string {
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	var out strings.Builder
	for i := 0; i < va.NumField(); i++ {
		x, y := va.Field(i).Interface(), vb.Field(i).Interface()
		if !reflect.DeepEqual(x, y) {
			fmt.Fprintf(&out, "  %s: %+v -> %+v\n", va.Type().Field(i).Name, x, y)
		}
	}
	return out.String()
}
