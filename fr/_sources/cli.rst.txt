.. _cli:

Interface en ligne de commande (CLI)
====================================

Introduction
------------

blunderDB embarque une interface en ligne de commande (CLI) complète dans le
même exécutable que l'interface graphique. La CLI est particulièrement utile
pour:

* **l'import en masse** de matchs: importer un répertoire entier de fichiers
  de matchs (XG, SGF, MAT, BGF…) en une seule commande,

* **l'automatisation**: intégrer blunderDB dans des scripts shell pour des
  sauvegardes régulières, des exports planifiés ou des chaînes de traitement,

* **l'utilisation sur serveur**: manipuler des bases de données sur des machines
  sans environnement graphique,

* **l'inspection rapide**: vérifier le contenu ou l'intégrité d'une base de
  données sans lancer l'interface graphique.

La CLI partage exactement le même format de base de données que l'interface
graphique. Toute opération effectuée en CLI est immédiatement visible dans
l'interface graphique et inversement.

Syntaxe générale
----------------

Le mode est détecté automatiquement: si le premier argument est une commande
CLI, blunderDB se lance en mode headless, sinon il lance l'interface graphique.

.. code-block:: bash

   # Mode graphique (aucun argument)
   ./blunderdb

   # Mode CLI
   ./blunderdb <commande> [options]

Commandes disponibles
---------------------

.. csv-table::
   :header: "Commande", "Description"
   :widths: 10, 40
   :align: center

   "create", "Crée une nouvelle base de données."
   "import", "Importe des données (match, position, lot)."
   "export", "Exporte des données."
   "search", "Recherche des positions avec filtres."
   "list", "Affiche le contenu de la base."
   "match", "Affiche les positions et analyses d'un match."
   "info", "Affiche les métadonnées de la base."
   "edit", "Modifie les métadonnées de la base."
   "verify", "Vérifie l'intégrité de la base."
   "delete", "Supprime des données."
   "help", "Affiche l'aide."
   "version", "Affiche la version."

Chaque commande accepte l'option ``--help`` pour afficher son aide détaillée.

create — Créer une base de données
-----------------------------------

Crée un nouveau fichier de base de données avec des métadonnées optionnelles.

.. code-block:: bash

   ./blunderdb create --db <chemin> [--user <nom>] [--description <texte>] [--force]

**Options:**

* ``--db`` — Chemin du fichier de base de données à créer (obligatoire).
* ``--user`` — Nom du propriétaire de la base.
* ``--description`` — Description de la base.
* ``--force`` — Écraser le fichier s'il existe déjà.

L'extension ``.db`` est ajoutée automatiquement si elle est absente. Les
répertoires parents sont créés si nécessaire.

**Exemple:**

.. code-block:: bash

   ./blunderdb create --db mes_matchs.db --user "Jean" --description "Matchs de tournoi 2025"

import — Importer des données
------------------------------

Importe des fichiers de matchs ou de positions dans la base de données.

.. code-block:: bash

   ./blunderdb import --db <chemin> --type <type> [options]

**Options:**

* ``--db`` — Chemin de la base de données (obligatoire).
* ``--type`` — Type d'import: ``match``, ``position`` ou ``batch`` (obligatoire).
* ``--file`` — Fichier à importer (pour ``match`` et ``position``).
* ``--dir`` — Répertoire à importer (pour ``batch``).
* ``--recursive`` — Scanner récursivement les sous-répertoires (défaut: oui).

Import d'un match
^^^^^^^^^^^^^^^^^

Formats supportés: eXtreme Gammon (``.xg``, ``.xgp``), GNUbg (``.sgf``),
Jellyfish (``.mat``, ``.txt``) et BGBlitz (``.bgf``).

.. code-block:: bash

   ./blunderdb import --db base.db --type match --file match.xg

Import de positions
^^^^^^^^^^^^^^^^^^^

Importe des positions depuis un fichier texte (une position JSON par ligne):

.. code-block:: bash

   ./blunderdb import --db base.db --type position --file positions.txt

Import par lot
^^^^^^^^^^^^^^

Importe tous les fichiers de matchs d'un répertoire en une seule opération.
C'est la méthode la plus efficace pour importer un grand nombre de matchs.

.. code-block:: bash

   # Import récursif (par défaut)
   ./blunderdb import --db base.db --type batch --dir ./matchs/

   # Import non récursif
   ./blunderdb import --db base.db --type batch --dir ./matchs/ --recursive=false

Un tableau récapitulatif indique pour chaque fichier si l'import a réussi
(✓), échoué (✗) ou s'il s'agit d'un doublon (⊘).

export — Exporter des données
------------------------------

Exporte le contenu de la base vers des fichiers.

.. code-block:: bash

   ./blunderdb export --db <chemin> --type <type> --file <sortie> [options]

**Options:**

