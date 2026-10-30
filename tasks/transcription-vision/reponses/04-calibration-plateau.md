Recherche du 2026-10-30 — claude.ai, recherche approfondie.
Prompt : ../prompts/04-calibration-plateau.md

---

# Calibration du plateau et robustesse géométrique pour blunderDB : revue de littérature technique (2015–2026)

Une homographie DLT normalisée calculée à partir de 4 clics (ou plus), plus un modèle du plateau en coordonnées rectifiées et un suivi léger de la dérive (corrélation de 4 à 8 imagettes de coins, plus un contrôle de cohérence projective), suffisent pour blunderDB. Il n'y a rien de plus lourd à faire : la littérature go/échecs (Carta & Corsolini 2015–2018 ; Wölflein & Arandjelović 2021) montre que la difficulté est dans le choix des images exploitables et dans la vérification par contraintes de jeu. La géométrie, elle, est un problème résolu. L'atout décisif de blunderDB est qu'il connaît déjà la liste des coups légaux : la reconnaissance du coup devient un choix parmi N positions candidates, pas une lecture libre des 30 pions.

## TL;DR

- **Modèle minimal** : homographie 3×3 calculée en JS (DLT normalisée de Hartley, 1997). Le plateau est décrit dans un repère rectifié de 15 × 13 « diamètres de pion ». Chaque point a un axe d'empilement et 5 cases d'emplacement, puis une compression au-delà de 5. On compte par profil 1D le long de chaque flèche dans l'image rectifiée, jamais en cherchant des positions exactes de pions. La parallaxe due à l'épaisseur est absorbée en échantillonnant une bande, pas un point.
- **Recalibration** : sur chaque image « calme », on mesure l'écart des coins par corrélation normalisée d'imagettes de référence. On reprend la règle de VideoKifu (2018) : une nouvelle grille n'est acceptée qu'après 2–3 images stables consécutives. Les petits écarts sont recalés automatiquement. Les grands écarts (zoom, recadrage de la régie) déclenchent une relocalisation par gabarit, puis une alerte. Un changement de plan se détecte par la chute du score de présence du plateau, pas par un détecteur générique de coupes.
- **Surimpression** : par-dessus la vidéo, on dessine les 24 triangles, la barre et les zones de sortie, plus 30 pions « fantômes » (ellipses projetées) de la position initiale 2-5-3-5. L'utilisateur voit en une seconde si les fantômes tombent sur les vrais pions. On ajoute des poignées déplaçables avec loupe ×4–6 (Kinovea et l'étude de validité de 2019 plaçaient les coins à 600 %) et une carte de résolution qui signale les points trop lointains.

## Key Findings

1. **Homographie** : la DLT à 4 points est exacte (8 équations, 8 inconnues). Sa stabilité dépend de la normalisation : centrage, puis mise à l'échelle pour une distance moyenne de √2 (Hartley, IEEE TPAMI 19(6), juin 1997).\[1\] Fan, Dai, Seo et He (arXiv 2304.10771, 2023, Proposition 1) démontrent qu'aucune paire de matrices de normalisation n'atteint un conditionnement de 1 (« There is no pair of normalization matrices T′ and T that results in κ(Â) = 1 ») : la normalisation améliore le conditionnement sans le rendre parfait. Avec exactement 4 points cliqués, c'est l'erreur de clic qui domine, pas l'arithmétique.
2. **Aucune publication académique sur la lecture du backgammon à partir de vidéo** n'a été trouvée. Les seules références sont des dépôts peu documentés. Le plus proche d'un modèle « attendu » est le cadre MCBR-RAG d'Ofir Marom (arXiv 2501.05030, 2025) : grâce à la symétrie du plateau, 4 points de repère suffisent pour découper le plateau en parties, et un CNN prédit par partie le nombre de pions et leur propriétaire. Il a été entraîné sur des images simulées (« we simulate 100,000 random Backgammon board images ») et appliqué à des diagrammes tirés de trois livres (Magriel, Robertie, Trice), pas à de la vidéo.
3. **Le go fournit le modèle le plus abouti** pour la vidéo à caméra fixe (VideoKifu, 2015–2018). Ses enseignements : micro-recalibration qui suit des mouvements de grille « not larger than about one stone's radius between consecutive frames » ; vérification asynchrone ; projection du centre de la pierre distincte de l'intersection de la grille, à cause de l'épaisseur ; approche « minimale » qui ne cherche que le dernier coup.
4. **Occlusions** : sur 269 photos d'une partie, 65 (1 sur 4, jusqu'à 6 d'affilée) étaient perturbées par des mains ou des bras (Carta & Corsolini, 2015).\[2\] Toutes les méthodes vidéo étudiées filtrent les images : différence d'images avec hystérésis, détecteur de mains, ou délai de stabilité.\[3\]\[4\]\[5\] Aucune ne cherche à lire à travers l'occlusion.
5. **Obliquité** : aucun seuil chiffré propre au backgammon n'a été trouvé. Kinovea reste fiable entre 90° et 45°, mais est meilleur à 90° (étude PLoS One de 2019).\[6\] VideoKifu (fig. 2, partie Metta-Martinelli, Bologne 2018, image 1440×1080 rectifiée en 500×500) signale que rectifier une vue oblique étire le côté lointain : « Stretching too much would blur the borders of the stones, making them invisible to the edge detector (Canny filter) ». Le bon critère est donc mesurable : la taille en pixels d'un pion lointain, calculée à partir de l'homographie.
6. **Contexte de captation** : les règles BMAB (mises à jour en 2024) imposent que les matchs soient filmés, non montés et publics.\[7\] Sur les pages consultées, elles ne fixent ni angle ni résolution. La FFBG (2022) recommande un support pour « filmer à la verticale sans déformation marquée ».\[8\] Les vidéos de transcription seront donc souvent proches de la vue de dessus, mais pas toujours.

