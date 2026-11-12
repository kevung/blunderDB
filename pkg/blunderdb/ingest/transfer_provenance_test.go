package ingest

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// analysesByHash reads every analysis of s, keyed by its position's Zobrist hash.
func analysesByHash(t *testing.T, s storage.Storage) map[uint64]domain.PositionAnalysis {
	t.Helper()
	ctx := context.Background()
	var ids []int64
	hashOf := map[int64]uint64{}
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
		hashOf[p.ID] = engine.ZobristHash(p)
	}
	loaded, err := s.Analyses().LoadMany(ctx, "", ids)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uint64]domain.PositionAnalysis{}
	for id, a := range loaded {
		out[hashOf[id]] = *a
	}
	return out
}

// withoutProvenance is a with what names its match, and its row's own ids and
// dates, cleared.
func withoutProvenance(a domain.PositionAnalysis) domain.PositionAnalysis {
	a.PositionID = 0
	a.Player1, a.Player2 = "", ""
	a.PlayedMove, a.PlayedCubeAction = "", ""
	a.PlayedMoves, a.PlayedCubeActions = nil, nil
	a.CreationDate, a.LastModifiedDate = time.Time{}, time.Time{}
	return a
}

// TestTransfer_OmitProvenance: a collection copied with OmitProvenance arrives
// with its analyses whole but nothing naming the match they were played in —
// no player, no played move or cube action, no match. Without it the copy
// carries them as before, and the source is never touched.
func TestTransfer_OmitProvenance(t *testing.T) {
	ctx := context.Background()
	src := seedLibrary(t, "test.xg", "lesson")
	var col int64
	for c, err := range src.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		col = c.ID
	}
	nameAnalyses(t, src, "Alice", "Bob")
	before := analysesByHash(t, src)

	copyCollection := func(omit bool, whole bool) (*sqlite.Storage, TransferReport) {
		dst, err := sqlite.Open(ctx, ":memory:", nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { dst.Close() })
		opts := WholeLibraryOptions()
		if !whole {
			opts.WholeLibrary = false
			opts.CollectionIDs = []int64{col}
		}
		opts.OmitProvenance = omit
		rep, err := Transfer(ctx, TransferEnd{S: src}, TransferEnd{S: dst}, opts, nil)
		if err != nil {
			t.Fatalf("transfer (omit provenance %v): %v", omit, err)
		}
		return dst, rep
	}

	kept, _ := copyCollection(false, false)
	withNames := analysesByHash(t, kept)
	named, played := 0, 0
	for _, a := range withNames {
		if a.Player1 == "Alice" && a.Player2 == "Bob" {
			named++
		}
		if len(a.PlayedMoves) > 0 || len(a.PlayedCubeActions) > 0 {
			played++
		}
	}
	if named != len(withNames) || played == 0 {
		t.Fatalf("by default the copy carries the players and the played moves: %d of %d analyses named, %d with played moves", named, len(withNames), played)
	}

	bare, _ := copyCollection(true, false)
	got := analysesByHash(t, bare)
	if len(got) == 0 || len(got) != len(withNames) {
		t.Fatalf("%d analyses arrived without provenance, %d with it", len(got), len(withNames))
	}
	for h, a := range got {
		if a.Player1 != "" || a.Player2 != "" || a.PlayedMove != "" || a.PlayedCubeAction != "" ||
			len(a.PlayedMoves) > 0 || len(a.PlayedCubeActions) > 0 {
			t.Errorf("position %x carries provenance: players %q/%q, played %q %v, cube %q %v",
				h, a.Player1, a.Player2, a.PlayedMove, a.PlayedMoves, a.PlayedCubeAction, a.PlayedCubeActions)
		}
		if a.CheckerAnalysis == nil && a.DoublingCubeAnalysis == nil {
			t.Errorf("position %x arrived without its evaluation", h)
		}
		if want := withoutProvenance(withNames[h]); !reflect.DeepEqual(withoutProvenance(a), want) {
			t.Errorf("position %x: the analysis differs beyond its provenance:\n got %+v\nwant %+v", h, withoutProvenance(a), want)
		}
	}

	whole, rep := copyCollection(true, true)
	if rep.Exported.Positions == 0 || rep.Exported.Matches != 0 || rep.Exported.Tournaments != 0 ||
		rep.Exported.Transcriptions != 0 || rep.Exported.Aliases != 0 {
		t.Errorf("a whole library without provenance read %+v, want positions and no match, tournament, transcription nor alias", rep.Exported)
	}
	for _, err := range whole.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		t.Error("the destination holds a match")
		break
	}
	for h, a := range analysesByHash(t, whole) {
		if a.Player1 != "" || a.Player2 != "" || len(a.PlayedMoves) > 0 || len(a.PlayedCubeActions) > 0 {
			t.Errorf("position %x of the whole library carries provenance", h)
			break
		}
	}

	if after := analysesByHash(t, src); !reflect.DeepEqual(after, before) {
		t.Error("a transfer changed the source's analyses")
	}
}

