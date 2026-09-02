# Rollouts de backgammon : réduction de variance, troncature, arrêt, reproductibilité et pièges

## TL;DR
- **Pour départager deux coups ou une décision de videau sur portable en quelques secondes à minutes, reproduisez la recette gnubg/XG : dés quasi-aléatoires (multiples de 36) + réduction de variance par « luck adjustment » (lookahead 1-ply) + dés communs aux candidats + troncature contrôlée, et arrêtez sur la JSD (différence d'équité ÷ écart-type combiné de la différence) plutôt que sur l'écart-type absolu.** La réduction de variance vaut, selon les sources de la communauté, un facteur d'environ 5 à 25 en nombre de parties équivalentes.
- **La statistique clé n'est pas l'écart-type d'un candidat mais celui de la *différence* : gnubg calcule JSD = (équité_meilleur − équité_alt) / √(σ_meilleur² + σ_alt²) ; avec dés communs la corrélation positive réduit la variance de la différence, donc n'additionnez pas naïvement les intervalles de confiance.** Un seuil JSD de 1,96 ≈ 95 % de confiance ; gnubg s'arrête par défaut à une JSD paramétrable (valeur historique documentée « 3 », souvent réglée entre 2,33 et 4 par les utilisateurs).
- **Le principal piège est le biais : rollout tronqué et rollout cubeful réintroduisent l'erreur du réseau (le videau est géré par une politique approximative n-ply), et les dés quasi-aléatoires violent l'hypothèse i.i.d., si bien que l'intervalle de confiance affiché est optimiste.** Les rollouts sont fiables pour l'équité *relative* (coup A vs B) et beaucoup moins pour l'équité *absolue* (une take/pass serrée).

## Key Findings

1. **gnubg et XG utilisent quatre techniques cumulées** : (a) réduction de variance par correction de la chance (« luck adjustment », idée de Fredrik Dahl / Jellyfish, décrite par Douglas Zare et David Montgomery) ; (b) dés quasi-aléatoires stratifiés sur les 2 premiers plies (rotation des 36, puis 1296 combinaisons) ; (c) dés communs / duplicate dice entre candidats ; (d) troncature avec évaluation terminale par le réseau, et troncature exacte sur base de bearoff two-sided.

2. **La réduction de variance est spectaculaire mais dépend de la qualité du réseau** : Montgomery écrit textuellement « When variance reduction makes those hundred games worth 2500 now we have something » (facteur ~25) ; les valeurs « equivalent games » de Jellyfish (864 essais annoncés « the equivalent of 15,618 » par Montgomery, « the equivalent of over 18,000 trials each » selon Chuck Bower) sont considérées comme trop optimistes par Montgomery lui-même ; Zare estime la réduction du bruit à un « facteur 1/5 à 1/10 » avec évaluation 1-ply, davantage en 2-3 ply. Tesauro & Galperin mesurent une réduction du taux d'erreur du joueur de base « by as much as a factor of 5 or more ».

3. **Formule de l'écart-type dans gnubg (source `rollout.c`)** : variance courante à la Welford avec correction (n−1), écart-type affiché = √(variance/n) (erreur-type de la moyenne). L'IC 95 % = ±1,96 × erreur-type.

4. **Défauts et réglages** : le contexte de rollout gnubg par défaut inclut troncature à 11 plies, dés quasi-aléatoires activés, réduction de variance activée, cubeful activé ; les nombres de parties usuels sont des multiples de 36 (36, 108, 216, 324, 648, 1296, 5184, 10368…). XG : défaut 1296 parties ; XGR++ = 360 parties tronquées (5 coups pour le jeu de pions, 7 pour le videau) avec réduction de variance, arrêt à 0,010 de confiance (min. 180 parties).

5. **Reproductibilité** : gnubg attribue la graine par *partie* (`nSeed + (trial << 8)`) et non par thread, précalcule les permutations quasi-aléatoires par numéro de partie, et accumule les résultats par sommation sûre — le résultat agrégé est donc indépendant de l'ordre d'achèvement des threads. Générateurs disponibles : ANSI, BSD, Blum-Blum-Shub, ISAAC, MD5, Mersenne Twister (par défaut), random.org, fichier, manuel.

## Details

### 1. Techniques de réduction de variance

**Principe (luck adjustment).** L'idée fondatrice, due à Fredrik Dahl (auteur de Jellyfish), est formalisée par Douglas Zare (« Hedging Toward Skill », GammonVillage 2000) via l'équation « Final − Initial = Net Luck + Net Skill ». On estime la chance nette d'un coup de dé comme la différence entre l'équité de la position après le meilleur coup et l'équité (une profondeur plus profonde) avant le lancer, puis on la soustrait du résultat. Zare insiste : pour être non biaisé, il faut évaluer la position initiale « un ply plus profond que la position après le lancer » et se demander seulement si le lancer était au-dessus ou en dessous de la moyenne (moyenne des continuations), sinon l'estimation de la chance ne s'annule pas à zéro. Le manuel gnubg (V1.00.0) décrit cela ainsi : « Variance reduction: when using lookahead evaluations, it can reduce errors by making use of the equity difference from one ply to the next… GNU Backgammon automatically performs variance reduction when looking ahead at least one ply. » (Le manuel renvoie explicitement à l'article de Zare pour la « variance reduction of skill ».) C'est confirmé par le code (`rollout.c`, bloc « Variance reduction » : pour chacun des 21 lancers, on cherche le meilleur coup en 0-ply et on calcule un `arMean` sur les 6×6 combinaisons pondérées).

