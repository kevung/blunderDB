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
   "identity", "Affiche ou déplace l'identité d'émetteur (clé de signature des filigranes)."
   "open", "Transforme un fichier protégé par mot de passe (.dbx) en base ordinaire."
   "search", "Recherche des positions avec filtres."
   "list", "Affiche le contenu de la base."
   "match", "Affiche les positions et analyses d'un match."
   "collection", "Gère les collections (liste, contenu, création, renommage, suppression, export)."
   "anki", "Paquets de répétition espacée (liste, statistiques, prévision, synchronisation)."
   "epc", "Calcule l'Effective Pip Count et le verdict de videau d'une position de sortie (XGID)."
   "bearoff", "Fabrique, liste, vérifie et supprime les bases de sortie."
   "analyze", "Écrit une analyse gammonNet pour chaque position qui n'en a aucune."
   "info", "Affiche les métadonnées de la base."
   "edit", "Modifie les métadonnées de la base."
   "verify", "Vérifie l'intégrité de la base."
   "vacuum", "Compacte le fichier de base de données, récupère l'espace libéré."
   "repair", "Recalcule les colonnes scalaires tirées de chaque analyse."
   "delete", "Supprime des données."
   "healthcheck", "Interroge un démon ``serve`` en marche : code 0 s'il est disponible."
   "completion", "Affiche un script de complétion shell (bash, zsh, fish)."
   "help", "Affiche l'aide."
   "version", "Affiche la version."
   "serve, migrate, call", "Mode serveur et migration vers PostgreSQL : voir :ref:`headless`."

Chaque commande accepte l'option ``--help`` pour afficher son aide détaillée.

create — Créer une base de données
-----------------------------------

Crée un nouveau fichier de base de données avec des métadonnées optionnelles.

.. code-block:: bash

   ./blunderdb create --db <path> [--user <name>] [--description <text>] [--force]

**Options:**

* ``--db`` — Chemin du fichier de base de données à créer (obligatoire).
* ``--user`` — Nom du propriétaire de la base.
* ``--description`` — Description de la base.
* ``--force`` — Écraser le fichier s'il existe déjà.
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json`` (chemin,
  version, utilisateur, description, date de création).

L'extension ``.db`` est ajoutée automatiquement si elle est absente. Les
répertoires parents sont créés si nécessaire.

**Exemple:**

.. code-block:: bash

   ./blunderdb create --db mes_matchs.db --user "Jean" --description "Matchs de tournoi 2025"

import — Importer des données
------------------------------

Importe des fichiers de matchs ou de positions dans la base de données.

.. code-block:: bash

   ./blunderdb import --db <path> --type <type> [options]

**Options:**

* ``--db`` — Chemin de la base de données (obligatoire).
* ``--type`` — Type d'import: ``match``, ``position`` ou ``batch`` (obligatoire).
* ``--file`` — Fichier à importer (pour ``match`` et ``position``).
* ``--dir`` — Répertoire à importer (pour ``batch``).
* ``--recursive`` — Scanner récursivement les sous-répertoires (défaut: oui).
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``.
* ``--fail-on-error`` — Échoue si au moins un élément (``position`` ou
  ``batch``) n'a pas pu être importé, même quand d'autres ont réussi.

Le code de retour obéit à quatre règles :

* **rien n'a été reconnu** — chaque fichier a échoué — : erreur, que
  ``--fail-on-error`` soit passé ou non ;
* **que des doublons** — chaque fichier était déjà en base — : succès. Un
  répertoire relancé sans nouveau fichier, la nuit ordinaire d'un script,
  sort en 0 avec ``duplicates`` seul non nul ;
* **échec partiel** (certains éléments importés, d'autres refusés) : erreur
  seulement si ``--fail-on-error`` est passé ;
* au moins un élément nouveau importé, sans ``--fail-on-error`` : succès,
  les fichiers refusés étant listés dans le tableau.

Import d'un match
^^^^^^^^^^^^^^^^^

Formats supportés: eXtreme Gammon (``.xg``, ``.xgp``), GNUbg (``.sgf``),
Jellyfish (``.mat``, ``.txt``) et BGBlitz (``.bgf``).

.. code-block:: bash

   ./blunderdb import --db base.db --type match --file match.xg

   # Successfully imported match (ID: 1)
   #
   # Match Details:
   #   Players: Kévin Unger vs Maxence Job
   #   Event: HSBT Paris 2023
   #   Match Length: 7
   #   Games: 7

``--format json`` livre les mêmes champs en un seul document :

.. code-block:: json

   {
     "type": "match",
     "match_id": 1,
     "player1": "Kévin Unger",
     "player2": "Maxence Job",
     "event": "HSBT Paris 2023",
     "location": "Paris, Fédération Française de Bridge",
     "match_length": 7,
     "games": 7
   }

Import de positions
^^^^^^^^^^^^^^^^^^^

Importe des positions depuis un fichier texte, **une position JSON par
ligne**. C'est exactement ce qu'écrit ``export --type positions`` : les deux
commandes se répondent, un export se réimporte tel quel, sans rien retoucher.

.. code-block:: bash

   ./blunderdb import --db base.db --type position --file positions.txt

   # Successfully imported 4 positions

Une ligne, telle qu'``export`` la produit — le damier en occupe l'essentiel,
vingt-six points suivis des sorties :

.. code-block:: text

   {"id":1,"board":{"points":[{"checkers":0,"color":0},{"checkers":1,"color":1},…],"bearoff":[0,0]},"cube":{"owner":-1,"value":0},"dice":[0,0],"score":[7,7],"player_on_roll":0,"decision_type":1,"has_jacoby":0,"has_beaver":0,"individually_imported":true,"flagged":false}

L'analyse et les commentaires ne voyagent pas par ce format : il porte la
position, rien d'autre. Pour déplacer une bibliothèque entière, c'est
``export --type database`` qu'il faut.

Import par lot
^^^^^^^^^^^^^^

Importe tous les fichiers de matchs d'un répertoire en une seule opération.
C'est la méthode la plus efficace pour importer un grand nombre de matchs.

.. code-block:: bash

   ./blunderdb import --db base.db --type batch --dir ./matchs/
   ./blunderdb import --db base.db --type batch --dir ./matchs/ --recursive=false
   ./blunderdb import --db base.db --type batch --dir ./matchs/ --format json --fail-on-error

Un tableau récapitulatif indique pour chaque fichier si l'import a réussi
(✓), échoué (✗) ou s'il s'agit d'un doublon (⊘). Un doublon n'est pas
compté comme un échec, et un lot qui n'en contient que des doublons est un
succès (voir les règles ci-dessus).

.. code-block:: text

   Batch importing from: ./matchs/ (recursive: true)

   Found 3 match file(s) to import

   [1/3] Importing: 02_NDT_FR.txt... ERROR: failed to parse file: ingest: parse gnubg file: invalid MAT file: no match header found
   [2/3] Importing: test.mat... DUPLICATE
   [3/3] Importing: test.xg... OK (ID: 1, 341 positions)

   ====================================================================
   IMPORT SUMMARY
   ====================================================================
   Status  File           ID  Player 1     Player 2     Games  Positions  Error
   ------  ----           --  --------     --------     -----  ---------  -----
   ✗       02_NDT_FR.txt                                0      0          failed to parse file: ingest: ...
   ⊘       test.mat                                     0      0
   ✓       test.xg        1   Kévin Unger  Maxence Job  7      341
   --------------------------------------------------------------------
   Total: 3 files | Success: 1 | Duplicates: 1 | Failed: 1 | Positions imported: 341

``--format json`` en donne la même chose, exploitable par un script : un
objet par fichier dans ``files``, puis les totaux. Une nuit tranquille laisse
``duplicates`` seul non nul et ``failed`` à zéro, et le code de retour à 0 ;
seul un lot où rien n'a été reconnu sort en erreur.

.. code-block:: json

   {
     "files": [
       {"file_path": "02_NDT_FR.txt", "success": false, "error": "failed to parse file: …"},
       {"file_path": "test.xg", "success": true, "positions": 341}
     ],
     "total": 3,
     "success": 1,
     "duplicates": 1,
     "failed": 1,
     "positions_imported": 341
   }

export — Exporter des données
------------------------------

Exporte le contenu de la base vers des fichiers.

.. code-block:: bash

   ./blunderdb export --db <path> --type <type> --file <output> [options]

**Options:**

* ``--db`` — Base source (obligatoire).
* ``--type`` — Type d'export: ``database``, ``positions``, ``matches`` ou
  ``mat`` (export d'un ou plusieurs matchs en transcription Jellyfish
  ``.mat``) (obligatoire).
* ``--file`` — Fichier de sortie (obligatoire, sauf pour ``--type mat``
  utilisé avec ``--dir``).
* ``--dir`` — Répertoire de sortie pour l'export ``.mat`` par lot (plusieurs
  matchs, un fichier par match ; sans ``--match-ids``, tous les matchs sont
  exportés).
