# blunderDB — test de charge du démon `serve` (2026-10-21)

Mesure seule, aucun code produit modifié. Outil : `scripts/bench-daemon-load.sh <base.db>`, qui
pilote le scénario `library` de `cmd/blunderdb-loadtest` (ajouté pour ce test : il tire ses
identifiants de la base servie). Programme : `tasks/optimisation-2026-10.md` ; format :
`tasks/balayage-perf-2026-10.md`.

## 1. Conditions

- Base : copie de `bench-scale/mesure-0.37/sg/sample.db` (270 Mo, 312 586 positions,
  33 319 matchs). Le script copie la base, l'ouvre une fois par le démon (migration : 0,1-0,2 s,
  `cmd/serve` ne rejoue pas le rescorage du bureau), garde ce gabarit « froid », puis lance un
  `stats.compute` qui remplit `match_stats` (**18,9 s / 21,7 s**) : gabarit « chaud ». Chaque
  niveau tourne sur sa propre copie d'un gabarit, démon neuf, `--rate-limit-rps 0`.
- Mélange `library` (poids) : `positions.load` 34, `analyses.load` 12, `search.find` 12
  (E>100, cube, E>50 pions ; 50 résultats), `search.count E>100` 5 (barre d'état),
  `matches.list` 12 (page de 100, à un décalage ou filtrée par joueur), `stats.compute` par
  joueur 12 et sans filtre 3, `comments.add` 7, `analyses.save` 3 (réécrit l'analyse lue).
  Boucle fermée, N clients, 30 s, montée 2 s, délai client 30 s, gzip accepté (client Go).
- Mesures : rapport par opération (p50/p95/p99/max, erreurs par code), RSS du démon toutes les
  0,5 s (`VmRSS`, séparée en `RssAnon` / `RssFile`), vidage des goroutines à mi-course
  (`--pprof-addr`), profil d'allocations en fin de course.
- Machine : 16 CPU, partagée. Deux passes complètes : passe 1 loadavg ~9, passe 2 ~5-6. Les
  écarts entre passes (jusqu'à ×2 sur les lourdes) sont ceux de la machine ; les classements
  sont stables.
- PostgreSQL : non mesuré. `blunderdb migrate` vers un PostgreSQL 16 local n'avait pas fini les
  positions après 58 min (§3.4, #753) ; arrêté.

## 2. Résultats

### 2.1 Global

| Passe | Run | Clients | RPS | Erreurs | p50 (ms) | p95 (ms) | p99 (ms) | RSS crête, Mio (anon, fichier) |
|---|---|---:|---:|---:|---:|---:|---:|---|
| 1 | chaud | 10 | 16 | 0 | 4,5 | 3 783 | 4 828 | 1 661 |
| 1 | chaud | 50 | 16 | 4 (timeout) | 549 | 14 588 | 23 918 | 2 730 |
| 1 | chaud | 200 | 19 | 71 (timeout) | 4 308 | 30 001 | 30 001 | 2 807 |
| 1 | froid | 50 | 27 | 147 (500) | 206 | 9 514 | 24 794 | 2 385 |
| 2 | chaud | 10 | 8 | 0 | 7,1 | 6 926 | 9 061 | 2 102 (657, 1 444) |
| 2 | chaud | 50 | 12 | 4 (timeout) | 1 087 | 17 888 | 29 524 | 2 517 (421, 2 096) |
| 2 | chaud | 200 | 14 | 76 (timeout) | 7 017 | 30 001 | 30 003 | 2 858 (683, 2 174) |
| 2 | froid | 50 | 16 | 128 (500, timeout) | 253 | 11 481 | 26 759 | 785 (191, 593) |

Le débit plafonne à 8-19 req/s quel que soit N : au-delà de 10 clients, la latence monte,
pas le débit.

### 2.2 Par opération (passe 1 ; passe 2 entre parenthèses pour p50)

| Opération | 10 : p50 / p99 (ms) | 50 : p50 / p99 | 200 : p50 / p99 | Erreurs 50 / 200 |
|---|---|---|---|---|
| `positions.load` | 0,6 (1,1) / 9 | 131 (312) / 3 094 | 1 961 (4 411) / 14 161 | 0 / 0 |
| `analyses.load` | 0,8 (1,9) / 8 | 159 (380) / 3 266 | 2 204 (3 531) / 10 453 | 0 / 0 |
| `comments.add` | 0,7 (1,5) / 9 | 160 (29) / 2 031 | 2 445 (2 887) / 14 533 | 0 / 0 |
| `analyses.save` | 3,8 (12) / 10 | 539 (1 191) / 3 431 | 7 907 (10 815) / 19 360 | 0 / 0-1 |
| `matches.list` (100) | 332 (564) / 476 | 738 (826) / 4 337 | 3 816 (6 046) / 21 102 | 0 / 0 |
| `search.find` (50) | 134 (213) / 1 304 | 9 956 (10 346) / 23 918 | 24 221 (25 159) / 30 004 | 0-1 / 40-43 |
| `search.count E>100` | 3 876 (6 660) / 4 526 | 13 835 (15 416) / 30 001 | 30 000 / 30 001 | 3-4 / 17 |
| `stats.compute` joueur | 1 436 (2 513) / 1 726 | 3 030 (4 071) / 10 942 | 18 655 (21 662) / 30 002 | 0 / 8-13 |
| `stats.compute` sans filtre | 4 828 (8 634) / 5 127 | 6 264 (9 258) / 13 935 | 20 269 (22 585) / 30 001 | 0 / 3-5 |

Toutes les erreurs à chaud sont des délais client de 30 s ; aucun 5xx, aucun `SQLITE_BUSY`.

### 2.3 Base froide (`match_stats` vide), 50 clients

| Opération | Passe 1 | Passe 2 |
|---|---|---|
| `stats.compute` (joueur + sans filtre) | **139 / 139 en 500** | **115 / 116 en 500** |
| `analyses.save` | 8 / 25 en 500 | 9 / 23 en 500 |
| `search.count` | 0 erreur, p50 18 s | 4 timeouts, p50 22 s |

Cause, reproduite à part (4 `stats.compute` simultanés sur la copie froide) : 3 réponses
`500 internal` en 76 ms, `match stats delete: database is locked (5) (SQLITE_BUSY)` ; la
quatrième répond après ~20 s. Le débit plus élevé du froid (16-27 req/s) vient de ces échecs
rapides.

### 2.4 Verrous et attente

- **`Database.mu` n'est pas sur le chemin du démon** : `internal/server` n'importe le paquet
  `database` que pour l'enregistrement de la chaîne de migration (`cmd/serve/main.go`) ; les
  requêtes passent par `storage.Storage` (pool `database/sql`, une transaction par opération).
  La question « les lectures se bloquent-elles entre elles sous `Database.mu` ? » ne se pose
  qu'au bureau et en CLI.
- **Le pool de 10 connexions est le goulot** (`sqlite.ConfigurePool`). Goroutines à mi-course :
  10 clients, ~0-2 en attente de connexion ; 50 clients, 34-37 dans `database/sql.(*DB).conn` ;
  200 clients, **~170**, dont 26-30 dans `shared.RefusesWrites` (une connexion prise pour lire
  `query_only` avant chaque lecture de statistiques), 12-28 dans `positionStore.Load`. Les 10
  connexions sont tenues par `scanEach`/`scanRows` (statistiques, recherche).
- **Remplissage de fond de `match_stats`** : le démon n'en a pas. `FillMatchStatsInBackground`
  est du bureau (`database/match_stats_fill.go`) ; dans le démon, chaque lecteur remplit ce qui
  manque, de façon synchrone, sans exclusion entre lecteurs. Il ne cède donc pas : il se bat
  (§3.1).
- Écritures à chaud : `comments.add`/`analyses.save` ne voient aucun `SQLITE_BUSY` ; leur
  latence est l'attente d'une connexion, pas du verrou d'écriture (`busy_timeout` 10 s non
  atteint).

### 2.5 Mémoire

- `VmRSS` 1,7-2,9 Gio, mais **`RssFile` en fait 1,4-2,2 Gio** : chaque connexion projette la
  base (`mmap_size` 256 Mio) et les pages partagées comptent une fois par projection. Mémoire
  propre (`RssAnon`) : 420-680 Mio à chaud, 190 Mio sur la base froide (statistiques en échec).
- Allocations cumulées sur 30 s : 4,7 Gio à 10 clients, 8,0 Gio à 200 ;
  `readPhaseCells` 26-30 %, `foldCells` 13-14 %, `perMatchFromTable` 9-10 %,
  `SearchStore.scanRows` 6 %, `compress/flate.NewWriter` 3-4 % (un compresseur neuf par
  réponse gzip).

## 3. Constats classés

### 3.1 Base froide : `stats.compute` en `SQLITE_BUSY` (500), remplissage jamais achevé sous charge — #750

- Constat : §2.3. `refreshMatchStatsBatch` (`sqlshared/match_stats.go`) calcule puis
  `DELETE`/`INSERT` dans une transaction différée (`withTx` → `BeginTx(ctx, nil)`) : quand un
  autre remplissage a validé entre-temps, la montée en écriture échoue sans attendre
  `busy_timeout`. N lecteurs refont N remplissages complets ; sous charge continue, aucun ne
  finit. Le code `internal` ne dit pas au client qu'il peut réessayer.
- Gain estimé : statistiques disponibles après un remplissage (~20 s) au lieu de jamais ;
  zéro 500.
- Effort : **S-M** (singleflight par portée, `BEGIN IMMEDIATE` côté SQLite, contention →
  503 `unavailable`).
- **Neuf.** Touche toute base servie juste après une migration qui vide `match_stats`.

### 3.2 Lectures unitaires en file derrière les lourdes dans le pool de 10 — #752

- Constat : §2.2 et §2.4. `positions.load` passe de 0,6 ms (p50, 10 clients) à 131-312 ms
  à 50 et 2,0-4,4 s à 200 ; p99 3-6 s à 50. Débit plafonné à 8-19 req/s ; 9-11 % de délais
  dépassés à 200, tous sur les routes lourdes.
- Hypothèse : chaque requête lourde garde une connexion 1-9 s ; dix suffisent à bloquer le
  pool. Ce n'est pas le verrou d'écriture (aucun `BUSY` à chaud).
- Gain estimé : `positions.load` p99 < 50 ms à 50 clients (×60), plus de timeouts sur les
  unitaires à 200 ; le débit des lourdes reste borné par le CPU.
- Effort : **M** pour une admission par classe de route (lourdes bornées à ~NumCPU/2, au-delà
  503 + `Retry-After`) ; **S** pour un pool à `max(10, NumCPU)` (à mesurer : RSS par
  connexion) et pour `RefusesWrites` lu une fois à l'ouverture.
- **Neuf** côté démon. La lenteur unitaire des lourdes est connue (§3.3).

### 3.3 `search.count E>100` et `stats.compute` : les lourdes elles-mêmes

- `search.count E>100` (barre d'état de chaque client) : 3,9-6,7 s à 10 clients (1,85 s seul
  en CLI), la première à dépasser 30 s dès 50. `stats.compute` sans filtre 4,8-8,6 s.
- **Connu** : optim A2/B3 (compte E>), balayage §3.5 (agrégat des cellules). Pas d'issue
  nouvelle ; sous charge, un compte par filtre mis en cache jusqu'à la prochaine écriture
  retirerait 5 % des requêtes et la plus longue d'entre elles.

### 3.4 `migrate` vers PostgreSQL : > 1 h pour 312 k positions — #753

- Constat : phase des positions inachevée après 58 min (`position` 82 Mo, ~20 ko/s, 8 % d'un
  CPU). `copyPositions` enchaîne par position `Positions().Save`, `Analyses().Save` (avec
  `requireOwned`, un `SELECT COUNT(*) … = ANY` par ligne) et `Comments().ByPosition`.
- Gain estimé : quelques minutes au lieu de plusieurs heures (copie par lots).
- Effort : **M**. **Neuf.** Il bloque aussi la mesure PostgreSQL de ce test.

### 3.5 Cause d'un 500 perdue derrière gzip — #751

- Constat : `compressWriter` (`middleware/compress.go`) n'implémente pas `SetErr` ;
  `errorBodyFor` perd la cause et la ligne de journal passe de `ERROR … err=` à `INFO` sans
  cause. Au `--log-level warn`, les 147 erreurs 500 de la passe 1 n'ont laissé **aucune**
  ligne. Reproduit avec et sans `curl --compressed`.
- Effort : **S**. **Neuf.** Sans gain de performance, mais sans lui §3.1 serait invisible en
  production.

### 3.6 RSS : surtout la projection de la base

- `RssFile` 1,4-2,2 Gio = ~8-10 projections de 256 Mio, partagées avec le cache de pages ;
  la mémoire propre reste sous 700 Mio. Pas de défaut ; un pool plus grand (§3.2) ajoute
  surtout de la `RssFile`, et jusqu'à 64 Mio de `cache_size` par connexion en `RssAnon`.

## 4. Ordre suggéré

#751 (S, prérequis du diagnostic), #750 (S-M, défaut visible), #752 en commençant par
`RefusesWrites` et la taille du pool (S), puis l'admission par classe (M) ; #753 avant toute
mesure PostgreSQL ; §3.3 reste dans A2/B3.

## 5. Après #750 et #752

Remède : remplissage de `match_stats` au démarrage du démon SQLite et un seul remplissage par
portée à la fois (les lecteurs l'attendent, hors de la porte d'admission) ; transactions SQLite
en `BEGIN IMMEDIATE` par défaut (`Transact`, `withTx`, et `BeginTx` qui prend le verrou par une
écriture vide), sauf connexion en lecture seule ; `TransactRead` (différé) pour `stats.compute`
et `withReadSnapshot`, qui ne tiennent donc plus le verrou d'écriture ; `SQLITE_BUSY` et ses codes
étendus (`&0xff`) → `storage.ErrBusy` → `503 unavailable` + `Retry-After` (le client `remote` ne
le relit que sur `details.reason = "busy"`) ; `RefusesWrites` mémorisé par base ouverte ; pool
SQLite du démon `max(10, GOMAXPROCS)` (16 ici, le bureau reste à 10) ; admission des routes
lourdes (`stats.*`, `search.count/find/ids/query/indexOf`, `matches.list`) à
`max(2, GOMAXPROCS/2)` = 8 places sur SQLite, file de 8 par place, attente 20 s, puis `503` +
`Retry-After` ; `--heavy-slots` règle ce nombre (0 = ce défaut, sans borne sur PostgreSQL ;
-1 = sans borne).

Le démon embarqué par gammonGo (`pkg/blunderdb/server`, PostgreSQL en production) ne reçoit ni
l'admission ni le remplissage au démarrage : son pool suit ses propres réglages, et le premier
lecteur de statistiques d'un tenant remplit les lignes de ce tenant (un remplissage par portée
à la fois).

Même script, `LEVELS="50 200" COLD_N=50`, même machine partagée, avant (`328a5b9e1`) et après
mesurés coup sur coup sur deux copies de `sample.db` (loadavg ~9-11 dans les deux cas).

| `positions.load` | 50 : p50 / p99 (ms) | 200 : p50 / p99 (ms) | froid 50 : p50 / p99 (ms) |
|---|---|---|---|
| avant | 47,6 / 1 853 | 1 016 / 8 610 | 13,0 / 237 |
| après | 0,6 / 7,4 | 14,7 / 98,1 | 0,6 / 9,6 |

| Run | Avant : RPS, erreurs | Après : RPS, erreurs |
|---|---|---|
| chaud, 50 | 23 ; 1 timeout | 23 ; 0 |
| chaud, 200 | 26 ; 69 timeout (`search.count`, `search.find`) | 6 034 ; 114 391 `503` sur les seules routes lourdes |
| froid, 50 | 55 ; 401 `500 internal` (383 `stats.compute`, 18 `analyses.save`) | 25 ; 0 |

- Froid : plus aucun `500` ; le remplissage du démarrage précède les lecteurs, et les
  `stats.compute` répondent comme sur la base chaude (7,9 s p50 pour `all`). Le débit brut
  d'avant comptait des `stats.compute` qui échouaient en < 1,5 s.
- Débit utile des routes lourdes (réponses 200) : 50 clients 11,6 → 11,9 req/s ; froid
  23 → 12,4 req/s (avant, l'essentiel en `search`/`matches.list` pendant que `stats.compute`
  échouait) ; 200 clients 12,8 → 5,6 req/s. À 200 clients en boucle fermée sans attente, la file
  lourde déborde : les lourdes reçoivent `503` en < 3,2 ms (p99) au lieu d'un timeout à 30 s,
  les lectures unitaires passent toutes (aucune erreur), mais le client de charge ne respecte
  pas `Retry-After` et ses ~3 800 refus/s prennent la CPU des lourdes admises. Les lourdes
  elles-mêmes restent §3.3.
- Écueil mesuré en cours de route : `BEGIN IMMEDIATE` sur un `stats.compute` entier (sa
  transaction de lecture pour les tables de sélection) tenait le verrou d'écriture : écritures
  en `SQLITE_BUSY` à 10 s. D'où `TransactRead` pour les lectures longues.

## 6. Reproduire

```bash
scripts/bench-daemon-load.sh ~/src/bench-scale/mesure-0.37/sg/sample.db > rapport.md
LEVELS="10 50" DURATION=60s COLD_N=0 scripts/bench-daemon-load.sh base.db
PG_DSN=postgres://… scripts/bench-daemon-load.sh base.db   # migre le gabarit chaud, tenant 1
```

Une passe SQLite complète (gabarits + 3 niveaux + froid) dure ~4 min. Rapports, vidages,
profils et journaux du démon restent dans `bench/daemon-load.*` (ignoré par git). Ne pas
exporter `GOTMPDIR` pour mesurer.
