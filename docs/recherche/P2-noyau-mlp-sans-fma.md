# Conception d'un noyau SIMD float32 sans FMA pour l'inférence d'un MLP dense (projet gammonNet)

## TL;DR

- **Le choix « sans FMA / ordre de sommation fixe » ne coûte RIEN en débit crête sur AMD Zen 3/Zen 4** : `VMULPS` s'exécute sur les pipes FP0/FP1 (débit réciproque 0,50) et `VADDPS` sur des pipes **disjointes** FP2/FP3 (0,50), donc 2 mul + 2 add par cycle = **2 MAC vectoriels/cycle**, identique à 2 FMA/cycle [MESURE, uops.info + AMD SOG]. En revanche il coûte ~0,75× sur Intel Golden/Raptor Cove (mul et add se disputent le port 1/5) et **~0,5× sur Apple** (NEON : `FMLA` fait 4 MAC/cycle, mul+add séparés seulement 2).
- **Design retenu** : couches denses 2–5 en layout **« position par voie »** (broadcast du poids, accumulation verticale bit-exact, tuile **I=6–8 sorties × B=8 positions**, ≥6 accumulateurs indépendants sur Zen / ≥12 sur Apple) ; couche 1 en layout **input-major sparse** (itération sur ~38 entrées actives, **jamais de `vgatherdps`**, microcodé ~40–52 µops sur Zen 3). Compiler impérativement avec **`-ffp-contract=off`** (défaut GNU C = `fast` = fusion FMA silencieuse).
- **Plancher de débit ~7,3 µs/position sur Zen à 4,5 GHz**, cible réaliste ~10–12 µs, conservatrice ~18 µs — soit **2,3× à 5,6× sous les 41 µs actuels**. L'estimation utilisateur de « 0,9 µs/position » est erronée d'un facteur ~8 (confusion lot/position ; voir §5).

---

## Key Findings

1. **AMD est la cible où le bit-exact est le moins pénalisant** : débit MAC crête inchangé vs FMA, seule la pression µop augmente (4 µops FP/cycle au lieu de 2).
2. **NEON/Apple paie le prix fort du no-FMA (~2×)** mais offre un avantage structurel : le `fmul`/`fmla` par élément indexé (`v.s[lane]`) supprime tout broadcast explicite.
3. **La sparsité de la couche 1 est décisive pour le petit réseau (196→32→5, couche 1 = 97 %)** mais marginale pour le gros réseau (couche 1 = 19 % des MAC ; gain global ≤ ~1,1 µs/position).
4. **Le piège gcc « accumulateur rechargé depuis la mémoire »** (absence de `__restrict`/de déroulage) explique vraisemblablement une part majeure des 41 µs : store-to-load forwarding (~4–5 cycles) sur le chemin critique → perte de 2–4×.
5. **B=8 suffit** : dès B=8 le noyau est nettement compute-bound sur Zen (besoin ~8 octets de poids/cycle contre 32 disponibles depuis le L3).

---

## Details

### Question 1 — Débit théorique et pratique de `vmulps`+`vaddps` sans FMA

**Tableau des unités FP par cible :**

