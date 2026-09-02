<!-- Fiche du plan tasks/gammonnet-perf/README.md. Le contexte, les décisions
et le tableau d'ensemble vivent dans le README ; cette fiche ne porte que son
propre travail et ses propres chiffres. -->

# F4 — Le panneau en direct utilise tous les cœurs [M] — #148 — **FAIT**

- [x] `domaineval.go` (`EvaluatePosition`) et `gammonnet_eval.go` (faits de position, régime
      de course) : `WithWorkers(LiveWorkers(ply))`. `LiveWorkers` rend `runtime.NumCPU()` à
      partir de 2 ply et **1 en dessous** — le palier 0-ply synchrone coûte moins d'une
      milliseconde et n'aurait que des barrières à y gagner. Le **lot** n'entre pas par là :
      il garde son `Searcher` série par goroutine (`NewBatchSearcher` / `EvaluatePositionWith`,
      F3), et le commentaire de `LiveWorkers` dit pourquoi.
- [x] `rollsInParallel` : le tourniquet statique `r += nw` est remplacé par une **file à
      compteur atomique** (`runRollTasks`), tâches pré-triées par coût décroissant (LPT).
      Les résultats vont dans des emplacements fixés avant le départ de la file ; la somme
      pondérée reste sérielle, par candidat, en `r` croissant, en float64.
- [x] **Frontière aplatie** (`deepenLevel`) : tous les candidats à approfondir d'un niveau
      partent dans **une seule file** — 3 × 21 = 63 tâches à la configuration canonique
      2 ply — avec **un seul `WaitGroup.Wait()` par décision** au lieu d'une barrière par
      candidat.
- [x] `WithWorkers` ne borne plus à `NumRolls` mais à `maxUsefulWorkers()` = 21 ×
      `max(Filter[1..Ply])` : brider à 21 laisserait les machines à plus de vingt cœurs
      sur la table maintenant qu'un niveau porte 63 tâches.
- [x] `TestParallelSearchIsBitIdentical` est devenu tabulaire : `WithWorkers` ∈
      {1, 2, `NumCPU`, 64}, tous verts. Gold, cube gold, parité et identité de noyau verts
      **sans retouche**, `-race` vert sur ces tests.
- [x] Effet de bord du branchement : construire 17 chercheurs par décision coûtait 13 ms et
      71 Mo. Les brouillons de niveau (`plays`, `cands`) et surtout les **générateurs de
      coups** (166 Ko pièce, six par chercheur — 1 Mo de `struct Searcher`) sont désormais
      alloués **au premier usage** : `sizeof(Searcher)` passe de 1 003 912 à 8 248 octets,
      la construction à 8 ouvriers de 13,3 ms / 71,5 Mo à **6,0 ms / 33,2 Mo**, et la
      recherche sérielle y gagne 3 % au passage (325 → 316 ms, min sur 9).

#### Mesure (2026-09-02)

Même machine que F3 : Ryzen 7 PRO 6850U (**8 cœurs physiques / 16 threads**, mobile
15-28 W, gouverneur `powersave` — pas de droits pour le passer en `performance`).
Décision 2-ply canonique (ouverture, 3-1, `DefaultConfig(2)`), cache vidé avant chaque
répétition, 13 répétitions, **p75 et p95, jamais de moyenne**. « 8 cœurs physiques » =
`taskset -c 0,2,4,6,8,10,12,14` + `GOMAXPROCS=8` (les frères SMT sont `(0,1)`, `(2,3)`…
sur cette puce : sans épinglage, `GOMAXPROCS=8` laisse Linux poser deux fils sur un même
cœur et perd un tiers du gain). Deux campagnes alternées A/B/A/B, le portable dérivant
thermiquement d'une dizaine de pourcents entre deux runs.

**Sur 8 cœurs physiques, 8 ouvriers** :

