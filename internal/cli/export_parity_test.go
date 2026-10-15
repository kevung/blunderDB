package cli

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/internal/server/middleware"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
	"github.com/kevung/blunderdb/pkg/blunderdb/transcript"
)

// One set of export options leaves the same way from the three entry points: the GUI's
// dialog (Database.ExportDatabase with what exportService.js sends when the library is
// listed), the CLI's `export --type database` with its flags, and the daemon's exports.sqlite
// with the same options as its JSON body. Their files are compared table by table, rows
// included, so a family one of them forgets or reads differently shows up here whatever that
// family is.
func TestExportParity_GUICLIAndDaemon(t *testing.T) {
	everything := func() ExportOptions {
		return ExportOptions{
			WholeLibrary: true, IncludeAnalysis: true, IncludeComments: true, IncludePlayedMoves: true,
			IncludeFilterLibrary: true, IncludeMatches: true, IncludeCollections: true,
			IncludeTournaments: true, IncludeLessons: true, IncludeAnkiDecks: true,
			IncludeTranscriptions: true, IncludeAliases: true, IncludeAnkiHistory: true,
		}
	}
	cases := []struct {
		name string
		gui  func(seeded) ExportOptions
		cli  func(seeded) []string
		body func(seeded) string
	}{
		{
			name: "whole library",
			gui:  func(seeded) ExportOptions { return everything() },
			cli:  func(seeded) []string { return nil },
			body: func(seeded) string { return "" },
		},
		{
			name: "families narrowed",
			gui: func(s seeded) ExportOptions {
				o := everything()
				o.LessonIDs, o.CollectionIDs, o.IncludeAnkiDecks, o.IncludeTranscriptions = []int64{s.lesson}, []int64{s.collection}, false, false
				return o
			},
			cli: func(s seeded) []string {
				return []string{fmt.Sprintf("--lesson-ids=%d", s.lesson), fmt.Sprintf("--collection-ids=%d", s.collection), "--decks=false", "--transcriptions=false"}
			},
			body: func(s seeded) string {
				return fmt.Sprintf(`{"lessonIDs":[%d],"collectionIDs":[%d],"includeAnkiDecks":false,"includeTranscriptions":false}`, s.lesson, s.collection)
			},
		},
		{
			name: "families left out",
			gui: func(seeded) ExportOptions {
				o := everything()
				o.IncludeLessons, o.IncludeFilterLibrary, o.IncludeAnkiHistory, o.IncludeTournaments = false, false, false, false
				return o
			},
			cli: func(seeded) []string {
				return []string{"--lessons=false", "--filters=false", "--anki-history=false", "--tournaments=false"}
			},
			body: func(seeded) string {
				return `{"includeLessons":false,"includeFilterLibrary":false,"includeAnkiHistory":false,"includeTournaments":false}`
			},
		},
		{
			// The CLI exports no list of positions; the daemon reads the desktop's list.
			name: "list of positions",
			gui: func(s seeded) ExportOptions {
				return ExportOptions{
					PositionIDs: s.positions, IncludeAnalysis: true, IncludeComments: true, IncludePlayedMoves: true,
					IncludeMatches: true, IncludeTranscriptions: true, IncludeAliases: true,
					IncludeCollections: true, CollectionIDs: []int64{s.collection},
				}
			},
			body: func(s seeded) string {
				return fmt.Sprintf(`{"wholeLibrary":false,"positionIDs":%s,"collectionIDs":[%d],"includeFilterLibrary":false,"includeLessons":false,"includeAnkiHistory":false}`,
					strings.ReplaceAll(fmt.Sprint(s.positions), " ", ","), s.collection)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cli, dbPath := setupCLIWithDB(t)
			ids := seedWholeLibrary(t, cli)
			// A second lesson, so that narrowing to the first leaves one behind.
			if _, err := cli.db.CreateLesson("Hors programme", ""); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()

			gui := filepath.Join(dir, "gui.db")
			opts := tc.gui(ids)
			opts.ExportPath = gui
			if _, err := cli.db.ExportDatabase(opts); err != nil {
				t.Fatalf("GUI export: %v", err)
			}

			daemon := filepath.Join(dir, "daemon.db")
			exportThroughDaemon(t, dbPath, daemon, tc.body(ids))

			outputs := map[string]string{"GUI": gui}
			if tc.cli != nil {
				cliOut := filepath.Join(dir, "cli.db")
				captureStdout(t, func() {
					args := append([]string{"export", "--db", dbPath, "--type", "database", "--file", cliOut}, tc.cli(ids)...)
					if err := cli.Run(args); err != nil {
						t.Fatalf("CLI export: %v", err)
					}
				})
				outputs["CLI"] = cliOut
			}

			want := dumpTables(t, daemon)
			if tc.name == "whole library" {
				if len(want["transcription"]) == 0 || len(want["anki_deck"]) == 0 || len(want["collection"]) == 0 || len(want["lesson"]) == 0 {
					t.Fatalf("the daemon's export lacks a seeded family: %d transcriptions, %d decks, %d collections, %d lessons",
						len(want["transcription"]), len(want["anki_deck"]), len(want["collection"]), len(want["lesson"]))
				}
				if drafts := strings.Join(want["transcription"], "\n"); strings.Contains(drafts, "/home/producteur") || !strings.Contains(drafts, "finale.mp4") {
					t.Errorf("the exported draft should name its video, not locate it: %s", drafts)
				}
			}
			if tc.name == "list of positions" && (len(want["position"]) == 0 || len(want["match"]) == 0) {
				t.Fatalf("the daemon's list export holds %d positions and %d matches, want both", len(want["position"]), len(want["match"]))
			}
			if tc.name == "families narrowed" && (len(want["lesson"]) != 1 || len(want["anki_deck"]) != 0) {
				t.Fatalf("the daemon's narrowed export holds %d lessons and %d decks, want 1 and 0", len(want["lesson"]), len(want["anki_deck"]))
			}
			for name, path := range outputs {
				got := dumpTables(t, path)
				for table, rows := range want {
					if !slices.Equal(got[table], rows) {
						t.Errorf("%s export, table %s: %d rows, daemon %d\n%s\nvs daemon\n%s",
							name, table, len(got[table]), len(rows), strings.Join(got[table], "\n"), strings.Join(rows, "\n"))
					}
				}
				for table := range got {
					if _, ok := want[table]; !ok {
						t.Errorf("%s export has table %s, the daemon's has not", name, table)
					}
				}
			}
		})
	}
}

