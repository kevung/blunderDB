# Pédagogie du backgammon et répétition espacée à réponse graduée : synthèse et recommandations pour blunderDB

## TL;DR
- **La conversion recommandée s'appuie sur les seuils d'erreur natifs des moteurs** : XG marque par défaut une « erreur » à partir de 0,020 d'équité normalisée et un « blunder » à partir de 0,080 (valeurs par défaut de l'auto-marquage : error 0.020, blunder 0.080, outplay 0.010). En calquant la note FSRS sur ces paliers (coup optimal → Easy/Good ; 0–0,020 → Good ; 0,020–0,080 → Hard ; ≥0,080 → Again), on obtient un barème directement comparable au PR réel.
- **Le PR est déjà une mesure continue d'erreur en millièmes d'équité normalisée (EMG)** ; blunderDB, XG et gnuBG partagent la même logique (erreur totale / décisions non forcées), ce qui rend un « PR d'entraînement » techniquement possible — mais structurellement biaisé vers le pire, car on ne révise que des positions ratées.
- **Les preuves d'efficacité sont surtout indirectes** : la littérature (effet de test, difficultés désirables, répétition espacée) soutient fortement le principe, mais aucune étude publiée ne mesure spécifiquement le gain de PR au backgammon ; les témoignages de progression restent anecdotiques.

## Key Findings

1. **Le PR est une mesure d'erreur continue déjà normalisée.** XG calcule PR = (équité totale perdue / nombre de décisions non forcées) × 500. L'unité est le millième d'équité normalisée (EMG) par décision. blunderDB reproduit exactement cette logique et aligne ses indicateurs PR / Snowie Error Rate / coût MWC sur XG et gnuBG.

2. **Les seuils d'erreur des moteurs fournissent un barème tout prêt.** XG (documentation officielle, auto-mark dialogue) : « Errors: Any play where the error in equity between the move played and the computer's pick is 0.020 or more · Blunders: … is 0.080 or more » ; valeurs par défaut « error:0.020, blunder:0.080, outplay:0.010 ». gnuBG : « doutable / mauvais / très mauvais » paramétrables (défaut « doutable » = 0,02). Douglas Zare considère toute erreur de jeu de pions > 0,080 en ouverture/milieu de partie comme un blunder, et ne compte pas les écarts < 0,020.

3. **La répétition espacée graduée est déjà pratiquée dans d'autres jeux avec une réponse objectivement mesurée** : Lichess note ses puzzles en Glicko-2 (« To determine the rating, each attempt to solve is considered as a Glicko-2 rated game between the player and the puzzle ») ; GTO Wizard attribue un GTOW Score de −100 % à +100 % et une perte d'EV par décision ; Chessable MoveTrainer utilise un système de niveaux 1-8. Tous convertissent une performance mesurée en planification de révision.

4. **FSRS traite les notes 1-4 de façon non binaire, et des variantes acceptent déjà des notes continues.** Le cœur de FSRS distingue bien Hard/Good/Easy ; seule la fonction de perte pendant l'optimisation binarise (Again=0, sinon=1). Des implémentations (ex. @squeakyrobot/fsrs) exposent un `autoRating` convertissant un temps de réponse en note continue 1,0–4,0.

5. **Un « PR d'entraînement » sera structurellement pire que le PR réel** à cause du biais de sélection (on ne stocke que les positions ratées), puis deviendra artificiellement bon par reconnaissance. Il faut le calibrer contre une cohorte de positions non vues.

## Details

### VOLET 1 — Méthodes d'entraînement au backgammon

**Revue de match.** La méthode centrale reconnue par tous les coachs est l'analyse systématique post-partie avec XG ou gnuBG. Le conseil courant est de « revoir au moins une partie par session, en se concentrant sur les positions où votre coup et celui du moteur diffèrent ». XG lui-même conseille dans sa documentation : « Analysez tous vos matchs systématiquement. »

