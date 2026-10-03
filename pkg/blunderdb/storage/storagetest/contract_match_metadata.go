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
