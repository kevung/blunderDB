# blunderDB — balayage de performance de la surface (2026-10-21)

Mesure seule, rien n'est optimisé. Script : `scripts/bench-sweep.sh <base.db> [binaire]`
(copie la base, chronomètre chaque opération, médiane de 3 sous 20 s, RSS crête par
`wait4`/`ru_maxrss`, tableau Markdown trié). Phase 1 des mesures et pistes A/B :
`tasks/optimisation-2026-10.md` (cité « optim » ci-dessous).

## 1. Conditions

- Base : copie de `bench-scale/mesure-0.37/sg/sample.db` (270 Mo) : 312 586 positions,
  312 544 analyses, **33 319 matchs**, 268 724 parties, 332 152 coups, 5 011 tournois,
  `database_version` 2.47.0. Aucune collection, paquet ni leçon : le script en crée un de
  chaque (500 positions `E>100`) sur la copie, hors chronomètre.
- Particularités de cette base, qui pèsent sur plusieurs lignes :
  - les 312 544 analyses sont au **format hérité** (JSON compressé zstd, octet `0x28`) ;
    `maintenance status` : `analysis.legacy never run`. Tout décodage passe par
    `json.Unmarshal` ; un `reencode` changerait les chiffres de décodage ;
  - le drapeau `played_decisions_pending` est levé : la première ouverture rejoue le pas
    de migration 2.39 et vide le cache `match_stats`, que la première lecture de
    statistiques remplit. Le script chronomètre ces deux coûts à part (lignes 5 et 6).
- Import : l'archive `~/dwn/bmab-2025-06-23.zip` n'existe plus sur la machine ; les 50 `.xg`
  viennent de `~/dwn/bmab-sample-2000` (1 sur 7 parmi ceux que la base contient déjà).
  Tous étant des doublons, le script supprime d'abord ces 50 matchs de la copie, hors
  chronomètre : la ligne mesure donc l'import de 50 matchs neufs dans une base de 33 k.
- `analyze` : 20 positions d'un match perdent leur analyse sur la copie, puis
  `analyze --match` (2-ply, 16 tâches) les remplit (19 évaluées, 1 refusée).
- Machine : 16 CPU, chargée par d'autres sessions. `/proc/loadavg` avant
  `4.23 8.93 12.53`, après `3.86 6.29 9.34`. Durée du balayage : 12 min 49 s.
- Toutes les opérations sortent en code 0.

## 2. Tableau complet

Durée en secondes ; « par ligne » quand la sortie a plus d'une ligne, sinon « par appel ».
Les CLI sont des processus complets (ouverture de la base comprise, ~0,1 s plancher).

