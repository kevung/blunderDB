# ADR-0077 — L'incertitude se montre, et le bilan de match oriente l'étude

Statut : acceptée.
Voir aussi : ADR-0010 (la chance par coup), ADR-0046 (les seuils d'erreur), ADR-0073 (le temps
d'une décision), ADR-0075 (L₇ et son intervalle), ADR-0076 (difficulté, erreur évitable).

## Contexte

Les statistiques montrent des PR par phase, par plan de jeu, par étiquette et par score, et
grisent une case sous dix décisions. Le seuil est arbitraire et dit mal ce qu'il faut savoir :
dix décisions d'un même match ne valent pas dix décisions de dix matchs, et une case de 300
décisions venues de deux matchs reste du bruit. Au niveau d'un match, le joueur voit son PR,
sa perte et la courbe des pertes, mais rien ne lui dit quoi revoir, s'il a perdu à cause des
dés ou du jeu, ni si ses erreurs sont celles de la précipitation ou celles d'une lacune.
Chaque seuil est fixé ici avant de regarder les résultats.

## Décision

1. **L'intervalle rééchantillonne la plus grande unité indépendante, comme pour L₇.**
   - *Un match* (bilan, détail) : les **parties**. PR = 500·Σe/Σn est un estimateur de rapport :
     variance du bootstrap calculée, G/(G−1)·Σ(e_g − R·n_g)²/(Σn)², sur les G parties où le
     joueur a une décision comptée ; L₇ garde la formule d'ADR-0075.
   - *Un agrégat* (PR global des statistiques, cellules de ventilation par phase, plan de jeu,
     étiquette et score) : les **matchs**. Les cellules stockées sont par match, pas par partie ;
     surtout, les parties d'un même match partagent adversaire, séance et fatigue, et les
     rééchantillonner séparément rétrécirait l'intervalle à tort. Sans filtre joueur, les deux
     sièges d'un match forment une seule unité : ils jouent les mêmes positions.
   - Il faut deux unités ; sinon pas d'intervalle. Bornes R ± 1,96·ES, la basse pas sous 0.
   - Calcul fermé, sans graine ni tirage (`domain.RatioPool`) : même résultat à chaque lecture,
     et les deux voies des statistiques (cellules, lignes) le reproduisent à l'identique.
   - **La bande remplace le grisage « < 10 décisions »** (`MinCellDecisions` disparaît) : une
     case sans intervalle est grisée, les autres montrent [bas – haut].
   - Réserves de la relecture d'ADR-0075 : une ligne par match d'une sélection sans joueur
     réunit deux sièges, son L₇ n'a **pas d'intervalle** (il mesurerait l'écart entre les
     adversaires) ; une ligne par match filtrée par joueur est une seule unité, sans intervalle
     non plus — c'est le bilan du match qui porte l'intervalle par parties, et le manuel le dit.
     `MatchBadges` ne déclare pas disponible un siège dont aucune perte n'est chiffrée.
2. **« 3 décisions à revoir »**, par joueur : parmi ses erreurs (seuil de la bibliothèque) dont
   la perte MWC est chiffrée, le rang est la **part évitable de la perte**, perte − difficulté
   (ADR-0076), soit perte × (1 − difficulté/perte) : le produit de la perte par le caractère
   évitable. Une difficulté inconnue compte 0. Égalités : la plus grosse perte, puis l'ordre du
   match. Trois au plus, d'une part évitable positive. Une erreur que le joueur de référence
   aurait faite aussi ne vaut pas une séance : on revoit ce qui était à sa portée.
3. **Résultat ajusté de la chance**, match fini, en points de MWC du joueur :
   - résultat R = issue (1 gagné, 0 perdu) − MWC au départ (table de la bibliothèque au score
     initial de la première partie ; 50 % à 0-0) ;
   - chance nette C = Σ chance de ses jets − Σ chance des jets adverses, chaque jet
     (`move.luck_mp`, équité) converti en MWC au score et au videau de sa position par la
     conversion des erreurs (`ConvertEMGLossToMWCLoss`, linéaire) ;
   - **résultat ajusté A = R − C**. Sous le modèle de martingale d'ADR-0075, son espérance est
     la perte de l'adversaire moins la sienne : le bilan affiche les deux côte à côte.
     A > 0 avec un match perdu : perdu par les dés ; A < 0 : perdu par le jeu.
   - Indisponible pour une partie money, un match inachevé, ou quand aucun jet ne porte de
     chance ; la couverture (jets mesurés / jets joués) est affichée, car une chance partielle
     biaise A.
4. **Erreur précipitée / erreur réfléchie** : par joueur et par type (pions, videau), le seuil
   est la **médiane des durées connues** de ses décisions chiffrées de ce type dans le match ;
   une erreur jouée plus vite que la médiane est précipitée, les autres réfléchies, les durées
   inconnues à part. Un seuil en secondes dépendrait de la cadence ; la médiane propre au
   joueur et au match, non : sans lien entre vitesse et erreur, les erreurs se partagent
   moitié-moitié. Une majorité précipitée appelle de la discipline (ralentir) ; une majorité
   réfléchie, de la connaissance (étudier la famille de positions).
5. **Où** : le bilan (`storage.MatchReview`, lecture `MatchReview` du contrat, sqlshared
   commun aux deux moteurs) est servi au panneau Match, à `match --format summary` et au
   serveur (`/v1/stats.matchReview`). Les intervalles des statistiques passent dans les mêmes
   structures que les PR (`PRInterval`), donc GUI, CLI et serveur. Aucun changement de schéma.
   Sous le choix « MWC 7 pts », le classement des joueurs gagne une colonne L₇, et l'onglet
   Erreurs et les cartes du tableau de bord lisent L₇ (pions/videau) au lieu de la perte brute.

## Conséquences

- Un PR d'une case peut avoir un intervalle large malgré beaucoup de décisions : c'est le
  but. Une case d'un seul match n'en a pas, quel que soit son effectif.
- Le bilan relit chaque décision du match ; c'est un match, pas une base.
- Le résultat ajusté dépend de la MET de la bibliothèque, comme L₇, et de la chance que
  l'outil d'analyse a écrite (ADR-0010) : un fichier sans chance n'en a pas.
