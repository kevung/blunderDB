package database

import (
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/duel"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// TestTranscription_SurvivesVacuum: Vacuum closes the *sql.DB and opens a new
// one; the transcription service must follow it, not keep the closed handle.
func TestTranscription_SurvivesVacuum(t *testing.T) {
	t.Parallel()
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(t.TempDir(), "t.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer d.Close()

	if _, err := d.CreateTranscription(transcript.Header{MatchLength: 7}); err != nil {
		t.Fatalf("CreateTranscription: %v", err)
	}
	if _, err := d.Vacuum(); err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	rows, err := d.ListTranscriptions()
	if err != nil {
		t.Fatalf("ListTranscriptions after Vacuum: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d drafts after Vacuum, want 1", len(rows))
	}
	if _, err := d.CreateTranscription(transcript.Header{MatchLength: 5}); err != nil {
		t.Fatalf("CreateTranscription after Vacuum: %v", err)
	}
}

// TestDuel_SurvivesVacuum: the Arbiter follows the new handle too, and the
// Duel in progress stays open across it.
func TestDuel_SurvivesVacuum(t *testing.T) {
	t.Parallel()
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(t.TempDir(), "t.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer d.Close()

	sides := [2]duel.SideSpec{{Kind: duel.SideExternal, Name: "A"}, {Kind: duel.SideExternal, Name: "B"}}
	st, err := d.CreateDuel(duel.Settings{MatchLength: 3, Sides: sides})
	if err != nil {
		t.Fatalf("CreateDuel: %v", err)
	}
	if _, err := d.Vacuum(); err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
	list, err := d.ListDuels()
	if err != nil {
		t.Fatalf("ListDuels after Vacuum: %v", err)
	}
	if len(list) != 1 || list[0].ID != st.State.ID || !list[0].Open {
		t.Fatalf("after Vacuum: %+v, want Duel %d still open", list, st.State.ID)
	}
	if _, err := d.OpenDuel(st.State.ID); err != nil {
		t.Fatalf("OpenDuel after Vacuum: %v", err)
	}
}

// TestServices_DroppedOnReplacedStore: a service made on a store that has
// since been replaced is not handed out again, so it cannot reach a closed
// handle nor answer with the previous library's sessions.
func TestServices_DroppedOnReplacedStore(t *testing.T) {
	t.Parallel()
	d := NewDatabase()
	if err := d.SetupDatabase(filepath.Join(t.TempDir(), "t.db")); err != nil {
		t.Fatalf("SetupDatabase: %v", err)
	}
	defer d.Close()

	d.mu.Lock()
	ts, ds := d.transcriptService(), d.duelService()
	d.rebuildStore()
	ts2, ds2 := d.transcriptService(), d.duelService()
	d.mu.Unlock()
	if ts2 == ts || ds2 == ds {
		t.Fatalf("services survived a replaced store: transcription %v, duel %v", ts2 == ts, ds2 == ds)
	}
}
