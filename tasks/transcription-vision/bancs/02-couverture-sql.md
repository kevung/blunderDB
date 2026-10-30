# Banc V0.2 — couverture des fenêtres en SQL (`roll_tick_ms` / `tick_ms`)

Question : l'analyse opportuniste à 1× suffit-elle comme chemin principal (ADR-0109 règle 1), ou
faut-il la pré-passe hiérarchique partout ? Les Repères valent-ils comme fenêtres
d'apprentissage faible (ADR-0108 règle 6) ?

Requête : [`02-couverture.sql`](02-couverture.sql), réutilisable sur toute base (lecture seule ;
seules des vues `TEMP` sont créées). Largeur de fenêtre et seuils dans la table `param`.

```bash
sqlite3 -readonly -header -markdown BASE.db < tasks/transcription-vision/bancs/02-couverture.sql
```

Base mesurée : `~/dwn/unger.db`, ouverte en lecture seule, non copiée. Seuls 3 matchs sur 286
portent des Repères : **284** (James MacNaughtan), **285** (Nicolas Sekri), **286** (Kamel
Guerbaa), soit 1061 coups à jet et 52 gestes de videau (1113 Actions). Le sens des Repères est
celui de l'ADR-0082 règles 1-2 : un coup à jet (`dice_1 > 0`) a un jet `r` et une action `t` ; un
double, une prise, une passe n'ont que `t`. L'Action précédente est celle qui précède dans la
partie, quel que soit son camp ; le premier jet d'une partie n'a pas d'action précédente et est
exclu des écarts action → jet (27 jets).

## Critère du verdict (fixé avant la mesure)

Vert si, avec des fenêtres denses [−1 ; +2] s (ADR-0109) posées sur chaque Repère, **l'union des
fenêtres couvre au moins 50 % de l'étendue annotée** (la pré-passe économiserait moins de la
moitié de l'analyse dense) **et** si au moins 70 % des jets suivent l'action précédente de
moins de 10 s. Sinon rouge.

## 1. Écarts jet → action et action → jet suivant (secondes, à rebours exclus)

| match |         ecart         |  n  | mediane_s | p10_s | p90_s | max_s |
|-------|-----------------------|-----|-----------|-------|-------|-------|
| 284   | jet -> action         | 129 |       4.7 |   0.6 |  21.2 |  66.2 |
| 284   | action -> jet suivant | 124 |       4.0 |   1.6 |   6.9 |  18.1 |
| 285   | jet -> action         | 311 |       4.6 |   0.0 |  18.6 |  84.5 |
| 285   | action -> jet suivant | 295 |       4.6 |   0.8 |   9.4 |  61.3 |
| 286   | jet -> action         | 348 |       4.2 |   0.2 |  13.6 |  54.8 |
| 286   | action -> jet suivant | 334 |       3.9 |   1.0 |   7.0 |  62.8 |
| TOUS  | jet -> action         | 788 |       4.4 |   0.0 |  16.8 |  84.5 |
| TOUS  | action -> jet suivant | 753 |       4.2 |   1.2 |   7.8 |  62.8 |

Un coup de pions dure une médiane de 4,4 s du jet à l'action (p90 16,8 s) ; le jet suivant tombe
4,2 s après l'action (p90 7,8 s). Les trois matchs sont voisins.

## 2. Part des jets selon le délai après l'action précédente

| match | jets | inf_3s | de_3_a_10s | sup_10s | a_rebours | inconnu | pct_inf_3s_mesures | pct_inf_10s_mesures |
|-------|------|--------|------------|---------|-----------|---------|--------------------|---------------------|
| 284   |  235 |     36 |         84 |       4 |         1 |     110 |               29.0 |                96.8 |
| 285   |  373 |     73 |        198 |      24 |         8 |      70 |               24.7 |                91.9 |
| 286   |  426 |    102 |        217 |      15 |         6 |      86 |               30.5 |                95.5 |
| TOUS  | 1034 |    211 |        499 |      43 |        15 |     266 |               28.0 |                94.3 |

Sur les jets mesurables, 28 % suivent l'action de moins de 3 s, 66 % de 3 à 10 s, 6 % de plus de
10 s (94 % au plus 10 s). 266 jets sur 1034 (26 %) sont inconnus : leur action précédente n'a pas
de Repère.

## 3. Repères manquants et à rebours

