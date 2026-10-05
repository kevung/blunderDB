# Jouer contre le moteur (Duel) — horloges, temps par décision, dés : état des sources

Rapport produit à partir de `tasks/duel/prompt-deep-search.md`. **Passe documentaire inégale, assumée** : les sections D (horloges) et E (temps comme donnée) ont été traitées en profondeur à partir des règlements et des sources primaires ; les sections A, B, C, F et G n'ont reçu qu'une passe courte et leurs trous sont marqués `[NON TROUVÉ]` plutôt que comblés. Marqueurs : `[MESURE]`, `[ÉDITEUR/DÉCLARÉ]`, `[EXTRAPOLÉ]`, `[NON TROUVÉ]`.

## TL;DR

- **Cadence de référence unique** : 2 minutes de réserve par point de match, plus 12 secondes de **délai simple** (« Simple Delay ») par coup, qui ne s'accumulent pas. C'est le texte de l'USBGF (règles en ligne) et du WBGF (édition 2.1, août 2023) ; c'est aussi le réglage de Backgammon Galaxy selon un outil tiers. `[ÉDITEUR/DÉCLARÉ]` pour Galaxy, règlement pour les deux autres.
- **Le délai n'est pas un Bronstein dans les textes en vigueur** : la version 2007 des règles américaines exigeait le « Bronstein », celle de 2013 et le WBGF 2.1 imposent le « Simple Delay ». Comme le délai de 12 s ne s'accumule pas, l'implémenter en incrément Fischer serait faux.
- **Sanction d'un dépassement : perte du match**, pas une pénalité en points (USBGF, WBGF). Cas particuliers : les deux joueurs à zéro sans pouvoir dire qui est tombé le premier, le match continue sans horloge, sauf « gin » où le joueur sûr de gagner est déclaré vainqueur (WBGF). Les pénalités en points (1 point par tranche de 5 minutes de retard) ne visent que le **retard au départ ou aux pauses**, jamais le temps de réflexion.
- **L'horloge ne s'arrête pas aux dés** : on termine son tour en frappant l'horloge, pas en ramassant les dés. Elle s'arrête entre les parties, pour un dé tombé, un litige, un appel au directeur, et quand on **refuse** un videau ou propose une concession ; accepter un videau relance l'horloge de l'adversaire.
- **Format `.xg`** : l'en-tête porte `TTimeSetting` (type Aucune/Fischer/Bronstein, durée initiale, temps ajouté ou réservé par coup, **pénalité en points** au dépassement, temps restant des deux camps) et **chaque enregistrement de videau** porte `TimeBot`/`TimeTop` (temps **restant**, pas durée de la décision). Aucun champ « durée de la décision » : elle se déduit par différence. Les quatre `.xg` de blunderDB sont tous **non chronométrés** (tous les champs à zéro, version 32). `[MESURE]`
- **Ailleurs, rien** : `.mat` (aucun champ), SGF de gnubg (propriétés `BL`/`WL` déclarées « not currently used »), replay bgammon.org (aucun horodatage de coup). Il n'existe donc **aucun format partagé** pour le temps par décision : blunderDB devra définir le sien.
- **Temps et erreur** : aux échecs, le temps restant prédit la bévue (taux qui monte vite quand t tend vers 0, plat au-delà de 10 s), mais la **difficulté de la position** pèse davantage ; la corrélation brute entre temps passé et erreur est trompeuse, car on réfléchit plus dans les positions difficiles. Rien de comparable trouvé au backgammon.

---

## D. Horloges de backgammon (priorité)

### D.1 Règlements en vigueur

