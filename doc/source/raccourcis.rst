.. _raccourcis:

Raccourcis clavier
==================

Les infobulles de la barre d'outils rappellent la touche de chaque bouton, sous
le nom qu'elle porte sur le clavier de la langue de l'interface : *Gauche*,
*Suppr*, *Page préc.*, *Page suiv.*, *Maj*. Les tableaux ci-dessous et la
modale d'aide emploient les mêmes noms.

.. _raccourcis_generaux:

Base de données
---------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "CTRL-N", "Créer une nouvelle base de données."
   "CTRL-O", "Ouvrir une base de données existante."
   "CTRL-MAJ-I", "Fusionner une base de données dans celle-ci."
   "CTRL-MAJ-S", "Exporter la base de données."
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
   "CTRL-MAJ-F", "Importer récursivement un dossier de fichiers de matchs/positions."
   "CTRL-C", "Copier une position dans le presse-papier."
   "CTRL-X", "Copier l'image du board dans le presse-papier (PNG)."
   "CTRL-X CTRL-X", "Copier l'image du board avec l'analyse dans le presse-papier (PNG)."
   "CTRL-V", "Coller une position depuis le presse-papier (détection automatique du format)."
   "CTRL-S", "Enregistrer une position."
   "CTRL-U", "Mettre à jour une position."
   "Del", "Supprimer la position courante (confirmation demandée)."
   "RETOUR ARRIERE", "En mode édition ou Eval : réinitialiser le board, le cube, le score et les dés."
   "CTRL-G", "Afficher les métadonnées de la position."
   "b", "Mettre la position affichée sur la Pile (collection « à revoir plus tard »), ou l'en retirer."
   "Double-clic hors du plateau", "Mettre la position affichée sur la Pile, ou l'en retirer, dans tous les modes."
   "Clic droit hors du plateau (édition, Eval)", "Ouvrir le menu du plateau : *Effacer la position*, comme RETOUR ARRIERE, ou *Position de départ*."
   "Clic droit sur le plateau (coup joué au plateau)", "Ouvrir le menu du plateau, qui commence par *Recommencer* : le coup est remis à zéro, pas la position."

.. _raccourcis_navigation:

Navigation
----------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "CTRL-R", "Recharger toutes les positions de la base de données."
   "Début, h", "Première position / Partie précédente (navigation match)."
   "Page préc.", "Recule d'une page (cent positions par défaut, réglable dans les Paramètres > Interface : 10, 50, 100, 500 ou 1 000 positions, ou 10 % de la liste ; au début de la liste, s'y arrête) ; dans un match, partie précédente."
   "GAUCHE, k", "Position précédente."
   "DROITE, j", "Position suivante."
   "HAUT, k", "Coup précédent (lorsqu'un coup est sélectionné dans l'analyse)."
   "BAS, j", "Coup suivant (lorsqu'un coup est sélectionné dans l'analyse)."
   "Fin, l", "Dernière position / Partie suivante (navigation match)."
   "Page suiv.", "Avance d'une page (même pas que *Page préc.* ; à la fin de la liste, s'y arrête) ; dans un match, partie suivante."
   "r", "Charger une position aléatoire."
   "ÉCHAP", "Quitter les résultats d'une recherche ``ss`` lancée depuis une collection ou un match : retour à la collection, ou au match sur le coup étudié."

Tant qu'une question du :ref:`panneau Entraînement <panneau_entrainement>` est
posée sur le plateau, les touches qui parcourent la liste ne la font pas
défiler : la question garde le plateau. Sur une décision de pions, le panneau
ayant le focus : RETOUR ARRIÈRE défait le dernier pas, ÉCHAP recommence le coup,
ENTRÉE le valide quand il est complet, comme un clic sur les dés ; dans le champ
de notation, ENTRÉE valide le coup tapé (``13/7 8/7``).

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
   "m", "Activer/désactiver le défi : l'analyse du panneau Analyse est masquée jusqu'à un clic, à chaque position."

