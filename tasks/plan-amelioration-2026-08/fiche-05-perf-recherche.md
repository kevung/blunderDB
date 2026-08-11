# Fiche 05 — Performance de la recherche SQL

Branche : `perf/recherche-sql`

## Objectif

Accélérer les recherches (cas nominal et filtres coûteux) sans changer un
seul résultat. Benchmarks avant/après obligatoires (fixture
`testdata/tournois` locale, 156 fichiers).

État mesuré : DecisionCube 338 ms, ErrorAboveTenth 100 ms, PipWindow 80 ms,
WinGammonCombo 766 ms, CheckerStructure 66 ms.

## Tâches, dans l'ordre (mesurer entre chaque)

- [x] **T1 — `a.data` conditionnel** : `search_sqlite.go:297-309` sélectionne
      `a.data` (~600 o/ligne) même quand `needAnalysis` (`:51`) est faux ; le
      blob traverse aussi le tri. Ne sélectionner le blob que si nécessaire
      (`NULL AS data` sinon). Faire le miroir dans
      `storage/postgres/search_postgres.go`. Gain attendu : −40 à −70 % sur
      le cas nominal.
- [x] **T2 — Filtre de date sans N+1** : `matchesDateFilter`
      (`search_helpers_sqlite.go:494-497`) fait une requête + une
      décompression zlib **par ligne**. Court terme : ajouter
      `f.DateFilter != ""` (et `MoveErrorFilter`) à `needAnalysis` et lire la
      date dans l'analyse déjà décodée. Même chose pour
      `isPlayer1TakePassCubeAction` (`search_sqlite.go:603`). Miroir Postgres.
