<!-- Lot du plan tasks/plan-amelioration-2026-09b/README.md. -->

# Lot B — Backend Go (database, storage, ingest, parser, CLI)

État vérifié le 2026-09-02 : `go vet` 0, `golangci-lint` 0, un seul `TODO`
(`ingest/bgfmap.go:264`), 0 `nolint`. Le prédicat `positionIsHeldSQL` est
identique dans ses trois copies (seule la prose a dérivé). Les items urgents
(DSN A.3, filtre `E` A.13) sont au lot A.

B.1 à B.9 = **étape 1 (fiabiliser)**, bugs à corriger avant toute perf.
B.10 à B.19 = **étape 2 (consolider)**, perf et dette.

---

## B.1 — Version majeure comparée en chaîne à l'import de base [S] — bug latent (#169)

`database/db_import_db.go:132` : `if importMajor > currentMajor` compare
`"10" > "9"` → faux. `compareVersions` existe (`db_migration.go:232`).
- [ ] Appeler `compareVersions` ; test avec `10.0.0` vs `2.15.0`.

## B.2 — Crawford codé en dur à `false` dans la conversion MWC→EMG GnuBG [M] — bug, échelle d'équité (#170)

`ingest/gnubgmap.go:401` : `convertGnuBGCubeMWCToEMG` appelle `GnuBGGetME(…, fCrawford=false)`.
Toutes les décisions de videau de la partie de Crawford importées d'un `.sgf`
sont converties avec la mauvaise référence. Touche l'invariant ADR-0019.
- [ ] Propager le drapeau Crawford du `gnubgparser.Game`.
- [ ] Test de parité XG↔GnuBG sur le même match (harnais `TestLuckAgreesAcrossFormats`),
      partie de Crawford incluse (fixture à produire depuis `testdata/`).
- [ ] Les bases déjà importées portent des équités fausses sur ces
      positions : documenter dans la note de release ; `verify` peut les
      compter (positions `crawford` avec `cube_error` non nul de source GnuBG).

## B.3 — `has_jacoby`/`has_beaver` jamais posés par les importeurs, mais dans le hash Zobrist [M] — bug, dédup (#171)

