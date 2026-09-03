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
- [x] Appeler `compareVersions` ; test avec `10.0.0` vs `2.15.0`.

## B.2 — Crawford codé en dur à `false` dans la conversion MWC→EMG GnuBG [M] — bug, échelle d'équité (#170)

`ingest/gnubgmap.go:401` : `convertGnuBGCubeMWCToEMG` appelle `GnuBGGetME(…, fCrawford=false)`.
Toutes les décisions de videau de la partie de Crawford importées d'un `.sgf`
sont converties avec la mauvaise référence. Touche l'invariant ADR-0019.
- [x] Propager le drapeau Crawford du `gnubgparser.Game` (`Game.CrawfordGame`,
      posé par `RU[Crawford:CrawfordGame]` en SGF).
- [x] Test de parité XG↔GnuBG sur le même match (harnais `TestLuckAgreesAcrossFormats`),
      partie de Crawford incluse : `charlot1-charlot2` est un match en 7 points
      dont la 4e partie (6-2) est la partie de Crawford dans les deux formats
      (`ingest/gnubg_crawford_test.go`).
- [x] Les bases déjà importées ne portent **aucune** équité fausse, constat
      réfuté par le code et pinné par deux tests : (1) `GnuBGGetME` prend la
      branche post-Crawford dès qu'un joueur est à 1-away, ce que le score
      d'une partie de Crawford dit toujours de lui-même — `fCrawford` vrai ou
      faux y donne le même MWC pour toute longueur de match, tout trait, tout
      videau ; (2) gnuBG n'écrit aucune analyse de videau dans la partie de
      Crawford (videau mort), donc la conversion n'y est jamais appelée. Pas
      de note de release ni de compteur `verify` : il n'y a rien à compter.

## B.3 — `has_jacoby`/`has_beaver` jamais posés par les importeurs, mais dans le hash Zobrist [M] — bug, dédup (#171)

`ingest/xgmap.go:60`, `gnubgmap.go:97`, `bgfmap.go:199,264` ne renseignent
pas ces champs ; seul `domain/xgid.go:138,141` le fait. `engine/zobrist.go:162-166`
les hache : une position d'argent importée d'un `.xg` (Jacoby=0) et collée en
XGID (Jacoby=1) donne deux lignes. Invariant n°1 cassé sur ce cas.
- [x] Décision : ADR-0028, amendement de l'ADR-0001. **Jacoby et beaver
      sortent du hash** — ce sont des règles de la *session*, pas de la
      position ; ils restent des colonnes. Alternative rejetée : les
      renseigner partout (les formats ne les portent pas tous, et deux
      sessions d'argent aux réglages différents stockeraient encore deux fois
      le même damier).
- [x] Bump `DatabaseVersion` → **2.18.0**. La migration ne décode aucun
      damier : un hash Zobrist est un XOR de clés, donc défaire un pliage,
      c'est réinjecter la même clé (`engine.RetiredFlagDelta`). Les deux clés
      restent **tirées** dans `engine.init` — les retirer décalerait toutes
      les clés suivantes et referait le hash de chaque position jamais
      stockée. Les lignes réunies par la conversion sont fusionnées sur la
      **plus ancienne** via `mergePositionInto` (déplace `move`,
      `collection_position`, `anki_card`, `anki_review_log`, `comment`,
      l'analyse si la gardée n'en a pas, et lève les marques collantes).
      Côté PostgreSQL : `migrations/014_zobrist_without_rule_flags.sql`, les
      deux clés en littéraux épinglés au moteur par
      `zobrist_retired_keys_test.go` ; seule migration non idempotente de la
      chaîne (le XOR est son propre inverse), `schema_migrations` en est le
      garde-fou.
- [x] Renseigner Jacoby/beaver depuis BGF (`useJacoby`/`useBeaver` en tête de
      fichier, uniquement en partie d'argent) : `bgfRules` remplace le
      paramètre `isCrawford` mort et clôt le seul TODO du dépôt.
      **Pas fait pour XG ni GnuBG** : les analyseurs n'exposent pas
      l'information (`gnubgparser.MatchMetadata` n'a pas le champ, `xgparser`
      non plus). Crawford n'a pas de champ sur `domain.Position` — le score le
      dit déjà — et n'en gagne pas un ici.