| version | p75 | p95 | ×p75 | ×p95 |
|---|---|---|---|---|
| `main` (e71e4649) | 68,2 ms / 64,9 ms | 69,5 ms / 67,7 ms | ×3,73 / ×3,85 | ×3,71 / ×4,19 |
| cette branche | 61,7 ms / **55,4 ms** | 63,1 ms / 56,0 ms | ×4,09 / **×4,43** | ×4,55 / ×4,89 |

**Sur 16 threads (SMT), 16 ouvriers** : `main` ×4,05 (63,2 ms), branche **×4,47** (54,9 ms)
— le SMT n'apporte donc **rien** au-delà des 8 cœurs physiques (×4,43 → ×4,47), exactement
ce que P3 annonçait pour du calcul vectoriel qui sature les unités FP.

**Ce que chaque étape a rendu séparément** (première campagne, `GOMAXPROCS=8` non épinglé,
8 ouvriers, p75) :

| état | ×p75 | temps |
|---|---|---|
| départ (tourniquet statique, barrière par candidat) | ×2,93 | 106,5 ms |
| file atomique LPT, barrière par candidat conservée | ×3,37 | 89,8 ms |
| frontière aplatie, ordre des lancers naturel | ×3,52 | 84,9 ms |
| frontière aplatie **+** ordre LPT | **×3,72** | 81,4 ms |

L'ordonnancement dynamique fait donc les deux tiers du chemin, l'aplatissement le tiers
restant, et le tri LPT en lui-même ne vaut que ~5 % — pour la raison mesurée ci-dessous.

**Temps passé en barrière** (occupation des ouvriers pendant le niveau parallèle, mesurée
en instrumentant `runRollTasks`, 8 ouvriers) : **~15 % d'oisiveté** avec 21 tâches par
barrière, **3 à 5 %** avec les 63 tâches du niveau aplati. C'est le seuil d'arrêt de P3
(~2 %) à peu de chose près ; descendre plus bas demanderait de découper le sous-arbre d'un
lancer, ce qui n'est pas justifié tant que le plafond réel est ailleurs (voir plus bas).

**Le coût des 21 lancers, mesuré — et le proxy « les doubles d'abord » est FAUX.**
`TestProbeRollCost` (24 positions du corpus, 2 ply) donne le nombre d'évaluations que
déclenche le sous-arbre de chaque lancer :

| rang | lancers |
|---|---|
| les plus chers | 2-6 et 3-6 (17 256), 2-3 (17 184), 3-4 (17 160), 1-4 (16 920) |
| les moins chers | 6-6 (11 232), 1-1 (11 736), 3-3 (11 928), 5-5 (11 976) |

Les doubles génèrent bien plus de coups légaux (1 800 pour 2-2 contre 168 pour 5-6) mais
sont parmi les **moins** chers : l'élagage n'en garde que douze, et la position qu'ils
laissent est plus contrainte, donc son sous-arbre est plus étroit. Seule la passe du petit
réseau paie la largeur, et elle est bon marché. L'écart total n'est que de **1,54× en
évaluations et 1,30× en temps** — la variance de coût que la fiche et P3 supposaient
n'existe pas, ce qui explique que LPT ne rende que 5 % et que l'essentiel du gain vienne du
nombre de tâches. L'ordre retenu (`rollsByCost`) est celui des évaluations, déterministe et
indépendant de la machine, et non celui des temps.

**Cache partagé : NON, et le chiffre qui le dit.** `TestProbeCacheHitRate`, décision 2-ply
canonique : une recherche sérielle consulte une seule table et voit donc exactement la
suite de recherches qu'un cache partagé verrait.

| ouvriers | évaluations | hits | taux |
|---|---|---|---|
| 1 (≡ cache partagé) | 13 438 | 1 704 | **11,25 %** |
| 8 (cache par ouvrier) | 14 214 | 928 | **6,13 %** |