// seeded names what seedWholeLibrary created that an export can narrow to.
type seeded struct {
	collection, lesson int64
	positions          []int64
}

// seedWholeLibrary puts one of each family an export carries into the database: a match in a
// tournament, a collection, a lesson, a saved filter, an Anki deck and a draft transcription.
func seedWholeLibrary(t *testing.T, cli *CLI) seeded {
	t.Helper()
	colID, posIDs := seedCollection(t, cli, "Primes", 3)
	matches, err := cli.db.GetAllMatches()
	if err != nil || len(matches) == 0 {
		t.Fatalf("GetAllMatches = %d, %v", len(matches), err)
	}
	tID, err := cli.db.CreateTournament("Open", "2026-03-12", "Paris")
	if err != nil {
		t.Fatal(err)
	}
	if err := cli.db.AddMatchToTournament(tID, matches[0].ID); err != nil {
		t.Fatal(err)
	}
	lessonID, err := cli.db.CreateLesson("Amorces", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.db.AddLessonStep(lessonID, "étape", "texte", 0, posIDs[0]); err != nil {
		t.Fatal(err)
	}
	if err := cli.db.SaveFilter("Primes", "s p>5"); err != nil {
		t.Fatal(err)
	}
	deckID, err := cli.db.CreateAnkiDeck("Drill", "", AnkiSourceCollection, colID, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.db.SyncAnkiDeck(deckID); err != nil {
		t.Fatal(err)
	}
	if _, err := cli.db.CreateTranscription(transcript.Header{MatchLength: 7, VideoSource: "/home/producteur/Vidéos/finale.mp4"}); err != nil {
		t.Fatal(err)
	}
	return seeded{collection: colID, lesson: lessonID, positions: posIDs}
}

const daemonTenant = "1"

// exportThroughDaemon serves the database file as the daemon does and saves to out its
// exports.sqlite answer to body, the JSON request ("" sends none).
func exportThroughDaemon(t *testing.T, dbPath, out, body string) {
	t.Helper()
	st, err := sqlite.Open(context.Background(), dbPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	// Saved searches are the one family a SQLite daemon keeps per tenant (filter_library's
	// scope column) while the desktop keeps its own under no tenant: the daemon's tenant gets
	// the same one.
	for f, err := range st.Filters().List(context.Background(), "") {
		if err != nil {
			t.Fatal(err)
		}
		if _, err := st.Filters().Save(context.Background(), daemonTenant, f.Name, f.Command); err != nil {
			t.Fatal(err)
		}
	}
	srv, err := server.New(server.Options{Storage: st})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	var reqBody io.Reader
	if body != "" {
		reqBody = strings.NewReader(body)
	}
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/v1/exports.sqlite", reqBody)
	req.Header.Set(middleware.TenantHeader, daemonTenant)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	file, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("exports.sqlite: %d %v %s", resp.StatusCode, err, file[:min(200, len(file))])
	}
	if err := os.WriteFile(out, file, 0o600); err != nil {
		t.Fatal(err)
	}
}

// dumpTables reads every table of an exported file as sorted, printable rows. What the export
// stamps with the time it ran — the file's date, a row's creation, a fresh card's due date —
// is left out: two exports of one library differ there and nowhere else.
func dumpTables(t *testing.T, path string) map[string][]string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	out := make(map[string][]string)
	for _, line := range queryLines(t, db, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`) {
		table := strings.TrimPrefix(line, "name=")
		query := `SELECT * FROM "` + table + `"`
		if table == "metadata" {
			query += ` WHERE key <> 'dateOfCreation'`
		}
		out[table] = queryLines(t, db, query)
	}
	return out
}

// queryLines runs query and prints each row as "column=value …", sorted.
func queryLines(t *testing.T, db *sql.DB, query string) []string {
	t.Helper()
	r, err := db.Query(query)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	defer r.Close()
	cols, err := r.Columns()
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for r.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := r.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		cells := make([]string, len(cols))
		for i, v := range vals {
			if strings.HasSuffix(cols[i], "_at") || cols[i] == "due" {
				v = "<when>"
			}
			cells[i] = fmt.Sprintf("%s=%v", cols[i], v)
		}
		lines = append(lines, strings.Join(cells, " "))
	}
	if err := r.Err(); err != nil {
		t.Fatal(err)
	}
	slices.Sort(lines)
	return lines
}

// A flag set to false takes its family out of `export --type database`; a list of ids alone
// brings the family in, narrowed to those.
func TestCLI_DatabaseExportFamilies(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	seedWholeLibrary(t, cli)
	run := func(args ...string) map[string][]string {
		t.Helper()
		out := filepath.Join(t.TempDir(), "out.db")
		captureStdout(t, func() {
			if err := cli.Run(append([]string{"export", "--db", dbPath, "--type", "database", "--file", out}, args...)); err != nil {
				t.Fatalf("export %v: %v", args, err)
			}
		})
		return dumpTables(t, out)
	}

	none := run("--collections=false", "--decks=false", "--tournaments=false")
	for _, table := range []string{"collection", "anki_deck", "tournament"} {
		if n := len(none[table]); n != 0 {
			t.Errorf("with its flag off, %s has %d rows", table, n)
		}
	}

	if _, err := cli.db.CreateCollection("Autre", ""); err != nil {
		t.Fatal(err)
	}
	picked := run("--collections=false", "--collection-ids=1")
	if n := len(picked["collection"]); n != 1 {
		t.Errorf("--collection-ids=1 exported %d collections, want 1", n)
	}
	if n := len(run()["collection"]); n != 2 {
		t.Errorf("--collections with no ids exported %d collections, want both", n)
	}
}

// `export --type matches --match-ids` writes those matches alone, each with its positions.
func TestCLI_MatchesExportByIDs(t *testing.T) {
	cli, dbPath := setupCLIWithDB(t)
	first, err := cli.db.ImportXGMatch(testdataPath("test.xg"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cli.db.ImportXGMatch(testdataPath("HsbtMarseille_main_ronde4_LamourDeCaslouGildas_UngerKevin_7p.xg")); err != nil {
		t.Fatal(err)
	}
	export := func(args ...string) *sql.DB {
		t.Helper()
		out := filepath.Join(t.TempDir(), "matchs.db")
		captureStdout(t, func() {
			if err := cli.Run(append([]string{"export", "--db", dbPath, "--type", "matches", "--file", out}, args...)); err != nil {
				t.Fatalf("export %v: %v", args, err)
			}
		})
		db, err := sql.Open("sqlite", out)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		return db
	}
	count := func(db *sql.DB, query string) int {
		t.Helper()
		var n int
		if err := db.QueryRow(query).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	one := export("--match-ids", fmt.Sprint(first))
	if n := count(one, `SELECT COUNT(*) FROM match`); n != 1 {
		t.Errorf("--match-ids %d exported %d matches, want 1", first, n)
	}
	if n := count(one, `SELECT COUNT(*) FROM move WHERE position_id IS NOT NULL AND position_id NOT IN (SELECT id FROM position)`); n != 0 {
		t.Errorf("%d moves point at no position", n)
	}
	if n := count(export(), `SELECT COUNT(*) FROM match`); n != 2 {
		t.Errorf("without --match-ids, %d matches exported, want 2", n)
	}
	if count(one, `SELECT COUNT(*) FROM position`) >= count(export(), `SELECT COUNT(*) FROM position`) {
		t.Error("one match carried as many positions as both")
	}
}
