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
- [x] Retirer le flag ; ajouter `kernel_identity_test` explicite sur macOS.
- [x] `GOOS=windows go vet ./...` dans le job `lint` (pour `filelock_windows.go`,
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
- [x] `-coverpkg=./pkg/blunderdb/...` ; `-coverprofile` dans `test-postgres`
      et merge.
- [x] Plancher non régressif (`total >= 51 %`) ; annoter le résumé si
      `needs.test.result != 'success'`.
- [x] Seconde passe `golangci-lint run --build-tags postgres ./...`.
- [x] Publier le top 10 des paquets lents dans le step summary (aujourd'hui :
      `database` 54,6 s, `cli` 27,9 s, `gammonnet` 19 s = 88 % du temps).

## E.3 — Paralléliser les tests Go [M] — vitesse CI ×3 (#219)

0 `t.Parallel()` sur 862 tests ; `database` 54,6 s en série, 419 s sous
`-race` (chemin critique) ; sharding par première lettre déséquilibré
(279 s / 419 s) et fragile (renommer un test déplace la charge).
- [x] `t.Parallel()` sur 335 des 407 tests de `database` et `cli`. Les 72
      autres touchent l'état du PROCESSUS (`t.Setenv`, `os.Stdout`) et sont
      exclus par fermeture transitive sur les helpers — un grep direct les
      manquait (`isolateIdentity` cache un `t.Setenv`, `captureStdout`
      remplace `os.Stdout`).
- [x] Les `os.Chdir` de `TestMain` sont restés : ils s'exécutent une fois
      avant les tests, pas pendant, et n'empêchent donc rien. Ce qui salissait
      la racine était `test_real_migration_test.go`, un test de mise au point
      SANS UNE SEULE ASSERTION (que des `t.Logf`) qui ouvrait `c.db` à la
      racine — donc le créait. Supprimé. Un `x.db` de `cli_test.go` est passé
      en `t.TempDir()`. Racine vérifiée propre après une suite complète.
- [x] Sharding par index : `scripts/go-test-shard.sh <paquet> <i> <n>`, câblé
      pour `database` (4 tranches) et `internal/cli` (2). Pas de suppression du
      sharding : **le détecteur de courses est ce qui borne ce job, et il ne se
      parallélise pas**. Mesuré le 2026-09-04 à 4 GOMAXPROCS (la forme d'un
      runner) : une MOITIÉ de `database` = 1352 s sous `-race -parallel 4`,
      pour un budget de 1200 s — d'où quatre tranches (~675 s). Sans `-race`
      et sur 16 cœurs, le même paquet passe de 159 s à 57 s, et son chemin
      critique de 112 s à 49 s (sous-tests de `TestStatsParity` parallélisés) :
      le parallélisme est réel, il n'est simplement pas ce qui borne la CI.
      `internal/cli` est sorti du shard `app` : ses tests les PLUS LENTS sont
      justement ceux qui capturent `os.Stdout`, donc sériels — seul le
      découpage par index répartit ce coût-là.
- [x] Les deux `time.Sleep(100 ms)` de `db_gammonnet_batch*_test.go` →
      synchronisation par canal sur l'entrée du yield. Pour le test qui compte
      les écritures, l'attente reste bornée mais porte sur un ÉTAT et non sur
      une durée : l'écriture est faite par la goroutine qui draine `results`,
      pas par le worker, donc le yield seul ne la garantit pas. Les autres
      (`busy_retry_test.go` 60 ms, `history_sqlite_test.go` 2 ms,
      `parallel_probe_test.go`) sont laissés : ils mesurent une fenêtre de
      temps ou fabriquent une contention, ce sont leurs sujets.
- Reste ouvert, consigné au BACKLOG : le CLI écrit sur `os.Stdout` (765
  `fmt.Print*`) au lieu d'un `io.Writer`, ce qui borne le gain sur ce paquet.

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
- [x] `t.Fatal` sur fixture manquante ; `tests/` redistribué vers
      `ingest`/`database` (le répertoire n'existe plus) ; `embed_test`
      corrigé ; `cmd/serve/main_test.go` et `migrate/remap_test.go` écrits.

## E.5 — Formatage Go, hooks et `make check` [S] — DX (#221)

8 fichiers `gofmt` sales ; `formatters:` de `.golangci.yml` sans formateur ;
pre-commit local (`.git/hooks`, front seulement, non versionné) ; `make check`
≠ CI (manque e2e, `release.sh --check`, PG, gofmt) malgré le commentaire
`Makefile:30` ; ni `.editorconfig`, ni devcontainer.
- [x] `formatters: enable: [gofmt, goimports]` ; l'arbre est propre au `gofmt -l`.
- [x] `.githooks/pre-commit` versionné (`make check-fast`) + `core.hooksPath`
      documenté dans `CONTRIBUTING.md`.
- [x] `make check-fast` / `check` / `check-all`.
- [x] `.editorconfig` et `.devcontainer/`.

---

## E.6 — Linters de complexité non régressifs [S] — dette (#222)

5 fonctions > 300 lignes (`find` 638, `Compute` 427, `runSearch` 427,
`migrate_1_9_0_to_2_0_0` 320, `CommitImportDatabase` 303), 20 > 120 ;
`gocyclo`/`gocognit`/`dupl` ni installés ni configurés.
- [x] `funlen` et `gocognit` activés avec seuil = maximum actuel. Remesurés le
      2026-09-04 après le découpage de `find` par B.15 : 573/316/462 →
      **438/180/200**, fixés chacun par une fonction différente
      (`Compute`, `parseSearchFlags`, `applyGoFilters`). La méthode compte :
      on mesure en **montant** le seuil jusqu'au silence, jamais en le
      descendant à 1 — `funlen` ne signale qu'une violation par fonction
      (lignes d'abord, puis instructions), donc à 1 presque tout répond
      « trop d'instructions » et les fonctions les plus longues ne disent
      jamais leur longueur.
- [x] `gosec` limité à `./internal/server/... ./pkg/blunderdb/issuance/...`
      (surfaces réseau + crypto), le refus global reste justifié
      (`.golangci.yml`).

## E.7 — Supply chain : attestations, SBOM, signatures [S] — sécurité (#223)

0 SBOM, 0 provenance ; image GHCR poussée sans attestation
(`build.yml:285-295`) ; `.sha256` sans signature détachée ; tag non signé
(`scripts/release.sh:202`) ; l'image n'est jamais démarrée en CI ; multi-arch
seulement sur tag (`build.yml:289`).
- [x] `actions/attest-build-provenance` sur les assets ; `provenance`/`sbom`
      sur l'image poussée (sur tag) ; `trivy` non bloquant sur l'image.
- [x] `minisign` optionnel sur `SHA256SUMS` (voir `packaging/minisign/`) ;
      signature du tag dans `release.sh` avec repli sur un tag léger.
- [x] Image démarrée et sondée en CI ; amd64+arm64 construits sans push.

## E.8 — Hygiène des jobs front [S] — vitesse CI (#224)

`frontend-test` lance vitest deux fois (`build.yml:311-322`) ; `npm ci` à
froid dans 3 jobs ; Playwright `workers` non fixé, retries sans signal flaky.
- [x] Un seul `npm test -- --coverage` : le second passage n'apportait aucun
      signal (la couverture ne change ni les tests joués ni leur verdict) et
      coûtait la moitié du temps du job.
- [x] Cache `node_modules` clé `package-lock.json`, la même dans les trois
      jobs front ; cache des navigateurs Playwright clé sur la version
      résolue de `@playwright/test`. Playwright : voir D.13 (livrée).

## E.9 — Suivi de performance dans le temps [M] — DX (#225)

Job `benchmark` 115 s par push, artefact que personne ne compare ; 0
`benchstat` ; `tasks/bench-*.txt` datent d'avril ; 11 benchmarks gammonNet
mais aucun sur `Probs`/`Decide`/lot (C.7).
- [x] `benchstat` contre une baseline versionnée, en `workflow_dispatch` +
      nightly ; le job `benchmark` de `build.yml`, qui tournait à chaque push
      sans jamais être comparé à quoi que ce soit, a disparu.
- [x] Baseline régénérée à chaque release.

## E.10 — Dépôt : poids du clone, `.dockerignore`, fixtures [S] — DX (#226)

Clone à 730 Mio (historique `gh-pages` : `.dvi` 113 Mo, `.doctrees` 88 Mo) ;
`.dockerignore` `*.db` non récursif → `testdata/tournois/live-main.db`
(215 Mo, gitignoré) part dans le contexte Docker local ; `testdata/` local
248 Mo.
- [x] `force_orphan: true` sur `peaceiris/actions-gh-pages` ; note « clone
      allégé » (`--single-branch`) à la fin de `doc/README.txt`, à côté d'une
      section neuve : **comment écrire un lot d'entrées `.po` par programme
      sans réécrire le fichier** (rendre chaque entrée avec Babel, remplacer
      le bloc), et comment vérifier au `--numstat` qu'on ne l'a pas fait.
- [x] `.dockerignore` : `**/*.db` (le motif nu n'est pas récursif, contrairement
      à `.gitignore`), `testdata/tournois/`, `.venv`.
- [x] Racine du checkout nettoyée (E.3 a supprimé la cause).

## E.11 — Modèles d'issue et automatisations GitHub [S] — communauté/DX (#227)

Pas d'`ISSUE_TEMPLATE` (le template de PR est excellent) ; Dependabot ne
couvre que les version updates ; Discussions activées et vides.
- [x] Deux formulaires YAML + `config.yml`.
- [x] Dependabot : npm mensuel groupé, gomod hebdo, actions hebdo, plus
      `docker` — les images de base n'avaient aucune couverture.
- [x] Action « stale » douce, label seulement.

## E.12 — Nightly [S] — fiabilité (#228)

Un seul `nightly.yml` regroupe ce qui ne doit pas tourner à chaque push :
gold de recherche (`BLUNDERDB_GOLD=1`), `-race -short` sur `gammonnet`,
fuzz 5 min/cible, benchstat, trivy, loadtest SQLite court (G.11), `test-os`
complet sans `-short`.
- [x] `nightly.yml` (quotidien, 03:00 UTC) ; issue automatique étiquetée
      `nightly` en cas de rouge — c'est elle qui a ouvert #317. **Décision
      écrite en tête du fichier : rien de ce qui a déjà un calendrier ne
      déménage.** `fuzz.yml` garde le fuzzing, le gold de recherche et le
      balayage arm64 ; `build.yml` garde Trivy, qu'un relecteur doit voir sur
      la PR qui l'a introduit. Trois calendriers, chacun justifié chez lui.

---

## Résumé du lot

| Fiche | Effort | Étape |
|---|---|---|
| E.1, E.2, E.4, E.5 | S | 1 |
| E.3 | M | 1 |
| E.6, E.7, E.8, E.10, E.11, E.12 | S | 2 |
| E.9 | M | 2 |
