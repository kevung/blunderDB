.. _manuel:

Manuel
======

Introduction
------------

blunderDB est un logiciel pour constituer des bases de données de
positions de backgammon. Sa force principale est de fournir un lieu unique
pour agréger les positions qu'un joueur a rencontrées (en ligne, en tournoi)
et de pouvoir les réétudier en les filtrant selon divers filtres arbitrairement
combinables. blunderDB peut également être utilisé pour créer des catalogues
de positions de référence.

Les positions sont stockées dans une base de données représentée par un fichier
*.db*.

Interactions principales
------------------------

Les principales interactions possibles avec blunderDB sont:

* ajouter une nouvelle position,

* modifier une position existante,

* copier l'image du board dans le presse-papier (PNG) via **Ctrl+X**, ou avec l'analyse complète via **Ctrl+X Ctrl+X**,

* supprimer une position existante,

* rechercher une ou plusieurs positions,

* importer des matchs depuis différentes sources (XG, GNUbg, BGBlitz, Jellyfish), y compris les commentaires depuis les fichiers XG,

* naviguer dans les coups d'un match importé,

* organiser les positions en collections,

* organiser les matchs en tournois.

L'utilisateur peut étiqueter librement les positions à l'aide de tags et les
annoter via des commentaires.

Description de l'interface
--------------------------

L'interface de blunderDB est constituée de haut en bas par:

* [en haut] la barre d'outils, qui rassemble l'ensemble des principales
  opérations réalisables sur la base de données,

* [au milieu] la zone d'affichage principale, qui permet d'afficher ou d'éditer des
  positions de backgammon,

* [en bas] la barre d'état, qui présente différentes informations sur la
  base de données ou la position courante, et intègre la ligne de commande.

Des panneaux peuvent être affichés pour:

* afficher les données d'analyse associées à la position courante issues
  d'eXtreme Gammon (XG), GNUbg, ou BGBlitz,

* afficher, ajouter ou modifier des commentaires,

* afficher la liste des matchs importés et naviguer dans les coups d'un match (panneau matchs),

* afficher et gérer les collections de positions (panneau collections),

* étudier les positions par répétition espacée (panneau Anki),

* afficher et gérer les tournois (panneau tournois),

* afficher les statistiques de performance (panneau Stats),

* calculer l'EPC (Effective Pip Count) d'une position de bearoff (panneau EPC),

* afficher les métadonnées de la base de données (panneau métadonnées),

* afficher la bibliothèque de filtres,

* afficher l'historique des recherches,

* afficher le journal des opérations (panneau log).

Des fenêtres modales peuvent s'afficher pour:

* afficher l'aide de blunderDB,

* paramétrer l'export de la base de données,

* afficher les métadonnées de la base de données.

La zone d'affichage principale met à disposition à l'utilisateur:

* un board afin d'afficher ou d'éditer une position de backgammon,

* le niveau et le propriétaire du cube,

* le compte de course de chaque joueur,

* le score de chaque joueur,

* les dés à jouer. Si aucune valeur n'est affichée sur les dés, la
  position des dés indique quel joueur a le trait et que la position est
  une décision de cube.

La barre d'état est structurée de gauche à droite par les informations
suivantes:

* la ligne de commande, accessible en appuyant sur la touche *ESPACE*,

* un message d'information lié à une opération réalisée par l'utilisateur,

* l'index de la position courante, suivi du nombre de positions dans la
  bibliothèque courante (ou les informations de coup/partie lors de la
  navigation dans un match).

.. note:: Dans le cas de positions issues d'une recherche par l'utilisateur, le
   nombre de positions indiqué dans la barre d'état correspond au nombre de
   positions filtrées.

.. _mode_normal:

Navigation dans les positions
-----------------------------

Par défaut, blunderDB permet de:

* faire défiler les différentes positions de la bibliothèque courante,

* afficher les informations d'analyse associées à une position,

* afficher, ajouter et modifier les commentaires d'une position.

.. tip:: Se référer à :ref:`raccourcis` pour les raccourcis disponibles.

.. _mode_edit:

Édition de positions
--------------------

L'appui sur la touche *TAB* ouvre le panneau de recherche et permet
d'éditer une position sur le plateau pour l'ajouter à la base de données
ou pour définir une structure de position à rechercher.
La distribution des pions, du videau, du score, et du trait peuvent être
modifiés à l'aide de la souris (voir :ref:`guide_edit_position`).

.. tip:: Se référer à :ref:`raccourcis` pour les raccourcis disponibles.

.. _mode_command:

La ligne de commande
--------------------

La ligne de commande, intégrée dans la barre d'état, permet de réaliser
l'ensemble des fonctionalités de blunderDB disponibles à l'interface
graphique: opérations générales sur la base de données, navigation de
position, affichage de l'analyse et/ou des commentaires, recherche de
positions selon des filtres... Après une première prise en main de
l'interface, il est recommandé de progressivement utiliser la ligne de
commande qui permet une utilisation puissante et fluide de blunderDB,
notamment pour les fonctionnalités de recherche de positions.

