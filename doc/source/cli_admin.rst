.. _cli_admin_chapitre:

Administration de la base
=========================

info — Métadonnées de la base
------------------------------

Affiche les métadonnées et les statistiques d'une base de données.

.. code-block:: bash

   ./blunderdb info --db <path> [--format <format>] [--estimate]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--format`` — Format de sortie: ``text`` ou ``json`` (défaut: ``text``).
* ``--estimate`` — Répond tout de suite sur une très grande base : au-delà de
  200 000 lignes, un nombre est une estimation (borne haute), listée dans
  ``approximate`` en JSON et précédée de « ≈ » en texte, et les blunders ne
  sont donnés que si leur décompte gardé en cache vaut encore (sinon pas de
  ``blunder_count``). Par défaut, toutes les lignes sont comptées ; le
  décompte des blunders est gardé en cache dans la base et refait après toute
  écriture qui peut le changer (import, analyse, suppression, changement de
  seuil).

**Exemples:**

.. code-block:: bash

   ./blunderdb info --db base.db

   # Database Information
   # ==================================================
   # Path: /home/jean/bg/base.db
   #
   # Metadata:
   #   Version: 2.31.0
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
   #   Blunders: 142
   #
   # Thresholds (millipoints):
   #   Error: 50
   #   Blunder: 100

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
     "library_settings": {
       "errorThresholdMP": 50,
       "blunderThresholdMP": 100
     },
     "metadata": {
       "database_version": "2.31.0",
       "dateOfCreation": "2026-09-06 02:43:51",
       "description": "Matchs de tournoi 2025",
       "user": "Jean"
     },
     "path": "/home/jean/bg/base.db",
     "stats": {
       "analysis_count": 3855,
       "blunder_count": 142,
       "game_count": 61,
       "match_count": 11,
       "move_count": 3766,
       "position_count": 3859
     }
   }

edit — Modifier les métadonnées
--------------------------------

Modifie le nom d'utilisateur, la description ou les seuils d'une base de
données.

.. code-block:: bash

   ./blunderdb edit --db <path> [options]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--user`` — Nouveau nom d'utilisateur.
* ``--description`` — Nouvelle description.
* ``--clear-user`` — Effacer le nom d'utilisateur.
* ``--clear-description`` — Effacer la description.
* ``--error-threshold`` — Seuil d'erreur, en millipoints : une décision
  coûtant au moins cela est une erreur.
* ``--blunder-threshold`` — Seuil de blunder, en millipoints : une erreur
  coûtant au moins cela est un blunder.
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``
  (``{"changes": [...]}``).

Au moins une option de modification est requise.

**Exemples:**

.. code-block:: bash

   ./blunderdb edit --db base.db --user "Marie" --description "Ma collection"
   ./blunderdb edit --db base.db --clear-description
   ./blunderdb edit --db base.db --error-threshold 20 --blunder-threshold 80

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
ou sans position, questions d'entraînement sans séance — et affiche une ligne ``WARNING`` avec le total s'il y en a. Une base saine répond ``Orphaned rows: none``. Des orphelins
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
       "blunder_count": 142,
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
       "reviews_without_position": 0,
       "training_items_without_session": 0
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
affichée avant compactage soit honnête, vérifie qu'il reste à côté du fichier
environ sa taille en espace libre, écrit la base compactée dans une copie
(``VACUUM INTO``) qui remplace ensuite le fichier, puis lance un ``ANALYZE``
pour rafraîchir les statistiques utilisées par le planificateur de requêtes.
Si un autre programme a la base ouverte, le fichier n'est pas remplacé : la
commande compacte sur place (``VACUUM``), ce qui demande environ deux fois la
taille du fichier en espace libre. Si l'espace disque manque, la commande
refuse de démarrer avec un message explicite plutôt que de risquer un
compactage interrompu.

Avant le compactage, chaque analyse encore rangée en JSON par une version
antérieure est réécrite au format binaire compact (voir ``reencode``).

**Exemple:**

.. code-block:: bash

   ./blunderdb vacuum --db base.db

   # Compacting database...
   #   Before: 128.4 MiB
   #   After:  41.2 MiB
   #   Reclaimed: 87.2 MiB

met — La table d'équité de match de la base
--------------------------------------------

Liste, importe ou choisit la table d'équité de match (MET) avec laquelle
gammonNet valorise les scores de match de la base : Kazaross-XG2, intégrée,
par défaut, ou une table explicite au format ``.xml`` de GNUbg. Chaque analyse
calculée enregistre sa table ; une analyse à un score de match calculée avec
une autre table que la table courante est sortie des statistiques. Changer de
table ne réécrit aucune analyse.

