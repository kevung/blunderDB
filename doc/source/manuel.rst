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

* les modes de fonctionnement.

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

Modes d'utilisation
-------------------

NORMAL

EDIT
COMMAND
rédiger une requête


Afin d'interagir avec la base de données, l'utilisateur réalise des
requêtes. Pour ce faire
Le mode COMMAND est activé à l'aide de la touche ESPACE

Le mode commande permet d'interagir avec la base de données courante via
l'émission d'une requête. Après la validation de cette dernière, la base
de données est immédiatement modifiée après la validation de la 

