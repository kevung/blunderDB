Recherche du 2026-10-30 — claude.ai, recherche approfondie.
Prompt : ../prompts/09-videau-et-score.md

---

# blunderDB — Sujet 09 « Petits objets : videau et score » : revue de littérature et recommandations

**Réponse courte : seule la surveillance de l'overlay du stream mérite d'être faite maintenant, et elle doit détecter un changement dans une zone fixe, pas lire les chiffres.** Il faut renoncer à lire la face du videau, différer la détection visuelle des doubles et prises, et ne pas suivre le cube avec un tracker générique. La raison est simple : la machine de transcription connaît déjà la valeur du cube (2 puissance le nombre de doubles), son propriétaire et le score. Les événements de videau sont rares et se saisissent d'une touche. Enfin, les études sur l'automatisation imparfaite montrent que les fausses alertes coûtent plus que les oublis.\[1\]

## TL;DR

- **Faire : un détecteur de changement sur l'overlay.** L'utilisateur dessine une fois la zone de l'overlay. On calcule une différence d'images ou un hash perceptuel sur cette imagette réduite, et le résultat sert de signal « fin de partie / score modifié ». Optionnellement, on lit les chiffres par gabarits appris automatiquement à partir du score déjà connu par la machine. Tesseract.js et PaddleOCR sont inutiles ici.
- **Différer : le geste de double ou de prise.** Si on le fait un jour, ce sera une simple *suggestion* fondée sur l'occupation de trois zones fixes (centre, côté A, côté B) entre deux lancers, jamais une validation automatique. La détection de mains (MediaPipe) n'est pas justifiée pour quelques événements par partie.
- **Renoncer :** à la lecture de la face (6 classes) et au suivi continu du cube par KCF, CSRT ou MOSSE. Pour la lecture, la valeur se déduit du nombre de doubles. Pour le suivi, aucune publication ne mesure ces trackers sur un objet de 10 à 40 px saisi à la main, et CSRT n'existe pas dans le build standard d'opencv.js.

## Key Findings

### 1. Suivi d'un petit objet (10–40 px)

**Les filtres de corrélation.**
- **MOSSE** (Bolme, Beveridge, Draper, Lui, CVPR 2010, DOI 10.1109/CVPR.2010.5539960) introduit le filtre à somme minimale d'erreur quadratique en sortie.\[2\] Il fournit aussi un critère de confiance, le *peak-to-sidelobe ratio* (PSR). Selon l'article lui-même (§3.5), la PSR de MOSSE « typically ranges between 20.0 and 60.0 » en suivi normal, et une chute « to around 7.0 » indique une occlusion ou un décrochage. Le tableau 1 de la version longue de l'article donne 669 images/s pour MOSSE, sur un Core 2 Duo à 2,4 GHz.
- **KCF** (Henriques, Caseiro, Martins, Batista, IEEE TPAMI 37(3):583–596, mars 2015, DOI 10.1109/TPAMI.2014.2345390) exploite la structure circulante des échantillons pour diagonaliser le problème par transformée de Fourier discrète.\[3\]\[4\]
- **CSR-DCF / CSRT** (Lukežič et al., CVPR 2017, DOI 10.1109/CVPR.2017.515 ; IJCV 126:671–688, 2018, DOI 10.1007/s11263-017-1061-3) ajoute une carte de fiabilité spatiale et des poids par canal, avec des descripteurs HoG et Colornames.\[5\]\[6\]\[7\]

**Contradiction mineure sur CSR-DCF.** Le résumé CVPR 2017 dit que CSR-DCF « runs in real-time on a CPU ».\[5\]\[8\] La version IJCV 2018 et la page du laboratoire disent « close to real-time on a CPU ».\[7\]\[9\]

