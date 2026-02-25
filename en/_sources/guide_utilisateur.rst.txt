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

.. note::
   Le collage automatique détecte le format de la source (XG, GNUbg, BGBlitz).

Importer un match
-----------------

blunderDB peut importer des matchs depuis différentes sources.

**Formats supportés:**

* eXtreme Gammon (XG): fichiers *.xg*
* GNUbg: fichiers *.sgf*
* Jellyfish: fichiers *.mat* et *.txt*
* BGBlitz: fichiers *.bgf* et *.txt*

**Pour importer un ou plusieurs fichiers de match:**

#. Appuyer sur *CTRL-I* ou cliquer sur le bouton "Import" dans la barre d'outils.

#. Sélectionner un ou plusieurs fichiers à importer.

#. blunderDB détecte automatiquement le format et importe le match.

#. Une fenêtre de progression affiche le nombre de fichiers importés, échoués
   et ignorés (doublons).

.. tip::
   Commande: ``i``

.. note::
   blunderDB détecte automatiquement les doublons et empêche l'import d'un
   match déjà présent dans la base de données.

Importer un dossier de matchs
------------------------------

Pour importer récursivement tous les fichiers de matchs contenus dans un
dossier et ses sous-dossiers:

#. Appuyer sur *CTRL-SHIFT-F* ou cliquer sur le bouton correspondant dans la
   barre d'outils.

#. Sélectionner le dossier contenant les fichiers de matchs.

#. blunderDB collecte et importe automatiquement tous les fichiers reconnus
   (*.xg*, *.sgf*, *.mat*, *.txt*, *.bgf*).

Glisser-déposer
----------------

blunderDB supporte le glisser-déposer. Il est possible de glisser-déposer
sur la fenêtre de blunderDB:

* des fichiers de match ou de position (*.xg*, *.sgf*, *.mat*, *.txt*, *.bgf*)
  pour les importer,

* des fichiers de base de données (*.db*) pour les ouvrir ou les fusionner
  avec la base de données courante,

* des dossiers pour importer récursivement tous les fichiers qu'ils contiennent.

Naviguer dans un match
-----------------------

Pour naviguer dans un match importé:

#. Ouvrir le panneau des matchs avec *CTRL-T*.

#. Double-cliquer sur un match pour entrer en mode MATCH.

#. Utiliser les touches *GAUCHE* / *DROITE* pour parcourir les coups.

#. Utiliser *PageUp* / *PageDown* pour passer d'une partie à l'autre.

#. Appuyer sur *CTRL-L* pour afficher l'analyse.

#. Appuyer sur *d* pour basculer entre l'analyse des coups de pions et du cube.

.. tip::
   Raccourci: *CTRL-TAB* pour basculer en mode MATCH / sortir du mode MATCH.
   Commande: ``m``

.. note::
   blunderDB mémorise la dernière position visitée dans chaque match. En
   revenant sur un match, la dernière position consultée est automatiquement
   restaurée.

Gérer le panneau des matchs
-----------------------------

Le panneau des matchs (*CTRL-T*) permet de:

* lister l'ensemble des matchs importés (triés du plus récent au plus ancien),

* trier les matchs par colonnes (joueur 1, joueur 2, date, longueur du match,
  tournoi),

* modifier les noms des joueurs ou la date en double-cliquant sur les champs,

* permuter les joueurs 1 et 2 à l'aide du bouton de permutation,

* assigner un match à un tournoi,

* supprimer un match à l'aide de la touche *Del*.

Gérer les collections
---------------------

Les collections permettent d'organiser des positions en groupes personnalisés.
Pour accéder au panneau des collections, appuyer sur *CTRL-K*.

**Créer une collection:**

#. Ouvrir le panneau des collections (*CTRL-K*).

#. Saisir le nom de la nouvelle collection et cliquer sur "Add".

**Ajouter des positions à une collection:**

#. Sélectionner les positions souhaitées.

#. Les ajouter à la collection depuis le panneau des collections.

**Parcourir une collection:**

* Double-cliquer sur une collection pour entrer en mode COLLECTION et
  parcourir ses positions.

.. tip::
   Commande: ``coll``

Gérer les tournois
------------------

Les tournois permettent d'organiser les matchs importés par événement.
Pour accéder au panneau des tournois, appuyer sur *CTRL-Y*.

**Créer un tournoi:**

#. Ouvrir le panneau des tournois (*CTRL-Y*).

#. Cliquer sur "Add" et saisir le nom du tournoi.

**Assigner un match à un tournoi:**

* Depuis le panneau des matchs (*CTRL-T*), utiliser le menu déroulant
  de la colonne tournoi pour assigner un match.

Calculer l'EPC
--------------

Le calculateur EPC (Effective Pip Count) permet de calculer les statistiques de
bearoff d'une position.

#. Exécuter la commande ``epc`` ou cliquer sur le bouton correspondant dans la
   barre d'outils.

#. Éditer la position des pions dans le jan (6 derniers points).

#. Les résultats sont affichés en temps réel: EPC, nombre moyen de lancers,
   écart type, pip count et wastage.

.. note::
   Le calculateur fonctionne pour les deux joueurs simultanément.

Afficher l'analyse d'une position importée depuis XG
----------------------------------------------------

Si une position analysée par XG, GNUbg ou BGBlitz a été importée dans
blunderDB, l'analyse peut être affichée en appuyant *CTRL-L*.

Si la position correspond à une décision de pions, les cinq meilleurs coups
sont affichés sur des lignes distinctes. Pour chaque ligne, les informations
fournies sont dans cet ordre, le coup de pion associé, l'équité normalisée,
l'erreur en équité du coup, les chances de gain, gammon et backgammon du
joueur, les chances de gain, gammon et backgammon de l'adversaire, le niveau
d'analyse. 

Si la position correspond à une décision de cube, le coût de chaque décision
est affiché ainsi que les chances de gain de la position.

Lorsque plusieurs moteurs d'analyse sont présents pour la même position
(par exemple XG et GNUbg), une colonne supplémentaire indique le moteur
d'origine de chaque analyse.

En mode MATCH, le coup effectivement joué est mis en évidence dans la liste
des coups. En mode NORMAL, si la position a été rencontrée dans plusieurs
matchs, tous les coups joués sont indiqués.

.. tip::
   En cliquant sur un coup dans le panneau d'analyse, les flèches
   correspondantes sont affichées sur le board.

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