| Cible | Pipes FP | VMULPS lat/tp/ports | VADDPS lat/tp/ports | VFMADD lat/tp/ports | MAC vec/cyc (split) | MAC vec/cyc (FMA) |
|---|---|---|---|---|---|---|
| **Zen 3** | 4 (FP0-3, 256b) | 3 / 0,50 / FP01 [MESURE] | 3 / 0,50 / FP23 [MESURE] | 4 / 0,50 / FP01 [MESURE] | **2** | 2 |
| **Zen 4** | 4 (FP0-3, 256b) | 3 / 0,50 / FP01 [MESURE] | 3 / 0,50 / FP23 [MESURE] | 4 / 0,50 / FP01 [MESURE] | **2** | 2 |
| **Golden/Raptor Cove** | 3 ports (0,1,5) | 0,50 / p01 [MESURE] | fast-adder 3c, bypass 2c / **p1,p5** [MESURE/DÉCLARÉ, Intel manuel 355308 : « Fast Adder/1,5/3 … performs FP ADD/SUB in 3 cycles … two cycles back-to-back »] | p01 [MESURE] | **~1,5** [EXTRAPOLÉ] | 2 |
| **Gracemont (E)** | 2 pipes 128b (FADD+FMUL par port, ports 20-22) | ~ [NON TROUVÉ, uops] | ~ | ~ | ~1 (128b) [EXTRAPOLÉ] | 2 (128b) |
| **Apple Firestorm (M1)** | 4 pipes NEON 128b (u11-14) | 4 / 0,25 [MESURE, Dougall Johnson] | 3 / 0,25 [MESURE] | FMLA 4 / 0,25 [MESURE] | **2** (128b) | 4 (128b) |
| **Apple M2-M4** | 4 pipes NEON 128b | ~4 / 0,25 [HYPOTHÈSE, hérité Firestorm] | ~3 / 0,25 | FMLA 4 / 0,25 | 2 (128b) | 4 (128b) |

**Résultat clé confirmé rigoureusement** (uops.info : `VMULPS_YMM_YMM_YMM` = `1*FP01`, tp 0,50 ; `VADDPS_YMM_YMM_YMM` = `1*FP23`, tp 0,50 ; AMD SOG 56665/57647 concordants) : sur Zen 3 et Zen 4, mul et add utilisent des paires de pipes **disjointes**. Un schéma mul-puis-add séparé soutient donc **2 vector-MAC/cycle = les mêmes MAC/cycle que 2 `VFMADD213PS`/cycle** (également `1*FP01`, tp 0,50, latence 4). Différences du chemin split : (a) 4 µops FP/cycle (les 4 pipes saturées) contre 2 en FMA, moins de marge pour d'autres opérations FP ; (b) latence d'une chaîne dépendante mul→add = 3+3=6 cycles vs 4 en FMA ; (c) double arrondi (précisément ce que le bit-exact impose).

**Accumulateurs indépendants requis pour saturer les ports** (latence de l'add d'accumulation × nombre de pipes d'add) :
- **Zen 3/4** : 3 × 2 (FP2/3) = **6**.
- **Golden/Raptor Cove** : ~3 × 2 (p1/p5) ≈ **6** (mais contention mul/add sur p1).
- **Apple Firestorm** : 3 × 4 = **12**.
- **Gracemont** : ~6 (128b).

**Coût des broadcasts :**
- **Intel (SKL→ICL→Golden Cove)** : `vbroadcastss ymm,[mem]` = pure µop de chargement (p23, tp 0,5, **aucune** µop FP/ALU) [MESURE, uops.info]. Quasi gratuit.
- **AMD Zen 3/4** : le broadcast mémoire **consomme une µop de pipe FP** (forme ZMM étiquetée `1*FP12`, tp mesuré ~1,0) — ce n'est **pas** load-only [MESURE partielle : confirmé sur la forme ZMM + AMD SOG ; la page YMM exacte n'a pas pu être ouverte]. Désavantage AMD : chaque broadcast vole un slot FP.
- **NEON (avantage structurel)** : `fmul v.4s, v.4s, v.s[lane]` évite tout broadcast ; un chargement de 4 poids `w[j][i..i+3]` sert 4 sorties via indexation de voie.
- **WASM SIMD128** : `v128.load32_splat` / `f32x4.splat`, `f32x4.mul`+`f32x4.add`, pas de FMA garanti (relaxed-SIMD exclu pour le bit-exact) — transposable sans difficulté.

### Question 2 — Tuile optimale et largeur de lot

En « position par voie », la colonne d'activations de l'entrée j occupe B/8 registres YMM (B/4 en NEON) ; pour I sorties : **I×(B/8) accus + (B/8) colonne + 1 broadcast**.

**AVX2 (16 YMM) :**

