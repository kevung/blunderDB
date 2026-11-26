package printsheet

import (
	"strings"
	"testing"
)

func TestLabelsComplete(t *testing.T) {
	for _, lang := range Languages() {
		c, ok := labels[lang]
		if !ok {
			t.Fatalf("no catalogue for %s", lang)
		}
		for k := range labels[DefaultLanguage] {
			if c[k] == "" {
				t.Errorf("%s lacks %q", lang, k)
			}
		}
	}
}

func TestRenderMatchMarks(t *testing.T) {
	cost := 213
	ms := &MatchSheet{
		Players: [2]string{"A <b>", "B"}, Length: 7, Plies: 3,
		Games: []SheetGame{{Number: 1, Away: [2]int{7, 7}, Winner: 1, Points: 2, Rows: [][2]Cell{
			{{Parts: []Part{{Dice: "31", Text: "8/5 6/5", Grade: GradeBlunder, Ref: 1, Kind: KindBlunder}}},
				{Parts: []Part{{Dice: "42", Text: "24/20 13/11", Grade: GradeError}}}},
			{{Parts: []Part{{Text: "Double", Cube: 2}}}, {Parts: []Part{{Text: "Take", Grade: GradeBlunder}}}},
		}}},
		Entries: []*Entry{{Ref: 1, Kind: KindBlunder, ArkID: "[77]31:(ySQmLhfA)", Game: 1, Row: 1, Played: "8/5 6/5", CostMP: &cost, Best: "13/10 6/5", Alt: "24/21 6/5", AltRank: 3, AltCostMP: 40}},
	}
	_, c := catalogue("fr")
	got, err := RenderMatch(ms, DefaultMatchOptions(), c)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`A &lt;b&gt; 0 – 0 B`,
		`<span class="bl">8/5 6/5</span><span class="g">??</span><span class="ref">1</span>`,
		`24/20 13/11<span class="g">?</span>`,
		`Double 2`,
		`<span class="bl">Prend</span><span class="g">??</span>`,
		`<div data-verso>`,
		`<span class="nt">[77]31:(ySQmLhfA)</span>`,
		`<b>8/5 6/5 −0.213</b> › 13/10 6/5 · 3. 24/21 6/5 −0.040`,
		`data-flow`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sheet lacks %q", want)
		}
	}
	// No code was asked for: none is drawn and the document embeds none.
	page := Document("fr", "t", []string{got}, "")
	if strings.Contains(string(page), "vnd.blunderdb.archive") {
		t.Error("an archive is embedded without a code")
	}
}
