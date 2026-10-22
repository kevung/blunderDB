# blunderDB — mesures d'optimisation (phase 1)

Ce rapport mesure et classe les pistes ; le dépôt n'a pas été modifié. Les mesures portent sur `main` à 1edb430f2, recopié par `git archive` dans `opt/src`. La machine a 28 cœurs et était chargée par d'autres ouvriers (load moyen 15 à 20). Chaque chiffre est une médiane, avec entre parenthèses le minimum et le maximum ; sauf mention contraire, il y a 5 répétitions.

La base de mesure est synthétique, produite par `blunderdb-synthdb -positions 300000`. Elle compte 300 154 positions, 300 154 analyses, 971 matchs, 60 tournois, 302 459 coups et pèse 193 Mo. Ses positions ne sont jamais jouées de plusieurs façons, ce qui la rend optimiste pour le filtre `E` (voir A2).

Les données brutes sont dans `opt/` :
- `bench-*.txt` : benchmarks ;
- `prof/*.cpu|*.mem|serve.heap` : profils pprof ;
- `nm-*.txt` : tailles des symboles ;
- `runs/` : CLI et daemon ;
- `front/` : mesures du frontend, faites par un sous-agent.

Le fichier de benchmarks ajouté pour ces mesures existe seulement dans la copie : `opt/src/pkg/blunderdb/database/zz_opt_bench_test.go`.

## 1. Taille

| Artefact | Octets | gzip -9 | xz -9 |
|---|---:|---:|---:|
| GUI `wails build -s -tags webkit2_41` (wails passe déjà `-ldflags "-w -s"`) | 46 213 120 | 19,6 Mo | 15,8 Mo |
| GUI `go build` + `-trimpath -s -w` | 46 135 072 (−78 ko) | 19,6 Mo | 15,8 Mo |
| GUI non strippé | 61 647 984 | 31,4 Mo | 27,3 Mo |
| GUI avec `frontend/dist` vide | 41 043 744 | | |
| daemon `cmd/serve` `-trimpath -s -w` (CGO=0) | 33 218 720 | 13,9 Mo | 11,3 Mo |
| daemon non strippé | 47 318 202 | | |

**Composition du binaire GUI** (sections ELF : `.text` 17,9 Mo, `.gopclntab` 12,8 Mo, `.rodata` 6,9 Mo, `.noptrdata` 4,5 Mo) :
- `frontend/dist` embarqué : **5,09 Mo**, sur 5,07 Mo de dist, qui se compresse en gzip à 2,60 Mo. Dans ce dist :
  - JS : 3,40 Mo, dont 1,53 Mo de locales ;
  - `help/*.bin` : 1,24 Mo, déjà gzippés ;
  - CSS : 230 ko ;
  - polices : 197 ko, dont NotoSansJP 178 ko.
- Données embarquées en Go :
  - réseau gammonNet : 2,25 Mo (float32 brut) ;
  - livre d'ouverture : 864 ko (gz) ;
  - base de démo : 320 ko (gz) ;
  - webui du daemon : 408 ko ;
  - icône : 112 ko ;
  - dictionnaires d'analyse : 98 ko ;
  - MET : 39 ko ;
  - réseau d'élagage : 26 ko.
- Code par module, hors BSS :

  | Module | Taille |
  |---|---:|
  | blunderdb `pkg/` | 7,7 Mo, dont 3,3 Mo de données ; code : sqlshared 0,97 Mo, server 0,74, cli 0,64, database 0,51, gui 0,47, ingest 0,46 |
  | modernc sqlite | 2,1 Mo |
  | crypto | 1,3 Mo |
  | pgx | 1,3 Mo (avec les symboles anonymes) |
  | net | 0,85 Mo |
  | wails | 0,46 Mo |
  | SDK MCP | 0,56 Mo (avec jsonschema-go 0,1) |
  | x/text | 0,33 Mo |
  | backgammon-tournoi | 0,29 Mo |
  | klauspost/compress | 0,28 Mo |

