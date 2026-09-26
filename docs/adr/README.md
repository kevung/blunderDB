# Architecture decision records

One file per decision, numbered in the order they were taken; the number is a stable
identifier, cited across the code and the documentation.

A record states the **current** decision, on one page: context, decision, consequences
(including the alternatives rejected), and the guard that holds it. It carries no dates and no
narrative — git keeps the history. A later change **rewrites** the record it changes instead of
stacking an amendment on it, and each rule lives in exactly one record: another record cites it
(`see ADR-0018 rule 2`) rather than restating it. A record is never deleted or renumbered, and a
rule number cited elsewhere keeps designating the same rule.

Read the record that governs a subsystem before changing it; `CLAUDE.md` names the invariants
that come out of them.

| # | Decision | Governs |
|---|---|---|
| [0001](0001-individually-imported-is-a-sticky-flag.md) | Individually-imported provenance is a sticky boolean, not a source enum | `individually_imported`, the `i` filter, retention |
| [0002](0002-gui-position-write-goes-through-one-backend-call.md) | Saving the board position is one backend call, not a frontend existence check | `:w` → `SaveIndividualPosition` |
| [0003](0003-search-picks-matches-tournaments-gui-keeps-ids-elsewhere.md) | GUI search picks matches/tournaments from a list; CLI and command line keep raw IDs | SearchPanel match/tournament picker |
| [0004](0004-host-capabilities-are-detected-and-fall-back-never-assumed.md) | Host capabilities are detected and fall back, never assumed | host capabilities: probe + pure policy |
| [0005](0005-serve-daemon-delegates-authentication.md) | The serve daemon performs no authentication and trusts X-Tenant-ID | `serve` daemon, `X-Tenant-ID`, tenant format |
| [0006](0006-source-tool-study-marks-are-position-properties.md) | Source-tool study marks are a sticky position property, not an auto-filled collection | XG `flagged` mark, the `fl` filter |
| [0007](0007-watermarks-mark-origin-and-nothing-else.md) | Watermarks mark origin and nothing else; the recipient's side records nothing | `issuance`: watermark, `.dbx` container |
| [0008](0008-one-type-scale-and-controls-inherit-it.md) | One type scale for the interface, and form controls inherit it | type scale, `frontend/src/style.css` |
| [0009](0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md) | Race win chances are read or convolved; cube verdicts are never estimated | `engine/race`: convolution, verdict never estimated |
| [0010](0010-luck-is-a-fact-of-the-move-never-recomputed.md) | Luck is a fact of the move, read from the source file, never recomputed | `move.luck_mp`, luck read at import |
| [0011](0011-gammonnet-is-ported-to-go-and-the-representation-boundary-sits-at-the-evaluator-s-edge.md) | gammonNet is ported to Go, and the representation boundary sits at the evaluator's edge | Go port of gammonNet, upstream rule, `EngineVersion` |
| [0012](0012-a-race-has-three-regimes-exact-evaluated-and-never-estimated.md) | A race has three regimes: exact, evaluated, and never estimated | the three race regimes |
| [0013](0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md) | Evaluations fill gaps; an imported analysis is never overwritten | batch gammonNet writes, never over an import |
| [0014](0014-the-integration-gate-is-judged-by-xg-not-by-the-gnubg-fixtures.md) | The integration gate is judged by XG, not by the gnubg fixtures | `TestIntegrationGate`, XG as arbiter |
| [0015](0015-blunderdb-serve-operates-on-a-library-it-does-not-expose-an-evaluator.md) | `blunderdb serve` operates on a library; it does not expose an evaluator | `serve` surface: no evaluation endpoint |
| [0016](0016-the-referential-is-a-property-of-the-position-and-the-search-honours-it.md) | The referential is a property of the position, and the search honours it | money/match referential, `MatchStateFromPosition` |
| [0017](0017-the-panel-shows-position-facts-plus-the-one-decision-the-board-asks.md) | The panel shows position facts, plus the one decision the board asks | Eval panel: facts vs. the one decision |
| [0018](0018-the-axis-of-a-fact-follows-the-list-it-is-read-against.md) | The axis of a fact follows the list it is read against | pre-roll axis, table idiom, Défi |
| [0019](0019-a-displayed-match-equity-is-normalised.md) | A displayed match equity is normalised — the MWC scale stays inside the engine | normalised equity at the edge, `EquityScale` |
| [0020](0020-a-cube-decision-has-one-shape-whatever-regime-produced-it.md) | A cube Decision has one shape, whatever regime produced it | shape of the cube Decision block |
| [0021](0021-position-facts-are-two-stacked-blocks-and-slack-is-never-spread.md) | Position facts are two stacked blocks on one grid, and slack is never spread between blocks | `PositionFactsTable` layout |
| [0022](0022-the-live-cube-curve-runs-to-the-average-win-not-to-the-cash-equivalent.md) | The live cube curve runs to the average win, not to the cash equivalent | Janowski live tails in `cube.go` |
| [0023](0023-the-search-values-its-leaves-with-the-cube.md) | The search values its leaves with the cube | cubeful leaves, `ConfigForPosition` |
| [0024](0024-the-evaluator-batches-positions-one-per-lane-and-keeps-the-scalar-sum-order.md) | The evaluator batches positions, one per lane, and keeps the scalar sum order | batched bit-identical kernel, parallelism |
| [0025](0025-a-review-card-asks-one-question-and-expects-one-grade.md) | A review card asks one question and expects one grade | Anki review card, mask, `AnalysisView` |
| [0026](0026-a-finite-deck-is-paced-by-the-session-not-by-the-day.md) | A finite deck is paced by the session, not by the day | Anki deck pacing, `session_limit` |
| [0027](0027-bearoff-databases-are-generated-not-shipped-and-verified-against-gnubg.md) | Bearoff databases are generated, not shipped, and verified against gnubg | `engine/bearoffgen`: generation, verification |
| [0028](0028-jacoby-and-beaver-are-rules-of-the-session-not-of-the-position.md) | Jacoby and beaver are rules of the session, not of the position | Zobrist excludes `has_jacoby`/`has_beaver` |
| [0029](0029-cube-efficiency-is-measured-per-cube-state-and-read-at-the-root.md) | Cube efficiency is measured per cube state, and read at the root until gammonNet says otherwise | cube efficiency per cube state |
| [0030](0030-analysis-blobs-are-zstd-with-a-shared-dictionary-and-a-blob-names-its-own-codec.md) | Analysis blobs are zstd with a shared dictionary, and a blob names its own codec | `analysiscodec.go`: zstd + dictionary |
| [0031](0031-one-colour-palette-and-the-migration-is-progressive.md) | One colour palette, and the migration is progressive | interface colour palette |
| [0032](0032-the-cube-level-inversion-becomes-a-closed-form-upstream.md) | The cube's level inversion becomes a closed form, and that is written upstream | closed-form `levelSolve`, shipped upstream |
| [0033](0033-the-evaluator-has-one-arithmetic-contract-no-gpu-no-wasm-kernel.md) | The evaluator has one arithmetic contract: no GPU, no WebAssembly kernel | no GPU or WASM evaluator backend |
| [0034](0034-the-in-app-help-is-generated-from-the-documentation.md) | The in-app help is generated from the documentation | `cmd/help-gen`: generated in-app help |
| [0035](0035-the-game-phase-is-derived-never-edited-and-recomputable.md) | The game phase is derived, never edited, and recomputable | `gamephase.go`, `position.game_phase` |
| [0036](0036-the-trash-is-a-snapshot-table-not-a-deleted-at-column.md) | The trash is a snapshot table, not a deleted_at column | `trash` table, restore, purge |
| [0037](0037-blunderdb-does-not-play-backgammon.md) | blunderDB does not play backgammon | product scope: no play mode |
| [0038](0038-a-named-theme-carries-the-board-palette-and-the-user-still-has-the-last-word.md) | A named theme carries the board palette, and the user still has the last word | named themes, board palette |
| [0039](0039-the-web-front-is-read-mostly-and-its-perimeter-is-locked.md) | Le front web est en consultation, son périmètre est verrouillé, et il est éteint par défaut | web front `internal/server/webui` |
| [0040](0040-training-is-a-tab-of-exercises-it-drills-what-is-calculated-anki-keeps-what-is-retained.md) | Training is a tab of exercises: it drills what is calculated, Anki keeps what is retained | Training tab, `training_*` log |
| [0041](0041-a-training-question-is-played-out-from-a-seed-not-drawn-at-random.md) | A training question is played out from a seed, not drawn at random | `engine/training`: seed + plies |
| [0042](0042-an-anki-card-can-be-a-score.md) | An Anki card can be a score | Anki score cards, `scores` deck |
| [0043](0043-a-neighbour-is-the-same-problem-nearby-and-like-is-a-ranking-token-of-the-search-grammar.md) | A neighbour is the same problem nearby, and `like` is a ranking token of the search grammar | `like` token, `sqlshared.Similar` |
| [0044](0044-transcribing-a-match-is-not-playing-one.md) | Transcribing a match is not playing one | transcribing ≠ playing |
| [0045](0045-a-transcription-is-a-draft-that-owns-its-match.md) | A transcription is a draft that owns its match | `pkg/blunderdb/transcript`: the draft owns its match |
| [0046](0046-error-and-blunder-thresholds-are-library-settings.md) | Error and blunder thresholds are library settings | error/blunder thresholds per library |
| [0047](0047-directing-a-tournament-creates-its-matches-before-they-are-played.md) | Diriger un tournoi, c'est créer ses Matchs avant qu'ils soient joués | tournament Direction, `pkg/blunderdb/direction` |
| [0048](0048-le-panneau-de-transcription-est-ordonne-par-la-frequence-non-par-le-voisinage.md) | Le panneau de transcription est ordonné par la fréquence, non par le voisinage | Transcription panel surface |
| [0049](0049-une-transcription-se-corrige-la-ou-le-curseur-est-et-le-transcript-montre-ce-qui-est-tape.md) | Une transcription se corrige là où le curseur est, et le transcript montre ce qui est tapé | in-place correction, drawn Entry |
| [0050](0050-supprimer-recule-sur-la-decision-precedente-et-une-partie-rouverte-se-continue-sur-place.md) | Supprimer recule sur la décision précédente, et une partie rouverte se continue sur place | Delete steps back; reopened game |
| [0051](0051-la-derniere-action-se-tape-comme-le-bout-du-document.md) | La dernière Action se tape comme le bout du document | typing on the last Action |
| [0052](0052-le-coup-se-joue-directement-au-plateau-et-se-tape-dans-sa-cellule.md) | Le coup se joue directement au plateau, et se tape dans sa cellule | move played on the board, editable cell |
| [0053](0053-le-score-d-une-partie-peut-etre-annonce-et-la-partie-est-jouee-a-ce-score.md) | Le score d'une partie peut être annoncé, et la partie est jouée à ce score | announced score, `set_score` |
| [0054](0054-le-trou-d-un-double-trait-est-une-case-et-seule-une-ecriture-deplace-le-curseur.md) | Le trou d'un double trait est une case, et seule une écriture déplace le curseur | double-turn hole, `HoldCursor` |
| [0055](0055-le-contexte-d-un-agent-est-un-budget-borne.md) | Le contexte d'un agent est un budget borné | agent context budget, `.claude/` |
