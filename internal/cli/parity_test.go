package cli

import (
	"context"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/kevung/blunderdb/internal/server"
	"github.com/kevung/blunderdb/pkg/blunderdb/database"
	"github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlite"
)

// This file checks the CLI/GUI/server parity invariant (CLAUDE.md) by
// reflection rather than by memory. Every exported method of
// database.Database — the surface the GUI binds wholesale through Wails — is
// looked up in databaseParity, which names the CLI command and the daemon
// route that expose it, or states why one mode deliberately lacks it. A new
// Database method that is neither covered nor allow-listed fails the test;
// so does a table entry naming a method, command or route that no longer
// exists, and so does a blank column with no reason. The reasons are the
// point: an absence is a decision (ADR-0015: modes differ in the form they
// expose because their roles differ, not because one was left half-built),
// and this table is where the decision is written down.

// parityEntry maps one Database method to its two headless faces.
type parityEntry struct {
	// CLI is "command" or "command sub-command" (the second token is checked
	// only for commands that expose a sub-command table). Empty when the CLI
	// deliberately lacks the method.
	CLI string
	// Server is the "/v1/<family>.<method>" route. Empty when the daemon
	// deliberately lacks the method.
	Server string
	// Why is required whenever CLI or Server is empty.
	Why string
}

// Reasons shared by several entries. Each names the decision, and the ADR
// when one exists.
const (
	whyLifecycle = "desktop lifecycle: the CLI opens the file through every command's --db and the daemon through --dsn at start-up; no runtime open/close/lock to expose"
	whyRawHandle = "raw *sql.DB seam for tests and the desktop's own SQL; never a mode's feature"
	whyGUIState  = "interactive GUI state (session, filter library, command and search history, last visited match): a script has nothing to read or restore there"
	whyGUIEdit   = "an editing gesture of the GUI (a position, a comment, a match's metadata or a list's order); a script imports, searches, lists and exports"
	whyTwoPhase  = "two-phase native .db import (analyse -> preview -> commit) is the shape of the GUI dialog; the CLI and the daemon import in one step (import --type database, /v1/imports.db)"
	whyIssuance  = "issuance is a person's act on a file they are producing (ADR-0007); the daemon operates on a library and signs, seals or opens nothing on anyone's behalf (ADR-0015)"
	whySubsetExp = "the daemon's exports.sqlite writes the whole tenant, optionally watermarked with the daemon's own signing identity (Options.Identity, --identity-dir); a subset export carrying its origin is the producer's desktop gesture (ADR-0007, ADR-0015)"
	whyReview    = "reviewing a card needs the board in front of the player: a GUI gesture (the daemon serves a client that has one)"
	whyCram      = "cram mode picks a random card for the board; the daemon's review loop goes through anki.nextCard and the CLI reviews nothing"
	whyWrapper   = "internal to the desktop wrapper: run by OpenDatabase / the importers themselves, not something a caller invokes"
	whyEngineRun = "the CLI's `analyze` and the daemon's gammonnet.analyzeMissing fill gaps (ADR-0013); replaying stale analyses after an engine bump is the GUI's maintenance dialog"
	whyServerEPC = "the CLI's `epc` computes from an XGID without a database; the daemon's positions.epc takes a position"
	whyMetadata  = "the metadata table is database infrastructure, not a tenant's data (« infrastructure de la base, pas une donnée de tenant » — ADR-0005, #156): global to every tenant and outside RLS, so the daemon reads its schema version (metadata.version) and nothing else; load/save/setVersion let one tenant read the others' session state, rewrite database_version and fail /readyz for the whole instance"
)

