<!-- Lot du plan tasks/plan-amelioration-2026-09b/README.md. -->

# Lot G — Serveur headless, PostgreSQL, pont GUI Go

État vérifié le 2026-09-02 : 135 routes `/v1` + 3 ops ; 22 documentées ; 31
avec un test HTTP sémantique (`routes_smoke_test` couvre le câblage des 135) ;
RLS sur 16/17 tables ; export unifié et pagination `listIds`/`loadByIds`
livrés ; `docker-serve` réel en CI. Points forts : cardinalité des métriques
testée, `databaseParity`, `index_parity_test`, `resumableDownload`, ADR-0015
tenu. Les deux critiques (tenant 0, `metadata`), les fuites d'erreurs,
`/healthz` sont au lot A.

G.1 à G.7 = **étape 1** ; G.8 à G.14 = **étape 2**.

---

## G.1 — Un exemple complet de déploiement derrière un proxy authentifiant [S] — sécurité, produit (#229)

`mode_headless.rst:61` dit « (nginx, Caddy…) » ; aucun `docker-compose*`,
aucun snippet. ADR-0005 fait du proxy toute la frontière de sécurité et
personne ne montre comment le configurer.
- [ ] `deploy/docker-compose.yml` : Caddy `forward_auth` (ou basic auth de
      démonstration) qui **retire** tout `X-Tenant-ID` client et **injecte**
      le tenant authentifié, `blunderdb-serve`, `postgres` avec RLS, volumes.
- [ ] Snippet nginx équivalent (`proxy_set_header X-Tenant-ID ""` puis
      `$authed_tenant`).
- [ ] Scénario « de zéro à un démon qui répond » dans `mode_headless.rst`
      (H.4) ; test CI qui monte le compose et fait un aller-retour (E.12
      nightly).

## G.2 — `serve` ignore les arguments positionnels [S] — fiabilité (#230)