- [x] Tests : `TestZobristIgnoresJacobyAndBeaver` (deux drapeaux, un hash),
      `TestMigrate_2_17_0_to_2_18_0_JacobyAndBeaverLeaveTheIdentity` (les deux
      sens de fusion, la ligne intacte, le commentaire et la marque collante
      qui suivent), `TestBGFRulesReachThePosition`.

Livrée avec B.5 et B.17 dans une **seule vague de schéma, 2.18.0** (voir le
résumé du lot).

## B.4 — Une révision Anki écrit la carte et le journal hors transaction [S] — bug (#172)

`storage/sqlite/anki_sqlite.go:449-471` (et PG) : `UPDATE anki_card` puis
`INSERT anki_review_log` séparés. `withTx`/`inTx` existent.
- [x] Une transaction ; `due`/`last_review` illisibles (`:435-440`) remontent
      une erreur au lieu d'un zéro-time silencieux (`anki.ErrUnreadableTimestamp`).
- [x] Extraire la logique FSRS dupliquée (`anki_sqlite.go:424-443` et PG) en
      `anki.ScheduleNext(card, params, rating, now)`, testée une fois.
      Dans `pkg/blunderdb/anki` et non `domain` : `domain` reste sans
      dépendance (CLAUDE.md) et l'ordonnanceur tire go-fsrs. Au passage, une
      note hors 1..4 est refusée (`storage.ErrInvalid`) : go-fsrs indexait
      ses poids avec, `rating=0` sur `/v1/anki.reviewCard` paniquait.

## B.5 — `analysis(position_id)` sans UNIQUE, `Save` hors transaction [M] — bug (#173)

`schema_sqlite.go:57-73`, `analyses_sqlite.go:39-75` : `SELECT` puis
`INSERT`/`UPDATE` ; deux `Save` concurrents insèrent deux lignes, `Load` en
prend une au hasard.
- [x] Index UNIQUE + `INSERT … ON CONFLICT(position_id) DO UPDATE` (SQLite et
      PG), les deux écritures de `Save` (l'analyse et le drapeau
      `is_cube_response`) dans un `withTx`. Migration : dédoublonner d'abord
      (garder la plus récente, `MAX(id)`), puis **supprimer** l'index non
      unique du même nom — `CREATE ... IF NOT EXISTS` ne retype pas un index
      existant — pour qu'`EnsureSchema` reconstruise l'unique à sa place.
      Contrat partagé : `Analysis/SaveIsAnUpsert` (les deux backends).
- [x] Contraintes `CHECK` de base dans la DDL fraîche : `dice_1/2 BETWEEN
      0 AND 6`, `cube_value >= 0`, `pip_1/2 >= 0`, `off_1/2 BETWEEN 0 AND 15`
      (les colonnes de sortie s'appellent `off_*`, pas `bearoff_*`),
      `rating BETWEEN 1 AND 4`. Sur SQLite une contrainte ne s'ajoute pas par
      `ALTER` : elles sont posées dans la DDL fraîche et rapportées par
      `verify` (`Database.CheckConstraints`, 10 règles) pour les bases
      migrées. Côté PostgreSQL, `015` les ajoute `NOT VALID` : les nouvelles
      lignes sont contrôlées, les anciennes laissées telles quelles — même
      marché.
- [x] `zobrist_hash NOT NULL` : **refusé, et pour trois raisons mesurées sur
      le code.** (1) `EnsureSchema` ajoute une colonne manquante par `ALTER
      TABLE ADD COLUMN`, et SQLite refuse une colonne `NOT NULL` sans
      défaut : un vieux fichier sans la colonne ne la recevrait plus jamais
      et ne s'ouvrirait plus du tout (`TestImport_OldFormatExportFixture` le
      montre). (2) `repairPositionsWithoutScalars` existe précisément pour
      *trouver* les lignes sans hash et les réparer, à chaque ouverture : la
      contrainte rendrait son propre cas de test inatteignable. (3) Elle ne
      vaudrait que pour les fichiers créés après 2.18.0, soit un schéma à deux
      vitesses pour une règle que le chemin d'écriture garantit déjà. La règle
      est donc énoncée là où elle peut dire la vérité sur *toutes* les bases —
      `CheckConstraints`, affichée par `verify` — et la réparation reste le
      remède. PostgreSQL suit, pour ne pas diverger.

