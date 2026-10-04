# Mesure de performance sur BMAB avant 0.37.0

Mesure, chiffrage, propositions — **aucune optimisation faite, aucun code modifié**. Le poids
part de [`plan-grosses-bases-2026-10/POIDS.md`](plan-grosses-bases-2026-10/POIDS.md), qui
n'est pas refait : ce document mesure ce que le schéma 2.31 et le codec binaire (ADR-0070,
ADR-0071) ont changé depuis.

## 0. Conditions

- **Binaire** : `main` à `180525cc7`, construit avec `GOTMPDIR` hors `/tmp` ; harnais de
  mesure (non committé) appelant `Database.OpenDatabase` avec `SetMigrationProgress`, et
  `MatchStore.ScoreMoves` qu'aucune commande n'appelle (§ 5).
- **Poste** : Ryzen 6850U, 16 threads, 14 Go de RAM, NVMe ext4. Temps mur et RSS maximal par
  `getrusage` des enfants ; 3 répétitions, la médiane est donnée sauf mention. La 1re ouverture
  (§ 1) a tourné sous charge (moyenne 27, sessions parallèles) ; tout le reste sur un poste au
  repos (charge 1 à 1,5).
- **Copie** : `~/src/bench-scale/mesure-0.37/run/`, copie de `bmab-europe.db` + `-wal` +
  `-shm`, refaite pour chaque répétition d'ouverture. **L'original n'a pas été ouvert** par
  blunderDB ni par `sqlite3` (seulement `cp`).
- **Place disque** : `/home` avait 5,6 Go libres pour une base de 13,9 Go. Le cache de
  compilation Go (`~/.cache/go-build`, 24 Go, régénérable) a été vidé pour faire tenir une
  copie. Une seule copie tient : le VACUUM (qui exige 2× la taille du fichier en libre) n'a
  pu être mesuré que sur un échantillon à 2 % (§ 4).

### État réel de l'original (lu sur la copie, après repli du WAL de 292 Mo)

| | |
|---|---|
| Fichier | 13 644 951 552 o + WAL 291 939 112 o ; page 4 Ko, `freelist_count` 0 |
| `database_version` | **`2.31.0`**, estampillé par un build 2.31 **antérieur** à la vague de poids (ADR-0071) |
| Dates | `position.match_date` et `analysis.creation_date` encore en **texte** (`DATETIME`) |
| Remplissage 2.30 | `backfill_position_match_date = 11 320 000` : **5,26 M positions sans date** ; `analysis_engine` NULL partout (provenance jamais dérivée) ; `creation_date` NULL partout |
| `match_stats` | **0 ligne** |
| Autres | pas d'`action_label`, `state` en texte JSON, index `idx_analysis_engine`/`depth` présents, `met_digest` (ancien nom de colonne), blobs 100 % JSON zstd niveau 7 |
| Volumes | 33 319 matchs, 268 724 parties, 16 584 349 positions (15 625 009 vivantes), 15 623 468 analyses, 16 584 350 coups, 7 833 commentaires (tous XG) |

Ouverte telle quelle, `main` ne la migrerait pas : la version dit déjà 2.31.0. **Chaque
répétition remet `database_version` à `2.30.0`** sur la copie avant l'ouverture : c'est ce
que vit une base 2.30, plus le reste du remplissage 2.30 qu'un utilisateur aurait de toute
façon. Le chemin réel d'un utilisateur de 0.36.0 part de **2.18.0** : les étapes 2.18 → 2.30
ne sont pas mesurées ici (repère : les étapes ≤ 2.28 → 2.30 coûtaient 3 min 42 s avant les
remplissages dans `~/src/bench-scale/migrate-timing.txt`).

## 1. Ouverture

