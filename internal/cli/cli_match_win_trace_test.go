package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

// `match` carries player 1's match winning chances: the json output under
// win_trace, the text output as an "MWC" line on each analysed decision.
func TestMatchOutputsCarryWinTrace(t *testing.T) {
	cli, _ := setupCLIWithDB(t)
	id, err := cli.db.ImportXGMatch(testdataPath("test.xg"))
	if err != nil {
		t.Fatalf("ImportXGMatch: %v", err)
	}
	match, err := cli.db.GetMatchByID(id)
	if err != nil {
		t.Fatalf("GetMatchByID: %v", err)
	}
	positions, err := cli.db.GetMatchMovePositions(id)
	if err != nil {
		t.Fatalf("GetMatchMovePositions: %v", err)
	}
	out, err := cli.formatMatchJSON(match, positions, nil)
	if err != nil {
		t.Fatalf("formatMatchJSON: %v", err)
	}
	var doc struct {
		WinTrace struct {
			Money     bool `json:"money"`
			Decisions []struct {
				MWCBefore *float64 `json:"mwc_before"`
			} `json:"decisions"`
		} `json:"win_trace"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json: %v", err)
	}
	if doc.WinTrace.Money || len(doc.WinTrace.Decisions) == 0 || doc.WinTrace.Decisions[0].MWCBefore == nil {
		t.Errorf("win_trace missing or empty: %+v", doc.WinTrace)
	}
	text, err := cli.formatMatchText(match, positions, nil)
	if err != nil {
		t.Fatalf("formatMatchText: %v", err)
	}
	if want := "  MWC (" + match.Player1Name + "): "; !strings.Contains(text, want) || !strings.Contains(text, "% -> ") {
		t.Errorf("text lacks the MWC line %q", want)
	}
}
