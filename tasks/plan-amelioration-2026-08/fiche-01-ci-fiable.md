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

- [ ] Nouveau job `test-postgres` (ubuntu-latest, Docker présent) :
      `go test -tags postgres -timeout 1200s ./pkg/blunderdb/storage/postgres/... ./pkg/blunderdb/migrate/...`.
      Ajouter une garde `BLUNDERDB_REQUIRE_PG=1` qui transforme le
      `t.Skipf` (testcontainers indisponible) en échec dur, et la poser dans ce job.
- [ ] Fuzz : distinguer crash (artefact présent → échec) de timeout (succès) ;
      réduire `-fuzztime` sous le timeout du step ; mettre en cache le corpus
      (`actions/cache` sur le répertoire de corpus go-fuzz).
- [ ] Benchmark : ajouter `-run '^$'` ; `-count=1` sur PR ; ne le déclencher
      que sur `main` et tags (ou le supprimer des PR).
- [ ] `concurrency: {group: wails-build-${{ github.ref }}, cancel-in-progress: (PR seulement)}`.
- [ ] `permissions: contents: read` au niveau workflow, élevé localement pour
      les jobs release/docs/pages qui publient.
- [ ] Playwright : cache navigateurs, `retries: process.env.CI ? 2 : 0`,
      remplacer les `waitForTimeout` par des attentes d'état.
- [ ] Étendre lint/format frontend : `eslint src/ tests/`,
      `prettier --check src/ tests/ *.config.js` (un commit de reformatage
      isolé si le premier passage réécrit des fichiers).
- [ ] Couverture visible, non bloquante : `go test -coverprofile` + résumé
      `go tool cover -func` dans le job summary ; `vitest run --coverage`
      (ajouter `@vitest/coverage-v8`) en artefact.
- [ ] `.golangci.yml` : supprimer l'exclusion morte `path: "^db\\.go$"`
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
