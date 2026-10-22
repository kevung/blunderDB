# Mesure : poids float16 (BGN6, gammonNet v1.8.1) contre float32, au score et en money

Refs #772.

## Conclusion

Arrondir les poids en demi-précision déplace les sorties du réseau d'environ 1e-4. Sur
11 000 comparaisons, aucune décision de videau ne change. Les choix de pions changent dans
12 cas sur environ 14 900 : tous sont des égalités ou des écarts inférieurs à 3,3e-4, pour
un coût mesuré en float32 d'au plus 3,25e-4 (au plus 1,2e-7 à 2-ply). Rien n'indique que la
demi-précision dégrade les décisions, ni au score ni en money.

## Méthode

Le code de sonde est **jetable et non versionné** : un lecteur BGN6 et quelques accès de test
dans `engine/gammonnet`, plus un programme de mesure, le tout dans un worktree hors dépôt
qui n'est pas fusionné.

- **Conversion BGN6 → float32.** Chaque demi est converti exactement, les sous-normaux sont
  renormalisés. La conversion est vérifiée sur les 63 488 demis finis : aucun écart.
  - Réseau principal (`strehl-prob5-512-512-256-256`) : 560 384 poids, dont 631 sous-normaux
    et 589 nuls ; aucun inf ni NaN.
  - Réseau d'élagage (`strehl-prune-32`) : 6 432 poids, dont 2 sous-normaux.
- **Parité avec les vecteurs de référence de gammonNet** (2 000 positions × 5 sorties, noyau
  scalaire et noyau par lots AVX) :
  - bin16 contre `reference16.bin` : max|Δ| = 5,96e-08. Le convertisseur et le noyau
    reproduisent donc le réseau f16 amont à l'arrondi float32 près.
  - f32 embarqué et `.bin` de la release contre `reference.bin` : 5,96e-08. Le
    `reference.bin` de la release est identique octet pour octet à
    `testdata/reference.bin`.
  - bin16 contre les sorties f32 : max|Δ| = 3,84e-4. C'est le coût réel de l'arrondi des
    poids.
- **Chemin de production.** Les deux réseaux passent par `EvaluatePositionWith` (génération
  des coups, MET, modèle de videau, réseau d'élagage), à 0-ply et à 2-ply. La mesure porte sur
  4 470 positions distinctes, issues d'un corpus de parties et de la base de démonstration.
- **Variantes de score.** Chaque position est évaluée à son score d'origine, à un score tiré
  au hasard dans une grille d'away (2..11 de chaque côté, plus Crawford et post-Crawford), et
  en money. Cela donne 13 410 tâches × 2 plies = 26 820 enregistrements.
- **Exclusions.** 1 704 enregistrements ne contiennent aucune décision (joueur bloqué sur la
  barre) et sont écartés.

## Résultats

| Périmètre | \|Δ\| d'équité (moyenne / p99 / max) | Décisions changées |
|---|---|---|
| Pions 0-ply, money (n=2 477) | 1,2e-4 / 4,3e-4 / 9,8e-4 | 0 |
| Pions 0-ply, score (n=4 954) | 1,1e-4 / 4,6e-4 / 1,0e-3 | 5 (0,10 %) |
| Pions 2-ply, money | 8,9e-5 / 3,7e-4 / 1,4e-3 | 2 (0,08 %) |
| Pions 2-ply, score | 8,9e-5 / 4,3e-4 / 1,9e-3 | 5 (0,10 %) |
| Videau 0/2-ply, money (n=1 395) | ND p99 ≈ 3,5–4,1e-4, DT max 2,3e-3 | 0 |
| Videau 0/2-ply, score (n=2 616) | ND max 2,6e-3, DT max 7,2e-3 | 0 |
| Prise 0/2-ply (n=61 money, 122 score) | ≤ 1,5e-3 | 0 |

- **Pions, 0-ply.** Les écarts entre les deux premiers coups valent de 0 à 1e-5, sauf un
  cas à 4-away/4-away (`bbBCCCB-AA----b----cb-b-b-:1:1:1:32`) : 6/3 6/4 devient 9/6 6/4,
  pour un coût de 3,2e-4.
- **Pions, 2-ply.** Les changements ne viennent que de deux positions, répétées sur
  plusieurs variantes de score (`--B--BE-A---A--AB--A---cb-:0:0:1:42` et
  `----AB--------------bb----:1:-1:1:53`). L'écart entre les deux premiers coups y est
  d'environ 1e-6, et l'équité affichée est identique à 1e-7 près.
- **Écart de 4e-2 / 9e-2 sur un coup quelconque.** Il s'agit d'un coup de rang inférieur que
  le filtre d'élagage a basculé près de son seuil : ce coup a été cherché moins profond dans
  une exécution que dans l'autre. Le coup retenu comme meilleur n'est jamais touché.
- **Sous-populations.** Crawford, post-Crawford, scores déséquilibrés (|diff| ≥ 3), 2-away
  impliqué, videau ≥ 2 : aucune ne se distingue. Leurs écarts sont ceux de l'ensemble.

## Balayage des décisions de videau serrées

- **Sélection.** Chaque décision de videau sans dés est croisée avec toute la grille de
  scores hors Crawford, videau centré ou à sa valeur d'origine : 278 880 candidats. Le réseau
  f32 à 0-ply en retient 34 120 situés à moins de 0,05 d'une frontière de décision ; 2 000
  d'entre eux sont évalués à 0-ply et à 2-ply.
- **Résultat.** 0 action changée sur 4 000.
- **Hors gains ou pertes quasi certains (|ND| < 0,95).** 1 308 décisions à moins de 0,05
  d'une frontière, 421 à moins de 0,01, 230 à moins de 0,002 : toujours 0 changement.

## Réserves

- **Peu de cas vraiment serrés.** Beaucoup des cas les plus proches sont sur la frontière
  par construction : ND = DT exactement (2-away, videau mort). L'échantillon de décisions
  serrées non dégénérées compte donc quelques centaines de cas, pas des milliers.
- **Mesure de distance approximative.** À 2-ply, deux enregistrements se sont déplacés de
  plus que leur distance à la frontière sans changer d'action. La mesure de distance
  employée (|ND − min(DT, DP)|, |DT − DP|) n'est pas exactement la règle d'action, qui gère à
  part les égalités et la zone « trop bon ».
- **Origine des positions.** Les positions viennent d'un seul corpus et de la base de
  démonstration. Les scores de la grille sont synthétiques et appliqués à des positions
  jouées à un autre score.
