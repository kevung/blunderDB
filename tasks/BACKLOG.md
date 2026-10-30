# Backlog — suivis ouverts et chantiers de fond

La seule liste des suivis ouverts du dépôt. Un item **ouvert** est rangé par domaine ; un item
**fait** passe dans la section *Historique* avec sa date et son commit. Les gros chantiers
méritent chacun leur propre décision (et éventuellement leur ADR) avant lancement — ne pas les
commencer « en passant ». Aucun plan en cours ne priorise ces items : chacun se reprend par sa
propre issue le jour où il est lancé.

## Ouvert — Backend

- **Statistiques : le filtre `--engine` par les cellules** — décidé : différé (2026-10-24). Il
  lit encore les décisions (copie de la sélection) ; passer par `match_stats_cell` demande le
  moteur dans la clé des cellules, autant de cellules en plus que de moteurs par match. Mesure
  de départ : `tasks/mesure-bmab-0.37.md` § 7.2 ter.
- **Bureau sur un tenant distant** (ADR-0100, ADR-0095). `Database.store` est typé
  `storage.Storage` et les opérations de fichier passent par `storage.LocalFile`, mais le
  wrapper suppose encore un fichier SQLite ouvert : « une base est ouverte » se lit `d.db !=
  nil` (environ 150 gardes) et l'ouverture (`SetupDatabase`/`OpenDatabase`) ne construit que
  `sqlite.New`. Restent en SQL sur `d.db` :
  - à déléguer au contrat : la lecture de version (`CheckVersion`, `CheckDatabaseVersion`,
    `db_schema.go`, vers `Metadata().Version`, qui rend `ErrNotFound` sur une ligne absente
    là où le wrapper rend `sql.ErrNoRows`), le repli table par table de `GetDatabaseStats`
    (`db_match.go`, vers `Metadata().Counts`), le dernier brouillon de transcription
    (`db_transcription_save.go`) ;
  - propres au fichier, à mettre derrière `LocalFile` ou à garder sur le handle : la session
    d'import en bloc `sqlite.BeginBulk` (`db_import_pipeline.go`), la vérification
    (`CountOrphans`, `CheckSchema`, `CheckConstraints`, `CheckCounters`, `db_verify.go`),
    `Exchange`/`Borrow` qui partagent le pool, et `RawConn`.
  Ouvrir un tenant distant demande une garde d'ouverture sur `store`, un point d'entrée qui
  reçoit une `storage.Storage`, les lectures du premier groupe par le contrat, et les
  opérations du second refusées (comme `Vacuum`, `errVacuumNotLocal`) sur un store sans
  `LocalFile`.
- **`migrate.Run` par lots avec reprise** (aujourd'hui une transaction unique pour toute la
  copie SQLite→PG, `migrate/migrate.go`) — attendre un besoin terrain.
- **`move.error_mp` sans lecteur ni tenue** (`tasks/mesure-bmab-0.37.md` § 9) : la 2.50 la
  garde inerte (jamais écrite ni lue) ; la version minimale de lecture
  (`min_reader_version`, posée à 2.50.0) arrête désormais les binaires qui la lisent, mais
  pas les 0.46.x et antérieures, qui l'ignorent : à peser avant la suppression réelle.
  Si `E` devait un jour s'y appuyer (SQL exact, sans phase Go), il faudrait
  une tenue à jour à chaque écriture d'analyse ou de coup et un moyen de distinguer « pas noté »
  de « non notable » (NULL ambigu).
- **Serveur : quota en octets sous PostgreSQL** — décidé : différé (2026-10-24). La mesure
  repose sur `pg_total_relation_size` et `reltuples`, globaux à la base et non au locataire, donc
  un canal auxiliaire faible (un locataire devine la taille des autres) ; et les 13 `COUNT` par
  appel ont un coût. À faire : compter par locataire, avec un cache ou un compteur tenu à jour.
- **Import XG, métadonnées sans colonne** (ADR-0067) — décidé : différé (2026-10-24) : les
  commentaires d'en-tête et de pied de match, l'horloge et la table d'équité (MET) ne sont pas
  importés. Cible : `match.metadata`, une fois les champs exposés dans xgparser (dont
  `parseCommentSegment` pour les commentaires).