.. _raccourcis_modes:

Actions
-------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "TAB", "Ouvrir le panneau de recherche (éditeur de position)."
   "ESPACE", "Ouvrir la ligne de commande."
   "ALT-1 … ALT-9", "Lancer le filtre épinglé de ce rang dans la bibliothèque de filtres."

.. _raccourcis_outils:

Outils
------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "CTRL-L", "Afficher/cacher l'analyse."
   "CTRL-MAJ-L", "Classer les positions voisines de la position courante."
   "CTRL-P", "Afficher/cacher les commentaires."
   "CTRL-MAJ-P", "Ouvrir/fermer la palette de commandes : commandes, onglets, filtres et matchs, retrouvés par un nom approché."
   "CTRL-J", "Afficher/cacher le panneau Entraînement."
   "CTRL-H", "Afficher/cacher le panneau Duel (un match contre le Bot)."
   "ENTRÉE (Duel)", "Valider le coup arrangé au plateau ; rien ne se reprend après."
   "ESPACE (Duel)", "Valider le coup arrangé au plateau, une fois tous les dés joués ; sans effet sur un coup partiel ou hors de votre tour."
   "RETOUR ARRIÈRE (Duel)", "Replacer les pions avant la validation."
   "CTRL-K", "Afficher/cacher le panneau Anki (répétition espacée)."
   "CTRL-F", "Afficher/cacher le panneau de recherche."
   "CTRL-Tab", "Afficher/cacher le panneau des matchs."
   "CTRL-B", "Afficher/cacher le panneau des collections."
   "CTRL-Y", "Afficher/cacher le panneau des tournois."
   "CTRL-D", "Afficher/cacher le panneau Stats."
   "CTRL-E", "Afficher/cacher le panneau Eval."
   "CTRL-MAJ-T", "Afficher/cacher le panneau Transcription (brouillons de matchs)."
   "J / K", "Dans la file des propositions d'un tournoi dirigé : descendre, monter."
   "ENTRÉE", "Dans la file des propositions : confirmer la proposition choisie."
   "TAB (page du tournoi)", "Premier arrêt dans l'onglet Direction : « Aller à la file » ; ENTRÉE pose le focus sur la file des propositions."
   "GAUCHE / DROITE", "Dans la fiche de résultat, hors d'un champ : choisir le joueur de gauche ou celui de droite ; ENTRÉE enregistre sa victoire."
   "CTRL-Z", "Reprendre la dernière décision d'un tournoi dirigé (hors d'un champ de saisie, où il annule la frappe)."
   "TAB (focus perdu)", "Sous la page d'un tournoi dirigé, quand le focus est tombé sur la page : le ramener sur le premier élément de la page, sans ouvrir la recherche."
   "PAGE PRÉC. / PAGE SUIV., DÉBUT / FIN", "Sous la page d'un tournoi dirigé : faire défiler la page, sans parcourir le plateau qu'elle cache."
   "ÉCHAP", "Fermer la fiche de résultat ou la reprise en cours."
   "?", "Afficher/cacher l'aide."

.. _raccourcis_duel:

Duel au plateau
---------------

.. csv-table::
   :header: "Geste", "Action"
   :widths: 7, 20
   :align: center

   "Clic sur le plateau ou sur les dés (avant le lancer)", "Lancer les dés. Le videau garde son sens : il propose de doubler."
   "Clic sur le videau (avant le lancer)", "Proposer de doubler ; « Doubler » ou « Annuler » confirme sur le plateau."
   "Clic sur un pion", "Le jouer aussitôt avec le dé de gauche encore libre, ou avec l'autre si celui-ci ne peut pas le jouer. Un dé joué est grisé ; le coup complet grise tous les dés."
   "Glisser un pion", "Le jouer vers le point où il est lâché, si un coup légal le permet."
   "Clic sur les dés (coup en cours)", "Coup complet : le valider. Sinon : intervertir les dés restant à jouer."
   "Clic droit sur le plateau (coup en cours)", "Reprendre tous les pions joués ; sans pion joué, ouvrir le menu du Duel."
   "Clic droit hors du plateau, ou hors de son coup", "Ouvrir le menu du Duel : Pile, abandon, suspension, arrêt."

