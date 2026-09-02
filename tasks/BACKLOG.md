# Backlog — suivis ouverts et chantiers de fond

Fusion, le 2026-09-02, de `tasks/FOLLOWUPS.md` (suivis différés de
l'optimisation v2.0.0 et du ticket #116) et de
`tasks/plan-amelioration-2026-08/BACKLOG.md` (chantiers de fond identifiés
par le plan d'août). Les deux anciens fichiers renvoient ici.

Règles : un item **fait** reste dans la section *Historique* avec sa date et
son commit ; un item **ouvert** est rangé par domaine. Les gros chantiers
méritent chacun leur propre décision (et éventuellement leur ADR) avant
lancement — ne pas les commencer « en passant ». Le plan courant qui
priorise ces items est `tasks/plan-amelioration-2026-09/README.md`.

## Ouvert — Backend

- **`statsCountedExpr` et les libellés de videau dégénérés** (#116 lot 0) :
  une décision de videau est comptée comme action *active* dès que
  `move.cube_action` n'est ni `''`, ni `No Double`, ni `NoDouble`. Une valeur
  `Unknown(D=…,T=…)` écrite par l'importeur XG pour un code non mappé
  (`convertCubeAction`/`convertRawCubeAction`, `ingest/xgmap.go`) passerait
  donc pour un vrai Double/Take/Pass et entrerait au dénominateur du PR.
  Aucune occurrence connue dans les corpus. Correctif si ça mord un jour :
  tester `engine.CanonicalCubeAction` plutôt que les graphies littérales.
- **`SwapPlayers`/`DeleteCascade` partagés** entre backends via closures SQL
  dialectales (~90 lignes dupliquées à l'octet près).
- **Découpage `db_session.go`** (603 lignes, 5 responsabilités) — les
  migrations mortes ont été supprimées (fiche 07), la précondition est levée.
- **Dictionnaire zlib partagé** pour `analysis.data` (−25 à −40 % de la plus
  grosse table ; nouveau format de blob versionné, fuzz à étendre) ; au
  passage, fusionner les fragments d'analyse en mémoire avant compression
  (aujourd'hui recompression niveau 9 par fragment) et réutiliser des
  prepared statements sur le chemin d'import.
- **`migrate.Run` par lots avec reprise** (aujourd'hui une transaction unique
  pour toute la copie SQLite→PG) — attendre un besoin terrain.
- **Colonne `creation_date` promue dans `analysis`** + index (pushdown SQL du
  filtre de date) : exige bump `DatabaseVersion` + triple synchro schéma +
  migration. La fiche 05 a réglé le N+1 ; ceci est l'étape structurelle.
- **Troisième copie des helpers de recherche** : `database/db_search.go` et `db_filter_match.go` réécrivent les fonctions pures de `storage/searchfilter` ; les faire importer le paquet (aucun cycle). Effort S.
- **Tests par backend redondants** : `comments_*_test.go` / `collections_*_test.go` de `storage/sqlite` et `storage/postgres` doublonnent les cas de contrat ajoutés le 2026-09-02. Effort S.
- **Étapes de migration 1.0.0→1.6.0** : la bizarrerie « table déjà présente ⇒ chaîne arrêtée » (`errStepNotApplicable`, registre `migrationSteps`) est conservée par fidélité ; rendre ces étapes inconditionnelles (leur DDL est `IF NOT EXISTS`). Effort S, test de migration à ajouter.
- **`cli_import.go` enregistre une position texte via `SavePosition`** et non `SaveIndividualPosition` : la provenance `individually_imported` (ADR-0001) n'est pas posée depuis la CLI. Effort S.
- **Drapeau `python-format` faux positif dans les `.po`** : Babel marque les msgid contenant « 12 % de » ; `msgfmt -c` échoue sur 2 à 4 entrées par langue, Sphinx s'en moque. Remède côté extraction (`no-python-format`) ou reformulation. Effort S.

## Ouvert — Moteur (dettes nommées dans les ADR)

- **`use_cube` à la recherche** (ADR-0016, point 4) : le port de `search.go`
  honore `use_match` ; la branche `use_cube` de `value_from_probs` reste à
  porter. Le résidu attendu de la gate d'intégration (2 cas rouges sur 669,
  `integration_gate_test.go`) est documenté dans le test et lui est
  attribué.
- **Renommage `race.Money` → `race.CubeVerdict`** et libellé de la colonne
  Équité (ADR-0016, points 5-6) : différés pour ne pas entrer en collision
  avec l'ADR-0017, désormais fusionné — plus rien ne bloque.
- **Une évaluation refusée est un état nommé, pas une cellule vide**
  (ADR-0017, Consequences) : aujourd'hui l'erreur est avalée.
- **Assertion Playwright « le panneau Eval ne défile pas »** (ADR-0017/0018) :
  la règle est énoncée, aucun test ne la garde.
- **Refuser de démarrer sans drapeau explicite « derrière un proxy »**
  (ADR-0005, option différée) : friction pour tout déploiement légitime pour
  attraper une erreur que la doc, `--help` et le Dockerfile signalent déjà.
  À revisiter si un déploiement nu survient réellement.

## Ouvert — Frontend

- **Parseur de recherche unique** : `commandProcessor.parseFilters` et
  `searchFilterService.parseSearchCommand` divergent déjà ; converger vers
  une grammaire unique testée (les deux suites existantes servent de filet
  croisé).
- **`openPanels` dérivé d'`activeTabStore`** : deux sources de vérité pour
  le panneau visible, état incohérent atteignable (onglet surligné, panneau
  vide) ; supprimer `tabHandler.js`.
- **`MatchDetailPane.svelte` + `InlineAutocomplete.svelte`** : découpage de
  MatchPanel et dédup de l'autocomplétion inline avec TournamentPanel.
- **Virtualisation des listes** (matchs, transcript ~300-500 coups,
  collections) et **LIMIT/pagination** de la recherche (aujourd'hui tout le
  jeu de résultats traverse l'IPC Wails en un message) ; `ListOpts` existe
  déjà dans le contrat storage, passé vide.
- **`generateXGID` est avec perte** (`positionService.js`) : longueur de match = plus grand score away, Crawford déduit, Jacoby/cube max émis à 0 ; le corpus `testdata/xgid_corpus.json` fige ce comportement. Encoder depuis `match_length` et le drapeau Crawford réel. Effort S.
- **`AnalysisPanel.svelte` garde sa copie des prédicats « coup joué »** ; `utils/analysisRows.js` exporte `playedMovePredicate`/`playedCubePredicate`. Effort S.
- **Escape avalé dans le panneau Recherche** tant qu'un champ a le focus (`SearchPanel.handleKeyDown` stoppe tout) ; **double-clic instable sur une ligne du panneau Match** (le premier clic ouvre le volet transcript qui décale la mise en page) ; **sortie du mode match** : `loadAllPositions` repart sur la dernière position, pas celle quittée. Trois points d'ergonomie relevés par les specs e2e. Effort S chacun.
- **`enterEPCMode` peut photographier le plateau EDIT vide** quand App enchaîne `exitEditMode` puis `enterEPCMode` avec un index sauvegardé inchangé (0→0, pas de redessin). Préexistant, documenté dans `modeMachine.js`. Effort S.
- **Plateau et `MatchInfoBar`** : le plateau ne se réajuste pas quand la barre de match apparaît (la capture d'écran force un `resize`) ; `panelLayoutStore.js` dit `DEFAULT_PANEL_HEIGHT = 380 « mirrors config.go »` alors que `config.go` dit 250. Effort S.
- **`PickList.svelte` partagé** Export/Picker (~110 lignes de CSS dupliquées) et **dialogue de sauvegarde de `SearchPanel`** à migrer sur `Modal.svelte`. Effort S.
- **ADR-0004** cite encore `NotoSansJP-Regular.ttf` (5,7 Mo) : la police est un sous-ensemble WOFF2 de 178 Ko depuis le 2026-09-02 (constat historique, une note suffit).

## Ouvert — Tests / CI

- **Smoke test GUI sur une vraie base de production migrée** (v2.0.0,
  phase 06) : parcourir chaque filtre de la fenêtre de recherche. Priorité
  basse ; couvert indirectement par `searchFilterService.test.js` (unitaires
  + round-trip).
## Ouvert — Produit / docs

- **`epc.race` / défi** : vérifier la couverture de l'aide intégrée
  (`help/*.js`) — le panneau Eval a été redessiné trois fois depuis la fiche
  10 (ADR-0017/0018/0021).
- **Soumissions humaines** : PR winget (`microsoft/winget-pkgs`, manifestes attachés à chaque release) et tap Homebrew `kevung/homebrew-tap` (cask attaché à chaque release) — voir `packaging/winget/README.md`, `packaging/homebrew/README.md`.
- **Job `test-os`** : retirer `continue-on-error: true` après le premier run vert sur windows/macos.
- **Skill `release-blunderdb`** : mentionner les nouveaux assets de release (manifestes winget, cask Homebrew, bundle Flatpak, image GHCR).

## Historique — items faits

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
