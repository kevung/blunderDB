Recherche du 2026-10-30 — claude.ai, recherche approfondie.
Prompt : ../prompts/03-detection-evenements.md

---

# Pré-passe d'événements pour la transcription vidéo de backgammon : revue de littérature et architecture recommandée

La meilleure approche pour cette pré-passe est une architecture sans modèle appris. On calcule par ROI des descripteurs d'énergie de changement, normalisés en gain et rendus robustes à l'éclairage par une transformée census. Une machine à états avec hystérésis et durées minimales les convertit en événements. Le flux optique dense est à écarter. Je n'ai trouvé aucune publication sur la détection d'événements dans des vidéos de backgammon. Les seuils proposés ici sont donc transposés de la soustraction de fond (ViBe, MOG2), du suivi d'échecs et de go, et des métriques de détection d'événements sonores et sportifs. Il faudra les valider sur vos propres matchs.

## TL;DR

- **Architecture** : 4 descripteurs par image, calculés à 10 i/s sur des `Uint8Array` : énergie de différence normalisée dans la ROI dés, fraction d'avant-plan dans la ROI plateau, indicateur de changement global (exposition, coupure) et distance de Hamming census au gabarit de départ. Une machine à états hiérarchique (mains / dés / partie) les exploite avec hystérésis et durées minimales. Lucas-Kanade, Farnebäck et les TCN n'apportent rien à ce stade : on peut s'en passer.
- **Point de départ** : seuil haut à environ 3σ du bruit et seuil bas à environ 1,5σ, estimés en continu ; *repos* = 5 images consécutives sous le seuil bas (0,5 s à 10 i/s) ; *mains hors* = 3 images ; *remise à zéro* = gabarit validé pendant au moins 2 s après la sortie des mains. Ces chiffres sont des hypothèses d'ingénierie fondées par analogie (ViBe R=20, MOG2 τ=0,5, « hysteretic n-frame thresholding » aux échecs). Aucun n'a été validé en littérature sur ce cas précis.
- **Pannes à surveiller** : (1) faux *lancers* dus à l'auto-exposition, aux overlays animés et aux coupures de caméra du stream ; (2) *repos* manqué parce que la main reste au-dessus des dés ou les ramasse aussitôt ; (3) *remise à zéro* fausse ou tardive quand les pions sont replacés en désordre. Pour l'évaluation, utiliser un F1 par événement avec tolérance (collar), comme en détection d'événements sonores (Mesaros et al., 2016), plus le décalage médian.

## Key Findings

1. **Rien sur le backgammon.** Aucun article à DOI ou arXiv n'a été trouvé sur la détection d'événements ou la transcription automatique de vidéos de backgammon (2015–2026). Les seuls systèmes dédiés repérés sont commerciaux. Digigammon, de Nekodu Technology (Digitimer + Transcriptor), utilise une webcam zénithale sous Windows, détecte les pions, les dés et le videau, puis transcrit via GNU Backgammon. Son site revendique « almost-real-time match transcription with 100% accuracy », ce qui relève du marketing et n'est ni publié ni vérifiable ; selon le forum bgonline.org (20 juin 2023), la licence coûte 1 000 $ et exige une tablette Windows tactile. TrackGammon (trackgammon.com, partenaire de Backgammon Galaxy) transcrit lui aussi des vidéos en fichiers MAT.
2. **Les systèmes de jeux de table voisins utilisent tous un portillon de mouvement.** Aux échecs, Wang & Green (IVCNZ 2013) écartent les images où un joueur masque le plateau par « hysteretic n-frame thresholding of difference images ».\[1\] Le nombre N, les seuils, la cadence et la précision sont absents du résumé, et le texte intégral n'a pas été obtenu : non trouvés. Des projets récents remplacent ce portillon par un détecteur de mains appris, ce qui est exclu sans modèle : le dépôt GitHub oliverfrost1/chess-video-move-detection utilise un « YOLOv11l model to detect hands and exclude frames with interference ». Au go, Chih-Lin Lin, Hsia-Hung Ou, Lung Hung Chen et Chih-En Kuo (Engineering Applications of AI, déc. 2025) annoncent 82,56 % de vidéos entièrement correctes sur 86 vidéos de tournoi (356 h), selon un protocole sans tolérance d'erreur où « a video is counted correctly only if all sampled frames match the reference », et une réduction de 70,23 % des erreurs liées aux occultations par les mains, grâce à un raisonnement de cohérence temporelle.
3. **La cadence d'échantillonnage est un risque documenté.** TCG-AR (Cioppa, Verdonck, Henry, Van Droogenbroeck et La Rocca, Université de Liège, arXiv:2607.02090) indique au §7.5 que sa boucle de reconnaissance « refreshes at roughly four frames per second », de sorte que la pose d'une carte et le passage de la main « can fall entirely between two recognition frames » ; le même article mentionne toutefois environ 10 i/s sur GPU (§5.3). Votre plage de 5–10 i/s est donc un minimum. Visez 10 i/s dans les ROI.
4. **La robustesse à l'éclairage est la difficulté principale de la soustraction de fond simple.** CDnet 2014 (11 catégories, 53 vidéos) conclut que la détection de changement reste fragile face aux ombres, à la faible lumière et aux mouvements de caméra, et qu'aucune méthode ne domine partout.\[2\]\[3\] Les descripteurs binaires locaux (census/LBP, LBSP de LOBSTER/SuBSENSE) sont invariants aux variations monotones d'intensité, donc tolérants à l'auto-exposition. Les différences d'intensité brutes ne le sont pas.\[4\]\[5\]\[6\]
5. **Côté plateforme, le décodage dans la webview est faisable partout, avec des réserves sur Linux.** `requestVideoFrameCallback` est disponible dans Chrome 83, Safari 15.4 et Firefox 132 (2024-10-29).\[7\] WebCodecs est disponible dans WebKitGTK 2.44 (2024-03-27), Firefox 130 et Safari 26 (2025-09-15).\[8\]\[9\] Sous WebKitGTK, les codecs dépendent des plugins GStreamer installés.\[10\] Un `OffscreenCanvas` WebGL en worker n'existe sous Safari qu'à partir de 17.0, et seulement sur macOS Sonoma ou plus récent : pas sur Ventura (WebKit bug 183720).\[11\]\[12\]\[13\]