**Positions de référence (reference positions).** Phil Simborg a un article dédié sur bkgm.com expliquant l'importance et l'usage des positions clés. L'UKBGF liste « apprendre quelques positions de référence », « apprendre des formules de course / bear-off » et « apprendre les nombres de Neil » parmi ses cinq axes d'amélioration. Des cursus payants structurés existent (zzbackgammon, avec parcours Intermédiaire / Avancé / World-class annonçant viser « le range PR 5 »).

**Comptage de pips.** Trois grandes méthodes documentées sur bkgm.com :
- *Cluster Counting* de Jack Kissane (mémorisation de clusters de référence, multiples de 5/10 ; Kissane est réputé compter presque n'importe quelle position en cinq secondes) ;
- *Half-Crossover* de Douglas Zare (estimation rapide + petit calcul) ;
- *Naccel* de Nack Ballard (compte en « supes » de 6 pips, exploite la géométrie du plateau), avec sa formule de course (« Nerf ») : si l'avance du leader est positive, diviser par 2 et ajouter 7 ; si négative, ajouter 8 (+1 pour redouble minimal, +4 pour passe marginale).

**Entraînement au cube et formules de course.** Nack Ballard recense les formules classiques : Robertie (8 % = double marginal, 9 % = redouble marginal, 12 % = take/pass limite), Weaver, Trice. La « loi de Woolsey » (si vous ne savez pas si l'adversaire doit prendre ou passer, doublez) et les nombres de référence (Neil's numbers) sont enseignés couramment.

**Rollouts « à l'aveugle ».** Le principe (deviner l'équité/le meilleur coup avant de consulter le rollout) est une application directe de l'effet de test ; c'est une pratique recommandée sur les forums mais sans étude dédiée.

**Preuves d'efficacité.** Elles sont majoritairement anecdotiques (témoignages de forums BGonline, r/backgammon, forum Galaxy). La seule donnée empirique solide transposable vient de la méta-analyse de Macnamara, Hambrick & Oswald (2014, *Psychological Science* 25(8):1608–1618) : « Percentage of variance in performance explained by deliberate practice was 26% for games (r = .51, p < .001), 21% for music, 18% for sports, 4% for education, and less than 1% for professions » — la pratique délibérée est importante mais loin d'être suffisante. Ericsson conteste ce chiffre en insistant sur l'individualisation de la pratique (coaching), qu'il estime sous-évaluée dans la méta-analyse. **À signaler : aucune étude publiée ne mesure directement le gain de PR d'une méthode d'entraînement donnée au backgammon.**

### VOLET 2 — Indicateurs de progression au-delà du PR

**Définition et historique.** Le PR (Performance Rating) est né du besoin de remplacer le « Snowie error rate ». Différences clés :
- *Snowie ER* = somme des erreurs (EMG) / nombre de coups des **deux** joueurs.
- *gnuBG / XG* = erreurs / décisions **non forcées** du seul joueur noté. Résultat ≈ 2× le Snowie ER.
- gnuBG affiche les deux ; une investigation sur ~300 matchs montre que l'ER gnuBG est en moyenne 1,4× l'ER Snowie.
- XG divise par 500 pour retomber « dans le même ordre de grandeur » que Snowie (diviser par 1000 donnerait des millipoints EMG par décision, jugé plus rigoureux par certains).

**Échelle de niveaux (ER Snowie, cubes + pions) :** 0–1,2 « Extra-terrestrial » ; 1,2–4,4 « World Class » ; 4,4–5,9 « Expert » ; 5,9–8,8 « Advanced » ; 8,8–12,6 « Intermediate » ; 12,6–18,5 « Beginner » ; >18,5 « Novice » (le seuil world-class est précisément 4,400 et en dessous). En échelle PR moderne (XG), on cite couramment : 0–3 world-class, 4–6 expert, 7–10 fort joueur de club, 11–15 intermédiaire (bandes indicatives, pas des seuils stricts).

