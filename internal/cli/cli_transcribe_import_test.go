package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// --import opens a draft on a .mat, which --check then replays; a file that is
// neither a .mat nor a document fails the command and writes no draft.
func TestTranscribeImport(t *testing.T) {
	c, dbPath := setupCLIWithDB(t)
	c.db.Close()

	out, err := runTranscribeOnFile(t, "--db", dbPath, "--import", filepath.Join("testdata", "gnubg_selfplay_drops_7p.mat"), "--format", "json")
	if err != nil {
		t.Fatalf("--import: %v", err)
	}
	var res transcribeImportResult
	if err := json.Unmarshal([]byte(out), &res); err != nil || res.DraftID == 0 || res.Actions == 0 || res.Games == 0 {
		t.Fatalf("--import printed %q (%v)", out, err)
	}
	if _, err := runTranscribeOnFile(t, "--db", dbPath, "--draft", itoa64(res.DraftID), "--check"); err != nil {
		t.Fatalf("the imported draft does not replay: %v", err)
	}

	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte(`{"header":{"match_length":5},"actions":[{"side":9,"kind":"double"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := runTranscribeOnFile(t, "--db", dbPath, "--import", bad); err == nil {
		t.Fatal("an unreadable document was imported")
	}
	if _, err := runTranscribeOnFile(t, "--import", bad); err == nil {
		t.Fatal("--import without --db")
	}
}
