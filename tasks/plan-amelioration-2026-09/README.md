# Plan d'amélioration blunderDB — audit du 2026-09-02

État de départ : `main` @ 46a937ee, arbre propre, 0.33.1 taguée le 2026-08-26.
Vérifié localement : `go vet` OK, `golangci-lint` 0 issue, `govulncheck` 0 vuln,
`go test ./...` vert (sans -race), vitest 826/826, eslint + prettier OK.
CI GitHub : **rouge sur les 9 derniers pushs** (job `test`).

Format : `[effort S/M/L]` — S ≤ ½ journée, M ≤ 2-3 jours, L = chantier.

---

## Lot 0 — Urgent (cette semaine)

### 0.1 Réparer la CI [S]
- Cause : `go test -race -timeout 1200s ./...` dépasse 20 min sur
  `pkg/blunderdb/engine/gammonnet` (TestUndisputedOpeningPlays 4m54 sous -race
  quand le timeout tombe). Localement le paquet prend 51 s sans -race.
- Fix : step dédié `go test -count=1 -timeout 300s ./pkg/blunderdb/engine/gammonnet`
  sans -race ; le run `-race` global en `-short` ou avec le paquet exclu ;
  `timeout-minutes:` sur les 15 jobs (aujourd'hui 0).
- Fichier : `.github/workflows/build.yml:58`.

### 0.2 Sortir la release 0.34.0 [M]
- 59 commits non publiés depuis 0.33.1, dont 25 `feat` (campagne gammonNet /
  panneau Eval, ADR-0016→0022, `blunderdb analyze`).
- À faire dans la release : ligne 0.33.1 absente du changelog `index.rst`
  (×9 langues) ; 472 chaînes de doc à traduire (59 × 8 langues, delta
  0.33.1→HEAD, surtout `manuel.rst`) ; fermer les issues #124→#131 (toutes
  livrées dans le code, encore ouvertes) ; #119 reste comme parapluie, #133 et
  #102 restent ouvertes.
- Piloter avec la skill `release-blunderdb`.

### 0.3 `rows.Err()` jamais lu dans les migrations [S]
- `database/db_migration.go` : 10 boucles `rows.Next()`, 0 `rows.Err()`.
  Une erreur d'itération produit une migration partielle marquée réussie.
- 26 sites au total (stats_sqlite 7, stats_postgres 7, db_anki 2).
- Fix : activer `rowserrcheck` dans `.golangci.yml`, corriger les sites.

### 0.4 Bug `CommitImportDatabase` [S/M]
- Déjà dans `tasks/plan-amelioration-2026-08/BACKLOG.md` : colonnes scalaires
  laissées NULL → la dédup casse au 2ᵉ import. Perte silencieuse côté
  utilisateur. Le promouvoir en fiche datée et le corriger.

### 0.5 Titulaire du fichier LICENSE [S]
- `LICENSE` porte « Copyright (c) 2024 Facteur Pat » ; `wails.json` et
  `conf.py` disent Kévin Unger. Harmoniser.
- Note : les tables `.bd` embarquées (gnubg_os6, gnubg_ts0, TS-06-11) sont des
  données produites par gnubg, pas du code GPL — aucune question de licence.
  La MET Kazaross-XG2 est déjà créditée. Rien à faire de ce côté.

---

## Lot 1 — Gains rapides (chacun S, indépendants)

### Backend
1. **8 index de recherche absents côté PostgreSQL** : `idx_analysis_backgammon1/2`,
   `idx_analysis_gammon2`, `idx_analysis_win2`, `idx_position_back_checkers_1/2`,
   `idx_position_no_contact`, `idx_position_pip_1` (ajoutés par la fiche 05 en
   SQLite, jamais portés). Migration `010_search_range_indexes.sql`.
   (Les 3 `idx_*_scope` manquants sont normaux : PG indexe par tenant.)
2. **Test de fumée piloté par `srv.Paths()`** : 102/131 routes `/v1/*` sans
   aucun test HTTP (anki 22, collections 17, tournois 12, stats 7…). Un test
   table-driven de ~80 lignes : sans tenant → 400 ; body `{}` → jamais 404/500 ;
   Content-Type conforme (json vs ndjson).
3. **Test de parité de schéma** : le schéma SQLite est déclaré 4 fois
   (`db.go` SetupDatabase, `db_schema.go` ensureAllTablesExist,
   `storage/sqlite/schema_sqlite.go`, chaîne de migration). Comparer
   `sqlite_master` normalisé entre base fraîche et base migrée.
