# gammonNet — accélérer les temps d'évaluation sans bouger un bit

Plan issu de la session de grilling du 2026-09-02. État de départ : `main` @ f4cd6cb3
(0.35.0). Décision d'architecture : [ADR-0024](../../docs/adr/0024-the-evaluator-batches-positions-one-per-lane-and-keeps-the-scalar-sum-order.md)
(proposée, à accepter quand la fiche F1 passe ses tests).

Format des fiches : `[effort S/M/L]` — S ≤ ½ journée, M ≤ 2-3 jours, L = chantier.
Une fiche = une branche = une PR, dans un worktree (`CLAUDE.md`, « Development Workflow »).

---

## 1. Constat mesuré

Machine : Ryzen 7 PRO 6850U, 8 cœurs / 16 threads, AVX2 + FMA, Go 1.25.13, `GOAMD64=v1`.
Position : ouverture 3-1, argent. Recette : `BLUNDERDB_PROBE=1 go test -run
TestProbeDecisionCost ./pkg/blunderdb/engine/gammonnet/` et `-bench BenchmarkEvaluate`.

| Mesure | Valeur |
|---|---|
| Forward pass grand réseau (196→512→512→256→128→5, 527 k MAC) | 404 µs, 0 alloc |
| 0-ply k=12 | 6 ms, 12 évals |
| 1-ply k=12 filtre 3 | 305 ms, 660 évals |
| 2-ply canonique `(0,1,3)` k=12, série | 5,5 s, 13 439 évals grand réseau, 36 395 petit |
| 2-ply canonique, `WithWorkers(8)` | 1,4 s |
| 2-ply canonique, `WithWorkers(16)` | 1,3 s |
| Même décision, C amont batché largeur 32 | 0,56 s |

Profil CPU d'une décision 2-ply : `Evaluator.Evaluate` 97,6 % (grand + petit réseau) ;
génération, tris, cache, videau : 2,4 %, soit ~0,15 s.

### Relevé du probe enrichi (F0, 2026-09-02)

`BLUNDERDB_PROBE=1 go test -run TestProbeDecisionCost`, noyau `go`, largeur de lot 8,
16 cœurs logiques. **Machine non oisive** pendant ce relevé, les six lignes s'enchaînant :
les durées valent ~2× celles du tableau ci-dessus, mesurées isolément. Les *ratios* et le
*remplissage* sont exploitables ; les durées absolues de référence restent celles du
premier tableau, à refaire au repos avant de juger F1.

| configuration | durée | évals | élagage | cache | remplissage |
|---|---|---|---|---|---|
| 0-ply k=12 | 8 ms | 12 | 16 | 0 | 87,5 % |
| 1-ply k=12 sans filtre | 2,159 s | 2 676 | 4 934 | 155 | 81,7 % |
| 1-ply k=12 filtre 3 | 593 ms | 660 | 1 140 | 40 | 81,2 % |
| 2-ply k=12 (0,1,3) | 10,998 s | 13 439 | 36 395 | 1 703 | **84,3 %** |
| 2-ply k=12 (0,2,8) | 53,641 s | 61 836 | 152 005 | 9 472 | 83,0 % |
| 2-ply k=12 (0,1,3), `WithWorkers(16)` | 2,128 s (×5,2) | 14 039 | 36 395 | 1 103 | 84,3 % |

