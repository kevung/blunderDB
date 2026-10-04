package database

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// A batch values a match score with the library's current table and records
// it; changing the current table rewrites nothing and turns the analysis
// "MET différente". The gnubg copy of Kazaross-XG2 is the built-in table.
func TestGammonNetBatchRecordsTheCurrentMET(t *testing.T) {
	d := newBatchTestDB(t)

	builtin, err := d.ImportMET("pkg/blunderdb/engine/testdata/met/Kazaross-XG2.xml")
	if err != nil {
		t.Fatalf("ImportMET(Kazaross-XG2): %v", err)
	}
	if builtin.ID != 0 {
		t.Errorf("gnubg's Kazaross-XG2 stored as table %d, want the built-in one", builtin.ID)
	}
	rk, err := d.ImportMET("pkg/blunderdb/engine/testdata/met/Rockwell-Kazaross.xml")
	if err != nil || rk.ID == 0 {
		t.Fatalf("ImportMET(Rockwell-Kazaross) = %+v, %v", rk, err)
	}
	if err := d.SetCurrentMET(rk.ID); err != nil {
		t.Fatal(err)
	}

	match := racePosition(8, 17, domain.White)
	match.Dice = [2]int{0, 0}
	match.Score = [2]int{3, 5}
	matchID, err := d.SavePosition(&match)
	if err != nil {
		t.Fatal(err)
	}
	money := racePosition(9, 16, domain.White)
	money.Dice = [2]int{0, 0}
	money.Score = [2]int{-1, -1}
	moneyID, err := d.SavePosition(&money)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := d.AnalyzeMissingWithGammonNet(context.Background(), 0, 0, 0, 1, nil, nil); err != nil {
		t.Fatal(err)
	}
	st, err := d.AnalysisMETStatus(matchID)
	if err != nil {
		t.Fatal(err)
	}
	if st.Name != rk.Name || st.Different {
		t.Fatalf("after the batch: %+v, want valued with %q and not different", st, rk.Name)
	}

	if err := d.SetCurrentMET(0); err != nil {
		t.Fatal(err)
	}
	if st, _ = d.AnalysisMETStatus(matchID); !st.Different || st.Current != "Kazaross-XG2" {
		t.Errorf("built-in current: %+v, want different", st)
	}
	if st, _ = d.AnalysisMETStatus(moneyID); st.Different {
		t.Errorf("money analysis reported different: %+v", st)
	}
	tables, err := d.ListMETs()
	if err != nil || len(tables) != 2 || !tables[0].Current {
		t.Errorf("ListMETs = %v, %v; want the built-in (current) then Rockwell-Kazaross", tables, err)
	}
}
