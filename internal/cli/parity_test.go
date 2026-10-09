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
// reflection. Every exported database.Database method must appear in
// databaseParity with its CLI command and daemon route, or a reason why a mode
// lacks it; stale entries and blank columns without a reason fail too. An
// absence is a decision (ADR-0015), and this table is where it is written.

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
	whyLifecycle        = "desktop lifecycle: the CLI opens the file through every command's --db and the daemon through --dsn at start-up; no runtime open/close/lock to expose"
	whyGUIState         = "interactive GUI state (session, filter library, command and search history, last visited match): a script has nothing to read or restore there"
	whyGUIEdit          = "an editing gesture of the GUI (a position, a comment, a match's metadata or a list's order); a script imports, searches, lists and exports"
	whyTwoPhase         = "two-phase native .db import (analyse -> preview -> commit) is the shape of the GUI dialog; the CLI and the daemon import in one step (import --type database, /v1/imports.db)"
	whyIssuance         = "issuance is a person's act on a file they are producing (ADR-0007); the daemon operates on a library and signs, seals or opens nothing on anyone's behalf (ADR-0015)"
	whySubsetExp        = "the daemon's exports.sqlite writes the whole tenant, optionally watermarked with the daemon's own signing identity (Options.Identity, --identity-dir); a subset export carrying its origin is the producer's desktop gesture (ADR-0007, ADR-0015)"
	whyReview           = "reviewing a card needs the board in front of the player: a GUI gesture (the daemon serves a client that has one)"
	whyCram             = "cram mode picks a random card for the board; the daemon's review loop goes through anki.nextCard and the CLI reviews nothing"
	whyWrapper          = "internal to the desktop wrapper: run by OpenDatabase / the importers themselves, not something a caller invokes"
	whyServerEPC        = "the CLI's `epc` computes from an XGID without a database; the daemon's positions.epc takes a position"
	whyMetadata         = "the metadata table is database infrastructure, not a tenant's data (« infrastructure de la base, pas une donnée de tenant » — ADR-0005, #156): global to every tenant and outside RLS, so the daemon reads its schema version (metadata.version) and nothing else; load/save/setVersion let one tenant read the others' session state, rewrite database_version and fail /readyz for the whole instance"
	whyStoragePrimitive = "a Storage primitive the desktop reaches through a coarser call — SavePosition, or an importer's own transaction, does this inside one operation; an HTTP client has no such operation and needs the piece"
	whyEnginePure       = "a pure function of the ENGINE on one position, no storage behind it: the GUI binds it on *gui.App (ComputeCubeMatrix) and the CLI has `cubematrix`, so all three modes answer — but there is nothing for the Database wrapper to hold"
	whyEngineEvaluate   = "a pure function of the ENGINE on one bare position, nothing read from or written to the tenant: the GUI evaluates the board at rest on *gui.App (StartEvaluationAtRest) and an HTTP or MCP client needs it as a route, which the CLI reaches through `blunderdb call /v1/gammonnet.evaluate` — there is nothing for the Database wrapper to hold"
	whyTenantQuota      = "the daemon's own accounting of what each tenant takes from a SHARED instance (--quota-*): a desktop or CLI database serves one person, who has nothing to share and no quota to read"
	whySuggestion       = "a constant of the domain, exposed on the wrapper only so the frontend reads it through the same binding as everything else; the CLI prints it beside `list --type tags` and the daemon returns it in the same answer as the vocabulary"
	whyExplain          = "the explanation is a THEME plus its measured deltas, rendered into a sentence by the client in its own language; the CLI prints an analysis, not a coaching line, and would have to carry its own nine-language templates to say anything here (#298)"
	whyStudyImpact      = "a composition of two routes the daemon already serves — /v1/stats.compute over each of the two date windows, and /v1/anki.reviewsByGameType — so a client assembles it without a route of its own, and the desktop assembles it here (#275)"
	whyQuiz             = "a quiz answer is a move played ON A BOARD (or a cube action clicked): the CLI has no board, and typing the notation would be a second way of naming a move to keep in step with the generator's. The daemon carries it for the web front J.5 will need (#294)"
	whyIdentifierDecode = "decoding a position identifier: pure, no storage. The GUI and the CLI read an OGID through parser.ParsePosition, like any other pasted position; only an HTTP client needs the identifier alone as a route, symmetrically with /v1/positions.fromXGID (#260)"
	whyDuelService      = "the Duel lives in pkg/blunderdb/duel over storage.DuelStore, not on the Database wrapper; the CLI's `duel` reads it through the same service (ADR-0072 rule 11)"
	whyTranscription    = "a transcription is typed IN FRONT OF A BOARD, gesture by gesture: the CLI's `transcribe` REPLAYS a transcription rather than typing one — it reads a document and never writes a gesture into one; it finishes, abandons or opens one on a match through the panel's own methods. A script types gestures through `call transcriptions.*`, the daemon's routes (ADR-0057), so no write sub-command is added"
	// whyTranscriptionMAT: the .mat text of a DRAFT, not of a saved Match —
	// `export --type mat` renders the other object; a script reads it from
	// `call transcriptions.exportMat`.
	whyTranscriptionMAT = whyTranscription
	whyDuelGated        = "the daemon's Duel routes are served only under --duel (ADR-0072 rule 11), outside Paths()"
	whyDuelClock        = "a Duel driven by the CLI has neither Cadence nor clock: each call is its own process (ADR-0072 rule 11), so suspending, resuming and flagging are the daemon's, whose process holds the open Duel"
	whyResumeOffer      = "the offer the desktop makes when a library is opened: a transcribed match whose targeted batch was cut short (ADR-0045 §8). Nothing is stored for it, so the fact exists only at the moment someone opens the library, which is a question only an interactive session asks. The other two modes have the DEED without the offer — `analyze --match` finishes exactly that batch, and a client of the daemon reads `toAnalyze` from transcriptions.finish"
	whyPureDomain       = "a pure function of the domain, no storage behind it: the GUI and the CLI import the package and call it in Go, only an HTTP client needs it as a route"
	whyTransport        = "a shape that exists because the transport is HTTP: a streamed JSON exchange, or cancelling a job that has no process to signal"
	whyPostgresOnly     = "PostgreSQL-only, and the Database wrapper is SQLite-only (storage/postgres has no desktop face)"
	whyMatchScoped      = "the daemon serves a library, and the match-scoped sweep exists for the transcription that has just written a match (ADR-0045 §8); a client of the daemon asks for the library sweep it already has"
	whyCtxVariant       = "context.Context variant of the method above (B.13, #181): the daemon already threads its request's own context through Storage directly and never calls the Database wrapper; this one is for the CLI, whose long-running commands (search, list --type stats, export) now cancel on Ctrl-C the way analyze already did"
	whySearchIndex      = "a navigation aid of the director's console: it feeds the GUI palette so a director reaches a player, a table or an event in one gesture; the CLI and the daemon have the same data in `tournament` and expose nothing of the console (whyDirection)"
	whyDirection        = "a Direction is what a tournament DIRECTOR decided while running a tournament (ADR-0047). The daemon serves its reads (/v1/directions.*, /v1/rencontres.*, ADR-0057) and, under `serve --direction` and `call`, its gestures (whyDirectionGesture); this method is neither, a tool of the director's console; the CLI's `tournament` subcommand — list, verify, standings, page, export, move, deliberately non-interactive since Nicomaque ships its own TD console — is the other headless face. This accessor hands out the persistence the direction package runs on; it carries no capability of its own"
	whyDirectionGesture = "a gesture of a Direction or a Rencontre (ADR-0057): the daemon serves it under `serve --direction`, `call` always, both with If-Match; the CLI adds no writing sub-command, `call` is its writing face (ADR-0056 as amended)"
	whyDirectionRead    = "a read of the Direction the daemon serves (ADR-0057) to a client that draws the view itself; the CLI's `tournament` subcommand prints the sheets a script needs (list, standings, page, export, verify), not this view"
	whyTrainingJournal  = "the Training journal records what the USER asked THEMSELVES in front of a board (ADR-0040 rule 6): a session exists only because someone answered it. A client that drills a person records and reads it over /v1/training.*, which `blunderdb call` also serves; a CLI sub-command would have no session of its own to run"
)

