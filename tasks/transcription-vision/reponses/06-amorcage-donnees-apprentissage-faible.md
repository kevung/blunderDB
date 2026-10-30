Recherche du 2026-10-30 — claude.ai, recherche approfondie.
Prompt : ../prompts/06-amorcage-donnees-apprentissage-faible.md

---

# Amorçage des données et apprentissage faiblement supervisé pour la transcription vidéo de backgammon (blunderDB)

Votre base de matchs transcrits à la main suffit à amorcer le système, à condition de la traiter comme une **supervision par horodatage bruitée** et non comme des étiquettes image par image. Il faut la nettoyer d'abord avec les contraintes du jeu (légalité du coup vis-à-vis des dés, cohérence plateau avant/après), plutôt qu'avec des méthodes génériques de bruit d'étiquettes. Pour l'amélioration continue, la solution la plus simple qui tienne n'est pas le ré-entraînement local : c'est un **classifieur à prototypes / k-NN sur des embeddings figés**, calculé dans la webview (ou en Go pur), avec un seuil de confiance fixé sur une courbe risque–couverture évaluée **par stream**.

## TL;DR

- **Chaîne d'amorçage** : pour chaque coup, on prend une fenêtre de quelques secondes autour de `roll_tick_ms`/`tick_ms`, décodée dans la webview. On y sélectionne un plateau de stabilité (faible différence inter-images et netteté maximale), on étiquette plusieurs images voisines avec les dés et le plateau de la base, puis on élimine les coups incohérents (coup illégal pour les dés saisis, plateau après ≠ plateau avant + coup). La littérature sur la supervision par horodatage (Moltisanti et al., CVPR 2019 ; Li, Abu Farha et Gall, CVPR 2021) valide le principe : partir de l'instant annoté et laisser le modèle ré-estimer la bonne fenêtre.
- **Amélioration continue** : l'entraînement du modèle de base se fait chez le développeur (Python autorisé), avec export en ONNX et inférence dans la webview (WASM partout, WebGPU seulement là où il existe). Sur la machine de l'utilisateur, on n'ajoute que des **prototypes/k-NN** issus de ses corrections. ONNX Runtime a déprécié tous ses paquets d'entraînement à partir de la v1.20 (novembre 2024) ; TensorFlow.js permet `fit()` en navigateur ; GoMLX a un backend « Go pur » portable mais lent. Le ré-entraînement local est donc possible, mais il n'est pas nécessaire au départ.
- **Tailles de corpus** : la littérature ne donne aucun chiffre primaire pour « > 99 % par dé » sur vidéo de stream (**non trouvé**). Les points d'appui les plus proches : un CNN entraîné de zéro atteint 99,76 % en validation sur des d6 recadrés (rapport non relu par des pairs, environ 16 000 images tous dés confondus) ; un détecteur entraîné sur 4 498 scènes Blender tombe de 0,959 à 0,776 sur des dés jamais vus ; chesscog atteint 99,83 % par case sur un jeu d'échecs inconnu après ré-ajustement sur 2 photos. Mes estimations d'ingénieur, signalées comme telles : environ 1 500–3 000 lancers propres répartis sur ≥ 3 streams pour l'amorçage, puis quelques dizaines de corrections par nouvelle caméra.

## Key Findings

