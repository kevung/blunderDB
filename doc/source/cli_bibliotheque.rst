.. _cli_bibliotheque_chapitre:

Bibliothèque : recherche, listes, collections
=============================================

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
* ``--comment-origin <o1,o2>`` — Uniquement les positions portant un
  commentaire de ces provenances : ``user``, ``xg``, ``gnubg``, ``bgf``,
  ``unknown`` (``co:``).
* ``--comment-author <nom>`` — Uniquement les positions portant un commentaire
  attribué à ce nom (nom entier, casse ignorée).
* ``--phase <p1,p2>`` — Uniquement les positions de ces phases de jeu :
  ``opening``, ``middlegame``, ``race``, ``bearoff`` (``ph:``).
* ``--game-type <t1,t2>`` — Uniquement les positions de ces plans de jeu :
  ``race``, ``bearin``, ``crunch``, ``backgame``, ``acepoint``, ``blitz``,
  ``primevprime``, ``mutualholding``, ``holding``, ``contact`` (``gt:``).
  Phase et plan sont des étiquettes dérivées du plateau, que
  ``blunderdb repair`` recalcule ; une valeur inconnue fait échouer la
  commande.

Les filtres de corpus portent sur le match où la position a été rencontrée ;
chacun équivaut à un jeton du langage de requête (:ref:`cmd_filter`) :

* ``--player <nom>`` — Les matchs où ce joueur a joué, d'un côté ou de
  l'autre ; la casse est ignorée et ``*`` remplace n'importe quelle suite de
  caractères (``pl"…"``). Avec ``--seat-only``, seulement les décisions prises
  par ce joueur (``pl!"…"``).
* ``--opponent <nom>`` — Avec ``--player``, les matchs contre cet adversaire ;
  seul, un nom d'un côté ou de l'autre (``op"…"``).
* ``--tournament-name <nom>`` — Les matchs des tournois de ce nom (``tn"…"``).
* ``--round <r1,r2>`` — Les matchs de ces rondes (``rd:``).
* ``--match-lengths <expr>`` — La longueur du match : ``7``, ``5,9``, ``>5``,
  ``<9`` (``ml``). À ne pas confondre avec ``--match-length``, qui accompagne
  le score de la position.
* ``--match-date <expr>`` — La date du match : ``2024``,
  ``2024-01..2024-12``, ``>2024-06``, ``<2024-06`` (``md``) ; ce n'est pas la
  date de l'analyse.
* ``--pr <expr>`` — Le PR du match du joueur qui a pris la décision :
  ``>8``, ``<5``, ``4,9`` (``pr``).
* ``--analysis <v1,v2>`` — Moteurs et profondeurs de l'analyse enregistrée :
  ``xg``, ``gnubg``, ``bgblitz``, ``hedgehog``, ``gammonnet``, ``3ply``,
  ``3ply+``, ``book``, ``rollout`` (``ad:``).
* ``--cube-response <type>`` — Un seul type de décision de videau :
  ``double`` (Double / No Double) ou ``takepass`` (Take / Pass).

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
``id``, ``xgid``, ``score``, ``cube``, ``decision_type`` (``checker`` ou
``cube``) et ``dice``, toujours présents ; ``best_move`` et ``equity``
n'apparaissent que si la position porte une analyse qui les renseigne. La ligne
``Found n position(s)`` va sur la sortie d'erreur : la sortie standard ne porte
que le JSON.

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

``--format xgid`` imprime un XGID par ligne, préfixé de ``XGID=``, et **rien
d'autre** sur la sortie standard. Le XGID enregistré avec l'analyse est
préféré, car il porte la longueur de match de la source ; à défaut, il est
régénéré depuis le damier. Le même XGID figure dans le champ ``xgid`` du JSON.

**Le langage de requête:**

Les drapeaux ci-dessus couvrent une partie des filtres seulement. ``--query``
donne accès au langage de requête de l'application — celui de la barre de
commande — et donc à tous les filtres qui ne se dessinent pas sur le plateau :
motif de coup, texte de commentaire, date d'ajout, équité, dés exclus, zones
et blots.