| Config | Accus | Colonne | Bcast | Total | Accus indép. | Verdict |
|---|---|---|---|---|---|---|
| I=4, B=8 | 4 | 1 | 1 | 6 | 4 | légèrement borné (< 6) |
| **I=6, B=8** | 6 | 1 | 1 | 8 | 6 | **optimal (sature FADD)** |
| I=8, B=8 | 8 | 1 | 1 | 10 | 8 | excellent, marge |
| I=4, B=16 | 8 | 2 | 1 | 11 | 8 | amortit mieux les poids |
| I=2, B=32 | 8 | 4 | 1 | 13 | 8 | I trop faible (relectures colonne) |

**NEON (32 V, 4 voies/reg, B=8 → 2 V/colonne)** : **I=8, B=8** = 16 accus + 2 colonne = 18 V, ≥12 requis → sature les 4 pipes. I=8,B=16 (32+4=36) déborde ; revenir à **I=6,B=16** (24+4=28 V).

**Trafic mémoire.** Les 2,1 Mo de poids ne tiennent ni dans le L2 Zen 3 (512 Ko) ni Zen 4 (1 Mo) [DÉCLARÉ, AMD SOG 56665 / Chips and Cheese : « doubling L2 size to 1 MB » sur Zen 4], ni tout à fait dans le L2 client Golden/Raptor Cove (1,25–2 Mo selon segment) [DÉCLARÉ, WikiChip Fuse], mais tiennent en L3 (32 Mo/CCD Zen ; 24–48 Mo LLC Apple). Octets de poids/MAC = 4/B. À 16 MAC scalaires/cycle sur Zen : besoin 64/B octets/cycle. Bande passante L1↔L3 = **32 octets/cycle** [MESURE, AnandTech/Cutress Zen 3 : « All cache read and write operations are done at 32 bytes per cycle »]. Condition compute-bound : 64/B ≤ 32 → **B ≥ 2**. **Dès B=8, largement compute-bound** ; B plus grand ne sert qu'au parallélisme et à l'amortissement des surcoûts fixes.

**Lot partiel (2-ply).** Le nombre de positions par nœud varie (21 jets × mouvements légaux, souvent 10–30, parfois 1–3 en fin de partie) [HYPOTHÈSE, domaine backgammon]. Stratégies, préférées dans l'ordre : (1) **file d'attente inter-nœuds** vidangée par blocs pleins de B ; (2) dispatch B∈{8,16,32} + reliquat masqué ; (3) padding B=8 avec entrées nulles + masque de sortie (peu coûteux car B=8 est petit et compute-bound). **Recommandation : B=8 + file d'attente.**

### Question 3 — Sparsité de la couche 1

Union ~38/196 → gain théorique **≈5,2×** sur la couche 1. Mais couche 1 = 100 352 MAC = **19 %** du gros réseau (526 976 MAC) → gain global ≤ ~1,1 µs/position ; pour le **petit réseau 196→32→5**, couche 1 = **97 %** → quasi 5× sur le total. **Priorité : petit réseau.**

- **(a) Layout input-major** : W1 transposé `[in][out]`, 512 poids contigus par entrée — exactement le feature transformer NNUE de Stockfish [DÉCLARÉ, nnue-pytorch docs : « Feature Transformer: Row-major layout [NUM_FEATURES][L1] … Facilitates sparse accumulation (sum active feature rows) »] et l'approche de gnubg (itération sur entrées actives). Accumulation dans un tampon 512×B.
- **(b) Gather proscrit** : `vgatherdps ymm` microcodé sur Zen 3 (~40–52 µops ; latences op1→1=8, op3→1=19) [MESURE, uops.info ZEN3] ; la forme de référence `vgatherqpd` (4 éléments) mesure tp=4,00 cycles / 23–24 µops sur Zen 3/4, donc `vgatherdps` (8 éléments) ~8–12 cycles [EXTRAPOLÉ]. Casse aussi prédiction et prefetch.
- **(c) Compaction inutile** : copier les ~38 lignes (~78 Ko) coûte autant que les lire. Bon schéma : **itérer sur les indices actifs, lire la ligne** (séquentiel intra-ligne, suivi par le prefetcher HW). Cohérent avec la mesure utilisateur « indirection plus lente que la boucle dense » [DÉCLARÉ, contexte].
- **(d) « Position par voie » exige des entrées transposées** (layout position-mineur `A[in][B]`) : la valeur de l'entrée j varie par voie, multipliée par le broadcast de `w[j][i]`. Coût du transpose 8×8 AVX2 ≈ 24 shuffles [FOLKLORE — chiffre classique], amorti sur 196 entrées.

