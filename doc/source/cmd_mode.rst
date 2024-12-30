.. _cmd_mode:

Liste des commandes
===================

.. _cmd_global:

Opérations globales
-------------------

.. csv-table::
   :header: "Commande", "Action"
   :widths: 10, 40
   :align: center

   "new, ne, n", "Crée une nouvelle base de données."
   "open, op, o", "Ouvre une base de données existante."
   "quit, q", "Ferme blunderDB."
   "help, he, h", "Ouvre l'aide de blunderDB."


.. _cmd_normal:

Mode NORMAL
-----------

.. csv-table::
   :header: "Commande", "Action"
   :widths: 10, 20
   :align: center

   "import, i", "Importe une position par fichier texte (txt)."
   "delete, del, d", "Supprime la position courante."
   "[number]", "Aller à la position d'indice indiqué."
   "list, l", "Afficher l'analyse de la position courante."
   "comment, co", "Afficher/écrire des commentaires."
   "#tag1 tag2 ...", "Etiqueter la position courante."


.. _cmd_edit:

Mode EDIT
---------

.. csv-table::
   :header: "Commande", "Action"
   :widths: 10, 20
   :align: center

   "write, wr, w", "Enregistre la position courante."
   "write!, wr!, w!", "Mettre à jour la position courante."
   "s", "Chercher des positions avec des filtres."
   "e", "Charger toutes les positions de la base de données."


.. _cmd_filter:

Filtres de recherche
--------------------

Les filtres ci-dessous doivent être juxtaposés lors d'une recherche,
c'est-à-dire après le début de commande ``s``.

.. _cmd_filter_pos:

.. warning:: Dans la recherche de positions, par défaut, blunderDB prend en
   compte la structure de pions courante, ignore la position du videau, du
   score et des dés. Pour prendre en compte la position du videau, du score,
   des dés, il faut le mentionner explicitement dans la recherche.

.. note::
   blunderDB considère qu'un pion arriéré (backchecker) est un pion
   situé entre le point 24 et le point 14.

.. note::
   blunderDB considère que le nombre de pions dans la zone est le nombre
   de pions situés entre le point 12 et le point 1.

.. tip::
   Les paramètres pour filtrer des positions peuvent être arbitrairement
   combinés.

.. csv-table::
   :header: "Requête", "Action"
   :widths: 10, 20
   :align: center

   "cube, cub, cu, c", "La position vérifie la configuration du cube."
   "score, sco, sc, s", "La position vérifie le score."
   "dice, dic, di, d", "La position vérifie les dés ou la décision de cube."
   "p>x", "Le joueur a au moins x pips de retard à la course."
   "p<x", "Le joueur a au plus x pips de retard à la course."
   "px,y", "Le joueur a entre x et y pips de retard à la course."
   "P>x", "Le joueur a une course au moins de x pips."
   "P<x", "Le joueur a une course au plus de x pips."
   "Px,y", "Le joueur a une course entre x et y pips."
   "e>x", "L'équité (en millipoints) de la position est supérieure à x."
   "e<x", "L'équité (en millipoints) de la position est inférieure à x."
   "ex,y", "L'équité (en millipoints) de la position est comprise entre x et y."
   "w>x", "Le joueur a des chances de gain supérieures à x %."
   "w<x", "Le joueur a des chances de gain inférieures à x %."
   "wx,y", "Le joueur a des chances de gain comprises à x % et y %."
   "g>x", "Le joueur a des chances de gammon supérieures à x %."
   "g<x", "Le joueur a des chances de gammon inférieures à x %."
   "gx,y", "Le joueur a des chances de gammon comprises à x % et y %."
   "b>x", "Le joueur a des chances de backgammon supérieures à x %."
   "b<x", "Le joueur a des chances de backgammon inférieures à x %."
   "bx,y", "Le joueur a des chances de backgammon comprises à x % et y %."
   "W>x", "L'adversaire a des chances de gain supérieures à x %."
   "W<x", "L'adversaire a des chances de gain inférieures à x %."
   "Wx,y", "L'adversaire a des chances de gain comprises à x % et y %."
   "G>x", "L'adversaire a des chances de gammon supérieures à x %."
   "G<x", "L'adversaire a des chances de gammon inférieures à x %."
   "Gx,y", "L'adversaire a des chances de gammon comprises à x % et y %."
   "B>x", "L'adversaire a des chances de backgammon supérieures à x %."
   "B<x", "L'adversaire a des chances de backgammon inférieures à x %."
   "Bx,y", "L'adversaire a des chances de backgammon comprises à x % et y %."
   "o>x", "Le joueur a au moins x pions sortis."
   "o<x", "Le joueur a au plus x pions sortis."
   "ox,y", "Le joueur a entre x et y pions sortis."
   "O>x", "L'adversaire a au moins x pions sortis."
   "O<x", "L'adversaire a au plus x pions sortis."
   "Ox,y", "L'adversaire a entre x et y pions sortis."
   "k>x", "Le joueur a au moins x pions arriérés."
   "k<x", "Le joueur a au plus x pions arriérés."
   "kx,y", "Le joueur a entre x et y pions arriérés."
   "K>x", "L'adversaire a au moins x pions arriérés."
   "K<x", "L'adversaire a au plus x pions arriérés."
   "Kx,y", "L'adversaire a entre x et y pions arriérés."
   "z>x", "Le joueur a au moins x pions dans la zone."
   "z<x", "Le joueur a au plus x pions dans la zone."
   "zx,y", "Le joueur a entre x et y pions dans la zone."
   "Z>x", "L'adversaire a au moins x pions dans la zone."
   "Z<x", "L'adversaire a au plus x pions dans la zone."
   "Zx,y", "L'adversaire a entre x et y pions dans la zone."
   "'tag_ou_motcle'", "Les commentaires de la position contient le tag/mot-clé."


Par exemple, la commande ``s s c p-20,-5 w>60 z>10 K2,3`` filtre toutes les
positions en prenant en compte la structure des pions, le score et le cube
de la position éditée où le joueur a entre 20 et 5 pips d'avance à la
course, avec au moins 60% de chances de gain, au moins 10 pions dans la
zone, et l'adversaire a entre 2 et 3 pions arriérés.
