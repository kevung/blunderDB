# Poste 4 — les deux mesures que personne n'avait séparées

Deux mesures, aucun code de moteur changé. La première a trouvé quelque chose,
la seconde a trouvé que la question ne se pose pas encore ici.

---

## 1. La sparsité, séparée par réseau

**Ce qu'on croyait.** `kernel.go` chiffrait la compaction des colonnes nulles à
**~6 %** globalement, et **−9 %** sur des plateaux sans rapport. Le chiffre était
juste et il était **agrégé** : il n'a jamais dit ce que chacun des deux réseaux y
mettait. gammonNet a fait la séparation (T89) et y a trouvé que son registre
écrivait la priorité à l'envers.

**L'instrument.** Un interrupteur de compaction **par évaluateur**
(`Evaluator.noSkipZeros`) — et par évaluateur, pas global, parce que c'est tout
l'objet du poste : un interrupteur global ne sépare rien. La même décision
2-ply au score est ensuite chronométrée sous les quatre combinaisons, entrelacées
dans le même processus, l'ordre tournant à chaque répétition. Les quatre
calculent les **mêmes bits** — la compaction est exacte en IEEE 754
(`acc + w×0,0 == acc`), ce que `kernel_identity_test.go` prouve — donc l'arbre
exploré est identique et l'A/B est propre.

**Le résultat**, douze cas × quatre répétitions, trois exécutions épinglées :

| compaction active sur | run 1 | run 2 | run 3 | médiane |
|---|---:|---:|---:|---:|
| aucun réseau (référence) | 0,0 % | 0,0 % | 0,0 % | — |
| **le grand seul** | **4,7 %** | **3,7 %** | **4,1 %** | **4,1 %** |
| **le petit seul** | −0,8 % | −0,3 % | +0,8 % | **−0,3 %** |
| les deux | 2,7 % | 3,8 % | 5,1 % | 3,8 % |

**Tout le gain appartient au grand réseau. La part du petit est indiscernable de
zéro** — les trois relevés l'encadrent. Le ~6 % agrégé n'était donc pas
« la compaction rend 6 % », c'était « la compaction du GRAND réseau rend 4 %, et
celle du petit rien ».

**Ce que ça change.** Rien à livrer : éteindre la compaction du petit réseau ne
rend rien de mesurable, et l'ADR-0003 comme la pratique de ce dépôt disent qu'un
poste sans gain mesurable n'est pas livré. Ce que ça change est la **priorité** :
tout effort futur sur la sparsité doit viser le premier layer du grand réseau
(512 sorties) et jamais celui du petit (32 sorties), dont l'économie est seize
fois plus petite pendant que la préparation coûte le même prix.

**Pourquoi le petit ne rend rien**, et c'est la même raison qu'en amont : il
porte **73 % des voies mais 9 % du temps**.

### La part de temps, mesurée et non supposée

`TestMeasureNetworkCost` chronomètre un lot plein à travers chaque réseau,
entrelacé, 2 000 paires — sur une **vraie fratrie** (les huit premiers coups
légaux d'une ouverture), pas huit plateaux quelconques dont l'union des entrées
actives est deux fois plus large.

| | ns par position dans un lot plein | évaluations d'une décision 2-ply | part de temps |
|---|---:|---:|---:|
| grand réseau (196→512→512→256→128→5) | **15 964** (min 13 875) | 12 883 (26,9 %) | **91,1 %** |
| petit réseau (196→32→5) | **572** (min 500) | 35 065 (**73,1 %**) | **8,9 %** |

Le rapport de coût est **×27,9**. Le dénominateur est **compté** (les compteurs
du chercheur), le numérateur **chronométré** : c'est un produit, mais un produit
dont les deux facteurs sont mesurés, ce que la mesure d'entrée d'amont n'était
pas.

**73 % des voies pour 9 % du temps** : la même forme qu'en amont (77 % / 5 %),
sur une machine et un langage différents. Optimiser le petit réseau vaut ici
**le dixième** d'optimiser le grand.

---

## 2. La tuile à huit accumulateurs — la question ne se pose pas encore ici

**Ce qu'annonce l'issue.** Le noyau d'ici plafonne à **six** accumulateurs, le
générateur avo manquant de registres généraux au-delà et attrapant BP, le
pointeur de trame — le noyau tombait une fois sur trois. gammonNet obtient
**huit accumulateurs vectoriels toujours**, arrangés en 8×1, 4×2, 2×4 ou 1×8
selon largeur et cible, en tuilant sur les **lignes de sortie**.

**Ce que la lecture du code amont établit, et qui n'avait pas été vu.** Les
quatre arrangements ne sont pas quatre allocations de registres différentes :
`src/gn_kernel_f32.h` les dérive mécaniquement de la largeur de lot.

```
GN_KERNEL_VECS = GN_EVAL_BATCH / GN_VEC_LANES
GN_KERNEL_ROWS = 8 / GN_KERNEL_VECS       (1 si VECS >= 8)
```

En AVX2, `GN_VEC_LANES` vaut 8. Amont, `GN_EVAL_BATCH` vaut **32**, donc
`VECS = 4` et `ROWS = 2` : l'arrangement retenu est **2×4**, et les huit
accumulateurs viennent des **quatre vecteurs par ligne**, pas d'un meilleur
allocateur.

Ici, `EvalBatchWidth` vaut **8**, soit exactement **un** registre YMM par ligne.
Donc `VECS = 1` et `ROWS = 8` : le seul arrangement disponible est **8×1** —
c'est-à-dire précisément celui qui manque de registres, huit pointeurs de ligne
vivants à la fois. **La technique d'amont ne transfère pas ; elle suppose
d'abord d'élargir le lot**, et la largeur de lot est « fixe et fait partie du
contrat » de l'ADR-0024, pas un bouton de réglage.

**Et l'élargir ne marcherait pas non plus, mesuré.** Le remplissage du lot est de
**84,3 %** à largeur 8 sur une décision 2-ply canonique (`BLUNDERDB_PROBE=1`,
donc environ 6,7 positions par passe de remplissage). Les mêmes passes dans un
lot de 32 rendraient **21 %** de remplissage : le noyau calculerait ~4,8 fois
l'arithmétique pour jeter le reste, afin de gagner le débit de deux
accumulateurs de plus. C'est perdu d'avance, et ce n'est pas une question de
noyau.

**La précondition réelle**, si quelqu'un y revient : **regrouper le travail**
pour que les lots soient pleins à largeur 32 — les vingt et un lancers d'un nœud,
ou plusieurs nœuds d'un même niveau (la piste #145/#146). C'est un chantier de
recherche, pas de noyau, et il précède la tuile au lieu d'en découler.

**La piste que ce dépôt avait notée** — remplacer les `tile` pointeurs de ligne
par un pointeur et des index d'échelle — ne débloque rien à elle seule : l'x86
adresse `base + index×échelle + déplacement`, soit **deux** registres, et
`ligne t, colonne j` en demande trois (`wp + t·rowBytes + jb`) tant que
`rowBytes` n'est pas une constante de compilation — ce qu'il n'est pas, `in`
variant d'une couche à l'autre. Le commentaire de `_asm/main.go` peut donc
rester : il dit ce qui coince, il ne promet pas que ce soit facile.