.. _raccourcis_vues:

Onglets de vues
---------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "CTRL-T", "Créer une nouvelle vue (copie de la vue courante)."
   "CTRL-W", "Fermer la vue courante."
   "CTRL-Page préc., MAJ-J", "Vue précédente."
   "CTRL-Page suiv., MAJ-K", "Vue suivante."
   "CTRL-1 … CTRL-9", "Aller directement à la n-ième vue."
   "Double-clic sur l'onglet", "Renommer la vue."

.. _raccourcis_command:

Ligne de commande
-----------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "HAUT", "Parcourir l'historique des commandes vers le haut."
   "BAS", "Parcourir l'historique des commandes vers le bas."
   "ÉCHAP", "Pendant une recherche : l'interrompre."

.. _raccourcis_search_history:

Historique de recherche
-----------------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic", "Sélectionner/désélectionner une recherche (afficher la position)."
   "Double-clic", "Exécuter la recherche."

.. _raccourcis_filter_library:

Bibliothèque de filtres
-----------------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic", "Sélectionner/désélectionner un filtre (afficher la position)."
   "Double-clic", "Exécuter la recherche du filtre."
   "Clic (sur l'étoile)", "Épingler ou désépingler le filtre."
   "Clic (sur une pastille épinglée)", "Exécuter la recherche du filtre épinglé."

.. _raccourcis_analysis:

Panneau d'analyse
-----------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic", "Sélectionner/désélectionner un coup (afficher/cacher les flèches)."
   "Ctrl+Clic", "Ajouter ou retirer un coup de la sélection à rouler."
   "Maj+Clic", "Étendre la sélection jusqu'au coup cliqué."
   "Clic droit", "Menu du coup : rouler les coups sélectionnés avec le réglage choisi, ou annuler le rollout en cours ; copier la position et l'analyse, ou la position et les coups sélectionnés."
   "HAUT, k", "Sélectionner le coup précédent (lorsqu'un coup est sélectionné)."
   "BAS, j", "Sélectionner le coup suivant (lorsqu'un coup est sélectionné)."
   "d", "Basculer entre l'analyse des coups et du cube (navigation match uniquement)."
   "r", "Lancer le rollout des coups sélectionnés (sans sélection, de la position) avec le réglage choisi ; une seconde pression l'arrête."
   "Esc", "Arrêter le rollout en cours, sinon désélectionner le coup. Si aucun coup sélectionné, fermer le panneau, sauf devant les résultats d'une recherche ``ss`` : y revenir à la collection ou au match."

.. _raccourcis_eval_panel:

Panneau Eval
------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic", "Sélectionner/désélectionner un coup (afficher/cacher les flèches)."
   "Ctrl+Clic", "Ajouter ou retirer un coup de la sélection à rouler."
   "Maj+Clic", "Étendre la sélection jusqu'au coup cliqué."
   "Clic droit", "Menu du coup : rouler les coups sélectionnés avec le réglage choisi, ou annuler le rollout en cours ; copier la position et l'analyse, ou la position et les coups sélectionnés."
   "HAUT, k", "Sélectionner le coup précédent (lorsqu'un coup est sélectionné)."
   "BAS, j", "Sélectionner le coup suivant (lorsqu'un coup est sélectionné)."
   "r", "Lancer le rollout des coups sélectionnés (sans sélection, de la position) avec le réglage choisi ; une seconde pression l'arrête."
   "Esc", "Arrêter le rollout en cours, sinon désélectionner le coup."

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
   "v", "Voir dans la vidéo la décision étudiée, une seconde avant le jet (match portant une source vidéo et un repère à cette décision)."
   "[ / ] (vidéo ouverte)", "Ralentir ou accélérer la lecture de la vidéo, par pas de 0,25 entre 0,25× et 4×."
   "/", "Aller au champ de filtre (joueur, événement, tournoi, date). *Esc* efface le filtre."
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

   "Double-clic, ENTRÉE (ligne au focus)", "Sélectionner un tournoi (afficher son détail). TAB atteint les lignes ; un clic simple ne fait que surligner."
   "HAUT, k", "Sélectionner le tournoi précédent, quand le panneau a le focus ou qu'aucune direction n'est affichée."
   "BAS, j", "Sélectionner le tournoi suivant, quand le panneau a le focus ou qu'aucune direction n'est affichée."
   "Double-clic (sur un match du tournoi)", "Naviguer dans le match."
   "Esc", "Annuler l'édition en cours, sinon effacer la recherche d'ajout de match, sinon désélectionner le tournoi, sinon fermer le panneau (par paliers)."

