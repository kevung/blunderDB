# Videau de Janowski en Go : miroitage de l'efficacité et inversion en forme close

## TL;DR
- **Miroitage :** l'efficacité `cube_x` doit être déterminée par nœud/feuille selon l'ÉTAT LOCAL du videau (donc « miroitée » avec le changement de trait à chaque ply), pas figée à la racine. C'est ce que fait gnubg — le paramètre X n'entre qu'à la feuille 0-ply, via le `cubeinfo` local. Figer X à la racine et tarifer la branche `eDT` avec l'efficacité du propriétaire courant est un **bug** ; son impact est modéré en moyenne mais se concentre sur les décisions de double dans les positions volatiles/à fort recube-vig.
- **Inversion :** la bissection à 60 itérations est inutile. Le prix cubeful de Janowski est affine par morceaux et monotone en p ; ses points de bascule (take point, cash point, too-good) ont des **formes closes exactes** (fonctions rationnelles de W, L, x). L'inversion se fait en O(nb de segments) : identification du segment contenant la valeur cible + une résolution linéaire.
- **gnubg vs XG :** les deux implémentent la même formule de Janowski pour l'argent (take points identiques). Dans gnubg, X dépend de la CLASSE de position (course/contact/crashed/bearoff), pas du propriétaire du videau ; XG étend Janowski au jeu au score. Aucune source publique n'indique que l'un ou l'autre fasse dépendre X du propriétaire.

## Key Findings

1. **Le modèle de Janowski (1993, *Take-Points in Money Games*) fournit des formes closes pour toutes les décisions de videau.** Take-point général `TP = (L−0.5)/(W+L+0.5x)`, cash-point `CP = (L+0.5+0.5x)/(W+L+0.5x)`, too-good `TG = (L+1)/(W+L+0.5x)`. Les équités cubeful sont linéaires par morceaux en p.

2. **gnubg applique l'efficacité X uniquement à la feuille (0-ply), via le `cubeinfo` local.** Le manuel gnubg V0.16 (section « n-ply Cubeful equities », gnu.org/software/gnubg/manual) l'indique verbatim : « Note that the 2-ply level does not use the cube efficiency, it's not used until at the 0-ply level, but it's possible to calculate an effective one by isolating x in the basic cube formula: x(eff) = (E(2-ply cubeful) − E(2-ply dead))/(E(2-ply live) − E(2-ply dead)). » Aux plies internes, gnubg ne fait qu'une récursion moyenne + décision de videau ; c'est la propagation du `cubeinfo` (inversion de `fMove`/`fCubeOwner`) qui « miroite » l'état du videau.

3. **Dans gnubg, X dépend de la CLASSE de position, pas du propriétaire :** contact/crashed = 0.68, bearoff 1-sided = 0.6, course = interpolation linéaire 0.6→0.7 selon le pip count. Le manuel gnubg (« The cube efficiency ») : « For race gnubg uses linear interpolation based on pip count for the player on roll. A pip count of 40 gives x=0.6 and 120 gives x=0.7. If the pip count is below 40 or above 120 values of x=0.6 and x=0.7 are used, respectively. » Le propriétaire du videau ne fait que sélectionner la branche d'équité (centré/possédé/adverse) dans `Cl2CfMoney`, pas la valeur de X.

4. **Le schéma du portage (X dépendant du propriétaire : 0.68 centré, 0.57 possédé, 0.69 adverse) n'est ni celui de gnubg ni celui du modèle de base de Janowski,** où x est un indice de vie du videau propre à la POSITION. Il évoque plutôt le « refined general model » de Janowski (indices x1, x2 distincts par joueur). Higgins (arXiv:1203.5692) montre que dans ce modèle « the take point … is a function only of x1 and the cash point only of x2 », et que les x1/x2 impliqués valent p.ex. 0.69/0.69 quand W=L=1 et divergent (~0.59/0.80) quand W=2, L=1 — soit exactement l'écart possédé/adverse évoqué par le portage.

5. **La bissection est superflue.** Janowski donne les seuils directement ; pour le jeu au score, le take point live est un produit récursif fini (donc encore une forme close), pas une recherche numérique.

## Details

### Contexte de source

