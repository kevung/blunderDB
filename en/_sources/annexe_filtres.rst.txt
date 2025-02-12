.. _annexe_filtres:

Annexe: Utilisation avancée des filtres
=======================================

.. warning::

   Cette section s'adresse aux utilisateurs avancés de blunderDB qui souhaitent
   exploiter pleinement les fonctionnalités de recherche de positions.

Les filtres sont au coeur de l'analyse des positions dans blunderDB.
Leur utilisation permet de rechercher des positions spécifiques relativement
précisément. Dans cette section, l'utilisation des filtres via le mode COMMAND
est détaillée. Le mode COMMAND est accessible en appuyant sur la touche ``ESPACE``.
Il permet de combiner avec l'habitude très rapidement des filtres et d'utiliser 
la bibliothèque de filtres.

Recherche de positions en ligne de commande
-------------------------------------------

Pour faire une recherche à l'aide de filtres, 

1. Basculer en mode EDIT avec la touche ``TAB``.
2. Editer la position courante.
3. Basculer en mode COMMAND avec la touche ``ESPACE``.
4. Utiliser la commande ``s`` suivie éventuellement de filtres.
5. Lancer la recherche avec la touche ``ENTREE``.

.. warning::
   Ne pas oublier d'effacer la position courante avant de lancer une recherche 
   (touche ``RETOUR ARRIERE``), si cette dernière n'est pas celle souhaitée, au 
   risque de filtrer abusivement des structures de pions.

.. note:: 
   La liste des filtres disponibles en mode COMMAND est fournie dans la
   :numref:`cmd_filter`.

Bibliotheque de filtres
-----------------------

La bibliothèque de filtres permet à l'utilisateur d'enregistrer des commandes de recherche
afin de faciliter ses études thématiques.

Pour ajouter un filtre à la bibliothèque,

1. Basculer dans le mode EDIT. 
2. Ouvrir la bibliothèque de filtre en appuyant sur ``CTRL-B``.
3. Editer la position courante.
4. Donner un nom au filtre.
5. Editer la commande de recherche.
6. Sauvegarder la commande de recherche à l'aide du bouton "Add".

.. tip::
   Lors de l'édition de la commande, il est possible d'utiliser les touches
   ``HAUT`` et ``BAS`` pour naviguer dans l'historique des commandes.

Pour utiliser un filtre enregistré dans la bibliothèque,

1. Basculer en mode NORMAL.
2. Ouvrir la bibliothèque de filtre en appuyant sur ``CTRL-B``.
3. Rechercher le filtre souhaité.
4. Double cliquer sur le filtre pour lancer la recherche.

Exemples
--------

Voici quelques exemples d'utilisation des filtres en mode COMMAND:

.. csv-table::
   :header: "Type de position", "Structure de pions", "Commande"
   :widths: 10, 10, 20
   :align: center

   "Courses", "", "s nc"
   "Courses petites", "", s nc P<70
   "Frapper au point 1", "", s m\"6/1*\"
   "Backgame 1-4", "portes 24, 21", s p>35
   "Décision de Take/Pass à -2 -4", "dés vides côté joueur du haut, score -2/-4", s s d
   "Envoi de too good", "dés vides côté joueur du bas", s d e>1000
   "Blitz avec au moins 20% de gammon", "portes dans le jan, hommes à la barre", s g>20
   "Position du tournoi Aachen2024", "", s t\"Aachen2024\"
   "Un pion arriéré à ramener", "", "s k1,1"
   "Quitter le point 20", "point 20", s m\"20/\"
   "Prime contre prime", "indiquer les primes", s
   "Ace-point bear-off", "point 1 pour l'adversaire", s P<60
   "Double avec au moins 20pip d'avance", "dés vides côté joueur du bas", s d p<-20