Livrée dans la vague 2.18.0, avec B.3 et B.17.

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
- [x] Diagnostic `parser.ErrUnrecognisedAnalysis` (« analysis block not
      recognised — XG language not supported? ») dès qu'un bloc d'analyse est
      visible (liste de coups indentée, `(G:…%)`) sans qu'aucun marqueur ne
      l'ait lu, ou que les coups sont lus mais aucune chance de gain ; remonté
      à la GUI par le toast d'erreur existant (le collage sur le plateau
      retombe sur la ligne XGID seule) et au serveur/`call` en 4xx. La CLI
      n'a pas de chemin d'import de texte XG (`import --type position` lit du
      JSON) : rien à remonter. XG n'existe qu'en EN/DE/FR/ES/JA/EL/RU (P9) ;
      ES/EL/RU **non ajoutés** faute d'échantillon vérifié, comportement
      documenté par `TestUnsupportedXGLanguageIsReportedNotSwallowed`.
- [x] `domain.AwayScores`, `domain.CubeExponent`, `(*Board).RecomputeBearoff()`
      partagés par les trois `createPositionFrom*` ; un joueur à plus de 15
      pions est refusé (`domain.TooManyCheckersError`), et l'erreur remonte
      jusqu'à `MapXG`/`MapGnuBG`/`MapBGF` en nommant le fichier, la partie et
      le coup (les mappers XG avalaient l'erreur en supprimant le coup).
- [x] Virgule acceptée comme séparateur décimal dans les captures numériques
      (`pf` lit les deux) ; le texte n'est plus réécrit, commentaires et
      ligne de version gardent leurs virgules (corpus mis à jour).
- [x] `os.Stat` avant `sql.Open` (`openExistingSQLite`) dans
      `AnalyzeImportDatabase`, `CommitImportDatabase`, `InspectIssuance` et
      `ingest.DBImporter` (`sqlite.Open` y créait même une base fraîche).
- [x] `storage.DuplicatePositionError{ExistingID}` (« this position already
      exists (id N) », `errors.Is(…, ErrConflict)`) retournée par `Update`
      des deux backends sur violation UNIQUE du hash, test de contrat commun ;
      la GUI affiche `status.positionAlreadyExistsWithId` (9 locales) et
      n'efface plus l'analyse avant que la mise à jour soit acceptée. La
      « proposition d'y aller » (navigation) reste à faire.

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
- [x] Les cinq points, avec un test de migration par point.

---

## B.10 — La recherche streame en apparence, matérialise en réalité [M] — perf (#178)

`sqlshared/search.go:26-39` : `Find` appelle `find` qui renvoie un
`[]domain.Position` complet puis re-yield ; `ListOpts` (`storage.go:103-108`)
n'est câblé nulle part ; `Find` n'a pas de pagination. N+1 sur trois filtres
(`:645` `t"…"`, `:658` `E`, `addPosition` 2ᵉ requête `move`) : 2 000 lignes
retenues = 4 000 aller-retours.
- [x] `Find(ctx, scope, f, ListOpts)` avec `LIMIT/OFFSET` poussés en SQL
      (`Dialect.LimitOffset`, les deux backends). `find` continue de
      matérialiser un `[]domain.Position` avant de re-yielder — les six
      phases de filtrage Go (masque bitboard non tight, repli mirroir,
      préchargements) ont toutes besoin du lot candidat complet avant de
      pouvoir décider une ligne — mais ce lot est désormais borné par
      `opts.Limit`, pas par la table entière : le vrai problème
      (matérialisation *non bornée*) est réglé, l'architecture "yield après
      scan complet" ne l'est pas et reste un chantier à part si besoin.
- [x] Commentaires (`loadCommentTexts`) et coups joués (`loadPlayer1Moves`)
      des ids candidats préchargés en requêtes `IN (…)` par lots de 900
      (`forEachIDBatch`), remplaçant les requêtes une-par-ligne.
- [x] Bancs avant/après sur `storage/sqlite/bench_test.go`
      (`BenchmarkSearchText`, `BenchmarkSearchMoveErrorMirror`, 5 000
      positions) : -14 % d'allocations/op sur le filtre `t"…"` (319305 →
      274205), -9 % sur `E` en recherche miroir (514306 → 469211) ; le temps
      mur est resté sur SQLite local dans le bruit de la machine (plusieurs
      builds Go concurrents d'autres chantiers), l'allocation par opération
      est la mesure fiable ici — le gain de round-trips compte surtout pour
      un backend réseau (PostgreSQL, ou un daemon `serve` chargé).
- [x] Pagination exposée : `/v1/search.find` (`limit`, `offset` dans
      `searchFindReq`), CLI `search --limit/--offset` (`cli_search.go`,
      poussés en SQL sauf avec `--error-min`/`--has-analysis`, filtrés
      après coup). GUI restée hors scope, propriété de D.8.

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
- [x] `verify` recalcule `match.game_count` et `game.move_count` à partir des
      lignes (`Database.CheckCounters`) et signale combien sont en désaccord
      et de combien au pire. **Rien n'est réécrit** : les deux compteurs
      enregistrent ce que contenait le *fichier source*
      (`GameCount: len(match.Games)`, `MoveCount: len(movesData)` dans
      `ingest`), et ce sont eux que la liste des matchs affiche — les
      remplacer par le décompte des lignes effacerait justement l'écart qu'il
      y a lieu de regarder.
- [x] FK sur `anki_review_log.deck_id` et `.position_id` dans la vague 2.18.0
      (DDL fraîche côté SQLite, `016_review_log_foreign_keys.sql` en
      `NOT VALID` côté PostgreSQL). SQLite n'ajoute aucune FK à une table
      existante : la différence est inscrite dans `migratedFromV1Allowed`
      (`schema_parity_test.go`) avec sa raison, et `CountOrphans` gagne les
      deux relations correspondantes pour compter ce qui pend dans une base
      migrée.
- [x] `index_parity_test.go` compare désormais **les colonnes et l'unicité**,
      pas seulement les noms : `tenant_id` (qui n'existe pas côté SQLite) est
      retiré, les colonnes du prédicat d'un index partiel sont repliées avec
      les colonnes de clé (`ON position(individually_imported) WHERE … = 1`
      d'un côté, `ON position (tenant_id) WHERE individually_imported` de
      l'autre), et une déclaration répétée dans plusieurs migrations est prise
      à sa **dernière** occurrence — celle avec laquelle la base finit. L'ordre
      des colonnes n'est délibérément pas comparé : il diffère légitimement dès
      que `tenant_id` mène d'un côté. 38 index SQLite, 35 PostgreSQL, trois
      exceptions justifiées.

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
| B.1, B.4, B.6, B.8, B.9, B.16, B.19 | S | 1 | — |
| B.17 | S | 1 | FK livrées en vague **2.18.0** |
| B.2, B.7 | M | 1 | — |
| B.3, B.5 | M | 1 | livrées en vague **2.18.0** |
| B.10, B.11, B.12, B.14, B.15 | M | 2 | B.12 étape 2 : version de blob |
| B.13, B.18 | L | 2-3 | — |

**Une seule vague de schéma**, livrée en **2.18.0** — et non 2.16.0 : A.2
(session) est partie seule en 2.17.0 avant celle-ci. La vague regroupe B.3
(hash sans Jacoby/beaver), B.5 (UNIQUE analysis, CHECK ; le `NOT NULL` sur le
hash est refusé, raisons en fiche) et B.17 (FK du journal de révision).
Trois commits, une seule version de schéma : trois versions successives
auraient imposé trois régénérations de la base de démonstration et trois
migrations à l'ouverture pour un même lot de corrections. B.12 (colonnes
`analysis_engine`/`analysis_depth`) n'est pas dedans et demandera sa propre
vague. Triple synchro faite : `database` (étape + DDL), `storage/sqlite`
(schéma frais), `storage/postgres` (migrations 014, 015, 016), plus
`migration_test.go` et le test de continuité de la chaîne.