- **Article primaire :** Rick Janowski, *Take-Points in Money Games*, Hoosier BG Club magazine / Das Backgammon Magazin, 1993 (PDF : bkgm.com/articles/Janowski/cubeformulae.pdf).
- **Code gnubg :** `eval.c` (par Gary Wong, 1998–2002 ; ~6285 lignes), release 1.08.003 (commit `b1582e2600095ed0e43256246ae5a244265b63cf`), miroir git GitHub `mormegil-cz/gnubg` et dépôt officiel Savannah `git.savannah.gnu.org/cgit/gnubg.git`. Signatures confirmées dans `eval.h` :
  ```c
  extern float Cl2CfMoney(float arOutput[NUM_OUTPUTS], cubeinfo * pci, float rCubeX);
  extern float Cl2CfMatch(float arOutput[NUM_OUTPUTS], cubeinfo * pci, float rCubeX);
  ```
- **Comparaison XG/gnubg :** Lasse Hjorth Madsen, « Janowski Formulas » (lassehjorthmadsen.github.io/bganalyses/analyses/janowski.html).
- **Discussions primaires :** bgonline.org (Timothy Chow, Rick Janowski, Chuck Bower) ; liste bug-gnubg (mars 2009, Massimiliano Maini / Christian Anthon) ; rec.games.backgammon (Jørn Thyssen).
- **Modèle jump (forme close alternative) :** Mark G. Higgins, arXiv:1203.5692.

### Les formules de Janowski (argent, modèle général de base)

Notations : p = probabilité de gain cubeless du joueur au trait ; W = valeur cubeless moyenne des parties gagnées ; L = valeur cubeless moyenne des parties perdues ; x = indice de vie du videau (cube-life index) ∈ [0,1]. Pour une course sans gammon, W = L = 1.

Équité cubeless (dead) : `E_dead = p(W+L) − L`.

Équités cubeful linéaires par morceaux (Appendice 1 de Janowski, CV = 1) :
- **Videau possédé (par le joueur au trait) :** `E_O = p(W + L + 0.5x) − L`  (éq. 5)
- **Videau indisponible (possédé par l'adversaire) :** `E_U = p(W + L + 0.5x) − L − 0.5x`  (éq. 6)
- **Videau centré :** `E_C = (4/(4−x))·[p(W + L + 0.5x) − L − 0.25x]`  (éq. 7)

Ces expressions sont affines en p sur le segment central (avant déclenchement du videau) ; l'équité complète est plafonnée aux frontières de prise/cash.

Formulation équivalente utilisée par gnubg (manuel) : `E(cubeful) = E(dead)·(1−x) + E(live)·x`, où `E(live)` est l'interpolation linéaire par morceaux entre les points **(0, −L), (TP_live, −1), (CP_live, +1), (1, +W)**. On vérifie l'algèbre : pour le videau possédé, `(1−x)[p(W+L)−L] + x[p(W+L+0.5)−L] = p(W+L+0.5x) − L` = éq. 5. Massimiliano Maini (liste bug-gnubg, 3 mars 2009) confirme les trois lignes utilisées par gnubg : « 5.1) Egeneral_own = Edead*(1−x) + Elive_own*x … 5.2) Egeneral_unav = Edead*(1−x) + Elive_unav*x … 5.3) Egeneral_cen = Edead*(1−x) + Elive_cen*x ».

Points de bascule (formes closes, modèle général) :
- Take-point : `TP = (L − 0.5)/(W + L + 0.5x)`
- Cash-point : `CP = (L + 0.5 + 0.5x)/(W + L + 0.5x)`
- Too-good : `TG = (L + 1)/(W + L + 0.5x)`
- Redouble : `RD = (L + x)/(W + L + 0.5x)`
- Dead-cube take point (x=0) : `TP = (L − 0.5)/(W + L)` ; live (x=1) : `TP = (L − 0.5)/(W + L + 0.5)`.

Remarque : la valeur 0.57 (proche du « possédé » 0.57 du portage) apparaît chez Janowski comme moyenne empirique : « Maximum divergence occurs when x is about 0.57, and typically ranges between 2.00% (W = 2, L = 2) and 3.75% (W = 1, L = 1) … I have used 0.57 as an average value » — mais c'est une moyenne sur la position, jamais une valeur attribuée au propriétaire.