.. _raccourcis_direction:

Page Direction
--------------

Sous la page Direction, *J*, *K*, *HAUT*, *BAS* et *ENTRÉE* vont à la file des
propositions, sauf quand le focus est sur une case de la grille des tables, où
*HAUT* et *BAS* changent de case. Les menus contextuels sont décrits dans le
manuel (:ref:`menus contextuels <direction_menus>`).

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "Clic droit, MENU, MAJ-F10", "Ouvrir le menu contextuel de l'objet focalisé : case de table, joueur, place de l'arbre, emplacement, proposition, ligne d'historique. HAUT/BAS parcourent le menu, ENTRÉE choisit, ÉCHAP le ferme."
   "GAUCHE, DROITE, HAUT, BAS, DÉBUT, FIN", "Passer d'une case de la grille des tables à l'autre (cases libres comprises) ; la grille ne prend qu'un arrêt de TAB."
   "1 à 9, puis 0 à 9", "Ouvrir la fiche de la table de ce numéro ; deux chiffres, dans les 0,4 s, pour une table au-delà de 9."
   "M, X", "Sur une case occupée, ouvrir la fiche (ou, si elle est ouverte, le champ de table) ; viser une table occupée échange les deux matchs."
   "/", "Ouvrir la recherche rapide : joueurs, tables, matchs en cours et épreuves de l'événement, retrouvés par un nom ou un numéro de table (:ref:`détail <direction_recherche>`). Sans effet dans un champ de saisie."
   "F11", "Mettre la page Direction en plein écran (barre d'outils, onglets, panneau et barre d'état masqués), ou en sortir. ÉCHAP en sort aussi, après avoir fermé le menu ou la fiche ouverts (:ref:`détail <direction_plein_ecran>`)."
   "Glisser une case occupée sur une autre", "À la souris : sur une case libre, déplacer le match ; sur une case occupée, échanger les deux matchs après confirmation. ÉCHAP annule le glisser."
   "Toutes les tables", "Sur la grille *Toutes les tables* d'un événement, les mêmes touches, menus et glisser-déposer agissent sur la table de n'importe quelle épreuve ; un échange avec une autre épreuve nomme les deux dans la confirmation."

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

.. _raccourcis_transcription_panel:

Panneau Transcription
---------------------