**Hybride recommandé** : couche 1 en « sortie par voie » sparse (une position à la fois, comme gnubg — évite le transpose des 196 entrées, exploite la sparsité par position) → transpose du résultat 512×B → couches 2–5 en « position par voie ». Le transpose intermédiaire 512×B (une fois/lot) est petit devant le calcul dense des couches 2–5. **À valider par benchmark** (le transpose peut annuler le gain si le gros réseau domine).

### Question 4 — Ce que gcc -O3 génère réellement

**Flags critiques :** **`-ffp-contract=off` OBLIGATOIRE** — le défaut GNU C/C++ est `=fast` (fusion FMA cross-statement, active sous -O3 avec `-march=native`) [DÉCLARÉ, gcc + Krister Walfridsson : « -ffp-contract=fast is enabled for C++ and GNU C … not for standard C »] ; `-std=c17`/`c99` implique aussi `=off`. Sans cela, résultats divergents x86 (FMA) vs ARM/WASM → bit-exact cassé. Ajouter `-fno-fast-math`, `__restrict`, `#pragma GCC unroll N`, `__attribute__((aligned(32)))`, éventuellement `-fno-tree-slp-vectorize`.

**Boucle `for(n=0;n<32;n++) acc[n]+=w*col[n];`** avec `-O3 -mavx2 -ffp-contract=off` : gcc 13/14 vectorise en **4 `vmulps` + 4 `vaddps`** (32/8). Piège : sans `__restrict` ni déroulage externe, `acc[]` ne tient pas en registres → **rechargé/restocké à chaque itération**, ajoutant la latence store-to-load forwarding (~4–5 cycles) sur le chemin critique → **perte 2–4×** [FOLKLORE bien établi ; symptôme Load-Hit-Store]. Explication probable d'une part des 41 µs.

**Diagnostic** : `-fopt-info-vec-all` / `-fopt-info-vec-missed` ; vérifier l'assembleur (`objdump -d`, godbolt) — chercher les `vmovups` de `acc` en tête/queue de boucle interne (signe du rechargement). **Remède** : accus en variables locales, `__restrict`, déroulage manuel de la boucle externe.

### Plan de noyau détaillé

**Layout mémoire.** Poids couches 2–5 : **row-major** `W[out][in]`, dim `in` paddée à ×8 (AVX2)/×4 (NEON), alignée 32 o. Poids couche 1 : **input-major** `W1[in][out]`. Activations : **position-mineur** `A[in][B]`, alignées 32 o, B multiple de 8. Accumulateurs : tampon 512×B en registres autant que possible.

**Ordre des boucles (couches denses) :**
```
pour chaque bloc de I sorties (i0..i0+I):
    zéro I×(B/8) accumulateurs YMM
    pour chaque entrée j (0..in-1):            # dimension de réduction
        col_j = A[j][0..B]                      # (B/8) YMM
        pour i dans le bloc I:
            wtmp = broadcast w[j][i]
            tmp     = vmulps(col_j, wtmp)       # FP0/1
            acc[i]  = vaddps(acc[i], tmp)       # FP2/3, ordre fixe → bit-exact
    activation ; stocker acc
```
La réduction sur j est séquentielle croissante dans chaque accumulateur → **bit-exact garanti** (aucune réassociation inter-voies).

