# Poids d'une base blunderDB — étude chiffrée (2026-10-04)

Étude sans modification du produit. Base de référence : `~/src/bench-scale/bmab-europe.db`
(12,77 Go, schéma 2.28.0, 15 623 468 analyses, 15 625 009 positions, 16 584 350 coups,
33 319 matchs), ouverte en lecture seule et immuable (`?mode=ro&immutable=1`), jamais
migrée ni copiée en entier. Poste : Ryzen 6850U, 16 threads, **chargé par d'autres travaux
pendant toute la mesure** (charge moyenne 30 à 45) : les temps CPU absolus sont pessimistes
d'un facteur 1,5 environ (le niveau 7 mesuré à 0,31 ms au repos dans [`MESURES.md`](MESURES.md)
ressort ici à 0,33-0,48 ms) ; les rapports entre options restent valables, les tailles sont
exactes.

Échelle d'extrapolation. D'après [`MESURES.md`](MESURES.md) § 6, le corpus BMAB est livré en
cinq copies identiques : ses ≈ 33 370 matchs distincts, c'est **cette base, à 0,2 % près**
(facteur ×1). L'hypothèse initiale du plan (≈ 190 000 fichiers, 40 à 70 M de positions)
correspond à un facteur **×2,6 à ×4,5** sur le nombre de positions ; les deux colonnes sont
données.

## 1. Où sont les octets

### 1.1 Schéma 2.28 mesuré (`dbstat`, base complète)

| Objet | Mo | Part | o/ligne | Remplissage (payload/page) |
|---|---:|---:|---:|---:|
| table `analysis` | 5 422 | 42,5 % | 347 | 0,92 |
| — dont blobs `analysis.data` | 4 433 | 34,7 % | 284 | |
| — dont colonnes scalaires + en-têtes | 990 | 7,8 % | 63 | |
| index de `position` (17) | 2 413 | 18,9 % | | 0,67-0,75 |
| table `position` | 2 170 | 17,0 % | 139 | 0,92 |
| index de `analysis` (11) | 1 535 | 12,0 % | | 0,67-0,83 |
| table `move` | 723 | 5,7 % | 44 | 0,83 |
| index de `move` (2) | 477 | 3,7 % | | 0,66-0,68 |
| `match`, `game`, leurs index, le reste | 27 | 0,2 % | | |
| **Total** | **12 767** | | **≈ 817 o/position** | |

- **Index : 4,43 Go, 34,7 %** de la base. Les plus gros : `idx_position_zobrist` 302 Mo,
  `idx_analysis_win_gammon_covering` 289 Mo, `idx_move_position` 241 Mo, `idx_move_game`
  235 Mo, `idx_analysis_position` 232 Mo, `idx_position_score_cube` 226 Mo ; un index à une
  colonne entière coûte ≈ 10,5 o par position (≈ 165 Mo).
- **Espace libre** : `freelist_count` = 0 ; octets inutilisés dans les pages 454 Mo (3,6 %).
  Le remplissage « payload/page » des index (0,66-0,83) mesure aussi les en-têtes de cellule ;
  un VACUUM ne le relève que pour les index alimentés dans le désordre (`zobrist`,
  `move_position`, `move_game`, `analysis_position` : −9 à −12 % chacun, ≈ 100 Mo en tout),
  les autres sont déjà au niveau d'une reconstruction triée (mesuré sur l'échantillon § 1.2).
- `match` ne pèse que 10,5 Mo (noms, événement, chemin ≈ 90 o par match) : les colonnes texte
  répétées de `match` ne sont pas un sujet de poids.

### 1.2 Schéma 2.30 (celui de `main`) — estimé sur un échantillon

La migration 2.30 n'a pas été jouée sur la base (elle l'aurait modifiée). Elle a été rejouée
sur un échantillon déterministe à 2 % (positions `id % 50 = 0`, leurs analyses et leurs coups,
tous les matchs et parties), reconstruit trié par `VACUUM` : 312 586 positions. Ses tailles
par ligne recoupent la base complète à 1 % près sur les tables (analysis 345,8 contre 347,1
o/ligne ; position 138,3 contre 138,9) et sur les index non désordonnés.

