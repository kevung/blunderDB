.. _roadmap:

Feuille de route
================

Cette page recense les pistes d'évolution envisagées pour blunderDB — ce
n'est **pas** un engagement de date ni d'ordre. blunderDB est développé par
une seule personne, sur son temps libre : une piste peut avancer vite,
attendre des mois, ou être abandonnée si une meilleure idée la remplace.

Le suivi précis se passe sur GitHub :

* les tickets ouverts sont regroupés par étape dans les jalons `Étape 3 —
  Étendre <https://github.com/kevung/blunderDB/milestone/4>`__ (ce qui suit
  cette page) et `Étape 4 — Fond
  <https://github.com/kevung/blunderDB/milestone/5>`__ (les chantiers plus
  lourds) ;
* pour proposer une idée ou en discuter, direction les `Discussions —
  catégorie Idées
  <https://github.com/kevung/blunderDB/discussions/categories/ideas>`__ ;
* chaque version publiée est annoncée dans les `Discussions — catégorie
  Annonces
  <https://github.com/kevung/blunderDB/discussions/categories/announcements>`__.

Étape 3 — Étendre
==================

Import et flux
--------------

* Un panneau de fin d'import : nombre de matchs importés, positions marquées,
  positions sans analyse, cinq pires erreurs (`#257
  <https://github.com/kevung/blunderDB/issues/257>`__).
* Surveiller un dossier (celui d'eXtreme Gammon par exemple) et importer
  automatiquement tout nouveau fichier (`#258
  <https://github.com/kevung/blunderDB/issues/258>`__).
* Après un import, parcourir une file d'étude qui ne montre que les
  blunders, les positions marquées et les décisions serrées du lot (`#259
  <https://github.com/kevung/blunderDB/issues/259>`__).
* Importer les matchs joués sur bgammon.org et reconnaître un identifiant de
  position OGID collé dans l'application (`#262
  <https://github.com/kevung/blunderDB/issues/262>`__).
* Garder trace de l'origine de chaque commentaire (utilisateur, eXtreme
  Gammon, GNUbg…) et les afficher tous, plutôt qu'un seul choisi au hasard
  (`#263 <https://github.com/kevung/blunderDB/issues/263>`__).

Catégoriser sans imposer de classification
-------------------------------------------

* Étiqueter automatiquement chaque position selon la phase de la partie
  (ouverture, milieu, course, bearoff), pour la filtrer et la retrouver dans
  les statistiques (`#264 <https://github.com/kevung/blunderDB/issues/264>`__).
