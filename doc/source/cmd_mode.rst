.. _cmd_mode:

Liste des requêtes
==================

.. warning::
   Dans la recherche de positions, par défaut, blunderDB prend en compte
   la structure de pions courante, ignore la position du videau et du
   score. Pour prendre en compte la position du videau ou du score, il
   faut le mentionner explicitement dans la requête.

.. note::
   blunderDB considère qu'un pion arriéré (backchecker) est un pion
   situé entre le point 24 et le point 14.

.. note::
   blunderDB considère que le nombre de pions dans la zone est le nombre
   de pions situés entre le point 12 et le point 1.

.. note::
   Les paramètres pour filtrer des positions peuvent être arbitrairement
   combinés.

Opérations globales
-------------------

.. csv-table::
   :header: "Requête", "Action"
   :widths: 5, 20
   :align: left

   ":n", "Crée une nouvelle base de données."
   ":o", "Ouvre une base de données existante."
   ":q", "Ferme blunderDB."

Interagir avec une position
---------------------------

.. csv-table::
   :header: "Requête", "Action"
   :widths: 5, 20
   :align: left

   ":i", "Importe une position par fichier texte (txt)."
   ":w", "Enregistre la position courante dans la bibliothèque
   courante."
   ":w!", "Après édition d'une position existante, modifie cette
   dernière dans la base de données."
   ":w *toto* *titi* ...", "Enregistre la position courante dans les bibliothèques
   *toto*, *titi*, ..."
   ":w -*toto*", "Retirer la position courante de la bibliothèque
   *toto*."
   ":LS", "Liste les bibliothèques auxquelles la position courante
   appartient."
   ":D", "Supprime la position courante."

Gérer les bibliothèques
-----------------------

.. csv-table::
   :header: "Requête", "Action"
   :widths: 5, 20
   :align: center

   ":e *toto*", "Ouvre la bibliothèque *toto*."
   ":mv *titi*", "Renomme la bibliothèque courante en *titi*."
   ":mv *toto* *titi*", "Renomme la bibliothèque *toto* en *titi*."
   ":cp *titi*", "Copie la bibliothèque courante dans la bibliothèque
   *titi*."
   ":cp *toto* *titi*", "Copie la bibliothèque *toto* dans *titi*."
   ":d", "Supprime la bibliothèque courante."
   ":d *toto*", "Supprime la bibliothèque *toto*."
   ":ls", "Liste les bibliothèques."

Rechercher des positions
------------------------

.. csv-table::
   :header: "Requête", "Action"
   :widths: 5, 20
   :align: center

   ":s cube ...

   :s cu ...

   :s c ...", "Filtre les positions vérifiant la configuration
   courante du cube."
   ":s score ...

   :s sc ...

   :s s", "Filtre les positions vérifiant le score courant."
   ":s o7", "Filtre les positions ayant au moins 7 pions sortis."
   ":s p-20,30", "Filtre les positions où le joueur 1 a entre 20 pips d'avance et
   30 pips de retard de course."
   ":s p<-10", "Filtre les positions où le joueur 1 a au moins 10 pips
   d'avance."
   ":s p>110", "Filtre les positions où le joueur 1 a au moins 110 pips
   de retard."
   ":s P5,40", "Filtre les positions ayant une différence de course entre
   5 et 40 pips."
   ":s k5", "Filtre les positions où le joueur 1 a 5 pions arriérés."
   ":s K2", "Filtre les positions où le joueur 2 a 2 pions arriérés."
   ":s z8", "Filtre les positions où le joueur 1 a 8 pions dans la zone."
   ":s Z6", "Filtre les positions où le joueur 2 a 6 pions dans la zone."
   ":s e800,1200", "Filtre les positions où le joueur 1 a une équité
   entre 800 et 1200."
   ":s e<200", "Filtre les positions où le joueur 1 a une équité
   inférieure à 200."
   ":s e>400", "Filtre les positions où le joueur 1 a une équité
   supérieure à 400."
   ":s w40,60", "Filtre les positions où le joueur 1 a des chances de
   gains entre 40% et 60%."
   ":s w<25", "Filtre les positions où le joueur 1 a des chances de
   gain inférieures à 25."
   ":s w>68", "Filtre les positions où le joueur 1 a des chances de
   gain supérieures à 68%."
   ":s g14,22", "Filtre les positions où le joueur 1 a des chances de
   gammon entre 14% et 22%."
   ":s g<27", "Filtre les positions où le joueur 1 a des chances de
   gammon inférieures à 27%."
   ":s g>45", "Filtre les positions où le joueur 1 a des chances de
   gammon supérieures à 45%."
   ":s bg4,8", "Filtre les positions où le joueur 1 a des chances de
   backgammon entre 4% et 8%."
   ":s bg<10", "Filtre les positions où le joueur 1 a des chances de
   backgammon inférieures à 10%."
   ":s bg>5", "Filtre les positions où le joueur 1 a des chances de
   backgammon supérieures à 5%."
   ":s W40,60", "Filtre les positions où le joueur 2 a des chances de
   gains entre 40% et 60%."
   ":s W<25", "Filtre les positions où le joueur 2 a des chances de
   gain inférieures à 25."
   ":s W>68", "Filtre les positions où le joueur 2 a des chances de
   gain supérieures à 68%."
   ":s G14,22", "Filtre les positions où le joueur 2 a des chances de
   gammon entre 14% et 22%."
   ":s G<27", "Filtre les positions où le joueur 2 a des chances de
   gammon inférieures à 27%."
   ":s G>45", "Filtre les positions où le joueur 2 a des chances de
   gammon supérieures à 45%."
   ":s BG4,8", "Filtre les positions où le joueur 2 a des chances de
   backgammon entre 4% et 8%."
   ":s BG<10", "Filtre les positions où le joueur 2 a des chances de
   backgammon inférieures à 10%."
   ":s BG>5", "Filtre les positions où le joueur 2 a des chances de
   backgammon supérieures à 5%."