| Source | Réserve | Délai par coup | Dépassement | Statut |
|---|---|---|---|---|
| USBGF, règles pour le jeu en présentiel | 2 min × longueur moyenne restante du match | 12 s, Simple Delay, « sauf avis contraire du directeur » | « A player loses the match when it is noticed that their reserve time has expired, unless they can then validly claim the game and match. » | Règlement en vigueur |
| WBGF Tournament Rules, édition 2.1 (août 2023), § 3.5 (v) et (vi) | `((RA+RB)/2) × durée par point`, RA et RB étant les points restant à chaque joueur ; 2 min par point en simple | 12 s, Simple Delay, « ne s'accumule pas » ; « or as otherwise specified by the Tournament Director » | § 4.3 (iii) : « The player whose time has run out has lost the match. » | Règlement en vigueur ; texte lu dans la reprise de Backgammon Germany |
| US Backgammon Clock Rules, mars 2013 | 2 min par point, moins 1 min par point déjà marqué par l'un ou l'autre | 12 s par coup (18 s en double) | « A player who exhausts his reserve time before he can validly claim the match is declared the loser. » (§ 5.3) | Historique |
| Mêmes règles, octobre 2007 | identique | identique, mais **système Bronstein** exigé | identique | Historique |
| Règles unifiées, février 2001 | tableau en minutes selon les points restants | aucun | **2 points de pénalité** à l'adversaire la première fois, 1 point ensuite, puis **+5 minutes** aux deux horloges | Abandonné `[EXTRAPOLÉ]` (aucune reprise dans les textes récents) |
| XG, structure `TTimeSetting` | `Time1` en secondes | `Time2` : temps ajouté (Fischer) ou réservé (Bronstein) par coup | `Penalty` et `PenaltyMoney` : **pénalité en points** « when running out of time » | Logiciel, non règlement |

Sources : <https://usbgf.org/tournament-rules/rules-for-in-person-play/>, <https://bkgm.com/tournaments/US-Backgammon-Clock-2013.html>, <https://bkgm.com/tournaments/US-Backgammon-Clock-2007.html>, <https://bkgm.com/tournaments/UnifiedClockRules.html>, <https://www.extremegammon.com/XGformat.aspx>, et le texte WBGF 2.1 repris par Backgammon Germany : <https://www.backgammon-deutschland.de/en/turnierregeln> (PDF `b88a41_1dd23e58ae374e7691f892a14acaf2a4.pdf` du même site).

Notes :

- **Réserve recalculée au score** `[EXTRAPOLÉ]` : « 2 min × longueur moyenne restante » (USBGF) et « 2 min par point, moins 1 min par point marqué par l'un ou l'autre » (2013) donnent la même valeur, 2·L − (a + b) minutes pour un match à L points au score a-b. Pour L = 7 à 0-0 : 14 min par joueur. Aucun des textes lus ne dit si la réserve est **reposée** à chaque partie ou **fixée une fois** pour le match ; XG a un drapeau `PerGame` pour les deux cas.
- **La version 2007 contredit la 2013** sur le type de délai. Le titre de la page 2013 conserve « Bronstein Simple Delay Clocks Only — October 2007 March 2013 », signe d'une réécriture partielle. Retenir le texte courant (USBGF et WBGF : Simple Delay).
- **Le Bronstein est une autre mécanique** : il restitue le temps utilisé jusqu'au plafond du délai (même effet que le délai simple tant que le coup est plus court que le plafond, mais la restitution passe par la réserve). Le Fischer ajoute le temps **avant** le coup et s'accumule. Seul le délai simple est conforme au règlement.
- **Débat sur le choix** : le fil « A better clock for backgammon » du forum bgonline.org propose une alternative (<https://www.bgonline.org/forums/webbbs_config.pl?noframes%3Bread=175745>) ; la page n'a pas pu être lue. `[NON TROUVÉ]`

### D.2 Ce qui arrête l'horloge, cube et abandon

Dans le WBGF 2.1 (§ 4.1 (vi), § 4.3, § 4.4 (iii), § 4.5) :

- Fin du tour : « In matches played using a game clock a player ends his turn by activating his opponent's time. » Ramasser les dés n'arrête rien ; sans horloge, le tour se termine en levant un dé. Si l'adversaire n'a aucun coup légal, il doit tout de même frapper l'horloge.
- **Pauses autorisées, sept cas** : partie terminée ; pauses (on note les temps) ; appel du directeur ; dés ramassés avant la fin du tour adverse ; dés ramassés avec l'horloge de l'adversaire déjà lancée ; contestation d'un coup illégal ; joueur qui estime la partie réglée (abandon d'une partie). Un lancer invalide permet de pauser et de **relancer le délai**.
- **Videau** : accepter = prendre le videau côté de l'accepteur **et lancer l'horloge de l'adversaire** ; refuser = recentrer le videau et **pauser l'horloge**. Doubler = frapper l'horloge (règles unifiées 2001, qui disent aussi : « After accepting the cube, a player says "take" and hits his clock »).
- **Abandon** : « All games and matches must be played to completion unless brought to an end by the pass of a double or, if the match is played using a game clock, by a player running out of time » ; sans contact, on peut concéder une partie simple, un gammon ou un backgammon.
- **Coup anticipé** : si l'adversaire ramasse les dés avant que le joueur ait fini, ce dernier peut pauser, finir son coup, et l'adversaire **perd le délai** à son tour suivant et doit attendre qu'il expire.
- **Dépassement constaté** : le temps est « réputé écoulé » quand un joueur ou le directeur le constate et le déclare ; on n'applique pas la perte automatique à la milliseconde, mais à la constatation. Un logiciel arbitre peut la déclarer à l'instant exact.