## Details

### 1. Homographie à quatre points en JavaScript sans bibliothèque

**Estimation.** Chaque correspondance (x, y) ↔ (u, v) donne 2 lignes de la matrice A (2n × 9). Avec h₃₃ = 1, 4 points donnent un système 8 × 8 résoluble par élimination de Gauss avec pivot partiel, soit environ 60 lignes de JS. Pour n > 4 (coins de la barre, coins intérieurs, points de raffinement), on résout aux moindres carrés. On peut prendre la SVD de A, ou plus simplement le plus petit vecteur propre de AᵀA (9 × 9) par itération inverse ou méthode de Jacobi. jsfeat (dépôt JS, sans date de version vérifiée) fournit un noyau « Homography2D » et des estimateurs RANSAC/LMEDS réutilisables comme référence de code.\[9\]

**Stabilité numérique (Hartley 1997).** Pour chaque ensemble de points, (1) on translate pour amener le centroïde à l'origine ; (2) on met à l'échelle pour que la distance moyenne à l'origine vaille √2 ; puis on calcule H̃ et on dénormalise : H = T′⁻¹ H̃ T.\[1\] Sans cette étape, les coordonnées en pixels (~10³) et les termes en x·u (~10⁶) donnent une matrice mal conditionnée. Fan, Dai, Seo et He (2023, Proposition 1) contredisent en partie l'idée que cette normalisation serait « optimale » : elle n'atteint pas un conditionnement unitaire. En pratique, pour 4 à 12 points, la normalisation de Hartley suffit. Deux contrôles à implémenter : (a) le déterminant de la partie 2 × 2 de H doit garder le même signe partout (pas de quadrilatère « retourné ») ; (b) le quadrilatère cliqué doit être convexe, sinon on refuse le clic.

**Raffinement par les bords.** Dubrofsky & Woodham (ISVC 2008, « Combining Line and Point Correspondences for Homography Estimation ») étendent la DLT aux droites. Une droite l se transforme par H⁻ᵀ, donc chaque bord détecté donne 2 équations, et la normalisation des droites reste une question ouverte selon les auteurs.\[10\] Mise en œuvre légère et recommandée : autour de chaque bord prédit, échantillonner des profils perpendiculaires (par exemple 20 profils de ±8 px). Sur chaque profil, prendre le maximum de gradient. Ajuster une droite robuste (moindres carrés tronqués), puis intersecter les 4 droites pour obtenir des coins sous-pixel. On retrouve ainsi la « régression linéaire des pixels de la ligne externe » de VideoKifu (2018), sans transformée de Hough complète. Le même article prévient que cette régression échoue quand les lignes sont masquées : il faut donc la faire sur la position de départ, plateau vide de mains.\[3\] En football, TVCalib (WACV 2023) note que dans les retransmissions, les segments de lignes sont plus visibles et plus fiables que les points-clés.\[11\] C'est un argument pour raffiner sur les bords plutôt que sur les coins.

**Cadre qui déborde du champ.** La DLT n'exige pas que les points soient dans l'image. Les poignées doivent donc pouvoir sortir du cadre vidéo, avec un canevas élargi d'environ 20 % autour de la vidéo. L'alternative est de laisser cliquer d'autres repères connus : coins intérieurs de l'aire de jeu, extrémités de la barre, pointes des triangles 1/12/13/24. Carta & Corsolini (2015) exigent que les coins soient à au moins un diamètre de pierre du bord de l'image pour leur suivi automatique.\[2\] La même contrainte vaut pour le suivi de dérive : si un coin est hors champ, il faut suivre d'autres imagettes (bords de la barre, angles des triangles).

### 2. Modèle géométrique du plateau

**Dimensions de référence : sources contradictoires, peu fiables.** Les fabricants et clubs convergent sur un pion « tournoi » de 1,75″ (≈ 44,5 mm) : Crisloid, New England Backgammon Club.\[12\]\[13\]\[14\]\[15\] Mais plusieurs sites de vente affirment des dimensions « officielles WBF » incompatibles entre elles : 21″ × 13″ fermé selon l'un, 53 × 44 cm avec un pion de 2,8 cm selon un autre.\[12\]\[16\]\[17\]\[18\] Ces chiffres ne sont pas étayés. **Conclusion : ne pas coder de dimensions absolues ; tout exprimer en unités de diamètre de pion d**, avec des proportions ajustables.

**Repère rectifié proposé (le modèle minimal).**
- Aire de jeu : 12 flèches en largeur plus la barre (≈ 1 d), soit environ 13 d × environ 11–13 d en hauteur selon le jeu. Les zones de sortie sont à droite ou à gauche, réglables. Une règle de fabrication (blog, non primaire) dit que 5 pions sur une flèche laissent encore voir la pointe : une flèche mesure donc environ 5 à 5,5 d.\[13\]\[19\]
- Point i : base sur le bord, axe d'empilement vers le centre. La case k (k = 0…4) a son centre à (k + 0,5)·d du bord.
- Au-delà de 5 pions : **aucune convention normalisée trouvée** (« non trouvé »). Deux pratiques se voient en vidéo, à modéliser comme hypothèses alternatives : (a) une 2ᵉ rangée posée à cheval, décalée de d/2, sur les premiers pions ; (b) une pile qui s'allonge vers le centre, au-delà de la pointe, avec chevauchement. Le modèle doit donc accepter n ∈ [0, 15] avec une longueur occupée L(n) monotone, et non des positions fixes.
- Piles qui penchent ou pions décentrés : on échantillonne une **bande** de largeur ≈ 0,8 d le long de l'axe et on lit un profil 1D (luminance, chroma). Le nombre de pions se déduit de la longueur occupée et des ruptures de contraste entre pions, et non de centres exacts.

