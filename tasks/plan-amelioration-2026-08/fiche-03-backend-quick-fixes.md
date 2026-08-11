# Fiche 03 — Corrections backend rapides

Branche : `fix/backend-quick-fixes`

## Objectif

Solder en une passe les défauts backend à effort S : modes de panne
silencieux, erreurs mal propagées, sorties polluées.

## Tâches

Chaque puce cite le constat ; corriger + test quand testable.

- [x] **`cmd/serve` ne peut pas migrer** : `registeredMigrator` est enregistré
      par un `init()` de `package database`, jamais importé par `cmd/serve`
      (`migrate_hook.go:20-25`, `storage/sqlite/sqlite.go:181-184` retourne
      `nil` silencieusement). Faire échouer `Migrate` explicitement quand la
      base n'est pas fraîche et qu'aucun migrateur n'est enregistré, ET
      ajouter l'import blank dans `cmd/serve`. Test : ouvrir une base
      ancienne sans migrateur → erreur claire.
- [x] **HTTP 200 sur erreur d'export** : `handlers_imports.go:110-122` pose
      les en-têtes binaires avant l'export ; en cas d'échec, enveloppe JSON en
      200/octet-stream. Poser les en-têtes après succès, utiliser
      `writeErrorDetails`. Test handler.
- [x] **`OpenDatabase`/`SetupDatabase` fuient le verrou** (`db.go:659-667`,
      `:206-220`) : sur échec de pragmas/migration, libérer le verrou, fermer
      le handle, laisser le wrapper dans un état sain. **`Close()` sans mutex**
      (`db.go:155-164`) : prendre `d.mu` ; protéger `Conn()`.
- [x] **CLI : stdout pollué** : `cli.go:133` et `:154` écrivent sur stdout
      avant toute commande → `info --format json | jq` casse. Rediriger vers
      stderr.
- [x] **`blunderdb version` affiche la version du schéma** (`cli.go:122-125`
      imprime `DatabaseVersion`). Injecter la version applicative via
      `-ldflags -X` (alimentée par `wails.json`/release.sh) avec repli
      « dev » ; afficher les deux (app + schéma).
- [x] **`printUsage` ignore `serve`, `call`, `migrate`** (`cli.go:96-119`
      vs `main.go:26-40`) : les ajouter avec un renvoi vers la doc headless.
- [x] **CLI : 54 `fmt.Errorf(…%v)`** dans `internal/cli/` : passer à `%w`
      pour rendre `errors.Is(err, storage.ErrNotFound)` possible.
- [x] **`call` bufferise toute la réponse** (`internal/server/call.go:117-124`,
      `httptest.NewRecorder`) : écrire un `http.ResponseWriter` minimal vers
      stdout implémentant `http.Flusher`.
- [x] **Purge d'orphelins en N+1** (3 copies : `database/db_match.go:316-325`,
      `sqlite/matches_sqlite.go:326-335` + `:394-428`,
      `postgres/matches_postgres.go:341-352` + `:409-440`) : rendre la purge
      ensembliste (un `DELETE … WHERE id IN (…) AND NOT (held)` par copie).
      Le prédicat reste énoncé 3 fois ; dépend des tests de la fiche 02.
- [x] **`positionIsHeldSQL` Postgres sans filtre tenant**
      (`matches_postgres.go:295-299`) : ajouter `tenant_id` aux 4 `EXISTS`
      par cohérence avec le reste du fichier.
- [x] **500 sans trace serveur** (`internal/server/errors.go:85-92`) : loguer
      l'erreur d'origine (logger ou contexte de requête) avant de masquer en
      « internal error ».
- [x] **`storage.Options.MigrationProgress` jamais câblé**
      (`storage.go:82-83`, ignoré par les deux backends) : le câbler côté
      sqlite (le passer au migrateur enregistré) ou le retirer du contrat.
- [x] **Binding mort `PositionExists`** (`db_position.go:14-49`, O(n) avec
      re-marshal JSON ; plus aucun appelant frontend) : supprimer méthode et
      binding ; régénérer `frontend/wailsjs` au prochain `wails dev`.
- [x] **Scans ignorés en silence** : les ~46 `rows.Scan → continue` des
      chemins d'export (`db_export.go`, `db_collection.go`,
      `db_tournament.go`) : `slog.Warn` + compteur remonté dans le résultat.
- [x] **`http.Server` sans `IdleTimeout`** (`server.go:64-68`) : en poser un
      (les Read/Write timeouts restent absents à cause du streaming NDJSON).