.. code-block:: bash

   ./blunderdb met --db <path> [--import <fichier.xml> [--current] | --use <id>]

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--import`` — Importe une table ``.xml`` de GNUbg. Une table déjà présente,
  ou identique à Kazaross-XG2, n'est pas ajoutée une seconde fois.
* ``--current`` — Avec ``--import`` : rend la table importée courante.
* ``--use`` — Rend courante la table d'identifiant donné ; ``0`` revient à
  Kazaross-XG2.
* ``--format`` — Format de sortie : ``text`` (défaut) ou ``json``.

La commande affiche ensuite les tables de la base, la courante marquée d'un
``*``. Le démon expose les mêmes opérations sur les routes ``met.list``,
``met.import``, ``met.setCurrent`` et ``met.ofAnalysis``, limitées au tenant
de l'appelant.

**Exemple:**

.. code-block:: bash

   ./blunderdb met --db base.db --import Rockwell-Kazaross.xml --current

   #     0  Kazaross-XG2
   # *   1  Rockwell/Kazaross 25 point MET

reencode — Réécrire les analyses au format compact
--------------------------------------------------

Les analyses sont rangées dans un format binaire compact, environ deux fois
plus petit que le JSON compressé des versions antérieures et plusieurs fois
plus rapide à relire. Une base créée avant ce format reste lisible telle
quelle : ses anciennes analyses sont converties quand elles sont réécrites (un
import qui les enrichit, un ``vacuum``). ``reencode`` les convertit toutes,
sans compacter le fichier : utile quand le disque n'a pas la place de la
copie compactée qu'écrit un ``vacuum``, ou sur un serveur PostgreSQL, qui n'a
pas de ``vacuum``.

.. code-block:: bash

   ./blunderdb reencode --db <path>

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``
  (``{"rewritten"}``, le nombre d'analyses réécrites).

La conversion avance par lots de 2 000 analyses, chacun dans sa propre
transaction. Interrompue, elle reprend au lancement suivant là où elle s'était
arrêtée : les analyses déjà converties sont sautées sans être relues. Une
analyse illisible est laissée telle quelle et signalée dans le journal. Elle
ne se lance jamais d'elle-même. Le démon expose la même opération sur la route
``maintenance.reencode``, limitée au tenant de l'appelant.

**Exemple:**

.. code-block:: bash

   ./blunderdb reencode --db base.db

   #   Analyses rewritten: 15623468

maintenance — L'état de santé de la base
----------------------------------------

Une ligne par opération de maintenance : ce qu'elle compte, son action et sa
dernière exécution. Rien ne se lance de soi-même : ni à l'ouverture de la base,
ni en lisant l'état.

.. code-block:: bash

   ./blunderdb maintenance status --db <path>
   ./blunderdb maintenance count <opération> --db <path>
   ./blunderdb maintenance ids <opération> --db <path>
   ./blunderdb maintenance run <opération> --db <path>

**Sous-commandes:**

* ``status`` — Toutes les opérations, avec les compteurs gratuits et la
  dernière exécution de chacune. Un compteur coûteux affiche
  ``(count on demand)`` : ``count`` le calcule.
* ``count`` — Calcule le compteur d'une opération, quel qu'en soit le coût.
* ``ids`` — Liste les identifiants que l'opération compte (positions ou
  matchs) ; leur nombre est la réponse de ``count``.
* ``run`` — Lance l'action d'une opération et l'inscrit comme sa dernière
  exécution (terminée, annulée ou en échec). Ctrl-C l'arrête entre deux lots.

**Opérations:**

.. csv-table::
   :header: "Opération", "Compteur", "Action"
   :widths: 22, 40, 38

   "``analysis.missing``", "Positions sans aucune analyse, à la demande.", "La commande ``analyze``."
   "``stats.matches``", "Matchs sans statistiques par match.", "Les calcule."
   "``analysis.columns``", "—", "Recalcule les colonnes tirées des analyses."
   "``moves.errors``", "—", "Note l'erreur des coups joués qui n'en ont pas."
   "``analysis.legacy``", "—", "Réécrit les analyses au format compact (``reencode``)."
   "``trash``", "Entrées de la corbeille.", "La commande ``trash``."
   "``vacuum``", "Octets libres du fichier (SQLite).", "La commande ``vacuum``."

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``.