4. **Extraire `search_helpers`** : ~292 lignes de fonctions pures copiées
   verbatim entre `storage/sqlite` et `storage/postgres` → paquet partagé.
5. **Découper `storagetest/contract.go`** (2 486 lignes, table de 39 cas déjà
   en place) en 6 fichiers ; ajouter les cas manquants `Comments/*` (0/8
   méthodes) et `Collections` (2/17).
6. **`go.mod`** : ligne `ignore frontend/node_modules` (Go 1.25) — aujourd'hui
   `go test ./...` compile `frontend/node_modules/flatted/golang/...` ;
   commenter le `replace go-webview2 v1.0.16` (downgrade sans justification) ;
   supprimer la ligne `// replace … /home/unger/go/pkg/mod`.
7. **`Database.Conn()`** exporté donc bindé par Wails (sérialise en `{}`) :
   remplacer par `Checkpoint()`/`Analyze()` d'intention.
8. **Documenter `cmd/calibrace`** (CLAUDE.md + `//go:generate` dans
   `correction_coeffs.go`).

### Frontend
9. **Coalescer les 15 `loadAllPositions()` d'`importService.js`** en un seul
   appel en fin de lot : aujourd'hui un import de dossier recharge toute la
   table des positions à travers l'IPC Wails à chaque fichier. Meilleur
   rapport gain/effort du front.
10. **Poids du binaire** : `NotoSansJP-Regular.ttf` (5,7 Mo, 2,9× tout le JS)
    embarqué dans chaque binaire des 4 plateformes pour un seul verdict en
    japonais. Sous-ensembler + WOFF2 → ~250 Ko. `chart.js` en
    `await import()` dans les 4 composants graphiques (chunk unique de 1,96 Mo).
11. **Nettoyage `package.json`** : `@wails/runtime` est une dépendance morte
    (0 import, le runtime vient de `wailsjs/`) ; `@vitest/coverage-v8` installé
    sans script `test:coverage`.
12. **Test-contrat XGID fantôme** : `domain/xgid_contract_test.go` et
    `testdata/xgid_corpus.json` citent `frontend/src/__tests__/xgidContract.test.js`
    qui n'existe plus (le parseur JS a été supprimé, la GUI parse via Go).
    Corriger les commentaires, ou mieux : ajouter `xgidCanonical` au corpus et
    tester `generateXGID` (positionService.js:86), seul encodeur XGID complet
    du projet, aujourd'hui non testé.
13. **Garde `font-size`** : ADR-0008 migré à 99,3 % (2 `0.92em` restants dans
    `ConfigModal.svelte:756` et `MetadataPanel.svelte:205`). Ajouter un test
    style `i18nKeys.sync.test.js` qui échoue sur toute valeur absolue.
14. **A11y** : `MergePlayersModal` est le seul `role="dialog"` sans
    `trapFocus` (action destructrice) ; `$state(new Set())` dans
    `CollectionPanel.svelte:52` → `SvelteSet` (dernière occurrence, permet de
    passer `prefer-svelte-reactivity` en `error`) ; warning vite-plugin-svelte
    `FileImportProgressModal.svelte:27` (dialog sans tabindex).
15. **Exporter `letter()`** depuis `keyboardService.js` : la règle est
    réécrite 5 fois (`HelpModal`, `AnalysisPanel`, `MatchPanel`,
    `TournamentPanel`) sans le garde `key.length === 1`.
16. **Étendre `helpVocabulary.sync.test.js`** aux 4 onglets d'aide × 9 langues
    (parité mesurée parfaite : 35 h3 / 42 li / 212 tr partout, mais seul le
    compteur `<tr>` de `commands` est testé).

