<!--
Rapport de recherche externe, produit le 2026-09-02 par une recherche
approfondie sur claude.ai à partir d'une demande écrite pour ce projet. Ce
n'est pas une note de conception de blunderDB : c'est la matière première qui a
servi à décider l'ADR-0026, et les niveaux de confiance de sa dernière section
(documenté / benchmark / académique / folklore / inférence) sont à lire avant
d'en citer un chiffre. Conservé tel quel, non révisé.
-->

# Quels réglages de répétition espacée exposer dans blunderDB : rapport de recherche

## TL;DR
- Pour un corpus fini de reconnaissance de motifs (20–300 cartes stables), les plafonds quotidiens d'Anki n'ont quasiment aucun sens ; le bon levier est un **nombre de cartes par session** ou une **durée de session**, pas un plafond de « nouvelles » ou de « révisions » par jour.
- Le réglage FSRS qui compte vraiment est la **rétention cible** ; ajuster cette seule valeur sans re-fitter les poids est défendable et documenté officiellement, tandis que toucher aux 19 poids est un piège reconnu (« should normally not be changed by humans »).
- À écarter explicitement : plafonds de révisions/jour (dangereux, « death spiral »), ordre de présentation configurable, réapprentissage complexe, les poids FSRS bruts, et surtout l'auto-ajustement de la rétention sur le taux de réussite observé.

## Réponses détaillées

### 1. Les plafonds quotidiens

**Ce que valent les deux limites dans Anki (documenté officiellement, manuel Anki — docs.ankiweb.net/deck-options.html).**
- `New cards/day` : plafonne le nombre de nouvelles cartes introduites par jour (défaut 20). Si on étudie moins que la limite ou qu'on saute un jour, il n'y a pas de report : le lendemain on revient à la valeur d'origine (« you won't be given more cards than your limit allows »).
- `Maximum reviews/day` : plafonne les cartes en révision (défaut 200). Quand la limite est atteinte, Anki n'affiche plus de révisions du jour même s'il en reste (« Anki will not show any more review cards for the day, even if there are more waiting »).
- **Interaction clé** : par défaut, la limite de révisions s'applique aussi aux nouvelles cartes — « By default, the review limit also applies to new cards, and no new cards will be shown when the review limit has been reached ». Une option « New Cards Ignore Review Limit » permet d'exempter les nouvelles. Le manuel précise aussi que les cartes d'apprentissage inter-jour (interday learning) comptent dans le plafond de révisions.
- **Règle empirique documentée** : « If you are consistently learning 20 new cards a day, you can expect your daily reviews to be roughly about 200 cards/day. » C'est le lien officiel entre débit d'entrée et charge de révision.

