# Classification automatique des positions de backgammon par type de jeu — synthèse sourcée et arbre de décision pour blunderDB

## TL;DR
- **La seule classification réellement déterministe, sourcée et implémentée en code est celle de gnubg** (`ClassifyPosition` dans `eval.c` : classes `OVER / RACE / CRASHED / CONTACT` + bases de bearoff) — mais ces classes servent à **aiguiller vers des réseaux de neurones spécialisés, pas à décrire un plan de jeu humain**. gnubg ne distingue ni blitz, ni holding, ni backgame, ni prime-vs-prime : tout cela tombe dans une unique classe `CONTACT`.
- **Les définitions des plans de jeu humains** (Magriel, Robertie, Woolsey, Trice, Ballard/Weaver, glossaires bkgm.com et USBGF) sont **qualitatives et convergentes sur les concepts, mais quasiment dépourvues de seuils numériques** délimitant un type d'un autre ; les rares seuils chiffrés robustes concernent le **videau en course** (Robertie : double à +8 %, redouble à +9 %, prise jusqu'à −12 % du pip count) et l'exigence de **≥ 2 ancres** pour un backgame — jamais la frontière entre types.
- Nous proposons ci-dessous un **arbre de 10 règles ordonnées, O(1) sur le vecteur des 26 points + pipcount**, qui réplique d'abord les frontières sourcées de gnubg puis ajoute une couche « plan de jeu humain » comme étiquettes principale/secondaire, chaque seuil étant explicitement marqué **SOURCÉ** ou **heuristique à calibrer**. Comme **aucune étude publiée ne mesure d'accord inter-classifieurs (kappa)** sur ce problème, la classification de blunderDB sera une convention interne à valider par échantillonnage manuel.

---

## Key Findings

1. **gnubg = seule « classification » = code déterministe.** `ClassifyPosition(const TanBoard anBoard, const bgvariation bgv)` renvoie une valeur de l'énumération `positionclass` de `eval.h`. L'ordre **exact vérifié dans le source** (miroir GitHub `mormegil-cz/gnubg`) est :
   `CLASS_OVER=0, CLASS_HYPERGAMMON1, CLASS_HYPERGAMMON2, CLASS_HYPERGAMMON3, CLASS_BEAROFF2, CLASS_BEAROFF_TS, CLASS_BEAROFF1, CLASS_BEAROFF_OS, CLASS_RACE, CLASS_CRASHED, CLASS_CONTACT`.
   (⚠️ Correction utile : l'ordre réel des quatre classes de bearoff est `BEAROFF2 → BEAROFF_TS → BEAROFF1 → BEAROFF_OS`, différent de l'ordre parfois cité.) Ces classes routent vers trois réseaux : `nnContact` (milieu de partie), `nnRace`, `nnCrashed`, plus les bases de bearoff. Formule du commentaire : `CLASS_CRASHED = « Contact, one side has less than 7 active checkers »`.

2. **Frontière CONTACT vs RACE** = position du pion le plus arriéré de chaque camp (`nBack`, `nOppBack`). Si les deux pions les plus arriérés se sont croisés (plus aucun contact possible) → `CLASS_RACE`, sinon `CLASS_CONTACT`.