La grammaire n'est écrite qu'à un seul endroit, :ref:`cmd_filter`. Sa table
donne chaque jeton, sa forme, et le drapeau de ``search`` qui lui correspond
quand il en existe un. Cette page ne la répète pas.

.. code-block:: bash

   ./blunderdb search --db base.db --query 's p>30 E>50'
   ./blunderdb search --db base.db --query 's m"13/11" t"blunder" pl"Alice" T>2026/01/01'

Le classement des voisines d'une position passe par la même grammaire :
``--query 's like42'``, seul ou suivi d'autres jetons pour restreindre
l'ensemble classé.

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

   ./blunderdb list --db <path> --type <type> [--limit <n>] [--offset <n>]

**Types:**

* ``matches`` — Liste des matchs importés.
* ``tournaments`` — Liste des tournois.
* ``positions`` — Liste des positions (limité à 10 par défaut ;
  ``--offset <n>`` saute les *n* premières). Seule la fenêtre affichée est
  lue, quelle que soit la taille de la base. Avec
  ``--format csv``, devient un **export tabulaire** : une ligne par position,
  avec son XGID, sa phase, son score, son videau, ses pips et les colonnes
  d'analyse dérivées.
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
* ``moves`` — **Export tabulaire** des coups enregistrés, un par ligne, avec le
  match auquel ils appartiennent recopié sur chaque ligne : identifiants,
  date, joueurs, longueur, numéro et type de coup, position, dés, coup joué,
  action de videau, chance. ``--format csv`` obligatoire.
* ``analyses`` — **Export tabulaire** des analyses stockées, une par ligne :
  moteur, profondeur, meilleur coup et son équité, erreur du coup joué,
  meilleure action de videau et son erreur, les six taux de gain.
  ``--format csv`` obligatoire.
* ``tags`` — Vocabulaire de tags de la base : chaque ``#mot`` écrit dans un
  commentaire, avec le nombre de **positions** qui le portent, du plus utilisé
  au moins utilisé. Sur une base sans aucun tag, affiche le vocabulaire
  recommandé plutôt qu'une liste vide (voir :ref:`tags`). Accepte
  ``--format json`` et ``--format csv``.
* ``study`` — Par plan de jeu, les positions distinctes révisées dans les
  paquets Anki sur les ``--days`` derniers jours (défaut : 30) et le PR des
  matchs joués avant et depuis, chacun avec son nombre de décisions ; un PR
  appuyé sur moins de dix décisions s'affiche « — ». Trois nombres côte à
  côte, non un effet : rien n'y tient compte des adversaires, du format ou
  des dés. Accepte ``--format json`` et ``--format csv``.

.. _export_tabulaire:

Exports tabulaires
^^^^^^^^^^^^^^^^^^

Trois types — ``positions``, ``moves`` et ``analyses`` — s'exportent en CSV
pour un notebook, un tableur ou un script :

.. code-block:: bash

   ./blunderdb list --db base.db --type positions --format csv > positions.csv
   ./blunderdb list --db base.db --type moves     --format csv > moves.csv
   ./blunderdb list --db base.db --type analyses  --format csv > analyses.csv

``--limit`` ne s'applique **que si vous le passez**. Sa valeur par défaut (10)
existe pour qu'un ``list`` affiché dans un terminal ne fasse pas défiler la
base entière ; un export, lui, part dans un fichier que lit un programme, et le
tronquer silencieusement à dix lignes serait un piège qu'on ne remarque qu'une
fois les chiffres faux.

**Les colonnes sont un contrat.** Un notebook ou un script écrit contre ces
noms doit continuer de fonctionner : les colonnes s'ajoutent à la fin, ne sont
jamais renommées ni réordonnées. Toutes les équités sont en **millipoints
entiers**, parce que c'est ainsi qu'elles sont stockées et parce qu'un flottant
dans un CSV invite une locale à le reformater.

**Parquet n'est pas proposé**, et c'est mesuré plutôt que dogmatique : une
bibliothèque colonnaire pèse plusieurs mégaoctets dans un binaire dont la
taille est un souci suivi, alors que tout ce à quoi cet export sert lit du CSV
en une ligne (``pd.read_csv``, ``polars.read_csv``, ``read.csv``, un tableur).
Parquet se justifie à des dizaines de millions de lignes ; une bibliothèque de
backgammon de dix ans en compte cent mille. Si un jour la différence se mesure
sur une vraie base, c'est cette mesure qui rouvrira la question.

