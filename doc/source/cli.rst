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
   "identity", "Affiche ou déplace l'identité d'émetteur (clé de signature des filigranes)."
   "open", "Transforme un fichier protégé par mot de passe (.dbx) en base ordinaire."
   "search", "Recherche des positions avec filtres."
   "list", "Affiche le contenu de la base."
   "match", "Affiche les positions et analyses d'un match."
   "collection", "Gère les collections (liste, contenu, création, renommage, suppression, export)."
   "anki", "Paquets de répétition espacée (liste, statistiques, prévision, synchronisation)."
   "epc", "Calcule l'Effective Pip Count et le verdict de videau d'une position de sortie (XGID)."
   "analyze", "Écrit une analyse gammonNet pour chaque position qui n'en a aucune."
   "info", "Affiche les métadonnées de la base."
   "edit", "Modifie les métadonnées de la base."
   "verify", "Vérifie l'intégrité de la base."
   "vacuum", "Compacte le fichier de base de données, récupère l'espace libéré."
   "delete", "Supprime des données."
   "healthcheck", "Interroge un démon ``serve`` en marche : code 0 s'il est disponible."
   "completion", "Affiche un script de complétion shell (bash, zsh, fish)."
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

   ./blunderdb import --db <chemin> --type <type> [options]

**Options:**

* ``--db`` — Chemin de la base de données (obligatoire).
* ``--type`` — Type d'import: ``match``, ``position`` ou ``batch`` (obligatoire).
* ``--file`` — Fichier à importer (pour ``match`` et ``position``).
* ``--dir`` — Répertoire à importer (pour ``batch``).
* ``--recursive`` — Scanner récursivement les sous-répertoires (défaut: oui).
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``.
* ``--fail-on-error`` — Échoue si au moins un élément (``position`` ou
  ``batch``) n'a pas pu être importé, même quand d'autres ont réussi.

N'importer aucun élément est **toujours** une erreur, que ``--fail-on-error``
soit passé ou non: un fichier ou un répertoire dont rien n'a été importé
n'est jamais un succès silencieux. Un échec **partiel** (certains éléments
importés, d'autres non) n'échoue la commande que si ``--fail-on-error`` est
passé.

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

   # Sortie exploitable par un script, échec si un fichier a été refusé
   ./blunderdb import --db base.db --type batch --dir ./matchs/ --format json --fail-on-error

Un tableau récapitulatif indique pour chaque fichier si l'import a réussi
(✓), échoué (✗) ou s'il s'agit d'un doublon (⊘). Un doublon n'est pas un
échec en soi: relancer un import par lot sur un répertoire déjà importé,
sans nouveau fichier, reste un succès.

export — Exporter des données
------------------------------

Exporte le contenu de la base vers des fichiers.

.. code-block:: bash

   ./blunderdb export --db <chemin> --type <type> --file <sortie> [options]

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

   # Export complet de la base
   ./blunderdb export --db base.db --type database --file sauvegarde.db

   # Export des positions en JSON
   ./blunderdb export --db base.db --type positions --file positions.txt

   # Export de matchs spécifiques
   ./blunderdb export --db base.db --type matches --file selection.db --match-ids 1,3,5

   # Export d'un match en transcription .mat (Jellyfish)
   ./blunderdb export --db base.db --type mat --match-ids 5 --file match5.mat

   # Export de plusieurs matchs (ou de tous) en .mat dans un répertoire
   ./blunderdb export --db base.db --type mat --match-ids 5,9,12 --dir sorties/
   ./blunderdb export --db base.db --type mat --dir sorties/

   # Export filigrané et protégé par mot de passe (fichier .dbx)
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

   ./blunderdb identity                                       # nom et empreinte
   ./blunderdb identity --name "Jean Dupont"                  # renommer
   ./blunderdb identity --export jean.bdbid --passphrase pw   # exporter vers une autre machine
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

**Le langage de requête:**

Les drapeaux ci-dessus couvrent une partie des filtres seulement. ``--query``
donne accès au langage de requête de l'application — celui de la barre de
commande, décrit dans :doc:`cmd_mode` — et donc à tous les filtres qui ne se
dessinent pas sur le plateau : motif de coup, texte de commentaire, joueur,
date, équité, dés exclus, zones et blots.