| | Rép. 1 (chargé) | Rép. 2 | Rép. 3 |
|---|---:|---:|---:|
| **Ouverture migrante 2.30 → 2.31** | 2 576 s | **1 529 s** | **1 527 s** |
| CPU utilisateur + système | 1 849 s | 1 336 s | 1 342 s |
| RSS maximal | 1 288 Mo | 1 365 Mo | 1 365 Mo |
| WAL maximal / disque libre consommé au pic | 6,09 Go / 7,1 Go | 6,09 Go / 6,5 Go | 6,09 Go / 6,5 Go |
| Réouverture (`OpenDatabase`, rien à faire) | 0,2 s | 0,3 s | 0,3 s |
| Réouverture par `blunderDB info` | 61,2 s | 56,9 s | 58,7 s |

Phases (secondes, rép. 2 / rép. 3) :

| Phase | s | Part |
|---|---:|---:|
| Retrait des index élagués | 6 / 2 | |
| `position.match_date` → entier | 71 / 66 | 4 % |
| `analysis.creation_date` → entier | 22 / 23 | 1 % |
| **`position.state` → BLOB** (28 `json_extract` par ligne) | **207 / 212** | 14 % |
| **`analysis.best_cube_action` → code** (remplissage + `DROP COLUMN` sur 6 Go) | **216 / 203** | 14 % |
| `move.move_type`, `cube_action` → codes | 56 / 57 | 4 % |
| Reconstruction des index (`provenance_pending`, `creation_date`, `match_date`) | 86 / 88 | 6 % |
| Fin du remplissage `match_date` 2.30 (5,26 M positions) | 40 / 42 | 3 % |
| **Provenance des analyses** (décodage des 15,6 M blobs JSON) | **607 / 609** | 40 % |
| `match_stats` (33 319 matchs) + `ANALYZE` | 220 / 226 | 14 % |

- CPU ≈ 87 % du temps mur : la migration est **séquentielle et liée au CPU**, pas au disque.
- La réouverture d'une base à jour est instantanée ; `info` coûte **≈ 57 s** à lui seul (des
  `COUNT(*)` sur des tables de 15-16 M lignes et le décompte des gaffes), pas l'ouverture.
- Le pic de WAL (6,09 Go) vient de la réécriture d'`analysis` par `DROP COLUMN` en une
  transaction : **il faut ≈ 6,5 Go libres en plus du fichier** pour migrer cette base.

## 2. Recherche, un jeton à la fois

`blunderDB search --format json --limit 1000` (≈ une première page), sur la copie migrée
(blobs encore JSON). Médiane de 3 ; tout jeton absent du tableau est **≤ 0,6 s**
(`base`, `--decision cube`, `--cube 2`, `--score1/2`, `dd`, `dr`, `nc`, `i`, `fl`, `xco`, `p`,
`P`, `o`, `k`, `z`, `bo`, `bj`, `e`, `md`, `pl`, `op`, `pl`+`op`, `tn"…"`, `tn2`, `ma1`,
`id7`, `ad:xg`, `ad:3ply`, `xD65`, `au"…"`).

| Jeton | Médiane | RSS | Lecture |
|---|---:|---:|---|
| `t"illegal"` (texte de commentaire) | **109 s** | 430 Mo | 86 résultats ; d'après le code, le texte est filtré en Go après le parcours SQL (`loadCommentTexts`), alors que `comment` n'a que 7 833 lignes |
| `E>200` **sans limite** | **90 s** | 1,2 Go | |
| `pl!"…" E>100 ph:race` | 33 s | 424 Mo | coût de `E` |
| `s cube p>30 E>50` | 31 s | 422 Mo | coût de `E` |
| `E>100` | **30 s** | 424 Mo | ensemble `multiPlayedPlayer1Positions` recalculé à chaque requête puis passé en liste `IN (…)`, préchargement des coups |
| `pr>8` | 13 s | 500 Mo | |
| `W>60` / `w>60` | 8,1 / 6,5 s | 264 Mo | |
| `pl!"…"` | 6,1 s | 422 Mo | |
| `--dice 6,5` | 2,8 s (1re : 9,9) | 414 Mo | |
| `b>5` / `g>20` | 3,7 / 3,3 s | 244 Mo | |
| `ml:7`, `ph:race`, `co`, `co:xg`, `rd:3`, `T>2025/01/01` | 1,0-2,2 s | ≤ 420 Mo | |