**Décompositions.** Les outils séparent le jeu de pions (checker play) des décisions de cube. XG et gnuBG donnent aussi le « luck-adjusted result » et le taux de chance, ainsi qu'un compte des coups marqués « très bon / doutable / mauvais / très mauvais ». Backgammon Galaxy (confirmé par son CEO Marc Olsen) utilise XG en 1-ply pour la chance et 3-ply pour les décisions ; le PR et le luck rate y sont normalisés. Le Galaxy Rating v2 suit une formule Elo modifiée : P(victoire) = 1/(1+10^((R2−R1)/800)), avec un facteur K = min(16, max(5, 32 − 0,012·moyenne des ratings)) et un facteur de longueur de match ^0,75 ; les points ne sont attribués que si le même joueur gagne le match ET le PR (« decisive result »).

**Significativité statistique.** Le PR d'un match unique est très bruité (écart-type de l'ordre de 2,0 mEMG pour des longueurs de 5-9 points), dominé par la sélection aléatoire des positions rencontrées ; le signal ne devient fiable qu'à partir d'une cinquantaine de matchs. Le PR étant pondéré par le nombre de décisions, on ne peut pas simplement moyenner les PR de plusieurs matchs — il faut sommer les erreurs et diviser par le total des décisions non forcées.

**Présentation par les outils.** blunderDB (v0.19+) affiche un panneau Stats avec PR (Performance Rate), Snowie Error Rate et coût MWC, une barre de filtres (joueur, tournoi, dates, type de décision, longueur de match), un onglet Dashboard (cartes de niveau / PR glissant / top blunders), un onglet Progression (courbe par tournoi, nuage de points par match) et un onglet Errors (répartition des actions de cube, histogramme d'ampleur d'erreur). Depuis v0.33 (26 août 2026), un onglet Players compare PR global / checker / cube, Snowie ER, blunders et chance par joueur, et l'onglet Errors distingue la *direction* des erreurs de cube (ligne *Offer* : doubles manqués/prématurés ; ligne *Answer* : passes/prises fautives). L'alignement PR/Snowie ER/MWC avec XG et gnuBG utilise un **seuil de 0,16 d'équité pour les cubes « proches »**.

### VOLET 3 — Répétition espacée à réponse graduée

**FSRS (Free Spaced Repetition Scheduler).** Modèle DSR : Difficulty (1-10), Stability (jours avant que la probabilité de rappel tombe à la rétention cible, souvent 90 %), Retrievability (probabilité actuelle de rappel). Les notes Again/Hard/Good/Easy (1-4) modulent D et S. Point crucial pour le projet, énoncé par le blog technique de référence (Expertium) : la note « ne signifie pas que FSRS traite les notes de façon binaire, il distingue bien Hard, Good et Easy ; la perte n'est binarisée (Again=0, sinon=1) que pendant l'optimisation ». FSRS a été intégré à Anki dans la v23.10, publiée le 31 octobre 2023 (initialement en option, opt-in, avant de devenir le planificateur recommandé) ; il a été créé par Jarrett Ye et la communauté open-spaced-repetition, avec des paramètres par défaut ajustés sur ~700 millions de révisions.

**SM-2 et l'échelle 0-5.** SuperMemo définit q de 0 à 5 : 0 « blackout total » ; 1 réponse fausse mais familière ; 2 réponse fausse mais facile à retenir ; 3 réponse correcte avec effort important ; 4 réponse correcte après hésitation ; 5 rappel parfait. **C'est déjà une note graduée de qualité de rappel.** Anki l'a réduite à 4 boutons (un seul « échec ») car « l'échec ne représente qu'une petite part des révisions, et ajuster l'ease se fait suffisamment en variant les réponses positives ». C'est directement pertinent pour blunderDB : une erreur en équité est précisément une mesure continue de « qualité de la réponse », analogue au q de SuperMemo, mais mesurée objectivement au lieu d'être auto-déclarée.