- `crypto/internal/fips140/drbg.memory` occupe 32 Mo de **BSS**. Ce n'est ni de la taille de fichier ni de la RSS tant que la page n'est pas touchée.

**Dépendances lourdes :**
- `golang.org/x/text` reste, importé directement par `storage` (norm/runes/transform) et `parser` (norm). Les parties language/precis (63 ko) viennent de x/net (idna).
- pgx est dans le GUI parce que `serve --backend postgres` et `migrate` vivent dans le même binaire : c'est un choix d'architecture.
- Le SDK MCP est importé par `internal/gui`, `internal/server`, `assistant` et `mcp`.
- testcontainers n'est **pas** dans les binaires : vérifié par `go list -deps`.

**CI :** les binaires Linux amd64 et Windows sont déjà `-upx` ; linux/arm64 et macOS ne le sont pas. UPX n'est pas installé ici, donc pas de mesure. UPX sur macOS est déconseillé : la signature et la notarisation cassent, et UPX 4 a retiré le support Mach-O récent.

## 2. Vitesse

### Ouverture, bibliothèque, listes (bindings `Database`, base de 300 k)
| Appel | Durée | Allocations/op |
|---|---:|---:|
| `OpenDatabase` + `GetDatabaseStatsEstimate` | 60 ms (46–97) | 0,5 Mo |
| `GetDatabaseStatsEstimate` (base ouverte) | 1,2 ms (1,1–2,4) | |
| `LoadPositionView` (affichage d'une position) | 0,63 ms (0,40–0,78) | 14 ko |
| `ListPositionIDs(0,1000)` | 1,4 ms (1,0–1,6) | |
| **`ListMatches` page 500, badges compris** | **1,06 s (1,00–1,27)** | 23 Mo, 0,8 M allocs |
| **`GetAllMatches` (971)** | **1,48 s (1,45–1,55)** | 46 Mo, 1,5 M allocs |
| **`GetAllTournaments` (60)** | **2,00 s (1,88–2,02)** | 47 Mo, 1,7 M allocs |

### Recherche
| Appel | Résultats | Durée | Allocations/op |
|---|---:|---:|---:|
| `LoadPositionIDsByFilters` sans filtre (GUI) | 300 154 | 322 ms (287–348) | 23 Mo |
| idem, cube | 120 056 | 211 ms (183–222) | 9 Mo |
| **idem, `E>100`** | 5 819 | **1,94 s (1,85–2,47)** | 36 Mo |
| idem, 6-5 au score 6-4 (étroit) | 11 | 20 ms (18–21) | 2,5 ko |
| `LoadPositionsByFilters` cube (CLI/daemon) | 120 058 | **2,03 s (1,92–2,08)** | **1,73 Go, 5,0 M allocs** |
| `LoadPositionsByFilters` `E>100` | 5 819 | 1,60 s (1,54–1,66) | 66 Mo |
| CLI `search --decision cube --limit 0 --format json`, processus complet | 120 058 | **52 s** (3 runs, ±1 s), RSS 746 Mo | |
| CLI `search --decision cube --limit 50` | 50 | 0,15 s | |

### Statistiques
| Appel | Durée | Allocations/op |
|---|---:|---:|
| `ComputeStats` (tableau de bord) | 1,04 s (0,98–1,05) | 38 Mo |
| **`GetPlayerTable`** | **2,33 s (2,13–3,15)** | 48 Mo, 1,7 M allocs |
| **`PlayerRanking`** | **2,05 s (1,98–3,12)** | 48 Mo |
| **`ComputeRecurringErrors`** | **1,86 s (1,84–2,00)** | 171 Mo |
| CLI `list --type stats`, processus complet | 1,41 s | RSS 236 Mo |

### Import et export
| Opération | Durée | Débit / remarque |
|---|---:|---:|
| `ImportXGMatch` d'un .xg (~370 positions) dans la base de 300 k | 490 ms (446–497) | 35 Mo, 0,6 M allocs |
| idem, base vide | 447 ms (439–459) | |
| CLI `import --type batch`, 11 .xg (4 081 positions, base neuve) | 5,60 s (5,48–6,27) | ~730 pos/s, RSS 140 Mo |
| synthdb, 300 k positions | 303 s (1 run) | 992 pos/s |
| **CLI `export --type database`, base de 300 k** | **210,7 s (209,2–210,7)**, 3 runs | RSS 330 Mo, **fichier de 940 Mo pour une source de 193 Mo** |

### gammonNet
| Repère | Valeur |
|---|---:|
| `Evaluate` scalaire, par position | 832 µs (764–900) |
| noyau AVX2 par lot, par position | 43 µs (41–52) |
| décision 2-ply k=12, en série, au money | 168 ms (146–192) |
| idem, au score | 726 ms (667–1071) |
| probas 2-ply, série / parallèle | 185 ms / 17,5 ms |
| lot d'analyse 0-ply, 28 goroutines | 9 516 pos/s (7 077–9 825) |
| lot d'analyse 2-ply, 28 goroutines | 22,0 pos/s (20,9–22,5) |

### Démarrage
Côté Go :
- `init` de tous les paquets : 7,4 ms (7,0–7,4), dont 4,3 ms pour `engine`.
- `blunderDB version` : 0,08 s, RSS 68 Mo (bibliothèques GTK/WebKit mappées) ; le daemon : 21 Mo.

Côté front : `measure-startup.mjs`, Chromium headless et mock Wails.

| Jalon | ×1 | CPU ×4 |
|---|---:|---:|
| prêt | 441 ms (371–505) | 1 021 ms (940–1 338) |
| FCP | 444 ms | 1 028 ms |
| JS chargé avant « prêt » | 1 198 ko (389 ko gzip), 82 fichiers | idem |
| préchargement oisif (46 chunks, 572 ko), fin à + | 574 ms | 1 079 ms |
| longues tâches du préchargement | 0 | 189 ms |

### Front, listes longues
La liste des matchs est virtualisée par `PanelTable` au-delà de 200 lignes.

| Matchs | Liste affichée (×1) |
|---:|---:|
| 1 | 559 ms |
| 5 000 | 613 ms (578–665) |
| 20 000 | 694 ms |

- 29 `<tr>` dans le DOM quel que soit N. Tri : 13 ms ; défilement : 30 ms.
- Navigation d'une position à la suivante : 16,8 ms (p90 20,5).

## 3. Mémoire

| Processus / scénario | Mesure |
|---|---:|
| Go, ouverture de la base de 300 k et compteur | RSS max 43 Mo (41–43) |
| Go, session type GUI : estimation, page de matchs, tournois, tous les ids, table des joueurs, tableau de bord, vue | RSS max **224 Mo** (224–224) |
| Go, recherche large en `[]Position` (chemin CLI/daemon) | RSS max **670 Mo** (646–683) |
| CLI, recherche large en JSON | RSS max 746 Mo |
| Daemon SQLite, `blunderdb-loadtest mixed`, 50 workers, 20 s, sans limite de débit | 653 RPS (652–655), p50 35 ms, p95 265 ms, p99 330 ms, **RSS crête 272 Mo (258–275)**, au repos ~15 Mo |
| idem, par endpoint (p50) | `search.find` 162 ms, `stats.compute` 102 ms, `positions.list` 19 ms, `positions.save` 17 ms |
| WebView, tas JS après GC | accueil 6,1 Mo, 1 536 nœuds ; 5 000 matchs 11,0 Mo, 2 999 nœuds |

Sur 41,5 Go alloués par le daemon pendant le test :
- 31,8 Go (76 %) viennent de `search.find`, par les copies de `[]Position` ;
- 12,4 Go (30 %) viennent de `compress/flate.NewWriter`, un nouveau writer gzip par réponse.

**Non mesuré :** la RSS du processus GUI réel (WebKitGTK). Il n'y a pas de Xvfb sur la machine, et j'ai refusé d'ouvrir une fenêtre sur l'écran de l'utilisateur. Le côté Go est couvert ci-dessus ; le tas JS est mesuré dans Chromium.

**Pool SQLite :** jusqu'à 10 connexions × `cache_size` de 64 Mo (plus un mmap de 256 Mo). C'est un plafond théorique de ~640 Mo de cache de pages, qui n'est pas atteint ici.

## 4. Profils des chemins les plus lents

1. **Recherche large en `[]Position`** (2,0 s, 1,73 Go/op) :
   - `Position` fait 536 octets. Elle est copiée par valeur dans trois `append` sans capacité initiale : `sqlshared/search.go:475` (`findWith`), `search.go:1012` (`applyGoFilters`) et `database/db_search.go:65`. Cela fait 1,44 Go/op à chaque site.
   - `runtime.memmove` représente 12,6 % du CPU.
   - Le même motif domine les allocations du daemon.
2. **Filtre `E>…`** (1,9 s pour 5 819 résultats) :
   - La condition `(err >= ? OR EXISTS(SELECT … move m1 JOIN move m2 …))` (`sqlshared/search_helpers.go:209`, `multiPlayedSQL`) exécute une sous-requête corrélée pour chaque position qui échoue au premier test, soit ~294 k sous-requêtes.
   - Mesure en sqlite3 natif :
     - colonne seule : 0,11 s ;
     - avec EXISTS : 0,74 s ;
     - ensemble « joué de plusieurs façons » calculé une seule fois par `GROUP BY` : 0,20 s.
3. **Badges des matchs et des tournois, table et classement des joueurs** (1 à 2,3 s) :
   - Le temps est passé presque entièrement dans `_sqlite3VdbeExec` : `MatchBadges`, `TournamentBadges` et `playerMWC7` (`sqlshared/stats.go`) font un agrégat complet à chaque appel.
   - C'est l'item déjà ouvert dans `BACKLOG.md`. À 971 matchs, cela fait 1 à 2 s par liste.
4. **Export** (0,7 ms/position, 210 s pour 300 k) :
   - `SaveAnalysisUncompressed` écrit l'analyse en **JSON** : 1 859 o en moyenne, contre 143 o en binaire zstd, soit 13× ; la table `analysis` passe de 56 Mo à 770 Mo.
   - Puis `CreateMove` → `PlayedDecisionArgs` **relit et re-décode ce JSON pour chaque coup** : 15,5 % du temps, dont 7 % de `json.Unmarshal`.
   - `CreateMove` pèse 28 % au total, `writePositions` 45 %.
5. **CLI `search --format json/table` sans limite** (52 s) :
   - `renderResults` appelle `LoadAnalysis` **deux fois par position** (directement, puis via `storedOrEncodedXGID`), en N+1 requêtes : 48 s sur 52.
   - `ActionLabelSQL` est reconstruit à chaque appel : 3,3 s.
6. **Import .xg** (~1,2 ms/position) :
   - SQLite représente ~36 % du temps.
   - `xgparser` passe par `encoding/binary.Read` réflexif : 10,8 %.
   - L'encodage zstd des analyses : 6 % ; `ingest.hasNonFinite` : 3 %.
   - `analysisStore.Merge` : 26 % ; `CreateMoves` : 16 %.
7. **gammonNet 2-ply** :
   - `denseAVX2ReLU` représente 71 % du temps, sur les couches cachées 512/512/256/256, et ce noyau est déjà vectorisé.
   - Le rassemblement des colonnes non nulles de la première couche (`network.go`, `EvaluateBatch` l. 178-182, copie de `row[j]` vers `batchWeight`) coûte ~8,7 %.
   - L'`Evaluate` scalaire (832 µs) ne pèse que 1,3 % : il sert seulement aux feuilles.

## 5. Pistes classées

Les gains sont estimés à partir des mesures ci-dessus. Effort : S (moins d'un jour), M (quelques jours), L (chantier).

### A. Sûres : aucun changement de comportement visible

| # | Piste | Gain estimé | Effort | Risque | Zone |
|---|---|---|---|---|---|
| A1 | CLI : charger les analyses des résultats **par lot**, une seule fois par position, au lieu de deux `LoadAnalysis` N+1 ; mettre `ActionLabelSQL` en cache | `search` large : 52 s → ~3–5 s ; RSS en baisse | S | faible (test de parité de sortie) | `internal/cli` `renderResults`, `storedOrEncodedXGID` ; `sqlite.analysisStore` (chargeur par lot existant, `DecodeAnalysesConcurrently`) |
| A2 | Filtre `E` : pour un appel sans pagination (`FindIDs` et `Find` complets), calculer une fois l'ensemble des positions jouées de plusieurs façons, au lieu de l'EXISTS corrélé ligne à ligne ; garder l'EXISTS pour la première page paginée | `E>100` : 1,9 s → ~0,4–0,6 s (×3 à ×4 d'après sqlite3 natif) | S-M | faible : les deux backends, tests de parité de recherche | `sqlshared/search.go:129`, `search_helpers.go:209` (et le dialecte PG) |
| A3 | Recherche en `[]Position` : pré-dimensionner ou filtrer en place, ne plus copier trois fois ; à terme, `[]*Position` ou un itérateur jusqu'au JSON | −1,3 Go/op d'allocations et ~−15–25 % de CPU sur une recherche large (memmove 12,6 %, GC) ; RSS crête du daemon en baisse | S | faible | `sqlshared/search.go:475,1012`, `database/db_search.go:39,65` |
| A4 | Daemon : réutiliser les writers gzip (`sync.Pool` + `Reset`) dans le middleware `Compress` | −30 % des allocations du daemon (12,4 Go sur 41,5), latence de queue | S | très faible | `internal/server/middleware/compress.go:103` |
| A5 | Export : garder en mémoire l'analyse décodée (ou ses arguments joués) écrite par `writePositions`, pour que `CreateMove` ne relise et ne re-décode pas le JSON à chaque coup | ~−15–20 % du temps d'export | S-M | faible : mêmes valeurs calculées, test d'égalité des bases exportées | `ingest/export_sqlite.go`, `sqlshared/played_decisions.go:70`, `sqlite.matchStore.CreateMove` |
| A6 | gammonNet : poids de la première couche stockés transposés (colonne par entrée) ; le rassemblement devient des copies contiguës | ~−5–8 % du temps d'une recherche 2-ply ; aucune arithmétique changée | M | faible si `kernel_identity_test` et l'empreinte restent identiques bit pour bit | `engine/gammonnet/network.go` (`EvaluateBatch`) ; optimisation d'implémentation, permise localement (CLAUDE.md du moteur) |
| A7 | `xgparser` : décodage manuel au lieu de `binary.Read` réflexif (dépôt de l'auteur) | ~−8–10 % du temps d'import d'un .xg | M | faible (corpus de fixtures) | `github.com/kevung/xgparser` |
| A8 | Front : `unicode-range` CJK sur NotoSansJP | −178 ko de téléchargement, de décodage et de mémoire de police hors japonais ; temps non mesurable | S | faible (contrôler visuellement ⧉, ⇢ et … dans WebKitGTK) | `frontend/src/style.css:56` |
| A9 | Front, petits re-rendus : `captureDoubtsStore.set([])` seulement si le tableau n'est pas vide ; tic d'1 s de `ProposalList` seulement quand un compte à rebours est actif | marginal | S | nul | `Board.svelte:557`, `direction/ProposalList.svelte:53` |
| A10 | Build : `-trimpath` dans `make build` et la CI (wails passe déjà `-s -w`) | −78 ko et des builds reproductibles | S | nul | `Makefile:26`, `build.yml` |

### B. Demandent une décision

| # | Piste | Gain estimé | Effort | Risque / question | Zone |
|---|---|---|---|---|---|
| B1 | **Exporter les analyses en binaire compressé**, plus en JSON | export de 300 k : 940 Mo → ~190 Mo (×5) ; temps d'export en baisse ; import du fichier plus rapide | S | Le JSON sert à la compatibilité avec les versions < 2.3.0. Une base exportée porte `database_version` 2.47 : ces versions peuvent-elles encore l'ouvrir ? Si elles ne le peuvent pas, la raison tombe. | `storage/sqlite/analyses_sqlite.go:301` (`SaveAnalysisUncompressed`) |
| B2 | **Badges, table et classement des joueurs pré-agrégés** (déjà dans le backlog) : badges mémorisés par match et invalidés à l'écriture, ou colonnes de synthèse tenues à l'import | listes de matchs et de tournois et table des joueurs : 1–2,3 s → quelques dizaines de ms | L | Schéma (`DatabaseVersion`, migration), parité des deux backends, invalidation | `sqlshared/stats.go` (`MatchBadges`, `TournamentBadges`, `playerMWC7`) |
| B3 | Indicateur dénormalisé « joué de plusieurs façons » sur `position`, tenu à l'écriture des coups : alternative plus complète à A2 | filtre `E` ~0,15 s, première page comprise | M | Changement de schéma et de migration ; les positions dédupliquées entre matchs doivent être maintenues | `sqlshared/search_helpers.go`, `db_schema.go` |
| B4 | `frontend/dist` embarqué compressé : gzip/brotli servis avec `Content-Encoding`, ou décompressés au lancement | binaire GUI −2,5 Mo (−5 %) | M | L'AssetServer de Wails et les schémas personnalisés de WebKitGTK/WebView2 doivent accepter `Content-Encoding` (à vérifier) ; sinon ~20 ms et ~5 Mo de RAM au lancement. Le gain disparaît en partie sous UPX ou dans l'archive de release. | `main.go:30`, `internal/gui/run.go` |
| B5 | Sortir pgx et `migrate`/`serve --backend postgres` du binaire GUI (balise de build ou binaire daemon dédié) | −1,3 Mo | M | Contredit « un seul binaire » (dispatch `os.Args[1]`) ; le CLI perdrait `migrate` | `main.go`, `storage/postgres`, `events/pgnotify` |
| B6 | Front : ne pas charger `en` en plus de la locale courante (clés de repli fusionnées au build) | ~30 ms ×1, ~70 ms ×4 au démarrage (estimation) | M | Une clé manquante s'afficherait brute | `frontend/src/i18n/index.js:9` |
| B7 | Front : ne pas charger MatchPanel (89 ko) avant « prêt » sans base ouverte | 10–25 ms ×1, 40–60 ms ×4 (estimation) | S-M | Premier affichage de la liste après ouverture | `uiStore.js:20` |
| B8 | Pool SQLite : `cache_size` 64 Mo × 10 connexions ; baisser à ~16 Mo, ou un cache partagé pour le GUI | plafond de RSS −480 Mo en pire cas (non atteint ici) | S | Peut ralentir les agrégats chauds : à mesurer avant | `storage/sqlite/sqlite.go:125,214` |
| B9 | gammonNet : quantification int8 ou poids float16 (`.bin` de 2,25 Mo → ~1,1 Mo, noyau ×2) | taille et vitesse du 2-ply | L | Change l'arithmétique : décision amont (gammonNet ADR-0003, invariant « le port suit l'amont ») | `engine/gammonnet` |
| B10 | UPX pour linux/arm64 ; pas pour macOS | ~−60 % de taille sur disque pour arm64 | S | Démarrage un peu plus lent, faux positifs antivirus ; à proscrire sur macOS (signature) | `build.yml:971` |

**B1, décision : exporter compressé.** Les versions de schéma < 2.3.0 sont celles des releases ≤ 0.16.0 (0.16.0 : `DatabaseVersion` 2.0.0 ; 0.17.0, première à compresser, 74de0d6e4 : 2.3.0 ; aucune release entre les deux). Leur `CheckVersion` ne compare que la majeure (`db.go:1612` de 0.16.0), donc 2.47 passerait ce contrôle ; mais `position.state` n'est plus un `Position` JSON complet : depuis 2.2.0 c'est le tableau JSON de 28 entiers (`EncodeBoardCompact`), depuis 2.31.0 le binaire de 28 octets (`EncodeBoardState`), qu'écrit l'export. 0.16.0 fait `json.Unmarshal` de cette colonne vers un `Position` (`PositionExists`, `LoadAllPositions`…), ce qui échoue sur l'un comme sur l'autre : elle échoue sur les positions bien avant les analyses. Le JSON des analyses ne protégeait plus aucun lecteur. L'export passe par `SaveExportedAnalysis` (encodage de la base vive) ; la lecture des anciens exports JSON reste (`TestImportV220DatabaseIntoV230`).

**Décisions du 2026-10-22 sur B.**

- **B2 : caduque, déjà livré** par la table `match_stats` (2c5259513 et suites). Mesure du 2026-10-22, base synthdb 300 k, CLI, ouverture (0,09 s) comprise : `matches` 0,16-0,19 s, `tournaments` 0,10-0,12 s, `players` 0,18-0,22 s, contre 1-2,3 s avant. Le premier remplissage de la table coûte 10,5 s, payé par le premier lecteur.
- **B3 : non** (A2 suffit). **B4 : non. B5 : non. B10 : non.**
- **B6 : fait.** Les clés de repli sont fusionnées au build (`frontend/i18nMergedLocales.js`, import `?merged`) : `en` n'est plus dans le chunk principal, une locale = un seul chargement, et un test vérifie que chaque locale porte toutes les clés anglaises (aucune clé brute). Mesure : au démarrage en français, le JSON chargé passe de 134 ko (`en`, chunk principal) + 147 ko (`fr`) à 153 ko (`fr` fusionné), soit −128 ko ; `JSON.parse` de chaque catalogue : ~1,0-1,2 ms. Le gain de temps est donc de l'ordre de la milliseconde, très en deçà des ~30 ms estimés : le gain réel est la taille du chunk principal et de la mémoire, pas la latence.
- **B8 : ne pas toucher.** Mesure du 2026-10-22, base synthdb 300 k, `cache_size` 64 Mo contre 16 Mo : écart de temps nul (listes, recherche large, E, statistiques, serve sous charge) ; RSS sans gain mesurable, dominée par les résultats matérialisés (~1,4 Go en recherche large).
- **B9 : étude en cours côté gammonNet**, rien ici.

### Écarté ou neutre
- **SDK MCP et x/text** : 0,56 et 0,33 Mo, tous deux utilisés pour de vrai. Le gain ne vaut pas l'effort.
- **testcontainers** : absent des binaires.
- **Init Go et ouverture de base** (7 ms, 60 ms) : négligeables.
- **Listes longues du front** : déjà virtualisées, quasi constantes jusqu'à 20 000 lignes.
- **Préchargement des chunks** : aucune longue tâche à ×1.

## Ordre suggéré
A1, A4, A3 puis A2 : quatre correctifs S, sûrs, sur les chemins les plus coûteux mesurés. Ensuite A5 et la décision B1 sur l'export (×5 de taille), puis le chantier B2 (badges et joueurs), qui est la principale latence visible dans le GUI sur une grosse base.
