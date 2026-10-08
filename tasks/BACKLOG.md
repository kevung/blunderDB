# Backlog — suivis ouverts et chantiers de fond

La seule liste des suivis ouverts du dépôt. Un item **ouvert** est rangé par domaine ; un item
**fait** passe dans la section *Historique* avec sa date et son commit. Les gros chantiers
méritent chacun leur propre décision (et éventuellement leur ADR) avant lancement — ne pas les
commencer « en passant ». Aucun plan en cours ne priorise ces items : chacun se reprend par sa
propre issue le jour où il est lancé.

## Ouvert — Backend

- **Statistiques : le filtre `--engine` par les cellules.** Il lit encore les décisions
  (copie de la sélection) ; passer par `match_stats_cell` demande le moteur dans la clé des
  cellules, autant de cellules en plus que de moteurs par match. Mesure de départ :
  `tasks/mesure-bmab-0.37.md` § 7.2 ter.
- **Embarqueurs sans blank-import de `database` (gammonGo, `pkg/blunderdb/server/embed.go`).**
  `st.Migrate` ne passe pas par l'ouverture du wrapper `Database` : une base SQLite 2.31 dont
  les tables de cellules ont l'ancienne forme n'y est pas reclusterisée (`reclusterDerivedTables`
  ne tourne qu'avec `EnsureSchema`), et ses statistiques restent lentes jusqu'à une ouverture
  par blunderDB.
- **`countedExpr` et les libellés de videau dégénérés** (`sqlshared/stats.go`) : une décision
  de videau est comptée comme action *active* dès que `move.cube_action` n'est ni `''`, ni
  `No Double`, ni `NoDouble`. Une valeur `Unknown(D=…,T=…)` écrite par l'importeur XG pour un
  code non mappé (`ingest/xgmap.go`) passerait donc pour un vrai Double/Take/Pass et entrerait
  au dénominateur du PR. Aucune occurrence connue dans les corpus. Correctif si ça mord un
  jour : tester `engine.CanonicalCubeAction` plutôt que les graphies littérales.
- **`SwapPlayers`/`DeleteCascade` dupliqués** entre `storage/sqlite/matches_sqlite.go` et
  `storage/postgres/matches_postgres.go` ; les partager par des closures SQL dialectales dans
  `sqlshared`.
- **Tests par backend redondants** : `comments_*_test.go` / `collections_*_test.go` de
  `storage/sqlite` et `storage/postgres` doublonnent les cas du contrat
  (`storagetest/contract_comments.go`, `contract_collections*.go`). Effort S.
- **`migrate.Run` par lots avec reprise** (aujourd'hui une transaction unique pour toute la
  copie SQLite→PG, `migrate/migrate.go`) — attendre un besoin terrain.
- **Décompte des gaffes d'`info` en cache** : sans `--estimate`, `GetDatabaseStats` lit
  `Counts` d'un coup et `blunderCount` (`sqlshared/metadata.go`) parcourt toute `analysis` —
  des secondes à des dizaines de secondes sur une base BMAB ; `--estimate` ne fait que sauter
  les gaffes au-delà de 200 000 lignes. Le garder en cache dans `metadata`, invalidé par
  import, suppression et changement de seuil. Effort M.
- **Course Stat/Rename du vacuum par remplacement** (`database/db_vacuum.go`, `vacuumBySwap`) :
  l'absence de `-wal` est constatée par `os.Stat` puis le fichier remplacé par `os.Rename` ; un
  autre processus qui ouvre la base entre les deux lit l'ancien inode et y écrit à perte.
  Fenêtre de quelques microsecondes, sans verrou de fichier pour la fermer. Prendre un verrou
  exclusif SQLite (`BEGIN EXCLUSIVE` sur une connexion tenue jusqu'au rename) ou un verrou
  consultatif. Effort S-M.
- **Journal d'import hors transaction** (`ingest.RecordOutcomes`, `ingest/journal.go`) : le
  journal est écrit après le commit du groupe de fichiers. Un crash entre les deux laisse un
  match écrit mais non journalisé ; à la reprise le fichier est relu et le match le couvre
  comme doublon de lui-même. Le journal ne ment pas sur ce qui est en base, mais l'issue
  « nouveau » est perdue. À faire : écrire les lignes dans la transaction du groupe (le
  pipeline les produit déjà avant le commit).
- **`move.error_mp` sans lecteur ni tenue** (`tasks/mesure-bmab-0.37.md` § 9) : seule
  `repair --move-errors` l'écrit ; ni l'import ni l'analyse n'appellent
  `RescorePositionMoves`, aucune requête ne la lit. Pour que `E` s'y appuie (SQL exact, sans
  phase Go), il faut une tenue à jour à chaque écriture d'analyse ou de coup et un moyen de
  distinguer « pas noté » de « non notable » (NULL ambigu).
- **Quatre copies des helpers de fusion d'analyse** : la fusion des coups (clé par notation, la
  profondeur la plus grande gagne, tri par équité), celle du videau et l'union des coups joués
  existent dans `database/db_import_common.go` et `database/db_analysis.go`, `ingest/merge.go`
  et `engine/gammonnet/supersede.go` (`enginePriority`, `mergeCheckerMoves`,
  `mergeCubeAnalyses`). Elles divergent : `ingest` départage une égalité d'équité par la
  notation (ordre total), les autres non (ordre de map). À faire : une seule implémentation (au
  niveau `engine`, qui a `NormalizeMove`) appelée par les quatre, avec la règle de départage
  d'`ingest`.
