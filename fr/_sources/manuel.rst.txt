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

* rechercher une ou plusieurs positions.

Modes
-----

Pour ce faire, l'utilisateur bascule dans des modes dédiés pour:

* la navigation et la visualisation de positions (mode NORMAL),

* l'édition de positions (mode EDIT),

* l'édition d'une requête pour filtrer des positions (mode COMMAND ou fenêtre de recherche).

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

* afficher les données d'analyse associées à la position courante issues d'eXtreme Gammon (XG),

* afficher, ajouter ou modifier des commentaires

Des fenêtres modales peuvent s'afficher pour:

* [mode EDIT uniquement] paramétrer les filtres de recherche,

* afficher l'aide de blunderDB.

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

