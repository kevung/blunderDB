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

Suivi : #145 (F0), #133 (F1), #146 (F2), #147 (F3), #148 (F4), #149 (F5). Le chantier
frère des frais hors réseau est #150.

Ordre imposé : F0 avant tout (les chiffres de départ sont irrécupérables ensuite), F1 avant
F2, F3 et F4 indépendantes de F1 (elles gagnent seules, et se cumulent), F5 en dernier.

### F0 — Le probe devient la mesure de référence [S] — **FAIT** (#145)

- [x] `cost_probe_test.go` : colonnes *remplissage* et *noyau* ; compteurs `batchFilled` /
      `batchSlotted` sur `Searcher`, exposés par `BatchFill()`.
- [x] Ligne « 2-ply canonique, `WithWorkers(NumCPU)` ».
- [x] Au passage : `Counters()` **agrège les workers**. Sans cela un relevé parallèle
      rapportait une fraction du coût, et rapetissait à mesure qu'on ajoutait des cœurs —
      l'inverse de ce qu'un probe sert à faire.
- [x] `kernel.go` : `EvalBatchWidth`, `KernelName()`, `batchSlots()` — la couture que F1
      remplira, posée maintenant pour que les mesures d'avant et d'après soient
      comparables.
- [x] `BenchmarkEvaluateBatch` : coût **par position** sur un lot de positions *distinctes*.
      `BenchmarkEvaluate` réévalue une seule position, ce qui flatte tout noyau et
      flatterait le plus un noyau groupé.
- [x] `BenchmarkDecision2Ply`, sauté sous `testing.Short()`.
- [x] Chiffres de départ consignés en section 1.

### F1 — Le noyau batché [L] — #133 — **FAIT** (branche `feat/gammonnet-batched-kernel`)

Le cœur du plan. Tout ce qui suit est bit-identique par construction, et le test le prouve.

- [x] **API** : `Evaluator.EvaluateBatch(features *[8][196]float32, n int, probs
      *[8][NumOutputs]float32)`. L'appelant remplit les `n` premières lignes et lit les `n`
      premiers résultats ; les voies au-delà sont **complétées par duplication** de la
      position `n-1`, jamais par des zéros. L'appelant ne raisonne donc pas sur la fin du
      lot, et une voie dupliquée est vérifiable (`==` avec sa jumelle) — c'est un test que
      le remplissage par zéros n'offrait pas. `Evaluate` (scalaire) est inchangé.