La dernière exécution est rangée dans la base, avec ses réglages ; un export
ne l'emporte pas. Une seule opération tourne à la fois. Le démon expose les
mêmes sous-commandes sur les routes ``maintenance.status``,
``maintenance.count``, ``maintenance.ids``, ``maintenance.run`` (en flux
NDJSON, annulable par ``maintenance.cancel``), limitées au tenant de
l'appelant ; une opération demandée pendant qu'une analyse gammonNet ou un
rollout tourne pour ce tenant est refusée.

**Exemple:**

.. code-block:: bash

   ./blunderdb maintenance run stats.matches --db base.db

   #   stats.matches: done, 1204 changed

players — Les graphies d'un joueur
----------------------------------

Une même personne signe parfois de plusieurs façons d'un fichier à l'autre
(« Doe J. », « John Doe »). Un **alias** dit qu'un nom est une autre graphie
d'un nom canonique. Les matchs déjà en base gardent les noms que leurs fichiers
écrivent ; les statistiques, la table Joueurs et la recherche ``pl"…"`` lisent
toutes les graphies comme une seule personne, et chaque import suivant enregistre
le nom canonique. La fusion de joueurs de la fenêtre des matchs crée les mêmes
alias.

.. code-block:: bash

   ./blunderdb players alias <add|list|remove|suggest> --db <chemin> [arguments]

**Actions:**

* ``add ALIAS CANONIQUE`` — ``ALIAS`` devient une graphie de ``CANONIQUE``. Un
  canonique qui est lui-même un alias est suivi jusqu'à son propre canonique, et
  les alias qui visaient ``ALIAS`` passent à ``CANONIQUE`` : un seul niveau,
  jamais de chaîne.
* ``list [--format text|json]`` — Les alias, rangés par nom canonique.
* ``remove ALIAS`` — Oublie un alias ; sort en erreur si ce n'en était pas un.
* ``suggest [--format text|json]`` — Propose les noms qui ne diffèrent que par la
  casse, les accents, la ponctuation ou l'ordre des mots, avec pour canonique la
  graphie la plus fréquente. Rien n'est enregistré : chaque proposition s'écrit
  comme la commande ``add`` qui l'appliquerait.

**Empreintes.** Les empreintes d'un match (``match_hash``, ``canonical_hash``)
gardent les noms du fichier : l'alias s'applique après elles, aux seuls noms
enregistrés. Un fichier importé avant que son alias existe se reconnaît donc
toujours à l'import suivant. Un match importé sous une autre graphie, dont les
dés sont ceux d'un match en base et dont les noms ne diffèrent que par des alias
connus, est reconnu comme le même match : ses analyses enrichissent l'existant,
aucun second match n'est créé.

.. code-block:: bash

   ./blunderdb players alias add --db base.db "Doe J." "John Doe"
   ./blunderdb players alias suggest --db base.db

**players merge** — La fusion de joueurs de la fenêtre des matchs, du même code
que la route ``matches.mergePlayers`` du démon : chaque nom devient une graphie
du nom canonique, comme autant de ``players alias add``.

.. code-block:: bash

   ./blunderdb players merge --db <chemin> --into <CANONIQUE> <NOM> [<NOM>...]

**players swap** — L'inversion des deux joueurs d'un match, que le fichier avait
nommés dans le mauvais ordre (route ``matches.swapPlayers``). Aucune position ni
aucune analyse ne change : chaque joueur garde ses statistiques, sous l'autre siège.
L'inversion est refusée tant qu'un brouillon de transcription est ouvert sur le
match, et pour un match joué dans blunderDB.

.. code-block:: bash

   ./blunderdb players swap --db <chemin> <ID_MATCH>

**players main** — Le joueur principal de la fenêtre des matchs (routes
``players.main.*``). Sans nom, affiche le joueur principal ; avec un nom, le
désigne et inverse en une seule transaction tous les matchs où lui ou l'un de ses
alias est joueur 2 ; chaque import ultérieur le place en joueur 1. ``--dry-run``
affiche les matchs à inverser sans rien changer ; ``--clear`` retire le joueur
principal sans rien inverser en retour. Un match où un brouillon de transcription
est ouvert, ou joué dans blunderDB, est laissé tel quel et listé à part.

.. code-block:: bash

   ./blunderdb players main --db <chemin> [--dry-run] [<NOM>]
   ./blunderdb players main --db <chemin> --clear

**players matches** — Les identifiants des matchs d'un joueur, à l'un ou l'autre
siège et sous tous ses alias (route ``players.matchIds``).

