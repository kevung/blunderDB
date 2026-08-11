# Fiche 05 — Performance de la recherche SQL

Branche : `perf/recherche-sql`

## Objectif

Accélérer les recherches (cas nominal et filtres coûteux) sans changer un
seul résultat. Benchmarks avant/après obligatoires (fixture
`testdata/tournois` locale, 156 fichiers).

État mesuré : DecisionCube 338 ms, ErrorAboveTenth 100 ms, PipWindow 80 ms,
WinGammonCombo 766 ms, CheckerStructure 66 ms.

## Tâches, dans l'ordre (mesurer entre chaque)

- [ ] **T1 — `a.data` conditionnel** : `search_sqlite.go:297-309` sélectionne
      `a.data` (~600 o/ligne) même quand `needAnalysis` (`:51`) est faux ; le
      blob traverse aussi le tri. Ne sélectionner le blob que si nécessaire
      (`NULL AS data` sinon). Faire le miroir dans
      `storage/postgres/search_postgres.go`. Gain attendu : −40 à −70 % sur
      le cas nominal.
- [ ] **T2 — Filtre de date sans N+1** : `matchesDateFilter`
      (`search_helpers_sqlite.go:494-497`) fait une requête + une
      décompression zlib **par ligne**. Court terme : ajouter
      `f.DateFilter != ""` (et `MoveErrorFilter`) à `needAnalysis` et lire la
      date dans l'analyse déjà décodée. Même chose pour
      `isPlayer1TakePassCubeAction` (`search_sqlite.go:603`). Miroir Postgres.
- [ ] **T3 — Restructuration win/gammon** (FOLLOWUPS #4) : remplacer le join
      trié par `WHERE p.id IN (SELECT position_id FROM analysis WHERE …)
      ORDER BY p.id` (plan vérifié : le TEMP B-TREE disparaît) ; étendre
      `idx_analysis_win_gammon` en `(player1_win_rate, player1_gammon_rate,
      position_id)` pour le rendre couvrant. Attendu : 766 → ~100 ms.
- [ ] **E3 — Index redondants** : supprimer `idx_position_score` (préfixe de
      `idx_position_score_cube`) et `idx_analysis_win1` (préfixe de
      `idx_analysis_win_gammon` étendu) des DDL (`schema_sqlite.go:247,251` +
      miroir `db_schema.go:361-392`). Valider par `EXPLAIN QUERY PLAN` sur les
      benchmarks score. NB : pour les bases existantes, les vieux index ne
      seront supprimés qu'à la faveur d'un VACUUM/migration future — se
      contenter de ne plus les créer (pas de bump de version pour ça).
- [ ] **T7 — Statistiques du planificateur** : `ANALYZE` n'existe que dans
      l'import CLI (`cli_import.go:372-377`). Ajouter `PRAGMA optimize` à la
      fermeture de base (`storage/sqlite`) et `ANALYZE` après un import GUI.
- [ ] Mettre à jour `tasks/FOLLOWUPS.md` (#4 → done avec chiffres).

## Critères de fin

- `go test ./...` vert ; `search_rewrite_test.go`, `search_sort_sqlite_test.go`
  et les tests de parité inchangés.
- Tableau avant/après des 5 benchmarks dans le message de commit final ;
  aucun benchmark dégradé.

## Risques & garde-fous

- Chaque étape est un commit séparé avec ses chiffres : si une étape ne
  rapporte rien, la retirer plutôt que l'empiler.
- Les résultats de recherche doivent être identiques à l'octet (ordre
  compris) — c'est le critère de non-régression, pas seulement « les tests
  passent ».
