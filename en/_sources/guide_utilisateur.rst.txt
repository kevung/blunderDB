.. _guide_utilisateur:

Guide utilisateur
=================

Ce guide est une introduction pratique à blunderDB pour une prise en main
rapide.

Créer une nouvelle base de données
----------------------------------

Pour créer une nouvelle base de données, cliquer dans la barre d'outils sur le
bouton "New Database". Choisir un chemin où enregistrer la base de données,
ainsi qu'un nom et cliquer sur "Save".

.. note::
   L'extension des bases de données blunderDB est *.db*.

.. tip::
   Raccourcis clavier: *CTRL-N*. Commande: ``n``


Ouvrir une base de donnée existante
-----------------------------------

Pour charger une base de données existante, cliquer dans la barre d'outils sur
le bouton "Open Database". Choisir le chemin où se trouve la base de données,
choisir le fichier *.db* et cliquer sur "Open".

.. tip::
   Raccourcis clavier: *CTRL-O*. Commande: ``o``

Importer et fusionner une base de données
-----------------------------------------

Pour importer et fusionner une autre base de données blunderDB dans la base de
données actuellement ouverte, cliquer dans la barre d'outils sur le bouton
"Import Database". Choisir le fichier *.db* à importer et cliquer sur "Open".

blunderDB va fusionner intelligemment les deux bases de données:

* Les positions qui n'existent pas dans la base de données actuelle seront
  ajoutées avec leurs analyses et commentaires.

* Les positions qui existent déjà seront mises à jour: les analyses seront
  complétées si manquantes, et les commentaires seront fusionnés (ajout des
  nouveaux commentaires sans dupliquer les existants).

* Un message résumera le nombre de positions ajoutées, fusionnées et ignorées.

.. note::
   L'import nécessite que les deux bases de données aient des versions de schéma
   compatibles. Il est possible d'importer une base de données d'une version
   inférieure ou égale dans une base de données de version supérieure.

.. caution::
   L'opération d'import modifie immédiatement la base de données actuellement
   ouverte. Il est recommandé de faire une copie de sauvegarde avant d'importer
   une autre base de données.

.. _guide_edit_position:

Editer une position
-------------------

Pour éditer une position, basculer en mode EDIT à l'aide de la touche *TAB*.
Editer la position à la souris:

* cliquer sur les points pour ajouter des pions. Le clic gauche attribue les
  pions au joueur 1. Le clic droit attribue les pions au joueur 2. Pour insérer
  une prime, cliquer sur le point de départ, maintenir le bouton appuyé,
  relacher sur le point d'arrivée. Cliquer sur la barre pour mettre des
  pions à la barre.

* pour effacer la position, double-clic sur une zone vide en dehors du board ou
  appuyer sur la touche *RETOUR ARRIERE*.

* pour envoyer le cube vers le joueur 1, clic gauche sur le cube. Pour envoyer
  le cube vers le joueur 2, click droit sur le cube.

* pour indiquer le joueur qui a le trait, cliquer à l'emplacement prévu des dés.

* pour éditer les dés, clic gauche pour augmenter la valeur d'un dé, clic droit
  pour augmenter la valeur d'un dé. Si la face des dés est vide, cela signifie
  que la position est une décision de cube.

* pour éditer le score des joueurs, clic gauche pour augmenter le score, clic
  droit pour réduire le score.

.. tip:: La saisie de la position avec la souris pour les pions se fait de la
   même manière que dans XG.

Ajouter une position à la base de données
-----------------------------------------

Après l'édition de la position précédente, blunderDB est dans le mode EDIT.

Pour enregistrer la position obtenue précédemment, faire *CTRL-S* ou appuiyer
dans la barre d'outils sur le bouton "Save Position".

.. tip:: Depuis le mode EDIT, basculer en mode COMMAND et exécuter: ``w``

Etiqueter une position
----------------------

Pour ajouter un tag *toto* à la position courante, basculer en mode COMMAND en appuyant sur *ESPACE*,
taper ``#toto`` et valider la commande en appuyant sur *ENTREE*.

Supprimer une position
----------------------

Pour supprimer la position courante de la base de données, faire *Del* ou
clicker dans la barre d'outils sur le bouton "Delete Position"

.. tip:: En mode COMMAND, exécuter ``d``.

.. caution:: La suppression de la position est définitive et ne nécessite
   aucune confirmation de la part de l'utilisateur.

Import une position depuis XG
-----------------------------

Pour importer une position directement depuis XG,

#. afficher dans XG la position à importer et appuyer *CTRL-C*,

#. afficher blunderDB et appuyer *CTRL-V*.

Afficher l'analyse d'une position importée depuis XG
----------------------------------------------------

Si une position analysée par XG a été importée dans blunderDB, l'analyse de XG
peut être affichée en appuyant *CTRL-L*.

Si la position correspond à une décision de pions, les cinq meilleurs coups
sont affichés sur des lignes distinctes. Pour chaque ligne, les informations
fournies sont dans cet ordre, le coup de pion associé, l'équité normalisée,
l'erreur en équité du coup, les chances de gain, gammon et backgammon du
joueur, les chances de gain, gammon et backgammon de l'adversaire, le niveau
d'analyse. 

Si la position correspond à une décision de cube, le coût de chaque décision
est affiché ainsi que les chances de gain de la position.

Exporter une position vers XG
-----------------------------

Pour exporter une position de blunderDB vers XG,

#. afficher dans blunderDB la position à exporter et appuyter *CTRL-C*,

#. afficher XG et appuyer *CTRL-V*.

Visualiser les différentes positions
------------------------------------

Pour visualiser les différentes positions de la bibliothèque courante, utiliser
les touches *GAUCHE* et *DROITE*. La touche *HOME* permet d'aller à la première
position. La touche *FIN* permet d'aller à la dernière position.

Pour afficher le bearoff à gauche, appuyer *CTRL-GAUCHE*. Pour afficher le
bearoff à droite, appuyer *CTRL-DROITE*.

Rechercher des positions selon des critères
-------------------------------------------

Pour rechercher des types de positions,

* basculer en mode EDIT en appuyant sur *TAB*,

* éditer la structure de position à rechercher. blunderDB va filtrer les
  positions ayant *a minima* la structure de pions saisie. Dans le
  doute, afin d'obtenir le maximum de résultats, effacer la position
  en appuyant sur la touche *RETOUR ARRIERE*. Editer si besoin la
  position du cube et le score.

Méthode 1 (simple): 

* Ouvrir la fenêtre de recherche (*CTRL-F*)

* Ajouter et paramétrer les filtres de recherche

* Valider en cliquant sur "Search".

Méthode 2 (avancée):


* basculer en mode COMMAND en appuyant sur *ESPACE*,

* écrire *s*, ajouter d'éventuels filtres supplémentaires (par exemple
  *cube* ou *score* pour prendre respectivement en compte le cube et le
  score. Voir :numref:`cmd_filter` pour une liste exhaustive des
  filtres disponibles).

* valider la requête en appuyant sur *ENTREE*.

Les positions affichées sont celles de la base de données ayant vérifié
les critères de recherche entrés par l'utilisateur.

