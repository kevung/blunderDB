.. _cmd_mode:

Liste des commandes
===================

La ligne de commande, située dans la barre d'état, s'ouvre en appuyant sur la
touche *ESPACE*. Lors de la saisie d'une commande, une liste de suggestions
apparaît automatiquement : la touche *TAB* (ou *MAJ-TAB*) parcourt les
propositions et complète la commande, tandis que *ÉCHAP* referme la liste (un
second *ÉCHAP* ferme la ligne de commande). Les touches *HAUT* et *BAS* restent
réservées à l'historique des commandes.

.. _cmd_global:

Opérations globales
-------------------

.. csv-table::
   :header: "Commande", "Action"
   :widths: 10, 40
   :align: center

   "new, ne, n", "Crée une nouvelle base de données."
   "open, op, o", "Ouvre une base de données existante."
   "import_db, idb", "Importe et fusionne une autre base de données."
   "export_db, edb", "Exporte la sélection courante vers une nouvelle base de données."
   "quit, q", "Ferme blunderDB."
   "help, he, h", "Ouvre l'aide de blunderDB."
   "tutorial, tour", "Ouvre le catalogue des visites guidées de l'interface."
   "demo", "Charge une base d'exemple (matchs, tournoi, collections, commentaires, paquet Anki, analyses) pour découvrir l'outil."
   "meta", "Affiche les métadonnées de la base de données."
   "epc", "Ouvre le panneau Eval (Effective Pip Count, probabilité de gain et verdict de videau en bearoff). ``epc`` est l'ancien nom de ce panneau, conservé."
   "met", "Ouvre la table d'équité de match Kazaross-XG2."
   "cm", "Ouvre la matrice du videau : le verdict de la position courante à tous les scores d'un match de 5, 7 ou 9 points."
   "tp2", "Ouvre la table des takepoints avec videau à 2."
   "tp2_live", "Ouvre la table des takepoints avec videau à 2 pour les courses longues."
   "tp2_last", "Ouvre la table des takepoints avec videau à 2 mort."
   "tp4", "Ouvre la table des takepoints avec videau à 4."
   "tp4_live", "Ouvre la table des takepoints avec videau à 4 pour les courses longues."
   "tp4_last", "Ouvre la table des takepoints avec videau à 4 mort."
   "gv1", "Ouvre la table des valeurs de gammon avec videau à 1."
   "gv2", "Ouvre la table des valeurs de gammon avec videau à 2."
   "gv4", "Ouvre la table des valeurs de gammon avec videau à 4."

.. _cmd_positions:

Positions et navigation
-----------------------

.. csv-table::
   :header: "Commande", "Action"
   :widths: 10, 20
   :align: center

   "import, i", "Importe une ou plusieurs positions/matchs par fichier (xg, xgp, sgf, mat, txt, bgf)."
   "delete, del, d", "Supprime la position courante (confirmation demandée) ; la suppression passe par la corbeille et reste annulable trente jours."
   "trash", "Ouvre la corbeille : ce qui a été supprimé, avec de quoi le restaurer."
   "[number]", "Aller à la position d'indice indiqué."
   "list, l", "Afficher l'analyse de la position courante."
   "comment, co", "Afficher/écrire des commentaires."
   "history, hi", "Ouvrir le panneau de recherche (l'historique de recherche se trouve dans son onglet *Historique*)."
   "stats, st", "Afficher/masquer le panneau de statistiques."
   "match, ma", "Afficher/cacher le panneau des matchs."
   "collection, coll", "Afficher/cacher le panneau des collections."
   "#tag1 tag2 ...", "Etiqueter la position courante."
   "e", "Charger toutes les positions de la base de données."
   "blunders, bl [n]", "Charger les pires erreurs (équité/MWC) dans la vue d'analyse, selon le filtre courant des statistiques. Un nombre optionnel choisit combien en charger (``bl 50``) ; par défaut 10."
   "m", "Naviguer dans le dernier match visité."

.. _cmd_edition:

Édition et recherche
--------------------

.. csv-table::
   :header: "Commande", "Action"
   :widths: 10, 20
   :align: center

   "write, wr, w", "Enregistre la position courante."
   "write!, wr!, w!", "Mettre à jour la position courante."
   "s", "Chercher des positions avec des filtres."
   "ss", "Chercher parmi les positions actuellement filtrées."
   

.. _cmd_filter:

Filtres de recherche
--------------------

Cette table est la référence de la grammaire de recherche : la ligne de
commande, la bibliothèque de filtres et le drapeau ``--query`` de
``blunderdb search`` lisent tous les mêmes jetons. La colonne *Équivalent
CLI* donne, quand il existe, le drapeau de ``search`` qui fait la même chose
(voir :ref:`cli`) ; un tiret signale un filtre que seule la grammaire
exprime.

