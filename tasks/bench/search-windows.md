# Recherche par fenêtres — mesures (chantier GB-C4, étape GB3.4)

`BLUNDERDB_SEARCH_DB=<copie> go test -run '^$' -bench '^BenchmarkSearchWindows_' -benchtime 3x
-benchmem ./pkg/blunderdb/database` (`search_window_benchmark_test.go`). Base synthétique de
GB0.1 (`cmd/blunderdb-synthdb`, graine 1) : 99 613 positions. 2026-10-03, Ryzen 7 PRO 6850U,
charge moyenne ≈ 27 (autres sessions) : ordres de grandeur, à re-mesurer machine au repos.

« Avant » est ce que faisait `LoadPositionIDsByFilters` : tout `Find`, chaque position
reconstruite, pour n'en garder que l'id. « Après » est ce que la GUI lit à présent.

| Filtre | Avant : tous les ids | Fenêtre 1 (100) | Count | Rang du dernier |
|---|---:|---:|---:|---:|
| Large, SQL : toute la base (99 613) | 2,32 s, 1,10 Go alloués | 0,5 ms, 5 Ko | 129 ms | 109 ms |
| Large, SQL : videau (40 094) | 1,07 s, 434 Mo | 46 ms | 82 ms | 84 ms |
| Étroit indexé : 6-5 à 6-4 (5) | 63 ms | 58 ms | 61 ms | 4 ms |
| Non indexé, en Go : date d'analyse (99 613) | 9,8 s, 1,42 Go | 117 ms, 2,4 Mo | 15,5 s | 20 s |

Lecture : sans phase Go, la fenêtre est une projection `SELECT p.id … LIMIT` et le compte un
`COUNT(*)`, sans aucun blob décodé. Avec une phase Go (date, équité, motif, miroir, texte, zones),
le balayage lit 256 candidats puis double jusqu'à 4 096 : la mémoire reste bornée par le morceau,
mais le compte et le rang restent un balayage complet avec décodage des blobs — le cas « non
indexé » de l'issue, qui attend la colonne de date de GB4.1.
