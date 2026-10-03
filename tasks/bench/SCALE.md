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

**Pourquoi 100 000 et pas 1 M** : à ce débit, 1 M de positions coûtait plus de 3 h de
génération. La base de 1 M a été produite après le codec du lot GB1, plus bas ; le job
`scale-bench` de `nightly.yml` la construit (une fois, puis en cache) avec
`SCALE_POSITIONS=1000000`.

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

# Après le lot GB1 — base de 1 M

Mesures du 2026-10-03, branche `feat/gb-c1-mesure` après fusion de `main` @ 4acd39446 (codec du
chantier GB-C2), même poste, au repos.

| Grandeur | Valeur |
|---|---|
| Commande | `go run ./cmd/blunderdb-synthdb -out scale-1m.db -positions 1000000` |
| Matchs / positions | 3 148 / 990 212 |
| Durée de génération | **11 min 4 s, 1 505 positions/s** (contre 88/s avant : ×17, le générateur n'ayant ni lecture de fichier ni démarrage de processus par match) |
| Pic RSS | 384 Mo |
| Taille | 803 Mo, ≈ 811 o/position |

Brut dans `scale-1m-apres-gb1.txt` (`-count 3`).

| Benchmark | Médiane de 3 | 100 000 (avant GB1) | Lignes rendues |
|---|---:|---:|---:|
| `ImportSingleXG` (base pleine) | **0,23 s** (0,55 s au premier passage, cache froid) | 4,95 s | — |
| `SearchWideCube` | 5,53 s | 0,70 s | 398 478 |
| `SearchWideErrorAboveTenth` | 2,03 s | 0,28 s | 22 037 |
| `SearchNarrowDiceAndScore` | 0,39 s | 51 ms | 11 |
| `StatsCompute` sans filtre | **66 s** (99 s cache froid) | 7,69 s | — |
| `ListPositionIDs` | 0,38 s | 59 ms | 990 216 |
| `GetAllMatches` | **5,73 s** | 0,75 s | 3 148 |

Lecture : l'import est réglé par le lot GB1 (×21 sur base pleine). Tout le reste croît
linéairement avec la base, sans index qui sauve une recherche étroite (0,39 s pour 11 lignes :
balayage) ; les stats sans filtre (66 s) et la liste des matchs (1,8 ms par match) sont
inutilisables à cette taille et le seront dix fois plus à 10 M — ce sont les cibles des lots 3
et 5 (stats matérialisées, GB5.6). Le premier passage, cache froid, est plus lent
que les suivants (×1,5 pour les stats, ×2,4 pour l'import) : le job de nuit, en un seul passage sur base fraîchement restaurée, est toujours
froid, donc comparable d'une nuit à l'autre.
