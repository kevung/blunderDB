# Un calcul visible réquisitionne les cœurs du lot à son prochain jet

Statut : acceptée. Amende ADR-0115 §2 (« la priorité décide qui tourne, pas sur combien de
cœurs ») et sa conséquence « le lot cède par son seul point de contrôle ».
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
2. **Le lot cède à l'intérieur de sa recherche.** Un chercheur de lot (`Searcher.YieldsTo`)
   lit, à chaque jet, un booléen atomique ; sous réquisition, le goroutine se gare et prête son
   jeton au réquisitionnaire : ses fourches et ses `Hold` le trouvent libre sans attendre. La
   pile du goroutine garde la recherche entière, rien d'achevé n'est perdu ni recalculé, et la
   réponse reste la même au bit près : se garer ne fait que retarder le jet suivant.
3. **Le jeton prêté reste compté au goroutine garé** : à la levée il repart sans faire la
   queue. Un auxiliaire garé ne peut donc jamais attendre un jeton que tient la fourche qui
   l'attend. Une position neuve du lot (`Cores.HoldBackground`) attend, elle, la levée.
4. **Pas de famine indéfinie** : un calcul visible est borné et le plus récent remplace le
   précédent (ADR-0115 §2) ; le lot reprend dès la levée de la dernière réquisition.

Écarté : une file de jetons à deux priorités seule — le lot tient ses jetons pour toute une
position, une file ne servirait le calcul visible qu'à la fin de celle-ci ; annuler et relancer
la position en cours — perd le travail fait ; céder à chaque nœud plutôt qu'à chaque jet — la
latence d'un jet à 2 plis (quelques millisecondes) suffit, pour une lecture de plus par nœud.

## Conséquences

- Le démon et la CLI ne réquisitionnent jamais : leur ordonnancement est inchangé.
- Annuler un lot garé ne le réveille qu'à la levée de la réquisition, au plus la durée d'un
  calcul visible.
- `InUse` compte un jeton prêté une fois, pour celui qui l'utilise.
- Mesuré (28 cœurs logiques, avx2, lot 2-ply sur tous les cœurs, décision 2-ply d'ouverture) :
  sans réquisition, 1,3 s médian (décision répartie) et 1,5 s médian, 6,3 s au pire (case qui
  prend un jeton) ; avec, 71 ms et 67 ms médians, 92 ms au pire.

Garde : `engine/gammonnet` `TestCoresClaimLendsTheBackgroundToken`,
`TestCoresClaimParksABatchSearchWithoutChangingIt` ; mesure :
`TestProbeVisibleLatencyUnderBatch` (`BLUNDERDB_PROBE=1`).
