# Un calcul visible réquisitionne les cœurs du lot à son prochain jet

Statut : acceptée. Précise ADR-0115 §2 et sa conséquence sur la cession du lot, réécrits en
conséquence : la priorité décide aussi sur combien de cœurs tourne le lot.
Voir aussi : ADR-0093 (budget de cœurs), ADR-0116.

## Contexte

Le lot gammonNet tient un jeton du budget du processus (`gammonnet.ProcessCores`) par position
et ne cédait à un calcul visible qu'à son point de contrôle, avant la position suivante. Une
position à 2 plis dure de l'ordre de la seconde : pendant ce temps, l'évaluation au repos ne
trouvait aucun jeton où répartir sa recherche et tournait seule contre tout le lot, et une case
de la carte des scores, qui prend un jeton (`HoldCore`), attendait dans l'ordre d'arrivée qu'une
position du lot s'achève. L'utilisateur attend d'un calcul affiché une réponse quasi immédiate,
lot en cours ou non.

## Décision

1. **Un calcul visible réquisitionne le budget** (`Cores.Claim`), de son entrée à sa sortie,
   en même temps qu'il entre au registre (`beginVisibleComputation`). Les réquisitions
   s'empilent ; la dernière levée rend la main.
2. **Le lot cède à l'intérieur de sa recherche.** Un chercheur de lot (`Searcher.YieldsTo`,
   le partagé comme celui de repli construit par position)
   lit, à chaque jet, un booléen atomique ; sous réquisition, le goroutine se gare et prête son
   jeton au réquisitionnaire : ses fourches et ses `Hold` le trouvent libre sans attendre. La
   pile du goroutine garde la recherche entière, rien d'achevé n'est perdu ni recalculé, et la
   réponse reste la même au bit près : se garer ne fait que retarder le jet suivant.
3. **Le jeton prêté reste compté au goroutine garé** : à la reprise il n'en reprend aucun, donc
   un auxiliaire garé n'attend jamais un jeton que tient la fourche qui l'attend. La reprise a
   lieu, pour tous les garés ensemble, une fois la dernière réquisition levée *et* les jetons
   pris en plus par les réquisitionnaires rendus : le budget n'est jamais dépassé.
4. **Le fond ne prend aucun jeton** tant qu'une réquisition est ouverte ou que des garés
   attendent leur reprise : ni position neuve (`Cores.HoldBackground`), ni auxiliaire d'une
   fourche du lot. Le fond seul ne tient donc jamais plus que le budget, et la reprise n'attend
   que les réquisitionnaires, qui n'attendent jamais le fond.
5. **L'annulation passe la réquisition** : un goroutine garé se réveille aussi à la fin du
   contexte lié par `BindContext`, et `HoldBackground` prend ce contexte ; le lot annulé se
   déroule aussitôt, sur le seul cœur qu'il tenait.
6. **Famine bornée par l'utilisateur** : le lot attend tant qu'un calcul visible tourne. Chacun
   est borné et le plus récent remplace le précédent (ADR-0115 §2), si bien que le lot n'attend
   qu'autant que l'utilisateur enchaîne les calculs affichés ; il reprend dès la levée.

Écarté : une file de jetons à deux priorités seule — le lot tient ses jetons pour toute une
position, une file ne servirait le calcul visible qu'à la fin de celle-ci ; annuler et relancer
la position en cours — perd le travail fait ; céder à chaque nœud plutôt qu'à chaque jet — la
latence d'un jet à 2 plis (quelques millisecondes) suffit, pour une lecture de plus par nœud.

## Conséquences

- Le démon et la CLI ne réquisitionnent jamais : leur ordonnancement est inchangé.
- Arrêter les tâches d'une bibliothèque (changement, fermeture, arrêt de l'application)
  arrête un lot garé sans attendre la fin du calcul visible.
- Le parent d'une fourche qui attend ses auxiliaires garés ne prête pas son jeton : le calcul
  visible dispose alors d'un cœur de moins que le budget.
- `InUse` compte un jeton prêté une fois, pour celui qui l'utilise.
- Mesuré (28 cœurs logiques, avx2, lot 2-ply sur tous les cœurs, décision 2-ply d'ouverture) :
  sans réquisition, 1,3 s médian (décision répartie) et 1,5 s médian, 6,3 s au pire (case qui
  prend un jeton) ; avec, 71 ms et 67 ms médians, 92 ms au pire.

Garde : `engine/gammonnet` `TestCoresClaimLendsTheBackgroundToken`,
`TestCoresStackedClaimsResumeAfterTheLast`, `TestCoresResumeWaitsForTheClaimantsTokens`,
`TestCoresCancelReachesTheParkedBackground`, `TestCoresClaimParksABatchSearchWithoutChangingIt`,
`TestCoresRepeatedClaimsMidForkLeaveTheAnswer` ; `internal/gui`
`TestStopGroupJobs_StopsABatchParkedBehindAVisibleComputation` ; mesure :
`TestProbeVisibleLatencyUnderBatch` (`BLUNDERDB_PROBE=1`).
