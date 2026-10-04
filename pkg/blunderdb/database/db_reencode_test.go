package database

import (
	"path/filepath"
	"testing"
)

// TestReencodeAnalyses_RewritesLegacyRows: a raw-JSON blob, as a release
// before the binary format left it, is rewritten once, and a second run finds
// nothing left to do.
func TestReencodeAnalyses_RewritesLegacyRows(t *testing.T) {
	t.Parallel()
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(t.TempDir(), "reencode.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer d.Close()
	res, err := d.conn().Exec(`INSERT INTO position (state) VALUES ('legacy')`)
	if err != nil {
		t.Fatal(err)
	}
	posID, _ := res.LastInsertId()
	if _, err := d.conn().Exec(`INSERT INTO analysis (position_id, data) VALUES (?, ?)`,
		posID, []byte(`{"positionId":1,"xgid":"XGID=legacy"}`)); err != nil {
		t.Fatal(err)
	}

	got, err := d.ReencodeAnalyses()
	if err != nil || got.Rewritten != 1 {
		t.Fatalf("ReencodeAnalyses = %+v, %v; want 1 rewritten", got, err)
	}
	var data []byte
	if err := d.conn().QueryRow(`SELECT data FROM analysis WHERE position_id = ?`, posID).Scan(&data); err != nil {
		t.Fatal(err)
	}
	a, err := decodeAnalysisFromStorage(data)
	if err != nil || a.XGID != "XGID=legacy" || a.PositionID != 1 || data[0] == '{' {
		t.Fatalf("re-encoded row = %+v, %v (first byte %#x)", a, err, data[0])
	}
	if again, err := d.ReencodeAnalyses(); err != nil || again.Rewritten != 0 {
		t.Fatalf("second run = %+v, %v; want nothing", again, err)
	}
}
