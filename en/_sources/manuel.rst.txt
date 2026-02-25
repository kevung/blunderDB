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

* supprimer une position existante,

* rechercher une ou plusieurs positions,

* importer des matchs depuis différentes sources (XG, GNUbg, BGBlitz, Jellyfish),

* naviguer dans les coups d'un match importé,

* organiser les positions en collections,

* organiser les matchs en tournois.

Modes
-----

Pour ce faire, l'utilisateur bascule dans des modes dédiés pour:

* la navigation et la visualisation de positions (mode NORMAL),

* l'édition de positions (mode EDIT),

* l'édition d'une requête pour filtrer des positions (mode COMMAND ou fenêtre de recherche),

* la navigation dans les coups d'un match importé (mode MATCH),

* le calcul de l'EPC (Effective Pip Count) à partir d'une position (mode EPC),

* la navigation dans une collection de positions (mode COLLECTION).

L'utilisateur peut étiqueter librement les positions à l'aide de tags et les
annoter via des commentaires.

Dans la suite du manuel, il est décrit l'interface graphique ainsi que
les principaux modes de blunderDB.

Description de l'interface
--------------------------

L'interface de blunderDB est constituée de haut en bas par:

* [en haut] la barre d'outils, qui rassemble l'ensemble des principales
  opérations réalisables sur la base de données,

* [au milieu] la zone d'affichage principale, qui permet d'afficher ou d'éditer des
  positions de backgammon,

* [en bas] la barre d'état, qui présente différentes informations sur la
  base de données ou la position courante.

Des panneaux peuvent être affichés pour:

* afficher les données d'analyse associées à la position courante issues
  d'eXtreme Gammon (XG), GNUbg, ou BGBlitz,

* afficher, ajouter ou modifier des commentaires,

* afficher la liste des matchs importés (panneau matchs),

* afficher et gérer les collections de positions (panneau collections),

* afficher et gérer les tournois (panneau tournois),

* afficher la bibliothèque de filtres,

* afficher l'historique des recherches.

Des fenêtres modales peuvent s'afficher pour:

* [mode EDIT uniquement] paramétrer les filtres de recherche,

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

* le mode courant (NORMAL, EDIT, COMMAND),

* un message d'information lié à une opération réalisée par l'utilisateur,

* l'index de la position courante, suivi du nombre de positions dans la
  bibliothèque courante.

.. note:: Dans le cas de positions issues d'une recherche par l'utilisateur, le
   nombre de positions indiqué dans la barre d'état correspond au nombre de
   positions filtrées.

.. _mode_normal:

Le mode NORMAL
--------------

Le mode NORMAL est le mode par défaut de blunderDB. Il est utilisé pour:

* faire défiler les différentes positions de la bibliothèque courante,

* afficher les informations d'analyse associées à une position,

* afficher, ajouter et modifier les commentaires d'une position.

.. tip:: Se référer à :ref:`raccourcis` pour les raccourcis disponibles.

.. _mode_edit:

Le mode EDIT
------------

Le mode EDIT permet d'éditer une position en vue de l'ajouter à
la base de données, ou de définir le type de position à rechercher.
Le mode EDIT est activé en appuyant sur la touche *TAB*.
La distribution des pions, du videau, du score, et du trait peuvent être
modifiés à l'aide de la souris (voir :ref:`guide_edit_position`).

.. tip:: Se référer à :ref:`raccourcis` pour les raccourcis disponibles.

.. _mode_command:

Le mode COMMAND
---------------

Le mode COMMAND permet de réaliser l'ensemble des fonctionalités de blunderDB
disponibles à l'interface graphique: opérations générales sur la base de
données, navigation de position, affichage de l'analyse et/ou des commentaires,
recherche de positions selon des filtres... Après une première prise en main de
l'interface, il est recommandé de progressivement utiliser ce mode qui permet
une utilisation puissante et fluide de blunderDB, notamment pour les
fonctionnalités de recherche de positions.

Pour basculer dans le mode COMMAND depuis tout autre mode, appuyer sur
la touche *ESPACE*. Pour envoyer une requête et quitter le mode COMMAND,
appuyer sur la touche *ENTREE*.

blunderDB exécute les requêtes envoyées par l'utilisateur sous réserve
qu'elles soient valides et modifie immédiatement l'état de la base de données
le cas échéant. Il n'y a pas d'actions de sauvegarde explicite de la part
de l'utilisateur.

.. tip:: Se référer à la :numref:`cmd_mode` pour la liste de commandes
   disponible en mode COMMAND.

.. _mode_match:

Le mode MATCH
--------------

Le mode MATCH permet de naviguer dans les coups d'un match importé.
Il est activé en appuyant sur *CTRL-TAB* ou en exécutant la commande ``m``.

Dans ce mode, l'utilisateur peut:

* parcourir les coups d'un match en utilisant les touches *GAUCHE* et *DROITE*,

* passer d'une partie à l'autre à l'aide des touches *PageUp* et *PageDown*,

* afficher l'analyse des coups (pions et cube) en appuyant sur *CTRL-L*,

* basculer entre l'analyse des coups de pions et du cube avec la touche *d*,

* voir le coup effectivement joué mis en évidence dans l'analyse.

Lorsque l'utilisateur entre en mode MATCH, le dernier match visité est
automatiquement chargé. La dernière position visitée dans chaque match
est mémorisée et restaurée.

.. tip:: Se référer à :ref:`raccourcis` pour les raccourcis disponibles.

.. _mode_epc:

Le mode EPC
-----------

Le mode EPC permet de calculer l'EPC (Effective Pip Count) d'une position
de bearoff. Il est activé en exécutant la commande ``epc`` ou en cliquant
sur le bouton correspondant dans la barre d'outils.

Dans ce mode, l'utilisateur édite la position des pions dans le jan
(6 derniers points) et les informations suivantes sont calculées
en temps réel pour chaque joueur:

* l'EPC (Effective Pip Count),

* le nombre moyen de lancers nécessaires (Mean Rolls),

* l'écart type (Standard Deviation),

* le pip count,

* le wastage (différence entre l'EPC et le pip count).

.. note:: Le calcul repose sur la base de données interne de bearoff
   à 6 points de GNUbg.

.. _mode_collection:

Le mode COLLECTION
------------------

Le mode COLLECTION permet de parcourir les positions d'une collection.
Il est activé en double-cliquant sur une collection dans le panneau
des collections (*CTRL-K*).

Dans ce mode, l'utilisateur peut naviguer parmi les positions de la collection
en utilisant les touches *GAUCHE* et *DROITE*. Pour quitter le mode COLLECTION
et revenir au mode NORMAL, appuyer sur *TAB*.

