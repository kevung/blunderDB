# Poste 2 — le garde-fou sur l'arrondi des tuiles

**Livré.** Ce n'est pas une optimisation et il n'y a rien à mesurer : c'est une
assertion là où il n'y en avait aucune.

## Le rappel, parce que le bug venait d'ici

`outDim & ^(tile-1)` n'arrondit au multiple inférieur **que pour une puissance
de deux**. À tuile 6 et `outDim` = 195 il rend 194, qui n'est pas un multiple de
6 ; la boucle `for j := 0; j < rounded; j += tile` atteint alors j = 192 et lit
les lignes de poids 192 à 197 dans une matrice qui en a 195 (#133). Les tests ne
l'ont pas vu : la tuile valait 4 quand ils ont été écrits, et 4 est une
puissance de deux.

Le défaut a été corrigé en changeant de forme — le générateur compte les sorties
restantes à rebours et n'arrondit plus du tout — **mais sans assertion**. Rien
n'empêchait de réintroduire le masque. gammonNet a écrit le garde-fou que ce
dépôt n'avait pas (`src/gn_tile.h`, chantier T90) ; ce poste le reprend.

## Ce qui est livré

`pkg/blunderdb/engine/gammonnet/tile.go` :

- **`roundDownMultiple(n, tile)`**, avec sa postcondition écrite : *le résultat
  est ≤ n **et** un multiple exact de la tuile*. La seconde moitié est
  précisément celle que le masque viole. Une tuile ou un n non positifs rendent
  0 plutôt que de diviser par zéro — un noyau qui tombe alors entièrement dans
  sa queue scalaire est lent, et la lenteur est un mode de défaillance qu'on
  voit, contrairement à une lecture hors matrice.
- **Une assertion de compilation** : `EvalBatchWidth` est une puissance de deux.
  L'hypothèse est réelle et matérielle — le noyau AVX2 adresse la colonne
  d'activations avec `Scale: lanes`, et une échelle d'adressage x86 ne peut
  valoir que 1, 2, 4 ou 8.

`_asm/main.go` (le générateur, autre module) : **trois assertions de
compilation** — `lanes` puissance de deux, `lanes ≤ 8`, `tile ≥ 1` — avec la
note explicite que **rien n'exige que `outDim` soit un multiple de `tile`**,
puisque c'est le compteur décroissant qui a remplacé l'arrondi.

Vérifié en acte : `lanes` porté à 6 arrête la compilation du générateur sur
`constant -4 overflows uint`. `EvalBatchWidth` à 6 ferait de même sur le paquet.

## Le test, à deux volets

`tile_test.go`, et le second volet est le point.

| volet | ce qu'il montre |
|---|---|
| positif — postcondition | balayage n ∈ [−3, 260] × tuile ∈ [−2, 40] : résultat ≤ n, multiple exact, reste < tuile, et **coïncidence avec le masque sur toute puissance de deux** (sans quoi le remplacement changerait le découpage des noyaux existants) |
| positif — en acte | un parcours tuilé à **tuile 6** sur une ligne allouée à la taille **exacte** (195) va au bout |
| **négatif** | la forme **masquée**, sur cette même ligne, **doit paniquer**. Si elle survit, ce n'est pas que le code est sain : c'est que la ligne n'était pas allouée à la taille exacte, et alors le volet positif ne prouve rien. |
| structurel | `TestNoKernelRoundsATileWithAMask` relit les sources du noyau et du générateur et refuse `&^ (n-1)` hors commentaire — exactement comme `TestGeneratedKernelHasNoFMAAndNoFramePointer` relit l'assembleur produit |

Le volet négatif est la moitié qui manquait à ce dépôt, et c'est la raison pour
laquelle amont l'a écrit : sans lui, un détecteur inactif ferait passer le volet
positif pour une preuve. Amont doit compiler le sien sous AddressSanitizer pour
qu'un débordement de trois flottants soit visible ; ici la vérification de
bornes des tranches est toujours active et la panique est déterministe. C'est
la seule chose que ce portage ait de plus facile.
