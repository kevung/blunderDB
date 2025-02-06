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
   "meta", "Affiche les métadonnées de la base de données."
   "met", "Ouvre la table d'équité de match Kazaross-XG2."
   "tp2", "Ouvre la table des takepoints avec videau à 2."
   "tp2_live", "Ouvre la table des takepoints avec videau à 2 pour les courses longues."
   "tp2_last", "Ouvre la table des takepoints avec videau à 2 mort."
   "tp4", "Ouvre la table des takepoints avec videau à 4."
   "tp4_live", "Ouvre la table des takepoints avec videau à 4 pour les courses longues."
   "tp4_last", "Ouvre la table des takepoints avec videau à 4 mort."
   "gv1", "Ouvre la table des valeurs de gammon avec videau à 1."
   "gv2", "Ouvre la table des valeurs de gammon avec videau à 2."
   "gv4", "Ouvre la table des valeurs de gammon avec videau à 4."

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
   "e", "Charger toutes les positions de la base de données."


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
   situé entre le point 24 et le point 19.

.. note::
   blunderDB considère que le nombre de pions dans la zone est le nombre
   de pions situés entre le point 12 et le point 1.

.. note::
   blunderDB considère que l'outfield s'étend entre le point 18 et le point 7.

.. note::
   blunderDB considère que le jan s'étend entre le point 1 et le point 6.
   
.. tip::
   Les paramètres pour filtrer des positions peuvent être arbitrairement
   combinés.

.. csv-table::
   :header: "Requête", "Action"
   :widths: 10, 20
   :align: center

   "cube, cub, cu, c", "La position vérifie la configuration du cube."
   "score, sco, sc, s", "La position vérifie le score."
   "d", "La position vérifie le type de décision (pion ou cube)."
   "D", "La position vérifie le lancer de dés."
   "nc", "La position est sans contact."
   "M", "La position ou celle miroir vérifie les filtres."
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
   "bo>x", "Le joueur a au moins x blots dans l'outfield."
   "bo<x", "Le joueur a au plus x blots dans l'outfield."
   "box,y", "Le joueur a entre x et y blots dans l'outfield."
   "BO>x", "L'adversaire a au moins x blots dans l'outfield."
   "BO<x", "L'adversaire a au plus x blots dans l'outfield."
   "BOx,y", "L'adversaire a entre x et y blots dans l'outfield."
   "jb>x", "Le joueur a au moins x blots dans le jan."
   "jb<x", "Le joueur a au plus x blots dans le jan."
   "jbx,y", "Le joueur a entre x et y blots dans le jan."
   "JB>x", "L'adversaire a au moins x blots dans le jan."
   "JB<x", "L'adversaire a au plus x blots dans le jan."
   "JBx,y", "L'adversaire a entre x et y blots dans le jan."
   "t'mot1;mot2;...'", "Les commentaires de la position contiennent au moins un des mots."
   "m'motif1,motif2,...\'", "Les meilleurs coups de pions contenant au moins un des motifs."
   "m'ND,DT,DP,...\'", "Les meilleurs décisions de videau de No Double/Take, Double Take, Double Pass."
   "T>x", "Date d'ajout de la position après x (AAAA/MM/JJ)."
   "T<x", "Date d'ajout de la position avant x (AAAA/MM/JJ)."
   "Tx,y", "Date d'ajout de la position entre x et y (AAAA/MM/JJ)."


.. note:: Filtrer les positions en fonction du lancer de dés (`D`) implique *a
   fortiori* de filtrer les positions en fonction du type de décision (`d`).

.. note:: Pour le filtre de différence relative à la course (`p>x`, `p<x`,
   `px,y`), le joueur est en retard à la course par rapport à l'adversaire si
   `x>0` et en avance si `x<0`. Exemple: `p<-10` : le joueur a au moins 10 pips
   d'avance à la course. `p50,70` : le joueur a entre 50 et 70 pips de retard à
   la course.

Par exemple, la commande ``s s c p-20,-5 w>60 z>10 K2,3`` filtre toutes les
positions en prenant en compte la structure des pions, le score et le cube
de la position éditée où le joueur a entre 20 et 5 pips d'avance à la
course, avec au moins 60% de chances de gain, au moins 10 pions dans la
zone, et l'adversaire a entre 2 et 3 pions arriérés.

.. _cmd_misc:

Commandes diverses
------------------

.. csv-table::
   :header: "Commande", "Action"
   :widths: 10, 40
   :align: center

   "clear, cl", "Efface l'historique des commandes."
   "migrate_from_1_0_to_1_1", "Migre la base de données de la version 1.0 à la version 1.1."
