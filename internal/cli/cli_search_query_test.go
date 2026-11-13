package cli

// cli_search_query_test.go — `blunderdb search --query`, the command bar's
// query language on the command line; it covers every filter field, where
// the flags cover only some.

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
)

func TestSearchQueryFillsFiltersNoFlagCanReach(t *testing.T) {
	t.Parallel()
	params, dbPath, err := (&CLI{}).parseSearchFlags([]string{
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
	_, _, err := (&CLI{}).parseSearchFlags([]string{"--db", "database.db", "--query", "s cube nosuchtoken"})
	if err == nil {
		t.Fatal("expected an error for an unknown token")
	}
	if !strings.Contains(err.Error(), "nosuchtoken") {
		t.Errorf("error %q does not name the offending token", err)
	}
}

func TestSearchQueryRefusesTheStarredLike(t *testing.T) {
	t.Parallel()
	_, _, err := (&CLI{}).parseSearchFlags([]string{"--db", "database.db", "--query", "s like42<12* nc"})
	if !errors.Is(err, searchquery.ErrRefused) || !strings.Contains(err.Error(), "like42<12*") {
		t.Errorf("starred like: err = %v, want a refusal naming the token", err)
	}
}

// The query and the filter flags are two spellings of the same thing, so the
// combination is refused rather than given a precedence rule. Flags that say
// where to search or how to print are not filters and stay compatible.
func TestSearchQueryRefusesFilterFlagsButAcceptsOutputFlags(t *testing.T) {
	t.Parallel()
	_, _, err := (&CLI{}).parseSearchFlags([]string{"--db", "database.db", "--query", "s cube", "--individual"})
	if err == nil || !strings.Contains(err.Error(), "--individual") {
		t.Fatalf("expected a refusal naming --individual, got %v", err)
	}

	params, _, err := (&CLI{}).parseSearchFlags([]string{
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
	params, _, err := (&CLI{}).parseSearchFlags([]string{"--db", "database.db", "--query", "s x"})
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
	params, _, err := (&CLI{}).parseSearchFlags([]string{"--query-help"})
	if err != nil {
		t.Fatalf("--query-help without --db: %v", err)
	}
	if !params.queryHelp {
		t.Error("queryHelp not set")
	}
}

// --cursor pages the search by the storage's cursor: following the printed
// cursor from the first page to the last reads the whole result once, in order.
func TestSearchCursorWalksTheWholeResult(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath("match_with_comment.xg")}); err != nil {
		t.Fatal(err)
	}
	search := func(args ...string) (out, status string, err error) {
		f, ferr := os.CreateTemp(t.TempDir(), "stderr")
		if ferr != nil {
			t.Fatal(ferr)
		}
		prev := os.Stderr
		os.Stderr = f
		out = captureOut(t, cli, func() {
			err = cli.Run(append([]string{"search", "--db", dbPath, "--format", "xgid"}, args...))
		})
		os.Stderr = prev
		raw, _ := os.ReadFile(f.Name())
		_ = f.Close()
		return out, string(raw), err
	}
	all, _, err := search("--limit", "0")
	if err != nil || strings.Count(all, "\n") < 7 {
		t.Fatalf("whole result: %d lines, %v", strings.Count(all, "\n"), err)
	}
	var walked strings.Builder
	cursor := ""
	for page := 0; ; page++ {
		if page > 1000 {
			t.Fatal("the cursor never ends")
		}
		out, status, err := search("--limit", "3", "--cursor", cursor)
		if err != nil {
			t.Fatal(err)
		}
		walked.WriteString(out)
		_, after, ok := strings.Cut(status, "Next cursor: ")
		if !ok {
			t.Fatalf("no cursor printed: %q", status)
		}
		if cursor, _, _ = strings.Cut(after, "\n"); cursor == "" {
			break
		}
	}
	if walked.String() != all {
		t.Errorf("pages by cursor:\n%s\nwhole result:\n%s", walked.String(), all)
	}
	// A cursor with `like` is refused, never served by the ranking with the
	// cursor dropped.
	for _, bad := range [][]string{{"--cursor", ""}, {"--limit", "3", "--offset", "2", "--cursor", ""}, {"--limit", "3", "--cursor", "", "--query", "s like1"}} {
		if _, _, err := search(bad...); err == nil {
			t.Errorf("%v: accepted", bad)
		}
	}
}
