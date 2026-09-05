# Persona 4 — Administratrice de club qui déploie le mode serveur

## Qui je suis (3 lignes)

Nadia, 45 ans, trésorière d'un club de 40 membres et administratrice système à
mi-temps : Docker, nginx, un VPS Debian, pas de Kubernetes et pas d'équipe.
Le club veut une bibliothèque de positions par membre et un coach qui voit les
matchs des élèves ; j'ai une heure pour décider si c'est déployable et sûr, et
un week-end pour le faire.

## Parcours suivi (liste ordonnée des pages/sections lues)

1. Mode headless (serveur) › Vue d'ensemble
2. Mode headless › Le démon serve (tableau des options, variables d'environnement)
3. Mode headless › Points d'accès ; Les bases de bearoff ; Les routes d'exploitation
4. Mode headless › Un seul tenant sur SQLite ; Sauvegarde et restauration
5. Mode headless › Déploiement avec Docker › Image publiée
6. Mode headless › Déploiement derrière un proxy authentifiant
7. Mode headless › Backend PostgreSQL et multi-utilisateurs ; Migrer une base SQLite vers PostgreSQL
8. Contrat d'API (tableau des familles, section Idempotence)
9. Téléchargement et installation › Image Docker (mode serveur)
10. Interface en ligne de commande (CLI) › Commandes disponibles ; healthcheck ; vacuum ; Sauvegarde régulière
11. Guide utilisateur › Déployer le mode serveur derrière un proxy
12. Foire aux questions › « blunderDB propose-t-il un mode serveur ? », « Où sont stockées mes données ? »
13. Glossaire › Tenant, Filigrane, Identité d'émetteur
14. Annexe : Schéma de la base de données
15. Historique des versions › 0.34.0, 0.35.0, 0.36.0

## Ce que j'ai trouvé en cinq minutes

Beaucoup, et de bonne qualité. La page headless est dense mais elle est écrite
par quelqu'un qui a déjà exploité un service : sondes de vivacité et de
disponibilité distinguées et *justifiées* (« un orchestrateur redémarre le
conteneur dont la vivacité échoue, et une base momentanément injoignable ne doit
pas relancer en boucle un démon sain »), `HEALTHCHECK` embarqué parce que
l'image distroless « n'a ni curl ni shell », limite de débit active par défaut
« pour qu'un fichier compose qui ne pense qu'à la base de données n'hérite pas
d'un démon sans aucune limite », arrêt gracieux qui annule les imports en cours,
métriques de travail en vol, pool PostgreSQL réglable. Le contrat d'API est
engendré du code. Le ton ne me vend rien : il me dit ce que la chose fait.

L'avertissement d'authentification, lui, m'a sauté aux yeux : encart
« Avertissement » juste sous le premier exemple de lancement, répété en encart
sous les exemples Docker, répété dans le tutoriel du guide, répété dans la FAQ.
Sur ce point précis, la réponse à la question 1 du cahier des charges est oui.

## Mes huit questions (réponse trouvée › page › section, ou « absente »)

1. **Ce qui est exposé sur le réseau** › Mode headless › Points d'accès + Les
   routes d'exploitation. Trouvé et complet : `/healthz`, `/readyz`, `/metrics`,
   `/v1/<famille>.<méthode>`, `/ops/`, `--pprof-addr`. Mais le préfixe des deux
   routes dangereuses est donné **deux fois de façon contradictoire** (constat 1).
2. **Qui authentifie** › Mode headless › Le démon serve (encart) et
   Déploiement derrière un proxy authentifiant. Trouvé, sans ambiguïté :
   « L'ADR-0005 fait du reverse-proxy toute la frontière de sécurité du démon ».
3. **Ce qui se passe si j'oublie le proxy** › partiellement. Le texte dit « Ne
   l'exposez jamais directement sur l'Internet public » mais ne décrit jamais la
   conséquence concrète : n'importe qui pose `X-Tenant-ID: 7` et lit la
   bibliothèque du membre 7, ou appelle la purge. Je l'ai déduite ; ce n'est pas
   écrit en une phrase.
4. **Comment séparer les membres** › Mode headless › Backend PostgreSQL et
   multi-utilisateurs. Le principe est clair (un entier par membre, associé par
   le proxy). La **recette** pour 40 membres sous nginx : absente du site
   (constat 4).
5. **Comment sauvegarder** › Mode headless › Sauvegarde et restauration.
   `pg_dump` / `pg_restore`, export par tenant, remise en place par `migrate` :
   trouvé. Sauvegarde du volume SQLite d'un conteneur : absente (constat 10).
6. **Comment mettre à jour** › absente. « Le schéma est créé et migré
   automatiquement au démarrage » est tout ce que j'ai. Aucune procédure, aucun
   « sauvegardez d'abord », aucun retour arrière (constat 9).
7. **Ce que fait l'image Docker** › Mode headless › Déploiement avec Docker +
   Image publiée, et Téléchargement › Image Docker. Trouvé et bien fait
   (distroless, statique, étiquettes OCI, deux architectures, healthcheck).
   Manque le répertoire de données et ce qu'il faut persister (constat 7).
8. **Ce que l'application de bureau des membres peut faire avec le serveur** ›
   **absente**. Le manuel ne mentionne pas une seule fois le mode serveur
   (constat 14).

Mon compose et ma config nginx, mentalement : le compose, je ne peux pas
l'écrire à partir du site — il est décrit, jamais montré (constat 2). La config
nginx, je peux l'écrire pour **un** tenant, et telle qu'elle est publiée elle ne
démarre pas (constat 3).

## Où je me suis égarée

- **Le préfixe des routes d'exploitation.** J'ai lu « Les routes
  d'exploitation » et noté ma règle nginx de refus sur `/ops/`. Six écrans plus
  bas, la même page m'a expliqué la purge sous `POST /v1/tenant.purge`. J'ai dû
  ouvrir le Contrat d'API pour trancher. Sur une règle de refus, se tromper de
  préfixe, c'est ne rien refuser du tout.
- **`deploy/`.** « Le dépôt fournit un exemple complet sous `deploy/` », puis un
  scénario qui commence par `cd deploy`. Je suis partie chercher ce dossier dans
  l'image Docker que je venais de tirer. Il est dans le dépôt git, qu'aucune
  phrase ne me dit de cloner et dont aucun lien ne m'est donné.
- **`--data-dir`.** Deux exemples de la section bearoff le passent à `serve`.
  Je l'ai cherché dans le tableau des options : il n'y est pas.
- **La CLI.** Je suis allée sur « Interface en ligne de commande » pour trouver
  `serve` et `migrate` : le tableau « Commandes disponibles » ne les contient
  pas, alors qu'il contient `healthcheck`.

## Ce qui a entamé ma confiance

- Une page qui se contredit elle-même sur le chemin d'une route destructrice.
- Un tutoriel dont la configuration nginx ne peut pas démarrer telle quelle
  (`listen 443 ssl;` sans `ssl_certificate`).
- Un encart qui affirme « Les exemples ci-dessus publient le port sur 127.0.0.1
  seulement pour cette raison » alors que le premier exemple `docker run` de la
  page publie `-p 8080:8080`.
- Une phrase de sécurité rendue avec ses accents graves de balisage :
  « Ne jamais exposer ``/ops/`` par le proxy public. » Si la coquille est sur
  *cette* phrase-là, je me demande ce que la relecture a couvert.
- « depuis la 0.37.0 » dans une page publiée alors que l'historique s'arrête à
  0.36.0. Je ne sais plus si je lis la doc de ce que j'installe.
- `sslmode=disable` dans les cinq chaînes de connexion PostgreSQL de la doc,
  jamais commenté.

À l'inverse, deux choses l'ont renforcée : « Ce scénario a été rejoué tel quel »
sur la démonstration Caddy, et l'aveu franc que l'authentification Basic du
Caddyfile « est une démonstration, pas une recommandation de production ».

## Ce qui manque

Ce que je n'ai pas trouvé et qui existe manifestement dans le produit : le
répertoire de données du démon dans le conteneur ; la recette nginx
multi-tenant (elle est dans `deploy/nginx-tenant-proxy.conf`, hors site) ; le
`docker-compose.yml` de référence (idem) ; le contenu de l'ADR-0005, cité trois
fois par son numéro et jamais résumé ailleurs que par la phrase qu'il justifie.

Ce qui n'existe pas et que la doc ne dit pas ne pas exister : un accès
inter-tenant. Mon coach ne peut pas voir les bibliothèques de ses élèves — le
glossaire pose que « Rien de ce qu'un tenant stocke n'est jamais visible à un
autre » —, et aucune page ne tire la conséquence ni ne propose le contournement
(un compte coach mappé sur le tenant de l'élève, ou un export). J'ai passé
vingt minutes à chercher une fonctionnalité qui n'est pas là.

Ce qui manque tout court : une procédure de mise à jour, une procédure de
sauvegarde du déploiement SQLite conteneurisé, un mot sur le cycle de vie d'un
tenant (comment il naît, comment on liste ceux qui existent), et un
dimensionnement même grossier (RAM, CPU, disque par membre).

## Constats

| # | Constat | Page › section | Gravité | Proposition |
|---|---|---|---|---|
| 1 | La même page donne deux chemins pour les routes destructrices : « /ops/tenant.purge (backend PostgreSQL) détruit les données d'un tenant » puis, plus bas, « Quand un tenant est décommissionné, POST /v1/tenant.purge supprime définitivement toutes ses données » (idem pour `maintenance.vacuum`). Le Contrat d'API tranche pour `/ops/`. Une règle de refus écrite sur le mauvais préfixe ne refuse rien. | Mode headless › Les routes d'exploitation vs Backend PostgreSQL et multi-utilisateurs | bloquant | Corriger les deux occurrences `/v1/` en `/ops/` et ajouter dans « Les routes d'exploitation » la règle nginx et Caddy de refus, écrite en toutes lettres. |
| 2 | Le déploiement de référence est décrit sans jamais être montré : « Le dépôt fournit un exemple complet sous deploy/ », puis « cd deploy » sans qu'aucune phrase ne dise de cloner le dépôt ni ne donne de lien. Et `docker compose up -d --build` reconstruit l'image, ce que « Image publiée » venait de me dire inutile. | Mode headless › Déploiement derrière un proxy authentifiant | bloquant | Reproduire le `docker-compose.yml` et le `Caddyfile` dans la page (ils font quelques dizaines de lignes), avec des liens permanents vers le dépôt, et une variante qui tire `ghcr.io/kevung/blunderdb-serve:<version>` au lieu de construire. |
| 3 | La configuration nginx du tutoriel ne démarre pas : « listen 443 ssl; » sans `ssl_certificate` ni `ssl_certificate_key`. Aucune page du site ne parle d'obtenir ou de configurer un certificat. | Guide utilisateur › Déployer le mode serveur derrière un proxy, étape 3 | bloquant | Compléter le bloc `server` (certificat, redirection 80→443) ou le réduire à `listen 80;` avec une phrase nommant explicitement le TLS comme prérequis à ajouter. |
| 4 | Le seul exemple nginx rendu sur le site fige « proxy_set_header X-Tenant-ID "1"; » : il ne couvre qu'un tenant. Le schéma multi-tenant n'existe sur le site que sous forme de description (« proxy_set_header X-Tenant-ID "" puis proxy_set_header X-Tenant-ID $tenant_id »), le fichier réel étant hors ligne. Un club de 40 membres n'a aucune recette. | Guide utilisateur › Déployer le mode serveur derrière un proxy ; Mode headless › Déploiement derrière un proxy authentifiant | bloquant | Ajouter au tutoriel une seconde étape « plusieurs membres » : bloc `map $remote_user $tenant_id`, effacement explicite de l'en-tête client, et la conduite à tenir pour un utilisateur non mappé (refuser, jamais retomber sur 1). |
| 5 | Le premier exemple Docker publie le port sur toutes les interfaces — « docker run --rm -p 8080:8080 » — et l'encart d'avertissement, quarante lignes plus bas, affirme « Les exemples ci-dessus publient le port sur 127.0.0.1 seulement pour cette raison ». Un administrateur pressé copie le premier bloc et expose son démon. | Mode headless › Déploiement avec Docker | bloquant | Passer ce `docker run` à `-p 127.0.0.1:8080:8080` comme les suivants, pour que l'encart dise vrai. |
| 6 | Le tout premier exemple de la page, « blunderdb serve --db ma_base.db --addr :8080 », écoute sur toutes les interfaces. L'avertissement qui suit interdit l'exposition publique sans dire de replier l'écoute sur la boucle locale. | Mode headless › Le démon serve | gênant | Écrire les deux exemples en `--addr 127.0.0.1:8080` et ajouter à l'encart « et liez l'écoute à 127.0.0.1 : le proxy est sur la même machine ». |
| 7 | « --data-dir » apparaît dans deux exemples (« blunderdb serve --data-dir /srv/bearoff ») mais ne figure ni dans le tableau des options de `serve`, ni dans la liste des variables d'environnement. Impossible de savoir où le conteneur écrit ses tables de bearoff ni quel volume monter pour ne pas repayer le calcul à chaque redémarrage. | Mode headless › Le démon serve (tableau) ; Les bases de bearoff | gênant | Ajouter `--data-dir` au tableau, avec sa variable d'environnement et le chemin par défaut dans l'image, et une phrase dans « Image publiée » sur le volume à monter. |
| 8 | La Row-Level Security tient en une phrase : « des politiques d'isolation par tenant sont installées et app.tenant_id est fixé par connexion ». Rien sur le rôle PostgreSQL à utiliser — un rôle superutilisateur ou `BYPASSRLS` contourne silencieusement toutes les politiques —, rien sur l'activation après coup sur une base existante, rien sur le coût. | Mode headless › Backend PostgreSQL et multi-utilisateurs | gênant | Trois puces sous l'option : le rôle de connexion doit être non-superutilisateur et sans BYPASSRLS ; ce qui se passe si `--rls` est activé sur une base déjà peuplée ; l'ordre de grandeur du surcoût mesuré. |
| 9 | Aucune procédure de mise à jour. « Le schéma est créé et migré automatiquement au démarrage » est la seule phrase ; l'annexe schéma prévient pourtant qu'« une base migrée vers un schéma récent ne peut plus être ouverte par des versions plus anciennes ». Que faire quand `/readyz` répond `version_mismatch` n'est écrit nulle part. | Mode headless › Backend PostgreSQL ; Annexe : Schéma de la base de données | gênant | Une section « Mettre à jour un déploiement » : sauvegarder d'abord, tirer l'étiquette de version (jamais `latest` en production), redémarrer, vérifier `/readyz`, et la mention explicite que la migration est à sens unique. |
| 10 | La sauvegarde ne couvre que PostgreSQL (`pg_dump`) et l'export par tenant. Rien sur le déploiement SQLite en conteneur pourtant proposé deux sections plus haut : sauvegarder le volume `blunderdb-data`, et le fait que le fichier est en WAL (une copie à chaud du seul `.db` n'est pas fiable). La recette `curl … 127.0.0.1:8080 -H "X-Tenant-ID: 42"` ne fonctionne que depuis l'hôte, court-circuitant le proxy — ce n'est pas dit. | Mode headless › Sauvegarde et restauration | gênant | Ajouter un troisième geste « le volume SQLite », préciser que la commande `curl` s'exécute sur l'hôte du démon, et donner l'équivalent passant par le proxy authentifié. |
| 11 | Les cinq chaînes de connexion PostgreSQL de la documentation portent `sslmode=disable`, sans un mot, alors que le déploiement de référence met la base dans un autre conteneur et qu'un VPS a souvent sa base ailleurs. | Mode headless › Le démon serve, Déploiement avec Docker, Migrer une base SQLite vers PostgreSQL | gênant | Une note sous le premier exemple : `sslmode=disable` ne convient qu'à un réseau privé de confiance ; nommer `require` / `verify-full` pour une base distante. |
| 12 | « Seules les sondes (/healthz, /readyz) et /metrics s'en passent » [de l'en-tête tenant] : `/metrics` est donc lisible par quiconque atteint le démon, et il publie la taille de la base et le travail en cours. La consigne de ne pas l'exposer par le proxy public n'est écrite nulle part, contrairement à celle sur `/ops/`. | Mode headless › Les routes d'exploitation ; Corrélation et métriques métier | gênant | Ajouter `/metrics` (et `--pprof-addr`) à la phrase qui dit quoi ne pas exposer, et le montrer dans l'exemple nginx. |
| 13 | Le cycle de vie d'un tenant n'est décrit nulle part : comment un tenant vient à l'existence (au premier écrit ? faut-il le créer ?), comment lister ceux qui existent, si son entier est réutilisable après purge. Pour ouvrir 40 comptes, je ne sais pas si j'ai un geste à faire côté serveur ou seulement côté proxy. | Mode headless › Backend PostgreSQL et multi-utilisateurs | gênant | Un paragraphe « Ouvrir et fermer un tenant » : il n'y a rien à créer, l'entier suffit ; ce que voit un tenant vide ; ce que `tenant.purge` laisse. |
| 14 | Ce que l'application de bureau d'un membre peut faire avec le serveur n'est écrit nulle part : le manuel ne mentionne pas une seule fois le mode serveur, et le seul chemin réel (`exports.sqlite` vers un `.db` que « l'application de bureau ouvre », `migrate` pour le renvoyer) est rangé sous « Sauvegarde et restauration ». Le lecteur doit deviner que l'application de bureau ne se connecte pas au serveur. | Manuel (absence) ; Mode headless › Sauvegarde et restauration | gênant | Une sous-section « Le poste de travail et le serveur » dans la page headless : l'application de bureau ouvre des fichiers, pas des URL ; les deux gestes d'aller-retour ; et un renvoi depuis le manuel. |
| 15 | Le cas d'usage « un coach voit les matchs de ses élèves » n'a pas de réponse. Le glossaire pose l'isolation absolue (« Rien de ce qu'un tenant stocke n'est jamais visible à un autre ») mais aucune page ne dit qu'il n'y a donc pas d'accès inter-tenant ni ce qu'on fait à la place. | Glossaire › Tenant ; Mode headless › Backend PostgreSQL | gênant | Une phrase explicite : il n'existe pas de lecture inter-tenant ; les contournements sont un compte du proxy mappé sur le tenant à consulter, ou l'export d'un `.db` par le membre. |
| 16 | La FAQ annonce que le mode serveur « sert par exemple à consulter ou importer des matchs depuis un navigateur ». Il n'y a aucune interface web : le démon répond en JSON. J'ai ouvert un navigateur sur le port avant de comprendre. | FAQ › blunderDB propose-t-il un mode serveur ? | mineur | Remplacer par « à piloter blunderDB depuis vos propres scripts ou une application maison, en HTTP + JSON (il n'y a pas d'interface web) ». |
| 17 | Dérive de version : « depuis la 0.37.0 comments.listAll, tournaments.list et collections.positions » alors que l'historique s'arrête à 0.36.0 ; et les étiquettes d'image sont épinglées à 0.34.0 dans la page headless, 0.35.0 dans le guide, `x.y.z` sur la page de téléchargement. | Mode headless › Les routes d'exploitation, Image publiée ; Guide utilisateur | mineur | Ne documenter que le publié, et n'épingler qu'une seule forme d'étiquette (`x.y.z` ou la version courante) dans toutes les pages. |
| 18 | Le tableau « Commandes disponibles » de la page CLI ne contient ni `serve`, ni `migrate`, ni `call`, alors qu'il contient `healthcheck` (« Interroge un démon serve en marche ») et que la page headless renvoie vers la CLI pour le détail. Et « Sauvegarde régulière » n'y parle que du fichier local. | Interface en ligne de commande (CLI) › Commandes disponibles | mineur | Ajouter les trois lignes au tableau avec un renvoi « voir Mode headless (serveur) », plutôt que de laisser croire que la liste est complète. |
| 19 | Le balisage fuit sur la phrase de sécurité la plus importante de la page : « Ne jamais exposer ``/ops/`` par le proxy public. » Les accents graves sont rendus tels quels. | Mode headless › Les routes d'exploitation | mineur | Corriger le balisage (les mêmes doubles accents graves visibles apparaissent aussi trois fois dans la page CLI). |
| 20 | L'ADR-0005 est invoqué trois fois par son numéro comme fondement de tout le modèle de sécurité, sans jamais être résumé ni lié depuis le site ; le glossaire définit Tenant, Filigrane et Identité d'émetteur mais ni RLS, ni purge, ni vidange, ni `/ops/`. | Mode headless › Déploiement derrière un proxy authentifiant ; Glossaire | mineur | Résumer l'ADR-0005 en trois phrases dans la page (le modèle de menace) avec un lien vers le dépôt, et ajouter RLS et purge au glossaire. |
