.. _raccourcis:

Raccourcis clavier
==================

.. note::

   Les raccourcis clavier sont indépendants de la disposition du clavier : ils
   restent accessibles de la même manière quelle que soit la disposition
   utilisée (AZERTY, QWERTY, QWERTZ, etc.).

.. note::

   Lorsque le curseur se trouve dans une zone de saisie (commentaire, champ de
   recherche, ligne de commande), les raccourcis d'édition de texte habituels
   s'appliquent au texte et non à la position : CTRL-C, CTRL-X et CTRL-V
   copient, coupent et collent la sélection, CTRL-A la sélectionne
   entièrement, CTRL-Z et CTRL-Y annulent et rétablissent.

.. _raccourcis_generaux:

Base de données
---------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "CTRL-N", "Créer une nouvelle base de données."
   "CTRL-O", "Ouvrir une base de données existante."
   "CTRL-SHIFT-I", "Importer une base de données."
   "CTRL-SHIFT-S", "Exporter la base de données."
   "CTRL-Q", "Fermer blunderDB."
   "CTRL-M", "Modifier les métadonnées de la base de données."

.. _raccourcis_position:

Position
--------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "CTRL-I", "Importer une ou plusieurs positions/matchs par fichier (xg, xgp, sgf, mat, txt, bgf)."
   "CTRL-SHIFT-F", "Importer récursivement un dossier de fichiers de matchs/positions."
   "CTRL-C", "Copier une position dans le presse-papier."
   "CTRL-X", "Copier l'image du board dans le presse-papier (PNG)."
   "CTRL-X CTRL-X", "Copier l'image du board avec l'analyse dans le presse-papier (PNG)."
   "CTRL-V", "Coller une position depuis le presse-papier (détection automatique du format)."
   "CTRL-S", "Enregistrer une position."
   "CTRL-U", "Mettre à jour une position."
   "Del", "Supprimer la position courante (confirmation demandée)."
   "RETOUR ARRIERE", "Réinitialiser le board, le cube, le score et les dés."
   "CTRL-G", "Afficher les métadonnées de la position."

.. note::

   Dans le panneau Eval (voir :ref:`panneau_epc`), RETOUR ARRIERE réinitialise
   vers les valeurs propres à ce panneau (score money, pas de dés posés) plutôt
   que vers celles du mode édition (score 7 partout, dés 3-1). Un double-clic
   en dehors du plateau produit la même réinitialisation.

.. note::

   Dans le panneau Eval et dans le panneau Recherche, le plateau est un
   brouillon et non une position de la base : CTRL-V y **pose la position sur
   le plateau** au lieu de l'importer dans la base, et CTRL-C copie le plateau
   affiché — son XGID est recalculé à partir des pions posés, sans l'analyse
   de la position consultée auparavant. La position copiée se colle ainsi
   telle quelle dans eXtreme Gammon ou dans une autre instance de blunderDB.

.. _raccourcis_navigation:

Navigation
----------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "CTRL-R", "Recharger toutes les positions de la base de données."
   "PageUp, h", "Première position / Partie précédente (navigation match)."
   "GAUCHE, k", "Position précédente."
   "DROITE, j", "Position suivante."
   "HAUT, k", "Coup précédent (lorsqu'un coup est sélectionné dans l'analyse)."
   "BAS, j", "Coup suivant (lorsqu'un coup est sélectionné dans l'analyse)."
   "PageDown, l", "Dernière position / Partie suivante (navigation match)."
   "r", "Charger une position aléatoire."

.. _raccourcis_affichage:

Affichage
---------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "CTRL-GAUCHE", "Orientation du board à gauche."
   "CTRL-DROITE", "Orientation du board à droite."
   "p", "Afficher/cacher le compte de course."

.. _raccourcis_modes:

Actions
-------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "TAB", "Ouvrir le panneau de recherche (éditeur de position)."
   "ESPACE", "Ouvrir la ligne de commande."

.. note::

   TAB n'ouvre le panneau de recherche que lorsque le focus se trouve sur le
   plateau (ou nulle part en particulier, ce qui est le cas la plupart du
   temps). Une fois le focus posé sur un bouton, un champ de saisie ou un
   lien, TAB reprend la navigation clavier standard entre les éléments de
   l'interface plutôt que de rouvrir ce panneau.

.. _raccourcis_outils:

Outils
------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "CTRL-L", "Afficher/cacher l'analyse."
   "CTRL-P", "Afficher/cacher les commentaires."
   "CTRL-K", "Afficher/cacher le panneau Anki (répétition espacée)."
   "CTRL-F", "Afficher/cacher le panneau de recherche."
   "CTRL-Tab", "Afficher/cacher le panneau des matchs."
   "CTRL-B", "Afficher/cacher le panneau des collections."
   "CTRL-Y", "Afficher/cacher le panneau des tournois."
   "CTRL-D", "Afficher/cacher le panneau Stats."
   "CTRL-E", "Afficher/cacher le panneau Eval."
   "?", "Afficher/cacher l'aide."

.. _raccourcis_vues:

