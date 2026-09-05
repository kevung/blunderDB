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

   # Servir une base SQLite locale sur le port 8080
   blunderdb serve --db ma_base.db --addr :8080

   # Servir un backend PostgreSQL
   blunderdb serve --backend postgres \
       --dsn "postgres://user:pass@host:5432/blunderdb?sslmode=disable" \
       --addr :8080

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
     - base de bearoff two-sided (``.bd``) optionnelle élargissant la base
       intégrée TS-06-06 pour l'analyse de course du point d'accès EPC ;
       le démon ne télécharge jamais de base lui-même — monter le fichier
       en volume et le désigner ici
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
* ``GET /metrics`` — métriques Prometheus (si ``--metrics`` est actif).

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

   blunderdb healthcheck --addr 127.0.0.1:8080 && echo "démon disponible"

La surface métier suit le schéma ``POST /v1/<famille>.<méthode>`` (par exemple
``/v1/positions.save``, ``/v1/matches.get``). Les familles couvrent les
positions, analyses, matchs, commentaires, collections, tournois, cartes Anki,
filtres, sessions, historique (recherche et commandes), recherche,
métadonnées, statistiques, import et export. Les endpoints de listing
renvoient un flux NDJSON (un objet JSON par ligne). Le serveur s'arrête
proprement sur ``SIGINT`` / ``SIGTERM``.

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
que le proxy puisse refuser les deux d'une seule règle. **Ne jamais exposer
``/ops/`` par le proxy public.**

L'option ``--ops-addr <hôte:port>`` va plus loin : les deux routes quittent
alors l'adresse ``--addr`` et ne sont plus servies que sur ce second
écouteur, à lier sur une interface d'administration. Sans cette option, elles
restent sur l'écouteur principal et c'est au proxy de les bloquer.

Ces routes exigent l'en-tête ``X-Tenant-ID`` comme toutes les autres — une
purge nomme le tenant qu'elle détruit, elle en a besoin plus que quiconque.
Seules les sondes (``/healthz``, ``/readyz``) et ``/metrics`` s'en passent.

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
quel par le poste de travail ; il n'existe pas encore d'export partiel côté
serveur (cette sélection est un geste du bureau ou de la CLI). Le corps JSON
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

Trois gestes, selon ce qu'on veut récupérer.

**Tout, sous PostgreSQL** — ``pg_dump`` est l'outil, et blunderDB n'a rien à
ajouter :

.. code-block:: bash

   pg_dump --format=custom --file=blunderdb.dump "postgres://…"
   pg_restore --dbname="postgres://…" blunderdb.dump

**Un tenant seul** — ``/v1/exports.sqlite`` écrit la base d'un tenant dans un
fichier ``.db`` ordinaire, celui que l'application de bureau ouvre :

.. code-block:: bash

   curl -X POST http://127.0.0.1:8080/v1/exports.sqlite \
     -H "X-Tenant-ID: 42" -o tenant-42.db

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

Déploiement avec Docker
-----------------------

Le dépôt fournit un ``Dockerfile.serve`` qui construit une image conteneur
minimale du démon : seul le binaire ``serve`` est compilé (Go pur, sans
interface graphique et sans CGO, donc lié statiquement), puis placé dans une
image *distroless*.

.. code-block:: bash

   # Construire l'image (depuis la racine du dépôt)
   docker build -f Dockerfile.serve -t blunderdb-serve .

   # Lancer le démon (le backend par défaut de l'image est postgres)
   docker run --rm -p 8080:8080 \
       -e BLUNDERDB_DSN="postgres://user:pass@hôte:5432/blunderdb?sslmode=disable" \
       blunderdb-serve

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
le numéro de version (par exemple ``0.34.0``), figé à jamais, et ``latest``,
qui suit la dernière version publiée. L'image est fournie pour ``linux/amd64``
et ``linux/arm64`` ; Docker choisit l'architecture de l'hôte.

.. code-block:: bash

   # Récupérer l'image d'une version donnée (recommandé en production)
   docker pull ghcr.io/kevung/blunderdb-serve:0.34.0

   # Lancer le démon avec un backend PostgreSQL
   docker run --rm -p 127.0.0.1:8080:8080 \
       -e BLUNDERDB_DSN="postgres://user:pass@hôte:5432/blunderdb?sslmode=disable" \
       ghcr.io/kevung/blunderdb-serve:0.34.0

   # Ou avec une base SQLite persistée dans un volume
   docker run --rm -p 127.0.0.1:8080:8080 -v blunderdb-data:/data \
       -e BLUNDERDB_BACKEND=sqlite -e BLUNDERDB_DSN=/data/blunderdb.db \
       ghcr.io/kevung/blunderdb-serve:0.34.0

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
   ci-dessus publient le port sur ``127.0.0.1`` seulement pour cette raison.

.. _headless_proxy_deployment:

Déploiement derrière un proxy authentifiant
--------------------------------------------

