# ADR-0075 — La perte MWC ramenée à sept points complète le PR

Statut : acceptée.
Voir aussi : ADR-0019 (une seule échelle d'équité sort du moteur), ADR-0046 (les seuils d'erreur).

## Contexte

Le PR divise une somme d'erreurs par un nombre de décisions. Ce dénominateur dépend de ce que
l'on compte comme décision — coups forcés, doubles évidents, prises artificielles — et c'est
lui que la critique du PR vise : le numérateur, lui, ne dépend de rien, puisqu'une
non-décision coûte zéro. La note de recherche de l'utilisateur (« M1 ») propose de lire
directement la perte totale de chances de gagner le match (MWC) : sous jeu parfait la MWC est
une martingale, donc contre un adversaire parfait un joueur qui perd L de MWC sur le match
garde q = 0,5 − L de chances. Chaque erreur pèse alors ce qu'elle a coûté, au score où elle a
été commise.

La perte par décision existe déjà : `decisionMWCLoss` → `engine.ConvertEMGLossToMWCLoss`, sur
les décisions comptées du PR, sommée par match (`MatchBadge.MWCLoss`). Le numérateur est celui
du PR ; seul le dénominateur change.

## Décision

1. **La valeur affichée est L₇ = L · √(7/N)**, la perte MWC du match ramenée à un match en
   7 points, en pourcentage de MWC. Sous le modèle de la formule FIBS (marche aléatoire), la
   perte d'un joueur de force donnée croît comme √N : L/√N est la grandeur qui se compare
   d'une longueur à l'autre, et la multiplier par √7 la remet dans une unité que le joueur
   connaît. **7 est une constante nommée** (`domain.MWC7Length`) : la longueur de tournoi la
   plus courante, et celle où L₇ est exactement la perte MWC qu'eXtreme Gammon affiche.
   Nom : « Perte MWC (éq. 7 pts) », en anglais « MWC loss (7-pt eq.) » ; type `domain.MWC7`.
2. **L'Elo face au moteur est une lecture secondaire** (infobulle, champ json) : inversion de la
   formule FIBS à 7 points, D = (2000/√7)·log10(q/(1−q)) avec q = 0,5 − L₇. Il classe dans le
   même ordre que L₇. Linéarisé, D ≈ −3474·L/√N. Au-delà de L₇ ≈ 0,49, q n'a plus d'inverse
   fini : q est **plancher à 1 %** (`MWC7MinWin`, D ≥ −1509) et la valeur est marquée
   `elo_floored` (« ≤ »). Des joueurs réels perdent plus d'un demi-match (test.xg : 51 %), car
   les erreurs commises après un retournement de chance s'additionnent ; L₇, lui, n'a aucun
   plancher et ne vaut jamais NaN.
3. **L'agrégat met en commun avant le rapport** : L₇ = √7·ΣL/Σ√N sur les unités (match,
   joueur). C'est additif (on garde ΣL et Σ√N), ce n'est pas une moyenne de rapports, un match
   en 7 points pèse plus qu'un match en 1 point, et un agrégat d'une seule unité rend cette
   unité inchangée, intervalle compris. Le PR de la même sélection reste ce qu'il est.
   Sélection sans joueur : les deux sièges de chaque match sont deux unités.
4. **L'intervalle à 95 % rééchantillonne la plus grande unité indépendante** : les parties
   pour un match, les unités (match, joueur) pour un agrégat. La variance est celle du
   bootstrap, calculée et non tirée — G/(G−1)·Σ(l_g − l̄)² pour la somme de G parties,
   n/(n−1)·Σ(L − R·√N)²/(Σ√N)² pour l'estimateur de rapport R — : déterministe sans graine,
   une passe, aucune table. Bornes L ± 1,96·ES (la borne basse ne descend pas sous 0), puis
   même transformation que la valeur. Il faut deux parties, ou deux unités ; sinon pas
   d'intervalle. Un intervalle qui ne serait pas affiché ferait lire un match isolé comme un
   rang : la note de recherche estime l'erreur-type d'un match en 7 points à environ 1,5 PR.
5. **Une partie money n'a pas de L₇** : pas de N. `available` est faux, l'interface et la CLI
   le disent au lieu d'afficher un nombre.
6. **Où** : badge de la liste des matchs (par joueur), détail du match, statistiques globales,
   par match et par tournoi (donc la progression), badge de tournoi (joueur de référence) ;
   CLI `stats`, `stats progression`, `match` ; le serveur sert les mêmes structures. Le calcul
   vit dans `domain` (formule, agrégat) et `sqlshared` (lectures), commun aux deux moteurs de
   stockage, sans changement de schéma : la voie des cellules lit le siège et la longueur déjà
   présents, l'intervalle d'un match vient de `mv.game_id` dans la requête du détail.

## Conséquences

- Aucune nouvelle conversion EMG → MWC : tout passe par `ConvertEMGLossToMWCLoss`, et suit
  donc ses corrections (Crawford, règles de comptage).
- L'intervalle d'une ligne par match des statistiques est celui de ses deux sièges, pas celui
  des parties : il faudrait relire chaque décision, ce que la voie des cellules évite. Le
  détail du match porte l'intervalle des parties.
- Le modèle √N est approximatif sous 7 points et autour du Crawford, et L₇ dépend de la MET,
  comme le MWC cost. Le tableau des joueurs (classement) ne l'affiche pas encore.
