# Fiche 01 — CI honnête et rapide

Branche : `ci/fiabilisation`

## Objectif

Que le vert CI signifie « tout ce qui est livré est testé » : le backend
PostgreSQL entre en CI, le fuzz cesse d'être rouge en permanence, les minutes
gaspillées sont récupérées.

## Constats

- `pkg/blunderdb/storage/postgres/postgres_test.go:1` : `//go:build postgres` ;
  aucun job ne passe `-tags postgres`. Contrat partagé, RLS, purge tenant,
  migration SQLite→PG : jamais exécutés en CI.
- Workflow Fuzz rouge depuis 2 semaines : `FuzzDecodeBoardCompact` échoue sur
  `context deadline exceeded` à 120 s (timeout, pas crash) ; corpus jeté à
  chaque run (330 « new interesting » perdus).
- Job `benchmark` : `go test -bench=. -count=3 ./...` sans `-run '^$'` →
  réexécute toute la suite de tests 3× (5 min 16 par PR), et `bench.txt`
  n'est comparé à rien.
- Pas de `concurrency` au niveau workflow : 3 commits = 3 matrices complètes.
- Pas de bloc `permissions:` dans `build.yml` (le `GITHUB_TOKEN` a les droits
  par défaut partout, y compris dans les jobs qui exécutent du code tiers).
- `govulncheck` en `continue-on-error: true` : purement décoratif.
- Playwright : pas de cache `~/.cache/ms-playwright`, pas de `retries` en CI,
  `waitForTimeout` en dur dans `autocomplete.spec.js:11-15`.
- `npm run lint`/`format:check` ne couvrent que `src/` — `tests/e2e/**` et les
  `*.config.js` échappent au lint et à prettier.
- Aucune mesure de couverture, ni Go ni frontend.

## Tâches

- [x] Nouveau job `test-postgres` (ubuntu-latest, Docker présent) :
      `go test -tags postgres -timeout 1200s ./pkg/blunderdb/storage/postgres/... ./pkg/blunderdb/migrate/...`.
      Ajouter une garde `BLUNDERDB_REQUIRE_PG=1` qui transforme le
      `t.Skipf` (testcontainers indisponible) en échec dur, et la poser dans ce job.
- [x] Fuzz : distinguer crash (artefact présent → échec) de timeout (succès) ;
      réduire `-fuzztime` sous le timeout du step ; mettre en cache le corpus
      (`actions/cache` sur le répertoire de corpus go-fuzz).
- [x] Benchmark : ajouter `-run '^$'` ; `-count=1` sur PR ; ne le déclencher
      que sur `main` et tags (ou le supprimer des PR).
- [x] `concurrency: {group: wails-build-${{ github.ref }}, cancel-in-progress: (PR seulement)}`.
- [x] `permissions: contents: read` au niveau workflow, élevé localement pour
      les jobs release/docs/pages qui publient.
- [x] Playwright : cache navigateurs, `retries: process.env.CI ? 2 : 0`,
      remplacer les `waitForTimeout` par des attentes d'état.
- [x] Étendre lint/format frontend : `eslint src/ tests/`,
      `prettier --check src/ tests/ *.config.js` (un commit de reformatage
      isolé si le premier passage réécrit des fichiers).
- [x] Couverture visible, non bloquante : `go test -coverprofile` + résumé
      `go tool cover -func` dans le job summary ; `vitest run --coverage`
      (ajouter `@vitest/coverage-v8`) en artefact.
- [x] `.golangci.yml` : supprimer l'exclusion morte `path: "^db\\.go$"`
      (ne matche plus rien) et restreindre l'exclusion `fmt.Print*` à
      `internal/cli/` ; corriger ce que la levée d'exclusion révèle.

## Critères de fin

- Un run CI vert exécute réellement le contrat storage sur PostgreSQL.
- Le workflow Fuzz est vert sur un run sans crash, et échouerait sur un crash.
- Durée totale d'un run PR réduite (benchmark sorti du chemin des PR).
- `golangci-lint run` vert avec la config resserrée.

## Risques & garde-fous

