.. _cli_matchs_chapitre:

Matchs, tournois et parties
===========================

match — Afficher un match
--------------------------

Affiche les positions et analyses d'un match importé. En sortie ``text`` et
``summary``, l'en-tête reprend ce que le fichier source dit du match quand il
le dit : classement Elo et expérience de chaque joueur, transcripteur, règles
Jacoby et Beaver d'une partie libre, programme qui a écrit le fichier ; la
sortie ``json`` les porte dans l'objet ``match``.

Quand le match a gardé la durée de ses décisions (un match joué contre un bot,
voir le Duel), la sortie ``text`` donne la durée de chaque décision, celle du
videau à part, et la sortie ``summary`` le total et la moyenne de chaque
joueur, avec la marque du joueur dont la réserve s'est épuisée en premier ; la
sortie ``json`` porte ``decision_ms`` et ``cube_decision_ms``. Une durée
inconnue n'est pas écrite. ``list --type timeerrors`` donne, par joueur et par
tranche de durée (moins de 5 s, 5 à 15 s, 15 à 30 s, plus de 30 s), le nombre
de décisions, l'erreur moyenne et la part de blunders ; ``--format json`` en
livre les lignes.

Pour un match joué ici par un Duel, les sorties ``text`` et ``summary``
ajoutent son origine : le départ, le germe des dés révélé et son SHA-256, à
comparer avec l'empreinte publiée à la création du Duel, la fin du match
(joué jusqu'au bout, arrêté avant la fin, ou perdu au temps et par qui), la
cadence, le joueur dont la réserve s'est épuisée en premier et le Bot (niveau,
version de gammonNet). Pour un match qui n'a pas été joué ici, elles écrivent
``Origin: not played here``. La sortie ``json`` porte l'origine sous
``origin`` (dont ``lost_on_time``), ``null`` pour un match qui n'a pas été joué
ici. Le serveur la sert par ``/v1/matches.origin`` : ``null`` pour un match
sans origine, 404 pour un match qui n'existe pas.

Pour un match analysé, la perte de chances de gagner le match de chaque coup
(``mwc_loss``, une fraction : 0,0123 vaut 1,23 %, ``null`` pour un coup non noté)
suit le même chemin : la sortie ``json`` porte la liste sous ``decision_losses``,
``text`` ajoute une ligne ``MWC loss`` aux coups notés et ``summary`` donne le total par
joueur, égal à la perte MWC de la liste des matchs. Chaque entrée porte aussi
sa difficulté (``difficulty``, la perte attendue d'un joueur de référence dans la
même position, même unité, ``null`` pour un coup non noté) et ``avoidable`` pour
une erreur évitable ; la sortie ``json`` ajoute ``difficulty_summary`` (par
joueur : ``difficulty``, ``excess``, ``ratio``, ``avoidable``), ``text`` les lignes
``Difficulty`` et ``Avoidable error``, ``summary`` une section ``Difficulty``.
Le serveur la sert par ``/v1/stats.matchDecisionLosses``.

