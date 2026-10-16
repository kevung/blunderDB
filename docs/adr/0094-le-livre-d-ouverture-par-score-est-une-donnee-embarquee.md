# Le livre d'ouverture par score est une donnée embarquée, produite par lots reprenables

Status: accepted.
See also: ADR-0019 (une seule échelle d'équité), ADR-0060 (rollouts), ADR-0093 (budget de cœurs), `tasks/etude-livre-ouverture-scores.md`

## Context

L'étude du livre des réponses au premier coup selon le score conclut : rollouts blunderDB
*Standard* (cubeful, 0 pli, 1296 parties, troncature 11, arrêt JSD 3), 315 positions de réponse
× ~20 scores × ~4 candidats, ≈ 46 jours CPU. Aucune table externe par score n'existe ; celles
de bkgm.com (money, sans licence) ne s'embarquent pas. Il reste à dire où vivent les données,
comment on les lit, et comment on les produit sans refaire trois jours de calcul à chaque
interruption.

## Decision

1. **Le livre est un fichier, pas un schéma.** `pkg/blunderdb/openingbook/book.jsonl.gz`, JSON
   Lines gzippé embarqué par `go:embed`, une ligne (`Line`) par position de réponse × score :
   ouverture, coup d'ouverture, réponse, score, XGID, version du moteur, signature et graine du
   rollout, puis chaque candidat (coup en notation exacte, équité, σ, parties, JSD), meilleur
   en tête. Aucune table de base, aucun `DatabaseVersion` : le livre est une donnée du binaire,
   identique pour tous, reproductible par sa graine — comme `met_kazaross_xg2.json`. Le fichier
   embarqué est vide tant que le calcul n'a pas eu lieu ; `Default()` rend alors un livre vide.
2. **La clé est (ouverture, réponse, score).** Le score s'écrit en scores *away* du point de
   vue du répondeur (au trait) puis de l'ouvreur : `4-2`, `1-4` (partie Crawford, la sentinelle
   1 de CONTEXT.md), `money`. Les 15 coups d'ouverture sont une table fixe (`OpeningPlays`) :
   la changer change les positions et appelle un nouveau livre.
3. **Une seule échelle sort** (ADR-0019) : équité normalisée au score, du côté du répondeur.
   Le livre montre des **écarts**, pas des équités absolues (`CubefulBias`).
4. **Écart et égalité** (`Line.Rank`) : écart = équité du meilleur − celle du coup ;
   σ(diff) = √(σ²meilleur + σ²coup), les deux estimations prises indépendantes comme le fait la
   JSD du rollout (les dés communs les corrèlent positivement : c'est prudent) ; IC 95 % =
   1,96·σ(diff) ; **à égalité** si écart < 1,96·σ(diff) + 0,005, le 0,005 étant le biais mesuré
   du réglage tronqué. Tout coup à égalité avec le meilleur est une bonne réponse.
5. **Les candidats** d'une ligne sont les 4 meilleurs à 2 plis *à ce score* (choix du rollout
   sans coup nommé), nommés par le rollout en notation exacte : deux coups ne partagent jamais
   un nom.
6. **La production est un outil de mainteneur**, `cmd/openingbook-gen`, pas une commande de
   blunderDB : l'utilisateur reçoit la donnée, il ne la calcule pas, donc pas de parité
   CLI/GUI/serveur à tenir. `run` ajoute chaque job fini au fichier de lot, synchronisé ;
   relancé, il saute ce qui y est, coupe une dernière ligne déchirée, ne réécrit jamais un job
   à moitié. Un job tourne sur un fil ; `--threads` jobs à la fois, GOMAXPROCS borné à autant
   (le choix des candidats à 2 plis forke dans ce budget, ADR-0093) ; priorité nice 19 par
   défaut. La table de bearoff exacte est coupée : à onze demi-coups de l'ouverture elle
   n'est jamais atteinte, et le livre ne dépend pas de ce qu'une machine a généré. `compile`
   trie le lot dans le fichier embarqué.

## Consequences

- Le livre complet (grille de 20 scores) pèse quelques centaines de ko gzippés dans le binaire.
- Les nombres d'une ligne ne dépendent ni de `--threads` ni de l'ordre des jobs.
- La signature imprimée reste celle des réglages du rollout et mentionne le bearoff exact
  « when covered » ; il ne l'est jamais ici.
- Un autre jeu de scores (`--scores full`, 48) ou une autre table d'ouvertures ajoute des
  lignes sans changer le format.