**Parallaxe.** L'homographie ne vaut que pour le plan du feutre. Carta & Corsolini (2015) montrent que la projection du centre géométrique d'une pierre ne coïncide pas avec l'intersection de la grille, sauf caméra à l'infini, et que l'écart est « substantiel » pour les lignes lointaines vues de bas. Ils estiment le point de vue par points de fuite et birapports.\[2\] VideoKifu (2018) ajoute qu'en vue latérale, la forme projetée dépend fortement de l'épaisseur des pierres.\[3\] Pour blunderDB, on peut faire plus simple : décaler la bande d'échantillonnage vers la caméra d'un facteur proportionnel à l'épaisseur du pion (≈ 0,2 d) et à l'obliquité locale, et faire déborder la bande de 0,3 d. Le modèle 3D complet (pose de caméra) n'est utile qu'en vue très oblique.

**Ce que font les travaux échecs/go pour compter.**
- Wölflein & Arandjelović (J. Imaging 7(6):94, 2 juin 2021) : RANSAC sur intersections → transformation projective vers une grille régulière → image rectifiée. Un CNN prédit l'occupation case par case, puis un autre classe les pièces. Erreur de 0,23 % par case, « 28 times better than the current state of the art », c'est-à-dire Mehta et al. (93,45 % de précision par case, soit 6,55 % d'erreur). Les découpes par case sont élargies vers le haut pour tenir compte de la hauteur des pièces, ce qui confirme l'idée de la bande élargie.
- Une prépublication de 2026 (arXiv 2610.09191) prédit 8 points-clés étiquetés du pourtour « en octogone » pour avoir des correspondances redondantes. Elle note que les vues basses augmentent les occlusions entre pièces.\[20\]
- L'argument probabiliste de Carta & Corsolini (2015) s'applique directement au backgammon : avec 99 % de réussite par intersection, une image de 361 intersections n'est juste que dans 0,99³⁶¹ = 2,7 % des cas. Lire seulement les cases qui ont pu changer améliore la fiabilité de plusieurs ordres de grandeur.\[2\] Pour blunderDB, on compare donc les 24 + 2 + 2 zones avant et après le coup, et on ne score que les zones touchées par au moins un coup légal.
- Dépôts backgammon (non datés, secondaires) : jvitors23 (rectification + cercles de Hough, comptage par flèche) annonce 90 % de flèches justes mais seulement 40 % de positions entièrement justes, ce qui illustre exactement l'effet multiplicatif. Le jeu Roboflow « BgScan » compte 57 images.\[21\]\[22\] Selon la recherche du sous-agent, alexlitz/video_analysis (2 commits, sans résultats publiés) apprend directement sur des images entières de vidéos YouTube, sans étape de calibration.\[23\] Trackgammon (Backgammon Galaxy) promet de « convertir des vidéos en MAT par vision », mais c'est un produit fermé et non documenté.\[24\]

### 3. Dérive de caméra et recalibration

**Ce que dit la littérature.** VideoKifu (arXiv 1807.01577, juillet 2018) suit la grille par appariement de gabarits (MatchTemplate) à chaque image. Ce suivi suit des mouvements « not larger than about one stone's radius between consecutive frames », ce qui couvre vibrations et petits chocs. Une routine asynchrone plus lourde gère les grands déplacements. Une nouvelle grille n'est acceptée qu'après 2–3 images stables consécutives, et la correction est « usually computed within nearly two seconds » ; la routine de Hough asynchrone se termine pendant que les routines principales analysent environ trois images. Carta & Corsolini (2015) détectent les changements de point de vue avec le **birapport complexe** des 4 coins (coordonnées vues comme nombres complexes). Ce birapport est invariant par rotation, translation et homothétie, mais pas par projection centrale : il distingue un panoramique ou un zoom (similitude) d'un vrai changement de perspective.\[2\]

**Stratégie recommandée (faisable en JS + WebGL2, sans WASM obligatoire).**
1. *À la calibration* : on mémorise 4 à 8 imagettes de référence de 32 × 32 px en niveaux de gris (coins, extrémités de la barre) et une miniature rectifiée du plateau de 128 × 96.
2. *Chaque seconde, sur les images calmes* : corrélation croisée normalisée de chaque imagette dans une fenêtre de ±16 px. Coût négligeable en JS pur.
   - Tous les décalages < 1 px : rien à faire.
   - Décalages cohérents de 1 à 16 px : on recalcule H à partir des nouvelles positions (DLT), après 3 images concordantes, sans alerte.
   - Rapport d'échelle ou birapport modifié au-delà d'un seuil : c'est un zoom ou un recadrage. On passe au point 3.
3. *Relocalisation* : corrélation de phase sur la miniature (FFT 2D en JS, 128 × 128 → quelques ms). Translation pure : corrélation de phase classique. Zoom et rotation : extension log-polaire de Reddy & Chatterji (IEEE TIP 5(8):1266–1271, 1996). Sinon, flux optique KLT pyramidal (Bouguet 2000, implémenté dans jsfeat) sur 30 à 50 coins de Shi-Tomasi du cadre, suivi d'une homographie RANSAC.\[9\]\[25\]\[26\]
4. *Validation* : on reprojette le modèle et on calcule un **score de présence** (corrélation des bords du cadre et des triangles avec la référence). Sous le seuil, la calibration est « perdue » : bandeau d'alerte non bloquant (« Plateau déplacé ? Recalibrer / Ignorer »), suspension de l'auto-détection et retour au mode manuel.

Le pas de 1 s et les seuils de ±16 px et 3 images sont des choix de conception à valider, pas des valeurs publiées, sauf la règle des 2–3 images de VideoKifu. Côté navigateur, `requestVideoFrameCallback` est disponible dans Chrome 83 (mai 2020), Safari 15.4 (14 mars 2022) et Firefox 132 (29 octobre 2024), donc « Baseline » depuis le 29 octobre 2024.\[27\] web.dev précise qu'il s'agit d'un « best effort » qui peut être décalé d'une image.\[28\] Pour un échantillonnage ~1 Hz, ce n'est pas gênant.

### 4. Perspective forte, angles et changements de plan

**Angles typiques.** D'après les sources de captation trouvées :
- La FFBG (janvier 2022) recommande une caméra sport sur un support Elgato Multi Mount pour « filmer à la verticale sans déformation marquée », option « recommandée pour le streaming ». Les deux joueurs enregistrent.\[8\]
- BMAB (page mise à jour le 7 juin 2024) exige des vidéos non montées et publiques, mais ne fixe ni angle ni résolution sur les pages consultées. Le document « match recording guidelines » qu'elle annonce n'a pas été trouvé.\[7\]
- L'ANZBGF (2023) exige que les noms des joueurs, l'horloge et la feuille de score soient visibles.\[29\]
- Le guide UKBGF (non daté) décrit une caméra d'action sur une potence, avec beaucoup de marge autour du plateau, et conseille d'éviter les reflets et de choisir des dés contrastés.\[30\]

On attend donc trois cas : vue zénithale (majoritaire en BMAB et en club), vue oblique derrière un joueur (retransmissions, vues de spectateurs), et incrustation double caméra (régies de tournoi). **Aucune statistique de répartition n'a été trouvée.**

**Seuil d'obliquité : non trouvé dans la littérature backgammon.** L'étude Kinovea (Puig-Diví et al., PLoS One, 2019) mesure une fiabilité acceptable de 90° à 45°, mais meilleure à 90°.\[6\] VideoKifu (2018, fig. 2 : image 1440×1080 de la partie Metta-Martinelli, Bologne 2018, rectifiée en 500×500) montre que rectifier une vue oblique étire le côté lointain : « Stretching too much would blur the borders of the stones, making them invisible to the edge detector (Canny filter) ». Il faut donc un **critère mesurable plutôt qu'un angle**. Pour chaque point, on calcule le jacobien de H au centre de la flèche, ce qui donne la taille locale d'un pion en pixels source, s_min et s_max. On peut ensuite afficher deux alertes (heuristiques à valider) : « lecture fiable » si un pion lointain fait ≥ 16 px de diamètre et si l'anisotropie (grand axe / petit axe de l'ellipse projetée) est ≤ 2,5 ; « points lointains incertains » sinon, avec marquage des flèches concernées en orange dans la surimpression. Cet indicateur couvre en même temps la résolution, le zoom et l'obliquité.

