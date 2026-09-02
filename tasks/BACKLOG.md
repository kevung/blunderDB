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
- **`runMigrationChain` en table** : remplacer les ~550 lignes de cascades
  `if version ==` par `[]step{from,to,fn}` + helper `addColumnIfMissing` ;
  corriger la comparaison lexicographique (`db_migration.go`) et le
  `defer rows.Close` retardé. Effort M, risque élevé (chemin des bases
  utilisateurs) — s'appuyer sur `migration_test.go`. Repris en lot 2 du plan
  2026-09.
- **Fusion des 13 helpers purs dupliqués** entre
  `search_helpers_sqlite.go`/`search_helpers_postgres.go` (~290-380 lignes
  identiques après normalisation des placeholders) → package partagé. La CI
  Postgres (fiche 01) tourne : plus de précondition.
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

- **`utils/rangeFilters.js` / `filterModel.js`** : remplacer les ~120
  `$state` plats de `SearchPanel` (22 filtres × 5 variables, réénumérées dans
  4 fonctions) par une table déclarative + round-trip testé. Le plus gros
  levier sur le coût marginal d'une feature de recherche. Effort L, TDD
  obligatoire. Filet : `searchFilterService.test.js`.
- **Parseur de recherche unique** : `commandProcessor.parseFilters` et
  `searchFilterService.parseSearchCommand` divergent déjà ; converger vers
  une grammaire unique testée (les deux suites existantes servent de filet
  croisé).
- **Composant `<Modal>` unique** (overlay + rôles + focus trap + Escape) :
  13 copies du CSS d'overlay, 2 conventions de z-index, 7 gestionnaires
  d'Escape dupliqués.
- **`utils/boardRenderer.js` / `boardScene.js`** : extraire les ~600 lignes
  de dessin pur de `Board.svelte` (précédent réussi : `boardGeometry.js`),
  séparer décor statique/couches dynamiques — `drawBoard()` recrée ~120
  nœuds à chaque survol de coup. Précondition : tests de caractérisation sur
  `getDisplayPosition()`.
- **`openPanels` dérivé d'`activeTabStore`** : deux sources de vérité pour
  le panneau visible, état incohérent atteignable (onglet surligné, panneau
  vide) ; supprimer `tabHandler.js`.
- **`MatchDetailPane.svelte` + `InlineAutocomplete.svelte`** : découpage de
  MatchPanel et dédup de l'autocomplétion inline avec TournamentPanel.
- **Virtualisation des listes** (matchs, transcript ~300-500 coups,
  collections) et **LIMIT/pagination** de la recherche (aujourd'hui tout le
  jeu de résultats traverse l'IPC Wails en un message) ; `ListOpts` existe
  déjà dans le contrat storage, passé vide.

## Ouvert — Tests / CI

- **Smoke test GUI sur une vraie base de production migrée** (v2.0.0,
  phase 06) : parcourir chaque filtre de la fenêtre de recherche. Priorité
  basse ; couvert indirectement par `searchFilterService.test.js` (unitaires
  + round-trip).
- **Contrat storage : familles restantes** (Anki complet, Collections,
  Tournaments, Comments, Metadata, Session) — remonter les tests par famille
  en supprimant les doublons par backend ; `storagetest/contract.go` à
  découper (2 486 lignes).
- **Matrice OS du job `test`** (windows/macos) — s'attendre à de vrais
  échecs (chemins, accents dans testdata) ; à faire quand la suite est
  stabilisée.
- **E2E des parcours produit** : import de match, recherche, suppression
  avec confirmation, collections drag, Anki review, export, édition de
  position. Infra mock Wails déjà en place ; les 5 specs actuelles ne
  couvrent aucun des 4 flux majeurs.
- **Job docs** : filtrage par chemins + cache pip + PDF LaTeX sur tag
  seulement (attention au chemin de publication des PDF sur tag, déjà cassé
  une fois).

## Ouvert — Produit / docs

- **Capture d'écran du README** : `doc/source/_static/screenshot.png` date du
  2026-04-20, montre un onglet Log retiré depuis et précède le panneau Stats.
  La fiche 10 a retiré la référence plutôt que de livrer une image
  trompeuse. Refaire la capture sur 0.34+ (`make dev`, base peuplée, fenêtre
  principale) et restaurer l'image ; **captures du manuel** (0 illustration
  hors annexes Windows) et **vidéo de démo** (issue #102) — dépendent d'une
  session graphique et d'un choix éditorial (localisation des captures).
- **`epc.race` / défi** : vérifier la couverture de l'aide intégrée
  (`help/*.js`) — le panneau Eval a été redessiné trois fois depuis la fiche
  10 (ADR-0017/0018/0021).

## Historique — items faits

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