Le panneau prend ces touches lorsqu'il a le focus. Devant la liste des
brouillons il en prend déjà quelques-unes, pour que reprendre le travail de la
veille ne demande pas la souris.

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "BAS, j / HAUT, k (liste des brouillons)", "Parcourir les brouillons. Le premier est surligné à l'ouverture : c'est le plus récemment modifié."
   "ENTREE (liste des brouillons)", "Ouvrir le brouillon surligné."
   "n (liste des brouillons)", "Ouvrir le formulaire de création."
   "Clic", "Ouvrir un brouillon de la liste."
   "1 … 6", "Saisir un dé. Sur le premier coup d'une partie : dé du joueur 1, puis dé du joueur 2 (le plus fort commence et joue les deux dés)."
   "1 … 6 (jet saisi)", "Valider le coup sélectionné et ouvrir le jet suivant. Sur une action relue, où le curseur est posé sur une action déjà écrite, le chiffre recommence le jet de cette action au lieu de valider."
   "1 … 6 (curseur sur le premier coup d'une partie)", "Taper un autre jet d'ouverture : dé du joueur 1 puis dé du joueur 2. Le plus fort l'emporte — gros dé d'abord, le coup revient au joueur 1 en bas ; petit dé d'abord, au joueur 2 en haut ; le premier candidat est présélectionné. Le même jet retapé ne change rien ; *s* donne le coup à l'autre joueur."
   "BAS, j", "Sélectionner le candidat suivant (les flèches du coup apparaissent sur le plateau)."
   "HAUT, k", "Sélectionner le candidat précédent."
   "Molette", "Sélectionner le candidat suivant ou précédent, au-dessus de la liste comme au-dessus du damier : le regard reste sur le plateau et les flèches défilent."
   "Clic (sur une ligne)", "Sélectionner ce candidat."
   "Double-clic (sur une ligne)", "Valider ce candidat."
   "Clic (sur le triangle des jets)", "Saisir le jet d'un seul geste : la case porte les deux dés, doubles sur la diagonale. Sur le premier coup d'une partie, le triangle laisse la place à une rangée de six dés, un clic donnant le dé d'un camp."
   "Clic, glisser (aucun dé saisi)", "Jouer le coup directement sur le damier : le pion va du point cliqué à sa destination, contraint aux coups légaux, et les deux dés se déduisent des pas joués."
   "Clic, glisser (jet saisi)", "Jouer le coup sur le damier, contraint aux coups légaux de ce jet : chaque pas joué ne garde dans la liste que les candidats qui le contiennent, et un coup légal achevé est enregistré aussitôt. Sur une action relue, il la remplace."
   "Glisser hors des règles (jet saisi)", "Poser le pion là où il est lâché, même depuis un point d'où aucun coup légal ne part, pour transcrire un coup illégal. La suite du coup se joue librement, au clic comme au glissé, et la liste des candidats cède la place à une ligne qui le rappelle."
   "ENTREE (coup hors des règles)", "Enregistrer le coup avec les dés saisis et le plateau obtenu, marqué coup illégal si aucun coup légal n'atteint ce plateau. Un coup hors des règles n'est jamais enregistré de lui-même."
   "RETOUR ARRIERE (coup en cours au plateau)", "Défaire le dernier pas joué au plateau. Les pas restants sont rejoués contraints tant qu'un coup légal les contient : défaire le seul pas hors des règles rend la liste."
   "ENTREE", "Valider le coup sélectionné (dernier coup d'une partie)."
   "RETOUR ARRIERE", "Effacer les deux dés saisis. C'est par là que passe la reprise d'un jet mal lu, puisqu'un chiffre valide."
   "Clic (sur les cases du jet)", "Effacer les deux dés saisis, comme RETOUR ARRIERE."
   "Esc", "Abandonner la saisie en cours."
   "d", "Doubler ou redoubler : le coup sélectionné est validé au passage, en une seule touche."
   "t", "Prendre le double proposé : le videau passe au preneur à la valeur doublée et le doubleur rejoue. Le curseur posé sur une cellule, écrit la prise à la place de l'action visée ; une passe devenue prise rouvre sa partie, et la suite s'y tape à la place."
   "p", "Passer le double proposé : la partie est gagnée à la valeur d'avant le double. Le curseur posé sur une cellule, écrit la passe à la place de l'action visée."
   "r puis 1, 2 ou 3", "Abandonner la partie pour le camp au trait : simple, gammon ou backgammon. Esc entre les deux touches annule sans rien enregistrer."
   "Clic (sur le videau)", "Proposer un double pour le camp au trait, comme la touche d. Devant une offre, le videau ne répond pas : la prise et la passe sont dans la rangée de boutons."
   "GAUCHE, h", "Reculer le curseur d'une action dans le transcript."
   "DROITE, l", "Avancer le curseur d'une action."
   "Clic (sur une cellule)", "Placer le curseur sur cette action, ou sur le tour manquant d'un double trait."
   "Double-clic (sur une cellule)", "Taper le coup de cette action au clavier, dans la cellule : 13/7 8/7*, bar/22, 6/off. Seul le coup se tape, les dés sont ceux de la cellule ; ENTREE l'enregistre, même illégal, et Esc referme la cellule sans rien écrire. Vaut pour un coup, une danse, un coup non consigné, et pour la cellule en pointillés de la saisie en cours dès que ses deux dés sont saisis."
   "Double-clic (sur le score d'une partie)", "Taper le score auquel cette partie a été jouée, dans l'en-tête : 3-2, 3–2 ou 3 2. ENTREE l'enregistre, un champ vidé revient au score que donnent les parties précédentes, et Esc referme le champ sans rien écrire. Un score qui diffère de celui-ci est marqué comme une incohérence. Pas de score en argent."
   "Clic droit (sur une cellule)", "Ouvrir les corrections de cette action : insérer avant, insérer après, supprimer, changer de camp. Le curseur est amené sur la cellule au passage."
   "CTRL-ENTREE", "Terminer le brouillon : écrire le match, ou remplacer celui dont il a été ouvert, et libérer le brouillon."
   "i", "Insérer une action devant celle du curseur (camp proposé pour que la suite reste cohérente)."
   "a", "Insérer une action derrière celle du curseur."
   "x, Del", "Supprimer la décision en cours d'édition — l'action du curseur, ou la saisie pas encore écrite — et reculer sur la précédente, prête à être corrigée ; les suivantes gardent leur camp. En bout de document, recule sur la dernière action."
   "s", "Donner l'action du curseur à l'autre camp."
   "CTRL-Z", "Annuler le dernier geste sur le brouillon."
   "CTRL-MAJ-Z", "Rétablir le geste annulé."
   "ESPACE (vidéo attachée)", "Lancer ou mettre en pause la vidéo. Sans vidéo, ESPACE ouvre la ligne de commande, comme ailleurs."
   "MAJ-GAUCHE / MAJ-DROITE (vidéo attachée)", "Reculer ou avancer la vidéo de 5 secondes."
   "CTRL-MAJ-GAUCHE / CTRL-MAJ-DROITE (vidéo attachée)", "Reculer ou avancer la vidéo d'une seconde."
   "v (vidéo attachée)", "Poser l'instant courant de la vidéo comme instant de l'action du curseur."
   "MAJ-V (vidéo attachée)", "Poser l'instant courant de la vidéo comme instant du jet de l'action du curseur."
   "[ / ] (vidéo attachée)", "Ralentir ou accélérer la lecture, par pas de 0,25 entre 0,25× et 4× (jusqu'à 2× pour YouTube)."
   "F11 (vidéo attachée)", "Passer en mode théâtre — la vidéo en plein écran, un petit plateau flottant par-dessus, le clavier de saisie toujours actif — ou en sortir. ÉCHAP en sort aussi ; dans le théâtre, RETOUR ARRIERE efface les dés saisis (:ref:`détail <transcription_theatre>`)."