**Benchmarks de petits objets (2023–2025).**
- BioDrone (arXiv 2402.04519, IJCV 2023) identifie « tiny target and fast motion » comme les deux principaux défis de robustesse du suivi d'objet unique.\[10\]
- XS-VID (arXiv 2407.18137, 2024) propose une piste de suivi d'objet unique dédiée aux petits objets : 374 séquences et 1,4 million de boîtes, dont 29,4 % font moins de 12² px et 72,2 % moins de 32² px. Dans sa table IV, SeqTrack (CVPR 2023) obtient une AUC de succès de 52,94 à 54,26, et le meilleur score cité dans l'article plafonne à 66,4 %, loin de la saturation.
- CST Anti-UAV (arXiv 2507.23473, 2025) et EV-UAV (arXiv 2506.23575, 2025, cibles d'environ 6,8 × 5,4 px en moyenne) confirment que les petites cibles restent un problème ouvert, même pour des réseaux profonds lourds.\[11\]\[12\]

**Ce que je n'ai pas trouvé.** Aucune étude ne mesure MOSSE, KCF ou CSRT sur un objet de 10 à 40 px qui est saisi à la main, sort du champ et est reposé ailleurs. C'est pourtant le cas typique du videau.

**Inférence (non mesurée).** Avec des cellules HoG de 4 px, un cube de 10 px ne donne que 2 à 3 cellules de côté. Le filtre manque alors de support. De plus, ces trackers sont conçus pour le court terme : dès que la main occulte le cube, la mise à jour du modèle risque d'apprendre la main.

**Disponibilité en JavaScript.**
- Le build wasm standard d'opencv.js n'inclut pas TrackerCSRT, même compilé avec `opencv_contrib` (ticket opencv_contrib #4021).\[13\]
- Il faut un build Emscripten personnalisé du module tracking. Un exemple est le dépôt KVGrygoriev/opencv_tracking_for_javascript, qui intègre TrackerKCF.\[14\]
- Le tutoriel LearnOpenCV (mis à jour pour OpenCV 4.14 et 5.0) indique que CSRT est « Not exposed by the tested builds » d'OpenCV 5.0.0.\[15\]
- La taille d'un bundle opencv.js est donnée à « ~8–12 MB » par une source de type blog.\[16\] C'est une source secondaire.

**Conclusion pratique.** Un suivi continu n'est pas nécessaire : la caméra est fixe et le cube n'a que trois emplacements légitimes (centre, côté de chaque joueur). Une **corrélation croisée normalisée (NCC) par zone**, calculée seulement aux instants clés, suffit. Ces instants sont fournis par la machine : avant chaque lancer et après chaque coup. Quelques dizaines de lignes de JS sur `getImageData` font l'affaire, sans opencv.js.

### 2. Lecture de la face du cube

**Précisions rapportées pour des cas proches** (je n'ai rien trouvé pour la face du videau elle-même, avec ses nombres 2 à 64) :
- **Chiffres imprimés en scène.** Sur SVHN, Goodfellow et al. (arXiv 1312.6082, 2013, révisé en 2014) obtiennent 97,84 % de précision par chiffre et 96,03 % par séquence, sur des imagettes de 54 × 54 px. Le modèle compte 11 couches et a été entraîné sur une infrastructure distribuée de Google.\[17\]\[18\]
- **Dés, nombre de points.** Un rapport de la Western Kentucky University (TopSCHOLAR, non daté) annonce 95,10 % à 99,76 % de précision de validation sur environ 16 000 images Kaggle.\[19\] Un dépôt GitHub (acabbar) rapporte seulement 86,46 % en test sur des images 28 × 28, mais sur 96 échantillons, ce qui est trop peu pour conclure.\[20\] Le dépôt nell-byler/dice_detection signale que les erreurs se concentrent sur les angles obliques et les dés peu contrastés.\[21\]
- **Backgammon.** Les travaux trouvés portent sur le plateau, les pions et les dés, jamais sur le videau : BackgammonCV (YOLO, GitHub), un jeu de 49 images annotées sur Roboflow (octobre 2023), l'application iOS Backgammon Eye (2025, reconnaissance de position par photo et export XGID).\[22\]\[23\]\[24\] **Non trouvé : aucun jeu de données ni article sur le videau.**

**Analyse.**
- Une face de 10 à 40 px, vue en perspective, compressée en H.264, porte des nombres à deux chiffres (16, 32, 64) dont les traits font 1 à 3 px. C'est en dessous de la résolution SVHN. Sans données annotées propres au videau, une précision proche de celle de SVHN n'est pas démontrable.
- Surtout, **l'information est redondante** : la valeur vaut 2 puissance le nombre de doubles acceptés, et la machine la connaît déjà.
- Seule exception : les cubes de départ posés sur 64 au centre, ou une mise en place erronée. La convention « cube au centre = 1 » s'applique sans lecture.

**Verdict : renoncer.**

### 3. Détection du geste de double, de prise et de passe

**Signature exploitable sans modèle.** La séquence d'un double est la suivante :
1. Le cube quitte sa zone (centre ou côté du joueur qui le possède).
2. Il réapparaît dans la zone de l'adversaire.
3. Tout cela se produit entre la fin d'un coup et le lancer suivant.

Les états de la machine délimitent la fenêtre temporelle à observer, et la NCC des trois zones donne l'occupation. Une prise se lit par l'absence de remise en place des pions et la reprise des lancers. Une passe se lit par la remise à zéro des pions, que le sujet « reconnaissance de position » doit déjà détecter.

**Faux positifs typiques (inférence, aucune mesure publiée trouvée) :**
- cube bousculé ou recentré sans double ;
- main posée sur la zone (occlusion prise pour un départ) ;
- cube tourné pour être plus lisible ;
- changement de lumière ou d'exposition automatique ;
- cube utilisé comme presse-papier ou caché par un gobelet ;
- double « verbal » sans geste, puis cube déplacé plus tard.

On distingue un double d'un cube bousculé ainsi : il faut un déplacement *d'une zone à une autre*, persistant plus de N images après stabilisation, et non une simple variation dans la même zone.

**Détection de mains.** MediaPipe Hand Landmarker détecte 21 points par main, avec un modèle de détection de paume puis un modèle de points clés. La documentation Google (mise à jour le 29 octobre 2026) donne une latence moyenne de 17,12 ms sur CPU et 12,27 ms sur GPU, mesurée sur un Pixel 6.\[25\] Selon Qualcomm AI Hub, les deux modèles pèsent 6,75 Mo et 7,70 Mo en float (14,45 Mo au total), pour 1,76 M et 2,01 M paramètres et une entrée en 256 × 256. C'est trop lourd pour quelques événements par partie. Il existe aussi un risque de portabilité : la délégation GPU dépend de WebGL ou WebGPU, et WebGPU n'est pas activé par défaut dans WebKitGTK (voir plus bas).

**Verdict : différer.** Si on le fait, ce sera en mode suggestion, avec un seuil réglé pour peu de fausses alertes.

### 4. Overlay du stream

**Faisabilité reconnue de longue date.** Des brevets d'analyse de matchs télévisés exploitent quatre indices de l'overlay de score (*score bug*) : il est unique, situé à un emplacement fixe, statique d'une image à l'autre, et présent pendant le jeu (US 7 653 131 et US 7 474 698, « Identification of replay segments »).\[26\]\[27\] Le brevet US 12 217 506 (« scoreboard text region detection ») repère la zone justement par son caractère statique face au reste de l'image, qui bouge.\[28\] Dans la littérature récente, Guo et al. (SIFT + OCR) et le cadre ASAP (Agarwal et al.) utilisent l'overlay comme « ancre sémantique stable », selon arXiv 2510.18405 (2025).\[29\] Le projet courtvision (2026) indique que l'overlay ESPN affiche souvent le score *plusieurs secondes après* l'action, ce qui impose une tolérance de décalage.\[30\]

**Coût de l'OCR généraliste en webview.**
- **Tesseract.js** : 15,34 Mo par défaut (ticket naptha/tesseract.js #806), dont 10,4 Mo de données de langue.\[31\] Le cœur wasm pèse 2,73 à 3,3 Mo (tesseract.js-core 5.1.0 sur jsDelivr).\[32\] Le fork tesseract-wasm descend à « ~2.1MB » compressé en Brotli.\[33\]\[34\]
- **PaddleOCR** : en_PP-OCRv5_mobile_rec pèse 7,5 Mo (85,25 % de précision moyenne selon la documentation PaddleOCR) et PP-OCRv5_mobile_det 4,7 Mo.\[35\] Le dépôt PaddleOCR annonce un SDK navigateur officiel, PaddleOCR.js.\[36\] L'entrée de 2026 qui le mentionne est une source éditeur, non vérifiée en usage.
- Ces deux solutions sont démesurées pour 2 à 6 chiffres d'une police fixe dans une zone fixe.

**Technique minimale recommandée.**
- **Détection de changement.** L'utilisateur trace l'overlay une fois. Toutes les 0,5 à 1 s, on réduit l'imagette (par exemple à 32 × 16 en niveaux de gris) puis on calcule la différence absolue moyenne ou un dHash par rapport à l'état de référence. Un changement stable pendant plus de 1 s devient un événement « overlay modifié ». La machine le rapproche ensuite de son état : une fin de partie attendue (passe, gammon) confirme, un changement inattendu déclenche une alerte douce.
- **Lecture optionnelle par gabarit auto-étiqueté.** La machine connaît le score : à chaque changement confirmé, elle peut découper les glyphes et les étiqueter *automatiquement* avec la valeur attendue. Elle constitue ainsi en quelques parties une bibliothèque de gabarits propre à ce stream, puis classe les chiffres par NCC. **Non trouvé : aucune publication ne décrit cet auto-étiquetage par la connaissance de l'état du jeu**, c'est une proposition.
- **Risques.** Overlays animés, transitions, publicités et caméras qui bougent : un hash perceptuel est plus tolérant qu'une différence pixel à pixel. Il faut aussi gérer l'absence d'overlay, fréquente sur les streams de club.

**Verdict : faire** la détection de changement. La lecture par gabarit est un « faire plus tard », peu coûteux.

### 5. Priorité : faut-il automatiser des événements rares ?

**Ce que dit la littérature.**
- **Bainbridge (1983)**, « Ironies of automation », *Automatica* 19(6):775–779, DOI 10.1016/0005-1098(83)90046-8.\[37\]\[38\] Automatiser transforme l'opérateur en surveillant de tâches qu'il pratique moins.\[39\]\[40\] Ici, l'utilisateur devrait surveiller une détection rare, ce qui est l'inverse d'un gain.
- **Wickens & Dixon (2007)**, *Theoretical Issues in Ergonomics Science* 8(3):201–212, DOI 10.1080/14639220500370105.\[41\] Cette méta-analyse (20 études, 35 points de données) trouve un point de bascule autour d'une fiabilité de **0,70**, sous lequel l'automatisation imparfaite fait pire que pas d'automatisation.\[42\]\[43\]\[44\]
- **Contradiction partielle** : Wickens, Dixon & Johnson (2006) trouvent qu'une fiabilité de 0,6 aidait encore les opérateurs dans leur tâche.\[45\] Le seuil dépend donc de la charge de travail et du type d'erreur.\[44\]
- **Dixon, Wickens & McCarley (2007)**, *Human Factors* 49(4):564–572, DOI 10.1518/001872007X215656.\[41\] Les automatismes sujets aux **fausses alertes** dégradent davantage la performance que ceux sujets aux oublis, car les fausses alertes atteignent à la fois l'obéissance aux alertes et la confiance dans le système.\[1\]
- **Annotation vidéo assistée.** Vondrick, Ramanan & Patterson (ECCV 2010, DOI 10.1007/978-3-642-15561-1_44 ; IJCV 2012) montrent un compromis coût/précision entre interpolation et travail humain.\[46\]\[47\] Ils notent que la saisie à cadence fixe peut coûter moins cher que de faire revisionner la vidéo.\[48\]

**Non trouvé.** Je n'ai trouvé aucune étude mesurant le gain d'automatiser des *événements rares* au sein d'un flux par ailleurs automatisé. Je n'en ai trouvé aucune non plus comparant directement « confirmer une suggestion » et « taper une touche » pour un événement ponctuel.

**Raisonnement économique (estimation, à vérifier par l'expérience ci-dessous).**
- Un événement de videau coûte une touche (d, t ou p) et l'utilisateur regarde déjà la vidéo pour confirmer coups et dés. Le gain maximal se compte donc en secondes par partie.
- Une fausse alerte coûte une interruption, un retour en arrière et une vérification visuelle.
- Le détecteur n'est rentable que si son taux de fausses alertes par heure de vidéo est presque nul.

L'overlay est le cas inverse : il ne demande ni attention ni frappe et fournit une vérification croisée du score, qui sert à détecter les dérives de la machine. Son coût d'erreur est faible.

### Contexte plateforme (webview, sans dépendance native)

**Récupération des images vidéo : `requestVideoFrameCallback` (rVFC).**
- Caniuse et web-features : Chrome et Edge 83 (mai 2020), Safari 15.4 (mars 2022), Firefox 132 (29 octobre 2024). Statut « Baseline Newly Available » depuis le 29 octobre 2024.\[49\]
- WebKit l'a implémenté dans le changeset r284528 (20 octobre 2021).\[50\]
- **Non trouvé** : une note de version WebKitGTK 2.44–2.52 confirmant l'activation par défaut.
- web.dev avertit que l'API est « best effort » et peut avoir une image de décalage.\[51\] Une PR GitHub (source non primaire) rapporte qu'en WebKit, rVFC ne signale qu'environ la moitié des images.\[52\] Pour des événements de videau qui durent des secondes, c'est sans conséquence.

**WebCodecs.**
- WebKitGTK : prise en charge annoncée en 2.44 (27 mars 2024), activation par défaut en 2.46 (septembre 2024) selon le compte rendu Igalia du WebKit Contributors Meeting 2024. En 2.48 (8 avril 2025), l'option `prefer-hardware` est enfin respectée.\[53\]\[54\]\[55\]
- Safari : support vidéo depuis 16.4, marqué « partiel » par caniuse jusqu'à 18.x, puis complet à partir de 26.0.\[56\]\[57\]
- Inutile ici : `drawImage` d'un `<video>` sur canvas puis `getImageData` sur de petites zones suffit.

**WebGPU.**
- Livré dans Safari 26.0 (septembre 2025), d'après le billet WebKit « WebKit Features in Safari 26.0 ».\[58\]
- Edge depuis la version 113, selon caniuse.\[59\]
- WebKitGTK : **non trouvé** comme activé par défaut, aucune mention dans les notes 2.48, 2.50 ou 2.52. WebKitGTK 2.52.0 est sorti le 18 mars 2026,\[60\] et 2.52.4 le 2 juin 2026.\[61\]

**WebNN.** Seulement en origin trial dans Edge 146–149 (2026), suspendu puis expiré selon le suivi du groupe W3C.\[62\] Aucune implémentation WebKit trouvée. Non utilisable.

**Conséquence.** Tout ce qui est recommandé ici tourne en JS pur sur CPU (canvas 2D, typed arrays), donc identiquement sur les trois webviews.

## Details : recommandation par sous-détection

| Sous-détection | Décision | Technique la plus simple qui tienne | Pourquoi |
|---|---|---|---|
| Overlay : changement | **Faire** | Zone tracée par l'utilisateur ; dHash ou différence absolue moyenne sur imagette 32 × 16 en niveaux de gris, toutes les 0,5–1 s ; événement si stable plus de 1 s | Zone fixe, statique par nature ; coût CPU négligeable ; vérifie le score connu |
| Overlay : lecture du score | Faire plus tard | Gabarits NCC auto-étiquetés par le score connu de la machine | Pas de modèle à embarquer ; Tesseract.js pèse 15,34 Mo par défaut\[31\] |
| Overlay : valeur du cube | Renoncer | Même principe que le score, si l'overlay l'affiche | Redondant avec l'état de la machine |
| Présence du cube par zone | Différer | NCC sur 3 zones fixes, calculée seulement aux transitions de la machine | Base nécessaire si l'on fait un jour la suggestion de double |
| Geste de double ou de prise | Différer | Changement d'occupation zone A → zone B entre la fin d'un coup et le lancer suivant, persistant N images ; suggestion seulement | Rare, une touche suffit ; les fausses alertes coûtent plus que les oublis |
| Passe | Renoncer en tant que détection propre | Réutiliser la détection de remise en place des pions et le changement d'overlay | Déjà couverte par d'autres signaux |
| Lecture de la face | Renoncer | Valeur = 2 puissance le nombre de doubles | Aucune donnée ni précision publiée à cette taille ; information redondante |
| Suivi continu (MOSSE, KCF, CSRT) | Renoncer | — | Cas « saisi à la main » non couvert ; CSRT absent d'opencv.js ; inutile avec caméra fixe |
| Détection de mains (MediaPipe) | Renoncer | — | 14,45 Mo de modèles (Qualcomm AI Hub) pour quelques événements par partie |

## 1. Verdict

Pour le videau, la vision ne vaut pas l'effort aujourd'hui. L'état du cube est entièrement déductible de la machine, la saisie coûte une touche, et les fausses alertes sont le risque principal, le plus coûteux d'après Dixon et al. 2007.\[1\] Pour l'overlay, l'effort est faible et le gain réel : vérification croisée du score et signal de fin de partie. C'est la seule sous-détection à faire maintenant, en détection de changement d'abord.

## 2. Recommandation technique

1. **Pipeline commun.** `<video>` local, puis rVFC si disponible, sinon `requestAnimationFrame` ou un minuteur échantillonné toutes les 250–500 ms. Ensuite `drawImage` sur un canvas réduit, puis `getImageData` sur les seules zones d'intérêt. Pas d'opencv.js, de WebGPU ni de modèle à télécharger.
2. **Overlay.** Une interface « tracer la zone du score », un dHash 64 bits sur l'imagette en niveaux de gris, un seuil de Hamming et une hystérésis temporelle de plus de 1 s. L'événement est confronté à l'état de la machine : confirmer si attendu, signaler si inattendu, ne jamais modifier le score seul.
3. **Plus tard, lecture par gabarits.** Découpe des glyphes par projection verticale, étiquetage automatique par le score connu, classification NCC avec un seuil de rejet. Le score lu ne sert qu'à la vérification.
4. **Si l'on reprend un jour le videau.** NCC sur trois zones (centre, côté A, côté B) calibrées à la première image ou par un clic de l'utilisateur. Calcul seulement entre « coup validé » et « dés saisis ». Une suggestion visuelle non bloquante, du type « double détecté ? [d] », avec un seuil réglé pour moins d'une fausse alerte par heure.

## 3. Incertitudes et expériences à mener

- **Taux de fausses alertes réel du détecteur de zones du cube.** Non trouvé dans la littérature. *Expérience* : constituer 15 à 20 vidéos de streams variés (angle, lumière, matériel) et annoter à la main chaque action de videau et chaque cube bousculé. Mesurer rappel, précision et fausses alertes par heure de la NCC par zone. Ne passer en « faire » que si le rappel dépasse 90 % avec moins d'une fausse alerte par heure.
- **Gain utilisateur réel.** Non trouvé dans la littérature. *Expérience* : test intra-sujet sur 5 matchs, saisie manuelle (d, t, p) contre suggestion à confirmer. Mesurer le temps total de transcription, les erreurs résiduelles et la charge perçue (NASA-TLX court). Si l'écart est inférieur au bruit, abandonner définitivement.
- **Robustesse du hash d'overlay.** Overlays animés, publicités et fondus. *Expérience* : sur 10 streams avec overlay, compter les changements détectés qui ne sont pas des changements de score, et ajuster la zone et l'hystérésis.
- **Support effectif de rVFC et de WebGPU dans WebKitGTK 2.52** : non confirmé par une source primaire. *Expérience* : test de fonctionnalités dans la build Wails Linux (`'requestVideoFrameCallback' in HTMLVideoElement.prototype`, `navigator.gpu`).
- **Lecture de la face** : aucune précision publiée à 10–40 px. Ce n'est utile que si un jour l'état de la machine devient indisponible, par exemple pour transcrire depuis le milieu d'un match.

## 4. Sources numérotées

1. *Visual object tracking using adaptive correlation filters* — Bolme, Beveridge, Draper, Lui — CVPR 2010 — https://doi.org/10.1109/CVPR.2010.5539960 — Introduit MOSSE et le critère de confiance PSR. Primaire.
2. *High-Speed Tracking with Kernelized Correlation Filters* — Henriques, Caseiro, Martins, Batista — IEEE TPAMI, 2015 — https://www.robots.ox.ac.uk/~joao/publications/henriques_tpami2015.pdf — KCF/DCF, structure circulante et transformée de Fourier discrète. Primaire.
3. *Discriminative Correlation Filter with Channel and Spatial Reliability* — Lukežič et al. — CVPR 2017 / arXiv 1611.08461 — https://openaccess.thecvf.com/content_cvpr_2017/papers/Lukezic_Discriminative_Correlation_Filter_CVPR_2017_paper.pdf — CSR-DCF (CSRT), « real-time on a CPU ». Primaire.
4. *Discriminative Correlation Filter Tracker with Channel and Spatial Reliability* — Lukežič et al. — IJCV 2018 — https://link.springer.com/article/10.1007/s11263-017-1061-3 — Version longue, « close to real-time ». Primaire.
5. *BioDrone: A Bionic Drone-based Single Object Tracking Benchmark* — Zhao et al. — arXiv 2402.04519 / IJCV 2023 — https://arxiv.org/abs/2402.04519 — Petites cibles et mouvement rapide comme défis principaux du suivi d'objet unique. Primaire.
6. *XS-VID: A Large-Scale Benchmark for Small Object Detection and Tracking in Videos* — 2024 — https://arxiv.org/html/2407.18137 — Piste de suivi d'objet unique pour petits objets, SeqTrack environ 53–54. Primaire.
7. *CST Anti-UAV* — 2025 — https://arxiv.org/html/2507.23473v1 — Suivi de très petits drones, 220 séquences. Primaire.
8. *Event-based Tiny Object Detection (EV-UAV)* — 2025 — https://arxiv.org/html/2506.23575v1 — Cibles d'environ 6,8 × 5,4 px. Primaire.
9. *TrackerCSRT not included in the wasm bundle*, issue #4021 — opencv/opencv_contrib (GitHub) — https://github.com/opencv/opencv_contrib/issues/4021 — CSRT absent du build wasm. Primaire (ticket).
10. *opencv_tracking_for_javascript* — KVGrygoriev (GitHub) — https://github.com/KVGrygoriev/opencv_tracking_for_javascript — Build Emscripten personnalisé du module tracking (KCF). Dépôt avec code.
11. *Object Tracking using OpenCV (C++/Python)* — LearnOpenCV — mis à jour 2026 — https://learnopencv.com/object-tracking-using-opencv-cpp-python/ — CSRT « not exposed » dans les builds OpenCV 5.0.0 testés. Secondaire.
12. *HTMLVideoElement API: requestVideoFrameCallback* — caniuse — 2026 — https://caniuse.com/mdn-api_htmlvideoelement_requestvideoframecallback — Chrome 83, Safari 15.4, Firefox 132. Primaire (tableau de support).
13. *Web platform features explorer — requestVideoFrameCallback()* — W3C WebDX — https://web-platform-dx.github.io/web-features-explorer/features/request-video-frame-callback/ — Baseline Newly Available le 29 octobre 2024. Primaire.
14. *Perform efficient per-video-frame operations* — web.dev (Google) — https://web.dev/articles/requestvideoframecallback-rvfc — « best effort », décalage possible d'une image. Primaire éditeur.
15. *Changeset 284528* — WebKit Trac — 20 octobre 2021 — https://trac.webkit.org/changeset/284528/webkit — Ajout de rVFC dans WebKit. Primaire.
16. *WebKitGTK 2.44 / 2.48 / 2.50 / 2.52 highlights et notes de version* — The WebKitGTK Project / Igalia — 2024–2026 — https://webkitgtk.org/2025/04/08/webkitgtk-2.48.html, https://webkitgtk.org/2026/03/18/webkitgtk-2.52-highlights.html — WebCodecs (2.44, défaut en 2.46), `prefer-hardware` (2.48), dates de sortie 2.52. Primaire.
17. *WebKit Contributors Meeting 2024 — Igalia status* — docs.webkit.org — 2024 — https://docs.webkit.org/Other/Contributor%20Meetings/ContributorMeeting2024.html — « WebCodecs … enabled by default in 2.46 ». Primaire.
18. *WebKit Features in Safari 26.0* — WebKit (Apple) — septembre 2025 — https://webkit.org/blog/17333/webkit-features-in-safari-26-0/ — WebGPU livré dans Safari 26.0. Primaire.
19. *Microsoft Edge release notes 146 / 148* — Microsoft Learn — 2026 — https://learn.microsoft.com/en-us/microsoft-edge/web-platform/release-notes/148 — WebNN en origin trial seulement. Primaire.
20. *Hand landmarks detection guide* — Google AI Edge — mis à jour le 29 octobre 2026 — https://developers.google.com/edge/mediapipe/solutions/vision/hand_landmarker — 21 points, latence Pixel 6 de 17,12 ms (CPU) et 12,27 ms (GPU). Primaire éditeur.
21. *MediaPipe-Hand-Detection* — Qualcomm AI Hub — https://aihub.qualcomm.com/models/mediapipe_hand — Tailles de 6,75–7,70 Mo et nombres de paramètres. Secondaire (portage tiers).
22. *Multi-digit Number Recognition from Street View Imagery using Deep CNNs* — Goodfellow, Bulatov, Ibarz, Arnoud, Shet — arXiv 1312.6082, 2013–2014 — https://arxiv.org/abs/1312.6082 — SVHN, 96,03 % par séquence et 97,84 % par chiffre. Primaire.\[63\]
23. *Using Machine Learning to Interpret Dice Rolls* — Western Kentucky University (TopSCHOLAR) — non daté — https://digitalcommons.wku.edu/cgi/viewcontent.cgi?article=1004&context=seas_faculty_pubs — CNN sur dés, 95,10–99,76 % en validation. Secondaire (rapport étudiant ou faculté).
24. *dice_detection* — nell-byler (GitHub) — https://github.com/nell-byler/dice_detection — Erreurs sur angles obliques et faible contraste. Dépôt avec code, de type blog.
25. *Multi-Class Dice Classification with a CNN* — acabbar (GitHub) — https://github.com/acabbar/Multi-Class-Dice-Classification-with-a-Convolutional-Neural-Network- — 86,46 % en test sur 96 images 28 × 28. Dépôt, faible fiabilité.
26. *BackgammonCV* — christiancorro (GitHub) — https://github.com/christiancorro/BackgammonCV — Reconnaissance d'état de plateau par YOLO, sans videau. Dépôt avec code.
27. *Backgammon Dataset v6* — Roboflow Universe — octobre 2023 — https://universe.roboflow.com/boardgamedetection/backgammon-eofws/dataset/6 — 49 images annotées (plateau, pions, dés). Secondaire.
28. *Backgammon Eye* — Wordeep Backgammon (App Store) — 2025 — https://apps.apple.com/us/app/backgammon-eye/id6744810330 — Reconnaissance de position par photo, export XGID. Marketing (blog).
29. *Reduce file sizes*, issue #806 — naptha/tesseract.js — https://github.com/naptha/tesseract.js/issues/806 — 15,34 Mo par défaut. Primaire (ticket).\[31\]
30. *tesseract-wasm* — Robert Knight (GitHub) — https://github.com/robertknight/tesseract-wasm — Environ 2,1 Mo compressé en Brotli, SIMD. Dépôt avec code.\[33\]
31. *Text Recognition Module / OCR pipeline* — documentation PaddleOCR — 2025–2026 — http://www.paddleocr.ai/main/en/version3.x/module_usage/text_recognition.html — en_PP-OCRv5_mobile_rec de 7,5 Mo, PP-OCRv5_mobile_det de 4,7 Mo. Primaire éditeur.\[35\]
32. *PaddleOCR* — PaddlePaddle (GitHub) — 2026 — https://github.com/PADDLEPADDLE/PADDLEOCR — Annonce de PaddleOCR.js, SDK navigateur. Primaire éditeur.\[36\]
33. *Identification of replay segments* (US 7 653 131 et US 7 474 698) et *Methods and systems for scoreboard text region detection* (US 12 217 506) — USPTO — https://image-ppubs.uspto.gov/dirsearch-public/print/downloadPdf/12217506 — Détection de l'overlay de score par emplacement fixe et caractère statique. Primaire (brevets).\[28\]
34. *Automated Wicket-Taking Delivery Segmentation… Using OCR-Guided YOLOv8* — 2025 — https://arxiv.org/pdf/2510.18405 — Revue de Guo et al. et d'ASAP : l'overlay comme ancre sémantique. Primaire.\[29\]
35. *courtvision*, PR #3 — aiden-guan (GitHub) — 2026 — https://github.com/aiden-guan/courtvision/pull/3 — OCR de l'overlay ESPN, score affiché en retard sur l'action. Dépôt, de type blog.\[30\]
36. *Ironies of automation* — Bainbridge — *Automatica*, 1983 — https://doi.org/10.1016/0005-1098(83)90046-8 — L'automatisation déplace l'humain vers la surveillance. Primaire.\[38\]
37. *The benefits of imperfect diagnostic automation: a synthesis of the literature* — Wickens & Dixon — TIES, 2007 — https://www.tandfonline.com/doi/abs/10.1080/14639220500370105 — Point de bascule à une fiabilité de 0,70. Primaire.\[43\]
38. *Imperfect Diagnostic Automation: An Experimental Examination of Priorities and Threshold Setting* — Wickens, Dixon, Johnson — HFES 2006 — https://doi.org/10.1177/154193120605000301 — Une fiabilité de 0,6 aidait encore, ce qui nuance le seuil de 0,70. Primaire.\[42\]
39. *On the Independence of Compliance and Reliance: Are Automation False Alarms Worse Than Misses?* — Dixon, Wickens, McCarley — *Human Factors*, 2007 — https://doi.org/10.1518/001872007x215656 — Les fausses alertes coûtent plus que les oublis. Primaire.\[1\]
40. *Efficiently Scaling Up Video Annotation with Crowdsourced Marketplaces* — Vondrick, Ramanan, Patterson — ECCV 2010 — https://link.springer.com/chapter/10.1007/978-3-642-15561-1_44 — Compromis coût/précision entre interpolation et annotation humaine. Primaire.\[47\]
41. *Efficiently Scaling up Crowdsourced Video Annotation* — Vondrick, Patterson, Ramanan — IJCV 2012 — http://www.djp3.net/wp-content/uploads/2016/07/djp3_JR_010.pdf — Bonnes pratiques et coût de la saisie à cadence fixe contre le revisionnage. Primaire.\[48\]

## Caveats

- Plusieurs faits de support navigateur viennent de caniuse ou de mémoire de l'agent auxiliaire et ne sont pas confirmés par la documentation Microsoft WebView2 : rVFC et WebGPU dans WebView2, WebAssembly SIMD, OffscreenCanvas. Je n'ai trouvé aucune source primaire WebKitGTK pour l'activation par défaut de rVFC, de WebGPU, de WebAssembly SIMD et d'OffscreenCanvas.
- Les seuils proposés (plus de 1 s, moins d'une fausse alerte par heure, rappel supérieur à 90 %, imagette 32 × 16) sont des choix d'ingénierie, pas des valeurs publiées.
- Les précisions de reconnaissance de dés citées viennent de rapports et de dépôts peu contrôlés, sur des images bien plus grandes que 10–40 px : elles ne se transposent pas au videau.

## Sources

1. [(PDF) On the Independence of Compliance and Reliance: Are Automation False Alarms Worse Than Misses?](https://www.researchgate.net/publication/6137978_On_the_Independence_of_Compliance_and_Reliance_Are_Automation_False_Alarms_Worse_Than_Misses)
2. [Visual object tracking using adaptive correlation filters - Oak Ridge National Laboratory](https://impact.ornl.gov/en/publications/visual-object-tracking-using-adaptive-correlation-filters/)
3. [High-Speed Tracking with Kernelized Correlation Filters](https://www.computer.org/csdl/journal/tp/2015/03/06870486/13rRUy3xY3Q)
4. [High-Speed Tracking with Kernelized Correlation Filters - ADS](https://ui.adsabs.harvard.edu/abs/2015ITPAM..37..583H/abstract)
5. [Discriminative correlation filter with channel and spatial reliability - CUED Publications database](http://publications.eng.cam.ac.uk/988965/)
6. [Discriminative Correlation Filter Tracker with Channel and Spatial Reliability](https://link.springer.com/article/10.1007/s11263-017-1061-3)
7. [Discriminative Correlation Filter with Channel and Spatial Reliability](https://arxiv.org/pdf/1611.08461)
8. [GitHub - alanlukezic/csr-dcf: Discriminative Correlation Filter with Channel and Spatial Reliability · GitHub](https://github.com/alanlukezic/csr-dcf)
9. [Discriminative Correlation Filter Tracker with Channel and Spatial Reliability](https://www.vicos.si/publications/lukezic2018discriminative/)
10. [\[2402.04519\] BioDrone: A Bionic Drone-based Single Object Tracking Benchmark for Robust Vision](https://arxiv.org/abs/2402.04519)
11. [Event-based Tiny Object Detection: A Benchmark Dataset and Baseline](https://arxiv.org/html/2506.23575v1)
12. [CST Anti-UAV: A Thermal Infrared Benchmark for Tiny UAV Tracking in Complex Scenes](https://arxiv.org/html/2507.23473v1)
13. [TrackerCSRT not included in the wasm bundle · Issue #4021 · opencv/opencv\_contrib · GitHub](https://github.com/opencv/opencv_contrib/issues/4021)
14. [GitHub - KVGrygoriev/opencv\_tracking\_for\_javascript: OpenCVs Tracking module for emscripten toolchain build · GitHub](https://github.com/KVGrygoriev/opencv_tracking_for_javascript)
15. [Object Tracking using OpenCV (C++/Python)](https://learnopencv.com/object-tracking-using-opencv-cpp-python/)
16. [WASM for browser OpenCV](https://theneuralbase.com/opencv/learn/advanced/wasm-for-browser-opencv/)
17. [Multi-digit Number Recognition from Street View](https://arxiv.org/pdf/1312.6082)
18. [\[1312.6082\] Multi-digit Number Recognition from Street View Imagery using Deep Convolutional Neural Networks](https://ar5iv.labs.arxiv.org/html/1312.6082)
19. [(PDF) Using Machine Learning to Interpret Dice Rolls - Engineering](https://www.academia.edu/90926624/Using_Machine_Learning_to_Interpret_Dice_Rolls)
20. [GitHub - acabbar/Multi-Class-Dice-Classification-with-a-Convolutional-Neural-Network-: Hey, This project explores the use of Convolutional Neural Networks (CNNs) for image classification by developing a deep learning model capable of recognizing the face value of dice from grayscale images. The objective was to build, train, and evaluate a CNN that can accurately classify images into six different classes. · GitHub](https://github.com/acabbar/Multi-Class-Dice-Classification-with-a-Convolutional-Neural-Network-)
21. [GitHub - nell-byler/dice\_detection: Project to detect and classify six-sided dice from images using deep learning. · GitHub](https://github.com/nell-byler/dice_detection)
22. [GitHub - christiancorro/BackgammonCV: Automatic recognition of the Backgammon game board.](https://github.com/christiancorro/BackgammonCV)
23. [Backgammon Object Detection Model (v6, 2023-10-12 11:56am) by boardgamedetection](https://universe.roboflow.com/boardgamedetection/backgammon-eofws/dataset/6)
24. [Backgammon Eye - App Store - Apple](https://apps.apple.com/us/app/backgammon-eye/id6744810330)
25. [Hand landmarks detection guide | Google AI Edge | Google for Developers](https://developers.google.com/edge/mediapipe/solutions/vision/hand_landmarker)
26. [Identification of replay segments](https://image-ppubs.uspto.gov/dirsearch-public/print/downloadPdf/7653131)
27. [Identification of replay segments](https://image-ppubs.uspto.gov/dirsearch-public/print/downloadPdf/7474698)
28. [Methods and systems for scoreboard text region detection](https://image-ppubs.uspto.gov/dirsearch-public/print/downloadPdf/12217506)
29. [Automated Wicket-Taking Delivery Segmentation and Trajectory-Based Dismissal-Zone Analysis in Cricket Videos Using OCR-Guided YOLOv8](https://arxiv.org/pdf/2510.18405)
30. [NBA broadcast analysis, learning loop, and Studio redesign by aiden-guan · Pull Request #3 · aiden-guan/courtvision · GitHub](https://github.com/aiden-guan/courtvision/pull/3)
31. [Reduce file sizes · Issue #806 · naptha/tesseract.js · GitHub](https://github.com/naptha/tesseract.js/issues/806)
32. [tesseract.js-core CDN by jsDelivr - A free, fast, and reliable Open Source CDN](https://cdn.jsdelivr.net/npm/tesseract.js-core@5.1.0/)
33. [GitHub - robertknight/tesseract-wasm: JS/WebAssembly build of the Tesseract OCR engine for use in browsers and Node · GitHub](https://github.com/robertknight/tesseract-wasm)
34. [tesseract wasm](https://npmjs.com/package/tesseract-wasm)
35. [PaddleOCR/docs/version3.x/pipeline\_usage/OCR.en.md at main · PaddlePaddle/PaddleOCR · GitHub](https://github.com/PaddlePaddle/PaddleOCR/blob/main/docs/version3.x/pipeline_usage/OCR.en.md)
36. [GitHub - PaddlePaddle/PaddleOCR: Turn any PDF or image document into structured data for your AI. A powerful, lightweight OCR toolkit that bridges the gap between images/PDFs and LLMs. Supports 100+ languages. · GitHub](https://github.com/PADDLEPADDLE/PADDLEOCR)
37. [Brief paper: Ironies of automation: Automatica (Journal of IFAC): Vol 19, No 6](<https://dl.acm.org/doi/abs/10.1016/0005-1098(83)90046-8>)
38. [Bainbridge, L. (1983), “Ironies of automation. Automatica”, (19(6), 775-779) - RailHOF](https://www.railhof.org/2022/10/10/bainbridge-l-1983-ironies-of-automation-automatica-196-775-779/)
39. [Ironies of Automation](https://en.wikipedia.org/wiki/Ironies_of_Automation)
40. [The inattentional gap in task conditioned AI models that omit otherwise reportable safety critical signals](https://arxiv.org/pdf/2606.26529)
41. [Stephen R. Dixon - Publications](https://neurotree.org/beta/publications.php?pid=108016)
42. [(PDF) Imperfect Diagnostic Automation: An Experimental Examination of Priorities and Threshold Setting](https://www.researchgate.net/publication/252737807_Imperfect_Diagnostic_Automation_An_Experimental_Examination_of_Priorities_and_Threshold_Setting)
43. [Crossing the boundaries of automation—Function allocation and reliability - ScienceDirect](https://www.sciencedirect.com/science/article/abs/pii/S1071581914001670)
44. [The benefits of imperfect diagnostic automation: a synthesis of the literature: Theoretical Issues in Ergonomics Science: Vol 8, No 3](https://www.tandfonline.com/doi/abs/10.1080/14639220500370105)
45. [Imperfect Diagnostic Automation: An Experimental Examination of Priorities and Threshold Setting - Christopher D. Wickens, Stephen R. Dixon, Nicholas Johnson, 2006](https://doi.org/10.1177/154193120605000301)
46. [Efficiently Scaling Up Video Annotation with Crowdsourced Marketplaces](https://link.springer.com/chapter/10.1007/978-3-642-15561-1_44)
47. [Efficiently Scaling up Video Annotation with Crowdsourced Marketplaces](https://mlanthology.org/eccv/2010/vondrick2010eccv-efficiently/)
48. [Efficiently Scaling up Crowdsourced Video Annotation](http://www.djp3.net/wp-content/uploads/2016/07/djp3_JR_010.pdf)
49. [Web platform features explorer - requestVideoFrameCallback()](https://web-platform-dx.github.io/web-features-explorer/features/request-video-frame-callback/)
50. [Changeset 284528](https://trac.webkit.org/changeset/284528/webkit)
51. [Perform efficient per-video-frame operations | Articles | web.dev](https://web.dev/articles/requestvideoframecallback-rvfc)
52. [Fix Safari frame loss in static-site pose extraction by zzzhushi · Pull Request #68 · zzzhushi/running-gait-analysis · GitHub](https://github.com/zzzhushi/running-gait-analysis/pull/68)
53. [WebKitGTK+ 2.48 highlights - The WebKitGTK Project](https://webkitgtk.org/2025/04/08/webkitgtk-2.48.html)
54. [WebKit Contributors Meeting 2024 - WebKit Documentation](https://docs.webkit.org/Other/Contributor%20Meetings/ContributorMeeting2024.html)
55. [What's new in WebKitGTK 2.44? - The WebKitGTK Project](https://webkitgtk.org/2024/03/27/webkigit-2.44.html)
56. [WebCodecs API](https://caniuse.com/webcodecs)
57. [WebCodecs: Browser Support, Features, Use Cases](https://www.testmuai.com/learning-hub/webcodecs-browser-support/)
58. [WebKit Features in Safari 26.0](https://webkit.org/blog/17333/webkit-features-in-safari-26-0/)
59. [WebGPU](https://caniuse.com/webgpu)
60. [\[webkit-gtk\] WebKitGTK 2.52.0 released](http://www.mail-archive.com/webkit-gtk@lists.webkit.org/msg04051.html)
61. [WebKitGTK 2.52.4 released! - The WebKitGTK Project](https://webkitgtk.org/2026/06/02/webkitgtk2.52.4-released.html)
62. [What is the current testing/shipping status of WebNN in Chrome and Edge, and what should developers use today? · Issue #964 · webmachinelearning/webnn · GitHub](https://github.com/webmachinelearning/webnn/issues/964)
63. [\[1312.6082\] Multi-digit Number Recognition from Street View Imagery using Deep Convolutional Neural Networks](https://arxiv.org/abs/1312.6082)