**La décision D4 est tranchée par ce chiffre.** Le remplissage à largeur 8 vaut 84,3 % à
2-ply canonique, très au-dessus du seuil de 70 % que le plan s'était donné : le lot par
nœud suffit, et **le regroupement des 21 lancers n'a pas lieu d'être**. La moitié ouverte
de la fiche F2 (#146) se ferme ici, avec une mesure et non un avis. Le C amont a besoin de
ce regroupement parce qu'il travaille à largeur 32, où le même flux de candidats ne remplit
que 14,5 % des voies ; à largeur 8 le problème n'existe pas.

Deux observations à garder pour F4 (#148) : le parallélisme rend ×5,2 ici sur 16 threads
(×4 mesuré isolément), et il **augmente le nombre d'évaluations** (14 039 contre 13 439,
+4,5 %) parce que chaque worker a son propre cache et redécouvre ce qu'un voisin a déjà
évalué. Un cache partagé rendrait ces 600 évaluations ; c'est chiffré, et c'est une fiche
à part.

### Ce qui explique ces chiffres

1. **Le forward pass est scalaire à accumulateur unique** (`network.go:242-257`) : 0,71 ns
   par MAC ≈ 2,5 cycles, la latence d'une addition float32 non pipelinée. Ce n'est pas la
   bande passante (5,6 Go/s effectifs). Le C amont vectorise un noyau *batché* (activations
   feature-major, 32 positions, une par voie) et obtient ×8,5 sur le même calcul, bit-exact.
   Go n'autovectorise pas (ADR-0011 l'a mesuré : ×0,97 en pur Go batché).
2. **`WithWorkers` n'est jamais appelé en production.** Seuls les tests l'utilisent
   (`gold_test.go`, `search_test.go`, `search_cube_test.go`, `integration_gate_test.go`).
   `EvaluatePosition` (`domaineval.go:186`) construit un `Searcher` série. Le commentaire de
   `db_gammonnet_batch.go:63-66` qui justifie le lot séquentiel par « une recherche utilise
   déjà plusieurs cœurs » est faux. ADR-0011 qualifie pourtant ce parallélisme de
   *requirement*.
3. **Le parallélisme existant plafonne à ×4 sur 16 threads** : tourniquet statique des 21
   lancers (`search.go:508-535`), pas de vol de travail alors que les doubles coûtent bien
   plus, une barrière par candidat approfondi (trois à la racine en 2-ply).
4. **80 % des 196 entrées sont nulles** (thermomètre creux : ~40 non-nuls par position,
   ~38 en union de fratrie mesuré amont). La couche 1 vaut 19 % du grand réseau et 97,5 %
   du petit ; sauter les zéros est exact en IEEE.
5. **Le lot analyse une position à la fois, avec un `Searcher` neuf de 5,5 Mo par position**
   (`db_gammonnet_batch.go:73-105`, `:177-204`).

### Ce qui n'explique pas ces chiffres (backlog, pas ce plan)

Les frais hors réseau listés par l'audit valent 0,15 s au total sur 5,5 s : allocation de
164 Ko par appel (`search.go:278`), `sort.SliceStable` réflexif trois fois par nœud
(`search.go:563`), dédup O(n²) dans `moves_gen.go:65-70`, `Valid()` sur des positions légales
par construction (`encoding.go:45`, `moves_gen.go:36`), ~240 bissections par valuation du
videau au score (`cube.go:509-526`) alors que les lookups MET sont invariants sur toute la
recherche. Ils deviendront visibles quand le réseau aura été divisé par huit ; ils sont
consignés dans `tasks/BACKLOG.md` avec ce profil comme preuve, et seront repris sur mesure.

---

## 2. Décisions prises (grilling du 2026-09-02)