`internal/server/serve.go:56` : `fs.Parse(args)` s'arrête au premier
non-drapeau, `NArg()` jamais vérifié. `docker run image serve --addr :9090`
(réflexe naturel, l'ENTRYPOINT étant déjà `serve`) démarre sur `:8080` sans
un mot.
- [ ] Erreur si `NArg() > 0` (tolérer un premier token `serve`).
- [ ] `--metrics`, `--cors-allow-origin`, `--rate-limit-rps/-burst` reçoivent
      leur `BLUNDERDB_*` (`envOr`).
- [ ] Rate limit **activé par défaut** à une valeur généreuse (50 rps/tenant,
      burst 100) plutôt qu'opt-in (`options.go:68-70`) ; cap dur sur le nombre
      de seaux (`ratelimit.go:48`) avec éviction LRU.

## G.3 — Migrations PostgreSQL : verrou et version [S] — fiabilité (#231)

`migrate_postgres.go:26-90` sans `pg_advisory_lock` : deux répliques (ou le
démon + `blunderdb migrate`) se marchent dessus. `schema_postgres.go:25` écrit
2.15.0, puis `002:31` ramène à 2.10.0, … `009:22` remet 2.15.0 : une
interruption laisse une base au schéma 2.15 déclarant 2.11 et `/readyz` en 503.
- [ ] `pg_advisory_lock(<clé fixe>)` autour de `Migrate`.
- [ ] Les fichiers de migration n'écrivent plus la version ; posée une fois
      en fin depuis `domain.DatabaseVersion`.
- [ ] Commentaire `rls_postgres.go:15-20` (« BeforeAcquire ») corrigé
      (`PrepareConn`).

## G.4 — Plafonds et validation d'entrée [S] — fiabilité (#232)

`handlers_positions.go:170-183` : `Limit` sans plafond, `IDs` non borné (32
Mio de corps ≈ 4 M d'ids dans `ANY($2)`) ; `decodeJSON` accepte tout
`Content-Type` ; pas de 405 (`server.go:63`) ; tenant brut journalisé sans
borne (`logging.go:34`) ; CORS sans `Vary: Origin`.
- [ ] `maxPageSize = 1000` appliqué dans `rpc` (`ErrInvalid` au-delà) ;
      `LoadByIDs` chunké.
- [ ] Refus d'un `Content-Type` non JSON sur `/v1` ; 405 + `Allow: POST` si le
      chemin est connu.
- [ ] Tenant tronqué à 64 caractères dans les logs (A.1 le rend numérique de
      toute façon) ; `Vary: Origin`, liste d'origines.

## G.5 — Routes ops séparées des routes tenant [M] — sécurité (#233)

`/v1/maintenance.vacuum` (`handlers_maintenance.go:23`) est une opération
globale sur le fichier SQLite partagé, accessible à tout tenant ;
`/v1/tenant.purge` (`handlers_tenant.go:19-22`) est destructif et
auto-déclenchable ; `metadata.*` (A.2) est du même ordre.
- [ ] Préfixe `/ops/` (vacuum, purge, metadata.setVersion, gammonnet.sweep de
      tout le fichier) documenté comme **à ne jamais exposer par le proxy** ;
      `--ops-addr` optionnel sur un listener séparé (comme `/metrics`).
- [ ] `parity_test.go` : catégorie `ops` avec raison.

## G.6 — Timeouts et arrêt gracieux [M] — fiabilité (#234)

`server.go:70-77` : seuls `ReadHeaderTimeout` et `IdleTimeout` (justifié :
NDJSON long) ; un client lent en écriture occupe un handler indéfiniment ;
pas de `LimitListener` ; `Shutdown` 15 s coupe un import de 512 Mio sans
événement `cancelled` ; le spool d'import (`handlers_imports.go:325-345`)
n'a pas de quota global (N × 512 Mio dans `$TMPDIR`) ; l'extension du fichier
téléversé alimente le nom temporaire (`:227-233`).
- [ ] `http.ResponseController.SetReadDeadline/SetWriteDeadline` par route
      (streamante vs ordinaire) ; `netutil.LimitListener` configurable.
- [ ] Annuler `imports` et `gammonnetJobs` avant `Shutdown`, émettre
      `{"event":"cancelled"}`.
- [ ] Compteur global d'octets en vol pour le spool ; extension par allowlist
      (`.xg`, `.xgp`, `.sgf`, `.mat`, `.bgf`, `.txt`, `.db`, `.dbx`).

## G.7 — Tests PostgreSQL : continuité et isolation [M] — fiabilité (#235)

`postgres_test.go` part d'une base fraîche (001 contient tout, 002-011 sont
des no-op) : aucune chaîne réelle testée alors que SQLite a
`TestMigrationSteps_ContinuousChain` (invariant). Isolation testée sur 3
familles / 16 ; 0 test HTTP à deux tenants ; FK sans `tenant_id`
(`001_initial:201-202,226-227,54,72,148,160`).
- [ ] Test qui bootstrape 001 « historique » puis rejoue 002…011 et compare le
      schéma final au bootstrap frais.
- [ ] Boucle d'isolation paramétrée par famille dans `storagetest` ; test HTTP
      A/B (avec A.1).
- [ ] FK composites `(tenant_id, parent_id)` dans la vague 2.16.0.
- [ ] Pool : `ConnectTimeout = 5 s`, `MaxConnIdleTime`, les quatre réglages en
      env ; gauges `blunderdb_pg_pool_{acquired,idle,max,wait_count}`.

---

## G.8 — Contrat d'API documenté et généré [M] — DX, produit (#236)

113/135 routes non documentées ; 0 OpenAPI ; le contrat consommé par gammonGo
n'existe que dans le Go. Les types `Req`/`Resp` sont déjà nommés.
- [ ] `openapi.yaml` généré depuis `Paths()` + réflexion sur les types (script
      `go run ./cmd/openapi-gen`), commité, test de non-dérive.
- [ ] Annexe Sphinx `api_reference.rst` générée (tableau famille → méthodes,
      schémas), 9 langues à la release.
- [ ] `pkg/blunderdb/server.Bootstrap` : exposer `MaxBodyBytes`, `Identity`,
      CORS, timeouts ; note « filigrane impossible sans `Identity` ».
- [ ] Documenter l'idempotence par méthode (majoritairement naturelle grâce à
      Zobrist) ; `Idempotency-Key` sur `collections.create`,
      `tournaments.create`, `anki.reviewCard`.

## G.9 — Pagination et compression sur les listes [M] — perf (#237)

`search.find`, `comments.listAll`, `collections.positions`,
`tournaments.list` streament tout ; aucune compression des flux NDJSON.
- [ ] `ListOpts` sur les familles listantes, `limit` par défaut côté serveur ;
      dépend de B.10.
- [ ] gzip conditionnel (`Accept-Encoding`) sur les réponses NDJSON.

## G.10 — Observabilité : corrélation et métriques métier [S] — DX (#238)

Pas de `X-Request-Id` ; pas de `traceparent` ; pas de métrique d'import en
vol, de sweep gammonNet, de taille de base, de pool ; pprof non exposé (bon).
- [ ] Middleware `RequestID` (entrant ou UUID), champ de log + en-tête de
      réponse, propagé dans le contexte ; `traceparent` relayé dans les logs.
- [ ] Gauges métier ; `--pprof-addr` optionnel sur un listener séparé.

## G.11 — Loadtest et sweep gammonNet [M] — perf (#239)

Loadtest solide (3 scénarios, p50/p95/p99, `tasks/headless/perf-baseline.md`)
mais tenants numériques seulement, jamais `--rls`, pas de CI. RLS = 2 RTT par
requête (`rls_postgres.go:23-36`, `set_config` + `RESET`) ≈ +100 % sur
`LoadPosition` 89 µs, jamais mesuré. Sweep serveur (`handlers_gammonnet.go:85,191-232`) :
`drainPositions` matérialise tout, N+1 `Analyses().Load`, sweeps illimités par
tenant.
- [ ] Scénario à tenants nommés (→ 400 après A.1), run `--rls` avec/sans,
      publier le delta ; envisager `SET LOCAL` dans la transaction.
