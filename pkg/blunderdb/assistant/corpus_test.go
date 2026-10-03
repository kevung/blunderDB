package assistant_test

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
)

// corpusEntry is one sentence of the measurement corpus: what a user writes,
// and what a good model does with it — the tool it calls and, for a search,
// the query whose canonical form the view must carry.
type corpusEntry struct {
	Lang   string `json:"lang"`
	Intent string `json:"intent"`
	Text   string `json:"text"`
	Tool   string `json:"tool"`
	Query  string `json:"query,omitempty"`
}

func loadCorpus(t testing.TB) []corpusEntry {
	t.Helper()
	raw, err := os.ReadFile("testdata/corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	var c []corpusEntry
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

// canonical is a query's normal form: two queries that select the same
// positions compare equal whatever their token order or spelling.
func canonical(q string) string {
	f, _ := searchquery.Parse(q)
	return searchquery.Format(f)
}

// The corpus covers the same intents in each of the nine languages of the
// application, and every expected query is one the grammar reads whole.
func TestCorpusCoversEveryLanguageAndParses(t *testing.T) {
	byLang := map[string][]string{}
	for _, e := range loadCorpus(t) {
		byLang[e.Lang] = append(byLang[e.Lang], e.Intent)
		if e.Query != "" {
			if _, diags := searchquery.Parse(e.Query); len(diags) > 0 || canonical(e.Query) == "" {
				t.Errorf("%s/%s: %q does not parse: %v", e.Lang, e.Intent, e.Query, diags)
			}
		}
		if e.Tool == "" || e.Text == "" {
			t.Errorf("%s/%s: incomplete entry", e.Lang, e.Intent)
		}
	}
	langs := []string{"de", "el", "en", "es", "fi", "fr", "it", "ja", "ru"}
	ref := byLang["fr"]
	sort.Strings(ref)
	for _, l := range langs {
		got := byLang[l]
		sort.Strings(got)
		if len(got) != len(ref) {
			t.Fatalf("%s: %d intents, want %d", l, len(got), len(ref))
		}
		for i := range got {
			if got[i] != ref[i] {
				t.Fatalf("%s: intents %v, want %v", l, got, ref)
			}
		}
	}
	if len(byLang) != len(langs) {
		t.Fatalf("languages %d, want %d", len(byLang), len(langs))
	}
}