1. **Votre bruit est surtout temporel, pas sémantique.** La littérature sur la « supervision par horodatage » traite précisément le cas d'un instant unique posé près de l'action. Moltisanti, Fidler et Damen (Université de Bristol ; Université de Toronto/NVIDIA ; CVPR 2019, arXiv:1904.04689) remplacent les bornes d'action par des distributions d'échantillonnage initialisées sur l'horodatage, mises à jour ensuite par la réponse du classifieur. Ces distributions « convergent vers la localisation et l'étendue des segments discriminants », et les auteurs écrivent : « Our update method improves top-1 test accuracy by up to 5.4%. across the evaluated datasets » (trois jeux de données). Li, Abu Farha et Gall (CVPR 2021) ajoutent une perte de confiance qui force la confiance du modèle à décroître de façon monotone quand on s'éloigne de l'horodatage.\[1\] Rahaman, Singhania, Thiery et Yao (National University of Singapore, ECCV 2022, arXiv:2207.10137, méthode EM-TSS) notent que « previous works rely on assigning hard labels, and performance rapidly collapses under subtle violations of the annotation assumptions », et proposent une variante robuste aux erreurs d'annotation. Pour vous, cela signifie qu'il ne faut jamais étiqueter l'image exacte de `roll_tick_ms` : il faut étiqueter une fenêtre, puis la resserrer.
2. **Le meilleur filtre de bruit est le jeu lui-même.** Le Confident Learning (Northcutt, Jiang et Chuang, JAIR 70:1373–1411, 2021) estime la distribution jointe entre étiquettes bruitées et étiquettes vraies à partir de probabilités hors échantillon.\[2\] Co-teaching (Han et al., NeurIPS 2018) entraîne deux réseaux qui s'échangent les exemples à faible perte, en s'appuyant sur le fait que les réseaux mémorisent d'abord les exemples propres.\[3\] Ces méthodes sont utiles, mais au backgammon on dispose d'un oracle gratuit : un coup transcrit doit être légal pour les dés transcrits, et le plateau après doit être égal au plateau avant plus le coup. Ce contrôle déterministe élimine une classe entière d'erreurs avant tout apprentissage.
3. **La lecture de dés en conditions contrôlées est facile ; la généralisation à une nouvelle caméra ne l'est pas.** Un rapport de la Western Kentucky University (2021, non relu par des pairs) obtient 99,76 % de validation sur des d6 recadrés avec un petit CNN entraîné de zéro. La validation y est faite en prenant une image sur 8 du même corpus, donc avec une fuite probable.\[4\] Une contribution open source de 2026 (YOLOX-Nano, 4 498 scènes Blender plus un jeu public) obtient 0,959 de « value accuracy » sur le test public mais seulement 0,776 sur les dés du mainteneur.\[5\] C'est exactement le risque « un seul stream » que vous décrivez.
4. **Les rendus synthétiques aident surtout comme randomisation de domaine, pas comme substitut.** Tobin et al. (IROS 2017) montrent qu'un détecteur entraîné uniquement sur des rendus à textures aléatoires transfère au réel.\[6\]\[7\] Chesscog (Wölflein et Arandjelović, J. Imaging 2021) entraîne sur 4 888 rendus Blender (0,23 % d'erreur par case). Pour s'adapter à un jeu réel inconnu, il lui faut toutefois un ré-ajustement sur 2 photos (99,83 % par case).\[8\] Pour des objets aussi simples que des dés et des pions, le synthétique sert à couvrir les angles et les éclairages ; le réel de vos streams reste indispensable pour le rendu vidéo (compression, overlays).
5. **Le fine-tuning en navigateur est fragile en 2026.** ONNX Runtime Web Training (introduit en version 1.17, en février 2024) ne tournait que sur CPU/WASM.\[9\]\[10\] La note de version v1.20.0 (1er novembre 2024) annonce : « All ONNX Runtime Training packages have been deprecated ».\[11\] Une issue GitHub confirme que la 1.20 ne livre plus les binaires `ort-training-wasm`.\[12\] TensorFlow.js garde un `model.fit()` asynchrone en navigateur.\[13\] GoMLX propose un backend « Go pur » décrit par ses auteurs comme « very portable but slow » ; son backend rapide dépend de XLA via des wrappers C, ce qui l'exclut sous votre contrainte « pas de CGO ».\[14\]\[15\]
6. **WebGPU n'est pas un dénominateur commun pour un binaire Wails.** Les versions sont documentées : Chrome/Edge 113 (mai 2023), Firefox 141 sous Windows (juillet 2025), Safari 26 (septembre 2025, sur macOS/iOS 26).\[16\]\[17\] Sous Linux, Chrome ne l'active par défaut qu'à partir de la 144 (Intel Gen12+) et de la 147 (NVIDIA).\[17\]\[18\]\[19\] Pour **WebKitGTK**, la webview Linux de Wails, aucune mention de WebGPU n'a été trouvée dans les notes 2.48 à 2.54 (la 2.54 date de septembre 2026).\[20\]\[21\] Il faut donc viser l'inférence en WASM/CPU et ne traiter WebGPU que comme un bonus sur Windows (WebView2) et macOS 26+ (WKWebView).
7. **Le seuil d'auto-validation doit venir d'une courbe risque–couverture.** Geifman et El-Yaniv (Technion, NeurIPS 2017, arXiv:1705.08500) construisent un classifieur sélectif à partir d'un réseau entraîné, avec garantie probabiliste sur le risque : « an unprecedented 2% error in top-5 ImageNet classification can be guaranteed with probability 99.9%, and almost 60% test coverage ». Le principe se transpose directement : on choisit le seuil qui donne un risque sélectif ≤ cible sur des **streams tenus à l'écart**, jamais sur des images du même match.
8. **Pour choisir les exemples à garder, privilégier d'abord la typicité, ensuite l'incertitude.** Hacohen, Dekel et Weinshall (ICML 2022) montrent qu'en petit budget les stratégies d'incertitude font souvent moins bien que le hasard (« cold start »), alors que des exemples typiques et diversifiés (TypiClust) donnent les meilleurs résultats. Sur CIFAR-10 avec 10 étiquettes en semi-supervisé, ils atteignent 93,2 %, soit +39,4 points sur une sélection aléatoire.\[6\] Pour une nouvelle caméra, il faut donc garder d'abord quelques exemples typiques par face et par position, puis seulement les cas limites.
9. **Côté droit, le cadre français est favorable, mais l'opt-out et les CGU des plateformes compliquent les choses.** L'article L122-5-3 III du CPI autorise la fouille « quelle que soit la finalité » sur des œuvres licitement accessibles, « sauf si l'auteur s'y est opposé de manière appropriée ». Les copies doivent être « détruites à l'issue de la fouille ».\[22\] L'article R122-28 admet que l'opposition soit exprimée « par tout moyen », y compris par des CGU.\[23\] YouTube a ajouté en décembre 2024 un réglage « entraînement par des tiers » désactivé par défaut, et ses CGU interdisent l'accès non autorisé (scraping).\[24\]\[25\]