### QUESTION 1 — Faut-il miroiter l'efficacité du videau avec le propriétaire à chaque ply ?

**Ce que fait gnubg.** Trois faits établis :

(a) *X n'entre qu'à la feuille.* La recherche cubeful 2-ply de gnubg est une récursion : boucle sur les 21 lancers, meilleur coup, évaluation n−1 ply, moyenne /36 ; à chaque nœud interne on évalue une décision de videau (no double vs double/take, combiné au pass), mais l'efficacité X n'est jamais utilisée avant le niveau 0-ply (manuel V0.16, cité en Key Finding 2). Timothy Chow (bgonline) le confirme : « Janowski only applies to the last ply of an evaluation … each previous ply is done by directly averaging the evaluations of the 21 possible rolls ».

(b) *Le propriétaire est propagé par le `cubeinfo` local et l'inversion.* La structure `cubeinfo` porte `fCubeOwner`, `fMove`, `nCube`, `nMatchTo`, `anScore`, `fCrawford`, etc. À chaque ply gnubg génère les positions filles avec `MakeCubePos` et inverse l'évaluation (`InvertEvaluation`/`InvertEvaluationCf`), de sorte que `fMove` bascule et que la relation `fCubeOwner == fMove` change de sens automatiquement. Pour une évaluation cubeful 2-ply, gnubg calcule ainsi les 0-ply : videau centré 1, adversaire possède 2, possédé 4, adversaire possède 8.

(c) *Dans `Cl2CfMoney`, la sélection de branche se fait sur le `cubeinfo` local.* Les trois cas sont `pci->fCubeOwner == −1` (centré → E_C), `pci->fCubeOwner == pci->fMove` (possédé → E_O), sinon (adverse → E_U). Chaque branche calcule `E_dead` et `E_live` puis renvoie `E_dead*(1−rCubeX) + E_live*rCubeX`. La valeur `rCubeX` passée provient de `EvalEfficiency`, fonction de la CLASSE de position de cette feuille (course/contact/crashed/bearoff), **et ne dépend pas du propriétaire du videau**.

**Conclusion pour gnubg :** le « miroitage » est intrinsèque et correct, parce que (1) X est réévalué à chaque feuille depuis la position locale et (2) la branche d'équité est choisie via le `cubeinfo` local (déjà inversé). gnubg ne fige jamais X à la racine et ne fait pas varier X par propriétaire.

**Ce que fait XG.** XG implémente la même formule de Janowski pour l'argent. Lasse Hjorth Madsen montre que les take points de XG (`Analyze|Cube Information`) et de gnubg (`Analyze|Market Window`) coïncident au chiffre près (« both programs probably implemented Janowski's take point formula in the same way for money games »). L'efficacité effective varie avec la POSITION : Madsen retrouve XG avec x≈0.622 dans un bearoff court, x≈0.7 en début de partie, et rappelle que « GNU Backgammon … use[s] a cube-life index, or cube efficiency, of 0.68 for contact positions … 0.60 rather than 0.68 in short races ». La documentation XG indique que « the Janowski formula has been extended to apply also in match play ». Aucune source publique n'indique une efficacité dépendant du propriétaire du videau.

**Position sur le portage.** Le schéma décrit (X figé à la racine selon le propriétaire : 0.68 centré / 0.57 possédé / 0.69 adverse, puis appliqué à toutes les feuilles pendant que le propriétaire est miroité) présente **deux incohérences** :

1. *X figé à la racine.* Dans le modèle de Janowski, x est l'indice de vie du videau de la POSITION (sa volatilité, la facilité à doubler efficacement). Il devrait être réévalué à chaque feuille, exactement comme gnubg le fait via `EvalEfficiency`. Figer x et l'appliquer à des feuilles de classes différentes (une course en fin d'arbre a x≈0.6, un contact x≈0.68) est une approximation ; son impact est amorti par le lookahead (Chow : « it doesn't change the cubeful evaluation by too much »), mais elle est systématiquement fausse dès que la classe diffère de la racine.

