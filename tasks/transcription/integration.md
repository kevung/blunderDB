# Transcription de matchs — intégration au reste du logiciel

Ce que la fonctionnalité touche, où, et ce qu'elle ne touche pas. Les faits ont été
vérifiés dans le code le 2026-09-07 ; les chemins sont ceux de cette date.

## 1. Base de données

| Sujet | Impact | Où |
|---|---|---|
| Schéma | `DatabaseVersion` 2.20.0 → **2.21.0** ; table `transcription(id, created_at, updated_at, format_version, match_id NULL REFERENCES match ON DELETE SET NULL, label, document TEXT)` | `domain.go:45` ; `db_migration.go` (`migrationSteps`, corps `return nil` + `EnsureSchema`) ; `db_schema.go` ; `storage/sqlite/schema_sqlite.go` ; `storage/postgres/migrations/021_*.sql` ; `migration_test.go` |
| Démo | `scripts/build-demo-db.sh` à relancer : `TestDemoDatabaseHasTheCurrentSchema` diffe `sqlite_master` | `internal/gui/demo_test.go:59` |
| Export de base | `ingest/export_sqlite.go` copie par **étapes**, pas par table : ajouter `writeTranscriptions` à la liste (`:218-227`) pour l'export complet ; rien pour l'export filtré (aucune étape = aucune ligne). `sqlite.Bootstrap` crée la table vide dans toute cible | `export_sqlite.go:196-227` |
| Remplacement d'un match | Nouveau mode de `ingest.WriteMatch` : `match_id` imposé, `game`/`move` supprimés en cascade, `match` mis à jour en place, purge `deleteOrphanedPositions` ; **pas de Trash** (il n'existe d'ailleurs pas de `trash.Match` : `DeleteMatch` ne snapshotte rien aujourd'hui) | `ingest/match.go:82` ; `database/db_match.go:279-318` |
| Prédicat de rétention | Inchangé ; les trois copies restent identiques | `positionIsHeldSQL` ×3 |
| Trash | Ne connaît ni les brouillons ni le remplacement | — |
| Hachages de match | `MatchHash`/`CanonicalHash` calculés sur le graphe à chaque enregistrement ; ils servent la dédup des imports, jamais l'identité du Match transcrit (son `id`) | `ingest/match.go:85-110` |
| Sentinelle Crawford | La transcription l'écrit ; **issue jumelle** : `xgmap.go`/`gnubg.go`/`bgf.go` ne l'écrivent pas → `MatchStateFromPosition` lit post-Crawford comme Crawford, videau mort ; correction + `repair` qui rehache les positions à 1-away | `domaineval.go:325-357` |
| Résignation | Aucune colonne ; `game.winner`/`points_won` suffisent ; `ingest/gnubg.go:133` jette encore les `resign` du SGF — hors périmètre, à noter | — |

## 2. Moteur et paquet `transcript`