| | Échantillon (hors match/game) | Extrapolé ×50 |
|---|---:|---:|
| Schéma 2.28 reconstruit | 252,3 Mo | ≈ 12,6 Go |
| Schéma 2.30 (index élagués, colonnes de provenance et `match_date` ajoutées) | 287,5 Mo (**+13,9 %**) | **≈ 14,4 Go** |

La 2.30 retire 5 index (≈ 860 Mo) mais ajoute davantage : `position.match_date` stocke le
texte `AAAA-MM-JJ hh:mm:ss +0000 UTC` (29 o) dans la ligne (+33 o/position) **et** dans
`idx_position_match_date` (**607 Mo**, le plus gros index de la base) ;
`analysis.creation_date` (texte de 19 o) et son index (448 Mo) ; `idx_analysis_engine`
(180 Mo, une seule valeur distincte dans BMAB) ; `idx_analysis_depth` (165 Mo) ;
`idx_analysis_win_gammon2_covering` (291 Mo).

## 2. Les blobs d'analyse

### 2.1 Format

`analysis.data` (`engine/analysiscodec.go`, ADR-0030) : JSON de `domain.PositionAnalysis`
compressé par zstd (klauspost, Go pur) avec le dictionnaire embarqué
`engine/analysis_dict.bin` (32 Ko, Dictionary_ID dans chaque trame). Écriture au niveau 7
(`SpeedBetterCompression`, sans checksum) ; `sqlite.Storage.Vacuum` recompresse au niveau 19
(avec checksum, ce qui distingue les deux). Le décodeur lit aussi zlib et JSON brut.

### 2.2 Mesures (balayage complet des 15 623 468 blobs)