**Déroulage** : boucle i complètement déroulée (I chaînes d'add indépendantes) ; boucle j par 2–4 (masquer la latence de chargement de col_j). Zen : I=6–8, B=8. NEON : I=8, B=8 avec `fmul` indexé.

**Activation [INCERTITUDE — non spécifiée dans la demande]** : gnubg/TD-Gammon utilise historiquement la **sigmoïde**. Recommandation si architecture ReLU interne + sigmoïde sortie : ReLU par `vmaxps(acc,0)` (bit-exact trivial), sigmoïde réservée aux 5 sorties (approximation polynomiale déterministe ou LUT partagée entre plateformes). **À clarifier.**

### Tableau des cycles/µs attendus (Zen 3/4, 4,5 GHz, B=8, 2 vec-MAC/cycle = 16 MAC scalaires/cycle) [EXTRAPOLÉ]

MACs : L1=100 352, L2=262 144, L3=131 072, L4=32 768, L5=640 (total 526 976).

| Couche | MAC | Cycles/lot(8) | Cycles/position | µs/position (crête) |
|---|---|---|---|---|
| L1 196→512 | 100 352 | 50 176 | 6 272 | 1,39 |
| L2 512→512 | 262 144 | 131 072 | 16 384 | 3,64 |
| L3 512→256 | 131 072 | 65 536 | 8 192 | 1,82 |
| L4 256→128 | 32 768 | 16 384 | 2 048 | 0,46 |
| L5 128→5 | 640 | 320 | 40 | 0,01 |
| **Total** | **526 976** | **263 488** | **32 936** | **7,32** |

- **Plancher (crête)** ~**7,3 µs/position** ; **réaliste (60–70 %)** ~**10–12 µs** ; **conservateur (40 %)** ~**18 µs**. Tous sous 41 µs (2,3× à 5,6×).
- **Sparsité couche 1** (38/196) : L1 → ~9 728 cycles/lot, soit −1,1 µs/position → gros réseau ~6,2 µs crête ; petit réseau ~5× sur le total.
- **Apple** (NEON 128b, no-FMA = 2 vec-MAC/cycle = 8 MAC scalaires/cycle) : plancher M1 (3,2 GHz) ~20,6 µs ; M4 (4,5 GHz) ~14,6 µs ; **le no-FMA coûte ~2×** (prix structurel du bit-exact sur ARM). Réaliste M1 ~30–40 µs, M4 ~20–25 µs.
- **Golden/Raptor Cove** (~1,5 MAC/cycle split) : plancher ~9,8 µs à 5 GHz. **Gracemont** ~2× plus lent (128b).

**Correction de l'estimation utilisateur** : « 527k/16 = 33k cycles ≈ 0,9 µs/position » confond lot et position. 33k cycles ≈ **une position** ; les 8 positions du lot terminent en ~263k cycles → **~7,3 µs/position amortie** (≈ ×8 l'estimation). Reste ~5,6× sous 41 µs.

### Catalogue de pièges

| Piège | Symptôme | Cause | Remède |
|---|---|---|---|
| **Acc en mémoire** | 2–4× trop lent, `vmovups` acc en boucle | pas de `__restrict`, acc non registre | accus locaux, `__restrict`, dérouler boucle externe |
| **1 seul accumulateur** | débit ≈ 1/latence | chaîne d'add dépendante | ≥6 accus (Zen)/≥12 (Apple) |
| **Désalignement** | loads split | tampons non alignés 32 | `aligned(32)`, `vmovaps` |
| **Aliasing** | vectorisation ratée | pointeurs supposés chevauchants | `__restrict` |
| **4K aliasing** | stalls store→load | adresses à multiple de 4 Ko | padder les strides |
| **Dénormaux** | ×100 + non-déterminisme | FTZ/DAZ différent entre plateformes | **fixer FTZ+DAZ identiquement** (MXCSR / FPCR) — critique bit-exact |
| **Transition SSE/AVX** | pénalité VZEROUPPER | mélange SSE/AVX | `vzeroupper`, tout en VEX |
| **Downclocking AVX** | fréquence réduite | AVX lourd (AVX-512) | rester AVX2 256b (déjà le cas) |
| **Contention L3** | débit chute multi-thread | threads de rollout | un jeu de poids par CCX |
| **Faux partage** | invalidations | accus de threads adjacents | padding 64 o/thread |
| **Page faults poids** | latence sporadique | 2,1 Mo non mappés | prefault / huge pages / warm-up |
| **Broadcast AMD sur pipe FP** | 4 pipes saturés vite | vbroadcastss = FP12 sur Zen | pré-broadcaster hors boucle j ; préférer NEON indexé |

