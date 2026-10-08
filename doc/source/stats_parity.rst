.. _stats_parity:

Annexe : Modèle de statistiques — alignement XG / gnuBG / blunderDB
=====================================================================

Cette page décrit comment blunderDB calcule les indicateurs qu'il affiche —
**PR** (*Performance Rating*), **Snowie Error Rate**, **perte MWC**, **chance** et
**victoires/défaites** — et comment ils s'alignent sur eXtreme Gammon (XG) et
gnuBG (référence ouverte).

.. contents::
   :local:
   :depth: 2


Définitions formelles
---------------------

PR (Performance Rating)
~~~~~~~~~~~~~~~~~~~~~~~

Le PR est l'erreur moyenne d'équité par décision comptée, multipliée par 500
comme le fait eXtreme Gammon :

.. math::

   \mathrm{PR} = \frac{\sum_i |\mathrm{erreur}_i|}{\mathrm{N_{compté}}} \times 500

- Le numérateur est la somme des erreurs en équité normalisée (EMG au score),
  chacune celle du coup joué dans son match.
- Le dénominateur :math:`N_\text{compté}` compte les décisions selon les règles
  d'XG : coups de pions non forcés, doubles proposés, prises et refus, et
  « pas de double » proches. Ces règles, leur motivation et leurs limites sont
  détaillées dans :ref:`metrique_pr`.
- Le facteur 500 est la convention d'XG. Ce n'est pas une conversion en
  millipoints : une erreur moyenne de 0,010 d'équité — soit 10 millipoints
  (mpt) — donne un PR de 5,0. Les erreurs d'une position, elles, se lisent bien
  en millipoints, c'est-à-dire en millièmes d'équité : le seuil de blunder
  ci-dessous, les filtres de recherche ``e>`` et ``E>``, l'histogramme des
  magnitudes du panneau Stats.

Snowie Error Rate
~~~~~~~~~~~~~~~~~

Le Snowie ER utilise le **même numérateur** que le PR, mais le dénominateur est
le nombre total de coups des deux joueurs, coups forcés inclus (toutes décisions,
sans filtre) :

.. math::

   \mathrm{SnowieER} = \frac{\sum_i |\mathrm{erreur}_i|}{N_\text{P1} + N_\text{P2}} \times 500

Référence : ``gnubg/formatgs.c:415–424``.

Le Snowie ER est plus stable entre les outils car son dénominateur ne dépend
d'aucune règle de comptage des décisions. Il sert de métrique de recoupement
XG ↔ gnuBG ↔ blunderDB.

.. important::

   **Le filtre « Joueur » ne s'applique qu'au numérateur.** Sélectionner un
   joueur restreint les erreurs sommées à ses propres décisions, mais le
   dénominateur reste le nombre de coups de pions des **deux** joueurs, sur les
   matchs retenus par le filtre. C'est ce qui distingue le Snowie ER du PR, et
   c'est la raison de la propriété suivante : sur une base ne contenant que des
   matchs entre deux joueurs, la somme de leurs deux Snowie ER est exactement
   le Snowie ER non filtré.

.. note::

   Le Snowie ER d'un joueur est typiquement environ la moitié de son PR, car le
   dénominateur inclut les coups des deux joueurs alors que le PR n'utilise que
   les décisions de ce joueur.

Perte MWC (Match Winning Chance)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

La perte MWC exprime en points de pourcentage de probabilité de gagner le match
l'effet cumulé des erreurs d'un joueur. Pour chaque décision, l'erreur EMG est
convertie en MWC via la table MET (Match Equity Table) au score courant :

.. math::

   \mathrm{MWCLoss} = \sum_i \mathrm{eq2mwc}(\mathrm{erreur}_i, \mathrm{score}_i)

Une prise ou un refus est converti au videau d'avant le double, comme la
décision du doubleur et comme le fait XG. La conversion et ses limites sont
détaillées dans :ref:`metrique_perte_mwc`.

Chance (*luck*)
~~~~~~~~~~~~~~~

La **chance** d'un lancer est l'écart entre ce que les dés ont donné et ce qu'ils
donnent en moyenne :

.. math::

   \mathrm{luck} = \mathrm{eq}(\text{meilleur coup avec les dés obtenus})
                 - \frac{1}{36}\sum_{d} \mathrm{eq}(\text{meilleur coup avec } d)

la somme portant sur les 36 issues du lancer (30 lancers non doubles pour le
premier coup de la partie, qui ne peut pas être un double). Une valeur positive
signifie un lancer favorable. Référence : ``gnubg/analysis.c:199–269``
(``LuckNormal`` / ``LuckFirst``, évaluation cubeful à 0 ply).

.. important::

   **blunderDB ne recalcule jamais la chance** : l'évaluateur intégré ne sert
   pas à cela. Il reprend telle quelle la valeur écrite par l'outil qui a
   analysé le match — ``ErrLuck`` pour eXtreme Gammon, la propriété ``LU`` pour
   gnuBG — après vérification que les deux partagent la même convention (même
   unité, positif = chanceux).

