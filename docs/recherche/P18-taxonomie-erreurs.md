# P18 — Catégorisation automatique des erreurs aux échecs et au poker : leçons transposables au backgammon

## TL;DR
- Les écosystèmes échecs et poker convergent tous vers **une seule unité de gravité universelle : la perte de probabilité de gain (Win%/EV/équité), pas l'unité brute du moteur** (centipawns bruts). Lichess convertit officiellement centipawns → Win% via une sigmoïde, chess.com classe ses coups sur la perte d'« Expected Points » (Inexactitude 5-10%, Erreur 10-20%, Gaffe 20%+, seuils officiels), et le poker mesure tout en « EV loss » (bb/100). **Le backgammon fait déjà cela nativement avec l'EMG/MWC — c'est un avantage de départ décisif.**
- Les catégories fiables sont peu nombreuses (3 niveaux de gravité + 2-3 étiquettes spéciales à base de règles), agrégées par phase/rue, par type de décision et par dimension (position, pièce, temps). Les explications textuelles fiables *sans LLM* sont **basées sur des gabarits (templates) + une base de connaissances de règles** (DecodeChess, Fritz) et se taisent quand aucune règle ne s'applique de façon confiante.
- Pour le backgammon, une taxonomie défendable en 6 thèmes se construit à partir de ce que gnubg/XG fournissent déjà : (1) séparation coup de pions / cube, (2) type d'erreur de cube, (3) gravité normalisée (gaffe isolée vs imprécision chronique), (4) phase/type de position, (5) précision de fin de partie/tempo, (6) compétence vs chance. Anti-patterns majeurs : trop de catégories, faux positifs (la surabondance de « Brilliant » chez chess.com), déclencher des « leaks » sur échantillon insuffisant, et confondre chance et compétence.

## Key Findings

### 1. L'unité de gravité : abandon de l'unité brute du moteur au profit de la probabilité de gain
- **Lichess** publie ouvertement ses formules (page officielle `lichess.org/page/accuracy`, code source `lila`). Conversion centipawns → Win% : `Win% = 50 + 50 * (2 / (1 + exp(-0.00368208 * centipawns)) - 1)` (constante k = 0,00368208). Précision d'un coup : `Accuracy% = 103.1668 * exp(-0.04354 * (winPercentBefore - winPercentAfter)) - 3.1669`. Lichess précise « The equation is based on real game data » ; la courbe est calibrée sur des parties de joueurs d'environ 2300 Elo (elle sous-estime donc légèrement les chances de gain des tout meilleurs). Lichess explique explicitement le problème des centipawns bruts : « perdre 300 centipawns dans une position égale est une gaffe majeure ; mais perdre 300 centipawns quand la partie est déjà gagnée ne change presque rien ». Le Win% vise à être indépendant de l'évaluation de la position.
- **Chess.com** utilise un modèle « Expected Points » (Classification V2) et un score de précision « CAPS2 ». Les seuils sont **officiellement documentés** (Help Center, article « How are moves classified? », perte d'Expected Points sur une échelle 0-1) : Best 0,00 ; Excellent 0,00-0,02 ; Good 0,02-0,05 ; **Inaccuracy 0,05-0,10 (5-10%)** ; **Mistake 0,10-0,20 (10-20%)** ; **Blunder 0,20-1,00 (20-100%)**. Citation officielle : « At 1.00, you have a 100% chance of winning, and at 0.00, you have a 0% chance of winning ». La conversion évaluation→Win% dépend aussi du **rating du joueur** et reste **privée** ; seuls les seuils sont publics.
- **Poker** : tout est mesuré en **EV loss** (perte d'espérance) exprimée en big blinds, et normalisée en **bb/100 mains**. GTO Wizard (help center) : « EV loss is defined as the expected value you lost during the hand if playing against a GTO solution ». PokerSnowie (manuel) : « For every betting round, the error rate sums up the EV cost of all errors, divided by the number of moves played ».
- **Backgammon (déjà en place)** : gnubg et XG mesurent tout en **équité normalisée EMG** (Equivalent to Money Game) et **MWC** (Match Winning Chance). C'est structurellement la même approche que le Win%/EV — le backgammon part avec une longueur d'avance conceptuelle.

### 2. Les taxonomies de gravité sont courtes (≈3 niveaux) + quelques étiquettes spéciales à règles
- **Lichess** : 3 niveaux — Inaccuracy (?!), Mistake (?), Blunder (??), calculés sur la perte de Win%.
- **Chess.com** : 6 niveaux de qualité (Best, Excellent, Good, Inaccuracy, Mistake, Blunder) + 3 étiquettes spéciales à règles (Brilliant !!, Great !, Miss) + Book. Définition officielle de « Brilliant » = « when you find a good piece sacrifice » avec conditions : ne pas être en mauvaise position après, et ne pas être déjà complètement gagnant sans le coup ; définition plus généreuse pour les joueurs débutants. « Great » = coup critique (transforme perdant→égal, égal→gagnant, ou seul bon coup). « Miss » = échec à punir l'erreur adverse / occasion gagnante ratée (le seuil d'évaluation dépend du rating).
- **gnubg** : 3 marques — Doubtful, Bad, Very bad (les classes Good/Very good existent dans le code mais **ne sont pas utilisées**). Seuils par défaut abaissés dans une version récente **de 0,04 / 0,08 / 0,16 à 0,03 / 0,06 / 0,12 point d'équité** (changelog officiel).
- **XG** : catégorise par taille d'erreur ; les seuils « blunder / error / outplay » correspondent respectivement à ~4% / 1% / 0,5% de chances de gagner (en jeu d'argent sans gammon ni cube — à traiter comme une caractérisation documentée mais dépendante du contexte, elle varie au score en match).
- **PokerSnowie** : Blunder (seuil par défaut EV loss > 2,0 bb, réglable dans les options), navigation par Moves / Errors / Blunders / Showdowns.
- **GTO Wizard** : code couleur à 3 niveaux — vert (correct), jaune (« Inaccuracy », action jouée < 3,5% du temps en GTO mais sans grosse perte d'EV), rouge (« Wrong Move »/« Blunder » : jamais joué en GTO et perte d'EV significative). Le help center « Measure Performance » documente aussi « Avg. EV Loss per mistake ».

