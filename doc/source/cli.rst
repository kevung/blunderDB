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
graphique : les deux écrivent le même fichier, il n'y a rien à synchroniser.

.. note::
   **Si l'application est ouverte pendant qu'un script écrit.** Le fichier
   est en mode WAL : une lecture ne bloque jamais une écriture, et les deux
   programmes travaillent sur la même base sans se gêner. Deux écritures, en
   revanche, se succèdent — la seconde attend le verrou d'écriture (dix
   secondes par instruction, plus quelques nouvelles tentatives) et n'échoue
   que si l'attente est épuisée, sur un message nommant SQLite :

   .. code-block:: text

      Error: failed to import match: sqlite: save match: database is locked (5) (SQLITE_BUSY)

   L'interface graphique ne surveille pas le fichier : elle continue
   d'afficher ce qu'elle avait chargé jusqu'à ce que ``CTRL-R`` recharge les
   positions. Rien n'est perdu, mais l'écran est en retard sur la base.

Syntaxe générale
----------------

Le mode est détecté automatiquement: si le premier argument est une commande
CLI, blunderDB se lance en mode headless, sinon il lance l'interface graphique.

.. code-block:: bash

   # GUI
   ./blunderdb

   # CLI
   ./blunderdb <command> [options]

Les exemples de cette page écrivent ``./blunderdb`` : le binaire tel qu'il est
téléchargé, appelé depuis le dossier où il se trouve. Installé par un paquet,
ou lié depuis un dossier du ``PATH`` (voir :ref:`telecharge_install`), il
s'appelle simplement ``blunderdb``.

Les options booléennes annoncées « défaut: oui » se désactivent par la forme
``--option=false`` — ``--recursive=false``, ``--analysis=false``. La forme
séparée par une espace n'existe pas : ``--recursive false`` laisse l'option à
sa valeur par défaut et traite ``false`` comme un argument de trop.

Commandes disponibles
---------------------

.. csv-table::
   :header: "Commande", "Description"
   :widths: 10, 40
   :align: center

   "create", "Crée une nouvelle base de données."
   "import", "Importe des données (match, position, lot)."
   "export", "Exporte des données."
   "transfer", "Copie une partie d'une bibliothèque dans une autre, sans fichier intermédiaire."
   "identity", "Affiche ou déplace l'identité d'émetteur (clé de signature des filigranes)."
   "open", "Transforme un fichier protégé par mot de passe (.dbx) en base ordinaire."
   "search", "Recherche des positions avec filtres."
   "list", "Affiche le contenu de la base."
   "match", "Affiche les positions et analyses d'un match."
   "transcribe", "Rejoue une transcription et rend compte de ce que le rejeu trouve."
   "duel", "Joue un Duel une Action par appel (création, coup, videau, abandon, arrêt)."
   "collection", "Gère les collections (liste, contenu, création, renommage, suppression, export)."
   "study", "La file d'étude : les blunders non traités du joueur de référence, et la marque « vu »."
   "lesson", "Gère les leçons (étapes ordonnées montrant collections et positions) et les exporte."
   "anki", "Paquets de répétition espacée (liste, statistiques, prévision, synchronisation)."
   "stats", "Statistiques calculées à part de ``list --type stats`` (erreurs récurrentes, classement, rapport…)."
   "training", "Le journal d'entraînement (séances, positions manquées)."
   "comment", "Lit et ajoute les commentaires d'une position, attribués à leur auteur."
   "elsewhere", "Ce que d'autres bases savent d'une position, et ses voisines chez elles."
   "cubematrix", "Verdict de videau d'une position à chaque score d'un match."
   "rollout", "Joue une position jusqu'au bout pour départager ses coups ou sa décision de videau (XGID ou OGID)."
   "epc", "Calcule l'Effective Pip Count et le verdict de videau d'une position de sortie (XGID ou OGID)."
   "capture", "Lit la position que montre une capture d'écran d'un plateau eXtreme Gammon."
   "openingbook", "Le coup d'ouverture ou la réponse d'une partie à un score, tiré du livre d'ouverture embarqué et jugé contre lui."
   "bearoff", "Fabrique, liste, vérifie et supprime les bases de sortie."
   "analyze", "Écrit une analyse gammonNet pour chaque position qui n'en a aucune."
   "tournament", "Lit un tournoi dirigé (classement, tableaux, pages) et confirme ses propositions."
   "players", "Les graphies d'un joueur (alias), la fusion et l'inversion des joueurs."
   "events", "Les graphies d'un événement (alias)."
   "trash", "La corbeille : liste, restauration, suppression annulable."
   "info", "Affiche les métadonnées de la base."
   "edit", "Modifie les métadonnées et les seuils de la base."
   "verify", "Vérifie l'intégrité de la base."
   "vacuum", "Compacte le fichier de base de données, récupère l'espace libéré."
   "met", "Liste, importe et choisit la table d'équité de match de la base."
   "reencode", "Réécrit les analyses au format compact."
   "repair", "Recalcule ce que la base dérive de ce qu'elle stocke."
   "maintenance", "L'état de santé de la base : compteurs, actions et dernières exécutions."
   "delete", "Supprime des données."
   "healthcheck", "Interroge un démon ``serve`` en marche : code 0 s'il est disponible."
   "mcp", "Offre les outils de la base à un assistant IA (Model Context Protocol)."
   "completion", "Affiche un script de complétion shell (bash, zsh, fish)."
   "help", "Affiche l'aide."
   "version", "Affiche la version."
   "serve, migrate, call", "Mode serveur et migration vers PostgreSQL : voir :ref:`headless`."

Chaque commande accepte l'option ``--help`` pour afficher son aide détaillée.

.. toctree::
   :maxdepth: 2

   cli_import_export
   cli_bibliotheque
   cli_matchs
   cli_analyse
   cli_entrainement
   cli_stats
   cli_admin

Exemples de flux de travail
-----------------------------

Import d'un répertoire de tournoi
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

.. code-block:: bash

   ./blunderdb create --db tournoi_paris.db --user "Jean" --description "Open de Paris 2025"
   ./blunderdb import --db tournoi_paris.db --type batch --dir ./matchs_open_paris/
   ./blunderdb list --db tournoi_paris.db --type stats

Sauvegarde régulière
^^^^^^^^^^^^^^^^^^^^

.. code-block:: bash

   ./blunderdb export --db production.db --type database --file sauvegarde-$(date +%Y%m%d).db

Analyse des erreurs
^^^^^^^^^^^^^^^^^^^

.. code-block:: bash

   # Les positions délicates, puis celles de videau
   ./blunderdb search --db production.db --error-min 0.1 --export blunders.db
   ./blunderdb search --db production.db --decision cube --error-min 0.05 --export cube_errors.db

   # Les coups réellement fautifs : au moins 100 millièmes d'équité perdus
   ./blunderdb search --db production.db --move-error-min 100 --format json

Codes de retour
---------------

* ``0`` — Succès.
* ``1`` — Erreur.

Cela permet d'utiliser la CLI dans des scripts avec gestion d'erreurs:

.. code-block:: bash

   if ./blunderdb import --db base.db --type match --file match.xg; then
       echo "OK"
   else
       echo "KO"
       exit 1
   fi