* ``--db`` — Base source (obligatoire).
* ``--type`` — Type d'export: ``database``, ``positions`` ou ``matches`` (obligatoire).
* ``--file`` — Fichier de sortie (obligatoire).
* ``--analysis`` — Inclure les analyses (défaut: oui).
* ``--comments`` — Inclure les commentaires (défaut: oui).
* ``--filters`` — Inclure la bibliothèque de filtres (défaut: oui).
* ``--played-moves`` — Inclure les coups joués (défaut: oui).
* ``--matches`` — Inclure les matchs (défaut: oui).
* ``--collections`` — Inclure les collections (défaut: non).
* ``--collection-ids`` — IDs de collections à exporter (séparés par des virgules).
* ``--match-ids`` — IDs de matchs à exporter (séparés par des virgules, vide = tous).
* ``--tournament-ids`` — IDs de tournois à exporter (séparés par des virgules).

**Exemples:**

.. code-block:: bash

   # Export complet de la base
   ./blunderdb export --db base.db --type database --file sauvegarde.db

   # Export des positions en JSON
   ./blunderdb export --db base.db --type positions --file positions.txt

   # Export de matchs spécifiques
   ./blunderdb export --db base.db --type matches --file selection.db --match-ids 1,3,5

search — Rechercher des positions
----------------------------------

Recherche des positions dans la base selon des critères combinables.

.. code-block:: bash

   ./blunderdb search --db <chemin> [options]

**Options principales:**

* ``--db`` — Base de données (obligatoire).
* ``--format`` — Format de sortie: ``table``, ``json`` ou ``xgid`` (défaut: ``table``).
* ``--limit`` — Nombre maximum de résultats (0 = illimité).
* ``--export`` — Exporter les résultats vers une nouvelle base.

**Filtres disponibles:**

