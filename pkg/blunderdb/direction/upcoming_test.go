package direction

import (
	"strings"
	"testing"
)

// The announced date goes under the title, in the host's words, and escaped: it is what the
// director typed.
func TestAnnounceUsesTheCatalogue(t *testing.T) {
	cat, err := NewCatalog([]byte(`{"term":{"announced":"Annoncée : {at}"}}`))
	if err != nil {
		t.Fatal(err)
	}
	got := announce("<h2>Feuille — Ronde 3</h2><table></table>", cat, " lundi <20 h> ")
	want := `<h2>Feuille — Ronde 3</h2><p class="resume">Annoncée : lundi &lt;20 h&gt;</p><table></table>`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestAnnounceWithoutCatalogueOrDate(t *testing.T) {
	if got := announce("<h2>R</h2>", nil, "lundi"); !strings.Contains(got, `<p class="resume">lundi</p>`) {
		t.Errorf("without a catalogue the date alone is printed: %s", got)
	}
	if got := announce("<h2>R</h2>", nil, "  "); got != "<h2>R</h2>" {
		t.Errorf("no date, no line: %s", got)
	}
}
