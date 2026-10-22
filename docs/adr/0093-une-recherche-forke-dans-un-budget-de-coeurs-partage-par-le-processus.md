# Une recherche forke dans un budget de cœurs partagé par le processus

Status: accepted.
See also: ADR-0011, ADR-0024 (remplace sa décision 5), gammonNet `docs/specs/recherche-parallele-spec.md` (v1.8.0)

## Context

Jusqu'ici, deux régimes ne devaient jamais s'empiler : le lot d'analyse menait une recherche
sérielle par goroutine (`NumCPU` goroutines), le panneau menait une recherche dont seule la
racine se répartissait sur un jeu fixe d'ouvriers (`WithWorkers`). D'où trois pertes : un lot
plus court que la machine (l'analyse d'un match de cinq positions) laissait les autres cœurs
inactifs, comme la queue de tout lot ; une requête seule du démon `serve` tournait sur un cœur ;
à 3 et 4 plis, la file de la racine (souvent 21 à 42 tâches de plusieurs secondes) finissait
sur la plus lente. gammonNet v1.8.0 a découpé la recherche à tous les niveaux, au bit près, et
sa spec en donne la transposition en Go.

## Decision

1. **Quatre boucles se découpent, à partir de `depth >= 2`** (`parallel.go`) : la file aplatie
   de la racine, la passe profonde d'un classement sous la racine (un candidat par tâche), le
   nœud de hasard (un jet par tâche, dans l'ordre LPT), les backups de distribution de
   `probsAt`. La passe peu profonde d'un nœud reste sur son fil : elle remplit les lots.
2. **Toute somme se prend après la barrière, dans l'ordre fixe** (index de jet croissant, tri
   sur le fil appelant) ; chaque tâche n'écrit que sa case. Le nombre de cœurs ne change ni un
   classement, ni une équité, ni une probabilité.
3. **Un budget de jetons, jamais attendu par un fork.** `fork` prête une tâche à un auxiliaire
   seulement si un jeton est libre, et l'exécute lui-même sinon : l'emboîtement ne peut pas
   s'interbloquer. `ProcessCores()` (GOMAXPROCS jetons) est le budget de tout le processus ;
   une goroutine qui cherche seule de son côté (une position d'un lot, une cellule de la
   matrice) en tient un (`HoldCore`). Rien d'empilé ne demande plus de GOMAXPROCS goroutines,
   et ce qu'un lot laisse libre sert aux recherches en cours.
4. **`CoresFor(ply)`** : le budget du processus à partir de 2 plis, série en dessous (seuil
   mesuré, ADR-0011). `NewBatchSearcher` et le panneau le prennent ; `WithWorkers(n)` reste
   un budget privé pour les mesures et les tests.
5. **L'équipe partage le cache**, derrière 4096 verrous par tranche de cases
   (`gn_evalcache.c`) : un succès rend ce que le réseau aurait calculé, seul le compte
   d'évaluations bouge. Les compteurs des auxiliaires sont rapatriés au fil qui ouvre le fork.
6. **`serve` garde sa limite.** Le pool d'analyse a son budget de `--analysis-workers`
   jetons ; une unité est facturée au quota du tenant pour son temps écoulé plus le temps
   prêté aux auxiliaires (`Searcher.LentTime`), si bien qu'étaler une recherche ne fait pas
   payer moins de CPU.
7. **`EngineVersion` reste `gammonNet v1.7.0`** : v1.8.0 ne change aucune valeur (mêmes poids,
   même recherche), donc aucune analyse stockée ne devient périmée. *Remplacé par ADR-0104.*

## Consequences

Mesures A/B alternées contre l'état précédent, 14 cœurs / 28 fils, machine chargée par
d'autres calculs : une décision 2-ply 102–119 ms contre 108–143 ms ; ply3 605–678 ms contre
746–817 ms (×12,4–13,5 contre ×10,9 au même temps série) ; un lot de cinq positions ×3,2 à 2 plis
comme à ply3 contre les mêmes recherches sérielles. −4 % d'évaluations à 2 plis, −8 % à ply3
(cache partagé). Le temps série du port Go n'est pas celui du C : seul le facteur se reprend.

## Guard

`TestParallelSearchFingerprint` (empreinte des classements à 1, 6 et NumCPU fils : normal,
thorough, ply3, `Probs` à 3 plis, match + videau à 2 et 3 plis ; une somme réordonnée y échoue
au dernier bit), `TestSharedBudgetSearchesAgree`, `TestSearchMatchesTheGoldFile`
(`BLUNDERDB_GOLD`, recherche parallèle contre l'or du C), `TestUnitCostCountsLentTime`.
Mesures : `TestProbeMulticoreLevels`, `TestProbeShortBatch` (`BLUNDERDB_PROBE`).