Pour ouvrir la ligne de commande, appuyer sur
la touche *ESPACE*. Pour envoyer une requête et fermer la ligne de
commande, appuyer sur la touche *ENTREE*.

blunderDB exécute les requêtes envoyées par l'utilisateur sous réserve
qu'elles soient valides et modifie immédiatement l'état de la base de données
le cas échéant. Il n'y a pas d'actions de sauvegarde explicite de la part
de l'utilisateur.

Pour affiner une recherche parmi les positions actuellement filtrées, utiliser
la commande ``ss`` suivie de filtres (ex: ``ss nc``, ``ss E>40``). La commande
``ss`` fonctionne après une recherche préalable. La fenêtre de recherche
(``CTRL-F``) propose également une case à cocher "Search in current results"
pour la même fonctionnalité.

.. tip:: Se référer à la :numref:`cmd_mode` pour la liste de commandes
   disponible en ligne de commande.

.. _mode_match:

Navigation dans les matchs
--------------------------

La navigation dans les matchs permet de parcourir les coups d'un match importé.
Elle est activée depuis le panneau des matchs (*CTRL-Tab*) en double-cliquant
sur un match ou en appuyant sur *ENTREE*. La commande ``m`` permet également
de reprendre la navigation dans le dernier match visité.

L'utilisateur peut:

* parcourir les coups d'un match en utilisant les touches *GAUCHE* et *DROITE*,

* passer d'une partie à l'autre à l'aide des touches *PageUp* et *PageDown*,

* afficher l'analyse des coups (pions et cube) en appuyant sur *CTRL-L*,

* basculer entre l'analyse des coups de pions et du cube avec la touche *d*,

* voir le coup effectivement joué mis en évidence dans l'analyse.

La dernière position visitée dans chaque match est mémorisée et restaurée
automatiquement.

.. tip:: Se référer à :ref:`raccourcis` pour les raccourcis disponibles.

.. _mode_collection:

Navigation dans les collections
-------------------------------

La navigation dans les collections permet de parcourir les positions d'une collection.
Elle est activée en double-cliquant sur une collection dans le panneau
des collections (*CTRL-B*).

L'utilisateur peut naviguer parmi les positions de la collection
en utilisant les touches *GAUCHE* et *DROITE*. L'ordre des collections
et des positions dans les collections peut être modifié par glisser-déposer.

.. _mode_anki:

Répétition espacée (Anki)
-------------------------

Le panneau Anki (*CTRL-K*) permet d'étudier des positions par répétition espacée
en utilisant l'algorithme FSRS. L'utilisateur peut créer des paquets à partir
de collections ou de résultats de recherche.

**Création de paquets :** Cliquez sur *New Deck* pour créer un paquet à partir
d'une collection ou des résultats de recherche courants. Les paquets basés sur
une recherche se synchronisent automatiquement à l'activation de l'onglet Anki.

**Révision :** Sélectionnez un paquet puis cliquez sur *Study* (ou double-cliquez
sur un paquet) pour commencer la révision des cartes dues. Chaque carte affiche
la position correspondante sur le plateau. Évaluez votre rappel avec les touches
*1* (À revoir), *2* (Difficile), *3* (Bien), ou *4* (Facile). Appuyez sur *Esc*
pour arrêter et revenir à la liste des paquets.

**Arrêt/Reprise :** Vous pouvez interrompre une session de révision à tout moment
avec *Esc*. Le bouton change en *Resume* et affiche votre progression.
Cliquez dessus pour reprendre là où vous vous êtes arrêté.

**Gestion des paquets :** Utilisez les boutons d'action pour renommer,
synchroniser, réinitialiser ou supprimer des paquets. Les paramètres FSRS
(rétention cible, intervalle maximum, aléa) peuvent être configurés par
paquet dans les Paramètres (icône engrenage).

.. _stats:

Panneau Stats
-------------

Introduction
~~~~~~~~~~~~~

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
~~~~~~~~~~~~~~~~~~~~~

Pour ouvrir le panneau Stats :

* Appuyer sur *CTRL-D*.
* Saisir la commande ``:stats`` ou ``:st`` dans la ligne de commande.

.. note::
   Le panneau se rafraîchit automatiquement à chaque modification du filtre.
   Il ne recalcule pas les statistiques lors d'un simple basculement PR ↔ MWC :
   les deux métriques sont calculées simultanément par le backend.

Barre de filtre
~~~~~~~~~~~~~~~

La barre de filtre, en haut du panneau, permet de restreindre le calcul à un
sous-ensemble de positions.

Perspective joueur
^^^^^^^^^^^^^^^^^^

La liste déroulante **Joueur** permet de filtrer les statistiques selon le
joueur analysé. blunderDB sélectionne automatiquement le joueur dont le nom
apparaît le plus souvent dans la base de données — modifiable à tout moment.

.. tip::
   Changer de joueur ne provoque pas de perte de données ; il suffit de
   re-sélectionner le joueur précédent dans la liste.

Filtres disponibles
^^^^^^^^^^^^^^^^^^^

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
~~~~~~~~~~~~~~~

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
~~~~~~~~~~~~~~~~