// serverOnly is the other direction of the parity check: every /v1 and /ops
// route must be reachable from databaseParity or named below with a reason.
// Sorted by route.
var serverOnly = map[string]string{
	// One Database method per alias gesture takes the kind ("player" or
	// "event"); the daemon spells the kind in the path.
	"/v1/events.alias.list":    "ListAliases(\"event\"), the CLI's `events alias list`: the parity row names the players route of the same method",
	"/v1/events.alias.set":     "SetAlias(\"event\", …), the CLI's `events alias add`: the parity row names the players route of the same method",
	"/v1/events.alias.remove":  "RemoveAlias(\"event\", …), the CLI's `events alias remove`: the parity row names the players route of the same method",
	"/v1/events.alias.suggest": "SuggestAliases(\"event\"), the CLI's `events alias suggest`: the parity row names the players route of the same method",
	// A batch import over HTTP outlives its request, so it has a handle to
	// read and to stop. The CLI and the desktop run the same pipeline in the
	// foreground and read its progress from the call itself.
	"/v1/imports.batch.status": "the progress of a batch import running in the daemon's background: the CLI and the desktop get it from the foreground call",
	"/v1/imports.batch.cancel": "stops a batch import running in the daemon's background: the CLI's is Ctrl-C and the desktop's is CancelImport",
	// Storage primitives the desktop reaches through a coarser call. The GUI
	// and the CLI never save a bare match row or ask whether a Zobrist hash is
	// present: SavePosition and the importers do that inside one operation.
	// A client speaking HTTP has no such operation and needs the pieces.
	"/v1/analyses.loadByIds":     "the batch face of LoadAnalysis for a client that shows a page of positions at once (a feed, a search result): over HTTP one round trip per position is the cost, where the desktop shows one board at a time and reads its analysis in-process; an exporter reaches the same batch through Storage (AnalysisStore.LoadMany)",
	"/v1/anki.reviewsByGameType": "the piece the desktop composes StudyImpact from, exposed on its own so an HTTP client can compose the same figures; the desktop reaches it through Database.StudyImpact rather than by itself (#275)",
	"/v1/matches.createGame":     whyStoragePrimitive,
	"/v1/matches.createMove":     whyStoragePrimitive,
	"/v1/matches.movesByMatch":   whyStoragePrimitive,
	"/v1/matches.save":           whyStoragePrimitive,
	"/v1/positions.exists":       whyStoragePrimitive,
	"/v1/stats.matchBadges":      whyStoragePrimitive,
	"/v1/stats.tournamentBadges": whyStoragePrimitive,
	"/v1/tournaments.get":        whyStoragePrimitive,

	// Pure functions of the domain, with no storage behind them. The GUI and
	// the CLI import the package and call them in Go; only an HTTP client
	// needs them as routes.
	"/v1/gammonnet.cubeMatrix": whyEnginePure,
	"/v1/gammonnet.evaluate":   whyEngineEvaluate,
	"/v1/tenants.quota":        whyTenantQuota,
	"/v1/positions.fromOGID":   whyIdentifierDecode,
	"/v1/positions.fromXGID":   whyPureDomain,
	"/v1/positions.legalMoves": whyPureDomain,
	"/v1/search.parse":         whyPureDomain,

	// Shapes that exist because the transport is HTTP.
	"/v1/exports.json":                    whyTransport,
	"/v1/imports.json":                    whyTransport,
	"/v1/gammonnet.analyzeMissing.cancel": whyTransport,
	"/v1/rollout.filter.cancel":           whyTransport,
	"/v1/search.query":                    whyTransport,
	// get: a remote client reads a draft back after a 409, the desktop holds
	// it in its session; undo/redo: the desktop sends them through
	// ApplyTranscriptionGesture (/v1/transcriptions.apply) as gesture kinds.
	"/v1/transcriptions.get":  whyTransport,
	"/v1/transcriptions.redo": whyTransport,
	"/v1/transcriptions.undo": whyTransport,

	// The Duel is the duel.Service's over the storage contract, not a Database
	// method; the CLI reaches it through `duel list` and `duel show`.
	"/v1/duels.get": whyDuelService,

	// Backend-specific, and the wrapper is SQLite-only.
	"/ops/tenant.purge": whyPostgresOnly,
}