### 3. Dimensions d'agrégation (le vrai cœur pédagogique)
- **Lichess Insights** est un « answer engine » qui croise un *metric* (axe Y), une *dimension* (axe X) et des *filtres*. Metrics (12) : Average centipawn loss, Move time, Game result, Game termination, Rating gain, Opponent rating, Moves per game, Piece moved, Opportunism (taux de punition des gaffes adverses), Luck, Centipawn Loss Bucket. Dimensions (16) : Opening, castling side, Queen trade, Piece moved, Move time, Material imbalance, Evaluation, **Game phase**, Centipawn Loss, etc. Détection algorithmique de phase (ouverture/milieu/finale) : basée sur le nombre de pièces majeures/mineures restantes et si les rangées arrière sont « sparse » (< 4 pièces = pièces développées).
- **Aimchess** note le jeu sur ~6-8 compétences : openings, tactics, resourcefulness (défense de positions inférieures), endgames, advantage capitalization, time management, resilience, puis sert des drills ciblés sur la faiblesse détectée.
- **GTO Wizard** agrège par rue (preflop/flop/turn/river), par position, par type d'action (bet/check/call/fold/raise), par taille de mise ; tri par « biggest EV losses ». Les GTO Reports affichent la « mistake list (sorted by EV loss in bb) ».
- **Backgammon (gnubg/XG)** sépare nativement **Chequer play** (jeu de pions) vs **Cube decisions** ; le cube est lui-même décomposé en 6 catégories d'erreur : missed double below CP, missed double above CP, wrong double below DP, wrong double above TG, wrong take, wrong pass. **Backgammon Studio Heroes** agrège par **type de position** (opening, blitz, holding game, priming, backgame, bearoff, race…).

### 4. Explications textuelles automatiques sans LLM : templates + base de règles + abstention
- **Fritz/ChessBase Tactical Analysis** : annotation verbale automatique historique (« Better is… », « The losing move », « White has a decisive advantage »), plus symboles/flèches/cases colorées et questions d'entraînement. Reconnue comme limitée par les utilisateurs (« can be quite dumb at best… limited to simple suggestions like : Better is, Good square for the knight, The losing move »).
- **DecodeChess** : explications en langage naturel **à base de gabarits + base de connaissances**, dérivées de l'arbre de recherche du moteur (Stockfish) — pas d'un LLM à l'origine. Utilise des templates à trous, ex. : « (player) wants to win the (piece) but has to play (xx)s first in order to remove a supporter of (square)s ». Se cantonne aux joueurs < 2000 Elo et reconnaît ses limites vs un humain.
- Le principe clé : **une règle ne s'exprime que si elle s'applique de façon confiante ; sinon on se tait** (abstention / selective prediction). gnubg illustre littéralement cela : quand un coup ou une décision de cube ne mérite aucune marque, le menu d'analyse est **vide** (« GNU Backgammon's analysis of this is empty. There was nothing wrong with not doubling »).

