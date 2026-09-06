# C.13 — Amont gammonNet : le registre des sept points

*Relevé du 2026-09-06. Ce fichier ne décide rien : il consigne, pour chacun
des sept points de la fiche C.13 (#200), ce qui est **mesuré**, ce qui est
**décidé** et ce qui **attend l'amont**. La règle qui les range tous là-bas
est l'ADR-0003 de gammonNet, reprise dans CLAUDE.md : une optimisation est
conceptuelle si son gain survit à un changement de langage, et une
optimisation conceptuelle s'écrit en amont, avec sa mesure.*

Un point n'est pas « à faire ici ». Ce que ce dépôt peut livrer sur chacun est
son **instrument de mesure**, son **chiffre**, et le **correctif proposé** ;
la décision est celle de gammonNet.

---

## 1. Forme close de `levelSolve` (C.7) — **MESURÉ, décidé, en attente amont**

`levelSolve` inverse une fonction affine par morceaux et monotone dont les
deux ou trois segments sont connus d'avance, et il l'inverse par **soixante
pas de bissection**.

- **Mesure** (2026-09-03, Ryzen 7 PRO 6850U, 16 cœurs, Go 1.25.13, décision
  2-ply k=12 au score 5-away/5-away) : `BenchmarkDecision2Ply` 193 ms en money,
  `BenchmarkDecision2PlyMatch` 306 ms au score ; profil de la seconde,
  `levelSolve` **35,4 %** cumulé. L'écart money↔score, 113 ms, est du même
  ordre que ce que la bissection prend.
- **Instrument** : `cube_closedform_measure_test.go`, avec
  `levelSolveClosed` — le correctif proposé, écrit ici pour être mesuré — et
  `TestClosedFormAgreesWithBisection`, qui tourne **toujours** et garantit que
  la forme close est bien la même fonction.
- **Décision** : ADR-0032, « la forme close de l'inversion du videau se décide
  en amont ». Le gain survit au changement de langage (la forme de
  l'algorithme, pas son écriture) et il n'est **pas** bit-identique — d'où
  l'amont, et d'où la jauge.
- **Attend** : le portage dans `gn_cube.c` et le nouveau `testdata/cube_gold.bin`.

## 2. Efficacité miroitée / branche DT (C.5) — **MESURÉ, écarté ici**

`SearchConfig.CubeX` est figé à la racine pendant que `CubeOwner` est miroité,
et `Decide` price `eDT` au coefficient du propriétaire courant.

- **Mesure** (669 décisions réelles) : **0,005** d'équité normalisée par
  feuille, **aucun verdict retourné**, **aucun coup changé**.
- **Décision** : ADR-0029. Les deux comportements suivent `gn_search.c` /
  `gn_cube.c` ligne pour ligne ; « corriger » l'un ou l'autre ici fabriquerait
  une divergence de portage et ferait rougir l'or du videau.
- **Attend** : rien de ce dépôt. Le correctif, s'il en faut un, est celui de
  gammonNet.

## 3. Réseau distillé 60-100 k MAC — **NON COMMENCÉ, priorité amont n° 1**

- **Chiffre** (rapport P4) : **×5 à ×9**, et c'est le seul gain de P4 qui
  « survit à la recherche » — les autres se dissolvent dans le coût de
  l'expectiminimax.
- **Ce que ça implique ici** : nouvelle Configuration au sens de CONTEXT.md,
  donc nouveau `EngineVersion`, donc **bases périmées**. Le support de cette
  péremption existe déjà : `analyze --stale` et son déclencheur (C.4).
- **Attend** : l'entraînement et la publication du réseau, en amont, avec sa
  jauge de force. Rien à écrire ici avant.

## 4. Filtres de coups à seuil d'équité (movefilter) — **NON COMMENCÉ**

`SearchConfig.Filter` est un **compte par profondeur** : « garde les n
meilleurs ». Le movefilter de gnubg est un triplet (accepte, extra, seuil
d'équité) : « garde les n meilleurs, plus tout ce qui est à moins de t du
meilleur ».

- **Ce que ça change** : le filtre actuel coupe au même endroit qu'une
  position soit tranchée ou serrée. Un seuil d'équité garde plus de candidats
  là où ils sont proches, et moins là où ils ne le sont pas.
- **Pourquoi c'est amont** : approximatif par construction, donc jauge de
  force ; et c'est la forme de l'algorithme, pas son écriture.
- **Attend** : la décision de spec en amont. Aucun instrument de mesure n'est
  écrit ici — ce serait mesurer une fonction qui n'existe pas encore.

## 5. Beaver / raccoon dans `Decide` — **DÉCIDÉ, documenté, non implémenté**

`domain.Position.HasBeaver` est stocké et transporté ; **rien** dans la
décision de videau ne le lit, et c'est écrit en toutes lettres dans l'en-tête
de `Decide` (#193/C.6).

- **La raison** : le beaver (« take → beaver ») change le propriétaire et la
  valeur du videau **à l'instant même** où la prise est décidée, ce que
  `Decide` ne peut pas exprimer — elle répond « double, prend, passe » pour
  UN état de videau, pas pour une séquence de deux décisions à deux valeurs.
- **La règle candidate** (du plan) : money, videau centré, le preneur
  redouble quand eDT > +1.
- **Attend** : la spec §2 de gammonNet. En attendant, `HasBeaver` est
  décoratif, et le dire est préférable à le laisser croire.

## 6. Réseau de course dédié hors bearoff pur — **NON COMMENCÉ**

`engine/race` couvre la zone de **bearoff pur** (les deux camps entièrement au
jan) ; la zone sans contact mais pas encore au jan est évaluée par le réseau
de contact, faute d'un réseau de course.

- **Où c'est écrit** : `race/eval.go` — « la zone de course reste une zone de
  bearoff pur ; élargir la zone est une décision d'affichage, pas de données ».
- **Ce que gnubg fait** : trois réseaux, `nnContact` / `nnRace` / `nnCrashed`,
  aiguillés par `ClassifyPosition` (rapport P5 §A). gammonNet en a un.
- **Attend** : un second réseau, donc un entraînement, donc l'amont. Le
  classificateur qui l'aiguillerait, lui, **existe déjà ici** depuis #291
  (`engine.ClassifyGameType`, dont `race` et `crunch` sont exactement les deux
  frontières de gnubg).

## 7. NEON arm64 (#151) — **HORS PÉRIMÈTRE, faute de machine**

- **Chiffre attendu** (rapport P2) : NEON **sans FMA** vaut ≈ 0,5× un `FMLA`,
  et l'invariant du noyau interdit la fusion (ADR-0024) — c'est précisément
  sur arm64 que l'interdiction protège, puisque Go contracte là et jamais sur
  amd64.
- **Ce qui manque** : une mesure du coût réel sur M1. Aucune machine arm64
  n'est disponible ici, et une boucle de vérification qui ne passerait que par
  la CI est une boucle où l'on ne mesure pas.
- **Ce qui existe déjà** : `kernel_identity_test.go`, qui exige l'égalité
  **bit pour bit** avec le repli pur-Go, et le balayage arm64 hebdomadaire de
  `fuzz.yml`. Un noyau NEON qui arriverait un jour a donc déjà son juge.

---

## Ce que le registre dit, une fois lu d'un bloc

Trois points sur sept sont **clos de ce côté-ci** : 1 et 2 ont leur mesure,
leur chiffre et leur ADR ; 5 a sa raison écrite dans le code qu'elle concerne.
Trois autres (3, 4, 6) attendent un **entraînement ou une spec** en amont, et
rien ne peut être écrit ici avant — écrire l'instrument de mesure d'une
fonction qui n'existe pas serait mesurer une intention. Le septième attend une
**machine**.

Aucun des sept ne demande du code dans blunderDB aujourd'hui, ce qui est la
réponse que la fiche C.13 appelait : « des décisions, pas des fiches ici ».