// databaseParity is the allow-list. Keep it sorted by method name.
var databaseParity = map[string]parityEntry{
	"AddComment":                     {Server: "/v1/comments.add", Why: whyGUIEdit},
	"AddMatchToTournament":           {Server: "/v1/tournaments.addMatch", Why: whyGUIEdit},
	"AddPositionToCollection":        {Server: "/v1/collections.addPosition", Why: whyGUIEdit},
	"AddPositionsToCollection":       {Server: "/v1/collections.addPositions", Why: whyGUIEdit},
	"AnalyzeImportDatabase":          {Why: whyTwoPhase},
	"AnalyzeMissingWithGammonNet":    {CLI: "analyze", Server: "/v1/gammonnet.analyzeMissing"},
	"AnalyzeStaleGammonNet":          {Why: whyEngineRun},
	"CancelImport":                   {Server: "/v1/imports.cancel", Why: "the CLI import is a foreground process: Ctrl-C is its cancel"},
	"CheckDatabaseVersion":           {CLI: "info", Server: "/v1/metadata.version"},
	"CheckSchema":                    {CLI: "verify", Why: "schema drift is what the desktop open's EnsureSchema could not add to a user's SQLite file (issue #177); the daemon's SQLite backend runs the same EnsureSchema on open, and PostgreSQL's schema comes from its versioned migrations alone — its audit is the operator's database tooling"},
	"CheckConstraints":               {CLI: "verify", Why: "the rules the fresh DDL states and SQLite cannot add to a table that already exists (CHECK, and the hash a row must not be without); a desktop file upgraded across ten versions is the only place they can be broken, and reporting them is a verify finding, not a route"},
	"CheckMatchExists":               {CLI: "import", Server: "/v1/matches.findByHash"},
	"CheckVersion":                   {Why: whyWrapper},
	"ClearCommandHistory":            {Server: "/v1/history.clear", Why: whyGUIState},
	"ClearSessionState":              {Server: "/v1/session.clear", Why: whyGUIState},
	"Close":                          {Why: whyLifecycle},
	"CollectionCoverage":             {Why: "feeds the GUI export dialog's per-collection 'n of m positions exported' figures; the CLI exports what it is told and the daemon exports the whole tenant"},
	"CommitImportDatabase":           {Why: whyTwoPhase},
	"ComputeEPCFromPosition":         {CLI: "epc", Server: "/v1/positions.epc", Why: whyServerEPC},
	"ComputeStats":                   {CLI: "list --type stats", Server: "/v1/stats.compute"},
	"Conn":                           {Why: whyRawHandle},
	"CopyPositionToCollection":       {Server: "/v1/collections.copyPosition", Why: whyGUIEdit},
	"CountOrphans":                   {CLI: "verify", Why: "orphaned game/move/analysis rows are the aftermath of the desktop pool enforcing foreign keys on one connection in ten (issue #157); the daemon's SQLite backend has always opened through DSN() and PostgreSQL enforces its keys server-side, so a library never carried any — its integrity is the operator's database tooling"},
	"CountPositionsWithoutAnalysis":  {CLI: "analyze", Server: "/v1/gammonnet.analyzeMissing"},
	"CreateAnkiDeck":                 {Server: "/v1/anki.createDeck", Why: "a deck is created from the GUI's current collection or search; the CLI lists, inspects and syncs decks"},
	"CreateCollection":               {CLI: "collection create", Server: "/v1/collections.create"},
	"CreateTournament":               {Server: "/v1/tournaments.create", Why: whyGUIEdit},
	"DeleteAnalysis":                 {Server: "/v1/analyses.delete", Why: whyGUIEdit},
	"DeleteAnkiDeck":                 {Server: "/v1/anki.deleteDeck", Why: "deleting a deck discards its review history: kept behind the GUI's confirmation"},
	"DeleteCollection":               {CLI: "collection delete", Server: "/v1/collections.delete"},
	"DeleteComment":                  {Server: "/v1/comments.deleteForPosition", Why: whyGUIEdit},
	"DeleteCommentEntry":             {Server: "/v1/comments.delete", Why: whyGUIEdit},
	"DeleteFilter":                   {Server: "/v1/filters.delete", Why: whyGUIState},
	"DeleteMatch":                    {CLI: "delete", Server: "/v1/matches.delete"},
	"DeletePosition":                 {Server: "/v1/positions.delete", Why: whyGUIEdit},
	"DeleteProtectedCopyPath":        {Why: whyIssuance + "; the CLI's `open` writes an ordinary database instead of keeping a protected copy around"},
	"DeleteSearchHistoryEntry":       {Server: "/v1/searchHistory.deleteEntry", Why: whyGUIState},
	"DeleteTournament":               {Server: "/v1/tournaments.delete", Why: whyGUIEdit},
	"ExportCollections":              {CLI: "collection export", Why: whySubsetExp},
	"ExportDatabase":                 {CLI: "export", Server: "/v1/exports.sqlite"},
	"ExportMatchMAT":                 {CLI: "export --type mat", Server: "/v1/matches.exportMat"},
	"ExportTournaments":              {CLI: "export --tournament-ids", Why: whySubsetExp},
	"GetAllAnkiDecks":                {CLI: "anki decks", Server: "/v1/anki.listDecks"},
	"GetAllCollections":              {CLI: "collection list", Server: "/v1/collections.list"},
	"GetAllComments":                 {Server: "/v1/comments.listAll", Why: "the CLI reaches comments through `search --has-comment`; the comment browser is a GUI panel"},
	"GetAllMatches":                  {CLI: "list --type matches", Server: "/v1/matches.list"},
	"GetAllPlayerNames":              {Server: "/v1/stats.playerNames", Why: "autocomplete for the GUI's player filter; `list --type players` prints every player with its figures"},
	"GetAllTournaments":              {CLI: "list --type tournaments", Server: "/v1/tournaments.list"},
	"GetAnkiDeckPositions":           {Server: "/v1/anki.deckPositions", Why: "the positions of a deck are the positions of its collection or search, which `collection show` and `search` list"},
	"GetAnkiDeckStats":               {CLI: "anki stats", Server: "/v1/anki.deckStats"},
	"GetAnkiForecast":                {CLI: "anki forecast", Server: "/v1/anki.forecast"},
	"GetAnkiDeckRetention":           {CLI: "anki retention", Server: "/v1/anki.retention"},
	"GetCollectionByID":              {CLI: "collection show", Server: "/v1/collections.get"},
	"GetCollectionPositions":         {CLI: "collection show", Server: "/v1/collections.positions"},
	"GetCommentsByPosition":          {Server: "/v1/comments.byPosition", Why: "the CLI prints a position's comment with `search --format json`; per-entry history is a GUI panel"},
	"GetDatabaseStats":               {CLI: "info", Server: "/v1/metadata.counts"},
	"GetDatabaseVersion":             {CLI: "info", Server: "/v1/metadata.version"},
	"GetGamesByMatch":                {Server: "/v1/matches.games", Why: "`match` prints a match position by position; the game/move split is the GUI navigator's and the daemon's"},
	"GetIssuanceInfo":                {CLI: "identity", Why: whyIssuance},
	"GetLastVisitedMatch":            {Server: "/v1/matches.lastVisited", Why: whyGUIState},
	"GetMatchByID":                   {CLI: "match", Server: "/v1/matches.get"},
	"GetMatchDetailStats":            {Server: "/v1/stats.matchDetail", Why: "per-match badges of the GUI's Matches tab; `list --type stats --match` is not a filter the CLI offers, `list --type players` and `--tournament` are"},
	"GetMatchMovePositions":          {CLI: "match", Server: "/v1/matches.movePositions"},
	"GetMatchTournament":             {Server: "/v1/tournaments.tournamentOf", Why: "`list --type tournaments` lists tournaments with their matches; the reverse lookup is a GUI label"},
	"GetMovesByGame":                 {Server: "/v1/matches.moves", Why: "`match` prints a match position by position; the game/move split is the GUI navigator's and the daemon's"},
	"GetNextAnkiCard":                {Server: "/v1/anki.nextCard", Why: whyReview},
	"GetPlayerTable":                 {CLI: "list --type players", Server: "/v1/stats.playerTable"},
	"GetPositionCollections":         {Server: "/v1/collections.collectionsOf", Why: "the GUI's 'in collections…' label; `collection show` lists the other direction"},
	"GetPositionIDsByMatch":          {Server: "/v1/stats.positionIdsByMatch", Why: "the CLI selects a match's positions with `search --match-ids`, which filters and prints them"},
	"GetPositionIDsByStatsSelection": {Server: "/v1/stats.positionIdsBySelection", Why: "drill-down from a statistics figure to the board; the CLI's `search` takes the same filters directly"},
	"GetPositionIDsByTournament":     {Server: "/v1/stats.positionIdsByTournament", Why: "the CLI selects a tournament's positions with `search --tournament-ids`, which filters and prints them"},
	"GetPositionIndexMap":            {CLI: "collection show", Server: "/v1/collections.positionIndexMap"},
	"GetPositionProvenance":          {Why: "the GUI's 'this position comes from matches…' tooltip; neither the CLI nor the daemon has a reader for it, and matches.movePositions covers the other direction"},
	"GetRandomAnkiCard":              {Why: whyCram},
	"GetStatsDateRange":              {Server: "/v1/stats.dateRange", Why: "bounds of the GUI's date picker; the CLI takes --from/--to as given"},
	"GetTournamentMatches":           {CLI: "list --type tournaments", Server: "/v1/tournaments.matches"},
	"ImportBGFMatch":                 {CLI: "import", Server: "/v1/imports.bgf"},
	"ImportBGFPosition":              {CLI: "import --type position", Server: "/v1/imports.position"},
	"ImportBGFPositionFromText":      {Server: "/v1/imports.position", Why: "the clipboard paste of a BGBlitz position; the CLI imports the file"},
	"ImportDatabase":                 {CLI: "import --type database", Server: "/v1/imports.db"},
	"ImportGnuBGMatch":               {CLI: "import", Server: "/v1/imports.gnubg"},
	"ImportGnuBGMatchFromText":       {Server: "/v1/imports.gnubg", Why: "the clipboard paste of a GNUbg match; the CLI imports the file"},
	"ImportXGMatch":                  {CLI: "import", Server: "/v1/imports.xg"},
	"ImportXGPPosition":              {CLI: "import --type position", Server: "/v1/positions.fromXGP"},
	"IsProtectedCopyPath":            {CLI: "open", Why: whyIssuance},
	"IsReadOnly":                     {Why: whyLifecycle + " (ADR-0004: the second desktop instance opens read-only; a CLI run is one process, the daemon owns its store)"},
	"ListPositionIDs":                {Server: "/v1/positions.listIds", Why: "the id list the GUI browses a library with (windows are fetched by LoadPositionsByIDs); `list --type positions` prints the positions themselves, which is what a script wants"},
	"LoadAllPositions":               {CLI: "list --type positions", Server: "/v1/positions.list"},
	"LoadAnalysis":                   {CLI: "search --format json", Server: "/v1/analyses.load"},
	"LoadCommandHistory":             {Server: "/v1/history.load", Why: whyGUIState},
	"LoadComment":                    {Server: "/v1/comments.text", Why: "the CLI prints a position's comment with `search --format json`"},
	"LoadEditPosition":               {Server: "/v1/filters.loadEditPosition", Why: whyGUIState},
	"LoadExcludePosition":            {Server: "/v1/filters.loadExcludePosition", Why: whyGUIState},
	"LoadFilters":                    {Server: "/v1/filters.list", Why: whyGUIState},
	"LoadMetadata":                   {CLI: "info", Why: whyMetadata},
	"LoadPosition":                   {CLI: "search --position-ids", Server: "/v1/positions.load"},
	"LoadPositionsByIDs":             {CLI: "search --position-ids", Server: "/v1/positions.loadByIds"},
	"LoadPositionsByFilters":         {CLI: "search", Server: "/v1/search.find"},
	"LoadPositionsByFiltersCore":     {CLI: "search", Server: "/v1/search.find"},
	"LoadSearchHistory":              {Server: "/v1/searchHistory.list", Why: whyGUIState},
	"LoadSessionState":               {Server: "/v1/session.load", Why: whyGUIState},
	"MergePlayers":                   {Server: "/v1/matches.mergePlayers", Why: whyGUIEdit},
	"MovePositionBetweenCollections": {Server: "/v1/collections.movePosition", Why: whyGUIEdit},
	"OpenDatabase":                   {Why: whyLifecycle},
	"OpenProtectedCopyPath":          {CLI: "open", Why: whyIssuance},
	"ParsePositionText":              {CLI: "import --type position", Server: "/v1/positions.parseText"},
	"RefreshSearchStatistics":        {Why: whyWrapper},
	"RemoveMatchFromTournament":      {Server: "/v1/tournaments.removeMatch", Why: whyGUIEdit},
	"RemovePositionFromCollection":   {Server: "/v1/collections.removePosition", Why: whyGUIEdit},
	"RemovePositionsFromCollection":  {Server: "/v1/collections.removePositions", Why: whyGUIEdit},
	"ReorderCollectionPositions":     {Server: "/v1/collections.reorderPositions", Why: whyGUIEdit},
	"ReorderCollections":             {Server: "/v1/collections.reorder", Why: whyGUIEdit},
	"ReorderTournamentMatches":       {Server: "/v1/tournaments.reorderMatches", Why: whyGUIEdit},
	"ResetAnkiDeck":                  {Server: "/v1/anki.resetDeck", Why: "resetting a deck discards its review history: kept behind the GUI's confirmation"},
	"ReviewAnkiCard":                 {Server: "/v1/anki.reviewCard", Why: whyReview},
	"SaveAnalysis":                   {Server: "/v1/analyses.save", Why: "an analysis reaches the CLI through an import or `analyze`; pasting one is a GUI gesture"},
	"SaveCommand":                    {Server: "/v1/history.save", Why: whyGUIState},
	"SaveComment":                    {Server: "/v1/comments.add", Why: whyGUIEdit},
	"SaveEditPosition":               {Server: "/v1/filters.saveEditPosition", Why: whyGUIState},
	"SaveExcludePosition":            {Server: "/v1/filters.saveExcludePosition", Why: whyGUIState},
	"SaveFilter":                     {Server: "/v1/filters.save", Why: whyGUIState},
	"SaveIndividualPosition":         {Server: "/v1/positions.save", Why: "the GUI's one backend call for a position brought in on its own (ADR-0002); the CLI's position importers set the sticky flag themselves (ADR-0001)"},
	"SaveLastVisitedPosition":        {Server: "/v1/matches.setLastVisitedPosition", Why: whyGUIState},
	"SaveMetadata":                   {CLI: "edit", Why: whyMetadata},
	"SavePosition":                   {CLI: "import --type position", Server: "/v1/positions.save"},
	"SaveSearchHistory":              {Server: "/v1/searchHistory.save", Why: whyGUIState},
	"SaveSessionState":               {Server: "/v1/session.save", Why: whyGUIState},
	"SearchComments":                 {Server: "/v1/comments.search", Why: "the CLI reaches comments through `search --has-comment`; full-text search over them is the GUI's comment browser"},
	"SetMatchTournamentByName":       {Server: "/v1/tournaments.setMatchByName", Why: whyGUIEdit},
	"SetMigrationProgress":           {Why: "progress callback of the GUI's migration dialog"},
	"SetupDatabase":                  {CLI: "create", Why: "the daemon bootstraps its store at start-up (Storage.Migrate)"},
	"SuggestMatFilename":             {CLI: "export --type mat", Server: "/v1/matches.exportMat"},
	"SwapMatchPlayers":               {Server: "/v1/matches.swapPlayers", Why: whyGUIEdit},
	"SyncAnkiDeck":                   {CLI: "anki sync", Server: "/v1/anki.sync"},
	"SyncAnkiDeckWithPositions":      {CLI: "anki sync", Server: "/v1/anki.syncWithPositions"},
	"UpdateAnkiDeck":                 {Server: "/v1/anki.updateDeck", Why: whyGUIEdit},
	"UpdateAnkiDeckParams":           {Server: "/v1/anki.updateDeckParams", Why: whyGUIEdit},
	"UpdateCollection":               {CLI: "collection rename", Server: "/v1/collections.update"},
	"UpdateCommentEntry":             {Server: "/v1/comments.update", Why: whyGUIEdit},
	"UpdateFilter":                   {Server: "/v1/filters.update", Why: whyGUIState},
	"UpdateMatch":                    {Server: "/v1/matches.update", Why: whyGUIEdit},
	"UpdateMatchComment":             {Server: "/v1/matches.updateComment", Why: whyGUIEdit},
	"UpdatePosition":                 {Server: "/v1/positions.update", Why: whyGUIEdit},
	"UpdateTournament":               {Server: "/v1/tournaments.update", Why: whyGUIEdit},
	"UpdateTournamentComment":        {Server: "/v1/tournaments.updateComment", Why: whyGUIEdit},
	"Vacuum":                         {CLI: "vacuum", Server: "/v1/maintenance.vacuum"},
}

