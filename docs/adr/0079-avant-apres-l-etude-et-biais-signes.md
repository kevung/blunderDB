# ADR-0079 — Avant/après l'étude d'une famille, et biais signés

Statut : acceptée.
Voir aussi : ADR-0077 (plan d'étude, familles), ADR-0075 (perte MWC), ADR-0046 (seuils d'erreur),
ADR-0019 (échelle d'équité).

## Contexte

Le plan d'étude (ADR-0077) dit quoi travailler ; rien ne dit si ce qui a été travaillé a
changé quelque chose en match réel. `list --type study` pose côte à côte, par plan de jeu, un
nombre de positions révisées et deux PR, sans intervalle et sans lien avec ce qui a été
étudié. Et une perte, même bien classée, ne dit pas dans quel sens le joueur se trompe : « tu
prends trop » se corrige mieux qu'un PR. Les définitions, fenêtres, intervalles et seuils sont
fixés ici **avant** d'avoir regardé ce qu'ils donnent sur une base réelle.

## Décision

### Avant/après par famille étudiée

1. **Famille.** Celle de l'ADR-0077 : (plan de jeu, nature de la décision, thème), ses membres
   les erreurs thématisées du filtre.
2. **Date d'étude.** Une famille est étudiée à la date de la **première action d'étude** sur
   l'une des positions de ses membres : une marque « étudiée » (`study_mark.marked_at`), une
   révision Anki, quelle que soit la note (`anki_review_log.reviewed_at`), une réponse de
   quiz sur la position (`training_item`, à la date de sa séance). Créer une carte, mettre une
   position en collection ou l'ouvrir n'est pas une action d'étude : rien n'y est daté, ou
   rien n'y est travaillé. La date est le jour UTC de cette action.
3. **Fenêtres.** *Avant* : les décisions du filtre jouées dans des matchs datés strictement
   avant ce jour ; *après* : strictement après. Le jour même est écarté (on ne sait pas si le
   match précède l'étude), comme un match sans date. Les bornes de dates du filtre bornent les
   deux fenêtres.
4. **Mesure.** Dans chaque fenêtre, le taux de perte de la famille r = Σℓ / N : ℓ la perte de
   MWC (ADR-0075, même conversion que le plan) des erreurs de la famille jouées dans la
   fenêtre, N les décisions comptées **en match** (longueur > 0) du même plan de jeu et de la
   même nature dans la fenêtre. Le dénominateur ne dépend pas du thème, qui n'existe que pour
   une erreur. Le gain est Δ = r_avant − r_après (positif : moins de perte après).
5. **Intervalle.** Chaque fenêtre est un comptage de Poisson composé, comme le plan :
   Var(r) ≈ Σℓ² / N². Intervalle à 95 % : Δ ± 1,96·√(Σℓ²_avant/N²_avant + Σℓ²_après/N²_après).
   Analytique et déterministe : les deux moteurs de stockage rendent le même nombre.
6. **Preuve minimale et verdict.** Au moins **30 décisions** du plan et de la nature dans
   **chaque** fenêtre ; en deçà, le verdict est « insuffisant » et aucun sens n'est affirmé.
   Sinon : « en progrès » si la borne basse de Δ est strictement positive, « en recul » si la
   borne haute est strictement négative, « indéterminé » sinon. Aucune affirmation sans borne.
7. **Lecture.** Un changement, pas un effet. Une famille est étudiée *parce qu'elle* coûtait :
   la régression vers la moyenne gonfle le gain, et rien ne contrôle les adversaires, le
   format ni les dés. L'interface et la CLI le disent à côté du chiffre.

### Biais signés

8. **Estimateur commun.** Chaque décision comptée porte x ∈ {+1, 0, −1} : +1 une erreur dans le
   sens « trop » (audace), −1 une erreur dans le sens « pas assez » (prudence), 0 sinon. Le
   biais est B = (P − M)/N, P et M les deux comptes. Intervalle à 95 % par l'approximation
   normale de la moyenne : B ± 1,96·√(((P + M)/N − B²)/N). Au moins **20 décisions** ;
   verdict « trop » si la borne basse est > 0, « pas assez » si la borne haute est < 0,
   « équilibré » sinon, « insuffisant » sous le minimum. Les coûts de P et M (millipoints)
   accompagnent les comptes. Le signe et le coût d'une décision sont ceux du coup ou de
   l'action joués dans son match (`move.decision_error_mp`, comme toute statistique), jamais
   les colonnes de la position, qui ne notent qu'un des matchs qui l'ont atteinte.
9. **Prises et refus.** Décisions : les réponses au videau comptées (prise ou refus joués,
   verdict du bot lisible), la classification de `storage.ClassifyCubeDirection`.
   P = prises fautives (le bot refuse), M = refus fautifs (le bot prend). B est exactement le
   taux de prise du joueur moins celui du bot sur les mêmes positions.
10. **Doubles prématurés et manqués, par score.** Décisions : les décisions de videau du
    porteur comptées (double proposé, ou pas de double signalé serré), même classification.
    P = doubles prématurés (le bot ne double pas, ou est trop bon), M = doubles manqués. Le
    biais est rendu sur l'ensemble et par case de score (away du joueur au trait, away de
    l'adversaire, lus par `domain.PointsAway` : un score post-Crawford est à 1 point ; la
    partie libre a sa propre case, jamais confondue avec un score), chaque case avec le
    même estimateur et le même minimum.
11. **Audace et prudence sur les blots.** Décisions : les décisions de pions comptées dont la
    position a du contact. Une décision dont le coup joué ne coûte rien porte 0. Sinon le
    plateau après le coup joué et après le meilleur coup de l'analyse sont reconstruits (le
    générateur de coups légaux, comme `engine.ExplainChecker`), et x est le signe de
    (blots du joueur après le coup joué − blots après le meilleur). Un coup est retrouvé par
    le plateau qu'il laisse, non par son orthographe : « 13/7(2) » ou « 8/5*/4 » d'une
    analyse désignent les coups que le générateur écrit pas à pas (`engine.CanonicalMove`,
    départagé par le nombre de frappes). Un coup qui reste introuvable sort du compte, il
    n'est pas deviné nul ; cette sélection n'est pas neutre (elle touchait d'abord les
    doubles et les frappes), donc sa part est rendue avec le biais. La mesure compte les
    blots sans les pondérer par leur exposition : un blot hors de portée pèse autant qu'un
    blot à six cases d'un pion adverse.
12. **Usage.** Une seule fonction de stockage par mesure sert la GUI (Stats), la CLI
    (`stats effect`, `stats biases`) et le serveur (`/v1/stats.studyEffect`,
    `/v1/stats.biases`).

## Conséquences

- Un joueur qui étudie peu ou joue peu voit « insuffisant » plutôt qu'un progrès imaginaire.
- Le biais d'une case de score n'est affirmé que si sa propre preuve suffit : la plupart des
  cases resteront muettes, et c'est la réponse honnête.
- Les seuils (30 et 20 décisions, 95 %) ne changent qu'avec une nouvelle ADR.
