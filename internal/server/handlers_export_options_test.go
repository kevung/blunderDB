package server

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// shareBody is the export body that hands one family's ids to another tenant:
// those rows and their positions, without the rest of the library or the
// sender's review history.
func shareBody(family string, ids ...int64) map[string]any {
	return map[string]any{
		"wholeLibrary": false, family: ids, "includeCollectionPositions": true,
		"includeFilterLibrary": false, "includeLessons": false, "includeAnkiHistory": false,
	}
}

// exports.sqlite honours the families the desktop and the CLI can leave out:
// the whole tenant carries the decks' review history, decks named by deckIds
// leave without it, and ankiHistory or transcriptions set to false drop them.
func TestExportSQLiteHonoursHistoryAndTranscriptions(t *testing.T) {
	ts, srv := newTestServerAndHandler(t)
	ctx := context.Background()
	st := srv.opts.Storage

	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "test.sgf"))
	if err != nil {
		t.Fatal(err)
	}
	uploadImportNamed(t, ts, "/v1/imports.gnubg", "test.sgf", fixture)
	var matchID int64
	for m, err := range st.Matches().List(ctx, testTenant, storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		matchID = m.ID
	}
	if _, err := st.Transcriptions().Save(ctx, testTenant, &storage.Transcription{MatchID: matchID, Label: "brouillon", Document: `{"header":{}}`}); err != nil {
		t.Fatal(err)
	}
	p := domain.InitializePosition()
	posID, err := st.Positions().Save(ctx, testTenant, &p)
	if err != nil {
		t.Fatal(err)
	}
	deck, err := st.Anki().CreateDeck(ctx, testTenant, "Ouvertures", "", domain.AnkiSourceSearch, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Anki().SyncWithPositions(ctx, testTenant, deck, []int64{posID}); err != nil {
		t.Fatal(err)
	}
	next, err := st.Anki().NextCard(ctx, testTenant, deck)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.Anki().ReviewCard(ctx, testTenant, next.Card.ID, 3); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name            string
		req             any
		reviews, drafts int
	}{
		{"whole tenant", nil, 1, 1},
		{"decks named", shareBody("deckIDs", deck), 0, 0},
		{"history off", map[string]any{"includeAnkiHistory": false}, 0, 1},
		{"drafts off", map[string]any{"includeTranscriptions": false}, 1, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := post(t, ts, "/v1/exports.sqlite", tc.req)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("status = %d", resp.StatusCode)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "export.db")
			if err := os.WriteFile(path, body, 0o600); err != nil {
				t.Fatal(err)
			}
			dst, err := sqlite.Open(ctx, path, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer dst.Close()
			reviews, drafts := 0, 0
			for _, err := range dst.Anki().ReviewLog(ctx, "", 0, 0) {
				if err != nil {
					t.Fatal(err)
				}
				reviews++
			}
			for _, err := range dst.Transcriptions().List(ctx, "") {
				if err != nil {
					t.Fatal(err)
				}
				drafts++
			}
			if reviews != tc.reviews || drafts != tc.drafts {
				t.Errorf("%d review events, %d drafts; want %d, %d", reviews, drafts, tc.reviews, tc.drafts)
			}
		})
	}
}

// The daemon streams a plain SQLite file: a password it would not apply is refused,
// on the export and on its plan alike.
func TestExportSQLiteRefusesPassword(t *testing.T) {
	ts := newTestServer(t)
	for _, route := range []string{"/v1/exports.sqlite", "/v1/exports.plan"} {
		resp := post(t, ts, route, map[string]any{"password": "secret"})
		resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s with a password: status %d, want 400", route, resp.StatusCode)
		}
	}
}
