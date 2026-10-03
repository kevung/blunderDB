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

## Échantillon BMAB réel

Mêmes bancs sur 299 matchs tirés du corpus BMAB (graine 521, 300 fichiers dont un refusé à
l'import) : 148 905 positions, base de 118 Mo. Import : 21 min de CPU utilisateur. Même machine,
charge moyenne ≈ 30 : ordres de grandeur.

| Filtre | Avant : tous les ids | Fenêtre 1 (100) | Count | Rang du dernier |
|---|---:|---:|---:|---:|
| Large, SQL : toute la base (148 905) | 3,58 s, 1,73 Go alloués | 0,6 ms, 5 Ko | 214 ms | 156 ms |
| Large, SQL : videau (56 865) | 1,56 s, 560 Mo | 71 ms | 120 ms | 147 ms |
| Étroit indexé : dés et score (36) | 13 ms | 4,6 ms | 4,3 ms | 6,6 ms |
| Non indexé, en Go : date d'analyse (148 904) | 33,9 s, 2,16 Go | 106 ms, 2,5 Mo | 23,0 s | 18,2 s |

Lecture : les écarts de la base synthétique tiennent sur des positions réelles. La première
fenêtre reste sous la seconde, même avec une phase Go (106 ms contre 34 s pour tout lire) ; le
compte d'un filtre Go est un balayage complet (23 s sur 149 k positions, 1,7 Go alloués en
tout, mémoire vive bornée par le morceau). C'est pourquoi la GUI affiche la fenêtre 1 avant le
compte et laisse Échap interrompre le balayage.

## Ouverture de la bibliothèque

`BLUNDERDB_SEARCH_DB=<copie> go test -run '^$' -bench '^BenchmarkLibraryOpen' -benchtime 5x
./pkg/blunderdb/database` (`library_open_benchmark_test.go`) : ce que la GUI attend avant le premier
plateau (ouvrir, compter, la page d'ids de la dernière position où `loadAllPositions` se pose, les
positions autour). Base synthétique de GB0.1 (`cmd/blunderdb-synthdb`, graine 1) à 1 000 046
positions, 3 148 matchs, 803 Mo ; 10 M ne tient pas sur le disque de cette machine (≈ 8 Go, 7 Go
libres). Charge moyenne ≈ 40 : ordres de grandeur.

| Étape | `OFFSET` depuis le début | Depuis la fin |
|---|---:|---:|
| Ouvrir la base | 3 ms | 7 ms |
| `CountPositions` | 2 ms | 4 ms |
| Page de la dernière position (1 000 ids) | 0,35 à 1,76 s | 14 ms |
| 50 positions autour | 4 ms | 3 ms |
| Compte + page + positions, bout à bout | 0,35 s | 19 ms |

Lecture : `OFFSET ≈ total` saute chaque ligne une à une, linéaire en la taille ; à 10 M la page
de la dernière position aurait dépassé les 2 s. SQLite répond désormais une fenêtre de la
seconde moitié en partant du dernier id (`ORDER BY id DESC`), en une requête. `GetAllMatches`,
que l'onglet Matchs charge en parallèle, prend 30 s sur ces 3 148 matchs : hors du chemin du
premier plateau, mais l'onglet reste vide pendant ce temps.
