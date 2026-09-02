# Gains algorithmiques qui *changent* le jeu d'un moteur MLP de backgammon — état de l'art et ordre de priorité

## TL;DR
- **Priorité 1 = distillation vers un réseau 60–100 k MAC** : c'est le seul levier qui attaque directement votre goulot (527 k MAC vs ~32 k pour gnubg) et le seul dont un précédent backgammon documenté (Backgammon-NN de Chris Whittington ; wildbg) montre qu'il *survit à la recherche*. **Priorité 2 = filtres de coups à seuil d'équité** (« movefilter » gnubg) : levier de vitesse le mieux mesuré du domaine (« less than 1% of all moves come out different » au niveau Normal). **Priorité 3 = quantisation int8 QAT déterministe** (2–4× de débit sur les couches affines, perte quasi nulle en per-channel), à condition de renoncer à l'accumulateur incrémental NNUE, inapplicable à un MLP dense.
- **Tables de transposition sur nœuds internes** et **Star1/Star2** offrent de vraies réductions de nœuds mesurées (Star2-TT : « a 37% reduction in nodes searched over plain Star2 » ; Star2 permet « depth-5 full-width searches (up from 3) »), mais entrent en **conflit direct avec l'inférence par lots** : leurs dépendances séquentielles annulent le gain SIMD/batch qui fait toute la vitesse de votre évaluateur dense. À n'envisager qu'en dernier.
- **Distinction critique pour votre jauge de force** : filtres, distillation, Star2-probing, réseaux d'élagage internes sont **approximatifs** (peuvent changer la décision → jauge obligatoire) ; un **cache d'évaluation** pur (façon gnubg EvalCache) et **Star1** sont **exacts** (ne changent jamais la décision, à noise=0), donc ne nécessitent qu'un contrôle de régression bit-exact et non une jauge de force.

## Key Findings

1. **Le seul gain « qui change le jeu » validé en backgammon moderne contre un arbitre externe est la distillation.** Backgammon-NN documente publiquement que toutes les autres pistes (largeur, features, routing) plafonnent, et que seule la distillation d'une recherche 2-ply / d'un maître externe (gnubg 2-ply) produit un gain qui « ne se dissout pas sous la recherche » — exactement le symptôme que vous observez (avantage 0-ply +0,00247 qui s'effondre à 2-ply).

2. **Les filtres de coups gnubg sont la technique la mieux paramétrée et la mieux mesurée.** Valeurs exactes documentées par niveau ; effet « moins de 1 % des coups changent » ; calibration récente sur benchmark Depreli par Philippe Michel et Joseph Heled (juillet 2024).

3. **La quantisation int8 est transposable mais avec une réserve majeure** : l'astuce d'accumulateur incrémental de NNUE ne s'applique PAS à votre MLP dense (entrée non creuse, pas de mises à jour incrémentales). Vous héritez du gain int8 sur les couches affines (2–4×) mais pas du gain structurel de NNUE.

4. **Star1 est exact ; Star2 est exact sur la valeur mais son gain vient du probing séquentiel, incompatible avec le batch.** C'est le point dur de votre architecture.

5. **Le « EvalCache » de gnubg est un cache d'évaluations de feuilles, pas une table de transposition sur nœuds internes** — défaut `CACHE_SIZE_DEFAULT 19` → 2^19 = 524 288 entrées, keyé par (position + contexte d'éval incluant le ply via `EvalKey(pec, nPlies, pci, fCubefulEquity)`), donc n'échange jamais de valeurs entre profondeurs et ne change jamais la décision.

## Tableau comparatif

