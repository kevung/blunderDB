package printsheet

import (
	"encoding/json"
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
		// ArkID, then game·row player and analysis in their own aligned cell.
		`<span class="nt">[77]31:(ySQmLhfA)</span><div><span class="wh">`,
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

// TestBodySizes: the scale the fitting script walks, from the Go constants,
// is the one the browser receives.
func TestBodySizes(t *testing.T) {
	got := BodySizes()
	if len(got) != 15 || got[0] != MaxBodyPt || got[len(got)-1] != MinBodyPt {
		t.Fatalf("BodySizes = %v, want 9 down to 5.5 by 0.25", got)
	}
	for i := 1; i < len(got); i++ {
		if d := got[i-1] - got[i]; d != StepBodyPt {
			t.Fatalf("step %d = %v, want %v", i, d, StepBodyPt)
		}
	}
	page := string(Document("en", "t", nil, ""))
	if want := "var SIZES = [9, 8.75, 8.5, 8.25, 8, 7.75, 7.5, 7.25, 7, 6.75, 6.5, 6.25, 6, 5.75, 5.5]"; !strings.Contains(page, want) {
		t.Errorf("script lacks %q", want)
	}
	if strings.Contains(page, "/*SIZES*/") {
		t.Error("placeholder left in the script")
	}
}

// TestMatchOptionsJSON: an object naming some choices keeps the defaults of
// the others.
func TestMatchOptionsJSON(t *testing.T) {
	var o MatchOptions
	if err := json.Unmarshal([]byte(`{"expanded":true,"comments":false}`), &o); err != nil {
		t.Fatal(err)
	}
	want := DefaultMatchOptions()
	want.Expanded, want.Comments = true, false
	if o != want {
		t.Errorf("got %+v, want %+v", o, want)
	}
}
