.. _cli_stats_chapitre:

Statistiques
============

.. _cli_stats:

stats — Erreurs récurrentes
---------------------------

Regroupe les erreurs d'un filtre par plan de jeu et par thème, la plus
coûteuse d'abord : le tableau *Erreurs récurrentes* de l'onglet Erreurs du
panneau Stats (voir :ref:`stats`). Les statistiques globales restent sous
``list --type stats``.

.. code-block:: bash

   ./blunderdb stats recurring --db <fichier> [options]

**Options:**

* ``--player <nom>`` — Uniquement les décisions de ce joueur.
* ``--opponent <nom>`` — Uniquement les matchs joués contre ce joueur ; avec
  ``--player``, l'autre siège.
* ``--tournament <ids>``, ``--from <AAAA-MM-JJ>``, ``--to <AAAA-MM-JJ>``,
  ``--decision-type all|checker|cube`` — Le même filtre que
  ``list --type stats``.
* ``--limit <n>`` — Nombre de groupes affichés en texte (défaut 20, ``0`` pour
  tous).
* ``--format text|json`` — Le JSON porte chaque groupe avec la liste complète
  de ses positions.
* ``--quiz`` — Tire au hasard des positions parmi celles des trois groupes
  les plus coûteux (``--quiz-size <n>``, défaut 20) et les affiche : ce sont
  les identifiants que le quiz et ``quiz_grade`` jugent. En JSON, champ
  ``Quiz``.
* ``--deck <nom>`` — Crée un paquet Anki de ce nom, rempli de toutes les
  positions des trois groupes les plus coûteux.
* ``--group <rang>`` — Avec ``--quiz`` ou ``--deck`` : le groupe de ce rang
  (1 pour le plus coûteux) au lieu des trois premiers.

Un thème de coup de pions est ``gammon``, ``blots``, ``point`` ou
``passive`` ; un thème de videau est ``offer_missed``, ``offer_premature``,
``answer_wrong_pass`` ou ``answer_wrong_take``. Les erreurs qu'aucune règle ne
nomme sortent du classement : elles sont listées à part, une ligne par plan de
jeu (champ ``Unthemed`` en JSON), parce que l'explication ne se prononce qu'à
partir de 60 mp, au-dessus du seuil *Erreur*. La colonne *COST (PR)* est la
part du PR du filtre que le groupe représente.

**Exemples:**

.. code-block:: bash

   ./blunderdb stats recurring --db base.db --player "Alice"
   ./blunderdb stats recurring --db base.db --decision-type checker --format json
   ./blunderdb stats recurring --db base.db --quiz --format json
   ./blunderdb stats recurring --db base.db --group 1 --deck "Mon pire groupe"

**stats plan** — Le plan d'étude du filtre, comme la carte *Plan d'étude* du
tableau de bord (voir :ref:`plan_etude`) : les familles d'erreurs (plan de jeu
× thème) classées par MWC récupérable, la somme sur leurs erreurs de la perte
moins la difficulté, avec un intervalle à 95 %. Une famille entre au plan à
partir de 5 erreurs chiffrées et d'un intervalle au-dessus de zéro ; les
autres sont listées à part, à confirmer (champ ``Tentative`` en JSON). Les
erreurs sans thème et celles sans MWC (partie libre) sont seulement comptées
(``Unthemed``, ``Unpriced``).

.. code-block:: bash

   ./blunderdb stats plan --db <fichier> [options]

Mêmes options de filtre que ``stats recurring``, plus :

* ``--limit <n>`` — Nombre de familles affichées en texte (défaut 10, ``0``
  pour toutes).
* ``--quiz`` (``--quiz-size <n>``, défaut 20), ``--deck <nom>`` — Comme pour
  ``stats recurring``, sur les positions des trois premières familles.
* ``--queue`` — Affiche la file d'étude de ces familles, une position par
  ligne, l'écart au joueur de référence le plus grand d'abord (JSON :
  ``Queue``).