| # | Operation | Duration (s) | Output lines | Per line / per call | Peak RSS (MiB) | Runs | Exit |
|---:|---|---:|---:|---:|---:|---:|---:|
| 1 | `export --type database (whole base)` | 188.82 | 4 | 47205250.0 µs/line | 476 | 1 | 0 |
| 2 | `list --type analyses --format csv --limit 0` | 97.58 | 312545 | 312.2 µs/line | 943 | 1 | 0 |
| 3 | `list --type moves --format csv --limit 0` | 85.64 | 332286 | 257.7 µs/line | 174 | 1 | 0 |
| 4 | `list --type positions --format csv --limit 0` | 69.97 | 312587 | 223.9 µs/line | 937 | 1 | 0 |
| 5 | `call stats.compute (first, fills the match stats cache)` | 23.48 | 1 | 23485 ms/call | 453 | 1 | 0 |
| 6 | `info (first open of the copy, derived caches)` | 22.68 | 18 | 1260277.8 µs/line | 293 | 1 | 0 |
| 7 | `import --type batch (50 files)` | 9.48 | 128 | 74078.1 µs/line | 491 | 1 | 0 |
| 8 | `search --decision checker --limit 0 --format json (wide)` | 7.64 | 3090712 | 2.5 µs/line | 1395 | 3 | 0 |
| 9 | `list --type study --limit 0` | 6.85 | 17 | 402882.4 µs/line | 281 | 3 | 0 |
| 10 | `search --query 'E>100' --limit 0` | 4.00 | 7758 | 515.3 µs/line | 352 | 3 | 0 |
| 11 | `stats effect --player <p>` | 3.48 | 10 | 347900.0 µs/line | 324 | 3 | 0 |
| 12 | `list --type stats` | 3.01 | 619 | 4862.7 µs/line | 281 | 3 | 0 |
| 13 | `call stats.compute (dashboard, no filter)` | 2.97 | 1 | 2967 ms/call | 254 | 3 | 0 |
| 14 | `list --type players --limit 0` | 2.75 | 7757 | 354.0 µs/line | 344 | 3 | 0 |
| 15 | `call stats.playerTable` | 2.69 | 1 | 2691 ms/call | 351 | 3 | 0 |
| 16 | `call stats.tournamentBadges` | 2.65 | 1 | 2653 ms/call | 356 | 3 | 0 |
| 17 | `call stats.recurringErrors` | 2.56 | 1 | 2563 ms/call | 396 | 3 | 0 |
| 18 | `call stats.ranking` | 2.44 | 1 | 2437 ms/call | 351 | 3 | 0 |
| 19 | `stats ranking` | 2.36 | 54 | 43685.2 µs/line | 342 | 3 | 0 |
| 20 | `stats contrast --player <p> --opponent <o>` | 2.05 | 5 | 409800.0 µs/line | 321 | 3 | 0 |
| 21 | `call search.count E>100 (status bar blunders)` | 1.85 | 1 | 1854 ms/call | 313 | 3 | 0 |
| 22 | `collection suggest --player <p>` | 1.85 | 14 | 132142.9 µs/line | 322 | 3 | 0 |
| 23 | `study queue --limit 0` | 1.78 | 51 | 34960.8 µs/line | 318 | 3 | 0 |
| 24 | `cubematrix <xgid> (2-ply)` | 1.71 | 12 | 142583.3 µs/line | 120 | 3 | 0 |
| 25 | `call search.ids E>100 (GUI search)` | 1.64 | 1 | 1641 ms/call | 317 | 3 | 0 |
| 26 | `stats tournaments --player <p>` | 1.58 | 73 | 21589.0 µs/line | 366 | 3 | 0 |
| 27 | `list --type tournaments --limit 0` | 1.43 | 25093 | 56.8 µs/line | 352 | 3 | 0 |
| 28 | `list --type stats --metric mwc --player <p>` | 1.43 | 237 | 6012.7 µs/line | 212 | 3 | 0 |
| 29 | `verify` | 1.36 | 22 | 61681.8 µs/line | 295 | 3 | 0 |
| 30 | `stats biases --player <p>` | 1.27 | 100 | 12710.0 µs/line | 321 | 3 | 0 |
| 31 | `stats recurring --player <p>` | 1.23 | 35 | 35057.1 µs/line | 322 | 3 | 0 |
| 32 | `call search.query 's cube E>100'` | 1.12 | 0 | 1119 ms/call | 167 | 3 | 0 |
| 33 | `stats plan --player <p>` | 1.11 | 40 | 27825.0 µs/line | 322 | 3 | 0 |
| 34 | `call stats.compute (dashboard, player)` | 1.02 | 1 | 1020 ms/call | 214 | 3 | 0 |
| 35 | `search --game-type race --limit 0 --format xgid` | 0.93 | 39408 | 23.7 µs/line | 421 | 3 | 0 |
| 36 | `stats breakdown --player <p>` | 0.91 | 194 | 4706.2 µs/line | 210 | 3 | 0 |
| 37 | `stats report --html --player <p>` | 0.90 | 0 | 901 ms/call | 210 | 3 | 0 |
| 38 | `stats progression --player <p>` | 0.90 | 685 | 1315.3 µs/line | 212 | 3 | 0 |
| 39 | `list --type matches --limit 0` | 0.88 | 290664 | 3.0 µs/line | 164 | 3 | 0 |
| 40 | `analyze --match <m> (20 positions, 2-ply)` | 0.88 | 23 | 38130.4 µs/line | 173 | 1 | 0 |
| 41 | `call matches.list (all, NDJSON)` | 0.79 | 33319 | 23.8 µs/line | 90 | 3 | 0 |
| 42 | `list --type timeerrors` | 0.64 | 1 | 640 ms/call | 102 | 3 | 0 |
| 43 | `search --player <p> --error-min 0.05 --limit 0` | 0.63 | 5473 | 115.8 µs/line | 184 | 3 | 0 |
| 44 | `call stats.matchBadges (page of 500)` | 0.50 | 1 | 503 ms/call | 99 | 3 | 0 |
| 45 | `stats engines` | 0.48 | 1 | 482 ms/call | 281 | 3 | 0 |
| 46 | `collection show --id <c> (500)` | 0.37 | 505 | 738.6 µs/line | 171 | 3 | 0 |
| 47 | `call tournaments.list (all)` | 0.37 | 5011 | 74.4 µs/line | 74 | 3 | 0 |
| 48 | `call positions.listIds (all)` | 0.36 | 1 | 356 ms/call | 147 | 3 | 0 |
| 49 | `search --decision cube --pip-min 80 --pip-max 120 --limit 0 --format xgid` | 0.31 | 5057 | 61.7 µs/line | 285 | 3 | 0 |
| 50 | `search --dice 6,5 --score1 2 --score2 4 --limit 0` | 0.29 | 90 | 3200.0 µs/line | 134 | 3 | 0 |
| 51 | `call tournaments.matches <t>` | 0.29 | 2971 | 96.6 µs/line | 72 | 3 | 0 |
| 52 | `anki export --deck <d> (.apkg)` | 0.28 | 1 | 281 ms/call | 85 | 3 | 0 |
| 53 | `list --type matches --query <player> --limit 0` | 0.27 | 5775 | 46.6 µs/line | 141 | 3 | 0 |
| 54 | `search --decision cube --limit 50` | 0.26 | 54 | 4870.4 µs/line | 62 | 3 | 0 |
| 55 | `stats tournament --id <t> --player <p>` | 0.25 | 74 | 3351.4 µs/line | 96 | 3 | 0 |
| 56 | `call search.ids cube (GUI search)` | 0.25 | 1 | 246 ms/call | 143 | 3 | 0 |
| 57 | `openingbook question --roll first` | 0.23 | 3 | 78000.0 µs/line | 51 | 3 | 0 |
| 58 | `openingbook status` | 0.23 | 1 | 232 ms/call | 54 | 3 | 0 |
| 59 | `call stats.playerNames` | 0.22 | 1 | 223 ms/call | 80 | 3 | 0 |
| 60 | `events alias suggest` | 0.20 | 766 | 258.5 µs/line | 73 | 3 | 0 |
| 61 | `players alias suggest` | 0.20 | 1061 | 184.7 µs/line | 78 | 3 | 0 |
| 62 | `stats windows --player <p>` | 0.19 | 102 | 1862.7 µs/line | 71 | 3 | 0 |
| 63 | `call matches.byPosition <p> (provenance)` | 0.19 | 1 | 188 ms/call | 61 | 3 | 0 |
| 64 | `stats training --player <p>` | 0.18 | 165 | 1097.0 µs/line | 73 | 3 | 0 |
| 65 | `call positions.load <p>` | 0.18 | 1 | 181 ms/call | 59 | 3 | 0 |
| 66 | `maintenance status` | 0.18 | 7 | 25571.4 µs/line | 70 | 3 | 0 |
| 67 | `call positions.list page 100 @1000` | 0.17 | 100 | 1740.0 µs/line | 60 | 3 | 0 |
| 68 | `call metadata.countsEstimate (status bar)` | 0.17 | 1 | 174 ms/call | 65 | 3 | 0 |
| 69 | `call analyses.load <p>` | 0.17 | 1 | 172 ms/call | 61 | 3 | 0 |
| 70 | `info` | 0.16 | 18 | 8944.4 µs/line | 210 | 3 | 0 |
| 71 | `call matches.movesByMatch <m>` | 0.16 | 39 | 4076.9 µs/line | 65 | 3 | 0 |
| 72 | `call collections.positions <c> page 50` | 0.16 | 50 | 3120.0 µs/line | 69 | 3 | 0 |
| 73 | `call lessons.get <l>` | 0.15 | 1 | 154 ms/call | 62 | 3 | 0 |
| 74 | `call anki.nextCard <d>` | 0.15 | 1 | 154 ms/call | 64 | 3 | 0 |
| 75 | `call anki.listDecks` | 0.15 | 1 | 154 ms/call | 60 | 3 | 0 |
| 76 | `call matches.origin <m>` | 0.15 | 1 | 153 ms/call | 60 | 3 | 0 |
| 77 | `call history.load` | 0.15 | 1 | 152 ms/call | 63 | 3 | 0 |
| 78 | `call collections.list` | 0.15 | 1 | 152 ms/call | 61 | 3 | 0 |
| 79 | `call search.find cube limit 50` | 0.15 | 50 | 3000.0 µs/line | 63 | 3 | 0 |
| 80 | `call lessons.list` | 0.15 | 1 | 149 ms/call | 64 | 3 | 0 |
| 81 | `list --type tags` | 0.14 | 8 | 18125.0 µs/line | 59 | 3 | 0 |
| 82 | `lesson show --id <l>` | 0.14 | 5 | 27000.0 µs/line | 59 | 3 | 0 |
| 83 | `lesson list` | 0.13 | 2 | 66000.0 µs/line | 58 | 3 | 0 |
| 84 | `training sessions` | 0.13 | 1 | 127 ms/call | 61 | 3 | 0 |
| 85 | `list --type imports` | 0.12 | 7 | 17857.1 µs/line | 57 | 3 | 0 |
| 86 | `tournament list` | 0.12 | 1 | 122 ms/call | 59 | 3 | 0 |
| 87 | `info --estimate` | 0.12 | 18 | 6777.8 µs/line | 61 | 3 | 0 |
| 88 | `match --id <m> --format json` | 0.12 | 6509 | 18.4 µs/line | 62 | 3 | 0 |
| 89 | `list --type positions --limit 100` | 0.12 | 503 | 238.6 µs/line | 115 | 3 | 0 |
| 90 | `anki retention --deck <d>` | 0.12 | 3 | 39666.7 µs/line | 58 | 3 | 0 |
| 91 | `players alias list` | 0.12 | 1 | 115 ms/call | 59 | 3 | 0 |
| 92 | `training missed` | 0.11 | 1 | 114 ms/call | 58 | 3 | 0 |
| 93 | `events alias list` | 0.11 | 1 | 113 ms/call | 60 | 3 | 0 |
| 94 | `anki stats --deck <d>` | 0.11 | 10 | 11300.0 µs/line | 57 | 3 | 0 |
| 95 | `collection list` | 0.11 | 5 | 22400.0 µs/line | 59 | 3 | 0 |
| 96 | `anki decks` | 0.11 | 5 | 22400.0 µs/line | 57 | 3 | 0 |
| 97 | `anki forecast --deck <d> --days 30` | 0.11 | 34 | 3264.7 µs/line | 60 | 3 | 0 |
| 98 | `epc <bearoff xgid>` | 0.02 | 5 | 3400.0 µs/line | 27 | 3 | 0 |
| 99 | `bearoff list` | 0.01 | 19 | 736.8 µs/line | 24 | 3 | 0 |

