# Recherche externe

Rapports de recherche approfondie commandés depuis blunderDB et versés ici tels quels.
Ils **documentent**, ils ne décident pas : une conclusion qui engage le code passe d'abord
par un ADR ou une fiche de `tasks/`, et les chiffres qu'ils citent sont mesurés chez nous
avant d'être crus. Chacun porte ses propres marqueurs de fiabilité (`[MESURE]`,
`[ÉDITEUR/DÉCLARÉ]`, `[EXTRAPOLÉ]`, `[NON TROUVÉ]`) — les lire.

| Fichier | Question posée | Ce qu'il a tranché | Utilisé par |
|---|---|---|---|
| [P1](P1-simd-go-bit-identique.md) | Écrire du SIMD en Go sans cgo, et garder le bit-à-bit entre AVX2, NEON et pur Go | **avo ne génère pas d'arm64** : AVX2 via avo, NEON en Plan 9 à la main, repli Go par build tag. Ne pas migrer vers `simd/archsimd` (amd64 seulement en Go 1.26, NEON en 1.27). `float32(a*b)+c` est une **barrière de fusion garantie par la spec**. Go n'active ni FTZ/DAZ ni FZ : dénormalisés identiques et conformes IEEE-754 sur les deux architectures. | #133 (F1), ADR-0024 |
| [P2](P2-noyau-mlp-sans-fma.md) | Concevoir un noyau MLP float32 groupé sans FMA sur Zen, Intel et Apple | **L'absence de FMA ne coûte rien sur Zen 3/4** (`VMULPS` sur FP0/FP1, `VADDPS` sur des pipes disjointes FP2/FP3), ~0,75× sur Intel, ~0,5× sur Apple. Tuile **6-8 sorties × 8 positions**, ≥6 accumulateurs. Couches 2-5 « position par voie », **couche 1 en input-major creux**. **Jamais de `vgatherdps`**. | #133 (F1) |
| [P3](P3-parallelisme-expectiminimax.md) | Ordonnancer un expectiminimax sur 21 lancers, en Go, de façon déterministe | Plafond réaliste **×5 à ×6,5 sur 8 cœurs physiques** ; le SMT n'apporte presque rien, donc le ×4 « sur 16 threads » est déjà proche du plafond. File atomique triée LPT (coût = nombre de coups générés), **frontière aplatie** : une file par niveau, un seul `Wait()`. Cache partagé seulement si le hit-rate monte de ≥5 points, avec tag de Hyatt contre les entrées déchirées. gnubg ne parallélise pas l'évaluation d'un coup. | #148 (F4) |
| [P4](P4-gains-algorithmiques-amont.md) | Les gains qui *changent* ce que le moteur joue | Ordre : distillation vers 60-100 k MAC, puis filtres à seuil d'équité, puis int8 QAT per-channel. **Tables de transposition sur les nœuds et Star1/Star2 entrent en conflit avec l'inférence par lots** — leurs dépendances séquentielles annulent le gain SIMD. Distinction utile : filtres, distillation et élagage interne sont **approximatifs** (jauge de force obligatoire) ; un cache d'évaluation pur et Star1 sont **exacts**. | gammonNet amont (décision D6), #150 |

Les questions posées sont conservées en annexe de `tasks/gammonnet-perf/README.md`, section 5.
