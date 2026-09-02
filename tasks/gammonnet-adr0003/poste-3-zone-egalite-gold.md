# Poste 3 — resserrer la zone d'égalité du gold

**Livré.** La publication amont était disponible : **gammonNet v1.3.0** est poussée
sur `origin` (tag `49fa0ad`, 2026-09-03), et son message de version dit
l'essentiel — *« le navigateur reçoit le noyau écrit à la main, et les ex æquo
cessent de dépendre de la libc »*. La consigne était de ne rien resserrer contre
un fichier produit par l'ancien tri ; le fichier a donc été **rejoué contre
v1.3.0** avant que la clause ne bouge.

## Ce qui a été vérifié avant de bouger la version épinglée

Bouger l'épingle de v1.2.1 à v1.3.0 est une décision plus grosse que resserrer
une tolérance : c'est changer la référence. Deux faits ont été établis d'abord,
en construisant le harnais `gold.c` aux deux versions et en comparant les
fichiers produits.

1. **Le harnais est déterministe ici.** Le gold reconstruit à v1.2.1 reproduit le
   fichier commité **octet pour octet**, les deux corpus. Toute différence
   observée ensuite est donc une différence d'amont, pas de compilateur ni de
   drapeaux.
2. **v1.2.1 → v1.3.0, ce qui bouge :**

| corpus | verdict |
|---|---|
| `search_gold.bin` (argent, sans videau, 85 décisions) | **bit à bit identique** |
| `search_cube_gold.bin` (match + videau, 123 décisions) | **41 équités de candidat** bougent, de **5,551e-17 à 2,220e-16** — une à deux ulp d'un double ; **aucun coup choisi ne change**, nulle part |

Contre une tolérance de 1e-6, 2,2e-16 est **dix ordres de grandeur** en dessous
de ce que la porte peut voir. L'épingle bouge donc sans rien remettre en cause :
le portage passe la porte à v1.3.0, marges 7,153e-07 (argent) et 3,902e-07
(match+videau), **0 ex æquo employé** des deux côtés.

## Ce qui a été trouvé en chemin, et qui remonte en amont

Les 41 équités ont été bissectées : elles viennent de **`23c5a64`**, le commit du
**videau par lot (T85)** — dont la note de mesure annonce un résultat *« bit à
bit »*, 12 600 classements rejoués sans une ligne de `diff`.

Les deux affirmations sont conciliables et il vaut la peine de dire comment :
**la forme par lot est bien bit à bit en arithmétique exacte**, et ce qui diffère
est la **contraction en FMA**. `segment()` calcule `y0 + (y1−y0)·t` ; avec
`-march=native` sur Zen 3 et le `-ffp-contract=fast` que gcc applique par défaut,
la multiplication et l'addition se fondent en une FMA dans une forme et pas dans
l'autre, selon ce que l'inlining laisse voir à l'optimiseur. Une FMA garde plus
de précision que la paire séparée : d'où l'ulp.

**À consigner en amont** : la preuve de T85 (`diff` sur des équités en
hexadécimal IEEE-754) est valable pour les drapeaux sous lesquels elle a tourné,
et ne l'est pas sous `-O3 -march=native`. C'est exactement le genre de chose
que l'ADR-0024 de blunderDB interdit chez lui (*no FMA, ever*), et le
harnais de comparaison amont gagnerait à porter `-ffp-contract=off`, ou à dire
sous quels drapeaux la bit-identité est affirmée.

## Le resserrement livré

La clause tolérait **deux** causes de désaccord sur le coup choisi. Une seule
subsiste.

- **Le départage des ex æquo — disparu.** La référence triait avec `qsort`, qui
  n'est pas stable ; sur une égalité parfaite le coup rendu dépendait de la libc.
  Depuis v1.3.0 gammonNet trie stablement, et **en reprenant la règle de ce
  portage** : à équité égale, l'ordre de génération est conservé. Les deux
  implémentations sont donc d'accord sur les ex æquo par construction. (Le
  recensement amont ajoute que la divergence n'était pas partout : le `qsort` de
  la glibc est stable en pratique, celui d'Emscripten ne l'est pas — donc ce
  fichier-ci n'a jamais été faux, seulement injustifié.)
- **Le bruit arithmétique — subsiste.** Deux coups dont les vraies équités sont
  plus proches que le double de l'écart port/référence peuvent s'échanger de rang
  sans que personne ait tort. C'est pourquoi la clause est **resserrée et non
  supprimée** : la supprimer rendrait la porte rouge le jour où deux coups
  passent à 7e-07 l'un de l'autre, ce que le tri stable ne corrige pas.

**Le resserrement** : `replayGold` **échoue** désormais si l'allocation est
employée, au lieu de la compter dans une ligne de journal. Elle ne l'a jamais
été — 0 sur les 85 décisions argent et les 123 décisions match+videau — et le
README du fichier d'or décrivait déjà un compte qui monte comme un signal.
Réemployer la zone redevient un acte délibéré, qui doit nommer les deux coups et
dire pourquoi.

## Comment rejouer

`testdata/gold/README.md` est à jour : épingle v1.3.0, et **deux unités de
traduction de plus** à lier (`gn_int8_model.c`, `gn_gemm_int8.c`) — le
`gn_infer_reference.c` de v1.3.0 référence `gn_int8_model_evaluate` même si ce
harnais ne charge jamais de modèle quantifié.
