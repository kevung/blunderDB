package cli

import (
	"flag"
	"io"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
)

// A corpus flag and its token are one filter: the flags must yield the same
// fields the query language parses from the equivalent tokens.
func TestSearchCorpusFlagsMatchTheirTokens(t *testing.T) {
	cases := []struct {
		args  []string
		query string
	}{
		{[]string{"--player", "Alice*"}, `s pl"Alice*"`},
		{[]string{"--player", "Alice", "--seat-only"}, `s pl!"Alice"`},
		{[]string{"--player", "Alice", "--opponent", "Bob"}, `s pl"Alice" op"Bob"`},
		{[]string{"--tournament-name", "Monte*"}, `s tn"Monte*"`},
		{[]string{"--round", "final,semi*"}, `s rd:final rd:semi*`},
		{[]string{"--match-lengths", "7"}, `s ml:7`},
		{[]string{"--match-lengths", ">5"}, `s ml>5`},
		{[]string{"--away", "4"}, `s aw:4`},
		{[]string{"--away", "2,5"}, `s aw:2,5`},
		{[]string{"--away", "<5"}, `s aw<5`},
		{[]string{"--away", ">3"}, `s aw>3`},
		{[]string{"--opponent-away", "1"}, `s AW:1`},
		{[]string{"--away", "2", "--opponent-away", "<5"}, `s aw:2 AW<5`},
		{[]string{"--crawford", "game"}, `s cr`},
		{[]string{"--crawford", "post"}, `s pcr`},
		{[]string{"--match-date", "2024-01..2024-12"}, `s md:2024-01..2024-12`},
		{[]string{"--match-date", "<2024-06"}, `s md<2024-06`},
		{[]string{"--pr", ">8"}, `s pr>8`},
		{[]string{"--pr", "4,9"}, `s pr4,9`},
		{[]string{"--analysis", "XG,3ply+"}, `s ad:xg ad:3ply+`},
		{[]string{"--cube-response", "takepass"}, `s dr`},
	}
	for _, c := range cases {
		fs := flag.NewFlagSet("search", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := defineSearchFlags(fs)
		if err := fs.Parse(c.args); err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		got, err := f.corpusFilters()
		if err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		want, diags := searchquery.Parse(c.query)
		if len(diags) > 0 {
			t.Fatalf("%s: %v", c.query, diags)
		}
		pairs := [][3]string{
			{"player", got.PlayerFilter, want.PlayerFilter},
			{"opponent", got.OpponentFilter, want.OpponentFilter},
			{"tournament", got.TournamentNameFilter, want.TournamentNameFilter},
			{"round", got.RoundFilter, want.RoundFilter},
			{"length", got.MatchLengthFilter, want.MatchLengthFilter},
			{"away", got.AwayFilter, want.AwayFilter},
			{"opponent away", got.OpponentAwayFilter, want.OpponentAwayFilter},
			{"crawford", got.CrawfordFilter, want.CrawfordFilter},
			{"date", got.MatchDateFilter, want.MatchDateFilter},
			{"pr", got.PlayerPRFilter, want.PlayerPRFilter},
			{"analysis", got.AnalysisProvenanceFilter, want.AnalysisProvenanceFilter},
			{"cube response", got.CubeResponseFilter, want.CubeResponseFilter},
		}
		for _, p := range pairs {
			if p[1] != p[2] {
				t.Errorf("%v: %s = %q, token %q gives %q", c.args, p[0], p[1], c.query, p[2])
			}
		}
	}
}

func TestSearchCorpusFlagsRefuseNonsense(t *testing.T) {
	for _, args := range [][]string{
		{"--seat-only"},
		{"--cube-response", "beaver"},
		{"--crawford", "yes"},
		{"--analysis", "deep blue"},
	} {
		fs := flag.NewFlagSet("search", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		f := defineSearchFlags(fs)
		if err := fs.Parse(args); err != nil {
			t.Fatal(err)
		}
		if _, err := f.corpusFilters(); err == nil {
			t.Errorf("%v: accepted", args)
		}
	}
}
