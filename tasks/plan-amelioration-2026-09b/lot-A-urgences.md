<!-- Lot du plan tasks/plan-amelioration-2026-09b/README.md. Le contexte, la
méthode et le tableau d'ensemble vivent dans le README ; ce fichier ne porte
que ses fiches. -->

# Lot A — Étape 0 : colmater (semaine 1)

Ce qui met en danger des données, une isolation ou une promesse écrite, et qui
se corrige en moins d'une journée chacun. Tout est **S** sauf mention. Ordre
libre, mais A.1 à A.4 avant tout le reste.

Format d'une fiche : constat vérifié (fichier:ligne), à faire, recette de
sortie, dépendances. Une fiche = une branche = une PR.

---

## A.1 — Un `X-Tenant-ID` non numérique tombe sur le tenant 0 [S] — sécurité, critique (#155)

**Constat.** `pkg/blunderdb/storage/tenant_context.go:24` et
`storage/postgres/positions_postgres.go:31` : `n, _ := strconv.ParseInt(scope, 10, 64)`,
erreur jetée. `alice`, `default`, `mon-tenant` valent tous `tenant_id = 0` et
partagent positions, matchs, analyses, collections, paquets Anki. Le
comportement est épinglé par `internal/server/middleware/tenant_test.go:83`
sans que la conséquence soit nommée ; la doc pousse dedans
(`mode_headless.rst` : `--scope` par défaut `default`, `migrate --tenant-id mon-tenant`) ;
le RLS n'attrape rien (le GUC vaut 0 aussi) ; `tenant.purge "alice"` détruit
tous les tenants nommés. Le loadtest ne génère que des tenants numériques
(`cmd/blunderdb-loadtest/scenario.go:117`), il ne l'a jamais vu.

**À faire.**
- [ ] `middleware.Tenant` : rejeter en 400 `code=invalid` tout scope qui n'est
      pas un entier décimal positif (message nommant le format attendu).
- [ ] Même garde dans `migrate --tenant-id`, `call --scope`, et dans
      `storage.ParseTenant` (retourner l'erreur au lieu de l'ignorer).
- [ ] Corriger `tenant_test.go:83` et la doc (`mode_headless.rst`, aide de
      `--scope`, `CLI_USAGE.md`) : exemples numériques uniquement.
- [ ] Décision à écrire dans l'ADR-0005 (amendement court) : « un tenant est un
      entier ; le proxy fait la correspondance nom → entier ». L'option table
      `tenant(scope TEXT UNIQUE, id BIGSERIAL)` est notée comme évolution
      possible, pas retenue ici.
- [ ] Test d'isolation à deux tenants **nommés** côté HTTP (A écrit, B lit → 404 ;
      `alice` → 400) et scénario loadtest à tenants nommés.

**Recette.** `curl -H 'X-Tenant-ID: alice' …` → 400 ; `TestTenantIsolationNamed`
vert ; `test-postgres` vert.

---

## A.2 — La table `metadata` est globale et exposée en écriture à tous les tenants [S/M] — sécurité, critique (#156)

**Constat.** `internal/server/handlers_metadata.go:31-36` expose
`metadata.load/save/setVersion` ; `storage/sqlshared/metadata.go:41,49,70`
ignorent `scope`. `metadata.load` renvoie l'état de session de tous les tenants
(`sqlshared/session.go:36` : clés `<scope>:session_*`) ; `metadata.save` peut
écrire `database_version` ; `metadata.setVersion` fait passer `/readyz` de
toute l'instance en 503 (`handlers/health.go:40`). `metadata` est hors RLS
(`rls_postgres.go:41`).

**À faire.**
- [ ] Retirer `metadata.load`, `metadata.save`, `metadata.setVersion` de `/v1`
      (les ajouter à la liste `serverOnly`/raisons de `parity_test.go` avec la
      raison « infrastructure, pas une donnée de tenant »).