L'`ADR-0005 <https://github.com/kevung/blunderDB/blob/main/docs/adr/0005-serve-daemon-delegates-authentication.md>`__
fait du reverse-proxy **toute** la frontière de sécurité du démon :
lui seul authentifie l'appelant, lui seul a le droit de poser l'en-tête
``X-Tenant-ID``, et il doit **retirer** systématiquement toute valeur envoyée
par le client avant d'y injecter le tenant authentifié — sans quoi n'importe
qui peut se faire passer pour n'importe quel tenant en le nommant lui-même.
Le dépôt fournit un exemple complet sous ``deploy/`` :

* ``deploy/docker-compose.yml`` — Caddy (authentification HTTP Basic de
  démonstration) devant ``blunderdb-serve`` et PostgreSQL (Row-Level Security
  activée) ; seul Caddy publie un port, ``blunderdb-serve`` et PostgreSQL
  vivent sur un réseau Docker interne (``internal: true``) qui n'a de route
  ni vers l'hôte ni vers l'Internet ;
* ``deploy/Caddyfile`` — la configuration de Caddy : authentifie, associe le
  compte authentifié à l'entier du tenant (``map``), puis l'injecte dans
  ``X-Tenant-ID`` après avoir explicitement effacé toute valeur reçue du
  client (garde ``header_up X-Tenant-ID ""`` suivie de l'injection) ;
* ``deploy/nginx-tenant-proxy.conf`` — le même schéma en extrait nginx
  (``proxy_set_header X-Tenant-ID ""`` puis ``proxy_set_header X-Tenant-ID
  $tenant_id``), pour qui a déjà un nginx en place ;
* ``deploy/README.md`` — le modèle de menace en trois phrases et ce qu'il ne
  faut jamais faire (exposer le démon nu).

L'authentification HTTP Basic du ``Caddyfile`` est une démonstration, pas une
recommandation de production : elle se remplace par ``forward_auth`` vers un
fournisseur d'identité réel (OIDC, SSO d'entreprise…), qui authentifie puis
transmet l'identité au même endroit du fichier.

**Scénario complet, de zéro à un démon qui répond :**

.. code-block:: bash

   cd deploy
   cp .env.example .env    # puis y définir POSTGRES_PASSWORD
   docker compose up -d --build

   # Sans authentification : rejeté avant même d'atteindre le démon.
   curl -i http://localhost:8080/v1/metadata.counts -d '{}'
   # HTTP/1.1 401 Unauthorized

   # Authentifié en tant qu'« alice », qui est mappée au tenant 1 : le démon
   # répond, quel que soit l'en-tête que le client tente d'envoyer lui-même.
   curl -u alice:demo-password http://localhost:8080/v1/metadata.counts -d '{}'
   curl -u alice:demo-password -H "X-Tenant-ID: 999" \
        http://localhost:8080/v1/metadata.counts -d '{}'
   # {"positions":0,"analyses":0,"matches":0,...} dans les deux cas — le journal
   # du démon (docker compose logs blunderdb-serve) montre tenant=1 pour les
   # deux requêtes : la valeur 999 envoyée par le client n'a jamais survécu à
   # la garde du Caddyfile.

   docker compose down -v

Ce scénario a été rejoué tel quel : ``blunderdb-serve`` et PostgreSQL
démarrent, Caddy authentifie et route, et le journal du démon confirme que
seul l'entier injecté par le proxy — jamais celui du client — atteint la
couche applicative.

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
compte à son entier, le démon ne devine jamais. L'option ``--rls`` active en complément la **Row-Level
Security** de PostgreSQL : des politiques d'isolation par tenant sont
installées et ``app.tenant_id`` est fixé par connexion. C'est une défense en
profondeur facultative, désactivée par défaut.

Quand un tenant est décommissionné, ``POST /v1/tenant.purge`` supprime
définitivement toutes ses données (positions, matchs, collections, historique,
etc.) sur le tenant courant (celui porté par ``X-Tenant-ID``), **ainsi que son
état de session** (dernière recherche, dernière position, onglets ouverts —
les lignes de la table ``session_state`` portant ce tenant) : l'opération
s'exécute dans une seule transaction, est idempotente (aucune erreur à purger
un tenant déjà vide ou à répéter l'appel) et n'affecte aucun autre tenant ni la
ligne globale de version de schéma. Elle n'est disponible qu'avec le backend
PostgreSQL — elle renvoie une erreur ``invalid`` sur un backend SQLite, qui n'a
pas de notion de tenant.

Symétriquement, ``POST /v1/maintenance.vacuum`` compacte le fichier SQLite du
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
       --from sqlite:///chemin/vers/base.db \
       --to   "postgres://user:pass@host:5432/db?sslmode=disable" \
       --tenant-id 42

   # Prévisualiser sans rien écrire
   blunderdb migrate --from sqlite:///chemin/vers/base.db \
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
     - base SQLite source (``sqlite:///chemin`` ou un simple chemin)
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

   # Lister toutes les méthodes disponibles
   blunderdb call --list

   # Lectures
   blunderdb call metadata.counts --db ma_base.db
   blunderdb call positions.list  --db ma_base.db --json '{"limit":10}'
   blunderdb call matches.get     --db ma_base.db --json '{"id":1}'

   # Écritures
   blunderdb call positions.save  --db ma_base.db --json '{"position":{...}}'
   blunderdb call matches.delete  --db ma_base.db --json '{"id":42}'

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