// serverPaths returns the daemon's /v1 route set, built from the same
// Server the smoke test walks (Paths() is what `call --list` prints).
func serverPaths(t *testing.T) map[string]bool {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	srv, err := server.New(server.Options{Storage: st, Logger: slog.New(slog.DiscardHandler)})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	paths := map[string]bool{}
	for _, p := range srv.Paths() {
		paths[p] = true
	}
	return paths
}

// TestDatabaseParity walks database.Database's exported methods against
// databaseParity, the CLI's handlers() (and its sub-command tables) and the
// daemon's Paths().
func TestDatabaseParity(t *testing.T) {
	c := &CLI{}
	commands := c.handlers()
	subcommands := map[string]map[string]func([]string) error{
		"collection": c.collectionHandlers(),
		"anki":       c.ankiHandlers(),
	}
	paths := serverPaths(t)

	typ := reflect.TypeOf(&database.Database{})
	seen := map[string]bool{}
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name // reflect lists exported methods only
		seen[name] = true
		entry, ok := databaseParity[name]
		if !ok {
			t.Errorf("Database.%s is neither covered nor allow-listed: add it to databaseParity with its CLI command and daemon route, or a reason for each missing mode", name)
			continue
		}

		switch {
		case entry.CLI != "":
			tokens := strings.Fields(entry.CLI)
			if _, ok := commands[tokens[0]]; !ok {
				t.Errorf("Database.%s: CLI command %q is not in handlers()", name, tokens[0])
			} else if subs, hasSubs := subcommands[tokens[0]]; hasSubs {
				if len(tokens) < 2 {
					t.Errorf("Database.%s: CLI command %q needs a sub-command", name, tokens[0])
				} else if _, ok := subs[tokens[1]]; !ok {
					t.Errorf("Database.%s: %q is not a sub-command of %q", name, tokens[1], tokens[0])
				}
			}
		case entry.Why == "":
			t.Errorf("Database.%s: no CLI command and no reason", name)
		}

		switch {
		case entry.Server != "":
			if !paths[entry.Server] {
				t.Errorf("Database.%s: route %q is not in Paths()", name, entry.Server)
			}
		case entry.Why == "":
			t.Errorf("Database.%s: no daemon route and no reason", name)
		}
	}

	for name := range databaseParity {
		if !seen[name] {
			t.Errorf("databaseParity names Database.%s, which no longer exists", name)
		}
	}
	for cmd := range subcommands {
		if _, ok := commands[cmd]; !ok {
			t.Errorf("sub-command table for %q, which is not in handlers()", cmd)
		}
	}
}
