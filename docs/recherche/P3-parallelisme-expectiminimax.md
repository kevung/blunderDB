# Parallélisation déterministe d'une recherche expectiminimax de backgammon en Go — architecture d'ordonnancement, pièges de déterminisme et plafond d'accélération

## TL;DR
- **Sur 8 cœurs physiques, visez un facteur d'accélération réaliste de ×5 à ×6,5** avec un ordonnancement dynamique (compteur atomique sur une file de tâches triée par coût décroissant, LPT) et une frontière aplatie sans barrière par candidat ; votre ×4 actuel « sur 16 threads » est vraisemblablement déjà proche du plafond réel si la machine n'a que 8 cœurs physiques, car le SMT n'apporte quasiment rien sur du code GEMM/FMA saturant les unités vectorielles. [EXTRAPOLÉ]
- **Le déterminisme bit-exact est atteignable et n'est PAS en tension avec un cache d'évaluation partagé sans verrou**, à condition que (a) la somme pondérée des 21 termes reste sérielle en ordre fixe, (b) le cache ne stocke jamais de « profondeur » variable (hit ≡ miss bit à bit), (c) les entrées déchirées (torn) soient détectées par un tag/checksum type Hyatt, et (d) tous les tris et bris d'égalité soient stables et indépendants de l'ordre d'achèvement. [HYPOTHÈSE consolidée]
- **gnubg ne parallélise PAS l'évaluation d'un coup unique** : il découpe les gros travaux (coups d'une analyse, essais d'un rollout) et laisse la somme finale sérielle ; XG parallélise historiquement les rollouts par candidat ; les moteurs d'échecs (Lazy SMP) sont délibérément non déterministes via leur table de transposition partagée — un modèle explicitement inadapté à votre garde bit-exacte. [ÉDITEUR/DÉCLARÉ]

---

## Key Findings

1. **gnubg : granularité grossière, somme sérielle, RNG par thread.** L'auteur du multithreading de gnubg, Jonathan (Jon) Kinsey, a répondu explicitement sur la liste bug-gnubg (08/03/2007, « RE: [Bug-gnubg] Multithread (silly) question? ») que le multithreading ne rend PAS le jeu (le choix d'un coup) plus rapide : « The answer is no. I went for the quick approach of splitting up big tasks (moves in a game and trials in the rollout), the other idea of multi-threading the evaluations sounds great (as everything would be quicker), it's just a whole lot more complicated/likely not to work. » [ÉDITEUR/DÉCLARÉ] La reproductibilité des rollouts est obtenue par un **contexte RNG séparé par thread** (« Give each thread separate random number context to try and get results the same as single-threaded rollouts », rollout.c, ChangeLog 2008) et par un découpage où chaque alternative s'arrête au bon nombre d'essais. [ÉDITEUR/DÉCLARÉ]

2. **XG (eXtreme Gammon) : parallélisme par candidat, peu documenté sur le déterminisme.** Historiquement, XG déroulait chaque coup candidat séquentiellement (« eXtreme rolls out one play … 1296 trials, then it rolls out the next candidate play »). [FOLKLORE de forum, BGonline 2009] L'éditeur (GameSite 2000 Ltd / Xavier Dufaure de Citres) déclare sur sa page « Multiple Threads in eXtreme Gammon » que XG « will use as many threads than your computer has, the limit is 64 », que doubler les cœurs ≈ double la vitesse des rollouts, et — point important pour votre calcul de plafond — « Note that an hyper threaded core is about 1.3 faster than a non-hyper threaded one ». [ÉDITEUR/DÉCLARÉ] Aucune documentation publique sur la reproductibilité bit-à-bit des rollouts XG selon le nombre de threads. [NON TROUVÉ]

3. **Échecs : la table de transposition partagée = non-déterminisme assumé.** Lazy SMP (Stockfish depuis la v7) fait tourner N threads sur la racine, coordonnés uniquement par une TT partagée lock-free ; l'ordre d'écriture dépend de l'OS, donc deux exécutions divergent. C'est l'anti-modèle de votre contrainte. Le **lockless hashing de Hyatt & Mann (2002, ICGA Journal Vol. 25 No. 1 ; idée originale attribuée à Harry Nelson)** protège l'intégrité d'une entrée par XOR de la signature avec les données, sans verrou — le motif exact étant `hashtable[index].key = key ^ data;` à l'écriture et `if ((hashtable[index].key ^ hashtable[index].data) == key)` à la lecture. Technique directement transposable à votre cache. [MESURE/DÉCLARÉ]

