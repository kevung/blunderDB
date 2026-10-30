Recherche du 2026-10-30 — claude.ai, recherche approfondie.
Prompt : ../prompts/02-etat-de-l-art-des-et-plateau.md

---

# État de l'art : lecture des dés et du plateau de backgammon depuis une vidéo — revue de littérature pour blunderDB (2015–2026)

La réponse courte : aucune source publiée ne fournit une solution clé en main pour le backgammon filmé. En revanche, la combinaison « calibration du plateau en 4 clics + détection de stabilité + comparaison différentielle restreinte aux coups légaux fournis par le moteur » est la voie la mieux étayée par la littérature des échecs. Elle est réalisable entièrement dans la webview, avec `<video>` + canvas pour l'extraction des images et ONNX Runtime Web (WASM) pour un petit classifieur de dés.

## TL;DR

- **Dés** : les travaux publiés en conditions contrôlées annoncent 99 à 100 % de dés lus correctement. Les projets ouverts testés hors de leur distribution d'entraînement se dégradent nettement (angles obliques, dés à faible contraste). Pour une caméra de stream fixe, l'estimation argumentée est de ~95–97 % par dé avec un compteur de points classique seul, et de ~98–99,5 % avec un petit CNN sur des vignettes recadrées et un vote sur plusieurs images stables. Pour fiabiliser, l'utilisateur doit ensuite confirmer.
- **Plateau / coup joué** : aucun projet de backgammon ne publie de précision mesurée sur vidéo réelle. Le meilleur chiffre ouvert (homographie + HoughCircles) donne 90 % de points bien comptés mais seulement 40 % de plateaux entièrement exacts. La reconnaissance absolue est donc à proscrire. En revanche, choisir parmi 2 à 30 coups candidats qui ne diffèrent que sur 2 à 4 points est un problème bien plus facile : on peut viser ~95–99 % en top-1 et >99 % en top-3, sous réserve de l'expérience décrite plus bas.
- **Échecs → transfert** : la reconnaissance absolue reste fragile hors du domaine d'entraînement (Chesscog : 93,86 % de plateaux exacts en synthétique, mais seulement 2,30 % sur ChessReD, avec 73,64 % d'erreur par case selon Masouris & van Gemert ; ChessReD : 15,26 %). Seul un modèle ViT-L de 2026 sature les benchmarks, et il est trop lourd pour la webview. Les systèmes vidéo qui marchent sont tous **différentiels** : détection de la main, états stables, puis confrontation avec la liste des coups légaux. C'est l'architecture à copier.

## Key Findings