.. code-block:: bash

   # Décisions de videau, 30 pips de retard, 50 millipoints d'erreur
   ./blunderdb search --db base.db --query 's cube p>30 E>50'

   # Filtres qu'aucun drapeau n'expose
   ./blunderdb search --db base.db --query 's m"13/11" t"blunder" pl"Alice" T>2026/01/01'

   # La liste des jetons compris
   ./blunderdb search --query-help

``--query`` remplace les drapeaux de filtre au lieu de s'y ajouter : les
combiner est refusé, avec le nom du drapeau en cause. Les drapeaux qui disent
*où* chercher et *comment* afficher — ``--db``, ``--format``, ``--limit``,
``--offset``, ``--export`` — restent valides.

Un jeton que rien ne reconnaît fait échouer la commande plutôt que de réduire
la recherche en silence. Deux limites tiennent à l'absence de plateau en ligne
de commande : le motif de damier ne se tape pas, et ``cube``, ``score`` et
``D`` se comparent au plateau de recherche, vide ici — utilisez ``--dice``,
``--cube`` et ``--score1``/``--score2``.

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

**Options (type ``stats`` uniquement):**

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

**Options (type ``imports`` uniquement):**

* ``--batch`` — Identifiant d'un lot : affiche son compte rendu complet au lieu
  de la liste.
* ``--format`` — Format de sortie: ``text`` ou ``json`` (défaut: ``text``).

La moitié mesurée du compte rendu est recalculée à chaque appel : un lot dont
les positions ont été analysées depuis rend les chiffres d'aujourd'hui, non ceux
du jour de l'import.

**Options (type ``players`` uniquement):**

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

**Exemples:**

.. code-block:: bash

   # Les imports enregistrés, puis le compte rendu de l'un d'eux
   ./blunderdb list --db base.db --type imports
   ./blunderdb list --db base.db --type imports --batch 3

   # Statistiques de la base
   ./blunderdb list --db base.db --type stats

   # Statistiques en MWC pour un joueur donné
   ./blunderdb list --db base.db --type stats --metric mwc --player "Alice"

   # Coups de pions uniquement, depuis une date
   ./blunderdb list --db base.db --type stats --decision-type checker --from 2026-01-01

   # Sortie JSON (pour un script)
   ./blunderdb list --db base.db --type stats --format json

   # Un tableau par joueur, borné aux dates d'une compétition
   ./blunderdb list --db base.db --type players --from 2026-03-01 --to 2026-03-08

   # Le même tableau en CSV (tableur ou script)
   ./blunderdb list --db base.db --type players --format csv

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

collection — Gérer les collections
----------------------------------

Gère les collections, ces ensembles de positions choisies à la main dans le
panneau Collections de l'interface graphique. Chaque sous-commande prend
``--db`` ; ``list`` et ``show`` acceptent ``--format text`` (défaut), ``json``
ou ``csv``, comme ``list``.

.. code-block:: bash

   ./blunderdb collection <sous-commande> [options]

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

   # Toutes les collections
   ./blunderdb collection list --db base.db

   # Positions de la collection 3, en CSV pour un tableur
   ./blunderdb collection show --db base.db --id 3 --format csv

   # Créer, renommer, supprimer
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

   ./blunderdb anki <sous-commande> [options]

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

   # Régime exact (les deux joueurs ont 6 pions ou moins)
   ./blunderdb epc 'XGID=-BBB------------------bbb-:0:0:1:00:0:0:0:0:10'

   # Avec la base TS-06-11 calculée (exact jusqu'à 11 pions par joueur)
   ./blunderdb epc --bearoff-ts ~/.local/share/blunderdb/gnubg_ts6x11.bd 'XGID=…'

bearoff — Bases de sortie
--------------------------

Fabrique et gère les bases de bearoff. Rien n'est téléchargé et rien n'est
embarqué : une table est calculée ici et vérifiée contre l'empreinte que gnubg
produit pour son domaine. Aucune sous-commande ne parle à une base de données —
une table de bearoff est de l'arithmétique sur le jeu, pas sur les positions de
quelqu'un — donc aucune ne prend ``--db``.

