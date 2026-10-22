.. _cli_entrainement_chapitre:

Étude, leçons et entraînement
=============================

study — La file d'étude transversale
------------------------------------

Liste les blunders du joueur de référence de la base, tous imports confondus,
que rien n'a encore traités : sans commentaire, sans carte Anki, hors de toute
collection et sans marque « vu » (voir :ref:`file_etude_transversale`). Chaque
sous-commande prend ``--db``.

.. code-block:: bash

   ./blunderdb study <subcommand> [options]

**Sous-commandes:**

* ``queue [--limit <n>] [--format text|json]`` — La file, du plus coûteux au
  moins coûteux ; cinquante positions au plus.
* ``mark --id <id>`` — Marque la position vue : elle sort de la file. La marque
  est une donnée de la base, jamais exportée.
* ``unmark --id <id>`` — Retire la marque : la position revient dans la file.

**Exemples:**

.. code-block:: bash

   ./blunderdb study queue --db base.db --limit 20
   ./blunderdb study mark --db base.db --id 1234

lesson — Gérer les leçons
-------------------------

Gère les leçons : des suites ordonnées d'étapes qu'un coach écrit une fois pour
un élève. Chaque étape a un titre, un texte et peut montrer une collection, une
position, les deux ou aucune. Une leçon se remet dans un fichier ``.dbx``
(sous-commande ``export``) et se lit dans l'interface graphique avec la
commande ``:le``. Chaque sous-commande prend ``--db`` ; ``list`` et ``show``
acceptent ``--format text`` (défaut) ou ``json``.

.. code-block:: bash

   ./blunderdb lesson <subcommand> [options]

**Sous-commandes:**

* ``list`` — Liste des leçons : id, nom, nombre d'étapes.
* ``show --id <id>`` — Les étapes d'une leçon, dans l'ordre : id d'étape,
  titre, texte, collection et position montrées.
* ``create --name <nom> [--description <texte>]`` — Crée une leçon vide.
* ``edit --id <id> [--name <nom>] [--description <texte>]`` — Renomme une leçon
  ou change sa description ; ce qui n'est pas donné est conservé.
* ``delete --id <id> --confirm`` — Supprime la leçon et ses étapes,
  définitivement (il n'y a pas de corbeille pour les leçons). Les collections
  et les positions que ses étapes montraient restent dans la base.
* ``add-step --lesson <id> [--title <titre>] [--text <texte> | --text-file
  <fichier>] [--collection <id>] [--position <id>]`` — Ajoute une étape à la
  fin de la leçon.
* ``edit-step --lesson <id> --step <id> [--title …] [--text … | --text-file …]
  [--collection <id>] [--position <id>]`` — Modifie une étape ; seuls les
  champs donnés changent, ``--collection 0`` ou ``--position 0`` retirent ce
  que l'étape montrait.
* ``remove-step --lesson <id> --step <id>`` — Retire une étape.
* ``reorder --lesson <id> --steps <id,id,…>`` — Fixe l'ordre des étapes ; la
  liste donne toutes les étapes de la leçon.
* ``export --id <id[,id…]> --out <fichier> [--analysis=false]
  [--comments=false] [--watermark <texte>] [--watermark-note <texte>]
  [--password <mot de passe>]`` — Exporte une ou plusieurs leçons, avec les
  collections et les positions que leurs étapes montrent, vers un nouveau
  fichier de base ; avec ``--password``, le fichier est un ``.dbx`` protégé
  (voir la commande ``export`` pour le filigrane). Avec ``--html`` (une seule
  leçon, ``--lang fr|en|de|el|es|fi|it|ja|ru``), ``--out`` est une page HTML
  autonome : titre, description, chaque étape avec son texte et son plateau,
  l'analyse enregistrée de chaque position repliée comme une réponse de quiz,
  et l'origine du filigrane de la base en pied de page, s'il y en a un. Le
  fichier ne contient ni script ni adresse réseau.
* ``done --step <id> [--undo]`` — Marque une étape comme faite dans cette base,
  ou retire la marque avec ``--undo`` : c'est la progression de celui qui lit
  la leçon, et aucun export ne l'emporte.
* ``progress --id <id> [--format text|json]`` — Montre les étapes de la leçon
  marquées faites, avec la date du geste.

Importer un fichier qui contient une leçon crée celle-ci avec ses étapes ; une
leçon dont le nom existe déjà dans la base n'est pas touchée, si bien que
réimporter le même fichier ne change rien. Lire une leçon n'enregistre rien chez
celui qui la lit ; seul ``done`` écrit sa progression.

L'export de toute la bibliothèque (``export --type database``) emporte toutes
les leçons avec ce que leurs étapes montrent ; un export partiel ne les emporte
que nommées, par ``lesson export``.

**Exemples:**

.. code-block:: bash

   ./blunderdb lesson create --db base.db --name "Jouer contre un prime"
   ./blunderdb lesson add-step --db base.db --lesson 1 \
       --title "Le temps" --text "Comptez les pips." --collection 3
   ./blunderdb lesson show --db base.db --id 1

   # Remettre la leçon à un élève, protégée et marquée de son origine
   ./blunderdb lesson export --db base.db --id 1 --out lecon.dbx \
       --password secret --watermark "Cours du 12 mars"

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
* ``card --id <id> --action suspend|unsuspend|bury|remove|undo [--format text|json]``
  — Agit sur une carte. *Suspendre* la met de côté sans perdre son historique
  (elle ne ressort plus en séance) ; *enterrer* la masque jusqu'au lendemain,
  sans rien dire de sa valeur ; *retirer* la supprime du paquet — la position,
  elle, reste dans la base, un paquet n'étant qu'une liste d'étude posée
  dessus ; *undo* annule la dernière note de la carte, qui retrouve l'état
  d'avant cette note, et affiche cet état. Sans note à annuler, la commande
  refuse : « the card has no review to undo ».
* ``log [--deck <id>] [--limit <n>] [--format text|json]`` — Journal des
  révisions, la plus récente d'abord (``--deck 0``, le défaut, couvre tous les
  paquets ; ``--limit`` vaut 20 par défaut). Le journal est ce que le
  planificateur a réellement reçu, par opposition à ce qu'il prévoit
  aujourd'hui : c'est le seul endroit où une note entrée par erreur se voit.