## Details

### 0. Chaîne d'acquisition (contrainte « tout dans la webview »)

- **Extraction des images.** Lire le `<video>` local et avancer par `currentTime` (seek) ou en lecture accélérée avec `requestVideoFrameCallback`. Dessiner chaque image dans un canvas réduit à 320×180 ou aux ROI, lire les pixels avec `getImageData`, puis transférer l'`ArrayBuffer` au Worker. web.dev précise que rVFC est « best effort » (« if things seem one frame off ») : utilisez `metadata.mediaTime` comme horodatage, jamais l'heure murale.\[14\]
- **Variante WebCodecs.** On peut décoder les images directement dans le Worker avec `VideoDecoder`, mais il faut alors démultiplexer le MP4 en JS. C'est plus rapide en pré-passe, mais les codecs disponibles sous WebKitGTK dépendent de GStreamer : d'après le README video-hfp (2026), VP9 est toujours présent alors que H.264 exige gstreamer1.0-libav ou VA-API.\[10\] Gardez le chemin `<video>` + canvas comme référence et WebCodecs comme accélération optionnelle détectée à l'exécution.
- **Coût.** 320×180 = 57 600 pixels. Une différence d'images avec seuillage coûte quelques opérations par pixel, soit moins d'un million d'opérations par image : négligeable en JS à 10 i/s. Ce chiffre est un calcul d'ordre de grandeur, pas une mesure publiée.

### 1. Détection de mouvement sans modèle : comparaison

| Méthode | Principe | Auto-exposition / scintillement | Overlays animés | Ombres de mains | Coût JS 320×180 | Verdict |
|---|---|---|---|---|---|---|
| Différence d'images consécutives | \|I_t − I_{t−1}\| > T | Mauvais : un saut de gain allume toute la ROI | Mauvais si l'overlay est dans la ROI | Les ombres en mouvement comptent comme mouvement | Minimal | Bon pour l'**énergie dés** (rouler / immobile) si l'on normalise le gain |
| Moyenne glissante de fond (B ← (1−α)B + αI) | Écart au fond lent | Mauvais à moyen : la dérive lente est absorbée, le saut brutal non | Mauvais | Idem | Minimal | Bon pour la **présence des mains** dans le plateau |
| MOG2 simplifié (Zivkovic 2004) | Mélange de gaussiennes par pixel, test d'ombre | Moyen | Moyen (l'overlay périodique finit dans le fond) | **Test d'ombre explicite** : pixel plus sombre que le fond, avec τ=0,5 (au-delà de 2× plus sombre, ce n'est pas une ombre) | ~10× la moyenne glissante | Utile surtout pour récupérer le test d'ombre\[15\] |
| ViBe (Barnich & Van Droogenbroeck 2011) | 20 échantillons par pixel, R=20 (niveaux de gris), 2 correspondances, mise à jour aléatoire 1/16\[16\] | Moyen | Bon (le fond multimodal absorbe les animations répétitives) | Moyen | Faible (1 comparaison par échantillon) | Bonne alternative à MOG, simple à coder |
| Census / LBP / LBSP | Code binaire des comparaisons locales, distance de Hamming | **Bon** : invariant aux transformations monotones de l'intensité | Moyen | Moyen à bon (une ombre douce change peu l'ordre local) | ~8 comparaisons par pixel | **Recommandé** pour le plateau et le gabarit\[6\]\[17\] |