| # | Décision | Alternative écartée et pourquoi |
|---|---|---|
| D1 | **Critère** : latence d'une décision 2-ply canonique sur un cœur, mesurée par `TestProbeDecisionCost`, cible **< 1 s** (5,5 s aujourd'hui) ; le parallélisme vient par-dessus. | Cibler seulement le lot : ×8 gratuit par goroutines inter-positions, mais le panneau resterait à 1,3 s au mieux. |
| D2 | **Exactitude** : le nouveau noyau est **bit-identique** au scalaire actuel — batch sur la dimension positions, une position par voie, float32, `j` croissant, multiplication et addition séparées, jamais FMA. | Tolérance 1e-6 (celle des gold) : ×1,5-2 de plus, mais résultats dépendants de l'architecture ; une base analysée sur Mac et Linux divergerait et « périmé » n'aurait plus de définition. Quantisation : changement de Configuration. |
| D3 | **Véhicule** : assembleur Go généré par **avo**, un noyau AVX2 amd64 et un noyau NEON arm64, sélection à l'exécution via `golang.org/x/sys/cpu` (déjà une dépendance), repli pur Go de même layout. **Amendée par F1** : avo ne génère que du x86, le NEON devra être écrit à la main et reste un suivi ; arm64 tourne sur le repli. | cgo : cassé pour `serve` (`CGO_ENABLED=0`), ADR-0011 a choisi le port Go pour l'éviter. Pur Go batché : ×0,97 mesuré. Go 1.26 `simd/archsimd` : amd64 seulement, expérimental ; peut remplacer avo plus tard. |
| D4 | **Lot par nœud, largeur 8** : `shallowFill` envoie ses ≤ 12 survivants au noyau, remplissage ~75 %. Regroupement des 21 lancers (design du C, 98 %) seulement si le remplissage mesuré < 70 %. | Regrouper d'emblée : découpe `rankPlays` en trois phases et entre en tension avec `rollsInParallel`. Débit crête identique à 8 ou 32 voies sur Zen 3 ; la largeur 32 n'économise que des relectures L3, qui ne sont pas le goulot. |
| D5 | **Deux régimes de parallélisme** : lot = inter-positions, `NumCPU` goroutines, un `Searcher` série réutilisé par goroutine, `yield` conservé ; panneau = intra-recherche `WithWorkers(NumCPU)` avec file de travail sur les 21 lancers par coût décroissant. | Un seul régime : le lot n'a pas besoin des barrières, le panneau n'a pas d'autre source de parallélisme. |
| D6 | **Strictement exact** : filtres à seuil d'équité, table de transposition sur les nœuds, réseau distillé, int8, mini-réseau aux nœuds internes sont **hors périmètre** et appartiennent à gammonNet (fiches T72-T74 amont). Une nouvelle Configuration se décide en amont avec la jauge de force, puis le port suit et `EngineVersion` change de nom. | Les inclure ici : les analyses stockées sous « gammonNet 2-ply k=12 » ne seraient plus comparables. |
| D7 | **Preuve continue** : sélecteur `BLUNDERDB_GAMMONNET_KERNEL=go\|avx2\|neon` (non documenté côté utilisateur), test `==` sur les bits entre assembleur et repli Go sur `reference.bin` + positions de recherche réelle, sur les trois runners CI ; le probe gagne les colonnes *remplissage* et *noyau*. | Se fier aux gold : ils tolèrent 1e-6, ce qui est précisément ce que D2 interdit. |
| D8 | **Consignation** : ADR-0024, ligne d'invariant dans `CLAUDE.md`, `analyze --jobs N` dans `CLI_USAGE.md`, pas de changement de `CONTEXT.md` (un lot et un noyau SIMD sont de l'implémentation). | — |

Hors périmètre, chacun avec sa propre fiche à venir : les trois recherches complètes que
`gammonnet_eval.go` lance par position (ADR-0017 mesure +36 % pour `preRollFacts` seul) ;
un cache partagé entre workers ; le préambule de `AnalyzeStaleGammonNet` qui décode le JSON
compressé de toutes les analyses pour trouver le moteur.

---

## 3. Fiches

Chaque fiche est un fichier, pour que le plan reste lisible et sous la limite de 500 lignes
que le dépôt impose aux documents. **Toutes sont faites.**

| Fiche | Sujet | Issue | Résultat |
|---|---|---|---|
| [F0](fiche-F0-probe.md) | Le probe devient la mesure de référence | #145 | Le remplissage des lots vaut 84,3 % : la décision D4 est tranchée, pas de regroupement des 21 lancers |
| [F1](fiche-F1-noyau.md) | Le noyau groupé, une position par voie | #133 | 505 → 17 µs par position sur des frères, bit-identique, aucune FMA dans le `.s` |
| [F2](fiche-F2-branchement.md) | La recherche alimente le noyau | #146 | Décision 2-ply série 5,5 s → 0,33 s ; remplissage réel 84,3 %, la simulation de F0 confirmée |
| [F3](fiche-F3-lot.md) | Le lot analyse en parallèle | #147 | ×4,08 sur 8 cœurs physiques ; analyses identiques quel que soit `--jobs` |
| [F4](fiche-F4-panneau.md) | Le panneau utilise tous les cœurs | #148 | ×4,43 (p75) — au niveau du plafond machine mesuré à ×3,98 |
| [F5](fiche-F5-consigner.md) | Consigner | #149 | ADR-0024 acceptée, invariants dans CLAUDE.md |