Conséquences pratiques :

* Les formats qui ne transportent pas la chance (**BGF**, **Jellyfish .mat**) ne
  produisent aucune valeur.
* Les matchs **déjà importés** n'en ont pas non plus : la chance n'a jamais été
  stockée avant la version 2.15.0 du schéma, et rien dans la base ne permet de
  la reconstituer. Il faut **réimporter** les fichiers source pour l'obtenir.
* Une valeur absente signifie *inconnue*, jamais *nulle* — zéro est un lancer
  réellement neutre. Les moyennes de chance ne divisent donc que par le nombre
  de lancers dont la chance est connue.
* Un match dont *tous* les lancers auraient une chance exactement nulle est
  considéré comme non analysé de ce point de vue (eXtreme Gammon écrit 0 aussi
  bien pour « neutre » que pour « non calculé ») : ses lancers restent inconnus.
  Côté gnuBG la distinction est explicite — un lancer non évalué porte
  ``LU[-inf]`` — et ces lancers restent inconnus eux aussi, plutôt que d'être
  convertis en une valeur.

Victoires et défaites
~~~~~~~~~~~~~~~~~~~~~

blunderDB **ne stocke pas de vainqueur au niveau du match** : seuls les points
de chaque partie le sont. Le résultat est donc dérivé à l'affichage, en cumulant
les points par siège :

* **Match en N points** — vainqueur = le joueur qui atteint N. Si personne ne
  l'atteint, le match est **inachevé** (journal tronqué, abandon) et ne compte
  ni victoire ni défaite, pour personne.
* **Partie d'argent** (sans cible) — vainqueur = celui qui a pris le plus de
  points ; à égalité, ni victoire ni défaite. C'est la convention du résultat de
  session de gnuBG (+1 / 0 / −1, ``gnubg/relational.c``).

Conséquence à connaître en lisant l'onglet Joueurs : **victoires + défaites peut
être inférieur au nombre de matchs**. L'écart, ce sont les matchs inachevés.

Erreurs et blunders
~~~~~~~~~~~~~~~~~~~

Une décision est comptée comme **erreur** dès que l'erreur d'équité associée
atteint le **seuil d'erreur** de la bibliothèque, et comme **blunder** dès
qu'elle atteint son **seuil de blunder**. Les deux se règlent dans l'onglet
*Bibliothèque* de la configuration et valent par défaut **0,050** et
**0,100 EMG** (50 et 100 mpt). La comparaison est *inclusive* : une erreur
d'exactement 0,100 est un blunder au seuil par défaut, dans tous les écrans
(répartition par action de videau, détail de match, tableau des joueurs), et
la même paire sert au compteur de la barre d'état et à la liste proposée
après un import.

Ces seuils sont propres à chaque bibliothèque, et les valeurs par défaut sont
propres à blunderDB : gnuBG classe les coups en trois crans (``0.03`` douteux,
``0.06`` mauvais, ``0.12`` très mauvais — ``gnubg/gnubg.c:281–286``) plutôt
qu'en une catégorie unique, et eXtreme Gammon trace les siennes à 0,020 et
0,080. La configuration propose trois préréglages : blunderDB (0,050 et
0,100), XG (0,020 et 0,080) et GNUbg (0,040 et 0,080).


Décisions comptées : XG et gnuBG
--------------------------------

blunderDB compte les décisions comme XG (:ref:`metrique_pr`). gnuBG suit
d'autres règles, si bien que son taux par décision ne se compare pas au PR
d'XG, même calculé sur la même analyse :

+--------------------+------------------------------+------------------------------+
| Décision           | XG et blunderDB              | gnuBG                        |
+====================+==============================+==============================+
| Coup de pions      | plus d'un coup légal, sauf   | plus d'un coup légal         |
|                    | si l'analyse les couvre tous | (``cMoves > 1``)             |
|                    | à la même équité au millième |                              |
+--------------------+------------------------------+------------------------------+
| Double proposé     | toujours                     | toujours                     |
+--------------------+------------------------------+------------------------------+
| Prise, refus       | toujours                     | toujours                     |
+--------------------+------------------------------+------------------------------+
| Pas de double      | ND > 0 et                    | décision « proche » selon    |
|                    | ND − min(D/T, D/P) < 0,200   | ``isCloseCubedecision``      |
+--------------------+------------------------------+------------------------------+

La condition ND > 0 n'est pas dans le manuel d'XG : XG ne compte pas le « pas
de double » d'un joueur qui est derrière, et les matchs de référence le
montrent.


Correspondance blunderDB ↔ XG ↔ gnuBG
---------------------------------------