## Details

### 1. Extraction d'imagettes à partir d'horodatages approximatifs

**Ce que dit la littérature.** Le cas « un instant annoté, proche mais pas exact » porte le nom de *timestamp supervision* ou *single-frame supervision* :
- Moltisanti et al., CVPR 2019, DOI 10.1109/CVPR.2019.01015 : distributions d'échantillonnage itérativement recentrées par le classifieur.\[26\]
- Li, Abu Farha et Gall, CVPR 2021, arXiv:2103.06669 : détection des changements d'action par une fonction d'énergie et perte de confiance monotone autour de l'horodatage ; code PyTorch public.\[27\]\[28\]
- Travaux 2022 (arXiv:2207.10137, ECCV ; arXiv:2210.06501) : robustesse aux erreurs et aux omissions d'annotation.

Ces méthodes ciblent la segmentation d'actions multi-classes. Votre problème est plus simple, car vous cherchez un état stable (dés posés, plateau immobile), pas une action.

**Recette recommandée (heuristique, validée par la logique plutôt que par un article dédié) :**
1. *Fenêtre* : de `roll_tick_ms − 2 s` à `tick_ms + 2 s` (à ajuster selon votre distribution réelle d'écarts, à mesurer d'abord sur 50 coups vérifiés à la main).
2. *Échantillonnage* : 5–10 images/s via `<video>` + seek + `drawImage` sur canvas dans la webview (aucun décodage hors webview).
3. *Stabilité* : différence absolue moyenne entre images consécutives, restreinte à la zone du plateau. On cherche le plus long plateau sous un seuil après le lancer (dés immobiles) et avant le geste de la main.
4. *Netteté* : variance du laplacien sur la zone des dés (mesure de focus classique ; je n'ai pas pu vérifier dans cette revue une comparaison primaire des opérateurs de focus, **non trouvé**). On garde le top-K net du plateau de stabilité.
5. *Accord entre voisines* : une fois un premier modèle disponible, on n'accepte une imagette que si ses K voisines donnent la même lecture. C'est l'équivalent pratique de la « confiance décroissante autour de l'horodatage ».
6. *Étiquetage* : les K images héritent des dés de la base ; l'image du plateau stable avant `roll_tick_ms` hérite du plateau « avant », celle d'après `tick_ms` du plateau « après ».
7. *Itération EM* : on ré-entraîne, puis on re-sélectionne dans la fenêtre l'image où le modèle est le plus confiant **sur l'étiquette connue**, comme dans Moltisanti 2019.

### 2. Étiquettes bruitées : quoi utiliser à l'échelle de quelques milliers d'exemples

- **D'abord les filtres déterministes du domaine** : légalité du coup pour les dés, cohérence plateau avant/après, cohérence entre coups successifs (le plateau après du coup *n* doit égaler le plateau avant du coup *n+1*, au retournement de perspective près). C'est gratuit et exact.
- **Ensuite le Confident Learning** (JAIR 2021, DOI 10.1613/jair.1.12125). Il demande seulement des probabilités hors échantillon (validation croisée 5 plis dans l'article) et fonctionne avec n'importe quel classifieur.\[2\] Il convient bien à quelques milliers d'exemples, car il ne demande pas de changer l'entraînement : on classe les exemples suspects et on les fait revoir.
- **Co-teaching** (NeurIPS 2018, arXiv:1804.06872) : conçu pour du bruit « extrême » (expériences sur MNIST/CIFAR bruités).\[29\] Un relecteur NeurIPS 2018 relevait déjà qu'on ne sait pas bien ce qui compte comme « extrêmement bruité » ni pourquoi la méthode ne marcherait bien qu'à fort bruit.\[30\] Avec un bruit de quelques pourcents après filtrage par les règles du jeu, son apport attendu est faible et son coût (deux réseaux) réel. Mon avis : à ne pas utiliser au départ.
- **Pertes robustes ou lissage d'étiquettes** : bon marché, à garder comme régularisation de base. Je n'ai pas de chiffre primaire spécifique aux petits corpus d'objets simples (**non trouvé**).

### 3. Peu de caméras : augmentation, synthétique, sur-apprentissage

- **Risque documenté** : l'écart 0,959 → 0,776 du détecteur de dés de 2026 (de la répartition publique à des dés réels inconnus) et le besoin de ré-ajustement de chesscog sur un jeu inconnu montrent que la précision « dans le domaine » ne dit rien de la précision sur une nouvelle caméra.\[5\]\[31\]
- **Augmentations pertinentes** (choisies d'après les modes de variation réels, pas d'après un article dédié aux dés) :
  - homographie / perspective modérée ;
  - rotation libre des imagettes de dés (un dé posé n'a pas d'orientation privilégiée) ;
  - jitter de couleur et de gamma, changement de teinte des dés et des pions (les jeux varient) ;
  - flou de mouvement et de défocalisation ;
  - **ré-encodage JPEG/H.264 à bas débit** et redimensionnement aval-amont (artefacts de stream) ;
  - incrustation d'overlays (bandeaux, scores, webcam) ;
  - occultation partielle par une main.
- **Synthétique** : la randomisation de domaine (Tobin et al., IROS 2017, DOI 10.1109/IROS.2017.8202133) est la référence. Elle a permis un détecteur réel précis à 1,5 cm entraîné uniquement sur des rendus à textures non réalistes.\[7\] Dans un pipeline spatial (arXiv:2101.05661), l'ajout de 1 300 images Blender augmentées fait passer la précision de 0,52 à 0,75 en conditions difficiles, mais elle reste stable (0,93 → 0,92) en conditions normales.\[32\] Le synthétique aide donc sur les cas durs, pas sur les cas faciles. Pour les dés et les pions, un générateur procédural simple suffit probablement : cubes à coins arrondis, pips en disques, plateau texturé aléatoirement, caméra fixe à angle variable. Aucune étude primaire mesurant l'apport du synthétique pour des **pions de backgammon** n'a été trouvée (**non trouvé**).
- **Garde-fou** : toujours évaluer en *leave-one-stream-out* (voir §5).

### 4. Apprentissage actif et auto-amélioration locale

**Quoi garder comme nouvel exemple :**
- toute **correction** de l'utilisateur (erreur du modèle sur une étiquette désormais vérifiée), qui a la valeur la plus haute ;
- toute **confirmation à confiance moyenne** (zone grise de la courbe risque–couverture) ;
- un **échantillon de confirmations à haute confiance**, pour garder la couverture des cas typiques, conformément au résultat de TypiClust en petit budget ;
- jamais les auto-validations sans confirmation humaine, pour éviter la dérive par auto-renforcement.

**Comment l'exploiter sans Python sur la machine de l'utilisateur :**

| Option | État 2026 | Verdict |
|---|---|---|
| Prototypes / k-NN sur embeddings figés (JS ou Go) | Trivial à implémenter ; SimpleShot (Wang et al., 2019, arXiv:1911.04623) montre qu'un plus-proche-voisin après centrage et normalisation L2 rivalise avec le méta-apprentissage\[33\] | **Recommandé** |
| TensorFlow.js `model.fit()` (tête seulement) | API documentée, asynchrone, backends WebGL/WASM | Option de phase 2\[13\]\[34\] |
| ONNX Runtime Web Training | Introduit en 1.17 (février 2024), CPU/WASM seulement, artefacts à préparer hors ligne en Python ; paquets d'entraînement dépréciés en 1.20 (novembre 2024)\[9\]\[11\] | **À éviter** |
| GoMLX backend « Go pur » | Pas de C, portable, mais « slow » selon les auteurs ; le backend XLA impose des wrappers C (CGO)\[14\]\[35\] | Possible pour une tête linéaire, peu utile |
| Gorgonia | Non évalué dans cette revue | **non trouvé** |
| Transformers.js | Orienté inférence ; aucun support d'entraînement documenté trouvé | **non trouvé** |

Le schéma recommandé est le suivant. Le modèle de base est un petit CNN (encodeur figé plus tête), exporté en ONNX, en inférence onnxruntime-web WASM. Chaque correction ajoute l'embedding de l'imagette à une table de prototypes par classe et par caméra, stockée en SQLite côté Go. La décision finale combine la tête du réseau et la distance au prototype local. C'est réversible, sans gradient, et cela ne stocke **que des vecteurs** (aucune image). Remarque sur la confidentialité : un embedding reste une donnée dérivée ; il faut le garder local.

### 5. Protocole d'évaluation

- **Séparation** : par vidéo au minimum, **par stream/caméra** de préférence (leave-one-stream-out). Les images d'un même lancer sont quasi identiques ; une séparation par image gonfle mécaniquement les scores. Le rapport WKU, avec une validation sur une image sur 8 du même corpus, illustre ce piège.\[4\]
- **Métriques** :
  - précision par dé (6 classes) et matrice de confusion (le 2 lu comme 3 est l'erreur typique rapportée en 2026) ;\[5\]
  - précision par lancer (les deux dés justes, ordre ignoré, doubles traités à part) ;
  - précision de détection du lancer (instant à ± X ms) ;
  - précision de **discrimination du coup** = top-1 parmi les coups légaux ;
  - précision de bout en bout par match.
- **Agrégation** : chesscog montre que 0,23 % d'erreur par case ne donne que 93,86 % de plateaux parfaitement justes.\[36\] Les erreurs élémentaires se composent, donc il faut viser la métrique au niveau du coup, pas seulement par élément.
- **Seuil d'autonomie** : on trace la courbe risque–couverture (Geifman et El-Yaniv 2017) par stream tenu à l'écart et on fixe le seuil pour un risque sélectif cible, par exemple ≤ 0,5 % au niveau du coup. On rapporte l'AURC et la couverture obtenue. On recalibre le seuil pour chaque nouvelle caméra à partir des premières confirmations.

### 6. Tailles de corpus

| Repère | Chiffre | Nature |
|---|---|---|
| d6 recadrés, CNN de zéro | 99,76 % val. ; environ 16 000 images tous dés confondus (jeu Kaggle) ; « thousands » pour l'entraînement\[4\] | Rapport WKU 2021, non relu, fuite probable |
| Petit CNN d6 28×28 | 86,46 % test sur 96 ; 307 exemples d'entraînement\[37\] | Dépôt amateur, non vérifié directement |
| Détecteur dés + 4 498 scènes Blender | 0,959 test public, 0,776 dés inconnus\[5\] | Contribution open source 2026 |
| Chesscog | 4 888 rendus → 0,23 % d'erreur/case ; 2 photos → 99,83 %/case sur jeu inconnu | Article J. Imaging 2021 (DOI 10.3390/jimaging7060094)\[8\]\[38\] |
| Sélection active petit budget | 10 étiquettes → 93,2 % CIFAR-10 (semi-supervisé) | ICML 2022 |

**Estimations d'ingénieur (non sourcées, à valider par l'expérience du §Incertitudes) :**
- *Phase 0 – amorçage* : 1 500–3 000 lancers nettoyés, répartis sur ≥ 3 streams. À K = 5 images × 2 dés, cela donne environ 15 000–30 000 imagettes de dés, fortement corrélées. Pour le plateau, quelques centaines de positions par stream (chaque position fournit environ 26 « cases » étiquetées). Avec un encodeur pré-entraîné (ImageNet ou synthétique), on peut viser la partie basse ; sans pré-entraînement, la partie haute plus du synthétique.
- *Phase 1 – nouvelle caméra* : 20–50 corrections ou confirmations par caméra, en couvrant chaque face au moins 3–5 fois, pour peupler les prototypes (par analogie avec les 2 photos de chesscog, qui couvraient 32 pièces chacune).
- *Phase 2 – ré-entraînement central périodique* : quand ≥ 5–10 streams distincts sont disponibles, ré-entraîner l'encodeur chez le développeur et redistribuer les poids.
- La cible « > 95 % de discrimination du coup » est plus facile que la reconnaissance complète du plateau : les coups légaux candidats ne diffèrent souvent que sur 2 à 4 points. Aucune source primaire ne le chiffre pour le backgammon (**non trouvé**).

### 7. Éthique et données

- **Droit d'auteur, France/UE** : directive 2019/790, art. 4, transposée par l'ordonnance du 24 novembre 2021 à l'article L122-5-3 III du CPI.\[39\]\[40\] La fouille est permise pour toute finalité sur des contenus licitement accédés, sauf opposition appropriée (lisible par machine pour les contenus en ligne ; R122-28 : « par tout moyen », y compris des CGU). Les copies sont stockées de façon sécurisée puis **détruites** à l'issue de la fouille.\[22\]\[23\] Sur demande des ayants droit, il faut prouver ce stockage et cette destruction (R122-27).\[41\] Votre principe « seuls les poids sont distribués » va dans ce sens. Il faut en plus documenter la purge des imagettes et tenir une liste des sources utilisées.
- **Opt-out** : il faut vérifier, pour chaque chaîne, ses métadonnées et ses CGU. Sur YouTube, le réglage « third-party training » est désactivé par défaut et lisible via l'API « Video Third-Party Trainability ».\[24\] Qu'il vaille opposition au sens de L122-5-3 III est **une question ouverte non tranchée** dans les sources trouvées. Aux États-Unis, le créateur YouTube David Gardner a déposé le 23 février 2026 une plainte (Gardner v. Runway AI Inc., C.D. Cal., n° 2:26-cv-01941) accusant Runway d'avoir « bypassed YouTube's copyright protections to illegally download user videos for its AI training » (Reuters, 24 février 2026) ; la plainte mentionne le réglage d'autorisation de l'entraînement par des tiers, mais relève du droit américain (CGU, concurrence déloyale californienne, DMCA) et non de L122-5-3. Prudence : n'utiliser pour l'entraînement central que des vidéos dont la chaîne a autorisé l'entraînement par des tiers, ou avec un accord explicite des streamers.
- **CGU des plateformes** : les CGU de YouTube interdisent l'accès et la reproduction non autorisés.\[25\]\[42\] Le fait que l'utilisateur ait un fichier local ne garantit pas la licéité de l'accès initial.
- **RGPD** : mains et visages sont des données personnelles potentielles. Il faut recadrer immédiatement sur le plateau et les dés, ne jamais conserver d'images plein cadre, garder les embeddings locaux par défaut et rendre explicite tout partage de corrections vers le développeur (opt-in).

### Évaluation des preuves (fiabilité)

| Élément | Date | Type | Fiabilité |
|---|---|---|---|
| Moltisanti, Fidler, Damen – supervision par horodatage unique | 2019 | CVPR, DOI 10.1109/CVPR.2019.01015 | primaire |
| Li, Abu Farha, Gall – segmentation par horodatage | 2021 | CVPR, arXiv:2103.06669, code | primaire |
| Cadre EM robuste à l'horodatage | 2022 | ECCV, arXiv:2207.10137 | primaire |
| Northcutt, Jiang, Chuang – Confident Learning | 2021 | JAIR, DOI 10.1613/jair.1.12125 | primaire |
| Han et al. – Co-teaching | 2018 | NeurIPS, arXiv:1804.06872 | primaire |
| Geifman, El-Yaniv – classification sélective | 2017 | NeurIPS, arXiv:1705.08500 | primaire |
| Hacohen, Dekel, Weinshall – TypiClust | 2022 | ICML/PMLR 162, arXiv:2202.02794 | primaire |
| Wang et al. – SimpleShot | 2019 | arXiv:1911.04623, code | primaire (préprint) |
| Tobin et al. – randomisation de domaine | 2017 | IROS, DOI 10.1109/IROS.2017.8202133 | primaire |
| Wölflein, Arandjelović – chesscog | 2021 | J. Imaging, DOI 10.3390/jimaging7060094, code | primaire |
| Pipeline spatial Blender | 2021 | arXiv:2101.05661 | primaire (préprint) |
| WKU – lecture de dés par CNN | 2021 | rapport institutionnel, sans DOI | secondaire/faible |
| Détecteur de dés YOLOX-Nano + Blender | 2026 | pull request GitHub | blog/faible |
| ONNX Runtime – entraînement Web, dépréciation v1.20.0 | 2024 | blog Microsoft, notes de version | primaire (éditeur) |
| TensorFlow.js – guide Layers / `fit()` | courant | documentation éditeur | primaire (éditeur) |
| GoMLX / gomlx-compute – backend Go pur | 2025–2026 | dépôt et documentation | primaire (projet) |
| WebGPU – web.dev, wiki gpuweb, caniuse | 2025–2026 | éditeurs et W3C | primaire |
| WebKitGTK 2.48–2.54 highlights | 2025–2026 | webkitgtk.org | primaire (absence de mention) |
| L122-5-3 et R122-23 à R122-28 CPI | en vigueur depuis 2021 | Légifrance | primaire |
| YouTube Third-Party Trainability API | 2024–2026 | Google Developers | primaire (éditeur) |
| Article TechCrunch sur l'opt-in YouTube | 16 déc. 2024 | presse | secondaire |

**Contradictions relevées :**
- WKU 2021 (99,76 %) contre détecteur 2026 (77,6 % sur dés inconnus). Il n'y a pas vraiment de contradiction, mais deux protocoles : validation intra-corpus contre généralisation hors domaine.
- Le rapport WKU indique 95,10 % dans son résumé et 95,12 % dans son tableau pour le d10.\[4\]
- Sur WebGPU dans Firefox, caniuse affiche « disabled by default », alors que web.dev et MDN indiquent un support dans Firefox 141 sous Windows.\[16\]\[43\] caniuse agrège par plateforme et marque partiel/désactivé tant que Linux n'est pas couvert.\[44\]

## 1. Verdict

Le jeu étiqueté existant est **suffisant pour amorcer** un lecteur de dés et un discriminateur de coups, à trois conditions : (a) traiter les horodatages comme des fenêtres et non comme des images ; (b) purger d'abord les incohérences par les règles du backgammon ; (c) évaluer par stream tenu à l'écart. Les méthodes génériques de bruit d'étiquettes (Confident Learning) servent d'outil de revue, pas de pierre angulaire ; Co-teaching est superflu à votre échelle. Le fine-tuning local n'est pas la solution la plus simple en 2026 : la voie d'entraînement d'ONNX Runtime est dépréciée et WebGPU est absent de WebKitGTK. Un **k-NN/prototypes local** sur embeddings, plus un **ré-entraînement central** périodique, couvrent le besoin.

## 2. Recommandation technique

**Chaîne d'amorçage (de la base aux imagettes étiquetées) :**
1. *Audit* : vérifier pour chaque coup la légalité (dés + plateau avant → coup) et la cohérence de la chaîne des plateaux. Exclure ou mettre en revue les échecs. Mesurer à la main, sur environ 50 coups, la distribution réelle de l'écart entre `roll_tick_ms` et l'instant où les dés deviennent lisibles.
2. *Calibrage géométrique par stream* : homographie du plateau saisie une fois par vidéo (4 clics), puis zones fixes : 24 points, barre, sorties, plateaux de dés, videau.
3. *Extraction dans la webview* : seek + canvas, 5–10 images/s dans la fenêtre, puis score de stabilité (différence inter-images) et de netteté (variance du laplacien). Garder le top-K = 5 du plateau stable, recadrer immédiatement et ne jamais conserver le plein cadre.
4. *Modèle v0 (chez le développeur, Python autorisé)* :
   - lecteur de dé = petit CNN, 6 classes plus « pas de dé / illisible » ;
   - classifieur de case = occupation et couleur, nombre de pions borné ;
   - mélange réel et synthétique procédural randomisé ;
   - augmentations de compression, de flou et d'overlays.
5. *Boucle EM sur la fenêtre* : re-sélectionner l'image la plus confiante pour l'étiquette connue, ré-entraîner 2–3 fois, puis lancer le Confident Learning pour lister les étiquettes suspectes à revoir.
6. *Décision de coup* : ne pas classer le coup directement. Calculer la vraisemblance de chaque coup légal en comparant le plateau prédit après au plateau résultant du coup candidat, et prendre l'argmax.
7. *Export* : ONNX, puis onnxruntime-web en WASM (WebGPU optionnel si détecté). Distribuer **uniquement des poids**.

**Amélioration continue (la plus simple qui tienne) :** pour chaque correction ou confirmation en zone grise, ajouter l'embedding à une table de prototypes locale par caméra. Sur un nouveau stream, seuil d'auto-validation désactivé jusqu'à ce que la courbe risque–couverture locale (sur les confirmations) atteigne la cible. Partage optionnel et explicite des corrections (étiquettes plus embeddings, sans images) pour le ré-entraînement central.

**Tailles à viser** (estimations à valider) :
- phase 0 : 1 500–3 000 lancers propres sur ≥ 3 streams ;
- phase 1 : 20–50 exemples par nouvelle caméra ;
- phase 2 : ré-entraînement central dès 5–10 streams.

## 3. Incertitudes

- **Distribution réelle des écarts d'horodatage** : inconnue. *Expérience* : annoter à la main l'instant de lisibilité sur 50 coups de 3 streams et tracer l'histogramme de `t_lisible − roll_tick_ms`.
- **Courbe d'apprentissage réelle** (« combien pour 99 % par dé ») : non trouvée dans la littérature. *Expérience* : entraîner sur 250, 500, 1 000 et 2 000 lancers, évaluer en leave-one-stream-out, puis extrapoler la loi de puissance de l'erreur. Comparer encodeur pré-entraîné et entraînement de zéro, avec et sans synthétique.
- **Apport du synthétique pour vos pions et dés** : non trouvé. *Expérience* : ablation réel seul / réel + synthétique sur le stream tenu à l'écart.
- **Prototypes contre fine-tuning de la tête** : *expérience* : sur un stream nouveau, comparer k-NN (5, 20 et 50 exemples) et `fit()` TensorFlow.js de la dernière couche, à budget identique.
- **WebGPU dans WebKitGTK** : aucune mention dans les notes officielles 2.48–2.54 (non trouvé, ce qui ne prouve pas l'absence). *Expérience* : tester `navigator.gpu` dans la webview Wails sur les distributions cibles.
- **Statut juridique de l'opt-in YouTube vis-à-vis de L122-5-3 III** et licéité de l'accès aux fichiers locaux : non tranché. Faire valider par un juriste avant tout entraînement central sur des streams tiers.
- **Décodage précis à l'image près via `<video>`** selon les moteurs (WebKitGTK/GStreamer, WebView2, WKWebView) : non vérifié dans cette revue. *Expérience* : mesurer l'erreur de seek sur une vidéo à compteur d'images incrusté.

## Sources

1. [Temporal Action Segmentation from Timestamp Supervision](https://arxiv.org/html/2103.06669v3)
2. [Confident Learning: Estimating Uncertainty in Dataset Labels](https://www.jair.org/index.php/jair/article/download/12125/26676/26691)
3. [Co-teaching: Robust training of deep neural networks with extremely noisy labels](https://proceedings.neurips.cc/paper/2018/hash/a19744e268754fb0148b017647355b7b-Abstract.html)
4. <https://digitalcommons.wku.edu/cgi/viewcontent.cgi?article=1004&context=seas_faculty_pubs>
5. [Dice → Seed sandbox with an on-device dice detector (beta) by Dado-hash · Pull Request #45 · Dado-hash/Sato-Wiki · GitHub](https://github.com/Dado-hash/Sato-Wiki/pull/45)
6. [Active Learning on a Budget: Opposite Strategies Suit High and Low Budgets](https://arxiv.org/pdf/2202.02794)
7. [Domain Randomization for Transferring Deep Neural Networks from Simulation to the Real World](https://arxiv.org/pdf/1703.06907)
8. [Journal of Imaging Article Determining Chess Game State From an Image](https://research-repository.st-andrews.ac.uk/bitstream/handle/10023/23299/W_lflein_2021_Determining_chess_game_state_JImaging_07_00094.pdf;jsessionid=F9FD42085018B6B34DCE7C6F8A2BC6BC?sequence=1)
9. [On-Device Training: Training a model in browser](https://opensource.microsoft.com/blog/2024/02/06/on-device-training-training-a-model-in-browser/)
10. [onnxruntime-training-examples/on\_device\_training/web at master · microsoft/onnxruntime-training-examples · GitHub](https://github.com/microsoft/onnxruntime-training-examples/tree/master/on_device_training/web)
11. [Release ONNX Runtime v1.20.0 · microsoft/onnxruntime · GitHub](https://github.com/microsoft/onnxruntime/releases/tag/v1.20.0)
12. [\[Web\] no training with wasm on node · Issue #22799 · microsoft/onnxruntime · GitHub](https://github.com/microsoft/onnxruntime/issues/22799)
13. [TensorFlow.js layers API for Keras users](https://js.tensorflow.org/tutorials/tfjs-layers-for-keras-users.html)
14. [pkg.go.dev](https://pkg.go.dev/github.com/gomlx/gomlx@v0.15.2)
15. [GitHub - gomlx/compute: Package compute provides a modular API for defining and executing multidimensional computation graphs with pluggable backends. · GitHub](https://github.com/gomlx/compute)
16. [WebGPU is now supported in major browsers | Blog | web.dev](https://web.dev/blog/webgpu-supported-major-browsers)
17. [WebGPU](https://en.wikipedia.org/wiki/WebGPU)
18. [WebGL 2 or WebGPU Not Supported? How to Fix It in Every Browser (2026)](https://app.cinevva.com/guides/webgl-webgpu-not-supported-fix)
19. [Implementation Status](https://github.com/gpuweb/gpuweb/wiki/Implementation-Status)
20. [WebKitGTK 2.54 highlights - The WebKitGTK Project](https://webkitgtk.org/2026/09/16/webkitgtk-2.54-highlights.html)
21. [WebKitGTK 2.54.0 released! - The WebKitGTK Project](https://www.webkitgtk.org/2026/09/16/webkitgtk2.54.0-released.html)
22. [Article L122-5-3 - Code de la propriété intellectuelle - Légifrance](https://www.legifrance.gouv.fr/codes/article_lc/LEGIARTI000044363192)
23. [Clause Fouille de textes et de données PROPRIETE INTELLECTUELLE - Campus de Neurochirurgie](https://campus.neurochirurgie.fr/article1885.html)
24. [YouTube Video Third-Party Trainability Home](https://developers.google.com/youtube/v3/video-trainability)
25. [YouTube logo](https://techcrunch.com/2024/12/16/youtube-will-let-creators-opt-out-of-third-party-ai-training)
26. [CVPR 2019 Open Access Repository](https://openaccess.thecvf.com/content_CVPR_2019/html/Moltisanti_Action_Recognition_From_Single_Timestamp_Supervision_in_Untrimmed_Videos_CVPR_2019_paper.html)
27. [Timestamp-Supervised Action Segmentation from the Perspective of Clustering](https://arxiv.org/html/2212.11694v2)
28. [Temporal Action Segmentation from Timestamp Supervision](https://github.com/ZheLi2020/TimestampActionSeg)
29. [\[PDF\] Co-teaching: Robust training of deep neural networks with extremely noisy labels](https://www.semanticscholar.org/paper/Co-teaching:-Robust-training-of-deep-neural-with-Han-Yao/e061d23b68e7d4aac5aece4794c044c80e638dca)
30. [Reviews: Co-teaching: Robust training of deep neural networks with extremely noisy labels](https://proceedings.neurips.cc/paper/2018/file/a19744e268754fb0148b017647355b7b-Reviews.html)
31. [Article Determining Chess Game State From an Image Georg Wölﬂein∗](https://arxiv.org/pdf/2104.14963v1)
32. [A Pipeline for Vision-Based On-Orbit Proximity Operations Using Deep Learning and Synthetic Imagery](https://arxiv.org/pdf/2101.05661)
33. [\[PDF\] SimpleShot: Revisiting Nearest-Neighbor Classification for Few-Shot Learning](https://www.semanticscholar.org/paper/8eab13a881348e8636582b636e1a84dba3ce8d9d)
34. [TensorFlow.js layers API for Keras users](https://www.tensorflow.org/js/guide/layers_for_keras_users)
35. [github.com](https://github.com/guygrigsby/compute)
36. [End-to-End Chess Recognition](https://arxiv.org/pdf/2310.04086)
37. [GitHub - acabbar/Multi-Class-Dice-Classification-with-a-Convolutional-Neural-Network-: Hey, This project explores the use of Convolutional Neural Networks (CNNs) for image classification by developing a deep learning model capable of recognizing the face value of dice from grayscale images. The objective was to build, train, and evaluate a CNN that can accurately classify images into six different classes. · GitHub](https://github.com/acabbar/Multi-Class-Dice-Classification-with-a-Convolutional-Neural-Network-)
38. [Determining Chess Game State from an Image](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC8321354/)
39. [Le droit de fouille ou « data mining » : principe et limitations - Enthemis](https://www.enthemis.com/le-droit-de-fouille-ou-data-mining-principe-et-limitations/)
40. [Le droit d'auteur à l'épreuve de l'intelligence artificielle. Fouille de textes et de données, réservation des droits et protégeabilité des contenus générés - Gdroit](https://gdroit.fr/droit-et-ia/reglement-ia/droit-auteur-donnees-entrainement/)
41. [Section 4 : Exceptions en vue de la fouille de textes et de données (Articles R122-23 à R122-28) - Légifrance](https://www.legifrance.gouv.fr/codes/section_lc/LEGITEXT000006069414/LEGISCTA000045960671/2023-01-01)
42. [storage.courtlistener.com](https://storage.courtlistener.com/recap/gov.uscourts.cacd.1008059/gov.uscourts.cacd.1008059.1.0.pdf)
43. [WebGPU](https://caniuse.com/webgpu)
44. [Browser Support](https://webcodingcenter.com/webgpu/why-webgpu-exists-and-where-it-stands--firefox-behind-a-flag-and-the-road-to-universal-support.html)
