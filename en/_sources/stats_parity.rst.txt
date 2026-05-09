.. _stats_parity:

Annexe : Modèle de statistiques — alignement XG / gnuBG / blunderDB
=====================================================================

Cette page décrit comment blunderDB calcule le **PR** (Performance Rate),
le **Snowie Error Rate** et la **perte MWC**, et comment ces métriques sont
alignées sur eXtreme Gammon (XG) et gnuBG (référence ouverte).

.. contents::
   :local:
   :depth: 2


Définitions formelles
---------------------

PR (Performance Rate)
~~~~~~~~~~~~~~~~~~~~~

Le PR (aussi appelé « error rate per decision » dans gnuBG) mesure l'erreur
moyenne en millièmes de point de jetons (millipoints, mpt) par décision
comptée.

.. math::

   \mathrm{PR} = \frac{\sum_i |\mathrm{erreur}_i|}{\mathrm{N_{compté}}} \times 500

- Le numérateur est la somme absolue des erreurs EMG (en equity cubeful) sur
  toutes les décisions du périmètre.
- Le dénominateur :math:`N_\text{compté}` est le nombre de **décisions comptées**
  (voir ci-dessous).
- Le facteur 500 convertit l'equity en millipoints (1 point = 1000 mpt, mais
  l'échelle est ×500 par convention XG/gnuBG — cf. ``gnubg/formatgs.c:399–409``).

Snowie Error Rate
~~~~~~~~~~~~~~~~~

Le Snowie ER utilise le **même numérateur** que le PR, mais le dénominateur est
le nombre total de coups des deux joueurs, coups forcés inclus (toutes décisions,
sans filtre) :

.. math::

   \mathrm{SnowieER} = \frac{\sum_i |\mathrm{erreur}_i|}{N_\text{P1} + N_\text{P2}} \times 500

Référence : ``gnubg/formatgs.c:415–424``.

Le Snowie ER est plus stable entre les outils car son dénominateur ne dépend pas
du filtre des décisions forcées/triviales. Il sert de métrique de recoupement
XG ↔ gnuBG ↔ blunderDB.

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

Référence : ``gnubg/analysis.c:1449–1464``.


Décisions comptées au dénominateur du PR
-----------------------------------------

blunderDB suit les mêmes règles d'exclusion qu'XG et gnuBG.

Coups de pions — décisions comptées
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Seuls les **coups non-forcés** sont comptés :

- Un coup est **forcé** si le dé n'offre qu'un seul coup légal (``cMoves == 1``
  dans ``gnubg/analysis.c:458``).
- Les coups forcés ont une erreur nulle par définition : le joueur n'avait pas
  le choix. Les inclure dans le dénominateur abaisserait artificiellement le PR.

Décisions de cube — décisions comptées
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

Seules les **décisions de cube proches** sont comptées :

- Une décision cube est **proche** si elle se situe dans la fenêtre d'équité
  ``[-0.16, +0.16]`` autour du point de redoublement (prédicat
  ``isCloseCubedecision`` dans ``gnubg/eval.c:5088–5100``).
- Un « No Double » trivial (équité très négative ou très positive) n'est pas
  une vraie décision stratégique ; l'inclure gonflerait le dénominateur et
  dépresserait le PR.

Résumé du filtre
~~~~~~~~~~~~~~~~~

+--------------------+---------------------------------------------+
| Type de décision   | Inclus dans :math:`N_\text{compté}` (PR)    |
+====================+=============================================+
| Coup non-forcé     | Oui                                         |
+--------------------+---------------------------------------------+
| Coup forcé         | Non                                         |
+--------------------+---------------------------------------------+
| Cube proche        | Oui                                         |
+--------------------+---------------------------------------------+
| No Double trivial  | Non                                         |
+--------------------+---------------------------------------------+
| Take / Pass        | Toujours (ce sont des réponses au double)   |
+--------------------+---------------------------------------------+


Correspondance blunderDB ↔ XG ↔ gnuBG
---------------------------------------

Les métriques sont alignées dans les limites suivantes (mesurées sur 3 matchs de référence) :

Comparaison XG ↔ blunderDB (même moteur d'analyse)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

+---------------------+------------------+
| Métrique            | Écart typique    |
+=====================+==================+
| Décisions totales   | ≤ 5              |
+---------------------+------------------+
| Coups non-forcés    | ≤ 7              |
+---------------------+------------------+
| PR                  | ≤ 0.10           |
+---------------------+------------------+
| Perte MWC           | ≤ 1.0 pp         |
+---------------------+------------------+
| Equity total (EMG)  | ≤ 0.05           |
+---------------------+------------------+

Comparaison gnuBG ↔ blunderDB (import SGF — moteurs différents)
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

+---------------------+-----------------------------------------------+
| Métrique            | Écart typique  | Cause principale             |
+=====================+================+==============================+
| PR (checker)        | ≤ 0.20         | equity cross-engine          |
+---------------------+----------------+------------------------------+
| Perte MWC           | ≤ 3.5 pp       | close-cube SGF incomplet     |
+---------------------+----------------+------------------------------+
| Snowie ER           | ≤ 0.50         | forcés sans analyse (SGF)    |
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

1. **Analyses complètes** — Le PR ne peut être calculé que sur les positions
   disposant d'une analyse. Des positions sans analyse sont comptées comme
   erreur zéro mais n'entrent pas dans :math:`N_\text{compté}`.

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

Les formules ont été vérifiées dans les fichiers source suivants (dépôt gnuBG) :

- ``gnubg/formatgs.c:399–409`` — PR (« Error rate per decision »).
- ``gnubg/formatgs.c:415–424`` — Snowie Error Rate.
- ``gnubg/analysis.c:458–462`` — Accumulation checker, exclusion des forcés (``cMoves > 1``).
- ``gnubg/analysis.c:1430–1474`` — Conversion EMG → MWC par décision.
- ``gnubg/analysis.c:1449–1464`` — Accumulation de la perte MWC (``eq2mwc``).
- ``gnubg/eval.c:5088–5100`` — Prédicat ``isCloseCubedecision`` (seuil 0.16).
