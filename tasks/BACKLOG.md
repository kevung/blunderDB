# Backlog — suivis ouverts et chantiers de fond

Fusion, le 2026-09-02, de `tasks/FOLLOWUPS.md` (suivis différés de
l'optimisation v2.0.0 et du ticket #116) et de
`tasks/plan-amelioration-2026-08/BACKLOG.md` (chantiers de fond identifiés
par le plan d'août). Les deux anciens fichiers renvoient ici.

Règles : un item **fait** reste dans la section *Historique* avec sa date et
son commit ; un item **ouvert** est rangé par domaine. Les gros chantiers
méritent chacun leur propre décision (et éventuellement leur ADR) avant
lancement — ne pas les commencer « en passant ». Le plan courant qui
priorise ces items est `tasks/plan-amelioration-2026-09b/README.md` (second
audit du 2026-09-02) ; le renvoi `→ fiche X.N (#NNN)` sur un item ouvert
désigne la fiche du plan (et son issue GitHub) qui le porte. Un item sans renvoi
n'est priorisé par aucune fiche. La bascule dans l'historique des items que le
plan a trouvés déjà faits a été opérée le 2026-09-02 (fiche A.14, #168).

## Ouvert — Backend

- **gammonNet — grouper la valuation du videau sur les candidats.** Le seul levier
  qui reste sur le videau, et il faut le cadrer en amont (`gn_cube.c`) avant de le
  porter. Mesuré le 2026-09-02 en écrivant puis en annulant l'optimisation
  annoncée par ADR-0011 : précalculer les `metAfter` **ne vaut rien** (1 % de
  gain, sous le plancher de bruit de ±15 %), parce que les lookups ne pèsent que
  11 % de `buildLevels` quand `levelSolve` en pèse 83 %, et que chacune de ses
  60 bissections est une division sur le chemin critique plus un branchement
  imprévisible — ~60 cycles irréductibles, les 60 itérations comme la forme des
  segments étant verrouillées par le gold. **Ne pas rouvrir « précalculer les
  MET ».** Ce qui marcherait est de valuer le videau *par lot* sur les candidats,
  exactement comme le noyau groupé a fait pour le réseau (#133). Le poste vaut
  ~3,9 % du profil au score avant le noyau, donc bien davantage maintenant.
  Précédé par la **forme close de `levelSolve`** → fiche C.7 (#194), dont le
  lot est la seconde marche.
- **gammonNet — `swapMatchState` alloue un `*MatchState` par nœud**, soit ~1 400
  allocations par décision au score, tout ce qui reste du compteur après #150
  (7 432 → 82 hors ce poste). Coût CPU négligeable, déchet évitable. Laissé de
  côté pour ne pas entrer en collision avec #148. → fiche C.10 (#197).
- **`internal/gui/gammonnet_eval.go` lance jusqu'à trois recherches complètes par
  position** : `EvaluatePosition`, puis `preRollFacts` qui construit un second
  `Searcher` et refait un `Probs` quand les dés sont posés (+36 % mesuré,
  ADR-0017), puis `evaluateRaceRegime` qui en construit un troisième. Chantier
  d'interface, pas de moteur. → fiche C.9 (#196).
- **`positionIDsWithStaleGammonNet` décode le JSON compressé de toutes les
  analyses** pour trouver le moteur, qui est dans le blob et non dans une colonne
  (`db_gammonnet_batch.go:148-167`). Préambule coûteux d'`AnalyzeStaleGammonNet`.
  → fiches B.12 (#180, le moteur sort du blob) et C.4 (#191, le lot ne retente
  plus à l'infini).
- **`statsCountedExpr` et les libellés de videau dégénérés** (#116 lot 0) :
  une décision de videau est comptée comme action *active* dès que
  `move.cube_action` n'est ni `''`, ni `No Double`, ni `NoDouble`. Une valeur
  `Unknown(D=…,T=…)` écrite par l'importeur XG pour un code non mappé
  (`convertCubeAction`/`convertRawCubeAction`, `ingest/xgmap.go`) passerait
  donc pour un vrai Double/Take/Pass et entrerait au dénominateur du PR.
  Aucune occurrence connue dans les corpus. Correctif si ça mord un jour :
  tester `engine.CanonicalCubeAction` plutôt que les graphies littérales.
- **`SwapPlayers`/`DeleteCascade` partagés** entre backends via closures SQL
  dialectales (~90 lignes dupliquées à l'octet près). → fiche B.14 (#182).
- **Dictionnaire zlib partagé** pour `analysis.data` (−25 à −40 % de la plus
  grosse table ; nouveau format de blob versionné, fuzz à étendre) ; au
  passage, fusionner les fragments d'analyse en mémoire avant compression
  (aujourd'hui recompression niveau 9 par fragment) et réutiliser des
  prepared statements sur le chemin d'import. → fiche B.12 (#180).
- **`migrate.Run` par lots avec reprise** (aujourd'hui une transaction unique
  pour toute la copie SQLite→PG) — attendre un besoin terrain. → fiche G.12
  (#240).
- **Colonne `creation_date` promue dans `analysis`** + index (pushdown SQL du
  filtre de date) : exige bump `DatabaseVersion` + triple synchro schéma +
  migration. La fiche 05 a réglé le N+1 ; ceci est l'étape structurelle.
- **Tests par backend redondants** : `comments_*_test.go` / `collections_*_test.go` de `storage/sqlite` et `storage/postgres` doublonnent les cas de contrat ajoutés le 2026-09-02. Effort S. → fiche B.14 (#182).
- **Étapes de migration 1.0.0→1.6.0** : la bizarrerie « table déjà présente ⇒ chaîne arrêtée » (`errStepNotApplicable`, registre `migrationSteps`) est conservée par fidélité ; rendre ces étapes inconditionnelles (leur DDL est `IF NOT EXISTS`). Effort S, test de migration à ajouter. → fiche B.9 (#177).
- **Drapeau `python-format` faux positif dans les `.po`** : Babel marque les msgid contenant « 12 % de » ; `msgfmt -c` échoue sur 2 à 4 entrées par langue, Sphinx s'en moque. Remède côté extraction (`no-python-format`) ou reformulation. Effort S. → fiche H.13 (#255).

## Ouvert — Moteur (dettes nommées dans les ADR)

- **Renommage `race.Money` → `race.CubeVerdict`** et libellé de la colonne
  Équité (ADR-0016, points 5-6) : différés pour ne pas entrer en collision
  avec l'ADR-0017, fusionné le 2026-08-31 — **plus rien ne bloque**, le nom
  ment sur trois régimes (`money.go:36`). → fiche C.3 (#190).
- **Refuser de démarrer sans drapeau explicite « derrière un proxy »**
  (ADR-0005, option différée) : friction pour tout déploiement légitime pour
  attraper une erreur que la doc, `--help` et le Dockerfile signalent déjà.
  À revisiter si un déploiement nu survient réellement.

## Ouvert — Frontend

- **Parseur de recherche unique** : `commandProcessor.parseFilters` et
  `searchFilterService.parseSearchCommand` divergent déjà ; converger vers
  une grammaire unique testée (les deux suites existantes servent de filet
  croisé). → fiche D.3 (#203).
- **`openPanels` dérivé d'`activeTabStore`** : deux sources de vérité pour
  le panneau visible, état incohérent atteignable (onglet surligné, panneau
  vide) ; supprimer `tabHandler.js`. → fiche D.10 (#210).
- **`MatchDetailPane.svelte`** : découpage de MatchPanel (1 386 lignes) ; la
  dédup de l'autocomplétion inline est faite (`EntityAutocomplete`, voir
  Historique). → fiche D.10 (#210).
- **Virtualisation des listes** (matchs, transcript ~300-500 coups,
  collections) et **LIMIT/pagination** de la recherche (aujourd'hui tout le
  jeu de résultats traverse l'IPC Wails en un message) ; `ListOpts` existe
  déjà dans le contrat storage, passé vide. → fiches D.8 (#208) et B.10 (#178).
- **`generateXGID` est avec perte** (`positionService.js`) : longueur de match = plus grand score away, Crawford déduit, Jacoby/cube max émis à 0 ; le corpus `testdata/xgid_corpus.json` fige ce comportement. Encoder depuis `match_length` et le drapeau Crawford réel. Effort S. → fiche D.11 (#211).
- **`AnalysisPanel.svelte` garde sa copie des prédicats « coup joué »** ; `utils/analysisRows.js` exporte `playedMovePredicate`/`playedCubePredicate`. Effort S. → fiche D.10 (#210).
- **Escape avalé dans le panneau Recherche** tant qu'un champ a le focus (`SearchPanel.handleKeyDown` stoppe tout) ; **double-clic instable sur une ligne du panneau Match** (le premier clic ouvre le volet transcript qui décale la mise en page) ; **sortie du mode match** : `loadAllPositions` repart sur la dernière position, pas celle quittée. Trois points d'ergonomie relevés par les specs e2e. Effort S chacun. → fiche D.1 (#201).
- **`enterEPCMode` peut photographier le plateau EDIT vide** quand App enchaîne `exitEditMode` puis `enterEPCMode` avec un index sauvegardé inchangé (0→0, pas de redessin). Préexistant, documenté dans `modeMachine.js`. Effort S. → fiche D.1 (#201).
- **Plateau et `MatchInfoBar`** : le plateau ne se réajuste pas quand la barre de match apparaît (la capture d'écran force un `resize`) ; `panelLayoutStore.js` dit `DEFAULT_PANEL_HEIGHT = 380 « mirrors config.go »` alors que `config.go` dit 250. Effort S. → fiche D.1 (#201).
- **`PickList.svelte` partagé** Export/Picker (~110 lignes de CSS dupliquées) et **dialogue de sauvegarde de `SearchPanel`** à migrer sur `Modal.svelte`. Effort S. → fiche D.9 (#209).
- **ADR-0004** cite encore `NotoSansJP-Regular.ttf` (5,7 Mo) : la police est un sous-ensemble WOFF2 de 178 Ko depuis le 2026-09-02 (constat historique, une note suffit).

## Ouvert — Tests / CI

- **Smoke test GUI sur une vraie base de production migrée** (v2.0.0,
  phase 06) : parcourir chaque filtre de la fenêtre de recherche. Priorité
  basse ; couvert indirectement par `searchFilterService.test.js` (unitaires
  + round-trip).
- **Le CLI écrit sur `os.Stdout`, pas sur un `io.Writer`** (constaté par E.3,
  #219, le 2026-09-04). 765 `fmt.Print*` répartis dans `internal/cli/` et un
  `captureStdout` de test qui remplace `os.Stdout` : un état global du
  PROCESSUS, donc 53 des 99 tests du paquet ne peuvent pas prendre
  `t.Parallel()` — c'est ce qui borne le gain sur ce shard, pas la durée des
  tests eux-mêmes. Le geste est de donner au `CLI` un `out io.Writer` (défaut
  `os.Stdout`) et de faire passer les écritures par lui ; il touche beaucoup de
  fichiers mais aucun comportement, et il débloque aussi les tests de sortie
  `--format json` qui aujourd'hui sérialisent. Chantier à part, pas « en
  passant ».
## Ouvert — Produit / docs

- **`epc.race` / défi** : vérifier la couverture de l'aide intégrée
  (`help/*.js`) — le panneau Eval a été redessiné trois fois depuis la fiche
  10 (ADR-0017/0018/0021). → fiche H.7 (#249).
- **Soumissions humaines restant après H.3 (#245)** : le tap Homebrew se pousse
  désormais tout seul sur tag une fois le secret et le dépôt en place (comme
  `aur.yml`), mais ces deux préalables restent à faire à la main, une fois :

  ```bash
  gh repo create kevung/homebrew-tap --public \
    --description "Homebrew tap for blunderDB" --clone
  # puis créer un token (classique ou fine-grained) donnant l'écriture sur
  # kevung/homebrew-tap uniquement, à https://github.com/settings/tokens?type=beta
  gh secret set HOMEBREW_TAP_TOKEN --body "<token>"
  ```

  Deux canaux restent entièrement manuels, chacun une PR contre un dépôt tiers
  revue par des humains : **winget** (`wingetcreate submit`, voir
  `packaging/winget/README.md`) et **Flathub** (build hors-ligne
  vendorant Go+npm, `docs/recherche/P16-distribution-desktop.md` en donne la
  recette ; effort de plusieurs semaines, non commencé).
- **Catalogues `doc/source/locale/fr/`** : ils existent, sont suivis par git, et
  contiennent 324 entrées **toutes vides** — le français est la langue source,
  Sphinx le rend depuis les `.rst`. Ils portent en outre des `msgid` dupliqués
  qui font échouer `msgfmt -c` (sans conséquence : rien ne les compile, et
  `doc-i18n-check.sh` ne les regarde pas). Ce sont des fichiers inertes qui
  brouillent toute vérification globale des catalogues. Candidats à la
  suppression pure et simple ; à trancher avant, la question de savoir si un
  jour on voudra traduire *depuis* le français vers un français simplifié.
  Constaté le 2026-09-03 en refermant l'écart de traduction.

## Historique — items faits

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

- **2026-09-02 — `use_cube` à la recherche** (ADR-0016, point 4) : fait le
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
  OR collant du drapeau (`storage/sqlite/positions_sqlite.go:68-75`) ; un
  `SaveIndividualPosition` distinct n'a plus d'objet.
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