Les répétitions sont stables (± 5 %) sauf la première lecture à froid de `--dice` et `b`.

## 3. Statistiques

| Commande | Médiane | RSS |
|---|---:|---:|
| **`list --type stats`** (toute la base) | **513 s** | **5,4 Go** |
| **`stats breakdown`** (toute la base) | **489 s** | **5,4 Go** |
| **`list --type stats --engine XG`** | **> 900 s** (2 × arrêté à 15 min ; 1 × arrêté à 31 min) | 4,8 Go |
| `list --type stats --player` | 51,7 s | 505 Mo |
| `stats recurring --player` | 39,9 s | 478 Mo |
| `stats progression --player` | 39,5 s | 507 Mo |
| `stats breakdown --player` | 34,6 s | 506 Mo |
| `list --type players` | 0,8 s (1re fois : **180 s**, 2,35 Go) | |
| `stats ranking`, `h2h`, `windows`, `list --type matches` | 0,1-0,6 s | ≤ 240 Mo |

- Les statistiques globales parcourent toutes les analyses et tous les coups en mémoire :
  **5,4 Go de RSS sur un poste de 14 Go**, plus de 8 min. Avec le filtre moteur, le plan
  retombe sur `SCAN mv` (déjà noté dans `SCHEMA-2-31.md`) et ne termine pas en 31 min.
- `list --type players` paie 180 s une seule fois, puis 0,8 s : quelque chose est rempli à la
  première lecture (hypothèse : un remplissage paresseux de `match_stats`, appelé par
  plusieurs chemins de lecture ; à confirmer).

## 4. Vacuum

Impossible sur la base entière ici (il exige 2× la taille du fichier en libre, ≈ 28 Go).
Mesuré sur un échantillon déterministe à 2 % tiré de la copie migrée (positions
`id % 50 = 0`, leurs analyses et coups, tous les matchs et parties ; 312 586 positions) :

| Échantillon | Avant | Après | Temps | RSS |
|---|---:|---:|---:|---:|
| Blobs binaires niveau 7 (après `reencode`), 3 rép. | 239,2 Mo | 227,8 Mo (−4,8 %) | **741 / 732 / 730 s** | 658 Mo |
| Blobs JSON (sortie de migration), 1 rép. | 282,2 Mo | 227,7 Mo (−19,3 %) | **744 s** | 684 Mo |
| Déjà compacté (rien à recompresser), 3 rép. | 227,8 Mo | 227,8 Mo | **3,5 s** | 502 Mo |

- **99,5 % du temps de `vacuum` est la recompression niveau 19** (`compactAnalyses`,
  séquentielle, ≈ 2,35 ms par blob) ; le VACUUM SQLite + `ANALYZE` font 3,5 s.
- Extrapolé à la base entière (×50 sur 15,6 M blobs) : **≈ 10,2 h** de recompression, quel
  que soit le format de départ.
- Ce que le niveau 19 rapporte par rapport au niveau 7 binaire : **158,4 → 155,8 o par blob
  (−1,6 %), ≈ 41 Mo sur toute la base**.

## 5. Remplissages et analyse

| Passe | Temps | Débit | RSS |
|---|---:|---:|---:|
| `MatchStore.ScoreMoves` (`move.error_mp`), lots de 5 000 | **596 s** | 27 800 coups/s ; 15 527 383 coups notés sur 16 584 350 | 415 Mo |
| `reencode` (JSON → binaire niveau 7), 15 623 468 blobs | **1 131 s** | 13 800 blobs/s, **séquentiel** (72 µs/blob) | 447 Mo |
| `analyze --stale` (aucune analyse gammonNet dans BMAB) | 6,9 s (1re : 23,6 s) | sonde seule | 414 Mo |
| `analyze` (positions sans analyse : 1 541 ; 1 364 évaluées, 177 refusées), ply 2, 16 jobs | 81,7 s | **≈ 17 positions/s** (0,65 s CPU/position) | 600 Mo |
| `analyze --stale --ply 1` sur ces 1 364 analyses, 3 rép. | 15,7 s | ≈ 160 positions/s (sonde 7 s déduite) | 590 Mo |

