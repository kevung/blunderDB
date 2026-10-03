package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

func TestLabelsMatchTheFrontendCatalogues(t *testing.T) {
	for _, lang := range Languages() {
		blob, err := os.ReadFile(filepath.Join("..", "..", "..", "frontend", "src", "i18n", "locales", lang+".json"))
		if err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		var cat struct {
			Report map[string]string `json:"report"`
		}
		if err := json.Unmarshal(blob, &cat); err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		for key, want := range labels[lang] {
			if cat.Report[key] != want {
				t.Errorf("%s report.%s = %q in the catalogue, %q in labels.go", lang, key, cat.Report[key], want)
			}
		}
	}
}

func TestRenderEscapesImportedTextAndKeepsTheDiagram(t *testing.T) {
	p := domain.InitializePosition()
	out := Render(Data{
		Language:  "fr",
		Generated: time.Date(2026, 1, 2, 3, 4, 0, 0, time.UTC),
		Blunders:  []Blunder{{Rank: 1, ErrorMP: 123456, Players: "<b>x</b> & y", Date: "2025-05-06 10:00", Diagram: Diagram(&p), BestMove: "8/5 6/5"}},
	})
	for _, want := range []string{`lang="fr"`, "&lt;b&gt;x&lt;/b&gt; &amp; y", "2025-05-06", "123.456", "<svg", "8/5 6/5", "Établi le 2026-01-02 03:04"} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	if strings.Contains(out, "<script") || strings.Contains(out, "http://") && !strings.Contains(out, "xmlns=\"http://www.w3.org/2000/svg\"") {
		t.Error("report is not self-contained")
	}
}

func TestRenderFallsBackToEnglish(t *testing.T) {
	if out := Render(Data{Language: "xx"}); !strings.Contains(out, "blunderDB report") {
		t.Error("unknown language did not fall back to English")
	}
}