- [x] **Routes manquantes** : exposer `stats.tournamentBadges` et
      `matches.findByHash` (`storage/stats.go:266`, `storage/matches.go:41`)
      — additions mécaniques au routeur RPC.

## Critères de fin

- [x] `go build ./cmd/serve` + scénario « base ancienne » → erreur explicite.
- [x] `blunderdb info --db x.db --format json | jq .` fonctionne.
- [x] Suite complète verte (`go test ./...`, `-tags postgres` avec Docker —
      75 s, contrat storagetest complet contre une vraie PostgreSQL).

## Risques & garde-fous

- La purge ensembliste ne se merge qu'après la fiche 02 (tests de rétention
  renforcés). Fiche 02 mergée avant le début de celle-ci ; filet en place.
- Ne pas renommer la route existante `anki.suspendCard` (compat clients) —
  respecté, aucune route existante renommée.

## Notes d'exécution

Exécutée dans `/home/unger/src/blunderDB-fiche03` (branche
`fix/backend-quick-fixes`), un commit par tâche (12 commits + 1 correctif
`.gitignore` préalable). `go vet`, `go build`, `go test ./...` et
`golangci-lint run` verts avant chaque commit ; suite complète + `-tags
postgres` (Docker, testcontainers-go) et suite frontend (`npm run lint`,
683 tests) revérifiées à la fin.

Choix faits sur les points laissés ouverts par la fiche :

- **`cmd/serve` migrate** : les deux volets demandés — `database` compile et
  tourne sous `CGO_ENABLED=0` (vérifié), donc l'import blank direct dans
  `cmd/serve/main.go` a suffi ; pas besoin du plan de repli (petit paquet
  d'enregistrement séparé). Côté `storage/sqlite`, `Migrate` compare
  désormais la version enregistrée à `domain.DatabaseVersion` quand aucun
  migrateur n'est disponible : déjà à jour → no-op (comportement historique
  inchangé) ; obsolète → erreur nommant l'import à ajouter.
- **`MigrationProgress`** : câblé (pas retiré), côté sqlite uniquement —
  Postgres migre en avant via de petits scripts SQL numérotés, pas un
  backfill ligne à ligne, donc la progression n'y a pas la même valeur.
  Premier appelant réel : `blunderdb migrate`, qui émet désormais des
  événements NDJSON `"schema-migration"` pendant la mise à niveau d'une
  base source ancienne.
- **Scans ignorés en silence — pas de compteur dans la valeur de retour** :
  `slog.Warn` posé sur tous les sites identifiés (db_export.go,
  db_collection.go, db_tournament.go — au-delà des ~46 annoncés, plusieurs
  variantes `if err == nil { … }` du même bug ont aussi été corrigées).
  Pour `ExportDatabase` (le cas le plus abouti), un compteur `skipped` local
  est agrégé et loggué en résumé de fin d'export, mais **pas remonté dans la
  signature de retour** : la fonction a déjà 40+ chemins de retour d'erreur
  et trois appelants CLI externes — changer sa signature pour un correctif
  « rapide » aurait été disproportionné. `db_collection.go`/
  `db_tournament.go` n'ont pas de compteur du tout (trop de fonctions aux
  signatures et appelants différents pour un changement de contrat cohérent
  dans ce même effort) ; le logging seul reste la valeur ajoutée réelle —
  avant, ces échecs ne laissaient absolument aucune trace.
- **`positionIsHeldSQL` Postgres + tenant_id** : fusionné avec la tâche de
  purge ensembliste (mêmes lignes touchées). Le garde-fou
  `tests/position_is_held_predicate_test.go` a dû être ajusté pour exclure
  `tenant_id` de la comparaison d'ensemble — asymétrie délibérée
  (Postgres est multi-tenant, SQLite/database ne le sont pas) ; vérifié que
  le test détecte toujours une vraie dérive (clause `flagged` retirée
  temporairement côté Postgres → échec, remise en place → succès).

Aucun fichier de la zone de coordination (`storage/metadata.go`,
`storage/sqlite/metadata_sqlite.go`, `storage/postgres/metadata_postgres.go`,
`storagetest/contract.go`) n'a été touché.

Point restant, hors périmètre de cette fiche : un `.gitignore` préexistant
(motif `blunderdb` sans slash) masquait tout nouveau fichier ajouté sous
`pkg/blunderdb/` à `git add` — corrigé en premier (`/blunderdb`, ancré à la
racine) car il bloquait l'ajout des nouveaux fichiers de test de cette
fiche elle-même.