// nameAnalyses writes the players of a match into every analysis of s, as an
// import that knows them does.
func nameAnalyses(t *testing.T, s storage.Storage, p1, p2 string) {
	t.Helper()
	ctx := context.Background()
	var ids []int64
	for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}
	loaded, err := s.Analyses().LoadMany(ctx, "", ids)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range loaded {
		a.Player1, a.Player2 = p1, p2
	}
	if err := s.Analyses().SaveMany(ctx, "", loaded); err != nil {
		t.Fatal(err)
	}
}

// TestTransfer_OmitProvenanceRefusesNamedMatches: a match or a tournament named
// by id cannot leave without the names it is made of.
func TestTransfer_OmitProvenanceRefusesNamedMatches(t *testing.T) {
	ctx := context.Background()
	src := seedLibrary(t, "test.xg", "lesson")
	var match int64
	for m, err := range src.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		match = m.ID
	}
	dst, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	opts := WholeLibraryOptions()
	opts.MatchIDs = []int64{match}
	opts.OmitProvenance = true
	if _, err := Transfer(ctx, TransferEnd{S: src}, TransferEnd{S: dst}, opts, nil); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("a named match without provenance: %v, want ErrInvalid", err)
	}
}

// TestTransfer_OmitProvenanceRefusesNamedTournaments: a tournament named by id,
// alone, cannot leave without the names it is made of.
func TestTransfer_OmitProvenanceRefusesNamedTournaments(t *testing.T) {
	ctx := context.Background()
	src := seedLibrary(t, "test.xg", "lesson")
	tour, err := src.Tournaments().Create(ctx, "", "Open de Lyon", "2026-03-01", "Lyon")
	if err != nil {
		t.Fatal(err)
	}
	dst, err := sqlite.Open(ctx, ":memory:", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	opts := WholeLibraryOptions()
	opts.TournamentIDs = []int64{tour}
	opts.OmitProvenance = true
	if _, err := Transfer(ctx, TransferEnd{S: src}, TransferEnd{S: dst}, opts, nil); !errors.Is(err, storage.ErrInvalid) {
		t.Errorf("a named tournament without provenance: %v, want ErrInvalid", err)
	}
}

// provenanceLibrary is a library whose every family names its matches: the
// players of the match in each analysis, an alias, a transcription, a living
// collection and saved filters that select by a player, and the producer's
// own name. It returns the living collection and the player its query names.
func provenanceLibrary(t *testing.T) (src *sqlite.Storage, living int64, player string) {
	t.Helper()
	ctx := context.Background()
	src = seedLibrary(t, "test.xg", "lesson")
	for m, err := range src.Matches().List(ctx, "", storage.MatchListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		player = m.Player1Name
	}
	if player == "" {
		t.Fatal("the fixture's match names no player")
	}
	if _, err := src.Transcriptions().Save(ctx, "", &storage.Transcription{FormatVersion: "1",
		Label: player + " – X", Document: `{"header":{"player1":"` + player + `"}}`}); err != nil {
		t.Fatal(err)
	}
	if err := src.Aliases().Set(ctx, "", storage.AliasPlayer, "Surnom", player); err != nil {
		t.Fatal(err)
	}
	for name, command := range map[string]string{"contre lui": `s op"` + player + `"`, "gros coups": "s E>0.1"} {
		if _, err := src.Filters().Save(ctx, "", name, command); err != nil {
			t.Fatal(err)
		}
	}
	living, err := storage.CreateCollection(ctx, src, "", "Ses coups", "", `s pl"`+player+`"`)
	if err != nil {
		t.Fatal(err)
	}
	return src, living, player
}

// fileHolds lists the table.column of the SQLite file at path that hold needle.
func fileHolds(t *testing.T, path, needle string) []string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	names := func(query string, args ...any) []string {
		rows, err := db.Query(query, args...)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				t.Fatal(err)
			}
			out = append(out, name)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return out
	}
	var found []string
	for _, table := range names(`SELECT name FROM sqlite_master WHERE type = 'table'`) {
		for _, c := range names(`SELECT name FROM pragma_table_info(?)`, table) {
			var n int
			if err := db.QueryRow(`SELECT COUNT(*) FROM "`+table+`" WHERE CAST("`+c+`" AS TEXT) LIKE ?`, "%"+needle+"%").Scan(&n); err != nil {
				t.Fatal(err)
			}
			if n > 0 {
				found = append(found, table+"."+c)
			}
		}
	}
	return found
}

