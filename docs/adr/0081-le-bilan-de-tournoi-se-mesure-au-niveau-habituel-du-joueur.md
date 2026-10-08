# ADR-0081 — Le bilan de tournoi se mesure au niveau habituel du joueur

Statut : acceptée.
Voir aussi : ADR-0075 (L₇ et son intervalle), ADR-0076 (difficulté), ADR-0077 (familles
d'erreurs), ADR-0078 (intervalles, bilan de match), ADR-0073 (durée d'une décision).

## Contexte

Après un tournoi, un joueur veut savoir s'il a joué à son niveau, ce qui lui a coûté, et si
quelque chose a cédé en cours de route : la fatigue des dernières rondes ou des fins de match
longues, les scores où un point décide de tout, la pendule. Le panneau Tournois donne le PR et
L₇ de l'épreuve, sans référence ni ventilation ; le bilan de match (ADR-0078) ne voit qu'un
match. Un tournoi compte de trois à dix matchs : chaque case qu'on y découpe est petite, et la
tentation de lire une tendance dans le bruit est forte. Les définitions et seuils sont fixés
ici avant d'avoir regardé ce qu'ils donnent sur une base réelle.

## Décision

1. **Le joueur** est un nom et ses alias (ceux des statistiques). Sans nom, c'est celui qui
   figure dans le plus de matchs du tournoi (à égalité, le premier dans l'ordre alphabétique) :
   le propriétaire de la base, presque toujours.
2. **Le niveau habituel** est ce joueur sur ses matchs en points (longueur 1 à 64) datés des
   **365 jours qui précèdent** le tournoi, ceux du tournoi exclus. La date du tournoi est sa
   date, à défaut celle de son premier match ; sans date, pas de niveau habituel. Seul l'avant
   compte : l'après mêlerait ce que le tournoi et son étude ont changé. Il faut **au moins
   5 matchs** dans la fenêtre ; en deçà, le bilan dit « niveau habituel inconnu ».
3. **Les mesures.** L₇ (ADR-0075) et PR, pour le tournoi et pour le niveau habituel, chacun
   avec l'intervalle d'ADR-0078 qui rééchantillonne les **matchs** ; une ronde seule
   rééchantillonne ses parties. Toutes les décisions passent par la lecture du bilan de match
   (`MatchDecisionLosses`) : mêmes décisions comptées, même conversion, mêmes totaux que les
   badges.
4. **La comparaison.** Δ = tournoi − habituel, écart-type √(σ_t² + σ_h²), chaque σ lu sur la
   demi-largeur haute de son intervalle (la basse est écrêtée à 0). « Moins bien » si la borne
   basse de Δ est positive, « mieux » si sa borne haute est négative, « dans l'habitude »
   sinon. Les deux côtés sont traités comme indépendants : ils partagent le joueur, et une
   covariance positive rendrait l'intervalle plus étroit, pas plus large ; la règle est
   prudente.
5. **Pas d'affirmation sans borne.** Une comparaison n'est rendue que si les deux côtés ont un
   intervalle **et au moins 20 décisions** ; sinon « échantillon insuffisant ». Les chiffres
   restent affichés, le verdict non.
6. **Les ventilations**, toutes en PR (L₇ est une mesure de match entier et ne se découpe
   pas), chaque case du tournoi face à la même case du niveau habituel :
   - **ronde** : chaque match du tournoi dans l'ordre du tournoi, avec l'adversaire, son PR
     (intervalle par parties) et son L₇, face au PR habituel global ;
   - **rang de la décision dans le match** (fatigue) : les décisions comptées du joueur dans
     l'ordre du match, par tranches de 30 (1–30, 31–60, 61–90, 91 et plus) ;
   - **scores de pression** : DMP (les deux joueurs à un point, après Crawford), partie
     Crawford, post-Crawford hors DMP, autres scores ;
   - **pendule** : décision rapide ou posée, de part et d'autre de la médiane des durées
     connues du joueur pour ce type de décision dans ce match (la règle d'ADR-0078) ; une
     durée inconnue n'entre dans aucune case. La réserve de la pendule n'est pas enregistrée
     coup par coup : la pression du temps restant ne se mesure pas encore.
7. **Les familles de l'épreuve** sont le plan d'étude d'ADR-0077 restreint aux erreurs du
   joueur dans le tournoi, mêmes règles (5 membres chiffrés, borne basse > 0) : au plus
   **trois**, par borne basse ; les familles « à confirmer » sont comptées, pas nommées.
8. **Où** : `storage.TournamentReview` (pur) et la lecture `TournamentReview` du contrat,
   sqlshared commun aux deux moteurs ; le panneau Tournois, `stats tournament` et
   `/v1/stats.tournamentReview`. Aucun changement de schéma.

## Conséquences

- Un tournoi de trois matchs ne dit presque jamais « moins bien » : c'est voulu. Le bilan
  montre les chiffres et l'incertitude, et réserve le verdict à ce qui tient.
- Le niveau habituel relit chaque décision de l'année précédente du joueur : quelques
  centaines de matchs, une lecture à la demande, pas une carte de tableau de bord.
- Les seuils (365 jours, 5 matchs, 20 décisions, tranches de 30, trois familles) ne changent
  qu'avec une nouvelle ADR.
