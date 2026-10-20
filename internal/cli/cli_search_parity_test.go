package cli

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

var updateSearchParity = flag.Bool("update-search-parity", false,
	"rewrite testdata/search_parity.golden from the current search output")

const searchParityGolden = "testdata/search_parity.golden"

// searchParityCases are the searches whose whole output is pinned: every
// format, a window pushed into SQL and one applied after the client-side
// filters, and the move-error filter both unbounded and paged, since the
// two take different SQL paths.
var searchParityCases = []struct {
	name string
	args []string
}{
	{"all-json", []string{"--limit", "0", "--format", "json"}},
	{"all-table", []string{"--limit", "0"}},
	{"all-xgid", []string{"--limit", "0", "--format", "xgid"}},
	{"cube-json", []string{"--decision", "cube", "--limit", "0", "--format", "json"}},
	{"checker-page-json", []string{"--decision", "checker", "--limit", "40", "--offset", "25", "--format", "json"}},
	{"error-min-json", []string{"--error-min", "0.05", "--limit", "0", "--format", "json"}},
	{"error-min-page-table", []string{"--error-min", "0.02", "--limit", "15", "--offset", "10"}},
	{"has-analysis-xgid", []string{"--has-analysis", "--limit", "0", "--format", "xgid"}},
	{"E100-json", []string{"--query", "E>100", "--limit", "0", "--format", "json"}},
	{"E100-page-json", []string{"--query", "E>100", "--limit", "7", "--offset", "3", "--format", "json"}},
	{"E20-60-table", []string{"--query", "E20,60", "--limit", "0"}},
	{"E-below-table", []string{"--query", "E<10", "--limit", "0"}},
}

// TestSearchOutputParity holds the CLI search to the exact bytes it printed on
// a library of real matches: an optimisation of the search or of its
// rendering must not move one id, one line or one digit. The golden file
// keeps a digest per case (the outputs run to megabytes); on a mismatch the
// test prints the first lines of the new output to start the diagnosis.
func TestSearchOutputParity(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	for _, f := range []string{
		"charlot1-charlot2_7p_2025-11-08-2305.xg",
		"HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg",
		"Kev_GammonNetNormal_2026-10-06_7p.xg",
		"match_with_comment.xg",
	} {
		if err := cli.Run([]string{"import", "--db", dbPath, "--type", "match", "--file", testdataPath(f)}); err != nil {
			t.Fatalf("import %s: %v", f, err)
		}
	}

	got := map[string]string{}
	outputs := map[string]string{}
	for _, c := range searchParityCases {
		var runErr error
		out := captureStdout(t, func() {
			runErr = cli.Run(append([]string{"search", "--db", dbPath}, c.args...))
		})
		if runErr != nil {
			t.Fatalf("%s: %v", c.name, runErr)
		}
		if len(out) < 100 {
			t.Fatalf("%s: suspiciously short output %q", c.name, out)
		}
		sum := sha256.Sum256([]byte(out))
		got[c.name] = hex.EncodeToString(sum[:])
		outputs[c.name] = out
	}

	if *updateSearchParity {
		names := make([]string, 0, len(got))
		for n := range got {
			names = append(names, n)
		}
		sort.Strings(names)
		var b strings.Builder
		for _, n := range names {
			fmt.Fprintf(&b, "%s %s\n", n, got[n])
		}
		if err := os.WriteFile(searchParityGolden, []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}

	want := readSearchParityGolden(t)
	for _, c := range searchParityCases {
		if want[c.name] == "" {
			t.Errorf("%s: no digest in %s (run with -update-search-parity)", c.name, searchParityGolden)
			continue
		}
		if got[c.name] != want[c.name] {
			out := outputs[c.name]
			if len(out) > 2000 {
				out = out[:2000]
			}
			t.Errorf("%s: output changed (sha256 %s, want %s); begins:\n%s", c.name, got[c.name], want[c.name], out)
		}
	}
}

func readSearchParityGolden(t *testing.T) map[string]string {
	t.Helper()
	f, err := os.Open(searchParityGolden)
	if err != nil {
		t.Fatalf("open golden: %v", err)
	}
	defer f.Close()
	want := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		name, sum, ok := strings.Cut(sc.Text(), " ")
		if ok {
			want[name] = sum
		}
	}
	return want
}
