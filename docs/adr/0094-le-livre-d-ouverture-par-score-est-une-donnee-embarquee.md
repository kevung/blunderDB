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
   Lines gzippé embarqué par `go:embed`, une ligne (`Line`) par position × score — position
   d'ouverture, ou position de réponse (voir l'amendement) : ouverture, coup d'ouverture,
   réponse, score, XGID, version du moteur, signature et graine du
   rollout, puis chaque candidat (coup en notation exacte, équité, σ, parties, JSD), meilleur
   en tête. Aucune table de base, aucun `DatabaseVersion` : le livre est une donnée du binaire,
   identique pour tous, reproductible par sa graine — comme `met_kazaross_xg2.json`. Le fichier
   embarqué est vide tant que le calcul n'a pas eu lieu ; `Default()` rend alors un livre vide.
2. **La clé est (ouverture, réponse, score)** — *amendée ci-dessous : (ouverture, coup
   d'ouverture, réponse, score), et une ligne d'ouverture sans réponse.* Le score s'écrit en scores *away* du point de
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

## Amendement — les deux premiers coups, et le réglage du rollout

L'exercice « quel coup à ce score ? » pose **les deux premiers jets** d'une partie au même
score : l'utilisateur joue l'ouverture, puis la réponse dans la position qu'elle laisse. Une
table d'ouvertures fixe (`OpeningPlays`) ne tient plus : le meilleur coup d'ouverture dépend du
score, et la réponse doit suivre celui que le livre retient *à ce score*.

1. **Deux sortes de lignes, une seule clé.** `Key` = (ouverture, coup d'ouverture, réponse,
   score). Une **ligne d'ouverture** a la réponse et le coup vides : les 15 jets non doubles ×
   les scores, l'ouvreur (Noir) au trait. Une **ligne de réponse** nomme le coup d'ouverture
   qu'elle suit — un candidat de la ligne d'ouverture du même score, dans sa notation exacte —
   et le jet de réponse. La table fixe disparaît.
2. **Deux phases, toujours reprenables.** `openingbook-gen run` roule la phase 1
   (`OpeningJobs`, 15 × scores) puis la phase 2 (`ReplyJobs`), lue dans le fichier de lot : les
   21 réponses après le meilleur coup d'ouverture de chaque score **et après chaque coup à
   égalité avec lui**, au plus `--after` coups (2 par défaut ; 1 = le meilleur seul). Couvrir
   un coup à égalité coûte 21 jobs par (ouverture, score) : au pire `--after 2` double la
   phase 2 (12 600 jobs au lieu de 6 300 sur la grille de 20 scores) ; le surcoût réel est la
   part des ouvertures dont le second coup est à égalité, connue après la phase 1
   (`--dry-run` la compte). Une réponse jouée après un coup à égalité que le livre ne suit pas
   se pose après le meilleur, et l'écran le dit.
3. **Un sous-ensemble se choisit** : `--openings`, `--replies`, `--scores` restreignent les
   jobs (vérification d'un échantillon contre un autre moteur, premières lignes) ; `--limit`
   coupe chaque phase.
4. **Le réglage est un paramètre, et la signature de chaque ligne le porte.** `--preset`
   (`fast`, `standard`, `full`) puis `--ply`, `--truncation`, `--min-games`, `--max-games`,
   `--jsd`, `--candidates` le surchargent ; l'exercice ne lit que les lignes. Deux usages :
   - **v1 rapide, `--preset fast`** (défaut) : tronqué à 7 demi-coups, 108–216 parties, arrêt
     JSD 3, 0 pli, 4 candidats — environ 10 h sur 28 fils pour la grille. Le livre est
     imprécis ; la règle d'égalité par σ(diff) élargit d'elle-même le groupe de tête, et
     l'écran affiche le nombre de parties et la signature sous l'analyse.
   - **version corrective, `--preset full`** (`FullSettings`) : 2 plis, non tronqué, cubeful,
     324–1296 parties, arrêt JSD 3, 4 candidats ; calcul de fond, après vérification d'un
     échantillon contre gnubg 2 plis. Coût mesuré : un job d'ouverture à 36 parties × 4
     candidats, un fil, n'a pas fini en 20 min ; soit plus de 8 s par partie-candidat, plus de
     12 h CPU par job à 1296 parties sans arrêt JSD, et de l'ordre de 150 000 h CPU (≈ 7 mois
     sur 28 fils) pour 300 + 12 600 jobs — une borne basse, que l'arrêt JSD réduit.
   La marge de 0,005 de la règle d'égalité (règle 4) mesure le biais du réglage tronqué ; elle
   reste pour toutes les lignes tant que le livre corrigé n'a pas mesuré le sien.
5. **Priorité** : nice 19 par défaut, et `--idle` passe chaque fil en `SCHED_IDLE` (Linux).
6. **L'exercice** est une fonction pure du livre (`Book.Draw`, `Book.Grade`), appelée par le
   panneau Entraînement (`*gui.App`), la commande `openingbook` et les routes
   `/v1/openingbook.*` : les trois modes tirent et jugent pareil. Une question exige les deux
   lignes (l'ouverture au score, la réponse après son meilleur coup) ; un livre vide est un état,
   pas une erreur : l'exercice le dit et ne démarre pas.
