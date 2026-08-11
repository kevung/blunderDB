# Plan d'amélioration — août 2026

Issu d'un audit complet (backend Go, frontend Svelte, tests/CI, docs/UX,
performance/espace) mené le 2026-08-11 sur la 0.32.0. Chaque fiche est une
étape autonome : une branche, un worktree, tests verts, merge sur main, push.

## Constats majeurs de l'audit

1. **Bug de livraison** : les trois exports `.db` (base complète via le chemin
   legacy, collections, tournois) écrivent un schéma obsolète (positions en
   JSON complet, `zobrist_hash` NULL, colonnes de filtre NULL) estampillé
   `DatabaseVersion` courante → chez le destinataire, dédup et recherches SQL
   muettes, fichiers ~14× trop gros. Vérifié sur artefacts réels.
2. **CI aveugle** : le backend PostgreSQL (~5 800 lignes, celui du daemon
   multi-tenant) est derrière `-tags postgres` que la CI ne passe jamais ;
   le workflow Fuzz est rouge depuis 2 semaines sur un faux positif.
3. **Rétention non verrouillée** : le critère `flagged` du prédicat
   `positionIsHeldSQL` (3 copies) n'est couvert par aucun test — une
   régression supprimerait silencieusement des positions utilisateur.
4. **Perf recherche** : `a.data` (~600 o/ligne) transféré et trié pour rien
   dans le cas nominal ; filtre de date en N+1 avec décompression par ligne ;
   requête win/gammon à 766 ms restructurable en ~100 ms ; ni `ANALYZE` ni
   `VACUUM` accessibles hors import CLI.
5. **Découvrabilité** : l'aide intégrée (9 langues) documente une commande
   supprimée et ignore 4 commandes + 7 filtres récents ; README avec exemple
   CLI faux et capture d'écran d'une UI disparue.
6. **UX destructive** : suppression de position, de la base bearoff (1,2 Go),
   reset Anki — sans confirmation ; recherche sans état d'occupation.

## Fiches, dans l'ordre d'exécution

| # | Fiche | Thème | Effort |
|---|-------|-------|--------|
| 01 | [fiche-01-ci-fiable.md](fiche-01-ci-fiable.md) | CI honnête : job Postgres, fuzz, benchmark, concurrency, caches | M |
| 02 | [fiche-02-filet-retention.md](fiche-02-filet-retention.md) | Tests rétention/purge + trous du contrat storage | M |
| 03 | [fiche-03-backend-quick-fixes.md](fiche-03-backend-quick-fixes.md) | Corrections backend S : serve/migrate, HTTP 200, stdout, %w… | M |
| 04 | [fiche-04-export-db.md](fiche-04-export-db.md) | **Bug export .db** : schéma courant, round-trip, allow-list | L |
| 05 | [fiche-05-perf-recherche.md](fiche-05-perf-recherche.md) | Perf SQL : a.data conditionnel, date, win/gammon, index, ANALYZE | M |
| 06 | [fiche-06-espace-vacuum.md](fiche-06-espace-vacuum.md) | Espace : commande vacuum CLI + GUI + doc | S |
| 07 | [fiche-07-frontend-quick-wins.md](fiche-07-frontend-quick-wins.md) | Code mort, i18n, confirmations destructives, busy state | M |
| 08 | [fiche-08-typographie-adr8.md](fiche-08-typographie-adr8.md) | Achever ADR-0008 : font inherit global, tokens dialogues | S |
| 09 | [fiche-09-accessibilite-clavier.md](fiche-09-accessibilite-clavier.md) | TabbedPanel clavier, aria, garde clavier partagée | M |
| 10 | [fiche-10-aide-docs.md](fiche-10-aide-docs.md) | Aide intégrée ×9 + test de sync, README, cli.rst, manuel | M |
| 11 | [fiche-11-tests-purs.md](fiche-11-tests-purs.md) | Tests unitaires purs : prédicats domain, config, utils | M |
| 12 | [fiche-12-perf-frontend.md](fiche-12-perf-frontend.md) | Board : redraws coalescés, resize via rAF | S |

Chantiers de fond identifiés mais **non planifiés ici** (gros, risqués, à
décider séparément) : [BACKLOG.md](BACKLOG.md).

## Règles d'exécution

- Un worktree par fiche (`git worktree add "$ROOT/../blunderDB-<fiche>" -b <branche>`),
  merge sur main puis push après validation ; jamais d'édition directe du
  checkout partagé.
- Avant merge : `go vet ./...`, `go test ./...`, `golangci-lint run`,
  `cd frontend && npm run lint && npm run format:check && npm test -- --run`.
- Toute feature visible embarque sa doc française (`doc/source/*.rst`) dans la
  même branche ; l'aide intégrée (`frontend/src/i18n/help/*.js`) compte comme
  surface de doc (leçon de l'audit).
- Un bump de `DatabaseVersion` exige la triple synchro schéma
  (database + storage/sqlite + storage/postgres) et un test de migration.
- Interdits inchangés : pas d'auth dans le daemon serve (ADR-0005), rien
  d'écrit côté destinataire (ADR-0007), export par allow-list uniquement.