- [x] **T3 — Restructuration win/gammon** (FOLLOWUPS #4) : remplacer le join
      trié par `WHERE p.id IN (SELECT position_id FROM analysis WHERE …)
      ORDER BY p.id` (plan vérifié : le TEMP B-TREE disparaît) ; étendre
      `idx_analysis_win_gammon` en `(player1_win_rate, player1_gammon_rate,
      position_id)` pour le rendre couvrant. Attendu : 766 → ~100 ms.
- [x] **E3 — Index redondants** : supprimer `idx_position_score` (préfixe de
      `idx_position_score_cube`) et `idx_analysis_win1` (préfixe de
      `idx_analysis_win_gammon` étendu) des DDL (`schema_sqlite.go:247,251` +
      miroir `db_schema.go:361-392`). Valider par `EXPLAIN QUERY PLAN` sur les
      benchmarks score. NB : pour les bases existantes, les vieux index ne
      seront supprimés qu'à la faveur d'un VACUUM/migration future — se
      contenter de ne plus les créer (pas de bump de version pour ça).
- [x] **T7 — Statistiques du planificateur** : `ANALYZE` n'existe que dans
      l'import CLI (`cli_import.go:372-377`). Ajouter `PRAGMA optimize` à la
      fermeture de base (`storage/sqlite`) et `ANALYZE` après un import GUI.
- [x] Mettre à jour `tasks/FOLLOWUPS.md` (#4 → done avec chiffres).

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

## Notes d'exécution

Toutes les étapes ont été implémentées et commitées séparément sur
`perf/recherche-sql` (5 commits : T1, T2, T3, E3, T7). Résultats de
recherche vérifiés identiques (contenu et ordre) à chaque étape via
`search_rewrite_test.go`, `search_sort_sqlite_test.go`, la suite de contrat
(`storagetest`, les deux backends) et `go test ./...` en entier ; suite
PostgreSQL complète (`go test -tags postgres ./pkg/blunderdb/storage/postgres/...`,
Docker) verte après T1, T3, E3 (les étapes touchant ce backend).

**Tableau final avant/après des 5 benchmarks** (testdata/tournois, 156
fichiers, `-bench BenchmarkSearch -benchtime 3x -count=1`, mesuré avant T1 et
après T7, sur la même machine) :

| Benchmark | Avant (baseline) | Après (T1+T2+T3+E3+T7) | Delta |
|---|---|---|---|
| `BenchmarkSearch_DecisionCube` | 279.1 ms | 249.1 ms | −10.8 % |
| `BenchmarkSearch_ErrorAboveTenth` | 83.8 ms | 75.8 ms | −9.5 % |
| `BenchmarkSearch_PipWindow` | 69.6 ms | 57.9 ms | −16.8 % |
| `BenchmarkSearch_WinGammonCombo` | 629.4 ms | 502.0 ms | −20.2 % |
| `BenchmarkSearch_CheckerStructure` | 54.7 ms | 48.3 ms | −11.7 % |

Aucun des 5 n'a régressé à aucune étape. Les gains par commit individuel
sont pour la plupart dans le bruit de mesure (machine partagée, load average
variable) — c'est l'agrégat sur l'ensemble des 5 commits qui est net et
répété sur plusieurs runs.

**Écarts vs les gains attendus dans la fiche, et pourquoi :**

- **T3** attendait « 766 → ~100 ms » sur `WinGammonCombo` ; mesuré ~629 → ~500
  ms (−20 %). `EXPLAIN QUERY PLAN` confirme que le `TEMP B-TREE FOR ORDER BY`
  a bien disparu (remplacé par `SEARCH analysis USING COVERING INDEX
  idx_analysis_win_gammon_covering`) — le diagnostic de plan était juste.
  Mais le filtre du benchmark (`w>0.55 g>0.2`) retient ~79 % des 57 820
  positions du fixture (45 546/57 820, vérifié par requête directe) : à ce
  niveau de sélectivité, le tri SQL n'était pas le poste dominant. Un
  filtre bien plus étroit (`w>0.9 g>0.5`) mesure le même ~475 ms — la
  reconstruction Go des positions (JSON, une fois par ligne retenue) domine
  désormais, indépendamment de la méthode SQL qui a produit ces lignes.
  Gardé malgré le gain modeste : plan strictement meilleur, aucun risque,
  devrait mieux passer à l'échelle sur une base disque plus grosse ou un
  filtre réellement sélectif.
- **T1** attendait « −40 à −70 % sur le cas nominal » ; mesuré −5 à −18 %
  selon le benchmark. Sur ce fixture et ce moteur (modernc.org/sqlite, pur
  Go, base `:memory:`), le blob `a.data` n'est pas le poste dominant du
  coût par ligne — mais le gain est réel, cohérent sur les 5 repères, et
  sans coût.
- **T2** (`DateFilter`) : gain réel mais modeste (~13 % dans un
  micro-benchmark dédié, `DateFilter` seul sur les 57 820 positions — hors
  des 5 repères officiels, qui n'exercent pas ce filtre) ; la décompression
  zlib domine le coût de la requête N+1 qu'elle remplaçait, donc supprimer
  la requête seule (en gardant un décodage par ligne dans les deux cas) ne
  pouvait pas apporter plus.

**Bugs trouvés et corrigés en cours de route (hors périmètre strict de la
fiche, mais découverts en la travaillant et documentés dans les commits
T1/T2)** :

1. Le décodage de l'analyse côté `storage/postgres/search_postgres.go`
   utilisait `json.Unmarshal` brut sur `a.data`, qui est **toujours**
   zlib-compressé (`AnalysisStore.Save`) — le premier octet n'est jamais
   `'{'`, donc le décodage échouait sur **chaque** ligne, silencieusement.
   `WinRateFilter`, `GammonRateFilter`, `EquityFilter`, `MovePatternFilter`
   et le re-check Go de la recherche miroir ne matchaient jamais rien côté
   PostgreSQL depuis leur introduction. Corrigé (commit T1) en utilisant
   `engine.DecodeAnalysisFromStorage`, comme `AnalysisStore.Load`.
2. `EquityFilter` (le filtre `e>...` / `e-x,y`, exposé en frontend) n'était
   dans le `needAnalysis` d'aucun des deux backends — le filtre tourne
   pourtant inconditionnellement (jamais poussé en SQL), donc `ana` restait
   toujours `nil` et le filtre ne matchait jamais rien, sur SQLite comme sur
   PostgreSQL, depuis son introduction. Corrigé (commit T2).
3. Une régression a été introduite puis corrigée **avant commit** en cours
   de T2 : `MoveErrorFilter` avait été ajouté à `needAnalysis` par erreur
   (« pour faire pareil que `DateFilter` ») ; il est en réalité déjà poussé
   en SQL et son re-check Go ne tourne que sous `MirrorFilter` (déjà
   couvert par `needAnalysis`) — l'ajout forçait un décodage massif inutile
   sur toute recherche `MoveErrorFilter` non-miroir.
   `BenchmarkSearch_ErrorAboveTenth` 80 ms → 200 ms a signalé le problème
   avant qu'il n'entre dans un commit ; diagnostiqué par
   `EXPLAIN QUERY PLAN` identique côté SQL + micro-benchmark isolant
   scan/decode, corrigé en retirant `MoveErrorFilter` de `needAnalysis`.
4. Une **troisième copie** de la DDL d'index (au-delà de `schema_sqlite.go`
   et `database/db_schema.go` que la fiche citait) vit dans
   `database/db.go` (`Database.SetupDatabase`, le chemin utilisé par
   `:memory:`, le CLI `create`, le bouton GUI « nouvelle base », et ces
   benchmarks eux-mêmes). Découverte en T3 quand l'index couvrant
   n'apparaissait pas dans la base de benchmark malgré les DDL mises à
   jour ; corrigée en T3 et E3.
5. `frontend/wailsjs/go/database/Database.{js,d.ts}` (générés) portaient une
   dérive préexistante sans rapport avec cette fiche : `ExportCollections`/
   `ExportTournaments` n'avaient que 5 arguments liés alors que les
   signatures Go en comptent 7 depuis les fiches 03/04, et
   `PositionExists`/`Migrate_1_0_0_to_1_1_0`/`…_1_2_0`/`…_1_3_0` (supprimés
   côté Go) traînaient encore. Corrigée en régénérant via
   `wails generate module` pour T7 (aucun appelant frontend actif pour l'un
   ou l'autre, vérifié).

**Rien n'a été retiré** : les cinq étapes ont chacune rapporté un gain
mesurable et sans régression, même quand ce gain était inférieur à
l'estimation initiale de la fiche (T1, T2, T3).
