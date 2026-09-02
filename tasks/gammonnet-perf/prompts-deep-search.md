<!-- Annexe du plan tasks/gammonnet-perf/README.md : les questions telles qu'elles ont
été posées. Les rapports sont sous docs/recherche/. -->

# Prompts de deep search

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