- **File d'étude transversale — la requête de la grammaire qui la reproduit.**
  La file (`StudyBacklog`, `sqlshared/importbatches.go`) n'a pas de jeton
  équivalent, pour trois raisons : elle compte les décisions comme Stats (`countedExpr` :
  coups forcés exclus, conventions du pas-de-double), ce que la grammaire n'exprime pas ;
  son coût est celui de l'analyse (`statsErrExpr`), alors que `E` filtre l'erreur du coup
  joué, coup par coup ; l'absence de carte, de collection et de marque « vu » n'a pas de
  jeton (`xco` couvre seulement le commentaire). Un jeton `nt` (« non traité ») couvrirait
  le dernier point ; l'égalité exacte demande en plus de choisir lequel des deux sens du
  coût et du décompte fait foi, puis de redéfinir la file sur la grammaire. Cette décision
  passe par Opus, avec un test d'égalité jeton ↔ `StudyBacklog` sur les deux backends.
  Décidé : différé (2026-10-24), après le filtre `aw` (livré).

- **Export `.apkg` : le démon garde tout en mémoire** (`pkg/blunderdb/apkg`). Les analyses se
  chargent par lot et les plateaux s'écrivent au fil du zip, mais le démon tamponne encore le
  paquet entier avant de répondre et la collection n'est pas écrite en flux : la mémoire d'un
  gros paquet n'est pas bornée.

- **Blobs « Double No »** : les blobs déjà stockés avec la graphie « Double No » ne sont pas
  réparés (le front la tolère).
- **Verrou mono-écrivain et `serve` SQLite.** `sqlite.Open` passe la porte de swap du vacuum
  mais ne prend pas le verrou mono-écrivain (ADR-0004) : le démon écrit à côté d'une instance
  de bureau. Le prendre rendrait le bureau lecture seule tant que le démon tourne. Décision de
  l'utilisateur (lot 2026-10-25b) : pas de verrou mono-écrivain pour `serve` SQLite ; l'entrée
  reste ouverte.

## Ouvert — Moteur

Les trois premiers items ne demandent pas de code dans blunderDB tant que gammonNet (amont) ou
une machine arm64 ne les a pas débloqués ; le registre des décisions amont et ses mesures est
`tasks/plan-amelioration-2026-09b/AMONT-GAMMONNET.md`.

- **Réseau distillé (60-100 k MAC)** — priorité amont n° 1 : P4 mesure ×5-9, le seul gain qui
  survit à la recherche. Décision amont gammonNet ; une nouvelle Configuration, donc une jauge
  de force avant adoption, et toutes les bases périmées (`analyze --stale` existe pour ça).
- **Réseau de course dédié** : attend un second réseau, donc un entraînement amont.
  L'aiguillage existe déjà ici (`engine.ClassifyGameType`, `engine/gametype.go` : `race` et
  `crunch` aux frontières de gnubg).
- **Noyau NEON arm64** pour l'inférence groupée : hors périmètre faute de machine arm64. Le
  juge existe (`engine/gammonnet/kernel_identity_test.go`, égalité bit pour bit) ; manque la
  mesure sur une machine arm64.
- **Beaver dans les rollouts** (ADR-0060 règle 4) : le rollout décide par `gammonnet.Decide`
  sans la règle Beaver (`engine/rollout/game.go`), alors que l'analyse directe la lit sous
  `has_beaver` en money (`DecideForSession`). Sa ligne double/prend est celle d'une simple
  prise (documenté dans `manuel.rst`). À faire : jouer la séquence beaver → raccoon dans la
  boucle de partie du rollout (videau à 4c puis 8c, propriétaire suivi), puis lire
  `HasBeaver` comme l'évaluateur.
- **Libellé Beaver dans le verdict** : quand la meilleure réponse est un beaver,
  `cubeActionLabel` (`engine/gammonnet/domaineval.go`) dit « Double, Take » (`DecideForSession`
  replie le beaver dans la branche double/prend) ; seule l'équité de la ligne en tient compte.
  À faire : un libellé « Double, Beaver » lu par `normalizeCubeAction` (`frontend/src/utils/cubeAction.js`),
  `CanonicalCubeAction`, `BestCubeVerdict` et une clé `cube.verdicts` traduite.
