# Mesurer la force d'un joueur de backgammon sans dépendre de la définition d'une « décision » : métriques, littérature et protocole d'évaluation sur les données BMAB

La meilleure alternative au PR est la **perte totale de probabilité de gain du match (MWC) par match, divisée par √N (N = longueur du match) et convertie en différence d'Elo attendue face au moteur**. Elle ne dépend d'aucune définition de « décision », se calcule sur un seul match et repose sur une identité exacte en espérance. Il faut la compléter par une perte par tour (dénominateur = nombre de tours, fixé par les règles) et par un indice de vraisemblance de type Regan pour corriger la difficulté. Le classement BMAB devrait être globalement retrouvé, mais avec des inversions explicables, surtout chez les joueurs « cube-lourds » et chez ceux dont les adversaires ou les longueurs de match sont atypiques.

## TL;DR

- **Le défaut du PR tient à son dénominateur, pas à son numérateur.** XG ne compte que les décisions « non évidentes » (seuils arbitraires de 0,001 et 0,200 d'équité). GNU BG ne compte que les coups non forcés et les décisions de cube « proches » (à 0,25 près). Snowie divise par les coups des deux joueurs. Une métrique dont le dénominateur est la longueur du match ou le nombre de tours, et qui exprime les pertes en MWC, élimine cette dépendance et ce levier de manipulation.
- **Proposition principale.** Calculer L = Σ des pertes de MWC du joueur (pions + cube + prises/refus) sur le match, puis la différence d'Elo intrinsèque D = (2000/√N)·log10(q/(1−q)) avec q = 0,5 − L, inversion directe de la formule FIBS. Compléments : perte par tour en mEMG, ratio à une perte de référence ajustée à la difficulté, et un paramètre de sensibilité estimé par maximum de vraisemblance (à la Regan), dans lequel les coups forcés ou triviaux ne pèsent automatiquement rien.
- **Sur BMAB, il faut s'attendre à une forte corrélation de rang avec le « Last 300 exp PR », mais pas à une identité.** C'est une hypothèse à tester : aucune étude publiée ne recalcule le classement BMAB avec d'autres métriques. Les écarts viendront de la règle BMAB « plus bas de deux analyses XG », des différences de moteur XG/GNU BG, du poids du cube dans le dénominateur (le PR cube des Grandmasters BMAB vaut 1,5 à 1,8 fois leur PR pions), de la composition des longueurs de match et de la force des adversaires. Avec la variance observée, il faut une quarantaine de matchs de 7 points par joueur pour départager des écarts de l'ordre de 0,5 PR.

## Key Findings

### 1. Ce que mesurent vraiment PR, ER et « Snowie ER »

- **XG (PR).** Le PR vaut 500 × équité perdue / nombre de décisions non évidentes. D'après le manuel XG2 (révision 1.01, septembre 2011), « A decision is a checker move or a cube double that is considered non-obvious by the computer ».\[1\] Sont considérés comme évidents :
  - les coups forcés ;\[1\]
  - les coups où le meilleur et le pire choix diffèrent de moins de 0,001 ;\[1\]
  - les situations de cube où l'équité sans doubler dépasse de 0,200 celle après le double (« obvious no double ») ;\[1\]
  - les situations où elle dépasse de 0,200 celle après un refus (« obvious too good »).\[1\]
  
  Les doubles effectifs et les prises sont toujours comptés.\[2\] Le facteur 500 (au lieu de 1000) est un compromis historique pour rester dans les ordres de grandeur de Snowie.\[3\]
- **GNU BG (ER mEMG).** Le manuel définit l'erreur par décision comme le total divisé par « the number of non-trivial decisions (i.e., the sum of unforced moves and close or actual cube decisions) ».\[4\]\[5\] Une décision de cube est « proche » si les équités pertinentes sont à moins de 0,25 l'une de l'autre ou si la position est « too good ».\[6\] Toujours selon le manuel, qui ne nomme pas l'auteur de l'étude : « An investigation of approximately 300 matches showed the on average the GNU Backgammon error rate with be 1.4 times higher than your Snowie 4 error rate ».
- **Snowie.** Total des erreurs du joueur divisé par le nombre de coups **des deux joueurs**, coups forcés compris. C'est pourquoi le chiffre est environ deux fois plus bas.\[3\]\[7\]\[8\]
- **Conséquence.** Les trois mesures ont le même numérateur (équité normalisée perdue) mais trois dénominateurs différents, dont deux dépendent de seuils propres au moteur.\[3\] Selon le fil twoplustwo « Error rating discrepancy between GNU Backgammon and Backgammon Galaxy », « if you have a greater than average number of forced moves, Gnu's error rate will be lower, if you have fewer than average forced moves, XG's error rate will be lower » ; XG écarte en outre les coups « non-forced but meaningless » (courses décidées à 100 %). Le même fil rapporte : « On average XG thinks that GNUbg plays with a 0.5 PR and GNUbg that XG does with an error rate of 1 or thereabouts ».

### 2. Les critiques documentées, qui fondent le cahier des charges

- **Douglas Zare, « Normalizing Errors » (bkgm.com).** Il montre que minimiser l'erreur selon la méthode Snowie revient à minimiser l'erreur totale, alors que minimiser l'erreur selon la méthode GNU peut ne pas le faire. Doubler un peu tôt pour simplifier la suite réduit le nombre de décisions de cube « non forcées » et peut donc augmenter l'ER cube.\[8\]\[9\] Il montre aussi que la normalisation EMG est gravement biaisée selon le score. Au DMP, il est « particulièrement facile » d'obtenir un faible taux d'erreur : de nombreux joueurs « advanced » y affichent un taux « world-class ».\[8\] Il suggère de multiplier les erreurs au DMP par environ 2 (0,390/0,195).\[10\]
- **Bob Koca (bgonline, juillet 2024, fil « Does XG's definition of PR annoy you? »).** Il relève :
  1. des définitions arbitraires de ce qui compte comme décision ;\[11\]
  2. l'imperfection du moteur, qui pénalise les longues parties de backgame et récompense ceux qui jouent « dans le style du bot » ;\[11\]
  3. des moyens artificiels de baisser son PR : prendre un refus serré pour ajouter des décisions dans une course simple, ou jouer 3/off plutôt que 2/off 1/off en fin de bear-off pour créer une décision supplémentaire ;\[11\]
  4. l'écart entre jeu pratique contre un adversaire réel et jeu théorique ;\[11\]
  5. le fait que le PR pondère de la même façon une erreur avec le cube à 8 et une erreur avec le cube à 1.\[11\]
- **Chuck Bower, « Quantifying Backgammon Skill » (GammOnLine, septembre 2001).** C'est la référence statistique la plus précise. L'écart-type de l'erreur **par coup** vaut 9,38 mppm pour Jellyfish et 23,40 mppm pour un humain typique (échelle Snowie). L'erreur-type d'un taux sur N coups vaut s.d./√N. Sur 81 matchs de 7 points (17 473 coups), il obtient JF = 1,502 ± 0,139 et Bower = 5,042 ± 0,347 (IC 95 %). Il souligne aussi un biais systématique « robot bias » qu'il estime grossièrement à 0,5 mppm, et le fait que les distributions d'erreur sont asymétriques, avec des valeurs aberrantes.\[12\]

### 3. Ce que l'on sait du lien entre PR, Elo et résultats

- **PR et Elo.** Xavier Dufaure de Citres a établi une relation empirique de **3 points de PR = 100 points d'Elo**.\[13\] Rick Janowski rapporte une « excellent correlation » avec cette relation sur les données des tournois IIBGF, pour les Elo supérieurs à 1500.\[14\] Il note aussi que la formule FIBS, fondée sur une marche aléatoire, ignore gammons, efficacité du cube et Crawford, mais reste cohérente avec les tables MET « Fish » pour les matchs de 7 points et plus.\[13\]\[14\] Le tableau d'équivalences d'XG associe par exemple PR 2,5-5,0 à un Elo de 2077-2162.\[15\]
- **Formule FIBS.** P(gain) = 1 / (1 + 10^(−D·√N/2000)), où D est la différence d'Elo et N la longueur du match (FAQ bkgm.com).\[16\]
- **Prédiction des résultats (Peter Ellis, « Skill v luck in determining backgammon winners », free range statistics, 19 mars 2016 : « approximately 650 matches » de 1 à 11 points sur FIBS et grid.gammon, analysés par XG).**
  - la chance seule prédit le vainqueur à 97,9 % ;\[17\]
  - l'écart d'erreur net seul, à 65,0 % ;\[17\]
  - les deux combinés, à 99,7 %.\[17\]
  
  La qualité de jeu explique donc une part réelle mais minoritaire de chaque résultat individuel. Dans le même billet, Ellis écrit : « my average PR hovers around 11 and Elo rating on FIBS currently at 1800 whereas XG thinks I should be only 1663 ». La correspondance PR/Elo dépend du pool de joueurs.
- **Tables PR → MWC (Wayne Joseph, reprises par backgammon101).** Dans un match en 25 points, un écart de 3 PR ne donne que 64/36 au meilleur joueur.\[18\]
- **Réduction de variance.** GNU BG publie un « luck adjusted result » (résultat réel + chance totale), qualifié de « variance reduction of skill » d'après Zare (« Hedging Toward Skill »).\[5\] Il est décrit comme une mesure non biaisée de la force relative des deux joueurs.\[4\] À l'inverse, la statistique « MWC against current opponent » (50 % − erreurs propres + erreurs adverses) est biaisée en faveur du bot analyseur.\[5\]\[19\] Selon David Montgomery (« Variance Reduction », GammOnLine, février 2000), une rollout réduite en variance de sa position d'exemple est typiquement aussi précise qu'une rollout ordinaire sur huit fois plus de parties, et en général la réduction de variance fait valoir chaque partie « about twenty-five regular games ». Une prépublication de T. Zoidis (2025, rxiv.org, non relue par les pairs) avertit que la réduction de variance fausse les mesures si l'objet étudié est précisément l'erreur.\[20\]

### 4. Les analogues aux échecs

- **Ken Regan et Guy Haworth, « Intrinsic Chess Ratings » (AAAI 2011).** Ils modélisent la probabilité de choisir un coup en fonction de sa valeur relative aux autres options, avec deux paramètres : la **sensibilité s** et la **consistance c**. Ces paramètres sont ajustés par régression sur des parties de joueurs dont l'Elo est connu, ce qui donne un « Intrinsic Performance Rating » (IPR) avec intervalles de confiance.\[21\]\[22\] Ils concluent à une relation lisse entre l'Elo et la qualité intrinsèque des coups.\[23\] Dans leur compendium (2012), un échantillon à Elo 2700 donne s = 0,078, c = 0,502, IPR 2690 (intervalle 2σ de 2648 à 2731) sur 7 032 coups.\[24\]
- **Lichess.** Lichess convertit l'évaluation en Win% (50 + 50·(2/(1+exp(−0,00368208·cp)) − 1)), puis calcule une « accuracy » par coup : 103,1668·exp(−0,04354·ΔWin%) − 3,1669. Lichess précise lui-même qu'il n'existe aucune manière universellement admise de calculer l'accuracy.\[25\] La critique classique (TalkChess) est que l'accuracy dépend du jeu de l'adversaire : des imprécisions commises dans une position déjà gagnante coûtent peu de Win%.\[26\] C'est exactement l'analogue de la compression des erreurs en MWC dans les positions décidées. Au backgammon, cette compression est justement une propriété voulue, car elle reflète l'enjeu réel.

## Details

### A. Propriétés souhaitables (cahier des charges argumenté)

| # | Propriété | Justification | Test opérationnel |
|---|---|---|---|
| P1 | Indépendance vis-à-vis du comptage des décisions | Seuils XG (0,001/0,200) et GNU (0,25) arbitraires, cause des écarts entre moteurs | La métrique est inchangée si l'on modifie les seuils « trivial » ou « proche » |
| P2 | Calculabilité sur un seul match, avec IC | Contrainte du cahier des charges ; un match = environ 100 à 250 tours par joueur | IC par bootstrap sur les parties |
| P3 | Invariance à la longueur du match | Les matchs BMAB vont de 7 à 17 points et plus ; le PR mélange des scores « faciles » (DMP) | Pente nulle de la métrique en fonction de N, à joueur fixé (modèle mixte) |
| P4 | Pondération par l'enjeu réel | Critique (v) de Koca : erreur avec le cube à 8 contre cube à 1 ; distorsions EMG selon le score (Zare) | Unités en MWC, sans normalisation EMG |
| P5 | Robustesse à la difficulté des positions | Le style de l'adversaire et les dés imposent des positions plus ou moins difficiles | Corrélation résiduelle nulle avec un indice de difficulté |
| P6 | Robustesse à la force et au style de l'adversaire | Exemple BMAB : PR adverse moyen de 9,01 pour Bynell contre 4,97 pour Myhr | Pente nulle en fonction du PR ou de l'Elo adverse |
| P7 | Cohérence money/match | Même formule en équité (money) et en MWC (match) | Même ordre de joueurs sur des sous-ensembles comparables |
| P8 | Additivité | Agrégation multi-matchs sans arbitraire (pas de moyenne de ratios) | Numérateur et dénominateur sommables, ou log-vraisemblance additive |
| P9 | Variance faible, IC calculables | Asymétrie, valeurs aberrantes (Bower) | Bootstrap, estimateurs robustes |
| P10 | Interprétabilité | Lecture en Elo ou en % de MWC plutôt qu'en « PR » | Conversion explicite en probabilité de gain |
| P11 | Stabilité entre versions et réglages moteur | 0-ply, 2-ply, rollouts ; XG contre GNU | Kendall τ entre classements obtenus avec différents réglages |
| P12 | Résistance à la manipulation | Exemples de Koca (prise artificielle, bear-off ralenti) | Aucun gain de score par ajout de décisions triviales |
| P13 | Validité prédictive | La finalité d'une mesure de force | Log-loss/Brier sur des matchs tenus à l'écart |

### B. Métriques proposées (formules)

**Notations.** Match m de longueur N (N = ∞ en money). Pour le joueur p, T_p(m) est l'ensemble de ses tours, c'est-à-dire chaque fois qu'il est au trait, avant l'éventuel double. Cette définition est **fixée par les règles**, pas par le moteur : elle inclut les coups forcés, les fermetures (dance) et le Crawford. Toutes les pertes sont cubeful, calculées par GNU BG en MWC (match) ou en équité × valeur du cube (money) :
- c_t : perte de cube avant le jet = max(MWC_ND, MWC_D) − MWC(action choisie), avec MWC_D = min(D/T, D/P) du point de vue du doubleur ;
- r_t : perte de réponse (prise ou refus) du joueur qui reçoit un double ;
- k_t : perte de pions = MWC(meilleur coup) − MWC(coup joué).

Une non-décision (coup forcé, cube mort, position triviale) contribue 0 par construction. **Le numérateur ne dépend donc d'aucune définition de décision ; seul le dénominateur en dépend, et c'est lui qu'il faut changer.**

**M1 — Perte totale de MWC et Elo intrinsèque (métrique principale).**
- L_p(m) = Σ_t (c_t + k_t) + Σ r_t, en points de MWC.
- Justification : sous jeu parfait, la MWC est une martingale. L'espérance du résultat vaut donc exactement 0,5 − E[L_p] + E[L_opp]. GNU BG utilise la même identité pour sa statistique « MWC against current opponent ».\[5\] Contre un adversaire parfait, le joueur aurait donc q = 0,5 − L_p de chances de gagner.
- Elo intrinsèque face au moteur : **D_p(m) = (2000/√N)·log10(q/(1−q))** (négatif), inversion de la formule FIBS.
- Linéarisation : D ≈ −3474·L/√N. La bonne normalisation par la longueur est donc **L/√N**, et non L/N : à force égale, la perte totale de MWC d'un joueur croît comme √N.
- Différence entre deux joueurs : D_p − D_q, directement convertible en probabilité de gain pour une longueur quelconque.
- Ordre de grandeur, à valider empiriquement : un joueur à environ 5 de PR sur un 7 points perd typiquement quelques points de MWC, ce qui donne D de l'ordre de −100 Elo face au moteur. C'est compatible en ordre de grandeur avec la règle « 3 PR = 100 Elo ».
- Avantages : P1, P2, P4, P8 (sommer les L et les √N par match), P10, P12. Il n'y a pas de dénominateur à gonfler : une prise artificielle est comptée au prix de sa perte réelle, rien de plus.
- Inconvénients : forte variance (dominée par quelques grosses erreurs) ; dépendance à la MET (utiliser la même partout, par exemple Kazaross-XG2 si disponible) ; validité de la loi en √N approximative sous 7 points et autour du Crawford (Janowski) ; pour le money game, remplacer par une perte d'équité par partie, sans conversion Elo directe.

**M2 — Perte par tour (« PR à dénominateur fixé par les règles »).**
- PT_p(m) = 1000 · Σ_t e_t / |T_p(m)|, en mEMG par tour, avec e_t la perte en EMG. Variante en MWC : 100·L_p/|T_p| (en % de MWC par tour).
- C'est l'esprit de Snowie (on compte tout, coups forcés compris), mais d'un seul côté. Le dénominateur ne dépend d'aucun seuil moteur.
- Avantages : comparable au PR en ordre de grandeur (environ PR × 2 × fraction de décisions non triviales ; calibrer par régression) ; décomposable pions/cube avec le même dénominateur.
- Inconvénients : faible incitation à allonger les parties (plus de tours faciles), bien moindre que l'incitation de Koca ; hérite des distorsions EMG si on garde l'EMG (préférer la variante MWC).

**M3 — Ratio à une perte de référence ajustée à la difficulté.**
- Pour chaque tour, on calcule la perte attendue d'un joueur de référence : d_t = Σ_i π_ref(i | s_t)·Δ_i, avec π_ref(i) ∝ exp(−Δ_i/τ), où Δ_i est l'écart de MWC du candidat i au meilleur coup (cube inclus : ND, D/T, D/P ; prise/refus).
- Indice : **DR_p(m) = Σ_t ℓ_t / Σ_t d_t** (1 = joue comme la référence), ou excès **EX_p(m) = Σ_t (ℓ_t − d_t)/√N**.
- τ est calibré une fois pour toutes pour que la population BMAB poolée ait DR = 1, ou pour reproduire un joueur de PR 5.
- Avantages : P5 et P6, car un adversaire qui crée des positions difficiles fait monter Σd_t autant que Σℓ_t ; les positions triviales ont d_t ≈ 0 et ne pèsent rien, sans seuil.
- Inconvénients : dépend du modèle de référence et de la liste de candidats stockée par GNU BG (filtres de coups) ; un ratio est instable sur un match très facile ; il faut fixer un plancher Σd_t ou utiliser EX.

**M4 — Indice de vraisemblance façon Regan (IPR backgammon).**
- Modèle : P_σ(i | s_t) ∝ exp(−(Δ_i/σ)^c), avec c fixé sur la population et σ estimé **par match** par maximum de vraisemblance : σ̂_p(m) = argmax Σ_t log P_σ(coup joué | s_t). L'IC s'obtient par l'information de Fisher ou par profil de vraisemblance.
- Propriété clé pour P1 : un coup forcé a une vraisemblance de 1 quel que soit σ, et un coup trivial (Δ des alternatives énormes) a une vraisemblance d'environ 1. Ils n'apportent **aucune information**, sans qu'il faille les exclure par un seuil. Le « poids » de chaque décision est l'information de Fisher qu'elle apporte.
- Conversion : régresser σ̂ (ou log σ̂) sur l'Elo BMAB ou le PR des joueurs à forte expérience, comme Regan l'a fait pour l'Elo FIDE.
- Avantages : statistiquement le plus efficace ; IC naturels ; ajustement à la difficulté intégré ; additif en log-vraisemblance.
- Inconvénients : suppose que les erreurs suivent un modèle de choix (humain fatigué, gaffes isolées : prévoir un mélange « gaffe/attention ») ; sensible à la troncature des listes de candidats ; moins intuitif.

**M5 — Résultat ajusté de la chance (validation externe, pas mesure absolue).**
- LAR_p(m) = résultat réel + chance totale (sortie standard de GNU BG).\[5\] En espérance, LAR_p − 0,5 ≈ L_opp − L_p.
- C'est une cible de validation à faible variance pour M1 : la cohérence entre « écart d'erreurs » et « LAR » teste la justesse du moteur.
- Inconvénient : mesure relative (dépend de l'adversaire) et biais si le moteur se trompe sur la chance.

**Séparer pions et cube sans définir une « décision de cube ».**
- Sommer séparément L^pions = Σ k_t et L^cube = Σ (c_t + r_t), avec le **même** dénominateur (|T_p| ou √N).
- Si un dénominateur propre au cube est souhaité, utiliser un critère de règle et non de moteur. **Opportunités d'accès au cube** = tours où le joueur est au trait avec un cube centré ou en sa possession, hors partie Crawford et hors cube mort par le score. **Doubles reçus** = nombre de fois où le joueur doit répondre.
- Ce choix compte. Sur les fiches BMAB, le PR cube est bien supérieur au PR pions : Myhr 4,86 contre 3,17 ; Kristensen 5,53 contre 2,95\[27\]\[28\] ; Bynell 7,40 contre 4,10. Avec le PR, le poids relatif du cube dans le chiffre global dépend entièrement du nombre de décisions de cube comptées. Avec M1 ou M2, il est fixé par l'enjeu réel.

**Synthèse des métriques face aux propriétés.**

| Métrique | P1 | P2 | P3 | P4 | P5/P6 | P8 | P11 | P12 | Interprétation |
|---|---|---|---|---|---|---|---|---|---|
| PR XG / ER GNU | ✗ | ✓ | ✗ (DMP) | ✗ (EMG) | ✗ | ~ | ✗ | ✗ | familière |
| M1 L/√N → Elo | ✓ | ✓ | ✓ (modèle √N) | ✓ | ✗ | ✓ | ~ | ✓ | Elo, P(gain) |
| M2 perte/tour | ✓ | ✓ | ~ | ✓ (variante MWC) | ✗ | ✓ | ~ | ~ | proche du PR |
| M3 ratio difficulté | ✓ | ✓ | ✓ | ✓ | ✓ | ~ | ~ | ✓ | « × la référence » |
| M4 IPR σ̂ | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ | ~ | ✓ | après calibration |
| M5 LAR | ✓ | ✓ | ✓ | ✓ | ✗ (relatif) | ✓ | ✓ (non biaisé) | ✓ | relatif |

**Variance et taille d'échantillon (ordre de grandeur, déduit de Bower).**
- Avec 23,4 mppm d'écart-type par coup (échelle Snowie, proche de l'échelle PR) et environ 224 coups (deux joueurs) par match de 7 points, l'erreur-type d'un **seul** match est d'environ 1,5 PR. C'est trop pour classer des joueurs sur un match : un match isolé situe un joueur dans une bande (Master contre Grandmaster), pas dans un rang.
- Sur 300 EP (environ 40 matchs de 7 points), l'erreur-type tombe à environ 0,25 PR, ce qui suffit à distinguer des écarts de l'ordre de 0,5 PR.
- La distribution est asymétrique : utiliser un bootstrap par parties (comme l'article PureTD, 2026, qui rééchantillonne au niveau de la partie), éventuellement winsorisé.\[29\]

### C. Données BMAB : ce qui est publié et comment c'est calculé

- **Méthodologie officielle (page About, mise à jour le 7 juin 2024).**
  - Analyse avec « the current version of eXtreme Gammon (XG) using the following settings: Standard World Class Analysis but with "Gigantic" search interval, 15x8 bearoff database, with resignation errors counting ».\[30\]
  - « We analyse each match at least twice and use the match file with the lowest overall PR. »\[30\]
  - Matchs live, chronométrés, pré-désignés, d'au moins 7 points, enregistrés en vidéo et transcrits.\[30\]
  - EP = somme des longueurs de match.\[30\]
  - PR = moyenne sur les matchs les plus récents jusqu'à 300 EP (le match à la frontière compte en entier).\[30\]
  - Un titre n'est jamais perdu ; les joueurs inactifs depuis 18 mois sont retirés du tableau PR.\[30\]
  - Les résultats publiés comprennent PR moyen, taux de victoire et Elo.\[30\] Les données enregistrées « will become public domain ».\[30\]
- **Seuils de titres (page d'accueil historique, 2014-2018, peut-être datée).** S1 ≤ 2,00 (≥ 300 EP) ; S2 ≤ 2,25 (≥ 300) ; S3 ≤ 2,50 (≥ 250) ; G1 ≤ 3,00 (≥ 200) ; G2 ≤ 3,50 (≥ 150) ; G3 ≤ 4,00 (≥ 100).\[31\] La synthèse USBGF donne : Super Grand Master ≤ 2,5 ; Grand Master ≤ 4 ; Master ≤ 6,5 ; Advanced ≤ 10 ; Intermediate ≤ 16.\[32\]
- **Ce que BMAB ne documente pas.** La formule de son Elo (les fiches montrent des « Rating » autour de 1500 et des « Peak rating (>400 exp.) », ce qui suggère un Elo de type FIBS, mais c'est une inférence), ni la définition XG exacte du PR par match pondéré par EP. Selon un échange rapporté sur twoplustwo, on additionne les erreurs sur les 300 derniers EP puis on multiplie par 500, ce qui revient à pondérer par décisions et non par match.\[33\]
- **Échantillon de fiches joueurs (relevé 2025-2026, dates hétérogènes).**

| Joueur | Titre | Last 300 exp PR | Expérience / matchs | PR global (pions / cube) | PR adverse | Victoires |
|---|---|---|---|---|---|---|
| Masayuki Mochizuki | S2 | 2,7636 | 4278 EP | — | — | —\[34\] |
| Ryan Rebelo | G1 | 3,0929 | 631 EP, 83 matchs | 3,3384 (3,0071 / 5,3404) | 5,1675 | 49,4 % |
| Thomas Myhr | G1 | 3,1005 | 4783 EP, 414 matchs | 3,4377 (3,1701 / 4,8605) | 4,9671 | 52,0 %\[27\] |
| Johan Moazed | G2 | 3,2119 | 635 EP | — | — | —\[35\] |
| Elias Kritikos | G2 | 3,3370 | 1694 EP | — | — | —\[36\] |
| Dmitriy Obukhov | G2 | 3,3695 | 3265 EP, 345 matchs | 3,5798 (3,2708 / 5,2166) | 6,7444 | 58,0 %\[37\] |
| Steen Mikkelsen | G1 | 3,3714 | 1897 EP | — | — | — |
| Marty Storer | G1 | 3,5543 | 3075 EP | — | — | — |
| Thomas Kristensen | G1 | 3,5679 | 2243 EP | 3,3463 (2,9519 / 5,5251) | 5,1866 | —\[28\] |
| Naoki Iketani | G3 | 3,7127 | 1721 EP, 125 matchs | 3,8687 | — | —\[38\] |
| Karsten Bredahl | G2 | 3,7320 | 4249 EP | — | — | — |
| Petko Kostadinov | G2 | 3,8330 | 2690 EP | — | — | —\[39\] |
| Art Benjamin | G3 | 3,8806 | 824 EP | — | — | —\[40\] |
| Volker Sonnabend | G3 | 4,5114 | 833 EP, 90 matchs | — | — | —\[41\] |
| Johan Bynell | G3 | 4,6604 | 3276 EP, 318 matchs | 4,6141 (4,1024 / 7,4006) | 9,0118 | 61,3 %\[42\] |

- **Lecture de ce tableau.**
  - Les titres ne sont pas un classement : Mochizuki est S2 avec un PR courant de 2,76, Bynell est G3 à 4,66. Il faut comparer au « Last 300 exp PR » (ou au PR global), pas au titre.
  - L'adversité varie du simple au double (PR adverse de 4,97 à 9,01). Le taux de victoire le plus élevé de l'échantillon (Bynell, 61,3 %) va avec le PR le plus élevé et les adversaires les plus faibles. C'est l'illustration directe de P6.
  - Selon l'article PureTD, le titre de Super Grandmaster n'est détenu que par « deux ou trois » joueurs vivants.\[29\] Je n'ai pas pu obtenir la liste officielle complète des titres ni le tableau de classement (pages non accessibles à l'outil), ce qui laisse une lacune pour la reconstruction du classement de référence.
- **Accès aux données.** L'utilisateur dispose des .xg. Le format XG est public depuis 2014 (documentation publiée par XG).\[43\] D'autres matchs de haut niveau sont téléchargeables en .xg et .mat sur la base itikawa.com, utile comme échantillon externe.\[44\]

### D. Protocole d'évaluation reproductible

**Étape 1 — Conversion XG → GNU BG.**
- GNU BG (dernière version 1.08.003) **n'importe pas nativement les .xg**. Sa liste d'import couvre .bgf, .gam, .mat, .pos, .sgf, .sgg, .tmg et .txt Snowie.\[45\] Le mainteneur Philippe Michel l'avait confirmé dès 2013 (« It doesn't »),\[46\] et la version 1.08.002 n'a ajouté que le collage d'XGID.\[47\]
- Outils : xgdatatools de Michael Petch, en Python (copie maintenue sur GitHub par oysteijo), pour lire les .xg et les convertir en .mat\[48\]\[49\] ;\[50\] ou le paquet Python gammonview (outil xg2gva, lecture .xg sans dépendances).\[51\] Éviter l'export .mat d'XG ≥ 2.02, qui écrit une entrée « Set Pos » non standard que GNU BG n'importe pas.\[52\]
- Contrôles : rejouer chaque match dans GNU BG et vérifier légalité des coups, dés, score final et vainqueur contre les métadonnées BMAB. Journaliser et exclure les matchs illégaux plutôt que de les corriger silencieusement. Conserver l'analyse XG embarquée dans le .xg, si présente, pour recalculer le PR XG par match et le comparer au PR BMAB publié (test de fidélité de la conversion).

**Étape 2 — Analyse GNU BG.**
- Référence : analyse 2-ply (« World Class ») pour pions et cube, filtres de coups larges, base de bear-off complète, même MET pour tous les matchs.
- Robustesse (P11) : réanalyser (a) tout le corpus en 0-ply, (b) un sous-échantillon stratifié en 3-ply ou 4-ply, (c) les positions où |Δ| > 0,02 entre les deux meilleurs candidats, ou où une erreur > 0,04 est détectée, en rollouts courts. C'est l'équivalent de la seconde passe XG ; la norme danoise utilise au contraire une analyse « one-pass » directement au niveau le plus fort.\[53\]
- Automatisation : lancer gnubg sans interface avec un script Python (import, « analyse match », extraction de la structure du match analysé : candidats et équités par tour, cube, chance). Exporter une table « un tour = une ligne » : joueur, score, cube, MWC des candidats, coup joué, ℓ_t, chance, partie.

**Étape 3 — Métriques par match.** Calculer M1 (L, L/√N, D), M2, M3, M4 et M5 pour chaque joueur et chaque match, avec IC par bootstrap sur les parties (B ≥ 2000), les deux joueurs d'une même partie étant rééchantillonnés ensemble.\[29\]

**Étape 4 — Agrégation par joueur.**
- Estimateurs poolés (Σ numérateurs / Σ dénominateurs) et non moyenne de ratios.
- Modèle mixte : y_pm = θ_p + β₁·f(N_m) + β₂·force adverse + β₃·difficulté moyenne + ε_pm, avec effet aléatoire joueur (rétrécissement des petits échantillons) et IC sur θ_p.
- Deux fenêtres : toute la carrière, et « 300 derniers EP » pour une comparaison à l'identique avec BMAB.

**Étape 5 — Comparaison au classement BMAB.**
- Spearman ρ et Kendall τ (avec IC bootstrap sur les joueurs) entre θ_p et le Last 300 exp PR, puis le PR global, restreints aux joueurs ≥ 300 EP. Analyse stratifiée par tranche (SGM/GM contre Master et en dessous).
- Analyse des discordances : lister les joueurs dont le rang change de plus de k places et les expliquer par la part du cube, le PR adverse, la composition des longueurs et la part de positions « faciles ».

**Étape 6 — Fiabilité.**
- Split-half (matchs pairs contre impairs, puis correction de Spearman-Brown) ; test-retest année n contre n+1 ; ICC.
- Une métrique est meilleure que le PR, à variance comparable, si elle a une fiabilité supérieure **pour un même nombre de matchs**.

**Étape 7 — Validité prédictive.**
- Découpage chronologique. Les métriques calculées avant la date t prédisent les matchs après t via P(gain) = 1/(1 + 10^(−ΔD·√N/2000)).
- Comparer log-loss et Brier à trois bases : Elo BMAB, PR BMAB converti (3 PR = 100 Elo), et un modèle sans information.
- Cible secondaire à faible variance : le LAR des matchs futurs.

**Étape 8 — Sensibilités.** Pentes de chaque métrique en fonction de N, du PR ou de l'Elo adverse, de la part de parties DMP ou Crawford et du niveau d'analyse (0-ply/2-ply/rollouts) ; Kendall τ entre les classements obtenus avec XG PR, GNU ER et chaque métrique.

## Recommendations

1. **Adopter M1 (L/√N → Elo intrinsèque) comme métrique principale, et M2 en variante MWC comme indicateur grand public.** M1 répond directement à l'objection sur la définition des décisions et se lit en probabilité de gain. Publier systématiquement l'IC par match.
2. **Construire M4 (σ̂ de type Regan) comme métrique de recherche.** C'est la seule qui règle à la fois la difficulté des positions et le poids des décisions triviales sans aucun seuil. La calibrer sur les joueurs BMAB ≥ 300 EP.
3. **Séparer pions et cube par l'enjeu, pas par le comptage.** Publier L^pions et L^cube avec un dénominateur commun ; si besoin, un taux cube par « opportunité d'accès au cube » défini par les règles.
4. **Ne jamais classer sur un seul match.** Afficher la bande (IC à 95 %) et réserver les rangs aux joueurs dépassant environ 300 EP.
5. **Commencer le projet BMAB par un test de fidélité.** Recalculer le PR XG depuis les .xg et vérifier qu'il reproduit le PR BMAB publié, puis calculer l'ER GNU BG classique. Ce n'est qu'ensuite que les nouvelles métriques peuvent être attribuées à la métrique elle-même plutôt qu'au moteur.
6. **Garder les réglages et la MET figés et versionnés** (version de gnubg, ply, filtres, MET, graine). Publier les tables « un tour = une ligne » pour la reproductibilité.

## Caveats

- **Réponse à la question 5 (retrouve-t-on le classement BMAB ?).** Ma réponse est une **prédiction argumentée, non un résultat mesuré**. Je n'ai trouvé aucune étude publiée qui recalcule le classement BMAB avec d'autres métriques ou avec GNU BG. Je m'attends à une corrélation de rang élevée sur l'ensemble des joueurs, car numérateurs et moteurs mesurent essentiellement les mêmes erreurs. Elle sera nettement plus faible à l'intérieur de la tranche des Grandmasters, où les écarts réels (environ 2,8 à 4,7 de PR dans l'échantillon) sont du même ordre que le bruit et que les biais systématiques. Causes attendues des écarts :
  - la règle BMAB « plus bas de deux analyses », qui biaise le PR XG vers le bas ;\[30\]
  - le biais de moteur (chaque bot se juge parfait ; désaccords XG/GNU, en particulier au cube : un banc d'essai bgsage rapporte un PR cube de 3,69 pour XG Roller++ contre 5,50 pour un autre moteur) ;\[54\]
  - le poids du cube ;
  - les longueurs de match et les scores faciles (DMP) ;
  - la force et le style des adversaires ;
  - les fenêtres temporelles différentes.
- **Fiabilité des sources.** Plusieurs données proviennent de forums (bgonline, twoplustwo, rec.games.backgammon) ou de pages secondaires (backgammon101, Wikipedia), notamment la règle « 3 PR = 100 Elo », les rapports XG/GNU et les tables de Wayne Joseph. Ce sont des résultats empiriques non publiés formellement. La prépublication de Zoidis n'est pas relue par les pairs. Les fiches BMAB sont des relevés à des dates différentes et évoluent.
- **Limites des modèles.** La loi en √N (FIBS) est une approximation (marche aléatoire, sans gammons ni Crawford). L'identité « 0,5 − L » n'est exacte qu'en espérance et que si le moteur est juste. M3 et M4 dépendent de choix de modélisation (τ, c, troncature des candidats) qu'il faut fixer avant de regarder la corrélation avec BMAB, pour éviter le surajustement.
- **Lacune de données.** La liste officielle complète des titres BMAB et le tableau de classement n'ont pas pu être extraits. La reconstruction du classement de référence doit partir des fiches joueurs ou d'un export demandé à BMAB.

## Sources

1. [eXtreme Gammon Version 2 Documentation revision 1.01 (September 9th 2011)](https://www.extremegammon.com/extremegammon2.pdf)
2. [eXtreme Gammon — Grokipedia](https://grokipedia.com/page/extreme_gammon)
3. [RE: Error rate and Luck adjusted result](https://www.mail-archive.com/bug-gnubg@gnu.org/msg08378.html)
4. [GNU Backgammon](http://gnu.ist.utl.pt/manual/gnubg/html_node/Output-of-an-analysis.html)
5. [Overall rating - GNU Backgammon Manual V0.16](https://www.gnu.org/software/gnubg/manual/html_node/Overall-rating.html)
6. [Cube statistics - GNU Backgammon Manual V0.16](https://www.gnu.org/software/gnubg/manual/html_node/Cube-statistics.html)
7. [gnubg analyze match](https://groups.google.com/g/rec.games.backgammon/c/5ojGc7tAonE)
8. [Normalizing Errors, by Douglas Zare](https://bkgm.com/articles/Zare/NormalizingErrors/)
9. [Snowie ER and XG elo - the new Performance rating (PR) system](https://www.bgonline.org/forums/webbbs_config.pl?noframes%3Bread=53424)
10. [Normalizing Errors, by Douglas Zare](https://bkgm.com/articles/Zare/NormalizingErrors/index.html)
11. [Does XG's definition of PR annoy you?](https://www.bgonline.org/forums/webbbs_config.pl?noframes%3Bread=213965)
12. [Quantifying Backgammon Skill, by Chuck Bower](https://www.bkgm.com/articles/GOL/Sep01/gol901.htm)
13. [win probability for a PR difference and match length](https://www.bgonline.org/forums/webbbs_config.pl?noframes%3Bread=130338)
14. [win probability for a PR difference and match length](https://www.bgonline.org/forums/webbbs_config.pl?read=130338)
15. [EXtreme Gammon](https://en.wikipedia.org/wiki/EXtreme_Gammon)
16. [Backgammon FAQ: Ratings](https://www.bkgm.com/faq/Ratings.html)
17. [Skill v luck in determining backgammon winners](https://freerangestats.info/blog/2016/03/19/elo-pr-luck)
18. [PR/ER](https://backgammon101.com/pr-er/)
19. [Green/Black](https://lists.gnu.org/archive/mbox/bug-gnubg/2003-07)
20. [Skill in Backgammon: Cubeful vs Cubeless Tilemachos Zoidis, June 2025](https://www.rxiv.org/pdf/2506.0144v1.pdf)
21. [Intrinsic Chess Ratings Intrinsic Chess Ratings AAAI 2011 Kenneth W. Regan1](https://cse.buffalo.edu/~regan/Talks/IntrinsicRatings.pdf)
22. [Intrinsic Chess Ratings Kenneth W. Regan∗ University at Buffalo](https://cdn.aaai.org/ojs/7951/7951-13-11479-1-2-20201228.pdf)
23. [Intrinsic Chess Ratings Kenneth W. Regan∗ University at Buffalo](https://cse.buffalo.edu/~regan/papers/pdf/ReHa11c.pdf)
24. [Intrinsic Ratings Compendium (WORKING DRAFT) Kenneth W. Regan Department of CSE](https://cse.buffalo.edu/~regan/papers/pdf/Reg12IPRs.pdf)
25. [Lichess Accuracy metric • lichess.org](https://lichess.org/page/accuracy)
26. [Calculating accuracy - Page 2 - TalkChess.com](https://talkchess.com/viewtopic.php?t=82651&start=10)
27. [Match Report for Thomas Myhr - BMAB](https://www.bgmastersab.com/matchlog?id=38)
28. [Match Report for Thomas Kristensen - BMAB](https://bgmastersab.com/matchlog?id=193)
29. [PureTD: Reinforcement Learning for Backgammon Money Games with No Evaluation-time Search](https://arxiv.org/pdf/2608.15146)
30. [About this site - BMAB](https://bgmastersab.com/about)
31. [BMAB](http://bgmastersab.com/index.html)
32. [Backgammon Masters Awarding Body (BMAB)](https://usbgf.org/results/awards-recognition/backgammon-masters-awarding-body-bmab/)
33. [What's my overall PR? - Backgammon Forum - Discuss Backgammon Strategy, Clubs and Books](https://forumserver.twoplustwo.com/138/backgammon-forum-hosted-bill-robertie/whats-my-overall-pr-1791818/)
34. [Match Report for Masayuki Mochizuki - BMAB](https://bgmastersab.com/asia/matchlog?id=31)
35. [Match Report for Johan Moazed - BMAB](https://bgmastersab.com/matchlog?id=130)
36. [Match Report for Elias Kritikos - BMAB](https://bgmastersab.com/matchlog?id=188)
37. [Match Report for Dmitriy Obukhov - BMAB](https://bgmastersab.com/matchlog?id=81)
38. [Match Report for Naoki Iketani - BMAB](https://www.bgmastersab.com/matchlog?id=2295)
39. [Match Report for Petko Kostadinov - BMAB](https://bgmastersab.com/matchlog?id=361)
40. [Match Report for Art Benjamin - BMAB](https://bgmastersab.com/matchlog?id=117)
41. [Match Report for Volker Sonnabend - BMAB](https://bgmastersab.com/matchlog?id=18)
42. [Match Report for Johan Bynell - BMAB](https://www.bgmastersab.com/matchlog?id=143)
43. [XG file format documentation](https://www.bgonline.org/forums/webbbs_config.pl?noframes%3Bread=152155)
44. [Database of BG Matches](http://itikawa.com/kifdb/herodb.cgi?table=bg)
45. [GNU Backgammon](https://www.gnu.org/software/gnubg/)
46. [Re: \[Bug-gnubg\] \*.xg file (extreme gammon) import support?](https://lists.gnu.org/archive/html/bug-gnubg/2013-04/msg00026.html)
47. [Re: Preview of forthcoming gnubg release](https://www.mail-archive.com/bug-gnubg@gnu.org/msg08266.html)
48. [XG File format](https://www.extremegammon.com/XGformat.aspx)
49. [GitHub - oysteijo/xgdatatools: Tools to work with eXtreme Gammon xg-files · GitHub](https://github.com/oysteijo/xgdatatools)
50. [Converting XG match files to mat or txt and M.Petch's scripts](https://groups.google.com/g/rec.games.backgammon/c/WpVoNwu1_78/m/WUaUhgQpb9wJ)
51. [gammonview · PyPI](https://pypi.org/project/gammonview/1.4.0/)
52. [Exporting XG files to other formats](https://www.bgonline.org/forums/webbbs_config.pl?noframes%3Bread=126226)
53. [Standardised PR (standard analysis settings)](https://groups.google.com/g/rec.games.backgammon/c/3cPAahIsD2M)
54. [Bot Performance — Open Sage vs eXtreme Gammon](https://www.bgsage.ai/botperformance/)
