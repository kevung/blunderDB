# Backlog — chantiers de fond identifiés, non planifiés

Gros travaux à valeur réelle mais risque/effort élevés. Chacun mérite sa
propre décision (et éventuellement son ADR) avant lancement. Ne pas les
commencer « en passant ».

## Backend

- **Bug `CommitImportDatabase` : colonnes scalaires NULL** (découvert en
  fiche 04) : le chemin legacy d'import de `.db` insère une position « neuve »
  sans remplir les colonnes scalaires de recherche, ce qui casse sa propre
  déduplication au second import et rend ces positions invisibles aux filtres
  SQL. Voir les Notes d'exécution de la fiche 04. Effort S/M, à corriger en
  priorité dans le prochain cycle (même famille que le correctif d'export).
- **Export unifié, schéma unique** : faire de `storage/sqlite/schema_sqlite.go`
  la source DDL unique (7 copies aujourd'hui, 93 `CREATE TABLE`) et fusionner
  les 3+1 chemins d'export en un exporteur paramétré par une sélection
  (`ingest/`), consommé par GUI/CLI/serveur. Règle aussi : matchs absents de
  l'export serveur, watermark côté serveur. Effort L, après fiche 04.
- **`runMigrationChain` en table** : remplacer les 548 lignes de cascades
  `if version ==` par `[]step{from,to,fn}` + helper `addColumnIfMissing` ;
  corriger la comparaison lexicographique (`db_migration.go:2003-2018`) et le
  `defer rows.Close` retardé (`:1700`). Effort M, risque élevé (chemin des
  bases utilisateurs) — s'appuyer sur `migration_test.go`.
- **Fusion des 13 helpers purs dupliqués** entre
  `search_helpers_sqlite.go`/`search_helpers_postgres.go` (~382 lignes
  identiques après normalisation des placeholders) → package partagé.
  S'y attaquer après que la CI Postgres (fiche 01) tourne.
- **`SwapPlayers`/`DeleteCascade` partagés** entre backends via closures SQL
  dialectales (~90 lignes dupliquées à l'octet près).
- **Découpage `db_session.go`** (5 responsabilités) après suppression des
  migrations mortes (fiche 07).
- **Dictionnaire zlib partagé** pour `analysis.data` (−25 à −40 % de la plus
  grosse table ; nouveau format de blob versionné, fuzz à étendre) ; au
  passage, fusionner les fragments d'analyse en mémoire avant compression
  (aujourd'hui recompression niveau 9 par fragment) et réutiliser des
  prepared statements sur le chemin d'import (le commentaire de
  `db_export.go:381-386` chiffre le gain sur le chemin voisin).
- **`migrate.Run` par lots avec reprise** (aujourd'hui une transaction unique
  pour toute la copie SQLite→PG) — attendre un besoin terrain.
- **Colonne `creation_date` promue dans `analysis`** + index (pushdown SQL du
  filtre de date) : exige bump `DatabaseVersion` + triple synchro schéma +
  migration. La fiche 05 règle déjà le N+1 ; ceci est l'étape structurelle.

## Frontend

- **`utils/rangeFilters.js`** : remplacer les 105 `$state` plats de
  `SearchPanel` (21 filtres × 5 variables, 8 points d'édition par filtre)
  par une table déclarative + round-trip testé. Le plus gros levier sur le
  coût marginal d'une feature de recherche. Effort L, TDD obligatoire.
- **Parseur de recherche unique** : `commandProcessor.parseFilters` et
  `searchFilterService.parseSearchCommand` divergent déjà ; converger vers
  une grammaire unique testée (les deux suites existantes servent de filet
  croisé).
- **Composant `<Modal>` unique** (overlay + rôles + focus trap + Escape) :
  13 copies du CSS d'overlay, 2 conventions de z-index, 7 gestionnaires
  d'Escape dupliqués.
- **`utils/boardRenderer.js`** : extraire les ~600 lignes de dessin pur de
  `Board.svelte` (précédent réussi : `boardGeometry.js`), séparer décor
  statique/couches dynamiques. Précondition : tests de caractérisation sur
  `getDisplayPosition()`.
- **`openPanels` dérivé d'`activeTabStore`** : deux sources de vérité pour
  le panneau visible, état incohérent atteignable (onglet surligné, panneau
  vide) ; supprimer `tabHandler.js`.
- **`MatchDetailPane.svelte` + `InlineAutocomplete.svelte`** : découpage de
  MatchPanel et dédup de l'autocomplétion inline avec TournamentPanel.
- **Virtualisation des listes** (matchs, transcript ~300-500 coups,
  collections) et **LIMIT/pagination** de la recherche (aujourd'hui tout le
  jeu de résultats traverse l'IPC Wails en un message).

## Tests / CI

- **Contrat storage : familles restantes** (Anki complet, Collections,
  Tournaments, Comments, Metadata, Session) — remonter les tests par famille
  en supprimant les doublons par backend.
- **Matrice OS du job `test`** (windows/macos) — s'attendre à de vrais
  échecs (chemins, accents dans testdata) ; à faire quand la suite est
  stabilisée.
- **E2E des parcours produit** : import de match, recherche, suppression
  avec confirmation, collections drag, Anki review, export, édition de
  position. Infra mock Wails déjà en place.
- **Job docs** : filtrage par chemins + cache pip (attention au chemin de
  publication des PDF sur tag, déjà cassé une fois).

## Produit / docs

- **Captures d'écran du manuel** (0 illustration hors annexes Windows) et
  **nouvelle vidéo de démo** (issue #102) — dépendent d'une session
  graphique et d'un choix éditorial (localisation des captures).
- **`epc.race` / défi** : vérifier la couverture d'aide intégrée après les
  fiches 07/10.
- **AUR** : relancer `gh run rerun 31328736714` quand l'AUR répond
  (mémoire projet ; pas de retry automatique — demande explicite).