2. *X dépendant du propriétaire mais NON miroité, et branche `eDT` mal tarifée.* Si on maintient des x dépendant du propriétaire (0.57/0.68/0.69), alors la cohérence interne EXIGE de re-sélectionner x selon l'état LOCAL du videau à chaque nœud — c'est-à-dire de miroiter x avec le propriétaire au même titre que la branche d'équité. Miroiter la branche (E_O/E_U/E_C) mais garder le x de la racine mélange les pentes de deux configurations. Surtout, la branche `eDT` (après double/take, le videau passe à l'adversaire) doit être tarifée avec la configuration « adverse » : sous un schéma dépendant du propriétaire, cela signifie x = 0.69 (adverse), pas 0.57 (possédé courant). Tarifer `eDT` avec l'efficacité du propriétaire courant est un **bug** : il fausse précisément le recube-vig, c'est-à-dire l'écart E_O − E_U que capture la distinction possédé/indisponible.

**Réponse directe :** OUI, il faut miroiter — mais la bonne façon est de faire de x une fonction de la POSITION de la feuille (classe/volatilité), réévaluée à chaque feuille, la branche d'équité étant sélectionnée par le `cubeinfo` local. Cela supprime l'ambiguïté du miroitage et reproduit gnubg/XG. Si le portage garde absolument des x dépendant du propriétaire (schéma type « refined general model » x1/x2), alors il DOIT les re-sélectionner selon l'état local à chaque nœud, et tarifer `eDT` avec x_adverse.

**Impact probable de l'erreur.** L'effet global de x sur l'équité cubeful est petit et amorti par le lookahead ; l'erreur est donc généralement modeste. Mais elle se concentre là où le recube-vig domine : décisions double/take vs no-double proches des frontières (cash/too-good), jeux de tenue (holding), back games et blitz — précisément les classes où x « vrai » s'écarte le plus de 0.68 (le manuel gnubg note que holding games et back games ont une efficacité plus basse, les blitz plus haute). La mauvaise tarification de `eDT` biaise directement les décisions de double, plus sensibles à x que les take points (Janowski : « the take-point is relatively insensitive to the J-Factor … [doubling-points] are far more sensitive »).

### QUESTION 2 — Inversion en forme close du prix « live » de Janowski

**Le point essentiel : la bissection est inutile.** Janowski donne les seuils directement en forme close (TP, CP, TG, RD ci-dessus). gnubg lui-même ne calcule pas les take points par inversion pour ses décisions de videau — manuel : « Note that gnubg doesn't calculate the take point or double point explicitly. The cube decision is simply made by comparing equities » ; mais son module « théorie » (`gtktheory.c`) et le calcul de la fenêtre de marché (`GetPoints`) affichent les take points en forme close via les formules de Janowski et les gammon rates (`getCurrentGammonRates`). Pour le jeu au score, le take point live est un produit récursif FINI. La documentation gnubg le formule ainsi : « The live cube take point is generally calculated as TP(live, n Cube) = TP(dead, n cube) * (1 − TP(live, 2n cube)) », déroulé jusqu'au dernier redouble possible (borné par le score) — donc encore une forme close, pas une recherche numérique.

**Inversion d'une fonction affine par morceaux monotone.** Quand on doit inverser l'équité cubeful pour un target E* arbitraire (par ex. un seuil), on exploite que `E_general(p) = (1−x)·E_dead(p) + x·E_live(p)` est affine par morceaux et monotone, avec les mêmes nœuds que E_live : `{q0=0, q1=TP_live, q2=CP_live, q3=1}`.

Aux nœuds : E_live = {−L, −1, +1, +W} ; E_dead(q) = q(W+L)−L ; donc `E_gen_i = (1−x)·E_dead(q_i) + x·E_live_i`. Aux extrémités E_gen(0) = −L et E_gen(1) = +W quel que soit x.

Algorithme (O(nb de segments)) :
1. Calculer les nœuds q_i et les équités E_gen_i.
2. Identifier le segment [q_i, q_{i+1}] tel que E_gen_i ≤ E* ≤ E_gen_{i+1}.
3. Résoudre linéairement : `p = q_i + (E* − E_gen_i)·(q_{i+1} − q_i)/(E_gen_{i+1} − E_gen_i)`.

**Pseudo-code (traitement complet des cas limites) :**