* ``--analysis`` — Inclure les analyses (défaut: oui).
* ``--comments`` — Inclure les commentaires (défaut: oui).
* ``--filters`` — Inclure la bibliothèque de filtres (défaut: oui).
* ``--played-moves`` — Inclure les coups joués (défaut: oui).
* ``--matches`` — Inclure les matchs (défaut: oui).
* ``--collections`` — Inclure les collections (défaut: non).
* ``--collection-ids`` — IDs de collections à exporter (séparés par des virgules).
* ``--match-ids`` — IDs de matchs à exporter (séparés par des virgules, vide = tous).
* ``--tournament-ids`` — IDs de tournois à exporter (séparés par des virgules).
* ``--password`` — Enveloppe le résultat dans un conteneur chiffré (``.dbx``).
* ``--watermark`` — Écrit une déclaration d'origine **signée** dans le fichier
  exporté (voir :ref:`diffusion_controlee`).
* ``--watermark-note`` — Texte libre associé au filigrane (conditions
  d'usage, contact) ; utilisé avec ``--watermark``.
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json`` (document
  résumant l'export: chemin, taille en octets, nombres).

**Exemples:**

.. code-block:: bash

   ./blunderdb export --db base.db --type database --file sauvegarde.db
   ./blunderdb export --db base.db --type positions --file positions.txt
   ./blunderdb export --db base.db --type matches --file selection.db --match-ids 1,3,5

   # .mat : un match, puis plusieurs (ou tous) dans un répertoire
   ./blunderdb export --db base.db --type mat --match-ids 5 --file match5.mat
   ./blunderdb export --db base.db --type mat --match-ids 5,9,12 --dir sorties/
   ./blunderdb export --db base.db --type mat --dir sorties/

   # .dbx : filigrané et protégé par mot de passe
   ./blunderdb export --db cours.db --type database --file cours-diffusion.dbx \
       --watermark "Cours de Jean Dupont — 12 mars 2026" \
       --watermark-note "Merci de ne pas rediffuser." \
       --password secret

Un filigrane est signé avec l'identité d'émetteur locale (voir la commande
``identity`` ci-dessous) : il est infalsifiable, mais pas inamovible — le
fichier reste une base SQLite ordinaire. Il ne protège rien, il indique
seulement d'où vient le fichier. Un mot de passe protège le *transport* du
fichier (la copie égarée, la pièce jointe envoyée par erreur), pas la base
elle-même : quiconque a reçu le mot de passe peut l'ouvrir. blunderDB
n'enregistre jamais rien côté destinataire (aucun registre, aucun journal) —
voir `ADR-0007 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0007-watermarks-mark-origin-and-nothing-else.md>`__.

identity — Identité d'émetteur
-------------------------------

Affiche ou déplace votre **identité d'émetteur** : la clé Ed25519 qui signe
chaque filigrane. Elle est créée d'elle-même au premier filigrane apposé ; il
n'y a rien à configurer. Elle appartient à une personne, pas à une base de
données : tout ce que vous marquez porte une seule empreinte publique.

.. code-block:: bash

   ./blunderdb identity
   ./blunderdb identity --name "Jean Dupont"
   ./blunderdb identity --export jean.bdbid --passphrase pw
   ./blunderdb identity --import jean.bdbid --passphrase pw

**Options:**

* ``--name`` — Change le nom affiché de l'identité.
* ``--export`` — Exporte l'identité vers un fichier ``.bdbid``.
* ``--import`` — Importe une identité depuis un fichier ``.bdbid``.
* ``--passphrase`` — Phrase de passe optionnelle protégeant le fichier
  exporté/importé (l'identité locale, elle, est volontairement non protégée).
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json`` (nom,
  empreinte, chemin de stockage).

Le fichier exporté permet à quiconque le détient de signer en votre nom — ne
le partagez pas. Renommer ne change qu'un libellé : les fichiers déjà marqués
conservent le nom sous lequel ils ont été scellés, et continuent de se
vérifier.

open — Ouvrir un fichier protégé
-----------------------------------

Transforme un fichier protégé par mot de passe (``.dbx``) en base ordinaire.
Le mot de passe est demandé une seule fois ; ensuite, c'est un fichier normal.

.. code-block:: bash

   ./blunderdb open --db cours.dbx --password secret
   ./blunderdb open --db cours.dbx --password secret --file ./mon-cours.db

**Options:**

* ``--db`` — Fichier ``.dbx`` à ouvrir (obligatoire).
* ``--password`` — Mot de passe du conteneur (obligatoire).
* ``--file`` — Chemin de sortie pour la base ordinaire (défaut: même nom,
  extension ``.db``).

Ce que le mot de passe protège : le *transport* du fichier — la copie égarée
dans un dossier de téléchargements, la pièce jointe envoyée par erreur. Pas la
base : quiconque a reçu le mot de passe peut l'ouvrir. L'en-tête du conteneur
est en clair, si bien que ``blunderdb info`` lit l'origine d'un fichier
protégé sans son mot de passe.

search — Rechercher des positions
----------------------------------

Recherche des positions dans la base selon des critères combinables.

.. code-block:: bash

   ./blunderdb search --db <path> [options]

**Options principales:**

* ``--db`` — Base de données (obligatoire).
* ``--format`` — Format de sortie: ``table``, ``json`` ou ``xgid`` (défaut: ``table``).
* ``--limit`` — Nombre maximum de résultats (0 = illimité).
* ``--offset`` — Ignorer les n premiers résultats avant de commencer à
  compter ; avec ``--limit``, c'est la pagination.
* ``--export`` — Exporter les résultats vers une nouvelle base.
* ``--query-help`` — Affiche la liste des jetons que ``--query`` comprend, et
  s'arrête là. Aucune base n'est ouverte : ``--db`` est inutile.

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
* ``--error-min`` — Seuil sur ce que **coûte une erreur dans la position** :
  l'écart entre le meilleur coup et le deuxième, ou la plus grande des trois
  erreurs de videau. En **points d'équité** — ``--error-min 0.1`` retient les
  positions où se tromper coûte au moins un dixième de point. Il ne dit rien
  de ce qui y a été joué.
* ``--move-error-min`` / ``--move-error-max`` — Seuil sur l'erreur du coup
  **effectivement joué** par le joueur 1. En **millièmes d'équité**
  (millipoints) : ``--move-error-min 50``, soit un vingtième de point. C'est
  le jeton ``E`` de la grammaire de recherche, écrit tel quel.
* ``--has-analysis`` — Uniquement les positions avec analyse.
* ``--off1-min`` / ``--off2-min`` — Pions sortis minimum (joueur 1/2).
* ``--match-ids`` — Filtrer par IDs de matchs (séparés par des virgules).
* ``--tournament-ids`` — Filtrer par IDs de tournois (séparés par des virgules).
* ``--position-ids`` — Filtrer par IDs de positions : intervalle ``2,7``
  (positions 2 à 7) ou liste explicite séparée par des points-virgules
  ``5;10;15``.
* ``--individual`` — Uniquement les positions importées seules, c'est-à-dire
  celles que vous avez ajoutées vous-même et non celles qu'un import de match
  a apportées.
* ``--flagged`` — Uniquement les positions marquées (*flag*) pour étude dans
  le logiciel d'origine (marques eXtreme Gammon). Non rétroactif : les
  matchs déjà importés doivent l'être à nouveau pour livrer leurs marques.
* ``--has-comment`` — Uniquement les positions portant un commentaire.
  L'origine n'est pas distinguée : une note tapée à la main et un commentaire
  apporté par l'import d'un match comptent tous les deux. Les commentaires de
  match ou de tournoi ne sont pas consultés.
* ``--no-comment`` — Uniquement les positions sans commentaire. Mutuellement
  exclusif avec ``--has-comment``.

.. warning::
   ``--error-min`` et ``--move-error-min`` ne mesurent pas la même chose et
   **ne prennent pas la même unité** : le facteur est mille. Le premier se
   donne en points d'équité (``0.1``), les deux autres en millièmes
   (``100``) — un point vaut 1000 millièmes. C'est ``--move-error-min`` qui
   répond à « où ai-je fauté » ; ``--error-min`` répond à « quelles positions
   étaient délicates ».

**Ce que search imprime:**

``--format table`` (le défaut) donne une ligne par position : l'identifiant,
le score, la valeur du videau, le type de décision, le lancer, la meilleure
décision et son équité. Les deux dernières colonnes restent vides pour une
position sans analyse.

.. code-block:: text

   Found 5 position(s)

   ID  Score  Cube  Type  Dice  Best Move  Equity
   --  -----  ----  ----  ----  ---------  ------
   2   7-7    0     cube        No Double  -0.005
   4   7-7    0     cube        No Double  -0.027
   6   7-7    0     cube        No Double  -0.161
   8   7-7    0     cube        No Double  0.256
   10  7-7    0     cube        No Double  -0.234

``--format json`` donne un tableau des mêmes positions. Les champs sont
``id``, ``score``, ``cube``, ``decision_type`` (``checker`` ou ``cube``) et
``dice``, toujours présents ; ``best_move``, ``equity`` et ``xgid``
n'apparaissent que si la position porte une analyse qui les renseigne. La
ligne ``Found n position(s)`` reste imprimée avant le tableau : un script qui
n'attend que du JSON doit sauter la première ligne, ou passer par
``--export``.

.. code-block:: json

   [
     {
       "id": 5266,
       "score": [
         5,
         4
       ],
       "cube": 1,
       "decision_type": "checker",
       "dice": [
         4,
         3
       ],
       "best_move": "10/3",
       "equity": 0.565
     }
   ]

``--format xgid`` imprime un XGID par ligne, et **rien d'autre**. Il
n'imprime que les positions dont l'analyse enregistrée porte un XGID : une
position collée dans l'application depuis un export texte, ou un fichier BGF
qui en transporte un. Les positions apportées par l'import d'un match XG,
GNUbg ou Jellyfish n'en portent pas, et la sortie est alors vide. La
sous-commande ``collection show``, elle, régénère le XGID depuis le damier.

**Le langage de requête:**

Les drapeaux ci-dessus couvrent une partie des filtres seulement. ``--query``
donne accès au langage de requête de l'application — celui de la barre de
commande — et donc à tous les filtres qui ne se dessinent pas sur le plateau :
motif de coup, texte de commentaire, joueur, date, équité, dés exclus, zones
et blots.

La grammaire n'est écrite qu'à un seul endroit, :ref:`cmd_filter`. Sa table
donne chaque jeton, sa forme, et le drapeau de ``search`` qui lui correspond
quand il en existe un. Cette page ne la répète pas.

.. code-block:: bash

   ./blunderdb search --db base.db --query 's p>30 E>50'
   ./blunderdb search --db base.db --query 's m"13/11" t"blunder" pl"Alice" T>2026/01/01'

``--query-help`` en rappelle la liste sans ouvrir de base :

.. code-block:: text

   $ ./blunderdb search --query-help
   blunderdb search --query — the interface's query language

   A query is the same text the application's command bar takes:
     s cube p>30 E>50        cube decisions, 30+ pips behind, 50+ millipoints of error
     s m"13/11" T>2026/01/01 played 13/11, imported this year

   Flags (no value):
     cube score   match the cube / the score of the position on the board
     d            match the decision type (checker or cube)
     …

   Ranges — each takes x>n, x<n or xa,b (lower-case: you; upper-case: the opponent):
     p P          pip count difference / absolute pip count
     …
     E            error of the played move, in millipoints
     T            creation date, T>2026/01/01

   Values:
     t"tag"       comment text (";" separates alternatives)
     …

``--query`` remplace les drapeaux de filtre au lieu de s'y ajouter : les
combiner est refusé, avec le nom du drapeau en cause. Les drapeaux qui disent
*où* chercher et *comment* afficher — ``--db``, ``--format``, ``--limit``,
``--offset``, ``--export`` — restent valides.

Un jeton que rien ne reconnaît fait échouer la commande plutôt que de réduire
la recherche en silence. Deux limites tiennent à l'absence de plateau en ligne
de commande : le motif de damier ne se tape pas, et les cinq jetons qui lisent
le plateau — ``cube``, ``score``, ``d``, ``D``/``D1`` et ``x`` — se comparent
ici à un plateau vide. Une recherche qui a besoin de l'un d'eux s'écrit
**entièrement** en drapeaux, puisque ``--query`` ne se combine pas avec eux —
par exemple, les décisions de videau en retard de 30 pips et fautives d'au
moins 50 millièmes :

.. code-block:: bash

   ./blunderdb search --db base.db --decision cube --pip-min 30 --move-error-min 50

**Exemples:**

.. code-block:: bash

   ./blunderdb search --db base.db --decision cube
   ./blunderdb search --db base.db --individual
   ./blunderdb search --db base.db --error-min 0.1
   ./blunderdb search --db base.db --tournament-ids 1 --export cubes.db

   # 6-5 dans les deux ordres, puis un 6 sur l'un des deux dés
   ./blunderdb search --db base.db --dice 6,5
   ./blunderdb search --db base.db --dice 6

   # Pagination
   ./blunderdb search --db base.db --format json --limit 10 --offset 20

list — Lister le contenu
--------------------------

Affiche le contenu de la base de données.

.. code-block:: bash

   ./blunderdb list --db <path> --type <type> [--limit <n>]

**Types:**

* ``matches`` — Liste des matchs importés.
* ``tournaments`` — Liste des tournois.
* ``positions`` — Liste des positions (limité à 10 par défaut).
* ``imports`` — Liste des imports enregistrés, du plus récent au plus ancien :
  identifiant, date, format, source, matchs importés / ignorés / enrichis,
  fichiers illisibles et positions nouvelles. Avec ``--batch <id>``, affiche le
  **compte rendu complet** d'un import : positions marquées, positions sans
  analyse, PR sur ce lot et cinq pires décisions (voir
  :ref:`compte_rendu_import`).
* ``stats`` — Rapport de statistiques de performance : PR / Snowie ER / MWC
  (global, pions, videau), PR glissant sur les N dernières décisions, top
  blunders, répartition par action de videau et histogramme des magnitudes
  d'erreur.
* ``players`` — Tableau comparatif, **une ligne par joueur** de la base :
  matchs, victoires/défaites, décisions comptées, PR global / pions / videau,
  Snowie ER, erreurs, blunders et chance. C'est le pendant en ligne de commande
  de l'onglet Joueurs du panneau Stats.

**Options** (type ``stats`` uniquement) :

* ``--metric`` — Métrique affichée: ``pr`` ou ``mwc`` (défaut: ``pr``).
* ``--player`` — Restreindre au joueur indiqué.
* ``--tournament`` — Restreindre à un ou plusieurs IDs de tournois (séparés
  par des virgules).
* ``--from`` — Date de début (AAAA-MM-JJ).
* ``--to`` — Date de fin (AAAA-MM-JJ).
* ``--decision-type`` — Type de décision: ``all``, ``checker`` ou ``cube``
  (défaut: ``all``).
* ``--top-blunders`` — Nombre de pires erreurs listées (défaut: 10).
* ``--format`` — Format de sortie: ``text`` ou ``json`` (défaut: ``text``).

**Options** (type ``imports`` uniquement) :

* ``--batch`` — Identifiant d'un lot : affiche son compte rendu complet au lieu
  de la liste.
* ``--format`` — Format de sortie: ``text`` ou ``json`` (défaut: ``text``).

La moitié mesurée du compte rendu est recalculée à chaque appel : un lot dont
les positions ont été analysées depuis rend les chiffres d'aujourd'hui, non ceux
du jour de l'import.

**Options** (type ``players`` uniquement) :

* ``--from`` / ``--to`` — Bornes de dates (AAAA-MM-JJ), par exemple les jours
  d'une compétition.
* ``--tournament`` — Restreindre à un ou plusieurs IDs de tournois.
* ``--format`` — Format de sortie: ``text``, ``json`` ou ``csv``
  (défaut: ``text``).

``--player`` et ``--decision-type`` ne s'appliquent **pas** à ce type : le
tableau porte sur tous les joueurs et ventile déjà pions et videau en colonnes
distinctes.

.. note::
   Un tiret « — » (champ vide en CSV) signale une valeur **jamais mesurée**, à
   ne pas confondre avec zéro. C'est le cas de la chance pour tout match importé
   avant la version 2.15.0 du schéma, ainsi que pour les formats qui ne la
   transportent pas (BGF, Jellyfish ``.mat``) : réimportez les fichiers source
   pour l'obtenir. La colonne ``luck_rolls`` indique sur combien de lancers
   porte la moyenne.

Chaque type imprime un bloc par élément, précédé du total trouvé. La ligne
finale rappelle que la liste est tronquée par ``--limit``, dont la valeur par
défaut vaut 10 pour les positions :

.. code-block:: text

   Found 3859 position(s):

   ID: 1
     Score: 7-7
     Player on roll: 0
     Decision: Checker play

   ID: 2
     Score: 7-7
     Player on roll: 0
     Decision: Cube action

   …

   (Showing 10 of 3859 positions, use --limit to see more)

**Exemples:**

.. code-block:: bash

   # Les imports enregistrés, puis le compte rendu de l'un d'eux
   ./blunderdb list --db base.db --type imports
   ./blunderdb list --db base.db --type imports --batch 3

   ./blunderdb list --db base.db --type stats
   ./blunderdb list --db base.db --type stats --metric mwc --player "Alice"
   ./blunderdb list --db base.db --type stats --decision-type checker --from 2026-01-01
   ./blunderdb list --db base.db --type stats --format json

   # Un tableau par joueur, borné aux dates d'une compétition
   ./blunderdb list --db base.db --type players --from 2026-03-01 --to 2026-03-08
   ./blunderdb list --db base.db --type players --format csv

   ./blunderdb list --db base.db --type matches
   ./blunderdb list --db base.db --type positions --limit 20

match — Afficher un match
--------------------------

Affiche les positions et analyses d'un match importé.

.. code-block:: bash

   ./blunderdb match --db <path> --id <id> [--format <format>] [--output <file>]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--id`` — ID du match à afficher (obligatoire).
* ``--format`` — Format de sortie: ``json``, ``text`` ou ``summary`` (défaut: ``json``).
* ``--output`` — Fichier de sortie (défaut: sortie standard).

**Exemples:**

.. code-block:: bash

   ./blunderdb match --db base.db --id 1 --format summary
   ./blunderdb match --db base.db --id 1 --format text
   ./blunderdb match --db base.db --id 1 --output match1.json

collection — Gérer les collections
----------------------------------

Gère les collections, ces ensembles de positions choisies à la main dans le
panneau Collections de l'interface graphique. Chaque sous-commande prend
``--db`` ; ``list`` et ``show`` acceptent ``--format text`` (défaut), ``json``
ou ``csv``, comme ``list``.

.. code-block:: bash

   ./blunderdb collection <subcommand> [options]

**Sous-commandes:**

* ``list`` — Liste des collections : id, nom, nombre de positions, description.
* ``show --id <id>`` — Positions d'une collection : id, index (le numéro
  1-based affiché dans la barre d'état de l'interface graphique), score, type
  de décision et XGID.
* ``create --name <nom> [--description <texte>]`` — Crée une collection vide.
* ``rename --id <id> --name <nom> [--description <texte>]`` — Renomme une
  collection (la description est conservée si elle n'est pas donnée).
* ``delete --id <id> [--confirm]`` — Supprime une collection ; ses positions
  restent dans la base.
* ``export --id <id[,id…]> --out <fichier.db> [--analysis=false]
  [--comments=false] [--watermark <texte>] [--watermark-note <texte>]`` —
  Exporte une ou plusieurs collections vers un nouveau fichier de base, par
  le même appel que la fenêtre d'export de l'interface graphique (voir la
  commande ``export`` pour le filigrane).

Le XGID affiché par ``show`` est celui enregistré avec l'analyse de la
position quand il existe (imports BGF et XGP) ; sinon il est généré depuis le
damier exactement comme le fait *Copier la position* dans l'interface
graphique — la longueur du match est alors le plus grand des deux scores
restants, une position enregistrée ne retenant pas la vraie.

**Exemples:**

.. code-block:: bash

   ./blunderdb collection list --db base.db

   # Found 2 collection(s):
   #
   # ID  Name              Positions  Description
   # --  ----              ---------  -----------
   # 1   Ouvertures blitz  0          À revoir
   # 2   Videaux ratés     0

Une base sans collection répond ``No collections found in database`` et sort
tout de même avec le code 0.

.. code-block:: bash

   ./blunderdb collection show --db base.db --id 3 --format csv

   ./blunderdb collection create --db base.db --name "Ouvertures blitz"
   ./blunderdb collection rename --db base.db --id 3 --name "Ouvertures"
   ./blunderdb collection delete --db base.db --id 3 --confirm

   # Exporter deux collections, marquées de leur origine
   ./blunderdb collection export --db base.db --id 3,4 --out ouvertures.db \
       --watermark "Cours de Jean Dupont - 12 mars 2026"

anki — Paquets de répétition espacée
-------------------------------------

Consulte et entretient les paquets de répétition espacée (FSRS) du panneau
Anki de l'interface graphique. La révision d'une carte demande le damier et
reste dans l'interface graphique ; la CLI liste, mesure et resynchronise.

.. code-block:: bash

   ./blunderdb anki <subcommand> [options]

**Sous-commandes:**

* ``decks [--format text|json|csv]`` — Liste des paquets : source, nombre de
  cartes, cartes dues, cartes nouvelles.
* ``stats --deck <id> [--format text|json]`` — Statistiques de révision d'un
  paquet : total, nouvelles, en apprentissage, à revoir, dues maintenant, et
  ses paramètres FSRS.
* ``forecast [--deck <id>] [--days <n>] [--format text|json|csv]`` — Cartes
  arrivant à échéance par jour civil (UTC) sur les ``n`` prochains jours
  (défaut 30, maximum 365) ; le jour 0 absorbe toutes les cartes en retard ;
  ``--deck 0`` (défaut) couvre tous les paquets.
* ``sync --deck <id>`` — Ajoute une carte pour chaque position de la source du
  paquet qui n'en a pas encore ; les cartes existantes gardent leur
  planification.
* ``retention --deck <id> [--format text|json]`` — Rétention mesurée d'un
  paquet, comparée à la cible que son propriétaire a choisie.
* ``card --id <id> --action suspend|unsuspend|bury|remove [--format text|json]``
  — Agit sur une carte. *Suspendre* la met de côté sans perdre son historique
  (elle ne ressort plus en séance) ; *enterrer* la masque jusqu'au lendemain,
  sans rien dire de sa valeur ; *retirer* la supprime du paquet — la position,
  elle, reste dans la base, un paquet n'étant qu'une liste d'étude posée
  dessus.
* ``log [--deck <id>] [--limit <n>] [--format text|json]`` — Journal des
  révisions, la plus récente d'abord (``--deck 0``, le défaut, couvre tous les
  paquets ; ``--limit`` vaut 20 par défaut). Le journal est ce que le
  planificateur a réellement reçu, par opposition à ce qu'il prévoit
  aujourd'hui : c'est le seul endroit où une note entrée par erreur se voit.

Un paquet fondé sur une collection relit sa collection. Un paquet fondé sur
une recherche conserve la recherche telle que l'interface graphique l'a
enregistrée (commande, damier et identifiants des positions trouvées à ce
moment-là) : la grammaire de recherche vit dans l'interface graphique, la CLI
resynchronise donc depuis les identifiants enregistrés et le signale sur la
sortie d'erreur — ouvrez le paquet dans l'interface graphique pour rejouer la
recherche elle-même.

**Exemples:**

.. code-block:: bash

   ./blunderdb anki decks --db base.db
   ./blunderdb anki stats --db base.db --deck 2 --format json
   ./blunderdb anki forecast --db base.db --deck 2 --days 14
   ./blunderdb anki sync --db base.db --deck 2
   ./blunderdb anki card --db base.db --id 12 --action suspend
   ./blunderdb anki log --db base.db --deck 2 --limit 50

   # Day         Due
   # ---         ---
   # 2026-09-02  12
   # 2026-09-03  4
   # ...
   #
   # 37 card(s) due over 14 day(s)

.. _cli_cubematrix:

cubematrix — Matrice du videau
-------------------------------

Donne le verdict de videau d'une position à **tous les scores** d'un match :
pour chaque case *away × away*, si la position se double et si elle se prend.
Calcul pur : aucune base de données n'est ouverte, la position arrive par son
XGID.

.. code-block:: bash

   ./blunderdb cubematrix [options] '<XGID>'

**Options:**

* ``--format`` — Format de sortie : ``text`` ou ``json`` (défaut : ``text``).
* ``--match-length`` — Longueur du match que la grille couvre, de 1 à 25
  (défaut : 7).
* ``--ply`` — Profondeur de recherche de chaque case, ``0`` ou ``2``
  (défaut : 2).
* ``--prune-k`` — Nombre de coups candidats retenus par le réseau d'élagage
  (défaut : 12).
* ``--jobs`` — Recherches menées en parallèle (défaut : une par cœur). La
  grille est identique quelle que soit la valeur ; seul le temps change.

Le score propre à la position est ignoré — la grille le remplace — mais son
**videau** est conservé : la question posée est « à quel score retournerais-je
*ce* videau ». La grille est post-Crawford d'un bout à l'autre.

Chaque case est une recherche à part entière, parce que le moteur tient compte
du score : une seule recherche relue à travers des équités de match
différentes serait fausse exactement là où le score compte.

**Exemples:**

.. code-block:: bash

   # Grille d'un match en 5 points
   ./blunderdb cubematrix --match-length 5 'XGID=-b----E-C---eE---c-e----B-:0:0:1:00:0:0:0:7:10'

   # Les équités de chaque case, pour un script
   ./blunderdb cubematrix --format json '<XGID>'

Sortie ``text`` : une grille dont les lignes sont les points restant à faire
au joueur au trait, les colonnes ceux de l'adversaire, puis la légende des
sigles ``ND`` / ``DT`` / ``DP`` / ``TG`` et la raison de chaque case refusée.

epc — Calculatrice EPC
------------------------

Calcule l'Effective Pip Count, la probabilité de gain et le verdict de videau
money d'une position de sortie donnée par XGID. Calcul pur : aucun fichier de
base de données n'est impliqué.

.. code-block:: bash

   ./blunderdb epc [options] '<XGID>'

**Options:**

* ``--format`` — Format de sortie: ``text`` ou ``json`` (défaut: ``text``).
* ``--bearoff-ts`` — Base bearoff two-sided optionnelle (``.bd``) élargissant
  la base intégrée TS-06-06 (également lue depuis la variable d'environnement
  ``BLUNDERDB_TS_PATH``). La base valide la plus large l'emporte ; un fichier
  invalide est ignoré avec un avertissement.

**Régimes.** Dans le domaine couvert par la base two-sided, la probabilité de
gain et l'analyse money du videau (cubeless, ND, D/T, D/P, verdict) sont
**exactes**. En dehors, la probabilité de gain est **estimée** (convolution
des distributions de lancers one-sided plus une correction calibrée) et
affichée avec sa marge d'erreur mesurée ; le verdict de videau n'est
volontairement jamais estimé (voir `ADR-0009 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0009-race-win-chances-are-read-or-convolved-cube-verdicts-are-never-estimated.md>`__).

**Exemples:**

.. code-block:: bash

   # Régime exact : six pions ou moins de chaque côté
   ./blunderdb epc 'XGID=-BBB------------------bbb-:0:0:1:00:0:0:0:0:10'

   # Avec la table TS-06-11 calculée : exact jusqu'à onze pions par joueur
   ./blunderdb epc --bearoff-ts ~/.local/share/blunderdb/gnubg_ts6x11.bd 'XGID=…'

bearoff — Bases de sortie
--------------------------

Fabrique et gère les bases de bearoff. Rien n'est téléchargé et rien n'est
embarqué : une table est calculée ici et vérifiée contre l'empreinte que gnubg
produit pour son domaine. Aucune sous-commande ne parle à une base de données —
une table de bearoff est de l'arithmétique sur le jeu, pas sur les positions de
quelqu'un — donc aucune ne prend ``--db``.

.. code-block:: bash

   ./blunderdb bearoff generate --ts <domain> [options]
   ./blunderdb bearoff list [options]
   ./blunderdb bearoff verify <file.bd> [options]
   ./blunderdb bearoff delete --ts <domain> [options]

Le domaine s'écrit comme sous ``makebearoff`` : ``6x9`` pour la table
two-sided à neuf pions par joueur, ``os8`` pour la table une face à huit
points (``os`` seul vaut ``os6``).

Les deux familles ne répondent pas à la même question. Une table **deux
faces** élargit le domaine où la probabilité de gain et le verdict de videau
sont exacts ; une table **une face** élargit la distance à laquelle un pion
peut se trouver sans que l'EPC se taise (jusqu'à dix points).

**generate.** Annonce la taille, la mémoire et le temps estimé avant de
commencer, puis affiche le pourcentage et le temps restant mesuré.

* ``--ts`` — Domaine deux faces à calculer, par exemple ``6x9``.
* ``--os`` — Domaine une face à calculer, en nombre de points : 6 à 12. Un et
  un seul des deux est requis.
* ``--cores`` — Cœurs à utiliser (défaut : tous sauf un).
* ``--data-dir`` — Où écrire (défaut : le dossier de données de l'application).
* ``--quiet`` — Pas de ligne de progression.

**CTRL-C met en pause.** Le signal est capté : l'état est écrit à côté de la
table et la même commande relancée reprend là où elle s'était arrêtée, au lieu
de tout recalculer. Une demi-heure d'arithmétique mérite d'être écrite.
``bearoff delete`` jette une reprise en attente. Seul le balayage deux faces
se met en pause ; le balayage une face est séquentiel et ``--cores`` ne lui
sert à rien.

**list.** Chiffre chaque domaine — taille, mémoire, temps sur cette machine —
et dit lesquels sont déjà présents, avec leur verdict, et lesquels ont un
calcul en pause. ``--format json`` pour un script, ``--cores`` pour changer
l'hypothèse de l'estimation.

**verify.** Répond ``verified`` (les mêmes octets que la référence),
``unverified`` (bien formée, mais aucune empreinte n'est enregistrée pour ce
domaine) ou ``corrupt`` (le fichier se contredit). Sort en erreur sur le
dernier cas : cette commande est faite pour être mise dans un script.

**delete.** Retire la table, la reprise en attente et les débris d'un calcul
mort. Un domaine par défaut est recalculé au prochain lancement de
l'application ; un domaine plus large ne l'est pas.

**Exemples:**

.. code-block:: bash

   # Ce que cette machine a, et ce que chaque domaine coûterait
   ./blunderdb bearoff list

   ./blunderdb bearoff generate --ts 6x9 --cores 4

   # OS-08 : l'EPC répond alors jusqu'à un pion sur la 8
   ./blunderdb bearoff generate --os 8

   # Sur un serveur, dans le volume que lit le démon
   ./blunderdb bearoff generate --ts 6x11 --data-dir /srv/bearoff

   ./blunderdb bearoff verify /srv/bearoff/gnubg_ts6x11.bd

analyze — Rattrapage gammonNet
---------------------------------

Écrit une analyse gammonNet pour chaque position qui n'en a aucune — le
rattrapage d'une bibliothèque constituée avant que cette fonctionnalité
existe (`ADR-0013 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md>`__,
`ADR-0015 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0015-blunderdb-serve-operates-on-a-library-it-does-not-expose-an-evaluator.md>`__).
C'est la même opération que le déclenchement
automatique après import et le bouton « Analyser maintenant » de l'interface
graphique, et que le point d'accès ``/v1/gammonnet.analyzeMissing`` du démon
``serve`` pour un tenant — trois formes différentes de la même opération, pas
trois logiques distinctes (voir :ref:`headless`).

.. code-block:: bash

   ./blunderdb analyze --db <path> [options]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--ply`` — Profondeur de recherche (défaut: 2, le paramètre canonique).
* ``--prune-k`` — Largeur d'élagage (défaut: 12, le paramètre canonique).
* ``--candidates`` — Nombre de coups candidats conservés par décision de
  déplacement (défaut: 10).
* ``--jobs`` — Nombre de positions analysées en parallèle (défaut: le nombre
  de cœurs de la machine).
* ``--compare`` — **N'écrit rien** : compare gammonNet aux analyses importées
  au lieu de combler des trous (voir plus bas).
* ``--limit`` — Avec ``--compare``, s'arrête après ce nombre de positions
  (0 = toutes).
* ``--format`` — Format de sortie: ``text`` (défaut, avec la progression) ou
  ``json`` (un seul document récapitulatif, imprimé à la fin).

**Le parallélisme** (``--jobs``). Les positions d'un lot sont indépendantes —
aucune recherche n'informe la suivante — donc elles sont réparties sur
``--jobs`` fils d'exécution, chacun avec son propre évaluateur. Les analyses
écrites sont **identiques quelle que soit la valeur de** ``--jobs`` ; seul le
temps de calcul change. ``--jobs 1`` laisse la machine libre pour autre
chose. L'annulation n'est pas affectée : Ctrl-C arrête le lot avant toute
nouvelle position, et tout ce qui était déjà calculé est écrit.

**La règle du trou** (`ADR-0013 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md>`__). Une position portant déjà une analyse —
XG, GNUbg, BGBlitz, ou un précédent passage de gammonNet — n'est jamais
touchée, quel que soit le moteur manquant. Seule une position **sans aucune**
analyse est écrite. La commande peut donc être relancée à tout moment sans
risque, et interrompue proprement : Ctrl-C annule sans rien perdre de ce qui
est déjà écrit, et la prochaine exécution reprend exactement là où la
précédente s'est arrêtée — aucun journal n'est nécessaire, puisque « les
positions sans analyse » est recalculé à chaque lancement.

**Exemple:**

.. code-block:: bash

   ./blunderdb analyze --db base.db

   # Analyzing 1204 position(s) with gammonNet (2-ply, k=12, 16 job(s))...
   #   1/1204 (0%)
   #   61/1204 (5%)
   #   ...
   #   1204/1204 (100%)
   # Done.

   ./blunderdb analyze --db base.db --jobs 1

.. _analyze_compare:

**``--compare`` : que vaut gammonNet sur *votre* bibliothèque ?**

La précision du moteur est mesurée ailleurs contre des corpus de référence et
contre la table de bearoff exacte. Aucune de ces mesures ne répond à la
question qu'un utilisateur se pose vraiment, qui porte sur **ses** positions :
sur les matchs importés d'XG, où le moteur embarqué est-il en désaccord avec
l'analyse venue du fichier, et que coûterait ce désaccord ?

``--compare`` répond, et **n'écrit rien**. Ce n'est pas une précaution mais
l'intérêt de la commande : l'ADR-0013 protège inconditionnellement une analyse
importée, et la comparaison peut donc se lancer sur une bibliothèque qu'on ne
veut surtout pas voir réécrite.

Le compte rendu donne :

* le **taux d'accord** sur la meilleure réponse, séparé entre coups de pions
  et décisions de videau — les deux n'ont rien à voir et un taux unique
  cacherait lequel des deux décroche ;
* le **coût du désaccord**, tarifé sur l'échelle de l'analyse importée : ce
  que le coup préféré par gammonNet vaut *selon le moteur importé*, moins ce
  que vaut son propre meilleur coup. Ce sens est le seul que les deux moteurs
  puissent chiffrer ensemble ; tarifer un désaccord deux fois inviterait à
  lire le plus petit des deux nombres ;
* la ventilation **par phase de partie**, qui est ce qui dit *où* les
  désaccords se concentrent ;
* les dix désaccords les plus coûteux, position par position.

Deux moteurs écrivent le même coup différemment — XG note « 13/7 » là où
gammonNet écrit « 13/8 8/7 », les frappes sont marquées d'un côté et pas de
l'autre, la répétition est parfois condensée en « (2) ». Ces différences sont
du dialecte et non du désaccord : la comparaison ramène les deux notations à
une forme canonique avant de les comparer. Sans cela, un corpus de test
affichait 78,8 % d'accord au lieu de 93,2 % — quinze points de faux
désaccords.

Un coup que le moteur importé n'a **pas listé** ne peut pas être tarifé sur
son échelle : il compte comme un désaccord de coût nul plutôt que d'un coût
inventé.

.. code-block:: bash

   # Comparer sur un échantillon de 500 positions
   ./blunderdb analyze --db base.db --compare --limit 500

   # compared: 118 decision(s)  (refused 2, failed 0)
   # same best answer: 93.2% (110/118)
   #   checker play:   93.7% (59/63)
   #   cube decision:  92.7% (51/55)
   # ...

trash — La corbeille
---------------------

Ce qui a été supprimé, et de quoi le remettre. Une suppression reste une
suppression : un instantané JSON de ce qui disparaît est écrit avant, et rien
d'autre dans la base ne sait que cette table existe — aucun filtre de
recherche, aucune statistique, aucune règle de rétention.

.. code-block:: bash

   ./blunderdb trash <sous-commande> --db <chemin> [options]

**Sous-commandes:**

* ``list`` — Ce qu'il y a dans la corbeille, du plus récemment supprimé au plus
  ancien.
* ``restore --id N`` — Remet l'entrée N et la retire de la corbeille.
* ``discard --id N`` — Supprime l'entrée N tout de suite, sans la restaurer.
* ``empty [--older-than J]`` — Vide la corbeille, ou seulement ce qui a plus de
  J jours.
* ``delete --kind K --id N`` — Supprime un objet **par la corbeille**, pour que
  le geste soit annulable. ``K`` vaut ``position``, ``collection`` ou
  ``comment``.

**Options communes:** ``--db`` (obligatoire), ``--kind``, ``--limit``
(défaut 50), ``--format`` (``text`` ou ``json``).

.. note:: ``blunderdb delete`` supprime toujours **sans** filet : un script qui
   supprime une position s'attend à ce qu'elle disparaisse, et laisser un
   instantané en silence ferait grossir un fichier que personne n'a demandé à
   voir grossir. C'est ``trash delete`` qui garde l'annulation.

Une restauration de position repasse par la déduplication Zobrist : elle ne
crée jamais de doublon, mais elle ne rend pas son ancien identifiant — la ligne
d'origine n'existe plus. Une position restaurée est la même position, sous un
nouveau numéro.

Ce qui a plus de trente jours est supprimé par ``blunderdb vacuum`` — jamais à
l'ouverture d'une base.

**Exemples:**

.. code-block:: bash

   # Supprimer une position en gardant l'annulation
   ./blunderdb trash delete --db base.db --kind position --id 412

   # Voir la corbeille, puis remettre une entrée
   ./blunderdb trash list --db base.db
   ./blunderdb trash restore --db base.db --id 3

   # Ne garder que ce qui a moins de trente jours
   ./blunderdb trash empty --db base.db --older-than 30

info — Métadonnées de la base
------------------------------

Affiche les métadonnées et les statistiques d'une base de données.

.. code-block:: bash

   ./blunderdb info --db <path> [--format <format>]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--format`` — Format de sortie: ``text`` ou ``json`` (défaut: ``text``).

**Exemples:**

.. code-block:: bash

   ./blunderdb info --db base.db

   # Database Information
   # ==================================================
   # Path: /home/jean/bg/base.db
   #
   # Metadata:
   #   Version: 2.19.0
   #   User: Jean
   #   Description: Matchs de tournoi 2025
   #   Date of Creation: 2026-09-06 02:43:51
   #
   # Statistics:
   #   Positions: 3859
   #   Analyses: 3855
   #   Matches: 11
   #   Games: 61
   #   Moves: 3766

``--format json`` ajoute l'origine du fichier — ``issuance`` porte le
filigrane s'il y en a un, et l'identité d'émetteur de cette machine :

.. code-block:: bash

   ./blunderdb info --db base.db --format json

.. code-block:: json

   {
     "issuance": {
       "watermarked": false,
       "issuerFingerprint": "1186-57FA-060C-9378",
       "issuerName": "unger"
     },
     "metadata": {
       "database_version": "2.19.0",
       "dateOfCreation": "2026-09-06 02:43:51",
       "description": "Matchs de tournoi 2025",
       "user": "Jean"
     },
     "path": "/home/jean/bg/base.db",
     "stats": {
       "analysis_count": 3855,
       "game_count": 61,
       "match_count": 11,
       "move_count": 3766,
       "position_count": 3859
     }
   }

edit — Modifier les métadonnées
--------------------------------

Modifie le nom d'utilisateur ou la description d'une base de données.

.. code-block:: bash

   ./blunderdb edit --db <path> [options]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--user`` — Nouveau nom d'utilisateur.
* ``--description`` — Nouvelle description.
* ``--clear-user`` — Effacer le nom d'utilisateur.
* ``--clear-description`` — Effacer la description.
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``
  (``{"changes": [...]}``).

Au moins une option de modification est requise.

**Exemples:**

.. code-block:: bash

   ./blunderdb edit --db base.db --user "Marie" --description "Ma collection"
   ./blunderdb edit --db base.db --clear-description

verify — Vérifier l'intégrité
-------------------------------

Vérifie l'intégrité de la base de données et, optionnellement, compare un match
avec son fichier source.

.. code-block:: bash

   ./blunderdb verify --db <path> [--match <id>] [--mat <file.mat>]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--match`` — ID du match à vérifier.
* ``--mat`` — Fichier MAT à comparer (utilisé avec ``--match``).
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``
  (statistiques, orphelins, écart de schéma, et la vérification du match le
  cas échéant).

Sans l'option ``--match``, la commande affiche les statistiques générales de la
base. Avec ``--match``, elle vérifie les données du match et peut les comparer
avec le fichier source original.

Chaque exécution contrôle aussi l'intégrité référentielle : elle compte les
lignes orphelines — parties sans match, coups sans partie, analyses de coup
sans coup, analyses sans position, entrées du journal de révision sans paquet
ou sans position — et affiche une ligne ``WARNING`` avec le total s'il y en a. Une base saine répond ``Orphaned rows: none``. Des orphelins
peuvent subsister dans une base écrite par une version qui n'appliquait pas les
clés étrangères sur toutes les connexions, ou avant que le journal de révision
ait les siennes ; ils ne sont rattachés à aucun match ni à aucun paquet et
n'occupent que de la place. La commande se termine tout de même avec le code
de sortie 0.

Chaque exécution compare aussi le schéma à la DDL de référence et liste les
tables, colonnes et index qui manquent à la base. L'ouverture d'une base ajoute
ce qui manque quand elle le peut et ne fait que journaliser ce qu'elle ne peut
pas ajouter (typiquement un index ``UNIQUE`` que des lignes en double
l'empêchent de reconstruire) : c'est ici que cet écart devient visible, et une
requête qui nomme l'un de ces éléments échoue tant que la cause n'est pas
corrigée. Une base saine répond ``Schema: matches the reference DDL``. Comme
les orphelins, un écart de schéma est un constat, pas un échec : le code de
sortie reste 0.

Chaque exécution contrôle enfin les règles que la DDL courante énonce mais que
SQLite ne sait pas ajouter à une table déjà créée : les contraintes ``CHECK``
de plage (dés entre 0 et 6, videau et pips positifs, sorties entre 0 et 15,
note de révision entre 1 et 4), le hash Zobrist qu'une ligne ne devrait jamais
omettre et l'unicité d'une analyse par position. Une base créée depuis la
version 2.18.0 du schéma les applique ; une base plus ancienne peut encore
porter des lignes qu'une base neuve refuserait, et ce sont elles qui sont
comptées ici, règle par règle. Une base saine répond ``Constraints: every row
satisfies the current DDL``. C'est un constat de plus : rien n'est réparé et le
code de sortie reste 0.

Chaque exécution recalcule enfin les deux compteurs dénormalisés,
``match.game_count`` et ``game.move_count``, à partir des lignes qu'ils
prétendent compter, et indique combien sont en désaccord et de combien au pire.
Tous deux sont écrits une seule fois, à l'import, d'après ce que contenait le
**fichier source**, et ce sont eux que la liste des matchs et la vue d'une
partie affichent : un petit écart est le plus souvent un import qui a sauté ce
qu'il ne savait pas convertir. Rien n'est réécrit — remplacer le compteur par
ce qui a été stocké effacerait justement l'écart qu'il y a lieu de regarder.
Une base saine répond ``Counters: game_count and move_count agree with the
rows``.

**Exemples:**

.. code-block:: bash

   ./blunderdb verify --db base.db
   ./blunderdb verify --db base.db --match 1
   ./blunderdb verify --db base.db --match 1 --mat original.mat

**En faire une garde.** Le code de sortie vaut 0 quoi que la commande
trouve : c'est ``--format json`` qui porte le verdict, et un script doit
lire les compteurs lui-même.

.. code-block:: json

   {
     "stats": {
       "analysis_count": 3855,
       "game_count": 61,
       "match_count": 11,
       "move_count": 3766,
       "position_count": 3859
     },
     "orphans": {
       "games_without_match": 0,
       "moves_without_game": 0,
       "move_analyses_without_move": 0,
       "analyses_without_position": 0,
       "reviews_without_deck": 0,
       "reviews_without_position": 0
     },
     "orphan_total": 0,
     "schema_drift": {
       "missing_tables": null,
       "missing_columns": null,
       "missing_indexes": null
     },
     "schema_drift_count": 0,
     "constraint_violations": [
       {"name": "position.zobrist_hash NOT NULL", "count": 0},
       {"name": "position.dice_1 BETWEEN 0 AND 6", "count": 0}
     ],
     "constraint_violation_total": 0,
     "counter_drift": {
       "matches_with_wrong_game_count": 0,
       "games_with_wrong_move_count": 53,
       "worst_game_count_gap": 0,
       "worst_move_count_gap": 2
     },
     "counter_drift_total": 53
   }

Trois champs valent une alarme : ``orphan_total``, ``schema_drift_count`` et
``constraint_violation_total``. Non nuls, ils décrivent une base à réparer.

.. code-block:: bash

   ./blunderdb verify --db base.db --format json \
     | jq -e '.orphan_total == 0 and .schema_drift_count == 0 and .constraint_violation_total == 0'

``counter_drift_total`` n'en fait pas partie, et l'exemple ci-dessus le
montre : la base qui l'a produit venait d'être importée et affiche déjà 53
parties dont le compteur de coups diffère de ce que les lignes contiennent.
Ces compteurs viennent du fichier source, pas de la base ; un écart raconte
l'import, il ne signale pas une corruption. Regardez-le, ne vous en servez
pas comme d'un seuil.

vacuum — Compacter la base de données
---------------------------------------

Récupère l'espace disque laissé par des suppressions (matchs, tournois,
purges): SQLite ne réduit jamais le fichier tout seul lorsqu'on supprime des
données, il faut le lui demander explicitement. C'est la seule façon de
déclencher un compactage — il ne se produit jamais automatiquement à
l'ouverture d'une base, car son coût est imprévisible sur une grosse base.

.. code-block:: bash

   ./blunderdb vacuum --db <path>

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``
  (``{"size_before", "size_after", "reclaimed"}``, en octets).

La commande commence par un ``wal_checkpoint(TRUNCATE)`` pour que la taille
affichée avant compactage soit honnête, vérifie qu'il reste sur le disque
environ deux fois la taille actuelle du fichier (SQLite reconstruit
entièrement la base avant de basculer dessus), effectue le ``VACUUM`` puis un
``ANALYZE`` pour rafraîchir les statistiques utilisées par le planificateur de
requêtes. Si l'espace disque manque, la commande refuse de démarrer avec un
message explicite plutôt que de risquer un compactage interrompu.

**Exemple:**

.. code-block:: bash

   ./blunderdb vacuum --db base.db

   # Compacting database...
   #   Before: 128.4 MiB
   #   After:  41.2 MiB
   #   Reclaimed: 87.2 MiB

repair — Recalculer les colonnes d'analyse
------------------------------------------

Recalcule les colonnes scalaires de chaque analyse à partir de l'analyse
elle-même, dont elles ne sont qu'une projection. Les analyses ne sont pas
touchées : ce sont les valeurs qu'on en avait tirées qui sont refaites.

.. code-block:: bash

   ./blunderdb repair --db <path>

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``
  (``{"repaired"}``, le nombre de lignes réellement changées).

Utile après une correction de la façon dont une analyse importée est lue. Le
cas s'est déjà produit : l'importeur XG écrit un « pas de double » de deux
façons, et la seconde était comprise comme un vrai double — la colonne portait
alors l'erreur d'un double qui n'avait jamais eu lieu. Corriger la lecture ne
changeait rien aux lignes déjà écrites ; cette commande les refait.

Rien ne la déclenche automatiquement, et c'est voulu : réécrire les colonnes
d'analyse de tout le monde à la simple ouverture d'une base n'est pas quelque
chose qu'un outil doit faire dans le dos de son utilisateur.

**Exemple:**

.. code-block:: bash

   ./blunderdb repair --db base.db

   # 42 analyses repaired.

delete — Supprimer des données
-------------------------------

Supprime un match et toutes les données associées (parties, coups, analyses).

.. code-block:: bash

   ./blunderdb delete --db <path> --type match --id <id> [--confirm]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--type`` — Type de suppression: ``match`` (obligatoire).
* ``--id`` — ID de l'élément à supprimer (obligatoire).
* ``--confirm`` — Supprimer sans demander de confirmation.
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``
  (``{"match_id": N, "deleted": true}``).

**Exemples:**

.. code-block:: bash

   # Confirmation interactive, puis sans confirmation (scripts)
   ./blunderdb delete --db base.db --type match --id 1
   ./blunderdb delete --db base.db --type match --id 1 --confirm

healthcheck — Sonder un démon
-----------------------------

Demande à un démon ``serve`` en marche (voir :doc:`mode_headless`) s'il est
disponible : une requête ``GET /readyz``, code de retour ``0`` si le démon
répond 200 (stockage joignable, schéma à la version attendue), ``1`` sinon —
stockage injoignable, schéma périmé, ou rien n'écoute à l'adresse. Aucun
fichier de base n'est ouvert.

.. code-block:: bash

   ./blunderdb healthcheck [--addr host:port] [--timeout 2s]

**Options:**

* ``--addr`` — Adresse d'écoute du démon (par défaut ``BLUNDERDB_ADDR``,
  sinon ``:8080``). Une adresse sans hôte (``:8080``) ou avec un hôte
  générique (``0.0.0.0``, ``[::]``) est sondée sur l'interface de bouclage.
* ``--timeout`` — Délai au-delà duquel la sonde abandonne (``2s`` par défaut).

C'est la commande que lance le ``HEALTHCHECK`` de l'image conteneur (une image
*distroless*, sans ``curl``) ; le binaire ``serve`` construit depuis
``cmd/serve`` la comprend aussi. Elle vaut tout autant dans un script ou une
unité systemd.

**Exemple:**

.. code-block:: bash

   ./blunderdb serve --db base.db --addr 127.0.0.1:8080 &
   ./blunderdb healthcheck --addr 127.0.0.1:8080 && echo "démon disponible"

   # ready

En cas d'échec la raison est affichée, ce que ``docker inspect`` reproduit
pour un conteneur ``unhealthy`` :

.. code-block:: text

   Error: healthcheck: http://127.0.0.1:8080/readyz answered 503 Service Unavailable (version_mismatch)

completion — Complétion shell
------------------------------

Affiche sur la sortie standard un script de complétion pour les noms de
sous-commandes. La liste des commandes intégrée à chaque script est générée
depuis la même table que ``blunderdb help`` et l'aiguillage de ``main.go``
(``handlers()``) : une nouvelle sous-commande est donc proposée par la
complétion dès qu'elle est câblée, sans rien à tenir à jour à la main.

.. code-block:: bash

   ./blunderdb completion <bash|zsh|fish>

**Exemples:**

.. code-block:: bash

   # bash
   source <(blunderdb completion bash)
   blunderdb completion bash | sudo tee /etc/bash_completion.d/blunderdb > /dev/null

   # zsh : un répertoire déjà sur $fpath
   blunderdb completion zsh > "${fpath[1]}/_blunderdb"

   # fish
   blunderdb completion fish | source

Les paquets installent cela automatiquement : le ``.deb``/``.rpm`` (nfpm) et
le paquet AUR génèrent les trois scripts depuis le binaire empaqueté au
moment de la construction, et le cask Homebrew exécute
``blunderdb completion <shell>`` une fois à l'installation via
``generate_completions_from_executable``. Rien n'est committé dans le dépôt :
la complétion ne peut donc jamais dériver de la table des sous-commandes.

version — Afficher la version
-----------------------------

Affiche la version de blunderDB et celle du schéma de base de données que ce
binaire écrit ; c'est la première chose à joindre à un rapport de bug.

.. code-block:: bash

   ./blunderdb version
   # blunderDB version 0.36.0 (database schema 2.19.0)

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
