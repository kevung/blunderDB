.. _headless:

============================
Mode headless (serveur)
============================

.. note::

   Cette section décrit un **mode avancé et facultatif** de blunderDB,
   destiné aux déploiements sur serveur, au multi-utilisateur et à
   l'automatisation. **L'usage normal et recommandé de blunderDB reste
   l'application de bureau** décrite dans les chapitres précédents. Si vous
   utilisez blunderDB seul, sur votre ordinateur, vous n'avez pas besoin de ce
   mode : vous pouvez ignorer ce chapitre sans rien perdre des fonctionnalités
   d'analyse.

Vue d'ensemble
==============

Le même binaire ``blunderdb`` peut, en plus de l'application de bureau et des
commandes en ligne (voir :ref:`cli`), fonctionner en **mode headless** :
sans interface graphique, piloté entièrement en ligne de commande ou par le
réseau. Ce mode regroupe trois usages :

* **le démon** ``serve`` — expose le moteur de blunderDB comme un service
  HTTP + JSON, pour faire tourner une base partagée sur un serveur et y
  accéder à plusieurs ;
* le dispatcher générique ``call`` — appelle n'importe quelle opération de
  stockage directement, en local, pour le scripting et les tests ;
* la commande ``migrate`` — transfère une base SQLite mono-utilisateur vers
  un backend PostgreSQL multi-utilisateur.

Ces trois usages s'appuient sur une **couche de stockage** commune qui sait
parler à deux backends : **SQLite** (le format de fichier ``.db`` habituel de
l'application de bureau) et **PostgreSQL** (pour les déploiements serveur
multi-utilisateurs).

.. _headless_serve:

Le démon ``serve``
==================

``blunderdb serve`` lance le moteur comme un service HTTP qui répond en JSON.
Il permet d'héberger une base de positions sur une machine et d'y accéder
depuis plusieurs clients.

.. code-block:: bash

   # sqlite
   blunderdb serve --db database.db --addr 127.0.0.1:8080

   # postgres
   blunderdb serve --backend postgres \
       --dsn "postgres://user:pass@host:5432/blunderdb?sslmode=disable" \
       --addr 127.0.0.1:8080

.. note::

   ``sslmode=disable`` ne convient qu'à un réseau privé de confiance — une base
   dans un conteneur voisin, sur un réseau qui n'a de route ni vers l'hôte ni
   vers l'Internet. Pour une base distante, ``sslmode=require`` chiffre la
   liaison et ``verify-full`` vérifie en plus le certificat du serveur et son
   nom d'hôte. Les autres chaînes de connexion de cette page portent
   ``sslmode=disable`` pour la même raison : elles décrivent toutes un réseau
   privé.

.. warning::

   **Le démon n'effectue aucune authentification.** Il fait confiance à
   l'en-tête de requête ``X-Tenant-ID`` et **doit** tourner derrière un
   reverse-proxy (nginx, Caddy…) chargé de l'authentification. **Ne l'exposez
   jamais directement sur l'Internet public.**

   ``X-Tenant-ID`` est l'**entier** du tenant (``1``, ``2``, ``42``…) : c'est
   au reverse-proxy de faire correspondre le compte authentifié à cet entier.
   Un nom (``alice``) est refusé avec ``400 invalid``, jamais converti.

**Options:**

.. list-table::
   :header-rows: 1
   :widths: 22 12 40

   * - Option
     - Défaut
     - Signification
   * - ``--db <chemin>``
     - –
     - fichier SQLite (raccourci pour ``--backend sqlite --dsn <chemin>``)
   * - ``--backend <type>``
     - ``sqlite``
     - backend de stockage : ``sqlite`` ou ``postgres``
   * - ``--dsn <chaîne>``
     - ``$BLUNDERDB_DSN``
     - chaîne de connexion du backend
   * - ``--addr <hôte:port>``
     - ``:8080``
     - adresse d'écoute
   * - ``--log-level <niveau>``
     - ``info``
     - niveau de journalisation : ``debug|info|warn|error``
   * - ``--metrics``
     - ``true``
     - expose ``/metrics`` (format Prometheus)
   * - ``--web``
     - ``false``
     - sert la page web de consultation sous ``/app/`` ; **éteinte par
       défaut**, voir plus bas
   * - ``--direction``
     - ``false``
     - sert les gestes de direction de tournoi et d'événement ; **éteints
       par défaut**, voir :ref:`headless_direction_gestures`
   * - ``--mcp-write``
     - ``false``
     - offre les outils d'écriture de ``/mcp`` ; **éteints par défaut**,
       voir :ref:`headless_mcp`
   * - ``--transcription``
     - ``false``
     - sert les gestes de transcription (``transcriptions.create``,
       ``apply``, ``finish``…) ; **éteints par défaut**, voir
       :ref:`headless_transcription`
   * - ``--transcription-ttl <durée>``
     - ``30m``
     - ferme une session de transcription inactive depuis plus longtemps
   * - ``--cors-allow-origin <origine>``
     - –
     - active CORS pour cette origine, une liste d'origines séparées par des
       virgules, ou ``*`` (désactivé par défaut) ; la réponse ne reflète que
       l'origine de la requête si elle figure dans la liste, avec
       ``Vary: Origin``
   * - ``--rate-limit-rps <n>``
     - ``50``
     - limite de requêtes par seconde et par tenant (0 = désactivé) ; activée
       par défaut à une valeur généreuse plutôt que sur option, pour qu'un
       fichier compose qui ne pense qu'à la base de données n'hérite pas d'un
       démon sans aucune limite
   * - ``--rate-limit-burst <n>``
     - ``100``
     - taille du seau de jetons pour les pics de requêtes
   * - ``--quota-positions <n>``
     - ``0``
     - positions qu'un tenant peut stocker, vérifiées au début d'un import :
       une fois la borne atteinte, l'import est refusé (413,
       ``storage_quota_exceeded``) ; ``positions.save`` et les autres écritures
       unitaires ne sont pas bornées ; 0 = illimité
   * - ``--quota-analysis-seconds <n>``
     - ``0``
     - secondes CPU de calcul du moteur par tenant et par jour UTC (429,
       ``quota_exceeded``) ; 0 = illimité
   * - ``--quota-imports <n>``
     - ``0``
     - imports d'un même tenant en cours à la fois (429, ``quota_exceeded``) ;
       0 = illimité
   * - ``--rls``
     - ``false``
     - PostgreSQL : active la Row-Level Security par tenant (défense en
       profondeur, sur option)
   * - ``--read-tenants``
     - ``false``
     - honore l'en-tête ``X-Read-Tenants`` des lectures ``across.*`` ;
       désactivé, il est refusé (``400``) — voir :ref:`headless_tenants_lus`
   * - ``--bearoff-ts <fichier>``
     - –
     - base de bearoff two-sided (``.bd``) optionnelle élargissant la table
       TS-06-06 pour l'analyse de course du point d'accès EPC ; le démon ne
       télécharge jamais de base — voir :ref:`headless_bearoff`
   * - ``--identity-dir <répertoire>``
     - –
     - répertoire de l'identité de signature du démon (créée au premier
       usage) ; nécessaire pour qu'``exports.sqlite`` puisse apposer un
       filigrane — voir plus bas
   * - ``--import-dir <répertoire>``
     - –
     - répertoire de la machine du démon que ``imports.batch`` peut lire par
       chemin ; désactivé par défaut (un lot n'arrive alors que sous forme
       d'archive) — voir plus bas
   * - ``--ops-addr <hôte:port>``
     - –
     - sert la famille ``/ops/`` (``maintenance.vacuum``, ``tenant.purge``)
       sur une adresse **séparée** de ``--addr``, et l'en retire ; vide (le
       défaut) les laisse sur l'écouteur principal, où c'est au proxy de
       refuser le préfixe — voir :ref:`headless_ops_routes`
   * - ``--pprof-addr <hôte:port>``
     - –
     - expose ``net/http/pprof`` sur une adresse **séparée** de ``--addr``
       (désactivé par défaut) ; débogage uniquement — ces points d'accès
       n'ont aucune notion de tenant et permettent de récupérer un profil
       mémoire ou CPU du processus entier, jamais à exposer publiquement ni
       sur la même adresse que ``/v1``

La plupart des options peuvent aussi être fournies par variable
d'environnement (``BLUNDERDB_BACKEND``, ``BLUNDERDB_DSN``, ``BLUNDERDB_ADDR``,
``BLUNDERDB_LOG_LEVEL``, ``BLUNDERDB_METRICS``, ``BLUNDERDB_CORS_ALLOW_ORIGIN``,
``BLUNDERDB_RATE_LIMIT_RPS``, ``BLUNDERDB_RATE_LIMIT_BURST``, ``BLUNDERDB_RLS``,
``BLUNDERDB_READ_TENANTS``, ``BLUNDERDB_TS_PATH``, ``BLUNDERDB_IDENTITY_DIR``, ``BLUNDERDB_IMPORT_DIR``, ``BLUNDERDB_OPS_ADDR``, ``BLUNDERDB_PPROF_ADDR``) :
un drapeau explicite reste prioritaire sur la variable correspondante.

Le démon n'a **pas** d'option de répertoire de données : il écrit ses tables de
bearoff dans ``$XDG_DATA_HOME/blunderdb``, ou à défaut
``~/.local/share/blunderdb``. C'est donc ``XDG_DATA_HOME`` qui les déplace —
voir :ref:`headless_bearoff`.

La table de seaux du limiteur de débit porte elle-même un plafond dur
(10 000 tenants distincts) : au-delà, chaque nouveau tenant évince le seau le
moins récemment utilisé plutôt que de laisser la table croître sans limite —
utile si un client envoie beaucoup de valeurs ``X-Tenant-ID`` distinctes,
volontairement ou non, entre deux purges périodiques des seaux inactifs.

``blunderdb serve`` refuse tout argument positionnel imprévu (au-delà du seul
``serve`` initial qu'un ``ENTRYPOINT`` déjà réduit au binaire nu laisse
passer) : sans cette vérification, un drapeau placé après un tel argument
était silencieusement ignoré — ``docker run image serve --addr :9090``,
réflexe naturel puisque l'``ENTRYPOINT`` de l'image vaut déjà ``serve``,
démarrait sur ``:8080`` sans un mot.

Points d'accès
--------------

Le service expose des points d'accès d'exploitation, toujours présents :

* ``GET /healthz`` — vivacité (le processus tourne) ;
* ``GET /readyz`` — disponibilité (le stockage répond et son schéma est à la
  version attendue) ;
* ``GET /metrics`` — métriques Prometheus (si ``--metrics`` est actif) ;
* ``GET /app/`` — la page web de consultation (si ``--web`` est actif).

.. _page_web:

La page web de consultation
---------------------------

``blunderdb serve --web`` sert une page sous ``/app/`` : une bibliothèque
consultable depuis une tablette ou un téléphone, sans installer quoi que ce
soit.

Elle sait faire **trois choses**, et cette liste est la décision, pas une
étape :

* **consulter** une position, son analyse et son plateau ;
* **chercher**, avec la même grammaire de jetons que la ligne de commande de
  l'application ;
* **réviser** un paquet Anki — réponse dévoilée et note donnée.

Elle ne sait pas éditer une position, importer, supprimer, gérer les
collections, les matchs, les tournois ou la configuration, et elle ne le saura
pas. Une fonctionnalité qui manque ici n'est pas un manque : c'est le
périmètre.

**Elle est éteinte par défaut, et ce défaut est la décision.** Le démon
n'authentifie personne : il fait confiance à l'en-tête ``X-Tenant-ID`` et doit
tourner derrière un mandataire qui authentifie. Livrer une interface
atteignable par un navigateur, allumée d'office, inviterait exactement le
déploiement que cette règle interdit.

La page **n'envoie aucun tenant** : c'est le mandataire qui pose l'en-tête,
comme pour n'importe quel autre client. En développement local, et là
seulement, ``/app/?tenant=1`` permet d'en nommer un — ce qui ne change rien à
la sécurité d'un démon qui accepte déjà cet en-tête de quiconque.

Les fichiers de la page sont servis **sans tenant**, à dessein : un navigateur
doit pouvoir charger la page avant que le mandataire ne lui attribue quoi que
ce soit, et une page ne contient aucune donnée.

Vivacité et disponibilité répondent à deux questions différentes. ``/healthz``
répond toujours 200 dès que le processus sert des requêtes, sans jamais
interroger le stockage : un orchestrateur redémarre le conteneur dont la
vivacité échoue, et une base momentanément injoignable ne doit pas relancer en
boucle un démon sain. ``/readyz`` répond 503 (avec ``status`` à ``down`` ou
``version_mismatch``) tant que la base ne répond pas ou que son schéma n'est
pas celui du binaire : le trafic est simplement détourné jusqu'à ce qu'elle
revienne.