- **Refuser de démarrer sans drapeau explicite « derrière un proxy »** (ADR-0005, option
  différée) : friction pour tout déploiement légitime pour attraper une erreur que la doc,
  `--help` et le Dockerfile signalent déjà. À revisiter si un déploiement nu survient
  réellement.

## Ouvert — Frontend

- **`MatchPanel.svelte` (1 772 lignes) : extraire la fiche dans un composant `MatchSheet`**,
  avant que deux panneaux puissent coexister. Le panneau Tournois héberge un `MatchPanel`
  (`embeddedMatchId`) et chaque effet global y est coupé par un garde `embedded` (liste, tri
  partagé, clavier, ids DOM `match-tab-*` en double) ; un effet ajouté sans garde touche l'état
  du vrai panneau. L'extraction du volet de détail règle les deux.

- **Restes de GB-C14 (libellés, vocabulaire, gestes)** : libellés de groupes de la barre
  d'outils non faits ; en-tête de panneau non factorisé dans `PanelTable` ; noms de touches en
  prose non vérifiés dans les langues (« SHIFT-Enter » dans les `.po` ru, ja, el et en) ; j/k ne
  déplacent pas le surlignage du panneau Tournois ; vérifier qu'un « Too good to double » importé
  n'est plus lu comme *no double*. Le bouton « Élargir » est abandonné (décision du
  2026-10-24).

## Ouvert — Tests / CI

- **Smoke test GUI sur une vraie base de production migrée** : parcourir chaque filtre de la
  fenêtre de recherche. Priorité basse ; couvert indirectement par
  `searchFilterService.test.js` (unitaires + round-trip).
- **Le CLI écrit sur `os.Stdout`, pas sur un `io.Writer`** : ~1 500 `fmt.Print*` répartis
  dans `internal/cli/` et un `captureStdout` de test qui remplace `os.Stdout` — un état global
  du processus, donc la plupart des tests du paquet ne peuvent pas prendre `t.Parallel()`.
  Le geste est de donner au `CLI` un `out io.Writer` (défaut `os.Stdout`) et de faire passer
  les écritures par lui ; il touche beaucoup de fichiers mais aucun comportement, et il
  débloque aussi les tests de sortie `--format json` qui aujourd'hui sérialisent. Chantier à
  part, pas « en passant ».
- **`wails generate module` omet `storage.ElsewherePlay` et `storage.ElsewhereNeighbour`** :
  les deux classes sont ajoutées à la main dans `frontend/wailsjs/go/models.ts` et une
  régénération les efface. À garder à chaque régénération, ou à corriger côté générateur.
- **Suites longues** : `go test -race` du paquet `database` prend ~43 min seul, et les suites
  gammonNet en `purego` ~40 min, au-delà des délais locaux usuels. Pour exécuter le noyau arm64
  sans machine dédiée, un `qemu-aarch64` statique (tiré de l'image `tonistiigi/binfmt`) suffit,
  en passant le binaire deux fois (`qemu-aarch64 BIN BIN -test.run …`, argv0 en premier).

## Ouvert — Produit / docs

- **Elo de performance dans les stats de corpus** (reste de GB5.6) — reporté après la 0.37.0
  par décision produit. Demande d'abord une ADR qui fixe la formule (Elo des adversaires lus
  dans les métadonnées de match, longueur de match, matchs sans Elo) ; puis une vue de corpus
  exposée en GUI, CLI et serveur comme le face-à-face.
- **Export fédéral du classement de club** — reporté par décision produit : le CSV/JSON
  générique par épreuve et par saison suffit. À reprendre seulement sur un besoin réel, avec
  un fichier modèle de la fédération visée (FFBG ou autre), testé contre ce modèle ; ne
  jamais coder un format supposé.
- **Soumissions humaines de distribution** : le tap Homebrew se pousse tout seul sur tag une
  fois le dépôt et le secret en place (comme `aur.yml`), mais ces deux préalables restent à
  faire à la main, une fois :

  ```bash
  gh repo create kevung/homebrew-tap --public \
    --description "Homebrew tap for blunderDB" --clone
  # puis créer un token (classique ou fine-grained) donnant l'écriture sur
  # kevung/homebrew-tap uniquement, à https://github.com/settings/tokens?type=beta
  gh secret set HOMEBREW_TAP_TOKEN --body "<token>"
  ```

  Deux canaux restent entièrement manuels, chacun une PR contre un dépôt tiers revue par des
  humains : **winget** (`wingetcreate submit`, voir `packaging/winget/README.md`) et
  **Flathub** (build hors-ligne vendorant Go+npm, `docs/recherche/P16-distribution-desktop.md`
  en donne la recette ; effort de plusieurs semaines, non commencé).

