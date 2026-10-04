# Schéma 2.31.0 complété — plans de requêtes et mesure

Plans EXPLAIN QUERY PLAN sur l'échantillon à 2 % de BMAB (ADR-0071), avant et après le retrait de
`idx_analysis_engine` / `idx_analysis_depth` et l'ajout de `idx_analysis_provenance_pending`.
Aucune requête de recherche ni de match_stats ne lisait ces index ; le filtre de statistiques par
moteur les lisait faute de statistiques, et retombe sur `move` puis `idx_analysis_position`.

## Avant

```
-- backfill probe
`--SEARCH analysis USING COVERING INDEX idx_analysis_engine (analysis_engine=?)
-- backfill batch
`--SEARCH analysis USING INDEX idx_analysis_engine (analysis_engine=? AND rowid>?)
-- stale scan
`--SEARCH analysis USING INDEX idx_analysis_position (position_id>?)
-- search engine token
|--SCAN p USING COVERING INDEX idx_position_match_date
`--SEARCH a EXISTS USING INDEX idx_analysis_position (position_id=?)
-- search depth token
|--SCAN p USING COVERING INDEX idx_position_match_date
`--SEARCH a EXISTS USING INDEX idx_analysis_position (position_id=?)
-- stats engine/depth filter
|--SEARCH a USING INDEX idx_analysis_engine (analysis_engine=?)
|--SEARCH mv USING INDEX idx_move_position (position_id=?)
|--SEARCH g USING INTEGER PRIMARY KEY (rowid=?)
`--SEARCH m USING INTEGER PRIMARY KEY (rowid=?)
-- match_stats group
|--SEARCH g USING COVERING INDEX idx_game_match (match_id=?)
|--SEARCH mv USING INDEX idx_move_game (game_id=?)
|--SEARCH a USING INDEX idx_analysis_position (position_id=?) LEFT-JOIN
`--USE TEMP B-TREE FOR GROUP BY
```

## Après

```
-- backfill probe
`--SCAN analysis USING INDEX idx_analysis_provenance_pending
-- backfill batch
`--SEARCH analysis USING INDEX idx_analysis_provenance_pending (id>?)
-- stale scan
`--SEARCH analysis USING INDEX idx_analysis_position (position_id>?)
-- search engine token
|--SCAN p USING COVERING INDEX idx_position_match_date
`--SEARCH a EXISTS USING INDEX idx_analysis_position (position_id=?)
-- search depth token
|--SCAN p USING COVERING INDEX idx_position_match_date
`--SEARCH a EXISTS USING INDEX idx_analysis_position (position_id=?)
-- stats engine/depth filter
|--SCAN mv
|--SEARCH g USING INTEGER PRIMARY KEY (rowid=?)
|--SEARCH m USING INTEGER PRIMARY KEY (rowid=?)
`--SEARCH a USING INDEX idx_analysis_position (position_id=?)
-- match_stats group
|--SEARCH g USING COVERING INDEX idx_game_match (match_id=?)
|--SEARCH mv USING INDEX idx_move_game (game_id=?)
|--SEARCH a USING INDEX idx_analysis_position (position_id=?) LEFT-JOIN
`--USE TEMP B-TREE FOR GROUP BY
```

## Mesure

Échantillon (positions `id % 50 = 0`, leurs analyses et coups, tous les matchs) ouvert par `main`
puis par cette branche, `VACUUM`, somme `dbstat` hors `match`/`game` : 292,6 Mo → 247,7 Mo (−15,3 %).
Relecture : 312 586 dates de position et 312 544 dates d'analyse identiques à la seconde.