Confirmation : `list --type moves --format csv` met **85,6 s** pour 332 286 lignes
(258 µs/ligne), 162 s pendant un passage concurrent d'un profil : l'ordre de grandeur annoncé
(80 s) tient.

## 3. Profils

Six opérations retenues sur le coût par ligne ou par appel (l'export entier en plus, par sa
durée brute et parce qu'il cache un N+1 neuf). Profils CPU pris par un test Go temporaire
(non versionné) appelant `CLI.Run` ou `server.RunCall` avec `-test.cpuprofile`, sur une
copie déjà ouverte de la base ; `go tool pprof -top`.

### 3.1 `list --type moves --format csv` — 85,6 s, 258 µs/ligne

- `exportMovesCSV` (`internal/cli/cli_list_export.go:109`) : `GetAllMatches`, puis
  `GetGamesByMatch` par match (33 k) et **`GetMovesByGame` par partie (268 k requêtes)**,
  `database/db_match.go:214` : 93 % du temps.
- Dans ces requêtes, **la préparation SQL fait 75 %** (`_sqlite3RunParser` 74 %,
  `_yy_reduce` 62 %), l'exécution 45 % (les deux se recouvrent par `sqlite3Reprepare`,
  38 %). `ActionLabelSQL` (`sqlshared/action_label.go:54`), reconstruit à chaque appel :
  6 %.
