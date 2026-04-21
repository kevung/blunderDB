.. _stats:

Panneau Stats
=============

Introduction
------------

Le panneau **Stats** permet d'analyser son niveau de jeu et de suivre sa
progression dans le temps à partir des positions importées dans la base de
données. Il calcule et affiche les indicateurs **PR** (Performance Rate) et
**MWC cost** (Match Winning Chance cost) pour l'ensemble des positions ou un
sous-ensemble filtré.

Le panneau Stats est particulièrement utile pour :

* **situer son niveau** par rapport aux seuils de référence (world-class,
  expert, avancé…) grâce au PR global ;

* **suivre sa progression** tournoi après tournoi ou match après match grâce
  aux graphiques de l'onglet Progression ;

* **identifier ses points faibles** : onglet Erreurs pour voir la répartition
  entre coups joués et décisions de videau, et la distribution des magnitudes
  d'erreur ;

* **accéder directement aux positions concernées** en cliquant sur n'importe
  quel indicateur (drill-down).

Ouverture du panneau
--------------------

Pour ouvrir le panneau Stats :

* Appuyer sur *CTRL-D*.
* Saisir la commande ``:stats`` ou ``:st`` dans la ligne de commande.

.. note::
   Le panneau se rafraîchit automatiquement à chaque modification du filtre.
   Il ne recalcule pas les statistiques lors d'un simple basculement PR ↔ MWC :
   les deux métriques sont calculées simultanément par le backend.

Barre de filtre
---------------

La barre de filtre, en haut du panneau, permet de restreindre le calcul à un
sous-ensemble de positions.

Perspective joueur
~~~~~~~~~~~~~~~~~~

La liste déroulante **Joueur** permet de filtrer les statistiques selon le
joueur analysé. blunderDB sélectionne automatiquement le joueur dont le nom
apparaît le plus souvent dans la base de données — modifiable à tout moment.

.. tip::
   Changer de joueur ne provoque pas de perte de données ; il suffit de
   re-sélectionner le joueur précédent dans la liste.

Filtres disponibles
~~~~~~~~~~~~~~~~~~~

* **Tournoi(s)** — restriction à un ou plusieurs tournois. Plusieurs tournois
  peuvent être sélectionnés simultanément.

* **Dates** — plage temporelle (*De* … *À*). Si seule la date de début est
  renseignée, les positions plus récentes sont incluses.

* **Type de décision** — Tous / Coups joués / Décisions de videau.

* **Longueur de match** — restriction à des longueurs de match précises (1, 3,
  5, 7, 9, 11, 13, 15, 21 points). Plusieurs longueurs peuvent être combinées.

Un bouton **Reset** remet tous les filtres à zéro (sauf le joueur
auto-détecté).

.. note::
   Les filtres sont persistés dans la configuration de blunderDB
   (``config.yaml``) et sont restaurés à la prochaine ouverture.

Toggle PR / MWC
---------------

Le bouton **PR / MWC** en haut du panneau bascule la métrique affichée dans
tous les onglets.

**PR (Performance Rate)**

  Mesure la qualité de jeu *money-game* : somme des erreurs en millièmes de
  point de backgammon, divisée par le nombre de décisions. Indépendant du
  score de match.

  Seuils de référence approximatifs :

  .. csv-table::
     :header: "Niveau", "PR"
     :widths: 20, 10
     :align: center

     "World-class", "< 3"
     "Expert", "3 – 5"
     "Avancé", "5 – 8"
     "Intermédiaire", "8 – 12"
     "Débutant", "> 12"

**MWC cost (Match Winning Chance cost)**

  Probabilité cumulée de victoire de match perdue à cause des erreurs, sur
  l'ensemble du jeu de données filtré. Calculé à partir de la MET
  Kazaross-XG2 embarquée dans blunderDB.

  .. caution::
     Le MWC cost **n'est pas applicable** aux positions *money-game* (sans
     enjeu de match). Ces positions sont exclues du calcul MWC.
     Les valeurs MWC dépendent de la MET utilisée ; elles ne sont pas
     directement comparables entre logiciels utilisant des METs différentes.

Le basculement PR ↔ MWC est instantané : aucun recalcul backend n'est
effectué.

Onglet Dashboard
----------------

L'onglet **Dashboard** donne une vue synthétique des indicateurs clés.

Cartes de niveau
~~~~~~~~~~~~~~~~

Trois cartes affichent le PR (ou MWC) pour :

* **All** — toutes les décisions (coups + videau) ;
* **Checker** — coups joués seulement ;
* **Cube** — décisions de videau seulement.