.. code-block:: bash

   ./blunderdb players matches --db <chemin> <NOM>

.. code-block:: bash

   ./blunderdb players merge --db base.db --into "John Doe" "Doe J." "J. Doe"
   ./blunderdb players swap --db base.db 42
   ./blunderdb players main --db base.db --dry-run "John Doe"

events — Les graphies d'un événement
------------------------------------

Les mêmes actions que ``players alias``, pour les noms d'événement (le champ
Event d'un fichier). À l'import, un match dont l'événement est un alias est rangé
dans le tournoi du nom canonique.

.. code-block:: bash

   ./blunderdb events alias add --db base.db "Open 2025" "Open d'automne 2025"
   ./blunderdb events alias list --db base.db

.. _cli_repair:

repair — Recalculer ce qui est dérivé
--------------------------------------

Recalcule ce que la base tire de ce qu'elle stocke : les colonnes scalaires de
chaque analyse à partir de l'analyse elle-même, dont elles ne sont qu'une
projection ; la phase et le type de jeu de chaque position à partir de son
damier ; et la sentinelle de Crawford de chaque score à partir du match d'où la
position vient, ou de l'XGID avec lequel elle est entrée. Les analyses ne sont pas touchées : ce sont les valeurs qu'on en
avait tirées qui sont refaites.

Une base qui a reçu des analyses par l'import d'une autre base (``import --type
database``, y compris par un dossier de synchronisation surveillé) avec une
version antérieure à celle-ci garde ces analyses sans leurs colonnes, ou avec
celles de l'analyse qu'elles ont remplacée : leurs blunders ne sont pas comptés
et leurs coups forcés passent pour des décisions. ``repair`` compare ces colonnes
à ce que l'analyse stockée en donne et refait celles qui diffèrent ; le même
contenu compte alors comme s'il était venu par ``transfer``.

.. code-block:: bash

   ./blunderdb repair --db <path>

**Options:**

* ``--db`` — Base de données (obligatoire).
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json`` — un compteur
  par passe : ``repaired`` (colonnes d'analyse), ``phases`` (positions
  reclassées), ``crawford`` (positions réhachées) et ``signatures_unverified``
  (commentaires signés qu'un réhachage laisse non vérifiés : une signature
  couvre l'identité de la position). Chacun dit le nombre de lignes réellement
  changées. Avec ``--stats``, s'y ajoute ``match_stats``
  (matchs recalculés) ; avec ``--move-errors``, ``move_errors`` (coups notés).
* ``--stats`` — Recalcule aussi, de zéro, les statistiques par match.
* ``--move-errors`` — Écrit aussi, pour chaque coup joué, l'équité qu'il a
  cédée face au meilleur coup, lue dans l'analyse de sa position (colonne
  ``error_mp`` de la table des coups). La passe est reprenable : interrompue,
  elle reprend où elle s'était arrêtée ; sur une très grande base elle dure
  plusieurs minutes. Côté serveur : ``matches.scoreMoves``.
* ``--duplicates`` — Ne lance aucune des passes : liste les paires de matchs
  que les dés disent identiques, sans rien fusionner (voir ci-dessous).