- Le job Postgres peut révéler des tests qui ne passaient plus : les corriger
  fait partie de la fiche (c'est le but).
- Ne pas toucher au chemin de publication des PDF/pages sur tag (déjà cassé
  une fois par le passé) : aucune condition nouvelle sur les jobs de release.

## Notes d'exécution

Travail fait dans `/home/unger/src/blunderDB-fiche01` (branche `ci/fiabilisation`),
5 commits :

1. `fix(lint): resserrer golangci-lint sur fmt.Print* et retirer une exclusion morte`
2. `test(postgres): garde BLUNDERDB_REQUIRE_PG pour durcir le job CI`
3. `ci: job PostgreSQL, concurrency, permissions, fuzz honnête, benchmark restreint`
4. `style: prettier sur tests e2e et configs`
5. `ci(e2e): Playwright fiable — cache navigateur, retries, fin des sleeps fixes`

### Ce qui a été fait tel que décrit

- **`test-postgres`** : job ajouté dans `build.yml`, `BLUNDERDB_REQUIRE_PG=1`
  posé en env de job. La garde a été implémentée dans les deux copies de
  `startPostgres()` (`pkg/blunderdb/storage/postgres/postgres_test.go` et
  `pkg/blunderdb/migrate/migrate_postgres_test.go` — `stats_parity_postgres_test.go`
  réutilise la même fonction, pas de troisième copie). **Validé en réel** :
  Docker était disponible dans le bac à sable ; `TestContract_Postgres` et la
  suite complète (`storage/postgres` + `migrate`, avec la garde active) sont
  passés, aucun test caché par le skip.
- **Fuzz** : `-fuzztime` par défaut abaissé à 90s, `timeout-minutes: 8` posé
  sur le step de fuzzing, `continue-on-error: true` dessus, et un step dédié
  qui ne fait échouer le job que si `testdata/fuzz/<Target>/` contient un
  fichier. Cache `actions/cache` sur `~/.cache/go-build/fuzz`, clé par cible +
  `github.run_id` avec `restore-keys` sans le run_id (pour retomber sur le
  cache le plus récent). Testé en local (compilation + exécution 3s d'une
  cible) mais pas le comportement du cache lui-même (spécifique à
  `actions/cache`, non simulable hors GitHub Actions).
- **Benchmark** : `-run '^$' -count=1`, job sous `if: github.event_name == 'push'`
  (sort donc du chemin des PR, ne tourne que sur push main/tag comme demandé).
- **Concurrency / permissions** : ajoutés au niveau workflow dans `build.yml`.
  `contents: write` posé explicitement sur les jobs `build`, `docs`, `pages`
  (les seuls qui publient — `softprops/action-gh-release` et
  `peaceiris/actions-gh-pages`). Aucune condition `if:` nouvelle sur les steps
  de release eux-mêmes ; le chemin de publication sur tag n'est pas touché.
- **Playwright** : cache `~/.cache/ms-playwright` keyé sur la version de
  `@playwright/test` lue dynamiquement dans `package-lock.json` (pas un hash
  de fichier — comme demandé) ; `retries: process.env.CI ? 2 : 0` dans
  `playwright.config.js` ; les 4 `waitForTimeout` restants (`autocomplete`,
  `tab-switch-stats`, `epc-bar-refreshes-on-return`, `tour`) remplacés par des
  attentes d'état (`toBeVisible`, `toHaveClass(/active/)`, `toContainText`,
  ou juste la disparition du sleep quand `waitForSelector`/`expect` suivant
  fournissait déjà l'attente réelle).
- **Lint/format frontend** : `lint` et `format:check` (et `format`, `lint:fix`)
  étendus à `tests/` et `*.config.js`. Extension a révélé un vrai problème :
  `installWailsMock()` documentait un paramètre `opts.dbExtra` jamais
  implémenté (eslint `no-unused-vars`) — supprimé avec le commentaire
  trompeur plutôt que masqué.
- **Couverture** : `go test -coverprofile=coverage.out -covermode=atomic` +
  `go tool cover -func` dans `$GITHUB_STEP_SUMMARY` + artefact `coverage.out`
  dans le job `test`. Côté frontend, `@vitest/coverage-v8` ajouté via
  `npm install --save-dev` (donc `package-lock.json` mis à jour par npm, pas
  à la main) et un step `npm test -- --run --coverage` non bloquant
  (`continue-on-error: true`) qui uploade `frontend/coverage/` en artefact.
  `frontend/coverage/` et `coverage.out` ajoutés au `.gitignore` racine.
- **`.golangci.yml`** : exclusion morte `path: "^db\\.go$"` supprimée ;
  exclusion `fmt.Print*` déplacée de `errcheck.exclude-functions` (globale)
  vers une règle `exclusions.rules` scopée à `^(internal/cli/|cmd/)`. La
  levée de l'exclusion a révélé 3 fichiers hors périmètre avec des
  `fmt.Print*`/`fmt.Fprint*` non vérifiés : `main.go` (7 appels),
  `internal/server/call.go` (1), `internal/server/metrics/metrics.go` (13) —
  corrigés avec `_, _ = fmt.Xxx(...)` (idiome Go standard pour une erreur
  ignorée intentionnellement), pas de `//nolint`. `golangci-lint run ./...`
  est revenu à 0 issue après.

### Écarts / points à vérifier humainement

- **`govulncheck` en `continue-on-error: true`** : listé dans les « Constats »
  de la fiche mais absent de la liste « Tâches » — laissé tel quel, hors
  périmètre de cette fiche.
- **Validation e2e Playwright locale incomplète.** Deux obstacles propres au
  bac à sable, pas au code : (1) le port 5173 par défaut est occupé par un
  service `gammonGo` sans rapport tournant dans le même environnement — testé
  en pointant temporairement `webServer`/les `page.goto()` sur le port 5199
  (changement non commité, reverté ensuite), mais (2) le téléchargement des
  binaires Chromium (`npx playwright install chromium`) a stagné à 18 Mo
  pendant plus de 10 minutes puis a été tué. `npx playwright test --list`
  confirme que les 18 tests des 4 fichiers restent bien découverts par
  Playwright après les changements (pas d'erreur de syntaxe/imports). Les
  remplacements de `waitForTimeout` ont été relus attentivement (voir liste
  ci-dessus) mais **n'ont pas tourné dans un vrai navigateur** — à surveiller
  au premier run CI (`frontend-e2e`) sur cette branche ; si un des remplacements
  s'avère trop strict (ex. `toHaveClass(/active/)` juste après un clic, avant
  que Svelte n'ait eu le temps de réagir), le correctif le plus probable est
  d'ajouter `{ timeout: ... }` explicite sur l'assertion concernée plutôt que
  de revenir à un sleep fixe.
- **`npm install --save-dev @vitest/coverage-v8`** a remplacé le symlink
  `frontend/node_modules` (pointant vers le checkout principal, mis en place
  pour que le hook pre-commit eslint fonctionne dans les worktrees fraîches —
  voir `project_worktree_precommit_node_modules` en mémoire) par une
  installation complète indépendante dans ce worktree. Résultat correct
  (hooks + tests + lint tous verts après), mais ce worktree a maintenant sa
  propre copie de `node_modules` au lieu de partager celle du checkout
  principal — plus d'espace disque, aucun autre effet observé.
- **`go test ./...` fait désormais apparaître un paquet
  `frontend/node_modules/flatted/golang/pkg/flatted` (`[no test files]`)** —
  conséquence du même remplacement symlink→dossier réel : ce paquet npm
  embarque du code Go que `go test ./...` découvre maintenant qu'il n'est
  plus derrière un symlink. Inoffensif (aucun test, `go vet`/`golangci-lint`
  ne le signalent pas), mais à garder en tête si `./...` doit un jour être
  restreint plus précisément.
- **Pas de run CI réel obtenu** (pas de push/PR effectué depuis ce worktree,
  conformément à la consigne « ne push pas »). Tous les jobs modifiés ont été
  validés soit en exécutant les commandes exactes en local (Go : `go vet`,
  `go test -race`, `golangci-lint run`, `go test -tags postgres` avec Docker
  réel ; frontend : `npm run lint`, `npm run format:check`, `npm test -- --run`),
  soit par relecture + `yaml.safe_load` pour la syntaxe des workflows. Le
  premier run GitHub Actions réel sur cette branche reste le test de bout en
  bout complet (notamment le comportement du cache `actions/cache`, la
  disponibilité de Docker sur `ubuntu-latest` pour `test-postgres`, et les
  jobs Playwright/e2e non validés localement ci-dessus).