.. code-block:: bash

   ./blunderdb bearoff generate --ts <domaine> [options]
   ./blunderdb bearoff list [options]
   ./blunderdb bearoff verify <fichier.bd> [options]
   ./blunderdb bearoff delete --ts <domaine> [options]

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

   # Calculer TS-06-09 sur quatre cœurs
   ./blunderdb bearoff generate --ts 6x9 --cores 4

   # Calculer OS-08 : l'EPC répond alors jusqu'à un pion sur la 8
   ./blunderdb bearoff generate --os 8

   # Sur un serveur, dans le volume que lit le démon
   ./blunderdb bearoff generate --ts 6x11 --data-dir /srv/bearoff

   # Vérifier une table reçue d'ailleurs
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

   ./blunderdb analyze --db <chemin> [options]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--ply`` — Profondeur de recherche (défaut: 2, le paramètre canonique).
* ``--prune-k`` — Largeur d'élagage (défaut: 12, le paramètre canonique).
* ``--candidates`` — Nombre de coups candidats conservés par décision de
  déplacement (défaut: 10).
* ``--jobs`` — Nombre de positions analysées en parallèle (défaut: le nombre
  de cœurs de la machine).
* ``--format`` — Format de sortie: ``text`` (défaut, avec la progression) ou
  ``json`` (un seul document récapitulatif, imprimé à la fin).

**Le parallélisme (``--jobs``).** Les positions d'un lot sont indépendantes —
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

   # Sur un seul cœur, pour laisser la machine à autre chose
   ./blunderdb analyze --db base.db --jobs 1

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
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``
  (``{"changes": [...]}``).

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

   # Vérification globale
   ./blunderdb verify --db base.db

   # Vérifier un match spécifique
   ./blunderdb verify --db base.db --match 1

   # Comparer avec le fichier source
   ./blunderdb verify --db base.db --match 1 --mat original.mat

vacuum — Compacter la base de données
---------------------------------------

Récupère l'espace disque laissé par des suppressions (matchs, tournois,
purges): SQLite ne réduit jamais le fichier tout seul lorsqu'on supprime des
données, il faut le lui demander explicitement. C'est la seule façon de
déclencher un compactage — il ne se produit jamais automatiquement à
l'ouverture d'une base, car son coût est imprévisible sur une grosse base.

.. code-block:: bash

   ./blunderdb vacuum --db <chemin>

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

   ./blunderdb repair --db <chemin>

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

   ./blunderdb delete --db <chemin> --type match --id <id> [--confirm]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--type`` — Type de suppression: ``match`` (obligatoire).
* ``--id`` — ID de l'élément à supprimer (obligatoire).
* ``--confirm`` — Supprimer sans demander de confirmation.
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``
  (``{"match_id": N, "deleted": true}``).

**Exemples:**

.. code-block:: bash

   # Supprimer avec confirmation interactive
   ./blunderdb delete --db base.db --type match --id 1

   # Supprimer sans confirmation (pour scripts)
   ./blunderdb delete --db base.db --type match --id 1 --confirm

healthcheck — Sonder un démon
-----------------------------

Demande à un démon ``serve`` en marche (voir :doc:`mode_headless`) s'il est
disponible : une requête ``GET /readyz``, code de retour ``0`` si le démon
répond 200 (stockage joignable, schéma à la version attendue), ``1`` sinon —
stockage injoignable, schéma périmé, ou rien n'écoute à l'adresse. Aucun
fichier de base n'est ouvert.

.. code-block:: bash

   ./blunderdb healthcheck [--addr hôte:port] [--timeout 2s]

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

   # bash : charger pour la session en cours
   source <(blunderdb completion bash)

   # bash : installation système (Debian/Ubuntu/Arch)
   blunderdb completion bash | sudo tee /etc/bash_completion.d/blunderdb > /dev/null

   # zsh : installer dans un répertoire déjà sur $fpath
   blunderdb completion zsh > "${fpath[1]}/_blunderdb"

   # fish : charger pour la session en cours
   blunderdb completion fish | source

Les paquets installent cela automatiquement : le ``.deb``/``.rpm`` (nfpm) et
le paquet AUR génèrent les trois scripts depuis le binaire empaqueté au
moment de la construction, et le cask Homebrew exécute
``blunderdb completion <shell>`` une fois à l'installation via
``generate_completions_from_executable``. Rien n'est committé dans le dépôt :
la complétion ne peut donc jamais dériver de la table des sous-commandes.

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