**Les dégâts (majoritairement folklore de communauté, mais convergent et recoupé par le manuel).**
- Le « death spiral »/backlog : sauter des jours empile les révisions, ce qui décourage et pousse à l'abandon. Documenté sur LessWrong (« An Opinionated Guide to Using Anki Correctly » : « This is the canonical Anki death spiral. Instead, set a daily limit on card reviews of 10 or 20 »), sur r/Anki et sur controlaltbackspace.org (page « Catching Up On Your Anki Reviews »).
- **Pourquoi limiter les révisions est réputé dangereux** : contrairement au plafond de nouvelles cartes (qui ralentit l'entrée), un plafond bas sur les révisions **tronque une file déjà due** — les cartes en retard restent cachées et s'accumulent silencieusement. Le manuel Anki ne montre par défaut pas plus de 1000 dues (controlaltbackspace : « By default, Anki decks have a daily limit of 200 reviews per day to avoid scaring people »), et recommande en cas de retard de monter `Maximum reviews/day` à 9999 puis d'arrêter les nouvelles cartes : « If you have a backlog of overdue review cards, it is recommended that you stop introducing new cards until you catch up with that backlog. » Le consensus est donc : plafonner les **nouvelles** est sain ; plafonner les **révisions** trop bas est risqué.

**Mécanismes récents (documenté : manuel + dépôt fsrs4anki-helper).**
- **Load balancer** : redistribue les échéances autour de leur date théorique (via le fuzz) pour lisser les pics ; intégré nativement à partir d'Anki 24.11, initialement add-on FSRS Helper. Il ne remplace pas les plafonds : il évite qu'ils soient nécessaires en aplatissant la courbe. (Discussion de conception : issue ankitects/anki #3116, L-M-Sherlock et Dae.)
- **Easy Days** : permet de réduire (~50 %) ou annuler (~0 %) les révisions certains jours de la semaine ; natif depuis 24.11. Ne fonctionne que pour les cartes en état « review », pas « (re)learning », et pas pour les intervalles < 3 jours.
- **Gestion du backlog** : options de tri (« Relative overdueness » en SM-2 ; son équivalent FSRS « Ascending retrievability ») pour prioriser ce qu'on est le plus susceptible d'oublier ; fonctions Postpone/Advance/Reschedule/Flatten dans le helper.
- Ces mécanismes **complètent** les plafonds (lissage a priori) plutôt qu'ils ne les remplacent.

**SuperMemo, Mnemosyne, WaniKani, Duolingo (mélange documenté / communauté).**
- **SuperMemo** : pas de plafond de charge ; Wozniak assume qu'on peut sauter des jours (« SuperMemo algorithm is not limited in any way, you can skip days without harm », supermemopedia.com), et que par l'effet d'espacement on peut même mieux retenir en apprenant plus lentement. Le forgetting index optimal (≈ inverse de la rétention) est documenté autour de 20–30 % pour le taux d'acquisition de connaissance maximal (supermemo.com, « Theoretical aspects of spaced repetition »), soit plus « permissif » que le 90 % d'Anki. Philosophie : la charge se régule par le débit d'entrée, pas par un plafond.
- **WaniKani** : pas de limite quotidienne native ; le pacing se fait en contrôlant le nombre de **leçons** (pas les révisions), avec la règle communautaire « rester sous 100 items Apprentice ». Des apps tierces (Gakugame) rajoutent un plafond de révisions justement parce que la file peut devenir écrasante (« You opened WaniKani. Saw 300+ reviews. Closed WaniKani »).
- **Duolingo** : pas de plafond de style Anki ; modèle propre Half-Life Regression (B. Settles & B. Meeder, 2016, « A Trainable Spaced Repetition Model for Language Learning », Proc. 54th Annual Meeting of the ACL, vol. 1, pp. 1848–1858, DOI 10.18653/v1/P16-1174 — HLR réduit l'erreur de plus de 45 % vs plusieurs baselines et, selon le TLDR Semantic Scholar, « improve Duolingo daily student engagement by 12% in an operational user study »), puis Birdbrain. Les sessions sont de taille bornée (7–20 mots/leçon). La charge est bornée par la structure de la session, pas par un compteur de dettes.
- **Mnemosyne** : ordonnanceur type SM-2, pas de culture de plafond quotidien fort.

### 2. Les réglages FSRS et leur ré-ajustement

**Ce que fait l'optimiseur (documenté : dépôt fsrs4anki + manuel).**
L'optimiseur parcourt l'historique de révisions et calcule, par apprentissage automatique (minimisant la **log loss**), les poids qui auraient le mieux prédit les résultats passés. Il ne prend qu'**une révision par jour et par carte** (« FSRS only takes into account one review per day »).

**Seuil de fiabilité (documenté, mais évolutif).**
- Anki < 24.04 : minimum 1000 révisions. Anki 24.04 : 400. Anki 24.06+ : plus de minimum imposé (« In Anki 24.06 and newer, there is no minimum limit »).
- Recommandation historique du guide FSRS/RemNote : « Before optimizing, you should do at least 1,000 reviews with the default weights – until you have plenty of data for the optimizer to work with, the default weights will be more effective than ones based on your study history. » La discussion GitHub ankitects/anki #3094 indique que la recherche permet de descendre le seuil jusqu'à 16.
- Fréquence de ré-optimisation conseillée : « Once per month should be more than enough », ou « chaque fois que le nombre de révisions double : 100, 200, 400… ».
- Point crucial pour blunderDB : le tutoriel FSRS insiste que « never optimizing at all and just sticking with the default weights would be a perfectly reasonable choice » et que les poids par défaut « are trained on a dataset including millions of reviews and should be excellent already right out of the box ». **Ne pas écrire d'optimiseur n'est pas un handicap majeur.**

**Gain mesuré défaut vs optimisé (documenté : benchmark).**
- Le srs-benchmark porte sur **10 000 utilisateurs d'Anki et ~727 millions de révisions** (dataset open-spaced-repetition/anki-revlogs-10k).
- Chiffres-clés (Expertium, expertium.github.io/Benchmark.html) : « FSRS-6 (recency) optimized vs FSRS-6 with default parameters: **84.3% superiority** » — donc dans ~16 % des cas les poids par défaut font aussi bien ou mieux (Expertium le souligne : « Wait, so in ~16% of cases, default parameters are better? »). Par ailleurs « FSRS-6 (with recency weighting) has a **99.6% superiority over Anki SM-2**, meaning that for 99.6% of users, log loss will be lower with FSRS-6 than with SM-2. »
- Les métriques : RMSE (bins), interprétable comme « l'écart moyen entre R prédit et R mesuré » (RMSE=0.05 ⇒ FSRS se trompe de 5 % en moyenne), et log loss (ce que l'optimiseur minimise réellement).
- Interprétation : l'optimisation apporte un gain réel mais **modeste** ; elle croît avec le volume d'historique. Pour un petit paquet stable, le gain d'un optimiseur maison serait faible et incertain. Rester sur les poids par défaut est justifié.

**Ajuster la rétention cible seule (documenté).**
- C'est le réglage « le plus important » selon le manuel : « This is the most important setting in FSRS. Higher retention leads to shorter intervals and more reviews per day. »
- Relation rétention→charge : croissante et **non-linéaire** (courbe en U côté charge/connaissance, avec un minimum de charge par utilisateur). Au-dessus de 90 % la charge « increases very quickly », au-dessus de 97 % elle devient « overwhelming ». La seule formule quantifiée **officielle** vient de SM-2 (manuel Anki, section Interval Modifier) : `log(rétention voulue)/log(rétention actuelle)` — passer de 85 % à 90 % impose « 35% more frequently » (soit +5 points de rétention ≈ +35 % de fréquence dans cette zone). Les chiffres popularisés « 90→95 % double la charge, 97 % la quadruple » sont des règles de communauté (blogs), **non officiels**.
- « Optimal retention » / CMRR (wiki « The Optimal Retention ») : il n'existe **pas de constante universelle** — les graphes du wiki sont volontairement sans valeur d'axe Y car l'optimum dépend des poids, du temps par carte, de la taille du paquet, etc. Depuis Anki 24.04 l'objectif est de minimiser le ratio charge/connaissance (minutes ÷ somme des probabilités de rappel).
- Fourchette recommandée aujourd'hui : plage permise 0.70–0.97 (étendue à 0.99 depuis Anki 23.10.1), défaut 0.90, « 80–95% reasonable ». Au-dessus de 0.97 déconseillé (« repetitions will be so frequent that you will dread doing your reviews »).
- **Ajuster la rétention sans re-fitter les poids est parfaitement défendable** : c'est exactement le levier de charge de travail que FSRS expose, indépendant de l'optimisation des poids.

**Auto-ajuster la rétention d'après le taux de réussite observé (position des auteurs).**
- Distinction officielle (Expertium, « Understanding retention in FSRS ») : « desired retention is what you want, and true retention is what you get. The closer they are, the better. » La rétention cible est un **choix de compromis** charge/connaissance, pas une cible à asservir.
- Les auteurs (Expertium, L-M-Sherlock) recommandent de combler l'écart desired/true en **ré-optimisant les poids**, pas en poussant la rétention cible pour chasser un taux. Expertium est explicite : le CMRR « has nothing to do with true retention, at all, whatsoever… It is not [peeking at your true retention]. »
- **Exception documentée** : le tutoriel FSRS autorise une hausse manuelle et bornée de la rétention cible « to compensate » si la true retention est durablement en dessous. Mais **aucun auteur n'endosse un mécanisme automatique** qui asservirait la rétention à un taux de réussite fixe.
- **Conclusion pour blunderDB : ne pas implémenter d'auto-ajustement de la rétention sur le taux de réussite.** Exposer la rétention comme un choix, et éventuellement afficher la « true retention » mesurée comme information, sans boucle de rétroaction automatique.

### 3. Ce qui compte vraiment pour un utilisateur

**Réglages réellement modifiés vs pièges (documenté + communauté).**
- Réellement utile et modifié : rétention cible, plafond de nouvelles cartes (contexte Anki où le corpus grossit), intervalle maximum.
- Pièges reconnus : les 19/21 poids FSRS (« They're combined because they are difficult to interpret and should normally not be changed by humans »), le bouton « Hard » utilisé comme un second « Again » (seul comportement que FSRS ne peut pas compenser, d'après le manuel), et le plafond de révisions réglé trop bas.

**Regrets / dépréciations des auteurs d'Anki (documenté).**
- Suppression des schedulers v1 puis v2 : le v3 est le seul depuis Anki/AnkiMobile 23.10 et AnkiDroid 2.17 (« the v3 scheduler is the default and only option »).
- Quand FSRS est activé, plusieurs réglages SM-2 « disparaissent » car « irrelevant » : Graduating interval, Easy bonus, Starting ease, Hard interval, New interval, Interval modifier. Mouvement délibéré de **simplification** — FSRS « has fewer user-customizable parameters than SM-2 ».
- CMRR (« Compute Minimum Recommended Retention ») comme bouton dédié a été retiré en Anki 25.07 au profit du simulateur « Help Me Decide ».
- L'option « Insertion Order » aléatoire est désormais déconseillée : « On recent Anki versions, you should leave this option set to Sequential, and adjust the display order instead. »

**Charge cognitive du paramétrage (inférence + littérature générale).**
- Le manuel Anki lui-même conseille de garder les défauts plusieurs semaines et prévient que « mistakes can reduce Anki's effectiveness » et « inappropriate adjustments may render Anki less effective ». Aveu implicite que trop d'options nuisent.
- La littérature générale (paradox of choice, Schwartz ; « sensible defaults » en design logiciel) soutient qu'exposer trop de réglages transfère une charge de décision à l'utilisateur et dégrade l'expérience. Pour un outil de niche comme blunderDB, chaque réglage exposé est une dette d'explication et de maintenance.

### 4. Le cas d'un corpus fini et thématique

**Répétition espacée et reconnaissance de motifs (documenté, académique).**
- La littérature radiologie/histologie/ECG confirme que l'espacement + l'entrelacement (interleaving) améliorent la reconnaissance de motifs et la rétention à long terme, mieux que la pratique massée. Revue systématique JACR (Thompson CP & Hughes MA, *J Am Coll Radiol* 2023 Nov;20(11):1092-1101, DOI 10.1016/j.jacr.2023.08.028, PMID 37683816) : sur **1 316 articles identifiés, seuls 8 essais** répondaient aux critères d'inclusion — preuves solides dans la littérature générale mais **peu d'études spécifiques à l'imagerie**.
- Entrelacement > blocage pour l'apprentissage de catégories : « interleaved practice of exemplars from multiple categories is superior for category learning » (radiologie pédiatrique, Springer, *Pediatric Radiology* 2019). Implication pour blunderDB : **mélanger les types de positions** dans une session est bénéfique, ce qui plaide à la fois contre un ordre trop rigide et contre l'idée qu'il faille exposer un réglage d'ordre.
- Expertise perceptuelle aux échecs : Gobet & Simon (1996), « Recall of random and distorted chess positions: Implications for the theory of expertise », *Memory & Cognition* 24:493–503 — « Comparison of a computer simulation with a human experiment supports the usual estimate that chess Masters store some **50,000 chunks** in memory » (estimation issue de Simon & Gilmartin 1973, fourchette 10 000–100 000). La reconnaissance de motifs de jeu est bien un objet de mémoire long terme — la répétition espacée est donc pertinente pour le backgammon.
- Aucune source ne documente des intervalles ou plafonds **spécifiques** à la reconnaissance de motifs, distincts de la mémorisation verbale ; FSRS s'applique tel quel.

**Outils d'échecs (documenté).**
- **Chessable (MoveTrainer)** : système propriétaire à **niveaux** (Level 1 → Level 8) plutôt que continu ; une réussite fait monter de niveau et allonge l'intervalle, un échec renvoie au niveau 1 (« If you get things wrong, you are back to the beginning »). Expose : choix du **schedule** (spaced repetition par défaut / custom / cyclique type Woodpecker où l'on fixe soi-même une date de fin de cycle) ; réglage d'intervalle par niveau. Beaucoup de réglages fins sont réservés au « Pro » (des utilisateurs jugent les défauts mal calibrés).
- **Chesstempo** : opening trainer avec répétition espacée ; expose des « advanced spaced repetition settings » (même aux utilisateurs gratuits), l'entrée de répertoire par PGN, la gestion automatique des transpositions, la **limitation de profondeur** d'entraînement, le périmètre (branche/répertoire/couleur), et l'entraînement ciblé sur les positions « the most resistant to spaced repetition learning ».
- **Listudy / Chessdriller / Lucas Chess** : alternatives libres, réglages minimaux (Chessdriller se présente comme alternative libre aux fonctions SR de Chessable/ChessTempo).
- Constat : ces outils exposent surtout **le périmètre d'entraînement** (quelle branche/variante, quelle profondeur) et **le rythme de cycle**, pas des plafonds quotidiens de style Anki.

**Un plafond quotidien a-t-il du sens pour un corpus fini ? (inférence sourcée).**
Non, essentiellement. Un plafond de « nouvelles cartes/jour » sert à étaler l'introduction d'un corpus qui grossit indéfiniment (langues, médecine). Pour un paquet de 20–300 cartes fixes, tout le corpus est introduit en quelques sessions, puis le régime devient purement des révisions espacées, dont le volume quotidien est **auto-limité** par l'algorithme et la taille du corpus. Le bon réglage est donc plutôt un **nombre de cartes par session** ou une **durée de session** (comme Chessable/Chesstempo bornent la session), qui donne à l'utilisateur un contrôle direct sur le temps sans tronquer dangereusement une file due.

## Tableau comparatif : logiciel → réglages exposés → portée → défaut

| Logiciel | Réglages de charge / ordonnancement exposés | Portée | Défaut |
|---|---|---|---|
| **Anki (FSRS)** | New cards/day ; Maximum reviews/day ; New cards ignore review limit ; Limits start from top ; Desired retention ; Maximum interval ; Learning/relearning steps ; Display order (gather/sort/new-review/review sort) ; Burying ; Easy Days ; Load balancing | Preset / par deck / today-only ; display order = deck sélectionné | 20 nouvelles/j ; 200 rév./j ; rétention 0.90 ; intervalle max 36500 j |
| **SuperMemo** | Pas de plafond de charge ; forgetting index (≈ inverse de rétention) ; débit d'entrée | Global / collection | Forgetting index ~10 % (rétention ~90 %) |
| **WaniKani** | Aucun plafond natif ; contrôle par le nombre de leçons | Global | Pas de limite ; heuristique communautaire <100 Apprentice |
| **Duolingo** | Rien d'exposé (HLR/Birdbrain automatique) ; taille de session fixe | Global | 7–20 mots/leçon |
| **Chessable (MoveTrainer)** | Schedule (spaced / custom / cyclique Woodpecker) ; intervalle par niveau (1–8) | Par cours | Spaced repetition à niveaux |
| **Chesstempo** | Advanced SR settings ; profondeur d'entraînement ; périmètre (branche/répertoire/couleur) ; ciblage des positions résistantes | Par set / répertoire | SR par défaut (premium pour custom) |
| **blunderDB (actuel)** | Rétention cible (0.7–0.99) ; intervalle max ; fuzz | Par paquet | — |

## Recommandation classée

**À exposer (2 à 5 réglages), par ordre de priorité :**
1. **Rétention cible** (déjà exposé) — le vrai bouton charge/qualité. Le garder, borné 0.80–0.95 en pratique (défaut 0.90). Seul réglage « puissant » qui ne demande pas de re-fitter les poids. *Statut : documenté officiellement.*
2. **Nombre de cartes par session** (ou durée de session) — NOUVEAU, à privilégier sur tout plafond quotidien. Adapté à un corpus fini et calqué sur ce que font les outils d'échecs. Donne le contrôle du temps sans tronquer une file due. *Statut : inférence sourcée.*
3. **Intervalle maximum** (déjà exposé) — utile pour un corpus qu'on veut « rafraîchir » périodiquement même quand FSRS proposerait des mois ; le garder mais avec un défaut élevé (p. ex. 365 j) et bien expliqué. *Statut : documenté.*
4. **Fuzz / lissage** (déjà exposé) — le garder activé par défaut ; c'est aussi le socle d'un futur lissage de charge (load balancing). *Statut : documenté.*
5. (Optionnel) **Réinitialisation / enterrement manuel d'une carte** exposé comme une action simple (pas un réglage) — utile pour retirer une position résolue une fois pour toutes.

**À écarter explicitement (et pourquoi) :**
- **Plafond de révisions/jour** — tronque une file due, crée le « death spiral », et n'a pas de sens sur un corpus fini où le volume est déjà auto-limité. *Documenté comme risqué : manuel Anki + communauté.*
- **Plafond de nouvelles cartes/jour** — utile seulement pour un corpus qui grossit ; ici tout le paquet est fini, préférer « cartes par session ». *Inférence sourcée.*
- **Ordre de présentation configurable** — la recherche sur l'entrelacement montre qu'un ordre mélangé (défaut) est bon ; exposer des options d'ordre ajoute de la charge cognitive pour un gain nul, voire nuit (un ordre fixe « leads to weaker memories », dit le manuel). Garder l'ordre non paramétrable, idéalement mélangé/entrelacé. *Documenté + académique.*
- **Les 19 poids FSRS** — « should normally not be changed by humans » ; sans optimiseur, les exposer serait un piège pur. *Documenté.*
- **Auto-ajustement de la rétention sur le taux de réussite** — contraire à la philosophie desired vs true retention ; risque d'emballement. *Position des auteurs.*
- **Réapprentissage configurable après échec** — complexité SM-2 que FSRS gère déjà ; laisser un comportement par défaut (retour en apprentissage court). *Documenté (FSRS masque ces options).*
- **Notion de « journée d'étude » / heure de bascule** exposée à l'utilisateur — à gérer en interne (voir pièges), pas à exposer.

## Pièges d'implémentation

- **Notion de journée et heure de bascule** : FSRS et Anki raisonnent en « jours ». Il faut une heure de bascule (`day_start`) car les révisions sont réparties par jour, « especially when reviews are divided by sleep » (dépôt fsrs-optimizer). Une erreur classique est de compter deux révisions du même jour comme deux points : l'optimiseur FSRS **n'utilise que la première révision du jour** par carte. Comme blunderDB journalise tout l'historique, prévoir la même règle si un jour on veut simuler/optimiser. Choisir une heure de bascule (p. ex. 4 h du matin) évite qu'une session nocturne compte pour le lendemain.
- **Ce qui se passe quand un plafond tronque un retard** : dans Anki, les cartes dues au-delà du plafond restent cachées et s'accumulent — d'où le conseil officiel de monter à 9999 pour voir l'ampleur réelle. Si blunderDB introduisait un plafond, il faudrait au minimum afficher le vrai nombre de dues et prioriser par « retrievability ascendante » (les plus à risque d'oubli d'abord), **jamais tronquer silencieusement**.
- **Interaction plafond × corpus fini** : sur 20–300 cartes, un plafond quotidien peut soit ne jamais mordre (inutile), soit, s'il est bas, créer un retard artificiel sur un corpus qui aurait tenu en une session — le pire des deux mondes. Préférer un objectif de session.
- **Rescheduling à la volée** : si l'utilisateur change la rétention cible, décider si on recalcule les échéances existantes (comme « Reschedule cards on change » d'Anki) ou seulement les futures. Anki avertit que les options « are not retroactive » par défaut ; documenter clairement le comportement choisi.
- **Fuzz et petits intervalles** : le fuzz/lissage ne s'applique pas aux intervalles < 3 jours dans le helper FSRS — attendre un comportement analogue et ne pas promettre un lissage parfait sur les cartes récentes.
- **Ne pas exposer un optimiseur factice** : sans entraîneur de poids, ne pas afficher de bouton « optimiser » qui ne ferait rien d'utile ; assumer les poids par défaut (documentés comme « excellent out of the box ») et le dire à l'utilisateur.