4. **Le pic de coût vient des doubles.** Les 6 doubles génèrent beaucoup plus de coups légaux que les 15 non-doubles (jusqu'à 4 demi-coups à placer contre 2), donc la tâche la plus longue domine le makespan. Trier les 21 tâches par coût décroissant (LPT — Longest Processing Time first) borne le makespan à 4/3 − 1/(3m) de l'optimum (Graham 1969). [MESURE pour LPT]

5. **Réductions flottantes et FMA : les pièges silencieux.** En Go, `x*y + z` peut être fusionné en FMA sur arm64/ppc64/s390x/riscv64 mais **jamais automatiquement sur amd64** (confirmé par golang/go issue #71204 : pas de FMA auto même avec `GOAMD64=v3`) ; `float64(x*y) + z` interdit la fusion (issue #17895), et pour `x+y*z` il faut écrire `x + float64(y*z)`, pas `float64(x+y*z)` (recommandation Go, issue #67029). La somme parallèle change le résultat car l'addition flottante n'est ni associative ni commutative. Relaxed-SIMD WASM rend `fma` et `min/max` non déterministes → incompatible avec la garde bit-exacte.

---

## Details

### Axe 1 — Comment les moteurs existants parallélisent et restent (ou non) déterministes

#### 1.1 GNU Backgammon (gnubg)

**Architecture du pool.** gnubg utilise un pool de threads persistant (backend GLib threads par défaut, Win32 en variante ; `--enable-threads` activé par défaut depuis 2011). Le nombre maximal de threads d'évaluation est un paramètre de compilation `--with-eval-max-threads`, **de valeur par défaut 48**. [ÉDITEUR/DÉCLARÉ, ChangeLog 2013-06-04]

**Granularité de la tâche.** gnubg met en file de **gros travaux** : un « move » (position à analyser) dans une analyse de partie, ou un « trial » (essai) dans un rollout — jamais l'évaluation d'une position unique par un réseau. L'idée de paralléliser les évaluations individuelles a été explicitement rejetée par l'auteur comme trop complexe et « likely not to work » (cf. Key Finding 1). [ÉDITEUR/DÉCLARÉ] C'est exactement la granularité que vous avez retenue (21 lancers), avec une différence : vous descendez plus fin car votre réseau est ~16× plus lourd (~527k MAC contre ~32k MAC pour le réseau contact de gnubg), ce qui rend le 2-ply beaucoup plus coûteux et justifie un ordonnancement plus soigné.

**Rollouts vs recherche d'un coup.** Les rollouts sont parallélisés par essais indépendants ; la recherche d'un coup (2-ply) ne l'est pas dans gnubg. C'est cohérent avec le fait que gnubg a un réseau minuscule où le 2-ply est déjà bon marché.

**Cache d'évaluation.** Le cache vit dans `lib/cache.c` / `lib/cache.h` (fonctions confirmées `CacheCreate`, `CacheResize`, wrapper `EvalCacheResize`, variable globale `cCache` ; comparaison de clés par `EqualKeys`, copie par `CopyKey`). Taille par défaut ≈ **21 Mo** (bumpée ×4 en 2010), maximum GUI ≈ 336 Mo. Commandes utilisateur `set cache <taille>` et `show cache` (statistiques). [ÉDITEUR/DÉCLARÉ, ChangeLog + commands.inc] **Le partage exact du cache entre threads et son mécanisme de verrouillage n'ont pas pu être vérifiés dans le code source** (fichiers `lib/cache.c` non récupérables pendant la recherche). [NON TROUVÉ — à vérifier en clonant `git.savannah.gnu.org/git/gnubg.git`]

**Déterminisme des rollouts.** Le mécanisme repose sur : (1) un contexte RNG **par thread**, (2) des dés quasi-aléatoires pré-tirés dans un tableau fixe avant le premier jeu — Manuel GNU Backgammon (« Rollout of cube decisions ») : « Before the first game of a rollout, gnubg creates a psuedo random array which it will use for all the games in the rollout. In effect it has already decided the roll sequence it will use for up to 128 rolls in every game of the rollout. » — avec stratification (« GNU Backgammon only stratifies the first 2 plies of a rollout » ; rotation du deuxième lancer pour n×1296 essais : « n games will start with 11-11… »), et (3) chaque alternative qui s'arrête au bon nombre d'essais indépendamment de la vitesse des autres. L'intention déclarée est d'obtenir « results the same as single-threaded rollouts ». [ÉDITEUR/DÉCLARÉ] Notez que ceci est une intention de reproductibilité statistique/structurelle, pas une certification bit-exacte publiée. [NON TROUVÉ pour la certification formelle]

#### 1.2 eXtreme Gammon (XG)

XG parallélise agressivement (jusqu'à 64 threads), le gain rollout étant quasi-linéaire en cœurs selon l'éditeur. Historiquement (forum BGonline, 2009), XG déroulait les candidats **séquentiellement** l'un après l'autre, contrairement à gnubg qui parallélisait déjà les trials — un point critiqué à l'époque. [FOLKLORE de forum] La reproductibilité bit-exacte des rollouts XG selon le nombre de threads n'est pas documentée publiquement. [NON TROUVÉ]

#### 1.3 Moteurs d'échecs et recherches à nœuds de hasard

- **Lazy SMP** (Stockfish v7+) : threads indépendants sur la racine, seule la TT partagée les coordonne. Non déterministe par construction (l'ordre d'écriture TT dépend de l'ordonnanceur OS). [MESURE/DÉCLARÉ]
- **YBWC** (Young Brothers Wait Concept), **DTS** (Dynamic Tree Splitting, Hyatt/Crafty), **ABDADA**, **PV-splitting** : schémas plus anciens à meilleur « time-to-depth » mais complexes. Comparaisons de force communautaires : « YBW > ABDADA > SHT even at 12 threads ». [FOLKLORE de forum]
- **Lockless hashing (Hyatt & Mann, 2002)** : « you accept that occasionally an entry gets corrupted by a race … but the entry contains a kind of checksum … XOR all other words of the entry with the word that contains the hash signature, before storing the latter ». Détecte les entrées déchirées sans verrou. Directement applicable à votre cache (voir §3.4). [MESURE/DÉCLARÉ]
- **`*`-minimax (Ballard 1983 ; Hauk, Buro & Schaeffer 2004, « Rediscovering *-Minimax Search » et « *-Minimax Performance in Backgammon », CG 2004, LNCS 3846, pp. 51-66 ; thèse T. Hauk, Univ. of Alberta 2004)** : Star1/Star2 généralisent l'élagage alpha-bêta aux nœuds de hasard. Résultat clé : « Star2 allows strong backgammon programs to conduct depth-5 full-width searches (up from 3) under tournament conditions on regular hardware without using risky forward-pruning techniques. » **Tension connue** : Star2 impose un sondage (probing) séquentiel strict des successeurs (dépendances fortes entre sous-arbres), incompatible avec l'inférence réseau par lots (batch GEMM) et avec une parallélisation large à la racine. Vous ne pouvez pas avoir à la fois l'élagage maximal de Star2 ET un batching massif parallèle : il faut choisir. Les mêmes auteurs présentent d'ailleurs des preuves empiriques que « good checker play in backgammon does not require deep searches », ce qui affaiblit l'argument pour Star2 dans votre cas — privilégiez la largeur parallélisable et le batching. [MESURE académique]

### Axe 2 — Schémas de vol de travail / files de tâches en Go

#### 2.1 Coûts mesurés des primitives Go (ordres de grandeur)

| Primitive | Coût typique | Source | Étiquette |
|---|---|---|---|
| `atomic.AddInt64` non contendu | ~0,3 ns/op | benchmark public (Gaborkoos 2025) | [MESURE] |
| `sync.Mutex` Lock/Unlock non contendu | ~0,8 ns/op (« a few ns ») | idem | [MESURE] |
| Envoi/réception sur channel | ~100 ns (classique), « ~75× » un mutex | jbu.io ; Gaborkoos | [MESURE] |
| `atomic.AddInt64` contendu (N goroutines) | dominé par invalidation de ligne de cache (faux partage) | benchmark C++ analogue | [EXTRAPOLÉ] |
| Création de goroutine | pile initiale 2 Ko (vs 2 Mo thread OS) | doc runtime Go | [MESURE] |

Conclusion : un **compteur atomique** est ~2-3 ordres de grandeur moins cher qu'un channel. Pour 21 tâches (ou même quelques centaines de sous-tâches), le channel n'est pas un goulot, mais le compteur atomique sur slice triée est le choix le plus simple et le plus rapide.

#### 2.2 Le pattern recommandé : compteur atomique sur slice triée par coût décroissant (LPT + guided self-scheduling)

C'est l'équivalent de `schedule(dynamic,1)` d'OpenMP. Tri LPT préalable, puis chaque worker fait `atomic.AddInt64(&idx, 1)` pour piocher la tâche suivante. Borne d'ordonnancement LPT : **makespan ≤ (4/3 − 1/(3m)) × optimum** (Graham 1969). [MESURE]

```go
// Tâches pré-triées par coût décroissant (LPT). Déterminisme : l'ORDRE des
// tâches et l'écriture de chaque résultat dans results[t.Idx] sont fixes ;
// seul QUI calcule quoi varie. La somme finale se fait ailleurs, en série.
func runLevel(tasks []Task, results []Term, nWorkers int) {
    var next int64 = 0
    var wg sync.WaitGroup
    for w := 0; w < nWorkers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for {
                i := atomic.AddInt64(&next, 1) - 1
                if int(i) >= len(tasks) {
                    return
                }
                t := tasks[i]
                // results[t.Idx] : index STABLE, indépendant de l'ordre d'achèvement.
                results[t.Idx] = evaluate(t) // emplacement dédié, pas d'accès concurrent
            }
        }()
    }
    wg.Wait()
}
// Ailleurs, en série, dans l'ordre croissant fixe :
// var sum float64 ; for i := 0; i < 21; i++ { sum += weight[i] * float64(results[i].Value) }
```

Comparaison des options d'ordonnancement :

| Option | Surcoût | Équilibrage | Déterminisme | Verdict |
|---|---|---|---|---|
| Partitionnement statique en tourniquet (actuel) | nul | mauvais (doubles concentrés) | OK | plafonné par le déséquilibre |
| Compteur atomique sur slice triée LPT | ~ns/tâche | quasi-optimal | OK (résultats en emplacements fixes) | **recommandé** |
| `errgroup.SetLimit` | faible | dépend du découpage | OK | acceptable, moins de contrôle sur l'ordre LPT |
| Pool persistant + channel bufferisé | ~100 ns/tâche | bon | OK | utile si sous-tâches nombreuses |
| Deque de vol de travail utilisateur | élevé (complexité) | excellent | OK mais dur à garder bit-exact | inutile : le scheduler Go vole déjà entre P |

#### 2.3 Le déséquilibre backgammon et la borne de makespan

Le speedup est borné par `total / max(tâche_la_plus_longue, total/m)`. Si un double coûte 3-4× un non-double typique et représente une large part du travail d'un niveau, la tâche la plus longue peut à elle seule fixer un plancher de makespan. **21 tâches sur 8 workers est trop grossier** : la variance de coût est trop grande, et 21 n'est pas divisible par 8 (répartition inégale du reste). Il faut **aplatir plus finement**. (Le nombre moyen chiffré de coups légaux par lancer, doubles vs non-doubles, issu d'une source mesurée, n'a pas été trouvé — seule la règle « doubles = 4 demi-coups » est documentée ; mesurez-le sur votre générateur de coups.) [NON TROUVÉ]

#### 2.4 Aplatissement de la frontière (BFS par niveau) pour supprimer les barrières

Votre barrière par candidat approfondi est le principal ennemi. Au lieu d'un DFS avec barrière à chaque candidat, **collectez tous les sous-arbres à approfondir de tout le niveau dans une seule file**, puis ordonnancez-les d'un coup. Cela remplace K barrières par 1 seule et donne beaucoup plus de tâches (21 lancers × jusqu'à 3 candidats ≈ 60+ sous-tâches), ce qui améliore l'équilibrage LPT.

```go
// Frontière : au lieu de barrières par candidat, on aplatit tout le niveau.
type Frontier struct{ tasks []Task; results []Term }

func (f *Frontier) Run(nWorkers int) {
    // tri LPT une seule fois
    sort.SliceStable(f.tasks, func(i, j int) bool {
        if f.tasks[i].Cost != f.tasks[j].Cost {
            return f.tasks[i].Cost > f.tasks[j].Cost // coût décroissant
        }
        return f.tasks[i].Idx < f.tasks[j].Idx // BRIS D'ÉGALITÉ STABLE et déterministe
    })
    runLevel(f.tasks, f.results, nWorkers)
}
```

#### 2.5 Batching des évaluations réseau (GEMM vs GEMV)

Regrouper les 12 meilleurs coups d'un lancer (ou de plusieurs lancers) en un **batch GEMM** amortit le coût mémoire et améliore l'intensité arithmétique par rapport à 12 GEMV séparés. Mais le batching est en tension avec le parallélisme fin : si chaque worker fait son propre batch, les batches sont petits ; si vous voulez de gros batches, il faut regrouper à travers les workers, ce qui recrée une synchronisation. **Recommandation** : batcher AU SEIN d'un worker (les 12 candidats d'un lancer forment un batch GEMM local), et paralléliser AU-DESSUS (les lancers/sous-arbres). Cela garde le déterminisme (chaque batch est calculé entièrement par un worker) et amortit le GEMM. XNNPACK (BSD-3-Clause) fournit les noyaux GEMM/GEMV.

#### 2.6 Vol de travail natif de Go

Le scheduler Go fait déjà du work stealing entre P (chaque P a sa runqueue locale, vole la moitié d'une autre quand vide ; `GOMAXPROCS` = nombre de P). **N'implémentez pas de deque de vol utilisateur** : c'est redondant, complexe, et met en péril le déterminisme. Le compteur atomique sur slice partagée donne déjà un équilibrage dynamique quasi-optimal, et le scheduler Go répartit les goroutines sur les cœurs. Note : à très haut `GOMAXPROCS`, la boucle de vol devient coûteuse quand les runqueues sont vides (golang/go #28808) — raison de plus pour fixer `GOMAXPROCS` = cœurs physiques.

#### 2.7 Mesure honnête de l'accélération

- **Cœurs physiques vs SMT** : le gain SMT pour du code GEMM/FMA saturant les unités vectorielles est faible voire nul, car les unités FMA sont partagées entre threads frères d'un même cœur. Sur Knights Landing, « any instruction that is not a vector FMA instruction represents a loss of 50% of peak » — le cœur est déjà saturé par un seul thread bien vectorisé. [MESURE] L'éditeur de XG mesure ~1,3× par cœur hyperthreadé — donc le SMT n'apporte que ~30 %, pas 100 %. [ÉDITEUR/DÉCLARÉ]
- **Méthodologie** : `testing.B` + `benchstat`, répétitions multiples, épinglage CPU, désactivation turbo/boost, gouverneur `performance`, `GOMAXPROCS` fixé au nombre de cœurs physiques testés. Changez **une variable à la fois**. Comparez des **distributions** (p75/p95/p99), pas des moyennes.
- **Diagnostic du plafond** : `perf stat` (IPC, stalls mémoire), `runtime/trace` (temps d'attente à la barrière), `pprof`. Si l'IPC est bas et les stalls mémoire hauts → bande passante mémoire. Si les workers attendent à la barrière → déséquilibre. Si le speedup plafonne pile au nombre de cœurs physiques → SMT inutile (attendu).

### Axe 3 — Cache d'évaluation partagé vs par worker

#### 3.1 Le point clé : hit ≡ miss bit à bit ⇒ le cache ne casse PAS le déterminisme

Puisqu'un hit rend exactement les mêmes bits qu'un miss (votre garantie), l'ordre d'arrivée et les évictions ne changent **aucune valeur de terme**. Le résultat reste bit-identique. Un cache partagé est donc légitime. [HYPOTHÈSE consolidée par le raisonnement]

#### 3.2 Le piège fatal à éviter absolument : la « profondeur » en cache

Dans les échecs, une entrée TT stocke une évaluation **à une profondeur variable** ; réutiliser une entrée peu profonde vs profonde change le résultat selon l'ordre → non déterminisme. **N'introduisez JAMAIS de profondeur/qualité variable dans votre cache.** Chaque clé doit mapper vers une valeur unique et immuable (l'évaluation à profondeur fixe de cette position). C'est la condition sine qua non de la compatibilité cache partagé ↔ bit-exact. [HYPOTHÈSE — piège transposé des échecs]

#### 3.3 Cache par worker vs partagé

| Critère | Cache par worker | Cache partagé sans verrou |
|---|---|---|
| Taux de hit | plus bas (N caches de taille S ne partagent rien) | plus haut (1 cache N×S, transpositions inter-workers captées) |
| Mémoire | duplication ×N | une seule copie |
| Faux partage | aucun | à éviter par alignement 64 o |
| Complexité | triviale | modérée (tag/checksum) |
| Déterminisme | trivial | OK si hit≡miss et pas de profondeur |
| Détection torn entry | inutile | **obligatoire** |

**Taux de hit attendu** : dans un arbre expectiminimax, beaucoup de positions se répètent entre lancers et entre candidats (transpositions). Aucune donnée chiffrée publiée sur le taux de hit du cache gnubg n'a été trouvée. [NON TROUVÉ] Ordre de grandeur plausible : les transpositions sont fréquentes au backgammon (plusieurs séquences de coups mènent à la même position), donc le cache partagé peut apporter un gain de hit non négligeable, surtout au 2-ply où les sous-arbres se recouvrent. [HYPOTHÈSE]

#### 3.4 Écriture « atomique » d'une entrée : le problème du torn write

- x86-64 : écriture atomique jusqu'à 128 bits (`cmpxchg16b`). **Impossible d'écrire 64 octets atomiquement** sur matériel courant (AVX-512 non garanti atomique ; `LSE128`/`STP` sur ARM récent, non portable). [MESURE/DÉCLARÉ]
- **Solution** : lockless hashing de Hyatt — stocker un tag/signature XORé avec les données (`entry.key = key ^ data`). À la lecture, re-XORer (`(entry.key ^ entry.data) == key`) ; si le résultat ne correspond pas à la clé, l'entrée est déchirée → traiter comme un miss (recalcul). La « course bénigne » est acceptable (hit≡miss), mais une **entrée déchirée non détectée donnerait une valeur fausse** → doit être détectée. [MESURE/DÉCLARÉ]
- Alternative : **seqlock** (compteur de version pair/impair) si vous préférez une sémantique explicite.

#### 3.5 Modèle mémoire Go et race detector

- Le modèle mémoire Go (go.dev/ref/mem) : « A data race is defined as a write … happening concurrently with another read or write … unless all the accesses involved are atomic ». Les programmes avec course sont formellement « incorrects » ; Go garantit DRF-SC pour les programmes sans course.
- Russ Cox reconnaît qu'un read d'un mot mémoire devrait « always return some value written to that word, though which value will be unpredictable » — mais ce n'est pas une garantie ferme du modèle. Les valeurs multi-mots (slices, interfaces, strings) peuvent être vues partiellement (« Off to the Races », research.swtch.com). [DÉCLARÉ]
- **Conséquence CI** : `go test -race` **signalera** votre cache partagé sans verrou comme une course, même bénigne. Options : (a) faire tous les accès cache via `sync/atomic` (`atomic.Uint64` par mot, alignés), ce qui satisfait le race detector et le modèle mémoire ; (b) exclure le cache partagé des builds `-race` (un cache par worker en mode `-race`, partagé en prod) — mais alors vous ne testez pas la vraie config. **Recommandation** : implémentez l'entrée en `atomic.Uint64` (chaque mot atomique), ce qui rend l'accès légal au sens du modèle Go, élimine le torn word au niveau mot, et laisse le tag Hyatt gérer la cohérence inter-mots. Vous gardez `-race` en CI.

#### 3.6 Schémas hybrides

- **Cache partagé lecture seule pré-rempli + cache d'écriture par worker** : élimine toute écriture concurrente. Bon si un socle de positions est connu (ouvertures, bearoff). Déterministe trivialement.
- **Cache par worker avec publication périodique** : complexité inutile ici.
- **Ne pas partager** : acceptable pour démarrer ; c'est votre config actuelle et elle est déjà bit-exacte.

### Catalogue des pièges de déterminisme (symptôme / cause / remède)

| # | Symptôme | Cause | Remède |
|---|---|---|---|
| 1 | Résultat varie selon le nombre de workers | Somme flottante parallèle (non associative) | Somme sérielle en ordre croissant fixe (déjà fait) ; vérifier moyennes de rollout, softmax, normalisation |
| 2 | Résultat diffère amd64 vs arm64/WASM | FMA fusionné sur arm64 mais pas amd64 | `float64(x*y) + z` pour interdire la fusion ; pour `x+y*z` écrire `x + float64(y*z)` |
| 3 | Build WASM ≠ build natif | relaxed-SIMD : `fma`, `min/max` implementation-defined | N'utiliser QUE SIMD128 pour les builds bit-exacts ; bannir `-mrelaxed-simd` |
| 4 | Le coup choisi change entre runs | Bris d'égalité instable sur évaluations égales | `sort.SliceStable` + clé de départage déterministe (index du coup) |
| 5 | Tri non reproductible | `sort.Slice` n'est PAS stable | Utiliser `sort.SliceStable` |
| 6 | L'ensemble des « 12 meilleurs » varie | Seuil d'élagage relatif au meilleur trouvé jusqu'ici (dépend de l'ordre) | Seuil fixe OU évaluer tous puis sélectionner par tri stable ; ne jamais faire dépendre le seuil de l'ordre d'achèvement |
| 7 | Itération non reproductible | Parcours de `map` Go volontairement randomisé | Ne jamais itérer une map dans le chemin déterministe ; utiliser slices triées |
| 8 | Résultat non reproductible sous charge | Budget de temps / timeout | Bannir toute recherche à budget temporel du chemin bit-exact |
| 9 | Valeur fausse rare en multi-thread | Entrée de cache déchirée (torn) | Tag/checksum Hyatt ou seqlock ; entrée en `atomic.Uint64` par mot |
| 10 | `-race` échoue | Cache partagé non atomique | Accès cache via `sync/atomic` |
| 11 | Résultat dépend de l'ordre via le cache | Profondeur/qualité variable stockée en cache | Interdire toute profondeur variable : 1 clé → 1 valeur immuable |

**Le piège #6 est le plus fréquent et le plus insidieux** : si votre sélection des 12 meilleurs après élagage utilise un seuil relatif au meilleur candidat *déjà* évalué, l'ordre d'achèvement des tâches change l'ensemble retenu → non-déterminisme. Rendez le seuil absolu, ou évaluez tous les candidats puis sélectionnez par un tri stable indépendant de l'ordre.

### Section « non trouvé » (lacunes explicites)
- Mécanisme de verrouillage exact et partage inter-thread du cache d'évaluation de gnubg (code `lib/cache.c` non inspecté). [NON TROUVÉ]
- Taux de hit chiffré du cache d'évaluation gnubg. [NON TROUVÉ]
- Certification formelle bit-exacte des rollouts gnubg selon le nombre de threads (seule l'intention est documentée). [NON TROUVÉ]
- Reproductibilité bit-à-bit des rollouts XG selon le nombre de threads. [NON TROUVÉ]
- Nombre moyen chiffré de coups légaux par lancer (doubles vs non-doubles) au backgammon, issu d'une source mesurée. [NON TROUVÉ — seule la règle « doubles = 4 demi-coups » est documentée]
- Présence/absence d'un checksum type Hyatt dans le cache gnubg. [NON TROUVÉ]

---

## Recommendations

**Étape 0 — Verrouiller la garde bit-exacte (avant toute optimisation).**
- Construire un test de régression qui compare, bit à bit, la sortie série vs parallèle sur un corpus de positions (ouverture, milieu, course, contact, bearoff).
- Intégrer les pièges #1–#8 du catalogue comme tests unitaires : forcer `float64(x*y)+z` partout dans le chemin réseau amd64 ; passer tous les tris en `sort.SliceStable` avec clé de départage explicite ; vérifier que le seuil « 12 meilleurs » est indépendant de l'ordre (piège #6).
- **Seuil de décision** : tant que ce test n'est pas vert à 100 % sur ≥10 000 positions, ne pas toucher à l'ordonnancement.

**Étape 1 — Remplacer le tourniquet statique par un compteur atomique + LPT.**
- Trier les 21 lancers par coût décroissant (coût = nombre de coups légaux générés, connu après génération ; les doubles en tête).
- Chaque worker pioche via `atomic.AddInt64`. Résultats écrits en emplacements fixes `results[idx]`. Somme sérielle inchangée.
- **Bénéfice attendu** : passer d'un speedup plafonné par le déséquilibre à un makespan à ~4/3 de l'optimum. **Seuil** : si le p95 du temps de niveau ne baisse pas de ≥20 %, profiler la barrière et la bande passante.

**Étape 2 — Aplatir la frontière pour supprimer les barrières par candidat.**
- Collecter tous les sous-arbres à approfondir de tout le niveau en une file unique triée LPT ; un seul `WaitGroup.Wait()` par niveau au lieu d'une barrière par candidat.
- **Bénéfice** : plus de tâches (≈60+), meilleur équilibrage, moins de synchronisation. **Seuil de décision** : mesurer via `runtime/trace` le temps passé en barrière ; s'il tombe sous ~2 % du temps total, arrêter d'optimiser la synchro.

**Étape 3 — Batching GEMM au sein du worker.**
- Regrouper les 12 candidats d'un lancer en un batch GEMM (XNNPACK, BSD-3-Clause). Garder le batch entièrement dans un worker (déterminisme + amortissement mémoire).
- **Seuil** : comparer batch GEMM vs 12 GEMV sur p75/p95 ; adopter si gain ≥15 % sans régression bit-exacte.

**Étape 4 — Cache partagé (optionnel, seulement si le hit-rate le justifie).**
- D'abord **mesurer** le taux de hit d'un cache par worker vs un cache partagé simulé, hors ligne. Si le gain de hit partagé est <5 points, **garder le cache par worker** (plus simple, trivialement déterministe).
- Si vous partagez : entrée alignée 64 o, mots en `atomic.Uint64`, tag Hyatt (XOR clé⊕données) pour détecter les torn entries → miss. Interdire toute profondeur variable (piège #11). Garder `go test -race` vert.
- **Seuil de décision** : n'activer le cache partagé que si (hit-rate ↑ ≥5 pts) ET (test bit-exact vert) ET (`-race` vert).

**Étape 5 — Mesure et communication honnêtes.**
- Rapporter p75/p95/p99, jamais des moyennes. Fixer `GOMAXPROCS` = cœurs physiques. Documenter turbo/gouverneur.
- Distinguer explicitement « ×N sur 8 cœurs physiques » de « ×N sur 16 threads SMT ».

---

## Ordre de grandeur du facteur d'accélération sur 8 cœurs physiques

Raisonnement (loi d'Amdahl appliquée au makespan) :
- **Fraction série** : la somme pondérée des 21 termes + la sélection finale est négligeable (<1 % du travail, ~21 additions vs ~527k MAC × dizaines d'évaluations). Amdahl pur autoriserait donc ≈ ×8. [EXTRAPOLÉ]
- **Borne de makespan (déséquilibre)** : avec 21 tâches très inégales et LPT, le makespan est ≤ 4/3 de l'optimum ; l'aplatissement de la frontière (≈60 tâches) réduit ce facteur vers ~1,1-1,2. Cela plafonne à ~×6,5-7 avant autres effets. [EXTRAPOLÉ]
- **Bande passante mémoire** : 8 cœurs faisant du GEMM 527k MAC peuvent saturer la bande passante ; selon la machine, cela retire 10-25 %. [HYPOTHÈSE]
- **SMT** : quasi nul pour du GEMM/FMA vectorisé (unités partagées). Donc « 16 threads » ≠ 16× ; le plafond réel est fixé par les **8 cœurs physiques**. [MESURE générale + ÉDITEUR/DÉCLARÉ XG ~1,3×/cœur HT]

**Conclusion chiffrée** : sur 8 cœurs physiques, un objectif réaliste après Étapes 1-3 est **×5 à ×6,5**. Votre **×4 « sur 16 threads » est presque certainement déjà proche du plafond si la machine n'a que 8 cœurs physiques** : les 16 threads SMT n'ajoutent quasiment rien sur du GEMM, et le déséquilibre statique actuel (doubles concentrés) vous coûte le reste. Le gain principal à espérer vient de l'équilibrage dynamique (Étape 1) et de la suppression des barrières (Étape 2), pas du SMT. Passer de ×4 à ×5,5-6 sur 8 cœurs physiques serait un excellent résultat et vous rapprocherait du plafond structurel. [EXTRAPOLÉ]

---

## Caveats
- Les coûts de primitives Go (ns) proviennent de benchmarks publics sur des machines variées ; **mesurez sur votre matériel cible** (`benchstat`).
- Les détails internes du cache gnubg (partage/verrou) n'ont pas pu être confirmés dans le code ; traitez-les comme indicatifs jusqu'à inspection de `lib/cache.c`.
- Le facteur d'accélération ×5-6,5 est une extrapolation fondée sur les bornes théoriques (LPT, Amdahl-makespan) et les propriétés connues du SMT sur GEMM, pas une mesure sur votre code.
- Licences des briques recommandées : **XNNPACK = BSD-3-Clause** (compatible distribution WASM, copyleft exclu OK) ; `golang.org/x/sync/errgroup` = BSD-3-Clause ; `sourcegraph/conc` = MIT ; toute brique GPL/AGPL (gnubg inclus) reste hors périmètre sauf comme oracle de mesure externe non redistribué.
- Relaxed-SIMD WASM : rappel ferme — la garde bit-exacte **ne survit pas** à relaxed-SIMD ; restez sur SIMD128.