Cinq jetons ne portent pas leur valeur : ils la lisent sur le plateau de
recherche. ``cube`` et ``score`` reprennent le videau et le score qui y sont
posés, ``d`` le type de décision, ``D`` et ``D1`` les dés, ``x`` la structure
dessinée dans l'onglet *Sauf*. Un lancer ne s'écrit donc jamais dans le
jeton : ``D65`` n'existe pas, seule la forme d'exclusion porte ses chiffres
(``xD65``). En ligne de commande, où il n'y a pas de plateau, ces jetons se
comparent à un plateau vide ; ce sont les drapeaux de la troisième colonne
qu'il faut y employer.

Les erreurs et les équités se comptent en **millièmes d'équité** — les
*millipoints* de la table ci-dessous : ``E>100`` retient les coups qui ont
coûté au moins un dixième de point, un point valant 1000 millièmes.

Deux recherches complètes :

* ``s p>30 w40,60 xco`` — plus de 30 pips de retard, entre 40 % et 60 % de
  chances de gain, aucun commentaire.
* ``s ph:race E>50 co:xg`` — en course, un coup ayant coûté au moins 50
  millièmes, et un commentaire venu d'eXtreme Gammon.

.. _cmd_filter_pos:

.. csv-table::
   :header: "Requête", "Action", "Équivalent CLI"
   :widths: 8, 26, 10
   :align: center

   "cube, cub, cu, c", "La position vérifie la configuration du cube.", "``--cube``"
   "score, sco, sc, s", "La position vérifie le score.", "``--score1`` ``--score2``"
   "d", "La position vérifie le type de décision (pion ou cube).", "``--decision``"
   "D", "La position vérifie le lancer de dés (les deux dés, peu importe l'ordre).", "``--dice 6,5``"
   "D1", "La position vérifie le lancer de dés sur le premier dé uniquement (la valeur du premier dé apparaît sur l'un des deux dés de la position).", "``--dice 6``"
   "xD65", "La position n'a **pas** été jouée avec le lancer 6-5 (peu importe l'ordre). La valeur est indiquée dans le jeton ; répétable pour exclure plusieurs lancers (``xD65 xD54``).", "—"
   "nc", "La position est sans contact.", "—"
   "ph:race", "La position est dans une phase de jeu donnée : ``opening`` (ouverture), ``middlegame`` (milieu de partie), ``race`` (course) ou ``bearoff`` (sortie des pions). Répétable (``ph:race ph:bearoff``). L'étiquette est calculée à partir du plateau, jamais modifiable ; la commande ``blunderdb repair`` la recalcule.", "``--phase``"
   "M", "La position ou celle miroir vérifie les filtres.", "—"
   "i", "La position a été importée seule, et non apportée par l'import d'un match.", "``--individual``"
   "fl", "La position a été marquée (*flag*) dans le logiciel d'origine, lors de l'import d'un match eXtreme Gammon.", "``--flagged``"
   "x", "La position ne contient aucun pion de la structure d'exclusion (onglet *Sauf* du panneau de recherche).", "—"
   "p>x", "Le joueur a au moins x pips de retard à la course.", "``--pip-min``"
   "p<x", "Le joueur a au plus x pips de retard à la course.", "``--pip-max``"
   "px,y", "Le joueur a entre x et y pips de retard à la course.", "``--pip-min`` ``--pip-max``"
   "P>x", "Le joueur a une course au moins de x pips.", "—"
   "P<x", "Le joueur a une course au plus de x pips.", "—"
   "Px,y", "Le joueur a une course entre x et y pips.", "—"
   "e>x", "L'équité (en millipoints) de la position est supérieure à x.", "—"
   "e<x", "L'équité (en millipoints) de la position est inférieure à x.", "—"
   "ex,y", "L'équité (en millipoints) de la position est comprise entre x et y.", "—"
   "E>x", "L'erreur du coup joué par le joueur 1 (en millipoints) est supérieure à x.", "``--move-error-min``"
   "E<x", "L'erreur du coup joué par le joueur 1 (en millipoints) est inférieure à x.", "``--move-error-max``"
   "Ex,y", "L'erreur du coup joué par le joueur 1 (en millipoints) est comprise entre x et y.", "``--move-error-min`` ``--move-error-max``"
   "w>x", "Le joueur a des chances de gain supérieures à x %.", "``--winrate-min``"
   "w<x", "Le joueur a des chances de gain inférieures à x %.", "``--winrate-max``"
   "wx,y", "Le joueur a des chances de gain comprises à x % et y %.", "``--winrate-min`` ``--winrate-max``"
   "g>x", "Le joueur a des chances de gammon supérieures à x %.", "—"
   "g<x", "Le joueur a des chances de gammon inférieures à x %.", "—"
   "gx,y", "Le joueur a des chances de gammon comprises à x % et y %.", "—"
   "b>x", "Le joueur a des chances de backgammon supérieures à x %.", "—"
   "b<x", "Le joueur a des chances de backgammon inférieures à x %.", "—"
   "bx,y", "Le joueur a des chances de backgammon comprises à x % et y %.", "—"
   "W>x", "L'adversaire a des chances de gain supérieures à x %.", "—"
   "W<x", "L'adversaire a des chances de gain inférieures à x %.", "—"
   "Wx,y", "L'adversaire a des chances de gain comprises à x % et y %.", "—"
   "G>x", "L'adversaire a des chances de gammon supérieures à x %.", "—"
   "G<x", "L'adversaire a des chances de gammon inférieures à x %.", "—"
   "Gx,y", "L'adversaire a des chances de gammon comprises à x % et y %.", "—"
   "B>x", "L'adversaire a des chances de backgammon supérieures à x %.", "—"
   "B<x", "L'adversaire a des chances de backgammon inférieures à x %.", "—"
   "Bx,y", "L'adversaire a des chances de backgammon comprises à x % et y %.", "—"
   "o>x", "Le joueur a au moins x pions sortis.", "``--off1-min``"
   "o<x", "Le joueur a au plus x pions sortis.", "—"
   "ox,y", "Le joueur a entre x et y pions sortis.", "—"
   "O>x", "L'adversaire a au moins x pions sortis.", "``--off2-min``"
   "O<x", "L'adversaire a au plus x pions sortis.", "—"
   "Ox,y", "L'adversaire a entre x et y pions sortis.", "—"
   "k>x", "Le joueur a au moins x pions arriérés.", "—"
   "k<x", "Le joueur a au plus x pions arriérés.", "—"
   "kx,y", "Le joueur a entre x et y pions arriérés.", "—"
   "K>x", "L'adversaire a au moins x pions arriérés.", "—"
   "K<x", "L'adversaire a au plus x pions arriérés.", "—"
   "Kx,y", "L'adversaire a entre x et y pions arriérés.", "—"
   "z>x", "Le joueur a au moins x pions dans la zone.", "—"
   "z<x", "Le joueur a au plus x pions dans la zone.", "—"
   "zx,y", "Le joueur a entre x et y pions dans la zone.", "—"
   "Z>x", "L'adversaire a au moins x pions dans la zone.", "—"
   "Z<x", "L'adversaire a au plus x pions dans la zone.", "—"
   "Zx,y", "L'adversaire a entre x et y pions dans la zone.", "—"
   "bo>x", "Le joueur a au moins x blots dans l'outfield.", "—"
   "bo<x", "Le joueur a au plus x blots dans l'outfield.", "—"
   "box,y", "Le joueur a entre x et y blots dans l'outfield.", "—"
   "BO>x", "L'adversaire a au moins x blots dans l'outfield.", "—"
   "BO<x", "L'adversaire a au plus x blots dans l'outfield.", "—"
   "BOx,y", "L'adversaire a entre x et y blots dans l'outfield.", "—"
   "bj>x", "Le joueur a au moins x blots dans le jan.", "—"
   "bj<x", "Le joueur a au plus x blots dans le jan.", "—"
   "bjx,y", "Le joueur a entre x et y blots dans le jan.", "—"
   "BJ>x", "L'adversaire a au moins x blots dans le jan.", "—"
   "BJ<x", "L'adversaire a au plus x blots dans le jan.", "—"
   "BJx,y", "L'adversaire a entre x et y blots dans le jan.", "—"
   "``t'mot1;mot2;...'``", "Les commentaires de la position contiennent au moins un des mots.", "—"
   "co", "La position porte un commentaire, quel qu'en soit le contenu.", "``--has-comment``"
   "xco", "La position ne porte aucun commentaire.", "``--no-comment``"
   "co:user", "La position porte un commentaire d'une provenance donnée : ``user`` (écrit par vous), ``xg``, ``gnubg``, ``bgf`` (apporté par l'import d'un match) ou ``unknown``. Répétable (``co:xg co:gnubg``).", "``--comment-origin``"
   "``m'motif1,motif2,...'``", "Les meilleurs coups de pions contenant au moins un des motifs.", "—"
   "``m'ND,DT,DP,...'``", "Les meilleures décisions de videau de No Double/Take, Double Take, Double Pass.", "—"
   "T>x", "Date d'ajout de la position après x (AAAA/MM/JJ).", "—"
   "T<x", "Date d'ajout de la position avant x (AAAA/MM/JJ).", "—"
   "Tx,y", "Date d'ajout de la position entre x et y (AAAA/MM/JJ).", "—"
   "max", "Rechercher dans le match d'identifiant x (ex: ma3).", "``--match-ids``"
   "max,y", "Rechercher dans les matchs d'identifiants x à y (ex: ma2,5).", "``--match-ids``"
   "tnx", "Rechercher dans le tournoi d'identifiant x (ex: tn1).", "``--tournament-ids``"
   "tnx,y", "Rechercher dans les tournois d'identifiants x à y (ex: tn1,3).", "``--tournament-ids``"
   "idx", "Rechercher la position d'identifiant x (ex: id12).", "``--position-ids``"
   "idx,y", "Rechercher les positions d'identifiants x à y (ex: id5,10).", "``--position-ids``"
   "``pl'nom'``", "Rechercher les positions issues d'un match impliquant le joueur indiqué, sur l'un ou l'autre camp (ex: ``pl'Alice'``). La casse est ignorée.", "—"

.. _cmd_misc:

Commandes diverses
------------------

.. csv-table::
   :header: "Commande", "Action"
   :widths: 10, 40
   :align: center

   "clear, cl", "Efface l'historique des commandes."
