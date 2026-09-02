<!-- Fiche du plan tasks/gammonnet-perf/README.md. Le contexte, les décisions
et le tableau d'ensemble vivent dans le README ; cette fiche ne porte que son
propre travail et ses propres chiffres. -->

# F1 — Le noyau batché [L] — #133 — **FAIT** (branche `feat/gammonnet-batched-kernel`)

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
