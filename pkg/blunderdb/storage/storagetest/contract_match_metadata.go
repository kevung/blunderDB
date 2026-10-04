// Contract case for the source metadata of a match: what the file says of
// the players and the session is stored and read back, NULL stays unknown,
// and ReplaceHeader rewrites it.
package storagetest

import (
	"context"
	"reflect"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func ptr[T any](v T) *T { return &v }

func testMatchSourceMetadata(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()

	full := domain.Match{Player1Name: "Alice", Player2Name: "Bob", MatchLength: 0}
	full.Player1Elo, full.Player2Elo = ptr(1854.5), ptr(1600.0)
	full.Player1Experience, full.Player2Experience = ptr(205), ptr(0)
	full.Transcriber = "Carol"
	full.HasJacoby, full.HasBeaver = ptr(true), ptr(false)
	full.EngineVersion = "eXtreme Gammon, file format 30"
	id, err := ms.Save(ctx, "", &full)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ms.Get(ctx, "", id)
	if err != nil {
		t.Fatal(err)
	}
	assertSourceMetadata(t, "saved", got, &full)

	bare := domain.Match{Player1Name: "Dave", Player2Name: "Erin", MatchLength: 5}
	bareID, err := ms.Save(ctx, "", &bare)
	if err != nil {
		t.Fatal(err)
	}
	got, err = ms.Get(ctx, "", bareID)
	if err != nil {
		t.Fatal(err)
	}
	// Unknown is not zero: a file that says nothing leaves every pointer nil.
	assertSourceMetadata(t, "bare", got, &bare)

	next := *got
	next.Player1Elo, next.HasBeaver, next.Transcriber = ptr(1700.25), ptr(true), "Frank"
	if err := ms.ReplaceHeader(ctx, "", bareID, &next); err != nil {
		t.Fatal(err)
	}
	got, err = ms.Get(ctx, "", bareID)
	if err != nil {
		t.Fatal(err)
	}
	assertSourceMetadata(t, "replaced", got, &next)

	// A match comment keeps its signature, and an edit signs it anew.
	signed := domain.Match{Player1Name: "Gina", Player2Name: "Hugo", MatchLength: 3,
		Comment: "From the file", CommentAuthor: "Carol"}
	signedID, err := ms.Save(ctx, "", &signed)
	if err != nil {
		t.Fatal(err)
	}
	if got, err = ms.Get(ctx, "", signedID); err != nil || got.Comment != "From the file" || got.CommentAuthor != "Carol" {
		t.Fatalf("saved comment = %q by %q, %v; want %q by %q", got.Comment, got.CommentAuthor, err, "From the file", "Carol")
	}
	if err := ms.UpdateComment(storage.WithCommentAuthor(ctx, "Ivy"), "", signedID, "Edited"); err != nil {
		t.Fatal(err)
	}
	if got, err = ms.Get(ctx, "", signedID); err != nil || got.Comment != "Edited" || got.CommentAuthor != "Ivy" {
		t.Errorf("edited comment = %q by %q, %v; want %q by %q", got.Comment, got.CommentAuthor, err, "Edited", "Ivy")
	}
}

func assertSourceMetadata(t *testing.T, label string, got, want *domain.Match) {
	t.Helper()
	pairs := []struct {
		name      string
		got, want any
	}{
		{"player1_elo", got.Player1Elo, want.Player1Elo},
		{"player2_elo", got.Player2Elo, want.Player2Elo},
		{"player1_experience", got.Player1Experience, want.Player1Experience},
		{"player2_experience", got.Player2Experience, want.Player2Experience},
		{"transcriber", got.Transcriber, want.Transcriber},
		{"has_jacoby", got.HasJacoby, want.HasJacoby},
		{"has_beaver", got.HasBeaver, want.HasBeaver},
		{"engine_version", got.EngineVersion, want.EngineVersion},
	}
	for _, p := range pairs {
		if !reflect.DeepEqual(p.got, p.want) {
			t.Errorf("%s: %s = %v, want %v", label, p.name, deref(p.got), deref(p.want))
		}
	}
}

func deref(v any) any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "<nil>"
		}
		return rv.Elem().Interface()
	}
	return v
}

// The match list's player filter reads a person, not a spelling: every
// alias of the named player, and its canonical, select their matches.
func testMatchListPlayerAliases(t *testing.T, s storage.Storage) {
	ctx := context.Background()
	ms := s.Matches()
	for _, m := range []domain.Match{
		{Player1Name: "Alice Martin", Player2Name: "Bob"},
		{Player1Name: "Carol", Player2Name: "Martin A."},
		{Player1Name: "Dave", Player2Name: "Erin"},
	} {
		if _, err := ms.Save(ctx, "", &m); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Aliases().Set(ctx, "", storage.AliasPlayer, "Martin A.", "Alice Martin"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Alice Martin", "Martin A."} {
		opts := storage.MatchListOpts{PlayerName: name}
		n, err := ms.Count(ctx, "", opts)
		if err != nil {
			t.Fatal(err)
		}
		listed := 0
		for _, err := range ms.List(ctx, "", opts) {
			if err != nil {
				t.Fatal(err)
			}
			listed++
		}
		if n != 2 || listed != 2 {
			t.Errorf("player %q: count %d, listed %d, want both spellings' 2 matches", name, n, listed)
		}
	}
	if n, err := ms.Count(ctx, "", storage.MatchListOpts{PlayerName: "Dave"}); err != nil || n != 1 {
		t.Errorf("unaliased player: count %d, %v, want 1", n, err)
	}
}
