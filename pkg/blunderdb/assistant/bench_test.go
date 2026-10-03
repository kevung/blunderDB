//go:build assistantbench

package assistant_test

// The assistant's measurement: every corpus sentence goes to a real model
// through the real tools on the demo database, and the model is scored on
// what it did, not on what it said. Run against a provider of your choice:
//
//	BLUNDERDB_BENCH_MODEL=qwen2.5:7b go test -tags assistantbench -run TestBench -v -timeout 2h ./pkg/blunderdb/assistant/
//
// BLUNDERDB_BENCH_BASEURL defaults to a local Ollama; BLUNDERDB_BENCH_KEY is
// the API key of a remote provider. The recommendation in the manual comes
// from this score.

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/assistant"
	"github.com/kevung/blunderdb/pkg/blunderdb/mcp"
)

type recordingDisplay struct{ queries []string }

func (d *recordingDisplay) OpenView(_, query string) error {
	d.queries = append(d.queries, query)
	return nil
}
func (d *recordingDisplay) ShowPosition(int64) error { return nil }

// scoreEntry says whether the model did what the sentence asked: for a
// search, a view opened on a query of the expected canonical form; otherwise,
// the expected tool called.
func scoreEntry(e corpusEntry, turn assistant.Turn, d *recordingDisplay) bool {
	if e.Query != "" {
		want := canonical(e.Query)
		for _, q := range d.queries {
			if canonical(q) == want {
				return true
			}
		}
		return false
	}
	for _, en := range turn.Entries {
		if en.Kind == assistant.KindTool && en.Tool == e.Tool {
			return true
		}
	}
	return false
}

func TestBench(t *testing.T) {
	model := os.Getenv("BLUNDERDB_BENCH_MODEL")
	if model == "" {
		t.Skip("set BLUNDERDB_BENCH_MODEL (and BLUNDERDB_BENCH_BASEURL, BLUNDERDB_BENCH_KEY for a remote provider)")
	}
	base := os.Getenv("BLUNDERDB_BENCH_BASEURL")
	if base == "" {
		base = assistant.PresetByID("ollama").BaseURL
	}
	p := assistant.Provider{BaseURL: base, Model: model, APIKey: os.Getenv("BLUNDERDB_BENCH_KEY")}
	engine := demoEngine(t)
	ok, total := map[string]int{}, map[string]int{}
	var elapsed time.Duration
	for _, e := range loadCorpus(t) {
		d := &recordingDisplay{}
		ctx := context.Background()
		cs, err := assistant.Connect(ctx, mcp.NewServer(engine, mcp.Options{Tenant: "1",
			Extensions: []mcp.Extension{mcp.DisplayTools(d)}}))
		if err != nil {
			t.Fatal(err)
		}
		sess, err := assistant.NewSession(ctx, p, cs)
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		turn, err := sess.Ask(ctx, e.Text)
		elapsed += time.Since(start)
		sess.Close()
		if err != nil {
			t.Fatalf("%s/%s: %v", e.Lang, e.Intent, err)
		}
		total[e.Lang]++
		if scoreEntry(e, turn, d) {
			ok[e.Lang]++
		} else {
			t.Logf("miss %s/%s: views %v", e.Lang, e.Intent, d.queries)
		}
	}
	langs := make([]string, 0, len(total))
	for l := range total {
		langs = append(langs, l)
	}
	sort.Strings(langs)
	var b strings.Builder
	sumOK, sum := 0, 0
	for _, l := range langs {
		fmt.Fprintf(&b, " %s %d/%d", l, ok[l], total[l])
		sumOK += ok[l]
		sum += total[l]
	}
	t.Logf("model %s: %d/%d (%.0f%%), %v per sentence;%s", model, sumOK, sum,
		100*float64(sumOK)/float64(sum), (elapsed / time.Duration(sum)).Round(time.Millisecond), b.String())
}