```
fonction inverser_equity_vers_p(Etarget, W, L, x, etat_videau, MET_context):
    # 1. Nœuds selon l'état du videau (centré/possédé/adverse) et la MET
    #    En argent: TP_live=(L-0.5)/(W+L+0.5), CP_live=(L+0.5)/(W+L+0.5) [x=1]
    #    En match: TP_live via produit récursif borné par le score (Kazaross-XG2):
    #              TP_live(n) = TP_dead(n) * (1 - TP_live_opp(2n))
    q   = [0.0, TP_live, CP_live, 1.0]
    Elv = [-L, -1.0, +1.0, +W]
    # 2. Équités générales aux nœuds
    E = tableau de même taille que q
    pour i dans 0..len(q)-1:
        Edead_i = q[i]*(W+L) - L
        E[i]    = (1-x)*Edead_i + x*Elv[i]
    # 3. Nettoyage: fusionner/ignorer les segments dégénérés
    (q, E) = compacter_noeuds(q, E, eps_p=1e-12)   # supprime q[i+1]-q[i] ~ 0
    # 4. Garde-fous de monotonie
    assert non_decroissante(E, tol=1e-9)            # sinon W,L pathologiques -> clamp
    # 5. Clamp hors domaine
    si Etarget <= E[0]:            retourner q[0]     # borne basse fermée
    si Etarget >= E[dernier]:      retourner q[dernier]
    # 6. Recherche du segment (bornes semi-ouvertes [q_i, q_{i+1}), dernier fermé)
    pour i dans 0..len(q)-2:
        si E[i] <= Etarget < E[i+1]  OU  (i==dernier-1 ET Etarget==E[i+1]):
            dE = E[i+1] - E[i]
            si abs(dE) < eps_slope:                  # segment plat
                retourner q[i]                        # convention: borne gauche
            t = (Etarget - E[i]) / dE
            retourner q[i] + t*(q[i+1] - q[i])
    # 7. Filet de sécurité (ne devrait jamais arriver si monotone)
    retourner clamp(q[0], q[dernier], ...)
```