Cliquer sur une carte charge dans le panneau d'analyse les positions du
sous-ensemble correspondant (drill-down).

.. note::
   Le nombre total de décisions est affiché en bas de chaque carte au survol.

PR glissant sur N dernières décisions
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Une ligne de valeurs PR (ou MWC) calculées sur les *N* dernières décisions
(N = 5, 10, 50, 100, 250, 500, 1000) permet de mesurer la tendance récente.
Les valeurs grisées correspondent à un N supérieur au nombre de décisions
disponibles.

Cliquer sur une valeur charge les *N* dernières positions correspondantes.

Top blunders
~~~~~~~~~~~~

La liste des 10 pires erreurs (ou MWC cost), triées par magnitude décroissante.
Cliquer sur une ligne charge la position concernée dans le panneau d'analyse.

Onglet Progression
------------------

L'onglet **Progression** présente l'évolution du niveau dans le temps.

Courbe par tournoi
~~~~~~~~~~~~~~~~~~

Un graphique en ligne affiche le PR (ou MWC) pour chaque tournoi (axe X :
ordre des tournois, axe Y : valeur de la métrique). Des bandes de couleur
matérialisent les seuils de niveau.

Cliquer sur un point du graphique ouvre un menu contextuel avec deux options :

* **Open tournament** — ouvre le tournoi dans le panneau Tournois.
* **Open positions** — charge toutes les positions du tournoi dans le panneau
  d'analyse.

Scatter plot par match
~~~~~~~~~~~~~~~~~~~~~~

Un nuage de points représente chaque match (axe X : date, axe Y : PR ou MWC).
La taille du point est proportionnelle au nombre de décisions dans le match.

Cliquer sur un point ouvre un menu contextuel :

* **Open match** — ouvre le match dans le panneau des matchs.
* **Open positions** — charge toutes les positions du match dans le panneau
  d'analyse.

Onglet Erreurs
--------------

L'onglet **Erreurs** décompose les sources d'erreurs.

Répartition par action de videau
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Un diagramme en barres affiche le PR (ou MWC) pour chaque type de décision
de videau : *NoDouble*, *DoubleTake*, *DoublePass*, *TooGood*. Chaque barre
indique également le nombre de décisions et le taux de blunders en infobulle.

Cliquer sur une barre charge les positions correspondant à cette action de
videau, **uniquement celles avec une erreur** (drill-down).

Répartition Checker / Cube
~~~~~~~~~~~~~~~~~~~~~~~~~~~

Un diagramme comparatif place côte à côte le PR des coups joués et des
décisions de videau. Cliquer sur une barre charge les positions du
sous-ensemble avec erreur.

Histogramme des magnitudes d'erreur
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Un histogramme distribue les erreurs selon leur magnitude en millièmes de
point (tranches : 0–5, 5–10, 10–25, 25–50, 50–100, ≥ 100). Cliquer sur
une barre charge les positions de la tranche.

Règle d'agrégation
------------------

.. important::
   Le PR d'un tournoi (ou d'un sous-ensemble quelconque) est calculé par
   la règle **somme/somme** — jamais comme moyenne des PR individuels des
   matchs.

   Formule :

   .. math::

      PR_{tournoi} = \frac{\sum_{i} \text{erreur}_i}{\text{nombre total de décisions}}

   **Exemple :** un joueur dispute deux matchs dans un tournoi —

   * Match A : 10 décisions, erreur totale 50 mp → PR = 5,0
   * Match B : 90 décisions, erreur totale 270 mp → PR = 3,0

   Moyenne naïve des PR : (5,0 + 3,0) / 2 = **4,0** *(incorrect)*

   Règle somme/somme : (50 + 270) / (10 + 90) = 320 / 100 = **3,2** *(correct)*

   La règle somme/somme est la seule qui résiste à la variation de longueur
   des matchs (un match en 21 points pèse plus qu'un match en 1 point).

MWC : limitations
-----------------

* Le MWC cost est calculé à partir de la **MET Kazaross-XG2**, table de
  référence de facto dans le backgammon compétitif. Les résultats ne sont
  pas directement comparables avec des logiciels utilisant d'autres METs.

* Les positions *money-game* (sans score de match) sont **exclues** du
  calcul MWC. Si votre base de données contient beaucoup de positions
  money-game, le MWC cost peut être sous-estimé ou indisponible.

* Le MWC cost est cumulatif sur l'ensemble du jeu de données filtré — pas
  un indicateur par décision. Il mesure l'impact total de vos erreurs sur
  vos chances de victoire.