* ``--family <rang>`` — Avec ``--quiz``, ``--deck`` ou ``--queue`` : la famille
  de ce rang (1 pour la première) au lieu des trois premières.

.. code-block:: bash

   ./blunderdb stats plan --db base.db --player "Alice"
   ./blunderdb stats plan --db base.db --player "Alice" --quiz --format json
   ./blunderdb stats plan --db base.db --family 2 --queue

**stats effect** — L'avant/après de chaque famille étudiée, comme la carte
*Avant/après l'étude* du tableau de bord (voir :ref:`avant_apres_etude`) : la
date de la première action d'étude sur l'une de ses positions, le taux de perte
en MWC pour 100 décisions de son plan et de sa nature avant et après ce jour,
le gain et son intervalle à 95 %, et le verdict (``improved``, ``worse``,
``undetermined``, ``insufficient`` sous 30 décisions par fenêtre).

.. code-block:: bash

   ./blunderdb stats effect --db <fichier> [options]

**stats biases** — Les biais signés du filtre, comme la carte *Biais signés*
(voir :ref:`biais_signes`) : prise/refus, doubles (sur l'ensemble et par
score) et blots, chacun avec ses deux comptes et leur coût, le biais, son
intervalle à 95 % et le verdict (``too_much``, ``too_little``, ``balanced``,
``insufficient`` sous 20 décisions). La partie libre a sa ligne ``money`` et un
score post-Crawford se lit à 1 point ; la part des coups de pions qui n'ont pas
pu être rejoués est donnée sous le tableau.

.. code-block:: bash

   ./blunderdb stats biases --db <fichier> [options]

Mêmes options de filtre que ``stats recurring`` (``--player``,
``--opponent``, ``--tournament``, ``--from``, ``--to``, ``--decision-type``, ``--format``).

.. code-block:: bash

   ./blunderdb stats effect --db base.db --player "Alice"
   ./blunderdb stats biases --db base.db --player "Alice" --format json

**stats tournament** — Le bilan d'un tournoi pour un joueur, comme le bouton
*Bilan* du panneau Tournois (voir :ref:`bilan du tournoi <bilan_tournoi>`) : PR et perte MWC
(éq. 7 pts) face au niveau habituel des 365 jours précédents, par ronde, par
rang de la décision dans le match, par score (``dmp``, ``crawford``,
``post_crawford``, ``other``) et par rythme (``quick``, ``considered``), chacun
avec son intervalle à 95 % et un verdict (``worse``, ``better``, ``usual``,
``insufficient`` sans intervalle, sous 20 décisions d'un côté ou sous 5 matchs
habituels), puis au plus trois familles d'erreurs du tournoi.

.. code-block:: bash

   ./blunderdb stats tournament --db <fichier> --id <n> [--player <nom>] [--format text|json]

Sans ``--player``, le joueur est le plus présent dans les matchs du tournoi.

.. code-block:: bash

   ./blunderdb stats tournament --db base.db --id 3
   ./blunderdb stats tournament --db base.db --id 3 --player "Alice" --format json

**stats training** — Le PR du quiz Décision, le PR des matchs et la rétention
Anki, repliés par fenêtre calendaire, comme l'onglet *Entraînement* du panneau
Stats (voir :ref:`stats`).

.. code-block:: bash

   ./blunderdb stats training --db <fichier> [options]

**Options:**

* ``--window week|month`` — La fenêtre calendaire (défaut ``week``).
* ``--player <nom>``, ``--tournament <ids>``, ``--from <AAAA-MM-JJ>``,
  ``--to <AAAA-MM-JJ>``, ``--decision-type all|checker|cube`` — Le filtre des
  matchs ; les journaux du quiz et d'Anki ne portent pas de joueur.
* ``--format text|json`` — Le JSON porte aussi la liste des sessions de quiz ;
  le texte et le JSON donnent le PR du quiz par plan de jeu.

Chaque série garde son nombre d'échantillons : une fenêtre sans décision y est
un tiret en texte, un compte nul en JSON, jamais un zéro.

**Exemples:**