**Pièges numériques.**
- *Segments dégénérés :* TP_live = CP_live quand W ou L rendent la fenêtre de marché nulle (ex. gammons extrêmes) → longueur nulle ; les compacter avant recherche.
- *Pentes nulles/quasi nulles :* si E_gen_{i+1} ≈ E_gen_i, la division explose ; garde `eps_slope`, convention de retour à la borne gauche.
- *Égalité aux jonctions :* définir des intervalles semi-ouverts `[q_i, q_{i+1})` (dernier fermé) rend la sélection déterministe et gère la monotonie non stricte.
- *Dépendance à la MET (Kazaross-XG2) :* en match, W et L dérivent d'écarts de MWC et varient par score et niveau de videau ; recalculer les nœuds à chaque nœud de l'arbre. Le take point live match utilise le produit récursif fini borné par le score (reste closed-form).
- *Reproductibilité bit-à-bit C↔Go :* gnubg travaille en `float` (float32) sur `arOutput`. Pour être bit-identique : (1) utiliser float32 là où gnubg utilise float ; (2) préserver l'ordre des opérations et l'associativité ; (3) **ne pas** fuser en FMA. En C, `-ffast-math`/`-mfma` peuvent fusionner `a*b+c` (une seule arrondi) ; le compilateur gc de Go ne fuse PAS automatiquement `a*b+c` en float32 (il ne sélectionne l'FMA que via `math.FMA` explicite), donc n'appelez pas `math.FMA` si la référence C ne fuse pas. Go n'a pas de précision excédentaire type x87 (intermédiaires float32 gardés en float32), ce qui rend l'égalité atteignable. Le risque de divergence bissection↔forme close est maximal près des jonctions : la bissection après 60 itérations a une erreur ≤ 2⁻⁶⁰ ≈ 8.7e−19 en p (donc quasi exacte), la forme close est exacte à quelques ULP ; l'écart provient de l'ordre des opérations, pas de la méthode.

**Preuve d'équivalence (oracle différentiel).**
- *Oracle :* la bissection de référence à 60 itérations.
- *Outils Go :* `testing/quick`, `github.com/leanovate/gopter`, `pgregory.net/rapid`.
- *Grilles :* p ∈ [0,1] (pas fin + points spéciaux 0, TP, CP, TG, 1) ; W, L ∈ [1, 3] ; x ∈ {0, 0.5, 0.57, 0.6, 0.68, 0.69, 0.7, 1} et continu ; états {centré, possédé, adverse} ; scores/MET (Kazaross-XG2) incluant DMP, Crawford, post-Crawford, positions gammon-riches.
- *Tolérances :* |Δp| ≤ 1e−9 absolu (ou ≤ quelques ULP) ; en match, comparer aussi en MWC.
- *Tests frontières :* p exactement aux nœuds ; W = L = 1 (gammonless, TP_dead = 0.25, TP_live = 0.20) ; x = 0 et x = 1 ; segments dégénérés.
- *Propriétés :* monotonie (E croissante), continuité aux jonctions (évaluer juste en dessous/au-dessus de chaque q_i, égalité à l'ULP), aller-retour `E(inverser(E*)) ≈ E*`, fuzzing sur entrées aléatoires alimentant les deux implémentations.

## Recommandations

**Étape 1 — Corriger le miroitage (priorité haute).** Remplacer « X figé à la racine » par une réévaluation de X à chaque feuille selon la CLASSE de la position (comme `EvalEfficiency` : contact/crashed 0.68, bearoff 0.6, course interpolée 0.6→0.7 sur pip 40→120). Laisser le `cubeinfo` local (fMove/fCubeOwner inversés à chaque ply) sélectionner la branche E_O/E_U/E_C. C'est la solution qui reproduit gnubg et supprime l'ambiguïté.

**Étape 2 — Si l'on conserve des x dépendant du propriétaire :** re-sélectionner x selon l'état LOCAL du videau à chaque nœud, et tarifer explicitement la branche `eDT` avec x_adverse (0.69) et l'équité E_U. Benchmark de bascule : comparer les décisions double/take/no-double avec vs sans correction sur un jeu de positions de holding/back game/blitz ; si l'erreur EMG dépasse ~1–2 millièmes sur ces classes, la correction est nécessaire.

**Étape 3 — Remplacer la bissection par la forme close.** Utiliser directement TP/CP/TG de Janowski (argent) et le produit récursif borné par le score (match). Bénéfice attendu : suppression des ~39 % de coût par décision au score. Valider par oracle différentiel (Étape 4) avant bascule.

**Étape 4 — Prouver l'équivalence :** property-based testing (rapid/gopter) contre la bissection, grilles ci-dessus, tolérance ≤ quelques ULP, tests de monotonie/continuité/aller-retour. Seuil de bascule : 0 échec sur ≥ 10⁷ cas aléatoires + tous les cas frontières.

**Étape 5 — Reproductibilité :** figer float32/float64 comme la référence, désactiver l'FMA implicite, verrouiller l'ordre des opérations ; test de non-régression bit-à-bit contre un vecteur de positions gnubg de référence.

## Caveats
- Le corps verbatim ligne-à-ligne de `Cl2CfMoney`/`Cl2CfMatch` n'a pas pu être extrait (le fichier fait ~6285 lignes ; les vues HTML tronquent avant la fonction, et les endpoints raw ont été bloqués). La structure des trois branches et le retour `E_dead*(1−rCubeX)+E_live*rCubeX` sont établis par les signatures `eval.h`, le manuel gnubg et la liste bug-gnubg (Maini, mars 2009), non par lecture directe des lignes. À vérifier sur `git.savannah.gnu.org/cgit/gnubg.git/plain/eval.c`.
- Les valeurs d'efficacité par propriétaire du portage (0.68/0.57/0.69) sont « mesurées en amont » ; leur provenance exacte n'est pas dans les sources publiques et ne correspond ni au modèle de base de Janowski ni au schéma gnubg (position-class). Elles évoquent le « refined general model » (x1, x2) — cf. Higgins.
- Le modèle de Janowski suppose gammon rates et efficacité constants sur la vie de la partie — simplification reconnue (Higgins 2012, jump model, propose une alternative en forme close basée sur la volatilité locale).
- L'affirmation « XG n'utilise pas d'efficacité dépendant du propriétaire » repose sur l'absence de preuve publique du contraire (moteur XG propriétaire, non documenté) ; à traiter comme forte présomption, non comme certitude.