3. **Frontière CRASHED** définie **verbatim par son auteur** Joseph Heled (liste bug-gnubg, 8 février 2012) : *« The number of active pieces has been arbitrarily set at 6, and the definition requires that you have at most 6 checkers not on points 1 or 2, accounting for the possibility of one checker from 2 sent back after the rest piled up. »* Le seuil 6 est explicitement « arbitraire » et choisi pour être **non cyclique** (une position issue d'un crash reste crashed), sinon les réseaux perdent en performance car « each net is trained only on its own kind of positions ».

4. **La littérature humaine est qualitative.** Fort consensus conceptuel (holding = 1 ancre en attente de tir ; backgame = 2+ ancres profondes ; blitz = attaque du jan adverse ; prime-vs-prime = deux amorces qui s'affrontent), mais peu de seuils. Les seuils numériques réellement sourcés portent sur le **videau**, pas sur la classification.

5. **Aucune mesure d'accord inter-classifieurs (kappa de Cohen) publiée** n'a été trouvée pour la classification en types de jeu. Les travaux académiques concernent l'apprentissage de la fonction de valeur (Tesauro/TD-Gammon, Berliner/BKG-SNAC), pas la classification en types. À dire franchement à l'utilisateur.

---

## Détails

### A. Le code gnubg (source primaire vérifiable)
gnubg emploie trois réseaux spécialisés : *« the contact net which is the main net for middlegame positions, the crashed net, and the race net »* (bkgm.com, *All About GNU Backgammon*). `ClassifyPosition` attribue la classe avant l'évaluation. L'énumération et son ordre (§Key Findings 1) sont confirmés sur `eval.h`. Ian Shaw complète la définition de « crashed » (même fil bug-gnubg) : une position est crashed quand **la plupart des pions d'un camp se sont entassés sur ses points 1 et 2**, lui laissant peu de flexibilité et aucun contrôle de son jan ni de l'outfield.

**Point crucial pour blunderDB :** ces classes existent uniquement pour router vers des réseaux entraînés chacun sur son type de position. Elles **ne correspondent pas** aux plans de jeu humains. blitz, holding game, prime-vs-prime, containment, mutual holding, ace-point game, backgames 1-2/1-3/… **tombent tous dans `CLASS_CONTACT`**. Si l'utilisateur veut ces étiquettes, il doit les définir lui-même — d'où l'arbre proposé.

Deux positions de référence figurent dans le fil bug-gnubg (Ian Shaw), utiles comme tests des classes CONTACT/CRASHED, avec Match ID `cAngAAAAAAAE` (match en 7 points) :
- **Crashed** : GNU Position ID `/z4AADBsuxsEAA` (X a empilé ses pions à l'avant, O crashé sur ses points bas).
- **Contact structuré** : GNU Position ID `sN0tADBsuxsEAA` (O conserve une bonne structure).

### B. eXtreme Gammon (XG)
XG encode l'état complet dans le **XGID** mais **sa classification interne n'est pas documentée publiquement** ; XG est closed-source et *« the details of its training are not publicly documented »* (arXiv, *PureTD*, 2026). XG affiche l'**EPC** (Effective Pip Count, dû à Trice) à côté du pip count sur son écran de coaching. XG segmente ses rapports par **décisions** (checker play / cube), non par taxonomie de types de position. **Aucun « position type » déterministe documenté** n'a été trouvé côté XG.

### C. BGBlitz (Frank Berger)
Comme gnubg, BGBlitz utilise des **réseaux spécialisés par phase** de jeu ; l'IA se nomme « TachiAI ». La littérature indépendante (arXiv *PureTD*) confirme cette philosophie de réseaux-par-phase pour la famille gnubg / Open Sage / BGBlitz / (probablement) XG, mais **BGBlitz ne publie pas ses seuils exacts**. BGBlitz intègre 500 positions tirées de 4 livres classiques comme cartes d'entraînement.

### D. Autres bots historiques et la leçon de Berliner
**BKG 9.8** (Hans Berliner, Carnegie-Mellon University) a battu **Luigi Villa le 15 juillet 1979 à Monte-Carlo**, dans un match en 7 points remporté **7-1** (ACM SIGART Bulletin, Berliner 1980 : *« On July 15, 1979 in Monte Carlo world history was made as a computer program for the first time beat a recognized world champion at his own game »*) ; match joué pour 5 000 $ devant ~200 spectateurs. Point directement pertinent pour blunderDB : Berliner avait d'abord **catégorisé les positions/coups** (running vs blocking) mais *« errors in comparing the two were often made »* aux frontières entre catégories ; il a alors abandonné les catégories dures au profit de **SNAC** (*Smoothness, Nonlinearity, Application Coefficients*), qui « warpe » l'espace pour un basculement **lisse** entre ~30 microstratégies simultanées. **Leçon : les frontières dures entre types produisent des erreurs de jugement** — d'où l'intérêt, pour blunderDB, d'étiquettes multiples et d'un flag d'ambiguïté plutôt qu'une classe unique tranchée. **TD-Gammon** (Tesauro) n'utilise aucune classification explicite en types.

### E. Définitions de la littérature (tableau par source)

| Type de jeu | Magriel *Backgammon* (1976) | Trice *Boot Camp* (2004) | Robertie / Woolsey | Glossaires bkgm.com / USBGF | gnubg |
|---|---|---|---|---|---|
| **Course / running game** | Amener tous ses pions au jan et sortir ; à jouer avec avance au pip | Chap. races & pipcount ; introduit l'EPC | Robertie : course pure, videau à +8 %/+9 %, prise −12 % | « Race » : phase sans contact | **`CLASS_RACE`** : pions arriérés croisés (SOURCÉ, code) |
| **Holding game** | Chap. « Holding Games » ; 1 ancre, frapper **dès que possible** ; employé quand on dérive derrière au pip | Chap. 6 « High Anchors And Holding Games » | Robertie : ~+15 % au pip pour un bon double d'un holding 5-point, prise facile | 1 ancre dans le jan adverse en attente de tir | `CLASS_CONTACT` |
| **Mutual holding** | — (dérive de l'holding) | — | Concept usuel : les deux camps tiennent une ancre haute | « both hold a point in opponent's home » | `CLASS_CONTACT` |
| **Back game** | Chap. « Holding Game and Backgame » ; **≥ 2 ancres**, frappe **tardive** (early hit futile) | Sections Backgame (Basic, Bear-In, Cube…) | Ortega (bkgm) : 1-2 gêne fort le bear-in mais manque souvent de timing ; 1-4/1-5/2-5 faibles | USBGF : *« two or more anchors in the opponent's home board »*, joueur « substantially behind » | `CLASS_CONTACT` |
| **Sous-types backgame (1-2, 1-3, 2-3, 2-4, 1-4, 3-4)** | — | — | Ortega : ancres rapprochées (1-2) fortes mais gourmandes en timing ; ancres écartées faibles | Nommés d'après les 2 points tenus dans le jan adverse | — |
| **Ace-point game** | — (holding profond) | Chap. 5 « The Defensive Ace Point » | Weak holding, gammon risk élevé | USBGF : tenir le point 1 adverse en attendant un tir au bear-in/off | `CLASS_CONTACT` (souvent `CLASS_CRASHED` en fin) |
| **Blitz / attacking** | Chap. « Priming and Blocking » (attaque) | Chap. 3 « Blitz! » | Attaque du jan, garder l'adversaire à la barre, viser le closeout | USBGF *Attacking Game* : hitter les blots dans son jan | `CLASS_CONTACT` |
| **Prime vs prime** | Chap. « Priming and Blocking » | Chap. « Prime-Vs-Prime », « Escape Or Crash? » | Le joueur **derrière** au pip est souvent favori (l'autre doit casser son amorce en premier) | USBGF : *« both players have a blockade of five or six points in a row »* | `CLASS_CONTACT` |
| **Containment** | Contrôle de l'outfield / timing | — | Woolsey (bkgm) : contenir le pion frappé en avançant l'amorce après un tir tardif | — | `CLASS_CONTACT` |
| **Crunch / crashed** | « Timing » (perte de timing → crunch) | Chap. « Escape Or Crash? » | Board crunché, perte de flexibilité | — | **`CLASS_CRASHED`** : ≤ 6 pions hors points 1-2 (SOURCÉ, Heled) |
| **Bear-in avec contact** | Bearing off against contact | Chap. « Backgame Bear-In » | Woolsey : bearing off against an anchor | — | `CLASS_CONTACT` / `CLASS_CRASHED` |

**Seuils numériques réellement sourcés** (tous liés au **videau**, pas à la frontière de type) :
- **Course** — Robertie : *« an 8% lead is usually enough to offer a good initial double. (The trailer need to be down no more than 12% to have a take.) »* ; formulation complète : **double à +8 %, redouble à +9 %, prise jusqu'à −12 %** du pip count ; règle optimale pour des courses d'environ **85 à 115 pips** (au-delà/en-deçà, ajuster). Table de Robertie (« Rule of 65 ») : 10 % d'avance ≈ 75 % de victoires, 14 %→80 %, 18 %→85 %, 24 %→90 %, 30 %→95 %.
- **Règle des 15 %** de Jacoby-Crawford (prise si retard ≤ 15 % du pip adverse), jugée **un peu trop libérale** par Kleinman (bkgm.com), surtout dans les longues courses (14 % peut être un pass en course moyenne, 12,5 % en longue course).
- **Holding game** — Robertie : il faut ~**+15 %** au pip pour un bon double dans un holding game 5-point typique, la prise restant facile.

### F. Format XGID (pour générer les positions-tests depuis blunderDB)
Structure : `XGID=<26 car. de plateau>:<cube>:<propriétaire cube>:<tour>:<dés>:<score X>:<score O>:<crawford>:<règles>:<longueur match>`.
Exemple réel : `XGID=-a-B--E-B-a-dDB--b-bcb----:1:1:-1:63:0:0:0:3:8` (tiré du manuel XG 2, p. 146).
- Les **26 caractères** couvrent : barre d'un camp + 24 points + barre de l'autre.
- `-` = point vide ; **minuscule** `a..p` = pions d'un camp (`a`=1 pion, `b`=2, …) ; **MAJUSCULE** `A..P` = pions du camp adverse (même échelle).
- `cube`=valeur (0=1, 1=2, 2=4…) ; `propriétaire`=−1/0/1 ; `tour`=1 ou −1 ; `dés` : `63`=6-3, `00`=pas encore lancés (décision de double) ; `crawford`=0/1 ; `longueur match`=0 pour partie d'argent.
Ce format permet à blunderDB de sérialiser ses 26 points directement.

### G. Absence de mesure d'accord inter-classifieurs
Aucune publication mesurant un **kappa** entre heuristiques de type de jeu (ou entre heuristique et jugement humain) n'a été trouvée. Les seules « segmentations par phase » publiées sont internes aux bots (les 3 classes gnubg ; les rapports d'erreur par décision de XG). **Conséquence pratique : la taxonomie de blunderDB sera une convention interne**, à valider par échantillonnage manuel, non par un standard existant.

---

## Arbre de règles proposé (ordonné, O(1), pseudo-code Go)

**Conventions.** `board[c][p]` = nb de pions du camp `c` au point `p`, `p ∈ [0..25]` (25 = barre, 0 = sortis), vu « du côté de X ». `backX` = plus grand `p` avec `board[X][p]>0`. « Ancre » de X = point avec `board[X][p] ≥ 2`. `ancresJanAdv(X)` = points 19..24 tenus par X (le jan adverse). `longuePrime(c)` = plus longue suite de points consécutifs faits. `pipDiff = pipX − pipAdv` (positif = X en retard). Ordre = **du plus spécifique au plus général**.

```go
// R1 — Terminé                                    [SOURCÉ: gnubg CLASS_OVER]
if off(X)==15 || off(O)==15 { return "over" }

// R2 — Course / no-contact                        [SOURCÉ: gnubg CONTACT/RACE]
// Les pions les plus arriérés des deux camps se sont croisés : aucun contact possible.
if backX + backO indiquent que les back-checkers ont franchi { return "race" }

// R3 — Bear-in avec contact                        [heuristique à calibrer ; concept sourcé Woolsey/Trice]
if maxPoint(X) <= 6 && ancreAdverseDansJan(X) {
    return principal:"bear-in avec contact", secondaire: selon ancre adverse
}

// R4 — Crashed / crunch                            [SOURCÉ: gnubg CLASS_CRASHED — Heled]
// « at most 6 checkers not on points 1 or 2 »
for c in {X,O} {
    if contact && checkersHorsPoints1et2(c) <= 6 { tag(c, "crunch/crashed") }
}

// R5 — Backgame (≥ 2 ancres dans le jan adverse)   [SOURCÉ concept+2 ancres ; seuil pipDiff NON sourcé]
if len(ancresJanAdv(X)) >= 2 {
    // sous-type = les deux points d'ancre exprimés en numérotation du jan adverse (1..6)
    // 24&23→"1-2" ; 24&22→"1-3" ; 23&22→"2-3" ; 23&21→"2-4" ; 24&21→"1-4" ; 22&21→"3-4"
    return principal:"backgame " + sousType
}

// R6 — Ace-point game                              [SOURCÉ concept USBGF ; seuilRetard heuristique]
if len(ancresJanAdv(X))==1 && board[X][24] >= 2 && pipDiff > seuilRetard {
    return principal:"ace-point game"
}

// R7 — Blitz (attaque du jan)                       [SOURCÉ concept ; seuils heuristiques]
if pointsFaitsJan(X) >= 3 && (board[O][25] > 0 || blotsAdversesDansJan(X) >= 1) {
    return principal:"blitz"
}

// R8 — Prime vs prime                               [SOURCÉ concept USBGF ; seuil 4 heuristique]
if longuePrime(X) >= 4 && longuePrime(O) >= 4 && pionPiege(X) && pionPiege(O) {
    return principal:"prime vs prime"
}

// R9 — Holding game (1 ancre haute) + mutual        [SOURCÉ concept Magriel/Robertie ; points nommés sourcés]
if len(ancresHautes(X)) == 1 {  // ancres hautes = points 18..21
    // sous-type : 20→"golden/20-point" ; 21→"21-point" ; 18→"bar-point/18" ; 22..24→"deep holding"
    if len(ancresHautes(O)) >= 1 { return principal:"mutual holding game" }
    return principal:"holding game " + sousType
}

// R10 — Containment / défaut                        [SOURCÉ concept containment: Woolsey]
if vientDeFrapper(X) && construitAmorceDevant(pionFrappe) { return "containment" }
return "contact indéterminé / early game"
```

**Justification de l'ordre :** on épuise d'abord les états sans contact (R1-R2, entièrement sourcés en code gnubg), puis les états de fin de partie très contraints (R3-R4), puis les plans définis par le **nombre et la profondeur des ancres** (R5-R6, R9), puis les plans « offensifs/structurels » (R7-R8), enfin le résiduel (R10). Chaque position est évaluée **des deux côtés** : X peut être en holding game pendant que O est en containment → produire une étiquette par camp. Prévoir **étiquette principale + secondaire** et un **flag d'ambiguïté** (voir ci-dessous).

---

## Cas ambigus à assumer
- **Holding game vs backgame** quand exactement 2 ancres mais l'une très avancée (ex. 20 + 24) : convention retenue = **backgame seulement si les 2 ancres sont dans le jan adverse (points 19-24)** ; sinon holding.
- **Blitz vs priming game** : recouvrement dès 3-4 points faits ; le « two-way forward game » est explicitement un **hybride** blitz/prime (Youngerman, bkgm.com) — accepter la double étiquette.
- **Race vs holding** quand le contact est marginal (une seule ancre profonde, `pipDiff` faible) : sensible au seuil de R2 ; documenter.
- **Backgame vs « two anchors » avec timing** : la viabilité dépend du **timing** (spares, board crunch), **non calculable de façon fiable en O(1)** ; étiqueter « backgame » sur la géométrie, laisser le timing à l'analyse moteur.
- **Transitions** : le type change après un coup → **recalcul systématique**, jamais d'édition manuelle (conforme à l'exigence « recalculable »).
- **Positions d'ouverture / début de partie** : n'appartiennent à aucun type → étiquette « early / contact indéterminé ».
- **Sensibilité aux seuils** : tout seuil heuristique (R3, R6, R7, R8) doit être un **paramètre nommé et versionné**, recalibrable sans réécrire le code.

---

## Positions-tests canoniques (XGID / IDs vérifiés)
- **Départ (contact/early)** — position initiale, à générer par blunderDB (format §F).
- **Crashed** — GNU Position ID `/z4AADBsuxsEAA`, Match ID `cAngAAAAAAAE` (7-pt) — **vérifié** (fil bug-gnubg, Ian Shaw). Type attendu : `CLASS_CRASHED` (R4).
- **Contact structuré / prime** — GNU Position ID `sN0tADBsuxsEAA`, Match ID `cAngAAAAAAAE` — **vérifié** (même fil). Type attendu : `CLASS_CONTACT` (R7-R10 selon structure).
- **Position de holding/anchor (exemple XG)** — `XGID=-a-B--E-B-a-dDB--b-bcb----:1:1:-1:63:0:0:0:3:8` (manuel XG 2, p. 146) — **cité tel quel** ; à re-vérifier dans blunderDB avant usage comme référence de holding.

⚠️ **Je ne cite aucun XGID que je n'ai pas pu vérifier.** Pour une couverture complète (une position par classe : race, ace-point, backgames 1-2…3-4, blitz, mutual holding, prime-vs-prime, containment), l'utilisateur doit **générer les XGID depuis blunderDB** puis vérifier chaque étiquette avec gnubg/XG. Les GNU Position IDs ci-dessus se convertissent en XGID via gnubg ou bgLog.

---

## Recommandations
1. **Étape 1 — répliquer exactement les 3 frontières SOURCÉES de gnubg** (`over` ; `race` via croisement des back-checkers ; `crashed` = ≤ 6 pions hors points 1-2). C'est reproductible, défendable et gratuit en performance. Marquer ces étiquettes « moteur-fidèles ».
2. **Étape 2 — ajouter la couche « plan de jeu humain »** (R3, R5-R10) comme **étiquettes secondaires**, chaque seuil non sourcé (R3, R6, R7, R8) portant le tag `heuristique_v1`.
3. **Étape 3 — calibrer par échantillonnage** : tirer 200-300 positions couvrant toutes les phases, les étiqueter à la main (idéalement 2 annotateurs), mesurer l'accord (**kappa de Cohen**) et ajuster les seuils. C'est le **seul** moyen de valider, puisqu'aucun standard externe n'existe.
4. **Toujours stocker deux étiquettes** (une par camp, plus principale/secondaire) **et un flag d'ambiguïté**, dans l'esprit du « smoothness » de Berliner — les frontières dures sont la principale source d'erreur.

**Seuils/benchmarks qui feraient changer ces recommandations :**
- Si un kappa inter-annotateurs < 0,6 sur une classe donnée → fusionner ou supprimer cette classe (frontière trop floue pour être utile).
- Si gnubg publie un jour une classification « plan de jeu » (peu probable) → l'adopter comme référence.
- Si l'utilisateur dispose déjà de l'analyse moteur (probas g/gg/bg) pour ≥ 80 % des positions → enrichir R6/R7 avec des seuils de probabilité de gammon (ex. blitz confirmé si P(gammon gagné) élevée), ce qui réduira l'ambiguïté ; mais l'arbre doit rester fonctionnel **sans** le moteur.

---

## Caveats
- **Les classes gnubg ne sont pas des plans de jeu** : ne jamais les présenter à l'utilisateur final comme « holding/backgame/blitz ». `CONTACT` regroupe presque tous les plans humains.
- Le **corps exact (if-statements littéraux) de `ClassifyPosition`** n'a pas pu être extrait verbatim (endpoints raw GitHub/Savannah non fetchables dans cette session) ; l'énumération, le prototype et les **seuils** sont confirmés par l'auteur (Heled) et le commentaire de `eval.h`, mais **la ligne de code exacte du test CONTACT/RACE et de `isCrashed` n'est pas citée mot pour mot**. Pour l'obtenir : `git.savannah.gnu.org/cgit/gnubg.git/tree/eval.c` (chercher `ClassifyPosition`, ~ligne 630).
- **Aucun seuil « frontière entre types humains » n'est sourcé** : tous les seuils de R3, R5(pipDiff), R6, R7, R8 sont **proposés** et à calibrer. Seuls les seuils du **videau** (Robertie : +8 %/+9 %/−12 % ; Jacoby-Crawford 15 %) et l'exigence **≥ 2 ancres** (backgame) sont sourcés — et ils ne délimitent pas les types entre eux.
- **XG et BGBlitz** ne publient pas de classification interne exploitable ; toute affirmation sur leurs « position types » serait spéculative.
- **Les XGID non vérifiés ne doivent pas être copiés tels quels** ; les deux GNU Position IDs cités sont vérifiés, l'XGID du manuel XG est cité de seconde main.