**Implémentations à réponse mesurée dans d'autres jeux :**
- *Chessable MoveTrainer* : note coup correct/incorrect, niveaux 1 à 8 ; un coup juste monte de niveau (intervalle plus long), un coup faux revient au niveau 1. Option de planning cyclique (méthode Woodpecker).
- *Lichess Puzzles* : rating Glicko-2, « chaque tentative de résolution est considérée comme une partie Glicko-2 entre le joueur et le puzzle » ; les puzzles thématiques rapportent moins de points car le thème est un indice ; le Puzzle Dashboard par thème sert à cibler les faiblesses. Puzzle Storm/Streak/Racer ajoutent la contrainte de temps.
- *Chess.com Puzzle Rush / Battle* : score sous contrainte de temps ; Chess Insights pour l'analyse.
- *GTO Wizard (poker)* : « The primary metric … is the GTOW Score. This assigns a value between (-100% to +100%) to each move » plus « total EV loss (in big blinds) … average EV loss per hand ». Catégories : Best Move / Correct Move / Inaccuracy (« jouée < 3,5 % du temps en GTO mais sans perte d'EV significative ») / Wrong Move / Blunder (« jamais jouée et perte d'EV significative »). C'est le modèle le plus proche de ce que blunderDB veut faire : une note dérivée d'une perte d'EV continue.

**Littérature académique.**
- Effet de test / retrieval practice : Roediger & Karpicke (2006), *The Power of Testing Memory*, *Perspectives on Psychological Science* 1(3):181–210 — récupérer bat relire pour la rétention à long terme.
- Difficultés désirables (Bjork, 1994 ; Bjork & Bjork, 2011) : les conditions qui rendent l'apprentissage plus dur à court terme (espacement, entrelacement, récupération) produisent un apprentissage plus durable et transférable.
- Feedback immédiat et correctif renforce l'effet de test (Roediger & Butler, 2011) ; une tentative ratée puis corrigée est un échafaudage, pas un dommage.
- Espacement : la pratique espacée bat la pratique massée ; les preuves sont mitigées sur la supériorité des intervalles expansifs (Karpicke & Roediger, 2007).
- Pratique délibérée : Ericsson vs critique de Macnamara et al. (26 % de variance pour les jeux).

**Pièges documentés.**
- *Biais de sélection* : ne réviser que ses erreurs surreprésente les positions difficiles.
- *Reconnaissance / leakage* : on finit par reconnaître la position au lieu de raisonner (le PR d'entraînement s'améliore artificiellement).
- *Variabilité du dé* : une même position peut se jouer avec plusieurs jets, ce qui complique la notion de « bonne réponse ».
- *Sur-apprentissage de positions spécifiques* plutôt que de principes (mémorisation vs compréhension).
- *Interleaving vs blocking* : entrelacer les thèmes est plus efficace que réviser par blocs, mais subjectivement plus difficile.

Un outil existant, AnkiGammon, transforme déjà des analyses XG/gnuBG en cartes Anki, avec un seuil d'erreur par défaut de 0,080, réglable séparément pour pions et cube — précédent utile mais qui reste sur une auto-évaluation Anki classique, sans dériver la note de l'erreur mesurée.

### VOLET 4 — Le quiz comme mesure