Un jet qui n'autorise aucun coup enregistre la danse de lui-même, sans touche
supplémentaire.

Avec une vidéo attachée, seule une validation explicite — *ENTREE*, le
double-clic sur un candidat, le coup achevé au plateau — pose l'instant de
l'action. Le chiffre du jet suivant, ou un geste de videau, valide sans
instant ; *v* le pose après coup. Les flèches se lisent à leur place sur le
clavier, quelle que soit sa disposition ; *[* et *]* se lisent au caractère
tapé, *ALTGR* compris sur un clavier qui les place ainsi. Sans vidéo, ces touches gardent leur
sens habituel.

Le chiffre a un seul sens : **il commence un jet là où le curseur est**. En bout
de document il n'y a rien sous le curseur, donc il valide le coup sélectionné
avant d'ouvrir le jet suivant — le meilleur coup joué coûte ainsi les deux dés
et rien de plus, sa validation étant portée par la première touche du tour
d'après. Sur une action déjà écrite, où l'on est revenu pour la corriger, il y a
quelque chose sous le curseur : le chiffre recommence le jet de cette action,
sur place. La différence se voit à l'écran, la cellule visée étant encadrée dans
le transcript. La **dernière** action du document fait exception : une fois son
jet retapé, le chiffre suivant la valide et ouvre la décision d'après, comme en
bout de document, et ENTREE y mène aussi.

