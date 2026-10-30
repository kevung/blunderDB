Recherche du 2026-10-30 — claude.ai, recherche approfondie.
Prompt : ../prompts/08-ux-humain-dans-la-boucle.md

---

# Assistant visuel « humain dans la boucle » pour la transcription blunderDB : revue de littérature UX (2015–2026)

L'assistant visuel peut faire gagner du temps, à une condition : chaque proposition doit se valider implicitement par l'action suivante, avec un « annuler » toujours disponible, et non par une confirmation explicite ou un compte à rebours. Il doit aussi s'abstenir (pause) dès que sa confiance est basse. La littérature est convergente sur ce point : la vérification humaine a un coût fixe que seul un taux d'erreur faible amortit. Ce seuil se situe vers 70 % de fiabilité pour une aide diagnostique (Wickens & Dixon, 2007)\[1\] et vers 30 % de taux d'erreur mot pour la post-édition ASR : selon Gaur, Lasecki, Metze & Bigham (W4A 2016, 160 participants), « By the time the ASR transcripts exceed 30% WER, workers are again faster just typing the content from scratch ». Les fausses validations font plus de tort à la confiance que les abstentions (Dixon, Wickens & McCarley, 2007).\[2\]

## TL;DR

- **Oui, mais en mode « proposer + valider par l'action suivante + annuler », pas en mode « confirmer chaque proposition » ni « valider après N secondes ».** Les confirmations répétées s'habituent très vite : dans Bravo-Lillo et al. (SOUPS 2014, 2 567 participants, condition contrôle), 50 participants sur 106 ont suivi la consigne modifiée après une exposition, contre 24 sur 114 après 20 expositions, soit une baisse des chances « by more than a factor of three ». L'undo est la parade documentée (Raskin, 2007 ; Gmail Undo Send, 5 s par défaut).
- **Régler l'automatique sur la précision, pas sur le rappel.** Il vaut mieux s'abstenir (mettre en pause, proposer 2–3 lectures sur les touches 1/2/3) que valider à tort. Les automatismes sujets aux fausses alarmes dégradent davantage la performance et la confiance que ceux sujets aux omissions (Dixon et al., 2007). La régression de Wickens & Dixon (2007), telle que résumée dans PMC4221095, situe le point de croisement à 70 % : « when using automation less than 70% reliable, task performance was worse than if the person were doing the task manually ». Il y a toutefois des contradictions, détaillées plus bas.
- **Mesurer avant de croire.** Le protocole proposé combine une prédiction KLM (Card, Moran & Newell, 1980) et un plan intra-sujet alterné (manuel / assisté / automatique) sur des segments de matchs appariés. Il relève le temps par tour, les gestes par tour, les corrections, les erreurs résiduelles (via erreurs injectées) et les abandons du mode automatique. L'assistant n'est retenu que s'il réduit le temps par tour sans augmenter les erreurs résiduelles.

## Key Findings — les cinq mécanismes à retenir