Les statistiques par match sont, pour chaque joueur de chaque match, ses
décisions, son erreur cumulée, son PR, ses blunders, sa chance et la provenance
dominante de ses analyses ; les statistiques les lisent au lieu de reparcourir
chaque décision. Elles s'écrivent avec le match à l'import et s'effacent à
chaque changement qui les altère (match supprimé, joueurs permutés, analyse
d'une de ses positions modifiée, seuil de blunder déplacé) ; la lecture suivante
recalcule ce qui manque. Une base migrée d'une version antérieure les calcule
une fois, à sa première ouverture. ``--stats`` les refait toutes : le recours si
cette tenue de comptes se trompait un jour.

**Doublons probables.** ``repair --duplicates`` compare les matchs par leurs
dés, jamais par leurs noms. Deux cas sont signalés : la même longueur, le même
score initial et les mêmes dés dans chaque partie sous d'autres noms de
joueurs, et un match dont les dés prolongent ceux d'un autre — un match tronqué
puis complété. Sous une paire aux mêmes dés, la sortie texte donne les
commandes ``players alias add`` qui feraient nommer aux deux matchs les mêmes
joueurs : une seule lecture quand un nom est commun aux deux (l'autre nom ne
peut être que l'autre joueur), sinon deux — siège pour siège, puis croisée —
numérotées, dont une seule est à lancer. La graphie du match le plus récent
devient l'alias, celle du plus ancien le nom canonique. ``--format json`` rend
``{"suspects": [...]}``, chaque paire avec ``kind`` (``same_dice`` ou
``longer``), ``matchId``, ``otherId``, ``players``, ``otherPlayers`` et, pour
une paire aux mêmes dés, ``pairings`` : la liste des lectures, chacune avec ses
``aliases`` (``alias``, ``canonical``). Un match importé avant que l'empreinte des dés
existe la reçoit au passage ; c'est la seule écriture de ce mode.

Utile après une correction de la façon dont une analyse importée est lue. Le
cas s'est déjà produit deux fois. L'importeur XG écrit un « pas de double » de
deux façons, et la seconde était comprise comme un vrai double — la colonne
portait alors l'erreur d'un double qui n'avait jamais eu lieu. Et une analyse
que blunderDB avait calculée lui-même ne savait pas quel coup avait été joué,
si bien qu'un match importé sans analyse gardait une erreur nulle partout et
un PR de 0,00 ; la colonne se recalcule à partir des coups du
match. Corriger la lecture ne change rien aux lignes déjà écrites ; cette
commande les refait.

La passe Crawford, elle, touche les positions elles-mêmes. Un score à ``1``
signifie « un point à faire, et cette partie EST la Crawford » ; ``0`` signifie
« un point à faire, la Crawford est derrière ». Tant que les importeurs n'ont
pas écrit cette distinction, toute position post-Crawford a été enregistrée
comme une position de Crawford, donc lue videau mort — là où le poursuivant
double en réalité à la première occasion. Corriger le score change le hachage
de la position : la ligne est donc réhachée, et fusionnée avec sa jumelle
correcte si la base en tient déjà une — l'analyse, les commentaires, les
collections, les cartes Anki et leur journal de révision, les coups du match et
les entrées de la corbeille qui la nomment suivent la ligne survivante. Une
position qu'aucun match ne désigne n'est corrigée que sur la parole de l'XGID
qu'elle a apporté d'un autre logiciel (XG, BGBlitz…) : quand le champ Crawford
de cet XGID dit que la partie n'est pas la Crawford, et que l'XGID décrit bien
cette position. Dans l'autre sens, une position sans match enregistrée à ``0``
des deux côtés passe à ``1`` quand l'XGID qu'elle a apporté est celui d'un match
en 1 point et qu'il la décrit : la seule partie d'un match en 1 point commence à
un point du but, c'est donc la Crawford, comme l'écrivent les importeurs. Le DMP
d'après la Crawford d'un match plus long reste à ``0`` : son XGID donne la
longueur de ce match. Un XGID que blunderDB a lui-même réécrit ne fait que
répéter le score enregistré et ne prouve rien. Toute autre position sans match
est laissée telle quelle : rien ne contredit ce que son score annonce.

Rien ne la déclenche automatiquement, et c'est voulu : réécrire les colonnes
d'analyse de tout le monde, ou réhacher des positions, à la simple ouverture
d'une base n'est pas quelque chose qu'un outil doit faire dans le dos de son
utilisateur.

**Exemple:**

.. code-block:: bash

   ./blunderdb repair --db base.db

   # 42 analyses repaired.
   # 7 positions reclassified.
   # 3 positions rehashed onto the right Crawford sentinel.

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

mcp — Offrir la base à un assistant IA
--------------------------------------

Sert les outils de la base à un assistant IA par le *Model Context Protocol*,
sur l'entrée et la sortie standard : c'est l'assistant qui lance la commande.
Les outils recherchent des positions dans la grammaire de la barre de
commande, lisent une position et son analyse, expliquent une erreur,
calculent les statistiques d'un joueur, listent matchs, tournois et
collections, et posent un quiz. Ils ne font que lire, sauf avec ``--write``.
La liste complète et l'équivalent HTTP du démon : :ref:`headless_mcp`.

.. code-block:: bash

   ./blunderdb mcp --db base.db [--write]

**Options:**

* ``--db`` — Fichier de base de données (obligatoire).
* ``--write`` — Offre aussi les outils qui écrivent : enregistrer une
  position, la commenter, créer et remplir une collection. Rien n'efface.

Comme ``call``, la commande migre le schéma d'une base ancienne à
l'ouverture, même sans ``--write``.

**Exemple:** déclarer la base à Claude Code.

.. code-block:: bash

   claude mcp add blunderdb -- blunderdb mcp --db /chemin/vers/base.db

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
   # blunderDB version 0.36.0 (database schema 2.31.0)