- **`ScoreMoves` n'a aucun appelant dans le produit** (ni CLI, ni GUI, ni serveur) : sur une
  base migrée, `move.error_mp` reste NULL. Mesuré par le harnais.
- Réanalyser BMAB entière au ply 2 : 15,6 M ÷ 17/s ≈ **10,6 jours** sur ce poste.
- `--stale --ply 1` réévalue les 1 364 analyses ply 2 **à chaque passe** (3 fois de suite)
  et `analysis_depth` reste à 2 ; `--stale --ply 2` ensuite ne trouve rien. La passe ne
  converge pas ou ne réécrit pas la profondeur : à vérifier (non diagnostiqué ici).

## 6. Poids

### 6.1 `dbstat` après migration 2.31 (blobs JSON)

Fichier **14,03 Go** (3 425 416 pages ; 13 906 libres = 57 Mo). Contenu : **10,23 Go de
données utiles, 2,33 Go d'octets inutilisés dans les pages**, index 4,33 Go.

| Objet | Mo | dont inutilisé | Commentaire |
|---|---:|---:|---|
| table `analysis` | 6 087 | 863 | blobs 4,43 Go |
| table `position` | 2 665 | **1 030 (39 %)** | `state` rétréci et dates réécrites sur place : trous dans chaque page |
| table `move` | 866 | 264 | |
| `idx_position_zobrist` | 302 | 28 | |
| `idx_analysis_win_gammon(2)_covering` | 295 + 294 | | |
| `idx_move_position` / `idx_move_game` | 241 / 235 | 26 / 29 | |
| `idx_analysis_creation_date` | 241 | 30 | entier, contre 448 Mo en texte (POIDS § 1.2) |
| `idx_analysis_position` | 232 | 29 | |
| `idx_position_match_date` | 226 | 15 | entier, contre 607 Mo en texte |
| 13 autres index de `position`, 7 de `analysis` | 2 040 | | |
| `match`, `game`, `match_stats`, le reste | 30 | | |

### 6.2 Après `reencode`

Blobs **283,5 → 158,4 o** en moyenne (−44 %, conforme à ADR-0070) : la charge d'`analysis`
passe de 5,07 à 3,12 Go (**−1,96 Go**)… mais **le fichier reste à 14,03 Go** : les octets
libérés restent en trous dans les pages (`analysis` : 2,83 Go inutilisés). Sans VACUUM,
`reencode` ne rend rien au disque.

### 6.3 Après VACUUM (extrapolé de l'échantillon)

Échantillon compacté : 227,8 Mo, dont 32,9 Mo de tables prises en entier (`match`, `game`,
`match_stats`…) ; le reste ×50 → **≈ 9,8 Go** pour la base entière, contre 14,03 Go
(**−30 %**). C'est le chiffre que POIDS visait (≈ 9 Go avec toutes ses options) : la 2.31 et le
codec binaire y sont presque, à condition de passer `reencode` puis un VACUUM.

### 6.4 Dictionnaire zstd réentraîné sur BMAB (charges binaires)

125 000 analyses tirées (graine 20261004), converties par `engine.MarshalAnalysisBinary` ;
25 000 pour entraîner (`zstd --train`), 100 000 pour mesurer avec klauspost (le codec du
produit). Charge binaire brute : **180,7 o**.

