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
     - active CORS pour cette origine (désactivé par défaut)
   * - ``--rate-limit-rps <n>``
     - ``0``
     - limite de requêtes par seconde et par tenant (0 = désactivé)
   * - ``--rate-limit-burst <n>``
     - ``2×rps``
     - taille du seau de jetons pour les pics de requêtes
   * - ``--rls``
     - ``false``
     - PostgreSQL : active la Row-Level Security par tenant (défense en
       profondeur, sur option)

La plupart des options peuvent aussi être fournies par variable
d'environnement (``BLUNDERDB_BACKEND``, ``BLUNDERDB_DSN``, ``BLUNDERDB_ADDR``,
``BLUNDERDB_LOG_LEVEL``, ``BLUNDERDB_RLS``).

Points d'accès
--------------

Le service expose des points d'accès d'exploitation, toujours présents :

* ``GET /healthz`` — vivacité (le processus tourne) ;
* ``GET /readyz`` — disponibilité (le stockage répond) ;
* ``GET /metrics`` — métriques Prometheus (si ``--metrics`` est actif).

La surface métier suit le schéma ``POST /v1/<famille>.<méthode>`` (par exemple
``/v1/positions.save``, ``/v1/matches.get``). Les familles couvrent les
positions, analyses, matchs, commentaires, collections, tournois, cartes Anki,
filtres, sessions, recherche, métadonnées, statistiques et import. Les
endpoints de listing renvoient un flux NDJSON (un objet JSON par ligne). Le
serveur s'arrête proprement sur ``SIGINT`` / ``SIGTERM``.

Deux méthodes de la famille ``positions`` décodent une position sans
l'enregistrer : ``positions.fromXGID`` reconstruit une position à partir d'une
chaîne XGID, et ``positions.fromXGP`` à partir d'un fichier de position unique
``.xgp``.

La famille ``anki`` gagne six méthodes qui étendent le planificateur à
répétition espacée (FSRS) : ``anki.reviewLog`` (journal de chaque révision —
notation et résultat FSRS — pour les statistiques de rétention et un
historique fidèle), ``anki.forecast`` (projection du nombre de cartes dues sur
les prochains jours, cartes en retard comprises), ``anki.suspendCard`` /
``anki.buryCard`` / ``anki.removeCard`` (retirer une carte de la file de
révision temporairement ou définitivement) et ``anki.optimizeParams`` (ajuste
le taux de rétention visé d'un paquet vers le taux de réussite observé sur ses
révisions).

.. _headless_postgres:

Backend PostgreSQL et multi-utilisateurs
========================================

Pour un déploiement partagé, blunderDB peut stocker les données dans
**PostgreSQL** plutôt que dans un fichier SQLite. Le backend est sélectionné
par ``--backend postgres`` et la chaîne de connexion ``--dsn``. Le schéma est
créé et migré automatiquement au démarrage.

Les données sont **cloisonnées par tenant** (locataire) : chaque requête porte
un identifiant de scope (en-tête ``X-Tenant-ID``, par défaut ``default``), ce
qui permet à plusieurs utilisateurs de partager la même instance sans voir les
données des autres. L'option ``--rls`` active en complément la **Row-Level
Security** de PostgreSQL : des politiques d'isolation par tenant sont
installées et ``app.tenant_id`` est fixé par connexion. C'est une défense en
profondeur facultative, désactivée par défaut.

Quand un tenant est décommissionné, ``POST /v1/tenant.purge`` supprime
définitivement toutes ses données (positions, matchs, collections, historique,
etc.) sur le tenant courant (celui porté par ``X-Tenant-ID``), **ainsi que son
état de session** (dernière recherche, dernière position, onglets ouverts —
les quelques lignes ``metadata`` préfixées par ce scope) : l'opération
s'exécute dans une seule transaction, est idempotente (aucune erreur à purger
un tenant déjà vide ou à répéter l'appel) et n'affecte aucun autre tenant ni la
ligne globale de version de schéma. Elle n'est disponible qu'avec le backend
PostgreSQL — elle renvoie une erreur ``invalid`` sur un backend SQLite, qui n'a
pas de notion de tenant.

.. _headless_migrate:

Migrer une base SQLite vers PostgreSQL
======================================

``blunderdb migrate`` copie une base SQLite mono-utilisateur vers un backend
PostgreSQL, sous un scope de tenant choisi — c'est le chemin pour « téléverser »
une bibliothèque de bureau vers un déploiement serveur.

.. code-block:: bash

   blunderdb migrate \
       --from sqlite:///chemin/vers/base.db \
       --to   "postgres://user:pass@host:5432/db?sslmode=disable" \
       --tenant-id mon-tenant

   # Prévisualiser sans rien écrire
   blunderdb migrate --from sqlite:///chemin/vers/base.db \
       --tenant-id mon-tenant --dry-run

La migration copie les **positions, leurs analyses et commentaires, les matchs
(parties + coups), les tournois (avec leurs liens de match) et les collections
(avec leur composition)**, en réattribuant les clés primaires et étrangères, le
tout dans une **seule transaction** côté destination : l'opération est atomique
(un échec laisse la destination intacte, il suffit de relancer). La progression
et le bilan final sont émis en NDJSON sur la sortie standard.

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
   * - ``--tenant-id <scope>``
     - –
     - scope de tenant de destination (obligatoire sauf en ``--dry-run``)
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
   * - ``--scope <chaîne>``
     - ``default``
     - scope de tenant (envoyé comme ``X-Tenant-ID``)
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