* ``export --deck <id> | --collection <id> --out <fichier.apkg> [--format apkg]
  [--lang <langue>]`` — Écrit un paquet, ou une collection, en paquet Anki
  **.apkg** que l'application Anki importe sur ordinateur ou téléphone : une
  carte par position, le plateau, le score, le videau et les dés au recto, le
  meilleur coup ou la bonne décision de videau, son équité et l'erreur du coup
  joué au verso, libellés dans la langue ``--lang`` (``en`` par défaut). Une
  collection vivante est évaluée à l'export, sous son plafond. L'export n'écrit
  rien dans la base ; un réexport importé dans Anki met ses cartes à jour au
  lieu de les dupliquer. Le daemon sert le même fichier sur
  ``POST /v1/anki.exportApkg``.

Un paquet fondé sur une collection relit sa collection ; fondé sur une
collection vivante, il rejoue sa requête sous le même plafond de 5 000
positions et signale sur la sortie d'erreur une source tronquée. Un paquet fondé sur
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
   ./blunderdb anki card --db base.db --id 12 --action undo
   ./blunderdb anki log --db base.db --deck 2 --limit 50
   ./blunderdb anki export --db base.db --deck 2 --out paquet.apkg --lang fr

   # Day         Due
   # ---         ---
   # 2026-09-02  12
   # 2026-09-03  4
   # ...
   #
   # 37 card(s) due over 14 day(s)

.. _cli_training:

training — Le journal d'entraînement
------------------------------------

Relit le journal de l'onglet Entraînement (voir :ref:`panneau_entrainement`) et
fait des questions ratées de quoi réviser. Le journal est écrit par
l'application graphique, ou par un client du démon à la route
``training.save`` ; la ligne de commande ne pose pas de questions.

**training sessions** — Les séances, la plus récente d'abord, avec leur
identifiant ; le PR n'existe que pour *Décision*.

.. code-block:: bash

   ./blunderdb training sessions --db <fichier> [options]

**Options:**

* ``--exercise <nom>`` — Un seul exercice : ``scores``, ``pips``, ``bearoff``,
  ``evaluation`` ou ``decision``.
* ``--limit <n>`` — Nombre de séances (défaut 20 ; 0 = toutes).
* ``--format text|json`` — Format de sortie.

**training missed** — Les positions ratées, chacune une fois, la plus récemment
ratée d'abord ; une question hors délai compte comme ratée. Seules les
questions de *Décision* gardent leur position.

.. code-block:: bash

   ./blunderdb training missed --db <fichier> [options]

**Options:**

* ``--exercise <nom>`` — Les séances d'un seul exercice.
* ``--session <id>`` — Une seule séance (un identifiant de ``training
  sessions``).
* ``--limit <n>`` — Nombre de positions (défaut 0 = toutes).
* ``--deck <nom>`` — Crée un paquet Anki de ces positions.
* ``--collection <nom>`` — Crée une collection de ces positions.
* ``--format text|json`` — Format de sortie ; en JSON, ``PositionIDs``, et
  ``DeckID`` ou ``CollectionID`` quand ils sont créés.

**Exemples:**

.. code-block:: bash

   ./blunderdb training sessions --db base.db --exercise decision
   ./blunderdb training missed --db base.db --session 12 --deck "Ratés de lundi"
   ./blunderdb training missed --db base.db --collection "Mes ratés" --format json
