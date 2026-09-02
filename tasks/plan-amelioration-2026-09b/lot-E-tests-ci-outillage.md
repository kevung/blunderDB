<!-- Lot du plan tasks/plan-amelioration-2026-09b/README.md. -->

# Lot E — Tests, CI, chaîne d'approvisionnement, outillage

État vérifié le 2026-09-02 (run `33683525857`) : 27 jobs verts, 11 min 37 de
mur, 3 172 s de runner cumulés ; couverture Go 51,1 % (artefact) ; front
54,2 % lignes. Points forts à préserver : `govulncheck` bloquant à 0,
timeouts sur tous les jobs, `concurrency` asymétrique, `Dockerfile.serve`
distroless nonroot, `hostile-smoke` (suite complète en 92 s), contrat storage
rejoué sur PG via testcontainers, `.golangci.yml` justifié ligne à ligne.
Les urgences (permissions, `main` non protégée, seeds fuzz) sont au lot A.

E.1 à E.5 = **étape 1** ; E.6 à E.12 = **étape 2**.

---

## E.1 — Le job `test-os` bloque enfin [S] — fiabilité (#217)

`build.yml:196` : `continue-on-error: true` (« à retirer après le premier run
vert ») ; Windows 154 s et macOS 139 s sont verts. Préalable de C.2 et de #151.
- [ ] Retirer le flag ; ajouter `kernel_identity_test` explicite sur macOS.
- [ ] `GOOS=windows go vet ./...` dans le job `lint` (pour `filelock_windows.go`,
      `diskspace_windows.go`).

## E.2 — La couverture dit faux et n'a pas de seuil [S] — fiabilité (#218)

- `test-postgres` (`build.yml:167-171`) n'a pas de `-coverprofile` →
  `storage/postgres` à 0 % dans le merge ; `sqlshared` à 1,4 % parce que
  mesuré sans `-coverpkg` (3 216 lignes non mesurées, dont `find` 638 l. et
  `Compute` 433 l.).
- `build.yml:95` : « visibility only, no threshold » ; `build.yml:98-99` : le
  job `coverage` fusionne même si un shard est rouge.
- `golangci-lint` sans `--build-tags postgres` (`build.yml:466`) : 13 fichiers
  de `storage/postgres`, 2 de `migrate`, 1 test serveur jamais lintés.
- [ ] `-coverpkg=./pkg/blunderdb/...` ; `-coverprofile` dans `test-postgres`
      et merge.
- [ ] Plancher non régressif (`total >= 51 %`) ; annoter le résumé si
      `needs.test.result != 'success'`.
- [ ] Seconde passe `golangci-lint run --build-tags postgres ./...`.
- [ ] Publier le top 10 des paquets lents dans le step summary (aujourd'hui :
      `database` 54,6 s, `cli` 27,9 s, `gammonnet` 19 s = 88 % du temps).

## E.3 — Paralléliser les tests Go [M] — vitesse CI ×3 (#219)

0 `t.Parallel()` sur 862 tests ; `database` 54,6 s en série, 419 s sous
`-race` (chemin critique) ; sharding par première lettre déséquilibré
(279 s / 419 s) et fragile (renommer un test déplace la charge).
- [ ] `t.Parallel()` sur les tests indépendants (125 sites `t.TempDir()`
      déjà) ; corriger d'abord les deux `os.Chdir` vers la racine
      (`database/main_test.go:20`, `internal/cli/main_test.go:20`) qui
      interdisent le parallélisme et laissent `a.db`, `c.db`, `toto*.db`,
      `Quiz*.db-wal`, `*.lock` à la racine → chemin de fixtures absolu via
      `runtime.Caller`.
- [ ] Sharder par index (`go test -list` + modulo) ou supprimer le sharding
      une fois parallélisé.
- [ ] Les 6 `time.Sleep` sur chemins concurrents (`busy_retry_test.go`,
      `history_sqlite_test.go`, `db_gammonnet_batch*_test.go`,
      `parallel_probe_test.go`) → canaux / horloge injectable (motif
      `middleware.RateLimiter` avec `now func()`).

## E.4 — Tests qui passent pour de mauvaises raisons [S] — fiabilité (#220)

- 142 `t.Skip`, dont ~14 « fixture absente ⇒ vert » (`test.sgf`, `test.xg`,
  `test.txt`, `test.mat`, gold, corpus).
- `tests/` racine : 6 fichiers hérités (`debug_xg_test.go`, noms réels dans
  `xg_import_test.go`), paquet sans code donc invisible à la couverture.
- `embed_test.go:29-36` teste `/v1/position.save` (singulier, inexistant) →
  passe via le 404.
- 6 paquets sans test : `migrate` (logique de remap pure, testable sur deux
  `sqlite.Storage` en mémoire), `storagetest`, `cmd/serve` (entrypoint publié
  en image), `calibrace`, `loadtest`, `extract_gnubg_stats`.
- [ ] `t.Fatal` sur fixture manquante (ou test-sentinelle) ; redistribuer
      `tests/` vers `ingest`/`database` ; corriger `embed_test` (400 +
      `code=invalid` + message nommant `X-Tenant-ID`) ; test de câblage
      `cmd/serve` ; test `migrate` sans Docker.

## E.5 — Formatage Go, hooks et `make check` [S] — DX (#221)