* Un vocabulaire de tags suggérés (``#blitz``, ``#backgame``,
  ``#containment``…) avec autocomplétion et un panneau qui les liste par
  fréquence (`#265 <https://github.com/kevung/blunderDB/issues/265>`__).
* Un cinquième onglet de statistiques ventilant le PR et le taux d'erreur par
  phase, par tag et par score away, sous forme de matrice (`#266
  <https://github.com/kevung/blunderDB/issues/266>`__).

Analyse et évaluation
----------------------

* Sur la position courante, le verdict de videau à tous les scores d'un match
  de 5, 7 ou 9 points, en une seule grille (`#267
  <https://github.com/kevung/blunderDB/issues/267>`__).
* Un PR calculé sur les matchs sans analyse importée, en comparant le coup
  joué à la recommandation de l'évaluateur intégré (`#268
  <https://github.com/kevung/blunderDB/issues/268>`__).
* Comparer côte à côte, sur une même position, ce que disent les différents
  moteurs déjà présents dans la base (`#269
  <https://github.com/kevung/blunderDB/issues/269>`__).
* Publier ce que vaut l'évaluateur intégré face à une table de référence
  exacte, et le mesurer en continu face aux imports eXtreme Gammon (`#270
  <https://github.com/kevung/blunderDB/issues/270>`__).
* Afficher les règles en vigueur (cube max, Jacoby, beaver) à côté du
  verdict, et permettre de choisir la table d'équités de match par base
  (`#271 <https://github.com/kevung/blunderDB/issues/271>`__).
* Publier une mesure du 3-ply (temps, gain de précision) et l'exposer comme
  réglage si le compromis le justifie (`#272
  <https://github.com/kevung/blunderDB/issues/272>`__).

Pédagogie
---------

* Un premier module d'entraînement chronométré : compte de pips, EPC, point
  de prise (`#273 <https://github.com/kevung/blunderDB/issues/273>`__).
* Se fixer un objectif de progression (« PR sous 5 d'ici trois mois ») et
  suivre l'écart sur la courbe (`#274
  <https://github.com/kevung/blunderDB/issues/274>`__).
* Relier les statistiques d'étude (Anki) aux statistiques de jeu réel : est-ce
  qu'une catégorie révisée s'améliore vraiment (`#275
  <https://github.com/kevung/blunderDB/issues/275>`__) ?
* Des cartes de videau chaînées : doubler, puis prendre, comme deux
  questions distinctes plutôt qu'une seule carte à deux temps (`#276
  <https://github.com/kevung/blunderDB/issues/276>`__).
* Des parcours pédagogiques : une séquence ordonnée de positions commentées,
  distribuable comme une base filigranée, pensée pour un coach (`#277
  <https://github.com/kevung/blunderDB/issues/277>`__).

Partager une analyse
---------------------

* Un rendu unique du plateau, exportable en SVG ou PNG, fidèle à ce que
  montre l'écran (`#278 <https://github.com/kevung/blunderDB/issues/278>`__).
* Un rapport HTML autonome d'une session, d'un match ou d'un tournoi,
  imprimable en PDF par le navigateur (`#279
  <https://github.com/kevung/blunderDB/issues/279>`__).
* Exporter en CSV ou Parquet, avec un carnet Jupyter d'exemple tenu à jour
  (`#280 <https://github.com/kevung/blunderDB/issues/280>`__).
* Un index, sur ce site, des bases filigranées que d'autres joueurs
  partagent avec une identité vérifiable (`#281
  <https://github.com/kevung/blunderDB/issues/281>`__).

Recherche et bibliothèque
--------------------------

* Des collections qui se réévaluent à l'ouverture (un filtre nommé, jamais
  figé), et la comparaison de deux joueurs côte à côte dans l'onglet Joueurs
  (`#282 <https://github.com/kevung/blunderDB/issues/282>`__).
* Écrire une recherche en langage courant (« mes blunders de videau au
  score ») plutôt qu'en jetons, de façon déterministe et hors ligne (`#283
  <https://github.com/kevung/blunderDB/issues/283>`__).

Interface et plateforme
------------------------

* Un écran d'accueil qui propose, quand aucune base récente n'est ouverte :
  visite guidée, base d'exemple, importer mes matchs, ouvrir une base
  (`#284 <https://github.com/kevung/blunderDB/issues/284>`__).
* Une corbeille pour les positions supprimées et un « annuler » sur les
  suppressions les plus courantes (`#285
  <https://github.com/kevung/blunderDB/issues/285>`__).
* Des thèmes nommés — clair, sombre, contraste élevé, imprimable (`#286
  <https://github.com/kevung/blunderDB/issues/286>`__).
* Une série de petits conforts d'usage quotidien : journal d'activité
  consultable, palette de commandes, badges de recherche plus lisibles
  (`#287 <https://github.com/kevung/blunderDB/issues/287>`__).
* Dire clairement ce qui marche pour synchroniser une base entre plusieurs
  postes (et ce qui ne marche pas), et un bouton pour fusionner deux bases
  (`#288 <https://github.com/kevung/blunderDB/issues/288>`__).
* Un client minimal et une politique de version pour l'API du mode serveur
  (`#289 <https://github.com/kevung/blunderDB/issues/289>`__).
* Remonter dans l'application des capacités qui n'existent aujourd'hui que
  côté serveur : suspendre une carte, journal de révision, optimisation FSRS
  (`#290 <https://github.com/kevung/blunderDB/issues/290>`__).

Étape 4 — Chantiers de fond
============================

Ces pistes changent la nature d'une partie du logiciel ; chacune attend une
décision explicite avant d'être lancée.

* Classer automatiquement le type de jeu d'une position (course, holding,
  backgame, blitz…), nommer les thèmes d'erreur récurrents (trop passif,
  timing, gammon sous-estimé…) et regrouper les blunders qui se répètent par
  coût cumulé — la promesse du nom du produit (`#291
  <https://github.com/kevung/blunderDB/issues/291>`__).
* Des rollouts tronqués pour départager deux coups très proches, avec
  intervalle de confiance affiché (`#292
  <https://github.com/kevung/blunderDB/issues/292>`__).
* « Des positions comme celle-ci » : retrouver les positions les plus
  proches de celle à l'écran (`#293
  <https://github.com/kevung/blunderDB/issues/293>`__).
* Un mode quiz qui teste plutôt qu'il ne mémorise, avec un PR d'entraînement
  comparable au PR réel (`#294
  <https://github.com/kevung/blunderDB/issues/294>`__).
* Une interface web en consultation sur le mode serveur, pour tablette et
  mobile (`#295 <https://github.com/kevung/blunderDB/issues/295>`__).
* Un mode club ou coach sur le serveur : partager une bibliothèque en
  lecture, recevoir les matchs des élèves (`#296
  <https://github.com/kevung/blunderDB/issues/296>`__).
* Un réseau d'évaluation distillé, beaucoup plus rapide, et un cache
  d'évaluation qui survit à la fermeture de l'application (`#297
  <https://github.com/kevung/blunderDB/issues/297>`__).
* Expliquer un blunder en une phrase, sans recourir à un modèle de langage
  (`#298 <https://github.com/kevung/blunderDB/issues/298>`__).
* Un assistant optionnel, strictement local et jamais activé par défaut, via
  Ollama (`#299 <https://github.com/kevung/blunderDB/issues/299>`__).

Ce qui n'est **pas** prévu
===========================

Jouer contre l'évaluateur intégré changerait la nature du produit (un moteur
de jeu complet : règles, saisie, historique de partie) ; ce n'est pas à
l'ordre du jour de blunderDB (`#300
<https://github.com/kevung/blunderDB/issues/300>`__).