**Vidéo multi-angles.** Les détecteurs génériques de changement de plan sont des réseaux 3D-CNN : TransNet V2 (arXiv 2008.04838, 2020), AutoShot (CVPR 2023).\[31\]\[32\] Une prépublication de 2026 (OmniShotCut, arXiv 2604.24762) rapporte des F1 de 0,75–0,82 pour ces méthodes sur son banc d'essai, avec des faiblesses sur les fondus.\[33\] C'est trop lourd et trop imprécis pour un besoin qui est en fait binaire : « la vue calibrée est-elle à l'écran ? ». Recommandation : le **score de présence** du §3, calculé à ~2 Hz, avec hystérésis (entrée > 0,7, sortie < 0,5, seuils à calibrer). En parallèle, un détecteur de coupe très bon marché : distance d'histogramme de luminance 32 classes entre images successives, utilisé seulement pour accélérer le passage à « vue perdue ». Pour l'incrustation, on permet **plusieurs calibrations nommées par vidéo** (« plein écran », « incrustation »). À la reprise après une coupe, on teste chaque calibration mémorisée et on garde la meilleure. Cela gère aussi le retour d'un plan de régie à un cadrage légèrement différent.

### 5. Occlusions

Résultats primaires :
- Carta & Corsolini (2015) : 25 % des images d'une partie réelle sont perturbées, jusqu'à 6 consécutives. Leur critère n°1 est la différence entre images consécutives sans perturbation.\[2\]
- Un article IEEE sur le suivi d'échecs par webcam zénithale (IEEE Xplore 6727033, date exacte non vérifiée) évite les images où un joueur masque le plateau par un « seuillage hystérétique sur n images des images de différence ».\[5\]
- VICE (Zhu, 2023) échantillonne des paires d'images avant et après le coup et détecte les objets « changement » sur la différence.\[34\]
- Des dépôts récents d'échecs vidéo (YOLOv11, 2024–2025) excluent les images où un détecteur de mains se déclenche.\[4\]
- VideoKifu (2018) note que « In most cases it takes three frames to detect a stone, that is about half to one second », et que sa routine de contrôle exige « Five frames instead of the usual three ». Dans la partie Hueber-Fanti (Bologne 2018), « one stone inside the bowl appeared in the frames as if it was over the goban ».