- Hypothèse : chaque requête porte deux `CASE` d'une quarantaine de branches (`ActionLabelSQL`) que
  modernc analyse à chaque appel, deux fois quand `sqlite_stat4` fait expirer l'instruction
  au premier `bind`. Le coût est un coût fixe par requête, multiplié par 268 k.
- Gain estimé : une seule requête jointe `move ⋈ game ⋈ match` lue en flux :
  85 s → ~2–4 s (×20 ou plus).
- Effort : **S**. Fichiers : `internal/cli/cli_list_export.go` ; une méthode de lecture en
  flux sur `Database` ou sur `MatchStore` (`database/db_match.go`, zone d'un autre agent :
  à coordonner, ou passer par `storage/sqlite` hors `matches_*.go`).
- **Neuf.** optim §4.5 et A1 ne visaient que `search` ; seul le coût d'`ActionLabelSQL`
  (3,3 s là) recoupe.

### 3.2 `list --type positions|analyses --format csv` — 70 s / 98 s, 224 / 312 µs/ligne, RSS 940 Mo

- `exportPositionsCSV` (`cli_list_export.go:81`) et `exportAnalysesCSV` : `LoadAllPositions`
  en entier (d'où 940 Mo), puis **`LoadAnalysis` par position** (312 k) : 90 % du temps.
- Dans `LoadAnalysis` (`database/db_analysis.go`) : la requête des coups joués
  (l. 254, avec `ActionLabelSQL`) 52 %, dont la préparation SQL 48 % et `Reprepare` 21 % ;
  la lecture de l'analyse (l. 242, `analysisStore.Load`) 29 %, dont le décodage du format
  hérité 19 % (`engine/analysiscodec.go:357`, `json.Unmarshal`).
- Hypothèse : même N+1 que 3.1 (coût fixe de préparation par requête), et la base entière
  tenue en mémoire au lieu d'un flux.
- Gain estimé : chargement par lot (`analyses.loadByIds` / `DecodeAnalysesConcurrently`
  existent) par paquets de quelques milliers, coups joués par une requête groupée :
  70–98 s → ~5–8 s ; RSS 940 → ~150 Mo en flux.
- Effort : **S**. Fichiers : `internal/cli/cli_list_export.go`,
  `pkg/blunderdb/database/db_analysis.go`.
- **Neuf** pour le CSV. Même remède que optim A1 (livré pour `search`) ; même motif que
  l'item « Export `.apkg` : analyses une par position » de `BACKLOG.md`.

**Après les exports en flux (#739), puis le recouvrement des pages.** `EachPositionAnalysis`
lit par pages de 2 000 (positions, `LoadMany`, `MovesByPositions`) : plus de N+1, RSS bornée.
Profil sur la même base (copie, 312 587 / 312 545 lignes) : le temps restait partagé, en
série, entre la lecture d'une page (~6 s de fil principal : `positions.List` 2,3 s,
`MovesByPositions` 2,7 s, `LoadMany` 1,4 s plus l'attente du décodage parallèle) et le
calcul des colonnes par l'appelant (~3,7 s, dont `PlayedCandidate` 1,5 s et
`CountLegalPlays` 1,3 s). Le CPU total (~37 s) est dominé par le décodage du format hérité
(`json.Unmarshal` 22 s, zstd 4,8 s), déjà réparti sur tous les cœurs.

Remède livré (`database/db_library_walk.go`, sortie inchangée, md5 identiques ordre
compris) : la page suivante est lue pendant que l'appelant consomme la courante (deux pages
au plus en mémoire), les coups joués d'une page sont lus pendant que ses analyses décodent,
et la tranche des positions est allouée à sa taille (630 Mo d'allocations en moins).
Médiane de 3, alternées, machine chargée (load 7–11) :

| Export CSV | avant | après |
|---|---|---|
| `list --type positions` | 12,1 s | 6,7 s |
| `list --type analyses` | 12,6 s | 8,2 s |

Reste : sur cette base, le décodage du format hérité (`reencode` le supprime) ; le calcul
des colonnes (coup joué, coups légaux) borne désormais le chemin critique.

### 3.3 Première lecture de statistiques (remplissage de `match_stats`) — 23,5 s par appel

- `StatsStore.Compute` → `matchStatsReadable` → `FillMatchStats`
  (`sqlshared/match_stats.go:443`) → `refreshMatchStatsBatch` (l. 327) : 85 %.
  `computeMatchStatsRows` 27 %, `writeMatchStatsCells` (l. 358) 46 %, les `INSERT INTO
  match_stats` ligne à ligne (l. 349, 66 638 lignes, chacune reparsée) ~31 % en `Exec`.
- Hypothèse : le cache est vide après le pas de migration 2.39 (et après tout rescoring) ;
  il est rempli d'un bloc, ligne à ligne, sans instruction préparée. L'utilisateur le paie
  au premier affichage du tableau de bord ou de `list --type stats`.
- Gain estimé : `INSERT … SELECT` ou instruction préparée réutilisée, cellules écrites en
  lot : 23 s → ~8–12 s (×2). Le régime établi (3.5) n'en profite pas.
- Effort : **M** (les deux backends ; parité des cellules). Fichiers :
  `sqlshared/match_stats.go`, `sqlshared/match_stats_cells.go`.
- **Partiellement connu** : les cellules et leur reclustering sont mesurés dans
  `tasks/mesure-bmab-0.37.md` ; le coût du remplissage d'une base de 33 k matchs ne l'est pas.
- **Fait (#742)**, après #724 et #732. Remesure sur une copie de `sample.db` migrée en 2.48.0,
  premier `call stats.compute` (66 638 lignes `match_stats`, 605 383 cellules) : **20,9 s**
  sur machine calme (deux passes, 20,8 et 21,1 s). Profil du remplissage : lectures
  (`scanEach`, `computeMatchStatsRows` et balayage des cellules) 55 %, écritures 40 %, dont
  14 % de compilation d'instructions (`sqlite3_prepare_v2`) — l'`INSERT` de 200 lignes des
  cellules, recompilé à chaque lot, et les 66 638 `INSERT INTO match_stats` d'une ligne.
  Changement : `match_stats` passe par `insertRows` (200 lignes par instruction) ;
  l'instruction pleine d'`insertRows` est préparée une fois par transaction
  (`sqlshared.Preparer`, implémenté par SQLite ; pgx garde déjà un cache d'instructions par
  connexion, PostgreSQL gagne surtout les 66 638 allers-retours devenus 334). Après : la
  compilation tombe de 14 % à 4 % du profil, le chemin d'écriture de −33 %, le remplissage de
  **−12 % environ** (profils CPU comparés à charge égale ; machine chargée à 30+ pendant les
  chronos, d'où l'absence de mesure murale fiable de l'après). Résultat identique au
  bit près (`stats.compute` comparé). Le ×2 estimé n'est pas atteint : il supposait des
  écritures majoritaires, ce sont les lectures qui dominent désormais — piste suivante, la
  `computeMatchStatsRows` (quatre lectures de `statsBaseJoin`) et le balayage des cellules (une
  cinquième), à fusionner en un passage par lot.

### 3.4 `list --type study` — 6,85 s pour 17 lignes

- `listStudy` → `StudyImpact` (`database/db_study_impact.go:57` et `:61`) →
  `statsByGameType` (l. 112) : **deux `ComputeStats` complets** (avant / après la date
  pivot) pour n'en garder que `PerGameType` : 91 %, tout dans `computeFromCells`.
- Hypothèse : le calcul entier du tableau de bord (ventilations, séries, tournois) est
  refait deux fois pour une seule ventilation par plan de jeu.
- Gain estimé : une lecture des seules cellules par plan de jeu sur les deux fenêtres :
  6,9 s → < 1 s.
- Effort : **S-M** (garder « le même calcul que l'onglet Stats » : extraire la ventilation
  plutôt que la recalculer à part). Fichiers : `database/db_study_impact.go`,
  `sqlshared/stats_from_cells.go`.
- **Neuf.** `stats effect` (3,5 s) a vraisemblablement le même motif, non profilé.

### 3.5 `stats.compute` en régime établi (tableau de bord) — 2,97 s par appel

- `computeFromCells` (`sqlshared/stats_from_cells.go:116`) 74 % ;
  `breakdownsFromCells` (l. 447 et 532) 38 % ; `readPhaseCells` 11 % ;
  `perTournamentFromTable` 8 % ; `rollingFromCells` 7 %. Le temps est dans le pas SQLite
  (`_sqlite3VdbeExec` 65 %) : balayage des 605 k cellules à chaque appel.
- Hypothèse : à 33 k matchs, les cellules par match restent proportionnelles à la base ;
  chaque ventilation relit et regroupe toutes les cellules. 1,04 s à 971 matchs (optim §2),
  3 s ici : linéaire en nombre de matchs.
- Gain estimé : cellules pré-agrégées sur la bibliothèque entière pour l'appel sans filtre
  (le cas du premier affichage) : 3 s → ~0,3 s ; avec filtre, rien.
- Effort : **L** (invalidation, deux backends, schéma). Fichiers : `sqlshared/stats_from_cells.go`,
  `sqlshared/match_stats.go`, schéma.
- **Connu** : famille de optim B2 et de l'item « Badges » de `BACKLOG.md`
  (`stats.playerTable` 2,7 s, `stats.tournamentBadges` 2,65 s, `stats.ranking` 2,4 s ont le
  même profil d'agrégat complet).

### 3.6 `export --type database` — 189 s (0,6 ms/position), fichier de 225 Mo

- `writePositions` (`ingest/export_sqlite.go:606`) 37 % : décodage du format hérité 13 %
  (`DecodeAnalysesConcurrently`), réencodage binaire 9 % (`SaveExportedAnalysis`,
  `storage/sqlite/analyses_sqlite.go:316,339`), `positionStore.SaveCreated` 8 %.
- **`writeTournaments` (l. 1121) 17 %, 34 s** : `tournamentStore.Get` par tournoi
  (5 011), `storage/sqlite/tournaments_sqlite.go:67`. Sa jointure
  `TournamentMatchAgg` (`sqlshared/tournament_date.go:31`) matérialise un agrégat sur
  `match` sans index sur `match.tournament_id` : `EXPLAIN QUERY PLAN` = `SCAN tm`, un
  balayage des 33 k matchs par tournoi.
- `writeMatches` (l. 1296) 12,5 % : `MovesByMatch` et `PlayScorer.Score`
  (`sqlshared/stats_move_grades.go:134`).
- Hypothèse : le N+1 des tournois est quadratique (tournois × matchs) et invisible à 60
  tournois ; le reste est le coût du format hérité de cette base.
- Gain estimé : lire les tournois par un seul `List`, ou un index `match(tournament_id)` :
  −34 s (−18 %). Un `reencode` préalable de la base retirerait la plus grande part des 13 %
  de décodage.
- Effort : **S** pour `writeTournaments` par `List` (`ingest/export_sqlite.go`) ; **M** pour
  l'index (schéma, `DatabaseVersion`, zone d'un autre agent).
- **Neuf** pour les tournois. Le reste recoupe optim §4.4, A5 et B1 (B1 livré : 225 Mo au
  lieu de ~1 Go).

## 4. Hors des six, à signaler

- **Bug, neuf** : `GetAllMatches` (`database/db_match.go:33`) passe tous les ids à
  `MatchBadges` (`sqlshared/stats.go:852`) ; au-delà de 32 766 matchs SQLite refuse
  (`too many SQL variables`), l'erreur n'est qu'un `slog.Warn` et **les badges manquent en
  silence** dans `list --type matches` et chez les autres appelants de `GetAllMatches`. Zone d'un autre agent :
  à lui signaler, pas corrigé ici.
- **Première ouverture** : 22,7 s (18,8 s isolée), une fois par base dont
  `played_decisions_pending` est levé : `RescoreAllPlayedDecisions`
  (`sqlshared/played_decisions.go:321`) 78 %, dont 52 % de décodage JSON du format hérité.
  Coût unique de migration, neuf mais secondaire.
- **Cause transversale** des N+1 (3.1, 3.2) : environ 0,25 ms par requête, surtout en
  préparation SQL sous modernc. Tout chemin qui boucle sur des requêtes unitaires
  (`apkg`, CSV, export des tournois) paie ce plancher ; mettre `ActionLabelSQL` en
  constante et préparer une fois ne suffit pas si `stat4` fait expirer l'instruction à
  chaque `bind` (hypothèse à vérifier avec `sqlite3_stmt_status(…REPREPARE)`).
- `search --query 'E>100'` : 4 s, 515 µs/ligne ; `search.ids E>100` 1,6 s,
  `search.count E>100` (barre d'état) 1,85 s : optim A2/B3, connu.
- `search --decision checker --limit 0 --format json` : 7,6 s, mais **RSS 1,4 Go** ; optim A3,
  connu.
- Les appels unitaires de la GUI (chargement d'une position, analyse, provenance, collections,
  paquets, leçons, historique) restent au plancher d'un processus `call` (~0,15 s).

## 5. Ordre suggéré

3.1 et 3.2 (S, ×10 à ×20, CLI seule), le `List` de 3.6 (S, −34 s), 3.4 (S-M), puis le bug
des badges au-delà de 32 766 matchs, avant 3.3 (M) et 3.5 (L, avec B2).
