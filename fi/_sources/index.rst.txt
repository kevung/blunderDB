.. _index:

blunderDB
=========

blunderDB est un logiciel pour constituer des bases de données de positions de
backgammon. Sa force principale est de constituer un lieu unique où aggréger
les positions qu'un joueur a pu rencontrer (en ligne, en tournoi) et de pouvoir
réétudier ces positions en les filtrant selon différents filtres combinables
arbitrairement. blunderDB peut également être utilisé pour constituer des
catalogues de positions de référence.

La présente documentation est structurée de la manière suivante:

* la section **téléchargement et installation** explique comment se procurer et
  lancer blunderDB.

* le **manuel** décrit le fonctionnement général de blunderDB.

* le **guide utilisateur** est une introduction pratique pour utiliser
  rapidement blunderDB.

* la liste des **commandes** ainsi que la liste des **raccourcis**
  clavier permettent une utilisation efficace de blunderDB.

* la section **interface en ligne de commande (CLI)** décrit les commandes
  disponibles pour l'import en masse, l'automatisation et le scripting.

* la **FAQ** fournit quelques réponses aux interrogations les plus fréquentes.

Historique des versions
=======================

.. csv-table::
   :header: "Version", "Date", "Cause et/ou nature des évolutions"
   :widths: 5, 7, 20
   :align: center
   :class: align-center-table

   0.1.0, 31/12/2024, "Création version beta."
   0.2.0, 06/01/2025, "Résolutions de bugs divers. 
   
   Ajout de tables de matchs/TP/GV.

   Ajout de filtres de recherche (coups, décision de videau, date).
   
   Ajout de métadonnées sur les positions.

   Fonction d'import/export entre instances de blunderDB.

   Ajout de fonction de métadonnées sur les bases de données.
   
   Introduction des numéros de versions (base de données et application)."
   0.3.0, 27/01/2025, "Résolutions de bugs divers. 

   Sauvegarde automatiquement le dimensionnement de la fenêtre.

   Importe les éventuels commentaires depuis XG."
   0.4.0, 03/02/2025, "Résolutions de bugs divers. 

   Ajout d'une icone pour blunderDB.

   Corrections de filtres.

   Ajout du support de MacOS."
   0.5.0, 04/02/2025, "Ajout de nouveaux filtres (miroir, non contact, jan blot, outfield blot)."
   0.6.0, 13/02/2025, "Ajout de la bibliothèque de filtres.

   Affichage de la version de la base de données dans les métadonnées."
   0.7.0, 16/02/2025, "Prise en charge du japonais et de l'allemand dans les exports de XG."
   0.8.0, 03/05/2025, "Possibilité de cacher le compte de course.
   
   Chargement d'une position aléatoire."
   0.9.0, 02/11/2025, "Correction de bug de la bibliothèque de filtres.
   
   Import/export de base de données.
   
   Affichage de flèches pour les coups sélectionnés.
   
   Raccourcis clavier pour l'import/export."
   0.10.0, 25/02/2026, "Import de matchs depuis eXtreme Gammon (XG/XGP), GNUbg (SGF), Jellyfish (MAT/TXT) et BGBlitz (BGF/TXT).

   Navigation dans les matchs: parcours des coups d'un match importé, avec mise en évidence du coup joué.

   Panneau des matchs: liste, tri, édition inline, permutation des joueurs, assignation de tournoi.

   Import par dossier récursif et import par glisser-déposer.

   Calculateur EPC (Effective Pip Count) avec base de données de bearoff GNUbg intégrée.

   Collections: regroupement personnalisé de positions.

   Tournois: regroupement de matchs par événement.

   Sauvegarde et restauration de l'état de session (dernière recherche, position courante).

   Migration automatique du schéma de base de données.

   Affichage multi-moteurs dans l'analyse.

   Filtre d'erreurs/blunders du joueur 1 dans les recherches.

   Export de la base de données avec sélection granulaire (matchs, collections, tournois, coups joués).

   Bouton de navigation dans les matchs.

   Compte de course (pipcount) dans la navigation des matchs.

   Interface en ligne de commande (CLI) complète.

   Réouverture automatique de la dernière base de données.

   Amélioration de la barre d'outils et des icônes."
   0.11.0, 06/03/2026, "Filtre de recherche dans les positions courantes.

   Ajout de filtres par match et par tournoi.

   Effacement automatique du plateau lors de l'ouverture du panneau de recherche."
   0.12.0, 19/03/2026, "Import de fichiers de position eXtreme Gammon (XGP) avec analyse."
   0.13.0, 28/03/2026, "Simplification de l'interface: la navigation dans les matchs et les collections se fait directement via les panneaux.

   Ligne de commande intégrée dans la barre d'état.

   Panneau Console renommé en panneau Log.

   Panneau EPC dédié dans le panneau inférieur.

   Copier/coller de position dans le panneau de recherche.

   Glisser-déposer pour réordonner les collections, les positions dans les collections, et les matchs dans les tournois.

   Colonne tournoi dans le panneau des matchs avec édition inline.

   Affichage automatique du panneau d'analyse après une recherche."
   0.14.0, 30/03/2026, "Panneau Anki dédié pour l'étude par répétition espacée (algorithme FSRS).

   Import des commentaires depuis les fichiers XG."
   0.15.0, 31/03/2026, "Export de la position en image PNG dans le presse-papier (board seul via Ctrl+X, ou board avec analyse via Ctrl+X Ctrl+X)."
   0.16.0, 18/04/2026, "Schéma de base de données v2.0.0 : déduplication des positions via hash Zobrist, colonnes de filtrage dénormalisées, préfiltre de motifs bitboard, journalisation WAL. Import par lot >=3x plus rapide, recherche filtrée <=100 ms sur 10k+ positions. NOTE : les fichiers DB créés avec la v0.16.0 ne peuvent pas être ouverts par les versions plus anciennes ; les anciennes DB sont migrées automatiquement sur place (faire une sauvegarde d'abord)."
   0.17.0, 20/04/2026, "Optimisation du stockage : compression zlib des données d'analyse (~80% de réduction), encodage compact des positions (~90% de réduction de la taille). Ajout de 5 index manquants pour améliorer les performances de recherche. Correction de la recherche par erreur de cube. Correction du mode EDIT après une recherche sans résultats. Restauration de l'état du panneau de recherche lors du changement d'onglets. Suppression de 62 instructions de débogage des chemins critiques."
   0.18.0, 20/04/2026, "Refactoring majeur du code : découpage de db.go (10k lignes) en 19 fichiers spécialisés, extraction de 7 modules de service depuis App.svelte (4888→469 lignes), consolidation des stores modaux/panneaux. Migration complète vers Svelte 5 runes. Remplacement de 9 modales de tableau par un composant générique DataTableModal. Ajout d'ESLint + Prettier + vitest (125 tests frontend) avec CI. Conformité WCAG 2.1 AA (focus visible, rôles ARIA, navigation clavier). Passage du mutex Database en RWMutex pour un meilleur parallélisme en lecture. Documentation CLI complète (CLI_USAGE.md + Sphinx FR/EN). Réécriture du README. Correction de tous les avertissements ESLint (46→0) et Vite (6→0)."
   0.19.0, 07/05/2026, "Ajout du panneau Stats : indicateurs PR (Performance Rate), Snowie Error Rate et MWC cost (Match Winning Chance cost), barre de filtre (joueur, tournoi, dates, type de décision, longueur de match), onglet Dashboard avec cartes de niveau / PR glissant / top blunders, onglet Progression avec courbe par tournoi et scatter plot par match, onglet Erreurs avec répartition par action de videau et histogramme des magnitudes. Drill-down interactif vers les positions / matchs / tournois depuis tous les indicateurs. Toggle PR / MWC instantané. Commande CLI list --type stats. Alignement des indicateurs PR / Snowie ER / MWC sur eXtremeGammon et gnuBG (seuil 0.16 d'équité pour les cubes proches). Correction du calcul de cube_error pour les décisions Double/Pass. Documentation du modèle de statistiques (:ref:`stats_parity`). Voir :ref:`stats`."
   0.20.0, 31/05/2026, "Ajout de la structure d'exclusion *Except* au panneau de recherche : exclut les positions contenant l'un des pions dessinés, avec marqueur « doit être vide » (double-clic) et nombre de pions par point non limité (commande x). Ajout de l'option « premier dé uniquement » au filtre de lancer de dés (variante D1, option CLI --dice). Panneau Commentaires : focus automatique du champ de saisie à l'ouverture et boutons éditer / supprimer toujours visibles. Correction du filtre « Search Text » qui ne trouvait pas tous les tags de commentaires."
   0.21.0, 01/06/2026, "Internationalisation de l'interface : l'intégralité de blunderDB (barre d'outils, panneaux, messages, aide) peut désormais être affichée au choix en anglais, français, allemand, italien, espagnol, finnois, japonais, grec ou russe. Ajout d'une fenêtre de configuration, accessible par un bouton en forme de rouage dans la barre d'outils, permettant de sélectionner la langue. Le choix de la langue est conservé d'une session à l'autre. Voir :ref:`configuration`."
   0.22.0, 02/06/2026, "Ajout d'un mode headless (serveur), avancé et facultatif, qui complète l'application de bureau : un démon ``serve`` exposant le moteur en HTTP + JSON, un backend PostgreSQL multi-utilisateur avec cloisonnement par tenant (et Row-Level Security en option), une commande ``migrate`` pour transférer une base SQLite vers PostgreSQL, et un dispatcher générique ``call`` donnant accès en ligne de commande à toutes les opérations de stockage. Import de positions uniques depuis de nouveaux formats (eXtreme Gammon ``.xgp``, BGBlitz texte, bibliothèque native ``.db``) avec enrichissement des doublons entre formats. Corrections : les panneaux ne provoquent plus d'erreur lorsqu'aucune base n'est ouverte et le panneau Commentaires ne boucle plus à l'ouverture. Voir :ref:`headless`."
   0.23.0, 05/06/2026, "Ajout de visites guidées de l'interface (tour général, recherche, matchs, tournois), rejouables depuis la barre d'outils ou la commande ``tour``, et d'une base d'exemple chargeable par la commande ``demo`` pour découvrir l'outil sans importer ses propres parties. Personnalisation des couleurs du plateau (fond, bordure, flèches, pions, dés, videau) depuis la fenêtre de configuration. Autocomplétion de la ligne de commande (touche *TAB*). Panneau de recherche : contrôle explicite du type de décision (Pions / Videau), avec un sous-type Double / Pas de double ou Prise / Passe pour les décisions de cube, synchronisé avec le plateau ; le videau proposé est affiché au centre du plateau pour les décisions de prise/passe. Recherche d'une position par son identifiant (filtre ``id``). Correction de l'attribution de l'erreur de videau au joueur 1. Voir :ref:`visites_guidees`."