**Masque recommandé (sans réseau de neurones).**
1. **Activité** : différence absolue entre l'image courante et une référence glissante, sur l'aire rectifiée sous-échantillonnée (par exemple 64 × 48). Un bloc est « actif » si sa différence dépasse un seuil.
2. **État calme** = aucune activité dans l'aire de jeu pendant T_calme (0,7–1,0 s, cohérent avec VideoKifu) **et** retour de la similarité hors cases « changées ».
3. **Peau** : Phung, Bouzerdoum & Chai (IEEE TPAMI 27(1):148–154, 2005) trouvent que le classifieur bayésien par histogramme et le MLP sont les meilleurs, et que l'espace colorimétrique choisi influe peu sur la performance.\[35\]\[36\] Un seuil fixe en CbCr reste un indice d'appoint, peu fiable avec des feutres ou des pions beiges ou rouges. Utiliser la peau seulement pour *prolonger* l'état « occupé », jamais pour le lever.
4. **Gobelet et videau posés** : ce sont des objets immobiles, donc invisibles pour l'activité. On les traite comme des « changements persistants hors des zones attendues ». Si un changement se trouve ailleurs que dans les zones touchées par un coup légal, on marque ces zones « incertaines » et on demande confirmation.
5. **Paire de comparaison** = dernière image calme avant le lancer contre première image calme après le dernier mouvement. C'est l'approche « minimale » : on ne score que les coups légaux candidats.

### 6. Multi-plateaux

**Aucune source primaire traitant spécifiquement des pièges à deux tables n'a été trouvée.** Les risques ci-dessous sont des inférences d'ingénierie, à tester :
- (a) La relocalisation par gabarit peut « sauter » sur l'autre table, identique. Il faut borner la fenêtre de recherche à ±25 % de la position précédente, et ne jamais faire de recherche globale sans confirmation de l'utilisateur.
- (b) Les mains ou le gobelet de l'autre table déclenchent l'activité. Il faut limiter le masque d'activité à l'aire calibrée plus une marge de 0,5 d.
- (c) La régie peut alterner entre les tables : le score de présence le détecte comme une « vue perdue ».
- (d) Orientation : sur la seconde table, le joueur suivi peut être assis de l'autre côté. Il faut faire choisir explicitement le sens de numérotation (quel coin est le 1-point du joueur du bas) et le vérifier sur la position initiale.
- (e) En vue oblique, la table du fond peut avoir une résolution insuffisante : l'indicateur du §4 le dira.

### 7. Ergonomie de la calibration

- **Kinovea** (doc 0.9.5) : la calibration par plan consiste à ajouter une « perspective grid » et à déplacer ses coins sur un rectangle connu. La grille est déplaçable par poignées circulaires. CTRL permet de l'étendre « en gardant le même plan virtuel » (aide 0.8.15). Depuis 0.8.20 (janvier 2013), la grille est aussi « trackable » (suivie dans la vidéo). L'étude de validité de 2019 plaçait les coins avec un **zoom de 600 %**.\[6\]\[37\]\[38\]\[39\]
- **Go** : GOimage2SGF/GEM propose une calibration automatique (4 pierres posées aux coins, échantillonnage des couleurs) et une calibration interactive qui « ajuste manuellement les coins ». Carta & Corsolini prévoient le pointage manuel des 4 coins en dernier recours.\[2\]\[40\]
- **Sport** : les cadres TVCalib et KaliCalib évaluent par **reprojection du modèle de terrain** sur l'image, ce qui est l'équivalent automatique de la surimpression.\[41\]\[42\]

**Conception recommandée pour blunderDB (≈ 10 s pour l'utilisateur).**
1. On met la vidéo en pause sur une image de position initiale (début de partie), proposée par défaut.
2. Quatre clics guidés, dans un ordre affiché (« coin du 1-point du joueur du bas », etc.). À chaque clic, une loupe ×4–6 de 120 px suit le curseur. Les poignées peuvent sortir du cadre.
3. Après le 4ᵉ clic, H est calculée aussitôt et la surimpression apparaît :
   - contours des 24 triangles en alternance de 2 couleurs, barre et zones de sortie en pointillés ;
   - **30 ellipses fantômes** de la position initiale 2-5-3-5 (2 + 5 + 3 + 5 = 15 par camp), colorées par camp ;\[43\]
   - flèches « incertaines » en orange selon l'indicateur de résolution.
4. L'utilisateur juge d'un coup d'œil : les fantômes doivent coiffer les vrais pions. Une touche (barre d'espace) fait clignoter la surimpression à 2 Hz pour révéler les décalages.
5. Raffinement automatique par les bords (§1), puis affichage du déplacement des coins en px. L'utilisateur accepte ou annule.
6. Option : cliquer les 2 zones de dés (rectangles) ; sinon on prend par défaut les deux moitiés de l'aire de jeu.
7. On stocke localement : H, les points cliqués, les imagettes de référence et l'identifiant de la vidéo (hash du fichier et durée). Rien n'est exporté.

## Recommendations