``summary`` se termine par une section ``Review``, le bilan du match
(:ref:`bilan du match <bilan_match>`) : par joueur, le PR et son intervalle à 95 % sur les
parties, la perte MWC ramenée à 7 points et son intervalle sur les parties, le résultat ajusté de la chance (avec le résultat, la chance nette,
l'écart des erreurs et la couverture des jets), les trois erreurs à revoir
(partie, coup, perte et part évitable) et le partage des erreurs entre
précipitées et réfléchies. Le serveur le sert par ``/v1/stats.matchReview``.
Dans ``stats``, le PR global et les lignes par phase, par étiquette et par score
portent leur intervalle à 95 % sur les matchs (colonne ``95 % CI``), et
``list --type players`` une colonne ``L7``.

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

**match repair** — Réécrit les positions, les analyses et les erreurs d'un match
à partir de son fichier source, celui que l'onglet **Infos** de sa fiche nomme.
Le match garde son identifiant, son tournoi, les noms des joueurs tels qu'ils
sont enregistrés (alias, corrections, inversion), sa date, son commentaire, sa
vidéo et ses repères, sa transcription, et les commentaires écrits sur ses
positions : un commentaire laissé sur une position que la réparation quitte
passe sur celle que le fichier nomme pour le même coup, sauf si une collection,
une carte Anki ou une leçon garde encore cette position ; le commentaire y reste
et le rapport le compte. Une analyse plus profonde que celle du fichier (un
rollout lancé depuis l'import) ou corrigée à la main reste. Un match déjà
conforme à son fichier n'est pas réécrit. ``--dry-run`` dit ce qui
changerait sans rien écrire. ``--list`` passe en revue tous les matchs qui
nomment un fichier source et donne ceux qui en diffèrent, avec le nombre de
coups dont la position change et la part de coups analysés avant et après, ainsi
que ceux dont le fichier est introuvable ; il n'écrit rien. Un chemin relatif
est cherché depuis le dossier de la base, puis depuis le dossier courant. La
réparation est refusée quand le fichier est introuvable, ne contient plus le
match (son empreinte n'est plus celle du match enregistré), ou quand les coups
enregistrés ne permettent pas de dire quel joueur du fichier est lequel. Le serveur la sert
par ``/v1/matches.repair``, le fichier source envoyé dans la requête.

.. code-block:: bash

   ./blunderdb match repair --db <chemin> (--id <ID> [--dry-run] | --list) [--format text|json]

.. code-block:: bash

   ./blunderdb match repair --db base.db --list
   ./blunderdb match repair --db base.db --id 42 --dry-run
   ./blunderdb match repair --db base.db --id 42

tournament — Lire un tournoi dirigé
------------------------------------

Lit un tournoi dirigé sans interface graphique. Diriger un tournoi de façon
interactive est le rôle de la console du moteur Nicomaque ; ces
sous-commandes **lisent**, aucune n'attend de saisie, et seules ``move`` et
``confirm`` écrivent.

.. code-block:: bash

   ./blunderdb tournament <sous-commande> --db <chemin> [options]

**Sous-commandes:**

* ``list [--format text|json]`` — Les tournois dirigés de la base, avec leur
  état, la version du moteur, l'événement auquel chacun appartient (si
  aucun, la colonne est vide) et la date de la dernière décision.
* ``verify --id N [--format text|json]`` — Rejoue la direction et signale tout
  avertissement résiduel. **Sort en erreur** s'il en reste un : c'est la
  vérification d'après-tournoi, et un script qui la passe sur les bases d'une
  saison veut un code de retour, pas une ligne à filtrer.
* ``standings --id N`` — Le classement en CSV, prix compris, dans la langue de
  l'interface.
* ``ranking --season [--rencontre N] [--from AAAA-MM-JJ] [--to AAAA-MM-JJ]
  [--points 25,18,15] [--participation P] [--elo] [--format csv|json]`` — Le
  classement de saison : les tournois clos d'un événement ou d'une période
  (bornes incluses, sur la date du tournoi ; sans filtre, tous les tournois
  dirigés), chaque place convertie en points par le barème (vainqueur en tête ;
  par défaut 25, 18, 15, 12, 10, 8, 6, 4, 2, 1), plus ``--participation`` par
  tournoi joué. Des ex æquo se partagent la moyenne des places qu'ils occupent.
  Une personne est reconnue d'un tournoi à l'autre par son nom. ``--elo``
  ajoute un Elo de club rejoué sur les matchs de la saison (formule FIBS,
  départ 1500). Le CSV donne une ligne par personne et une colonne de points
  par tournoi ; un tournoi non clos est listé mais ne rapporte rien.
* ``page --id N|--rencontre N [--out <dossier>]`` — La page HTML d'affichage
  d'une épreuve (``--id``), ou la page murale d'un événement (``--rencontre`` :
  une ligne par table, quelle que soit l'épreuve qui l'occupe). Exactement l'un
  des deux est requis. Sans ``--out`` elle part sur la sortie standard ; avec,
  elle est écrite dans le dossier, qui devient celui de la direction ou de
  l'événement.
* ``export --id N`` — Le journal d'événements brut, rejouable par les outils du
  moteur. Le journal est toute la vérité d'une direction : le classement, les
  arbres et les avertissements en sont rejoués. Un outil qui lit cette sortie
  n'a besoin d'aucun blunderDB.
* ``move --id N --match M --table T [--format text|json]`` — Change la table
  d'un match en cours, comme le glisser-déposer d'une case sur une autre dans
  la grille. Si la table visée est occupée, les deux matchs **échangent** leurs
  tables ; une table hors service est refusée. Dans un événement, si la table
  est occupée par une autre épreuve, l'échange se fait entre les deux épreuves :
  un changement de table est écrit dans chaque journal. Affiche la table de
  chaque match en cours.
* ``hall --rencontre N [--format text|json]`` — Toutes les tables d'un événement : une
  ligne par table, quelle que soit l'épreuve qui l'occupe (épreuve, match,
  joueurs), puis les propositions de toutes les épreuves en une seule file
  (``proposal``), les joueurs qui attendent depuis le plus longtemps d'abord, et
  enfin les matchs retenus faute de table ou de joueur libre (``held``, avec leur
  raison). C'est la grille que la vue
  *Toutes les tables* de la Direction affiche ; les tables sont groupées par
  salle quand l'événement en a, et nommées quand elles ont un nom.
* ``proposals --id N [--format text|json]`` — La file des propositions du
  moteur, numérotée : matchs, tirages, passage de phase, repêchage d'un qualifié
  de poule retiré.
* ``confirm --id N --n K [--format text|json]`` — Confirme la proposition numéro
  K de ``proposals``, comme *Lancer* dans le panneau, puis affiche la file qui
  suit.
* ``tables --rencontre N|--tournament N [--format text|json]`` — Les propriétés
  des tables (nom, salle, réservée, attitrée à) et les salles où joue chaque
  épreuve d'un événement (``--rencontre``), ou les propriétés d'une épreuve qui
  joue seule (``--tournament``). Lecture seule : l'écriture passe par ``call``
  (``rencontres.setTables``, ``rencontres.setEventRooms``,
  ``directions.setTables``).

**Options communes:** ``--db`` (obligatoire), ``--id`` (obligatoire sauf pour
``list``, ``ranking``, ``page --rencontre``, ``hall`` et ``tables``), ``--format``.

**Exemples:**

.. code-block:: bash

   ./blunderdb tournament list --db base.db
   ./blunderdb tournament verify --db base.db --id 3
   ./blunderdb tournament standings --db base.db --id 3 > classement.csv
   ./blunderdb tournament ranking --db base.db --season --from 2026-01-01 --to 2026-12-31 --elo > saison.csv
   ./blunderdb tournament page --db base.db --id 3 --out /tmp/affichage
   ./blunderdb tournament page --db base.db --rencontre 1 --out /tmp/evenement
   ./blunderdb tournament export --db base.db --id 3 > journal.json
   ./blunderdb tournament move --db base.db --id 3 --match m4 --table 7
   ./blunderdb tournament hall --db base.db --rencontre 1
   ./blunderdb tournament tables --db base.db --rencontre 1
   ./blunderdb tournament proposals --db base.db --id 3
   ./blunderdb tournament confirm --db base.db --id 3 --n 1

.. _cli_transcribe:

transcribe — Rejouer une transcription
--------------------------------------

Rejoue une transcription et rend compte de ce que la relecture y trouve. La
source est un fichier ``.mat``, un match de la bibliothèque ou un brouillon de
transcription — exactement l'une des trois. Un match est lu par le ``.mat``
qu'il produirait à l'export : ce qui est rejoué est donc ce qu'un export
contiendrait.

.. code-block:: bash

   ./blunderdb transcribe --mat <fichier> [--check] [--render <sortie>]
   ./blunderdb transcribe --db <path> --match <id> --check
   ./blunderdb transcribe --db <path> --draft <id> --check
   ./blunderdb transcribe --db <path> --match <id> --edit [--accept-losses]
   ./blunderdb transcribe --db <path> --draft <id> --finish|--abandon
   ./blunderdb transcribe --db <path> --materialize <document.json>

**Options:**

* ``--mat`` — Fichier ``.mat`` à rejouer.
* ``--db`` — Base de données, pour ``--match`` et ``--draft``.
* ``--match`` — Identifiant du match de la bibliothèque à rejouer.
* ``--draft`` — Identifiant du brouillon de transcription à rejouer.
* ``--check`` — Liste les incohérences trouvées (comportement par défaut).
* ``--render`` — Réécrit la transcription en ``.mat`` à ce chemin.
* ``--format`` — Format de sortie: ``text`` (défaut) ou ``json``.
* ``--edit`` — Ouvre un brouillon sur le ``--match`` (ou rend celui qui y est
  déjà ouvert).
* ``--accept-losses`` — Avec ``--edit`` sur un match importé : accepte que ses
  analyses et commentaires puissent être perdus.
* ``--finish`` — Termine le ``--draft`` : écrit son match, ou remplace celui
  dont il a été ouvert, et libère le brouillon.
* ``--abandon`` — Abandonne le ``--draft`` : le supprime sans match ; un match
  dont il a été ouvert reste tel quel.
* ``--yes`` — Avec ``--abandon`` sur un brouillon jamais terminé : confirme que
  tout ce qui y est écrit est perdu.
* ``--materialize`` — Document JSON (``header`` et ``actions``, dés compris) à
  écrire en match d'un seul coup, ou pas du tout.

``--check`` nomme chaque incohérence avec le numéro de l'action et la partie où
elle se trouve : coup illégal, deux tours de suite pour le même joueur, action
de videau impossible, action au-delà de la fin du match, coup dont les pas
n'utilisent pas ses propres dés, premier coup d'une partie joué sur un double,
qu'aucun jet d'ouverture ne peut être, coup non consigné — la cellule ``???`` que
gnubg écrit quand il n'a pas gardé le coup joué, et qui n'est pas une danse —,
score annoncé incohérent — une partie dont la ligne de score n'est pas celle
que donnent les parties précédentes, rejouée au score écrit.

Une incohérence est **rapportée, jamais opposée** : rien n'est refusé pour elle
et le code de sortie reste 0 quoi que la relecture trouve. Un code non nul
signale un vrai échec — fichier illisible, base qui ne s'ouvre pas, sortie
impossible à écrire. Un script qui veut agir sur les constats les lit dans
``--format json``, où un fichier cassé et une partie contenant un coup illégal
ne se confondent pas.

``--render`` réécrit la transcription en ``.mat``, ce qui permet de vérifier
l'aller-retour sur un fichier réel, en dehors des tests.

``--materialize`` écrit, sans brouillon, le match que joue un document
d'actions arrivé d'ailleurs. Les règles sont appliquées strictement : la
première action refusée fait échouer la commande avec son rang (à partir de 0)
et son motif, et rien n'est écrit. Les durées de décision du document
(``decision_ms``, ``cube_decision_ms``) sont reportées sur les coups.

Seules trois autres options écrivent, par les mêmes méthodes que le panneau
Transcription : ``--edit`` ouvre un brouillon sur un match existant,
``--finish`` le termine — le match est remplacé sous le même identifiant —, et
``--abandon`` supprime un brouillon sans match, et exige ``--yes`` pour un
brouillon jamais terminé, qui emporte tout ce qui y est écrit. Un match importé
porte des analyses et des commentaires qu'un ``.mat`` ne porte pas : ``--edit``
en donne le compte au plus et refuse sans ``--accept-losses``.

**Exemple:**

.. code-block:: bash

   ./blunderdb transcribe --mat match.mat --check

   # match.mat: 7 point match, 4 game(s), 203 action(s)
   #   Final score: 9-2
   # Inconsistencies: none

.. _cli_duel:

duel — Jouer un Duel
--------------------

Joue un Duel sans interface, une Action par appel. Chaque appel est son propre
processus : il reprend le Duel dans la base, joue une Action et la réécrit. Les
durées des coups d'un Duel joué ainsi sont inconnues, jamais nulles.

.. code-block:: bash

   ./blunderdb duel create --db <path> (--length <n> | --money [--jacoby]) [--side1 <côté>] [--side2 <côté>]
   ./blunderdb duel create --db <path> --length <n> [--start <XGID>] [--away <j1>,<j2>] [--reroll] [--after-cube] [--single-game] [--minutes-per-point <min> --delay <s>]
   ./blunderdb duel show --db <path> --id <id>
   ./blunderdb duel list --db <path>
   ./blunderdb duel move --db <path> --id <id> --play "24/18 13/11"
   ./blunderdb duel roll|double|take|pass|resign --db <path> --id <id>
   ./blunderdb duel forfeit --db <path> --id <id> --side 1|2
   ./blunderdb duel contribute --db <path> --id <id> --side 1|2 --value <texte>
   ./blunderdb duel stop|discard --db <path> --id <id>

**Sous-commandes :**

* ``create`` — Démarre un Duel et joue jusqu'à la première décision d'un Côté
  externe. ``--length`` (1 à 25 points) ou ``--money`` ; ``--start`` (XGID du
  Départ) ; ``--name1`` et ``--name2`` ; ``--side1`` et ``--side2``
  (``external``, ou ``bot:<niveau>`` avec ``instant``, ``normal`` ou
  ``thorough``, ou ``external:<configuration>@<moteur>``, un Côté externe qui
  déclare le Bot qui joue derrière lui : l'origine du Match l'enregistre comme
  déclaré, non attesté) ; ``--discard-at-end`` jette le brouillon à la fin au lieu
  d'écrire le Match ; ``--combined-seed`` ne lance aucun dé avant l'apport de
  chaque Côté externe au germe. ``--away`` donne le score de la première
  partie en points manquants, ``joueur1,joueur2`` (``1`` : la partie Crawford,
  ``0`` : un point après elle), et remplace celui de ``--start`` ; sans
  ``--start``, la partie part de la position initiale à ce score.
  ``--reroll`` fait tirer à nouveau le jet que porte ``--start`` ;
  ``--after-cube`` commence après la décision de videau du Départ (le joueur
  au trait lance aussitôt, un double offert est pris) — par défaut le Duel
  commence avant elle, et un double offert attend la réponse de
  l'adversaire. ``--single-game`` arrête le Duel à la fin de sa première
  partie, écrite seule comme Match. ``--minutes-per-point`` et ``--delay``
  fixent la cadence : la réserve de chaque joueur en minutes par point du
  match (Standard 2, Speed 0,4) et le délai par coup en secondes (12, 10) ;
  ``--time-out`` (``continue`` ou ``lose_match``) ce que fait une réserve
  épuisée. Un Départ que les règles n'admettent pas (pions, score hors de la
  longueur, deux joueurs en partie Crawford, videau) est refusé avec son motif.
  Avec deux Bots, le match se joue en entier dans cet appel
  (``--side1 bot:instant --side2 bot:instant``) ; une session en argent entre
  deux Bots est refusée, car elle ne finirait jamais, sauf en une seule partie.
* ``show`` — Score, ce que le Duel attend et, pour un coup, les jeux légaux. Le
  germe des dés n'est jamais affiché avant la fin : l'empreinte le représente.
* ``list`` — Les Duels en suspens.
* ``roll``, ``move``, ``double``, ``take``, ``pass``, ``resign`` — Une Action du
  Côté que le Duel attend (``--side 1|2`` pour le nommer) ; ``--play`` donne le
  coup en notation, dans l'ordre qu'on veut ; ``--level`` (1 à 3) la valeur d'un
  abandon ; ``--revision`` refuse l'Action si le Duel a bougé.
* ``contribute`` — L'apport d'un Côté externe au germe combiné (``--side 1|2``,
  ``--value``, 1 à 64 octets), une seule fois, avant le premier lancer ; le
  dernier apport lance les dés. Les lancers sortent alors du HMAC-SHA256 du
  germe scellé sur l'apport de chaque Côté précédé de sa longueur, et l'origine
  du Match porte les apports.
* ``forfeit`` — Le Côté ``--side`` (requis) abandonne le match : la partie en
  cours va à l'autre Côté pour les points qui le portent à la longueur (en
  argent, une simple à la valeur du videau), et le Match s'écrit gagné par
  lui.
* ``stop`` — Arrête une session en argent et écrit son Match tel qu'il est,
  la partie en cours sans vainqueur. Un match en points est refusé : il ne
  s'écrit qu'entier (``forfeit``, ``discard``). Arrêter n'est pas abandonner.
* ``discard`` — Jette le Duel : rien n'en est écrit.

Toutes prennent ``--db`` et ``--format`` (``text`` ou ``json``).