Sources : <https://www.backgammon-deutschland.de/en/turnierregeln>, <https://usbgf.org/tournament-rules/rules-for-in-person-play/>, <https://bkgm.com/tournaments/UnifiedClockRules.html>, <https://bkgm.com/tournaments/US-Backgammon-Clock-2013.html>, <https://bkgm.com/articles/Woolsey/ClocksInBackgammon.html> (« punching the clock signifies that the move is complete »).

### D.3 Pénalités qui ne sont pas la perte au temps

- **Retard au départ ou aux pauses** : USBGF, un point de pénalité par tranche de 5 minutes de retard, forfait quand le total dépasse **la moitié de la longueur du match** ; pauses de 5 minutes entre parties, limitées à `⌊L/6⌋`. WBGF 2.1, article « PENALTIES » des heures de début et pauses : même barème, mais le joueur fautif est « réputé avoir perdu » quand les points dépassent la moitié du match.
- **Jeu lent** (WBGF 2.1 § 2.3) : avertissement, puis **imposition de l'horloge pour le reste du match** ; un joueur peut demander l'horloge ou un moniteur. L'horloge n'est donc obligatoire que sur annonce, par ordre du directeur, ou d'office dans un tournoi « à obligation ».
- **Horloge activée par erreur** (§ 4.3 (iv)) : devoir d'avertir, sinon temps rendu, avertissement, points de pénalité, perte de partie ou de match.

### D.4 Cadences nommées exploitables pour la tranche #577

Valeurs fondées sur les textes ci-dessus ; les noms sont les nôtres `[EXTRAPOLÉ]`.

| Nom | Réserve | Délai simple | Source de la valeur |
|---|---|---|---|
| **Tournoi** (défaut) | 2 min × (longueur moyenne restante), soit 2·L − (a + b) min, recalculée au score | 12 s | USBGF, WBGF 2.1 |
| **Tournoi doubles** | 2,5 min par point, moins 1,25 min par point marqué | 18 s | Règles américaines 2013 (doubles) ; hors périmètre d'un duel humain contre Bot |
| **Rapide « 3+12 »** | 3 min, fixe | 12 s | Préréglage « the Backgammon Galaxy control » d'une application tierce, défaut 3:00 + 12 s `[ÉDITEUR/DÉCLARÉ]` ; <https://source.hansdezwart.nl/hansdezwart/bgclock> |
| **Rapide « 2+12 » / « 3+15 »** | 2 min ou 3 min | 12 s ou 15 s | Même application tierce, trois préréglages : 2m/12s, 3m/12s, 3m/15s |
| **Sans horloge** | aucune | aucun | WBGF 2.1 § 3.5 (i) : l'horloge est option, préférence, obligation ou imposition |