Les métriques sont alignées dans les limites ci-dessous. Ce sont des **bornes
vérifiées par la suite de tests**, et non des écarts moyens : le test compare
match par match les chiffres de blunderDB à ceux enregistrés pour les matchs de
référence du dépôt, et échoue dès qu'un seul écart dépasse la borne. Les
références XG portent sur cinq matchs en 7 points, les deux joueurs de chacun ;
les références gnuBG sur deux d'entre eux. La version d'eXtreme Gammon qui a
produit ces chiffres n'est pas enregistrée.

Comparaison XG ↔ blunderDB (même analyse)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Les comptes de décisions — coups de pions, doubles, prises, décisions de
videau proches — sont **exacts**. Les écarts restants sont ceux de l'affichage
d'XG, qui arrondit ce que blunderDB additionne au millipoint :

+------------------------------------+------------------+
| Métrique                           | Écart maximal    |
+====================================+==================+
| Décisions comptées                 | 0                |
+------------------------------------+------------------+
| PR                                 | ≤ 0,02           |
+------------------------------------+------------------+
| Perte MWC (totale, pions, doubles, | ≤ 0,06 point     |
| réponses)                          |                  |
+------------------------------------+------------------+
| Erreur d'équité (EMG)              | ≤ 0,004          |
+------------------------------------+------------------+

Écarts résiduels connus
~~~~~~~~~~~~~~~~~~~~~~~~

Quatre joueurs des matchs de référence s'écartent davantage, chacun pour une
cause qui tient à ce que le fichier ne transporte pas :

* **Coup non analysé** (deux joueurs). XG écrit, pour un coup qu'il n'a pas
  noté, une erreur de −1000 et le laisse hors du compte ; cette valeur n'est
  pas lue à l'import, et blunderDB note le coup à partir des candidats
  stockés. blunderDB compte une décision de plus ; le PR s'écarte jusqu'à
  0,09, la perte MWC jusqu'à 0,45 point.
* **Sortie jugée hors du fichier** (un joueur). XG compte comme décision une
  sortie dont les candidats stockés couvrent tous les coups légaux à la même
  équité : il l'a jugée sur des évaluations que le fichier ne garde pas.
  blunderDB compte une décision de moins ; le PR s'écarte de 0,02.
* **Somme des erreurs de pions** (un joueur). Les comptes et le videau
  concordent ; la somme des erreurs des coups stockés dépasse de quelques
  millipoints le total de pions d'XG (2,150 contre 2,147).

Comparaison gnuBG ↔ blunderDB (import SGF — moteurs différents)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

+---------------------+----------------+------------------------------+
| Métrique            | Écart maximal  | Cause principale             |
+=====================+================+==============================+
| PR (pions)          | ≤ 0,40         | règle des coups forcés,      |
|                     |                | équités entre moteurs        |
+---------------------+----------------+------------------------------+
| Perte MWC           | ≤ 3,5 points   | videau proche incomplet      |
|                     |                | dans le SGF                  |
+---------------------+----------------+------------------------------+
| Snowie ER           | ≤ 0,50         | forcés sans analyse (SGF)    |
+---------------------+----------------+------------------------------+

.. note::

   Les fichiers SGF (gnuBG) n'incluent pas les alternatives pour les coups
   forcés, ce qui signifie que blunderDB ne peut pas détecter tous les coups
   forcés à l'import SGF. Cela crée un écart structurel sur le Snowie ER
   (dénominateur légèrement différent).


Valider vos propres chiffres
-----------------------------

Si vos valeurs PR ou MWC divergent des chiffres XG, vérifier les points
suivants :

1. **Analyses complètes** — Le PR ne peut être calculé que sur les décisions
   disposant d'une analyse. Un coup que l'analyse ne note pas n'entre ni au
   numérateur ni dans :math:`N_\text{compté}`.

2. **Version XG** — XG peut changer ses calculs entre versions. blunderDB
   s'aligne sur le comportement observé des versions récentes.

3. **Format d'import** — Les fichiers SGF gnuBG produisent des écarts plus
   importants sur le cube (voir tableau ci-dessus) car le fichier n'inclut
   pas les analyses complètes pour tous les cubes.

4. **Migration de base** — Après mise à jour de blunderDB, les bases existantes
   sont migrées automatiquement. Faire une sauvegarde avant d'ouvrir une base
   avec une nouvelle version.


Référence gnuBG
---------------

Les formules de gnuBG citées sur cette page se trouvent dans les fichiers
source suivants (dépôt gnuBG) :

- ``gnubg/formatgs.c:399–409`` — taux d'erreur par décision (« Error rate per decision »).
- ``gnubg/formatgs.c:415–424`` — Snowie Error Rate.
- ``gnubg/analysis.c:458–462`` — Accumulation pions, exclusion des forcés (``cMoves > 1``).
- ``gnubg/analysis.c:1430–1474`` — Conversion EMG → MWC par décision.
- ``gnubg/analysis.c:1449–1464`` — Accumulation de la perte MWC (``eq2mwc``).
- ``gnubg/eval.c:5088–5100`` — Prédicat ``isCloseCubedecision``.
