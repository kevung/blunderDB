package database

import (
	"context"
	"path/filepath"
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

// Importing a database whose analyses were valued with a club table brings
// the table, merged by digest with the receiver's copy and not made current,
// and the analyses name the receiver's id for it (ADR-0068, rule 5).
func TestImportDatabaseCarriesTheMET(t *testing.T) {
	path := filepath.Join(t.TempDir(), "club.db")
	src := NewDatabase()
	if err := src.SetupDatabase(path); err != nil {
		t.Fatal(err)
	}
	srcRK, err := src.ImportMET("pkg/blunderdb/engine/testdata/met/Rockwell-Kazaross.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := src.SetCurrentMET(srcRK.ID); err != nil {
		t.Fatal(err)
	}
	var positions [2]Position
	for i := range positions {
		positions[i] = racePosition(8+i, 17, domain.White)
		positions[i].Dice = [2]int{0, 0}
		positions[i].Score = [2]int{3, 5}
		if _, err := src.SavePosition(&positions[i]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := src.AnalyzeMissingWithGammonNet(context.Background(), 0, 0, 0, 1, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := src.Close(); err != nil {
		t.Fatal(err)
	}

	d := newBatchTestDB(t)
	// An unrelated table first, so the receiver's id for Rockwell-Kazaross
	// is not the source's.
	if _, err := d.store.MatchEquityTables().Save(context.Background(), "", domain.MatchEquityTable{Name: "Other", Digest: "other", Source: "<met/>"}); err != nil {
		t.Fatal(err)
	}
	rk, err := d.ImportMET("pkg/blunderdb/engine/testdata/met/Rockwell-Kazaross.xml")
	if err != nil {
		t.Fatal(err)
	}
	// Position 1 is held without analysis: the imported one is inserted
	// beside it; position 0 is new.
	held := positions[1]
	heldID, err := d.SavePosition(&held)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := d.ImportDatabase(path); err != nil {
		t.Fatalf("ImportDatabase: %v", err)
	}

	tables, err := d.ListMETs()
	if err != nil || len(tables) != 3 || !tables[0].Current {
		t.Fatalf("ListMETs = %v, %v; want the built-in (current) and the two held tables", tables, err)
	}
	newID, err := d.SavePosition(&positions[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := d.SetCurrentMET(rk.ID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{newID, heldID} {
		st, err := d.AnalysisMETStatus(id)
		if err != nil {
			t.Fatal(err)
		}
		if st.Name != rk.Name || st.Different {
			t.Errorf("position %d: %+v, want valued with the receiver's %q", id, st, rk.Name)
		}
	}
}