Recommandations concrètes :
- **Normalisation du gain.** Avant toute différence, divisez chaque ROI par sa moyenne (ou appariez les histogrammes) et mesurez le gain global sur une *ROI témoin* hors plateau, par exemple un bout de table ou de décor. Si plus de 60 % des pixels de toutes les ROI changent en même temps, il s'agit d'un **changement global** (exposition, coupure, fondu) : produisez *incertain* et réinitialisez les fonds. Ce seuil de 60 % est une hypothèse à calibrer, pas une valeur publiée.
- **Seuils adaptatifs.** Estimez le bruit σ de chaque ROI par la médiane des écarts absolus (MAD) de la différence d'images sur les périodes calmes. Seuil pixel ≈ 2,5–3σ, ce qui équivaut grosso modo à R=20 de ViBe pour un bruit de 6–8 niveaux. Calculez l'énergie de ROI E = fraction de pixels au-dessus du seuil, avec un seuil haut E_hi (≈ 3× la médiane calme) et un seuil bas E_lo (≈ 1,5×). Cette transposition est une hypothèse d'ingénierie de ma part.
- **Overlays.** Faites masquer une fois par vidéo, lors de la calibration, les rectangles d'overlay (score, horloge, logos) qui recoupent une ROI. Un pixel qui « bouge » plus de 30 % du temps sur la première minute est candidat au masquage automatique (hypothèse).
- **Scintillement secteur 50/60 Hz.** À 5–10 i/s, il se replie (aliasing) en oscillation lente de la luminance moyenne. La normalisation par la moyenne de ROI et la census le neutralisent en grande partie. Non trouvé : aucune étude quantitative sur ce point.

### 2. Flux optique : inutile pour la pré-passe

- Farnebäck (2003) calcule un flux dense par expansion polynomiale sur une pyramide. OpenCV le présente comme plus précis mais nettement plus coûteux que Lucas-Kanade creux. Le portage FPGA de 2019 (ACM) souligne sa « high computation complexity ».\[18\]\[19\] Il n'existe pas d'implémentation JS pure maintenue : OpenCV.js est du WASM, donc hors contrainte « sans bibliothèque ». Je n'ai trouvé aucun benchmark JS à 320×180.
- Lucas-Kanade pyramidal existe en JS pur (jsfeat, `optical_flow_lk.track`), mais il suit des points épars.\[20\] Les dés qui roulent sont petits, flous et à texture répétitive (les pips), c'est-à-dire le pire cas pour LK.
- **Ce que le flux apporterait** : la *direction* du lancer (pour savoir qui a lancé) et la distinction main / dé. Une **mise en correspondance de blocs (SAD) sur des blocs 8×8 dans la seule ROI dés**, avec une recherche ±4 px sur l'image réduite, suffit pour cela. Pour une ROI de 80×60 px, cela fait 75 blocs × 81 positions × 64 pixels ≈ 0,4 M opérations par image (calcul, pas mesure). À réserver à la question « qui a lancé » ; la détection d'événements n'en a pas besoin.

### 3. Segmentation temporelle dans les vidéos de jeux de table

- **Échecs.** Victor Wang et Richard D. Green (IVCNZ 2013, p. 299–304) : portillon de mouvement à hystérésis sur N images ; selon leur résumé, « a move is recognized when the current occupancy grid matches the chessboard that results from one of the possible moves from the previous occupancy grid », ces coups et grilles étant « supplied by a chess engine ». C'est exactement votre schéma (moteur de backgammon et coups classés). VICE (2023) relève qu'un joueur peut entamer son coup alors que le bras adverse occulte encore le plateau : il n'y a pas toujours d'image « plateau libre » entre deux coups.\[21\] ChessQueries (arXiv, 2026) publie SLCC, 2 174 images de retransmissions du Saint Louis Chess Club, et note que les erreurs se concentrent sous les fortes occultations.\[22\]
- **Go.** VideoKifu (Carta & Corsolini, arXiv 2018) traite une vidéo non surveillée et décrit le piège des pierres capturées qui ne sont pas retirées tout de suite : sans supervision, le système les prend pour un nouveau coup et corrompt toute la suite.\[23\]\[24\] L'équivalent au backgammon serait un pion frappé mal posé sur la barre. Lin et al. (2025) : 82,56 % au niveau vidéo, 377,5 i/s pour la reconnaissance d'état du plateau, F1 0,988, et −70,23 % d'erreurs liées aux occultations.\[25\] Le fonctionnement détaillé de l'algorithme d'occultation n'a pas été trouvé (texte intégral inaccessible).
- **Cartes et casino.** DeepGamble (arXiv 2020, blackjack) déclenche un vote majoritaire seulement au **changement d'état**.\[26\] Le projet poker de CS231n (Stanford, 2024 ; travail étudiant, fiabilité limitée) suit un automate d'état de partie.\[27\] Un brevet de reconnaissance d'activité sur table de jeu (US 10 956 750 / 11 580 746) horodate un événement quand un objet apparaît dans une ROI prédéfinie : c'est le même principe ROI → événement.\[28\]\[29\]
- **Dés.** Les systèmes de Sic Bo (D-i-Shaker de TCS John Huxley, brevets japonais) lisent les dés en environnement contrôlé, avec caméra et éclairage intégrés au gobelet.\[30\]\[31\]\[32\] Le brevet WO2017140385A1 mentionne une « gesture recognition » pour distinguer une main des dés dans la zone de lancer.\[33\] Hsu et al. utilisent plusieurs caméras en conditions non contrôlées.\[34\]\[35\] Le dépôt Kishaan/Dice-Detection lit les dés « after they settle down on the tray », avec des mains piégeuses et des dés transparents.\[36\] Aucune de ces sources ne publie de critère de repos chiffré : non trouvé.
- **HMM et TCN.** Aucun HMM ni TCN appliqué à un jeu de plateau n'a été trouvé. En segmentation d'actions générique, MS-TCN (CVPR 2019) atteint F1@10 = 76,3, Edit = 67,9 et Acc = 80,7 sur 50Salads ; MS-TCN++ (2020) atteint 80,7 / 74,3 / 83,7.\[37\] Les auteurs montrent qu'un modèle à un seul étage produit beaucoup de sur-segmentation, ce qui illustre pourquoi un lissage temporel (durées minimales) est indispensable.\[38\] Un TCN exigerait des centaines de séquences annotées. **Conclusion** : machine à états maintenant ; un HMM à durées explicites, ou un petit TCN sur vos 4–6 descripteurs, n'aurait de sens qu'une fois le jeu de test constitué et si la machine à états plafonne.