La sous-commande ``blunderdb healthcheck`` (présente aussi dans le binaire
``serve`` de l'image conteneur) effectue une requête ``GET /readyz`` sur le
démon local et rend ``0`` s'il est disponible, ``1`` sinon ; l'adresse est
celle de ``--addr`` ou de ``BLUNDERDB_ADDR``, par défaut ``:8080``. C'est le
``HEALTHCHECK`` de l'image Docker, et elle vaut tout autant dans un script ou
une unité systemd :

.. code-block:: bash

   blunderdb healthcheck --addr 127.0.0.1:8080 && echo ready

La surface métier suit le schéma ``POST /v1/<famille>.<méthode>`` (par exemple
``/v1/positions.save``, ``/v1/matches.get``). Les familles couvrent les
positions, analyses, matchs, commentaires, collections, tournois, cartes Anki,
filtres, sessions, historique (recherche et commandes), recherche,
métadonnées, réglages de bibliothèque, statistiques, import et export. Les endpoints de listing
renvoient un flux NDJSON (un objet JSON par ligne). Le serveur s'arrête
proprement sur ``SIGINT`` / ``SIGTERM``.

Une erreur rend l'enveloppe ``{"error":{"code":…,"message":…}}``. Le code
``not_found`` dit qu'une ressource nommée n'existe pas ; ``unknown_route``,
lui aussi en 404, dit que le démon ne sert pas la méthode appelée : un client
et un démon de versions différentes, ou une famille que le démon ne sert
qu'avec un drapeau. Un client ne conclut à l'absence d'une donnée que sur
``not_found``.

``positions.save`` rend ``{"id":…,"created":…}``. ``created`` vaut ``true``
pour le seul appel qui a inséré la position, et c'est l'écriture elle-même qui
le dit : un client qui copie une position puis son analyse, et doit défaire la
copie après un échec, ne supprime la position que s'il l'a créée, sans la
course d'un ``positions.exists`` préalable.

.. _headless_versionnage:

Ce que ``/v1`` promet
~~~~~~~~~~~~~~~~~~~~~

Un client écrit contre ``/v1`` doit continuer de fonctionner. La règle tient en
trois lignes, et elle est plus utile écrite que devinée :

* **Ce qui existe ne change pas de sens.** Une route de ``/v1`` n'est ni
  renommée, ni supprimée, ni resignifiée. Un champ de requête ou de réponse
  n'est ni renommé, ni retiré, ni changé de type.
* **Ce qui s'ajoute s'ajoute.** Une route nouvelle, un champ **optionnel** de
  requête, un champ nouveau dans une réponse : un client qui les ignore
  continue de marcher, c'est la définition retenue de « compatible ». Un client
  doit donc ignorer les champs qu'il ne connaît pas plutôt que de les refuser.
* **Le reste, c'est** ``/v2``. Rendre obligatoire un champ qui ne l'était pas,
  changer une unité, changer le sens d'un code d'erreur : ce sont des ruptures,
  et elles vivent sous un autre préfixe, à côté de ``/v1``, le temps que les
  clients traversent.

Deux précisions qui ont leur importance. Les routes ``/ops/`` ne sont **pas**
couvertes : elles servent à l'exploitation d'un déploiement, changent avec lui,
et ne sont pas une API pour des programmes tiers. Et le **contrat lui-même est
généré** depuis la table de routes du démon (``openapi.yaml``,
:ref:`api_reference`) : il ne peut pas décrire autre chose que ce que le
serveur sert.

.. _headless_lecons:

Les leçons
~~~~~~~~~~

Les leçons (:ref:`lecons`) se lisent et s'écrivent sous le tenant de l'appelant
par neuf routes : ``/v1/lessons.list``, ``lessons.get`` (la leçon et ses
étapes, dans l'ordre), ``lessons.create``, ``lessons.update``,
``lessons.delete``, ``lessons.addStep``, ``lessons.updateStep``,
``lessons.removeStep`` et ``lessons.reorderSteps``. La suppression d'une leçon
est définitive et laisse les collections et les positions que ses étapes
montraient. Les outils MCP ``list_lessons`` et ``lesson`` les lisent.

.. _headless_transcription:

Transcrire par l'API
~~~~~~~~~~~~~~~~~~~~

La famille ``transcriptions.*`` permet à un client externe de transcrire un
match geste par geste, avec la même logique que le bureau. Les lectures
(``list``, ``get``, ``exportMat``, ``losses``) sont toujours servies. Les
gestes (``create``, ``open``, ``editMatch``, ``apply``, ``undo``, ``redo``,
``close``, ``finish``, ``abandon``) ne le sont qu'avec ``serve
--transcription`` : sans ce drapeau, ces routes répondent 404.

``create`` et ``open`` rendent l'état du brouillon, sa ``revision`` et un
``sessionId``. ``apply``, ``undo``, ``redo``, ``close`` et ``finish`` nomment
ce ``sessionId`` : absent → **400**, session expirée ou inconnue → **410** ; le
client rouvre alors le brouillon (``open``), curseur en fin de document.
``abandon`` ne nomme pas de session : il supprime le brouillon sous la seule
révision de ``If-Match``.
Chaque geste qui écrit porte la révision vue en dernier dans l'en-tête
``If-Match`` et rend la suivante :

* ``If-Match`` absent → **428** ;
* révision périmée → **409** ; l'enveloppe d'erreur donne la révision
  courante (``details.revision``) et l'état frais du brouillon
  (``details.state`` : document, révision, session et curseur), que le client
  affiche avant de rejouer son geste s'il tient encore.

La révision n'avance que quand le document change (en-tête et actions) :
déplacer le curseur ou entrer un dé de l'action en cours n'écrit rien et rend
la même révision. Une session est celle du brouillon, pas celle d'un client :
``open`` rend la session vivante quand il y en a une, et les onglets ou postes
qui la partagent partagent aussi le curseur et la pile d'annulation.

La session ne garde que la pile d'annulation, le curseur et la saisie en
cours : le brouillon est écrit après chaque geste qui le change, une session
perdue (inactivité, redémarrage, autre instance) ne perd aucun geste.
``transcriptions.get`` rend la révision en ``ETag`` et répond 304 à un
``If-None-Match`` qui la nomme.

``finish`` enregistre le Match et supprime le brouillon, ``abandon`` le
supprime sans Match, ``close`` ne libère que la session. ``editMatch`` ouvre un
brouillon sur un Match existant et rend, pour un match importé, le décompte
des analyses et commentaires que la transcription ne garde pas
(``losses.lossy``). L'analyse du match enregistré se lance par
``gammonnet.analyzeMissing``.

.. warning::

   Le démon n'authentifie personne : ouvrir l'écriture, c'est la confier au
   proxy (:ref:`headless_proxy_deployment`). Un rôle « transcripteur » est une règle du
   proxy sur le préfixe ``/v1/transcriptions.``, pas une notion du démon.

.. _headless_client_python:

Un client Python
~~~~~~~~~~~~~~~~

``clients/python/`` contient un client minimal, sans dépendance hors
bibliothèque standard — le démon parle POST et JSON, ce que ``urllib`` et
``json`` couvrent entièrement :

.. code-block:: python

   from blunderdb import Client

   api = Client("http://127.0.0.1:8080", tenant=1)
   print(api.metadata_counts())

   for position in api.positions_list({"limit": 10}):
       print(position["id"])

Il est en **deux moitiés, et c'est voulu**. ``_generated.py`` porte une méthode
par route, engendrée depuis la table de routes du démon par
``go run ./cmd/openapi-gen`` : une surface écrite à la main dériverait le jour
où une route est ajoutée, et personne ne s'en apercevrait avant qu'un
utilisateur ne le fasse. ``client.py`` porte le transport — la session,
l'en-tête de tenant, l'enveloppe d'erreur, la lecture du NDJSON — et il est
écrit à la main. Ce qui change avec l'API est engendré ; ce qui change avec le
jugement ne l'est pas.

Les noms de méthode sont ``famille_opération`` en snake_case :
``/v1/positions.loadByIds`` devient ``positions_load_by_ids()``. La
famille est conservée parce que plusieurs familles partagent un nom
d'opération (``list``, ``delete``), et qu'un ``list()`` nu entrerait en
collision.

``events()`` suit ``/v1/events`` et rend un dictionnaire par message (voir
:ref:`headless_events`).

Un échec lève ``APIError``, qui porte l'enveloppe du démon telle quelle : le
``code`` (ce sur quoi un programme branche), le ``message`` (ce qu'une personne
lit), le statut HTTP et les détails.

.. _headless_bootstrap:

Embarquer le moteur dans un programme Go
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

``pkg/blunderdb/server.Bootstrap`` ouvre le stockage et rend un jeu de
gestionnaires **dans le processus appelant**, sans écouter sur un port. C'est
la porte d'entrée d'un parent de confiance — gammonGo — qui veut la
bibliothèque de positions sans faire tourner un démon à côté ni parler HTTP à
lui-même.

Ce que cela suppose est explicite : le parent est **de confiance**. Il n'y a
pas de tenant à vérifier, pas d'en-tête à valider, pas de limiteur de débit —
ces choses appartiennent au démon parce qu'il fait face à un réseau, et
l'ADR-0005 dit pourquoi. Un programme qui embarque le moteur choisit lui-même
son tenant et répond de ses appels.

.. _headless_direction:

Direction de tournoi et événements
----------------------------------

Les tournois dirigés au poste de travail et les événements qui les regroupent
(``rencontre`` dans l'API et ses routes ``/v1/rencontres.*``) se lisent par l'API, sous le tenant de l'appelant, avec le même code que le
poste de travail. La lecture est toujours servie ; les gestes (saisir un
résultat, appairer, créer un événement) ne le sont que sous ``serve --direction``
(:ref:`headless_direction_gestures`).

* ``directions.list`` et ``directions.directory`` lisent tout le tenant : la
  liste des tournois dirigés, l'annuaire des joueurs.
* Les autres ``directions.*`` prennent ``{"tournamentId": N}`` :
  ``directions.get`` (la vue complète : propositions, classement, matchs en
  cours), ``directions.participants``, ``directions.freeParticipants``,
  ``directions.tableGrid``, ``directions.brackets``, ``directions.standings``,
  ``directions.standingsCsv``, ``directions.history`` (filtres facultatifs
  ``player`` et ``match``), ``directions.clock``, ``directions.slots``,
  ``directions.lastDecision``, ``directions.pageHtml`` et
  ``directions.pairingSheetHtml`` (avec ``round``).
* ``rencontres.list``, puis ``rencontres.get`` et ``rencontres.pageHtml``
  avec ``{"id": N}``. ``rencontres.pageHtml`` rend la page murale de l'événement,
  un document HTML autonome dans le champ ``html`` : un écran mural l'affiche
  et la relit périodiquement.
* ``rencontres.ranking`` rend le classement de saison, comme ``blunderdb
  tournament ranking --season`` : ``rencontreId``, ``from``, ``to``, ``points``,
  ``participation`` et ``elo``, tous facultatifs ; sans ``rencontreId`` ni
  période, tous les tournois dirigés du tenant comptent.

Les pages sont rendues en français, la langue du moteur de direction. Un
tournoi qui n'est pas dirigé, ou qui appartient à un autre tenant, répond
``404``.

**Lectures conditionnelles.** Chacune de ces routes rend un en-tête ``ETag``.
Renvoyé dans ``If-None-Match``, il obtient ``304`` sans corps tant que rien de
ce que la route lit n'a changé. Toute écriture change l'``ETag`` aussitôt : un
geste dans le tournoi ou dans un tournoi du même événement, le rattachement
d'un match, un brouillon démarré depuis un emplacement, le renommage d'un
tournoi, la modification de l'événement. Répondre ``304`` ne rejoue aucun tournoi,
ce qui rend peu coûteuse une page murale qui interroge toutes les quelques
secondes. Seul ce qui dépend de l'heure fait exception : les propositions,
l'horloge et les pages sont calculées au moment de la lecture, et un ``ETag``
vaut donc au plus une minute. Un client qui relit voit ainsi passer une
échéance ou une pause dans la minute.

Ces routes sont des ``POST``. Pour ce verbe, la RFC 9110 (§13.1.2) répond
``412`` à un ``If-None-Match`` vérifié. Le démon répond pourtant ``304`` : le
corps de la requête ne porte que les paramètres d'une lecture sans effet, qui
se comporte comme un ``GET``. La forme ``If-None-Match: *`` est refusée
(``400``), car elle ne désigne aucune réponse que le client aurait déjà. Une
requête invalide (``round`` négatif, par exemple) est refusée avant toute
condition.

.. code-block:: bash

   curl -si -X POST http://127.0.0.1:8080/v1/rencontres.pageHtml \
     -H 'X-Tenant-ID: 1' -d '{"id":1}' | grep -i '^etag'
   curl -si -X POST http://127.0.0.1:8080/v1/rencontres.pageHtml \
     -H 'X-Tenant-ID: 1' -H 'If-None-Match: W/"…"' -d '{"id":1}'
   # HTTP/1.1 304 Not Modified

Comme le reste de ``/v1``, ces routes n'authentifient personne : derrière le
proxy (:ref:`headless_proxy_deployment`), quiconque atteint le préfixe
``/v1/directions.`` d'un tenant lit ses tournois, noms des joueurs compris. Un
proxy qui réserve ces lectures à certains utilisateurs le fait par une règle
sur ce préfixe et sur ``/v1/rencontres.``.

.. _headless_direction_gestures:

Les gestes de direction
~~~~~~~~~~~~~~~~~~~~~~~

``blunderdb serve --direction`` ouvre les gestes que le poste de travail fait
sur un tournoi dirigé et sur un événement. Sans ce drapeau, ces routes
répondent ``404``, comme absentes. ``call`` les sert toujours.

* ``directions.create`` (``tournamentId``, ``config``, ``seed``),
  ``directions.setConfig`` et ``directions.previewConfig`` (``config``, la
  configuration au format JSON du moteur) ;
* les inscriptions : ``directions.enterParticipants`` (``players``),
  ``directions.addParticipant`` (``name``, ``club``, ``rating`` ; avec
  ``section`` et ``key``, un retardataire prend une place d'exemption),
  ``directions.updateParticipant``, ``directions.withdraw``,
  ``directions.reinstate``, ``directions.makeAbsent``,
  ``directions.makeAvailable``, ``directions.addPair``,
  ``directions.updatePair`` ;
* le déroulement : ``directions.confirmProposal`` (``action``, telle que
  ``directions.get`` la propose), ``directions.confirmAllProposals``,
  ``directions.startMatch``, ``directions.enterResult``,
  ``directions.enterForfeit``, ``directions.moveMatchToTable``,
  ``directions.cancelMatch``, ``directions.correctResult``,
  ``directions.close``, ``directions.reopen``, ``directions.addNote``,
  ``directions.attachMatch``, ``directions.detachMatch`` ;
* l'événement : ``rencontres.create``, ``rencontres.update``,
  ``rencontres.attach``, ``rencontres.detach``, ``rencontres.trash``,
  ``rencontres.setTableOutOfService``, ``rencontres.setBreaks`` ;
* les propriétés des tables : ``rencontres.setTables`` (``id``, ``tableSettings``,
  une entrée par table qui en porte : numéro, nom, salle, réservée, attitrée à),
  ``rencontres.setEventRooms`` (``id``, ``tournamentId``, ``rooms``, les salles
  où l'épreuve joue ; aucune, c'est toutes les tables) et ``directions.setTables``
  (``tournamentId``, ``tableSettings``) pour une épreuve qui joue seule.

Un geste de tournoi rend la vue complète du tournoi, comme ``directions.get`` ;
un geste d'événement rend l'événement. Le service réécrit ensuite les pages
d'affichage dans le dossier que la base désigne, comme au poste de travail.
Une page qui ne peut pas s'écrire (dossier disparu, disque plein) n'annule pas
le geste : la réponse porte un en-tête ``Direction-Page-Warning`` par page non
écrite (``tournament 3``, ``rencontre 2``), sans le chemin du serveur, et le
poste de travail l'affiche dans sa barre d'état.

Un geste que les règles refusent (nom vide, table occupée, tournoi qui n'a pas
commencé, configuration rejetée par le moteur) rend ``400`` avec le motif. Une
panne du démon ou de sa base rend ``500``, sans détail : le motif reste dans le
journal du démon.

**Version obligatoire.** Toute lecture d'un tournoi ou d'un événement rend un
en-tête ``Direction-Version``, et tout geste le renvoie dans ``If-Match`` :

* sans ``If-Match`` (ou avec ``*``), le geste est refusé : ``428`` ;
* si quelqu'un a écrit depuis cette lecture, le geste est refusé : ``409``. Le
  champ ``details`` de l'erreur porte l'état frais et sa ``version`` : le
  client relit, puis rejoue son geste s'il reste valable ;
* sinon le geste s'applique et rend la nouvelle version dans
  ``Direction-Version``.

La comparaison se fait dans la transaction du geste, sous un verrou de la base
(verrou consultatif PostgreSQL par tournoi ou par événement, verrou d'écriture
SQLite) : de deux gestes envoyés sur la même lecture, un seul s'applique, qu'ils
passent par un même démon, par deux démons sur une même base PostgreSQL, ou par
le poste de travail et ``call`` sur un même fichier. Le geste écrit tout ou
rien. Un tournoi joué dans un événement a la
version de son événement, si bien qu'un geste dans une épreuve sœur la change
aussi. ``directions.create`` et ``rencontres.create`` ne visent rien
d'existant et ne prennent pas de version.

**Idempotence.** Un geste qui porte un en-tête ``Idempotency-Key`` ne
s'applique qu'une fois : renvoyé avec la même clé, il rend la première réponse,
avec ses en-têtes (``Direction-Version`` compris) et
``Idempotency-Replayed: true``. Un double clic ou une reprise réseau ne saisit
pas deux résultats ; deux envois simultanés de la même clé n'exécutent le geste
qu'une fois. Seule une réponse réussie est retenue.

* La clé est liée au corps de la requête : la même clé avec un autre corps rend
  ``422``.
* Le rejeu passe avant le contrôle de version : il rend la réponse retenue sans
  ``428`` ni ``409``, même si la version a bougé depuis.
* Les clés vivent en mémoire, dans chaque instance du démon, pendant 24 heures,
  au plus 1 000 par tenant : un redémarrage les oublie, et une autre instance
  ne les connaît pas.

.. code-block:: bash

   curl -si -X POST http://127.0.0.1:8080/v1/directions.get \
     -H 'X-Tenant-ID: 1' -d '{"tournamentId":3}' | grep -i '^direction-version'
   curl -s -X POST http://127.0.0.1:8080/v1/directions.enterResult \
     -H 'X-Tenant-ID: 1' -H 'If-Match: "…"' -H 'Idempotency-Key: t4-r2' \
     -d '{"tournamentId":3,"matchId":"m7","winner":"aa","scoreA":7,"scoreB":3}'

.. warning::

   Le démon n'authentifie personne (ADR-0005). Avec ``--direction``,
   quiconque le proxy laisse passer saisit des résultats. Le moteur ne connaît
   aucun rôle (directeur, arbitre, lecteur) : un rôle est une règle du proxy,
   qui réserve ``/v1/directions.`` et ``/v1/rencontres.`` aux directeurs, ou
   n'y laisse passer que les lectures. Ne lancez jamais ``--direction`` sur un
   démon joignable sans ce proxy, même sur le Wi-Fi d'un club.

.. _headless_events:

Être prévenu des gestes : ``/v1/events``
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

``GET /v1/events`` est un flux *Server-Sent Events* (``text/event-stream``) :
un message par geste validé du tenant, publié après l'écriture en base, jamais
pour un geste refusé ou annulé. Le message dit ce qui a bougé et sa nouvelle
version, pas l'état : le client relit ce qu'il affiche, avec
``If-None-Match``.

* ``event: rencontre`` — ``rencontreId``, ``tournamentIds`` (les épreuves de
  l'événement, avant et après le geste) et ``version`` ;
* ``event: direction`` — ``tournamentId`` et ``version``, pour un tournoi
  joué hors de tout événement ;
* ``event: transcription`` — ``transcriptionId`` et ``revision`` ; un
  brouillon abandonné ou terminé porte ``removed`` (et ``matchId`` pour
  Terminer).

``removed: true`` signale ce qui n'existe plus. La route n'est servie qu'avec
``--direction`` ou ``--transcription`` : sans eux, le démon n'écrit rien qu'il
aurait à annoncer, et ``/v1/events`` répond ``404``. Comme toute route
``/v1/``, elle exige ``X-Tenant-ID`` : un abonné n'entend que son tenant. Un
tenant tient au plus 16 flux ouverts à la fois ; au-delà, ``429``. Le poste de
travail emprunte le même service mais n'y branche aucun bus : ses gestes ne
sont pas annoncés.

Les paramètres ``tournament``, ``rencontre`` et ``transcription`` (identifiants
séparés par des virgules, ou répétés) restreignent l'abonnement : un message
passe s'il nomme l'un d'eux. Un tournoi d'un événement reçoit les messages de
son événement. Un paramètre inconnu ou un identifiant invalide rend ``400``.

.. code-block:: bash

   curl -N http://127.0.0.1:8080/v1/events?rencontre=2 -H 'X-Tenant-ID: 1'

**Pas d'historique.** Le démon ne garde aucun message. Tout flux s'ouvre sur
``event: resync``, avec un ``id`` : le client a pu manquer des gestes avant de
se connecter, ou entre deux connexions, et relit tout ce qu'il affiche. Le
motif est ``reconnected`` quand la requête porte ``Last-Event-ID``,
``subscribed`` sinon. Un abonné trop lent, dont la
file de 64 messages est pleine, est déconnecté après le même ``resync`` : il
ne retarde jamais un geste. Le flux annonce un délai de reconnexion de
3 secondes.

**À travers un proxy.** Un commentaire ``: ping`` part toutes les 25 secondes
pour qu'un proxy ne coupe pas un flux silencieux ; ``X-Accel-Buffering: no``
demande à nginx de ne pas le mettre en tampon. Le flux n'est pas compressé,
échappe au délai des requêtes ordinaires et ne compte qu'une requête pour la
limitation de débit. L'arrêt du démon ferme tous les flux ; un abonnement
demandé pendant l'arrêt reçoit ``503``.

**Plusieurs instances.** Sur SQLite, une seule instance tient la base : le bus
en mémoire suffit. Sur PostgreSQL, dès que ``--direction`` ou
``--transcription`` est actif, chaque instance relaie ses gestes aux autres par
``LISTEN/NOTIFY``, sur le canal ``blunderdb_events`` : un abonné relié à une
instance entend un geste validé sur une autre, ou passé par ``call`` sur la
même base. Le tenant voyage dans la notification et l'instance qui la reçoit ne
la remet qu'aux abonnés de ce tenant. Chaque instance ouvre deux connexions de
plus (``application_name`` ``blunderdb-events-…`` pour l'écoute,
``blunderdb-notify-…`` pour l'envoi) ; une instance qui ne peut pas écouter au
démarrage refuse de démarrer. ``call`` annonce sans écouter, et sert sa
requête même s'il ne peut pas annoncer.

Tout rôle autorisé à se connecter peut émettre sur ce canal, sous ``--rls``
aussi. Une notification reçue n'est crue que si son tenant est valide et sa
sorte connue ; le reste est journalisé et ignoré. Une notification forgée peut
au pire faire relire leurs données aux abonnés d'un tenant.

* La notification part après l'écriture en base, comme le message local. Deux
  pertes restent sans ``resync`` : une instance tuée entre l'écriture et la
  notification, et un arrêt qui ne peut pas envoyer en 2 secondes ce qui reste
  en file. Le geste est validé, mais les flux déjà ouverts sur les autres
  instances ne l'apprennent qu'à la reconnexion de leur client.
* Une connexion d'écoute perdue est rétablie, avec une attente croissante de
  250 ms à 30 s. Les gestes des autres instances passés pendant la coupure
  sont perdus : à la reprise, chaque abonné de l'instance reçoit un ``resync``
  de motif ``missed``. Une notification trop longue pour PostgreSQL
  (8 000 octets), ou qu'une instance n'a pas pu envoyer, arrive aux autres
  comme ce même ``resync`` pour le tenant concerné.
* Les ``id`` du flux sont propres à chaque instance. Un client qu'un
  répartiteur de charge envoie vers une autre instance n'en tire rien : le
  ``resync`` qui ouvre tout flux lui fait relire ce qu'il affiche.

.. _headless_bearoff:

Les bases de bearoff
--------------------

Le démon calcule ses deux tables par défaut au démarrage, en arrière-plan
(TS-06-06 pour le verdict de videau, OS-06 pour l'EPC) : environ six secondes
d'un cœur, une fois, dans son répertoire de données —
``$XDG_DATA_HOME/blunderdb``, ou à défaut ``~/.local/share/blunderdb``. Rien
n'est téléchargé et rien
n'est embarqué dans le binaire (`ADR-0027 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0027-bearoff-databases-are-generated-not-shipped-and-verified-against-gnubg.md>`__).
Si ce dossier est en lecture seule, les tables sont tenues en mémoire pour la
durée du processus : le service démarre, il paie simplement le calcul à chaque
redémarrage.

Un domaine plus large ne se calcule pas au démarrage — TS-06-11 pèse 1,2 Go et
prend des minutes, ce n'est pas quelque chose qu'un service décide seul. C'est
à l'opérateur de le fabriquer, avec la CLI, dans le volume que le démon lira :

.. code-block:: bash

   # generate
   blunderdb bearoff generate --ts 6x11 --data-dir /srv/data/blunderdb

   # serve
   XDG_DATA_HOME=/srv/data blunderdb serve --db database.db
   blunderdb serve --db database.db \
       --bearoff-ts /srv/data/blunderdb/gnubg_ts6x11.bd

Le premier lancement laisse le démon trouver la table tout seul dans son
répertoire de données ; le second la désigne par son chemin, où qu'elle soit.
``--data-dir`` est une option des sous-commandes ``bearoff``, jamais de
``serve``.

``blunderdb bearoff list --data-dir /srv/data/blunderdb`` dit ce que le volume
contient et ce que chaque domaine coûterait ; ``blunderdb bearoff verify`` sort
en erreur sur une table corrompue, ce qui en fait une sonde de démarrage
utilisable telle quelle. Voir :ref:`cli` pour le détail.

.. _headless_ops_routes:

Les routes d'exploitation
-------------------------

Deux appels ne s'arrêtent pas au tenant qui les passe, et vivent donc sous un
préfixe à part, ``POST /ops/<famille>.<méthode>`` :

* ``/ops/maintenance.vacuum`` (backend SQLite) réécrit **tout** le fichier,
  données de tous les tenants comprises, et tient un verrou d'écriture pendant
  l'opération ;
* ``/ops/tenant.purge`` (backend PostgreSQL) détruit les données d'un tenant,
  et le tenant détruit est celui que nomme l'en-tête que l'appelant contrôle.

Le démon n'authentifie personne (voir plus bas) : une route joignable par un
tenant est une route que **tout** tenant peut appeler. Le préfixe existe pour
que le proxy puisse refuser les deux d'une seule règle. **Ne jamais exposer**
``/ops/`` **par le proxy public.** Sous nginx, la règle tient en une ligne du
bloc ``server`` ; sous Caddy, en deux lignes du site :

.. code-block:: nginx

   location /ops/    { return 403; }
   location /metrics { return 403; }

.. code-block:: text

   @closed path /ops/* /metrics
   respond @closed 403

L'option ``--ops-addr <hôte:port>`` va plus loin : les deux routes quittent
alors l'adresse ``--addr`` et ne sont plus servies que sur ce second
écouteur, à lier sur une interface d'administration. Sans cette option, elles
restent sur l'écouteur principal et c'est au proxy de les bloquer.

Ces routes exigent l'en-tête ``X-Tenant-ID`` comme toutes les autres — une
purge nomme le tenant qu'elle détruit, elle en a besoin plus que quiconque.
Seules les sondes (``/healthz``, ``/readyz``) et ``/metrics`` s'en passent.

C'est pourquoi la règle de refus ci-dessus couvre aussi ``/metrics`` : n'exigeant
aucun tenant, il est lisible par quiconque atteint le démon, et il publie la
taille de la base et le travail en cours, tous tenants confondus. Il se consulte
depuis la machine du démon, ou par un chemin que le proxy réserve à
l'exploitation. Le troisième point à ne jamais exposer n'est pas une route mais
un écouteur : celui de ``--pprof-addr``, qui n'a aucune notion de tenant et
livre un profil du processus entier. Il se lie à une interface
d'administration, jamais publié par le proxy.

Ce qui n'est **pas** passé sous ``/ops/`` : ``/v1/gammonnet.sweepStale``. Le
rattrapage est coûteux mais il est cadré au tenant appelant ; ce sont la
limite de débit et les jauges de travail en vol qui le bornent, pas une
frontière de confiance.

Le contrat complet — chaque méthode, sa requête et sa réponse — est généré
depuis le code source et versionné : ``openapi.yaml`` à la racine du dépôt
(format OpenAPI, schémas compris) et son annexe lisible, :ref:`api_reference`
(tableau famille par famille). Les deux sont régénérés par
``go run ./cmd/openapi-gen`` et un test dédié échoue si l'un des deux prend du
retard sur les routes réellement enregistrées.

Chaque requête ``/v1`` accepte un corps JSON (``Content-Type:
application/json``, ou aucun en-tête — un corps d'un autre type est refusé
avec ``400 invalid`` plutôt que d'échouer sur un message d'analyse JSON
confus) ; une méthode connue appelée avec le mauvais verbe HTTP répond
``405``, l'en-tête ``Allow`` nommant le seul verbe accepté. Les méthodes de
liste qui acceptent un ``limit`` refusent au-delà de 1000 lignes par page
(``400 invalid``) plutôt que d'honorer une valeur sans plafond.

Les familles listantes acceptent toutes ``limit`` et ``offset`` :
``positions.list``, ``positions.listIds``, ``matches.list``, ``search.find``,
``anki.reviewLog``, ``comments.listAll``,
``tournaments.list`` et ``collections.positions``. Les deux valent zéro par
défaut, ce qui veut dire ce que cela a toujours voulu dire : tout. **Il n'y a
pas de plafond implicite** — un flux n'est pas retenu en mémoire, donc une
liste sans borne coûte du temps et de la bande passante mais jamais l'équilibre
du démon, alors qu'une limite par défaut silencieuse ferait lire à un client
une liste tronquée en la croyant complète. Ce que ces deux paramètres apportent
est la possibilité de paginer, à qui le veut.

Chaque connexion TCP est bornée en lecture/écriture par requête — un budget
généreux pour les appels ordinaires, bien plus large pour les routes qui
streament (listes NDJSON, imports/exports, rattrapage gammonNet) — et leur
nombre simultané est plafonné, au-delà duquel une connexion supplémentaire
attend qu'une des premières se libère plutôt que de recevoir sans limite un
fil d'exécution par connexion. Un arrêt gracieux (``SIGINT``/``SIGTERM``)
annule d'abord tout import et tout rattrapage gammonNet en cours — chacun
répond par un dernier évènement ``{"event":"cancelled"}`` plutôt que de voir
sa connexion coupée sans explication — avant de fermer le serveur dans le
délai de grâce habituel. Le fichier temporaire d'un import téléversé ne
retient de l'extension d'origine que celles connues du démon
(``.xg``, ``.xgp``, ``.sgf``, ``.mat``, ``.bgf``, ``.ogxm``, ``.txt``, ``.db``,
``.dbx``), et l'ensemble des imports simultanés — tous tenants confondus —
partage un quota global d'octets déposés sur disque : au-delà, un nouvel
import est refusé (``too many requests``) plutôt que de laisser croître sans
borne l'occupation de ``$TMPDIR``.

``/v1/imports.json`` relit un export JSON de blunderDB en comblant les vides :
l'analyse qu'il porte ne s'écrit que sur une position qui n'en a pas encore,
sans jamais remplacer une analyse existante, et les rollouts des deux côtés
sont gardés.

La famille ``search`` offre trois portes sur la même recherche.
``search.find`` prend l'objet de filtres complet, champ par champ.
``search.query`` prend une requête écrite dans le langage de la barre de
commande de l'application (``s cube p>30 E>50``, décrit dans
:doc:`cmd_mode`) et streame les mêmes positions ; c'est la seule façon
d'atteindre depuis le réseau les filtres qui n'ont pas de champ évident —
motif de coup, texte de commentaire, joueur, date, dés exclus, zones et
blots. ``search.parse`` ne cherche rien : elle répond ce qu'une requête veut
dire — les filtres qu'elle dénote, sa forme canonique (deux requêtes
équivalentes ont la même, ce qui rend une recherche enregistrée comparable)
et ses diagnostics.

Une requête portant un jeton que rien ne reconnaît est refusée
(``400 invalid``, le jeton nommé) plutôt qu'exécutée en réduisant la
recherche en silence. Un jeton compris mais sans effet ici — ``x``, qui
active la structure d'exclusion, laquelle est un plateau et non du texte —
voyage dans l'en-tête ``X-BlunderDB-Query-Diagnostics`` afin que le corps
reste du NDJSON de positions pour tous les clients existants.

Deux méthodes de la famille ``positions`` décodent une position sans
l'enregistrer : ``positions.fromXGID`` reconstruit une position à partir d'une
chaîne XGID, et ``positions.fromXGP`` à partir d'un fichier de position unique
``.xgp``.

``POST /v1/exports.sqlite`` exporte tout le tenant courant — positions,
collections, matchs, tournois, analyses, commentaires, coups joués,
bibliothèque de filtres et paquets Anki — dans un fichier SQLite ouvrable tel
quel par le poste de travail. Le corps JSON de la requête est optionnel :
``watermarkOrigin`` / ``watermarkNote`` apposent un filigrane signé de
l'identité propre du démon (``--identity-dir``) — sans ces champs, l'export ne
porte aucun filigrane ; les demander sans identité configurée échoue avec le
code ``invalid``. ``collectionIds`` restreint l'export à ces collections et à
leurs positions, avec analyses, commentaires et coups joués, sans la
bibliothèque de filtres ni les paquets Anki.

**Importer un dossier ou un corpus** se fait par ``imports.batch``, qui passe
par le même pipeline que ``blunderdb import --type batch`` : mêmes matchs,
mêmes empreintes ``canonical_hash``, un doublon n'est écrit qu'une fois. L'appel
rend aussitôt ``{"importId": …, "files": N}`` et l'import continue en tâche de
fond, le temps qu'il faut. Le lot arrive sous l'une de ces deux formes :

* une **archive** ``.zip`` ou ``.tar`` envoyée en multipart (champ ``file``) ; seuls les
  fichiers importables en sont extraits, et la taille de l'archive comme celle
  de son contenu décompressé sont bornées par ``ImportMaxBodyBytes`` (512 Mio
  par défaut) ;
* un **chemin local au démon**, ``{"path": "corpus", "recursive": true}``, lu
  dans le répertoire que l'opérateur a ouvert avec ``--import-dir``. Sans cette
  option, tout chemin est refusé (code ``invalid``) ; avec elle, un chemin qui
  sort du répertoire, lien symbolique compris, l'est aussi. Le démon n'authentifie
  personne : nommer un répertoire, c'est laisser tout appelant y lire des
  fichiers de match. C'est la voie des gros corpus, qu'on ne pousse pas par
  HTTP.

Un match déjà en base n'est pas réécrit, mais ses analyses plus profondes que
celles rangées les remplacent ; ``?skip_duplicates=true``, accepté par
``imports.batch`` comme par l'import d'un fichier, l'ignore sans rien en
reprendre que les marques, comme ``--skip-duplicates`` en ligne de commande.

``Idempotency-Key`` est accepté : rejouer l'appel avec la même clé rend le même
``importId`` (en-tête ``Idempotency-Replayed: true``) sans relancer l'import. Pour
un lot reçu en archive, la clé désigne le lot, pas le contenu de l'archive.

``imports.batch.status`` (``{"importId": …}``) rend l'état (``receiving``,
``running``, ``done``, ``cancelled`` ou ``failed``) et la progression
mesurée par le pipeline : fichiers traités, matchs importés, doublons, fichiers en
erreur, positions, octets lus, débit, durée restante estimée. Les cent premières
erreurs sont listées avec leur fichier, le total est exact. Un lot terminé reste
lisible une heure. ``imports.batch.cancel`` l'arrête : les groupes de fichiers
déjà validés restent dans la base, le groupe en cours est annulé.
``imports.report`` rend, d'après le ``batchId`` du statut, le rapport de fin
d'import. ``imports.files`` (``{"batchId": …}``) rend le journal du lot : pour
chaque fichier, son chemin, sa taille, sa date, son empreinte SHA-256 et le
résultat (``new``, ``duplicate``, ``enriched`` ou ``error``) avec le match
obtenu ou le message. Pour continuer un lot interrompu, ``imports.batch``
accepte ``"resume": <batchId>`` (champ ``resume`` d'un envoi multipart) : les
fichiers déjà journalisés avec le même chemin, la même taille et la même date
sont sautés sans être lus, ceux de même contenu sont lus mais pas analysés.

.. code-block:: bash

   curl -X POST http://127.0.0.1:8080/v1/imports.batch \
        -H 'X-Tenant-ID: club-lyon' -F file=@corpus.zip
   curl -X POST http://127.0.0.1:8080/v1/imports.batch.status \
        -H 'X-Tenant-ID: club-lyon' -H 'Content-Type: application/json' \
        -d '{"importId":"3f9c…"}'

**Partager une collection entre tenants** passe par le client, jamais par une
lecture d'un tenant dans l'autre : le tenant qui donne appelle
``exports.sqlite`` avec ``collectionIds`` (et un filigrane, pour que le
receveur sache d'où vient le fichier), le tenant qui reçoit envoie le fichier
à ``imports.db``. Chaque requête porte son propre ``X-Tenant-ID`` ; le proxy
décide qui a le droit de faire l'une et l'autre. À l'import, une collection
rejoint celle du même nom chez le receveur, ou est créée ; ses positions s'y
ajoutent à la suite, sans doublon. Une collection vivante du receveur ne
reçoit aucune position : sa requête fait son contenu. L'import d'une base
dans l'application de bureau suit la même règle.

.. code-block:: bash

   curl -X POST http://127.0.0.1:8080/v1/exports.sqlite \
        -H 'X-Tenant-ID: club-lyon' -H 'Content-Type: application/json' \
        -d '{"collectionIds":[4],"watermarkOrigin":"Club de Lyon"}' -o ouvertures.db
   curl -X POST http://127.0.0.1:8080/v1/imports.db \
        -H 'X-Tenant-ID: alice' -F file=@ouvertures.db

La famille ``training`` tient le journal de l'onglet Entraînement :
``training.save`` ajoute une séance (``exercise``, ``seedSource``, comptes,
``items``) et rend son ``id`` (``Idempotency-Key`` accepté) ;
``training.sessions`` relit les séances, la plus récente d'abord (``exercise``
et ``limit`` facultatifs) ; ``training.numberStats`` agrège les items d'un
exercice par type de nombre. Les questions, elles, sont tirées par le client.

``gammonnet.evaluate`` évalue une position nue (``position`` ou ``xgid``), sans
rien lire ni écrire dans le tenant : avec dés, les meilleurs coups
(``candidates``, 5 par défaut, au plus 20) ; sans dés, la décision de videau.
``ply`` va de 0 à 2 (2 par défaut) ; une recherche plus profonde est le travail
d'``analyzeMissing``.

La famille ``anki`` gagne six méthodes qui étendent le planificateur à
répétition espacée (FSRS) : ``anki.reviewLog`` (journal de chaque révision —
notation et résultat FSRS — pour les statistiques de rétention et un
historique fidèle), ``anki.forecast`` (projection du nombre de cartes dues sur
les prochains jours, cartes en retard comprises), ``anki.suspendCard`` /
``anki.buryCard`` / ``anki.removeCard`` (retirer une carte de la file de
révision temporairement ou définitivement) et ``anki.retention`` (taux de
réussite mesuré sur les révisions d'un paquet, lu en regard de la cible que son
propriétaire a fixée).

.. note::

   ``anki.retention`` remplace ``anki.optimizeParams``, qui ajustait la cible
   vers le taux observé et pouvait l'écrire. La cible de rétention est un
   **choix** sur le compromis charge/qualité, le taux mesuré en est le
   **résultat**, et asservir l'un à l'autre est le mécanisme que les auteurs de
   FSRS écartent. La méthode mesure, sans jamais écrire.

La famille ``stats`` fournit ``stats.playerTable``, qui renvoie une ligne de
statistiques par joueur (matchs, victoires/défaites, décisions comptées, PR
global / pions / videau, Snowie Error Rate, erreurs, blunders et chance) sur
les matchs retenus par le filtre transmis. Comme dans l'interface graphique, ce
tableau n'honore du filtre que la période, les tournois et la longueur des
matchs : la sélection d'un joueur et le type de décision sont ignorés, puisque
le tableau porte sur tous les joueurs et ventile déjà pions et videau en
colonnes distinctes. Le champ ``luck_known`` indique si la chance a été mesurée
pour ce joueur ; ``luck_rate_mp`` ne doit pas être lu quand il vaut ``false``,
une chance inconnue n'étant pas une chance nulle.

Le filtre transmis aux méthodes ``stats`` accepte, à côté de ``PlayerName``, un
champ ``PlayerAliases`` : les autres orthographes sous lesquelles la même
personne a signé. Le nom d'un joueur étant saisi à la main dans chaque fichier,
une même personne apparaît couramment sous plusieurs graphies, et un filtre qui
n'en retient qu'une calcule sur une partie des matchs sans que rien n'ait l'air
anormal. Le champ est purement additif : les décisions de n'importe lequel des
noms sont retenues. Fusionner les noms en base (``MergePlayers``) est l'autre
réponse, à réserver aux bases qu'on n'a pas reçues de quelqu'un d'autre —
elle réécrit les matchs de tout le monde.

Deux méthodes complètent la parité avec l'interface graphique :
``stats.tournamentBadges`` renvoie, pour chaque tournoi de la base, l'indicateur
affiché sur sa vignette (PR du joueur de référence), et ``matches.findByHash``
indique si un match donné est déjà présent, à partir des deux empreintes de
détection de doublon — de quoi éviter un import redondant avant de l'engager.
``matches.duplicates`` liste les paires de matchs que les dés disent
identiques — mêmes dés sous d'autres noms, ou version tronquée puis complétée —
comme ``repair --duplicates``, sans rien fusionner.

Les autres graphies d'un joueur ou d'un événement se gèrent par
``players.alias.list``, ``players.alias.set`` (``{"alias": …, "canonical": …}``),
``players.alias.remove`` (``{"alias": …}``, renvoie ``removed``) et
``players.alias.suggest`` (les noms qui ne diffèrent que par la casse, les
accents, la ponctuation ou l'ordre des mots, sans rien enregistrer) ; les mêmes
sous ``events.alias.*`` pour les événements. ``matches.mergePlayers`` crée ces
alias plutôt que de réécrire les matchs. Un import enregistre le nom canonique
(les empreintes gardent les noms du fichier), et ``stats.compute``,
``stats.playerTable``, ``stats.playerNames`` et la recherche par joueur lisent
toutes les graphies comme une seule personne.

Le champ ``winner`` d'une partie, reçu par ``matches.createGame`` et renvoyé par
``matches.games``, a un seul codage : ``1`` pour le joueur 1, ``-1`` pour le
joueur 2, ``0`` pour une partie inachevée. Un client qui envoie encore ``0``,
``1`` ou ``-1`` au sens de gnubg (``0`` pour le joueur 1, ``1`` pour le
joueur 2) inscrit le gagnant inverse.

``analyses.repair`` recalcule les colonnes dénormalisées d'une analyse (dont
``cube_error``) à partir de son analyse complète, et renvoie le nombre de
lignes **réellement** corrigées. Ces colonnes ne sont qu'une projection : une
erreur de projection se répare donc sans réimporter les fichiers source.
L'opération est explicite et n'est jamais déclenchée toute seule — ni à
l'ouverture d'une base, ni par une migration, le schéma n'étant pas en cause.
Une analyse illisible est laissée telle quelle plutôt que remise à zéro. Le cas
d'usage connu : les non-doubles étiquetés « Double No » par gnuBG, dont la
lecture était fautive avant la version 0.33.0 et qui portaient l'erreur d'un
double qui n'a jamais eu lieu.

``gammonnet.analyzeMissing`` déclenche le rattrapage gammonNet du tenant
courant : écrire une analyse pour chaque position qui n'en a aucune
(`ADR-0013 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md>`__,
`ADR-0015 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0015-blunderdb-serve-operates-on-a-library-it-does-not-expose-an-evaluator.md>`__).
C'est une opération de **bibliothèque** — elle lit et écrit des
positions et des analyses stockées — jamais un évaluateur nu :
``blunderdb serve`` opère sur une bibliothèque, ``gammonnet serve`` évalue une
position. La réponse est un flux NDJSON (``started``, ``progress``, puis
``done`` ou ``error``/``cancelled``), sur le même modèle que les points
d'accès d'import ; ``gammonnet.analyzeMissing.cancel`` (avec le ``job_id`` reçu
dans l'évènement ``started``) annule un rattrapage en cours et sert
indifféremment pour un rattrapage ou une réanalyse (ci-dessous). C'est la même
opération que le déclenchement automatique après import et le geste explicite
de l'interface graphique, et que la sous-commande ``blunderdb analyze`` (voir
:ref:`cli`) — trois formes, une seule logique.

``gammonnet.sweepStale`` est le pendant de ``analyzeMissing`` pour la
réanalyse plutôt que le comblement : chaque position dont l'analyse est
entièrement issue de gammonNet mais périmée — une version de moteur plus
ancienne que celle en cours d'exécution, ou une profondeur différente de
``ply`` — est réévaluée à la profondeur demandée. Le prédicat de péremption
est partagé avec le même lot de l'interface graphique et de
``blunderdb analyze --stale`` (aucune duplication de la logique entre les
trois modes) ; une position portant une analyse XG, GNUbg ou BGBlitz n'est
jamais touchée, quel que soit son contenu gammonNet — la protection
d'`ADR-0013 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0013-evaluations-fill-gaps-an-imported-analysis-is-never-overwritten.md>`__
reste inconditionnelle. Même forme NDJSON qu'``analyzeMissing``, et
l'évènement final de chacune des deux routes porte la répartition
``evaluated``/``refused``/``failed`` : une position que gammonNet refuse
d'évaluer (un score de match hors de la portée de sa table, une décision de
videau que le modèle refuse) compte comme ``refused``, pas ``failed`` — elle
n'est jamais retentée en vain sur la passe suivante, contrairement à une
position réellement en échec.

``rollout.position`` joue une position de la bibliothèque (``positionId``) par
un :ref:`rollout <cli_rollout>` et rend, pour chaque candidat, l'équité, son
intervalle à 95 % et la JSD ; ``rollout`` porte les réglages (``fast``,
``standard`` ou ``standard,ply=1``…), ``store`` enregistre le rollout terminé
comme une seconde analyse, à côté de celle que porte la position, qu'il ne
remplace jamais. Une position nue (un XGID) est refusée : le démon opère sur une
bibliothèque. ``rollout.filter`` est la forme en lot de
``blunderdb analyze --rollout`` : les positions que choisit ``query`` (le
langage de la recherche) et qui ne portent pas encore de rollout aux mêmes
réglages sont jouées l'une après l'autre et enregistrées au fil de l'eau, en
flux NDJSON (``started``, ``progress`` après chaque série de parties, puis
``done``, ``cancelled`` ou ``quota_exceeded``) ; ``rollout.filter.cancel`` l'annule avec son
``job_id``. Un tenant ne mène qu'un lot à la fois, rollout ou gammonNet.
``rollout.list`` lit les rollouts enregistrés d'une position.

Corrélation et métriques métier
--------------------------------

Chaque requête reçoit un identifiant de corrélation : celui que le client (ou
un reverse-proxy) envoie dans l'en-tête ``X-Request-Id``, sinon un
identifiant généré, dans les deux cas renvoyé sur la même en-tête de la
réponse et ajouté à la ligne de journal de fin de requête (champ
``request_id``). Un ``traceparent`` (`W3C Trace Context
<https://www.w3.org/TR/trace-context/>`__) éventuellement présent est relayé
tel quel dans cette même ligne de journal — le démon ne l'analyse ni ne le
valide, il n'embarque aucune bibliothèque de traçage : c'est un pont pour
corréler ces journaux avec un pipeline de traçage qui tournerait en amont,
rien de plus.

Au-delà du volume de requêtes et de leur latence, ``/metrics`` publie des
jauges sur le travail en vol, invisible autrement à un import ou un lot
gammonNet bloqué (une seule requête très longue, pas beaucoup de requêtes) :

* ``blunderdb_imports_inflight`` — imports en cours, tous tenants confondus ;
* ``blunderdb_import_spool_bytes`` — octets actuellement réservés sur le
  quota de spool d'import (voir ``--rate-limit-*`` plus haut pour le
  pendant requêtes/seconde) ;
* ``blunderdb_gammonnet_sweep_inflight`` — rattrapages gammonNet en cours,
  tous tenants confondus ;
* ``blunderdb_database_size_bytes`` — taille du fichier SQLite principal, ou
  ``pg_database_size`` sous PostgreSQL (base entière, pas par tenant, comme
  les jauges de pool de connexions ci-dessous) ; absente tant qu'aucune
  mesure n'a encore été publiée.

Un profil mémoire ou CPU du processus est accessible en démarrant avec
``--pprof-addr <hôte:port>`` (``net/http/pprof``) : désactivé par défaut, et
volontairement sur une adresse séparée de ``--addr`` puisque ces points
d'accès n'ont aucune notion de tenant.

.. _headless_docker:

Compression des flux
---------------------

Les listes NDJSON répètent les mêmes noms de champs à chaque ligne. Le démon
les compresse quand le client l'accepte : envoyez ``Accept-Encoding: gzip`` et
la réponse revient en ``Content-Encoding: gzip``. Mesuré sur une liste de
matchs : **13,5 %** de la taille d'origine sur mille lignes, 14,6 % sur cent.

La compression ne change rien au caractère incrémental du flux — chaque
enregistrement est poussé au client comme avant, il est seulement compressé en
chemin. Elle ne s'applique qu'aux réponses NDJSON, JSON et texte : un export de
base ou un conteneur ``.dbx`` est déjà compressé, le regzipper ne ferait que le
grossir. ``Accept-Encoding: gzip;q=0`` la refuse explicitement.

Un seul tenant sur SQLite
--------------------------

Le backend SQLite n'a **pas** de colonne de tenant : toutes les données y sont
dans les mêmes tables, sans cloison. Le démon refuse donc, sur ce backend, tout
``X-Tenant-ID`` autre que ``1`` — accepter les autres reviendrait à servir à
chacun les lignes de tous derrière un en-tête qui prétend le contraire. Un
déploiement qui a réellement plusieurs tenants a besoin du backend PostgreSQL.

.. _headless_tenants_lus:

Lire plusieurs tenants
----------------------

Un coach qui lit les matchs de ses élèves, un club qui partage une bibliothèque :
la relation entre ces comptes vit chez l'hôte qui les authentifie, jamais dans le
démon. Le proxy l'exprime par l'en-tête ``X-Read-Tenants``, une liste de tenants
séparés par des virgules (``X-Read-Tenants: 2, 3``), qu'il pose à côté de
``X-Tenant-ID``. Le démon lui fait confiance comme à ``X-Tenant-ID`` et
n'autorise rien lui-même
(`ADR-0063 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0063-une-lecture-peut-porter-sur-les-tenants-que-le-proxy-liste.md>`__).

La fonction est **désactivée par défaut**, et désactivée veut dire refusée : tant
que le démon n'est pas lancé avec ``--read-tenants`` (ou
``BLUNDERDB_READ_TENANTS=true`` ; ``Config.TrustReadTenants`` pour un hôte qui
embarque le moteur), toute requête qui porte un ``X-Read-Tenants`` non vide est
refusée (``400``), quelle que soit la route. Ne l'activer qu'une fois le proxy
configuré pour retirer toute valeur envoyée par le client et poser lui-même la
liste.

Seules les lectures ``/v1/across.*`` regardent cet en-tête. Sur toute la
liste : ``across.searchFind``, ``across.matchesList``, ``across.statsCompute`` et
``across.playerTable`` ; elles lisent ``X-Tenant-ID`` d'abord, puis chaque tenant
listé dans l'ordre de l'en-tête, 64 tenants distincts au plus en tout. Sur un
tenant de la liste, nommé avec l'id : ``across.matchesGet``,
``across.matchMovePositions`` (les positions d'un match, coup par coup, par
pages ``limit`` / ``offset``) et
``across.analysesLoadByIds`` ; un tenant absent de la liste y est refusé. Chaque
résultat porte son tenant d'origine (``"tenant": "2"``), car un id n'est unique
que dans son tenant ; une position porte aussi son hachage Zobrist
(``"zobrist"``), qui désigne le même plateau dans tous les tenants. ``limit``
s'applique à chaque tenant ; 0 vaut 1000, et davantage est refusé. Dans un flux
NDJSON, une erreur sur un tenant tardif arrive en dernière ligne, après les
résultats des tenants déjà lus : le flux entier est alors en échec.

.. code-block:: bash

   curl -s http://127.0.0.1:8080/v1/across.matchesList \
     -H 'X-Tenant-ID: 1' -H 'X-Read-Tenants: 2, 3' -d '{"limit":20}'

Toute écriture reste dans ``X-Tenant-ID`` : aucune autre route ne lit
``X-Read-Tenants``. Sans l'en-tête, une lecture ``across.*`` ne porte que sur
``X-Tenant-ID``. Un en-tête mal formé (un nom, un élément vide, plus de 64
tenants) ou envoyé sur plusieurs lignes refuse la requête entière, quelle que
soit la route. Sur SQLite, qui n'a qu'un tenant, la liste ne peut contenir que
``1`` : l'en-tête n'y élargit rien. Ces routes sont propres au serveur : le
bureau et ``call`` n'ont qu'un tenant.

Une requête ``across.*`` coûte jusqu'à 64 lectures au stockage, mais la limite
de débit (``--rate-limit-rps``) ne la compte qu'une fois, pour ``X-Tenant-ID`` :
dimensionner la base et cette limite en conséquence, ou faire borner la liste par
le proxy. Le journal d'accès d'une route ``across.*`` porte la liste reçue
(champ ``read_tenants``). L'en-tête ne figure pas parmi les en-têtes CORS
autorisés : seul le proxy l'écrit, jamais un navigateur.

Quatre lectures servent le club et le coach, sur la même liste
(`ADR-0065 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0065-club-et-coach-lisent-a-travers-les-tenants-et-n-ecrivent-que-chez-soi.md>`__) :

- **commentaires du coach** : le coach range chez lui la position d'un élève
  (``positions.save`` sous son propre ``X-Tenant-ID``, même hachage) et la
  commente (``comments.add``) ; rien n'est écrit chez l'élève.
  ``across.commentsByZobrist`` (``{"zobrists": [...]}``, 1000 hachages au plus)
  rend les commentaires que chaque tenant lu a écrits sur ces plateaux, chacun
  avec son tenant, l'id de la position chez lui et son origine ;
- **bibliothèque partagée** : ``across.collectionsList`` liste les collections
  des tenants lus, ``across.collectionPositions`` (``tenant``,
  ``collectionId``) les positions de l'une d'elles, chacune avec son hachage. La
  bibliothèque est lue sur place, jamais copiée : partager une collection par
  fichier (export puis import) en donne au contraire une copie au receveur ;
- **classement de club** : ``across.clubRanking`` fusionne les tables de joueurs
  des tenants lus en un seul classement, meilleur PR d'abord, sur le filtre de
  statistiques donné (``filter.DateFrom`` et ``filter.DateTo`` pour une
  période). Chaque ligne porte son tenant ; un nom n'est jamais fusionné d'un
  tenant à l'autre. ``players`` (paires ``tenant``, ``name``) ne garde que ces
  joueurs, ``minDecisions`` écarte ceux qui ont moins de décisions comptées ; à
  PR égal, le rang est partagé, et une ligne sans décision comptée a le rang 0.
  ``filter.TournamentIDs`` y est refusé (``400``) : un id de tournoi ne vaut
  que dans son tenant. La réponse est une page (``limit``, 100 par défaut, 1000
  au plus ; ``offset``) avec ``total``, le nombre de lignes du classement
  entier. Ce n'est pas le classement de saison des tournois dirigés.

.. _headless_sauvegarde:

Sauvegarde et restauration
---------------------------

Quatre gestes, selon ce qu'on veut récupérer.

**Tout, sous PostgreSQL** — ``pg_dump`` est l'outil, et blunderDB n'a rien à
ajouter :

.. code-block:: bash

   pg_dump --format=custom --file=blunderdb.dump "postgres://…"
   pg_restore --dbname="postgres://…" blunderdb.dump

**Tout, sous SQLite en conteneur** — le fichier est ouvert en mode WAL (le
démon encode ``journal_mode(WAL)`` dans sa chaîne de connexion, pour toutes les
connexions du pool) : à côté de ``blunderdb.db`` vivent un ``-wal`` et un
``-shm``, et les écritures les plus récentes sont dans le ``-wal``. Copier le
seul ``.db`` d'un démon en marche donne donc un fichier incomplet, sans que
rien ne le signale. Deux façons sûres :

* **arrêter le démon, puis copier le volume entier** — à l'arrêt les trois
  fichiers sont cohérents, et c'est le volume, pas le ``.db`` seul, qui est
  l'unité à sauvegarder ;
* **ne pas copier le fichier du tout** : ``/v1/exports.sqlite`` (ci-dessous)
  écrit un ``.db`` complet pendant que le démon tourne, et c'est le seul geste
  qui ne demande aucune interruption.

``/ops/maintenance.vacuum`` replie bien le WAL dans le fichier principal avant
de le réécrire, mais il ne gèle pas la base : l'écriture qui suit repart dans le
WAL. C'est une commande de compactage, pas une méthode de sauvegarde.

**Un tenant seul** — ``/v1/exports.sqlite`` écrit la base d'un tenant dans un
fichier ``.db`` ordinaire, celui que l'application de bureau ouvre :

.. code-block:: bash

   curl -X POST http://127.0.0.1:8080/v1/exports.sqlite \
     -H "X-Tenant-ID: 42" -o tenant-42.db

Cette commande s'exécute **sur la machine du démon** : elle vise l'écouteur
local, court-circuite le proxy, et pose donc elle-même l'en-tête du tenant.
Depuis l'extérieur, c'est le proxy qu'on interroge, et le tenant est celui du
compte authentifié — l'en-tête n'est pas à donner, le proxy efface celui du
client avant d'injecter le sien :

.. code-block:: bash

   curl -u alice:… -X POST \
     https://blunderdb.example.com/v1/exports.sqlite -o tenant-alice.db

**Remettre ce fichier en place** — ``migrate`` le recopie sous le tenant voulu :

.. code-block:: bash

   ./blunderdb migrate --from tenant-42.db --to "postgres://…" --tenant-id 42

``migrate`` refuse d'écrire dans un tenant qui contient déjà quelque chose, et
dit quoi (« 128 positions, 3 matchs ») ; ``--on-conflict skip`` passe outre et
laisse la déduplication par empreinte Zobrist fusionner les positions.

Ce que ``migrate`` **ne copie pas**, et qu'il annonce en fin de course avec le
compte exact : les paquets Anki et leurs cartes, la bibliothèque de filtres,
les historiques de recherche et de commandes, et l'état de session. Ce sont des
données d'usage de l'application de bureau ; les positions auxquelles elles
renvoient, elles, ont bien été déplacées.

Les **seuils d'erreur et de blunder**, eux, sont copiés : ce ne sont pas des
données d'usage mais l'habitude de lecture dont dépendent les comptes, et un
tenant qui compterait autrement que le fichier dont il vient ferait de la
migration un changement de sens muet.

Le tenant règle les siens par ``POST /v1/librarySettings.load`` et
``/v1/librarySettings.save``. Contrairement à ``metadata``, qui est une
infrastructure globale exposée en lecture seule, la table des réglages porte un
``tenant_id`` et vit sous Row-Level Security : un tenant qui écrit ses seuils
n'atteint que ses propres lignes.

.. _headless_poste_serveur:

Le poste de travail et le serveur
----------------------------------

L'application de bureau ouvre des **fichiers** ``.db``, pas des URL : elle ne se
connecte à aucun démon ``serve``, et il n'existe nulle part de champ où saisir
une adresse. Le serveur et le poste de travail échangent des fichiers, en deux
gestes symétriques :

* **du serveur vers le poste** — ``POST /v1/exports.sqlite`` écrit tout le
  tenant courant dans un ``.db`` que l'application de bureau ouvre tel quel
  (voir :ref:`headless_sauvegarde`) ;
* **du poste vers le serveur** — ``blunderdb migrate`` recopie un ``.db`` sous
  le tenant voulu (voir :ref:`headless_migrate`).

Il n'existe **aucune lecture inter-tenant**. Le cloisonnement est total : rien
de ce qu'un tenant stocke n'est visible à un autre, par aucune route, et aucun
appel ne prend un tenant en paramètre — chaque requête ne connaît que celui que
le proxy lui a posé. Un entraîneur qui veut voir les matchs de ses élèves a donc
deux chemins, tous deux explicites :

* lui ouvrir dans le proxy un compte supplémentaire, associé au tenant de
  l'élève : c'est la table de correspondance du proxy, jamais le démon, qui
  décide du tenant qu'une session voit ;
* lui demander un export — le ``.db`` produit par ``exports.sqlite`` ou par la
  fenêtre d'export de l'application de bureau — et l'ouvrir sur son propre
  poste.

Déploiement avec Docker
-----------------------

Le dépôt fournit un ``Dockerfile.serve`` qui construit une image conteneur
minimale du démon : seul le binaire ``serve`` est compilé (Go pur, sans
interface graphique et sans CGO, donc lié statiquement), puis placé dans une
image *distroless*.

.. code-block:: bash

   # build
   docker build -f Dockerfile.serve -t blunderdb-serve .

   # run
   docker run --rm -p 127.0.0.1:8080:8080 \
       -e BLUNDERDB_DSN="postgres://user:pass@host:5432/blunderdb?sslmode=disable" \
       blunderdb-serve

La construction se lance depuis la racine du dépôt, et le backend par défaut de
l'image est ``postgres``.

L'image écoute sur le port 8080 et se configure par variables d'environnement
(``BLUNDERDB_BACKEND``, ``BLUNDERDB_DSN``, ``BLUNDERDB_ADDR``,
``BLUNDERDB_RLS``). Elle déclare un ``HEALTHCHECK`` qui lance toutes les
30 secondes ``blunderdb healthcheck`` (une requête sur ``/readyz`` — l'image
*distroless* n'a ni ``curl`` ni shell) : ``docker ps`` affiche l'état
``healthy`` ou ``unhealthy`` du conteneur, et Compose ou un orchestrateur
peuvent attendre que le démon soit disponible avant de démarrer ce qui en
dépend.

.. _headless_docker_image:

Image publiée
~~~~~~~~~~~~~

Il n'est pas nécessaire de construire l'image soi-même : chaque version
publiée de blunderDB pousse la sienne sur le registre GitHub (GHCR), sous le
nom ``ghcr.io/kevung/blunderdb-serve``. Deux étiquettes sont disponibles :
le numéro de la version, figé à jamais sur cette image, et ``latest``, qui suit
la dernière version publiée. Toute la documentation les note
``ghcr.io/kevung/blunderdb-serve:<version>`` : c'est le numéro d'une version
publiée qui prend la place de ``<version>``, et c'est cette forme, jamais
``latest``, qu'un déploiement de production épingle. L'image est fournie pour
``linux/amd64`` et ``linux/arm64`` ; Docker choisit l'architecture de l'hôte.

.. code-block:: bash

   # pull
   docker pull ghcr.io/kevung/blunderdb-serve:<version>

   # postgres
   docker run --rm -p 127.0.0.1:8080:8080 \
       -e BLUNDERDB_DSN="postgres://user:pass@host:5432/blunderdb?sslmode=disable" \
       ghcr.io/kevung/blunderdb-serve:<version>

   # sqlite
   docker run --rm -p 127.0.0.1:8080:8080 \
       -v blunderdb-data:/data \
       -e BLUNDERDB_BACKEND=sqlite -e BLUNDERDB_DSN=/data/blunderdb.db \
       ghcr.io/kevung/blunderdb-serve:<version>

``/data`` est le point de montage que l'image prépare, avec les droits de son
utilisateur non privilégié, et son ``XDG_DATA_HOME`` : le volume qu'on y monte
ne sert pas qu'à la base, les tables de bearoff y sont calculées une fois,
dans ``/data/blunderdb``, et retrouvées aux démarrages suivants. Sans volume,
elles sont recalculées à chaque démarrage du conteneur — quelques secondes —
et le démon le dit au démarrage s'il ne peut pas les écrire (*could not
prepare the bearoff tables; the exact regime will be unavailable*), auquel cas
il sert normalement, avec le seul régime estimé sur les positions de sortie.

L'image porte les étiquettes OCI usuelles (``org.opencontainers.image.source``,
``.version``, ``.revision``, ``.licenses``) : ``docker inspect`` dit de quel
commit et de quelle version elle provient. Elle est construite par l'intégration
continue à partir du ``Dockerfile.serve`` du dépôt, exactement comme ci-dessus ;
construire localement ou tirer l'image publiée donne le même binaire.

.. warning::

   Comme le démon lui-même, le conteneur n'effectue **aucune
   authentification** (`ADR-0005 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0005-serve-daemon-delegates-authentication.md>`__) : il fait confiance à l'en-tête
   ``X-Tenant-ID`` tel qu'il le reçoit. Il doit être placé derrière un
   reverse-proxy chargé de l'authentification, qui fixe cet en-tête lui-même,
   et ne jamais être exposé directement sur l'Internet public. Les exemples
   ci-dessus publient le port sur ``127.0.0.1`` seulement pour cette raison,
   et ``--addr`` se lie de même à ``127.0.0.1`` : le proxy est sur la même
   machine.

.. _headless_proxy_deployment:

Déploiement derrière un proxy authentifiant
--------------------------------------------

L'`ADR-0005 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0005-serve-daemon-delegates-authentication.md>`__
fait du reverse-proxy **toute** la frontière de sécurité du démon :
lui seul authentifie l'appelant, lui seul a le droit de poser l'en-tête
``X-Tenant-ID``, et il doit **retirer** systématiquement toute valeur envoyée
par le client avant d'y injecter le tenant authentifié — sans quoi n'importe
qui peut se faire passer pour n'importe quel tenant en le nommant lui-même.
Le modèle de menace tient en une phrase : le démon suppose un réseau interne
de confiance, et quiconque le joint directement est, pour lui, le tenant
qu'il prétend être.
Le dépôt fournit un exemple complet, prêt à lancer, dans le répertoire
`deploy/ <https://github.com/kevung/blunderDB/tree/main/deploy>`__. Il vit dans
le dépôt git, pas dans l'image conteneur : il faut donc **cloner le dépôt**, ou
télécharger les deux fichiers reproduits ci-dessous ainsi que
`deploy/.env.example <https://github.com/kevung/blunderDB/blob/main/deploy/.env.example>`__
dans un même répertoire.

Le fichier Compose met Caddy — authentification HTTP Basic de démonstration —
devant ``blunderdb-serve`` et PostgreSQL, Row-Level Security activée. Seul Caddy
publie un port : les deux autres services vivent sur un réseau Docker déclaré
``internal: true``, qui n'a de route ni vers l'hôte ni vers l'Internet, quels
que soient les ``ports:`` qu'une modification ultérieure leur ajouterait.

.. literalinclude:: ../../deploy/docker-compose.yml
   :language: yaml
   :caption: deploy/docker-compose.yml

Le ``Caddyfile`` authentifie, associe le compte authentifié à l'entier du tenant
(``map``), puis l'injecte dans ``X-Tenant-ID`` **après** avoir explicitement
effacé toute valeur reçue du client : la garde ``header_up X-Tenant-ID ""``
précède l'injection, de sorte qu'un en-tête envoyé par le client ne peut
atteindre le démon quelles que soient les modifications ultérieures du fichier.

Il en va de même pour ``X-Read-Tenants`` (:ref:`headless_tenants_lus`) : le
proxy retire celui du client, et ne le pose que s'il connaît la relation entre
les comptes ; les exemples du dépôt n'en connaissent aucune et le retirent
toujours.

.. literalinclude:: ../../deploy/Caddyfile
   :language: text
   :caption: deploy/Caddyfile

Deux autres fichiers complètent le répertoire :
`deploy/nginx-tenant-proxy.conf <https://github.com/kevung/blunderDB/blob/main/deploy/nginx-tenant-proxy.conf>`__
reprend le même schéma en extrait nginx (``proxy_set_header X-Tenant-ID ""``
puis ``proxy_set_header X-Tenant-ID $tenant_id``, avec le bloc
``map $remote_user $tenant_id``), pour qui a déjà un nginx en place ;
`deploy/README.md <https://github.com/kevung/blunderDB/blob/main/deploy/README.md>`__
énonce le modèle de menace et ce qu'il ne faut jamais faire.

L'authentification HTTP Basic du ``Caddyfile`` est une démonstration, pas une
recommandation de production : elle se remplace par ``forward_auth`` vers un
fournisseur d'identité réel (OIDC, SSO d'entreprise…), qui authentifie puis
transmet l'identité au même endroit du fichier. Les deux mots de passe et les
deux comptes de la table de correspondance sont à remplacer de même.

`deploy/Caddyfile.oidc <https://github.com/kevung/blunderDB/blob/main/deploy/Caddyfile.oidc>`__
en est la recette OpenID Connect : Caddy interroge oauth2-proxy
(``forward_auth`` sur ``/oauth2/auth``), qui répond 202 avec l'adresse du
compte connecté dans ``X-Auth-Request-Email``, ou renvoie vers la page de
connexion du fournisseur. Le bloc ``map`` associe cette adresse à l'entier du
tenant, et la même garde ``header_up X-Tenant-ID ""`` précède l'injection.
Le service oauth2-proxy à ajouter au fichier Compose figure en tête du
fichier.

Quotas par tenant
~~~~~~~~~~~~~~~~~

Une instance partagée borne ce que chaque tenant lui prend avec
``--quota-positions``, ``--quota-analysis-seconds`` et ``--quota-imports``
(sans option, rien n'est borné). Le temps de calcul compte chaque calcul du
moteur demandé par le tenant : ``gammonnet.analyzeMissing``,
``gammonnet.sweepStale``, ``gammonnet.compare``, ``gammonnet.cubeMatrix``,
``gammonnet.evaluate``, ``rollout.position`` et ``rollout.filter``. Il se
compte en secondes CPU : le temps écoulé multiplié par le nombre de recherches
menées à la fois, si bien qu'un calcul réparti sur tous les cœurs coûte autant
que le même travail mené position par position. Une fois le temps du jour
épuisé, ces routes répondent 429 avec le code ``quota_exceeded``. Un balayage
ou un ``rollout.filter`` en cours garde ce qu'il a enregistré et finit sur
l'évènement ``quota_exceeded`` au lieu de ``done`` ; un ``rollout.position``
interrompu répond 429 et n'enregistre rien ; une comparaison interrompue rend
ce qu'elle a replié avec ``quotaExceeded: true`` et, dans ``gathered``, le
nombre de positions qu'elle devait examiner. Le compte repart à zéro à minuit
UTC et vit en mémoire : un redémarrage du démon le remet à zéro. Le quota de
positions est vérifié au début d'un import, qui n'est pas interrompu en route :
un tenant peut le dépasser d'autant que ses imports en cours ajoutent.
``positions.save`` et les autres écritures unitaires ne le vérifient pas.
Chaque refus porte dans ``details`` la borne (``quota``, ``limit``) et l'usage
(``used``). ``tenants.quota`` rend au tenant appelant les bornes et son
usage : positions stockées, secondes de calcul du jour, imports en cours.

Les quotas sont une comptabilité du démon, pas une frontière : ils
s'appliquent au tenant que le proxy a posé dans ``X-Tenant-ID``.

**Scénario complet, de zéro à un démon qui répond :**

.. code-block:: bash

   git clone https://github.com/kevung/blunderDB.git
   cd blunderDB/deploy
   cp .env.example .env    # POSTGRES_PASSWORD
   docker compose up -d --build

   # 401
   curl -i http://localhost:8080/v1/metadata.counts -d '{}'

   # 200
   curl -u alice:demo-password http://localhost:8080/v1/metadata.counts -d '{}'
   curl -u alice:demo-password -H "X-Tenant-ID: 999" \
        http://localhost:8080/v1/metadata.counts -d '{}'

   docker compose logs blunderdb-serve
   docker compose down -v

La première requête est rejetée par Caddy, avant même d'atteindre le démon. Les
deux suivantes sont authentifiées comme « alice », que la table de
correspondance associe au tenant 1 : elles renvoient le même corps
(``{"positions":0,"analyses":0,"matches":0,…}``) et le journal du démon porte
``tenant=1`` pour l'une comme pour l'autre — la valeur 999 envoyée par le client
n'a pas survécu à la garde du ``Caddyfile``. Ce scénario a été rejoué tel quel.

Pour tirer l'image publiée plutôt que de la construire, remplacer dans
``docker-compose.yml`` les trois lignes ``build:`` du service
``blunderdb-serve`` par une ligne ``image:``, puis lancer
``docker compose up -d`` sans ``--build`` :

.. code-block:: yaml

   blunderdb-serve:
     image: ghcr.io/kevung/blunderdb-serve:<version>
     restart: unless-stopped

Le fichier Compose publie le port de Caddy sur toutes les interfaces
(``8080:80``) : c'est ce qu'on attend d'un proxy, qui est là pour être joint. Ce
qui ne doit jamais être publié, c'est le démon — et il ne l'est pas, il n'a
aucun ``ports:``.

.. _headless_mise_a_jour:

Mettre à jour un déploiement
-----------------------------

Le schéma est migré automatiquement au démarrage, et cette migration est **à
sens unique** : une base migrée vers un schéma récent n'est plus lisible par une
version antérieure de blunderDB (voir :ref:`annexe_db_migration`). L'ordre des
gestes compte donc.

#. **Sauvegarder d'abord**, avant tout le reste : c'est la seule marche arrière
   (voir :ref:`headless_sauvegarde`).
#. **Tirer l'étiquette de la version voulue**, jamais ``latest`` en production.
   ``latest`` suit la dernière version publiée : le déploiement qui l'épingle
   change de version au gré des redémarrages, sans qu'on l'ait décidé ni que la
   sauvegarde de l'étape 1 soit forcément récente.
#. **Redémarrer le démon** sur la nouvelle image. Il migre le schéma avant de
   servir la moindre requête ; si la migration échoue, il s'arrête sur
   l'erreur plutôt que de servir une base à moitié migrée.
#. **Vérifier la sonde de disponibilité.** ``GET /readyz`` répond ``200`` et
   ``{"status":"ready","version":"…"}`` quand le stockage répond et que son
   schéma est celui du binaire ; ``503`` et ``{"status":"down"}`` quand la base
   est injoignable ; ``503`` et
   ``{"status":"version_mismatch","version":"…","expected":"…"}`` quand les deux
   schémas diffèrent — la réponse nomme celui de la base et celui que le binaire
   attend. ``blunderdb healthcheck`` rend le même verdict en code de retour.

Un ``version_mismatch`` qui persiste après le redémarrage, c'est le retour en
arrière : un binaire plus ancien devant une base déjà migrée. Il n'existe pas de
migration descendante ; c'est la sauvegarde de l'étape 1 qu'il faut restaurer.

.. important::

   **Avant d'activer** ``--read-tenants`` **sur un déploiement existant**, mettre
   à jour le proxy : un proxy configuré avant cet en-tête ne retire que
   ``X-Tenant-ID`` et transmettrait tel quel un ``X-Read-Tenants`` envoyé par le
   client, qui lirait alors d'autres tenants. Sans l'option, le démon refuse cet
   en-tête : un proxy qui le laisse passer se voit à ses réponses ``400``.

.. _headless_postgres:

Backend PostgreSQL et multi-utilisateurs
========================================

Pour un déploiement partagé, blunderDB peut stocker les données dans
**PostgreSQL** plutôt que dans un fichier SQLite. Le backend est sélectionné
par ``--backend postgres`` et la chaîne de connexion ``--dsn``. Le schéma est
créé et migré automatiquement au démarrage.

Les données sont **cloisonnées par tenant** (locataire) : chaque requête porte
l'identifiant de son tenant (en-tête ``X-Tenant-ID``, un entier décimal
positif comme ``1`` ou ``42``), ce qui permet à plusieurs utilisateurs de
partager la même instance sans voir les données des autres. Un identifiant qui
n'est pas un tel entier — un nom comme ``alice`` ou ``default``, ``0``, ``007``
— est refusé avec ``400 invalid`` : c'est le reverse-proxy qui associe un
compte à son entier, le démon ne devine jamais.

Row-Level Security
------------------

L'option ``--rls`` active en complément la **Row-Level Security** de
PostgreSQL. À chaque démarrage, le démon installe sur chaque table portant un
``tenant_id`` une politique ``tenant_isolation`` qui ne laisse passer que les
lignes du tenant nommé par le paramètre de session
``current_setting('app.tenant_id')``, et la force jusqu'au propriétaire de la
table (``FORCE ROW LEVEL SECURITY``). Ce paramètre est posé sur la connexion à
sa sortie du pool et remis à zéro à son retour ; une connexion sans tenant ne
voit aucune ligne et n'en insère aucune. C'est une défense en profondeur
facultative, désactivée par défaut : le filtrage par tenant du code applicatif
reste en place dans les deux cas.

* **Le rôle de connexion doit être ordinaire** : ni superutilisateur, ni
  ``BYPASSRLS``. PostgreSQL laisse ces deux-là traverser toutes les politiques
  sans un mot, et l'isolation redevient celle du code applicatif seul. Ce même
  rôle doit en revanche posséder les tables, puisque c'est lui qui exécute les
  ``ALTER TABLE`` et les ``CREATE POLICY``.
* **Sur une base déjà peuplée, il n'y a rien à migrer** : la pose des politiques
  est du DDL idempotent, rejoué à chaque démarrage après la migration de schéma.
  Aucune donnée n'est déplacée, aucune ligne réécrite ; activer ou retirer
  ``--rls`` n'est qu'un redémarrage.
* **Le coût est mesuré** : sur la lecture d'une position, 101,8 µs sans, 177,0 µs
  avec, soit **+73,8 %** — même conteneur, mêmes lignes, deux pools ne
  différant que par ce drapeau. Il se paie sur chaque emprunt de connexion au
  pool (pose puis remise à zéro du paramètre) et sur le prédicat que chaque
  requête traverse en plus, jamais sur le volume de données.

Ouvrir et fermer un tenant
--------------------------

Il n'y a **rien à créer** côté serveur : un tenant n'est pas un enregistrement,
c'est l'entier que portent ses lignes. La base n'a pas de table des tenants et
le démon n'en tient aucune liste — ouvrir un compte, c'est ajouter une entrée à
la table de correspondance du proxy, et le premier écrit du membre fait exister
son tenant.

Un tenant vide répond comme une base vide, sans erreur : ``metadata.counts``
renvoie des zéros et les listes ne renvoient rien.

Quand un tenant est décommissionné, ``POST /ops/tenant.purge`` supprime
définitivement toutes ses données (positions, matchs, collections, historique,
etc.) sur le tenant courant (celui porté par ``X-Tenant-ID``), **ainsi que son
état de session** (dernière recherche, dernière position, onglets ouverts —
les lignes de la table ``session_state`` portant ce tenant) : l'opération
s'exécute dans une seule transaction, est idempotente (aucune erreur à purger
un tenant déjà vide ou à répéter l'appel) et n'affecte aucun autre tenant. Elle
efface les lignes de ce tenant dans **toutes** les tables qui en portent un et
ne laisse que ce qui n'appartient à personne : la table ``metadata``, dont la
ligne globale de version de schéma, et le journal des migrations. Le tenant
purgé redevient donc exactement un tenant vide, et son entier se réattribue.
Elle n'est disponible qu'avec le backend PostgreSQL — elle renvoie une erreur
``invalid`` sur un backend SQLite, qui n'a pas de notion de tenant.

Compactage et pool de connexions
--------------------------------

``POST /ops/maintenance.vacuum`` compacte le fichier SQLite du
daemon — le pendant du bouton « Compacter la base » de l'interface graphique
et de la commande ``blunderdb vacuum`` (voir :ref:`cli`), avec la même garde
d'espace disque — et renvoie les tailles avant et après (``sizeBefore``,
``sizeAfter``, en octets). Elle n'est disponible qu'avec le backend SQLite ;
sur PostgreSQL, qui n'a pas de fichier à compacter, elle renvoie une erreur
``invalid``.

Le pool de connexions PostgreSQL se règle par variable d'environnement :
``BLUNDERDB_POSTGRES_MAX_CONNS`` (50 par défaut), ``BLUNDERDB_POSTGRES_MIN_CONNS``
(5), ``BLUNDERDB_POSTGRES_MAX_CONN_LIFETIME`` (``1h``),
``BLUNDERDB_POSTGRES_HEALTH_CHECK_PERIOD`` (``30s``),
``BLUNDERDB_POSTGRES_CONNECT_TIMEOUT`` (``5s`` — au-delà, une base injoignable
échoue vite plutôt que de bloquer sur le délai TCP du système
d'exploitation) et ``BLUNDERDB_POSTGRES_MAX_CONN_IDLE_TIME`` (``30m`` — une
connexion ouverte pour un pic de trafic ne reste pas indéfiniment dans le
pool une fois le pic passé). Chaque valeur est une durée au format Go
(``5s``, ``30m``, ``1h``) ; absente ou invalide, elle retombe sur son défaut.
Quand ``--metrics`` est actif, l'état du pool est exposé en continu sur
``/metrics`` : ``blunderdb_pg_pool_acquired`` (connexions actuellement
utilisées), ``_idle`` (disponibles), ``_max`` (plafond configuré) et
``_wait_count`` (nombre cumulé d'``Acquire`` ayant dû attendre une connexion
libre).

.. _headless_migrate:

Migrer une base SQLite vers PostgreSQL
======================================

``blunderdb migrate`` copie une base SQLite mono-utilisateur vers un backend
PostgreSQL, sous un tenant choisi — l'entier que le reverse-proxy enverra dans
``X-Tenant-ID`` pour cet utilisateur — c'est le chemin pour « téléverser » une
bibliothèque de bureau vers un déploiement serveur.

.. code-block:: bash

   blunderdb migrate \
       --from sqlite:///path/to/database.db \
       --to   "postgres://user:pass@host:5432/db?sslmode=disable" \
       --tenant-id 42

   # --dry-run
   blunderdb migrate --from sqlite:///path/to/database.db \
       --tenant-id 42 --dry-run

La migration copie les **positions, leurs analyses et commentaires, les matchs
(parties + coups), les tournois (avec leurs liens de match) et les collections
(avec leur composition)**, en réattribuant les clés primaires et étrangères, le
tout dans une **seule transaction** côté destination : l'opération est atomique
(un échec laisse la destination intacte, il suffit de relancer). La progression
et le bilan final sont émis en NDJSON sur la sortie standard. Si la base source
est assez ancienne pour nécessiter sa propre mise à niveau de schéma sur place,
celle-ci s'exécute d'abord et émet ses propres événements
``"schema-migration"`` (phase/effectué/total) avant que la copie ligne à ligne
ne commence.

.. list-table::
   :header-rows: 1
   :widths: 24 12 40

   * - Option
     - Défaut
     - Signification
   * - ``--from <uri>``
     - –
     - base SQLite source (``sqlite:///<chemin>`` ou un simple chemin)
   * - ``--to <dsn>``
     - –
     - DSN PostgreSQL de destination (``postgres://…``)
   * - ``--tenant-id <n>``
     - –
     - tenant de destination, un entier décimal positif (obligatoire sauf en
       ``--dry-run`` ; un nom comme ``mon-tenant`` est refusé)
   * - ``--dry-run``
     - –
     - compte ce qui serait copié sans rien écrire
   * - ``--on-conflict <politique>``
     - ``""``
     - ``""`` interrompt si le tenant a déjà des données ; ``skip`` fusionne
       (déduplication des positions par hash Zobrist)

.. note::

   Ne sont pas (encore) migrés les états applicatifs : decks/cartes Anki,
   bibliothèque de filtres, historique de recherche et de commandes, et
   métadonnées de session. La priorité est la migration de la bibliothèque de
   positions et de l'historique de matchs.

.. _headless_call:

Le dispatcher générique ``call``
================================

En complément des sous-commandes historiques (:ref:`cli`), ``blunderdb call``
expose **toutes** les opérations de stockage directement, en local. Il passe
par les mêmes gestionnaires que le démon ``serve`` : le comportement est donc
identique à ``POST /v1/<famille>.<méthode>``. C'est utile pour le scripting et
les tests d'intégration.

.. code-block:: bash

   # --list
   blunderdb call --list

   # read
   blunderdb call metadata.counts --db database.db
   blunderdb call positions.list  --db database.db --json '{"limit":10}'
   blunderdb call matches.get     --db database.db --json '{"id":1}'

   # write
   blunderdb call positions.save  --db database.db --json '{"position":{...}}'
   blunderdb call matches.delete  --db database.db --json '{"id":42}'

   # a gesture of a tournament Direction, with the version a read printed
   blunderdb call directions.enterResult --db database.db --if-match '…' \
     --json '{"tournamentId":3,"matchId":"m7","winner":"aa"}'

**Options:**

.. list-table::
   :header-rows: 1
   :widths: 22 14 40

   * - Option
     - Défaut
     - Signification
   * - ``--db <chemin>``
     - –
     - fichier SQLite (raccourci pour ``--backend sqlite --dsn <chemin>``)
   * - ``--backend <type>``
     - ``sqlite``
     - ``sqlite`` ou ``postgres``
   * - ``--dsn <chaîne>``
     - ``$BLUNDERDB_DSN``
     - chaîne de connexion du backend
   * - ``--scope <n>``
     - ``1``
     - tenant, un entier décimal positif (envoyé comme ``X-Tenant-ID`` ; un
       nom comme ``alice`` est refusé)
   * - ``--json <chaîne>``
     - ``{}``
     - corps de la requête au format JSON
   * - ``--json-file <chemin>``
     - –
     - lit le corps de la requête depuis un fichier
   * - ``--list``
     - –
     - affiche toutes les méthodes ``<famille>.<méthode>`` et quitte
   * - ``--if-match <version>``
     - –
     - version envoyée en ``If-Match``, exigée par les gestes de direction
       (:ref:`headless_direction_gestures`) et de transcription
       (:ref:`headless_transcription`)

``call`` sert les gestes de transcription sans drapeau : il travaille sur un
fichier local, comme la CLI. Chaque appel est un processus neuf, donc sa propre
session : le ``sessionId`` peut être omis, et il n'y a pas d'annulation d'un
appel à l'autre.

La réponse JSON (ou le flux NDJSON pour les endpoints ``*.list``) est écrite
sur la sortie standard. En cas d'erreur, le processus se termine avec un code
non nul et l'enveloppe ``{"error":{…}}`` est imprimée sur la sortie standard
pour rester analysable (par exemple avec ``jq``). Une réponse qui porte un
en-tête ``Direction-Version`` l'imprime sur la sortie d'erreur : c'est la
valeur que le geste suivant passe à ``--if-match``. ``call`` sert les gestes de
direction sans drapeau, comme la CLI, puisqu'il s'exécute en local.

.. _headless_mcp:

Outils pour un assistant IA (MCP)
=================================

blunderDB n'embarque aucun modèle de langage : il offre ses outils à
l'assistant que vous utilisez déjà (Claude Code, Claude Desktop, un client
local), par le *Model Context Protocol*. L'assistant cherche, lit et explique ;
blunderDB répond avec ses propres chiffres.

Les outils passent par les mêmes gestionnaires que ``/v1`` et ``call`` :

.. list-table::
   :header-rows: 1
   :widths: 30 70

   * - Outil
     - Ce qu'il rend
   * - ``database_overview``
     - comptes, période des matchs, version du schéma, joueurs fréquents
   * - ``search_positions``
     - positions d'une recherche dans la grammaire de la barre de commande
       (décrite dans l'outil), avec sa forme canonique
   * - ``search_comments``, ``saved_searches``
     - commentaires contenant des mots ; recherches enregistrées
   * - ``get_position``
     - une position, son analyse (meilleurs coups ou videau), le coup joué et
       le commentaire
   * - ``explain_error``
     - le thème de l'erreur, son coût en millipoints et la meilleure décision
   * - ``similar_positions``, ``decode_position``, ``legal_moves``,
       ``race_epc``
     - positions voisines ; lecture d'un XGID ; coups légaux ; EPC de course
   * - ``list_players``, ``player_aliases``, ``player_stats``,
       ``recurring_errors``, ``training_stats``
     - joueurs ; leurs alias et les graphies proposées ; PR global, pions,
       videau, par phase ; erreurs qui reviennent ; PR du quiz et rétention
       Anki contre le PR réel
   * - ``head_to_head``, ``pr_by_window``, ``player_ranking``
     - face-à-face de deux joueurs ; PR par fenêtre calendaire ; classement
       par PR
   * - ``list_matches``, ``get_match``, ``list_tournaments``
     - matchs, détail d'un match, tournois
   * - ``list_collections``, ``collection_positions``, ``study_decks``
     - collections et leurs positions ; paquets de révision
   * - ``list_lessons``, ``lesson``
     - leçons ; les étapes d'une leçon, dans l'ordre
   * - ``quiz_draw``, ``quiz_grade``
     - tire une position sans sa réponse, puis note la réponse donnée
   * - ``evaluate``
     - évaluation gammonNet d'une position donnée en texte, sans
       l'enregistrer : meilleurs coups ou décision de videau
   * - ``anki_next``
     - la prochaine carte due d'un paquet de révision
   * - ``transcribe_list``, ``transcribe_get``, ``transcribe_mat``
     - transcriptions de matchs ; détail d'une transcription ; son texte
       ``.mat``
   * - ``direction_list``, ``direction_standings``, ``direction_season``
     - tournois dirigés ; classement d'un tournoi ; classement de saison
   * - ``rollout``
     - rollout d'une position de la base : équité, intervalle à 95 % et JSD
       par candidat
   * - ``club_matches``, ``club_match_positions``, ``club_comments``,
       ``club_library``, ``club_library_positions``, ``club_ranking``
     - lectures à travers les tenants que liste le proxy
       (:ref:`headless_tenants_lus`) : matchs des élèves, positions d'un match
       avec leur hachage, commentaires écrits sur ces plateaux, bibliothèque
       partagée, classement de club ; sans ``X-Read-Tenants``, ce tenant seul

Seuls cinq outils écrivent — ``save_position``, ``comment_position``,
``create_collection``, ``add_to_collection`` et ``anki_review``, qui note une
carte tirée par ``anki_next`` — et ils ne sont offerts que sur demande :
``--write`` en local, ``--mcp-write`` sur le démon. Tous les autres ne font que
lire ; ``rollout`` gagne cependant, quand l'écriture est offerte, l'argument
``store``, qui enregistre le rollout à côté de l'analyse de la position. Aucun
outil n'efface.

**En local**, l'assistant lance ``blunderdb mcp`` sur un fichier (voir
:ref:`cli`). Pour Claude Code :

.. code-block:: bash

   claude mcp add blunderdb -- blunderdb mcp --db /chemin/vers/base.db

**Sur le démon**, les mêmes outils répondent en HTTP sur ``POST /mcp``
(transport *streamable HTTP*, sans session). Comme ``/v1``, ``/mcp`` exige
``X-Tenant-ID`` et chaque outil travaille dans ce tenant ; un programme qui
embarque ``pkg/blunderdb/server`` le sert aussi. Le démon n'authentifie
personne (ADR-0005) : ``/mcp`` se protège au proxy comme ``/v1``, et
``--mcp-write`` s'y décide comme ``--direction``. Chaque appel ``/v1`` que
fait un outil repasse par toute la chaîne du démon : il est journalisé, compté
dans les métriques et imputé à la limite de débit du tenant, en plus de la
requête ``/mcp`` qui le porte. Un appel d'outil coûte donc plusieurs requêtes ;
aucune n'en est exemptée.

Comme ``call``, ``blunderdb mcp`` migre le schéma d'une base ancienne à
l'ouverture, même sans ``--write``.
