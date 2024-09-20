.. _manuel:

Manuel
======

blunderDB est un logiciel pour consistuer des bases de données de
positions. Les positions sont stockées dans une base de données représentée par un fichier
*.db*.

Les principales interactions possibles avec blunderDB sont:

* ajouter une nouvelle position,

* modifier une position existante,

* supprimer une position existante,

* rechercher une ou plusieurs positions.

Pour ce faire, l'utilisateur bascule dans des modes dédiés pour la
visualisation (mode NORMAL), l'édition de positions (mode EDIT),
l'édition d'une requête pour filtrer des positions (mode COMMAND).

Dans la suite du manuel, il est décrit:

* l'interface graphique,

* les modes de blunderDB.

Description de l'IHM
--------------------

L'IHM de blunderDB est constituée de haut en bas par:

* [en haut] la barre de menus, qui rassemble l'ensemble des principales
  opérations réalisables sur la base de données,

* [au milieu] la zone d'affichage principale, qui permet d'afficher ou d'éditer des
  positions de backgammon,

* [en bas] la barre d'état, qui présente différentes informations sur la
  base de donnnées ou la position courante.

La zone d'affichage principale met à disposition à l'utilisateur:

* un board afin d'afficher ou d'éditer une position de backgammon,

* le niveau et le propriétaire du cube,

* le compte de course de chaque joueur,

* le score de chaque joueur,

* les dés à jouer. Si aucune valeur est affichée sur les dés, la
  position des dés indique quel joueur a le trait et que la position est
  une décision de cube.

La barre d'état est structurée de gauche à droite par les informations
suivantes:

* le mode courant (NORMAL, EDIT, COMMAND),

* le nom de la bibliothèque courante. Toutes les positions sont ajoutés
  à la bibliothèque principale intitulée *main*,

* l'index de la position courante, suivi du nombre de positions dans la
  bibliothèque courante. Dans le cas de positions issus d'une recherche
  par l'utilisateur, le nombre de positions correspond au nombre de
  positions filtrées,

* un message d'information.

.. _mode_normal:

Le mode NORMAL
--------------

Le mode NORMAL est le mode par défaut de blunderDB. Il est utilisé pour:

* faire défiler les différentes positions de la bibliothèque courante,

* afficher les informations d'analyse associées à une position.

.. tip:: Se référer à la section :ref:`raccourcis_modaux` pour les
   raccourcis de navigation du mode NORMAL.

.. _mode_edit:

Le mode EDIT
------------

Le mode EDIT permet d'éditer une position en vue où bien de l'ajouter à
la base de données, ou bien de définir le type de position à rechercher.
Le mode EDIT est activé en appuyant sur la touche *TAB*.
La distributions des pions, du videau, du score, du trait oeuvent être
modifiés à l'aide de la souris (voir :ref:`guide_edit_position`) ou du clavier (voir
:ref:`raccourcis_position`).

.. tip:: Se référer à la section :ref:`raccourcis_modaux` pour les
   raccourcis de navigation du mode EDIT.

Le mode COMMAND
---------------

Le mode COMMAND permet à l'utilisateur d'émettre une requête à la base
de données afin de:

* ajouter une nouvelle position ou mettre à jour une position existante,

* ajouter une position dans une bibliothèque,

* renommer, copier, supprimer une bibliothèque,

* lister les bibliothèques existantes,

* rechercher des types de positions selon divers critères librement
  combinables.

Pour basculer dans le mode COMMAND depuis tout autre mode, appuyer sur
la touche *ESPACE*. Pour envoyer une requête et quitter le mode COMMAND,
appuyer sur la touche *ENTREE*.

blunderDB exécute les requêtes envoyées par l'utilisateur sous réserve
qu'elles soient valides et modifie immédiatement l'état de la base de données
le cas échéant. Il n'y a pas d'actions de sauvegarde explicite de la part
de l'utilisateur.

.. tip:: Se référer à la section :ref:`raccourcis_modaux` pour les
   raccourcis de navigation du mode COMMAND.