Le seul chiffre mesuré sur l'usage réel : Bower et Heinz, de novembre 2002 à avril 2005, 179 matchs en 7 points à **12 s libres par coup plus 15 minutes de réserve** ; les joueurs ont consommé en moyenne **environ 9,7 s sur les 12 s**, les matchs ont duré de 9 à 92 minutes, deux forfaits seulement, 1 h 45 suffit pour plus de 98 % des matchs `[MESURE]`. Source : <https://www.bkgm.com/articles/Bower/ClockExperiment.html>. Woolsey cite « 12 seconds per move with a reserve bank of 22 minutes » pour un match en 11 points (soit 2 min par point) <https://bkgm.com/articles/Woolsey/ClocksInBackgammon.html>.

Recommandation d'implémentation (déduction) `[EXTRAPOLÉ]` :

1. Modèle : `réserve` (secondes) + `délai` (secondes, non cumulable). À chaque tour, `consommé = max(0, durée − délai)` retranché de la réserve ; le délai se **réarme** à chaque tour.
2. Le tour se termine à la **validation du coup** (équivalent de frapper l'horloge), jamais au lancer des dés.
3. Videau : l'horloge du joueur doublé tourne pendant sa décision ; refuser la stoppe et clôt la partie ; accepter transmet la main.
4. Dépassement : **perte du match** par défaut, avec une option « pénalité en points » (comme `Penalty` de XG) réservée aux modes d'entraînement. Cas des deux horloges à zéro sans objet contre un Bot.
5. Le Bot ne doit pas être soumis à la même horloge, ou alors avec un budget fixe indépendant de la réserve humaine (voir section C).

### D.5 Implémentations logicielles

- **XG** : horloge intégrée à trois modes (`tsNone`, `tsFischer`, `tsBronstein`) d'après la structure du format ; cadences par défaut et interface `[NON TROUVÉ]` (pas de manuel lisible en ligne). <https://www.extremegammon.com/XGformat.aspx>
- **GNU Backgammon** : aucune horloge, déjà acquis ; la propriété SGF `BL`/`WL` y est « not currently used ». <https://git.savannah.gnu.org/cgit/gnubg.git/plain/sgf.h>
- **Backgammon Galaxy** : « if you run out of time, you forfeit » (extrait de recherche d'un fil de 2+2, <https://forumserver.twoplustwo.com/138/backgammon-forum-hosted-bill-robertie/backgammon-galaxy-some-questions-1742777/>, **page inaccessible, citation non vérifiée**) ; cadence par défaut non publiée par l'éditeur `[NON TROUVÉ]`, seulement le préréglage 3+12 d'un outil tiers.
- **Backgammon Studio / Heroes, BGBlitz, bgammon.org** : cadence `[NON TROUVÉ]` dans cette passe. La fiche P9 cite 30 s + 30 s pour bgammon.org ; non revérifié, et la spécification du protocole (`PROTOCOL.md`) ne mentionne ni horloge ni délai.
- **UKBGF et EUBGF** : les pages « The Newcomer's Guide to… Clocks » et « About Time… » de l'UKBGF répondent 401, le règlement EUBGF en PDF n'est pas lisible par l'outil. Le texte du WBGF 2.1 est celui que reprend l'écosystème européen. `[NON TROUVÉ]` pour le texte propre de l'UKBGF.

---

## E. Le temps par décision comme donnée d'analyse

### E.1 Format `.xg` d'eXtreme Gammon

Spécification publique figée (<https://www.extremegammon.com/XGformat.aspx>). Constantes : `tsNone = 0`, `tsFischer = 1`, `tsBronstein = 2` ; `SaveFileVersion = 30` (« 28 is XG 2.00, 30 for 2.10 »). Déclarations exactes :

```pascal
TTimeSetting = record    // 32 bytes
  ClockType : integer;   // tsNone,tsFischer,tsBronstein
  PerGame   : Boolean;   // time is for session reset after each game
  Time1     : integer;   // initial time in sec
  Time2     : integer;   // time added (fisher) or reserved (Bronstein) per move in sec
  Penalty   : integer;   // point penalty when running out of time (in point)
  TimeLeft1 : integer;   // current time left
  TimeLeft2 : integer;   // current time left
  PenaltyMoney: integer; // point penalty when running out of time (in point)
end;
```

- `TSaveRec` (en-tête, 2560 octets) : `TimeSetting: TTimeSetting; // v25: Time setting for the session`.
- Enregistrement de **videau** (`tsCube`) : `TimeBot,TimeTop : integer; // v28: time left for both players`.
- Enregistrement de **coup** (`tsMove`) : **aucun champ de temps** de jeu. Les champs `TimeDelayMove*`/`TimeDelayCube*` (v26) concernent un **rollout différé** (« position is marked for later RO »), pas une durée.
- `TimeLimit` dans le contexte de rollout est une limite de calcul en minutes, sans rapport.

Conséquences :

1. **Le temps n'est pas une durée par décision mais un temps restant**, noté dans l'enregistrement de videau, c'est-à-dire une fois par tour si XG écrit un enregistrement de videau avant chaque coup `[EXTRAPOLÉ]` (192 à 337 enregistrements de videau pour 4 matchs de 7 points dans `testdata/`, ordre de grandeur d'un par tour). `TimeBot`/`TimeTop` sont en haut et en bas du plateau, pas « joueur 1/joueur 2 » (sémantique exacte non documentée).
2. **Durée de la décision** = temps restant avant moins après, corrigée de l'incrément Fischer ou de la restitution Bronstein : dérivable, pas stockée. Impossible de séparer un coup de 3 s d'un coup de 11 s sous 12 s de délai, puisque la réserve n'en garde pas la trace `[EXTRAPOLÉ]`.
3. **Mesure sur nos fichiers** : lecture avec `xgdatatools` (Python) de `testdata/*.xg` : les quatre fichiers sont en version 32, `ClockType = 0`, tous champs `TimeSetting` à zéro, `TimeBot = TimeTop = 0` sur tous les enregistrements de videau lus `[MESURE]`. Un fichier chronométré réel (XG contre XG avec horloge, ou export de site) n'a pas été trouvé : le remplissage effectif de `TimeBot`/`TimeTop` reste **non vérifié** `[NON TROUVÉ]`.
4. Les analyseurs relisent-ils ce champ ? Les parseurs `xgdatatools` et `xgparser` lisent `TimeSetting` en en-tête ; la restitution à la revue par XG lui-même `[NON TROUVÉ]`.

### E.2 Autres formats

| Format | Temps enregistré ? | Source |
|---|---|---|
| `.mat` (Jellyfish) | **Non** : aucun champ, texte de coups et de scores | `[EXTRAPOLÉ]` d'après la pratique ; format de fait sans spécification officielle |
| SGF de gnubg | Propriétés générales `BL` et `WL` (temps restant noir/blanc) **déclarées mais « not currently used »** | <https://git.savannah.gnu.org/cgit/gnubg.git/plain/sgf.h> (copie locale `~/src/gnubg/sgf.h`) |
| Replay bgammon.org (`.match`) | **Non** : une ligne de métadonnées avec un horodatage de **début de partie**, puis les événements (`1 r 5-3 13/8 24/21`, `1 d 2 1`) sans temps | <https://raw.githubusercontent.com/tslocum/bgammon/main/REPLAY.md> `[MESURE]` (lu) |
| `.xg` | Oui, voir E.1 | ci-dessus |
| Exports Galaxy, Heroes, BGBlitz `.bgf` | `[NON TROUVÉ]` : aucun champ de temps documenté ; Galaxy et Studio passent par `.xg` (P9) | |

Il n'existe donc aucune convention à respecter pour le temps de décision : blunderDB pourra enregistrer le sien sans conflit (champ propre à la base, jamais réexporté dans un format qui n'en a pas, pour respecter l'allow-list de métadonnées exportées).

### E.3 Temps de réflexion et erreur : littérature

- **Aux échecs, le temps restant prédit la bévue** : Anderson, Kleinberg et Mullainathan, sur des finales à 7 pièces ou moins évaluées par tables de finales : le taux de bévue « increases sharply as t → 0, but flattens out for t > 10 » secondes `[MESURE]` ; la **difficulté** de la position (β) explique davantage que le niveau ou le temps : exactitude de 73 % pour la difficulté contre 55 % (niveau) et 53 % (temps restant) ; un écart de 0,2 de β pèse autant que 600 Elo. Résumé : <https://reasonabledeviations.com/notes/papers/chess_blunders> (le papier original n'a pas été lu directement).
- **Pic de bévues au seuil d'alerte de Lichess** : 68 millions de parties d'un mois, pics à 20 s (1 min), 40 s (5 min), 60 s (10 min) ; l'auteur note que c'est une corrélation, sans contrôle causal `[MESURE]` non revue par des pairs. <https://lichess.org/forum/redirect/post/1SL2qHTI>
- **Piège de la corrélation** : un billet de Lichess montre que plus de temps passé va avec **moins** de précision, parce qu'on réfléchit plus dans les positions difficiles. <https://lichess.org/@/pawnzac/blog/correlation-causation-and-move-time/Dc6cffdW> `[ÉDITEUR/DÉCLARÉ]` (analyse d'un joueur).
- **Au backgammon** : aucune étude trouvée sur temps de réflexion et erreur `[NON TROUVÉ]`. La littérature d'Anderson donne la méthode transposable : mesurer l'erreur **à difficulté contrôlée** (ici, nombre de coups légaux, écart d'équité entre les deux meilleurs, type de décision) et le temps **restant** séparément du temps **passé**.
- **Métriques à retenir** `[EXTRAPOLÉ]` : temps passé par type de décision (ouverture, coup forcé, videau, prise) ; temps passé contre difficulté (équité perdue du deuxième meilleur, nombre de coups légaux) ; erreurs sous pression (temps restant sous un seuil, par exemple moins de 20 % de la réserve) ; exclusion des **coups forcés** (un seul coup légal) du calcul des moyennes.

### E.4 Affichage à la revue

Le « Better game clock history » de Lichess est référencé (<https://lichess.org/blog/WOEVrjAAALNI-fWS/a-better-game-clock-history>) mais n'a pas pu être lu (erreur réseau). Affichages de Chess.com, XG et Galaxy : `[NON TROUVÉ]`. Retour d'usage : aucun trouvé.

---

## A. Affaiblir un bot de façon crédible (passe courte)

- **Maia (échecs)** : réseau entraîné sur des parties humaines, neuf modèles de 1100 à 1900 Elo, pas de recherche arborescente ; il prédit le coup humain dans plus de 50 % des cas (46 à 52 % contre 33 à 46 % pour les bases), chaque modèle culmine à son niveau cible, et il prédit les bévues à 71,7 % (individuelles) et 76,9 % (collectives sur une même position) `[MESURE]`. Il **n'affaiblit pas** un moteur fort : il en réentraîne un sur des données humaines. Source : <https://alphaxiv.org/abs/2006.01855>. Transposition au backgammon : exigerait un corpus de matchs humains annotés et un réseau de politique ; **aucune tentative trouvée** `[NON TROUVÉ]`.
- **Autres moteurs** (XG, BGBlitz, Snowie, Galaxy, Ace, Sage, bgammon.org, wildbg), `UCI_Elo` de Stockfish, niveaux de KataGo, formule FIBS, tables van den Doel, volume de matchs pour valider un niveau, déterminisme : **non traités dans cette passe**, `[NON TROUVÉ]`.

## B. Politique de match au-delà de l'évaluateur

Abandon, videau, beaver, raccoon, automatismes de confort : **non traités**, `[NON TROUVÉ]`. Seule donnée recueillie : le WBGF 2.1 § 4.5 autorise la concession d'une partie simple, d'un gammon ou d'un backgammon **seulement s'il n'y a plus de contact** ou si aucun autre résultat n'est possible ; hors cela, « the players are not permitted to agree on the outcome of points, of a game or a match ». Pour un arbitre logiciel, la concession n'est donc licite qu'à contact rompu, et le refus de videau reste libre. <https://www.backgammon-deutschland.de/en/turnierregeln>

## C. Temps de réflexion du bot

Allocation du temps par difficulté, pondering, délai artificiel perçu comme naturel : **non traités**, `[NON TROUVÉ]`. Observation utile tirée de la section D : un humain consomme en moyenne 9,7 s sur un délai de 12 s (Bower et Heinz), ordre de grandeur d'un délai de réponse artificiel naturel `[EXTRAPOLÉ]`.

## F. Des dés auxquels on croit

- **Backgammon.com** met en avant, pour la confiance dans ses dés, un générateur cryptographique côté serveur, une certification d'un laboratoire de jeu (juillet 2026) et une attestation du WBGF (août 2026) que l'algorithme est « sufficiently strong » ; **ni graine publiée, ni statistiques** ; l'argument est l'audit externe plus l'analyse de chaque match `[ÉDITEUR/DÉCLARÉ]`. <https://backgammon.com/fair-dice>
- **Commit-reveal** : le schéma standard hache la graine du serveur avant le match, puis la révèle ; les résultats se dérivent par HMAC-SHA256 de la graine serveur, d'une graine client et d'un compteur. Description générale d'un schéma à deux entrées : <https://dev.to/gigavariance/provably-fair-gaming-commit-reveal-schemes-explained-f53> (blog, source secondaire).
- Utilisation réelle dans le backgammon en ligne, schémas à deux parties distantes, coûts, implémentations dans des jeux de cartes : **non traités**, `[NON TROUVÉ]`.

## G. Expérience de jeu et API

- **Lichess Board API** : flux d'événements `gameFull` puis `gameState`, horloge exposée par `wtime`, `btime`, `winc`, `binc` (temps restants et incréments), reconnexion et idempotence du coup annoncées ; résumé produit par l'outil de lecture, **la page de référence n'a pas été relue en entier**. <https://lichess.org/api#tag/Board> `[ÉDITEUR/DÉCLARÉ]`
- **Protocole bgammon.org** : TCP `bgammon.org:1337` ou WebSocket `wss://ws.bgammon.org`, réponses JSON via `loginjson`/`registerjson`, `replay <id>` et `history <username> [page]` ; lu directement. <https://raw.githubusercontent.com/tslocum/bgammon/main/PROTOCOL.md> `[MESURE]`
- Options de lancement, mode tuteur, annulation de coup, enchaînement vers la revue dans XG, BGBlitz, Ace, Galaxy ; FIBS CLIP : **non traités**, `[NON TROUVÉ]`.

---

## Questions restées sans réponse fiable

1. **Réserve reposée à chaque partie ou fixée pour le match ?** Aucun texte lu ne le dit ; XG a un drapeau `PerGame` pour les deux cas.
2. **Remplissage réel de `TimeBot`/`TimeTop`** dans un `.xg` chronométré : aucun fichier chronométré disponible pour le vérifier ; la sémantique « haut/bas » reste à confirmer.
3. **Cadence par défaut et sanction exactes de Galaxy, Heroes/Studio, BGBlitz, XG** : aucune source primaire lue ; seul un préréglage 3+12 d'une application tierce et un extrait de forum non vérifié.
4. **Textes UKBGF et EUBGF propres** : inaccessibles à l'outil (401, PDF opaque).
5. **Temps de réflexion et erreur au backgammon** : aucune étude.
6. **Sections A, B, C, G (hors Lichess et bgammon.org), F (hors fiche backgammon.com)** : à refaire si la tranche concernée les exige.