| Dictionnaire | L3 | **L7** (écriture) | L19 (5 000) | Encodage L7 / L19 |
|---|---:|---:|---:|---:|
| aucun | 169,1 | 168,6 | 170,8 | 8 µs / 36 µs |
| **embarqué** (64 Ko, corpus du dépôt) | 163,1 | **158,2** | 155,1 | 22 µs / 2,45 ms |
| BMAB 16 Ko | 163,9 | 158,7 | 155,7 | |
| BMAB 32 Ko | 163,1 | 157,4 | 153,3 | |
| BMAB 64 Ko | 161,4 | **155,9** (−1,5 %) | 152,0 | 23 µs / 2,39 ms |
| BMAB 112 Ko | 160,4 | 154,5 (−2,3 %) | 150,3 | |

- Réentraîner gagne **1,5 à 2,3 %** des blobs, soit 36 à 58 Mo sur BMAB : **ne vaut pas** un
  second dictionnaire binaire.
- Le dictionnaire lui-même ne rapporte que 6 % (168,6 → 158,2 o) : l'essentiel du gain vient
  de l'encodage binaire, plus du zstd.
- Le niveau 19 rapporte 2 % pour un encodage **110 fois** plus lent.

### 6.5 Redondances restantes (au-delà de POIDS)

- **Trous de pages** : 2,33 Go après migration, 4,3 Go après `reencode` ; le seul vrai poids
  « redondant » de la 2.31. Seul un VACUUM le rend.
- Options de POIDS non reprises par la 2.31, chiffres de POIDS inchangés : `analysis` clé
  `position_id` (−0,41 Go), `move` WITHOUT ROWID (−0,19 Go), `page_size` 16 Ko (−0,32 Go).
- Retirer `positionId`, noms et dates du blob binaire (variante E de POIDS) : 3 points de
  plus sur les blobs (≈ 0,1 Go) ; non remesuré.

## 7. Goulots classés par gain