* ``--decision`` — Type de décision: ``checker`` ou ``cube``.
* ``--dice`` — Lancer de dés. ``5,3`` cherche les positions où les deux dés
  correspondent (peu importe l'ordre). ``5`` cherche les positions où un 5
  apparaît sur l'un des deux dés (la valeur du deuxième dé est ignorée).
  Implique ``--decision checker`` si aucune valeur de ``--decision`` n'est
  donnée.
* ``--pip-min`` / ``--pip-max`` — Intervalle de différence de pip count.
* ``--winrate-min`` / ``--winrate-max`` — Intervalle de taux de victoire (%).
* ``--cube`` — Valeur du videau.
* ``--score1`` / ``--score2`` — Scores des joueurs.
* ``--match-length`` — Longueur du match.
* ``--error-min`` — Erreur d'équité minimale.
* ``--move-error-min`` / ``--move-error-max`` — Erreur du coup joué (millipoints).
* ``--has-analysis`` — Uniquement les positions avec analyse.
* ``--off1-min`` / ``--off2-min`` — Pions sortis minimum (joueur 1/2).
* ``--match-ids`` — Filtrer par IDs de matchs (séparés par des virgules).
* ``--tournament-ids`` — Filtrer par IDs de tournois (séparés par des virgules).
* ``--individual`` — Uniquement les positions importées seules, c'est-à-dire
  celles que vous avez ajoutées vous-même et non celles qu'un import de match
  a apportées.

**Exemples:**

.. code-block:: bash

   # Rechercher les décisions de videau
   ./blunderdb search --db base.db --decision cube

   # Retrouver les positions que vous avez ajoutées vous-même
   ./blunderdb search --db base.db --individual

   # Rechercher les positions avec erreur >= 0.1
   ./blunderdb search --db base.db --error-min 0.1

   # Rechercher dans un tournoi et exporter
   ./blunderdb search --db base.db --tournament-ids 1 --export cubes.db

   # Rechercher les positions avec un lancer de dés 6-5 (peu importe l'ordre)
   ./blunderdb search --db base.db --dice 6,5

   # Rechercher les positions où un 6 a été obtenu sur l'un des deux dés
   ./blunderdb search --db base.db --dice 6

   # Sortie JSON limitée à 10 résultats
   ./blunderdb search --db base.db --format json --limit 10

list — Lister le contenu
--------------------------

Affiche le contenu de la base de données.

.. code-block:: bash

   ./blunderdb list --db <chemin> --type <type> [--limit <n>]

**Types:**

* ``matches`` — Liste des matchs importés.
* ``tournaments`` — Liste des tournois.
* ``positions`` — Liste des positions (limité à 10 par défaut).
* ``stats`` — Statistiques globales (positions, analyses, matchs, parties, coups).

**Exemples:**

.. code-block:: bash

   # Statistiques de la base
   ./blunderdb list --db base.db --type stats

   # Liste des matchs
   ./blunderdb list --db base.db --type matches

   # Premières 20 positions
   ./blunderdb list --db base.db --type positions --limit 20

match — Afficher un match
--------------------------

Affiche les positions et analyses d'un match importé.

.. code-block:: bash

   ./blunderdb match --db <chemin> --id <id_match> [--format <format>] [--output <fichier>]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--id`` — ID du match à afficher (obligatoire).
* ``--format`` — Format de sortie: ``json``, ``text`` ou ``summary`` (défaut: ``json``).
* ``--output`` — Fichier de sortie (défaut: sortie standard).

**Exemples:**

.. code-block:: bash

   # Résumé d'un match
   ./blunderdb match --db base.db --id 1 --format summary

   # Détails de chaque position
   ./blunderdb match --db base.db --id 1 --format text

   # Export JSON vers un fichier
   ./blunderdb match --db base.db --id 1 --output match1.json

info — Métadonnées de la base
------------------------------

Affiche les métadonnées et les statistiques d'une base de données.

.. code-block:: bash

   ./blunderdb info --db <chemin> [--format <format>]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--format`` — Format de sortie: ``text`` ou ``json`` (défaut: ``text``).

**Exemples:**

.. code-block:: bash

   # Afficher les informations
   ./blunderdb info --db base.db

   # Sortie JSON (pour un script)
   ./blunderdb info --db base.db --format json

edit — Modifier les métadonnées
--------------------------------

Modifie le nom d'utilisateur ou la description d'une base de données.

.. code-block:: bash

   ./blunderdb edit --db <chemin> [options]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--user`` — Nouveau nom d'utilisateur.
* ``--description`` — Nouvelle description.
* ``--clear-user`` — Effacer le nom d'utilisateur.
* ``--clear-description`` — Effacer la description.

Au moins une option de modification est requise.

**Exemples:**

.. code-block:: bash

   # Modifier l'utilisateur et la description
   ./blunderdb edit --db base.db --user "Marie" --description "Ma collection"

   # Effacer la description
   ./blunderdb edit --db base.db --clear-description

verify — Vérifier l'intégrité
-------------------------------

Vérifie l'intégrité de la base de données et, optionnellement, compare un match
avec son fichier source.

.. code-block:: bash

   ./blunderdb verify --db <chemin> [--match <id>] [--mat <fichier.mat>]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--match`` — ID du match à vérifier.
* ``--mat`` — Fichier MAT à comparer (utilisé avec ``--match``).

Sans l'option ``--match``, la commande affiche les statistiques générales de la
base. Avec ``--match``, elle vérifie les données du match et peut les comparer
avec le fichier source original.

**Exemples:**

.. code-block:: bash

   # Vérification globale
   ./blunderdb verify --db base.db

   # Vérifier un match spécifique
   ./blunderdb verify --db base.db --match 1

   # Comparer avec le fichier source
   ./blunderdb verify --db base.db --match 1 --mat original.mat

delete — Supprimer des données
-------------------------------

Supprime un match et toutes les données associées (parties, coups, analyses).

.. code-block:: bash

   ./blunderdb delete --db <chemin> --type match --id <id> [--confirm]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--type`` — Type de suppression: ``match`` (obligatoire).
* ``--id`` — ID de l'élément à supprimer (obligatoire).
* ``--confirm`` — Supprimer sans demander de confirmation.

**Exemples:**

.. code-block:: bash

   # Supprimer avec confirmation interactive
   ./blunderdb delete --db base.db --type match --id 1

   # Supprimer sans confirmation (pour scripts)
   ./blunderdb delete --db base.db --type match --id 1 --confirm

Exemples de flux de travail
-----------------------------

Import d'un répertoire de tournoi
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

.. code-block:: bash

   # Créer une base dédiée au tournoi
   ./blunderdb create --db tournoi_paris.db --user "Jean" --description "Open de Paris 2025"

   # Importer tous les matchs du répertoire
   ./blunderdb import --db tournoi_paris.db --type batch --dir ./matchs_open_paris/

   # Vérifier le résultat
   ./blunderdb list --db tournoi_paris.db --type stats

Sauvegarde régulière
^^^^^^^^^^^^^^^^^^^^

.. code-block:: bash

   # Export complet pour sauvegarde
   ./blunderdb export --db production.db --type database --file sauvegarde-$(date +%Y%m%d).db

Analyse des erreurs
^^^^^^^^^^^^^^^^^^^

.. code-block:: bash

   # Extraire les blunders dans une base séparée
   ./blunderdb search --db production.db --error-min 0.1 --export blunders.db

   # Extraire les erreurs de videau
   ./blunderdb search --db production.db --decision cube --error-min 0.05 --export cube_errors.db

Codes de retour
---------------

* ``0`` — Succès.
* ``1`` — Erreur.

Cela permet d'utiliser la CLI dans des scripts avec gestion d'erreurs:

.. code-block:: bash

   if ./blunderdb import --db base.db --type match --file match.xg; then
       echo "Import réussi"
   else
       echo "Échec de l'import"
       exit 1
   fi
