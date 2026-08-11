# Fiche 03 — Corrections backend rapides

Branche : `fix/backend-quick-fixes`

## Objectif

Solder en une passe les défauts backend à effort S : modes de panne
silencieux, erreurs mal propagées, sorties polluées.

## Tâches

Chaque puce cite le constat ; corriger + test quand testable.

- [ ] **`cmd/serve` ne peut pas migrer** : `registeredMigrator` est enregistré
      par un `init()` de `package database`, jamais importé par `cmd/serve`
      (`migrate_hook.go:20-25`, `storage/sqlite/sqlite.go:181-184` retourne
      `nil` silencieusement). Faire échouer `Migrate` explicitement quand la
      base n'est pas fraîche et qu'aucun migrateur n'est enregistré, ET
      ajouter l'import blank dans `cmd/serve`. Test : ouvrir une base
      ancienne sans migrateur → erreur claire.
- [ ] **HTTP 200 sur erreur d'export** : `handlers_imports.go:110-122` pose
      les en-têtes binaires avant l'export ; en cas d'échec, enveloppe JSON en
      200/octet-stream. Poser les en-têtes après succès, utiliser
      `writeErrorDetails`. Test handler.
- [ ] **`OpenDatabase`/`SetupDatabase` fuient le verrou** (`db.go:659-667`,
      `:206-220`) : sur échec de pragmas/migration, libérer le verrou, fermer
      le handle, laisser le wrapper dans un état sain. **`Close()` sans mutex**
      (`db.go:155-164`) : prendre `d.mu` ; protéger `Conn()`.
- [ ] **CLI : stdout pollué** : `cli.go:133` et `:154` écrivent sur stdout
      avant toute commande → `info --format json | jq` casse. Rediriger vers
      stderr.
- [ ] **`blunderdb version` affiche la version du schéma** (`cli.go:122-125`
      imprime `DatabaseVersion`). Injecter la version applicative via
      `-ldflags -X` (alimentée par `wails.json`/release.sh) avec repli
      « dev » ; afficher les deux (app + schéma).
- [ ] **`printUsage` ignore `serve`, `call`, `migrate`** (`cli.go:96-119`
      vs `main.go:26-40`) : les ajouter avec un renvoi vers la doc headless.
- [ ] **CLI : 54 `fmt.Errorf(…%v)`** dans `internal/cli/` : passer à `%w`
      pour rendre `errors.Is(err, storage.ErrNotFound)` possible.
- [ ] **`call` bufferise toute la réponse** (`internal/server/call.go:117-124`,
      `httptest.NewRecorder`) : écrire un `http.ResponseWriter` minimal vers
      stdout implémentant `http.Flusher`.
- [ ] **Purge d'orphelins en N+1** (3 copies : `database/db_match.go:316-325`,
      `sqlite/matches_sqlite.go:326-335` + `:394-428`,
      `postgres/matches_postgres.go:341-352` + `:409-440`) : rendre la purge
      ensembliste (un `DELETE … WHERE id IN (…) AND NOT (held)` par copie).
      Le prédicat reste énoncé 3 fois ; dépend des tests de la fiche 02.
- [ ] **`positionIsHeldSQL` Postgres sans filtre tenant**
      (`matches_postgres.go:295-299`) : ajouter `tenant_id` aux 4 `EXISTS`
      par cohérence avec le reste du fichier.
- [ ] **500 sans trace serveur** (`internal/server/errors.go:85-92`) : loguer
      l'erreur d'origine (logger ou contexte de requête) avant de masquer en
      « internal error ».
- [ ] **`storage.Options.MigrationProgress` jamais câblé**
      (`storage.go:82-83`, ignoré par les deux backends) : le câbler côté
      sqlite (le passer au migrateur enregistré) ou le retirer du contrat.
- [ ] **Binding mort `PositionExists`** (`db_position.go:14-49`, O(n) avec
      re-marshal JSON ; plus aucun appelant frontend) : supprimer méthode et
      binding ; régénérer `frontend/wailsjs` au prochain `wails dev`.
- [ ] **Scans ignorés en silence** : les ~46 `rows.Scan → continue` des
      chemins d'export (`db_export.go`, `db_collection.go`,
      `db_tournament.go`) : `slog.Warn` + compteur remonté dans le résultat.
- [ ] **`http.Server` sans `IdleTimeout`** (`server.go:64-68`) : en poser un
      (les Read/Write timeouts restent absents à cause du streaming NDJSON).
- [ ] **Routes manquantes** : exposer `stats.tournamentBadges` et
      `matches.findByHash` (`storage/stats.go:266`, `storage/matches.go:41`)
      — additions mécaniques au routeur RPC.

## Critères de fin

- `go build ./cmd/serve` + scénario « base ancienne » → erreur explicite.
- `blunderdb info --db x.db --format json | jq .` fonctionne.
- Suite complète verte (`go test ./...`, `-tags postgres` si Docker dispo).

## Risques & garde-fous

- La purge ensembliste ne se merge qu'après la fiche 02 (tests de rétention
  renforcés).
- Ne pas renommer la route existante `anki.suspendCard` (compat clients).
