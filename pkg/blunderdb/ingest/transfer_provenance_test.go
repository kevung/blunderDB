package ingest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
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
	for name, command := range map[string]string{"contre lui": `s op"` + player + `"`, "gros coups": "s E>0.1", "gagnants": "s w>50"} {
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
// filter (one that reads the played move stays too), an alias, a transcription or the producer's metadata — while the
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
	if !slices.Equal(kept, []string{"gagnants"}) {
		t.Errorf("saved filters in the file: %v; want only the one that reads the position alone", kept)
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

// needle is a name planted in every place of a query the parser lets a word
// stand: no file exported without provenance may hold it.
const needle = "Zanzibar"

// anonymityCorpus is one query per filter of the grammar, and per way a text
// the parser does not understand can ride in a query. living says whether the
// query may leave (rewritten) without provenance when the comments leave too;
// bare, when they do not.
var anonymityCorpus = []struct {
	query        string
	living, bare bool
}{
	// What the position holds: leaves, rewritten.
	{"s w>50", true, true},
	{"s c w>50 g>10 b>1 W>50 G>10 B>1", true, true},
	{"s nc P>100 k>1 K>1 z>5 Z>5 o5 O>0", true, true},
	{"s bo>0 BO>0 bj>0 BJ>0 e>0 xD65", true, true},
	{"s d fl aw:2 AW>3 M", true, true},
	{"s cr", true, true},
	{"s T>2020/01/01", true, true},
	{`s m"13/11"`, true, true},
	{"s ph:race", true, true},
	{"s gt:holding", true, true},
	{"s ad:3ply+", true, true},
	{"ss w>50", true, true},
	// What the comments hold: leaves with them only.
	{"s #monaco2026", true, false},
	{"s co", true, false},
	{"s xco", true, false},
	{"s co:user", true, false},
	// What the played move holds: never leaves.
	{"s E>0.1", false, false},
	{"s E<0.05", false, false},
	{"s tm>5", false, false},
	{"s mg>50", false, false},
	{"s dr", false, false},
	{"s dd", false, false},
	// What names the match: never leaves.
	{"s like42", false, false},
	{"s like42<12*", false, false},
	{"s like5 w>50", false, false},
	{"s md:2020", false, false},
	{"s md>2019-06", false, false},
	{"s ml:7", false, false},
	{"s pr>5", false, false},
	{"s n>1", false, false},
	{"s i", false, false},
	{"s rd:" + needle, false, false},
	{`s pl"` + needle + `"`, false, false},
	{`s op"` + needle + `"`, false, false},
	{`s tn"` + needle + `"`, false, false},
	{`s au"` + needle + `"`, false, false},
	{`s t"` + needle + `"`, false, false},
	// A second value the parser drops without a word.
	{`s m"13/11" m"` + needle + `"`, false, false},
	{`s w>50 pl"a" pl"` + needle + `"`, false, false},
	// A token the parser does not know, or knows and ignores.
	{`s w>50 hello"` + needle, false, false},
	{"s w>50 w>60" + needle, false, false},
	{"s w>50 x", false, false},
	// A word a prefix rule or an open vocabulary takes for a value.
	{"s w" + needle, false, false},
	{"s id1" + needle, false, false},
	{"s b" + needle[:1] + "<-11", false, false},
	{"s #" + strings.ToLower(needle), false, false},
	{"s co #" + strings.ToLower(needle), false, false},
	{"s " + strings.ToLower(needle), false, false},
	{"s T" + needle, false, false},
	{"s ph:" + strings.ToLower(needle), false, false},
	{"s gt:" + strings.ToLower(needle), false, false},
	{"s co:" + strings.ToLower(needle), false, false},
	{"s ad:" + strings.ToLower(needle), false, false},
	{`s m"` + needle + `"`, false, false},
}

// queryWords are the tokens and quoted values of a query, for comparing two
// queries by what they hold.
func queryWords(query string) map[string]bool {
	words := map[string]bool{}
	for _, w := range strings.Fields(query) {
		words[w] = true
	}
	return words
}

// TestExport_OmitProvenanceQueryNeverLeavesAsWritten: without provenance a
// living collection either leaves frozen on what it selects at the source, or
// living on the query the parser writes back — never with a string the
// rewritten form does not hold — and selects at the destination the positions
// it selects at the source.
func TestExport_OmitProvenanceQueryNeverLeavesAsWritten(t *testing.T) {
	ctx := context.Background()
	src, _, _ := provenanceLibrary(t)
	n := 0
	for p, err := range src.Positions().List(ctx, "", storage.ListOpts{Limit: 40}) {
		if err != nil {
			t.Fatal(err)
		}
		if n++; n%2 == 0 {
			if _, err := src.Comments().Add(ctx, "", p.ID, "joli coup #monaco2026"); err != nil {
				t.Fatal(err)
			}
		}
	}
	name := func(i int) string { return fmt.Sprintf("corpus %02d", i) }
	selected := make([]int, len(anonymityCorpus))
	for i, c := range anonymityCorpus {
		// Past the validation of CreateCollection: a query written by another
		// version reaches the export all the same.
		id, err := src.Collections().Create(ctx, "", name(i), "")
		if err != nil {
			t.Fatal(err)
		}
		if err := src.Collections().SetFilterQuery(ctx, "", id, c.query); err != nil {
			t.Fatal(err)
		}
		if ev, err := storage.EvaluateCollection(ctx, src, "", id, 0); err == nil {
			selected[i] = len(ev.PositionIDs)
		} else {
			selected[i] = -1 // unreadable at the source: nothing to compare
		}
	}

	export := func(omit, comments bool) string {
		opts := WholeLibraryOptions()
		opts.OmitProvenance, opts.IncludeComments = omit, comments
		plan, err := OptionsFor(ctx, src, "", opts)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "export.db")
		if _, err := ExportSQLite(ctx, src, "", path, plan); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if held := fileHolds(t, export(false, true), needle); !slices.Contains(held, "collection.filter_query") {
		t.Fatalf("with provenance the queries do not carry %q (held in %v): the test data proves nothing", needle, held)
	}

	for _, comments := range []bool{false, true} {
		path := export(true, comments)
		if held := fileHolds(t, path, needle); len(held) > 0 {
			t.Errorf("comments %v: the file holds %q in %v", comments, needle, held)
		}
		dst, err := sqlite.Open(ctx, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		arrived := map[string]storage.Collection{}
		for c, err := range dst.Collections().List(ctx, "") {
			if err != nil {
				t.Fatal(err)
			}
			arrived[c.Name] = *c
		}
		for i, c := range anonymityCorpus {
			got, ok := arrived[name(i)]
			if !ok {
				t.Errorf("comments %v, %q: the collection did not leave", comments, c.query)
				continue
			}
			f, _ := searchquery.Parse(c.query)
			rewritten := searchquery.Format(f)
			want := ""
			if (comments && c.living) || (!comments && c.bare) {
				want = rewritten
				if strings.HasPrefix(c.query, "ss ") {
					want = "s" + want
				}
			}
			if got.FilterQuery != want {
				t.Errorf("comments %v, %q: arrived with query %q, want %q", comments, c.query, got.FilterQuery, want)
			}
			held := queryWords(rewritten)
			for w := range queryWords(strings.TrimPrefix(got.FilterQuery, "s")) {
				if !held[w] {
					t.Errorf("comments %v, %q: arrived holding %q, which its rewritten form %q does not", comments, c.query, w, rewritten)
				}
			}
			ev, err := storage.EvaluateCollection(ctx, dst, "", got.ID, 0)
			if err != nil {
				t.Errorf("comments %v, %q: unreadable at the destination: %v", comments, c.query, err)
				continue
			}
			if selected[i] >= 0 && len(ev.PositionIDs) != selected[i] {
				t.Errorf("comments %v, %q (arrived %q): selects %d positions at the destination, %d at the source",
					comments, c.query, got.FilterQuery, len(ev.PositionIDs), selected[i])
			}
		}
		dst.Close()
	}
}

// selectionAtoms are tokens of every filter the grammar writes, several values
// each, from which the queries of the selection property are drawn.
var selectionAtoms = []string{
	"c", "s", "d", "dr", "dd", "D", "D1", "xD65", "xD11", "nc", "M", "i", "fl", "cr", "pcr",
	"co", "xco", "co:user", "co:import", "#monaco2026", "#absent",
	"ph:race", "ph:opening", "ph:middlegame", "ph:bearoff", "gt:holding", "gt:contact", "gt:race", "gt:blitz",
	"ad:3ply+", "ad:book", "ad:rollout", "ad:4ply",
	"p>10", "p<5", "p-5,5", "P>100", "P<80", "w>50", "w<40", "g>10", "g<5", "b>1", "W>50", "G>10", "B>1",
	"o1", "o>0", "O>0", "k>1", "k0", "K>1", "z>5", "Z>5", "bo>0", "BO>0", "bj>0", "BJ>0",
	"e>0", "e<-0.2", "e-0.1,0.1", "T>2020/01/01", "T<2020/01/01", `m"13/11"`, `m"24/23"`,
	"aw:2", "aw>3", "AW>3", "AW:5",
	"like", "like<12", "like*", "like42", "like42<12*", "like5", "id1", "id1;2", "id7 id9",
	"E>0.1", "E<0.05", "tm>5", "mg>50", "n>1", "ml:7", "md:2020", "pr>5", "rd:1", "ma1", "tn1",
	`pl"a"`, `op"a"`, `tn"a"`, `au"a"`, `t"joli"`,
}

// TestExport_OmitProvenanceSelectsWhatTheSourceSelects: a living collection
// that leaves living without provenance selects at the destination the very
// positions it selects at the source — whatever its query, drawn here from
// every filter of the grammar, alone and combined. A filter that reads a row
// the file does not carry fails here whatever it is classified as.
func TestExport_OmitProvenanceSelectsWhatTheSourceSelects(t *testing.T) {
	ctx := context.Background()
	src, _, player := provenanceLibrary(t)
	atoms := append(slices.Clone(selectionAtoms), `pl"`+player+`"`, `op"`+player+`"`, `t"`+player+`"`)
	n := 0
	for p, err := range src.Positions().List(ctx, "", storage.ListOpts{}) {
		if err != nil {
			t.Fatal(err)
		}
		n++
		if n%3 == 0 {
			if _, err := src.Comments().Add(ctx, "", p.ID, "joli coup #monaco2026"); err != nil {
				t.Fatal(err)
			}
		}
		if n%5 == 0 {
			if _, err := src.Positions().RaiseFlag(ctx, "", p); err != nil {
				t.Fatal(err)
			}
		}
	}

	queries := make([]string, 0, 700)
	for _, a := range atoms {
		queries = append(queries, "s "+a)
	}
	rng := rand.New(rand.NewPCG(1, 2))
	for len(queries) < 700 {
		k := 2 + rng.IntN(3)
		parts := make([]string, k)
		for i := range parts {
			parts[i] = atoms[rng.IntN(len(atoms))]
		}
		queries = append(queries, "s "+strings.Join(parts, " "))
	}

	hashes := func(s storage.Storage) map[int64]uint64 {
		out := map[int64]uint64{}
		for p, err := range s.Positions().List(ctx, "", storage.ListOpts{}) {
			if err != nil {
				t.Fatal(err)
			}
			out[p.ID] = engine.ZobristHash(p)
		}
		return out
	}
	selection := func(s storage.Storage, of map[int64]uint64, id int64) ([]uint64, error) {
		ev, err := storage.EvaluateCollection(ctx, s, "", id, 1<<20)
		if err != nil {
			return nil, err
		}
		out := make([]uint64, len(ev.PositionIDs))
		for i, id := range ev.PositionIDs {
			out[i] = of[id]
		}
		slices.Sort(out)
		return out, nil
	}

	name := func(i int) string { return fmt.Sprintf("tirage %03d", i) }
	srcHash := hashes(src)
	atSource := make([][]uint64, len(queries))
	readable := make([]bool, len(queries))
	for i, q := range queries {
		id, err := src.Collections().Create(ctx, "", name(i), "")
		if err != nil {
			t.Fatal(err)
		}
		if err := src.Collections().SetFilterQuery(ctx, "", id, q); err != nil {
			t.Fatal(err)
		}
		if sel, err := selection(src, srcHash, id); err == nil {
			atSource[i], readable[i] = sel, true
		}
	}

	for _, comments := range []bool{false, true} {
		opts := WholeLibraryOptions()
		opts.OmitProvenance, opts.IncludeComments = true, comments
		plan, err := OptionsFor(ctx, src, "", opts)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "export.db")
		if _, err := ExportSQLite(ctx, src, "", path, plan); err != nil {
			t.Fatal(err)
		}
		dst, err := sqlite.Open(ctx, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		dstHash := hashes(dst)
		arrived := map[string]storage.Collection{}
		for c, err := range dst.Collections().List(ctx, "") {
			if err != nil {
				t.Fatal(err)
			}
			arrived[c.Name] = *c
		}
		living, telling, gaps := 0, 0, 0
		var mute []string // the filters that leave and tell nothing on this library
		for i, q := range queries {
			got, ok := arrived[name(i)]
			if !ok {
				t.Errorf("comments %v, %q: the collection did not leave", comments, q)
				continue
			}
			if !readable[i] {
				if got.FilterQuery != "" {
					t.Errorf("comments %v, %q: unreadable at the source, yet it left living as %q", comments, q, got.FilterQuery)
				}
				continue
			}
			sel, err := selection(dst, dstHash, got.ID)
			if err != nil {
				t.Errorf("comments %v, %q (arrived %q): unreadable at the destination: %v", comments, q, got.FilterQuery, err)
				continue
			}
			if got.FilterQuery != "" {
				living++
				if len(sel) > 0 && len(sel) < len(srcHash) {
					telling++
				} else if i < len(atoms) {
					mute = append(mute, atoms[i])
				}
			}
			if !slices.Equal(sel, atSource[i]) {
				if gaps++; gaps <= 25 {
					t.Errorf("comments %v, %q (arrived %q): selects %d positions at the destination, %d at the source",
						comments, q, got.FilterQuery, len(sel), len(atSource[i]))
				}
			}
		}
		t.Logf("comments %v: %d queries, %d left living, %d of them select part of the library, %d gaps; alone, these select nothing or everything here: %v",
			comments, len(queries), living, telling, gaps, mute)
		if living < 100 || telling < 50 {
			t.Errorf("comments %v: %d queries left living, %d of them telling: the draw proves nothing", comments, living, telling)
		}
		dst.Close()
	}
}