Ce qui est en train d'être tapé se dessine dans le transcript, en pointillés, à
la place où il sera écrit : une correction recouvre la cellule qu'elle remplace,
une insertion ouvre une cellule entre ses deux voisines, et le camp se lit à la
colonne. Rien n'est enregistré avant la validation.

Une insertion au milieu du document continue d'insérer : la validation ouvre une
cellule vide à la suite, et l'action suivante s'insère à son tour au lieu
d'écraser celle d'après. La fin de la partie ou un déplacement du curseur y met
fin.

Un geste qui n'a rien à faire le dit, une fois, dans la barre d'état :
« rien à annuler » sur une pile vide, « aucune action sous le curseur » en bout
de document. La phrase s'efface d'elle-même et rend la place à l'action
attendue.

Reculer le curseur sur une action puis ressaisir la corrige **en place** : la
validation remplace l'action et le curseur revient là où il était — ou, sur la
dernière action, passe au bout du document, où la transcription continue. Si les dés
sont corrigés et que le coup enregistré reste un coup légal du nouveau jet, il
est conservé ; sinon le premier candidat du nouveau jet est proposé et le coup
est signalé « à revoir » jusqu'à la validation. Avancer ou reculer le curseur
après avoir changé quelque chose enregistre la correction au passage.

Rien n'est refusé ni supprimé : insérer une action du même camp que sa voisine
crée un double trait, supprimer une action peut en créer un autre, changer un
camp peut rendre illégaux les coups qui suivent. Ces incohérences sont marquées
dans le transcript, jamais corrigées d'office, et le curseur se place sur la
première d'entre elles après chaque geste qui écrit — sauf là où l'on continue : après une
suppression, une insertion ou une partie rouverte, il reste où l'on tape. Se
déplacer ou taper un dé ne le déplace jamais. Le tour qu'un double trait a perdu
est une case du transcript : h et l s'y arrêtent, et un jet tapé là s'insère
pour le camp à qui il manquait. La pile d'annulation vit en mémoire :
elle est perdue à la fermeture du brouillon.

Le panneau lui-même — la liste des brouillons, la création, la saisie, le
transcript et la barre du brouillon — est décrit dans
:ref:`panneau_transcription`.

.. _raccourcis_contact_sheet:

Planche-contact
---------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "GAUCHE, DROITE, HAUT, BAS", "Se déplacer dans la grille."
   "j, k", "Vignette suivante, vignette précédente."
   "Début, Fin", "Première, dernière vignette de la page."
   "Page préc., Page suiv.", "Page précédente, page suivante."
   "ENTREE, Clic", "Ouvrir la position sur le plateau et refermer la planche."
   "Esc", "Refermer la planche."

.. _raccourcis_help_panel:

Panneau d'aide
--------------

.. csv-table::
   :header: "Raccourci", "Action"
   :widths: 7, 20
   :align: center

   "GAUCHE, h", "Onglet précédent."
   "DROITE, l", "Onglet suivant."
   "HAUT, k", "Défiler vers le haut."
   "BAS, j", "Défiler vers le bas."
   "ESPACE", "Page suivante."
   "Page préc.", "Haut du contenu."
   "Page suiv.", "Bas du contenu."
   "/", "Chercher dans l'aide : les occurrences sont comptées et surlignées à la frappe ; Entrée passe à l'occurrence suivante, MAJ-Entrée à la précédente."
   "?, CTRL-F, Esc", "Fermer l'aide."
