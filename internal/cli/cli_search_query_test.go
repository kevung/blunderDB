package cli

// cli_search_query_test.go — `blunderdb search --query`, the flag that hands
// the command line the same query language the application's command bar
// speaks (B.18, #186). The twenty-four filter flags covered twenty-four of the
// forty-five filter fields; the query covers every one a user can type.

import (
	"strings"
	"testing"
)

func TestSearchQueryFillsFiltersNoFlagCanReach(t *testing.T) {
	t.Parallel()
	params, dbPath, err := parseSearchFlags([]string{
		"--db", "database.db",
		"--query", `s cube nc M p>30 E>50 m"13/11" t"blunder" pl"Alice" T>2026/01/01 xD65 ma3 id7`,
	})
	if err != nil {
		t.Fatalf("parseSearchFlags: %v", err)
	}
	if dbPath != "database.db" {
		t.Errorf("dbPath = %q", dbPath)
	}
	f := params.filters
	for _, tc := range []struct {
		name string
		got  any
		want any
	}{
		{"IncludeCube", f.IncludeCube, true},
		{"NoContactFilter", f.NoContactFilter, true},
		{"MirrorFilter", f.MirrorFilter, true},
		{"PipCountFilter", f.PipCountFilter, "p>30"},
		{"MoveErrorFilter", f.MoveErrorFilter, "E>50"},
		{"MovePatternFilter", f.MovePatternFilter, `m"13/11"`},
		{"SearchText", f.SearchText, `t"blunder"`},
		{"PlayerFilter", f.PlayerFilter, `pl"Alice"`},
		{"DateFilter", f.DateFilter, "T>2026/01/01"},
		{"ExceptDiceFilter", f.ExceptDiceFilter, "65"},
		{"MatchIDsFilter", f.MatchIDsFilter, "3"},
		{"PositionIDsFilter", f.PositionIDsFilter, "7"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}

func TestSearchQueryRefusesUnknownTokens(t *testing.T) {
	t.Parallel()
	_, _, err := parseSearchFlags([]string{"--db", "database.db", "--query", "s cube nosuchtoken"})
	if err == nil {
		t.Fatal("expected an error for an unknown token")
	}
	if !strings.Contains(err.Error(), "nosuchtoken") {
		t.Errorf("error %q does not name the offending token", err)
	}
}

// The query and the filter flags are two spellings of the same thing, and a
// precedence rule between them would be a rule nobody could remember. Refusing
// the combination is the honest answer; a flag that says where to search or how
// to print is not a filter and stays compatible.
func TestSearchQueryRefusesFilterFlagsButAcceptsOutputFlags(t *testing.T) {
	t.Parallel()
	_, _, err := parseSearchFlags([]string{"--db", "database.db", "--query", "s cube", "--individual"})
	if err == nil || !strings.Contains(err.Error(), "--individual") {
		t.Fatalf("expected a refusal naming --individual, got %v", err)
	}

	params, _, err := parseSearchFlags([]string{
		"--db", "database.db", "--query", "s cube",
		"--format", "json", "--limit", "5", "--offset", "2", "--export", "out.db",
	})
	if err != nil {
		t.Fatalf("output flags should stay compatible with --query: %v", err)
	}
	if params.format != "json" || params.limit != 5 || params.offset != 2 || params.outputDB != "out.db" {
		t.Errorf("output flags lost: %+v", params)
	}
}

// A token the grammar understands but cannot act on here (`x`, the exclusion
// structure, which is a board) is carried out as a diagnostic so runSearch can
// say so, rather than narrowing nothing in silence.
func TestSearchQueryCarriesNoEffectDiagnostics(t *testing.T) {
	t.Parallel()
	params, _, err := parseSearchFlags([]string{"--db", "database.db", "--query", "s x"})
	if err != nil {
		t.Fatalf("parseSearchFlags: %v", err)
	}
	if len(params.diags) != 1 || params.diags[0].Token != "x" {
		t.Fatalf("diags = %+v, want one entry for `x`", params.diags)
	}
}

// --query-help must not need a database: it is documentation, and asking for it
// without --db is the natural thing to do.
func TestSearchQueryHelpNeedsNoDatabase(t *testing.T) {
	t.Parallel()
	params, _, err := parseSearchFlags([]string{"--query-help"})
	if err != nil {
		t.Fatalf("--query-help without --db: %v", err)
	}
	if !params.queryHelp {
		t.Error("queryHelp not set")
	}
}