Annexe : [les prompts de deep search](prompts-deep-search.md) tels qu'ils ont été posés.
Les rapports qu'ils ont produits sont sous [`docs/recherche/`](../../docs/recherche/).

## Résultat d'ensemble

Décision 2-ply canonique, même machine, même position, du matin au soir du 2026-09-02 :

| | avant | après |
|---|---|---|
| série | 5,5 s | **0,277 s** |
| tous les cœurs | 1,3 s | **0,072 s** |
| 0-ply | 6 ms | < 1 ms |
| lot de 200 positions | 302 s | 74 s |

**Aucune équité n'a bougé d'un bit.** La parité rend toujours 5,960e-08, les gold passent
sans retouche, et `TestParallelSearchIsBitIdentical` exige l'identité binaire entre la
recherche série et la recherche parallèle.

**Deux prémisses ont été réfutées par la mesure, et c'est le meilleur de ce plan** : le
précalcul des équités de match du videau, qu'ADR-0011 annonçait, vaut 1 % et non le gain
espéré (F3 de #150) ; et le proxy « les doubles coûtent le plus cher » est faux, les doubles
sont parmi les moins chers une fois l'élagage passé (fiche F4).

## 4. Estimation, à contredire par la mesure

| Étape | Décision 2-ply canonique | Lot de 10 000 positions (8 cœurs) |
|---|---|---|
| Aujourd'hui | 5,5 s (1,3 s avec les workers de test) | ~15 h |
| F3 seule | 5,5 s | **~3 h 40 (mesuré)** |
| F1 + F2 | ~0,8-0,9 s estimé → **0,345 s mesuré** | ~2 h 30 (série, estimé) |
| F1 + F2 + F3 | **0,345 s mesuré** | à remesurer (voir ci-dessous) |
| F1 + F2 + F4 | ~0,15-0,25 s estimé → **0,055 s mesuré** (8 cœurs physiques) | — |

La ligne « F3 seule » n'est plus une estimation : le lot rend **×4,1 sur 8 cœurs
physiques**, pas ×8 (voir la fiche F3), et la machine — un mobile 15-28 W dont la
fréquence tous cœurs s'effondre, et dont le L3 ne tient pas huit `Searcher` — en est la
cause, pas le lot.

**La ligne qui cumule F1+F2 et F3 est délibérément laissée vide, et c'est le point le
plus intéressant de cette mesure.** Les deux facteurs ne se multiplient pas : le lot
plafonne à ×4,1 précisément *parce que* huit `Searcher` se disputent la bande passante
et le L3 en y faisant passer les mêmes 2,1 Mo de poids. Le noyau groupé change ce
régime — il lit les mêmes poids pour huit positions au lieu d'une — donc le facteur
d'échelle du lot doit être **remesuré** sur le code d'aujourd'hui, pas déduit. C'est la
recette de la fiche F3, à rejouer.

La ligne F4 est en revanche **meilleure que son estimation** : 55 ms p75 pour la décision
2-ply canonique sur 8 cœurs physiques, contre 150-250 ms espérés. Le facteur intra-recherche
(×4,4) bute sur exactement le même mur matériel que le lot (×4,1) et pour la même raison —
la sonde de plafond de F4 le mesure directement, sans aucune synchronisation.

Le reste (2,4 %, ~0,15 s) devient alors 15-20 % d'une décision : c'est le moment de
rouvrir le backlog des frais hors réseau, avec un profil, pas avant.

---