- **PostgreSQL, écriture par tenant** : `positions_postgres.go` enveloppe la `pgconn.PgError`
  dans `ErrConflict` (texte du pilote vers le client, sans oracle) ; `Tournaments().AddMatch`
  avec un match étranger ou absent réussit en silence (UPDATE 0 ligne) ; `Analyses.Save` en
  course avec une suppression rend un autre message (même tenant).
- **Serveur : quota en octets sous PostgreSQL** : la mesure repose sur
  `pg_total_relation_size` et `reltuples`, globaux à la base et non au locataire, donc un canal
  auxiliaire faible (un locataire devine la taille des autres) ; et les 13 `COUNT` par appel
  ont un coût. À faire : compter par locataire, avec un cache ou un compteur tenu à jour.
- **Import XG, métadonnées sans colonne** (ADR-0067) : commentaires d'en-tête et de pied de
  match, horloge, table d'équité (MET) ne sont pas importés ; exigent une colonne (ou
  `match.metadata`) et, pour les commentaires, que xgparser expose `parseCommentSegment`.
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

- **`MatchEquityTableStore.Save` croit le `Digest` fourni** (`sqlshared/met.go`). Le
  digest est la clé de dédoublonnage et le nom de la table pour `analysis.met_id` ; il est
  calculé par `engine.MET.Digest()` sur les valeurs parsées, pas sur les octets de `Source`.
  Les deux appelants le recalculent depuis la source (`mets.Import`, et `mets.Carrier` pour
  l'export, l'import, la fusion et la migration de bases), mais le contrat ne le garantit
  pas : un futur appelant pourrait encore enregistrer un digest incohérent. Remède : `Save`
  parse `Source` (`engine.ParseGnubgMET`), recalcule et refuse `ErrInvalid` sur écart. Les
  ~15 fixtures des contrats (`storagetest/contract_schema_2_31.go`, `contract_met_*.go`)
  passent des sources factices (`<met/>`, digest `"aaa"`) à réécrire en tables gnubg
  valides, sur les deux backends. Effort S-M, test de contrat « digest incohérent refusé ».

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

- **`openPanels` dérivé d'`activeTabStore`** : deux sources de vérité pour le panneau visible
  (`openPanels` est un `writable` séparé dans `stores/uiStore.js`, tenu par
  `services/tabHandler.js`) ; un état incohérent est atteignable (onglet surligné, panneau
  vide). Dériver `openPanels` d'`activeTabStore` et supprimer `tabHandler.js`.
- **`MatchPanel.svelte` (1 772 lignes)** : en extraire le volet de détail d'un match.
- **Rollout GUI** : un job instantané peut perdre son bandeau de fin
  (`startRolloutOfCurrent`, `services/rolloutService.js`, remet `outcome: null` après le
  démarrage).
- **Export GUI : sélecteur de paquets** : l'export serveur sait se limiter à des collections,
  leçons ou paquets Anki (`collectionIds`, `lessonIds`, `deckIds`), le dialogue d'export de la
  GUI n'offre pas ce choix.