| # | Goulot | Mesure | Proposition | Gain estimé | Risque |
|---|---|---|---|---|---|
| 1 | **`vacuum` recompresse au niveau 19, en série** | ≈ 10,2 h sur BMAB pour 41 Mo (0,4 %) | Ne compacter que les blobs non binaires, au niveau 7 (le chemin de `reencode`), ou sauter la compaction ; paralléliser par lots si elle reste | **≈ 10 h → ≈ 20 min** (≈ 4 min parallélisé) | faible : même format, le lecteur lit les deux niveaux |
| 2 | **`vacuum` exige 2× le fichier en libre** | 28 Go pour BMAB ; impossible sur ce poste | Documenter ; étudier `VACUUM INTO` vers le même volume + remplacement (1× au lieu de 2×, le WAL n'enfle pas) | rend le VACUUM possible : **−4,2 Go (−30 %)** | moyen : remplacement de fichier, base ouverte ailleurs |
| 3 | **Statistiques globales** | 8,5 min, 5,4 Go RSS ; filtre moteur > 15 min | Les calculer depuis `match_stats` (déjà par match) ; corriger le plan du filtre moteur (`SCAN mv`) | **minutes → secondes**, RSS ÷ 10 | moyen : parité des chiffres à tenir (tests de contrat) |
| 4 | **Ouverture 2.31** : 25,5 min, CPU séquentiel | provenance 607 s, `state` 207 s, codes d'action 210 s | Paralléliser le décodage de la provenance (`DecodeAnalysesConcurrently` existe) ; convertir `state` en Go plutôt que 28 `json_extract` par ligne | **≈ −10 min** (provenance ÷ 4-8, `state` ÷ 3) | moyen : migration, à rejouer sur l'échantillon |
| 5 | **`t"…"`** | 109 s pour 86 résultats | Restreindre d'abord par `EXISTS` sur `comment` (7 833 lignes) avant le filtre Go | **109 s → < 1 s** | faible |
| 6 | **`E`** (erreur du coup joué) | 30 s par requête ; 90 s sans limite | Ne pas recalculer `multiPlayedPlayer1Positions` à chaque requête ; s'appuyer sur `move.error_mp` une fois rempli (§ 5) | **30 s → quelques s** | moyen : dépend de #8 |
| 7 | **`reencode` séquentiel** | 19 min | Décodage/encodage parallèle par lots | **≈ −15 min** | faible |
| 8 | **`ScoreMoves` jamais appelé** | 10 min sur BMAB | Le brancher (après l'ouverture, ou dans `repair`/`reencode`) | rend `error_mp` utile | faible |
| 9 | **`info`** | 57 s | Compter par `MAX(id)`/statistiques ou à la demande | **−57 s** par appel | faible |
| 10 | **Pic de WAL de la migration** | +6,5 Go libres requis | `wal_checkpoint(TRUNCATE)` entre les phases ; éviter `DROP COLUMN` sur `analysis` (colonne laissée NULL) | pic ÷ 2 environ (estimé, non mesuré) | faible |
| 11 | `pr`, `w/W`, `pl!` | 6-13 s | à examiner sur plans de requêtes | quelques s | — |
| 12 | Dictionnaire réentraîné | −1,5 à −2,3 % des blobs | **ne pas faire** | 36-58 Mo | — |

### 7.1 Points #1 et #7 traités (branche `perf/vacuum-rapide`)

Mesuré sur l'échantillon à 2 % en blobs JSON (312 544 analyses, 16 cœurs, machine partagée
donc bruitée), binaire de la branche contre le binaire de la mesure :

| | Avant | Après |
|---|---:|---:|
| `vacuum` (JSON → binaire, VACUUM, ANALYZE) | **744 s** (282,2 → 227,7 Mo, § 4) | **13,7 s** (282,2 → 228,6 Mo) |
| `vacuum`, blobs déjà binaires niveau 7 (autre base, 239,2 Mo) | 730-741 s | pas de réécriture : VACUUM + ANALYZE seuls (≈ 3,5 s) |
| `reencode` | 24,1 s | **11,5 s** |

- `vacuum` ne recompresse plus au niveau 19 : il ne réécrit que les blobs hérités, au niveau 7
  (le chemin de `reencode`). Le niveau 19 ne rendait que 1,6 % de la taille d'un blob.
  Même base de départ des deux côtés : l'échantillon JSON de 282,2 Mo (le « Avant » est la
  ligne « Blobs JSON » du § 4 ; `vacuum` affiche des Mio : 269,1 → 218,0). Taille finale
  228,6 Mo contre 227,7 Mo avec le niveau 19 : +0,9 Mo, soit +0,4 % du fichier (≈ 2,6 o par
  blob, ≈ 41 Mo sur BMAB). Sur BMAB : ≈ 10 h de recompression en moins.
- La recompression se fait sur tous les cœurs (`engine.RecompressAnalysesConcurrently`),
  par lots de 2 000 comme avant, pour `vacuum` et pour `reencode` (les deux backends).
- Piège trouvé : `zstdEncoder` et `zstdDecoder` sont à concurrence 1, ce qui sérialise
  tout appel. Une première version parallèle, sur ces instances partagées, n'allait pas plus
  vite que la boucle séquentielle (31 s contre 24 s). Le passage en masse a ses propres
  instances, une place par cœur. `DecodeAnalysesConcurrently` (lecture) souffre du même
  verrou : suivi possible.

Pour la release : #1 et #2 décident si un utilisateur de BMAB peut réellement récupérer les
30 % promis par la 2.31 ; #3 et #5 sont les seules attentes de plusieurs minutes dans un
usage courant ; #4 est payé une fois par base.

## 8. Reproduire

Scripts dans `~/src/bench-scale/mesure-0.37/` : `mesure.py` (temps, CPU, RSS), `rep.sh`
(copie, remise en 2.30.0, ouverture, réouvertures), `tokens.sh` + `tokens.tsv`, `stats.sh`,
`post.sh` (remplissages, `dbstat`, `reencode`), `stale.sh`, `sample.sh` + `vac.sh`
(échantillon à 2 % et vacuum), `phases.py` ; résultats bruts `*.jsonl` et `*.out`. Le
harnais Go (`cmd/zz-mesure`, modes `open`, `scoremoves`, `dictsample`, `dictmeasure`) n'est
pas committé.