.. code-block:: bash

   ./blunderdb stats training --db base.db --player "Alice"
   ./blunderdb stats training --db base.db --window month --format json

**stats h2h** — Deux joueurs l'un contre l'autre : les matchs qu'ils ont
disputés ensemble, le PR de chacun dans chaque match et sur l'ensemble, et le
bilan des matchs menés à terme.

.. code-block:: bash

   ./blunderdb stats h2h --db <fichier> --player <nom> --opponent <nom> [options]

**stats contrast** — Les positions que deux joueurs ont tous deux eu à jouer et
où l'un a bien joué (sous le seuil Erreur) et l'autre non, l'écart le plus large
d'abord ; un joueur est jugé sur son pire coup de la position. ``--limit N``
garde les N premières lignes du texte ; le JSON donne tout, avec les
``position_id``.

.. code-block:: bash

   ./blunderdb stats contrast --db <fichier> --player <nom> --opponent <nom> [options]

**stats windows** — Le PR sur une fenêtre calendaire glissante : une ligne par
mois, chacune couvrant ce mois et les précédents de la fenêtre
(``--window month``, ``quarter`` ou un nombre de mois).

``--by-phase`` découpe chaque fenêtre par phase de jeu (ouverture, milieu de
partie, course, sortie…) : sous chaque ligne, une ligne par phase avec ses
matchs, ses décisions et son PR, les mêmes chiffres que la ventilation par
phase de ``stats breakdown`` restreinte à la fenêtre ; une phase sans décision
comptée dans la fenêtre n'y figure pas. Le démon HTTP les rend dans
``per_phase`` quand la requête ``stats.prByWindow`` porte ``"byPhase": true``.

.. code-block:: bash

   ./blunderdb stats windows --db <fichier> [--player <nom>] [--window quarter] [--by-phase]

**stats ranking** — Le classement des joueurs par PR, le plus bas en tête,
parmi ceux qui ont au moins ``--min-decisions`` décisions comptées (500 par
défaut) ; deux PR égaux partagent un rang. ``--limit N`` garde les N premiers.

.. code-block:: bash

   ./blunderdb stats ranking --db <fichier> [--min-decisions 1000] [--limit 20]

**stats engines** — Les moteurs d'analyse présents dans la base, sous le nom
exact que ``--engine`` attend ; le même choix que le menu *Moteur* du panneau
Stats.

.. code-block:: bash

   ./blunderdb stats engines --db <fichier> [--format json]

**Options communes:** ``--tournament <ids>``, ``--from <AAAA-MM-JJ>``,
``--to <AAAA-MM-JJ>``, ``--decision-type all|checker|cube``,
``--format text|json``, et le filtre de provenance ``--engine <nom>`` (le
moteur d'analyse, nom exact) et ``--min-depth <n>`` (la profondeur minimale de
l'analyse), que ``list --type stats`` et ``list --type players`` acceptent
aussi. ``h2h`` et ``windows`` lisent les statistiques par match et refusent le
filtre de provenance, qui porte sur chaque décision.

**Exemples:**

.. code-block:: bash

   ./blunderdb stats h2h --db base.db --player "Alice" --opponent "Bob"
   ./blunderdb stats contrast --db base.db --player "Alice" --opponent "Bob"
   ./blunderdb stats windows --db base.db --player "Alice" --window quarter --format json
   ./blunderdb stats ranking --db base.db --min-decisions 1000 --limit 20
   ./blunderdb list --type stats --db base.db --player "Alice" --min-depth 3

**stats matches** — Les matchs qu'un chiffre du corpus compte : ceux où le
joueur (tout joueur sans ``--player``) a une décision comptée sous le filtre,
du plus ancien au plus récent. Leurs identifiants se passent à
``list --type matches --ids``.

.. code-block:: bash

   ./blunderdb stats matches --db base.db --player "Alice" --from 2025-01-01 --to 2025-03-31
   ./blunderdb list --type matches --db base.db --ids 12,15,31

**stats tournaments** — Les tournois que compte le nombre de tournois du
Tableau de bord : ceux qui portent une décision comptée du joueur (de tout
joueur sans ``--player``) sous le filtre, par identifiant.

.. code-block:: bash

   ./blunderdb stats tournaments --db base.db --player "Alice" --from 2025-01-01
   ./blunderdb stats tournaments --db base.db --format json

**stats progression** — Le PR de chaque match par ordre de date, le PR de
chaque tournoi et le PR glissant sur les N dernières décisions : l'onglet
*Progression* du panneau Stats (voir :ref:`stats`).

.. code-block:: bash

   ./blunderdb stats progression --db <fichier> [options]

**Options:**

* ``--player <nom>``, ``--tournament <ids>``, ``--from <AAAA-MM-JJ>``,
  ``--to <AAAA-MM-JJ>``, ``--decision-type all|checker|cube`` — Le même filtre
  que ``list --type stats``.
* ``--format text|json`` — Le JSON porte les matchs, les tournois et les PR
  glissants.

**Exemples:**

.. code-block:: bash

   ./blunderdb stats progression --db base.db --player "Alice"
   ./blunderdb stats progression --db base.db --format json

**stats breakdown** — Le PR du filtre ventilé par phase de jeu, plan de jeu,
étiquette de commentaire, score (away contre away) et action de videau, avec la
direction des erreurs de videau : l'onglet *Ventilations* du panneau Stats (voir
:ref:`stats`). Les options sont celles de ``stats progression``.