| match | actions | coups_a_jet | deux_reperes | jet_sans_action | action_sans_jet | aucun | cubes | cube_sans_t | rebours_ja | rebours_aj |
|-------|---------|-------------|--------------|-----------------|-----------------|-------|-------|-------------|------------|------------|
| 284   |     256 |         242 |          129 |             113 |               0 |     0 |    14 |           0 |          0 |          1 |
| 285   |     401 |         383 |          312 |              71 |               0 |     0 |    18 |           0 |          1 |          8 |
| 286   |     456 |         436 |          348 |              88 |               0 |     0 |    20 |           0 |          0 |          6 |
| TOUS  |    1113 |        1061 |          789 |             272 |               0 |     0 |    52 |           0 |          1 |         15 |

L'action manque sur 272 des 1061 coups à jet (26 %), et nulle part le jet : le transcripteur pose
le jet systématiquement, l'action seulement sur validation explicite (ADR-0082 règle 3). Aucun
geste de videau n'est sans instant. À rebours : 1 coup (action avant son jet) et 15 jets avant
l'action précédente, soit 1,5 % des paires mesurables.

## 4. Largeur des Repères comme fenêtres

La fenêtre du jet va de l'action précédente au jet (les dés se posent, sont lus, puis le jet est
tapé : l'événement est dans cet intervalle) ; la fenêtre de l'action va du jet à l'action.

| match |       fenetre       |  n  | moyenne_s | pct_inf_3s | pct_inf_10s | pct_sup_30s |
|-------|---------------------|-----|-----------|------------|-------------|-------------|
| 284   | fenetre de l'action | 129 |       8.8 |       27.9 |        72.1 |         5.4 |
| 284   | fenetre du jet      | 124 |       4.3 |       29.0 |        96.8 |         0.0 |
| 285   | fenetre de l'action | 311 |       8.1 |       37.3 |        77.2 |         6.1 |
| 285   | fenetre du jet      | 295 |       5.5 |       24.7 |        91.9 |         1.0 |
| 286   | fenetre de l'action | 348 |       6.2 |       34.5 |        83.3 |         1.7 |
| 286   | fenetre du jet      | 334 |       4.5 |       30.5 |        95.5 |         0.3 |
| TOUS  | fenetre de l'action | 788 |       7.4 |       34.5 |        79.1 |         4.1 |
| TOUS  | fenetre du jet      | 753 |       4.8 |       28.0 |        94.3 |         0.5 |

Médiane de 4,2 s pour la fenêtre du jet (p90 7,8 s, 0,5 % au-delà de 30 s) et de 4,4 s pour celle
de l'action (p90 16,8 s, 4,1 % au-delà de 30 s). Une fenêtre de [−1 ; +2] s (3 s) est plus étroite
que la fenêtre médiane : centrée sur le Repère, elle ne contient pas l'événement à coup sûr ; les
fenêtres entre Repères le contiennent, au prix d'une queue côté action (le geste de validation
traîne).

## 5. Couverture de l'étendue annotée par des fenêtres denses [−1 ; +2] s

| match | reperes | etendue_min | fenetres_min | couverture_pct |
|-------|---------|-------------|--------------|----------------|
| 284   |     369 |        73.1 |         17.1 |           23.4 |
| 285   |     623 |        96.8 |         28.9 |           29.9 |
| 286   |     744 |        80.8 |         33.9 |           41.9 |
| TOUS  |    1736 |       250.8 |         79.9 |           31.9 |

Sensibilité (même requête, `param` modifié) : ±5 s (synthèse §03) couvre 66,2 % ;
[−0,5 ; +1] s couvre 16,7 %. Les Repères présents sont 1736 pour 2174 attendus (80 %) : les
actions manquantes sous-estiment la couverture, de quelques points.

## Verdict : rouge

Sur le critère fixé plus haut, la couverture [−1 ; +2] s est de **31,9 %** (23-42 % selon le
match), sous les 50 % : la pré-passe hiérarchique économiserait environ les deux tiers de
l'analyse dense, et l'analyse opportuniste seule n'est pas justifiée par la couverture. La seconde
condition est satisfaite (94 % des jets à 10 s ou moins de l'action précédente) mais ne suffit
pas : elle dit que les événements sont rapprochés, non que les fenêtres couvrent la vidéo.
**ADR-0109 règle 1 à réviser** : la pré-passe hiérarchique (1–2 img/s) reste utile partout, et
l'opportuniste à 1× en complément, non l'inverse. Le verdict est sensible à la largeur : à ±5 s il
bascule vert (66 %), si bien que la largeur de fenêtre que la détection exige en pratique (exp. 1
de la synthèse §03, pas 0,5 s) décide, et doit se mesurer sur les images avant de réviser.

Limites : trois matchs d'un seul transcripteur, une seule caméra ; l'étendue ignore les trous de
plus de 10 min ; les Repères tapés retardent l'événement réel d'un délai inconnu, ce qui décale
les fenêtres centrées sur le Repère.
