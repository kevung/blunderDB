# Les calculs sans état sur une position nue sont admis sous quota et à deux plis au plus

Statut : acceptée. Amende l'ADR-0015.
Voir aussi : ADR-0013 (évaluation au repos et rattrapage), ADR-0068 (table d'équités de la bibliothèque).

## Contexte

L'ADR-0015 refuse à `serve` tout point d'entrée qui évalue une position hors d'une
bibliothèque : gammonGo a `gammonnet serve` pour cela, et deux entrées vers un même moteur
divergent. La règle n'a pas tenu : `gammonnet.cubeMatrix` puis `gammonnet.evaluate` sont
entrés, parce que la parité CLI/GUI/serveur (CLAUDE.md) les exige — le panneau Éval et la
matrice du videau travaillent sur le plateau affiché, souvent non enregistré, et un client
HTTP n'a pas d'autre accès à ces réponses. La carte des jets (`gammonnet.rolls`) et la carte
des scores (`gammonnet.scoreMap`) sont du même genre. Ce que l'ADR-0015 voulait éviter, un
évaluateur concurrent de `gammonnet serve`, reste à éviter ; ce qu'il interdisait en fait,
c'est un calcul sans borne hors de la file et du compte du tenant.

## Décision

1. **Un calcul sans état sur une position nue est admis.** La requête porte la position
   (`position` ou `xgid`) ; la réponse ne dépend que d'elle et de la table d'équités du tenant
   (ADR-0068). Rien n'est lu d'autre dans le tenant, rien n'y est écrit. Sont dans ce cas :
   `gammonnet.evaluate`, `gammonnet.cubeMatrix`, `gammonnet.rolls`, `gammonnet.scoreMap`.
2. **Sous la file et le quota d'analyse.** Chaque recherche passe par le pool d'analyse
   partagé (`analysisPool`), à son tour parmi les tenants, et son temps est décompté du quota
   `--quota-analysis-seconds` ; le quota épuisé, la route répond 429 avant tout calcul.
3. **Deux plis au plus pour une position et pour une carte.** Une réponse rendue en ligne
   reste bornée (`gammonnet.ValidateMapPly`, et la même borne pour `evaluate`) ; une recherche
   plus profonde est le travail des balayages (`analyzeMissing`, `sweepStale`), qui écrivent
   dans la bibliothèque. La matrice du videau prend la même borne de profondeur ; sa longueur
   par défaut et ses bornes de longueur ne changent pas.
4. **Pas d'évaluateur général.** Ni probabilités brutes à profondeur libre, ni lot de positions
   arbitraires : une route nouvelle de ce genre répond à une question de l'interface, rend la
   même forme que la GUI et la CLI, et entre dans la liste du point 1 par un amendement.

## Conséquences

- L'ADR-0015 reste vrai pour ce qu'il garde : `serve` ne devient pas `gammonnet serve`.
- Le guide du mode serveur nomme ces routes dans la liste du quota et de la file d'analyse.
- Rejeté : retirer `evaluate` et `cubeMatrix` pour revenir à la lettre de l'ADR-0015 — la parité
  y perd les réponses que la GUI donne sur un plateau non enregistré.
- Rejeté : les admettre sans quota — un tenant monopoliserait les travailleurs partagés.

## Garde

`internal/server/quota_test.go` (`TestQuotaRefusesAnalysisOnceSpent` couvre les quatre
routes), `internal/server/handlers_gammonnet_rolls_test.go` et
`handlers_gammonnet_scoremap_test.go` et `handlers_gammonnet_cubematrix_test.go` (profondeur
au-delà de 2 refusée).