// databaseParity is the allow-list. Keep it sorted by method name.
var databaseParity = map[string]parityEntry{
	"AddComment":                        {CLI: "comment add", Server: "/v1/comments.add"},
	"SetCommentAuthor":                  {Why: "the signature of the comments this process writes: the desktop sets it from « Votre nom », the CLI from --author; the daemon reads X-User-Name per request instead"},
	"AddMatchToTournament":              {Server: "/v1/tournaments.addMatch", Why: whyGUIEdit},
	"BuryAnkiCard":                      {CLI: "anki card", Server: "/v1/anki.buryCard"},
	"AddPositionToCollection":           {Server: "/v1/collections.addPosition", Why: whyGUIEdit},
	"AddPositionsToCollection":          {Server: "/v1/collections.addPositions", Why: whyGUIEdit},
	"AnalyzeImportDatabase":             {Why: whyTwoPhase},
	"AnalyzeMatchWithGammonNet":         {CLI: "analyze --match", Why: whyMatchScoped},
	"AnalyzeMissingWithGammonNet":       {CLI: "analyze", Server: "/v1/gammonnet.analyzeMissing"},
	"AnalyzeStaleGammonNet":             {CLI: "analyze --stale", Server: "/v1/gammonnet.sweepStale"},
	"CancelSearch":                      {Why: "Escape on the GUI's browsed search, which holds no request of its own to drop; a CLI search is a foreground process stopped by Ctrl-C, and a daemon search stops when its client drops the request, whose context the handler threads into the scan"},
	"LoadRollouts":                      {CLI: "rollout --list", Server: "/v1/rollout.list"},
	"PositionsToRollout":                {CLI: "analyze --rollout", Server: "/v1/rollout.filter"},
	"RolloutFiltered":                   {CLI: "analyze --rollout", Server: "/v1/rollout.filter"},
	"PlanRollout":                       {CLI: "analyze --rollout", Server: "/v1/rollout.filter"},
	"PlanRolloutIDs":                    {Why: "a list on screen is the GUI's: the CLI and the daemon select by query"},
	"PositionsToRolloutIDs":             {Why: "a list on screen is the GUI's: the CLI and the daemon select by query"},
	"RunRolloutPlan":                    {CLI: "analyze --rollout", Server: "/v1/rollout.filter"},
	"RolloutPosition":                   {CLI: "rollout", Server: "/v1/rollout.position"},
	"RolloutPositions":                  {CLI: "analyze --rollout", Server: "/v1/rollout.filter"},
	"CancelImport":                      {Server: "/v1/imports.cancel", Why: "the CLI import is a foreground process: Ctrl-C is its cancel"},
	"CheckDatabaseVersion":              {CLI: "info", Server: "/v1/metadata.version"},
	"CheckSchema":                       {CLI: "verify", Why: "schema drift is what the desktop open's EnsureSchema could not add to a user's SQLite file (issue #177); the daemon's SQLite backend runs the same EnsureSchema on open, and PostgreSQL's schema comes from its versioned migrations alone — its audit is the operator's database tooling"},
	"CheckCounters":                     {CLI: "verify", Why: "game_count and move_count are what the desktop match list displays, written once from the source file; recomputing them is a verify finding, not a route"},
	"CheckConstraints":                  {CLI: "verify", Why: "the rules the fresh DDL states and SQLite cannot add to a table that already exists (CHECK, and the hash a row must not be without); a desktop file upgraded across ten versions is the only place they can be broken, and reporting them is a verify finding, not a route"},
	"CheckMatchExists":                  {CLI: "import", Server: "/v1/matches.findByHash"},
	"CheckVersion":                      {Why: whyWrapper},
	"Checkpoint":                        {Why: whyWrapper + "; truncates the WAL after an import (internal/cli/cli_import.go), not a caller-facing step"},
	"ClearCommandHistory":               {Server: "/v1/history.clear", Why: whyGUIState},
	"ClearSessionState":                 {Server: "/v1/session.clear", Why: whyGUIState},
	"Close":                             {Why: whyLifecycle},
	"CollectionCoverage":                {Why: "feeds the GUI export dialog's per-collection 'n of m positions exported' figures; the CLI exports what it is told and the daemon exports the whole tenant"},
	"CommitImportDatabase":              {Why: whyTwoPhase},
	"ComputeEPCFromPosition":            {CLI: "epc", Server: "/v1/positions.epc", Why: whyServerEPC},
	"ComputeStats":                      {CLI: "list --type stats", Server: "/v1/stats.compute"},
	"StatsReportHTML":                   {CLI: "stats report", Server: "/v1/stats.report"},
	"StatsReportHTMLCtx":                {Why: whyCtxVariant},
	"ComputeStatsCtx":                   {Why: whyCtxVariant},
	"CopyPositionToCollection":          {Server: "/v1/collections.copyPosition", Why: whyGUIEdit},
	"CountOrphans":                      {CLI: "verify", Why: "orphaned game/move/analysis rows are the aftermath of the desktop pool enforcing foreign keys on one connection in ten (issue #157); the daemon's SQLite backend has always opened through DSN() and PostgreSQL enforces its keys server-side, so a library never carried any — its integrity is the operator's database tooling"},
	"CountPositionsWithoutAnalysis":     {CLI: "analyze", Server: "/v1/gammonnet.analyzeMissing"},
	"CountMatchPositionsToAnalyze":      {CLI: "analyze --match", Why: whyMatchScoped},
	"CountPositionsWithStaleGammonNet":  {CLI: "analyze --stale", Server: "/v1/gammonnet.sweepStale"},
	"CreateAnkiDeck":                    {Server: "/v1/anki.createDeck", Why: "a deck is created from the GUI's current collection or search; the CLI lists, inspects and syncs decks"},
	"CreateCollection":                  {CLI: "collection create", Server: "/v1/collections.create"},
	"CreateLesson":                      {CLI: "lesson create", Server: "/v1/lessons.create"},
	"ListLessons":                       {CLI: "lesson list", Server: "/v1/lessons.list"},
	"GetLesson":                         {CLI: "lesson show", Server: "/v1/lessons.get"},
	"UpdateLesson":                      {CLI: "lesson edit", Server: "/v1/lessons.update"},
	"DeleteLesson":                      {CLI: "lesson delete", Server: "/v1/lessons.delete"},
	"AddLessonStep":                     {CLI: "lesson add-step", Server: "/v1/lessons.addStep"},
	"UpdateLessonStep":                  {CLI: "lesson edit-step", Server: "/v1/lessons.updateStep"},
	"RemoveLessonStep":                  {CLI: "lesson remove-step", Server: "/v1/lessons.removeStep"},
	"ReorderLessonSteps":                {CLI: "lesson reorder", Server: "/v1/lessons.reorderSteps"},
	"SetLessonStepDone":                 {CLI: "lesson done", Server: "/v1/lessons.setStepDone"},
	"LessonDoneSteps":                   {CLI: "lesson progress", Server: "/v1/lessons.doneSteps"},
	"CreateTournament":                  {Server: "/v1/tournaments.create", Why: whyGUIEdit},
	"DeleteAnalysis":                    {Server: "/v1/analyses.delete", Why: whyGUIEdit},
	"DeleteAnkiDeck":                    {Server: "/v1/anki.deleteDeck", Why: "deleting a deck discards its review history: kept behind the GUI's confirmation"},
	"DeleteCollection":                  {CLI: "collection delete", Server: "/v1/collections.delete"},
	"DeleteComment":                     {Server: "/v1/comments.deleteForPosition", Why: whyGUIEdit},
	"DeleteCommentEntry":                {Server: "/v1/comments.delete", Why: whyGUIEdit},
	"DeleteFilter":                      {Server: "/v1/filters.delete", Why: whyGUIState},
	"DeleteMatch":                       {CLI: "delete", Server: "/v1/matches.delete"},
	"DeletePosition":                    {Server: "/v1/positions.delete", Why: whyGUIEdit},
	"DeleteProtectedCopyPath":           {Why: whyIssuance + "; the CLI's `open` writes an ordinary database instead of keeping a protected copy around"},
	"DeleteSearchHistoryEntry":          {Server: "/v1/searchHistory.deleteEntry", Why: whyGUIState},
	"DeleteTournament":                  {Server: "/v1/tournaments.delete", Why: whyGUIEdit},
	"ExportCollections":                 {CLI: "collection export", Why: whySubsetExp},
	"ExportDatabase":                    {CLI: "export", Server: "/v1/exports.sqlite"},
	"ExportDatabaseCtx":                 {Why: whyCtxVariant},
	"ExportMatchMAT":                    {CLI: "export --type mat", Server: "/v1/matches.exportMat"},
	"MatchMAT":                          {CLI: "export --type mat", Server: "/v1/matches.exportMat"},
	"ExportTournaments":                 {CLI: "export --tournament-ids", Why: whySubsetExp},
	"GetAllAnkiDecks":                   {CLI: "anki decks", Server: "/v1/anki.listDecks"},
	"GetAllCollections":                 {CLI: "collection list", Server: "/v1/collections.list"},
	"GetAllComments":                    {CLI: "comment list", Server: "/v1/comments.listAll"},
	"GetAllMatches":                     {CLI: "list --type matches", Server: "/v1/matches.list"},
	"ListMatches":                       {CLI: "list --type matches", Server: "/v1/matches.list"},
	"CountMatches":                      {CLI: "list --type matches", Server: "/v1/matches.count"},
	"GetAllPlayerNames":                 {Server: "/v1/stats.playerNames", Why: "autocomplete for the GUI's player filter; `list --type players` prints every player with its figures"},
	"GetAllTournaments":                 {CLI: "list --type tournaments", Server: "/v1/tournaments.list"},
	"ListAnkiDeckPositionIDs":           {Server: "/v1/anki.deckPositionIds", Why: "the id windows the GUI browses a deck with; a script lists a deck's source (`collection show`, `search`)"},
	"CountAnkiDeckPositions":            {Server: "/v1/anki.deckPositionCount", Why: "the length of the browsed deck, read without loading it; the deck's own counters are in `anki` stats"},
	"IndexOfAnkiDeckPosition":           {Server: "/v1/anki.indexOfDeckPosition", Why: "the rank of a position in the browsed deck, so the GUI lands on it without holding the id list"},
	"ListCollectionPositionIDs":         {Server: "/v1/collections.positionIds", Why: "the id windows the GUI browses a collection with; `collection show` prints the positions themselves"},
	"CountCollectionPositions":          {Server: "/v1/collections.countPositions", Why: "the length of the browsed collection, read without loading it; `collection list` prints each collection's count"},
	"IndexOfCollectionPosition":         {Server: "/v1/collections.indexOfPosition", Why: "the rank of a position in the browsed collection, so the GUI lands on it without holding the id list"},
	"GetAnkiDeckPositions":              {Server: "/v1/anki.deckPositions", Why: "the positions of a deck are the positions of its collection or search, which `collection show` and `search` list"},
	"GetAnkiDeckStats":                  {CLI: "anki stats", Server: "/v1/anki.deckStats"},
	"GetAnkiForecast":                   {CLI: "anki forecast", Server: "/v1/anki.forecast"},
	"GetAnkiDeckRetention":              {CLI: "anki retention", Server: "/v1/anki.retention"},
	"GetAnkiReviewLog":                  {CLI: "anki log", Server: "/v1/anki.reviewLog"},
	"GetCollectionByID":                 {CLI: "collection show", Server: "/v1/collections.get"},
	"GetCollectionPositions":            {CLI: "collection show", Server: "/v1/collections.positions"},
	"GetCommentsByPosition":             {CLI: "comment list", Server: "/v1/comments.byPosition"},
	"GetDatabaseStats":                  {CLI: "info", Server: "/v1/metadata.counts"},
	"GetDatabaseStatsEstimate":          {CLI: "info", Server: "/v1/metadata.countsEstimate"},
	"GetLibrarySettings":                {CLI: "info", Server: "/v1/librarySettings.load"},
	"GetDatabaseVersion":                {CLI: "info", Server: "/v1/metadata.version"},
	"GetGamesByMatch":                   {Server: "/v1/matches.games", Why: "`match` prints a match position by position; the game/move split is the GUI navigator's and the daemon's"},
	"GetIssuanceInfo":                   {CLI: "identity", Why: whyIssuance},
	"GetLastVisitedMatch":               {Server: "/v1/matches.lastVisited", Why: whyGUIState},
	"GetMatchByID":                      {CLI: "match", Server: "/v1/matches.get"},
	"GetMatchDetailStats":               {Server: "/v1/stats.matchDetail", Why: "per-match badges of the GUI's Matches tab; `list --type stats --match` is not a filter the CLI offers, `list --type players` and `--tournament` are"},
	"GetMatchMoveGrades":                {Server: "/v1/stats.matchMoveGrades", Why: "the marks of the GUI's Transcript; the CLI reaches the same plays with `search ma<id> E>x`, whose `E` filter scores a play by the same rule (checkerPlayError, engine.CubeActionError)"},
	"GetTimeErrors":                     {CLI: "list", Server: "/v1/stats.timeErrors"},
	"GetMatchDecisionLosses":            {CLI: "match", Server: "/v1/stats.matchDecisionLosses"},
	"GetMatchReview":                    {CLI: "match", Server: "/v1/stats.matchReview"},
	"GetTournamentReview":               {CLI: "stats tournament", Server: "/v1/stats.tournamentReview"},
	"GetMatchTimeSummary":               {CLI: "match", Server: "/v1/stats.matchTimeSummary"},
	"GetMatchOrigin":                    {CLI: "match", Server: "/v1/matches.origin"},
	"GetMatchMovePositions":             {CLI: "match", Server: "/v1/matches.movePositions"},
	"GetMatchTournament":                {Server: "/v1/tournaments.tournamentOf", Why: "`list --type tournaments` lists tournaments with their matches; the reverse lookup is a GUI label"},
	"GetMovesByGame":                    {Server: "/v1/matches.moves", Why: "`match` prints a match position by position; the game/move split is the GUI navigator's and the daemon's"},
	"GetLinkedAnkiCard":                 {Server: "/v1/anki.linkedCard", Why: whyReview},
	"GetNextAnkiCard":                   {Server: "/v1/anki.nextCard", Why: whyReview},
	"GetPlayerTable":                    {CLI: "list --type players", Server: "/v1/stats.playerTable"},
	"GetPositionCollections":            {Server: "/v1/collections.collectionsOf", Why: "the GUI's 'in collections…' label; `collection show` lists the other direction"},
	"GetPositionIDsByMatch":             {Server: "/v1/stats.positionIdsByMatch", Why: "the CLI selects a match's positions with `search --match-ids`, which filters and prints them"},
	"GetPositionIDsByStatsSelection":    {Server: "/v1/stats.positionIdsBySelection", Why: "drill-down from a statistics figure to the board; the CLI's `search` takes the same filters directly"},
	"GetStatsBreakdownPositionCounts":   {Server: "/v1/stats.breakdownPositionCounts", Why: "the clickable figures of the GUI's Breakdowns tab, each the length of a drill-down; the CLI's `search` takes the same filters directly"},
	"GetPositionIDsByTournament":        {Server: "/v1/stats.positionIdsByTournament", Why: "the CLI selects a tournament's positions with `search --tournament-ids`, which filters and prints them"},
	"GetPositionIndexMap":               {CLI: "collection show", Server: "/v1/collections.positionIndexMap"},
	"GetPositionProvenance":             {Why: "the GUI's 'this position comes from matches…' tooltip; neither the CLI nor the daemon has a reader for it, and matches.movePositions covers the other direction"},
	"GetRandomAnkiCard":                 {Why: whyCram},
	"GetStatsDateRange":                 {Server: "/v1/stats.dateRange", Why: "bounds of the GUI's date picker; the CLI takes --from/--to as given"},
	"GetTournamentMatches":              {CLI: "list --type tournaments", Server: "/v1/tournaments.matches"},
	"ExplainDecision":                   {Server: "/v1/positions.explain", Why: whyExplain},
	"ListTranscriptions":                {Server: "/v1/transcriptions.list", Why: whyTranscription},
	"ListDuels":                         {CLI: "duel list", Server: "/v1/duels.list"},
	"CreateDuel":                        {CLI: "duel create", Why: whyDuelGated},
	"OpenDuel":                          {Why: whyDuelClock + "; " + whyDuelGated},
	"SuspendDuel":                       {Why: whyDuelClock + "; " + whyDuelGated},
	"FlagDuel":                          {Why: whyDuelClock + "; " + whyDuelGated},
	"PlayDuel":                          {CLI: "duel move", Why: whyDuelGated},
	"StopDuel":                          {CLI: "duel stop", Why: whyDuelGated},
	"ForfeitDuel":                       {CLI: "duel forfeit", Why: whyDuelGated},
	"ContributeDuel":                    {CLI: "duel contribute", Why: whyDuelGated},
	"DuelOffer":                         {Why: "the creation form's choices; the CLI's and the daemon's create refuse an unknown level or Cadence with the list, and `duel create --help` states them"},
	"CreateTranscription":               {Server: "/v1/transcriptions.create", Why: whyTranscription},
	"OpenTranscription":                 {Server: "/v1/transcriptions.open", CLI: "transcribe --draft", Why: whyTranscription},
	"CloseTranscription":                {Server: "/v1/transcriptions.close", Why: whyTranscription},
	"ApplyTranscriptionGesture":         {Server: "/v1/transcriptions.apply", Why: whyTranscription},
	"TranscriptionMAT":                  {Server: "/v1/transcriptions.exportMat", Why: whyTranscriptionMAT},
	"FinishTranscription":               {Server: "/v1/transcriptions.finish", CLI: "transcribe --finish", Why: whyTranscription},
	"MaterializeTranscription":          {Server: "/v1/transcriptions.materialize", CLI: "transcribe --materialize", Why: whyTranscription},
	"AbandonTranscription":              {Server: "/v1/transcriptions.abandon", CLI: "transcribe --abandon", Why: whyTranscription},
	"EditMatchTranscription":            {Server: "/v1/transcriptions.editMatch", CLI: "transcribe --edit", Why: whyTranscription},
	"MatchTranscriptionLosses":          {Server: "/v1/transcriptions.losses", CLI: "transcribe --edit", Why: whyTranscription},
	"ExportTranscriptionMAT":            {Why: whyTranscription},
	"SuggestTranscriptionMatFilename":   {Server: "/v1/transcriptions.exportMat", Why: whyTranscription},
	"PendingTranscriptionAnalysis":      {Why: whyResumeOffer},
	"GradeQuizChecker":                  {Server: "/v1/quiz.gradeChecker", Why: whyQuiz},
	"GradeQuizCheckerMove":              {Server: "/v1/quiz.gradeCheckerMove", Why: whyQuiz},
	"GradeQuizCube":                     {Server: "/v1/quiz.gradeCube", Why: whyQuiz},
	"ImportBGFMatch":                    {CLI: "import", Server: "/v1/imports.bgf"},
	"ImportOGXMMatch":                   {CLI: "import", Server: "/v1/imports.ogxm"},
	"ImportBGFPosition":                 {CLI: "import --type position", Server: "/v1/imports.position"},
	"ImportBGFPositionFromText":         {Server: "/v1/imports.position", Why: "the clipboard paste of a BGBlitz position; the CLI imports the file"},
	"ImportDatabase":                    {CLI: "import --type database", Server: "/v1/imports.db"},
	"ImportGnuBGMatch":                  {CLI: "import", Server: "/v1/imports.gnubg"},
	"ImportGnuBGMatchFromText":          {Server: "/v1/imports.gnubg", Why: "the clipboard paste of a GNUbg match; the CLI imports the file"},
	"ImportXGMatch":                     {CLI: "import", Server: "/v1/imports.xg"},
	"ImportFiles":                       {CLI: "import", Server: "/v1/imports.batch"},
	"SetSkipDuplicates":                 {CLI: "import --skip-duplicates", Server: "/v1/imports.batch"},
	"ImportXGPPosition":                 {CLI: "import --type position", Server: "/v1/positions.fromXGP"},
	"IsProtectedCopyPath":               {CLI: "open", Why: whyIssuance},
	"IsReadOnly":                        {Why: whyLifecycle + " (ADR-0004: the second desktop instance opens read-only; a CLI run is one process, the daemon owns its store)"},
	"RefuseReadOnly":                    {Why: whyLifecycle + " (the read-only guard the rollout methods run themselves; the GUI asks it to refuse before a job starts)"},
	"ListPositionIDs":                   {CLI: "list --type positions --offset", Server: "/v1/positions.listIds", Why: "the id windows the GUI browses a library with (positions are fetched by LoadPositionsByIDs); `list --type positions` prints the positions themselves, which is what a script wants"},
	"CountPositions":                    {CLI: "list --type positions", Server: "/v1/positions.count"},
	"IndexOfPosition":                   {Server: "/v1/positions.indexOf", Why: "the rank of a position in the browsed library, so the GUI lands on it without holding the id list; a script addresses positions by id"},
	"LoadAllPositions":                  {CLI: "list --type positions", Server: "/v1/positions.list"},
	"LoadAnalysis":                      {CLI: "search --format json", Server: "/v1/analyses.load"},
	"LoadCommandHistory":                {Server: "/v1/history.load", Why: whyGUIState},
	"LoadComment":                       {Server: "/v1/comments.text", Why: "the CLI prints a position's comment with `search --format json`"},
	"LoadEditPosition":                  {Server: "/v1/filters.loadEditPosition", Why: whyGUIState},
	"LoadExcludePosition":               {Server: "/v1/filters.loadExcludePosition", Why: whyGUIState},
	"LoadFilters":                       {Server: "/v1/filters.list", Why: whyGUIState},
	"LoadMetadata":                      {CLI: "info", Why: whyMetadata},
	"LoadPosition":                      {CLI: "search --position-ids", Server: "/v1/positions.load"},
	"SearchPositionIDs":                 {CLI: "search", Server: "/v1/search.ids"},
	"CountPositionsByFilters":           {Server: "/v1/search.count", Why: "the length of a search result the GUI browses by windows; `search --limit/--offset` pages the survivors and a script reads the end of the result where it ends"},
	"IndexOfPositionByFilters":          {Server: "/v1/search.indexOf", Why: "the rank of a position in a browsed search result, so the GUI lands on it without holding the id list; a script addresses positions by id"},
	"LoadPositionIDsByFilters":          {Why: "the whole id list of a search the GUI holds bounded (a sub-search within a list on screen, an Anki deck filled from a search); a plain search is browsed by SearchPositionIDs windows. The CLI's `search` and the daemon's /v1/search.find page the survivors and print or stream the positions themselves, which is what a script or a remote client wants"},
	"LoadPositionView":                  {Why: "the GUI board reads a position's analysis and comment in one binding call to save an IPC round trip; the CLI and the daemon have no round-trip cost to save and expose the two halves apart (LoadAnalysis, LoadComment)"},
	"LoadPositionsByIDs":                {CLI: "search --position-ids", Server: "/v1/positions.loadByIds"},
	"LoadPositionsByFilters":            {CLI: "search", Server: "/v1/search.find"},
	"LoadPositionsByFiltersCore":        {CLI: "search", Server: "/v1/search.find"},
	"LoadPositionsByFiltersCoreCtx":     {Why: whyCtxVariant},
	"LoadSearchHistory":                 {Server: "/v1/searchHistory.list", Why: whyGUIState},
	"LoadSessionState":                  {Server: "/v1/session.load", Why: whyGUIState},
	"AddDirectionNote":                  {Server: "/v1/directions.addNote", Why: whyDirectionGesture},
	"AttachMatchToSlot":                 {Server: "/v1/directions.attachMatch", Why: whyDirectionGesture},
	"DetachMatchFromSlot":               {Server: "/v1/directions.detachMatch", Why: whyDirectionGesture},
	"SlotOfMatch":                       {Why: whyDirection},
	"Slots":                             {Server: "/v1/directions.slots", Why: whyDirectionRead},
	"StartTranscriptionFromSlot":        {Why: whyDirection},
	"UnattachedMatches":                 {Why: whyDirection},
	"AddParticipant":                    {Server: "/v1/directions.addParticipant", Why: whyDirectionGesture},
	"AddPair":                           {Server: "/v1/directions.addPair", Why: whyDirectionGesture},
	"AttachToRencontre":                 {Server: "/v1/rencontres.attach", Why: whyDirectionGesture},
	"CreateRencontre":                   {Server: "/v1/rencontres.create", Why: whyDirectionGesture},
	"DetachFromRencontre":               {Server: "/v1/rencontres.detach", Why: whyDirectionGesture},
	"GetRencontre":                      {Server: "/v1/rencontres.get", Why: whyDirectionRead},
	"ListRencontres":                    {Server: "/v1/rencontres.list", Why: whyDirectionRead},
	"Pairs":                             {Why: whyDirection},
	"PreviewAttachToRencontre":          {Why: whyDirection},
	"RencontreOf":                       {Why: whyDirection},
	"RencontreSearchIndex":              {Why: whySearchIndex},
	"DirectionSearchIndex":              {Why: whySearchIndex},
	"RencontrePageHTML":                 {CLI: "tournament page", Server: "/v1/rencontres.pageHtml"},
	"SetRencontreBreaks":                {Server: "/v1/rencontres.setBreaks", Why: whyDirectionGesture},
	"SetRencontreTables":                {Server: "/v1/rencontres.setTables", Why: whyDirectionGesture},
	"SetEventRooms":                     {Server: "/v1/rencontres.setEventRooms", Why: whyDirectionGesture},
	"SetDirectionTables":                {Server: "/v1/directions.setTables", Why: whyDirectionGesture},
	"TablePlan":                         {CLI: "tournament tables", Why: "the effective table properties and rooms of one event; the daemon carries them in /v1/directions.get (tableSettings, rooms) and /v1/rencontres.get, not as a route of their own"},
	"SetRencontreOutputDir":             {Why: whyDirection},
	"SetRencontreTableOutOfService":     {Server: "/v1/rencontres.setTableOutOfService", Why: whyDirectionGesture},
	"TrashRencontre":                    {Server: "/v1/rencontres.trash", Why: whyDirectionGesture},
	"WriteRencontrePage":                {CLI: "tournament page", Why: whyDirection},
	"UpdatePair":                        {Server: "/v1/directions.updatePair", Why: whyDirectionGesture},
	"UpdateRencontre":                   {Server: "/v1/rencontres.update", Why: whyDirectionGesture},
	"Brackets":                          {Server: "/v1/directions.brackets", Why: whyDirectionRead},
	"Clock":                             {Server: "/v1/directions.clock", Why: whyDirectionRead},
	"CloseDirection":                    {Server: "/v1/directions.close", Why: whyDirectionGesture},
	"History":                           {Server: "/v1/directions.history", Why: whyDirectionRead},
	"ReopenDirection":                   {Server: "/v1/directions.reopen", Why: whyDirectionGesture},
	"SinceLastGesture":                  {Why: whyDirection},
	"Standings":                         {Server: "/v1/directions.standings", Why: whyDirectionRead},
	"StandingsCSV":                      {CLI: "tournament standings", Server: "/v1/directions.standingsCsv"},
	"SeasonRanking":                     {CLI: "tournament ranking", Server: "/v1/rencontres.ranking"},
	"SeasonCSV":                         {CLI: "tournament ranking", Why: "the CSV rendering of SeasonRanking; the daemon returns the same rows as JSON"},
	"CancelMatch":                       {Server: "/v1/directions.cancelMatch", Why: whyDirectionGesture},
	"EnterParticipants":                 {Server: "/v1/directions.enterParticipants", Why: whyDirectionGesture},
	"EntrySuggestions":                  {Why: whyDirection},
	"Participants":                      {Server: "/v1/directions.participants", Why: whyDirectionRead},
	"UpdateParticipant":                 {Server: "/v1/directions.updateParticipant", Why: whyDirectionGesture},
	"WithdrawParticipant":               {Server: "/v1/directions.withdraw", Why: whyDirectionGesture},
	"ReinstateParticipant":              {Server: "/v1/directions.reinstate", Why: whyDirectionGesture},
	"MakeParticipantAbsent":             {Server: "/v1/directions.makeAbsent", Why: whyDirectionGesture},
	"MakeParticipantAvailable":          {Server: "/v1/directions.makeAvailable", Why: whyDirectionGesture},
	"CorrectResult":                     {Server: "/v1/directions.correctResult", Why: whyDirectionGesture},
	"FinishedMatches":                   {Why: whyDirection},
	"LastDecision":                      {Server: "/v1/directions.lastDecision", Why: whyDirectionRead},
	"ConfirmAllProposals":               {Server: "/v1/directions.confirmAllProposals", Why: whyDirectionGesture},
	"EnterForfeit":                      {Server: "/v1/directions.enterForfeit", Why: whyDirectionGesture},
	"EnterResult":                       {Server: "/v1/directions.enterResult", Why: whyDirectionGesture},
	"MoveMatchToTable":                  {CLI: "tournament move", Server: "/v1/directions.moveMatchToTable"},
	"TableGrid":                         {Server: "/v1/directions.tableGrid", Why: whyDirectionRead},
	"RencontreTableGrid":                {CLI: "tournament hall", Why: "the Hall merges the tables of every event of a Rencontre; the daemon serves each event's grid (/v1/directions.tableGrid) and the Rencontre (/v1/rencontres.get), not yet the merged view"},
	"ConfirmProposal":                   {Server: "/v1/directions.confirmProposal", Why: whyDirectionGesture},
	"CreateDirection":                   {Server: "/v1/directions.create", Why: whyDirectionGesture},
	"FreeParticipants":                  {Server: "/v1/directions.freeParticipants", Why: whyDirectionRead},
	"StartMatchManually":                {Server: "/v1/directions.startMatch", Why: whyDirectionGesture},
	"DeleteDirection":                   {Why: whyDirection},
	"DirectionStore":                    {Why: whyDirection},
	"GetDirection":                      {Server: "/v1/directions.get", Why: whyDirectionRead},
	"PreviewDirectionConfig":            {Server: "/v1/directions.previewConfig", Why: whyDirectionGesture},
	"SetDirectionStrings":               {Why: whyDirection},
	"WriteDirectionPage":                {CLI: "tournament page", Why: whyDirection},
	"DirectionPageHTML":                 {CLI: "tournament page", Server: "/v1/directions.pageHtml"},
	"DirectionPairingSheetHTML":         {Server: "/v1/directions.pairingSheetHtml", Why: whyDirectionRead},
	"WriteDirectionPairingSheet":        {Why: whyDirection},
	"DirectionUpcomingSheetHTML":        {Why: whyDirection},
	"WriteDirectionUpcomingSheet":       {Why: whyDirection},
	"DirectionRounds":                   {Why: whyDirection},
	"Directory":                         {Server: "/v1/directions.directory", Why: whyDirectionRead},
	"DirectorySources":                  {Why: whyDirection},
	"DirectoryEntrants":                 {Why: whyDirection},
	"DirectoryCSV":                      {Why: whyDirection},
	"ParseDirectoryCSV":                 {Why: whyDirection},
	"DirectionFreeSlots":                {Why: whyDirection},
	"AddParticipantAtSlot":              {Server: "/v1/directions.addParticipant", Why: whyDirectionGesture},
	"DirectionJournalJSON":              {CLI: "tournament export", Why: whyDirection},
	"HasDirection":                      {Why: whyDirection},
	"ListDirections":                    {CLI: "tournament list", Server: "/v1/directions.list"},
	"SetDirectionConfig":                {Server: "/v1/directions.setConfig", Why: whyDirectionGesture},
	"SetDirectionOutputDir":             {Why: whyDirection},
	"LoadTrainingNumberStats":           {Server: "/v1/training.numberStats", Why: whyTrainingJournal},
	"LoadTrainingSessions":              {CLI: "training sessions", Server: "/v1/training.sessions"},
	"LoadTrainingMissed":                {CLI: "training missed", Server: "/v1/training.missed"},
	"MergePlayers":                      {CLI: "players merge", Server: "/v1/matches.mergePlayers"},
	"MovePositionBetweenCollections":    {Server: "/v1/collections.movePosition", Why: whyGUIEdit},
	"OpenDatabase":                      {Why: whyLifecycle},
	"OpenProtectedCopyPath":             {CLI: "open", Why: whyIssuance},
	"ParsePositionText":                 {CLI: "import --type position", Server: "/v1/positions.parseText"},
	"RefreshSearchStatistics":           {Why: whyWrapper},
	"RemoveMatchFromTournament":         {Server: "/v1/tournaments.removeMatch", Why: whyGUIEdit},
	"RemovePositionFromCollection":      {Server: "/v1/collections.removePosition", Why: whyGUIEdit},
	"RemovePositionsFromCollection":     {Server: "/v1/collections.removePositions", Why: whyGUIEdit},
	"ReorderCollectionPositions":        {Server: "/v1/collections.reorderPositions", Why: whyGUIEdit},
	"ReorderCollections":                {Server: "/v1/collections.reorder", Why: whyGUIEdit},
	"ReorderTournamentMatches":          {Server: "/v1/tournaments.reorderMatches", Why: whyGUIEdit},
	"ResetAnkiDeck":                     {Server: "/v1/anki.resetDeck", Why: "resetting a deck discards its review history: kept behind the GUI's confirmation"},
	"ReviewAnkiCard":                    {Server: "/v1/anki.reviewCard", Why: whyReview},
	"SaveAnalysis":                      {Server: "/v1/analyses.save", Why: "an analysis reaches the CLI through an import or `analyze`; pasting one is a GUI gesture"},
	"SaveCommand":                       {Server: "/v1/history.save", Why: whyGUIState},
	"SaveComment":                       {Server: "/v1/comments.add", Why: whyGUIEdit},
	"SaveEditPosition":                  {Server: "/v1/filters.saveEditPosition", Why: whyGUIState},
	"SaveExcludePosition":               {Server: "/v1/filters.saveExcludePosition", Why: whyGUIState},
	"SaveFilter":                        {Server: "/v1/filters.save", Why: whyGUIState},
	"SaveLibrarySettings":               {CLI: "edit --error-threshold / --blunder-threshold", Server: "/v1/librarySettings.save"},
	"SaveIndividualPosition":            {Server: "/v1/positions.save", Why: "the GUI's one backend call for a position brought in on its own (ADR-0002); the CLI's position importers set the sticky flag themselves (ADR-0001)"},
	"SaveLastVisitedPosition":           {Server: "/v1/matches.setLastVisitedPosition", Why: whyGUIState},
	"SaveMetadata":                      {CLI: "edit", Why: whyMetadata},
	"SavePosition":                      {CLI: "import --type position", Server: "/v1/positions.save"},
	"SaveSearchHistory":                 {Server: "/v1/searchHistory.save", Why: whyGUIState},
	"SaveSessionState":                  {Server: "/v1/session.save", Why: whyGUIState},
	"SaveTrainingSession":               {Server: "/v1/training.save", Why: whyTrainingJournal},
	"SearchComments":                    {Server: "/v1/comments.search", Why: "the CLI reaches comments through `search --has-comment`; full-text search over them is the GUI's comment browser"},
	"SetMatchTournamentByName":          {Server: "/v1/tournaments.setMatchByName", Why: whyGUIEdit},
	"SetBeforeSwitch":                   {Why: "hook by which the GUI stops its batches before the open file changes; the CLI and the daemon never switch file under a running job"},
	"SetMigrationProgress":              {Why: "progress callback of the GUI's migration dialog"},
	"SetupDatabase":                     {CLI: "create", Why: "the daemon bootstraps its store at start-up (Storage.Migrate)"},
	"SuggestMatFilename":                {CLI: "export --type mat", Server: "/v1/matches.exportMat"},
	"SwapMatchPlayers":                  {CLI: "players swap", Server: "/v1/matches.swapPlayers"},
	"SyncAnkiDeck":                      {CLI: "anki sync", Server: "/v1/anki.sync"},
	"SyncAnkiDeckWithPositions":         {CLI: "anki sync", Server: "/v1/anki.syncWithPositions"},
	"UpdateAnkiDeck":                    {Server: "/v1/anki.updateDeck", Why: whyGUIEdit},
	"UpdateAnkiDeckParams":              {Server: "/v1/anki.updateDeckParams", Why: whyGUIEdit},
	"UpdateCollection":                  {CLI: "collection rename", Server: "/v1/collections.update"},
	"UpdateCommentEntry":                {Server: "/v1/comments.update", Why: whyGUIEdit},
	"UpdateFilter":                      {Server: "/v1/filters.update", Why: whyGUIState},
	"SetFilterPinned":                   {Server: "/v1/filters.setPinned", Why: whyGUIState},
	"SetCollectionFilter":               {CLI: "collection filter", Server: "/v1/collections.setFilter"},
	"TogglePile":                        {CLI: "collection pile", Server: "/v1/pile.toggle"},
	"IsPositionOnPile":                  {CLI: "collection pile", Server: "/v1/pile.state"},
	"PileCollectionID":                  {CLI: "collection pile", Why: "the Pile's id, read by the GUI to open it; the daemon lists collections"},
	"FreezeCollection":                  {CLI: "collection freeze", Server: "/v1/collections.freeze"},
	"CreateLivingCollection":            {CLI: "collection create --query", Server: "/v1/collections.create"},
	"EvaluateCollection":                {CLI: "collection evaluate", Server: "/v1/collections.evaluate"},
	"StudyImpact":                       {CLI: "list --type study", Why: whyStudyImpact},
	"SimilarPositions":                  {CLI: "search --query 's like42'", Server: "/v1/positions.similar"},
	"RankPositionsByFilters":            {CLI: "search --query 's like42'", Server: "/v1/positions.similar"},
	"RankPositionIDsByFilters":          {CLI: "search --query 's like42'", Server: "/v1/positions.similar"},
	"UpdateMatch":                       {Server: "/v1/matches.update", Why: whyGUIEdit},
	"SetMatchVideoSource":               {Server: "/v1/matches.setVideoSource", Why: whyGUIEdit},
	"UpdateMatchComment":                {Server: "/v1/matches.updateComment", Why: whyGUIEdit},
	"UpdatePosition":                    {Server: "/v1/positions.update", Why: whyGUIEdit},
	"UpdateTournament":                  {Server: "/v1/tournaments.update", Why: whyGUIEdit},
	"UpdateTournamentComment":           {Server: "/v1/tournaments.updateComment", Why: whyGUIEdit},
	"RemoveAnkiCard":                    {CLI: "anki card", Server: "/v1/anki.removeCard"},
	"RepairAnalyses":                    {CLI: "repair", Server: "/v1/analyses.repair"},
	"RepairGamePhases":                  {CLI: "repair", Server: "/v1/positions.reclassifyPhases"},
	"RepairCrawfordSentinel":            {CLI: "repair", Server: "/v1/positions.repairCrawford"},
	"RebuildMatchStats":                 {CLI: "repair", Server: "/v1/stats.rebuildMatchStats"},
	"ScoreMoves":                        {CLI: "repair", Server: "/v1/matches.scoreMoves"},
	"FindDuplicateMatches":              {CLI: "repair --duplicates", Server: "/v1/matches.duplicates"},
	"ListAliases":                       {CLI: "players alias list", Server: "/v1/players.alias.list"},
	"SetAlias":                          {CLI: "players alias add", Server: "/v1/players.alias.set"},
	"RemoveAlias":                       {CLI: "players alias remove", Server: "/v1/players.alias.remove"},
	"SuggestAliases":                    {CLI: "players alias suggest", Server: "/v1/players.alias.suggest"},
	"BeginImportBatch":                  {CLI: "import", Server: "/v1/imports.xg", Why: whyBatchIsTheImport},
	"FinishImportBatch":                 {CLI: "import", Server: "/v1/imports.xg", Why: whyBatchIsTheImport},
	"ResumeImportBatch":                 {CLI: "import", Server: "/v1/imports.batch", Why: whyBatchIsTheImport},
	"PendingImportFiles":                {CLI: "import", Server: "/v1/imports.batch", Why: whyBatchIsTheImport},
	"ImportJournal":                     {CLI: "import", Server: "/v1/imports.files"},
	"ImportReport":                      {CLI: "list", Server: "/v1/imports.report"},
	"CompareWithGammonNet":              {CLI: "analyze", Server: "/v1/gammonnet.compare"},
	"CountPositionsWithForeignAnalysis": {CLI: "analyze", Server: "/v1/gammonnet.compare"},
	"ListImportBatches":                 {CLI: "list", Server: "/v1/imports.list"},
	"SetAnkiCardSuspended":              {CLI: "anki card", Server: "/v1/anki.suspendCard"},
	"Vacuum":                            {CLI: "vacuum", Server: "/ops/maintenance.vacuum"},
	"ReencodeAnalyses":                  {CLI: "reencode", Server: "/v1/maintenance.reencode"},
	"ImportMET":                         {CLI: "met --import", Server: "/v1/met.import"},
	"ListMETs":                          {CLI: "met", Server: "/v1/met.list"},
	"SetCurrentMET":                     {CLI: "met --use", Server: "/v1/met.setCurrent"},
	"AnalysisMETStatus":                 {Server: "/v1/met.ofAnalysis", Why: whyGUIEdit},
	"TrashPosition":                     {CLI: "trash", Server: "/v1/trash.deletePosition"},
	"TrashCollection":                   {CLI: "trash", Server: "/v1/trash.deleteCollection"},
	"TrashCommentEntry":                 {CLI: "trash", Server: "/v1/trash.deleteComment"},
	"TrashMatch":                        {CLI: "trash", Server: "/v1/trash.deleteMatch"},
	"RestoreFromTrash":                  {CLI: "trash", Server: "/v1/trash.restore"},
	"DiscardFromTrash":                  {CLI: "trash", Server: "/v1/trash.discard"},
	"EmptyTrash":                        {CLI: "trash", Server: "/v1/trash.empty"},
	"ListTrash":                         {CLI: "trash", Server: "/v1/trash.list"},
	"CountTrash":                        {CLI: "trash", Server: "/v1/trash.count"},
	"StudyBacklog":                      {CLI: "study queue", Server: "/v1/study.backlog"},
	"SetPositionStudied":                {CLI: "study mark", Server: "/v1/study.setStudied"},
	"ImportStudyQueue":                  {CLI: "list", Server: "/v1/imports.studyQueue"},
	"Tags":                              {CLI: "list", Server: "/v1/comments.tags"},
	"ComputeRecurringErrors":            {CLI: "stats recurring", Server: "/v1/stats.recurringErrors"},
	"ComputeRecurringErrorsCtx":         {CLI: "stats recurring", Server: "/v1/stats.recurringErrors"},
	"ComputeTrainingStats":              {CLI: "stats training", Server: "/v1/stats.training"},
	"ComputeTrainingStatsCtx":           {CLI: "stats training", Server: "/v1/stats.training"},
	"HeadToHead":                        {CLI: "stats h2h", Server: "/v1/stats.headToHead"},
	"PlayerContrast":                    {CLI: "stats contrast", Server: "/v1/stats.playerContrast"},
	"PlayerContrastCtx":                 {CLI: "stats contrast", Server: "/v1/stats.playerContrast"},
	"PRByWindow":                        {CLI: "stats windows", Server: "/v1/stats.prByWindow"},
	"PlayerRanking":                     {CLI: "stats ranking", Server: "/v1/stats.ranking"},
	"HeadToHeadCtx":                     {CLI: "stats h2h", Server: "/v1/stats.headToHead"},
	"PRByWindowCtx":                     {CLI: "stats windows", Server: "/v1/stats.prByWindow"},
	"PlayerRankingCtx":                  {CLI: "stats ranking", Server: "/v1/stats.ranking"},
	"StudyPositionIDs":                  {CLI: "stats recurring", Server: "/v1/stats.studyIds"},
	"ComputeStudyPlan":                  {CLI: "stats plan", Server: "/v1/stats.studyPlan"},
	"ComputeStudyPlanCtx":               {CLI: "stats plan", Server: "/v1/stats.studyPlan"},
	"ComputeStudyEffect":                {CLI: "stats effect", Server: "/v1/stats.studyEffect"},
	"ComputeStudyEffectCtx":             {CLI: "stats effect", Server: "/v1/stats.studyEffect"},
	"ComputeDirectionalBiases":          {CLI: "stats biases", Server: "/v1/stats.biases"},
	"ComputeDirectionalBiasesCtx":       {CLI: "stats biases", Server: "/v1/stats.biases"},
	"StudyPlanPositionIDs":              {CLI: "stats plan", Server: "/v1/stats.studyPlanIds"},
	"StudyPlanQueue":                    {CLI: "stats plan", Server: "/v1/stats.studyPlanQueue"},
	"SuggestReferencePositions":         {CLI: "collection suggest", Server: "/v1/collections.suggest"},
	"SuggestReferencePositionsCtx":      {CLI: "collection suggest", Server: "/v1/collections.suggest"},
	"CreateStudyDeck":                   {CLI: "stats recurring", Server: "/v1/anki.createStudyDeck"},
	"RecommendedTags":                   {CLI: "list", Server: "/v1/comments.tags", Why: whySuggestion},
}