| # | Mécanisme | Ce qu'il fait dans blunderDB | Source principale qui le justifie |
|---|---|---|---|
| 1 | **Proposition pré-remplie + validation implicite par l'action suivante + undo** | Lecture des dés et coup proposé affichés dans l'indicateur permanent ; la saisie du lancer suivant (ou Entrée) valide ; Ctrl+Z / Retour arrière défait le dernier tour validé | Raskin, « Never Use a Warning When you Mean Undo » (A List Apart, 2007) ; Bravo-Lillo et al., SOUPS 2014 (habituation) ; Amershi et al., CHI 2019 (G8 « support efficient dismissal », G9 « support efficient correction ») |
| 2 | **Abstention calibrée sur la précision** (seuil d'auto-validation élevé, sinon pause) | L'automatique ne valide seul que si la lecture est très sûre ; sinon il gèle la vidéo et attend | Dixon, Wickens & McCarley, Human Factors 2007 (fausses alarmes > omissions) ; Wickens & Dixon 2007 (seuil ~0,70) ; Horvitz, CHI 1999 (agir / dialoguer / ne rien faire selon l'utilité espérée)\[3\] |
| 3 | **Confiance catégorielle (3 niveaux) + N-best, jamais de pourcentage brut** | Indicateur sûr / à vérifier / incertain (couleur + forme + texte) ; en cas de doute, 2–3 lectures concurrentes sur touches 1/2/3 | Google PAIR, People + AI Guidebook (2019) : seaux de confiance et « N-best alternatives » ; Zhang, Liao & Bellamy, FAT* 2020 (la confiance aide la calibration, l'explication locale non) |
| 4 | **Surimpression de la lecture sur l'image + image gelée + zoom sur la zone douteuse à la reprise** | Cadre autour des dés détectés avec la valeur lue ; en pause, image figée et vignette agrandie des dés | Altmann & Trafton, 2002/2004 (memory for goals, résomption facilitée par des indices) ; Trafton, Altmann & Brock, 2005 (indice « flagrant » plus efficace) |
| 5 | **Ligne de temps à marqueurs, saut événement par événement au clavier, marqueurs incertains distincts** | Tab / Maj+Tab (ou touches dédiées) pour aller au lancer suivant / précédent ; filtre « incertains seulement » | Hudl Sportscode (Tab / Shift+Tab = instance suivante / précédente) ; CVAT (R / E = keyframe suivante / précédente, K = keyframe) |

**Protocole de mesure** (détaillé en section 7) :
- calcul KLM préalable ;
- plan intra-sujet à alternance de conditions (A = manuel, B = assisté, C = automatique) sur segments appariés ;
- erreurs injectées pour mesurer la complaisance ;
- NASA-TLX brut et SUS en fin de bloc ;
- critère de décision chiffré fixé à l'avance.

## Details

### 1. Patterns documentés d'annotation semi-automatique

**Labellisation (CVAT, Label Studio)** — le mieux documenté.
- **CVAT, 2019–2025.**
  - Son vocabulaire officiel distingue trois types d'annotation : *manuelle* ; *semi-automatique* (« created mainly automatically, but the user provides some data (e.g. interpolation) ») ; *automatique* (« without a person in the loop »).\[4\]
  - En mode Track, l'utilisateur pose des keyframes (touche K) et l'outil interpole linéairement entre elles. La touche O marque l'objet « outside ». R / E sautent à la keyframe suivante / précédente. Espace lance le défilement automatique des images.\[5\]
  - **Proposition** : la forme interpolée apparaît sur les images intermédiaires.
  - **Correction** : déplacer la forme crée une nouvelle keyframe.
  - **Refus** : supprimer, ou séparer / fusionner des pistes (M pour Merge, fonction Split).
  - **Retour en arrière** : Ctrl+Z standard.
  - CVAT a intégré en 2024–2025 un « SAM 2 Tracker » qui propage un masque sur N images choisies par l'utilisateur.\[6\] Sa propre documentation (CVAT Academy) reconnaît le « besoin de validation » : « annotators must manually review and adjust annotations, which can take additional time ».\[7\]
- **Label Studio (v1.3.0, 2021 ; v1.12.0, 2024).** Les pré-annotations interactives renvoyées par un backend ML peuvent être en « Auto accept annotation suggestions ». Sinon, « the regions appear, but you can reject or approve them manually, either individually or all at once ».\[8\] Ce choix explicite entre acceptation automatique et acceptation manuelle correspond exactement aux deux niveaux souhaités (automatique / assisté). La v1.12.0 corrige d'ailleurs un bug où ce bouton « was not working as expected ».\[9\]
- **Supervisely** : non trouvé (pas de documentation primaire consultée).

**Annotation sportive (Hudl Sportscode).**
- **Saisie** : en codage live, une touche-bouton crée une « instance ». Des durées *lead / lag* (pré-roll / post-roll) étendent automatiquement l'instance avant / après l'appui. Dans Sportscode Replay, une instance codée dure par défaut 20 s (10 s lead + 10 s lag).\[10\]\[11\]
- **Raccourcis (fiche officielle Sportscode 11, 2017)** :\[12\]
  - Tab / Shift+Tab : instance suivante / précédente ;\[12\]
  - Ctrl+E : éditer une instance ;
  - Ctrl+I / Ctrl+O : fixer début / fin ;
  - Échap : annuler les codes actifs ;
  - ⌘Z : annuler ;
  - Ctrl+= : note live.
- **Évolutions** : les notes de version 12.2.x ajoutent des touches d'ajustement d'instance. La 12.83.1 (16 juillet 2026) corrige un ralentissement lors de l'import de données XML live, c'est-à-dire de données d'événements externes fusionnées au codage manuel.\[13\]
- **Ce qu'il en ressort** : le pattern dominant est « l'humain tape vite, la machine élargit la fenêtre, on corrige après coup ». Il n'y a pas de proposition machine à confirmer.
- **Non trouvé** : Nacsport, Dartfish, Metrica, LongoMatch, procédure de collecte StatsBomb/Opta (pas de source primaire consultée).

**Retransmission d'échecs (DGT LiveChess).**
- Le manuel LiveChess 2 (rev. 2018) précise trois points :\[14\]
  - « Illegal moves will not be accepted by LiveChess and not be registered in the game data » ;\[14\]
  - « Question marks indicate incorrect or illegal moves » ;\[14\]
  - l'opérateur peut faire une « manual move correction » pour reconstruire une partie « when moves are lost by player errors, detection errors or network failures ».\[14\]
- **Ce qu'il en ressort** : la légalité filtre la reconnaissance. C'est directement transposable, puisque blunderDB connaît déjà les coups légaux pour un lancer donné. L'anomalie est signalée (« ? »), et la correction se fait en différé par un opérateur.
- **Limites observées** :
  - un fil du forum de l'English Chess Federation (2022, source forum) rapporte qu'un roque illégal en Olympiade n'a pas été relayé alors qu'une fenêtre d'événements aurait dû l'afficher : l'alerte existait, personne ne la surveillait ;\[15\]
  - l'issue #67 du dépôt lichess-org/broadcaster (août 2025) note qu'un PGN contenant un coup illégal bloque toute mise à jour de la partie.\[16\]

**Sous-titrage assisté (ASR).**
- Les données quantitatives sont surtout académiques (section 2).
- Sperber et al. (LREC 2016) comparent saisie « from scratch » et post-édition : la post-édition a un « inherent overhead » de vérification et de navigation entre erreurs, et la saisie directe redevient plus rapide quand l'effort de frappe de correction est élevé.\[17\]
- Le même groupe (« Transcribing Against Time », arXiv 1709.05227, 2017) choisit d'afficher l'hypothèse ASR à côté d'une saisie depuis zéro plutôt que de la pré-remplir.\[18\] C'est une forme d'assistance « non intrusive ».
- **Non trouvé** : documentation primaire Subtitle Edit, Descript, éditeur de sous-titres YouTube, aeneas.

### 2. Le coût de la vérification : où l'assistance devient plus lente que le manuel

- **Seuil de fiabilité ~70 %.**
  - Wickens & Dixon (Theoretical Issues in Ergonomics Science, 2007, DOI 10.1080/14639220500370105) ont fait une méta-régression sur 20 études et 35 points de données. Résultat : « a reliability of 0.70 was the ‘crossover point’ below which unreliable automation was worse than no automation at all ».\[1\]\[19\]
  - **Contradiction** : Wickens, Dixon & Johnson (HFES 2006) trouvent que « reliability as low as 0.6 aided human performance ». Le même résumé ajoute cependant que « directing attention to false-alarm prone automation actually degrades human-system performance », ce qui plaide pour un réglage orienté précision. Une autre étude citée sur ResearchGate rapporte des bénéfices même à 30 %. Le seuil dépend donc de la tâche et de la charge.
- **Seuil ASR ~30 % WER.**
  - Gaur, Lasecki, Metze & Bigham (W4A 2016, 160 participants MTurk, 16 extraits TED-LIUM, WER de 15 à 55 %) : « By the time the ASR transcripts exceed 30% WER, workers are again faster just typing the content from scratch ».
  - Point crucial pour blunderDB : Gaur et al. (2016) observent eux-mêmes que « If the WER is high enough (45% in our study), then workers identify how bad it is, erase the text generated from the ASR, and simply type from scratch », et la latence culmine « at approximately 45% WER ». **Les utilisateurs perçoivent la dégradation bien après qu'elle les a ralentis.**
  - Une analyse de 2018 sur le Parlement islandais (arXiv 1807.11893 ; Fong, Borsky, Helgadóttir & Gudnason, 239 discours de l'Althingi) reprend un seuil de 25 % issu de travaux antérieurs sur la transcription de cours, et conclut pour sa part qu'« a system achieving a 12.6% edit distance would match the performance of manual transcribers ».
- **Fausses alarmes vs détections manquées.**
  - Dixon, Wickens & McCarley (Human Factors 49(4), 2007, DOI 10.1518/001872007X215656, 32 étudiants) : « False alarm-prone automation hurt overall performance more than miss-prone automation », et affecte à la fois la *compliance* et la *reliance*, alors que les omissions n'affectent que la *reliance*.\[2\]\[20\]
  - **Contradiction** : Kocielnik, Amershi & Bennett (CHI 2019, DOI 10.1145/3290605.3300641) trouvent, pour un assistant de détection de demandes de réunion dans les e-mails, qu'une version orientée *rappel* (plus de faux positifs) était perçue comme plus précise et mieux acceptée qu'une version orientée *précision*.\[21\]\[22\] Explication probable : le coût relatif d'une omission y était plus élevé.
  - **Transposition à blunderDB.** Une *fausse validation* automatique (mauvais coup enregistré sans arrêt) est l'analogue d'une fausse alarme non vue, et son coût est élevé : elle corrompt la base. Une *abstention* (pause inutile) coûte une touche. On penche donc vers la précision.
- **Complaisance et biais d'automatisation.**
  - Parasuraman & Manzey (Human Factors 52(3), 2010, DOI 10.1177/0018720810376055) : la complaisance s'observe « in both naive and expert participants and cannot be overcome with simple practice ». Le biais produit des erreurs d'omission et de commission.\[23\]
  - Parasuraman & Riley (Human Factors 39(2), 1997) posent la typologie use / misuse / disuse / abuse.\[24\]
  - Lee & See (Human Factors 46(1), 2004) définissent la confiance « calibrée » : elle doit correspondre à la fiabilité réelle.\[25\]
- **Pré-annotation et ancrage.**
  - Fort & Sagot (Linguistic Annotation Workshop, 2010) : la pré-annotation accélère l'étiquetage POS mais biaise les annotateurs vers les erreurs de l'outil.\[26\]\[27\]
  - Berzak et al. (arXiv 1605.04481, 2016) montrent un « anchoring bias effect » vers la sortie du parseur.\[28\]
  - **Contradiction** : Schulz et al. (ACL 2019) trouvent que « even mediocre suggestion models have a positive effect … while annotation biases are negligible ».\[29\]
  - arXiv 2306.09307 (2023) : la saisie depuis zéro prend « almost 1.7 times longer » que la correction de pré-annotation, à qualité égale.\[30\]
  - « Bias in the Loop » (arXiv 2509.08514, 2025) : quand corriger demande plus d'effort, les annotateurs corrigent moins. Les annotateurs sceptiques envers l'IA sont plus exacts.\[31\] **Rendre la correction aussi bon marché que l'acceptation est donc une exigence, pas un confort.**

### 3. Affichage de la confiance

- **Google PAIR, People + AI Guidebook (2019 ; v2 en ligne).**
  - Contre les pourcentages : « showing more granular confidence can be confusing … what should I do when the system is 85.8% certain vs. 87% certain? » et « Numeric confidence indicators are risky ».\[32\]
  - Pour les catégories : « categorize confidence values into buckets, such as High / Medium / Low … Clearly indicate what action a user should take under each category ».\[32\]
  - Pour le N-best : « the system can display the N-best alternative results … especially useful in low-confidence situations. Showing multiple options prompts the user to rely on their own judgement ».\[32\]
  - Avertissement : « A misleadingly high confidence … may cause users to blindly accept a result ».\[32\]
- **Amershi et al. (CHI 2019, DOI 10.1145/3290605.3300233)** : 18 lignes directrices validées auprès de 49 designers sur 20 produits.\[33\] Les plus pertinentes ici :
  - G2 « make clear how well the system can do what it can do » ;
  - G10 « scope services when in doubt » ;
  - G11 « make clear why the system did what it did ».
- **Zhang, Liao & Bellamy (FAT* 2020, arXiv 2001.02114)** : afficher un score de confiance améliore la calibration de la confiance, mais pas la précision conjointe. L'explication locale n'a eu aucun effet.\[34\]\[35\] **Une explication textuelle « j'ai lu 6-3 car… » n'a donc pas de soutien empirique direct.** La surimpression du cadre de détection sur l'image est préférable, car c'est une *preuve vérifiable* plutôt qu'une justification.
- **Buçinca, Malaya & Gajos (CSCW 2021, DOI 10.1145/3449287)** : ajouter des explications ne réduit pas la sur-confiance. Les « cognitive forcing functions » la réduisent, mais les participants les ont jugées le moins favorablement.\[36\]\[37\]\[38\] Pour blunderDB, ce coût n'est acceptable que pour les cas incertains, pas à chaque tour.
- **Délai proportionnel à la confiance** : non trouvé comme pattern évalué.
- **Visualisation d'incertitude générale (Hullman et al., etc.)** : non consultée dans cette revue.

### 4. Validation par délai (« accepter si silence N secondes »)

- **Précédents à N fixe.**
  - Gmail Undo Send : 5 s par défaut, réglable sur 5, 10, 20 ou 30 s (Google Help ; blog Google, 7 février 2023).\[39\] Une page SEO évoque une option à 60 s en 2026. Elle se contredit elle-même\[40\] et n'est confirmée par aucune page Google : la valeur à retenir reste 30 s maximum.
  - La checklist Google Cast « Autoplay » demande un compte à rebours et des actions « Play » / « Stop » explicites. L'annonce « Up Next » doit arriver au moins 30 s avant la fin.\[41\]
  - Durée du compte à rebours YouTube : non confirmée par une source primaire (8 s ou 15 s selon des sources tierces contradictoires).\[42\]\[43\]
- **Risques documentés.**
  - Aza Raskin (A List Apart, juillet 2007) : « after clicking ‘Okay’ countless times … we'll probably click ‘Okay’ this time too » et « Never use a warning when you mean undo ».\[44\]
  - Bravo-Lillo, Cranor, Komanduri, Schechter & Sleeper (SOUPS 2014, 2 567 participants, tableau 1, condition contrôle) : 50 participants sur 106 ont suivi la consigne modifiée après une exposition, contre 24 sur 114 après 20 expositions, soit une baisse des chances « by more than a factor of three ». Le temps de réponse médian passe de 10 s à 1,2 s.
  - Vance et al. (CHI 2017, DOI 10.1145/3025453.3025896) : chute « dramatic » de l'attention neuronale dès la *deuxième* exposition. L'habituation est « an obligatory neurobiological response, rather than willful carelessness ».\[45\]
  - Warm, Parasuraman & Matthews (Human Factors 50(3), 2008) : la vigilance « requires hard mental work and is stressful ».\[24\] Regarder un compte à rebours défiler à chaque tour est une tâche de vigilance.
- **Conséquence.** Un délai visible à chaque tour devient vite invisible : l'utilisateur le laisse expirer par réflexe, ce qui revient à valider sans regarder. La validation par l'action suivante est meilleure, car elle est *gratuite* : le lancer suivant doit être saisi de toute façon. Elle se combine avec un undo multi-niveaux et une revue a posteriori des tours « à vérifier ».
- **Non trouvé** : étude publiée comparant directement délai d'auto-acceptation et validation implicite dans un outil de transcription ou de tagging.

### 5. Pause sur incertitude et reprise

- **Memory for goals** (Altmann & Trafton, Cognitive Science 26, 2002) : les buts suspendus décroissent avec la durée de l'interruption ; la reprise dépend d'indices environnementaux associés.\[46\]
- **Résomption** (Altmann & Trafton, CogSci 2004) : la première action après interruption prend plus de temps, avec un « resumption lag » de 3,8 s, soit le double du temps de base dans leur tâche. Les indices disponibles avant l'interruption réduisent ce délai.\[47\]
- **Indices** (Trafton, Altmann & Brock, HFES 2005) : un indice « blatant » (flèche rouge sur la dernière action) fait reprendre plus vite qu'un indice subtil ou qu'aucun indice.\[48\]
- **Mixed-initiative** (Horvitz, CHI 1999, DOI 10.1145/302979.303030) : agir seul seulement si l'utilité espérée dépasse celle de l'inaction. Entre les deux seuils, « employ dialog to resolve key uncertainties ».\[3\]\[49\]
- **Ce qui en découle pour la pause** :
  - geler l'image exacte du lancer ;
  - zoomer sur les dés ;
  - mettre en évidence (indice flagrant) la zone douteuse ;
  - proposer 2–3 lectures N-best sur 1/2/3 et garder la saisie libre au clavier comme chemin de secours.
- **Active learning UI** : non trouvé de source primaire spécifique consultée.

### 6. Saut d'événement en événement

- Les deux sources primaires vérifiées convergent sur une paire suivant / précédent à une touche :
  - Sportscode : Tab / Shift+Tab pour l'instance suivante / précédente ;\[50\]
  - CVAT : R / E pour la keyframe suivante / précédente, K pour une keyframe, Espace pour la lecture.\[5\]
- Raccourcis J/K/L et marqueurs de Premiere, DaVinci Resolve et Final Cut : non vérifiés dans cette revue (budget de recherche épuisé). La convention J/K/L (arrière / pause / avant) est largement répandue, mais nous ne la citons pas avec source primaire. ELAN / ANVIL : non trouvé.
- **Marquer les incertains** :
  - aucune source primaire évaluée trouvée ;
  - la recommandation PAIR (catégories + action associée) s'applique ;
  - des marqueurs à forme différente (pas seulement à couleur différente, par accessibilité) permettent un filtre « incertains seulement ».

### 7. Mesures de réussite et protocole léger

- **KLM** (Card, Moran & Newell, CACM 23(7), 1980, DOI 10.1145/358886.358895) :
  - opérateurs K ≈ 0,28 s (frappe d'un utilisateur moyen), M ≈ 1,35 s (préparation mentale), H ≈ 0,4 s (main clavier ↔ souris) ;\[51\]\[52\]\[53\]
  - erreur RMS du modèle d'environ 21 % dans l'article original.\[53\]
- **Prédiction pour un tour, hors recherche dans la vidéo** :
  - *manuel* ≈ M + 2K (dés) + M + ~3K (choix dans la liste) + K ≈ 4,4 s ;
  - *assisté* ≈ M (vérifier) + K ≈ 1,6 s ;
  - *automatique avec validation implicite* ≈ 0 geste supplémentaire, mais un M de surveillance si l'utilisateur regarde.
  - Le gain théorique est d'environ 2,8 s par tour. Il s'annule si l'utilisateur doit réécouter ou revérifier chaque tour : un retour en arrière de quelques secondes de vidéo plus M dépasse ce gain. C'est l'analogue blunderDB du seuil de 30 % WER.
- **Métriques à relever automatiquement** (journal des frappes horodaté, local) :
  - temps par tour ;
  - gestes par tour ;
  - corrections par match ;
  - undo par match ;
  - pauses par match ;
  - abandons du mode automatique (bascule vers assisté / manuel) ;
  - **erreurs résiduelles**, mesurées par comparaison avec une transcription de référence et par **erreurs injectées** : 5 % de propositions volontairement fausses, pour mesurer la complaisance comme dans les études de biais d'automatisation.
- **Plan** : intra-sujet à alternance de conditions (ABCACB…) sur des segments de 10 tours tirés de matchs comparables, sur soi-même puis sur 3–5 utilisateurs. NASA-TLX brut et SUS en fin de bloc. Ces deux instruments sont classiques (Hart & Staveland, 1988 ; Brooke, 1996) mais n'ont pas été revérifiés en source primaire dans cette revue.
- **Critère fixé à l'avance** : l'assisté est retenu si le temps médian par tour baisse d'au moins 30 % et si les erreurs résiduelles ne dépassent pas celles du manuel. L'automatique est retenu si, en plus, au moins 80 % des erreurs injectées sont rattrapées.

## Verdict

L'assistant visuel est souhaitable et compatible avec les principes du panneau, à trois conditions :
1. il ne fait jamais disparaître une frappe ni ne choisit sans trace : validation implicite, undo, journal ;
2. son mode automatique est réglé pour s'abstenir plutôt que se tromper ;
3. son gain est mesuré, pas supposé.

La validation par délai visible à chaque tour est à écarter comme mécanisme principal. La littérature sur l'habituation (Raskin 2007 ; Bravo-Lillo 2014 ; Vance 2017) prédit qu'elle dégénère en validation réflexe.\[44\]\[45\]\[54\] Si on la conserve, ce doit être uniquement comme filet en lecture continue, avec un N fixe court (≈ 2–3 s, à calibrer), jamais proportionnel à la confiance (aucun appui empirique trouvé).

## Recommandation technique (blunderDB : clavier d'abord, webview Wails, pas de dépendance native)

1. **Pipeline dans la webview uniquement.**
   - Utiliser `<video>` + `HTMLVideoElement.requestVideoFrameCallback()` pour échantillonner les images, puis les dessiner dans un canvas (OffscreenCanvas / Worker si disponible) et lancer l'inférence en JS/WASM, sans CGO.
   - rVFC est disponible dans Chrome 83+ (donc WebView2), Safari 15.4+ (mars 2022, donc WKWebView) et Firefox 132+. Il est « Baseline newly available » depuis le 29 octobre 2024.\[55\]\[56\]
   - Pour WebKitGTK, le support GStreamer a été intégré (bug WebKit 233541, résolu fin 2021), mais caniuse ne suit pas WebKitGTK (issue Fyrd/caniuse #6807).\[57\]\[58\] Il faut donc une détection de fonctionnalité et un repli sur `requestAnimationFrame` + `currentTime`.
2. **Filtre de légalité d'abord.** Comme DGT LiveChess qui refuse les coups illégaux :
   - ne proposer que des lectures de dés compatibles avec un coup légal observé ;
   - reconnaître le coup joué *parmi les coups légaux* déjà listés par le moteur. Cela réduit le problème de vision à une classification parmi quelques candidats.
3. **Trois états, une seule touche par état.**
   - *Sûr* (vert + coche) : pré-rempli. En assisté, Entrée valide. En automatique, la saisie ou la détection du lancer suivant valide implicitement.
   - *À vérifier* (orange + losange) : pré-rempli, jamais auto-validé.
   - *Incertain* (rouge + point d'interrogation) : pause, image gelée, zoom sur les dés, N-best sur 1/2/3, saisie libre toujours possible.
   - L'indicateur permanent affiche toujours l'effet de la prochaine touche, par exemple « Entrée = valider 6-3 · 24/18 13/10 ».
4. **Undo multi-niveaux** (Ctrl+Z / Retour arrière) qui revient au tour précédent et repositionne la vidéo. Pas de boîte de confirmation.
5. **Ligne de temps à marqueurs.** Tab / Maj+Tab pour le lancer suivant / précédent, une touche pour « incertain suivant », marqueurs de forme distincte. Revue de fin de match des seuls tours *à vérifier*.
6. **Seuils par vidéo.** Calibrer le seuil « sûr » sur les 10 premiers tours de chaque vidéo, en assisté obligatoire, puisque angle, lumière et matériel varient. N'autoriser l'automatique que si la précision observée sur ces tours atteint un niveau fixé (≥ 95 % proposé, à valider par l'expérience).
7. **Journalisation locale** de chaque proposition, de sa catégorie et de la décision utilisateur, pour alimenter le protocole de mesure.

## Incertitudes (et l'expérience à mener)

- **Seuil réel de rentabilité pour blunderDB.** Les seuils publiés (0,70 ; 30 % WER ; 25 % WER) viennent d'autres tâches et se contredisent partiellement. *Expérience* : plan alterné A/B/C décrit en section 7, en faisant varier artificiellement la fiabilité des propositions (100 %, 95 %, 85 %, 70 %) pour tracer la courbe temps par tour × fiabilité et trouver le point de croisement avec le manuel.
- **Fausses validations vs abstentions.** Dixon et al. (2007) et Kocielnik et al. (2019) divergent. *Expérience* : comparer deux réglages de seuil, l'un orienté précision (plus de pauses), l'autre orienté rappel (plus d'auto-validations, quelques erreurs), et mesurer confiance déclarée, abandons de l'automatique et erreurs résiduelles.
- **Validation implicite vs délai.** Aucune étude directe trouvée. *Expérience* : sur soi-même, 3 sessions par mécanisme, avec 5 % d'erreurs injectées. Comparer le taux de rattrapage et le temps par tour.
- **Robustesse de rVFC dans WebKitGTK selon les distributions.** Non vérifié en conditions réelles. *Expérience* : sonde de capacités au démarrage, avec mesure du nombre d'images reçues par seconde et de la précision de `mediaTime` sur Ubuntu LTS, Fedora et Debian stable.
- **Explication locale.** Zhang et al. (2020) n'y trouvent pas d'effet, et Buçinca et al. (2021) montrent qu'elle ne réduit pas la sur-confiance.\[36\] L'intérêt d'une surimpression (preuve visuelle) plutôt que d'un texte est une inférence. *Expérience* : A/B surimpression vs pas de surimpression, mesure des erreurs injectées rattrapées.
- **Non trouvé** : documentation primaire Nacsport, Dartfish, Metrica, LongoMatch, StatsBomb/Opta, Subtitle Edit, Descript, YouTube captions editor, aeneas, ELAN, ANVIL, Supervisely ; raccourcis officiels Premiere / Resolve / Final Cut ; littérature d'uncertainty visualization générale ; durée officielle du compte à rebours YouTube.

## Sources

1. The benefits of imperfect diagnostic automation: a synthesis of the literature — C. D. Wickens, S. R. Dixon — 2007 — https://www.tandfonline.com/doi/abs/10.1080/14639220500370105 — Méta-régression (20 études) : 0,70 = seuil sous lequel l'automatisation est pire que rien. — Primaire.
2. On the Independence of Compliance and Reliance: Are Automation False Alarms Worse Than Misses? — S. R. Dixon, C. D. Wickens, J. S. McCarley — 2007 — https://pubmed.ncbi.nlm.nih.gov/17702209/ — Les fausses alarmes nuisent plus que les omissions et touchent compliance + reliance. — Primaire.
3. Imperfect Diagnostic Automation: An Experimental Examination of Priorities and Threshold Setting — C. D. Wickens, S. R. Dixon, N. Johnson — 2006 — https://doi.org/10.1177/154193120605000301 — Une fiabilité de 0,6 aide encore ; contredit partiellement le seuil 0,70. — Primaire.
4. Complacency and Bias in Human Use of Automation: An Attentional Integration — R. Parasuraman, D. H. Manzey — 2010 — https://www.semanticscholar.org/paper/b0e5a85803bb959ed2cbd47c51009cc48059c02c — Complaisance et biais : phénomènes attentionnels, non éliminés par l'entraînement. — Primaire (notice).\[23\]
5. Humans and automation: Use, misuse, disuse, abuse — R. Parasuraman, V. Riley — 1997 — https://scholar.google.com/citations?user=F5iBMtEAAAAJ — Typologie fondatrice des usages de l'automatisation (référence via notice). — Secondaire (notice bibliographique).\[24\]
6. Trust in automation: Designing for appropriate reliance — J. D. Lee, K. A. See — 2004 — https://arxiv.org/pdf/2303.14300 — Notion de confiance calibrée (référence citée dans la bibliographie de cet article). — Secondaire (citation).\[25\]
7. Guidelines for Human-AI Interaction — S. Amershi et al. (Microsoft) — 2019 — https://www.microsoft.com/en-us/research/wp-content/uploads/2019/01/Guidelines-for-Human-AI-Interaction-camera-ready.pdf — 18 lignes directrices validées avec 49 designers sur 20 produits. — Primaire.
8. Will You Accept an Imperfect AI? — R. Kocielnik, S. Amershi, P. N. Bennett — 2019 — https://www.microsoft.com/en-us/research/wp-content/uploads/2019/01/chi19_kocielnik_et_al.pdf — Version rappel mieux acceptée que version précision ; techniques d'ajustement des attentes. — Primaire.
9. Effect of Confidence and Explanation on Accuracy and Trust Calibration in AI-Assisted Decision Making — Y. Zhang, Q. V. Liao, R. K. E. Bellamy — 2020 — https://arxiv.org/pdf/2001.02114 — La confiance affichée calibre la confiance ; l'explication locale est sans effet. — Primaire.
10. To Trust or to Think: Cognitive Forcing Functions Can Reduce Overreliance on AI — Z. Buçinca, M. B. Malaya, K. Z. Gajos — 2021 — https://arxiv.org/pdf/2102.09692 — Les explications ne réduisent pas la sur-confiance ; le forçage cognitif oui, au prix de la satisfaction. — Primaire.\[36\]
11. People + AI Guidebook — Explainability + Trust — Google PAIR — 2019 (v2 en ligne) — https://pair.withgoogle.com/chapter/explainability-trust/ — Seaux de confiance, N-best, risques des pourcentages. — Primaire (éditeur).
12. Principles of Mixed-Initiative User Interfaces — E. Horvitz — 1999 — https://erichorvitz.com/chi99horvitz.pdf — Seuils d'utilité espérée entre action, dialogue et inaction. — Primaire.
13. Memory for goals: an activation-based model — E. M. Altmann, J. G. Trafton — 2002 — https://www.interruptions.net/literature/Altmann-CogSci02.pdf — Décroissance des buts suspendus ; rôle des indices. — Primaire.
14. Task Interruption: Resumption Lag and the Role of Cues — E. M. Altmann, J. G. Trafton — 2004 — https://escholarship.org/content/qt18b4r661/qt18b4r661.pdf — Resumption lag de 3,8 s ; les indices le réduisent. — Primaire.
15. Huh, what was I doing? How people use environmental cues after an interruption — J. G. Trafton, E. M. Altmann, D. P. Brock — 2005 — https://gregtrafton.com/papers/trafton.hfes05_0.pdf — Un indice flagrant accélère la reprise. — Primaire.
16. The effects of automatic speech recognition quality on human transcription latency — Y. Gaur et al. — 2016 — http://www.cse.umich.edu/awards/pdfs/asr-threshold_w4a.pdf — Au-delà de 30 % WER, la saisie depuis zéro est plus rapide. — Primaire.
17. Incorporating ASR methods into the transcription of police-suspect interviews — Frontiers in Communication — 2023 — https://www.frontiersin.org/journals/communication/articles/10.3389/fcomm.2023.1165233/full — Synthèse : la dégradation n'est perçue que vers 45 % WER. — Secondaire.
18. Optimizing Computer-Assisted Transcription Quality with Iterative User Interfaces — M. Sperber et al. — 2016 — https://aclanthology.org/L16-1314.pdf — Surcoût inhérent de la post-édition (vérification + navigation). — Primaire.
19. Transcribing Against Time — M. Sperber et al. — 2017 — https://arxiv.org/html/1709.05227 — Hypothèse ASR affichée à côté d'une saisie depuis zéro. — Primaire.
20. Manual Post-editing of Automatically Transcribed Speeches from the Icelandic Parliament — 2018 — https://arxiv.org/pdf/1807.11893 — Seuil de rentabilité évoqué vers 25 % WER. — Primaire.
21. Influence of Pre-Annotation on POS-Tagged Corpus Development — K. Fort, B. Sagot — 2010 — https://aclanthology.org/P19-1265.pdf (cité et discuté dans Schulz et al. 2019) — Pré-annotation plus rapide mais biaisante. — Secondaire (via citation).
22. Analysis of Automatic Annotation Suggestions for Hard Discourse-Level Tasks — C. Schulz et al. — 2019 — https://aclanthology.org/P19-1265.pdf — Même des suggestions médiocres aident ; biais négligeables. — Primaire.
23. Anchoring and Agreement in Syntactic Annotations — Y. Berzak et al. — 2016 — https://arxiv.org/pdf/1605.04481 — Biais d'ancrage vers la sortie du parseur. — Primaire.
24. Quality and Efficiency of Manual Annotation: Pre-annotation Bias — 2023 — https://arxiv.org/pdf/2306.09307 — Saisie depuis zéro ≈ 1,7× plus lente, qualité égale. — Primaire.
25. Bias in the Loop: How Humans Evaluate AI-Generated Suggestions — 2025 — https://arxiv.org/pdf/2509.08514 — Moins de corrections quand corriger coûte plus ; sceptiques plus exacts. — Primaire (préprint).
26. CVAT Track mode / Vocabulary — CVAT.ai — 2024–2025 — https://docs.cvat.ai/docs/annotation/manual-annotation/modes/track-mode-basics/ — Keyframes, interpolation, raccourcis K/O/M, types manuel / semi-auto / auto. — Primaire (éditeur).
27. CVAT Now Supports Video Annotation with SAM 2 — CVAT.ai — 2024 — https://www.cvat.ai/resources/changelog/video-annotation-sam-2 — Tracker SAM 2 propagé sur N images. — Primaire (éditeur).
28. Label Studio — Label and annotate data (interactive pre-annotations) — HumanSignal — 2021–2025 — https://labelstud.io/guide/labeling/ — Auto-accept ou acceptation / rejet individuel ou global. — Primaire (éditeur).
29. Label Studio Releases 1.12.0 — HumanSignal — 2024 — https://github.com/HumanSignal/label-studio/releases — Refonte ML backend ; correctif du bouton auto-accept. — Primaire (éditeur).
30. Sportscode 11 Keyboard Shortcuts — Hudl — 2017 — https://static.hudl.com/craft/sc11-keyboardshortcuts.pdf — Tab / Shift+Tab, Ctrl+E/I/O, Échap. — Primaire (éditeur).
31. Hudl Sportscode Release Notes — Hudl — 2026 — https://www.hudl.com/releases/sportscode — 12.83.1 (16/07/2026), import XML live ; ajustement d'instances. — Primaire (éditeur).
32. Code an Instance in Hudl Sportscode Replay — Hudl Support — non daté — https://support.hudl.com/s/article/code-an-instance-in-hudl-replay-replay?language=en_US — Instance par défaut de 20 s (10 lead + 10 lag). — Primaire (éditeur).
33. Manual DGT LiveChess version 2 — DGT — 2018 — https://www.euroschach.de/media/products/dgt-schachbrett-software-livechess-2-user-manual-en-rev-1802e.pdf — Coups illégaux refusés, « ? », correction manuelle par l'opérateur. — Primaire (éditeur, miroir revendeur).
34. Arbiters and illegal moves — English Chess Forum — 2022 — https://ecforum.org.uk/viewtopic.php?t=12885 — Alerte LiveChess non surveillée en Olympiade. — Blog / forum.
35. Necessary improvements for using the Broadcaster (#67) — lichess-org — 2025 — https://github.com/lichess-org/broadcaster/issues/67 — Un coup illégal bloque toute mise à jour. — Primaire (dépôt).
36. Never Use a Warning When you Mean Undo — Aza Raskin, A List Apart — 2007 — https://alistapart.com/article/neveruseawarning/ — Habituation aux confirmations ; préférer l'undo. — Blog / essai d'expert.
37. Harder to Ignore? Revisiting Pop-Up Fatigue and Approaches to Prevent It — C. Bravo-Lillo et al. — 2014 — https://www.usenix.org/system/files/soups14-paper-bravo-lillo.pdf — Habituation : détection divisée par 3 ; médiane de 10 s à 1,2 s. — Primaire.
38. How Polymorphic Warnings Reduce Habituation in the Brain — B. B. Anderson et al. — 2015 — https://dl.acm.org/doi/10.1145/2702123.2702322 — Avertissements polymorphes plus résistants à l'habituation (IRMf). — Primaire.
39. What Do We Really Know about How Habituation to Warnings Occurs Over Time? — A. Vance et al. — 2017 — https://doi.org/10.1145/3025453.3025896 — Chute de l'attention dès la 2e exposition. — Primaire.
40. Vigilance requires hard mental work and is stressful — J. S. Warm, R. Parasuraman, G. Matthews — 2008 — https://scholar.google.com/citations?user=F5iBMtEAAAAJ — Coût de la vigilance soutenue (notice). — Secondaire (notice).\[24\]
41. How to undo send in Gmail for up to 30 seconds — I. Levine, Google Keyword blog — 2023 — https://blog.google/products-and-platforms/products/gmail/how-to-unsend-email-gmail/ — 5 s par défaut, jusqu'à 30 s. — Primaire (éditeur).
42. Cast Autoplay — Design Checklist — Google for Developers — non daté — https://developers.google.com/cast/docs/design_checklist/cast-autoplay — Compte à rebours, Play / Stop, « Up Next » au moins 30 s avant la fin. — Primaire (éditeur).
43. The keystroke-level model for user performance time with interactive systems — S. K. Card, T. P. Moran, A. Newell — 1980 — https://iihm.imag.fr/blanch/ens/2010-2011/M1/EIHM/cours/1980-Card-KLM.pdf — Opérateurs KLM, M = 1,35 s, erreur RMS ~21 %. — Primaire.\[53\]
44. Using the Keystroke-Level Model to Estimate Execution Times — D. Kieras — non daté (~2001) — https://web.eecs.umich.edu/~kieras/docs/GOMS/KLM.pdf — K = 0,28 s recommandé, H = 0,4 s, M = 1,2–1,35 s. — Secondaire (tutoriel académique).\[51\]\[52\]
45. HTMLVideoElement API: requestVideoFrameCallback — caniuse — 2026 — https://caniuse.com/mdn-api_htmlvideoelement_requestvideoframecallback — Chrome 83+, Safari 15.4+, Firefox 132+. — Primaire (base de compatibilité).\[55\]
46. requestVideoFrameCallback() — Web platform features explorer — 2024 — https://web-platform-dx.github.io/web-features-explorer/features/request-video-frame-callback/ — Baseline newly available depuis le 29/10/2024. — Primaire.\[56\]
47. [GStreamer] requestVideoFrameCallback support (bug 233541) — WebKit Bugzilla — 2021 — https://bugs.webkit.org/show_bug.cgi?id=233541 — Support rVFC côté GStreamer (WebKitGTK), résolu. — Primaire.\[58\]
48. Implement HTMLVideoElement.requestVideoFrameCallback() (bug 211945) — WebKit Bugzilla — 2020–2022 — https://bugs.webkit.org/show_bug.cgi?id=211945 — Implémentation WebKit, résolue. — Primaire.\[59\]
49. Add the « WebKitGTK and WPE WebKit » engine (#6807) — Fyrd/caniuse — non daté — https://github.com/Fyrd/caniuse/issues/6807 — caniuse ne suit pas WebKitGTK. — Primaire (dépôt).\[57\]

## Sources

1. [The benefits of imperfect diagnostic automation: a synthesis of the literature: Theoretical Issues in Ergonomics Science: Vol 8, No 3](https://www.tandfonline.com/doi/abs/10.1080/14639220500370105)
2. [On the independence of compliance and reliance: are automation false alarms worse than misses? - PubMed](https://pubmed.ncbi.nlm.nih.gov/17702209/)
3. [Mixed Initiative Interaction MPhil ACS module P230 - Alan Blackwell](https://www.cl.cam.ac.uk/teaching/2324/HCAI/HCAI-2023-mixed-initiative.pdf)
4. [Vocabulary](https://docs.cvat.ai/docs/getting_started/vocabulary/)
5. [cvat/cvat/apps/documentation/user\_guide.md at master · TaSeeMba/cvat · GitHub](https://github.com/TaSeeMba/cvat/blob/master/cvat/apps/documentation/user_guide.md)
6. [CVAT Accelerates Video Annotation with SAM 2 Integration](https://www.cvat.ai/resources/changelog/video-annotation-sam-2)
7. [Track Mode in CVAT: Video Annotation & Keyframes](https://www.cvat.ai/academy/track-mode)
8. [Label Studio Documentation — Label and annotate data](https://labelstud.io/guide/labeling/)
9. [Releases: HumanSignal/label-studio](https://github.com/HumanSignal/label-studio/releases)
10. [Code an Instance in Hudl Sportscode Replay • Hudl Support](https://support.hudl.com/s/article/code-an-instance-in-hudl-replay-replay?language=en_US)
11. [SportsCode Tutorial.pdf - HUDL SPORTSCODE TUTORIAL SPORTSCODE CHAPTER 1. I. IMPORTING VIDEO & MULTIPLE ANGLES 1. Launch application 2. Click file click](https://www.coursehero.com/file/132680469/SportsCode-Tutorialpdf/)
12. [Sportscode 11 Keyboard Shortcuts Save ⌘ + S Quit ⌘ + Q Close window ⌘ + W Copy](https://static.hudl.com/craft/sc11-keyboardshortcuts.pdf?mtime=20170307160144)
13. [Hudl Sportscode Release Notes and Changelog](https://www.hudl.com/releases/sportscode)
14. [1 Manual DGT LiveChess version 2 (Tournament software)](https://www.euroschach.de/media/products/dgt-schachbrett-software-livechess-2-user-manual-en-rev-1802e.pdf)
15. [Arbiters and illegal moves - English Chess Forum](https://ecforum.org.uk/viewtopic.php?t=12885)
16. [Necessary improvements for using the Broadcaster · Issue #67 · lichess-org/broadcaster · GitHub](https://github.com/lichess-org/broadcaster/issues/67)
17. [Optimizing Computer-Assisted Transcription Quality with ...](https://aclanthology.org/L16-1314.pdf)
18. [Transcribing Against Time](https://arxiv.org/html/1709.05227)
19. [Crossing the boundaries of automation—Function allocation and reliability - ScienceDirect](https://www.sciencedirect.com/science/article/abs/pii/S1071581914001670)
20. [On the Independence of Compliance and Reliance: Are Automation False Alarms Worse Than Misses? - Stephen R. Dixon, Christopher D. Wickens, Jason S. McCarley, 2007](https://journals.sagepub.com/doi/10.1518/001872007X215656)
21. [Will You Accept an Imperfect AI? Exploring Designs for ...](http://library.usc.edu.ph/ACM/CHI2019/1proc/paper411.pdf)
22. [Will You Accept an Imperfect AI?: Exploring Designs for Adjusting End-user Expectations of AI Systems](https://www.researchgate.net/publication/332745507_Will_You_Accept_an_Imperfect_AI_Exploring_Designs_for_Adjusting_End-user_Expectations_of_AI_Systems)
23. [Complacency and Bias in Human Use of Automation: An Attentional Integration](https://www.semanticscholar.org/paper/Complacency-and-Bias-in-Human-Use-of-Automation:-An-Parasuraman-Manzey/b0e5a85803bb959ed2cbd47c51009cc48059c02c)
24. [Raja Parasuraman - Google Scholar](https://scholar.google.com/citations?user=F5iBMtEAAAAJ&hl=en)
25. [Fast, Accurate, but Sometimes Too-Compelling Support: The Impact of Imperfectly Automated Cues in an Augmented Reality Head-Mounted Display on Visual Search Performance](https://arxiv.org/pdf/2303.14300)
26. [Quality aspects of annotated data](https://link.springer.com/article/10.1007/s11943-023-00332-y)
27. [Influence of Pre-Annotation on POS-Tagged Corpus Development — Lacuna](https://lacuna.tiptreesystems.com/work/influence-of-pre-annotation-on-pos-tagged-corpus-development/wrk_244602be53c0bbd846b41746c2adc131)
28. [arXiv:1605.04481v3 \[cs.CL\] 21 Sep 2016](https://arxiv.org/pdf/1605.04481)
29. [Analysis of Automatic Annotation Suggestions for Hard ...](https://aclanthology.org/P19-1265.pdf)
30. [Quality and Efficiency of Manual Annotation: Pre-annotation Bias](https://arxiv.org/pdf/2306.09307)
31. [Bias in the Loop: How Humans Evaluate AI-Generated Suggestions](https://arxiv.org/pdf/2509.08514)
32. <https://pair.withgoogle.com/chapter/People%20+%20AI%20Guidebook%20-%20Explainability%20+%20Trust.pdf>
33. [Guidelines for Human-AI Interaction (2019)](https://scispace.com/papers/guidelines-for-human-ai-interaction-5arf95wktw)
34. [Calibrating Trust in AI-Assisted Decision Making](https://www.ischool.berkeley.edu/sites/default/files/sproject_attachments/humanai_capstonereport-final.pdf)
35. [Efect of Confidence and Explanation on Accuracy and Trust](http://qveraliao.com/fat2020.pdf)
36. [Cognitive Forcing Functions Can Reduce Overreliance on ...](https://arxiv.org/pdf/2102.09692)
37. [To Trust or to Think: Cognitive Forcing Functions Can Reduce Overreliance on AI in AI-assisted Decision-making - ADS](https://ui.adsabs.harvard.edu/abs/2021arXiv210209692B/abstract)
38. [To Trust or to Think: Cognitive Forcing Functions Can Reduce Overreliance on AI in AI-assisted Decision-making](https://www.researchgate.net/publication/351120800_To_Trust_or_to_Think_Cognitive_Forcing_Functions_Can_Reduce_Overreliance_on_AI_in_AI-assisted_Decision-making)
39. [How to unsend an email in Gmail and increase your undo send time](https://blog.google/products-and-platforms/products/gmail/how-to-unsend-email-gmail/)
40. [How to Unsend an Email in Gmail After an Hour or a day?](https://itsourcecode.com/blogs/how-to-unsend-an-email-in-gmail-after-an-hour-or-a-day/)
41. [Cast Autoplay](https://developers.google.com/cast/docs/design_checklist/cast-autoplay?authuser=0)
42. [How I beat one of YouTube's most hated features without turning it off](https://www.androidpolice.com/how-i-beat-one-of-youtubes-most-hated-features-without-turning-it-off/)
43. [How to stop 'Up Next' videos from autoplaying on YouTube](https://www.idownloadblog.com/2016/02/06/disable-autoplay-youtube/)
44. [Never Use a Warning When you Mean Undo](https://alistapart.com/article/neveruseawarning/)
45. <http://library.usc.edu.ph/ACM/CHI%202017/1proc/p2215.pdf>
46. [Memory for goals: an activation-based model](https://www.interruptions.net/literature/Altmann-CogSci02.pdf)
47. [UC Merced Proceedings of the Annual Meeting of the Cognitive Science Society](https://escholarship.org/content/qt18b4r661/qt18b4r661.pdf)
48. [Gregtrafton](https://gregtrafton.com/papers/trafton.hfes05_0.pdf)
49. [Principles of mixed-initiative user interfaces](https://dl.acm.org/doi/10.1145/302979.303030)
50. [Hudl Sportscode keyboard shortcuts ‒ DefKey](https://defkey.com/hudl-sportscode-2023-shortcuts)
51. [Using the Keystroke-Level Model to Estimate Execution Times David Kieras](https://www.cs.umd.edu/~golbeck/INST631/KSM.pdf)
52. [Using the Keystroke-Level Model to Estimate Execution Times David Kieras](https://web.eecs.umich.edu/~kieras/docs/GOMS/KLM.pdf)
53. [The keystroke-level model for user performance time ... - IIHM](https://iihm.imag.fr/blanch/ens/2010-2011/M1/EIHM/cours/1980-Card-KLM.pdf)
54. <https://www.usenix.org/system/files/soups14-paper-bravo-lillo.pdf>
55. [HTMLVideoElement API: requestVideoFrameCallback](https://caniuse.com/mdn-api_htmlvideoelement_requestvideoframecallback)
56. [Web platform features explorer - requestVideoFrameCallback()](https://web-platform-dx.github.io/web-features-explorer/features/request-video-frame-callback/)
57. [Add the "WebKitGTK and WPE WebKit" engine for not-exactly-WebKit browsers like Epiphany (GNOME Web) and embedded devices · Issue #6807 · Fyrd/caniuse · GitHub](https://github.com/Fyrd/caniuse/issues/6807)
58. [233541](https://bugs.webkit.org/show_bug.cgi?id=233541)
59. [211945](https://bugs.webkit.org/show_bug.cgi?id=211945)
