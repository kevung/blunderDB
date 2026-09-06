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
   * - ``--rls``
     - ``false``
     - PostgreSQL : active la Row-Level Security par tenant (défense en
       profondeur, sur option)
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
``BLUNDERDB_TS_PATH``, ``BLUNDERDB_IDENTITY_DIR``, ``BLUNDERDB_OPS_ADDR``, ``BLUNDERDB_PPROF_ADDR``) :
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
métadonnées, statistiques, import et export. Les endpoints de listing
renvoient un flux NDJSON (un objet JSON par ligne). Le serveur s'arrête
proprement sur ``SIGINT`` / ``SIGTERM``.

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
* **Le reste, c'est ``/v2``.** Rendre obligatoire un champ qui ne l'était pas,
  changer une unité, changer le sens d'un code d'erreur : ce sont des ruptures,
  et elles vivent sous un autre préfixe, à côté de ``/v1``, le temps que les
  clients traversent.

Deux précisions qui ont leur importance. Les routes ``/ops/`` ne sont **pas**
couvertes : elles servent à l'exploitation d'un déploiement, changent avec lui,
et ne sont pas une API pour des programmes tiers. Et le **contrat lui-même est
généré** depuis la table de routes du démon (``openapi.yaml``,
:ref:`api_reference`) : il ne peut pas décrire autre chose que ce que le
serveur sert.

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
(``.xg``, ``.xgp``, ``.sgf``, ``.mat``, ``.bgf``, ``.txt``, ``.db``,
``.dbx``), et l'ensemble des imports simultanés — tous tenants confondus —
partage un quota global d'octets déposés sur disque : au-delà, un nouvel
import est refusé (``too many requests``) plutôt que de laisser croître sans
borne l'occupation de ``$TMPDIR``.

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
quel par le poste de travail ; l'export porte toujours toute la base, la
sélection d'un sous-ensemble est un geste du bureau ou de la CLI. Le corps JSON
de la requête est optionnel et n'accepte que ``watermarkOrigin`` /
``watermarkNote``, pour apposer un filigrane signé de l'identité propre du
démon (``--identity-dir``) — sans ces champs, l'export ne porte aucun
filigrane ; les demander sans identité configurée échoue avec le code
``invalid``.

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
   FSRS écartent. La méthode mesure désormais, sans jamais écrire.

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
l'évènement final de chacune des deux routes porte désormais la répartition
``evaluated``/``refused``/``failed`` : une position que gammonNet refuse
d'évaluer (un score de match hors de la portée de sa table, une décision de
videau que le modèle refuse) compte comme ``refused``, pas ``failed`` — elle
n'est jamais retentée en vain sur la passe suivante, contrairement à une
position réellement en échec.

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

La réponse JSON (ou le flux NDJSON pour les endpoints ``*.list``) est écrite
sur la sortie standard. En cas d'erreur, le processus se termine avec un code
non nul et l'enveloppe ``{"error":{…}}`` est imprimée sur la sortie standard
pour rester analysable (par exemple avec ``jq``).