Un **notebook Jupyter d'exemple** accompagne ces exports
(``notebooks/blunderdb-analyse.ipynb`` dans le dépôt) : PR dans le temps,
distribution des magnitudes d'erreur, dix pires décisions avec leur XGID. Il
n'utilise rien d'autre que ces trois fichiers CSV, et il est **exécuté toutes
les nuits en intégration continue** — un notebook qu'on ne lance pas est un
notebook qui a cessé de fonctionner sans que personne le sache.

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
* ``--engine`` / ``--min-depth`` — Ne compte que les décisions analysées par ce
  moteur (nom exact, tel qu'enregistré) ou à cette profondeur au moins (en
  plis) ; acceptés aussi par le type ``players``.
* ``--format`` — Format de sortie: ``text`` ou ``json`` (défaut: ``text``).

**Options** (type ``matches`` uniquement) :

* ``--query`` — Ne garde que les matchs dont un joueur, l'événement, le lieu,
  la manche, le tournoi ou la date contient ce texte (sans tenir compte de la
  casse).
* ``--sort`` — Ordre : ``date`` (défaut, le plus récent d'abord), ``date_asc``,
  ``length_asc``, ``length_desc``, ``player1``, ``player1_desc``, ``player2``,
  ``player2_desc``, ``tournament``, ``tournament_desc`` ou ``opponent``.

Le filtre et le tri sont faits par la base, comme dans le panneau Matchs ; le
total annoncé est celui des matchs retenus. Le serveur les offre sur
``/v1/matches.list`` (champs ``text`` et ``sort``) et ``/v1/matches.count``.

**Options** (type ``imports`` uniquement) :

* ``--batch`` — Identifiant d'un lot : affiche son compte rendu complet au lieu
  de la liste.
* ``--queue`` — Avec ``--batch`` : la **file d'étude** du lot plutôt que son
  compte rendu — les positions qui méritent un second regard, dans l'ordre où
  les parcourir (voir :ref:`file_etude`). Les décisions qui ont coûté quelque
  chose d'abord, puis les positions marquées dans le logiciel d'origine, puis
  les décisions de videau serrées ; une position n'y figure qu'une fois.
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
finale rappelle quelle fenêtre est affichée, tronquée par ``--limit`` (dont la
valeur par défaut vaut 10 pour les positions) et décalée par ``--offset`` :

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

   (Showing 1-10 of 3859 positions, use --offset and --limit to see more)

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

.. _cli_collection:

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
* ``create --name <nom> [--description <texte>] [--query <requête>]`` — Crée
  une collection vide ou, avec ``--query``, une collection vivante d'emblée.
  Une requête dont un jeton n'est reconnu par aucune règle est refusée avant
  que rien ne soit écrit.
* ``filter --id <id> --query <requête>`` — Rend une collection **vivante** :
  son contenu devient le résultat d'une recherche, réévalué chaque fois
  qu'elle est ouverte. La requête s'écrit dans la grammaire de recherche de
  l'application (voir :ref:`cmd_filter`). ``--clear`` la rend à sa liste faite
  à la main, en gardant les positions qu'elle contenait.
* ``freeze --id <id>`` — Fige une collection vivante : les positions que sa
  requête sélectionne maintenant deviennent sa composition, la requête
  s'efface. Refusé sur une collection qui n'est pas vivante.
* ``evaluate --id <id> [--limit <n>] [--format text|json]`` — Lit les
  identifiants des positions d'une collection, en rejouant maintenant la
  requête d'une collection vivante. La réponse est bornée par un plafond
  déclaré de 5 000 positions (``--limit`` l'abaisse) et donne toujours le
  nombre réel de positions et l'indicateur ``truncated`` : jamais de
  troncature muette. En texte, les identifiants vont sur la sortie standard et
  le décompte sur la sortie d'erreur.
* ``pile [--position-id <id> | --xgid <XGID>]`` — Met une position sur la
  Pile (la collection « à revoir plus tard »), ou l'en retire quand elle y est ;
  une position donnée par XGID qui n'est pas dans la base y est d'abord écrite,
  comme une position apportée seule. Sans position, affiche la Pile.
* ``rename --id <id> --name <nom> [--description <texte>]`` — Renomme une
  collection (la description est conservée si elle n'est pas donnée).
* ``delete --id <id> [--confirm]`` — Supprime une collection ; ses positions
  restent dans la base.
* ``export --id <id[,id…]> --out <fichier.db> [--analysis=false]
  [--comments=false] [--matches=false] [--transcriptions=false]
  [--aliases=false] [--watermark <texte>] [--watermark-note <texte>]
  [--password <mot de passe>]`` — Exporte une ou plusieurs collections vers
  un nouveau fichier de base, par le même exporteur que la fenêtre d'export
  de l'interface graphique et que ``export --type database`` : le fichier
  emporte aussi les matchs où se sont jouées les positions, avec leurs
  transcriptions et les alias de leurs joueurs et événements (voir la
  commande ``export`` pour le filigrane et le mot de passe).
* ``suggest [--player <nom>] [--match <ids>] [--tournament <ids>] [--from
  <date>] [--to <date>] [--decision-type all|checker|cube] [--size <n>]
  [--collection <nom>] [--deck <nom>] [--format text|json]`` — Propose des
  **positions de référence** parmi les erreurs du filtre : celles dont la
  leçon couvre le plus de MWC récupérable de leurs erreurs voisines, sans
  quasi-doublon ni position déjà traitée (voir
  :ref:`positions de référence <positions_reference>`). ``--size`` vaut 20
  par défaut, 50 au plus. Rien n'est écrit sans ``--collection``, qui range
  les positions proposées dans une nouvelle collection, ou ``--deck``, qui en
  fait un paquet Anki. Chaque ligne donne la famille, le score d'une
  référence de videau, les erreurs et matchs qu'elle résume, le MWC couvert
  et sa raison (qui donne le chiffre compté quand la leçon compte pour
  moitié) ; ``--format json`` rend les mêmes composantes.

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

   # Dix positions de référence d'Alice sur un tournoi, gardées en collection
   ./blunderdb collection suggest --db base.db --player "Alice" --tournament 4 \
       --size 10 --collection "Références : open de mars"

comment — Les commentaires d'une position
------------------------------------------

Une position porte autant de commentaires qu'on veut, chacun attribué à son
auteur (texte libre). Un commentaire reçu dans un export filigrané porte en
plus la signature de son émetteur : ``list`` l'affiche dans la colonne
*Signed by* — nom, empreinte, et ``(unverified)`` s'il ne correspond plus à ce
qui a été signé —, et la provenance dans la colonne *Origin* (vide pour un
commentaire écrit dans blunderDB).

.. code-block:: bash

   ./blunderdb comment <sous-commande> --db <chemin> [options]

**Sous-commandes:**

* ``add --position N --text T [--author A]`` — Ajoute un commentaire à la
  position N, attribué à A (sans ``--author``, il n'a pas d'auteur).
* ``list [--position N] [--author A]`` — Liste les commentaires d'une position,
  ou de toute la base, éventuellement ceux d'un seul auteur.

**Options communes:** ``--db`` (obligatoire) ; ``list`` accepte ``--format``
(``text`` ou ``json``).

**Exemples:**

.. code-block:: bash

   ./blunderdb comment add --db base.db --position 412 --text "Videau trop tôt" --author Alice
   ./blunderdb comment list --db base.db --position 412
   ./blunderdb comment list --db base.db --author Alice --format json

elsewhere — Ailleurs : une position dans d'autres bases
-------------------------------------------------------

Cherche une position, par son hachage Zobrist, dans d'autres fichiers de base
passés à la même commande, et rend pour chacun ce qu'il en sait : son analyse,
ses commentaires, les matchs où la position a été jouée (joueurs, longueur,
date, coup joué et son coût), et ses positions les plus proches (même
classement que le jeton ``like`` de la recherche, en pips de pions déplacés).

Le coût d'un coup joué ailleurs est jugé avec les seuils d'erreur et de blunder
de la base ``--db``, jamais avec ceux de la base lue : ``error``, ``blunder``,
``good``, ou ``unscored`` sans analyse. Une analyse faite à un score de match
avec une autre table d'équité que la table courante de ``--db`` est signalée ;
ses coûts en match sont affichés mais non jugés (``other-met``), comme partout
ailleurs dans blunderDB. Les autres fichiers sont ouverts en lecture seule :
rien n'y est écrit, et une base placée dans un dossier en lecture seule (partage
réseau, clé, montage) se lit telle quelle, sans créer de fichier à côté. Un fichier
illisible (absent, d'une version antérieure ou plus récente, pas une base
blunderDB) est signalé ligne ``Not read:`` sous son nom (champ ``error`` en JSON)
et sauté : les autres fichiers répondent, et la commande ne rend un code de
sortie non nul que si aucun n'a pu être lu. Un fichier d'une version antérieure est refusé (l'ouvrir
une fois dans blunderDB le met à jour), un fichier d'une version plus récente
aussi (mettre blunderDB à jour). Au-delà de 50 matchs où la position a été
jouée, seuls les 50 derniers importés sont lus : la sortie le dit, avec le
nombre total (``matchesTotal`` et ``playsTruncated`` en JSON).

.. code-block:: bash

   ./blunderdb elsewhere --db <chemin> --other <fichier> [--other <fichier>...] (--position N | --xgid X | --zobrist H) [options]

**Options:**

* ``--db`` (obligatoire) — La base qui juge : ses seuils et sa table d'équité.
* ``--other`` (obligatoire, répétable) — Un fichier de base où chercher.
* ``--position N`` — La position, par son identifiant dans ``--db``.
* ``--xgid X`` — La position, par son XGID.
* ``--zobrist H`` — La position, par son hachage (décimal) ; ses voisines ne
  sont alors cherchées que dans les bases qui la détiennent.
* ``--neighbours N`` — Nombre de voisines par base (défaut : 5, 0 : aucune,
  50 au plus).
* ``--max-distance D`` — Écarte les voisines à plus de D pips de pions
  déplacés (0 : sans plafond).
* ``--format`` — ``text`` (défaut) ou ``json``.

Le démon sert la même lecture à travers les tenants : ``across.positionElsewhere``
(voir :ref:`headless`).

**Exemples:**

.. code-block:: bash

   ./blunderdb elsewhere --db moi.db --other club.db --position 412
   ./blunderdb elsewhere --db moi.db --other a.db --other b.db --xgid "XGID=-b----E-C---eE---c-e----B-:0:0:1:52:0:0:0:0:10"
   ./blunderdb elsewhere --db moi.db --other club.db --zobrist 1234567890 --format json

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
  le geste soit annulable. ``K`` vaut ``position``, ``collection``,
  ``comment`` ou ``match``.

**Options communes:** ``--db`` (obligatoire), ``--kind``, ``--limit``
(défaut 50), ``--format`` (``text`` ou ``json``). Pour ``list``, ``--kind``
restreint la liste à un type d'objet : ``position``, ``collection``,
``comment``, ``match`` ou ``anki_card``.

.. note:: ``blunderdb delete`` supprime toujours **sans** filet : un script qui
   supprime une position s'attend à ce qu'elle disparaisse, et laisser un
   instantané en silence ferait grossir un fichier que personne n'a demandé à
   voir grossir. C'est ``trash delete`` qui garde l'annulation.

Une restauration de position repasse par la déduplication Zobrist : elle ne
crée jamais de doublon. Une position restaurée reprend son ancien numéro ; si
la même position a été enregistrée entre-temps, c'est celle-là qui est gardée,
et elle reçoit ce qui lui manque. Un match restauré reprend lui aussi son
numéro d'origine, avec ses parties, ses coups, ses analyses et ses positions ;
il est refusé si un autre match occupe ce numéro, ou si un match de même
empreinte de fichier a été importé entre-temps. Ce qu'une restauration réussie
n'a pas pu remettre — une place de Direction prise ou disparue, un tournoi
supprimé — est signalé par un avertissement.

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
