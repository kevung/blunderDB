# Livre des 2nd rolls par score — étude de faisabilité (#628)

Étude seulement : aucun code produit. Question posée : peut-on bâtir, avec une précision
suffisante, un livre des réponses au premier coup (2nd roll) selon le score de match, et à
quel coût ? Les mesures ci-dessous ont été faites sur la machine `melbaa` (28 fils, chargée
par d'autres sessions : charge 55 à 80 pendant les mesures). Les temps sont donc donnés en
**secondes CPU** (utilisateur + système), plus stables que le temps mural sur une machine
partagée, et pessimistes par rapport à une machine dédiée.

## Résumé

| Question | Réponse |
|---|---|
| Volume | 315 positions (15 ouvertures × 21 réponses) × ~20 scores × 4 candidats ≈ **25 000 rollouts de candidat** |
| Précision | blunderDB *Standard* rejoint gnubg au même réglage : écart moyen de **0,009** sur l'écart entre deux coups (max 0,019). Le réglage 0-ply tronqué lui-même s'écarte de **0,006** en moyenne (max 0,015) d'un rollout de référence 2-ply complet |
| Coût | **245 s CPU** par candidat (1296 parties), identique en money et à un score. Livre complet ≈ **46 jours CPU**, soit ~2 jours muraux sur 28 fils dédiés |
| Licence | Aucune source externe ne couvre les scores. Les seules tables publiées (bkgm.com) sont en money cubeless, sans licence déclarée : **ne pas les embarquer**. Des données produites par nous (blunderDB ou gnubg) nous appartiennent |
| Présentation | Écart au meilleur coup ± IC 95 % de la différence. **Égalité** déclarée sous ~0,01, ce qui concerne environ un quart des positions en money |

**Recommandation** : produire le livre avec `engine/rollout` au réglage *Standard*
(cubeful, équité normalisée au score), arrêt JSD ≥ 3 conservé. Déclarer « à égalité » tout
coup dont l'écart au meilleur est inférieur à **1,96 × σ(différence) + 0,005**, soit environ
0,010–0,013 avec 1296 parties. Ne pas viser une précision plus fine que ces 0,005 de biais du
réglage 0-ply : un rollout 2-ply coûte 7 fois plus pour un gain non mesurable ici. Budget :
~46 jours CPU pour 20 scores et 4 candidats, à lancer par lots reprenables (une ligne JSONL
par position et par score).

## 1. Volume

- **Positions** : 15 ouvertures (le meilleur coup de chaque lancer non double) × 21 réponses =
  **315 positions**. Les pages de bkgm.com déroulent aussi les ouvertures alternatives
  (32 coups d'ouverture, 672 positions) ; un livre qui les couvrirait double le volume.
- **Candidats** : en money, 1,28 coup en moyenne se tient à moins de 0,05 du meilleur (voir la
  distribution ci-dessous). Au score, le meilleur coup change (gammons, Crawford). Prendre
  l'**union des 3 meilleurs coups aux scores voisins** donne ~4 candidats par position.

  | Coups à moins de 0,05 (money, 672 positions) | 1 | 2 | 3 | 4 | 5 | 6+ |
  |---|---|---|---|---|---|---|
  | Positions | 222 | 170 | 112 | 78 | 53 | 37 |

- **Scores** : la grille complète 2-away à 7-away de chaque côté compte 36 scores, plus
  12 scores Crawford (1-away contre N-away, dans les deux sens), soit 48. La « vingtaine »
  de l'issue (par exemple les paires de {2, 3, 4, 5, 7}, plus quelques Crawford et la money)
  ramène ce nombre à ~20.
- **Total** : 315 × 20 × 4 ≈ **25 200 rollouts de candidat** (60 500 avec 48 scores).

## 2. Précision de `engine/rollout`

### Protocole

**Référence.** Les pages « Replies » de bkgm.com (`bkgm.com/openings/rollouts.html`)
publient, pour chaque ouverture et chaque réponse, les coups à moins de 0,05 du meilleur. Ce
sont des rollouts gnubg 0.14, 2-ply, 46 656 parties, non tronqués, cubeless money, avec
réduction de variance ; l'erreur-type annoncée est inférieure à 0,001. C'est la seule
référence publiée et chiffrée que j'aie trouvée. Il n'en existe aucune par score, ni aucune
cubeful.

**Échantillon.** 8 positions tirées au hasard (graine 628) parmi celles qui ont au moins
trois coups dans la table, stratifiées par l'écart entre les deux premiers coups :

- 3 positions à écart ≤ 0,006 ;
- 3 positions entre 0,008 et 0,02 ;
- 2 positions entre 0,025 et 0,05.

Les 3 meilleurs coups de la référence sont déroulés, soit 16 écarts mesurés.

**Comparaisons.** Les quatre séries jouent 1296 parties, troncature 11, réduction de
variance et dés quasi-aléatoires :

- **B** : blunderDB `rollout --preset standard --jsd 0` (0-ply, cubeful, sans arrêt
  anticipé).
- **Gcf** : gnubg 1.08.003 au même réglage, cubeful.
- **Gcl** : gnubg 1.08.003 au même réglage, cubeless.

Comparer Gcl à la référence isole l'erreur du *réglage* (0-ply tronqué contre 2-ply complet).
Comparer Gcf à Gcl isole l'effet du *videau*. Comparer B à Gcf isole le *moteur*, à recette
égale.

### Résultats

Écart du meilleur coup de la référence aux deux suivants (équité du meilleur − équité du
coup ; un signe négatif veut dire que l'ordre est inversé) :

| Position (ouverture, réponse) | Réf. | B | Gcf | Gcl |
|---|---|---|---|---|
| 6-2 13/5, 5-1 | +0,003 / +0,025 | +0,013 / +0,061 | +0,007 / +0,042 | +0,009 / +0,027 |
| 4-1 13/9 6/5, 5-1 | +0,001 / +0,042 | −0,007 / +0,068 | −0,006 / +0,055 | −0,005 / +0,041 |
| 6-5 24/13, 5-2 | +0,002 / +0,035 | +0,009 / +0,062 | −0,005 / +0,065 | +0,000 / +0,040 |
| 5-4 24/20 13/8, 4-1 | +0,009 / +0,026 | +0,018 / +0,064 | +0,032 / +0,071 | +0,018 / +0,041 |
| 6-4 24/18 13/9, 2-1 | +0,009 / +0,046 | +0,008 / +0,076 | +0,002 / +0,073 | +0,002 / +0,048 |
| 3-2 13/11 13/10, 5-1 | +0,009 / +0,019 | −0,004 / +0,024 | +0,012 / +0,030 | +0,007 / +0,022 |
| 5-2 13/8 6/4, 6-3 | +0,037 / +0,045 | +0,046 / +0,066 | +0,045 / +0,052 | +0,026 / +0,033 |
| 4-1 24/20 6/5, 5-4 | +0,039 / +0,049 | +0,045 / +0,053 | +0,051 / +0,065 | +0,036 / +0,044 |

| Comparaison (16 écarts) | Biais moyen | Écart absolu moyen | Max |
|---|---|---|---|
| Gcl − réf. (réglage 0-ply tronqué) | −0,000 | 0,006 | 0,015 |
| Gcf − Gcl (effet du videau) | +0,013 | 0,014 | 0,030 |
| **B − Gcf (moteur, même recette)** | **+0,001** | **0,009** | **0,019** |
| B − réf. | +0,013 | 0,016 | 0,038 |

**Meilleur coup identique à la référence** : B 6/8, Gcf 6/8, Gcl 7/8. Chaque désaccord
porte sur une position où la référence sépare les deux coups de ≤ 0,009.

**Erreur-type** : B donne σ = 0,002 à 0,003 par candidat, soit σ(différence) ≈ 0,004 en
sommant les variances. La JSD du paquet fait le même calcul, sans le terme de covariance des
dés communs, donc de façon conservatrice. L'IC 95 % d'un écart est d'environ ±0,008.

### Lecture

1. **Le videau change les écarts sans les fausser.** En money, les écarts cubeful valent
   1,56 fois les écarts cubeless (1,34 à 1,78 sur 11 écarts > 0,015). C'est attendu : le
   videau amplifie la valeur d'un avantage. B − réf. ne mesure donc pas une erreur, et le
   livre doit comparer cubeful à cubeful.
2. **À recette égale, blunderDB vaut gnubg.** Les écarts B et Gcf diffèrent de 0,009 en
   moyenne. Le bruit de deux rollouts indépendants en explique ~0,005 ; il reste ~0,005 à
   0,007 d'écart propre aux moteurs, dont on ne peut dire lequel a raison sans une référence
   cubeful plus profonde.
3. **Le plafond, c'est le réglage, pas le moteur.** Même gnubg en 0-ply tronqué s'écarte de
   0,006 en moyenne (max 0,015) d'un rollout 2-ply complet. Aucun réglage *Standard*,
   blunderDB ou gnubg, ne départage deux coups à 0,005 ; un livre honnête les déclare à
   égalité.

L'issue demandait si `engine/rollout` tient la précision de gnubg : **oui, au même réglage**.
Il ne tient pas la précision d'un rollout gnubg 2-ply complet, mais gnubg 0-ply non plus.

## 3. Coût et licence

### Mesures (s CPU par candidat, 1296 parties, troncature 11)

| Moteur | s CPU par candidat | Remarque |
|---|---|---|
| blunderDB Standard (0-ply), money | 244 (8 positions) | 112 s murales pour 3 candidats sur la machine chargée |
| blunderDB Standard, 7 points, 3-away contre 5-away | 245 | Même coût au score |
| gnubg 1.08 0-ply cubeful / cubeless | 78 / 74 | 3,2 fois plus rapide |
| gnubg 1.08 2-ply cubeful | ~1 730 | Extrapolé de 36 parties (48 s CPU) ; 7 fois blunderDB Standard |

L'arrêt JSD ≥ 3 après 324 parties de *Standard* aurait coupé 10 des 16 candidats non
meilleurs de l'échantillon, d'où ~65 % du coût plein avec 4 candidats.

### Budget du livre (315 positions, 4 candidats)

| Réglage | 20 scores | 48 scores |
|---|---|---|
| blunderDB Standard, arrêt JSD | **~46 j CPU** (~1,7 j muraux sur 28 fils dédiés) | ~110 j CPU (~4 j) |
| blunderDB Standard, 1296 parties pleines | ~71 j CPU | ~170 j CPU |
| gnubg 0-ply, arrêt JSD | ~15 j CPU | ~36 j CPU |
| gnubg 2-ply, arrêt JSD | ~330 j CPU (~12 j muraux) | ~790 j CPU |

Sur la machine telle qu'elle était chargée pendant la mesure, compter environ trois fois le
temps mural (≈ 6 jours pour 20 scores en blunderDB). Re-dérouler en 1-ply les seules
positions serrées (écart < 0,02, ~38 % en money) coûterait ~40 j CPU de plus (ADR-0060 :
1-ply ≈ 3 fois le coût). Le gain attendu (≤ 0,004 d'après ADR-0060) est inférieur à la marge
d'égalité.

**gnubg plutôt que blunderDB ?** 3 fois plus rapide, pour une précision équivalente. En
revanche, gnubg 1.08 ne déroule pas un coup nommé en ligne de commande (`rollout` refuse tout
argument). Il faut dérouler la position *après* le coup, adversaire au trait, sous un
pseudo-terminal (`script`), sans quoi gnubg n'imprime pas le résultat. Le livre ne porterait
alors pas une signature de rollout blunderDB (ADR-0060), et l'équité au score devrait être
convertie. Cette économie ne vaut pas une chaîne externe à maintenir.

### Licence

- **bkgm.com** : aucune licence déclarée, donc tous droits réservés par défaut. Les tables
  sont de surcroît en money cubeless et calculées avec gnubg 0.14. Inutilisables pour le livre
  par score : à citer comme référence de contrôle, pas à embarquer.
- **Aucune table par score n'est publiée.** Les ouvrages qui en traitent (Steg, *Understanding
  the Opening Replies* ; Ballard et Weaver, *Book A*) sont protégés et ne livrent pas une table
  exploitable. Les rollouts JellyFish de Johanni portent sur un seul score.
- **Données produites par nous** : la sortie de blunderDB (MIT) nous appartient. La sortie de
  gnubg (GPL-3) n'est pas couverte par la GPL, puisque ce n'est pas du code dérivé ; un livre
  calculé avec gnubg pourrait donc être embarqué, avec mention de l'outil.

## 4. Présentation

- **Par coup** : l'équité (money ou normalisée au score, ADR-0019), l'écart au meilleur coup
  et l'IC 95 % *de l'écart*, plus le nombre de parties et la signature du rollout. Afficher
  un IC par coup trompe : deux coups aux intervalles disjoints peuvent rester à égalité une
  fois le biais du réglage compté, et des intervalles qui se chevauchent n'empêchent pas une
  différence nette quand les dés communs corrèlent les deux coups.
- **Règle d'égalité** : le coup *c* est à égalité avec le meilleur si
  `écart(c) < 1,96 · σ(différence) + 0,005`. Le 0,005 est le biais mesuré du réglage 0-ply
  tronqué (§ 2) ; avec 1296 parties, le seuil vaut ~0,010 à 0,013. Le livre affiche alors
  « à égalité » au lieu d'un vainqueur. En money, cela concerne ~23 % des positions (écart de
  référence ≤ 0,010) et ~30 % jusqu'à 0,015.
- **Exercice « quel coup à ce score ? »** : tout coup du groupe à égalité en tête est une
  bonne réponse. La correction montre ce groupe et ce qui le sépare du coup suivant.
- **Stockage** : une ligne par (ouverture, réponse, score) avec, pour chaque candidat,
  l'équité, σ, le nombre de parties, la JSD et la signature. La table est une donnée
  embarquée, reproductible par sa graine.

## Risques relevés pendant l'étude

- **`engine.CanonicalMove` confond deux coups qui ne laissent pas le même plateau.** Elle
  retire les marques de prise : `6/5* 5/1*` et `6/2 2/1*` (4-1) donnent tous deux `6/1`. Le
  commentaire affirme pourtant que deux coups de même forme canonique laissent le même
  plateau. Les appels concernés sont `rollout.moveBranches`, où `byCanonical` garde le dernier
  arrivé, `analysiscodec.go:767`, `explain.go:212`, `gammonnet/compare.go` et
  `apkg/cards.go`. Un coup importé en dialecte gnubg ou XG (`6/5*/1*`) peut ainsi être résolu
  vers un autre coup. Dans cette étude, 4 des 24 coups de l'échantillon tombaient dans ce
  cas ; ils ont été redéroulés avec la notation exacte de blunderDB. Le livre devra nommer
  ses coups dans cette notation exacte.
- Les rollouts blunderDB sont cubeful avec `CubefulBias` : le classement est fiable,
  l'équité absolue moins. Le livre doit montrer des écarts, pas des équités absolues.

## Reproduire

Scripts jetables, hors dépôt :

1. Analyser les pages `r??-*-data.html` de bkgm.com.
2. Construire les XGID des positions de réponse avec `domain.LegalMoves` et
   `domain.EncodeXGID`.
3. `blunderdb rollout --preset standard --jsd 0 --format json --move … <XGID>`.
4. gnubg : `set xgid <position après le coup>`, `set rollout …`, puis `rollout`, sous
   `script -qec`.

Les mesures de l'étude ont pris ~35 min de temps mural (~4 h CPU).