**Gains mesurés (chiffres sourcés).**
- David Montgomery (« Variance Reduction », GammOnLine, févr. 2000) : « Using this feature a rollout of a hundred games might well be just as accurate as a rollout of 2500 games done without the feature » → **facteur ~25**.
- Montgomery (« Questions and Answers… », mars 2000) : la valeur « equivalent games » de Jellyfish (864 parties annoncées équivalentes à 15 618, jusqu'à « over 18,000 » selon Chuck Bower) est jugée peu fiable ; il recommande d'ignorer « equivalent games » et de se concentrer sur l'écart-type. À prendre comme borne haute optimiste.
- Zare : « My guess is that there is a factor of 1/5 to 1/10 as much noise as before, and that this would be reduced much more by using 2-ply or 3-ply evaluations. »
- Tesauro & Galperin (« On-line Policy Improvement using Monte-Carlo Search », NIPS 1996-97) : réduction du taux d'erreur du joueur de base « by as much as a factor of 5 or more » ; la présentation ultérieure (Tesauro 2002) chiffre un joueur à rollouts « 5-6 times more accurate than its base 1-ply player, and twice as accurate as the corresponding 3-ply player ».

**Dés quasi-aléatoires / stratification (gnubg).** Le manuel : si n×36 parties sont demandées, n parties commencent par 11, 2n par 21, etc. (rotation du 1er lancer) ; si n×1296, le 2e lancer est aussi tourné ; le 3e l'est si le nombre de parties est proportionnel à 46656. gnubg ne stratifie que les 2 premiers plies. Pour les positions initiales, si la séquence commence par un double, elle est sautée. C'est pourquoi les nombres de parties sont des multiples de 36 (36×36 = 1296 étant l'étalon).

**Dés communs (common random numbers).** Kit Woolsey (« Computers and Rollouts », GammOnLine janv. 2000) : « For play vs. play decisions, duplicate dice are used — i.e. for each trial of play A, play B is then rolled with the same dice. » Cela induit une corrélation positive entre les deux séries → la variance de la différence chute. Woolsey note aussi la limite : pour des positions qui divergent vite (jeu d'ouverture) ou mènent à des types de jeu très différents, les dés communs aident peu.

**XG (eXtreme Gammon).** XG propose « Variance Reduction » (option recommandée par défaut dans l'aide et sur les forums) et des dés quasi-aléatoires. XGR++ (le niveau de rollout tronqué le plus fort) : 360 parties avec VR, tronqué après 5 coups (jeu de pions) / 7 coups (videau), les 2 premières décisions en 3-ply puis 2-ply pour les pions et 3-ply pour le videau, arrêt à 0,010 de confiance (min. 180 parties). XG utilise « XG Roller+ » pour les décisions de videau dans ses réglages « World Class ». XG n'a longtemps pas parallélisé les rollouts (contrairement à gnubg) ; il expose min./max. parties, temps max, confiance à atteindre, troncature en bearoff.

**Snowie / Jellyfish / BGBlitz.** Jellyfish (Dahl) a introduit VR et le rollout interactif ; Snowie 3 a apporté la VR pour les rollouts *cubeful* (Montgomery : « another breakthrough »). Ces moteurs n'exposent pas autant de réglages que gnubg.

### 2. Troncature (truncation)

**Compromis biais/variance.** Le manuel gnubg est explicite : tronquer « introduit un biais systématique » mais (i) accélère → plus de parties (échange erreur d'échantillonnage contre erreur systématique) ; (ii) les positions diffèrent d'une partie à l'autre, donc les erreurs se décorrèlent partiellement ; (iii) pour deux coups candidats, le biais résiduel est corrélé et s'annule en grande partie. Conclusion du manuel : **les rollouts tronqués conviennent mieux à l'équité *relative* (« which is the better move here? ») qu'à l'équité *absolue* (« at this match score I need 29% wins to accept a dead cube; can I take? »).**

**Chiffres.** Le contexte de rollout gnubg utilise une troncature par défaut à 11 plies (constaté dans un fichier .sgf réel : `n-truncation: 11`). Attention à la terminologie de profondeur : le fil « GnuBg settings to mimic XGR++ » précise qu'un « truncate at ply 7 » produit « depth 7 » et que gnubg appelle « 0-ply » ce que XG appelle « 1-ply ». XGR++ tronque à 5 coups (pions) / 7 coups (videau).

**Course / bearoff exact.** gnubg peut tronquer sur sa base two-sided (« Race database truncation ») et évaluer « with no error at all » — ce qui supprime le biais et la grosse variance de fin de partie. Options : `set rollout bearofftruncation exact on/off` et `onesided on/off`. La base 1-sided couvre 15 pions sur les 6 premières points ; la 2-sided est gardée en mémoire. Un fil bug-gnubg montre qu'une transition mal calée (net de course → base de bearoff) peut créer un artefact ; aller jusqu'au point 7 ou 11 plutôt que 6 aide.

**Cubeful tronqué : le problème du videau au point de coupe.** Au point de troncature, l'équité cubeful est estimée par le réseau selon une politique de videau approximative ; c'est la principale source de biais des rollouts cubeful tronqués, d'autant plus grave que c'est justement quand on ne fait pas confiance au n-ply qu'on lance un rollout.

### 3. Critères d'arrêt (stopping rules)

**Écart-type et erreur-type (code source).** Dans `rollout.c`, gnubg maintient une variance courante à la Welford :
`aarVariance[alt][j] = aarVariance[alt][j]*(1 − 1/(n−1)) + n·(rMuNew − aarMu)²`, puis affiche
`aarSigma[alt][j] = sqrtf(aarVariance[alt][j] / n)` — c'est l'**erreur-type de la moyenne** (σ/√n avec correction (n−1) intégrée dans la récurrence). Les sorties de probabilité de gain sont bornées à [0,1]. L'IC 95 % = moyenne ± 1,96 × erreur-type.

**JSD (joint standard deviation).** C'est la statistique d'arrêt centrale, calculée dans `check_jsds()`. Après tri des candidats par équité décroissante :
`JSD_alt = (équité_meilleur − équité_alt) / √(σ_meilleur² + σ_alt²)` (avec un plancher de 1e-8 au dénominateur). C'est le nombre d'écarts-types de la *différence* entre le meilleur coup et le candidat. Interprétation communautaire (fil bug-gnubg / bgonline) : une JSD de 1,96 ≈ 95 % de chance que le coup en tête soit réellement meilleur que le suivant. **Point crucial et piège** : le code additionne simplement les deux variances (σ_meilleur² + σ_alt²) **sans terme de covariance**, alors que les dés communs induisent une corrélation positive. Le bénéfice des dés communs entre donc *implicitement* (chaque σ est plus petit) et non via une soustraction de covariance explicite — la JSD affichée est donc conservatrice à cet égard, mais l'IC global reste optimiste à cause du quasi-aléatoire (voir §5).

**Règle d'arrêt.** Si `fStopOnJsd` est actif ET que le candidat a joué au moins `nMinimumJsdGames` parties ET que sa JSD dépasse `rJsdLimit`, ce coup « is no longer worth rolling out » et est abandonné. Il existe aussi une règle d'arrêt sur l'écart-type absolu (`check_sds` / `fStopOnSTD` / `rStdLimit` : « stop when abs(value/std) < this »). Pour les rollouts de videau, les deux alternatives (double/no-double) s'arrêtent ensemble quand la JSD minimale dépasse la limite.

**Valeur par défaut de la limite JSD.** Un fil bgonline (leobueno, 17 avril 2011) indique textuellement : « The default GNUbg setting, for stopping rollouts when JSD reaches a given level, is 3. » Des utilisateurs recommandent 4 pour plus de sûreté, ou d'utiliser équité/JSD comparé à une table de confiance. (La constante exacte de `rcRollout` n'a pas pu être vérifiée dans le source ; considérer 3 comme valeur historique documentée par la communauté, certaines versions récentes citant 2,33.)

**Nombres de parties.** Multiples de 36 pour préserver le quasi-aléatoire : 36, 108, 216, 324, 648, 1296 (=36²), 5184, 10368, 20736, 25920… 1296 est le défaut XG et le seuil « minimal sérieux » ; les experts commencent souvent à 5184 et montent. Woolsey avertit dès 2000 que « même 1296 essais peuvent ne pas suffire » (une ouverture 4-2 rollée 1296× donnait un résultat aberrant, corrigé au-delà de 5000).

**Différence entre deux moyennes appariées.** Montgomery donne la règle pratique : quand les deux σ sont proches (cas usuel à nombre de parties égal), l'écart-type de la différence ≈ 1,4 × moyenne des deux σ (√2). Donc deux IC individuels qui se chevauchent *ne signifient pas* absence de différence significative — il faut regarder l'écart-type de la différence (la JSD).

### 4. Reproductibilité

**Graine et générateurs.** Commande `set rng {ansi|bsd|bbs|isaac|md5|mersenne|random.org|file|manual}`, graine par `set seed`. Mersenne Twister est le défaut (« This should be an excellent pseudo-random number generator »). L'ANSI rand() est déconseillé pour les rollouts car « any small biases in the dice could accumulate over hundreds or thousands of trials and distort the results ». Particularité MD5 : la graine s'incrémente de 1 par lancer, donc la séquence de graine n+1 = celle de n décalée d'un lancer.

**Déterminisme sous parallélisme.** gnubg parallélise en découpant les *parties* (trials) entre threads, pas les évaluations (confirmé par Jon Kinsey sur bug-gnubg : « splitting up big tasks (moves in a game and trials in the rollout) »). La graine est attribuée **par partie** : `InitRNGSeed(nSeed + (trial << 8), …)`, et les permutations quasi-aléatoires sont précalculées par numéro de partie (`QuasiRandomSeed`, `RolloutDice` indexé sur `iGame`). Ainsi la partie n a toujours la même séquence de dés quel que soit le thread ou l'ordre d'exécution. L'accumulation se fait par sommation/incréments sûrs (`MT_SafeInc`, `MT_SafeAdd`) dans des accumulateurs par alternative → le résultat agrégé est indépendant de l'ordre d'achèvement. Seul le chemin « position initiale » utilise une variable statique `nSkip` marquée « not multi-thread safe » (saut des doubles au 1er lancer). Le port Android de gnubg revendique d'ailleurs des rollouts « seed-reproducible » byte-à-byte et un harnais de comparaison trial-par-trial avec le gnubg de bureau.

**Signature d'un rollout (à consigner).** gnubg imprime déjà une ligne canonique, par ex. : « Full cubeful rollout with var.redn. 1296 games, Mersenne Twister dice gen. with seed 858145082 and quasi-random dice / Play: world class 2-ply … / Cube: 2-ply … ». Pour qu'un résultat reste comparable et republiable, consignez : nombre de parties (et min/max), ply du jeu de pions, ply de la décision de videau, troncature (on/off, profondeur, bearoff exact/one-sided), VR on/off, quasi-aléatoire on/off, graine, générateur, version des poids/réseau, MET utilisée, cubeful/cubeless, money/match + score, Jacoby/beavers, et la JSD atteinte. XG produit une signature analogue (XGID, « Rollout: N Games rolled with Variance Reduction. Dice Seed: … Moves: 3-ply, cube decisions: XG Roller … Confidence … », version, auteur, date).

### 5. Pièges connus

**Cubeful vs cubeless.** Le rollout cubeful gnubg prend les décisions de videau à une profondeur fixée (`set rollout cubedecision plies n`) selon une politique approximative ; le rollout est de fait tronqué aux décisions « double/drop » dans la branche live-cube, si bien que les pourcentages « cubeless » affichés dans un rollout cubeful **ne sont pas de vrais cubeless** (« the percentages are not really true cubeless ones, as the rollouts is truncated », bug-gnubg 2003). Zare va jusqu'à écrire qu'« il ne semble pas y avoir de moyen précis de roller une décision de doubler ou non ». Recommandation communautaire : pour une **décision de videau**, faites un rollout *cubeful* avec la décision de videau au ply le plus élevé possible (3-ply / XG Roller+) ; pour un **choix de coup** (checker play) sans enjeu de videau immédiat, un cubeless money sans Jacoby est un bon compromis universel (choix de « Backgammon Camp » : money, no Jacoby, cubeful 3-ply comme référence polyvalente).

**Jacoby et beavers (money).** Ils changent le résultat : sans Jacoby, les gammons comptent même sans videau tourné ; avec Jacoby ils ne comptent qu'après un double. Un rollout XG « Money, Jacoby, Beaver » peut révéler un paradoxe de Kauder (double/beaver optimal, ex. rollout Taper_Mike de 31 104 parties). Consignez toujours le statut Jacoby/beaver. « Cubeless equity in money game do not take into account the Jacoby status » (notes de version XG).

**Match, Crawford, score.** Un rollout à un score de match donné utilise la MET (p. ex. Kazaross-XG2, Woolsey) et diffère d'un rollout money ; la conversion équité↔MWC intervient dans le calcul de la JSD (`mwc2eq`, `se_mwc2eq`). Le manuel avertit que les nombres « MWC contre l'adversaire courant » et l'ajustement de chance sont biaisés vers le bot analyseur.

**Biais auto-référentiel.** Un rollout mesure « la meilleure décision *en supposant que les deux camps jouent ensuite exactement comme le bot* » (Montgomery). Le réseau qui évalue est celui qui joue : les faiblesses systématiques (back games, positions rares hors distribution d'entraînement) se propagent. Woolsey documente des misplays 1-ply grossiers (double-as, point d'as) que le bot corrige seulement en 3-ply — d'où l'intérêt d'un jeu de pions ≥ 2-ply dans le rollout et de vérifier comment le bot joue les premiers lancers de la position.

**VR mal implémentée = biais.** Zare et Montgomery soulignent que la correction de chance doit être *non biaisée* (évaluer la position avant lancer une profondeur plus profond que la moyenne des continuations) ; sinon on réintroduit un biais. Dans des positions que le bot comprend mal (ex. Kauder paradox, Jellyfish 2.0), la VR pouvait *augmenter* l'écart-type (« equivalent games » < parties réelles).

**Quasi-aléatoire et validité des IC.** La stratification et les dés communs violent l'hypothèse i.i.d. sous-jacente à l'IC classique σ/√n. En pratique, la stratification rend les IC **optimistes** (l'écart-type calculé comme si les parties étaient indépendantes sous-estime l'incertitude réelle) : Montgomery et Zare recommandent de traiter les « equivalent games » avec méfiance et de ne jamais surinterpréter une différence à ~2 σ. À l'inverse, la JSD sans terme de covariance est plutôt conservatrice. Bilan : traitez l'IC affiché comme un ordre de grandeur, exigez une JSD confortable (≥ 3) avant de trancher, et méfiez-vous des décisions serrées (< 0,010 d'équité) où aucun nombre de parties raisonnable ne conclura.

## Recommendations

Réglages concrets pour votre moteur Go (prob5, 0→2-ply, videau Janowski+MET, SIMD, multi-cœurs). Reproduisez la logique gnubg/XG.

**Réglages transverses (toujours) :**
- Dés **quasi-aléatoires** stratifiés sur les 2 premiers plies + **dés communs** entre les deux candidats. Nombres de parties **multiples de 36**.
- **Réduction de variance** par luck adjustment 1-ply, en évaluant la position avant lancer une profondeur plus profonde que la moyenne des continuations (sinon biais).
- **Graine par partie** (`seed + trial<<8`), permutations précalculées par numéro de partie, accumulation par sommation déterministe → reproductibilité multi-thread garantie.
- **Arrêt sur JSD** = Δéquité/√(σ_A²+σ_B²), pas sur l'IC individuel. Affichez l'IC 95 % = moyenne ± 1,96·σ/√n et la JSD.
- **Troncature exacte** dès l'entrée en base de bearoff two-sided (course) pour supprimer biais et variance de fin de partie.

**Profil RAPIDE (~2 s)** — dégrossir, écarter un blunder évident :
- Jeu de pions **0-ply** (prob5 brut) ou 1-ply, décision de videau 0-1-ply.
- **Tronqué à ~5-7 plies**, VR on, dés communs, quasi-aléatoire.
- **216 parties** (min 108), arrêt anticipé si **JSD ≥ 3**.
- Usage : confirmer une hint, éliminer un candidat clairement inférieur. Ne pas publier.

**Profil STANDARD (~30 s)** — recommandation fiable pour la plupart des coups :
- Jeu de pions **2-ply**, décision de videau **2-ply cubeful** (votre Janowski+MET).
- Tronqué à **~11 plies** (défaut gnubg) OU non tronqué avec bearoff exact.
- **1296 parties** (=36²), min 324, arrêt si **JSD ≥ 3** ; cubeful pour une décision de videau, cubeless money no-Jacoby acceptable pour un pur choix de coup.
- Usage : la grande majorité des départages coup A/B.

**Profil PRÉCIS (~5 min)** — décisions serrées, matériel publiable :
- Jeu de pions **2-ply**, videau **2-ply cubeful** (le plus profond que le budget permet).
- Non tronqué (ou 11+ plies) + bearoff exact two-sided.
- **5184 à 10368 parties**, min 1296, arrêt si **JSD ≥ 4**.
- Consignez la **signature complète** (voir §4) pour reproductibilité/comparaison.
- Usage : cube serré, deux coups à < 0,02 d'équité.

**Seuils qui changent la décision :**
- Si les **IC 95 % des deux candidats se chevauchent** → insuffisant ; augmentez les parties (rappel : ÷2 de l'écart-type coûte ×4 parties).
- Si **JSD < 2** après le budget max → la décision est trop serrée pour être tranchée par rollout ; choisissez sur d'autres critères (simplicité, score, style adverse).
- Si **Δéquité < 0,010** → traitez les deux coups comme équivalents.
- Si le bot **misplaye visiblement les premiers lancers** en 0/1-ply → montez le ply du jeu de pions avant de conclure.
- Pour une **décision de videau** : préférez toujours cubeful avec videau au ply max ; ne vous fiez pas aux « cubeless » d'un rollout cubeful.

## Caveats
- **Constantes par défaut exactes non toutes vérifiées dans le source** : le contexte `rcRollout` (valeurs littérales de `rJsdLimit`, `nMinimumGames`, `nMinimumJsdGames`, troncature) est initialisé dans `gnubg.c`/`eval.c`, non capturé mot pour mot. La limite JSD par défaut « 3 » vient d'un fil bgonline (2011) ; certaines versions récentes citent 2,33. À vérifier dans l'initialiseur `rcRollout` si la valeur exacte est requise.
- Les **facteurs de réduction de variance** (25×, 18×, 5-10×) sont des estimations d'experts (Montgomery, Zare, Tesauro) et dépendent fortement de la position et de la qualité du réseau ; Montgomery lui-même juge les « equivalent games » de Jellyfish/Snowie trop optimistes. Ne les prenez pas comme garanties.
- Les **IC et JSD reposent sur des hypothèses (i.i.d.) que le quasi-aléatoire et les dés communs violent** ; considérez-les comme indicatifs, pas comme des tests statistiques rigoureux.
- Certaines infos XG proviennent de forums (bgonline, twoplustwo) et d'une page Grokipedia (à valeur secondaire) ; les réglages XGR++ (360 parties, 5/7 coups, 0,010) sont corroborés par plusieurs sources indépendantes et par l'aide XG citée sur les forums.
- Votre moteur étant « self-referential » (prob5 joue et évalue), attendez-vous au biais auto-référentiel : les rollouts vous diront le meilleur coup *contre votre propre bot*, pas la vérité absolue.