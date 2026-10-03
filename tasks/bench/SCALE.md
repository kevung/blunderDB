# Benchmarks d'échelle — référence avant le lot GB1

Mesures du 2026-10-03, `main` @ 8aaafec12, Ryzen 7 PRO 6850U (16 threads, 14 Go), SQLite sur
NVMe. Un import CLI tournait en parallèle sur un autre cœur (mesures GB0.2) : les chiffres sont
une référence d'ordre de grandeur, à re-mesurer machine au repos avant d'être cités ailleurs.

## La base

`go run ./cmd/blunderdb-synthdb -out scale-100k.db -positions 100000` (graine 1, fixtures
`testdata/*.xg` et `testdata/*/*.xg` sauf `testdata/test.xg`) :

| Grandeur | Valeur |
|---|---|
| Matchs / positions | 315 / 99 613 |
| Durée de génération | 19 min 1 s, **88 positions/s** (120/s au départ, 88/s en moyenne : le débit baisse quand la base grossit) |
| Pic RSS du générateur | 296 Mo |
| Taille | 77,3 Mo, **≈ 776 o/position** |

**Pourquoi 100 000 et pas 1 M** : au débit actuel, 1 M de positions coûte plus de 3 h de
génération, et 10 M plus d'une journée. La base de 1 M sera produite après le codec du lot GB1
(chantier GB-C2), qui vise ×8 à ×10 sur ce débit ; le job `scale-bench` de `nightly.yml` la
construit (une fois, puis en cache) avec `SCALE_POSITIONS=1000000`.

## Les chiffres

`BLUNDERDB_SCALE_DB=… go test -run '^$' -bench '^BenchmarkScale_' -benchtime 1x -count 3
-benchmem ./pkg/blunderdb/database` ; brut dans `scale-100k-avant-gb1.txt`.

| Benchmark | Médiane de 3 | Lignes rendues |
|---|---:|---:|
| `ImportSingleXG` (465 positions, base pleine) | 4,95 s | — |
| `SearchWideCube` (décisions de videau) | 0,70 s | 40 096 |
| `SearchWideErrorAboveTenth` (`E>100`) | 0,28 s | 2 205 |
| `SearchNarrowDiceAndScore` (6-5, score 6-4) | 51 ms | 5 |
| `StatsCompute` sans filtre | **7,69 s** | — |
| `ListPositionIDs` | 59 ms | 99 617 |
| `GetAllMatches` | **0,75 s** | 315 |

Lecture : à 100 000 positions déjà, les stats sans filtre prennent près de 8 s et la liste des
matchs 0,75 s pour 315 matchs (≈ 2,4 ms par match : une requête par match, à confirmer au
profil). L'import d'un fichier sur base pleine (4,9 s) est 2,5 fois plus lent que sur base vide
(1,8 s, README du plan § 1). Ces trois chemins sont ceux que la base de 1 M départagera.

Reproductibilité : sur trois passages, l'écart à la médiane reste sous 25 % (pire cas
`ListPositionIDs`, 51 à 70 ms) ; le seuil ×1,5 de `scripts/bench-scale-compare.sh` est choisi
au-dessus de ce bruit.
