# Poste 1 — valuer le videau par lot sur les candidats

**Verdict : le lot est réfuté ici.** Il est exact au bit près, il tient les deux
dispositifs d'exactitude que la spécification impose, et il rend une décision 2-ply au
score **12,3 % plus lente**. Ce qui est livré à sa place est une **levée** — sortir des
soixante pas de bissection tout ce qui n'y dépend pas de `p` — qui rend **16,5 %** sur
la même décision.

C'est le cas d'école de l'ADR-0003 lu à l'envers, et il était prévu par elle : le gain
d'une optimisation conceptuelle ne survit pas forcément au changement de langage, comme
la sparsité de la couche 1 rend 16 % en C et 6 % en Go.

---

## 1. La mesure d'entrée, prise ici

`TestMeasureCubeShare` — pour chaque décision, la même position valuée sans puis avec le
videau, alternée, médiane sur douze cas × trois répétitions.

| | avant cette verticale | après |
|---|---|---|
| part du videau dans une décision 2-ply au score | **38,7 %** | 33,0 % |
| ns par valuation (dénominateur **compté**, pas supposé) | **3 849** | 3 154 |
| part du videau en argent | nulle par construction (forme fermée, aucune chaîne d'enjeux) | idem |

Le dénominateur est compté par `Searcher.CubeValuations()`, ajouté ici pour cette
verticale sur le modèle de `gn_search_cube_valuations()` : il s'incrémente là où
`nodeValue` appelle réellement `Value`, donc jamais sur un nœud terminal ni sous
`UseCube` éteint. Amont avait découvert que sa mesure d'entrée supposait « un nœud évalué
porte une valuation », ce qui est faux dans trois sens à la fois.

**Le poste est réel et il vaut d'être attaqué : 38,7 % ici contre 20,5 % en amont.** Le
videau pèse relativement plus lourd dans ce portage parce que son noyau réseau, lui, est
déjà vectorisé (ADR-0024).

## 2. Ce qui a été écrit, et ce qui a été mesuré

Le portage suit la spécification amont point par point
(`docs/specs/t34-videau-spec.md` §7.1) : `buildLevels` coupé en `buildLevelAnchors` (par
candidat) et `resolveLevels` (par niveau) ; un `cubeValueBatch` menant les soixante pas
de toutes les voies en pas cadencé ; **largeur de voie fixe** et **nombre d'itérations
fixe** ; les consultations `metAfter` **non** remontées (le piège nommé, déjà mesuré à
1 % et annulé ici en #150) ; le chemin argent scalaire.

Les quatre preuves d'exactitude sont vertes et sans tolérance :

| preuve | portée | résultat |
|---|---|---|
| `TestCubeBatchMatchesScalar` | 141 distributions **réelles** (sorties brutes du grand réseau) × 3 possessions × 7 états | `==`, pas `approx` |
| `TestCubeBatchSplitInvariance` | les mêmes, en un lot / coupées à 37 / une par une | bits identiques |
| `TestLiftedSolveMatchesUnlifted` | la levée contre la bissection d'avant | bits identiques |
| suites d'or, parité, identité de noyau, `TestParallelSearchIsBitIdentical` | inchangées | vertes |

### Le poste isolé, entrelacé, épinglé

`TestMeasureCubePost` : trois valuations de la même fratrie chronométrées dos à dos,
ordre tournant, 400 paires, médiane et minimum. Trois exécutions consécutives se
reproduisent à ~2 %.

|  voies | avant ns/val | levé ns/val | lot ns/val | levée × | lot vs levé × | lot vs avant × |
|---:|---:|---:|---:|---:|---:|---:|
| 2 | 2 309 | 1 528 | 2 525 | **1,51** | 0,61 | 0,91 |
| 4 | 2 272 | 1 515 | 2 422 | **1,50** | 0,63 | 0,94 |
| 8 | 2 408 | 1 624 | 2 472 | **1,48** | 0,66 | 0,97 |
| 12 | 2 649 | 1 899 | 2 599 | **1,40** | 0,73 | 1,02 |
| 16 | 3 181 | 2 435 | 2 986 | **1,31** | 0,82 | 1,07 |
| 24 | 3 961 | 3 076 | 4 048 | **1,29** | 0,76 | 0,98 |
| 32 | 4 408 | 3 395 | 4 438 | **1,30** | 0,77 | 0,99 |
| 48 | 4 235 | 3 415 | 4 349 | **1,24** | 0,79 | 0,97 |
| 64 | 4 617 | 3 670 | 4 763 | **1,26** | 0,77 | 0,97 |

### La décision entière

`TestMeasureCubeLift` et l'A/B lot allumé / éteint, douze cas, cinq répétitions,
entrelacés décision par décision, épinglés :

| | rapport médian | lecture |
|---|---|---|
| levée | **0,835** | **16,5 % de moins** sur une décision 2-ply au score (douze cas de 0,48 à 0,92) |
| lot | **1,123** | **12,3 % de PLUS** (douze cas de 1,07 à 1,23 — aucun ne va dans l'autre sens) |

## 3. Pourquoi le lot perd ici alors qu'il gagne là-bas

En C, `level_live` est inlinée dans la bissection : un pas est une chaîne courte bornée
par la **latence** d'une division, et mener douze voies de front la recouvre. D'où le
×2,43 d'amont.

En Go, `levelLive` n'est **pas** inlinable (`go build -gcflags=-m` : *cannot inline
levelLive*, trop complexe). Chaque pas payait donc un appel non inliné, un `switch` sur
la possession, le test « niveau mort » et les deux soustractions de segment — soixante
fois par point de rupture, six points de rupture par valuation. **Sortir tout cela des
soixante pas suffit à rendre le pas assez court pour que le prédicteur et la fenêtre
d'exécution dans le désordre le couvrent déjà.** La bissection cesse d'être bornée par
la latence, donc il ne reste plus rien à recouvrir.

Ce qui reste au lot, ce sont ses coûts : l'état des voies vit en mémoire au lieu de
vivre en registres, et sa forme sans branche — celle que la spécification impose, à
raison, pour que l'erreur de prédiction ne vide pas le pipeline de toutes les voies en
vol — paye **une division de plus par pas**.

Le détail qui le montre : la levée mesurée avec une forme **sans branche** ne rend que
×0,8 à ×0,9 (elle est plus lente que l'original), et le lot y bat alors le scalaire de
×1,3. C'est exactement la même mesure lue à travers deux formes différentes du même
calcul, et c'est ce qui a failli faire créditer l'amont d'un gain qui appartenait au
compilateur. Les deux formes coexistent donc, une par site : `laneCurve.at` (branchue,
une division) pour la bissection sérielle, `laneCurve.atSelect` (sans branche, deux
divisions) pour le pas cadencé.

## 4. Ce qui est livré, et ce qui ne l'est pas

**Livré** (`cube.go`) :

- `buildLevelAnchors` / `resolveLevels` — la coupe qu'impose la spécification amont. Elle
  ne coûte rien et elle est la bonne forme, que le lot vive ou non.
- `laneCurve` + `laneCurve.at` + la levée dans `levelSolve`. **Optimisation
  d'implémentation**, propre à ce portage : elle n'a rien à faire remonter en amont, où
  gcc la fait déjà.
- `Searcher.CubeValuations()` — le dénominateur compté.
- Les instruments `TestMeasureCubeShare`, `TestMeasureCubePost`, `TestMeasureCubeLift`.

**Non livré, mais gardé en fichier de test** (`cube_batch_experiment_test.go`) : le
portage complet du lot, avec ses tests d'exactitude et sa mesure. Il n'est pas supprimé
parce que **le verdict « ne paye pas » est un verdict de machine autant que de langage** :
l'ADR-0003 demande que chaque consommateur remesure, et quelqu'un sur un autre processeur
doit pouvoir rejouer l'expérience sans la réécrire.

## 5. À faire remonter en amont

L'ADR-0003 dit que **les mesures remontent toujours, y compris celles qui réfutent**.
Trois choses à consigner là-bas :

1. Le portage Go a mesuré le lot à **×0,89 sur la décision entière** là où le C mesure
   ×1,13. La spécification §7.1 annonce que la forme « se transportera telle quelle en
   Go » ; c'est faux, et la raison est nommée (inlining de `level_live`).
2. La levée des constantes de segment hors de la bissection rend **16,5 %** ici. En C
   elle est probablement nulle (gcc la fait), mais **elle n'a jamais été mesurée là-bas**,
   et le §2 de T85 attribue les 2,7 µs à la latence de division seule.
3. La forme sans branche est un **gain sous lot et une perte sous scalaire**. Le C
   n'applique le `select` qu'au pas cadencé, ce qui est le bon choix ; il vaut la peine
   d'écrire pourquoi, parce qu'ici l'appliquer aussi au scalaire coûtait 10 à 20 %.