| Grandeur | Valeur |
|---|---|
| Blobs | 4 432 553 917 o, **moyenne 283,7 o** ; 100 % au niveau 7 (aucun au niveau 19 : la base n'a jamais été compactée) |
| JSON décompressé | 29,2 Go, moyenne 1 870 o (taux 6,6) |
| Distribution (échantillon 125 000) | compressé p10 132, p50 194, p90 527, p99 564, max 633 o ; JSON p50 1 100, p90 3 604 o. Bimodale : décision de videau seule (≈ 38 %) ou coup à 1 coup légal ≈ 130-200 o ; coup avec 10 candidats ≈ 520 o |
| Contenu | cube seul 38 %, cube + pions 37 %, pions seuls 25 % ; 1 à 12 coups candidats (10 dans 33 % des cas) ; jamais de `allCubeAnalyses` ni de rollout dans BMAB ; moteur « XG » partout |

### 2.3 Ce qui se répète ou ne sert pas

- **Les noms de clés JSON** (≈ 25 par coup candidat) : le dictionnaire les absorbe déjà
  (sans dictionnaire, 519 o au lieu de 227 au niveau 7 libzstd).
- **Les deux horodatages RFC 3339 à la nanoseconde** (`creationDate`, `lastModifiedDate`) :
  18 chiffres de nanosecondes aléatoires par blob, incompressibles — 5,7 % des octets
  compressés à eux seuls.
- **`positionId`** (redit `analysis.position_id`) et **`player1`/`player2`** (noms des
  joueurs du premier match importé, redits dans `match`) : ≈ 2,7 %.
- **Champs dérivables** : `opponentWinChance` = 100 − `playerWinChance` (99,998 %) ;
  `equityError` = équité du meilleur − équité du coup (91 %) ; `index` = rang.
- **Précision** : déjà quantifiée à l'affichage à l'import (`RoundToMillipoint` pour les
  équités, `RoundToHundredthPercent` pour les probabilités : 2 décimales de pourcentage,
  3 d'équité, conformes à XG). Une quantification plus grossière perdrait de l'information
  affichée : **pas d'option ici**.
- **Analyses identiques** (même contenu hors `positionId`, noms, dates, coups joués), balayage
  complet : 337 756 lignes redondantes (2,2 %) en 57 751 groupes, presque toutes des
  fins de partie triviales (`1/off` à 100 %, 7 985 copies) ; elles sont petites et ne
  représentent que **0,9 % des octets de blobs**.

## 3. Options mesurées

Méthode commune des blobs : 125 000 blobs tirés uniformément sur `analysis.id` (graine
20261004) ; 25 000 servent à entraîner les dictionnaires, **100 000 servent à mesurer**.
Deux implémentations de zstd ont été mesurées : klauspost (celle du produit, Go pur, le
démon `cmd/serve` exclut cgo) et libzstd (référence, via python-zstandard) ; **klauspost
compresse nettement moins bien que libzstd au même numéro de niveau** (283 o contre 227 o au
niveau 7, voir § 3.1), ce sont donc ses chiffres qui comptent.

### 3.1 Niveaux et dictionnaire zstd (JSON inchangé)

klauspost (le codec du produit), 100 000 blobs aux niveaux 3 et 7, 5 000 au niveau 19 (coût
d'encodage) ; taille moyenne en octets, temps par blob sur le poste chargé :

| Dictionnaire | L3 | L7 | L19 (5 000) | Encodage L7 / L19 | Décodage L7 |
|---|---:|---:|---:|---:|---:|
| actuel 32 Ko | 306,2 | **283,1** | 251,6 | 0,33 ms / 5,7 ms | 30 µs |
| réentraîné 32 Ko | 305,3 | 281,6 | 250,3 | 0,53 ms / 9,3 ms | 19 µs |
| réentraîné 64 Ko | 294,2 | 273,1 | 243,5 | 0,75 ms / 11,2 ms | 24 µs |

- Le niveau 7 actuel donne 283,1 o sur l'échantillon pour 283,7 o mesurés sur la base entière :
  l'échantillon est représentatif.
- Niveau 19 contre 7 sur **les mêmes** 5 000 blobs : 251,6 contre 276,6 o, **−9,0 %** ; le
  niveau 3 perd 8 % (299,4 o).
- Réentraîner le dictionnaire (64 Ko) : **−3,5 %** au niveau 7, pour un encodage plus lent.

libzstd, mêmes 100 000 blobs, taille moyenne en octets :

| Dictionnaire | L3 | L7 | L19 |
|---|---:|---:|---:|
| aucun | | 519 | |
| actuel 32 Ko | 268,5 | 227,3 | 206,8 |
| réentraîné 16 Ko | 260,8 | 223,5 | 202,0 |
| réentraîné 32 Ko | 259,7 | 221,2 | 199,7 |
| réentraîné 64 Ko | 265,2 | 220,9 | 196,6 |
| réentraîné 128 Ko | 265,1 | 226,9 | 196,2 |

Le dictionnaire actuel est bon : le réentraîner sur BMAB ne gagne que 1 à 3 %, quelle que
soit sa taille (16 à 128 Ko). Où il vivrait : **embarqué** (`//go:embed`, un second
Dictionary_ID enregistré dans le décodeur, aucune migration de lignes : ADR-0030 l'a prévu)
plutôt que versionné en base : 32-64 Ko par base ne coûtent rien, mais un dictionnaire en
base doit voyager avec chaque export, être lu avant le premier blob et être résolu à
l'import d'une autre base, pour 1 à 3 % de gain.

### 3.2 Encodage avant compression

Prototype d'encodage binaire (`binenc.py`, varints, énumérations pour profondeur et action
de videau, nombres en entiers ×100 / ×1000 avec échappement float64 si la valeur n'est pas
exacte, `opponentWinChance` et `equityError` dérivés quand ils le sont) : **aller-retour
vérifié sans écart sur les 125 000 analyses**. Variante « D » : tous les champs, dates en
texte ; variante « E » : sans `positionId`, noms ni dates (qui iraient en colonnes ou
disparaîtraient).

| Variante (dictionnaire réentraîné 64 Ko sur la variante) | brut | libzstd L3 | L7 | L19 |
|---|---:|---:|---:|---:|
| A — JSON actuel | 1 863 | 259,1 | 220,6 | 196,3 |
| C — JSON, dates tronquées à la seconde | 1 843 | 250,8 | 208,1 | 186,5 |
| B2 — JSON sans `positionId` ni noms | 1 816 | 257,2 | 214,6 | 190,8 |
| B — JSON sans `positionId`, noms ni dates | 1 706 | 234,4 | 191,5 | 172,5 |
| D — binaire complet | 242 | 153,6 | 154,9 | 143,6 |
| E — binaire sans `positionId`, noms ni dates | 165 | 120,5 | 125,1 | 115,0 |

Variante D (binaire complet) avec klauspost, dictionnaire réentraîné sur la variante :

| Dictionnaire | L3 | L7 | L19 (5 000) | Décodage L7 |
|---|---:|---:|---:|---:|
| binaire 32 Ko | 180,3 | 173,7 | 162,9 | 8 µs |
| binaire 64 Ko | 178,5 | **172,1** | 160,7 | 4 µs |

Contre 283,1 o aujourd'hui : **−39 % sur les blobs** au niveau 7 sans changer de niveau ; le
niveau 19 n'y ajoute plus que 4,5 % (168,2 → 160,7 o sur les mêmes 5 000). La variante E n'a
été mesurée qu'en libzstd ; au rapport E/D de libzstd (125,1/154,9) elle donnerait ≈ 139 o
avec klauspost (−51 %).

Coût CPU côté Go du JSON que l'encodage binaire supprimerait : `json.Unmarshal` **117 µs** et `json.Marshal` 39 µs par analyse, contre 30 µs de décompression zstd : lire une analyse coûte ≈ 147 µs aujourd'hui, dont 80 % de JSON ; le binaire se décompresse en 4 µs et se décode en quelques µs (non mesuré en Go), soit **un ordre de grandeur de moins** à la lecture (un affichage de 1 000 résultats : ≈ 0,15 s → ≈ 0,01 s de CPU).

### 3.3 Déduplication des analyses identiques

0,9 % des octets de blobs au mieux (§ 2.3), avant le coût d'une table de contenus partagés
(une référence par ligne + un index d'empreinte, ≈ 15-25 o par analyse) : **bilan négatif**.
Les positions sont déjà dédupliquées par Zobrist (ADR-0001).

### 3.4 Schéma (échantillon 2 %, à partir du 2.30 reconstruit)

| Variante cumulée | Échantillon | Écart | Extrapolé ×50 |
|---|---:|---:|---:|
| 2.30 | 287,5 Mo | — | ≈ 14,4 Go |
| V2 : `analysis` clé `position_id INTEGER PRIMARY KEY` (supprime `id` et `idx_analysis_position`, raccourcit les index couvrants) | 279,2 Mo | −8,3 Mo (−2,9 %) | −0,41 Go |
| V3 : dates en entiers Unix (`position.match_date`, `analysis.creation_date`) | 253,1 Mo | −26,1 Mo (−9,1 %) | **−1,30 Go** |
| V4 : énumérations entières (`best_cube_action`, `move.move_type`, `move.cube_action`, `analysis_engine`) et `position.state` en BLOB de 28 o au lieu du texte JSON de 63 o | 235,5 Mo | −17,6 Mo (−6,1 %) | −0,88 Go |
| V5 : `move` WITHOUT ROWID sur `(game_id, move_number)` (remplace `idx_move_game`) | 231,7 Mo | −3,8 Mo (−1,6 %) | −0,19 Go |

- **WITHOUT ROWID** n'apporte rien à `analysis` au-delà de V2 (la clé entière y est déjà
  l'alias du rowid ; des lignes de 350 o sont en outre au-delà de ce que SQLite recommande
  pour WITHOUT ROWID) ; pour `move` il gagne 1,6 % mais `move.id` est référencé
  (`move_analysis`, curseurs de navigation).
- **page_size** (VACUUM après `PRAGMA page_size`) : 2.30 → 8 Ko −1,6 %, 16 Ko −2,2 % ; sur
  V4 : −1,4 % et −2,0 %. Aucun changement de schéma, mais la base est en WAL, où
  `page_size` ne change qu'en quittant WAL le temps du VACUUM ; les lectures ponctuelles
  (recherche par Zobrist, page de résultats) liraient 2 à 4 fois plus d'octets par page.
- **Index élagables** : la 2.30 a déjà élagué sur plans de requêtes réels
  (`tasks/search-query-plans.txt`). Restent à examiner de la même façon
  `idx_analysis_engine` (180 Mo, une seule valeur dans BMAB, ne peut servir à rien sur
  une base mono-moteur) et `idx_analysis_depth` (165 Mo, 6 valeurs) : jusqu'à −345 Mo
  (−2,4 %), ou un index partiel. Les cardinalités mesurées : `game_phase` 4, `game_type` 11,
  dés 24, `back_checkers` 14, `off` 226, `pip_diff` 507, score/videau 2 037.
- **Colonnes texte de `match` normalisées** (table de joueurs, d'événements) : 10,5 Mo en
  tout, gain < 0,1 % — hors sujet pour le poids (utile pour les alias, GB5.2, pas pour
  l'octet). Les noms qui pèsent sont ceux **recopiés dans chaque blob** (§ 3.2, B2).

### 3.5 VACUUM avec compaction niveau 19 (déjà dans le produit)

- Gain sur les blobs : −9,0 % (283 → ≈ 258 o), **≈ −0,40 Go** sur cette base (−3,1 %), et ≈ 100 Mo de réordonnancement des index désordonnés.
- Coût : la passe de compaction (`vacuum_sqlite.go`) est **séquentielle**, un blob après
  l'autre : 5,7 à 6,7 ms d'encodage par blob mesurés sous charge (≈ 4 ms au repos), soit **≈ 17 à 28 h** pour 15,6 M blobs sur cette base. Le VACUUM lui-même réécrit la base entière dans un
  fichier temporaire de même taille (14 Go libres sur ce poste : impossible ici).
- Décodage : pas de différence mesurable sous la charge du poste (9 à 37 µs selon les passes).

## 4. Synthèse et extrapolation

Base de départ : schéma 2.30 (`main`, la 2.31 n'ajoute que `move.error_mp` et
`analysis.met_digest`, NULL pour les analyses XG de BMAB). Gains estimés indépendamment ;
le cumul tient compte des recouvrements (la compaction L19 ne s'ajoute pas au binaire).

| Option | Gain ×1 (BMAB) | Gain ×3,5 (40-70 M pos.) | Part | Risque |
|---|---:|---:|---:|---|
| Référence 2.30 | 14,4 Go | ≈ 50 Go | | |
| Blobs en binaire, sans `positionId`/noms/dates (E) | −2,2 Go | −7,7 Go | −15 % | moyen |
| Blobs en binaire complet (D) | −1,7 Go | −6,1 Go | −12 % | moyen |
| Dates en entiers Unix (V3) | −1,30 Go | −4,6 Go | −9,1 % | faible |
| Énumérations entières + `state` en BLOB (V4) | −0,88 Go | −3,1 Go | −6,1 % | moyen |
| `analysis` clé `position_id` (V2) | −0,41 Go | −1,4 Go | −2,9 % | moyen |
| Compaction L19 (existe) | −0,40 Go | −1,4 Go | −3,1 % | nul, mais 17-28 h CPU |
| Index `engine` + `depth` retirés | −0,35 Go | −1,2 Go | −2,4 % | faible |
| `page_size` 16 Ko | −0,32 Go | −1,1 Go | −2,2 % | moyen (lectures ponctuelles) |
| `move` WITHOUT ROWID (V5) | −0,19 Go | −0,7 Go | −1,6 % | moyen |
| Dictionnaire JSON réentraîné | −0,16 Go | −0,5 Go | −1,1 % | faible |
| Déduplication des analyses | ≤ −0,04 Go brut, **négatif** net | | | |
| Normalisation des textes de `match` | < −0,01 Go | | | |
| **Cumul V2+V3+V4+V5 + binaire E + index** | **≈ −5,4 Go → ≈ 9,0 Go** | **≈ 31 Go** | **≈ −38 %** | |

## 5. Recommandation (gain/risque)

Par ordre gain/risque :

1. **Dates en entiers** (`position.match_date`, `analysis.creation_date`) : −9 % pour une
   migration mécanique (réécriture de deux colonnes et reconstruction de deux index) et une
   conversion au bord du stockage ; aucun format de blob touché. Le plus gros index de la
   base (`idx_position_match_date`, 607 Mo) passe de 29 o de texte à un entier de 4-6 o. À
   faire de préférence avant que des bases 2.30 de cette taille n'existent chez les
   utilisateurs : la migration 2.30 est celle qui les crée.
2. **Codec binaire pour `analysis.data`** : le plus gros gain (−12 à −15 %, −39 à −51 % des
   blobs) **et** une lecture dix fois moins chère (le JSON fait 80 % du coût de décodage).
   ADR-0030 prévoit déjà la détection de format par en-tête : un nouveau format s'ajoute à la
   lecture sans migration, les lignes anciennes restent lisibles et une passe reprenable
   (comme la compaction actuelle) peut convertir. Risque : l'aller-retour doit être exact
   (prototype vérifié sur 125 000 analyses ; à fuzzer en Go) et PostgreSQL stocke le même
   blob. Retirer du blob `positionId`, `player1`/`player2` et les deux horodatages
   (variante E) ajoute 3 points et supprime de l'information redondante.
3. **Index `idx_analysis_engine` et `idx_analysis_depth`** : −2,4 %, à trancher sur plans
   de requêtes comme l'a fait la 2.30 (ou en index partiel).
4. **Énumérations entières et `state` en BLOB** : −6 %, mais `state` est lu partout et la
   parité PostgreSQL double le travail ; après les trois premiers.
5. **Compaction L19 du VACUUM** : à la taille de BMAB elle est séquentielle et coûte 17 à
   28 h pour −3 % ; la paralléliser par lots si elle reste, et la cantonner au seul gain du
   niveau 19 si le codec binaire arrive (−4,5 % seulement sur des blobs binaires).

**À ne pas faire** : déduplication des analyses (bilan négatif), dictionnaire versionné en
base (1 à 3 % contre une complication d'export/import ; un second dictionnaire embarqué
suffit), quantification plus grossière (perdrait de l'information affichée), WITHOUT
ROWID et `page_size` (1,6-2,2 % contre des clés référencées et des lectures ponctuelles plus
coûteuses), normalisation des textes de `match` pour le poids.

Ordre de grandeur final : la base BMAB passerait de ≈ 14,4 Go (2.30) à ≈ 9 Go ; à l'échelle
initiale du plan (×3,5), de ≈ 50 Go à ≈ 31 Go.

## 6. Reproduire

Scripts dans `~/src/bench-scale/poids/` pendant l'étude (supprimés depuis, recopiés
ci-dessous dans leur principe) :

1. `dbstat` : `sqlite3 'file:…/bmab-europe.db?mode=ro&immutable=1' "select name,
   count(*), sum(pgsize), sum(payload), sum(unused), sum(ncell) from dbstat group by name"`
   (le `sqlite3` d'Arch 3.53 a `ENABLE_DBSTAT_VTAB` ; ≈ 15 min).
2. Échantillon : 125 000 `analysis.id` tirés par `random.sample(range(1, max+1), 125000)`
   (graine 20261004), lus par paquets de 500 `WHERE id IN (…)` dans une base de travail ;
   mélange graine 1, 25 000 pour entraîner, 100 000 pour mesurer.
3. Balayage complet : 8 processus lisant `analysis` par plages de 500 000 ids, décompression
   avec `analysis_dict.bin`, empreinte blake2b-64 du JSON sans `positionId`, noms, dates et
   coups joués, `sort | uniq -c`.
4. Dictionnaires : `zstandard.train_dictionary(taille, échantillons, k=…, d=8)` (fastcover
   à paramètres fixes ; l'optimisation automatique de k/d est trop lente sur 46 Mo).
5. klauspost : petit programme Go (`go.mod` avec `replace` vers ce dépôt) chargeant les
   100 000 JSON, `zstd.NewWriter` avec `WithEncoderLevel(SpeedDefault |
   SpeedBetterCompression | EncoderLevelFromZstd(19))`, `WithEncoderDict`,
   `WithEncoderConcurrency(1)` ; niveau 19 mesuré sur 5 000 blobs (son coût).
6. Schéma : base d'échantillon `ATTACH 'file:…?mode=ro&immutable=1' AS src` puis
   `INSERT … SELECT … WHERE id % 50 = 0` dans le DDL 2.28 extrait de `.schema` ; DDL 2.30
   rejoué à la main (index de `prunedIndexes2_30` supprimés, colonnes et index de
   `schema_sqlite.go` ajoutés, `match_date` dérivé par jointure move → game → match,
   `creation_date` synthétique au format `time.DateTime`) ; chaque variante = script SQL +
   `VACUUM` + somme `dbstat` hors match/game.