**Construire un « PR d'entraînement » comparable au PR réel.** Il faut reprendre exactement les conventions des moteurs :
- même unité (millièmes d'EMG) ;
- ne pas compter les décisions forcées (un seul coup légal) ni les décisions triviales (courses 100 % décidées, coups « non close ») ;
- pour le cube, XG n'analyse en 3-ply que si l'écart No Double / Double est < 0,200 (au-delà, la décision est « évidente ») ; blunderDB utilise un seuil de 0,16 pour les cubes proches, aligné sur XG/gnuBG.

**Formule XG :** PR = équité totale perdue × 500 / nombre de décisions non forcées.

**Conversion MWC → EMG.** En match, l'erreur brute est en MWC (chance de gain du match) ; elle est normalisée en EMG (équité équivalente en money game) pour rendre les erreurs comparables entre scores. Zare souligne que la normalisation EMG distord certains scores : au DMP il est « particulièrement facile » d'obtenir un ER bas, et une même faute conceptuelle (sauver le gammon) peut coûter 0,002 ou 2,000 EMG selon le score.

**Biais d'un PR d'entraînement et corrections :**
- *Structurellement pire* (sélection de positions ratées) → comparer à un « PR de référence » sur positions non vues.
- *Artificiellement bon avec la répétition* (reconnaissance) → séparer « première vue » et « révisions », suivre les deux courbes.
- *Pas de fatigue ni de pression de temps* → optionnellement chronométrer.
- *Distribution non représentative* (surreprésentation du difficile, absence de contexte de match/momentum) → intégrer une cohorte de contrôle de positions correctement jouées, échantillonnées aléatoirement, pour calibrer.

## Recommandations

### 1. Barème de conversion « erreur en équité → note FSRS »

Le barème s'appuie directement sur les seuils natifs de XG (erreur 0,020 ; blunder 0,080, valeurs par défaut de l'auto-marquage confirmées par la documentation XG) et sur la pratique de Zare. On raisonne en équité normalisée (EMG) sur le coup effectivement joué.

**Barème de base (jeu de pions et cube confondus, en équité EMG) :**

| Erreur mesurée e (EMG) | Interprétation moteur | Note FSRS | Justification |
|---|---|---|---|
| e = 0 (coup optimal) | coup du moteur | **Easy** | joué parfaitement |
| 0 < e ≤ 0,020 | sous le seuil d'« erreur » XG | **Good** | quasi-correct, non compté comme erreur par XG |
| 0,020 < e ≤ 0,080 | « erreur » XG | **Hard** | erreur réelle mais non grossière |
| e > 0,080 | « blunder » XG | **Again** | faute grave, à revoir vite |