Onglets de vues
---------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "CTRL-T", "Créer une nouvelle vue (copie de la vue courante)."
   "CTRL-W", "Fermer la vue courante."
   "CTRL-PageUp, MAJ-J", "Vue précédente."
   "CTRL-PageDown, MAJ-K", "Vue suivante."
   "CTRL-1 … CTRL-9", "Aller directement à la n-ième vue."
   "Double-clic sur l'onglet", "Renommer la vue."

.. note::

   Le sens de MAJ-J / MAJ-K est inversé par rapport à j / k : *j* avance
   (position suivante) et *k* recule (position précédente), alors que
   *MAJ-J* revient à la vue précédente et *MAJ-K* passe à la vue suivante.
   C'est voulu (pas un raccourci à corriger) — MAJ-J/MAJ-K suivent la
   convention CTRL-PageUp/CTRL-PageDown à laquelle ils sont associés, pas
   celle de j/k.

.. _raccourcis_command:

Ligne de commande
-----------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "HAUT", "Parcourir l'historique des commandes vers le haut."
   "BAS", "Parcourir l'historique des commandes vers le bas."

.. _raccourcis_search_history:

Historique de recherche
-----------------------

L'historique de recherche est l'onglet *Historique* du panneau de recherche
(*CTRL-F* ou *TAB*).

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic", "Sélectionner/désélectionner une recherche (afficher la position)."
   "Double-clic", "Exécuter la recherche."

.. _raccourcis_filter_library:

Bibliothèque de filtres
-----------------------

La bibliothèque de filtres est l'onglet *Enregistrés* du panneau de recherche
(*CTRL-F* ou *TAB*).

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic", "Sélectionner/désélectionner un filtre (afficher la position)."
   "Double-clic", "Exécuter la recherche du filtre."

.. _raccourcis_analysis:

Panneau d'analyse
-----------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic", "Sélectionner/désélectionner un coup (afficher/cacher les flèches)."
   "HAUT, k", "Sélectionner le coup précédent (lorsqu'un coup est sélectionné)."
   "BAS, j", "Sélectionner le coup suivant (lorsqu'un coup est sélectionné)."
   "d", "Basculer entre l'analyse des coups et du cube (navigation match uniquement)."
   "Esc", "Désélectionner le coup. Si aucun coup sélectionné, fermer le panneau."

.. _raccourcis_eval_panel:

Panneau Eval
------------

La liste des coups du panneau *Eval* (voir :ref:`panneau_epc`) se parcourt
comme celle du panneau d'analyse. Ces raccourcis agissent dès qu'un coup a été
sélectionné d'un clic.

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic", "Sélectionner/désélectionner un coup (afficher/cacher les flèches)."
   "HAUT, k", "Sélectionner le coup précédent (lorsqu'un coup est sélectionné)."
   "BAS, j", "Sélectionner le coup suivant (lorsqu'un coup est sélectionné)."
   "Esc", "Désélectionner le coup."

.. _raccourcis_match_panel:

Panneau des matchs
------------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic", "Sélectionner un match."
   "Double-clic", "Naviguer dans le match."
   "HAUT, k", "Sélectionner le match précédent."
   "BAS, j", "Sélectionner le match suivant."
   "ENTREE", "Charger le match sélectionné."
   "Del", "Supprimer le match sélectionné."
   "Esc", "Désélectionner/fermer le panneau."

.. _raccourcis_anki_panel:

Panneau Anki (répétition espacée)
----------------------------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "ESPACE, Clic", "Afficher la réponse (l'analyse enregistrée de la position)."
   "1", "Évaluer : À revoir (échec, revoir bientôt)."
   "2", "Évaluer : Difficile."
   "3", "Évaluer : Bien."
   "4", "Évaluer : Facile."
   "p", "Afficher/cacher le compte de course (identique au raccourci général, disponible pendant la révision)."
   "Esc", "Arrêter la révision et revenir à la liste des paquets (reprise possible)."

.. _raccourcis_tournament_panel:

Panneau des tournois
---------------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic, Double-clic", "Sélectionner un tournoi (afficher son détail)."
   "HAUT, k", "Sélectionner le tournoi précédent."
   "BAS, j", "Sélectionner le tournoi suivant."
   "Double-clic (sur un match du tournoi)", "Naviguer dans le match."
   "Esc", "Annuler l'édition en cours, sinon effacer la recherche d'ajout de match, sinon désélectionner le tournoi, sinon fermer le panneau (par paliers)."

.. _raccourcis_collection_panel:

Panneau des collections
------------------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic", "Ajouter/retirer la position courante de la collection survolée."
   "Double-clic", "Ouvrir la collection."
   "Del", "Retirer la position courante (ou les positions cochées) de la collection ouverte."
   "Esc", "Revenir à la liste des collections, sinon désélectionner la collection, sinon fermer le panneau (par paliers)."

.. note::

   Ce panneau ne capture pas les raccourcis de navigation :
   PageUp/h, GAUCHE/k, DROITE/j, PageDown/l naviguent dans les positions de
   la collection ouverte exactement comme décrit dans
   :ref:`raccourcis_navigation`.