.. code-block:: bash

   ./blunderdb stats breakdown --db <fichier> [options]

**Exemples:**

.. code-block:: bash

   ./blunderdb stats breakdown --db base.db --player "Alice"
   ./blunderdb stats breakdown --db base.db --format json

**stats summary** — Le niveau du filtre sans aucun identifiant : effectifs, PR
(pions, videau) et son intervalle à 95 %, perte de MWC par match en 7 points,
PR par phase de jeu et par score, chaque ligne avec son effectif et son
intervalle. Aucune position, aucun match, tournoi, adversaire ni étiquette n'y
figure, ni les décisions les plus coûteuses : c'est ce qu'un profil public ou
une carte de visite peut montrer sans rien divulguer. Le démon HTTP le sert à
la route ``stats.summary`` ; ``stats.compute`` reste inchangé. Les options sont ``--player``,
``--tournament``, ``--from``, ``--to``, ``--decision-type`` et ``--format`` ;
il n'a pas de ``--window``.

.. code-block:: bash

   ./blunderdb stats summary --db base.db --player "Alice"
   ./blunderdb stats summary --db base.db --player "Alice" --format json

**stats report** — Le rapport HTML du panneau Stats (voir :ref:`rapport_html`) :
les indicateurs du filtre et ses dix décisions les plus coûteuses avec leur
diagramme, en un seul fichier autonome que le navigateur imprime en PDF. Le
démon HTTP le sert à la route ``stats.report``, du même générateur.

.. code-block:: bash

   ./blunderdb stats report --db <fichier> --html [options]

**Options:**

* ``--html`` — Obligatoire : le HTML est le seul format.
* ``--output <fichier>`` — Le fichier écrit (défaut : la sortie standard).
* ``--lang <code>`` — La langue du rapport : ``fr``, ``en``, ``de``, ``el``,
  ``es``, ``fi``, ``it``, ``ja`` ou ``ru`` (défaut ``en``).
* ``--player <nom>``, ``--tournament <ids>``, ``--from <AAAA-MM-JJ>``,
  ``--to <AAAA-MM-JJ>``, ``--decision-type all|checker|cube`` — Le filtre : le
  rapport nomme son périmètre.

Les diagrammes y sont dessinés avec la palette par défaut ; seule l'application
graphique emploie la palette du plateau choisie à l'écran.

**Exemples:**

.. code-block:: bash

   ./blunderdb stats report --db base.db --html --output rapport.html
   ./blunderdb stats report --db base.db --html --player "Alice" --lang fr