- [ ] Garder `metadata.version` (lecture) et `metadata.counts`.
- [ ] Sortir la session de `metadata` : table `session(tenant_id, key, value)`
      (SQLite : `scope`), couverte par RLS et par `tenant.purge`. Bump
      `DatabaseVersion` 2.15.0 → 2.16.0, triple synchro schéma
      ([[project_schema_triple_sync]]), migration qui déplace les clés `*:session_*`.
- [ ] Test : `metadata.load` en 404 ; session A invisible à B (contrat storage
      + HTTP).

**Recette.** `routes_smoke_test` à jour ; `TestMigrationSteps_ContinuousChain`
vert ; PG : test de continuité (voir G.7) ou au minimum `TestMigratePostgres`.

**Dépendances.** A.1 (sinon deux tenants nommés partagent aussi la session).

---

## A.3 — Les PRAGMAs SQLite ne s'appliquent qu'à une connexion sur dix [S] — bug, corruption silencieuse (#157)

**Constat.** `pkg/blunderdb/database/db.go:223` et `:297` : `sql.Open("sqlite", path)`
avec un chemin nu, puis `ConfigurePool` porte le pool à 10 connexions, puis
`ApplyPragmas` ne touche que la connexion qu'il reçoit (son commentaire le
dit). Les connexions 2 à 10 tournent avec `foreign_keys=OFF` et
`busy_timeout=0`. `DeleteMatch` (`db_match.go:301`) et `DeletePosition`
reposent sur `ON DELETE CASCADE` : supprimer un match peut laisser
`game`/`move`/`move_analysis` orphelins. Invisible aux tests (`:memory:`
épinglé à 1 connexion). `storage/sqlite/sqlite.go:47` fait déjà
`sql.Open("sqlite", DSN(dsn))`.

**À faire.**
- [ ] `sql.Open("sqlite", sqlite.DSN(path))` aux deux sites.
- [ ] Test fichier : ouvrir 3 connexions concurrentes (`db.SetMaxOpenConns(3)`,
      3 goroutines bloquées dans une transaction), lire `PRAGMA foreign_keys`
      et `PRAGMA busy_timeout` sur chacune.
- [ ] `blunderdb verify` : compter les orphelins `game`/`move` sans parent et
      les signaler (les bases existantes peuvent en porter).

**Recette.** Test rouge avant, vert après ; `verify` sur une base de prod
personnelle rapporte le nombre d'orphelins.

---

## A.4 — Durcir le dépôt GitHub et les workflows [S] — sécurité, supply chain (#158)

**Constat.** `default_workflow_permissions: write` au niveau dépôt ;
`.github/workflows/aur.yml` sans bloc `permissions:` (token write-all avec les
secrets AUR SSH) ; `main` non protégée (404 sur `/branches/main/protection`) ;
secret scanning, push protection et Dependabot security updates désactivés ;
`allowed_actions: all`, `sha_pinning_required: false` ; 52 `uses:` par tag
mouvant alors que `SECURITY.md` dit « pinned by commit SHA ».