## Benchmarks / seuils qui feraient changer ces recommandations
- Si blunderDB accumule à terme un historique conséquent **par utilisateur** (ordre de grandeur ≥ quelques centaines à ~1000 révisions), écrire un optimiseur de poids devient défendable — le gain attendu reste néanmoins modeste (superiority 84,3 % vs défaut, ~16 % des cas où le défaut est meilleur).
- Si l'usage montre des files régulièrement > ~50–100 cartes dues par jour (peu probable sur corpus fini), envisager un **lissage de charge** (load balancing sur le fuzz) plutôt qu'un plafond.
- Si des utilisateurs se plaignent d'un écart persistant true/desired retention, la bonne réponse est d'implémenter l'optimisation des poids — **pas** un asservissement automatique de la rétention.

## Niveaux de confiance
- **Documenté officiellement (manuel Anki, dépôts open-spaced-repetition)** : mécanique des plafonds, disparition des options SM-2 sous FSRS, plage de rétention (0.70–0.97/0.99), seuils d'optimisation (1000→400→0), load balancer/easy days (24.11), suppression v1/v2 et du bouton CMRR (25.07), formule SM-2 de fréquence, règle « une révision/jour ».
- **Benchmark / articles des auteurs** : gain 84,3 % optimisé vs défaut et 99,6 % vs SM-2 (Expertium), dataset 10 000 utilisateurs / ~727 M révisions, HLR Duolingo (ACL 2016, +12 % engagement / −45 % erreur), distinction desired/true retention.
- **Académique** : chunks aux échecs (Gobet & Simon 1996, ~50 000), entrelacement > blocage (Springer), revue JACR 2023 (8 études sur 1 316).
- **Folklore de communauté (convergent)** : « death spiral », heuristiques WaniKani (<100 Apprentice), chiffres « 90→95 % double la charge » (blogs, non officiels).
- **Inférence sourcée** : pertinence d'un « nombre de cartes / durée de session » plutôt qu'un plafond pour un corpus fini ; charge cognitive du paramétrage.