`ingest/xgmap.go:60`, `gnubgmap.go:97`, `bgfmap.go:199,264` ne renseignent
pas ces champs ; seul `domain/xgid.go:138,141` le fait. `engine/zobrist.go:162-166`
les hache : une position d'argent importée d'un `.xg` (Jacoby=0) et collée en
XGID (Jacoby=1) donne deux lignes. Invariant n°1 cassé sur ce cas.
- [ ] Décision (ADR court, amendement de l'ADR-0001) : **Jacoby et beaver
      sortent du hash** — ce sont des règles de la partie, pas de la
      position ; ils restent des colonnes. Alternative rejetée : les
      renseigner partout (les formats ne les portent pas tous).
- [ ] Bump `DatabaseVersion` → migration qui recalcule `zobrist_hash` et
      fusionne les doublons révélés (conserver la ligne la plus ancienne,
      re-pointer `move`, `collection_position`, `comment`, `anki_card`).
- [ ] Renseigner quand même Jacoby/beaver/Crawford depuis BGF (`bgfmap.go:264`,
      seul TODO du dépôt) et GnuBG.
- [ ] Test : même position, deux drapeaux → une ligne.

Dépendance : coordonner avec A.2 et B.7 (un seul bump 2.16.0 pour les trois).

## B.4 — Une révision Anki écrit la carte et le journal hors transaction [S] — bug (#172)

`storage/sqlite/anki_sqlite.go:449-471` (et PG) : `UPDATE anki_card` puis
`INSERT anki_review_log` séparés. `withTx`/`inTx` existent.
- [ ] Une transaction ; `due`/`last_review` illisibles (`:435-440`) remontent
      une erreur au lieu d'un zéro-time silencieux.
- [ ] Extraire la logique FSRS dupliquée (`anki_sqlite.go:424-443` et PG) en
      `domain.ScheduleNext(card, params, rating, now)`, testée une fois.

## B.5 — `analysis(position_id)` sans UNIQUE, `Save` hors transaction [M] — bug (#173)

`schema_sqlite.go:57-73`, `analyses_sqlite.go:39-75` : `SELECT` puis
`INSERT`/`UPDATE` ; deux `Save` concurrents insèrent deux lignes, `Load` en
prend une au hasard.
- [ ] Index UNIQUE + `INSERT … ON CONFLICT(position_id) DO UPDATE` (SQLite et
      PG). Migration : dédoublonner d'abord (garder la plus récente).
- [ ] `zobrist_hash NOT NULL` dans la même migration (`schema_sqlite.go:26,248` :
      NULLABLE sous UNIQUE = plusieurs NULL tolérés, exactement le bug
      `CommitImportDatabase` corrigé hier). `repairPositionsWithoutScalars`
      tourne avant.
- [ ] Contraintes `CHECK` de base : `dice_1/2 BETWEEN 0 AND 6`, `cube_value >= 0`,
      `pip_1/2 >= 0`, `bearoff_1/2 BETWEEN 0 AND 15`, `rating BETWEEN 1 AND 4`.
      Sur SQLite, une contrainte ne s'ajoute pas par `ALTER` : les poser dans
      la DDL fraîche, et dans `verify` pour les bases migrées.

Même bump que A.2/B.3.

## B.6 — Erreurs avalées qui faussent ou vident un résultat [S] — bug (#174)

- `sqlshared/stats.go:355,362` : `_ = s.DB.QueryRow(…).Scan(&snowieSumErr)` ;
  7 boucles `Compute`/`MatchDetail` font `if err := rows.Scan(…); err != nil { return }`
  depuis une closure → PR calculé sur un sous-ensemble, affiché comme vrai.
- `sqlshared/search_helpers.go:20-33,66,104-107` : une erreur SQL devient « ne
  correspond pas » — une base verrouillée fait disparaître des résultats.
- `db_gammonnet_batch.go:226,248,265` : `NewBatchSearcher` en erreur, une
  évaluation ratée, un `SaveAnalysis` en erreur : aucun log, compteur plus bas.
- `db_gammonnet_batch.go:145` : analyse illisible réputée « à jour ».
- `db_import_db.go:217,220` : `AnalyzeImportDatabase` ne compare que le
  premier commentaire (pas d'`ORDER BY`), alors que `loadCommentText` joint
  tous.
- [ ] Propager partout (les signatures le permettent) ; `slog.Warn` avec id et
      erreur dans le lot ; agréger `failed`/`refused` dans le retour (voir C.4).
- [ ] Tests : stats avec une ligne corrompue → erreur, pas un PR.

## B.7 — Import : robustesse et messages [M] — robustesse (#175)

- `parser/parser.go:68-72` : 4 langues d'analyse XG reconnues (FR/EN/JA/DE),
  échec muet (`Analysis` vide, `err == nil`) pour ES/IT/etc. alors que le
  produit est documenté en 9 langues.
- Bearoff = `15 − total` sans garde aux trois `createPositionFromX` : un
  fichier corrompu donne un bearoff négatif qui traverse Zobrist et EPC.
- `parser.go:82` : virgule→point réécrit aussi les commentaires.
- `db_import_db.go:109`, `ingest/db.go:33` : `sql.Open` sur un chemin
  inexistant crée un `.db` vide sur le disque de l'utilisateur.
- `positions_sqlite.go:152-171` : éditer une position vers un hash existant
  remonte `UNIQUE constraint failed` brut.
- [ ] Diagnostic « bloc d'analyse non reconnu (langue ?) » remonté à la GUI
      (toast) et à la CLI. **Le rapport [P9](../../docs/recherche/P9-formats-de-fichiers.md)
      tranche la liste** : XG n'existe qu'en anglais, allemand, français, espagnol,
      japonais, grec et russe — donc n'ajouter que **ES, EL, RU** (jamais IT, FI, NL, PT :
      ces interfaces XG n'existent pas). Les libellés exacts restent à relever sur des
      échantillons réels ; ne pas inventer de marqueur.
- [ ] Refuser l'import avec un message nommant le jeu et le coup si un joueur
      a ≠ 15 pions ; helper `domain.AwayScores`, `domain.CubeExponent`,
      `(*Board).RecomputeBearoff()` partagés par les trois importeurs.
- [ ] Normaliser la virgule seulement dans les champs numériques.
- [ ] `os.Stat` avant `sql.Open` (ou DSN `mode=ro`).
- [ ] Erreur de domaine « cette position existe déjà (id N) » avec proposition
      d'y aller.

## B.8 — CLI : codes de retour, `ExitOnError`, sortie JSON [S/M] — DX (#176)

- 17 `FlagSet` en `flag.ExitOnError` → `os.Exit(2)` non documenté, chemins
  d'erreur intestables.
- `cli_import.go:194-228,419` : import de positions/lot **entièrement raté** →
  exit 0 (contredit `CLI_USAGE.md:1100`).
- `--format json` absent de `import`, `export`, `verify`, `vacuum`, `analyze`,
  `create`, `edit`, `delete`, `identity` ; `importBatch` a déjà
  `BatchImportResult` structuré.
- `runSearch` fait 427 lignes (`cli_search.go:16-443`).
- `Database.Conn()` exporté donc lié par Wails (`db.go:149`) ; un appelant
  duplique `RefreshSearchStatistics`.
- [ ] `ContinueOnError` + retour d'erreur ; documenter le code 2 ou l'éliminer.
- [ ] Erreur si aucun élément importé ; `--fail-on-error` pour l'échec partiel.
- [ ] `--format json` sur les 9 commandes restantes, `import` d'abord.
- [ ] `parseSearchFlags` / `renderResults`.
- [ ] `Checkpoint()` d'intention, dé-exporter `Conn`.
- [ ] Complétion shell `blunderdb completion bash|zsh|fish` générée depuis
      `handlers()` ; installée par nfpm/PKGBUILD/Homebrew.

## B.9 — Migrations : branche morte, comparaison de texte, étapes conditionnelles [S] — dette (#177)

- `db_migration.go:188-197` : `Scan` sur `ErrNoRows` renvoie l'erreur brute, le
  message « required table X does not exist » est inatteignable.
- `db_migration.go:265` : `addColumn` compare `err.Error()` au texte du driver
  → `pragma_table_info` (motif de `schema_sqlite.go:542`).
- `db_migration.go:78-82` : étapes 1.0.0→1.6.0 « table présente ⇒ chaîne
  arrêtée » (`errStepNotApplicable`), DDL en `IF NOT EXISTS` → inconditionnelles.
- `schema_sqlite.go:192-197` : six `ALTER TABLE` dans la DDL fraîche → replier
  dans les `CREATE TABLE`.
- `schema_sqlite.go:379-388` : `EnsureSchema` dégrade en `Warn` → `verify`
  refait le diff contre la DDL de référence et le signale.
- [ ] Les cinq points, avec un test de migration par point.

---

## B.10 — La recherche streame en apparence, matérialise en réalité [M] — perf (#178)

`sqlshared/search.go:26-39` : `Find` appelle `find` qui renvoie un
`[]domain.Position` complet puis re-yield ; `ListOpts` (`storage.go:103-108`)
n'est câblé nulle part ; `Find` n'a pas de pagination. N+1 sur trois filtres
(`:645` `t"…"`, `:658` `E`, `addPosition` 2ᵉ requête `move`) : 2 000 lignes
retenues = 4 000 aller-retours.
- [ ] `Find(ctx, scope, f, ListOpts)` avec `LIMIT/OFFSET` (ou curseur par id)
      poussés en SQL ; scan + yield réels.
- [ ] Pré-charger commentaires et coups joués des ids candidats en une
      requête `IN (…)` (`forEachIn` existe).
- [ ] Bench avant/après sur `testdata/` (le job `benchmark` sert enfin, E.9).
- [ ] Exposer la pagination : `/v1/search.find` (`limit`, `cursor`), CLI
      `search --limit/--offset`, GUI (voir D.8).

Préalable de D.8 (pagination front) et de I.x (catégorisation).

## B.11 — Chemins d'import et de réparation qui chargent tout en mémoire [M] — perf (#179)

- `sqlite/analyses_sqlite.go:170-205` (et PG `:172-191`) : `RepairDenormalisedColumns`
  accumule `data` de **toutes** les analyses, contre son propre commentaire.
- `ingest/db.go:42-73` : `DBImporter.Import` matérialise la source, N+1
  analyses et commentaires ; `LoadMany` (décodage parallèle) inutilisé.
- `db_import_db.go:80,196,199,217,220` : jusqu'à 5 requêtes par position,
  jouées deux fois (analyse puis commit).
- `db_gammonnet_batch.go:265,285` : une transaction et un verrou par position,
  `LoadPosition` par position.
- `matches_sqlite.go:385-420` : `SwapPlayers` = Load + Save + UPDATE par position.
- [ ] Pages d'ids (`ORDER BY id LIMIT n`), `LoadMany`/`LoadByIDs`, écriture
      par paquets de 200 dans une transaction, jointures dans la requête
      principale.
- [ ] Bench mémoire (`-benchmem`, `runtime.ReadMemStats`) sur une base de
      50 k positions générée.

## B.12 — Compression des blobs d'analyse [M] — perf, taille de base (#180)

`engine/analysiscodec.go:27` : zlib niveau 9 par fragment, recompression à
chaque fusion pendant l'import, pas de dictionnaire.
- [ ] Étape 1 (sans changement de format) : fusionner les fragments en
      mémoire, une seule compression ; mesurer niveau 6 vs 9 (temps × taille).
- [ ] Étape 2 (format versionné) : **zstd niveau 19 avec dictionnaire partagé**, via
      `github.com/klauspost/compress/zstd` (pur Go, pas de cgo) — recommandation chiffrée
      du rapport [P11](../../docs/recherche/P11-compression-blobs.md), qui établit que le
      **dictionnaire est le levier principal** et qu'un format binaire (CBOR,
      MessagePack) ne rapporte plus que 10 à 20 % une fois la compression appliquée, donc
      **garder le JSON**. Migration par **octet de version en tête de blob** (zlib
      commence par `78 9C`, zstd par `28 B5 2F FD`), recompression par lots en tâche de
      fond, `vacuum` comme déclencheur. Bump de la version de blob, pas du schéma ; fuzz
      étendu avec un seed par format.
- [ ] `analysis_engine` **et** `analysis_depth` promus en colonnes indexées
      (bump schéma, même vague que B.3/B.5 si le calendrier le permet) :
      supprime le décodage complet de `positionIDsWithStaleGammonNet` (C.7).

## B.13 — Contexte et annulation dans le wrapper `Database` [L, progressif] — DX (#181)

135 méthodes exportées, 14 avec `ctx`, 104 `context.Background()` ; une
recherche ou un `ComputeStats` de 30 s ne peut pas être annulé depuis la GUI.
- [ ] Variantes `…Ctx` sur les trois chemins longs (recherche, stats, export),
      le front passe un `AbortController` traduit en annulation Wails (motif
      de `beginCancellableImport`).
- [ ] `db_import_common.go:50` utilise le contexte annulable disponible.
- [ ] `db.go:189,263` : vérifier `d.db.Close()`.
- [ ] Le reste famille par famille, sans date.

## B.14 — Duplication résiduelle entre backends [M] — dette (#182)

- `anki_sqlite.go` (615 l.) vs `anki_postgres.go` (659 l.) : 30 méthodes
  identiques au dialecte près ; `sqlshared` couvre déjà search/stats/comments/
  filters/session/history.
- `SwapPlayers`/`DeleteCascade` ~90 lignes identiques.
- `createPositionFromX` ×3 (B.7 les factorise).
- [ ] Porter `anki` puis `collections` dans `sqlshared` (closure dialectale sur
      `Execer`) ; laisser matches/positions/tournaments divergents.
- [ ] Les tests par backend redondants (`comments_*_test.go`,
      `collections_*_test.go`) tombent avec.

## B.15 — Découpage des fonctions et fichiers hors gabarit [M] — dette (#183)

`SearchStore.find` 638 l. (`search.go:41-679`, 6 blocs de taux copiés
`~540-620`), `StatsStore.Compute` 433 l. (`stats.go:283-715`, 12 requêtes),
`CommitImportDatabase` 306 l., `migrate_1_9_0_to_2_0_0` 320 l. Fichiers
> 600 l. : `stats.go` 1323, `export_sqlite.go` 938, `matches_postgres.go` 905,
`xgmap.go` 878, `matches_sqlite.go` 864, `position_match.go` 703.
- [ ] `find` → `buildWhere` / `scanRows` / `applyGoFilters` (struct portant
      `f`, `ana`, `ctx`) ; table `{filtre, extracteur}` pour les six taux.
- [ ] `Compute` → une fonction par section, clause de base factorisée.
- [ ] `CommitImportDatabase` → décision / écriture.
- [ ] Linter `funlen`/`gocognit` non régressif (E.6) verrouille le résultat.
- [ ] Tests unitaires directs de `buildWhere` (entrées → SQL + args) : c'est
      ce qui fait remonter `sqlshared` de 1 % (E.2).

## B.16 — Observabilité du backend [S] — dette (#184)

0 `slog.Error` (92 `Warn`, 19 `Info`, 6 `Debug`) ; la GUI logue à `Warn`
(`logging.go:9-11`) donc « database upgraded », « import analysis: toAdd/… »
sont muets ; `config.go:6,312,319` utilise `log` standard.
- [ ] Échelle : `Error` = l'utilisateur doit voir ; `Warn` = dégradé récupérable.
- [ ] `Info` par défaut en GUI ; `config.go` sur `slog`.
- [ ] Journal fichier avec rotation (G.13) reçoit tout.

## B.17 — Schéma : compteurs dénormalisés et clés étrangères manquantes [S] — dette (#185)

`match.game_count`, `game.move_count` sans garde ; `anki_review_log.deck_id`/
`position_id` sans FK ; `index_parity_test.go:41` ne compare que les noms.
- [ ] `verify` recalcule les compteurs et signale l'écart ; FK ajoutées dans la
      vague 2.16.0 ; parité d'index sur les colonnes (normalisées).

## B.18 — Grammaire de recherche côté Go [L] — DX, parité (#186)

Aucun `ParseSearchCommand` dans `pkg/` : la grammaire vit dans
`commandProcessor.js` et `searchFilterService.js` (divergentes, D.3).
`domain.SearchFilters` a 45 champs, `blunderdb search` 24 drapeaux ; motif de
damier, `mirror`, `movePattern`, `date`, `equity`, `t"…"`, `xD`, zones/blots,
tri n'existent pas en CLI ni sur `/v1`.
- [ ] Paquet `pkg/blunderdb/searchquery` : `Parse(string) (SearchFilters, []Diag)`
      + `Format(SearchFilters) string` (aller-retour), corpus de test partagé
      avec le JS (`testdata/search_query_corpus.json`, comme le corpus XGID).
- [ ] `blunderdb search --query "…"`, `/v1/search.query`.
- [ ] Le JS garde l'autocomplétion (`commandVocabulary.js`) et délègue le
      parsing au Go via un binding (ou conserve un parseur JS **généré** depuis
      le même corpus — décision à prendre en fiche).
- [ ] `parity_test.go` : les 45 champs ont chacun un jeton.

Dépendance : D.3 (parseur unique côté JS) est l'étape 1 ; B.18 l'étape 2.

## B.19 — Dépendances et build [S] — gouvernance (#187)

- `testcontainers-go` dépendance directe pour un test → ~50 modules
  indirects dans `go.sum` et la surface `govulncheck` (`cmd/serve` n'en
  importe aucun).
- `replace go-webview2 => v1.0.16` (`go.mod:117-121`) « reason never recorded ».
- Pas de directive `toolchain` ; `go-version` répété 8× en CI ; Dockerfile
  `golang:1.25-alpine` flottant ; Node 23 (ligne impaire) répété 4×.
- [ ] Sous-module `tests/postgres/` (ou build tag + `go.work`) pour isoler
      testcontainers ; sinon l'accepter et l'écrire dans `go.mod`.
- [ ] Lever le `replace` à la prochaine release Windows après smoke test du
      `.exe` ; sinon justifier.
- [ ] `toolchain go1.25.13`, `env: GO_VERSION` unique, Dockerfile épinglé au
      patch ; Node 22 LTS.

---

## Résumé du lot

| Fiche | Effort | Étape | Bump schéma |
|---|---|---|---|
| B.1, B.4, B.6, B.8, B.9, B.16, B.17, B.19 | S | 1 | — (B.17 : FK en vague 2.16.0) |
| B.2, B.3, B.5, B.7 | M | 1 | B.3, B.5 : **2.16.0** avec A.2 |
| B.10, B.11, B.12, B.14, B.15 | M | 2 | B.12 étape 2 : version de blob |
| B.13, B.18 | L | 2-3 | — |

**Une seule vague de schéma** (2.16.0) regroupe A.2 (session), B.3 (hash sans
Jacoby/beaver), B.5 (UNIQUE analysis, NOT NULL hash, CHECK), B.12 (colonnes
`analysis_engine`/`analysis_depth`), B.17 (FK). Triple synchro + migration +
test de continuité SQLite **et** PG (G.7).
