.. _cli_import_export_chapitre:

Importer, exporter, transférer
==============================

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
* ``--watch`` — Avec ``--type batch`` : ne s'arrête pas, et importe chaque
  fichier de match **à mesure qu'il apparaît** dans ``--dir`` (Ctrl-C pour
  arrêter).
* ``--watch-every`` — Intervalle entre deux regards de ``--watch`` (défaut:
  10s, plancher 2s).
* ``--resume`` — Avec ``--type batch`` : reprend le lot de ce numéro (affiché
  au départ de l'import, ou ``batch_id`` en JSON). Les fichiers de même
  chemin, taille et date que dans son journal sont sautés sans être lus, ceux
  de même contenu sont lus mais pas analysés ; les fichiers en erreur sont
  retentés.
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``.
* ``--fail-on-error`` — Échoue si au moins un élément (``position`` ou
  ``batch``) n'a pas pu être importé, même quand d'autres ont réussi.
* ``--skip-duplicates`` — Ignore un match déjà en base sans en reprendre les
  analyses plus profondes (les marques d'étude restent appliquées). Par
  défaut, un doublon remplace chaque analyse rangée par la sienne quand elle
  est strictement plus profonde, et la ligne l'indique : ``DUPLICATE (N
  analyses deepened)``.

Une décision dont l'analyse contient une valeur qui n'est pas un nombre fini
(NaN ou infini) est importée sans cette analyse, et le reste du match entre
normalement ; le compte rendu du lot en donne le nombre (``Decisions imported
without their analysis``).

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

Surveiller un dossier
^^^^^^^^^^^^^^^^^^^^^

``--watch`` transforme l'import de répertoire en surveillance : la commande ne
rend pas la main et importe chaque fichier de match qui **apparaît** dans le
dossier. C'est la forme sans interface du :ref:`dossier surveillé
<dossier_surveille>` de l'application.

.. code-block:: bash

   # Importer ce que le dossier contient déjà, puis surveiller ce qui arrive
   ./blunderdb import --db base.db --type batch --dir ~/XG/Matches
   ./blunderdb import --db base.db --type batch --dir ~/XG/Matches --watch

Seuls les fichiers qui apparaissent sont importés : ce que le dossier contient
au démarrage est enregistré comme connu et laissé tranquille — pointer une
surveillance sur quatre ans de matchs ne doit pas les importer tous. Les deux
commandes ci-dessus se composent donc exactement comme on l'espère.

Un fichier n'est importé qu'une fois sa **taille stable**, c'est-à-dire vu
deux fois inchangé : un match qu'un autre programme est en train d'écrire
grossit d'un regard à l'autre, et l'importer à moitié écrit donnerait une
erreur syntaxique sur laquelle personne ne peut agir. Le dossier n'est pas
parcouru récursivement. Un partage réseau devenu illisible n'arrête pas la
surveillance, et son contenu ne passe pas pour nouveau à son retour.

Une base ``.db`` qui apparaît dans le dossier est fusionnée comme par
``import --type database`` : c'est ainsi qu'arrive l'export incrémental d'une
autre machine (voir :ref:`synchroniser_deux_machines`).

Ctrl-C arrête entre deux fichiers, jamais au milieu de l'un : le fichier en
cours finit son import et son compte rendu s'affiche avant que la commande ne
rende la main.

Import d'un match
^^^^^^^^^^^^^^^^^

Formats supportés: eXtreme Gammon (``.xg``, ``.xgp``), GNUbg (``.sgf``),
Jellyfish (``.mat``, ``.txt``), BGBlitz (``.bgf``) et HedgeHog (``.ogxm``).

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

Import d'une base
^^^^^^^^^^^^^^^^^

Fusionne une autre base blunderDB (*.db*) dans celle de ``--db``, selon les
règles de la fusion de l'application de bureau et d'``imports.db`` en mode
serveur (:ref:`guide_fusionner_base`) : positions, matchs, tournois,
collections, leçons, paquets Anki, brouillons et alias, ainsi que la Direction
des tournois et leur Rencontre. Importer deux fois le même fichier ne change
rien. La sortie texte compte les matchs ajoutés et déjà présents, les
Directions et Rencontres relues, et nomme les collections vivantes figées ou
laissées sur une liste manuelle, et les familles que la base est trop ancienne
pour porter.

.. code-block:: bash

   ./blunderdb import --db base.db --type database --file club.db
   ./blunderdb import --db base.db --type database --file club.db --format json

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

Chaque fichier du lot est journalisé (chemin, taille, date, SHA-256, match
obtenu ou erreur). Le numéro du lot est affiché au départ ; si l'import est
interrompu, ``--resume`` le continue sans analyser de nouveau ce qui est décidé. Avec
``--format json``, l'objet final porte ``batch_id`` et le tableau ``journal``.

.. code-block:: bash

   ./blunderdb import --db base.db --type batch --dir ./matchs/ --resume 12

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
* ``--transcriptions`` — Inclure le brouillon de transcription de chaque match
  exporté (défaut: oui).
* ``--aliases`` — Inclure les alias de joueurs et d'événements que les matchs
  exportés écrivent, pour que la fusion de joueurs les suive (défaut: oui).
* ``--collections`` — Inclure les collections (défaut: oui).
* ``--tournaments`` — Inclure les tournois (défaut: oui).
* ``--decks`` — Inclure les paquets Anki (défaut: oui).
* ``--lessons`` — Inclure les leçons, avec les collections et les positions
  que leurs étapes montrent (défaut: oui).
* ``--anki-history`` — Emporter l'historique de révision des paquets exportés :
  l'état de chaque carte et le journal de ses révisions (défaut: oui).
* ``--collection-ids`` — IDs de collections à exporter (séparés par des
  virgules, vide = toutes).
* ``--match-ids`` — IDs de matchs à exporter (séparés par des virgules, vide = tous).
* ``--tournament-ids`` — IDs de tournois à exporter (séparés par des virgules,
  vide = tous).
* ``--deck-ids`` — IDs de paquets Anki à exporter avec leurs positions
  (séparés par des virgules, vide = tous).
* ``--lesson-ids`` — IDs de leçons à exporter (séparés par des virgules,
  vide = toutes).

* ``--password`` — Enveloppe le résultat dans un conteneur chiffré (``.dbx``).
* ``--watermark`` — Écrit une déclaration d'origine **signée** dans le fichier
  exporté (voir :ref:`diffusion_controlee`).
* ``--watermark-note`` — Texte libre associé au filigrane (conditions
  d'usage, contact) ; utilisé avec ``--watermark``.
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json`` (document
  résumant l'export: chemin, taille en octets, nombres).
* ``--incremental`` — Avec ``--type database`` : n'exporte que ce qui est entré
  dans la bibliothèque depuis le dernier export incrémental vers le même
  dossier, puis avance ce repère dans la base exportée. ``--file`` peut alors
  être un dossier, où un fichier ``sync-<machine>-<date>.db`` est créé (voir
  :ref:`synchroniser_deux_machines`). Un nom de fichier déjà pris est refusé :
  le fichier en place porte le delta précédent, que l'autre machine n'a
  peut-être pas encore fusionné.
* ``--sync-cursor`` — Avec ``--incremental`` : nom du repère lu et avancé, à la
  place du dossier de destination.
* ``--plan`` — Avec ``--type database`` ou ``matches`` : compte, famille par
  famille, ce que l'export écrirait, sans écrire de fichier (``--file`` est
  alors facultatif). Les comptes sont ceux que l'export rapporte ensuite ; les
  collections vivantes qui partiraient figées et les paquets « recherche » dont
  la source perdrait une partie sont nommés. Avec un autre type, l'option est
  refusée.
* ``--plan-ids <famille>`` — Comme ``--plan``, mais rend une page des
  identifiants derrière un compte : ``positions``, ``comments`` (les positions
  qui les portent), ``matches``, ``tournaments``, ``directions`` (les tournois dont la Direction part),
  ``rencontres`` (les Rencontres qui partent avec un tournoi), ``collections``, ``lessons``, ``ankiDecks``, ``transcriptions`` ou ``filters``. ``--offset`` et ``--limit``
  choisissent la page (1000 identifiants au plus).

``--type database`` exporte toute la bibliothèque, comme l'export du serveur
et comme la fenêtre d'export quand la liste affichée est la bibliothèque :
positions, matchs, tournois (avec leur Direction : journal, paires, tables et
rattachement des matchs exportés ; leur Rencontre part dès qu'un de ses tournois part), collections, leçons, paquets Anki, transcriptions, alias et filtres. Une option à ``false`` retire sa famille (``--collections=false``) ;
une liste d'IDs la restreint à ces éléments et l'inclut.
Un brouillon de transcription part avec le match qu'il a produit ; un brouillon
qui n'a pas encore produit de match ne part que si tous les matchs partent.
``--type matches`` emporte aussi le brouillon de chaque match.

**Exemples:**

.. code-block:: bash

   ./blunderdb export --db base.db --type database --file sauvegarde.db
   ./blunderdb export --db base.db --type positions --file positions.txt
   ./blunderdb export --db base.db --type matches --file selection.db --match-ids 1,3,5

   # .mat : un match, puis plusieurs (ou tous) dans un répertoire
   ./blunderdb export --db base.db --type mat --match-ids 5 --file match5.mat
   ./blunderdb export --db base.db --type mat --match-ids 5,9,12 --dir sorties/
   ./blunderdb export --db base.db --type mat --dir sorties/

   # Ce qui est nouveau depuis le dernier export vers ce dossier, pour une
   # autre machine qui le surveille
   ./blunderdb export --db base.db --type database --incremental --file ~/Sync/vers-portable/

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

.. _cli_transfer:

transfer — Copier d'une bibliothèque à une autre
------------------------------------------------

Copie une partie d'une bibliothèque dans une autre, sans fichier
intermédiaire à manipuler. La sélection est celle de ``export --type
database`` ; la fusion est celle de l'import d'une base ``.db`` : une
position que la destination possède déjà n'est pas ajoutée une seconde fois
(elle est reconnue à son hachage), ses analyses et ses commentaires se
fusionnent, et un match, une collection, une leçon ou un paquet Anki déjà
présent n'est pas dupliqué. Copier de A vers B puis de B vers A ne crée donc
aucun doublon.

.. code-block:: bash

   ./blunderdb transfer --from <source.db> --to <destination.db> [options]

**Options:**

* ``--from`` — Bibliothèque source (obligatoire ; elle doit exister et être au
  schéma de cette version de blunderDB). Une bibliothèque plus ancienne est
  refusée, car la mettre à jour serait y écrire : ouvrez-la une fois dans
  blunderDB. Une bibliothèque plus récente est refusée aussi : mettez
  blunderDB à jour.
* ``--to`` — Bibliothèque de destination (obligatoire ; créée si elle
  n'existe pas). Elle ne peut pas être la source.
* ``--analysis``, ``--comments``, ``--played-moves``, ``--matches``,
  ``--transcriptions``, ``--aliases``, ``--collections``, ``--tournaments``,
  ``--decks``, ``--lessons`` — Comme pour ``export`` : une option à ``false``
  retire sa famille (défaut : oui pour toutes).
* ``--collection-ids``, ``--match-ids``, ``--tournament-ids``,
  ``--deck-ids``, ``--lesson-ids`` — Une liste d'IDs de la source restreint
  sa famille à ces éléments et l'inclut. Un ID que la source ne contient pas
  est une erreur, et rien n'est copié.
* ``--format`` — ``text`` (défaut) ou ``json`` (ce qui a été lu dans la
  source et ce que la destination en a pris).

**Quelles positions partent.** Sans liste d'IDs, toutes les positions de la
source partent, rangées ou non. Dès qu'une liste d'IDs est donnée, seules
partent les positions qu'atteignent les familles retenues (les membres d'une
collection, les coups d'un match, les étapes d'une leçon, les cartes d'un
paquet), avec leurs analyses et leurs commentaires. Une collection part
toujours avec ses membres. Un match nommé par ``--match-ids`` emporte la fiche
de son tournoi (nom, date, lieu), mais pas les autres matchs de ce tournoi ;
avec ``--tournaments=false``, il arrive sans tournoi.

Rien n'est écrit qui dise d'où vient la copie : ni filigrane, ni journal
d'import, ni métadonnée de la source ; la destination garde les siennes, et
la source ne reçoit rien. La bibliothèque de filtres et l'historique de
révision des paquets Anki ne sont pas copiés : un paquet arrive sans
historique, et un paquet du même nom déjà présent garde le sien.

**Exemples:**

.. code-block:: bash

   # Toute une bibliothèque dans une autre
   ./blunderdb transfer --from club.db --to perso.db

   # Deux collections et leurs positions, rien d'autre
   ./blunderdb transfer --from club.db --to perso.db --collection-ids 3,7 \
       --matches=false --tournaments=false --lessons=false --decks=false

   # Un match, ses positions et son tournoi, en retour
   ./blunderdb transfer --from perso.db --to club.db --match-ids 12 \
       --collections=false --lessons=false --decks=false

Le serveur offre le même transfert entre deux tenants, tiré par le tenant qui
reçoit (voir :ref:`mode_headless_transfert`).

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