### 5. Recherche académique : la gravité = perte d'équité, et la prédiction d'erreur humaine
- **Guid & Bratko (2006), « Computer Analysis of World Chess Champions » (ICGA Journal, Vol. 29 No. 2, juin 2006, p. 65-73)** : critère de base = « average difference between moves played and best evaluated moves » (perte d'équité moyenne par coup), complété par le « rate of blunders » et une mesure de la complexité de la position (l'erreur croît avec la complexité). Moteur : Crafty limité à « 12 plies plus quiescence search ». Résultat notable : le vainqueur était Vladimir Kramnik (sa meilleure performance, vs Kasparov, Londres 2000, avec une « average error » de seulement 0,0903) — l'article originel citait Capablanca comme meilleur selon le critère d'erreur moyenne, illustrant la sensibilité des résultats au moteur/à la profondeur. C'est le fondement académique de l'idée « qualité = perte d'équité moyenne » — exactement la métrique du PR au backgammon.
- **Maia / McIlroy-Young, Sen, Kleinberg & Anderson (2020), « Aligning Superhuman AI with Human Behavior » (KDD '20)** : modèles entraînés par tranche de rating pour **prédire le coup humain** et **prédire si un humain va commettre une gaffe**. Seuils confirmés dans Maia-2 (Tang et al., NeurIPS 2024) : « Blunders reduce the expected win-rate by ≥10%, Errors by 5-10%, and Optimal by ≤0% ». Le modèle de prédiction de gaffe est entraîné sur ~182M coups « blunder » vs ~272M « non-blunder » ; un random forest de base passe de 56,4% (position seule) à ~63% (position + métadonnées). **Leçon transposable : calibrer la détection d'erreur sur le niveau du joueur, pas sur la perfection du bot.**

## Details

### Tableau comparatif — catégories, seuils, unités, dimensions

| Outil | Unité de gravité | Catégories & seuils | Dimensions d'agrégation | Documenté ? |
|---|---|---|---|---|
| **Lichess** | Win% (dérivé de centipawns par sigmoïde) | Inaccuracy / Mistake / Blunder (sur perte de Win%) ; Accuracy% par courbe exponentielle | Insights : 12 metrics × 16 dimensions (phase, pièce, temps, ouverture…) | **Officiel & open source** (formules publiées, code lila) |
| **Chess.com** | Expected Points (Win%), dépendant du rating | Best 0 / Excellent 0-.02 / Good .02-.05 / Inaccuracy .05-.10 / Mistake .10-.20 / Blunder .20-1.0 + Brilliant/Great/Miss (règles) ; précision CAPS2 | Game Report par phase ; Insights (openings, phases, heure) | **Seuils officiels** ; formule éval→Win% privée |
| **Aimchess** | Scores composites par compétence | openings, tactics, resourcefulness, endgames, advantage capitalization, time management, resilience | Par compétence + comparaison à pairs de même niveau | Marketing ; méthodo non publiée en détail |
| **DecodeChess** | (explicatif, pas de score de gravité) | Concepts positionnels/tactiques via templates | Par position, par concept | Semi-documenté (blog) |
| **ChessBase/Fritz** | centipawns + précision % | Erreurs/inexactitudes ; annotations verbales + symboles | Par partie, par coup | Documenté (aide produit) |
| **GTO Wizard** | EV loss (bb), bb/100 | vert / jaune (Inaccuracy, <3,5%) / rouge (Wrong Move, Blunder) ; tri par EV loss ; EV loss per mistake | Rue, position, type d'action, taille de mise | **Documenté** (help center) |
| **PokerSnowie** | EV loss (bb) ; Error rate propriétaire ; SnowieScore 0-100 | Blunder (>2,0 bb par défaut) ; Moves/Errors/Blunders/Showdowns | Par rue ; balance des ranges | Documenté (manuel) |
| **PioSolver/GTO+** | EV (% du pot ou bb) | Node locking ; écart stratégie joueur vs GTO ; rapports d'agrégation | Par nœud, par flop, aggregation reports | Documenté (docs) |
| **gnubg** | EMG / MWC | Doubtful/Bad/Very bad (0,03/0,06/0,12) ; cube décomposé en 6 sous-types ; rating Awful!→Supernatural | Chequer vs cube ; luck vs skill | **Officiel & open source** |
| **XG** | EMG normalisé | Erreur par taille ; PR = équité perdue / décision × 500 | Chequer vs cube ; profils de session | Documenté (aide) |
| **BG Studio Heroes** | EMG (gnubg/XG) | erreurs/blunders surlignés | **Par type de position** (opening, blitz, holding, priming, backgame, bearoff, race) | Guide utilisateur |

### Le PR au backgammon : la métrique la plus mûre du lot
Le **Performance Rating (PR)** de XG = équité moyenne perdue par décision non triviale × 500. Bandes officielles XG (manuel eXtreme Gammon, reprises par Wikipedia citant le *Financial Times*, 28/07/2023) :

| PR | Étiquette | Elo approx. |
|---|---|---|
| 0,0-2,5 | World Champ | 2162-2240 |
| 2,5-5,0 | World Class | |
| 5,0-7,5 | Expert | |
| 7,5-12,5 | Advanced | |
| 12,5-17,5 | Intermediate | |
| 17,5-22,5 | Casual Player | |
| 22,5-30,0 | Beginner | |
| 30,0+ | Distracted | |

Seules les décisions « non évidentes » comptent (une décision est évidente si l'écart d'équité entre meilleur et pire choix est < 0,001, entre autres critères). gnubg produit un équivalent (mEMG) et attribue un « Overall rating » d'Awful! à Supernatural selon le taux d'erreur normalisé.

**Débat de normalisation (documenté, à trancher explicitement)** : gnubg divise les erreurs par le nombre de coups **non forcés** (« unforced »), Snowie divise par le nombre **total** de coups. Douglas Zare (article « Normalizing Errors ») argumente que la méthode Snowie est supérieure. C'est un choix de conception réel qui affecte le chiffre affiché et donc l'étiquette.

### Séparation chance / compétence — spécificité que le backgammon gère déjà
gnubg calcule un « Luck rate » (différence entre l'équité après le meilleur coup pour le dé obtenu et l'équité moyenne sur tous les dés possibles) et un « Luck adjusted result » (réduction de variance, d'après les travaux de Douglas Zare). Les échecs n'ont pas ce problème ; le poker le gère via la variance/taille d'échantillon. Pour un outil de backgammon, **présenter clairement l'erreur (compétence) séparée de la chance est un atout pédagogique majeur** que les joueurs comprennent déjà.

## Recommendations — Taxonomie transposable au backgammon (6 thèmes défendables)

Chaque thème est calculable à partir de ce que gnubg/XG fournissent déjà (équités EMG, probabilités win/gammon/backgammon, décisions de cube, rollouts).

**Étape 1 — Fixer l'unité et les seuils AVANT toute étiquette.**
Utiliser l'**EMG** comme unité unique de gravité (comme Win%/EV ailleurs). Adopter 3 niveaux calqués sur gnubg : Doubtful ≥ 0,03, Bad ≥ 0,06, Very bad ≥ 0,12 (seuils par défaut modernes gnubg, à recalibrer selon le public). Ne **jamais** afficher les centièmes bruts sans contexte de gravité.

**Thème 1 — Erreurs de jeu de pions (checker play) vs erreurs de cube.** Séparation native gnubg/XG. Critère : EMG perdu, ventilé sur les deux compteurs. Première dimension, non négociable — les erreurs de cube valent souvent plus que les erreurs de pions et doivent être priorisées.

**Thème 2 — Type d'erreur de cube.** Reprendre les 6 sous-catégories gnubg : double manqué sous le point de cash, double manqué au-dessus, double erroné sous le point de drop, double erroné au-dessus du too-good, prise erronée, passe erronée. Critère : EMG perdu par sous-type, comparé au seuil de décision (CP/DP/TG).

**Thème 3 — Gravité normalisée (régularité vs gaffe isolée).** Distinguer, comme le fait la distribution de centipawns en échecs, « une falaise » (une seule très grosse erreur = problème de vigilance) vs « une pente douce » (beaucoup de petites imprécisions = problème de compréhension). Critère : PR/mEMG + distribution des erreurs (part du total d'EMG venant des 1-2 pires décisions). Ce diagnostic oriente vers des remèdes opposés.

**Thème 4 — Erreur par phase / type de position.** Reprendre les tracks de Backgammon Studio Heroes : opening, blitz, holding game, priming, backgame, bearoff, race. Critère : EMG moyen par type de position (nécessite un classifieur ; gnubg expose déjà des « position classes »).

**Thème 5 — Précision de fin de partie / tempo.** Analogue au « time management » d'Aimchess et au bearoff. Critère : EMG en bearoff/course (positions à faible variance où l'erreur est purement technique, donc corrigeable vite) vs positions de contact.

**Thème 6 — Compétence vs chance (méta-thème de présentation).** Toujours afficher le Luck-adjusted result à côté de l'erreur, pour empêcher l'utilisateur d'attribuer ses défaites à la malchance. Critère : Luck rate gnubg.

**Étape 2 — Explications en une phrase, sans LLM.** Gabarits à trous déclenchés par des règles à seuil, sur le modèle DecodeChess/Fritz. Ex. : « Erreur de cube (−0,15 EMG) : vous auriez dû doubler ; votre équité dépassait le point de cash. » Règle d'or : **si aucune règle ne dépasse son seuil de confiance, ne rien dire** (abstention, comme gnubg qui laisse le champ vide).

**Étape 3 — Drill-down.** De la statistique agrégée (« vos erreurs de backgame coûtent le plus ») vers la liste des positions concrètes, puis vers un rollout à la demande sur la position litigieuse (XG/gnubg le permettent).

**Seuils qui changeraient les recommandations :**
- Le signal PR ne devient fiable qu'à partir de **~50 matchs** ; sur un seul match, l'écart-type du PR est élevé (source GamesGrid). En dessous de cet échantillon (ou de quelques centaines de décisions non forcées), **ne pas afficher de « leak » par catégorie** — les forums backgammon rappellent qu'un PR sur un seul match est « à prendre avec des pincettes ».
- Si les faux positifs d'une étiquette dépassent ~10-15% (règle qui déclenche à tort), la retirer plutôt que la garder.

## Caveats & Anti-patterns à éviter (sourcés)

1. **Trop de catégories / étiquettes « feel-good » qui perdent leur sens.** Chess.com a dû refondre « Brilliant » : les utilisateurs se plaignaient qu'il y en avait trop (« Brilliant moves aren't all that brilliant »). Une définition trop laxiste vide le label de sens ; trop stricte le rend inexistant. Leçon : garder les étiquettes spéciales rares et défendables.

2. **Faux positifs = perte de confiance.** L'abondance de « Brilliant » a généré des accusations de triche et du ressentiment. La refonte CAPS→CAPS2 visait justement à recalibrer (« replicate the feeling of being graded on a test »). Les faux positifs détruisent la crédibilité de tout le système.

3. **Déclarer une « leak » sur échantillon insuffisant.** En poker, VPIP a besoin de 500+ mains, PFR 1000+, 3-bet 100+ opportunités pour être fiable ; en dessous, « treat the HUD numbers as random ». GTO Wizard ajoute des « sample-size tooltips and insufficiency indicators ». Transposition BG : ne pas conclure à une faiblesse de backgame sur 3 parties (fiabilité PR à ~50 matchs).

4. **Confondre erreur ponctuelle d'EV et biais systématique.** GTO Wizard : « don't just look at one hand with a large EV loss… look for similar trends ». La valeur pédagogique est dans le **pattern récurrent**, pas dans la gaffe isolée.

5. **Confondre résultat et qualité de décision (results-oriented).** « Your biggest losses were not necessarily your biggest mistakes. » Au backgammon, séparer explicitement chance et compétence (luck-adjusted) évite ce piège.

6. **Les buckets cachent les causes.** Critique documentée d'Aimchess : « 'Endgame weakness' is a bucket, and buckets hide causes » — l'étiquette compresse justement l'information pédagogique utile. Une catégorie doit pointer vers des positions concrètes, pas rester un mot abstrait.

7. **L'unité brute du moteur trompe l'utilisateur.** Le problème « 300 centipawns dans une position gagnée ≠ 300 centipawns en position égale » est la raison même du passage au Win% chez Lichess/chess.com. Au backgammon, l'EMG résout déjà cela — ne pas régresser vers des chiffres bruts non normalisés.

8. **Explications automatiques trop ambitieuses.** Fritz est reconnu « quite dumb at best ». Rester sur des phrases template fiables et rares plutôt que de générer du texte qui sera faux une fois sur cinq.

**Incertitudes signalées :**
- Les seuils de Win% de chess.com (5-10 / 10-20 / 20-100%) sont **officiellement documentés**, mais la **formule interne éval→Win%** reste privée et dépend du rating du joueur.
- La méthodologie d'Aimchess n'est pas publiée en détail ; ses scores sont un mélange de signal et de bruit selon les critiques de coachs.
- Les seuils XG « 4% / 1% / 0,5% » de chances de gagner pour blunder/error/outplay valent en jeu d'argent sans gammon ni cube ; ils varient au score en match.
- Les résultats de Guid & Bratko sont sensibles au moteur et à la profondeur (Crafty à 12 plies) — plusieurs réanalyses ultérieures (Stockfish) donnent des classements différents ; l'apport durable est la **méthode** (perte d'équité + complexité), pas le classement.