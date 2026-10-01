package database

import (
	"context"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// A draft written before version 3 opened each game with an opening Action. It
// is reopened converted: the tie and the opening are gone, the play that
// followed is the game's first and carries the score the tie declared.
func TestTranscription_Version2DraftReopensConverted(t *testing.T) {
	db := newTestDB(t)
	const doc = `{"format_version":2,"header":{"match_length":7},"actions":[
		{"side":0,"kind":"opening","dice":[4,4],"score":[1,0]},
		{"side":0,"kind":"opening","dice":[6,3]},
		{"side":0,"kind":"checker","dice":[6,3],"steps":[{"from":24,"to":18},{"from":13,"to":10}]}
	],"cursor":3}`
	id, err := db.store.Transcriptions().Save(context.Background(), "", &storage.Transcription{FormatVersion: "2", Document: doc})
	if err != nil {
		t.Fatalf("saving a version-2 draft: %v", err)
	}

	state, err := db.OpenTranscription(id)
	if err != nil {
		t.Fatalf("OpenTranscription: %v", err)
	}
	ann := state.Annotated
	if ann.Document.FormatVersion != transcript.FormatVersion {
		t.Fatalf("format version = %d, want %d", ann.Document.FormatVersion, transcript.FormatVersion)
	}
	actions := ann.Document.Actions
	if len(actions) != 1 || actions[0].Kind != transcript.KindChecker || actions[0].Side != 0 {
		t.Fatalf("actions = %+v, want player 1's first play alone", actions)
	}
	if actions[0].Score == nil || *actions[0].Score != [2]int{1, 0} {
		t.Fatalf("score = %v, want the 1-0 declared on the tie", actions[0].Score)
	}
	if !ann.Actions[0].OpensGame || len(ann.Games) != 1 || ann.Games[0].InitialScore != [2]int{1, 0} {
		t.Fatalf("game = %+v, want one game played at 1-0", ann.Games)
	}
	if ann.Document.Cursor != 1 {
		t.Errorf("cursor = %d, want the end", ann.Document.Cursor)
	}
}