### Outillage / process
17. **Job `docs`** : `paths:` sur `doc/**`, `cache: pip`, PDF LaTeX sur tag
    seulement (646 s + ~1 Go d'apt sur chaque PR, même sans `.rst` touché).
18. **`govulncheck` en `continue-on-error: true`** (`build.yml:309`) alors que
    CLAUDE.md dit « CI enforces ». Retirer le flag.
19. **`dependabot.yml`** (gomod, npm `/frontend`, github-actions) — rien ne
    couvre npm ni les 13 actions non pinnées. Pinner par SHA au moins
    `github-actions-deploy-aur` (clé SSH AUR) et `action-gh-release`.
20. **`make check`** enchaînant les 7 commandes pré-push de CLAUDE.md, et
    `scripts/release.sh --check` dans le job `lint` (le script existe, rien ne
    l'appelle).
21. **`.dockerignore`** : le contexte `docker build -f Dockerfile.hostile .`
    local pèse ~1,9 Go (`gnubg_ts6x11.bd` 1,2 Go, `.venv` 115 Mo, `gnubg/`
    79 Mo, `build/`). Ajouter `.venv/` au `.gitignore` (ignoré nulle part).
    Supprimer `tmp/`, `a.db`, `c.db`, `toto*.db`, les WAL orphelins.
22. **CLAUDE.md faux sur deux points** : `analyze` manque dans la liste des
    commandes CLI (la phrase même qui dit « one place ») ; `CheckVersion` est
    dans `db_schema.go:426`, pas `db_migration.go`, et ne compare que la
    majeure. `SKILL.md:374` de release-blunderdb oublie `wails.json`.
23. **`docs/adr/README.md`** : index des 22 ADR (n°, titre court, statut, date,
    amende / amendé par). Aucun index aujourd'hui.
24. **`SECURITY.md`** (le produit chiffre des bases et expose un démon sans
    auth par conception), `CODEOWNERS`, template de PR rappelant la checklist
    CLAUDE.md (doc FR dans la branche, DatabaseVersion, prédicat ×3).
25. **README** : 5 fonctionnalités manquantes (Stats, gammonNet, diffusion
    contrôlée .dbx, export .mat, modes serve/call/migrate), 1 périmée
    (« EPC calculator » → panneau Eval), AUR/deb/rpm non cités, badge doc
    statique, section Contributing en deçà de la CI, aucune capture
    (FOLLOWUPS #7).
26. **Hygiène des tickets** : fermer #124→#131 ; labelliser ; fusionner
    `tasks/FOLLOWUPS.md` et `tasks/plan-amelioration-2026-08/BACKLOG.md` en un
    `tasks/BACKLOG.md` ; cocher `tasks/ts-bearoff/README.md` (21 cases non
    cochées alors que c'est livré) ; ouvrir des issues pour les 5 dettes
    nommées dans les ADR (use_cube 0016, no-scroll Playwright 0018, cellule
    vide sur refus moteur 0017, flag proxy 0005, 2 cas rouges de
    TestIntegrationGate).
27. **Doc** : épingler `doc/requirements.txt` (0 version épinglée, produit les
    PDF de release) ; supprimer `doc/source/locale/fr/` (locale source, 0 %).

---

## Lot 2 — Chantiers M

### Backend
- **Linters** : activer `errorlint` (30), `rowserrcheck` (10), `sqlclosecheck`
  (38, dont `db_migration.go:1700` déjà noté au BACKLOG), `bodyclose`,
  `copyloopvar` ≈ 80 corrections mécaniques ; lever l'exclusion errcheck sur
  `_test.go` (la tâche 12 qui la justifiait est faite). Ne pas activer `noctx`.
- **Registre déclaratif de migrations** : `runMigrationChain` = 558 lignes de
  `if version == …` ; passer à `[]{from, to, fn}` (~120 lignes), scinder
  `db_migration.go` (2 077 l.), et aligner sur le modèle `.sql` numéroté de
  PostgreSQL pour les étapes DDL pures.
- **Parité CLI** : `anki` (22 routes serveur, 15 méthodes GUI, 0 commande
  CLI) et `collections` (17 routes, 0 sous-commande). Route serveur
  `maintenance.vacuum`. Test de parité automatisé `Database` ↔ `handlers()`
  ↔ `srv.Paths()` avec allow-list.
- **Tests manquants** : `handlers/health.go` (sonde readiness), `metrics/`,
  `X-Tenant-ID` malformé, deadlock-guard sur les 139 méthodes de `Database`
  (RWMutex non ré-entrant, aucune garde).
- **Job `docker-serve`** : `Dockerfile.serve` n'est jamais construit en CI ni
  publié (GHCR). Job `flatpak` : le manifeste existe, jamais construit.

### Frontend
- **`filterModel.js`** : 127 `$state` dans `SearchPanel.svelte`, dont 22
  filtres × 5 variables réénumérées à la main dans 4 fonctions (~620 lignes
  d'énumération, 28 % du composant) + une 5ᵉ fois dans
  `searchFilterService.js`. Table déclarative de 22 entrées. Filet existant :
  `searchFilterService.test.js` (582 l.).
- **`App.svelte`** (763 l.) : ~250 lignes déplaçables en 5 lots S — littéral
  d'analyse vide recopié d'`analysisStore` (34 l.), 16 modales à plat dont 9
  `DataTableModal`, poignée de redimensionnement, drag & drop, 22 callbacks
  de Toolbar.
- **Panneaux** : `PanelTable.svelte` + `utils/inlineEdit.js` +
  `utils/reorder.js` + `EntityAutocomplete.svelte` : 8 sélecteurs CSS
  répétés dans 4 panneaux (221 l., déjà divergents sur `.icon-btn`), édition
  inline ×8, autocomplete tournoi ×2.
- **`boardScene.js`** puis couche statique/dynamique two.js : `drawBoard()`
  (623 l.) fait `two.clear()` et recrée ~120 nœuds dont la moitié statiques
  (24 triangles, 24 libellés) à chaque survol de coup.
- **`analysisRows.js`** : `clipboardService.js` redessine l'analyse sur canvas
  avec `formatEquity` écrit 4 fois — l'image copiée peut diverger de l'écran.
- **`modeMachine.js`** : sortir l'automate EDIT/EPC/MATCH/COLLECTION et ses 6
  globales de `positionService.js` (1 574 l., 1 test sur 40 exports).
- **Tests** : `sessionService` (restauration au démarrage, 0 test),
  `AnkiPanel` (1 358 l., 0 test), `MergePlayersModal` (fusion irréversible,
  0 test) ; 3 specs e2e sur les flux principaux (recherche, navigation match,
  import) — les 5 specs actuelles ne couvrent aucun des 4 flux majeurs.

### Doc / distribution
- Job `docs-i18n-check` (msgmerge contre le .pot régénéré, échec sur tag).
- Capture d'écran README + manuel (0 illustration hors annexes), vidéo #102.
- winget + Homebrew cask (binaires non signés : ce sont les canaux qui
  contournent le mieux SmartScreen/Gatekeeper).

---

## Lot 3 — Chantiers de fond (L)

1. **Le wrapper `database/` ne délègue pas** : 11 646 lignes hors tests,
   370 appels SQL bruts, 27 références à `d.store`. Seuls position, stats,
   search et mat_export (~700 l., 6 %) sont portés sur `storage/sqlite`.
   Collections, tournois, anki, sessions, commentaires existent en double
   avec deux jeux de tests. Migrer famille par famille (comment → collection
   → tournament → anki → session), ce qui supprime aussi `Database.mu`
   (269 prises de verrou global que les backends n'ont plus) et le trou de
   contexte (2/139 méthodes avec ctx). Prérequis : le CLI est structurellement
   lié au wrapper (`internal/cli/aliases.go`).
2. **Adaptateur `Execer` partagé sqlite/postgres** : 1 875 lignes strictement
   identiques entre les 16 paires de fichiers (stats 550, search 261,
   matches 229…), `stats_*.go` déjà en dérive documentaire. Unifier stats,
   search, history, metadata, session, filters, comments (~1 200 l.) ;
   laisser divergentes matches/positions/anki/tournaments/collections.
3. **Pagination de `LoadAllPositions`** : toute la table est matérialisée et
   sérialisée en JSON via Wails (~1 Ko/position) à chaque rechargement ;
   `ListOpts` existe déjà dans le contrat, passé vide. Touche les 3 modes.
4. **Schéma DDL unique** : 7 copies / 93 `CREATE TABLE` (BACKLOG 2026-08).
5. **Composant `<Modal>` unique** (13 copies du CSS d'overlay, 7 handlers
   Escape).
6. **#133** noyau d'inférence vectorisé (×7,38 côté C) — amont gammonNet.
7. Soumission Flathub (build from-source offline) ; matrice OS du job `test`.

---

## Ce qui est sain (ne pas y toucher)
- Invariant Svelte 5 store : 6 `.subscribe()`, tous justifiés et commentés.
- i18n labels : 871 clés × 9 langues, 0 écart, 4 tests-gardes bidirectionnels.
- `CLI_USAGE.md` ↔ `handlers()` : écart zéro.
- Gestion d'erreurs Go : 0 `log.Fatal`, 0 `os.Exit` hors main, 2 `panic`
  légitimes.
- `internal/server` : table de routage unique, middleware ordonné et justifié,
  cardinalité des métriques bornée.
- `.gitignore` commenté incident par incident ; `CONTEXT.md` à jour (2026-08-31).