Soit **+5,12 points**, tout juste au-dessus du seuil de 5 points de P3 — et cette valeur
**surestime** le gain, la table sérielle n'ayant que 65 536 entrées pour tout le travail
là où les huit tables en ont huit fois plus au total. Ce que ces 5 points achètent en
temps : 776 évaluations de moins sur 14 214, soit **5,5 % du travail réseau, ~4 % de la
décision**. En face : le tag de Hyatt classique XORe la clé avec les données et suppose
donc une **signature d'un mot**, alors que ce cache compare la **position entière** (29
octets) — c'est précisément ce qui garantit qu'une collision de hachage produit un *miss*
et jamais une évaluation fausse. Le garder demanderait une variante multi-mots (8
`atomic.Uint64` et une somme de contrôle par consultation) sur le chemin le plus chaud du
moteur. Quatre pour cent ne paient pas ce risque, surtout quand la recherche est déjà à
93-100 % du plafond de la machine. **Décision : cache par ouvrier conservé.**

**Pourquoi ×4,4 et non ×5,5 : c'est la machine, mesuré.** `TestProbeMachineCeiling` fait
tourner N décisions 2-ply **indépendantes**, une par goroutine, en boucle, et compare les
débits en régime établi : aucune barrière, aucun déséquilibre, rien de partagé que les
poids en lecture seule. C'est le parallélisme embarrassant, donc le plafond du matériel
pour ce travail-ci.

| goroutines | débit× (8 cœurs physiques) |
|---|---|
| 4 | ×2,87 – ×3,00 |
| 8 | **×3,98** |
| 12 | ×4,21 – ×4,42 |

La recherche parallélisée atteint ×4,09-4,43 à 8 ouvriers : elle est **au niveau, voire
au-dessus, de ce plafond** (la sonde de plafond est légèrement pessimiste — huit décisions
indépendantes touchent huit ensembles de positions disjoints, là où huit ouvriers d'une
même décision partagent la racine). Le ×5,5 visé n'est pas atteignable sur cette machine
quel que soit l'ordonnancement : c'est le même mur que F3 a rencontré et diagnostiqué
depuis l'autre côté (×4,08 inter-positions, identique à 1-ply et 2-ply). Sur un 6850U
mobile en `powersave`, la fréquence tous cœurs soutenue est très inférieure au boost
mono-cœur, et huit chercheurs relisent chacun les 2,1 Mo de poids dans un L3 partagé.
**Le plafond est à remesurer sur une machine de bureau non contrainte avant de conclure
quoi que ce soit sur l'ordonnancement.**

#### Sondes laissées en place

`pkg/blunderdb/engine/gammonnet/parallel_probe_test.go`, toutes derrière
`BLUNDERDB_PROBE` (elles n'assertent rien) :
`TestProbeParallelSpeedup` (p50/p75/p95 par nombre d'ouvriers),
`TestProbeMachineCeiling` (plafond matériel, débit en régime établi),
`TestProbeRollCost` (coût des 21 lancers, trois classements),
`TestProbeCacheHitRate` (taux de hit par ouvrier contre cache partagé simulé).

Recette : `taskset -c 0,2,4,6,8,10,12,14 env GOMAXPROCS=8 BLUNDERDB_PROBE=1 go test
-count=1 -run TestProbeParallelSpeedup -v ./pkg/blunderdb/engine/gammonnet/`
(`-count=1` est obligatoire : sans lui `go test` sert le résultat en cache et l'on
« mesure » deux fois le même run).

#### Laissé ouvert

- **`probsAt` (la décision de videau) garde sa boucle de 21 lancers sérielle.** Seul le
  `rankPlays` interne est parallèle, ce qui donne 21 barrières de 21 tâches au lieu d'une
  de 441 : c'est le chemin qui profite le moins de F4. Fiche à part.
- **Le pool est reconstruit à chaque décision** (6 ms, 33 Mo à 8 ouvriers, soit ~10 % de la
  latence). `Searcher.Reconfigure` (F3) permettrait au panneau de garder son chercheur et
  ses ouvriers d'une position à l'autre, comme le lot le fait déjà.
- **Le plafond matériel n'a été mesuré que sur ce portable.** Les chiffres de plafond, et
  donc la conclusion « c'est la machine », demandent une contre-mesure sur une machine de
  bureau.