1. **Verdict** : faisable entièrement dans la webview, sans dépendance native. La géométrie (DLT normalisée, modèle rectifié, corrélation de phase/KLT) tient en quelques centaines de lignes de JS. Le risque principal n'est pas géométrique. Il est (a) dans l'accès fiable aux pixels vidéo, en particulier sous WebKitGTK, et (b) dans les vidéos obliques ou à faible résolution. Le meilleur levier de fiabilité est d'utiliser les coups légaux comme hypothèses.
2. **Recommandation technique** :
   - Homographie : DLT normalisée de Hartley, 4 à 12 points, poignées hors cadre, contrôle de convexité, raffinement par profils de gradient perpendiculaires aux bords.
   - Modèle : repère en unités de diamètre de pion ; 28 zones (24 flèches, 2 barres, 2 sorties) ; bandes d'échantillonnage élargies de 0,3 d vers la caméra ; longueur occupée L(n) pour n ≤ 15.
   - Lecture : différence entre images calmes avant et après le coup, restreinte aux zones touchées par les coups légaux, score par candidat et top-3 présenté à l'utilisateur.
   - Dérive : NCC de 4 à 8 imagettes à ~1 Hz ; acceptation après 3 images concordantes ; birapport complexe pour distinguer un zoom d'un changement de perspective ; relocalisation par corrélation de phase (log-polaire pour le zoom) ; alerte non bloquante.
   - Plans : score de présence avec hystérésis, plusieurs calibrations nommées par vidéo.
   - Accès aux images : `requestVideoFrameCallback` + `drawImage` vers un canevas (OffscreenCanvas/WebGL2 pour le redressement). WebCodecs n'est pas nécessaire. WebKitGTK annonce le décodage WebCodecs via GStreamer depuis 2.42 (présentation Igalia), mais un dépôt tiers de 2026 signale un plantage de `WebKitWebProcess` dans `VideoDecoder::create`. Il exigerait aussi un démultiplexage en JS (MP4Box.js).\[44\]\[45\]
3. **Surimpression** : triangles + barre + zones de sortie + 30 fantômes de la position initiale + carte de résolution + clignotement. C'est la vérification « en une seconde » demandée.

## Caveats

**Incertitudes et expériences à mener**
- *Accès aux pixels sous WebKitGTK* : un bug historique (WebKit 159405, 2016) empêchait de peindre une vidéo dans un canevas quand l'accélération du compositing était active.\[46\] Une correction récente (WebKit PR 78459, bug 329101) traite une lecture de canevas erronée après un changement de codec.\[47\] **Expérience** : sur les 3 moteurs, lire 200 images avec `drawImage` + `getImageData` sur des fichiers H.264/VP9/AV1, vérifier qu'elles diffèrent et que le timing (`mediaTime`) est cohérent, et mesurer le coût par image en 1080p.
- *Seuil d'obliquité* : non trouvé. **Expérience** : filmer une position connue (initiale plus 5 positions types) à 90°, 70°, 55° et 40°, en 720p et 1080p. Mesurer le taux de flèches correctement comptées en fonction de la taille de pion lointain calculée par H, et fixer les seuils de l'indicateur à partir de la courbe.
- *Empilements au-delà de 5* : aucune convention trouvée. **Expérience** : annoter 50 images issues de vidéos BMAB publiques avec des piles de 6 à 10 pions et mesurer L(n) réel.
- *Seuils de dérive (±16 px, 3 images, 1 Hz)* : ce sont des choix de conception. **Expérience** : injecter des translations et zooms synthétiques (1–5 %) dans 10 vidéos et mesurer faux positifs et délai de détection.
- *Dimensions « officielles »* : contradictoires et non sourcées. Ne pas s'y fier.
- *Pièges multi-tables* : aucun retour d'expérience publié n'a été trouvé. Les risques listés sont inférés.
- *Règles BMAB de cadrage* : le document « match recording guidelines » n'a pas été trouvé. Les angles réels des vidéos de transcription restent à échantillonner.