- **Restes de GB-C14 (libellés, vocabulaire, gestes)** : pas de bouton « Élargir » ;
  libellés de groupes de la barre d'outils non faits ; en-tête de panneau non factorisé dans
  `PanelTable` ; noms de touches en prose non vérifiés dans les huit langues (« SHIFT-Enter »
  en fi, ja, ru dans l'aide engendrée) ; j/k ne déplacent pas le surlignage du panneau
  Tournois ; vérifier qu'un « Too good to double » importé n'est plus lu comme *no double*.

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

## Ouvert — Produit / docs

- **Elo de performance dans les stats de corpus** (reste de GB5.6) — reporté après la 0.37.0
  par décision produit. Demande d'abord une ADR qui fixe la formule (Elo des adversaires lus
  dans les métadonnées de match, longueur de match, matchs sans Elo) ; puis une vue de corpus
  exposée en GUI, CLI et serveur comme le face-à-face.
- **Export fédéral du classement de club** — reporté par décision produit : le CSV/JSON
  générique par épreuve et par saison suffit. À reprendre seulement sur un besoin réel, avec
  un fichier modèle de la fédération visée (FFBG ou autre), testé contre ce modèle ; ne
  jamais coder un format supposé.
- **Direction, moteur de tournoi** : `PileOfCells/backgammon-tournoi#26` (N26, un qualifié
  retiré d'une poule garde sa place en phase suivante) reste ouverte.
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
- **Catalogues `doc/source/locale/fr/`** : suivis par git, ils ne contiennent que des entrées
  vides — le français est la langue source, Sphinx le rend depuis les `.rst`. Rien ne les
  compile et `doc-i18n-check.sh` ne les regarde pas ; ce sont des fichiers inertes qui
  brouillent toute vérification globale des catalogues. Candidats à la suppression ; à
  trancher avant : voudra-t-on un jour traduire *depuis* le français vers un français
  simplifié.

### Étude : du diagnostic à l'action

Constat : le diagnostic (erreurs récurrentes, ventilations, temps × erreur) et la pratique
(quiz, Anki, entraînement, file d'étude) sont riches, mais rien ne répond à « que dois-je
travailler maintenant ? », le bruit n'est jamais montré, la priorité ne regarde que le coût,
et l'effet de l'étude n'est jamais mesuré. Ordre proposé ci-dessous ; s'appuie sur la perte de
MWC par décision (#597), M1 (L₇, #598) et M3 (difficulté par décision, #599). Chaque seuil se fixe dans une ADR
*avant* de regarder les résultats ; chaque métrique est documentée en détail dans le manuel.

- **Intervalles de confiance partout, et bilan de match** (#600). Afficher la bande (IC 95 %,
  bootstrap par parties) du PR, de L₇ et des cellules de ventilation, à la place du seul
  grisage « < 10 décisions ». Au niveau du match : un encart « 3 décisions à revoir » (perte ×
  caractère évitable M3, un clic vers le coup), le résultat ajusté de la chance à côté du
  score (la chance est déjà importée), et la distinction erreur précipitée / erreur réfléchie
  (temps de décision : discipline contre connaissance).
- **Positions de référence proposées (panneau Collections, `collection suggest`)** (#602). Critères
  combinés, raison affichée par position : représentativité (centre d'une famille de ses
  erreurs, distance `like`), fréquence de la famille dans toutes ses décisions, coût évitable
  (M3), leçon nette (écart meilleur/second large, verdict stable entre profondeurs, rollout si
  possible), diversité (dédoublonnage par similarité), non traitée (même prédicat que la file
  d'étude), références de videau par score. Portée match / tournoi / base / filtre courant,
  taille 10/20/50, liste proposée à cocher → collection figée ou vivante, deck Anki ou quiz en
  un clic. Coût du regroupement par similarité mesuré (ADR-0077) : 166 ns par paire, 8 s pour
  10 000 erreurs en paires complètes — il faut un index de voisinage, qui servira aussi les
  familles par similarité du plan d'étude.
- **Fermer la boucle, et biais directionnels** (#603). Par famille étudiée : erreur en match réel
  avant/après l'étude, avec IC (la vue `list --type study` n'a pas de colonne de gain). Biais
  signés plutôt que perte seule : taux de prises/refus contre le bot, doubles prématurés contre
  manqués par score, prudence/audace sur les blots — « tu prends trop » se corrige mieux qu'un
  PR.
- **Bilan de tournoi** (#604). L₇ avec IC comparé au niveau habituel, 2-3 familles d'erreurs de
  l'épreuve, erreur selon la ronde et selon le rang de la décision dans le match (fatigue),
  aux scores de pression (DMP, Crawford) et sous la pendule.

## Historique — items faits

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
