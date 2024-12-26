.. _guide_utilisateur:

Guide utilisateur
=================

Ce guide est une introduction pratique à blunderDB pour une prise en main
rapide.

Créer une nouvelle base de données
----------------------------------

Pour créer une nouvelle base de données, cliquer sur "File", "New Database".
Choisir un chemin où enregistrer la base de données, ainsi qu'un nom et cliquer
sur "Save".

.. note::
   L'extension des bases de données blunderDB est *.db*.

.. tip::
   Raccourcis clavier: *CTRL-N*. Requête: *:n*


Ouvrir une base de donnée existante
-----------------------------------

Pour charger une base de données existante, cliquer sur "File", "Open
Database". Choisir le chemin où se trouve la base de données, choisir le
fichier *.db* et cliquer sur "Open".

.. tip::
   Raccourcis clavier: *CTRL-O*. Requête: *:o*

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

* pour éditer les dés, clic gauche pour augmenter la valeur d'un dé, clic droit
  pour augmenter la valeur d'un dé.

* pour indiquer le joueur qui a le trait, cliquer à l'emplacement prévu des dés
  à hauteur de sommet de point.

* pour éditer le score des joueurs, clic gauche pour augmenter le score, clic
  droit pour réduire le score.

Une fois la position éditée, appuyer sur *TAB* pour rebasculer en mode NORMAL.

.. tip:: La saisie de la position avec la souris pour les pions se fait de la
   même manière que dans XG.

.. tip:: La position peut être intégralement saisie uniquement par le clavier
   (voir :ref:`raccourcis_position`). Avec un peu d'entrainement, la saisie de
   position peut ainsi être très rapide.

Ajouter une position à la base de données
-----------------------------------------

Pour enregistrer la position obtenue précédemment,

* basculer en mode COMMAND en appuyant sur *ESPACE*,

* taper la requête *:w*,

* valider la commande en appuyant sur *ENTREE*.

Un message s'affiche dans la barre d'état indiquant
"Position written to database."

Ajouter une position à une bibliothèque
---------------------------------------

Pour ajouter une position à une bibliothèque *toto*,

* basculer en mode COMMAND en appuyant sur *ESPACE*,

* taper *:w toto*,

* valider la commande en appuyant sur *ENTREE*.

Un message s'affiche dans la barre d'état indiquant
"The position has been added to toto."

Retirer une position d'une bibliothèque
---------------------------------------

Pour retirer la position précédente de la bibliothèque *toto*,

* basculer en mode COMMAND en appuyant sur *ESPACE*,

* taper *:w -toto*,

* valider la commande en appuyant sur *ENTREE*.

Un message s'affiche dans la barre d'état indiquant
"The position has been removed from toto."

Supprimer la position de la base de données
-------------------------------------------

Pour supprimer définitivement la position de la base de données,

* basculer en mode COMMAND en appuyant sur *ESPACE*,

* taper *:D*,

* valider la commande en appuyant sur *ENTREE*.

Un message s'affiche dans la barre d'état indiquant
"Position deleted."

Import une position depuis XG
-----------------------------

Pour importer une position directement depuis XG,

* afficher dans XG la position à importer et appuyer *CTRL-C*,

* afficher blunderDB et appuyer *CTRL-V*.

Un message s'affiche dans la barre d'état indiquant
"Position imported."

Afficher l'analyse d'une position importée depuis XG
----------------------------------------------------

Si une position analysée par XG a été importée dans blunderDB, l'analyse de XG
peut être affichée en appuyant *CTRL-L*.

Si la position correspond à une décision de pions, les cinq meilleurs coups
sont affichés sur des lignes distinctes. Pour chaque ligne, les informations
fournies sont dans cet ordre, le coup de pion associé, l'équité normalisée,
l'erreur en équité du coup, les chances de gain, gammon et backgammon du joueur
1, les chances de gain, gammon et backgammon du joueur 2, le niveau d'analyse. 

Exporter une position vers XG
-----------------------------

Pour exporter une position de blunderDB vers XG,

* afficher dans blunderDB la position à exporter et appuyter *CTRL-C*,

* afficher XG et appuyer *CTRL-V*.

Visualiser les différentes positions
------------------------------------

Pour visualiser les différentes positions de la bibliothèque courante, utiliser
les touches *GAUCHE* et *DROITE*. La touche *HOME* permet d'aller à la première
position. La touche *FIN* permet d'aller à la dernière position.

Pour afficher le bearoff à gauche, appuyer *CTRL-GAUCHE*. Pour afficher le
bearoff à droite, appuyer *CTRL-DROITE*.

Pour inverser l'orientation du board, appuyer sur *CTRL-HAUT* ou *CTRL-BAS*.

Rechercher des positions selon des critères
-------------------------------------------

Pour rechercher des types de positions,

* basculer en mode EDIT en appuyant sur *TAB*,

* éditer la structure de position à rechercher. blunderDB va filtrer les
  positions ayant *a minima* la structure de pions saisie. Dans le
  doute, afin d'obtenir le maximum de résultats, effacer la position
  en appuyant sur la touche *RETOUR ARRIERE*. Editer si besoin la
  position du cube et le score.

* basculer en mode COMMAND en appuyant sur *ESPACE*,

* écrire *:s*, ajouter d'éventuels filtres supplémentaires (par exemple
  *cube* ou *score* pour prendre respectivement en compte le cube et le
  score. Voir :ref:`cmd_filter_pos` pour une liste exhaustive des
  filtres disponibles).

* valider la requête en appuyant sur *ENTREE*.

Les positions affichées sont celles de la base de données ayant vérifié
les critères de recherche entrés par l'utilisateur.

