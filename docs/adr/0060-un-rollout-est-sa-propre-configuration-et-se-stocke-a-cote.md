# ADR-0060 — Un rollout est sa propre Configuration, et se stockera à côté de l'Analysis

Statut : acceptée.
Voir aussi : ADR-0011, ADR-0013, ADR-0019, ADR-0024, ADR-0029 ; `docs/recherche/P8-rollouts.md`.

## Contexte

Deux coups à 0,005 ne se départagent pas par une recherche : c'est la principale raison de
retourner dans XG. gammonNet n'a pas de rollout ; le dépôt a ce qu'il faut pour en écrire un
(`Searcher` réutilisable, `Decide`, `EquityScale`, la base de sortie two-sided, le parallélisme
déterministe d'ADR-0024). La recette est fixée par P8 : réduction de variance, dés communs,
quasi-aléatoire, troncature exacte en bearoff, graine par partie, arrêt sur la JSD. Le risque
propre à un rollout est le biais : un rollout cubeful ou tronqué réintroduit l'erreur du modèle
de videau, et ses intervalles sont optimistes.

## Décision

1. **Écrit ici, porté ensuite.** `pkg/blunderdb/engine/rollout`, au-dessus du `Searcher`, sans
   toucher à l'arithmétique de `engine/gammonnet`. Le portage dans gammonNet est consigné dans
   `tasks/plan-amelioration-2026-09b/AMONT-GAMMONNET.md` ; jusque-là ce paquet est la référence.
2. **Un rollout est sa propre Configuration** (CONTEXT.md) : `rollout.EngineVersion`
   (`blunderDB rollout v1 / gammonNet v1.2.1`) plus ses paramètres. Tout changement de procédure
   qui déplace un nombre (dés, réduction de variance, politique de videau, valeur des feuilles)
   incrémente `v1`. Les paramètres sont ceux de `Settings` : troncature (demi-coups, 0 = jusqu'au
   bout), parties min/max (multiples de 36, sans quoi le premier lancer n'est plus stratifié), seuil JSD, ply, nombre de candidats, graine. `Signature()` en donne la
   ligne complète, `DepthLabel()` l'`AnalysisDepth` (« Rollout 216 games (0-ply, truncated 7) »,
   classé au-dessus de tout ply par `domain.AnalysisDepthRank`).
3. **Trois réglages.** *Rapide* : tronqué à 7, 216 parties, min 108, JSD ≥ 3. *Standard* : 1296
   parties, min 324, JSD ≥ 3, tronqué à 11 comme le défaut de gnubg. *Libre* : tout paramètre.
   Les deux préréglages jouent à **0 ply** (les candidats, eux, sont choisis à 2 ply au moins) : une décision 2-ply coûte environ 200 ms (mesure
   d'AMONT-GAMMONNET §1), soit des heures pour 1296 parties ; 1 ply triple le temps de *Rapide*
   (mesuré : 35 s contre 11 s, ouverture 3-1, 5 candidats, 16 cœurs) pour un écart d'au plus 0,004 sur
   les équités. Monter le ply reste un réglage *Libre*.
4. **Coups et videau dès v1, toujours cubeful.** Avec dés : les `Candidates` meilleurs coups à
   `Ply` (ou ceux nommés), chacun joué depuis la position qui en résulte. Sans dés : deux
   branches, *Pas de double* (le joueur au trait a déjà décliné ce tour) et *Double/Prend*
   (videau doublé, chez l'adversaire) ; *Double/Passe* vaut +1 exactement et ne se joue pas.
   Dans une partie, le videau est offert, pris ou passé par `gammonnet.Decide` à `Ply` ; une
   feuille tronquée vaut la valeur de cette décision. Le résultat porte `CubefulBias` dès que le
   modèle de videau intervient : **le classement est fiable, l'équité absolue moins** (P8 §2, §5).
   Pas de beaver (Decide n'en a pas) ; Jacoby compte en money tant que le videau est centré —
   dans la décision de videau, la feuille et l'enjeu final ; `SearchConfig` n'ayant pas de règle
   Jacoby, les feuilles internes d'une recherche l'ignorent, comme dans gammonNet.
5. **La recette P8, toujours active, sans interrupteur.**
   - *Réduction de variance 1-ply* : avant chaque lancer, la valeur du joueur au trait est la
     moyenne, sur les 21 lancers, du meilleur coup 0-ply ; la chance du lancer joué est sa
     valeur moins cette moyenne, et la somme des chances est retirée du résultat. Son espérance
     est nulle exactement, quelles que soient les erreurs du réseau, parce que la moyenne porte
     sur la fonction même évaluée au lancer joué (le piège de Zare).
   - *Dés communs* : la partie n de chaque candidat joue les mêmes dés.
   - *Quasi-aléatoire* : par bloc de 1296 parties, le premier lancer de la partie u est
     `p1[u mod 36]`, le second `p2[(u/36 + u mod 36) mod 36]` (carré latin cyclique), les
     permutations retirées par bloc ; tout multiple de 36 parties voit chaque premier et chaque
     second lancer autant de fois, 1296 voient chaque paire une fois.
   - *Troncature exacte* : une partie s'arrête dès que la base two-sided couvre la sortie et que
     chaque camp a sorti un pion (la base est sans gammon). En money, ses équités cubeful
     donnent la valeur ; à un score, sa probabilité exacte passe par le modèle de videau.
   - *Graine par partie et somme ordonnée* : les dés de la partie n sont une fonction de
     `(graine, n)` ; l'unité de travail est (partie, candidat) ; les résultats d'un lot de 36
     parties sont sommés dans l'ordre des parties après le lot. Le résultat est identique bit à
     bit quel que soit le nombre de workers (`TestReproducibleAcrossWorkers`, 1 et 7). Une base
     two-sided présente ou absente change les nombres : `ExactBearoff` le dit.
6. **Sortie.** Par candidat : équité moyenne, σ (erreur-type, comme gnubg et XG), IC 95 %
   (1,96 σ), parties jouées, JSD contre le meilleur — `(Δ équité) / √(σ₁² + σ₂²)`, sans terme de
   covariance, donc conservatrice avec des dés communs. **Arrêt anticipé** après `MinGames` : un
   coup dont la JSD atteint le seuil cesse d'être joué, le rollout s'arrête quand il n'en reste
   qu'un ; pour le videau, quand *Pas de double* contre *Double* et *Prend* contre *Passe* sont
   tous deux tranchés. Annulable par `context` (les lots achevés restent, le lot interrompu est
   jeté entier, donc le résultat partiel est lui aussi reproductible) ; progression rappelée
   après chaque lot.
7. **Une échelle à la sortie** (ADR-0019) : points money par unité du videau de la position, ou
   équité normalisée au score. Une partie se compte en points money ou en chances de gagner le
   match de la racine ; les deux sont affines dans l'équité de sortie, donc moyenne, σ et JSD se
   convertissent une seule fois, à la fin. Aucune échelle interne ne sort.
8. **Stockage : une seconde Analysis, à côté.** Un rollout s'écrit dans
   `PositionAnalysis.Rollouts` (`domain.RolloutAnalysis`) : `AnalysisEngine =
   rollout.EngineVersion`, `AnalysisDepth = DepthLabel()`, `Signature()`, les paramètres, et par
   candidat l'équité, σ, l'IC 95 %, les parties et la JSD. Il ne remplace **aucune** entrée de
   `CheckerAnalysis` ni de `DoublingCubeAnalysis` (ADR-0013) — y entrer l'aurait fait : la fusion
   par coup garde le rang de profondeur le plus haut, et « Rollout » passe au-dessus de « XG
   Roller++ ». Un rollout par `Signature` : relancé aux mêmes réglages, il garde la série la plus
   longue ; à d'autres réglages, il s'ajoute. Chaque fusion d'analyse (`SaveAnalysis`, l'import)
   reprend les rollouts existants, pour qu'un appelant qui les ignore ne les efface pas. Un
   rollout annulé n'est jamais écrit. Les colonnes indexées restent celles de l'analyse
   principale ; une position que seul un rollout analyse les tire de son meilleur rollout
   (`ColumnSource`), et la recherche la trouve. Tout vit dans le blob JSON : **aucune colonne,
   aucun bump de schéma**. Le rassemblement (positions d'une requête sans rollout de cette
   `Signature`, d'où la reprise), la boucle et l'écriture sont `pkg/blunderdb/rollouts`, sur le
   contrat de stockage, partagés par la GUI et la CLI (via `Database`) et par `serve`.

## Jauge

Le corpus `testdata` ne contient aucun rollout complet XG ; il contient des décisions analysées
en **XG Roller++** (360 parties tronquées avec réduction de variance), c'est-à-dire le rollout
tronqué d'XG. Sur les 30 premières décisions de coups de `testdata/test.xg` dont les deux
meilleurs coups sont en XG Roller++ et à moins de 0,06 l'un de l'autre, le préréglage *Rapide*
(deux candidats, sans base two-sided) mesure :

| | rollout *Rapide* | gammonNet 2-ply |
|---|---|---|
| écart moyen à l'écart XG entre les deux coups | **0,0066** (max 0,032) | 0,0146 |
| même ordre qu'XG quand XG les sépare de ≥ 0,01 (15 cas) | **15/15** | 14/15 |
| ordre d'XG quand la JSD atteint 3 (14 cas) | **14/14** | — |
| écart moyen à l'équité XG du meilleur coup | 0,0098 (max 0,057) | — |

Le cas `-CCDaB-----a--aab--bcbBbA-:1:1:-1:32:1:0:0:5:0` (partie 4, coup 129) est le départage :
2-ply préfère 11/8 10/8 de 0,060, XG Roller++ préfère 21/19 14/11 de 0,024, le rollout aussi
(+0,024, JSD 3,5). `TestRolloutOverturnsTheSearch` garde ce cas, hors `-short` et `-race`. `TestGaugeAgainstXGRollouts` relit le corpus et roule
30 décisions — celles dont la notation XG coïncide avec un coup légal de blunderDB, un ensemble
voisin du tableau — : écart moyen mesuré 0,0076, seuil 0,015 (entre le rollout et le 2-ply), et
l'ordre d'XG exigé dès qu'XG sépare les coups de 0,01 ou que la JSD atteint 3. L'écart n'est pas nul et ne doit pas l'être : deux
réseaux, deux politiques de videau, deux troncatures.

## Conséquences

- Coût mesuré (16 cœurs, 0 ply) : *Rapide* sur l'ouverture 3-1, cinq coups, 11 s ; deux coups,
  3 à 4 s. Le temps est celui du réseau (la réduction de variance évalue les 21 lancers à chaque
  demi-coup) ; il baissera avec gammonNet, pas ici.
- `blunderdb rollout` joue une position donnée ou une position d'une base (`--store` l'écrit) ;
  `analyze --rollout` joue les positions d'une requête ; `serve` expose `rollout.position`
  (une position de la bibliothèque, jamais une position nue : ADR-0015) et `rollout.filter` ;
  l'outil MCP `rollout` lit, et n'écrit qu'avec le drapeau d'écriture.
- Rejeté : un rollout non cubeful par défaut — une décision de videau en a besoin, et un seul
  mode garde une seule Configuration. Rejeté : un interrupteur de réduction de variance — la
  recette est la Configuration. Rejeté : arrêter sur l'IC d'un candidat — c'est l'écart-type de
  la différence qui tranche (P8 §3).