### Micro-benchmarks (par risque décroissant)

Mesure : `rdtsc`/`rdpmc` + `perf` (`fp_ret_sse_avx_ops`, `fpu_pipe_assignment` sur Zen) ; `kperf`/`mach_absolute_time` sur macOS.

1. **[RISQUE MAX] Split mul+add vs FMA saturé.** 6–8 accus YMM, 10⁶ itér. **Seuil : split ≥0,95× FMA → valider le design.**
2. **[RISQUE MAX] Nombre d'accumulateurs.** Balayer I∈{1,2,4,6,8}. **Seuil : plateau à I=6 (Zen)/12 (Apple).**
3. **[RISQUE ÉLEVÉ] Acc mémoire vs registres (piège gcc).** Boucle naïve ±`__restrict`/déroulage vs noyau manuel. **Seuil : quantifier le 2–4×.**
4. **[RISQUE ÉLEVÉ] Broadcast AMD (FP) vs Intel (load-only).** `fp_ret_sse_avx_ops`. **Seuil : si broadcast vole >10 % des slots FP, restructurer.**
5. **[RISQUE MOYEN] Sparsité couche 1 input-major vs dense** (petit ET gros réseau). **Seuil : ≥3× sur petit ; si <1,5× sur gros, ne pas activer.**
6. **[RISQUE MOYEN] Gather vs indices actifs.** **Seuil : gather attendu 5–10× plus lent.**
7. **[RISQUE MOYEN] B∈{8,16,32} + remplissage partiel.** **Seuil : confirmer B=8 compute-bound.**
8. **[RISQUE MOYEN] NEON indexé vs broadcast explicite** (M-series).
9. **[RISQUE FAIBLE] Transpose 196×B.**
10. **[RISQUE FAIBLE] FTZ/DAZ + bit-exactitude x86/ARM/WASM.**
11. **[RISQUE FAIBLE] Roofline L3 multi-thread.**

---

## Recommendations

**Étape 1 (avant de figer quoi que ce soit) — valider les hypothèses à risque max.** Écrire les benchmarks 1, 2 et 3. Décision *go/no-go* sur l'architecture split-mul-add : si le débit split atteint ≥0,95× le FMA sur Zen (attendu), figer le design « position par voie » ; sinon reconsidérer (par ex. réordonnancement des accumulateurs). Confirmer que ≥6 accumulateurs saturent les pipes FADD.

**Étape 2 — écrire le noyau dense couches 2–5** en « position par voie », tuile I=6–8 × B=8, `-ffp-contract=off`, accus en registres, boucle i déroulée. Comparer aux 41 µs : cible intermédiaire **< 15 µs/position** sur Zen avant sparsité.

**Étape 3 — traiter la couche 1.** Décider via le benchmark 5 : si gammonNet utilise majoritairement le **petit réseau** (196→32→5), la sparsité input-major est prioritaire (~5× sur le total). Si le **gros réseau** domine, la sparsité n'apporte que ~1,1 µs — l'implémenter seulement si le budget de développement le permet. Bannir `vgatherdps` dans tous les cas.

**Étape 4 — portage NEON puis WASM.** Sur Apple, exploiter le `fmul`/`fmla` indexé (pas de broadcast). Accepter le facteur ~2× du no-FMA comme coût du bit-exact. Vérifier l'identité bit-à-bit x86/ARM/WASM (benchmark 10) avec FTZ/DAZ fixés.