- [ ] Sweep : `LEFT JOIN … WHERE a.id IS NULL` en SQL, itération en flux, un
      job en vol par tenant (`ErrConflict`).
- [ ] Loadtest court dans le nightly (E.12).

## G.12 — `migrate` SQLite → PostgreSQL [S puis L] — DX (#240)

`--batch-size` est un drapeau mort (`migrate/cli.go:50`) ; transaction unique
sans reprise ; sonde « destination non vide » ne regarde que les positions
(`migrate.go:58-64`) ; Anki, filtres, historiques, session non copiés
(documenté) ; copie ligne à ligne, progression par famille.
- [ ] Retirer le drapeau ; sonde via `Metadata().Counts` ; avertissement
      chiffré en fin (« N decks, M cartes, K révisions non migrés »).
- [ ] `LoadMany` côté source, `pgx.CopyFrom` côté cible, progression tous les
      1 000 ; reprise par lots quand un besoin terrain existe.
- [ ] Section « sauvegarde et restauration » dans `mode_headless.rst`
      (`pg_dump`, export par tenant via `exports.sqlite`, réimport via
      `migrate`).
- [ ] Backend SQLite du daemon : `TenantFilter() = "1=1"` (`adapter.go:23-26`)
      → refuser tout `X-Tenant-ID` non vide, ou l'écrire en gras dans le
      chapitre headless. Choix : refuser.

## G.13 — GUI Go : configuration, journal, arguments [S] — fiabilité, produit (#241)

- `config.go:15` : le fichier s'appelle `config.yaml` et contient du JSON ;
  `:381` écriture non atomique à chaque geste ; `:333` + `main.go:96-99` :
  config corrompue → `os.Exit(1)` ; pas de `config_version`.
- `logging.go:15-19` : stderr seulement, invisible depuis un lanceur ; pas de
  `recover` global ; pas de vérification de version (68 releases / 2,4 ans).
- `main.go:24-51` : `runGUI()` ignore `os.Args[1]` alors que `blunderdb.desktop`
  passe `%f` ; 0 `MimeType`.
- ~211 méthodes bindées (`main.go:113`), `DeleteFile` ne contrôle que le
  suffixe (`app.go:148`) ; `ReadFileContent`/`OpenPositionDialog` chargent tout
  en `string` (`app.go:186,274`) ; chemin interpolé dans osascript/powershell
  (`app.go:352,361`) ; `DisableWebViewDrop=false` non gardé par un test.
- [ ] `config.json` (lecture des deux noms, migration une fois), écriture
      `tmp + rename` (motif `bearoff_download.go:57`), `.bak` + défaut sur
      corruption, `config_version: 1` + table de migrations.
- [ ] Journal `$XDG_STATE_HOME/blunderDB/blunderdb.log` avec rotation par
      taille + bouton « ouvrir le dossier des journaux » ; `recover` + dialogue
      « erreur inattendue, journal ici ».
- [ ] Ouvrir la base passée en argument ; `MimeType=application/x-blunderdb`
      + `shared-mime-info` (`.db` est trop générique : associer `.dbx` et un
      alias) ; Windows/macOS : associations dans l'installeur et l'`Info.plist`.
- [ ] Vérification de version opt-in (API GitHub Releases, capacité
      optionnelle au sens d'ADR-0004), notification non bloquante en barre
      d'état ; désactivée par défaut sur les canaux gestionnaires de paquets
      (détection : variable d'environnement posée par le paquet ou chemin
      d'installation).
- [ ] Ne lier que ce que le front appelle ; `DeleteFile` restreint ; plafond
      de taille + message ; chemin par argument ; test Go sur `options.App`.
- [ ] Erreurs GUI avec un code stable (`CodeNotFound/Invalid/Conflict/Internal`
      comme le serveur), traduites côté front ; « annulé par l'utilisateur »
      distinct d'un échec.

## G.14 — Parité dans les deux sens [S test / M fonctions] — produit (#242)

`parity_test.go` (excellent) est unidirectionnel : 21/135 routes hors table.
Six capacités n'existent que sur le serveur, sans décision écrite :
`anki.suspendCard`, `anki.buryCard`, `anki.removeCard`, `anki.reviewLog`,
`anki.optimizeParams`, `analyses.repair`.
- [ ] Assertion inverse (toute route est dans `databaseParity` ou dans
      `serverOnly` motivée).
- [ ] Remonter les six sur `Database` + GUI (menu contextuel d'une carte :
      suspendre/enterrer/retirer ; onglet Anki : journal, optimisation ;
      Configuration : réparer les analyses) + CLI ; doc `manuel.rst`.

---

## Résumé du lot

| Fiche | Effort | Étape |
|---|---|---|
| G.1, G.2, G.3, G.4 | S | 1 |
| G.5, G.6, G.7 | M | 1 |
| G.10, G.13 | S | 2 |
| G.8, G.9, G.11, G.12, G.14 | M | 2 |

Sortie de l'étape 1 du lot : image `ghcr.io/kevung/blunderdb-serve` publiée
avec `HEALTHCHECK`, compose d'exemple, tenants numériques, `/ops` séparé.