**Sources (numérotées ; fiabilité)**
1. Hartley R. I., « In Defense of the Eight-Point Algorithm », IEEE TPAMI 19(6):580–593, juin 1997, DOI 10.1109/34.601246. Normalisation (centrage, √2) pour le conditionnement de la DLT. Primaire.
2. Fan B., Dai Y., Seo Y., He M. (Northwestern Polytechnical University / Sogang University), « A Revisit of the Normalized Eight-Point Algorithm and A Self-Supervised Deep Solution », arXiv 2304.10771, 2023. Proposition 1 : aucune normalisation n'atteint un conditionnement de 1. Primaire (prépublication).
3. Dubrofsky E., Woodham R. J., « Combining Line and Point Correspondences for Homography Estimation », ISVC 2008. DLT étendue aux droites. Primaire (consulté via un extrait).
4. Wölflein G., Arandjelović O., « Determining Chess Game State from an Image », J. Imaging 7(6):94, 2 juin 2021, DOI 10.3390/jimaging7060094 ; arXiv 2104.14963 ; code chesscog.\[48\] Rectification RANSAC, occupation par case, 0,23 % d'erreur. Primaire.
5. Carta A., Corsolini M., « A New Approach to an Old Problem: the Reconstruction of a Go Game through a Series of Photographs », arXiv 1508.03269, 2015. Micro-recalibration, birapport complexe, parallaxe, 65/269 images occultées, approche minimale. Primaire.
6. Carta A., Corsolini M., « VideoKifu, or the automatic transcription of a Go game », arXiv 1807.01577, juillet 2018. Suivi ±1 rayon, acceptation après 2–3 images, ~2 s, Hough circulaire, limites d'obliquité. Primaire.
7. Quintana D. M. et al., « LiveChess2FEN », arXiv 2012.06858, décembre 2020 (+ code). Détection de plateau accélérée, < 1 s par position. Primaire.\[49\]
8. « Chess Position Recognition using Faster R-CNN », arXiv 2610.09191, 2026. 8 points-clés du pourtour pour une homographie redondante ; occlusions en vue basse. Primaire (prépublication).
9. Zhu K., « VICE: View-Invariant Chess Estimation », SciTePress, 2023. Paires d'images avant/après, différence, occlusion par la main. Primaire.
10. « Chess move tracking using overhead RGB webcam », IEEE Xplore 6727033 (date non vérifiée). Seuillage hystérétique sur n images pour exclure l'occlusion. Primaire (résumé seul).
11. Marom O., « A General Retrieval-Augmented Generation Framework for Multimodal Case-Based Reasoning Applications » (MCBR-RAG), arXiv 2501.05030, 2025, section backgammon. 4 repères suffisent ; CNN de comptage par partie ; 100 000 images simulées ; diagrammes de livres (Magriel, Robertie, Trice). Primaire (prépublication).
12. jvitors23/backgammon-checker-detection-openCV, GitHub, non daté. Rectification + Hough ; 90 % de flèches / 40 % de positions justes. Dépôt, secondaire.
13. alexlitz/video_analysis, Tranks27/BackgammonTracking, Trackgammon (Backgammon Galaxy), non datés (rapport du sous-agent). Lecture vidéo sans calibration documentée. Dépôt/commercial, secondaire.
14. Theiner J., Ewerth R., « TVCalib », WACV 2023, arXiv 2207.11709. Recalage de terrain ; segments plus fiables que les points-clés. Primaire.
15. Kinovea, documentation 0.9.5 « Calibration mechanisms » ; aide 0.8.15 ; forum 0.8.20 (12 janvier 2013). Grille de perspective, poignées, CTRL, grille trackable. Éditeur, primaire.
16. Puig-Diví A. et al., « Validity and reliability of the Kinovea program… in 4 perspectives », PLoS One, 2019 (PMC6550386). Zoom 600 %, fiabilité 90°–45°, meilleure à 90°. Primaire.
17. Reddy B. S., Chatterji B. N., « An FFT-based technique for translation, rotation, and scale-invariant image registration », IEEE TIP 5(8):1266–1271, 1996, DOI 10.1109/83.506761. Corrélation de phase log-polaire. Primaire.
18. Bouguet J.-Y., « Pyramidal Implementation of the Lucas Kanade Feature Tracker », Intel/Stanford, 2000 ; jsfeat (GitHub). KLT pyramidal, homographie/RANSAC en JS. Primaire / dépôt.
19. Souček T., Lokoč J., « TransNet V2 », arXiv 2008.04838, 2020 ; AutoShot, CVPR 2023 ; OmniShotCut, arXiv 2604.24762, 2026. Détection de plans ; F1 0,75–0,82. Primaire.
20. Phung S. L., Bouzerdoum A., Chai D., « Skin Segmentation Using Color Pixel Classification », IEEE TPAMI 27(1):148–154, 2005, DOI 10.1109/TPAMI.2005.17. Bayes/histogramme et MLP meilleurs ; espace colorimétrique peu déterminant.\[36\] Primaire.
21. caniuse / web-features explorer / web.dev, « requestVideoFrameCallback » (Chrome 83, Safari 15.4, Firefox 132 ; Baseline 29 octobre 2024 ; « best effort »). Éditeurs/documentation, primaire.
22. web-features explorer « WebCodecs » (Safari 26, 15 septembre 2025 ; Firefox 130)\[50\] ; Normand P. (Igalia), « WebCodecs in WebKit, with GStreamer! » (WebKitGTK 2.42) ; Zeitgleich/TeamCraft issue #824 (plantage de VideoDecoder sous WebKitGTK 4.1). Primaire / dépôt.
23. WebKit Bugzilla 159405 (GTK, vidéo vers canevas avec compositing accéléré, 2016) ; WebKit PR 78459 / bug 329101 (lecture de canevas après changement de codec).\[46\]\[47\] Primaire.
24. BMAB, « About » (bgmastersab.com, mis à jour le 7 juin 2024) ; ANZBGF « About BMAB » (2023) ; FFBG, page tournoi BMAB (8–9 janvier 2022)\[8\] ; UKBGF « Recording your Matches » (non daté). Règles et pratiques de captation. Primaire (fédérations) / guide.
25. Crisloid « Backgammon FAQ » ; New England Backgammon Club « Backgammon Sets » ; sites marchands sur les « dimensions WBF » (contradictoires). Pion de 1,75″.\[13\] Blog/commercial, faible.
26. GOimage2SGF/GEM (GitHub) ; Roboflow « Backgammon Board Scan » (57 images). Calibration interactive ; jeu de données. Dépôt, secondaire.

## Sources