Sommaire
========

.. toctree::
   :maxdepth: 2
   :numbered:

   telecharge_install
   manuel
   guide_utilisateur
   cmd_mode
   raccourcis
   cli
   faq
   mode_headless
   annexe_filtres
   annexe_windows_securite
   annexe_mac_securite
   annexe_db_scheme
   stats_parity

.. youtube:: Ln7XKVFqfUk
   :width: 100%

.. youtube:: HkY4iXjxMeI
   :width: 100%

.. raw:: html

   <div style="margin-top: 20px;"></div>

.. _contacts:

Contacts
========

Auteur: Kévin Unger <blunderdb@proton.me>.
Vous pouvez aussi me trouver sur Heroes sous le pseudo postmanpat.

J'ai développé blunderDB initialement pour mon usage personnel afin de
pouvoir détecter des motifs dans mes erreurs. Mais il est très agréable
d'avoir un retour surtout quand on a dépensé un paquet d'heures de
conception, codage, débuggage... Aussi n'hésitez pas à m'écrire pour
faire part de votre retour d'expérience. Tous les retours (constructifs)
sont bienvenus.

Voici plusieurs manières de discuter:

* rejoindre le serveur Discord de blunderDB: https://discord.gg/DA5PpzM9En

* m'écrire un mail à blunderdb@proton.me,

* discuter avec moi, si on se retrouve dans un tournoi,

* sur Github,

  * ouvrir un ticket: https://github.com/kevung/blunderDB/issues

  * pour des corrections de bugs ou des propositions d'amélioration,
    créer une pull request.

Faire un don
============

Si vous appréciez blunderDB et que vous voulez soutenir les développements passés et futurs, vous pouvez

* me payer un verre si on a le plaisir de se rencontrer!

* faire un petit don par PayPal à l'adresse blunderdb@proton.me

Remerciements
=============

Je dédie ce petit logiciel à ma compagne Anne-Claire et notre tendre
fille Perrine. Je tiens à remercier tout particulièrement quelques amis:

* *Tristan Remille*, de m'avoir initié au backgammon avec joie et
  bienveillance; de montrer la Voie dans la compréhension de ce
  merveilleux jeu; de continuer à m'encourager malgré mes piètres
  tentatives de mieux jouer.

* *Nicolas Harmand*, joyeux camarade depuis maintenant plus d'une dizaine
  d'années dans de chouettes aventures, et un fantastique partenaire de jeu
  depuis qu'il a choppé le virus du backgammon.