// whyBatchIsTheImport: an import batch opens and closes with an import in
// every mode, never by hand; reading one back is exposed everywhere
// (ImportReport / list --type imports / imports.report).
const whyBatchIsTheImport = "opened and closed by an import, never as a gesture of its own"

// serverPaths returns the daemon's /v1 route set, built from the same
// Server the smoke test walks (Paths() is what `call --list` prints).
func serverPaths(t *testing.T) map[string]bool {
	t.Helper()
	st, err := sqlite.Open(context.Background(), ":memory:", nil)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	// The gestures of a Direction are counted: `call` serves them, and so does `serve --direction`.
	srv, err := server.New(server.Options{Storage: st, Logger: slog.New(slog.DiscardHandler), EnableDirection: true, Transcription: true})
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
	t.Parallel()
	c := &CLI{}
	commands := c.handlers()
	subcommands := map[string]map[string]func([]string) error{
		"collection": c.collectionHandlers(),
		"anki":       c.ankiHandlers(),
		"lesson":     c.lessonHandlers(),
		"study":      c.studyHandlers(),
		"stats":      c.statsHandlers(),
		"training":   c.trainingHandlers(),
		"comment":    c.commentHandlers(),
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

	// The other direction: every route the daemon serves is reachable from a
	// Database method above, or named in serverOnly with a reason.
	covered := map[string]bool{}
	for _, e := range databaseParity {
		if e.Server != "" {
			covered[e.Server] = true
		}
	}
	for path := range paths {
		if covered[path] {
			continue
		}
		if why, ok := serverOnly[path]; !ok {
			t.Errorf("route %s is served by the daemon but reachable from no Database method: add it to databaseParity, or to serverOnly with the reason it is the daemon's alone", path)
		} else if why == "" {
			t.Errorf("route %s is in serverOnly with no reason", path)
		}
	}
	for path := range serverOnly {
		if !paths[path] {
			t.Errorf("serverOnly names %s, which the daemon no longer serves", path)
		}
		if covered[path] {
			t.Errorf("serverOnly names %s, which databaseParity already covers", path)
		}
	}
	for cmd := range subcommands {
		if _, ok := commands[cmd]; !ok {
			t.Errorf("sub-command table for %q, which is not in handlers()", cmd)
		}
	}
}