1. **La stratégie différentielle est le consensus de fait des systèmes vidéo d'échecs, de 2008 à 2025.** Elle est utilisée par Piškorec et al. (image de référence plus seuil de différence sur N images statiques), Wang & Green (seuillage hystérétique des images de différence sur n images, avec coups et grilles d'occupation fournis par un moteur), VICE (paires d'images avant/après un coup) et Roman (Brno, 2025 : YOLOv8, pause pendant la présence de la main, combinaisons de cases changées confrontées à la liste des coups légaux).\[1\]\[2\]\[3\]\[4\] Le dépôt oliverfrost1 (YOLOv11, avec un modèle YOLOv11l dédié aux mains) applique le même principe.\[5\]
2. **La reconnaissance absolue ne généralise pas.** Le détecteur de Chesscog, réglé sur des images synthétiques, ne localise le plateau que sur 34,38 % des photos de test réelles de ChessReD (732 images), selon Masouris & van Gemert (« the corresponding accuracy in our dataset is 34.38% »). Les auteurs de ChessQueries (2026) mesurent un échec de la détection des coins de Chesscog sur 65,7 % / 96,8 % / 50,0 % des images de ChessReD / SLCC / CVChess.
3. **Le backgammon est très peu étudié.** On trouve 4 à 6 dépôts amateurs, sans aucune évaluation publiée sur vidéo, et un article arXiv (Marom, 2025) qui travaille sur des diagrammes de livres.\[6\] Aucune source ne traite explicitement le comptage de piles de plus de 5 pions ni la détection des pions sortis.
4. **Les applications mobiles de scan existent (GammonSnap, Backgammon Eye, Sage Snap), mais aucune ne publie de précision.**\[7\]\[8\] Sage Snap prévoit lui-même qu'« un point mal lu se corrige d'un tapotement », ce qui revient à reconnaître des erreurs de lecture.\[9\]\[10\]\[11\]
5. **Côté navigateur, tout le nécessaire est disponible dans les trois moteurs, à des dates différentes.** requestVideoFrameCallback est présent depuis Safari 15.4 (2022) et Chrome 83 (2020).\[12\] WebCodecs est complet depuis Chrome 94 (2021) et Safari 26 (septembre 2025), et passe par GStreamer dans WebKitGTK.\[13\]\[14\]\[15\] WebGPU est livré dans Safari 26.\[16\] Le statut de WebGPU dans WebKitGTK : non trouvé.

## Details

### 1. Lecture de la valeur des dés

**Méthodes classiques (seuillage, blobs, composantes connexes).**
- **Golsteyn (blog, non daté dans la source consultée).** Flou médian, puis `SimpleBlobDetector` d'OpenCV en supposant les points circulaires. Les points sont regroupés en dés (clustering) et la valeur est donnée par le nombre de points du groupe. Un blob non circulaire est traité comme une face non horizontale ou comme du bruit.\[17\] Aucune précision n'est rapportée.
- **RoboRealm (tutoriel).** Ce tutoriel documente exactement le cas « deux dés qui se touchent ». La zone entre deux dés en contact crée un blob blanc parasite, ce qui fausse le décompte (17 au lieu de 15, l'arrière-plan comptant pour un blob de plus). La correction consiste à filtrer par intensité ou par circularité (seuil 0,60).\[18\]
- **« Image Identification Scheme for Dice Game » (ResearchGate, date non vérifiée).** Plan R au lieu des niveaux de gris, binarisation d'Otsu, rejet des objets non circulaires et recherche « IMLDSM » fondée sur la structure des dés. Les auteurs annoncent **100 %** sur plus de 200 images de test de formes, styles, tailles et couleurs variés. Les difficultés citées sont un fond de la même couleur que les dés, des dés aux arêtes arrondies, des dés qui se touchent, et le zoom, qui rend la taille des points irrégulière.\[19\]\[20\]
- **Hsu et al., « Multivariate dice recognition using invariant features » (date non vérifiée).** Homographie, identification des points et dés blancs, verts, bleus et rouges, en conditions de casino. L'erreur d'homographie est d'environ 1,7 pixel sur une image de 192×144. Les auteurs mesurent les taux de reconnaissance selon trois angles de vue et neuf conditions, avec un objectif fixé à 99,9 %.\[21\]
- **Brevets.** US 10828556 décrit un scanner à plat qui capture l'image entre 10 et 350 ms après l'arrêt des dés et annule le lancer si un geste (main, doigt) est détecté avant l'arrêt.\[22\] US 12676051 (craps) décrit une double lecture des points, par caméras proche-UV et en lumière ambiante, avec alerte en cas de désaccord.\[23\] C'est un précédent industriel utile : redondance et validation temporelle.

**CNN et détecteurs.**
- **nell-byler/dice_detection (GitHub, avec jeu de données Kaggle).** 250 photos annotées à la main avec labelImg, détecteur SSD-MobileNet v1 FPN en 640×640, et un classifieur 1–6 inspiré de LeNet-5. L'auteur note des erreurs sur les « angles de vue obliques » et les « dés à faible contraste (points rouges sur plastique bleu/noir translucide) ».\[24\] Un fork (debuglevel) précise : « does not really work well on new data ».\[25\]
- **WKU, « Using Machine Learning to Interpret Dice Rolls » (2020).** Images Kaggle recadrées en 100×100 et passées en niveaux de gris. Pour le d6 : **99,34 %** en entraînement et **99,76 %** en validation.\[26\] La validation provient toutefois de la même distribution (mêmes caméras fixes).
- **buehlpa/Dice (ZHAW).** Pipeline hybride directement transposable. La vision classique distingue trois états de scène (« vide », « en mouvement », « dés immobiles ») par un simple seuil. Elle isole ensuite des vignettes quasi carrées en niveaux de gris, que classe un modèle entraîné sur les découpes de nell-byler et sur des images maison.\[27\]
- **YOLO-nano.** YOLO11-N compte 2,6 M de paramètres pour une entrée de 640 et 39,5 mAP50-95 sur COCO (tableau du dépôt nomi30701, source secondaire).\[28\] Aucune précision YOLO-nano publiée spécifiquement pour la lecture de dés : non trouvé.

**Cas difficiles, état des connaissances.**
| Cas | Ce que dit la littérature | Source |
|---|---|---|
| Perspective / angle oblique | Principale cause d'erreur des détecteurs CNN entraînés de face | nell-byler\[24\] |
| Dés sombres à points clairs, faible contraste | Erreurs explicites ; un seuillage à polarité fixe échoue | nell-byler ; RoboRealm (filtre par couleur) |
| Deux dés qui se touchent | Blob parasite entre les dés ; filtre de circularité ou d'intensité | RoboRealm ; « Image Identification Scheme »\[18\] |
| Dé cocké | Non trouvé (aucune source ne le traite) | — |
| Dé partiellement hors cadre | Images retirées manuellement des jeux de données (le cas est donc évité, pas résolu) | Kaggle ucffool\[29\] |
| Reflets | Non trouvé de manière quantifiée ; la double caméra proche-UV / ambiante du brevet craps vise ce type de problème | US 12676051\[23\] |
| Dés transparents, mains | Traités qualitativement dans un projet vidéo classique | Kishaan/Dice-Detection\[30\] |

**Jeux de données publics de dés.**
| Jeu | Taille | Licence | Remarques |
|---|---|---|---|
| Kaggle koryakinp « D6 Dice – Images » | 11 000 images | CC0 | Même angle et orientation de caméra : très homogène\[31\] |
| Kaggle koryakinp « D6 Dice – Videos » | 13 500 vidéos (5–7 dés) | CC0 | Seul jeu vidéo trouvé\[32\] |
| Kaggle ucffool « Dice d4…d20 » | ~16 000 images, 480×480 | non vérifiée | Plateau tournant, 2 angles fixes minimum ; images hors cadre retirées\[29\] |
| Kaggle nomihsa965 « Dice Detection Upper View » | 705 images | non vérifiée | Vue de dessus, format YOLOv5, 6 classes\[33\] |
| Kaggle nellbyler « d6-dice » | ~250 photos annotées | non vérifiée | Boîtes et valeurs\[24\]\[34\] |
| Roboflow roboflow-gw7yv/dice | 359 images annoncées | domaine public | La description cite aussi 154 + 388 + 13 images : **comptes incohérents** dans la source\[35\] |
| Roboflow dicedataset/dice-txtg5 | 441 images | CC BY 4.0 | Détection\[36\] |
| Roboflow yolo-hkw8z/dice-detection | 132 images | CC BY 4.0 | Détection\[37\] |

Aucun de ces jeux ne montre des dés de backgammon filmés par une caméra de stream sur un plateau. Il faudra constituer un jeu de données maison.

### 2. État d'un plateau de backgammon

| Projet / source | Méthode | Précision rapportée | Contraintes | Données | Licence | Fiabilité |
|---|---|---|---|---|---|---|
| christiancorro/BackgammonCV (dernière mise à jour juillet 2022) | YOLO (poids sur Google Drive) | Non trouvé | Non documentées | Non documentées | GPL-3.0 | Dépôt, faible\[38\]\[39\] |
| nitzankoll/backGammon-analyzer + article Medium (18/01/2025) | YOLOv5 ; plateau en deux moitiés, barre comme classe à part ; coordonnées → quadrants → 6 points ; chaîne gnubg `set board simple` | Qualitatif seulement (« accurately detected ») | **Photo strictement de dessus** ; HoughCircles abandonné (paramètres non transférables) ; boîtes du plateau trop grandes en perspective | 30 images au départ, puis « élargi » (taille finale non donnée) | MIT | Dépôt + blog\[40\]\[41\] |
| jvitors23/backgammon-checker-detection-openCV | Homographie à partir de **coins fournis à la main** + HoughCircles + comptage par point | **90 % de points corrects, 40 % de plateaux exacts** | Calibration manuelle | Photos 2019, vérité terrain JSON, effectif non indiqué | Aucune | Dépôt\[42\] |
| Marom, arXiv 2501.05030 (2025) — en réalité un article sur un cadre RAG (MCBR-RAG) où la lecture du plateau n'est qu'une étape | CNN qui prédit 4 repères, découpage en 24 points + barre, 2e CNN qui prédit nombre et propriétaire des pions par point | Pas de chiffre direct ; « Our case retention rate with this procedure is approximately 90% » (§4.2) | Diagrammes de livres, pas de photos | 100 000 images simulées ; 1 015 cas réels tirés de Magriel (*Backgammon*), Robertie (*501 Essential Backgammon Problems*) et Trice (*Backgammon Bootcamp*) | — | Primaire |
| Roboflow « Backgammon » (boardgamedetection, v6 du 12/10/2023) | Détection board / dice / bar / dark_p / light_p | mAP@50 98,2 % (chiffre Roboflow) | Dés détectés mais **non lus** | 49 images | CC BY 4.0 | Faible (jeu minuscule)\[43\]\[44\] |
| BotGammon (A. Borenshtein, Maker Faire Rome, oct. 2024) | « Board detection » + réseau de neurones pour les dés + gnubg CLI | Non trouvé ; « précision millimétrique » promotionnelle et non mesurée | Robot, caméra fixe | Non trouvé | « Unspecified » | Secondaire (Hackster)\[45\]\[46\]\[47\] |
| Tranks27/BackgammonTracking | Journalisation de parties par vision (vidéo YouTube) | Non trouvé | Non trouvé | Non trouvé | Non trouvé | Vidéo seulement\[48\] |
| NadavBenda/robotic-backgammon | YOLO sur PC/Jetson (prévu) | Aucun (étape encore à la feuille de route) | Robot | — | MIT | Dépôt\[49\] |
| Arkadium Backgammon Dataset (Hugging Face) | Journaux de parties humaines + captures d'écran WebP | — | **Plateaux numériques**, pas de photos | 1K–10K | CC BY 4.0 | Primaire, hors domaine\[50\] |

**Comptage des piles, barre, pions sortis.** Aucune source ne traite explicitement les piles de plus de 5 pions, ni la détection des pions sortis. La barre est traitée comme une classe ou une tranche séparée chez nitzankoll, Marom et Roboflow.\[6\]\[40\]\[43\] Les approches « un pion = une détection » vues de dessus supposent implicitement que chaque pion reste visible. Pour les piles chevauchantes, la piste la plus solide est celle de Marom : régresser ou classer le *compte* par tranche de point.\[6\]

**Applications mobiles.**
| Application | Plateforme / date | Affirmations | Précision publiée |
|---|---|---|---|
| GammonSnap (Baumliebhaber) | Android, Google Play depuis mars 2026 ; ~1,9 k téléchargements | « Works with any backgammon board. No setup required. » | Non trouvé\[7\]\[9\]\[51\]\[52\] |
| Backgammon Eye / Gammon Eye | iOS v1.1 du 15/05/2025 ; Android | Reconnaissance photo instantanée, export XGID, surimpression des coups | Non trouvé\[8\]\[10\]\[53\] |
| Sage Snap / Backgammon Sage Pro (bgsage.ai, M. Higgins) | iOS / Android | Photo prise de dessus avec un guide de cadrage ; « un point mal lu se corrige d'un tapotement », y compris dés et videau | Non trouvé\[11\]\[54\] |
| Trackgammon (Backgammon Galaxy) | Service | Vidéo → fichier MAT par vision | Non trouvé (accroche marketing seulement)\[55\] |

Ces applications travaillent sur **une photo prise de dessus avec un guide de cadrage**, pas sur une vidéo de stream en perspective.\[11\] Leur existence montre que le problème est traitable. Leur conception (correction manuelle intégrée) montre aussi qu'il n'est pas résolu à 100 %.

### 3. Transfert depuis les échecs

| Système | Méthode | Précision | Conditions | Données / licence |
|---|---|---|---|---|
| Chesscog (Wölflein & Arandjelović, J. Imaging 2021, arXiv 2104.14963) | Lignes + RANSAC → homographie ; CNN d'occupation puis CNN de pièce | 93,86 % de plateaux exacts, 0,23 % d'erreur par case (synthétique) ; sur un jeu inédit après apprentissage few-shot (2 images) : 88,89 % de plateaux exacts et 0,17 % d'erreur par case, soit 99,83 % de cases justes, tels que rapportés par Masouris & van Gemert (§5.1) | Orientation fournie ; 27 images de test | 4 888 images Blender (OSF) ; code GitHub |
| LiveChess2FEN (Mallasén Quintana et al., arXiv 2012.06858, 2020) | Détection du plateau accélérée + CNN par case + connaissances du domaine | 92 % pour les pièces, 95 % pour la détection du plateau, < 1 s sur Jetson Nano | Embarqué | Code GitHub, ONNX / TensorRT\[56\]\[57\] |
| ChessReD (Masouris & van Gemert, VISAPP 2024, arXiv 2310.04086) | ResNeXt-101 de bout en bout, 64 têtes × 13 classes | 15,26 % de plateaux exacts (~7× Chesscog, qui tombe à 2,30 % de plateaux sans erreur sur ce jeu, avec 73,64 % d'erreur par case et 42,87 cases fausses par plateau, tableau 3) | Smartphones, angles variés | 10 800 photos, 100 parties, 3 téléphones (Apple iPhone 12, Huawei P40 pro, Samsung Galaxy S8), découpage 6 479 / 2 192 / 2 129 (train/val/test) ; ChessReD2K : 2 078 images issues de 20 parties, avec boîtes et coins |
| CVChess (Abeykoon et al., Université de Toronto, arXiv 2511.11522, 2025) | Redressement en vue de dessus + CNN | 1 145 plateaux exacts sur 1 790 (63,96 %) en test sur ChessReD ; sur de nouvelles photos iPhone 15 Pro (rejouant Kasparov–Topalov 1999), seulement 133 sur 445 (29,8 %) et 65,17 % de cases justes | « we still lost about one-sixth of our images due to failed board detection » | — |
| ChessQueries (Seytre, arXiv 2608.30762, 31/08/2026) | ViT-L + décodeur de type DETR (64 requêtes) | 99,5 % ChessReD, 98,5 % ChessCog, **87,1 % SLCC (retransmissions)**, 87,6 % CVChess ; LLM multimodaux généralistes : 0–2,6 % | Trop lourd pour la webview (ViT-L) | SLCC : 2 174 images de retransmissions, CC BY-NC-SA\[58\] |
| Faster R-CNN (arXiv 2610.09191, 2026) | Détection d'objets | 99,57 % par case, 76,61 % par plateau (synthétique) | — | Jeu Chesscog\[59\] |

**Systèmes vidéo différentiels et leurs pannes.**
- **Sokić (2008)** : image de différence entre images avant, pendant et après le coup. L'auteur estime qu'on peut dépasser 95 % de coups reconnus **en lumière contrôlée**.\[60\]
- **Piškorec et al.** : vidéo 640×484 à 5 images/s, 74 coups. Un coup est validé après N images statiques consécutives (N = 8 testé). Panne documentée : si le joueur garde la main **immobile** au-dessus de l'échiquier, l'image peut être prise à tort pour un état stable.\[3\]
- **Wang & Green (webcam RGB au-dessus du plateau)** : seuillage hystérétique sur n images ; coups et grilles d'occupation fournis par un moteur d'échecs.\[2\]
- **VICE (Zhu, 2023)** : paires d'images avant/après, détection d'« objets de changement ». Pannes : occlusion entre pièces, bras du joueur (exemple d'Aronian), ombres, et, en blitz, un joueur qui commence son coup avant que l'adversaire ait lâché sa pièce, si bien qu'aucune image n'est libre de mains.\[4\]
- **Roman (Brno, Excel@FIT 2025)** : 92,6 % mAP@0.5 sur 12 classes de pièces. À cause du bruit de détection, il peut y avoir plus de deux cases changées : toutes les combinaisons sont générées puis filtrées par les coups légaux.\[1\] S'il reste plusieurs candidats, un filtrage supplémentaire est nécessaire.\[1\]
- **Poggi (blog)** : soustraction de fond et seuil sur le nombre de pixels pour détecter l'entrée et la sortie de la main. Panne observée : les coups sur la rangée la plus proche du joueur, où la main recouvre peu l'échiquier. Correctif : élargir la zone de détection de la main.\[61\]
- **DGT** : ChessQueries rappelle que les problèmes techniques des échiquiers électroniques DGT sont fréquents lors des retransmissions.\[58\] Les capteurs ne sont donc pas une vérité terrain parfaite non plus.

**Transposition au backgammon.** Le backgammon est *plus favorable* que les échecs pour l'approche différentielle. L'ensemble des coups légaux est petit et fourni par le moteur. Les pions sont de deux couleurs seulement, sans classification de type de pièce. Et un coup ne modifie que 2 à 4 points, plus éventuellement la barre. Il est en revanche *moins favorable* sur deux aspects : les piles se chevauchent, et les dés comme le videau se trouvent sur le plateau, où ils peuvent masquer des points.

### 4. Faisabilité dans la webview

| Brique | Chromium / WebView2 | WKWebView (macOS) | WebKitGTK (Linux) |
|---|---|---|---|
| `<video>` + `requestVideoFrameCallback` + canvas | Chrome 83 (19/05/2020) | Safari 15.4 (14/03/2022) | Moteur WebKit ; version exacte dans WebKitGTK non vérifiée\[12\] |
| WebCodecs `VideoDecoder` | Chrome / Edge 94 (septembre 2021) | Partiel (vidéo seulement) de 16.4 à 18.7 ; complet depuis Safari 26 (15/09/2025) ; correctif d'ordre des images H264 dans 26.4 (mars 2026) | Via GStreamer ; option prefer-hardware respectée depuis 2.48 (avril 2025) ; **plantage de WebKitWebProcess signalé au décodage H.264** (issue TeamCraft #824)\[13\]\[14\]\[15\]\[62\]\[63\] |
| WebGPU | Chrome 113 | Safari 26 (macOS Tahoe 26) ; WKWebView non activé par défaut selon une source secondaire | Non trouvé\[64\]\[65\] |
| ONNX Runtime Web | WASM par défaut ; WebGPU via `onnxruntime-web/webgpu` | Cité comme fonctionnant dans Safari 26 (blog WebKit) | WASM attendu ; non vérifié\[16\]\[66\] |
| OpenCV.js / TensorFlow.js | Non vérifié dans cette recherche | idem | idem |

Note de contradiction : pour requestVideoFrameCallback dans Firefox, le bugzilla Mozilla 1800882 indique « fixed » pour Firefox 130, tandis que web-features et caniuse donnent Firefox 132 (29/10/2024).\[12\]\[67\] La première version a probablement été livrée derrière une préférence. Sans incidence pour blunderDB, qui n'embarque pas Gecko.

**Conséquence pratique.** WebCodecs exige un démultiplexeur JavaScript et présente un risque de plantage sous WebKitGTK. La voie robuste et commune aux trois moteurs est `<video>` avec seek ou lecture, `requestVideoFrameCallback`, puis `drawImage` + `getImageData` sur un canvas. Elle suffit, car il ne faut que quelques images stables par coup, pas toutes les images.

### 5. Précision atteignable (estimations argumentées)

**Par dé.** Le plafond « laboratoire » est de 99–100 % (200 images ; 99,76 % en validation intra-distribution).\[26\] Le plancher « hors distribution » est un détecteur entraîné sur 250 photos de face qui « ne marche pas bien sur de nouvelles données ».\[24\]\[25\] Avec une caméra fixe, il est possible de normaliser la vignette (homographie locale, taille connue), de choisir la polarité de seuillage par vidéo et de voter sur 5 à 10 images stables. J'estime ~95–97 % par dé pour le comptage de blobs seul, et **~98–99,5 %** pour un CNN léger (moins de 100 k paramètres, vignettes de 32×32 à 64×64) entraîné sur des vignettes maison. Les dés cockés et partiellement masqués sont à détecter et à **signaler** plutôt qu'à lire. Le contrôle de cohérence par le moteur aide : un lancer dont aucun coup légal ne correspond au plateau d'arrivée observé devient suspect. Ce sont des estimations, pas des mesures publiées.

**Discrimination entre 2 et 30 candidats.** Le meilleur chiffre absolu ouvert (90 % de points bien comptés) implique qu'environ un plateau sur deux contient au moins une erreur.\[42\] En différentiel, on ne mesure que le *changement* sur 2 à 4 zones connues, avec une hypothèse fermée. On compare par exemple l'image de différence des tranches de points à la signature attendue de chaque candidat (ajout ou retrait en haut de pile, barre). Les candidats qui ne diffèrent que par la hauteur d'une pile au-delà de 5 pions (6 contre 7) sont les plus durs. J'estime **~95–99 % en top-1 et >99 % en top-3** sur des images stables sans main. La présentation d'une liste triée à confirmer est donc la bonne interface.

### 6. Gestes de videau
Aucune source sur la détection d'un videau (cube) ou de ses gestes (proposer, prendre, passer) n'a été trouvée. Piste dérivée : un objet cubique qui change de position ou d'orientation entre deux états stables, avec la valeur lue comme un dé à faces numérotées. Non validé.

## Recommendations

1. **Verdict.** Prototyper le système *différentiel assisté*, pas une reconnaissance absolue. La littérature (échecs 2008–2026, backgammon amateur) montre que l'absolu échoue hors domaine, tandis que la confrontation aux coups légaux transforme le problème en un choix fermé, facile à confirmer pour l'utilisateur.

2. **Recommandation technique — trois techniques à prototyper en premier.**
   - **(a) Calibration manuelle + homographie + détection de stabilité.** L'utilisateur clique les 4 coins du plateau et la zone des dés une fois par vidéo. L'homographie (8 inconnues, en JS pur) permet de découper 24 tranches de points, la barre et les zones de sortie. Les états stables sont détectés par différence d'images avec hystérésis et N images consécutives, plus un détecteur de main par masque de mouvement élargi, comme chez Piškorec, Wang & Green et Poggi.\[2\]\[3\]\[61\] Aucune dépendance native n'est nécessaire.
   - **(b) Score différentiel des coups candidats.** Pour chaque coup légal classé par le réseau de backgammon, calculer la signature attendue (tranches modifiées, sens du changement, hauteur du sommet de pile). La comparer à la différence observée entre l'état stable avant et après le coup, par corrélation sur des profils le long de l'axe de chaque pointe. Présenter le top-3 classé. Commencer par des descripteurs simples (variance, couleur moyenne par segment de pointe) avant tout CNN.
   - **(c) Lecture des dés en deux étages.** Localisation par différence et stabilité dans la zone des dés, puis comptage de blobs avec polarité auto et filtre de circularité. Ajouter un mini-CNN ONNX (WASM) entraîné sur des vignettes extraites des vidéos de l'utilisateur, avec vote temporel et seuil de confiance qui déclenche la saisie manuelle.
   - **Extraction d'images** : `<video>` + `requestVideoFrameCallback` + canvas. Garder WebCodecs comme optimisation détectée à l'exécution, et le désactiver sous WebKitGTK tant que la stabilité n'est pas vérifiée.

3. **Ne pas faire** : entraîner un YOLO de plateau complet en premier (aucun jeu de données réaliste n'existe, et les essais amateurs exigent des vues de dessus), ou s'appuyer sur un LLM multimodal (0–2,6 % de plateaux exacts aux échecs selon ChessQueries).\[40\]\[58\]

## Caveats

**Incertitudes et expériences à mener.**
- **Précisions estimées, non mesurées.** Expérience : annoter 5 vidéos de matériels, angles et éclairages différents (environ 300 lancers et 600 coups). Mesurer (i) la précision par dé pour blobs seuls et pour blobs + CNN, (ii) le top-1 et le top-3 du score différentiel, ventilés selon le nombre de candidats (2–5, 6–15, 16–30) et selon la hauteur des piles concernées (5 pions ou moins, plus de 5).
- **Hauteur des piles au-delà de 5** : non traitée dans la littérature. Tester si le profil le long de la pointe distingue 6 de 7 pions selon l'angle de caméra.
- **Disponibilité de requestVideoFrameCallback et de WebGPU dans WebKitGTK et WKWebView embarqué** : non vérifiée en primaire. Faire un test de fonctionnalités (feature detection) dans les trois binaires Wails.
- **Applications mobiles** : leurs performances sont inconnues. Un banc d'essai sur 20 photos de stream permettrait de les situer.
- **Dates de certaines sources** (articles de reconnaissance de dés de type casino, blog Golsteyn) : non vérifiées. Les chiffres de ChessQueries et de l'article Faster R-CNN (2026) sont des préprints arXiv non relus par des pairs.
- **Corpus** : les sources primaires sont Chesscog, LiveChess2FEN, ChessReD, CVChess, ChessQueries, Marom, Roman (Brno) et la documentation WebKit / ONNX Runtime / caniuse / bugzilla. Les sources secondaires sont Hackster, testmuai et les fiches d'applications. Relèvent du blog Golsteyn, Poggi, Medium nitzankoll et les dépôts README sans évaluation.

## Sources

1. [http://excel.fit.vutbr.cz Digitalization and visualization of chess games from](https://excel.fit.vutbr.cz/submissions/2025/075/75.pdf)
2. [Chess move tracking using overhead RGB webcam](https://www.semanticscholar.org/paper/Chess-move-tracking-using-overhead-RGB-webcam-Wang-Green/f86ae2293690e875798fa1c0b9c2018f9228a959)
3. [Computer vision system for the chess game reconstruction](https://matijapiskorec.github.io/static/pdf/Computer%20vision%20system%20for%20the%20chess%20game%20reconstruction.pdf)
4. [VICE: View-Invariant Chess Estimation Kevin Zhu](https://www.scitepress.org/Papers/2023/121672/121672.pdf)
5. [GitHub - oliverfrost1/chess-video-move-detection: Deep learning project to automatically detect chess moves from a video input using YOLOv11. · GitHub](https://github.com/oliverfrost1/chess-video-move-detection)
6. <https://arxiv.org/pdf/2501.05030>
7. [GammonSnap Backgammon Analyzer](https://play.google.com/store/apps/details?id=com.gammonsnap&hl=en_ZA)
8. [Backgammon Eye - Games App](https://mwm.ai/apps/backgammon-eye/6744810330)
9. [GammonSnap](https://www.gammonsnap.de/)
10. [Backgammon Eye - App Store - Apple](https://apps.apple.com/dk/app/backgammon-eye/id6744810330)
11. [Sage Snap — Photograph a backgammon position, see the best play](https://snap.bgsage.ai/)
12. [Web platform features explorer - requestVideoFrameCallback()](https://web-platform-dx.github.io/web-features-explorer/features/request-video-frame-callback/)
13. [Linux desktop: WebKitWebProcess crashes when a screen share starts; the window stays grey while the call goes on · Issue #824 · Zeitgleich/TeamCraft · GitHub](https://github.com/Zeitgleich/TeamCraft/issues/824)
14. [web-platform-dx.github.io](https://web-platform-dx.github.io/web-features-explorer/features/webcodecs)
15. [WebCodecs: Browser Support, Features, Use Cases](https://www.testmuai.com/learning-hub/webcodecs-browser-support/)
16. [WebKit Features in Safari 26.0](https://webkit.org/blog/17333/webkit-features-in-safari-26-0/)
17. [Quentin Golsteyn](https://golsteyn.com/writing/dice/)
18. [Automated Counting Dice](http://www.roborealm.com/tutorial/Counting_Dice/index.php)
19. [Image Identification Scheme for Dice Game](https://www.researchgate.net/publication/229039403_Image_Identification_Scheme_for_Dice_Game)
20. [Dice Image Recognition Scheme Using Pattern Comparison Technique](https://www.researchgate.net/publication/239763662_Dice_Image_Recognition_Scheme_Using_Pattern_Comparison_Technique)
21. [(PDF) Multivariate dice recognition using invariant features](https://www.researchgate.net/publication/258808321_Multivariate_dice_recognition_using_invariant_features)
22. [Dice recognition device and method of recognizing dice](https://image-ppubs.uspto.gov/dirsearch-public/print/downloadPdf/10828556)
23. [Live action craps table with optical hand and dice tracking](https://image-ppubs.uspto.gov/dirsearch-public/print/downloadPdf/12676051)
24. [GitHub - nell-byler/dice\_detection: Project to detect and classify six-sided dice from images using deep learning. · GitHub](https://github.com/nell-byler/dice_detection)
25. [GitHub - debuglevel/tensorflow-dice-detection: Project to detect and classify six-sided dice from images using deep learning. \[does not really work well on new data\] · GitHub](https://github.com/debuglevel/tensorflow-dice-detection)
26. [Using Machine Learning to Interpret Dice Rolls - TopSCHOLAR](https://digitalcommons.wku.edu/cgi/viewcontent.cgi?article=1004&context=seas_faculty_pubs)
27. [github.com](https://github.com/buehlpa/Dice)
28. [GitHub - nomi30701/yolo-multi-task-onnxruntime-web: Yolo11 multi task browser, Object Detection, Pose Estimation, Instance Segmentation Power by onnx-web. support WebGPU wasm(cpu). Realtime process camera, Add your custom model · GitHub](https://github.com/nomi30701/yolo-multi-task-onnxruntime-web)
29. [Dice: d4, d6, d8, d10, d12, d20 Images](https://www.kaggle.com/datasets/ucffool/dice-d4-d6-d8-d10-d12-d20-images)
30. [GitHub - Kishaan/Dice-Detection: Computer Vision project in which traditional computer vision techniques are used to determine the location and numbers of pips on the top side of the dice from a video.](https://github.com/Kishaan/Dice-Detection)
31. [D6 Dice - Images](https://www.kaggle.com/datasets/koryakinp/d6-dices-images)
32. [D6 Dice - Videos](https://www.kaggle.com/datasets/koryakinp/d6-dice-videos)
33. [Dice Detection Upper View](https://www.kaggle.com/datasets/nomihsa965/dice-detection-upper-view)
34. [Dataset Viewer](https://huggingface.co/datasets/UE-APC-INRAE/deepvisiontools-demo-datasets)
35. [Dice Object Detection Dataset by Roboflow](https://universe.roboflow.com/roboflow-gw7yv/dice)
36. [Dice Object Detection Dataset by DICEDATASET](https://universe.roboflow.com/dicedataset/dice-txtg5)
37. [Dice Detection Computer Vision Dataset - yolo](https://universe.roboflow.com/yolo-hkw8z/dice-detection-3rsln)
38. [backgammon · GitHub Topics · GitHub](https://github.com/topics/backgammon?o=desc&s=forks)
39. [GitHub - christiancorro/BackgammonCV: Automatic recognition of the Backgammon game board.](https://github.com/christiancorro/BackgammonCV)
40. [How I Used Machine Learning to Beat My Friends at Backgammon](https://medium.com/@nitzankolatacz/how-i-used-machine-learning-to-beat-my-friends-at-backgammon-fb541ec1c0e5)
41. [GitHub - nitzankoll/backGammon-analyzer: A Python-based backgammon game analyzer leveraging GNU Backgammon and YOLOv5 for game state detection and analysis.](https://github.com/nitzankoll/backGammon-analyzer)
42. [GitHub - jvitors23/backgammon-checker-detection-openCV: Backgammon checker detection openCV.](https://github.com/JvitorS23/backgammon-checker-detection-openCV)
43. [Backgammon Object Detection Model by boardgamedetection](https://universe.roboflow.com/boardgamedetection/backgammon-eofws)
44. [Backgammon Object Detection Model (v6, 2023-10-12 11:56am) by boardgamedetection](https://universe.roboflow.com/boardgamedetection/backgammon-eofws/dataset/6)
45. [Maker Faire](https://makerfaire.com/yearbook/2024-projects/?sf_paged=36)
46. <https://www.hackster.io/news/alon-borenshtein-s-arduino-powered-botgammon-robot-plays-a-mean-game-of-backgammon-b054b5ef4477>
47. [Alon Borenshtein - General Motors](https://www.linkedin.com/in/alon-borenshtein-13859a8/)
48. [Backgammon game logging via Computer Vision - YouTube](https://www.youtube.com/watch?v=4-uQVnv0d44)
49. [GitHub - NadavBenda/robotic-backgammon: Autonomous robotic backgammon system powered by CoreXY mechanism, ESP32, and Jetson AI computer vision.](https://github.com/NadavBenda/robotic-backgammon)
50. [ArkadiumBackgammon / README.md](https://huggingface.co/datasets/ArkadiumInc/ArkadiumBackgammon/blob/main/README.md)
51. [GammonSnap Backgammon Analyzer - Free APK ...](https://www.appbrain.com/app/gammonsnap-backgammon-analyzer/com.gammonsnap)
52. [GammonSnap Backgammon Analyzer - Apps on Google Play](https://play.google.com/store/apps/details?id=com.gammonsnap&hl=en)
53. [Backgammon Eye - Apps on Google Play](https://play.google.com/store/apps/details?id=com.bgar&hl=en)
54. [Backgammon Sage Pro](https://sage.bgsage.ai/game-analysis)
55. [Trackgammon - Backgammon Match Transcription](https://www.backgammongalaxy.com/trackgammon)
56. [GitHub - davidmallasen/LiveChess2FEN: Predict live chess games into FEN notation. · GitHub](https://github.com/davidmallasen/LiveChess2FEN)
57. [\[PDF\] LiveChess2FEN: a Framework for Classifying Chess Pieces based on CNNs](https://www.semanticscholar.org/paper/LiveChess2FEN:-a-Framework-for-Classifying-Chess-on-Quintana-Garc%C3%ADa/382d83755eb85ad6b60108c9d417eab70abdecf8)
58. [ChessQueries: Toward Better Chess Board Recognition](https://arxiv.org/html/2608.30762v1)
59. [Chess Position Recognition using Faster R-CNN](https://arxiv.org/html/2610.09191v1)
60. [Simple computer vision system for chess playing robot ...](https://people.etf.unsa.ba/~esokic/mypapers/2008ISSPIT.pdf)
61. [Automatic chess move recognition - Alessio Poggi Portfolio](https://www.alessiopoggi.org/blog/acmr/)
62. [WebKitGTK+ 2.48 highlights - The WebKitGTK Project](https://webkitgtk.org/2025/04/08/webkitgtk-2.48.html)
63. [WebKit Features for Safari 26.4](https://webkit.org/blog/17862/webkit-features-for-safari-26-4/)
64. [WebGPU](https://en.wikipedia.org/wiki/WebGPU)
65. [WebGPU: Browser Support, Features, Limitations](https://www.testmuai.com/learning-hub/webgpu-browser-support/)
66. [Using the WebGPU Execution Provider](https://onnxruntime.ai/docs/tutorials/web/ep-webgpu.html)
67. [1800882 - (interop-2024-requestVideoFrameCallback) Implement HTMLVideoElement.requestVideoFrameCallback()](https://bugzilla.mozilla.org/show_bug.cgi?id=1800882)