// TestExport_OmitProvenanceLeavesNoName: a file exported without provenance
// holds the player's name nowhere — not in a living collection's query, which
// leaves frozen on the members it selects at the source, not in a saved
// filter, an alias, a transcription or the producer's metadata — while the
// same export with provenance carries each of them.
func TestExport_OmitProvenanceLeavesNoName(t *testing.T) {
	ctx := context.Background()
	src, living, player := provenanceLibrary(t)
	selected, err := storage.EvaluateCollection(ctx, src, "", living, 0)
	if err != nil || len(selected.PositionIDs) == 0 {
		t.Fatalf("the living collection selects %d positions at the source (%v); the test needs some", len(selected.PositionIDs), err)
	}

	export := func(omit bool) (string, ExportReport) {
		opts := WholeLibraryOptions()
		opts.OmitProvenance = omit
		plan, err := OptionsFor(ctx, src, "", opts)
		if err != nil {
			t.Fatal(err)
		}
		plan.Metadata = map[string]string{"user": player, "description": "les matchs de " + player}
		path := filepath.Join(t.TempDir(), "export.db")
		rep, err := ExportSQLite(ctx, src, "", path, plan)
		if err != nil {
			t.Fatal(err)
		}
		return path, rep
	}

	with, rep := export(false)
	held := fileHolds(t, with, player)
	for _, want := range []string{"collection.filter_query", "filter_library.command", "metadata.value"} {
		if !slices.Contains(held, want) {
			t.Fatalf("with provenance %s does not hold the name (held in %v): the test data proves nothing", want, held)
		}
	}
	if rep.Aliases == 0 || rep.Transcriptions == 0 {
		t.Fatalf("with provenance the file carries %d aliases and %d transcriptions; the test needs both", rep.Aliases, rep.Transcriptions)
	}

	bare, rep := export(true)
	if held := fileHolds(t, bare, player); len(held) > 0 {
		t.Errorf("without provenance the file still holds %q in %v", player, held)
	}
	if rep.Aliases != 0 || rep.Transcriptions != 0 || rep.Matches != 0 {
		t.Errorf("without provenance the file carries %d aliases, %d transcriptions, %d matches; want none", rep.Aliases, rep.Transcriptions, rep.Matches)
	}
	dst, err := sqlite.Open(ctx, bare, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	var kept []string
	for f, err := range dst.Filters().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		kept = append(kept, f.Name)
	}
	if !slices.Equal(kept, []string{"gros coups"}) {
		t.Errorf("saved filters in the file: %v; want only the one that names nobody", kept)
	}
	found := false
	for c, err := range dst.Collections().List(ctx, "") {
		if err != nil {
			t.Fatal(err)
		}
		if c.Name != "Ses coups" {
			continue
		}
		found = true
		ev, err := storage.EvaluateCollection(ctx, dst, "", c.ID, 0)
		if err != nil {
			t.Fatal(err)
		}
		if c.FilterQuery != "" || len(ev.PositionIDs) != len(selected.PositionIDs) {
			t.Errorf("the living collection arrived with query %q and %d positions; want it frozen on the %d it selects at the source",
				c.FilterQuery, len(ev.PositionIDs), len(selected.PositionIDs))
		}
	}
	if !found {
		t.Error("the living collection did not leave")
	}
}