**Seuils qui changeraient les décisions :** (a) si le benchmark 1 montre split < 0,9× FMA sur Zen → envisager de relâcher le bit-exact uniquement sur les couches internes (garder l'ordre fixe seulement sur la couche de sortie) ; (b) si B=8 s'avère memory-bound (contredit l'analyse 32 o/cycle) → passer à B=16 ; (c) si le remplissage moyen du lot < 8 en pratique → prioriser la file d'attente inter-nœuds sur le dispatch multi-largeur.

---

## Caveats

- **[NON TROUVÉ]** Latence/débit exacts VMULPS/VADDPS 256b sur **Gracemont** (extrapolés depuis la structure 2×128b, ports 20-22).
- **[NON TROUVÉ]** Timings NEON précis (FMUL/FADD/FMLA) sur **Avalanche (M2), Everest (M3/M4)** — hérités de Firestorm (M1, Dougall Johnson) par hypothèse.
- **[NON TROUVÉ]** Page uops.info YMM exacte de `VBROADCASTSS` sur Zen 3/4 (confirmée sur la forme ZMM + AMD SOG) et débit *summary* exact de `VGATHERDPS` YMM (extrapolé depuis `VGATHERQPD`).
- **[INCERTITUDE]** Fonction d'activation de gammonNet (sigmoïde vs ReLU) non spécifiée — impacte la stratégie bit-exact de la non-linéarité.
- **[INCERTITUDE]** Bande passante L3→cœur par cœur pour Apple/Intel (le 32 o/cycle est solidement sourcé pour Zen uniquement).
- Toutes les valeurs [EXTRAPOLÉ] de µs/position supposent une exécution parfaitement pipelinée sans stalls mémoire ; les mesures réelles doivent servir d'arbitre. Les cibles « réaliste/conservatrice » intègrent une décote empirique (60–70 % / 40 % de la crête) qu'il faut vérifier par profiling.

### Références (URL + licence des codes)

- **uops.info** — https://uops.info (VMULPS/VADDPS/VFMADD213PS/VBROADCASTSS/VGATHERDPS, ZEN3/ZEN4).
- **Agner Fog**, Instruction tables & Microarchitecture — https://www.agner.org/optimize/ (GFDL).
- **AMD SOG Family 19h** : Zen 3 (56665), Zen 4 (57647) — https://www.numberworld.org/blogs/2024_8_7_zen5_avx512_teardown/57647_zen4_sog.pdf
- **Intel 64/IA-32 Optimization Reference Manual** (355308, fast adder Golden Cove p1/p5) — https://cdrdv2-public.intel.com/779559/355308-Software-Optimization-Manual-048-Changes-Doc-2.pdf
- **Dougall Johnson**, Apple Firestorm SIMD/FP tables — https://dougallj.github.io/applecpu/firestorm-simd.html
- **Chips and Cheese** (Zen 3 V-Cache, Golden Cove, Infinity Fabric) — https://chipsandcheese.com
- **AnandTech** — Zen 3 deep dive (32 B/cycle) — https://at-web1.www.anandtech.com/show/16214/
- **Stockfish NNUE** (GPL-3, *lecture technique uniquement*) — https://github.com/official-stockfish/nnue-pytorch/blob/master/docs/nnue.md
- **gnubg** (GPL-3, *référence de mesure/lecture uniquement*) — https://git.savannah.gnu.org/cgit/gnubg.git (neuralnet.c, neuralnetsse.c, simd.h)
- **XNNPACK** (**BSD-3-Clause, recommandé**) — https://github.com/google/XNNPACK (micro-noyaux mr×nr, Marat Dukhan)
- **LIBXSMM** (BSD-3, petits GEMM JIT NEON) — https://scalable.uni-jena.de/opt/sme/intro.html
- **-ffp-contract / déterminisme flottant** — https://kristerw.github.io/2021/11/09/fp-contract/ + doc gcc.
- **Nasu (2018)**, « Efficiently Updatable Neural-Networks » (origine NNUE).