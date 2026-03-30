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
Elle est activée depuis le panneau des matchs (*CTRL-T*) en double-cliquant
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