### Machine à états recommandée

Trois automates couplés tournent à 10 i/s. Toutes les durées sont exprimées en images à 10 i/s ; à 5 i/s, divisez-les par deux avec un plancher de 2.

**A. Mains (ROI plateau, fraction d'avant-plan F_b ; fond = moyenne glissante α≈0,02 mise à jour seulement hors mains, ou ViBe-lite ; pixel = census-Hamming ≥ 2 ou écart normalisé > 3σ, avec test d'ombre MOG2 : rapport I/B dans [0,5 ; 0,95] et chroma stable → ombre, non comptée)**
- HORS → DEDANS si F_b > F_hi pendant ≥ 2 images (0,2 s). Émettre *mains dans le plateau* à la première image au-dessus du seuil (horodatage rétroactif).
- DEDANS → HORS si F_b < F_lo pendant ≥ 3 images (0,3 s). Émettre *mains hors du plateau* à la première image sous le seuil.
- Après HORS, **figer le fond** (photo de position) pour la reconnaissance du coup.

**B. Dés (ROI dés, énergie de différence normalisée E_d ; on combine avec A, car une main au-dessus de la ROI dés est détectée par l'avant-plan de la ROI dés élargie)**
- CALME → LANCER si E_d > E_hi pendant ≥ 1 image et qu'il n'y a pas de changement global. Émettre *lancer*.
- LANCER → STABILISATION dès que E_d < E_lo.
- STABILISATION → REPOS si E_d < E_lo pendant **N_repos = 5 images (0,5 s)** **et** que l'avant-plan de la ROI dés est compact, c'est-à-dire 2 blobs de la taille d'un dé, sans grand blob « main ». Émettre *repos* et conserver l'image médiane des 5 pour la lecture des dés.
- Retour à LANCER si E_d > E_hi pendant STABILISATION (dé qui repart) ; *repos* n'est pas émis.
- Délai de garde : LANCER sans REPOS pendant 4 s → *incertain* (dés ramassés, hors ROI ou masqués).

**C. Partie**
- Après chaque *mains hors* et à chaque *repos*, calculer la distance census de la ROI plateau rectifiée au gabarit de position initiale.
- CANDIDAT_RAZ si la distance est < D_raz sur ≥ 80 % des 24 flèches × 2 côtés, puis RAZ confirmée si l'état reste stable ≥ 20 images (2 s) **et** qu'aucune main n'est dans le plateau. Émettre *remise à zéro* horodatée au début du dernier épisode stable.
- Contrôle de cohérence avec le moteur : une remise à zéro n'est plausible qu'en fin de partie (course terminée, abandon ou videau refusé), ou en tout début de vidéo.

**Signal *incertain*** : changement global, délai de garde des dés, épisode de mains de plus de 20 s (rangement), coupure de plan détectée (distance d'histogramme globale au-dessus du seuil), ou toute transition contradictoire (*repos* sans *lancer*).

### 4. Repos des dés : pièges et parades

| Piège | Symptôme | Parade |
|---|---|---|
| Dé qui roule lentement ou oscille | E_d frôle E_lo | Exiger 5 images consécutives ; mesurer aussi le déplacement du centroïde des blobs (< 1 px) |
| Main qui reste au-dessus | E_d bas (main immobile) mais grand blob d'avant-plan | Test de compacité : blob > 3× la surface d'un dé → pas de repos |
| Dés ramassés immédiatement | REPOS très court, puis LANCER/mains | Émettre quand même *repos* si 3 images stables ou plus, marqué *incertain* ; garder la meilleure image pour la lecture |
| Dés hors ROI (sur la barre, sur un pion, sur la table) | Le délai de garde expire | Prévoir une ROI dés « large » = toute la moitié du plateau ; en dernier recours, *incertain* et saisie manuelle |
| Dés dans le gobelet (lancer refusé, « cocked dice ») | Mouvement du gobelet sans dés visibles | Les dés ne sont pas visibles, il n'y a rien à lire : *incertain* |
| Dés cassés ou non à plat | Dé incliné contre un pion | Hors du périmètre du détecteur de mouvement : à signaler au lecteur de dés |

La durée de 0,5 s n'est pas tirée de la littérature : aucun critère chiffré n'a été trouvé. Elle se justifie par le fait qu'à 10 i/s, 5 images sous le seuil bas filtrent un dé qui oscille encore, tout en restant bien plus court que le délai typique avant que le joueur ne déplace ses pions. Ce dernier point est à mesurer sur votre corpus.

### 5. Remise à zéro par gabarit

- **Homographie.** Calibrée une fois via les 4 coins du plateau, comme le décrivent ChessQueries et VICE pour l'échiquier. Rectifier en image canonique, par exemple 240×160, puis découper les 24 flèches et la barre.
- **Descripteur.** Census par cellule, distance de Hamming au gabarit appris sur la position de départ observée dans la même vidéo, ce qui absorbe la couleur des pions et du plateau. À défaut, compter les pions par flèche avec un profil d'intensité le long de la flèche.
- **Rangement en désordre.** Pendant le replacement des pions (plusieurs secondes, mains fréquentes), l'automate A reste DEDANS ou oscille. La règle « stable ≥ 2 s sans main » évite de valider un état intermédiaire. Ajoutez la contrainte de **monotonie** : la distance au gabarit doit décroître sur l'épisode, puis rester basse. Si l'épisode dure plus de 20 s, émettre *incertain* plutôt qu'une RAZ datée au hasard.
- Le risque de faux positif est faible, car la position initiale est très improbable en cours de partie. Le risque de faux négatif est plus élevé : pions de travers, plateau déplacé pendant le rangement, d'où l'intérêt de recalculer l'homographie si la distance globale reste élevée alors qu'il n'y a pas de mains.

### 6. Qui a lancé

- Non trouvé : aucune étude sur la fiabilité de la règle « côté de chute = lanceur ». La règle officielle impose de lancer dans la moitié droite de son propre côté. Elle n'est pas toujours respectée en partie amicale ou en stream.
- **Recommandation.** Traiter la règle ROI gauche / droite comme un **a priori faible**, combiné à : (a) l'alternance des tours, a priori fort, avec exceptions documentées (début de partie, après une remise à zéro) ; (b) le côté d'entrée de la main dans la ROI dés, via la mise en correspondance de blocs à la première image du *lancer* ; (c) surtout, la **couleur des pions déplacés** dans le coup reconnu ensuite, qui tranche sans ambiguïté. Quand (a) et la ROI divergent, émettre *incertain* plutôt que choisir.
- Si les deux joueurs lancent dans la même moitié, la règle ROI est inutilisable : on se repose sur (a) et (c).

### 7. Évaluation

- **Métrique principale : F1 par événement avec tolérance.** Un événement prédit est un vrai positif s'il tombe dans ±δ d'un événement de référence de même type, avec appariement un à un (glouton ou hongrois). La détection d'événements sonores utilise un collar de 200 ms sur l'onset (Mesaros, Heittola & Virtanen, *Applied Sciences* 6(6):162, 2016 ; DCASE 2017–2019).\[39\]\[40\] L'action spotting sportif utilise une Average-mAP « tight » sur des tolérances de 1 à 5 s (SoccerNet-v2, 2021).\[41\]
- **Tolérances proposées** (à 10 i/s) : *lancer* et *repos* ±0,5 s ; *mains dans / hors* ±0,5 s ; *remise à zéro* ±3 s ; *incertain* évalué comme taux de couverture des erreurs (part des erreurs signalées comme incertaines). Rapportez aussi les courbes F1 en fonction de δ (0,2 / 0,5 / 1 / 2 s), à la manière de SoccerNet.
- **Métriques secondaires** : décalage médian signé et intervalle interquartile par type, pour détecter un biais systématique comme un *repos* toujours en retard de 0,3 s ; score Edit (Levenshtein sur la séquence d'événements), qui mesure la sur-segmentation comme en segmentation d'actions (MS-TCN) ; et, métrique produit, **le taux de coups pour lesquels l'utilisateur n'a rien eu à corriger**.
- **Protocole d'annotation.**
  1. Choisir 3 à 5 matchs déjà transcrits et horodatés, variés en caméra, éclairage, overlay et gobelet, et en tirer environ 2 h de vidéo (de l'ordre de 300 à 500 coups).
  2. Pré-remplir la ligne de temps à partir des horodatages de coups existants, qui donnent la position approximative de chaque *lancer* et de chaque coup.
  3. Faire affiner par un annotateur, au pas d'image à 10 i/s, dans un outil maison qui affiche la vidéo et les évènements : début du *lancer* = première image où un dé quitte la main ; *repos* = première image où les deux dés sont immobiles ; *mains dans / hors* = première et dernière image où une main recoupe le plateau ; *remise à zéro* = première image de la position initiale complète sans main.
  4. Faire annoter de façon indépendante 10 % du corpus par un second annotateur, et mesurer l'accord inter-annotateurs : F1 entre annotateurs à ±0,2 s et écart médian. Cet accord donne le plafond atteignable et fixe la tolérance δ réaliste.
  5. Séparer validation (réglage des seuils) et test (rapport final) **par match**, jamais par extrait du même match, pour la même raison que l'exigence d'évaluation indépendante de la scène soulevée pour CDnet.\[42\]

## Recommendations

1. **Verdict.** Une pré-passe sans modèle est réaliste et suffisante pour un usage « confirmer ou corriger ». Elle reproduit le schéma des suiveurs d'échecs et de go (portillon de mouvement, puis comparaison d'état contrainte par le moteur de jeu), qui reste la référence chaque fois qu'on n'a pas de détecteur de mains appris. Le flux optique dense et les modèles temporels appris ne se justifient pas au premier livrable.
2. **Recommandation technique.**
   - Pipeline : `<video>` + rVFC/seek → canvas réduit aux ROI → Worker → descripteurs `{E_d, F_b, G_global, D_raz, (bloc-SAD direction)}` → 3 automates → ligne de temps JSON `{t, type, confiance, raison}`.
   - Valeurs de départ : seuil pixel 3σ (MAD), E_hi = 3 × la médiane calme, E_lo = 1,5 ×, fond α = 0,02, test d'ombre τ = 0,5 ; *lancer* ≥ 1 image, *repos* 5 images, *mains dans* 2 images, *mains hors* 3 images, RAZ stable 20 images, délai de garde dés 4 s, épisode mains > 20 s → *incertain*.
   - Calibration par vidéo : 4 coins du plateau, ROI dés gauche et droite, ROI témoin, masques d'overlay. Apprendre le gabarit de départ sur la première position de la vidéo.
   - Enregistrer tous les descripteurs par image (quelques Ko par minute) pour pouvoir rejouer et régler les seuils hors ligne sans redécoder.
3. **Les trois pannes à surveiller.**
   - **Changements globaux** (auto-exposition, balance des blancs, overlay animé, coupure, replay) → faux *lancers* et faux *mains dans*. Indicateur : taux de *lancer* sans *repos* et sans coup ensuite.
   - **Repos manqué ou prématuré** (main au-dessus, ramassage immédiat, dé hors ROI). Indicateur : proportion de *lancer* suivis du délai de garde.
   - **RAZ pendant le rangement** (validation trop tôt ou jamais). Indicateur : écart entre l'horodatage de RAZ et le premier *lancer* de la nouvelle partie.

## Caveats

- **Seuils sans source directe.** Aucun chiffre de seuil ou de durée n'a été trouvé pour le backgammon, ni même pour les échecs (paramètres de Wang & Green non trouvés, texte intégral inaccessible). Les valeurs proposées sont des transpositions argumentées (ViBe R=20 et φ=16 ; MOG2 τ=0,5 ; collars DCASE de 200 ms ; tolérances SoccerNet de 1 à 5 s), pas des résultats.
- **Sources secondaires ou commerciales.** Digigammon (100 % revendiqués) et les fiches Sic Bo sont commerciales. Le projet poker CS231n est étudiant. Le support des codecs WebKitGTK provient d'un README tiers (video-hfp, 2026, « untested » sur WebKitGTK). La date de support de WebCodecs dans Safari diffère selon les sources : Remotion (mai 2025) indique un VideoDecoder déjà présent dans Safari, alors que web-features date le support complet de Safari 26 (2025-09-15).\[9\]\[43\] Les deux sont compatibles si l'on distingue VideoDecoder seul (Safari 16.4+) et l'API complète.
- **Expérience à mener.**
  1. Annoter 2 matchs (environ 1 h) selon le protocole.
  2. Faire tourner la machine à états avec les valeurs de départ.
  3. Balayer N_repos ∈ {3, 5, 8}, E_hi/E_lo ∈ {2/1,2 ; 3/1,5 ; 4/2} et census ou non, puis tracer F1@0,5 s et le décalage médian par type.
  4. Répéter à 5 et à 10 i/s pour vérifier le risque signalé par TCG-AR.
  5. Mesurer le temps par image dans le Worker sur WebKitGTK, WebView2 et WKWebView (non trouvé dans la littérature).
  
  Critère de succès proposé : F1 ≥ 0,95 pour *lancer* et *repos* à ±0,5 s, et moins de 5 % de coups nécessitant une correction d'événement.

## Sources (corpus et fiabilité)

1. Wang, Green — « Chess move tracking using overhead RGB webcam », IVCNZ 2013, DOI 10.1109/IVCNZ.2013.6727033 — portillon de mouvement « hysteretic n-frame thresholding », coups validés par moteur — primaire (résumé seul consulté).
2. Lin, Ou, Chen, Kuo — « End-to-end Go (Weiqi) game record reconstruction from live broadcast videos », Engineering Applications of AI, 2025 (ScienceDirect S0952197625024868) — 82,56 % au niveau vidéo, −70,23 % d'erreurs d'occultation — primaire (résumé, points clés).
3. Carta, Corsolini — « VideoKifu, or the automatic transcription of a Go game », arXiv:1807.01577, 2018 — transcription vidéo du go, pièges des captures — primaire.
4. Zhu — « VICE: View-Invariant Chess Estimation », SciTePress 2023 — occultation par le bras pendant l'enchaînement des coups — primaire.
5. « ChessQueries: Toward Better Chess Board Recognition », arXiv:2608.30762, 2026 — jeu SLCC (2 174 images de retransmissions), erreurs sous occultation — primaire (préprint).\[44\]
6. « TCG-AR: Real-Time Multi-View AR for Trading Card Game Streaming », arXiv:2607.02090, 2026 — reconnaissance à ~4 i/s qui manque les poses rapides — primaire (préprint).
7. « DeepGamble », arXiv:2012.08011, 2020 — vote majoritaire au changement d'état (blackjack) — primaire.
8. Hung, Moberly, Souliman — « Poker Game State Detection », CS231n Stanford, 2024 — automate d'état de partie — secondaire (travail étudiant).\[27\]
9. Wang, Jodoin, Porikli et al. — « CDnet 2014: An Expanded Change Detection Benchmark Dataset », CVPR Workshops 2014 — 11 catégories, fragilité face aux ombres et à l'éclairage — primaire.
10. Barnich, Van Droogenbroeck — « ViBe: A universal background subtraction algorithm », IEEE TIP 20(6), 2011 — N=20, R=20, φ=16 — primaire.
11. Zivkovic — « Improved adaptive GMM for background subtraction », ICPR 2004, DOI 10.1109/ICPR.2004.1333992, et documentation OpenCV MOG2 (τ=0,5) — primaire.
12. Bilodeau et al. — LBSP, CRV 2013 ; St-Charles et al. — LOBSTER/SuBSENSE — descripteurs binaires robustes à l'éclairage — primaire (via résumés et revue arXiv:1611.09099).\[5\]\[45\]
13. Farnebäck — « Two-Frame Motion Estimation Based on Polynomial Expansion », 2003 ; tutoriel OpenCV.js ; FPGA Farnebäck HLS, ACM 2019 (DOI 10.1145/3289602.3294005) — coût du flux dense — primaire / documentation.
14. jsfeat — bibliothèque JS de vision (LK pyramidal) — dépôt de code.
15. Abu Farha, Gall — « MS-TCN », CVPR 2019, arXiv:1903.01945 ; Li et al. — « MS-TCN++ », arXiv:2006.09220, 2020 — F1@k, Edit, sur-segmentation — primaire.
16. Mesaros, Heittola, Virtanen — « Metrics for Polyphonic Sound Event Detection », Applied Sciences 6(6):162, 2016 ; pages métriques DCASE 2017/2019 — collar de 200 ms — primaire.
17. Giancola et al. — SoccerNet, arXiv:1804.04527, 2018 ; Deliège et al. — SoccerNet-v2, 2021 ; SoccerNet 2023 Challenges, arXiv:2309.06006 — Average-mAP loose (5–60 s) / tight (1–5 s) — primaire.
18. caniuse et web-features explorer — `requestVideoFrameCallback` (Firefox 132, 2024-10-29), WebCodecs (Safari 26, 2025-09-15), OffscreenCanvas WebGL (Safari 17) — documentation éditeur / agrégateur de compatibilité.\[46\]
19. web.dev — « Perform efficient per-video-frame operations » (rVFC « best effort ») — documentation éditeur (Google).
20. WebKitGTK — « What's new in WebKitGTK 2.44 », 2024-03-27 (WebCodecs) ; WebKit bug 183720 (pas de WebGL en worker sous Safari 17.1 sur Ventura) ; WebKit PR 76958 (décodeur AV1 GStreamer, 2026) — primaire.\[11\]
21. Remotion — documentation WebCodecs, mai 2025 — état du support navigateur — blog / documentation tierce.
22. MikeParkerGit/video-hfp — README, 2026 — dépendance des codecs WebKitGTK à GStreamer — dépôt (non testé sur WebKitGTK).
23. Digigammon — site produit — transcription backgammon par webcam zénithale — commercial.
24. Kishaan/Dice-Detection ; WO2017140385A1 (Dice recognition device) ; Hsu et al. — « Multivariate dice recognition using invariant features » ; TCS John Huxley D-i-Shaker — lecture de dés — dépôt / brevet / primaire / commercial.\[33\]\[34\]
25. Brevets US 10 956 750 et 11 580 746 — « automated table game activity recognition » (horodatage d'événement par ROI) — brevet.\[28\]\[29\]

## Sources

1. [Chess move tracking using overhead RGB webcam](https://www.semanticscholar.org/paper/Chess-move-tracking-using-overhead-RGB-webcam-Wang-Green/f86ae2293690e875798fa1c0b9c2018f9228a959)
2. [(PDF) CDnet 2014: An Expanded Change Detection Benchmark Dataset](https://www.academia.edu/21667064/CDnet_2014_An_Expanded_Change_Detection_Benchmark_Dataset)
3. [ZBS: Zero-shot Background Subtraction via Instance-level Background Modeling and Foreground Selection](https://arxiv.org/pdf/2303.14679)
4. [Invariant texture analysis through Local Binary Patterns](https://arxiv.org/html/1111.7271)
5. [On the Role and the Importance of Features for Background Modeling and Foreground Detection](https://arxiv.org/pdf/1611.09099)
6. [Census transform](https://en.wikipedia.org/wiki/Census_transform)
7. [Web platform features explorer - requestVideoFrameCallback()](https://web-platform-dx.github.io/web-features-explorer/features/request-video-frame-callback/)
8. [What's new in WebKitGTK 2.44? - The WebKitGTK Project](https://webkitgtk.org/2024/03/27/webkigit-2.44.html)
9. [web-platform-dx.github.io](https://web-platform-dx.github.io/web-features-explorer/features/webcodecs)
10. [GitHub - MikeParkerGit/video-hfp: High frame precision video for the web: a \<video\>-shaped element on WebCodecs — seamless loops, frame-exact cuts, prepared targets that land on the next display frame — on ordinary files · GitHub](https://github.com/MikeParkerGit/video-hfp)
11. [183720](https://bugs.webkit.org/show_bug.cgi?id=183720)
12. [WebKit Features in Safari 17.0](https://webkit.org/blog/14445/webkit-features-in-safari-17-0/)
13. [Revisit disabled \`OffscreenCanvas\` support for Safari · Issue #150801 · flutter/flutter · GitHub](https://github.com/flutter/flutter/issues/150801)
14. [Perform efficient per-video-frame operations | Articles | web.dev](https://web.dev/articles/requestvideoframecallback-rvfc)
15. [Motion Analysis and Object Tracking — OpenCV 2.4.13.7 documentation](https://docs.opencv.org/2.4.13.7/modules/video/doc/motion_analysis_and_object_tracking.html?highlight=backgroundsubtractorMOG)
16. [(PDF) ViBe: A Universal Background Subtraction Algorithm for Video Sequences](https://www.researchgate.net/publication/224206851_ViBe_A_Universal_Background_Subtraction_Algorithm_for_Video_Sequences)
17. [Computer Vision Winter Workshop 2010, Libor ˇSpaˇcek, Vojtˇech Franc (eds.)](https://cmp.felk.cvut.cz/~matas/papers/trefny-emb_lbp-cvww10.pdf)
18. [A FPGA Implementation of Farneback Optical Flow by High-Level Synthesis](https://dl.acm.org/doi/10.1145/3289602.3294005)
19. [Optical Flow with OpenCV: Lucas-Kanade vs. Farneback](https://eureka.patsnap.com/article/optical-flow-with-opencv-lucas-kanade-vs-farneback)
20. [JSFeat - JavaScript Computer Vision Library.](https://inspirit.github.io/jsfeat/)
21. [VICE: View-Invariant Chess Estimation Kevin Zhu](https://www.scitepress.org/Papers/2023/121672/121672.pdf)
22. [ChessQueries: Toward Better Chess Board Recognition](https://arxiv.org/html/2608.30762v1)
23. [VideoKifu, or the automatic transcription of a Go game](https://arxiv.org/pdf/1807.01577)
24. [VideoKifu, or the automatic transcription of a Go game](https://www.groundai.com/project/videokifu-or-the-automatic-transcription-of-a-go-game/1)
25. [End-to-end Go (Weiqi) game record reconstruction from live broadcast videos - ScienceDirect](https://www.sciencedirect.com/science/article/abs/pii/S0952197625024868)
26. [DeepGamble: Towards unlocking real-time player ...](https://arxiv.org/pdf/2012.08011)
27. [Poker Game State Detection Jack Hung, Luke Moberly, Michael Souliman](https://cs231n.stanford.edu/2024/papers/poker-game-state-detection.pdf)
28. [System and method for automated table game activity recognition](https://image-ppubs.uspto.gov/dirsearch-public/print/downloadPdf/11580746)
29. [System and method for automated table game activity recognition](https://image-ppubs.uspto.gov/dirsearch-public/print/downloadPdf/10956750)
30. [Detection device capable of accurately reading dots on dice](https://image-ppubs.uspto.gov/dirsearch-public/print/downloadPdf/8216067)
31. [Detection device capable of accurately reading dots on dice](https://image-ppubs.uspto.gov/dirsearch-public/print/downloadPdf/8998698)
32. [D-i-Shaker](https://tcsjohnhuxley.com/us/product/d-i-shaker/)
33. [WO2017140385A1 - Dice recognition device & method of recognizing dice - Google Patents](https://patents.google.com/patent/WO2017140385A1/en)
34. [(PDF) Multivariate dice recognition using invariant features](https://www.researchgate.net/publication/258808321_Multivariate_dice_recognition_using_invariant_features)
35. [Stereo vision based dice recognition system and stereo vision based dice recognition method for uncontrolled environments](https://image-ppubs.uspto.gov/dirsearch-public/print/downloadPdf/8724888)
36. [GitHub - Kishaan/Dice-Detection: Computer Vision project in which traditional computer vision techniques are used to determine the location and numbers of pips on the top side of the dice from a video.](https://github.com/Kishaan/Dice-Detection)
37. [1 MS-TCN++: Multi-Stage Temporal Convolutional Network for Action Segmentation](https://arxiv.org/pdf/2006.09220)
38. [MS-TCN: Multi-Stage Temporal Convolutional Network for Action Segmentation](https://arxiv.org/pdf/1903.01945)
39. [Sound event detection in domestic environments - DCASE](https://dcase.community/challenge2019/task-sound-event-detection-in-domestic-environments)
40. [Metrics - DCASE](https://dcase.community/challenge2017/metrics)
41. [Unifying Global and Local Scene Entities Modelling for Precise Action Spotting](https://arxiv.org/pdf/2404.09951)
42. [3DFR: A Swift 3D Feature Reductionist Framework for Scene Independent Change Detection](https://arxiv.org/pdf/1912.11891)
43. [Processing video with WebCodecs and @remotion/media-parser](https://www.remotion.dev/docs/media-parser/webcodecs)
44. [TCG-AR: Real-Time Multi-View Augmented Reality for Trading Card Game Streaming](https://arxiv.org/pdf/2607.02090)
45. [Change Detection in Feature Space Using Local Binary Similarity Patterns](https://www.researchgate.net/publication/260730460_Change_Detection_in_Feature_Space_Using_Local_Binary_Similarity_Patterns)
46. [OffscreenCanvas API: getContext: \`webgl\` context](https://caniuse.com/mdn-api_offscreencanvas_getcontext_webgl_context)