**À faire.**
- [ ] Réglages dépôt : `default_workflow_permissions: read` ; secret scanning +
      push protection + Dependabot security updates ; `allowed_actions:
      selected` (actions/*, github/*, + les 12 tierces déjà pinnées) ;
      `sha_pinning_required: true`.
- [ ] `aur.yml` : `permissions: contents: read` en tête.
- [ ] Ruleset sur `main` : status checks requis `test (*)`, `lint`,
      `govulncheck`, `frontend-lint`, `frontend-test` ; historique linéaire ;
      pas de force-push.
- [ ] Pinner par SHA les 52 `uses:` (Dependabot `github-actions` maintient
      ensuite), ou corriger la phrase de `SECURITY.md`. Choix : pinner.
- [ ] `build.yml:513-514` : le job `build` n'a plus `contents: write` ; un job
      `release` tag-only télécharge les artefacts et publie (le commentaire du
      fichier décrit déjà ce remède).

**Recette.** `gh api repos/kevung/blunderDB/actions/permissions/workflow` →
`read` ; une PR de test ne peut pas être fusionnée avec `test` rouge.

---

## A.5 — Les corpus de seeds fuzz ne tournent plus dans le job `test` [S] — fiabilité (#159)

**Constat.** `build.yml:49-52` : les shards `database-*` filtrent par
`-run '^Test[A-I]'` / `'^Test[^A-I]'` ; `-run` filtre aussi `Fuzz*`. Vérifié :
`go test -run '^Test[A-I]' -v ./pkg/blunderdb/domain/` lance 0 `Fuzz`, `-run Fuzz` en
lance 9. `FuzzBGFApplyCheckerMove` et ses seeds de régression ne tournent
plus, contrairement à ce que `fuzz.yml:5-8` affirme.

**À faire.**
- [ ] Shard supplémentaire (ou `-run '^(Test[A-I]|Fuzz)'`) qui joue tous les
      `Fuzz*` en mode seeds.
- [ ] Dédupliquer `database/db_import_bgf_fuzz_test.go` et la cible d'`ingest`
      (même fonction fuzzée deux fois) ; ajouter la cible manquante à `fuzz.yml`.
- [ ] `fuzz.yml` : `fuzztime` 90 s → 5 min par cible (le timeout de step le
      permet).

**Recette.** Le log du job `test` montre `FuzzXxx … seed#N` pour chaque cible.

---

## A.6 — Trois trous du serveur HTTP [S] — sécurité (#160)

**Constat.**
- Six sites renvoient `err.Error()` brut au client (DSN, chemins, SQL) :
  `handlers_maintenance.go:31`, `handlers_tenant.go:26`, `handlers_matches.go:172`,
  `handlers_imports.go:273,322`, `handlers_gammonnet.go:87`, `ndjson.go:31`,
  alors que `writeStorageError` (`errors.go:96`) masque déjà les internes.
- `server.go:129` exempte tout `/v1/imports.*` de `limitBody`, y compris
  `imports.cancel` qui n'a pas de `MaxBytesReader` (`handlers_imports.go:221`
  n'en pose que sur l'upload).
- `handlers_imports.go:60` `belongsTo(id, scope)` compare par préfixe : le
  tenant `a` satisfait `belongsTo("a-1-5", "a")` et peut annuler les imports du
  tenant `a-1` ; les ids sont un compteur global énumérable.

**À faire.**
- [ ] Router les six sites par `writeStorageError` ; pour NDJSON, `codeForErr`
      + message masqué si `internal`.
- [ ] Exempter la liste exacte des routes d'upload, pas le préfixe.
- [ ] Registre d'imports : stocker `scope` à part, comparer par égalité ; id
      opaque aléatoire (`crypto/rand`, 16 octets hex).
- [ ] Tests : erreur interne → corps `{"code":"internal","message":"internal error"}` ;
      `imports.cancel` avec corps de 64 Mio → 413 ; tenant `a` ne peut pas
      annuler l'import de `a-1`.

**Dépendances.** A.1 rend le troisième point moins probable (tenants
numériques) mais pas impossible (`1` et `1-…` n'existent plus ; garder le
correctif).

---

## A.7 — 96 chaînes en français sur les huit sites traduits publiés [S] — qualité (#161)

**Constat.** `scripts/doc-i18n-check.sh` sur `main` : 12 chaînes non traduites
× 8 langues (cli 2, manuel 5, raccourcis 5), venues des 4 commits doc
postérieurs au tag 0.35.0 (`9a15838d`, `36354ebe`, `0576b770`, `ab88f3da`).
Le job `pages` déploie à chaque push sur `main` (`build.yml:1018`) : les sites
en/de/el/es/fi/it/ja/ru affichent ces paragraphes en français.

**À faire.**
- [ ] Traduire les 12 msgid dans les 8 `.po` (rappel [[project_sphinx_po_regen]] :
      `sphinx-build -b gettext` dans `doc/build/gettext`, chemin relatif).
- [ ] Règle de process dans `CONTRIBUTING.md` (voir H.2) et dans le template de
      PR : « un `.rst` modifié = ses `.po` dans le même commit ».
- [ ] Job `pages` : déployer sur tag **et** sur `main` seulement si
      `docs-i18n-check` est vert (job `needs:` + `--strict`).

**Recette.** `doc-i18n-check.sh` → `0 untranslated + 0 fuzzy` ; les 8 sites
n'affichent plus de français.

---

## A.8 — La base de démonstration embarque des noms de personnes réelles [S] — risque (#162)

**Constat.** `internal/gui/demo.db.gz` (425 ko, dans chaque binaire distribué,
`.exe`, `.app`, `.deb`, `.rpm`, AUR, Flatpak, image Docker) : `player1_name`/
`player2_name` = noms réels de joueurs (six personnes), sans trace de
consentement. Incohérent avec ADR-0007. La base démontre aussi la moitié du
produit : 0 commentaire, 0 collection, 0 paquet Anki, analyses XG seulement,
`database_version` 2.9.0 migrée à chaque ouverture.

**À faire.**
- [ ] Régénérer `demo.db.gz` via la CLI : pseudonymes fictifs (ou « Joueur A /
      Joueur B ») dans `match` et `tournament` ; `database_version` courante.
- [ ] Enrichir : 2-3 collections thématiques, 10 commentaires taggés
      (`#blunder`, `#cube`), un paquet Anki avec 20 cartes et un journal de
      révisions, 50 positions analysées par gammonNet (`blunderdb analyze`).
- [ ] Script `scripts/build-demo-db.sh` versionné, pour que la régénération
      soit reproductible à chaque changement de schéma.
- [ ] Vérifier que le tour `general` et la doc (`manuel.rst` § `demo`) restent
      justes.

**Recette.** `sqlite3 demo.db "select distinct player1_name from match"` ne
contient aucun nom réel ; les panneaux Commentaires/Collections/Anki ne sont
plus vides en démo.

---

## A.9 — Aucune notice tierce dans les artefacts distribués [S] — risque (#163)

**Constat.** Pas de `THIRD_PARTY.md`/`NOTICE` racine. Le binaire embarque deux
bases produites par GNU Backgammon (`engine/gnubg_os6.bd`, `engine/race/gnubg_ts0.bd`)
sans notice ; `engine/gammonnet/NOTICE.gammonNet` renvoie à un `THIRD-PARTY.md`
inexistant ; les paquets n'installent que `LICENSE` ; les crédits n'existent
que dans l'onglet « À propos » de l'aide intégrée (base unilatérale seulement)
et la section *Remerciements* d'`index.rst` ne cite ni gammonNet, ni Strehl,
ni Kazaross-XG2, ni GNUbg, ni Schiemann alors que `manuel.rst:920-922` le promet.
Rappel : les tables `.bd` sont des données, pas du code GPL ([[project_audit_2026_09_plan]]) ;
il s'agit de créditer, pas de relicencier.

**À faire.**
- [ ] `THIRD_PARTY.md` racine : gammonNet/Strehl (MIT), MET Kazaross-XG2,
      Schiemann (coefficients de course), GNUbg one-sided **et** two-sided
      bearoff (données générées), xgparser/gnubgparser/bgfparser, modernc.org/
      sqlite, Wails, Svelte, two.js, chart.js, driver.js, Noto Sans JP (OFL),
      Nunito (OFL).
- [ ] Installé par nfpm (`.deb`/`.rpm`), PKGBUILD, tarball `INSTALL.txt`,
      Flatpak, `Dockerfile.serve` (`/usr/share/doc`).
- [ ] Section *Crédits* dans `index.rst` (9 langues à la release) et dans
      l'onglet À propos de l'aide (9 fichiers).
- [ ] Corriger le lien mort de `NOTICE.gammonNet`.

---

## A.10 — Le binaire Linux s'appelle `blunderDB`, la doc dit `blunderdb` [S] — adoption (#164)

**Constat.** `build/linux/nfpm.yaml:30` et `packaging/aur/PKGBUILD.in:22`
installent `/usr/bin/blunderDB` ; le tarball contient `blunderDB`
(`build.yml:626`). `CLI_USAGE.md`, `doc/source/cli.rst` (19 commandes) et
`telecharge_install.rst:70` (`./blunderdb`) utilisent la minuscule : `command
not found` pour tout utilisateur `.deb`/`.rpm`/AUR qui suit la doc. Homebrew
et winget ne sont pas touchés.

**À faire.**
- [ ] Symlink `/usr/bin/blunderdb → blunderDB` dans nfpm et PKGBUILD ; le
      tarball livre les deux noms (ou `INSTALL.txt` explique).
- [ ] `telecharge_install.rst:70` corrigé ; `.po` dans le même commit (A.7).
- [ ] Test de packaging en CI : après `dpkg -i`, `blunderdb version` répond.

---

## A.11 — Trois correctifs cryptographiques à une ligne [S] — sécurité (#165)

**Constat.**
- `issuance/container.go:55-77` + `crypto.go:52` : `gcm.Seal(nil, nonce, plaintext, nil)` —
  l'en-tête clair du `.dbx` (version, filigrane, sel, nonce) n'est pas lié à la
  charge utile par l'AEAD.
- `container.go:102,115,142` : trois `os.ReadFile` sans borne alors que
  `maxContainerPayload` (2 Gio) n'est vérifié qu'à l'écriture (`:57-60`).
- `engine/analysiscodec.go:48-55` : `zlib.NewReader` + `io.ReadAll` sans
  `LimitReader` sur un blob de base — une base importée d'un tiers peut porter
  une bombe de décompression.

**À faire.**
- [ ] Passer les octets d'en-tête en `additionalData` de `Seal`/`Open`. Le
      format change : **version de conteneur 2**, lecture de la v1 conservée
      (sans AAD), écriture en v2 ; test des deux chemins.
- [ ] `os.Stat` avant lecture, refus au-delà de `maxContainerPayload` ;
      `UnwrapContainer` streame vers le fichier de sortie.
- [ ] `io.LimitReader(r, maxAnalysisBytes)` (16 Mio) ; la cible
      `FuzzDecodeAnalysisFromStorage` reçoit un seed « bombe ».
- [ ] Stocker `t/m/p` d'Argon2id dans l'en-tête d'identité (`identity.go:44-48`)
      et vérifier `stored.Version` (`identity.go:175-206`).

**Recette.** Un `.dbx` v1 s'ouvre toujours ; un `.dbx` v2 dont l'en-tête est
modifié d'un octet échoue à l'`Open` ; un blob zlib de 10 Go nominaux est
refusé en < 1 s.

---

## A.12 — `/healthz` interroge la base [S] — fiabilité (#166)

**Constat.** `internal/server/handlers/health.go:24` : `Live` appelle
`Storage.Version` (un `SELECT`). Sous Kubernetes, une base momentanément
injoignable fait échouer la *liveness* et redémarre en boucle un processus
sain ; `/readyz` (`:34`) existe pour ça.

**À faire.**
- [ ] `Live` renvoie 200 sans toucher au stockage.
- [ ] `Dockerfile.serve` : `HEALTHCHECK CMD ["/usr/local/bin/blunderdb","healthcheck"]`
      via une sous-commande qui fait un GET local sur `/readyz` (l'image est
      distroless, pas de curl).
- [ ] Doc `mode_headless.rst` : liveness vs readiness en deux phrases.

---

## A.13 — Le filtre d'erreur de coup rend un résultat différent d'un lancement à l'autre [S] — bug (#167)

**Constat.** `storage/sqlshared/search_helpers.go:63-93` construit la liste des
coups joués depuis une `map` puis `:148-163` `break` au premier ; une position
jouée différemment dans deux matchs a plusieurs `checker_move`, l'erreur
retenue dépend de l'itération de la map. Une recherche `E>0.1` change d'un
lancement à l'autre.

**À faire.**
- [x] Décider et écrire : le filtre retient **l'erreur maximale** parmi les
      coups joués sur la position (c'est ce que l'utilisateur veut : « ai-je
      un jour fait un blunder ici »). Écrit dans `matchesMoveErrorFilter`,
      `CONTEXT.md` (Deduplication) et `cmd_mode.rst` (note sous la table des
      filtres). Le même défaut touchait le videau (`E` sur une décision de
      cube jouée deux fois) ; la recherche simple (non miroir) filtrait sur la
      colonne dénormalisée, qui note le premier coup trié — déterministe mais
      pas le max : elle laisse maintenant passer les positions jouées
      plusieurs façons vers le prédicat Go (`player1MultiPlayedSQL`), le
      blob n'étant décodé que pour celles-là. `database/` délègue à
      `storage` (rien à corriger).
- [x] Trier ou agréger, test avec une position jouée deux fois (fixture en
      mémoire via le contrat storage, `Search/MoveErrorFilterMaxOverPlays`,
      SQLite et PostgreSQL), 20 exécutions identiques, cas « max » explicite
      côté pion et côté videau, recherche simple et miroir.

---

## A.14 — Hygiène des tickets et du BACKLOG [S] — process (#168)

**Constat.** #119 (parapluie gammonNet) est livré et ouvert ; 3 des 4 issues
ouvertes n'ont aucun label ; `tasks/BACKLOG.md` porte cinq items périmés :
`use_cube` (livré, ADR-0023), « évaluation refusée = état nommé » (livré,
`gammonnet_eval.go:71-78`), `cli_import` / `SaveIndividualPosition` (livré
autrement : OR collant `positions_sqlite.go:68-75`), « troisième copie des
helpers de recherche » (`db_search.go` fait 59 lignes), « découpage
`db_session.go` 603 lignes » (254 lignes), assertion Playwright « Eval ne
défile pas » (`tests/e2e/eval-panel-no-scroll.spec.js`), dédup autocomplétion
inline (`EntityAutocomplete`). `docs/adr/README.md` s'arrête à 0024 (0025 mergé).

**À faire.**
- [ ] Fermer #119 avec un commentaire renvoyant aux ADR 0011-0024 et au
      README ; ouvrir deux successeurs étroits : « rollouts tronqués » (J.2) et
      « comparaison XG vs gammonNet sur sa base » (I.14).
- [ ] Labels : créer `import`, `search`, `stats`, `anki`, `eval`, `serve`,
      `i18n`, `packaging`, `docs` ; labelliser #151, #127, #102 ; marquer 3
      `good first issue` (coquilles FAQ H.9, symlink A.10, ADR-0025 index).
- [ ] `tasks/BACKLOG.md` : déplacer les sept items dans *Historique* avec commit
      et date ; corriger la ligne `race.Money` (« plus rien ne bloque »).
- [ ] `docs/adr/README.md` : ligne 0025.
- [ ] Rouvrir #114 (OGXM) et #61 (Heroes) reformulées, pointant I.4 et I.5.

---

## Sortie du lot

Tout le lot A est fusionné ⇒ publier **0.35.1** (correctifs de sécurité A.1, A.2,
A.3, A.6, A.11 ; note de release qui les nomme sans détailler l'exploitation).
La skill `release-blunderdb` s'applique ; les patchs n'ont pas de ligne de
changelog ([[project_changelog_major_only]]).