| # | Technique | Exact / Approx. | Gain de vitesse rapporté [étiquette] | Perte de force mesurée [étiquette] | Méthode de mesure de la source | Qualité source | Effort impl. | Risque | Applicabilité projet |
|---|-----------|-----------------|--------------------------------------|-------------------------------------|---------------------------------|----------------|--------------|--------|----------------------|
| 1 | Filtres de coups à seuil (movefilter gnubg) | **Approximatif** (change la décision par élagage) | Supremo ≈ 2× plus lent que World Class pour gain minime ; réglage Michel « 12@0.24 + 6@0.12 » ≈ +20 % vs WC [MESURE] ; « 12 au lieu de 16 » −20–25 % temps [ÉDITEUR] | « less than 1% of all moves come out different » (nets d'élagage) [MESURE] ; coût jeu Depreli WC −16,74 vs Supremo −16,14 [MESURE] | Benchmark Depreli (corpus positions) + timing sur échantillon de matchs | **Élevée** (code + auteurs primaires) | **Faible** | Faible | **Directe** ; vous l'avez déjà (« 8 meilleurs à 0,16 ») |
| 2 | Table de transposition sur nœuds internes (expectiminimax) | **Approximatif si bornes réutilisées entre profondeurs** ; exact si valeurs exactes même profondeur | Star2-TT : « a 37% reduction in nodes searched over plain Star2 » ; « it takes 31% less time for Star2-TT to reach depth 13 » (jeu « Dice », prof. 13) [MESURE] | Non mesurée en équité en backgammon [NON TROUVÉ] | Nombres de nœuds + temps sur 50 positions test | Élevée (Veness & Blair 2007) mais **domaine « Dice », pas backgammon** | Élevé | **Élevé** (collisions, correction bornes, casse le batch) | **Faible** ; peu de transpositions réelles en BG + conflit batch |
| 3 | Distillation vers réseau 60–100 k MAC | **Approximatif** (nouvelle fonction d'éval → nouvelles décisions) | ∝ réduction MAC ; cible 60–100 k = **5–9×** moins de MAC [EXTRAPOLÉ] | Distillation gnubg-2-ply : net 256×128 bat le champion self-play **53,4 %** sur 40 000 parties ; ~parité gnubg à profondeur égale (49,1 %) [MESURE] | Appariement massif (20–40 k parties), dés en miroir, arbitre gnubg | **Élevée** (Backgammon-NN, wildbg, dépôts publics) | Moyen | Moyen | **Directe et prioritaire** |
| 4 | Quantisation int8 déterministe (QAT per-channel) | **Approximatif** (erreur de quant. → décisions rares changées) ; **reproductible bit-exact** si entier pur | Stockfish : « ~60 M éval/s » CPU en int8 ; NNUE >> float [ÉDITEUR] ; 2–4× typique sur couches affines [EXTRAPOLÉ] | NNUE : erreur « negligible » car réseau peu profond [ÉDITEUR] ; INT8 générique : chute top-1 < 1–2 % [MESURE, autre domaine] | fishtest SPRT (Elo), corpus | **Élevée** (Stockfish, nnue-pytorch, Jacob et al.) | Moyen-élevé | Moyen | **Partielle** : gain affine OUI, accumulateur incrémental NON |
| 5 | Réseaux d'élagage aux nœuds internes | **Approximatif** | gnubg : nets de prune « increases the speed considerably » [ÉDITEUR] | « less than 1% of all moves come out different » (Segrave) [MESURE] | Analyse corpus (Segrave) | Élevée (doc gnubg) | Moyen | Moyen | **Directe** ; c'est déjà votre « petit réseau distillé » d'élagage |
| 6 | Star1 / Star2 (Ballard) | **Star1 = exact** ; **Star2 = exact sur valeur** (probing = bornes, pas approximations) | Star2 : « depth-5 full-width searches (up from 3) » [MESURE] ; Ballard : Star1 « reduce the complexity... by 25–30 percent », probing « more than 50 percent » [MESURE] ; Star2-TT −37 % de plus [MESURE] | Aucune (algos exacts) — mais Hauk : « good checker play in backgammon does not require deep searches » [MESURE] | Nombres de nœuds, jeu « Dice » + fonction d'éval gnubg | **Élevée** (Hauk/Buro/Schaeffer 2004) | Élevé | **Élevé** (séquentiel → casse le batch) | **Faible** ; incompatible inférence par lots |

## Details

### Technique 1 — Filtres de coups à seuil d'équité (movefilter gnubg)

**Sources primaires.**
- Code : `eval.h` (`movefilter defaultFilters[MAX_FILTER_PLIES][MAX_FILTER_PLIES]`, `aaamfMoveFilterSettings[NUM_MOVEFILTER_SETTINGS][…]`, struct `movefilter{Accept, Extra, Threshold}`), `set.c`, `gnubg.c` (presets `MOVEFILTER_NORMAL`), `FindnSaveBestMoves` (dépôt github.com/mormegil-cz/gnubg, miroir CVS). Licence **GPL-3 → oracle de mesure uniquement, jamais source de code/poids**.
- Doc : gnubg.org « Evaluation settings » et GNU Backgammon Manual V0.16 « Move filters ».
- Liste : bug-gnubg, thread « Move filters », Philippe Michel & Joseph Heled, 7–8 juillet 2024 (auteur historique des filtres = Joseph Heled : « the testing I did 20+ years ago, when I developed them »).

**Paramétrage exact (un filtre = {accepter n, ajouter jusqu'à m, seuil d'équité}).**
- **Normal / World Class** (2-ply) : au 0-ply, « keep the first 0 0-ply moves and up to 8 more moves within equity 0.16 » ; « Skip pruning for 1-ply moves ». C'est votre « 8 meilleurs à moins de 0,16 ».
- **Supremo** : « keep the first 0 0-ply moves and up to 16 more moves within equity 0.32 » (2-ply).
- **4-ply** : 0-ply « 8 dans 0,160 » puis 2-ply « accepter 0, ajouter jusqu'à 2 dans 0,040 ».
- Exemple MatchDog (3-ply) : `movefilter 3 0 → 16@0.320`, `movefilter 3 2 → 4@0.080`.
- Illustration doc : sur l'ouverture 4-2, meilleur coup +0,207 ; seuil 0,160 ⇒ garder les coups d'équité > +0,047 ⇒ 3 coups passent au 2-ply.

**Calibration.** Historiquement empirique (Heled). Recalibration mesurée (2024, Philippe Michel) sur le **benchmark Depreli** + timing d'analyse de matchs : baseline World Class = coût de jeu **−16,74** en 38,72 s ; Supremo **−16,14** en 70,39 s (« almost twice as slow » pour un gain minime). Le réglage recommandé par Michel — `2-ply : 0/12@0.24 puis 0/6@0.12` — donne **−16,45 en 31,74 s** (« 20% faster than WC and more than twice as [fast as] Supremo »). Au 3-ply, ajouter un filtre 1-ply « 8@0.16 » + 2-ply « 4@0.06 » est « almost 20% faster with no significant change in strength (less than 1% in the Depreli benchmark) ».

**Change les décisions ?** OUI (approximatif) → jauge de force requise.

**Applicabilité.** Directe, effort faible, aucun problème de licence (technique, pas code copié). Meilleur rapport gain/risque du tableau.

### Technique 2 — Tables de transposition sur nœuds internes d'un expectiminimax

**Sources primaires.**
- Veness & Blair, *Effective Use of Transposition Tables in Stochastic Game Tree Search*, IEEE CIG 2007. **Source la plus directe** sur les TT aux nœuds de hasard.
- Ballard 1983 (fondations), Hauk/Buro/Schaeffer 2004 (backgammon).
- Zobrist 1970/1990 (hachage) ; clé de position gnubg = **10 octets** (14 car. Base64), encodage bit-string documenté (manuel gnubg « A technical description of the Position ID », starting position = `E0 73 F0 01 30 E0 73 F0 01 30`).

**Résultats mesurés.** Veness & Blair introduisent **Star2-TT** : le stockage aux nœuds de hasard doit conserver **borne inférieure ET supérieure** (avec leurs profondeurs), car un nœud de hasard peut avoir les deux quand la somme pondérée sort de la fenêtre. Résultat verbatim : « these procedures, in combination with the Star2 algorithm, give a 37% reduction in nodes searched over plain Star2 for the game of Dice at a search depth of 13 », et « on average, it takes 31% less time for Star2-TT to reach depth 13 compared to Star2 » — **mais dans le jeu « Dice », pas le backgammon**. Point crucial : « We cannot use such a rule [incremental window update] because we can never be sure of what successor information will be determined from the transposition table before we begin searching the successors. »

**Distinction cache vs TT (essentielle).**
- **Cache d'évaluation** (gnubg EvalCache) : mémoïsation d'évaluations de **feuilles**, keyé par (clé de position 10 o. + `evalcontext{fCubeful, nPlies:3, fUsePrune, fDeterministic, rNoise}` via `EvalKey(pec, nPlies, pci, fCubefulEquity)`). Défaut **`CACHE_SIZE_DEFAULT 19` → 2^19 = 524 288 entrées** ; max GUI `CACHE_SIZE_GUIMAX 23` = 2^23. API `EvalCacheStats(pcUsed, pcLookup, pcHit)`, `EvalCacheFlush`, `EvalCacheResize` ; commande `set cache <n>`. **Le ply fait partie de la clé → aucune réutilisation entre profondeurs → EXACT, ne change JAMAIS la décision** (à noise=0, mémoïsation bit-exacte).
- **Vraie TT sur nœuds internes** : réutilise une valeur/borne calculée à une **profondeur différente** → peut changer la décision → **approximatif**, jauge requise.

**Taux de collision / clé.** En backgammon, les transpositions réelles dans un arbre 2-ply sont **rares** (chaque coup avance des dames, peu de chemins reconvergent en 2 demi-coups). Le bénéfice attendu est faible, contrairement au jeu « Dice » (grille m×m) que les auteurs qualifient de « prime candidate for the transposition table improvement, since many different lines of play can lead to the same position ».

**Applicabilité.** **Faible.** Peu de transpositions en BG 2-ply + maintien de bornes séquentielles qui casse votre inférence par lots. Recommandation : garder un **cache d'évaluation pur** (exact, sûr) et ne PAS bâtir de TT sur nœuds internes.

### Technique 3 — Distillation vers un réseau de 60–100 k MAC

**Sources primaires.**
- Backgammon-NN (Chris Whittington), rapport de développement public (whittingtonchess.com/backgammon-report) + dépôt GitHub. Licence à vérifier avant réutilisation de code ; les *résultats* sont utilisables comme référence.
- wildbg — dépôt public de référence **carsten-wenderdel/wildbg** (Rust, **licence permissive**), dépôt d'entraînement `wildbg-training` public. wildbg : « As of January 2024, it reaches an error rate of roughly 5.9 for 1-pointers when being analyzed with GnuBG 2-ply » ; entraîné par **rollouts cubeless money-game + apprentissage supervisé** (pas de TD-learning ; « No TD-learning or other reinforcement learning is used »).
- Fond théorique : expert iteration / policy distillation (AlphaZero) ; Tesauro (TD-Gammon, CACM 1995).

**Résultats mesurés (Backgammon-NN).**
- Distillation du **2-ply** dans l'éval statique : premier gain « qui survit à la recherche » (~52 % vs champion à 1-ply, PPG +0,06 ; parité gnubg 0-ply). Les gains 0-ply antérieurs (bucketing v1.7, routing v1.8) « evaporated at 1-ply » — exactement votre symptôme.
- Distillation d'un **maître externe (gnubg 2-ply)** : net **256×128** entraîné from scratch sur 22,5 M positions bat le champion self-play **53,4 %** sur 40 000 parties à **coût par coup identique** (z +13,70) ; ~46,1 % vs gnubg-2-ply (handicap de profondeur) et **49,1 % à profondeur égale** (parité, sur 3 000 parties money).
- **Plafond du maître (teacher ceiling) confirmé** : « a network fitted to labels its own engine produced cannot become better than that engine » → il faut un maître hors boucle.
- **Qualité de label >> volume** : parité avec le champion (3 M parties self-play) atteinte avec **400 k** positions labellisées par rollout (~375× moins), puis saturation. La **profondeur, pas la largeur, est le levier** (256 mono-couche fait match nul ; 256×128 gagne).
- Choix du net : 256×128 distillé retenu plutôt qu'un 512×256 car « statistiquement indiscernable » (z −1,50) tout en coûtant 1,58× moins par éval.

**Change les décisions ?** OUI (approximatif) → jauge requise. Mais c'est *le* levier qui adresse votre goulot MAC.

**Applicabilité.** **Directe et prioritaire.** Cible 60–100 k MAC atteignable avec 256×128 (ordre de grandeur cohérent avec Backgammon-NN). Contrainte de licence : **gnubg ne sert que d'oracle** pour labelliser (mesure), ce qui est licite ; ne copiez ni poids ni code GPL. wildbg (permissif) est une référence d'architecture réutilisable.

### Technique 4 — Quantisation int8 déterministe (QAT per-channel)

**Sources primaires.**
- nnue-pytorch `docs/nnue.md` (Stockfish) : schéma int8 poids/entrées, int16 où nécessaire, **accumulation int32** via SIMD ; ClippedReLU clampé à 0..127 ; « Quantization inevitably introduces error… however in the case of NNUE networks, which are relatively shallow, this error is negligible ».
- Code Stockfish `nnue/layers/affine_transform.h`, discussions TalkChess : `_mm256_maddubs_epi16` (**vpmaddubsw**, produit octet non-signé × octet signé → int16 saturé), `_mm256_madd_epi16` (**vpmaddwd**), accumulation `_mm256_add_epi32`. Sur AVX-VNNI/AVX512-VNNI : **vpdpbusd** (dot product int8→int32 en une instruction, **sans saturation** ; « There is no such issue on other CPU architectures (x64 with VNNI and Arm) »).
- Jacob et al. 2018, Krishnamoorthi 2018 ; Intel neural-compressor (« per-channel pour poids, per-tensor pour activations »).
- WASM : `i32x4.dot_i16x8_s` (PR WebAssembly/simd #127, dot product 2-wide int16→int32 ; « The 4-element dot product producing a 32-bit result never overflows »). SIMD128 déterministe (Safari 16.4+). Le `i8x16.dot_i7x16` relaxé est **relaxed-SIMD → non déterministe (« that lane's result is implementation defined ») → exclu de vos builds bit-exacts**.

**Gain de vitesse.** Stockfish : NNUE int8 « ~60 M évaluations/s » CPU [ÉDITEUR] ; le passage NNUE a donné « > 80 Elo on fishtest » (60 000 parties @ 10+0.1, ELO 92,77 ±2,1) [MESURE, mais mêle NNUE + int8]. Facteur brut int8 vs float32 sur couches affines typiquement **2–4×** [EXTRAPOLÉ].

**Perte de force.** NNUE : « negligible » (réseau peu profond) [ÉDITEUR]. INT8 générique (autre domaine, LLM) : « typical top-1 accuracy drops under 1–2%, and ≤0.5% with tuned MSE-optimal clipping » [MESURE].

**Saturation/overflow.** `vpmaddubsw`/`VPMADDUBSW` peut saturer en int16 (ONNX Runtime : « it can happen that the output does not fit into a 16-bit integer and has to be clamped ») → per-channel + QAT limitent les grands poids (Stockfish a retiré un chemin lent d'affine transform à cause de « a lot high weights », gain jusqu'à 3 %). `vpdpbusd` et `i32x4.dot_i16x8_s` n'ont pas ce problème (accumulation int32).

**L'accumulateur incrémental NNUE est-il applicable ? NON.** NNUE tire l'essentiel de sa vitesse d'une **couche d'entrée creuse mise à jour incrémentalement** (« its purpose is to accumulate between 0 to 30 rows of weights » pour HalfKP). Votre MLP dense a une **entrée dense de 196 float** (encodage Tesauro) recalculée à chaque position : **pas de sparsité, pas de mise à jour incrémentale possible**. Vous héritez donc du gain int8 sur les **couches affines** (512→512→256→128→5) mais PAS du gain structurel d'accumulateur de NNUE. Nuance clé.

**Déterminisme bit-exact multi-plateforme.** Atteignable en **entier pur** avec SIMD128 (`i32x4.dot_i16x8_s`), en évitant tout relaxed-SIMD et tout float intermédiaire. Avantage propre de l'int8 vs float (le float WASM peut différer via FMA/ordre de réduction).

**Change les décisions ?** Marginalement (approximatif : l'erreur de quant. peut basculer des décisions serrées) → jauge légère + test de régression bit-exact.

**Applicabilité.** Partielle mais réelle : combinez-la avec la distillation (QAT int8 sur le net 60–100 k) pour cumuler réduction MAC × débit int8.

### Technique 5 — Réseaux d'élagage aux nœuds internes

**Sources primaires.** GNU Backgammon Manual V0.16 « Pruning neural networks » ; analyse de **Jim Segrave** citée dans le manuel ; flag `fUsePrune` dans `evalcontext`. Analogues théoriques : réseau de politique AlphaZero comme filtre, policy-guided pruning.

**Fonctionnement gnubg.** Un jeu de **réseaux d'élagage** (« pruning nets ») séparés, plus légers, fait une évaluation 0-ply pour **présélectionner** les coups candidats avant l'évaluation profonde. C'est exactement le rôle de votre « petit réseau distillé » d'élagage. Le format `.sgf` gnubg a une version dédiée : « given the speedup and performance improvements, I assume we will drop reduction entirely » (les prune nets ont remplacé l'ancienne « reduction »).

**Résultats mesurés.** Verbatim (manuel gnubg) : « This increases the speed considerably and it doesn't lose much playing strength… Jim Segrave has just done an analysis of this and found that less than 1% of all moves come out different with the pruning nets activated. In most of these positions the move would not have made any difference to the game at all. » Gain de vitesse : « considerable » [ÉDITEUR, non chiffré précisément].

**Change les décisions ?** OUI (approximatif) mais effet mesuré < 1 %. → jauge requise, tolérance élevée.

**Applicabilité.** Directe — vous l'avez déjà. Levier d'optimisation : **distiller aussi le réseau d'élagage** en int8 et calibrer son seuil comme un movefilter. L'élagage aux nœuds **internes** (1-ply) vs feuilles (0-ply) doit être mesuré séparément — c'est le sens du filtre 1-ply de Michel (technique 1).

### Technique 6 — Star1 / Star2 de Ballard

**Sources primaires.**
- Ballard, *The *-Minimax Search Procedure for Trees Containing Chance Nodes*, Artificial Intelligence 21(3):327–350, 1983 (DOI 10.1016/S0004-3702(83)80015-0). Verbatim : algo initial « reduce the complexity of an exhaustive search strategy by 25–30 percent » ; l'algorithme avec probing « is shown to reduce search by more than 50 percent » (avec ordre aléatoire).
- **Hauk, Buro & Schaeffer, *-Minimax Performance in Backgammon*, Computers and Games 2004, LNCS 3846, pp. 51–66 (DOI 10.1007/11674399_4)** + Hauk, *Search in Trees with Chance Nodes*, M.Sc. thesis, U. Alberta 2004. **Sources primaires backgammon.** Ils ont utilisé la fonction d'éval de **gnubg**.
- Veness & Blair 2007 (Star2-TT, jeu « Dice »).

**Résultats mesurés.**
- **Star2 permet « depth-5 full-width searches (up from 3) under tournament conditions on regular hardware without using risky forward-pruning techniques ».** Chiffre-clé backgammon.
- Résultat majeur additionnel : **« with today's sophisticated evaluation functions good checker play in backgammon does not require deep searches »** — recoupe votre observation (avantage 0-ply qui s'effondre à 2-ply) et **plaide contre** investir dans la profondeur.
- Star2-TT : « a 37% reduction in nodes searched over plain Star2 » (Dice, prof. 13).

**Exact ou approximatif ?** **Star1 rend exactement le même résultat qu'Expectimax** (« STAR1 results in an algorithm which returns the same result as EXPECTIMAX, and uses fewer node expansions ») → **EXACT**. **Star2 est également exact sur la valeur** : sa phase de probing établit des **bornes** (pas des approximations) ; si le probing échoue, il retombe sur Star1. Donc Star1 et Star2 **ne changent pas la décision** — ils ne nécessitent PAS de jauge de force, seulement un test de régression.

**Le point dur (batch).** Les gains de Star1/Star2 viennent de **fenêtres alpha-bêta rétrécies séquentiellement** entre successeurs et d'une phase de probing enfant par enfant. Ces **dépendances séquentielles strictes** sont incompatibles avec votre inférence par lots : vous ne pouvez pas évaluer les 21 lancers × ~20 coups en un seul passage GEMM si chaque évaluation dépend du résultat de la précédente. Le gain de nœuds est donc **annulé par la perte du débit batch** (rappel : Backgammon-NN mesure que le passage GEMV→GEMM batché apporte 2,5× — c'est ce débit que Star2 sacrifierait).

**Variantes compatibles batch ?** Pistes mais **aucune mesure backgammon** : (a) Veness & Blair évoquent la parallélisation aux nœuds de hasard (« the risk of performing redundant work at chance nodes might be much lower ») ; (b) *Monte Carlo *-Minimax Search* (Lanctot et al.) combine échantillonnage et bornes ; (c) CHANCEPROBCUT (Winands, CIG 2009) fait du forward-pruning aux nœuds de hasard (approximatif). Aucune ne restaure le débit d'un GEMM plein. **[NON TROUVÉ]** : variante *-minimax vectorisée avec gain de force *mesuré en backgammon* compatible batch.

**Applicabilité.** **Faible** dans un pipeline batché. Star1 (exact) pourrait servir de filet de sécurité, mais le probing Star2 est à écarter tant que votre vitesse repose sur le batch.

## Recommendations

**Jauge de force commune (à figer avant tout).** Arbitre = **gnubg 3-ply** (reproductible ; licence GPL n'affecte pas l'usage comme oracle de mesure ; **n'utilisez jamais ses poids/code dans le produit**). Métrique = **perte d'équité moyenne par décision** contre l'arbitre sur un **corpus stratifié figé et versionné** (par phase : contact / course / crashed / bear-off, pondéré par fréquence réelle). Complément : appariement de matchs à dés en miroir. **Pièges gnubg à connaître** : `set evaluation chequerplay eval plies N` n'affecte PAS la commande `eval` (gnubg calcule toujours la table 0/1/2-ply complète — parsez la bonne ligne) ; les positions résolues par bearoff DB n'impriment que la ligne statique (un parser comptant les lignes « 2 ply: » se bloque et biaise l'échantillon vers les parties rapides).

**Tailles d'échantillon** (écart-type d'équité par coup ≈ 0,03–0,05) : pour détecter **0,002 équité/décision** au seuil 95 %, prévoir **≥ ~10^5–10^6 décisions** stratifiées (cohérent avec les 20–40 k parties de Backgammon-NN et le SPRT fishtest de Stockfish). Seuil de décision : adopter si perte d'équité < 0,001/décision ET IC à 95 % excluant une régression ; rejeter si l'IC exclut tout gain. Pour les techniques **exactes** (cache pur, Star1) : pas de jauge de force, mais **test de régression bit-exact** (0 écart admis à noise=0).

**Ordre de priorité argumenté :**

1. **Distillation → réseau 60–100 k MAC (256×128), maître = gnubg 3-ply (oracle).** *Pourquoi #1* : attaque directement votre goulot (527 k → ~60–100 k MAC = 5–9×), et c'est le seul levier dont un précédent backgammon (Backgammon-NN) prouve qu'il **survit à la recherche** — ce que vos gains 0-ply ne font pas. Protocole : labelliser un corpus stratifié par gnubg 3-ply (les 5 probabilités, pas seulement l'équité), QAT dès l'entraînement, entraînement *from scratch* (Backgammon-NN montre que le warm-start devient nuisible quand le maître surclasse l'élève : « scratch 53.1% vs warm-started 52.4% »). Benchmark de bascule : parité d'équité/décision vs le net 527 k actuel à profondeur 2 ET −0,001/décision vs gnubg-3-ply. Si la perte dépasse 0,003/décision, remonter à ~150 k MAC (384×192).

2. **Filtres de coups à seuil (0-ply + 1-ply).** *Pourquoi #2* : meilleur rapport gain/risque, effort faible, réversible. Protocole : partir de « 8@0.16 », tester la grille de Michel (`0/12@0.24 + 0/6@0.12`) et resserrer jusqu'au seuil de perte. Benchmark : viser +20 % de vitesse à < 0,0005 équité/décision perdue (Michel obtient +20 % pour « less than 1% in the Depreli benchmark »).

3. **Quantisation int8 QAT per-channel (couches affines), SIMD128 `i32x4.dot_i16x8_s`, entier pur, bit-exact.** *Pourquoi #3* : se cumule multiplicativement avec #1 (int8 sur le net distillé). Protocole : QAT per-channel poids / per-tensor activations, accumulation int32, ClippedReLU 0..127 ; valider bit-exactitude Safari 16.4+/natif ; exclure tout relaxed-SIMD. Benchmark : 2–4× débit affine à perte < 0,0005/décision ; si la saturation int16 (`vpmaddubsw`) coûte > 0,001/décision → basculer sur chemin `vpdpbusd`/accumulation int32 native.

4. **Réglage du réseau d'élagage interne** (déjà présent) : distiller aussi le prune net en int8, calibrer son seuil comme un movefilter, mesurer l'effet 1-ply séparément du 0-ply. Cible : < 1 % de coups changés (parité Segrave).

5. **Cache d'évaluation pur (exact)** : reproduire l'EvalCache gnubg (clé = position 10 o. + contexte incluant le ply ; ~2^19–2^21 entrées ; jamais de réutilisation inter-profondeurs). Exact → pas de jauge, seulement régression bit-exacte. Gain de vitesse pur, risque nul.

6. **Star1 / TT interne / Star2** : **à éviter** tant que la vitesse repose sur l'inférence par lots. Ne réévaluer que si vous ajoutez un mode « analyse profonde » non batché (rollouts natifs), où le probing séquentiel redevient rentable ; dans ce cas Star2 (exact) est préférable à une TT approximative.

## Catalogue de pièges (symptôme / cause / remède)

- **Symptôme** : un gain net à 0-ply disparaît à 2-ply (IC contenant zéro). **Cause** : le gain était un « truc 0-ply » que la recherche corrige déjà (cas Backgammon-NN v1.7/v1.8). **Remède** : distiller de la *valeur de recherche* (2-ply/3-ply), pas de l'éval statique ; valider systématiquement au ply où l'app joue.
- **Symptôme** : Star2 réduit les nœuds mais le temps réel augmente. **Cause** : le probing séquentiel détruit le débit batch GEMM (vous perdez le 2,5× de l'inférence groupée). **Remède** : réserver Star2 aux modes non batchés (rollouts) ; sinon, préférer filtres + cache.
- **Symptôme** : divergence de décision entre WASM et natif sur positions serrées. **Cause** : float non associatif (FMA, ordre de réduction) ou relaxed-SIMD. **Remède** : pipeline int8 entier pur avec `i32x4.dot_i16x8_s` ; bannir relaxed-SIMD des builds reproductibles.
- **Symptôme** : chute de force après quantisation. **Cause** : saturation int16 de `vpmaddubsw` sur poids élevés, ou per-tensor sur les poids. **Remède** : per-channel sur les poids + QAT clippant les grands poids ; ou chemin `vpdpbusd`/int32.
- **Symptôme** : harness de mesure vs gnubg qui « discarde » des parties et donne des scores biaisés. **Cause** : positions bearoff n'imprimant que la ligne statique ; parser bloqué → survivants = parties rapides. **Remède** : gérer explicitement les lignes bearoff ; résoudre les courses par pip count.
- **Symptôme** : le net distillé plafonne exactement au niveau du champion. **Cause** : teacher ceiling (labels générés par le moteur lui-même). **Remède** : maître hors boucle (gnubg comme oracle).
- **Symptôme** : plus de données d'entraînement, aucun gain en parties. **Cause** : la validation-loss baisse mais ne se traduit plus en Elo (knee détectable seulement en head-to-head). **Remède** : décider par appariement direct, jamais par loss.

## Caveats
- La quasi-totalité des chiffres de *vitesse* de Star1/Star2/TT proviennent du jeu **« Dice »** (Veness & Blair) ou de conditions matérielles 2004–2007, **pas de mesures backgammon 2-ply modernes**. Transposition prudente.
- Le facteur « 2–4× » de l'int8 sur MLP dense est **extrapolé** de NNUE/quantisation générique ; sur votre 196→512→512→256→128→5 il doit être **mesuré**, pas supposé.
- Les résultats Backgammon-NN sont un **rapport de développement auto-publié** (non revu par les pairs) mais richement instrumenté (tailles d'échantillon, z-scores, corrections d'erreurs documentées) → qualité élevée mais à recouper.
- wildbg est cité « Bernhard Berger » dans votre contexte ; le dépôt public de référence est **carsten-wenderdel/wildbg** (Rust, licence permissive). Vérifiez l'attribution exacte avant citation formelle.
- Le « 75–95 % » de réduction de nœuds Star2 de votre contexte n'a **pas** été retrouvé tel quel dans les sources primaires (Hauk : profondeur 5 vs 3 ; Ballard : 25–30 %, > 50 % avec probing ; Veness-Blair : −37 % Star2-TT vs Star2). À traiter comme **[FOLKLORE/EXTRAPOLÉ]**.
- Les « > 80 Elo » de Stockfish mêlent l'apport NNUE ET int8 ; l'effet de la quantisation seule (isolée) n'est pas publié séparément.

## Section « Non trouvé »
- **[NON TROUVÉ]** Perte de force *en équité par décision* d'une TT sur nœuds internes *en backgammon* (mesures existantes = jeu « Dice », en nœuds/temps seulement).
- **[NON TROUVÉ]** Taux de succès (hit rate) numérique publié de l'EvalCache gnubg (l'API `EvalCacheStats(pcUsed, pcLookup, pcHit)` existe, mais aucun chiffre publié trouvé).
- **[NON TROUVÉ]** Struct exacte de l'entrée de cache gnubg (`lib/cache.c`/`cache.h`) — champs (`cacheNode`) et fonction de hachage non récupérés verbatim (fetch des fichiers bruts bloqué ; à ouvrir directement sur cgit.git.savannah.gnu.org/cgit/gnubg.git/tree/lib/).
- **[NON TROUVÉ]** Variante *-minimax vectorisée / compatible batch avec gain de force *mesuré en backgammon*.
- **[NON TROUVÉ]** Facteur d'accélération int8 vs float32 mesuré sur un MLP dense de backgammon spécifiquement (par opposition à NNUE chess).
- **[NON TROUVÉ]** Chiffre primaire précis pour la « réduction 75–95 % » attribuée à Star2 dans votre contexte de projet.
- **[NON TROUVÉ]** Elo/équité exacte perdue par la quantisation int8 seule (isolée) dans Stockfish/Lc0.