1. [Normalization in DLT triangulation methods (3). · Amy Tabb](https://amytabb.com/tips/tutorials/2021/11/04/normalization-DLT/)
2. <https://arxiv.org/pdf/1508.03269>
3. <https://arxiv.org/pdf/1807.01577>
4. [GitHub - oliverfrost1/chess-video-move-detection: Deep learning project to automatically detect chess moves from a video input using YOLOv11. · GitHub](https://github.com/oliverfrost1/chess-video-move-detection)
5. [Chess move tracking using overhead RGB webcam](https://ieeexplore.ieee.org/abstract/document/6727033/)
6. [Validity and reliability of the Kinovea program in obtaining angles and distances using coordinates in 4 perspectives - PMC](https://pmc.ncbi.nlm.nih.gov/articles/PMC6550386/)
7. [About this site - BMAB](https://bgmastersab.com/about)
8. [FFBG](https://www.ffbg.fr/competitions/live/204-bmab-backgammon-masters-awarding-body)
9. [GitHub - patricio272/jsfeat: JavaScript Computer Vision library. · GitHub](https://github.com/patricio272/jsfeat)
10. [(PDF) Combining Line and Point Correspondences for Homography Estimation](https://www.researchgate.net/publication/220845575_Combining_Line_and_Point_Correspondences_for_Homography_Estimation)
11. [TVCalib: Camera Calibration for Sports Field Registration in Soccer](https://arxiv.org/pdf/2207.11709)
12. [How to Setup a Backgammon Board: Official Tournament Rules (2026)](https://bacobackgammon.com/blogs/news/how-to-setup-backgammon-board-rules)
13. [How to Make Backgammon Boards - Our Pastimes](https://ourpastimes.com/make-backgammon-boards-5403839.html)
14. [Backgammon FAQS](https://www.crisloid.com/backgammon-faq-learn-about-points-standard-sizes-strategy-and-more/)
15. [Backgammon Sets](https://nebackgammon.org/index.php/backgammon-sets/)
16. [Standard Backgammon Board Dimensions: A Complete Guide for Players](https://woodronic.com/blogs/learn-with-us/standard-backgammon-board-dimensions-a-complete-guide-for-players)
17. [Backgammon Board Dimensions: Standard & Tournament Size Guide (2024) - Sensazioni.org](https://sensazioni.org/backgammon-board-dimensions-guide)
18. [Backgammon Board Dimensions](https://www.backgammonindia.com/backgammon_board_dimensions/)
19. [Backgammon Board Guide: Sizes, Checkers & What's In The Box](https://tabletopwire.com/backgammon-board-guide/)
20. [Chess Position Recognition using Faster R-CNN](https://arxiv.org/html/2610.09191)
21. [Backgammon Board Scan Object Detection Dataset by BgScan](https://universe.roboflow.com/bgscan/backgammon-board-scan)
22. [GitHub - jvitors23/backgammon-checker-detection-openCV: Backgammon checker detection openCV. · GitHub](https://github.com/JvitorS23/backgammon-checker-detection-openCV)
23. [GitHub - alexlitz/video\_analysis](https://github.com/alexlitz/video_analysis)
24. [Trackgammon - Backgammon Match Transcription](https://www.backgammongalaxy.com/trackgammon)
25. [Pyramidal Implementation of the Lucas Kanade Feature Tracker](http://robots.stanford.edu/cs223b04/algo_tracking.pdf)
26. [(PDF) An IDL/ENVI implementation of the FFT-based algorithm for automatic image registration](https://www.researchgate.net/publication/47503503_An_IDLENVI_implementation_of_the_FFT-based_algorithm_for_automatic_image_registration)
27. [Web platform features explorer - requestVideoFrameCallback()](https://web-platform-dx.github.io/web-features-explorer/features/request-video-frame-callback/)
28. [Perform efficient per-video-frame operations | Articles | web.dev](https://web.dev/articles/requestvideoframecallback-rvfc)
29. [About BMAB -](https://anzbgf.org/about-bmab/)
30. [Recording your Matches - UKBGF](https://ukbgf.com/members/articles/recording-your-matches/)
31. [\[2008.04838\] TransNet V2: An effective deep network architecture for fast shot transition detection](https://arxiv.org/abs/2008.04838)
32. [\[2304.06116\] AutoShot: A Short Video Dataset and State-of-the-Art Shot Boundary Detection](https://ar5iv.labs.arxiv.org/html/2304.06116)
33. [OmniShotCut: Holistic Relational Shot Boundary Detection with Shot-Query Transformer](https://arxiv.org/html/2604.24762v2)
34. [VICE: View-Invariant Chess Estimation Kevin Zhu](https://www.scitepress.org/Papers/2023/121672/121672.pdf)
35. [(PDF) Skin Segmentation Using Color Pixel Classification: Analysis and Comparison](https://www.researchgate.net/publication/8103235_Skin_Segmentation_Using_Color_Pixel_Classification_Analysis_and_Comparison)
36. ["Skin segmentation using color pixel classification: analysis and compa" by Son Lam Phung, A. Bouzerdoum et al.](https://ro.uow.edu.au/infopapers/256/)
37. [Calibration mechanisms — Kinovea 0.9.5 documentation](https://www.kinovea.org/help/en/measurement/calibration.html)
38. [Experimental version - 0.8.20 (Page 1) — General — Kinovea - Forums](https://www.kinovea.org/en/forum/viewtopic.php?id=664)
39. [Kinovea - Using overlay grids](https://www.kinovea.org/help/0.8.15/en/103.html)
40. [GitHub - nickhuang99/GOimage2SGF: using GO board image to generate SGF file to record game play](https://github.com/nickhuang99/GOimage2SGF)
41. [TVCalib: Camera Calibration for Sports Field Registration ...](https://mm4spa.github.io/tvcalib/)
42. [KaliCalib: A Framework for Basketball Court Registration](https://ar5iv.arxiv.org/html/2209.07795)
43. [Backgammon Board Setup: Complete Visual Guide (2026)](https://backgammonhit.com/articles/backgammon-board-setup/)
44. [WebCodecs in WebKit WebCodecs in WebKit With GStreamer! With GStreamer!](https://indico.freedesktop.org/event/5/contributions/170/attachments/107/165/WebCodecs%20in%20WebKit,%20with%20GStreamer!.pdf)
45. [Linux desktop: WebKitWebProcess crashes when a screen share starts; the window stays grey while the call goes on · Issue #824 · Zeitgleich/TeamCraft · GitHub](https://github.com/Zeitgleich/TeamCraft/issues/824)
46. [159405](https://bugs.webkit.org/show_bug.cgi?id=159405)
47. [\[GStreamer\] Canvas readback of a \<video\> fails after a codec switch: paint() wraps the previous frame with the new caps' video info by ehfd · Pull Request #78459 · WebKit/WebKit · GitHub](https://github.com/WebKit/WebKit/pull/78459)
48. [\[2104.14963v1\] Determining Chess Game State From an Image](https://arxiv.org/abs/2104.14963v1)
49. [LiveChess2FEN: a Framework for Classifying Chess Pieces based on CNNs](https://deepai.org/publication/livechess2fen-a-framework-for-classifying-chess-pieces-based-on-cnns)
50. [web-platform-dx.github.io](https://web-platform-dx.github.io/web-features-explorer/features/webcodecs)