- [x] **Layout** : activations feature-major `act[j*B+n]`, poids inchangés (row-major
      `[out][in]`, format `BGNN`). Chaque sortie `i` : `acc[0..B) = bias[i]`, puis pour `j`
      croissant `acc[n] += w[i][j] * act[j*B+n]`. Zéro allocation par appel (les tampons
      vivent dans l'`Evaluator`, alloués à la première utilisation groupée).
- [x] **Sparsité de la couche 1** : union des indices non nuls collectée à la
      transposition, poids compactés dans un tampon contigu par lot (l'indexation indirecte
      est plus lente, et `vgatherdps` est microcodé à ~40-52 µops sur Zen 3 — P2).
      **Mesurée, et le résultat est moins net que prévu** : la compaction déplace
      `out × k` flottants pour économiser `(196−k)/196` de la couche 1, qui vaut 19 % du
      grand réseau. Sur le lot que la recherche assemble réellement — huit coups d'un même
      lancer, union ~32 sur 196 — elle rend **≈ 6 %**. Sur huit plateaux sans rapport
      (union ~64, ce que fait `BenchmarkEvaluateBatch`) elle **coûte 9 %**. Conservée,
      mesurée sur des frères ; le commentaire de `kernel.go` porte les deux chiffres.
- [x] **Noyau AVX2** via avo : `VBROADCASTSS` du poids, `VMULPS` puis `VADDPS` séparés,
      `VMAXPS` pour ReLU, `VZEROUPPER` en sortie, sigmoïde des 5 sorties en Go.
      **Tuile = 6 sorties × 8 voies**, et non 4 : sur Zen 3/4 `VADDPS` a une latence de 3
      cycles sur deux pipes disjointes (FP2/FP3), il faut donc 6 chaînes indépendantes
      pour les saturer (P2). Mesuré : 6 rend **12 % de plus** que 4. Huit ne passe pas —
      l'allocateur d'avo manque de registres généraux et attrape **BP**, le pointeur de
      trame, ce qui fait tomber le noyau une fois sur trois à un endroit sans rapport.
      `kernel_avx2_amd64_test.go` relit le `.s` produit et refuse un `FMADD` ou un `BP`.
- [ ] ~~**Noyau NEON** via avo (arm64)~~ — **impossible : avo ne génère que du x86**
      (v0.6.0 n'expose que `avo/x86` ; confirmé par la note de recherche P1). ADR-0024
      amendée en conséquence. La suite est de l'assembleur Plan 9 arm64 **écrit à la main**
      (`simd/archsimd` est Go 1.26+, et arm64 seulement à partir de 1.27 ; ce dépôt est en
      1.25). Volontairement **non écrit ici** : rien dans cette chaîne d'outils n'exécute
      de l'arm64 — pas même `qemu` — donc un noyau NEON écrit maintenant ne pourrait pas
      passer `kernel_identity_test.go`, et livrer un chemin rapide non vérifié est
      exactement ce que le critère d'acceptation 4 de #133 interdit. **Suivi à ouvrir**,
      la couture prend une entrée de plus. arm64 tourne sur le repli Go : correct,
      simplement non accéléré.
- [x] **Repli pur Go** de même layout, même ordre de sommation, `float32(a*b)` explicite
      (barrière de fusion **garantie par la spec Go**, seule protection contre le FMADD que
      le compilateur émet d'office sur arm64 — P1). C'est la référence du test bit-à-bit.
      Attention : `unconvert`/`gopls` proposent de retirer ces conversions ; elles ne sont
      pas redondantes.
- [x] **Sélecteur** : `BLUNDERDB_GAMMONNET_KERNEL=go|avx2`, lu une fois par
      `sync.OnceValues`. Défaut = meilleur disponible (`cpu.X86.HasAVX2`). Une valeur
      indisponible est refusée **au chargement** : `Load` (donc `Embedded`) rend l'erreur,
      et toute la suite rougit avec le message qui nomme les noyaux disponibles.
- [x] **Dénormalisés** : prouvé et non supposé. `TestKernelPreservesSubnormals` fait passer
      un produit qui vaut 1e-45 (0x00000001, deux ulps au-dessus de zéro) dans chaque
      noyau et refuse un zéro. Go ne pose ni FTZ/DAZ (MXCSR 0x1F80) ni FZ (FPCR à 0) et
      `CGO_ENABLED=0` écarte le risque qu'une bibliothèque C `-ffast-math` les pollue (P1).
- [x] **Le cas `acc = -0`** a sa propre preuve, parce que c'est la seule faille de la
      sparsité : sauter une colonne nulle laisse `-0` là où l'addition de `+0` aurait rendu
      `+0`. `TestKernelNegativeZeroIsNeutralisedByReLU` montre que le ReLU ramène les deux
      à `+0` **et** que sans ReLU les deux diffèrent — donc que le raccourci est confiné à
      une couche qui en a une (le code le garantit : `skipZeros := last > 0`).
- [x] **Test bit-à-bit** (`kernel_identity_test.go`) : les 2 000 vecteurs de
      `testdata/reference.bin` **et** 2 155 positions d'une expansion 1-ply réelle de
      l'ouverture (les coups racine et leurs enfants, soit l'ensemble qu'une recherche de
      profondeur 1 évalue), grand réseau **et** réseau d'élagage, passés dans le noyau actif
      et dans le repli Go de la même machine, `==` sur chaque bit des 5 sorties. Lots
      pleins, lots partiels de 1 à 7 (avec vérification que la voie de bourrage rend
      exactement sa jumelle), et `TestDenseKernelsAgreeOnRandomLayers` compare les
      **activations intermédiaires** couche par couche sur douze formes, dont celles qui ne
      sont pas multiples de la tuile.
- [x] `parity_test.go` (5,960e-08 contre le C, inchangé), `gold_test.go`,
      `cube_gold_test.go`, `search_test.go` : **verts sans retouche**. D2 tient.
- [x] **Cible mesurée** : ≤ 60 µs par position → **atteinte avec un facteur 3 de marge**.

  | mesure (1 goroutine, `-benchtime 1s`, min de 4 relevés) | ns/position |
  |---|---|
  | `BenchmarkEvaluate` — scalaire, une position rejouée | 512 506 |
  | `BenchmarkEvaluateBatch` — scalaire, 8 positions distinctes | 505 838 |
  | `BenchmarkEvaluateBatchKernel` — **avx2**, 8 plateaux sans rapport | **20 020** |
  | `BenchmarkEvaluateBatchKernelSiblings` — **avx2**, 8 frères (ce que la recherche donne) | **16 910** |
  | `BenchmarkEvaluateBatchKernelPartial` — **avx2**, 1 voie utile sur 8 | 123 926 |
  | `BenchmarkEvaluateBatchKernel` — repli **go** groupé | 473 214 |
  | C amont, gcc -O3, largeur 32 (rappel) | 41 000 |

  Soit **×25 sur le scalaire mesuré dans les mêmes conditions**, et **×2,4 sur le C**.
  Le noyau tourne à ~1,2 cycle par paire mul+add vectorielle, soit 41 % du crête théorique
  de Zen 3 — exactement la fourchette « conservatrice » que P2 annonçait, et deux fois
  mieux que le C, qui souffre vraisemblablement du piège gcc « accumulateur rechargé
  depuis la mémoire » (P2, question 4).

  **Machine NON oisive** pendant ce relevé (d'autres agents travaillaient sur le même
  dépôt ; `BenchmarkEvaluate` sort à 512 µs contre 404 µs au repos en section 1). Les
  *ratios* sont pris dans le même processus et tiennent ; les absolus sont pessimistes
  d'environ 25 %. À refaire au repos si le chiffre doit servir de référence.

  Le repli pur Go groupé rend **×1,1 sur le scalaire** au lieu du ×0,97 mesuré par
  ADR-0011 : la différence est la sparsité de la couche 1, pas la vectorisation, que Go ne
  fait toujours pas.

- [x] `go vet`, `go vet` en cross (linux/arm64, darwin/arm64, windows/amd64, `-tags purego`),
      `golangci-lint` v2.11.4 : 0 issue. Le `.s` porte l'en-tête « Code generated by command
      … DO NOT EDIT », le générateur vit dans `_asm/` (module à part, donc avo n'entre pas
      dans les dépendances de blunderDB) et `go generate ./pkg/blunderdb/engine/gammonnet/`
      le reproduit à l'octet près.

**Reste ouvert après F1** : le noyau NEON (ci-dessus) ; le branchement dans la recherche,
qui est F2 (#146) — `EvaluateBatch` n'est encore appelé par personne d'autre que ses tests,
donc aucun chiffre de bout en bout ne bouge tant que F2 n'est pas faite.

### F2 — La recherche alimente le noyau [M] — #146 — **FAIT** (branche `feat/gammonnet-search-batches`)

- [x] `shallowFill` trie puis groupe : positions terminales, hits de cache et encodages
      refusés sortent du lot (traitement inchangé), les survivants sont encodés dans le
      brouillon du `Searcher` et partent par tranches de `EvalBatchWidth`. La convention du
      lot partiel reste celle du noyau — au-delà de `n`, les voies dupliquent la position
      `n-1` — donc l'appelant ne raisonne jamais sur la queue. `flushBatch` redistribue et
      tient les mêmes compteurs : `evals` + `cache.store` pour le grand réseau, `pruneEvals`
      pour l'élagage. Le petit réseau d'élagage passe par le même chemin et **son contrat ne
      bouge pas** : ni lecture ni écriture du cache (`cache.go`).
- [x] Le tampon de lot (`batchFeat`, `batchProbs`, `batchOf` : 6,3 Ko) vit dans le
      `Searcher`. **Zéro allocation ajoutée par nœud**, vérifié : `-benchmem` passe de
      546 056 o / 7 430 all. à 1 035 400 o / 7 440 all. à `-benchtime 1x`, mais à
      668 392 o / 7 432 all. à `-benchtime 4x` — le surcoût s'amortit exactement par
      quatre, c'est donc le brouillon *paresseux* des deux `Evaluator` (F1, ~465 Ko),
      une fois par `Searcher`, et rien sur le chemin chaud.
- [x] ~~Le probe donne le remplissage~~ — **fait, et D4 est tranchée** : 84,3 % à 2-ply
      canonique (section 1), au-dessus du seuil de 70 %. Le regroupement des 21 lancers
      n'aura pas lieu.
- [x] `TestParallelSearchIsBitIdentical`, `gold_test.go` (`BLUNDERDB_GOLD=1` : 85 + 123
      décisions, max|Δ| 7,153e-07 et 3,902e-07), `cube_gold_test.go`, `parity_test.go`,
      `kernel_identity_test.go` : **verts sans retouche** ; `go vet ./...`, `golangci-lint`
      v2.11.4 (0 issue), `go test ./...`, `-race` sur le paquet : verts. L'`integration_gate`
      échoue **par les deux mêmes décisions qu'avant** (0,0552 et 0,0738, à la quatrième
      décimale, question de profondeur/MET documentée dans le test) et tourne en **105 s au
      lieu de 765 s** — ×7,3 sur une charge réelle.
- [x] **Cible mesurée** : décision 2-ply canonique série **0,345 s** contre 5,85 s avant —
      D1 (< 1 s) **atteinte avec un facteur 3 de marge**.

  `BenchmarkDecision2Ply -benchtime 1x -count 5`, cinq décisions à froid : **5,33 / 5,58 /
  5,85 / 5,89 / 5,98 s** avant, **0,341 / 0,342 / 0,347 / 0,347 / 0,349 s** après. Soit **×17**, contre ×25 pour le noyau en isolation : l'écart est ce que la décision
  passe hors réseau, qui devient visible exactement comme ADR-0024 l'annonçait (2,4 %
  avant, ~15 % maintenant — c'est le chantier #150). **Machine NON oisive** (charge 4 à 6,
  d'autres agents sur le dépôt) ; `BenchmarkEvaluateBatchKernelSiblings` sort entre 13,8
  et 23,2 µs/position dans les deux séries, donc les deux colonnes ont vu la même machine.

  Probe correspondant (`BLUNDERDB_PROBE=1`, noyau avx2, mêmes conditions) :

  | configuration | durée | évals | élagage | cache | remplissage |
  |---|---|---|---|---|---|
  | 0-ply k=12 | 1 ms | 12 | 16 | 0 | 87,5 % |
  | 1-ply k=12 sans filtre | 70 ms | 2 676 | 4 934 | 155 | 81,7 % |
  | 1-ply k=12 filtre 3 | 16 ms | 660 | 1 140 | 40 | 81,2 % |
  | 2-ply k=12 (0,1,3) | **360 ms** | 13 438 | 36 395 | 1 704 | **84,3 %** |
  | 2-ply k=12 (0,2,8) | 1,659 s | 61 836 | 152 005 | 9 472 | 83,0 % |
  | 2-ply k=12 (0,1,3), `WithWorkers(16)` | 117 ms | 14 038 | 36 395 | 1 104 | 84,3 % |

- [x] **Le remplissage réel vaut ce que #145 avait simulé** : 84,3 % à 2-ply canonique, à
      la décimale. `batchFilled`/`batchSlotted` mesurent désormais le lot au lieu de le
      simuler et disent la même chose — ce qui valide rétroactivement D4.
- [x] **Une seule dérive de compteur, sans effet** : 13 438 évaluations au lieu de 13 439,
      1 704 hits au lieu de 1 703. La version position par position rangeait le candidat
      *i* avant de chercher le *i+1* ; le lot cherche les huit avant de ranger les huit, et
      évince donc une entrée de moins dans cette table à adressage direct. Un hit rend les
      bits qu'un calcul aurait rendus (`cache.go`) : seuls les compteurs bougent. Et deux
      candidats d'un même appel ne sont jamais la même position (`moves_gen` déduplique par
      plateau résultant), donc rien ne s'évalue deux fois dans un lot.

### F3 — Le lot analyse en parallèle [M] — #147

- [ ] `db_gammonnet_batch.go` : `AnalyzeMissingWithGammonNet` et `AnalyzeStaleGammonNet`
      distribuent les ids à `jobs` goroutines ; chaque goroutine possède **un `Searcher`
      série réutilisé** (cache conservé d'une position à l'autre : hit et miss donnent le
      même bit, `cache.go:8-14`, donc licite et gratuit). Supprimer le commentaire faux des
      lignes 63-66.
- [ ] `yield` : chaque goroutine l'appelle avant de prendre une position ; la sémantique « le
      lot cède en une position » devient « en au plus `jobs` positions », à documenter dans
      le commentaire.
- [ ] Écriture en base : une seule goroutine écrit (canal de résultats), `Database.mu` ne
      voit pas de contention ; l'ordre d'écriture ne change pas le résultat.
- [ ] Annulation par `ctx` conservée, vérifiée avant chaque position.
- [ ] `gammonnet.EvaluatePosition` gagne une variante qui prend un `*Searcher` fourni
      (ou `NewSearcher` devient réutilisable via `Reset`) ; la version actuelle reste pour
      les appelants unitaires.
- [ ] CLI `analyze --jobs N` (défaut `runtime.NumCPU()`), affichage de progression
      inchangé ; `CLI_USAGE.md` et l'aide de `cli_analyze.go`. Serveur (`handlers_gammonnet.go`)
      et GUI : `NumCPU`, rien d'exposé.
- [ ] Test : un lot de N positions en parallèle écrit les mêmes analyses (au bit) que le lot
      série, quel que soit `jobs`.
- [ ] **Cible mesurée** : ×(cœurs physiques) sur 200 positions, à noter dans ce fichier.

### F4 — Le panneau en direct utilise tous les cœurs [M] — #148

- [ ] `domaineval.go:186` (`EvaluatePosition`) et `gammonnet_eval.go` : `WithWorkers(NumCPU)`
      sur la recherche 2-ply de fond. Le 0-ply synchrone reste série.
- [ ] `rollsInParallel` (`search.go:508-535`) : remplacer le tourniquet `r += nw` par une
      file (compteur atomique) sur les 21 lancers **ordonnés par coût décroissant** (les
      doubles d'abord, ou un ordre mesuré par F0). La somme pondérée reste série en `r`
      croissant (`search.go:473-477`) : le parallélisme change qui calcule chaque terme,
      jamais l'ordre d'addition.
- [ ] Réduire les barrières : à la racine, les `Filter[depth]` candidats approfondis sont
      indépendants ; les approfondir en parallèle (chacun déclenchant sa propre file de
      lancers) plutôt qu'un après l'autre. Chaque worker garde son `Searcher`/cache.
- [ ] `TestParallelSearchIsBitIdentical` vert pour `WithWorkers` ∈ {1, 2, NumCPU, 64}.
- [ ] **Cible mesurée** : **×5,5 sur 8 cœurs physiques** (`GOMAXPROCS=8`), p75/p95 et non
      des moyennes. Révisée à la baisse par P3 : le plafond structurel est ×5 à ×6,5, le
      SMT n'apporte presque rien sur du calcul vectoriel, et le ×4 « sur 16 threads »
      d'aujourd'hui est donc déjà proche de ce que le déséquilibre statique permet.

### F5 — Consigner [S] — #149

- [ ] ADR-0024 : passer de *proposed* à *accepted*, y inscrire les chiffres finaux de F0.
- [ ] `CLAUDE.md`, section *Invariants* : « Le noyau réseau n'utilise jamais FMA ni
      réassociation ; tout nouveau chemin de calcul passe `kernel_identity_test.go` contre le
      repli pur Go ; `WithWorkers` et le lot inter-positions sont des exigences de
      production, pas des outils de test. »
- [ ] `docs/adr/README.md` : ligne 0024, relation « exécute le suivi borné de 0011 ».
- [ ] `tasks/BACKLOG.md` : les frais hors réseau, avec le nouveau profil (post-F2) comme
      chiffre d'entrée.
- [ ] Manuel utilisateur : rien (aucune durée n'y est citée). `CLI_USAGE.md` : `--jobs`.

---

## 4. Estimation, à contredire par la mesure

| Étape | Décision 2-ply canonique | Lot de 10 000 positions (8 cœurs) |
|---|---|---|
| Aujourd'hui | 5,5 s (1,3 s avec les workers de test) | ~15 h |
| F3 seule | 5,5 s | ~2 h |
| F1 + F2 | ~0,8-0,9 s estimé → **0,345 s mesuré** | ~2 h 30 (série) |
| F1 + F2 + F3 | ~0,8-0,9 s | ~20 min |
| F1 + F2 + F4 | ~0,15-0,25 s | — |

Le reste (2,4 %, ~0,15 s) devient alors 15-20 % d'une décision : c'est le moment de
rouvrir le backlog des frais hors réseau, avec un profil, pas avant.

---

## 5. Annexe — prompts de deep search pour Claude web

> **Les quatre ont été exécutés le 2026-09-02 et leurs rapports sont versés sous
> `docs/recherche/` (index et synthèse dans son README).** Ce qu'ils ont changé :
> P1 confirme qu'avo ne génère pas d'arm64 et que le déterminisme repose sur la spec Go,
> pas sur la chance ; P2 recommande une tuile 6-8 sorties × 8 positions et une couche 1 en
> input-major creux, et mesure que l'absence de FMA est **gratuite sur Zen 3/4** ; P3
> abaisse la cible de F4 de ×6 à **×5,5 sur 8 cœurs physiques** et donne la règle de
> décision du cache partagé ; P4 écarte les tables de transposition sur les nœuds, qui
> entrent en conflit avec l'inférence par lots. Les questions restent ci-dessous telles
> qu'elles ont été posées.

Chaque prompt est autonome. Les réponses alimentent F1 (P1, P2) et F4 (P3) ; P4 ne sert
qu'à gammonNet amont (D6) et peut attendre.

### P1 — SIMD en Go en 2026, et le bit-à-bit entre AVX2, NEON et pur Go

> Je maintiens un port Go d'un réseau de neurones MLP (196→512→512→256→128→5, float32, ReLU)
> qui doit rester **bit-identique** à une implémentation C de référence : accumulation
> float32 dans l'ordre croissant des indices, multiplication et addition séparées, **jamais
> de FMA**, jamais de réassociation. Le forward pass actuel est une boucle scalaire Go à
> 404 µs par évaluation ; je veux un noyau vectorisé sur la dimension *batch* (une position
> par voie SIMD, 8 voies AVX2 / 4 voies NEON), qui préserve donc l'ordre de sommation de
> chaque voie. Cibles : linux/amd64, windows/amd64, darwin/universal (amd64 + arm64), et un
> conteneur `CGO_ENABLED=0` multi-arch — donc pas de cgo. Toolchain Go 1.25.
>
> Fais une recherche approfondie et sourcée (dépôts, docs officielles, issues Go, billets
> techniques récents) sur :
> 1. L'état en 2026 des façons d'écrire du SIMD en Go sans cgo : **avo** (maturité, support
>    arm64/NEON, exemples de noyaux float32 batchés, pièges ABI0/ABIInternal, `NOSPLIT`,
>    alignement des arguments, `VZEROUPPER`), l'assembleur Plan 9 à la main, et le paquet
>    expérimental **`simd/archsimd`** de Go 1.26 (`GOEXPERIMENT=simd`) : portée réelle
>    (amd64 seulement ?), stabilité, garanties sur l'absence de contraction FMA.
> 2. Le **déterminisme bit-à-bit** entre ces chemins : Go positionne-t-il FTZ/DAZ dans
>    MXCSR ou FZ dans FPCR (AArch64) ? Le runtime ou le ramasse-miettes touchent-ils ces
>    registres ? Y a-t-il des cas connus où `VMULPS`+`VADDPS` et `FMUL`+`FADD` diffèrent
>    d'IEEE 754 (dénormalisés, arrondi, NaN) ? Et comment garantir qu'un repli pur Go
>    `float32(a*b) + c` ne soit jamais contracté par le compilateur sur arm64 (où Go fuse).
> 3. Des exemples de projets Go qui font exactement cela — inférence MLP ou GEMM float32
>    en assembleur avo avec repli Go et test d'identité bit-à-bit (par exemple dans
>    l'écosystème des moteurs d'échecs NNUE en Go, des bibliothèques de similarité
>    vectorielle, gorgonia/gonum, etc.) — et comment ils organisent le `go:generate`, la
>    sélection à l'exécution via `golang.org/x/sys/cpu`, et la CI multi-arch.
>
> Livrable : une synthèse avec recommandations concrètes (outil, structure de fichiers,
> tests), les pièges connus avec références, et un squelette de noyau AVX2 avo pour une
> couche dense `out[i][n] = bias[i] + Σ_j w[i][j]·act[j][n]` sur 8 voies sans FMA.

### P2 — Conception micro-architecturale d'un noyau MLP batché sans FMA

> Je conçois un noyau float32 pour un MLP 196→512→512→256→128→5 (527 k MAC, 2,1 Mo de
> poids row-major) évalué par lots de positions, **une position par voie SIMD**, avec la
> contrainte de ne pas utiliser FMA (multiplication puis addition séparées, ordre de
> sommation fixe par voie). Cibles : AMD Zen 3/Zen 4 (AVX2, 256 bits), Intel récents, Apple
> M1-M4 (NEON 128 bits). L'entrée est très creuse (~40 non-nuls sur 196, ~38 en union sur un
> lot de 8), les couches internes sont denses.
>
> Recherche approfondie et sourcée (manuels d'optimisation AMD/Intel/Apple, uops.info,
> Agner Fog, billets sur les GEMM à petit batch, code de gnubg, Stockfish NNUE pour
> comparaison) sur :
> 1. Le **débit théorique et pratique** de `vmulps`+`vaddps` sans FMA sur Zen 3/4 et sur
>    NEON : ports, latences, combien de MAC par cycle on peut espérer avec 1, 2 ou 4
>    registres accumulateurs par voie, et le coût des broadcasts de poids
>    (`vbroadcastss` depuis la mémoire vs registre).
> 2. La **tuile optimale** : combien de sorties `i` traiter simultanément pour réutiliser la
>    colonne d'activations chargée (par exemple 4 sorties × 8 voies = 4 accumulateurs), et
>    la largeur de lot (8, 16, 32) qui équilibre pression sur les registres, relectures des
>    poids (2,1 Mo ne tiennent pas en L2 mais en L3) et remplissage réel du lot.
> 3. L'**exploitation de la sparsité de la première couche** : union des indices non nuls
>    du lot, compaction des poids dans un tampon contigu vs indexation indirecte (un
>    projet C a mesuré l'indirection *plus lente* que la boucle dense), et si cela vaut la
>    peine pour un petit réseau 196→32→5 où la couche 1 pèse 97 %.
> 4. Ce que **gcc -O3 génère réellement** pour une boucle `acc[n] += w * col[n]` à largeur
>    fixe 32 (`-fopt-info-vec`), puisque c'est le point de comparaison : 41 µs par position.
>
> Livrable : un plan de noyau (layout mémoire, ordre des boucles, tuile, déroulage) avec les
> chiffres attendus par couche et par cible, et une liste des micro-benchmarks à écrire pour
> valider chaque hypothèse avant de figer le design.

### P3 — Ordonnancer un expectiminimax sur 21 lancers de dés, en Go, de façon déterministe

> Un moteur de backgammon en Go fait une recherche expectiminimax : à chaque nœud, 21
> lancers de dés distincts (pondérés 1/36 ou 2/36), pour chaque lancer la génération des
> coups légaux, un élagage par petit réseau, une évaluation par grand réseau des 12
> meilleurs, puis l'approfondissement de 1 à 3 candidats. Le coût par lancer est très
> inégal (les doubles génèrent beaucoup plus de coups). Aujourd'hui les 21 lancers de la
> racine sont distribués à N goroutines en tourniquet statique, avec une barrière à chaque
> candidat approfondi : ×4 mesuré sur 16 threads. Contrainte absolue : le résultat doit être
> **bit-identique** à la version série — la somme pondérée des 21 termes est faite après, en
> série, dans l'ordre croissant ; le parallélisme ne doit changer que *qui* calcule chaque
> terme. Chaque worker a son propre cache d'évaluation (un hit et un miss donnent le même
> bit).
>
> Recherche approfondie et sourcée sur :
> 1. Comment **gnubg** parallélise ses évaluations (son pool de threads, la granularité :
>    par coup candidat, par lancer, par nœud ?) et comment il garde ses résultats
>    déterministes ; idem pour XG si documenté, et pour les moteurs d'échecs qui parallélisent
>    des recherches à nœuds de hasard.
> 2. Les schémas de **vol de travail / files de tâches en Go** adaptés à un arbre peu
>    profond (2-4 plies) et large (21 × ~12 × 21 nœuds) : compteur atomique sur un tableau
>    de tâches ordonnées par coût décroissant, `errgroup`, pools de workers persistants,
>    parallélisation des candidats approfondis en plus des lancers ; le coût d'une barrière
>    goroutine vs le gain ; et comment mesurer le facteur d'accélération honnêtement (cœurs
>    physiques vs SMT).
> 3. L'intérêt et le coût d'un **cache d'évaluation partagé** entre workers (table
>    direct-mapped sans verrou, écriture atomique de 64 octets, tolérance aux courses
>    bénignes) par rapport à un cache par worker, quand le résultat d'un hit est
>    garanti identique à celui d'un miss.
>
> Livrable : une recommandation d'architecture d'ordonnancement pour ce cas, avec les
> pièges de déterminisme, et un ordre de grandeur du facteur d'accélération atteignable sur
> 8 cœurs physiques.

### P4 — (optionnel, pour gammonNet amont) accélérations qui changent la Configuration

> Pour un moteur de backgammon à réseau MLP (527 k MAC par évaluation, recherche 2-ply avec
> élagage par un petit réseau distillé), je veux connaître l'état de l'art des gains
> algorithmiques qui **changent** ce que le moteur joue, pour les évaluer séparément avec
> une jauge de force : filtres de coups à seuil d'équité à la gnubg (« les 8 meilleurs à
> moins de 0,16 »), tables de transposition sur les nœuds internes d'un expectiminimax,
> distillation vers un réseau de 60-100 k MAC, quantisation int8 déterministe (QAT
> per-channel, `vpmaddubsw`/`vpdpbusd`, `i32x4.dot_i16x8_s` en WASM), réseaux d'élagage aux
> nœuds internes, Star1/Star2 de Ballard. Pour chacun : sources primaires (code de gnubg,
> publications, forums bkgm/rec.games.backgammon, Stockfish NNUE pour l'int8), gain de
> vitesse rapporté, perte de force mesurée, et méthode de mesure. Livrable : un tableau
> comparatif et un ordre de priorité argumenté.