8 fichiers `gofmt` sales ; `formatters:` de `.golangci.yml` sans formateur ;
pre-commit local (`.git/hooks`, front seulement, non versionné) ; `make check`
≠ CI (manque e2e, `release.sh --check`, PG, gofmt) malgré le commentaire
`Makefile:30` ; ni `.editorconfig`, ni devcontainer.
- [ ] `formatters: enable: [gofmt, goimports]` ; corriger les 8 fichiers.
- [ ] `.githooks/pre-commit` versionné (`make check-fast` : gofmt, vet, eslint,
      prettier) + `core.hooksPath` documenté dans `CONTRIBUTING.md`.
- [ ] `make check` (rapide) / `make check-all` (parité CI, y compris
      `make test-pg` qui dit clairement s'il a besoin de Docker).
- [ ] `.editorconfig` ; `.devcontainer/` minimal (Go 1.25.13, Node 22,
      webkit2gtk-4.1) — utile aussi pour un contributeur occasionnel.

---

## E.6 — Linters de complexité non régressifs [S] — dette (#222)

5 fonctions > 300 lignes (`find` 638, `Compute` 427, `runSearch` 427,
`migrate_1_9_0_to_2_0_0` 320, `CommitImportDatabase` 303), 20 > 120 ;
`gocyclo`/`gocognit`/`dupl` ni installés ni configurés.
- [ ] `funlen` et `gocognit` activés avec seuil = maximum actuel, abaissé à
      chaque découpage (B.15).
- [ ] `gosec` limité à `./internal/server/... ./pkg/blunderdb/issuance/...`
      (surfaces réseau + crypto), le refus global reste justifié
      (`.golangci.yml:14-22`).

## E.7 — Supply chain : attestations, SBOM, signatures [S] — sécurité (#223)

0 SBOM, 0 provenance ; image GHCR poussée sans attestation
(`build.yml:285-295`) ; `.sha256` sans signature détachée ; tag non signé
(`scripts/release.sh:202`) ; l'image n'est jamais démarrée en CI ; multi-arch
seulement sur tag (`build.yml:289`).
- [ ] `actions/attest-build-provenance` sur les assets ; `provenance: true`,
      `sbom: true` dans `build-push-action` ; `trivy` sur l'image (non
      bloquant d'abord).
- [ ] `minisign` (ou cosign keyless) sur `SHA256SUMS` ; tag `-s` dans
      `release.sh` ; clé publique dans le README et `telecharge_install.rst`.
- [ ] `docker run --rm -d` + `curl /readyz` après le build ; construire
      amd64+arm64 sans push sur `main`.

## E.8 — Hygiène des jobs front [S] — vitesse CI (#224)

`frontend-test` lance vitest deux fois (`build.yml:311-322`) ; `npm ci` à
froid dans 3 jobs ; Playwright `workers` non fixé, retries sans signal flaky.
- [ ] Un seul `npm test -- --coverage` ; job front unique (lint + test + e2e)
      ou cache `node_modules` clé `package-lock.json`.
- [ ] Voir D.13 pour Playwright.

## E.9 — Suivi de performance dans le temps [M] — DX (#225)

Job `benchmark` 115 s par push, artefact que personne ne compare ; 0
`benchstat` ; `tasks/bench-*.txt` datent d'avril ; 11 benchmarks gammonNet
mais aucun sur `Probs`/`Decide`/lot (C.7).
- [ ] `benchstat` contre une baseline versionnée
      (`tasks/bench/baseline.txt`) sur un sous-ensemble stable
      (`storage/sqlite`, `sqlshared`, `gammonnet -short`), seuil ±10 %,
      commentaire de PR ; le job passe en `workflow_dispatch` + nightly.
- [ ] Baseline régénérée à chaque release (skill `release-blunderdb`).

## E.10 — Dépôt : poids du clone, `.dockerignore`, fixtures [S] — DX (#226)

Clone à 730 Mio (historique `gh-pages` : `.dvi` 113 Mo, `.doctrees` 88 Mo) ;
`.dockerignore` `*.db` non récursif → `testdata/tournois/live-main.db`
(215 Mo, gitignoré) part dans le contexte Docker local ; `testdata/` local
248 Mo.
- [ ] `force_orphan: true` sur `peaceiris/actions-gh-pages` ; note dans le
      README de doc sur le clone allégé (`--single-branch`).
- [ ] `.dockerignore` : `**/*.db`, `testdata/tournois/`, `.venv/`.
- [ ] Nettoyer la racine du checkout (E.3 supprime la cause).

## E.11 — Modèles d'issue et automatisations GitHub [S] — communauté/DX (#227)

Pas d'`ISSUE_TEMPLATE` (le template de PR est excellent) ; Dependabot ne
couvre que les version updates ; Discussions activées et vides.
- [ ] Deux formulaires YAML (bug : version, OS, canal d'installation, fichier
      source ; suggestion) + `config.yml` (Discord, Discussions).
- [ ] Dependabot : npm mensuel groupé, gomod hebdo, actions hebdo.
- [ ] Action « stale » douce (90 jours, label seulement, jamais de fermeture
      automatique).

## E.12 — Nightly [S] — fiabilité (#228)

Un seul `nightly.yml` regroupe ce qui ne doit pas tourner à chaque push :
gold de recherche (`BLUNDERDB_GOLD=1`), `-race -short` sur `gammonnet`,
fuzz 5 min/cible, benchstat, trivy, loadtest SQLite court (G.11), `test-os`
complet sans `-short`.
- [ ] Le fichier ; badge « nightly » dans le README ; notification en cas
      de rouge (issue automatique avec label `nightly`).

---

## Résumé du lot

| Fiche | Effort | Étape |
|---|---|---|
| E.1, E.2, E.4, E.5 | S | 1 |
| E.3 | M | 1 |
| E.6, E.7, E.8, E.10, E.11, E.12 | S | 2 |
| E.9 | M | 2 |