**Traitement du temps de réponse.** Recommandation : ne PAS laisser le temps déclasser une réponse optimale (contrairement à SM-2 où l'hésitation fait passer de 5 à 4), car en backgammon la justesse prime. On peut néanmoins utiliser un temps très long sur un coup optimal comme signal secondaire pour rétrograder Easy → Good (option désactivable). Le temps ne doit jamais surclasser une erreur.

**Cas « bon coup trouvé par hasard ».** C'est le principal ennemi de la validité. Deux parades : (a) sur les décisions de cube et les positions à plans multiples, demander une justification/second choix ; (b) traiter la première vue d'une position différemment des révisions, et exiger plusieurs succès consécutifs avant d'allonger fortement l'intervalle (ce que FSRS fait naturellement via la stabilité). Un coup optimal isolé donne Good, pas Easy, tant que la stabilité est faible.

**Faut-il différencier cube et checker ?** Oui, légèrement. Les erreurs de cube ont une distribution plus dispersée et un « coût par erreur » moyen plus élevé ; Zare abaisse d'ailleurs ses seuils au DMP (blunder à 0,050, erreur à 0,010). Recommandation : garder le même barème par défaut, mais permettre un profil « strict » (erreur 0,010 / blunder 0,050) pour les scores type DMP et pour les décisions de cube, cohérent avec la pratique experte. Ne PAS moduler le barème par difficulté intrinsèque de la position : l'erreur en équité capture déjà l'enjeu, et le paramètre Difficulty de FSRS s'ajuste tout seul aux positions chroniquement ratées.

### 2. Alimenter FSRS proprement

- Conserver l'unité EMG et n'inscrire au « PR d'entraînement » que les décisions non forcées, exactement comme XG/gnuBG/blunderDB (seuil de cube « proche » à 0,16).
- Journaliser deux PR distincts : **PR première vue** (le seul comparable au PR réel) et **PR révision** (mesure la mémorisation, pas la compétence brute).
- Maintenir une **cohorte de contrôle** : injecter périodiquement des positions correctement jouées (non « blunderisées ») pour estimer le biais de sélection.
- Exploiter le mode Cram déjà présent dans blunderDB (sert des positions aléatoires sans altérer le calendrier SRS) pour l'échauffement et les cohortes de contrôle.

### 3. Routine hebdomadaire type (pour la documentation blunderDB)

Chaque élément est rattaché à une justification sourcée ; les durées précises restent indicatives faute d'étude spécifique au backgammon.

- **10-15 min/jour de révision SRS** des positions dues (barème ci-dessus). *Justification : effet de test + espacement (Roediger & Karpicke 2006 ; Bjork).*
- **2-3 matchs joués et intégralement analysés/semaine** dans XG ou gnuBG, positions ratées importées dans blunderDB. *Justification : revue de match, conseil universel des coachs et de la documentation XG (« Analysez tous vos matchs systématiquement »).*
- **1 session de drill de positions de référence/semaine** (ouverture, take points, bear-off), en mode Cram de blunderDB pour ne pas perturber le calendrier SRS. *Justification : positions de référence (Simborg, bkgm.com) ; UKBGF.*
- **5-10 min de comptage de pips 2-3×/semaine** (une méthode : Cluster/Half-Crossover/Naccel). *Justification : bkgm.com, prérequis des décisions de course et de cube.*
- **1 revue thématique/semaine** ciblant la faiblesse détectée par le panneau Stats (ex. cube « Answer », backgame). *Justification : entrelacement ciblé sur les faiblesses (Lichess Puzzle Dashboard ; GTO Wizard).*
- **Suivi mensuel du PR réel** agrégé sur plusieurs matchs (le PR d'un match isolé est trop bruité, σ≈2,0 mEMG). *Justification : significativité statistique.*

### Seuils qui feraient changer la recommandation
- Si le PR révision plonge sous le PR réel de plus de ~1 mEMG, c'est un signe de mémorisation par reconnaissance : allonger les intervalles ou renouveler le stock de positions.
- Si le PR d'entraînement première vue converge vers le PR réel, le corpus est représentatif ; sinon, rééquilibrer (moins de blunders extrêmes, plus de décisions moyennes).
- Si le taux de réussite observé sur un deck s'écarte durablement de la rétention cible, utiliser le nudge automatique de blunderDB (`anki.optimizeParams`) pour recaler la cible.

## Caveats
- **Preuves d'efficacité limitées** : le transfert positions isolées → performance en partie est plausible (difficultés désirables, effet de test) mais non démontré empiriquement pour le backgammon ; les gains de PR chiffrés circulant sur les forums (type « de PR 8 à PR 4 ») sont anecdotiques et non vérifiables.
- **Seuils XG confirmés depuis la documentation officielle** (auto-mark : error 0.020, blunder 0.080, outplay 0.010), mais entièrement personnalisables par l'utilisateur ; les bandes de niveaux PR (0-3 world-class, etc.) sont indicatives et non des seuils stricts.
- **La normalisation EMG est imparfaite** à certains scores (DMP notamment), ce qui affecte tout PR, réel comme d'entraînement.
- **Le contenu exact du modèle statistique interne de blunderDB** (formule PR verbatim, définition précise du seuil 0,16, et surtout le mécanisme précis de notation du panneau Anki — auto-dérivée de l'erreur ou auto-déclarée) n'a pu être extrait au-delà du changelog officiel et de la structure des pages ; à confirmer sur les pages « Annex: Statistics model — XG/gnuBG/blunderDB alignment » et « Anki Panel » de la documentation.
- **La variabilité du dé** limite la notion de « bonne réponse » : une position peut avoir plusieurs jets ; blunderDB stocke une position + un jet donné, ce qui est cohérent avec une carte SRS unique mais ne teste pas la robustesse du raisonnement à d'autres jets.
- **FSRS dans Anki 23.10 était opt-in**, pas activé par défaut ; la formulation « par défaut » vaut pour des versions ultérieures/déploiements où il est devenu le planificateur recommandé.