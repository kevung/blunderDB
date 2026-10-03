# Mesures sur le corpus BMAB — GB0.2

Mesuré le 2026-10-03 sur le poste du plan (Ryzen 7 PRO 6850U, 16 threads, 14 Go, NVMe).
Corpus : l'archive BMAB du 2025-06-23 (un `.zip` de 22 Go), lue en place. Rien du corpus
n'entre dans le dépôt : ni fichier, ni nom de joueur ou d'événement — seulement des comptes.

## 1. Le corpus entier, depuis l'index de l'archive

Lu dans le répertoire central du `.zip` (CRC-32 et taille de chaque membre), sans rien
extraire ; le regroupement par (CRC-32, taille) a été vérifié par SHA-256 sur 20 groupes tirés
au hasard : 20 sur 20 identiques octet pour octet.

| Grandeur | Valeur |
|---|---|
| Fichiers `.xg` | 166 713 (24,7 Go décompressés) |
| Taille d'un fichier | médiane 142 Ko, moyenne 148 Ko |
| Contenus distincts | **33 370** |
| Fichiers en double | **133 343, soit 80,0 %** |
| Structure des doublons | 33 315 groupes de **5 copies** exactement (plus 28 de 3 et 27 de 2) |

Le corpus est rangé en cinq dossiers régionaux, et chaque match figure dans les cinq, sous le
même numéro et avec le même contenu. Le corpus réel, c'est 33 370 matchs et ≈ 4,9 Go, pas
190 000 fichiers et 21 Go.

## 2. L'échantillon

`scripts/sample-corpus.sh bmab-2025-06-23.zip 2000 <sortie>` (graine 518 par défaut) : 2 000
fichiers tirés parmi 166 713, 289 Mo, 35 s depuis l'archive. 42 des 2 000 sont des copies
SHA-256 d'un autre fichier de l'échantillon, conformément au facteur 5 ci-dessus.

## 3. Import à la CLI, avant le lot GB1 (`main` @ 8aaafec12)

`blunderdb import --type batch` sur l'échantillon, base neuve. L'import a été arrêté par la
limite de 2 h de la tâche de fond après 1 237 fichiers ; les chiffres portent sur ces 1 237.

| Grandeur | Valeur |
|---|---|
| Fichiers traités | 1 237 en 7 196 s : 1 220 importés, 16 refusés comme doublons (`match_hash`), 1 en erreur |
| Positions écrites | 608 785 (≈ 499 par match, pas 380) |
| **Débit** | **84,6 positions/s** sur l'ensemble, ≈ 5,9 s par fichier, mais le poste était partagé (génération de la base synthétique, benchmarks, tests) ; **143 positions/s** machine au repos, sur 50 fichiers de plus importés dans la même base pleine (24 833 positions en 173 s) |
| Taille de base | 470,8 Mo pour 592 525 positions : **795 o/position** — `analysis` 320 o, `position` 138 o, `move` 42 o, index 292 o (37 %) |
| Pic RSS | **566 Mo** pour le lot de 50 fichiers sur la base de 600 000 positions (313 Mo sur base neuve, README § 1) |
| Fichiers refusés | 1,3 % de doublons exacts ; 0,08 % d'erreurs (1 fichier : « xg get file segments: seek … invalid argument », fichier tronqué ou segment mal formé) |

## 3 bis. Import à la CLI, après le lot GB1 (`main` @ 4acd39446, codec du chantier GB-C2)

Même échantillon, même commande, base neuve, poste au repos ; les 2 000 fichiers en entier.

| Grandeur | Avant (§ 3) | Après |
|---|---|---|
| Fichiers | 1 237 traités en 2 h (arrêt) | **2 000 en 1 433 s** : 1 957 importés, 42 doublons exacts (les 42 copies SHA-256 du § 2), 1 erreur (le même fichier) |
| Positions écrites | 608 785 | 979 035, 948 681 distinctes (dédup Zobrist 3,1 %) |
| **Débit** | 84 positions/s (partagé), 143 (repos) | **683 positions/s**, ≈ 0,7 s par fichier : ×4,8 sur le chiffre au repos |
| Taille | 795 o/position | 820 o/position — `analysis` 345 o (codec plus rapide, un peu moins dense), index 293 o |
| Pic RSS | 566 Mo (50 fichiers sur base pleine) | 458 Mo sur tout l'import |

À ce débit, les 33 370 matchs distincts du corpus (≈ 16,6 M de positions) s'importent en
**≈ 7 h** sur un cœur, contre 32 à 55 h avant le lot 1.

## 4. Doublons au-delà du fichier

| Niveau | Mesure |
|---|---|
| Fichiers identiques (SHA-256) | 80,0 % du corpus (§ 1) ; l'import les écarte déjà par `match_hash` en 0,06 s chacun |
| Même match sous un autre nom de fichier (`match_hash`) | confondu avec le précédent : les 16 refus de l'échantillon sont tous des copies régionales |
| Même match sous une autre graphie (longueur + suite des dés et du videau) | **0** parmi les 1 220 matchs importés |
| Positions dédupliquées par Zobrist entre matchs distincts | 608 785 coups pour 592 525 positions : **2,7 %** ; 5 411 positions (0,9 %) apparaissent dans au moins deux matchs et portent 3,5 % des coups — des ouvertures, comme le plan le supposait |

## 5. Graphies et profondeurs

| Inventaire | Mesure |
|---|---|
| Noms de joueurs | 1 159 graphies ; une fois pliées (casse, accents, ponctuation, espaces) 1 118 ; **38 noms (3,3 %) ont au moins deux graphies** |
| Événements | 672 graphies, 631 pliées, **33 événements ont au moins deux graphies** ; 2 vides ou faits de symboles |
| Longueur de match | 7 pts 32 %, 9 pts 28 %, 11 pts 25 %, 13 et plus 15 % ; aucun money game |
| Dates | 2013 à 2025 ; 150 matchs sans événement |
| Profondeur, videau | 4-ply 64 %, 2-ply 29 %, XG Roller 7 %, le reste < 0,1 % |
| Profondeur, meilleur coup | 4-ply 74 %, 3-ply 14 %, XG Roller 10 %, Book 0,7 % |
| Étiquette inattendue | 17 analyses portent la profondeur « 100 », à examiner pour GB4.1 |

Pour GB5.2 : les alias de joueurs portent sur quelques pour cent des noms, le pliage simple
(casse, accents, ponctuation) en trouve la plupart ; un alias saisi à la main reste nécessaire
pour le reste. Pour GB5.3 : l'empreinte sans les noms n'a rien trouvé dans l'échantillon, ce
qui ne dit rien des corpus assemblés à la main. Pour GB4.1 et GB5.1 : quatre étiquettes
couvrent plus de 99 % des analyses ; une réanalyse plus profonde ne remplacera que les 2-ply et
XG Roller.

## 6. Hypothèses du plan corrigées

| Grandeur | Plan (§ 2 du README) | Mesuré, extrapolé à 33 370 matchs |
|---|---|---|
| Fichiers distincts | ≈ 190 000 | **33 370** |
| Positions brutes | ≈ 72 M | **≈ 16,6 M** (499 par match) |
| Positions uniques | 40 à 70 M | **≈ 16 M** (dédup Zobrist 2,7 % mesurée, un peu plus à l'échelle) |
| Durée d'import | 90 à 170 h | **32 à 55 h** avant le lot 1 (143 positions/s au repos, 84 sur poste partagé) ; **≈ 7 h** après (683 positions/s) |
| Taille de base | 30 à 55 Go | **≈ 13 Go**, dont ≈ 4,7 Go d'index |