### Étude : du diagnostic à l'action

Constat : le diagnostic (erreurs récurrentes, ventilations, temps × erreur) et la pratique
(quiz, Anki, entraînement, file d'étude) sont riches, mais rien ne répond à « que dois-je
travailler maintenant ? », le bruit n'est jamais montré, la priorité ne regarde que le coût,
et l'effet de l'étude n'est jamais mesuré. Ordre proposé ci-dessous ; s'appuie sur la perte de
MWC par décision (#597), M1 (L₇, #598) et M3 (difficulté par décision, #599). Chaque seuil se fixe dans une ADR
*avant* de regarder les résultats ; chaque métrique est documentée en détail dans le manuel.

- **Familles par similarité du plan d'étude** (suite de #602). La grille de voisinage de
  l'ADR-0080 (`storage/reference.go`, 0,5 s pour 50 000 erreurs en 20 groupes) rend
  abordable un regroupement par distance `like` ; le plan d'étude groupe encore par thème seul.

## Historique — items faits

- **2026-10-31 — Export GUI : sélecteur de paquets** : déjà livré (`ExportDatabaseModal.svelte`, listes collections, leçons et paquets ; d64107236, 64bb447ea).
- **2026-10-31 — `mergeWatchedLibraries` et le plancher de lecture** : la base refusée par le
  plancher est nommée dans le bandeau du dossier surveillé, avec le schéma requis (#862).

- **2026-10-31 — stockage** : `isForeignKeyViolation` teste le nom de la contrainte, le préfixe
  `postgres:` n'est plus doublé sur l'erreur de commit d'une suppression de match, le crochet
  `transferCommitted` est atomique (#864).
- **2026-10-31 — relevé des entrées « Ouvert » déjà livrées** : empreinte d'une position copiée
  prise dans la transaction d'import (#828, `8225586de`) ; `SwapPlayers` et `DeleteCascade`
  écrits une fois dans `sqlshared` (#829, `1ac6cbff5`) ; signature et origine d'un commentaire
  dans l'export JSON et `comments.add`, tests `comments_*`/`collections_*` des backends retirés
  (#831, `06b8e6cb7`) ; plancher de lecture partagé dans `sqlshared`, source d'import et
  `OpenReadOnly` contrôlés, démon ancien signalé par la CLI, cas de contrat du plancher écarté de
  `remote` (#825, `5152f2c8b`) ; test du vacuum sans délai de 300 ms, `filelock.ObserveWaits`
  (#827, `82f1e084f`) ; réouverture de l'ancienne bibliothèque après un CTRL-O annulé (#823) et
  refus d'une base trop récente traduit (#824), `bb9ec1a19` ; commentaire de `build.yml` sans
  numéro de patch Go ; `backgammon-tournoi#26` (N26) fermée, repêchage dans Nicomaque
  (`ad6dbee77`).

- **2026-10-25 — lot 2026-10-25b** (fusionné sur `main`) : #823 à #833, #802, #836 et #837.
  Décision : pas de verrou mono-écrivain pour `serve` SQLite (entrée Backend laissée ouverte).
- **2026-10-25 — lot du 2026-10-25** (fusionné sur `main`) : version minimale de lecture et clés
  de schéma réservées dans `MetadataStore.Save`, plancher contrôlé avant toute écriture de
  l'ouverture (#815) ; progression et annulation de CTRL-O (#816, #817) et phases de migration
  affichées dans le front ; bandeau de fin du rollout (#818) ; digest MET recalculé (#819) ;
  verrou de swap du vacuum (#820) ; restes de parité de l'analyse, mode dégradé sans `withPlayed`
  (#821).
- **2026-10-24 — lot de la 2.50** (fusionné sur `main`) : filtre de recherche `aw`, les points
  qu'il reste à marquer à chacun des deux joueurs (#801) ; parité des analyses du démon avec le
  wrapper, `analyses.save` fusionne, `analyses.merge` et `withPlayed` existent (#803, #804) ;
  `Migrate` sans migrateur enregistré reclusterise les tables de cellules, ce qui règle les
  embarqueurs sans blank-import de `database` (#805) ; un job de rollout fini avant le retour de
  `startJob` garde son bandeau de fin, et undici 8.12.3 (#806) ; une décision de videau compte
  par son sens et non par ses graphies, `countedExpr` (#807) ; export `.apkg` : analyses par
  lot et plateaux dessinés au fil du zip (#808) ; le journal d'import s'écrit dans la
  transaction du groupe (#811) ; `DatabaseVersion` 2.50 : les colonnes d'analyse et de coup
  sont dérivées à nouveau, ce qui couvre les imports de base et de matchs sans coups joués et
  l'appariement du coup joué par plateau ; `move.error_mp` reste inerte, `ScoreMoves` est retiré
  (#812) ; progression et annulation de la migration à l'ouverture dans un nouveau groupe de
  bibliothèque (#813) ; retrait de la base de démo intégrée et des visites guidées (#814).
  Retirés comme caducs : le décompte des gaffes de la barre d'état.

- **2026-10-05 — relevé avant 0.37.0, items trouvés faits dans le code** : la valuation du
  videau par lot (sans objet depuis la forme close de `levelSolve`, gammonNet v1.4.0,
  `11e9ddbd5`, qui livre aussi le filtre en triplet et le beaver de `DecideForSession`) ;
  `swapMatchState` disparu ; le panneau Eval ne construit plus qu'un `Searcher` par frappe
  (`internal/gui/gammonnet_eval.go`) ; le moteur d'une analyse est la colonne
  `analysis_engine`, lue sans décoder le blob ; le blob d'analyse est un binaire zstd à
  dictionnaire (ADR-0070) ; `analysis.creation_date` est une colonne indexée ; les étapes de
  migration conditionnelles (`errStepNotApplicable`) ne sont plus ; le drapeau
  `python-format` est neutralisé par `scripts/doc-po-update.sh` ; un seul parseur de recherche
  JS (`parseSearchTokens`) ; listes virtualisées
  (`panels/PanelTable.svelte`) ; `generateXGID` (`services/xgid.js`) encode le drapeau Crawford ;
  `AnalysisPanel` importe `utils/playedMarks.js` ; `DEFAULT_PANEL_HEIGHT` vaut 250 comme
  `config.go` ; `PickList.svelte` partagé ; `race.CubeVerdict` ; ADR-0004 ne cite plus la
  police non sous-ensemblée ; issues de la simulation de direction 2026-09 (D5-D9) closes.

- **2026-09-06 — suites de la critique de la documentation** (branche
  `chore/critique-reste`, `tasks/critique-doc-2026-09/`) : l'image
  `blunderdb-serve` pose `XDG_DATA_HOME=/data` et déclare le volume (elle ne
  calculait jamais ses tables de bearoff) ; la fusion des analyses classe les
  profondeurs par `domain.AnalysisDepthRank` au lieu de comparer les chaînes
  (« 2-ply » l'emportait sur « 10-ply ») ; `search --error-min` ignore les
  positions sans analyse ; un lot d'import fait de doublons seuls sort en 0 ;
  les bandes de niveau du PR sont traduites dans les neuf locales
  (`stats.grade.*`) et le manuel les nomme en français ; quatre locales
  disaient « dé » pour le videau de la carte *PR Cube* ; les captures
  `panel_*.png` et `screenshot.png` sont régénérées par `make screenshots`
  (les navigateurs Playwright sont installés sur ce poste, port 5174).
  Reste, faute de Windows : les captures SmartScreen en anglais.

- **Job `test-os`** : `continue-on-error: true` retiré, le job est bloquant. Fiche E.1 (#217), fusionnée le 2026-09-03.

- **2026-09-02 — `use_cube` à la recherche** (ADR-0016, point 7) : fait le
  2026-09-02 (3657ea1a, ADR-0023) : chaque feuille de la recherche est valuée
  par le modèle de videau à l'état de videau de la position ; gammonNet v1.2.1
  épinglé par `EngineVersion`.
- **2026-08-31 — Une évaluation refusée est un état nommé, pas une cellule vide**
  (ADR-0017, Consequences ; ADR-0019 règle 4) : fait le 2026-08-31 (a4b0592c) :
  `EvaluatePosition` renvoie `Refused bool` comme une donnée
  (`internal/gui/gammonnet_eval.go:71-78`) et le panneau nomme l'état
  (`cubeDecision.js`).
- **2026-08-31 — Assertion Playwright « le panneau Eval ne défile pas »**
  (ADR-0017/0018) : fait le 2026-08-31 (a4b0592c) :
  `frontend/tests/e2e/eval-panel-no-scroll.spec.js` mesure
  `scrollHeight − clientHeight` du panneau dans chaque régime.
- **2026-07-26 — Provenance `individually_imported` depuis la CLI** (ADR-0001) :
  fait autrement, dès le 2026-07-26 (132d3562) : `cli_import.go:211` pose
  `IndividuallyImported = true` sur la position et `PositionStore.Save` fait un
  OR collant du drapeau (`storage/sqlite/positions_sqlite.go:68-75`).
- **2026-06-13 — Troisième copie des helpers de recherche** : fait le 2026-06-13
  (a9a872bc) : `db_filter_match.go` supprimé, `database/db_search.go` fait 59
  lignes et délègue à `storage` ; l'item avait été écrit après coup.
- **2026-09-02 — Découpage `db_session.go`** (603 lignes, 5 responsabilités) :
  fait le 2026-09-02 (cf984285) : la famille sessions délègue à
  `storage/sqlite`, le fichier fait 254 lignes.
- **2026-09-02 — Dédup de l'autocomplétion inline** Match/Tournament : fait le
  2026-09-02 (b3bdd99f) : `components/EntityAutocomplete.svelte` partagé par
  `MatchPanel` et `TournamentPanel`, test `EntityAutocomplete.test.js`. Le
  découpage `MatchDetailPane` reste ouvert (fiche D.10, #210).
- **2026-09-02 — Réponse masquée d'une carte Anki, validée sur une vraie base**
  (ADR-0025). Base bâtie à la CLI depuis `testdata/` (349 positions et analyses
  XG d'un match), paquet créé et synchronisé, puis l'application réelle pilotée
  par son pont Wails de développement : la réponse dévoilée est identique au
  blob SQLite de la position, et la notation a bien fait avancer FSRS en base
  (carte neuve → apprentissage, journal écrit, compteurs du paquet à jour).
  Piège rencontré : une instance `wails dev` tournait déjà sur une autre base
  et c'est elle qu'on pilote si on ne vérifie pas — `-devserver` et
  `-frontenddevserverurl` isolent la sienne.

- **2026-09-02 — Composant `<Modal>` unique** : fait le 2026-09-02 (2dc51a36) : `components/Modal.svelte`, 13 modales migrées, 0 warning a11y.
- **2026-09-02 — Contrat storage : familles restantes** : fait le 2026-09-02 : cas Comment/*, Collection/*, Anki/RandomCard, batch (LoadByIDs, ByPositions…) ; le wrapper `database/` délègue désormais toutes ses familles à `storage`.
- **2026-09-02 — Matrice OS du job `test`** : fait le 2026-09-02 (70ab6f2e, 1c6fa7b5) : job `test-os` windows/macos en -short ; fuites de handles SQLite dans les tests corrigées, garde `/proc/self/fd` sous Linux.
- **2026-09-02 — Capture d'écran du README** : fait le 2026-09-02 (e46b896d) : capture Playwright de l'interface réelle sur mock Wails, `SCREENSHOT=1 npx playwright test screenshot`.
- **2026-09-02 — `runMigrationChain` en table** : fait le 2026-09-02 (0503c994) : registre `migrationSteps`, fichiers `db_migration_v*.go`, test de continuité.
- **2026-09-02 — Fusion des 13 helpers purs dupliqués** : fait le 2026-09-02 (b0054ab3) : paquet `storage/searchfilter`.
- **2026-09-02 — `utils/rangeFilters.js` / `filterModel.js`** : fait le 2026-09-02 (dd1aa7b1) : `services/filterModel.js`, SearchPanel 2 226 → 1 485 lignes.
- **2026-09-02 — `utils/boardRenderer.js` / `boardScene.js`** : fait le 2026-09-02 (fcd81620) : `utils/boardScene.js`, `boardInteractions.js`, couche statique/dynamique.
- **2026-09-02 — E2E des parcours produit** : fait le 2026-09-02 (451ce542) : specs recherche, navigation match, import de position.
- **2026-09-02 — Job docs** : fait le 2026-09-02 (ba578f9c) : filtrage par chemins, cache pip, PDF sur tag seulement.
- **2026-09-02 — Export unifié, schéma unique** (commits `dac6d630` DDL
  unique ; `475c1b4a`, `bb77b247`, `1cd53334` export unifié) :
  `storage/sqlite/schema_sqlite.go` est la source DDL unique, et
  `ingest.ExportSQLite` remplace les quatre chemins d'export
  (`ExportDatabase`, `ExportCollections`, `ExportTournaments`, l'export du
  serveur) — GUI/CLI/serveur lisent tous via `storage.Storage`, sur SQLite
  comme PostgreSQL. Les deux écarts connus sont corrigés : l'export du
  serveur (`exports.sqlite`) portait un `Selection` vide (rien n'en
  sortait, ni positions ni matchs) et n'avait aucune identité de signature ;
  il exporte maintenant tout le tenant et peut apposer un filigrane
  (`Options.Identity`, `--identity-dir`). Parité GUI/serveur testée
  (`ingest/export_parity_test.go`).
- **2026-09-02 — Bug `CommitImportDatabase`, colonnes scalaires NULL** (commits bd33df8d, 086f5466) : la branche « position neuve » écrit désormais via `PositionStore.Save` (hash + colonnes canoniques), et une réparation idempotente à l'ouverture (`repairPositionsWithoutScalars`, sans bump de `DatabaseVersion`) rattrape les lignes existantes.
| Fait le | Item | Origine | Où |
|---|---|---|---|
| 2026-05-21 | Index `idx_position_pip_diff`, `idx_position_dice`, `idx_position_score_cube`, `idx_game_match` ajoutés (PipWindow inchangé — le planificateur scanne à raison quand >50 % des lignes matchent ; l'index dés sert aux requêtes OR ; la sous-requête tournoi ne SCANne plus `game`). | v2.0.0 phases 02/05 | `3f924f54` |
| 2026-06-12 | `wails build` exercé à chaque tag : la matrice CI construit les 4 plateformes, `wails dev` compile le même chemin localement ; vérifié vert sur le tag 0.26.1 (run 27267855792). | v2.0.0 phase 06 | `0de54c50` |
| 2026-08-09 | Publication AUR automatisée (`aur.yml`, paquet `blunderdb-bin`) ; le rerun manuel demandé en août n'a plus lieu d'être, le workflow refonctionne depuis 0.33.0 (2026-08-26). | BACKLOG 2026-08 | `1ba1838b` |
| 2026-08-11 | `BenchmarkSearch_WinGammonCombo` : requête restructurée en `p.id IN (SELECT position_id FROM analysis WHERE …)`, index couvrant `idx_analysis_win_gammon_covering (player1_win_rate, player1_gammon_rate, position_id)` ; le TEMP B-TREE disparaît (EXPLAIN vérifié). Gain modeste sur la fixture (629 → ~500 ms) : le filtre du bench matche 79 % des lignes, la reconstruction Go des positions est le goulot, pas le tri SQL. Index redondants `idx_position_score`/`idx_analysis_win1` supprimés (E3). | v2.0.0 phase 05, fiche 05 T3 | `4032a70a` |
| 2026-08-11 | Sous-commande `blunderdb vacuum` + bouton « Compacter la base » (fiche 06) — publiée en 0.33.0. | plan 2026-08 | `67332e9b` |
| 2026-08-26 | gnubgparser v1.3.0 ne lisait jamais la chance qu'il parsait (`LU[-0.00537]` à un seul champ, `parseLuck` en exigeait deux) : corrigé amont, **gnubgparser v1.4.0** (traite aussi `LU[-inf]` et la chance d'un nœud « set dice ») ; blunderDB dépend de v1.4.0, `TestMapGnuBGCarriesLuck` et `TestLuckAgreesAcrossFormats` pinent le résultat contre l'import XG du même match. | #116 lot 1 | `96e1ca7d` |
