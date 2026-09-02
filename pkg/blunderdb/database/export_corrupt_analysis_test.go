package database

import (
	"path/filepath"
	"strings"
	"testing"
)

// corruptAnalysisBlobs are stored analysis payloads that cannot be decoded,
// one per failure point of engine.DecodeAnalysisFromStorage: a zlib stream whose
// body is garbage (decompression fails) and a payload that looks like JSON
// but is not (the unmarshal fails).
var corruptAnalysisBlobs = map[string][]byte{
	"truncated zlib": {0x78, 0x9c, 0x01, 0x02, 0x03},
	"invalid JSON":   []byte(`{"xgid": "half-written`),
}

// TestExport_CorruptAnalysisIsSkipped injects an undecodable analysis for one
// of the two positions and checks what the export produces: the position
// itself still travels, the good analysis travels, and nothing — no empty
// row, no nil-derived JSON — is written for the corrupt one, and the failure
// is logged where the batch was decoded (AnalysisStore.LoadMany). That is the
// outcome that lets the recipient import the file.
func TestExport_CorruptAnalysisIsSkipped(t *testing.T) {
	for name, corrupt := range corruptAnalysisBlobs {
		t.Run(name, func(t *testing.T) {
			db, dir, cleanup := setupExportTestDB(t)
			defer cleanup()

			positions := allPositions(t, db)
			if len(positions) != 2 {
				t.Fatalf("fixture has %d positions, want 2", len(positions))
			}
			corruptID := positions[0].ID
			if _, err := db.db.Exec(`UPDATE analysis SET data = ? WHERE position_id = ?`, corrupt, corruptID); err != nil {
				t.Fatalf("corrupting analysis %d: %v", corruptID, err)
			}

			logs := captureSlog(t)
			exportPath := filepath.Join(dir, "export.db")
			doExport(t, db, exportPath, defaultExportParams())

			edb := openExportDB(t, exportPath)
			defer edb.Close()

			if n := countRows(t, edb, "position"); n != 2 {
				t.Errorf("position rows = %d, want 2 (a bad analysis must not drop its position)", n)
			}
			if n := countRows(t, edb, "analysis"); n != 1 {
				t.Errorf("analysis rows = %d, want 1 (the corrupt one is skipped, not written empty)", n)
			}
			var orphans int
			if err := edb.QueryRow(`SELECT COUNT(*) FROM analysis a LEFT JOIN position p ON p.id = a.position_id WHERE p.id IS NULL`).Scan(&orphans); err != nil {
				t.Fatal(err)
			}
			if orphans != 0 {
				t.Errorf("%d analysis rows point at no exported position", orphans)
			}
			// loadExportAnalysis fails on a row that is not valid JSON.
			analyses := loadExportAnalysis(t, edb)
			if len(analyses) != 1 || analyses[0].XGID != "test-xgid-2" {
				t.Errorf("exported analyses = %+v, want only test-xgid-2", analyses)
			}
			if !strings.Contains(logs.String(), "decoding stored analysis") {
				t.Errorf("the skipped analysis must be logged:\n%s", logs.String())
			}
		})
	}
}