| Sujet | Impact |
|---|---|
| `pkg/blunderdb/transcript/` | Nouveau paquet **pur** : sa seule dépendance interne est `domain`. `ingest` importe `storage`, donc `transcript` ne l'importe pas — il rend `(*domain.Match, []*domain.Game, map[int64][]*domain.Move)`, le triplet que `ingest.RenderMAT` prend déjà (`mat_export.go:59`) et dont l'appelant fait un `ingest.MatchGraph`. API : `Document`, `Action`, `Apply(doc, Gesture) (Document, error)`, `Replay(doc, from int) Annotated`, `MatchParts(doc)`, `FromMAT(text) Document` |
| `domain.LegalMoves` | Second consommateur ; inchangé. La correspondance se fait par **plateau résultant** (`LegalPlay.Result.Board`), jamais par notation |
| `domain.Position` | Score away avec sentinelle ; `HasJacoby`/`HasBeaver` posés ; `DecisionType` selon l'Action |
| gammonNet | Aucun changement. Candidats : `EvaluatePositionImmediate` (0-ply, `internal/gui/gammonnet_eval.go:99`) sur la position du Cursor avec ses dés ; `StartEvaluationAtRest` **n'est pas** appelé par la saisie. Analyse : `AnalyzeMissingWithGammonNet` gagne un filtre « positions de ce match » (`db_gammonnet_batch.go:90`) |
| `shutdown` | `internal/gui/run.go:74` : appeler `CancelGammonNetBatch` (et l'équivalent bearoff) avant `Close()` |
| `ingest.RenderMAT` | En-têtes `[Site]`, `[Round]`, `[EventDate]`, `[Transcriber]` (le parseur les lit déjà, `matparser.go:86-91`) ; partie terminée sans dernier coup (« Wins N points » seul) ; `mat_export_test.go` étendu |

## 3. Interface Wails (`internal/gui/`)

Bindings nouveaux, tous sur `App` ou `Database` selon la règle « logique sur `Database`,
exposée aux trois modes » :

| Binding | Rôle |
|---|---|
| `ListTranscriptions() []TranscriptionSummary` | ligne « brouillon en cours » du panneau Match, liste du panneau |
| `CreateTranscription(header) (id, Annotated)` | création |
| `OpenTranscription(id) Annotated` | ouverture, Replay complet |
| `ApplyTranscriptionGesture(id, gesture) Annotated` | tous les gestes ; écrit la ligne quand une Action change |
| `SaveTranscriptionAsMatch(id) (matchID, summary)` | §4 du fonctionnel ; lance le lot ciblé |
| `ExportTranscriptionMAT(id, path)` / `SuggestTranscriptionMatFilename(id)` | export |
| `CloseTranscription(id)` | suppression de la ligne |
| `LegalMoves(pos)` | **existe déjà** (`legal_moves.go:24`, ajouté pour #294) |
| `AnalyzeMatchMissing(matchID)` | lot ciblé (ou paramètre du lot existant) |

`frontend/wailsjs/` est régénéré, jamais édité.

## 4. Frontend

| Sujet | Impact | Où |
|---|---|---|
| Onglet | entrée `transcription` dans `DEFAULT_TABS` (`labelKey`, icône SVG inline, `Ctrl+Maj+T`), branche de montage `{:else if}` | `TabbedPanel.svelte:39-52, 281-303` |
| Mode | `TRANSCRIBE` dans `statusBarModeStore` ; `enterTranscribeMode`/`exitTranscribeMode` dans `modeMachine.js`, appelés par `App.svelte` sur le changement d'onglet ; entrée = photographie de la position étudiée, sortie = restauration, comme EDIT/EPC | `modeMachine.js`, `App.svelte:203-215` |
| Store | `transcriptionStore.js` : document annoté, Cursor, état de la machine à touches, pile undo/redo (mémoire) | nouveau |
| Composants | `TranscriptionPanel.svelte` (saisie + Transcript), `TranscriptView.svelte` (deux colonnes, réutilisable par le panneau Match plus tard), `DiceTriangle.svelte` (lot 2) ; la liste des candidats réutilise `CandidateMovesTable.svelte` avec `baseline` | nouveaux |
| Plateau | position du Cursor via `positionStore` ; flèches via `selectedMoveStore` ; lot 2 : `quizPlayStore` réarmé par le panneau (le réducteur `quizPlay.js` est déjà branché dans `boardInteractions.js:318-328`) ; déplacement libre = mode EDIT existant (pose par hauteur) suivi d'une validation « ce plateau est le coup joué » | `Board.svelte`, `boardInteractions.js` |
| Raccourcis | `keyboardService.js` : branche `Ctrl+Maj+T` ; classe du panneau ajoutée à la garde de focus ; **`keyboardShortcuts.sync.test.js`** exige l'entrée dans `raccourcis.rst` | `keyboardService.js:~385` |
| Commandes | `transcribe` / `tr` dans `commandProcessor.js` et `commandVocabulary.js` (`commandVocabulary.sync.test.js`, `helpVocabulary.sync.test.js` → `cmd_mode.rst` + 8 `.po`) | — |
| Panneau Match | ligne « brouillon en cours » par brouillon (ouvre l'onglet) ; le Match enregistré s'y affiche comme les autres ; `id` stable | `MatchPanel.svelte` |
| Panneau Analyse | rien : le coup joué du Match enregistré est marqué par `playedMovePredicate` comme pour un import | — |
| Panneau Eval | rien ; l'utilisateur peut basculer et revenir, le store garde le brouillon | — |
| Stats / PR | rien à coder : le Match compte quand il est enregistré et analysé | — |
| Anki / Collections | ajouter la position du Cursor à une collection ou un paquet **n'est pas** possible avant l'enregistrement (elle n'a pas d'identité) ; après, comme partout | — |
| i18n | clés dans les **neuf** JSON ; `i18nKeys.sync.test.js`, `i18nOrphanKeys.sync.test.js` ; `colorTokens.sync.test.js` (aucun hex, aucun `#nnn` dans `<style>`), `fontScale.sync.test.js` | `frontend/src/i18n/locales/*.json` |
| e2e | specs de gestes (lot 3) ; le port se déplace par `BLUNDERDB_E2E_PORT`, déjà en place | `frontend/tests/e2e/` |

## 5. Documentation

| Fichier | Contenu | Prix estimé |
|---|---|---|
| `doc/source/manuel.rst` | section « Transcription » : le panneau, la création, la saisie, la correction, l'enregistrement, l'export, le brouillon | ≈ 45 msgid × 8 = 360 lignes de catalogue |
| `doc/source/raccourcis.rst` | `Ctrl+Maj+T` ; tableau des touches du panneau (`1–6`, `j/k`, `h/l`, `i/a/x/s`, `d/t/p/r`, Entrée, Retour, `Ctrl+Z`) | ≈ 18 msgid × 8 |
| `doc/source/cmd_mode.rst` | `transcribe` / `tr` | ≈ 3 msgid × 8 |
| `doc/source/cli.rst`, `CLI_USAGE.md` | `blunderdb transcribe` (lot 3) ; `scripts/doc-inventory.sh` bloque sinon | ≈ 10 msgid × 8 |
| `doc/source/annexe_db_scheme.rst` | version 2.21.0 seulement (note conceptuelle, pas de journal) | 1 msgid |
| `make help` | bundles régénérés ; `go test ./cmd/help-gen` | — |
| `historique.rst` | à la release, par le skill | — |

Total ≈ 75 msgid, ≈ 600 lignes de catalogue sur huit langues, dont quatre non relues ici :
c'est le prix d'une section, pas d'une page ; il est accepté parce qu'aucune page existante
ne peut porter la saisie. `scripts/doc-po-update.sh` puis `scripts/doc-i18n-check.sh`
« all translations complete ».

## 6. Ce qui n'est pas touché

CLI d'import (un `.mat` s'importe déjà) ; `serve` et le front web (ADR-0039) ; le dossier
surveillé ; le format des analyses ; le prédicat de rétention ; le Trash ; l'identité
Zobrist ; gammonNet et ses gold ; le quiz/Entraînement (le réducteur `quizPlay.js` est
réutilisé tel quel, ADR-0040 dit qu'il est « rebound unchanged »).