L'onglet **Dashboard** donne une vue synthétique des indicateurs clés.

Cartes de niveau
^^^^^^^^^^^^^^^^

Trois cartes affichent le PR (ou MWC) pour :

* **All** — toutes les décisions (coups + videau) ;
* **Checker** — coups joués seulement ;
* **Cube** — décisions de videau seulement.

Cliquer sur une carte charge dans le panneau d'analyse les positions du
sous-ensemble correspondant (drill-down).

.. note::
   Le nombre total de décisions est affiché en bas de chaque carte au survol.

PR glissant sur N dernières décisions
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Une ligne de valeurs PR (ou MWC) calculées sur les *N* dernières décisions
(N = 5, 10, 50, 100, 250, 500, 1000) permet de mesurer la tendance récente.
Les valeurs grisées correspondent à un N supérieur au nombre de décisions
disponibles.

Cliquer sur une valeur charge les *N* dernières positions correspondantes.

Top blunders
^^^^^^^^^^^^

La liste des 10 pires erreurs (ou MWC cost), triées par magnitude décroissante.
Cliquer sur une ligne charge la position concernée dans le panneau d'analyse.

Onglet Progression
~~~~~~~~~~~~~~~~~~

L'onglet **Progression** présente l'évolution du niveau dans le temps.

Courbe par tournoi
^^^^^^^^^^^^^^^^^^

Un graphique en ligne affiche le PR (ou MWC) pour chaque tournoi (axe X :
ordre des tournois, axe Y : valeur de la métrique). Des bandes de couleur
matérialisent les seuils de niveau.

Cliquer sur un point du graphique ouvre un menu contextuel avec deux options :

* **Open tournament** — ouvre le tournoi dans le panneau Tournois.
* **Open positions** — charge toutes les positions du tournoi dans le panneau
  d'analyse.

Scatter plot par match
^^^^^^^^^^^^^^^^^^^^^^

Un nuage de points représente chaque match (axe X : date, axe Y : PR ou MWC).
La taille du point est proportionnelle au nombre de décisions dans le match.

Cliquer sur un point ouvre un menu contextuel :

* **Open match** — ouvre le match dans le panneau des matchs.
* **Open positions** — charge toutes les positions du match dans le panneau
  d'analyse.

Onglet Erreurs
~~~~~~~~~~~~~~

L'onglet **Erreurs** décompose les sources d'erreurs.

Répartition par action de videau
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Un diagramme en barres affiche le PR (ou MWC) pour chaque type de décision
de videau : *NoDouble*, *DoubleTake*, *DoublePass*, *TooGood*. Chaque barre
indique également le nombre de décisions et le taux de blunders en infobulle.

Cliquer sur une barre charge les positions correspondant à cette action de
videau, **uniquement celles avec une erreur** (drill-down).

Répartition Checker / Cube
^^^^^^^^^^^^^^^^^^^^^^^^^^^

Un diagramme comparatif place côte à côte le PR des coups joués et des
décisions de videau. Cliquer sur une barre charge les positions du
sous-ensemble avec erreur.

Histogramme des magnitudes d'erreur
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Un histogramme distribue les erreurs selon leur magnitude en millièmes de
point (tranches : 0–5, 5–10, 10–25, 25–50, 50–100, ≥ 100). Cliquer sur
une barre charge les positions de la tranche.

Règle d'agrégation
~~~~~~~~~~~~~~~~~~

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
~~~~~~~~~~~~~~~~~

* Le MWC cost est calculé à partir de la **MET Kazaross-XG2**, table de
  référence de facto dans le backgammon compétitif. Les résultats ne sont
  pas directement comparables avec des logiciels utilisant d'autres METs.

* Les positions *money-game* (sans score de match) sont **exclues** du
  calcul MWC. Si votre base de données contient beaucoup de positions
  money-game, le MWC cost peut être sous-estimé ou indisponible.

* Le MWC cost est cumulatif sur l'ensemble du jeu de données filtré — pas
  un indicateur par décision. Il mesure l'impact total de vos erreurs sur
  vos chances de victoire.

.. _mode_epc:

Calculateur EPC
---------------

Le panneau EPC permet de calculer l'EPC (Effective Pip Count) d'une position
de bearoff. Il est activé en appuyant *CTRL-E*, en cliquant sur l'onglet
EPC dans le panneau inférieur, ou en exécutant la commande ``epc``.

Dans ce panneau, l'utilisateur édite la position des pions dans le jan
(6 derniers points) et les informations suivantes sont affichées
en temps réel dans le panneau EPC dédié pour chaque joueur:

* l'EPC (Effective Pip Count),

* le nombre moyen de lancers nécessaires (Mean Rolls),

* l'écart type (Standard Deviation),

* le pip count,

* le wastage (différence entre l'EPC et le pip count).

Lorsque les deux joueurs ont des pions dans leur jan, une section
de comparaison affiche les différences d'EPC et de pip count.

Pour quitter le panneau EPC, appuyer sur *CTRL-E* ou basculer sur
un autre onglet.

.. note:: Le calcul repose sur la base de données interne de bearoff
   à 6 points de GNUbg.

