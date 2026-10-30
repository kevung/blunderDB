# Synthèse de la bibliographie — assistant de transcription par la vidéo

Une section par réponse, dans l'ordre des prompts : ce qu'elle confirme, ce qu'elle change au
cadre du `README.md`, les expériences qu'elle impose. La feuille de route et les ADR se tirent d'ici,
pas des réponses brutes.

## 01 — Accès aux images dans les trois webviews

**Confirme le cadre.** Lire les pixels d'un `<video>` est faisable sur WebKitGTK, WebView2 et
WKWebView : `drawImage` + `getImageData`, `crossorigin="anonymous"` posé avant `src`, et
`Access-Control-Allow-Origin` sur **toutes** les réponses du loopback — 200, 206 et OPTIONS — sans
redirection. Un seul 206 sans en-tête teinte le canvas ; l'erreur n'apparaît qu'à `getImageData`
(`SecurityError`), jamais à `drawImage`. `requestVideoFrameCallback` existe partout dans nos
versions (WebKitGTK ≥ 2.36) ; l'horodatage à garder est `metadata.mediaTime`, pas `currentTime`.

**Change le cadre.**
- *La pré-passe « 1 h en 5 min à 10 img/s » n'existe pas.* Aucun moteur ne tient 120 images/s par
  `seek` (meilleur cas Chromium : 10–19 ms par seek en 720p, soit 400–700 s pour 36 000 images) ni
  par `playbackRate` (plafonné par le compositeur à ~60 rappels/s ; instable sous GStreamer, bugs
  245850/245852 encore vus en 2.42.4). Remplacer par une **analyse hiérarchique** : passe grossière
  à 1–2 img/s (seeks séquentiels, un seul en vol ; ou WebCodecs sur images-clés là où
  `VideoDecoder.isConfigSupported` accepte le H.264 du fichier), puis passe dense à 10 img/s sur
  ±5 s autour des événements. Les lancers durent plusieurs secondes : 1–2 img/s suffit à les
  repérer. La passe grossière peut aussi être opportuniste, pendant que l'utilisateur regarde à 1×.
- *Pas de mémoire partagée.* `SharedArrayBuffer` est absent de WebKitGTK 2.42–2.48 (activé en
  septembre 2026 seulement, et sous origine isolée, que le schéma Wails ne donne pas). Un worker
  monothread par tâche, transfert d'`ArrayBuffer` par `postMessage` ; pas de threads Wasm.
- *WebKitGTK ≥ 2.46.4 pour le chemin confortable* : `OffscreenCanvas` par défaut (2.46.0), WebCodecs
  actif par défaut (2.46 selon Igalia, annoncé en 2.44 — contradiction à lever par détection),
  correctifs mémoire de `getImageData` (2.46.4). En 2.42/2.44 : réduction d'image sur le thread
  principal, worker classique. Toujours par détection de fonctionnalité, jamais par version.
- *Le `<video>` ne se transfère pas* : `drawImage` vers un canvas 320×180 reste sur le thread
  principal (~230 Ko par image, négligeable à 10 img/s) ; le worker ne reçoit que le tampon réduit.
  Sous WebKitGTK 2.46, `willReadFrequently: true` force un canvas non accéléré : mesurer.

**Non retenu.** La réponse recommande de servir la vidéo par l'AssetServer Wails (même origine,
`Range` depuis v2.3.0). Le dépôt a déjà tranché l'inverse : le pipeline média de WebKitGTK ne fait
pas de requêtes partielles à travers un schéma d'URL personnalisé (ADR-0082, contexte ;
`internal/gui/media.go`). La réponse elle-même note que ce point n'est pas documenté. On garde le
loopback et on lui ajoute CORS.

**Risque découvert, indépendant de la vision.** Chromium ≥ 142 (octobre 2025) soumet à permission
les requêtes d'une page vers le loopback (*Local Network Access*, permissions `local-network` /
`loopback-network`, Chrome 147 étend aux WebSockets). Le comportement de WebView2 depuis
`http://wails.localhost` vers `http://127.0.0.1:port` n'est pas documenté : la lecture vidéo
actuelle sous Windows peut déjà être concernée. À mesurer en priorité, hors de ce chantier si
nécessaire (événement `PermissionRequested` de WebView2 comme repli).

**Expériences imposées (bancs à écrire avant la phase 1).**
1. CORS sur 206 : un MP4 de 1 Go en loopback, en-tête retiré sur un 206 sur deux, `getImageData`
   lève-t-il `SecurityError` sur les trois moteurs ; puis une redirection 302 en cours de lecture.
2. LNA WebView2 : invite ou blocage vers `127.0.0.1` selon la version d'Edge (≥ 142, ≥ 146).
3. Banc de seeks : H.264 1080p30, image-clé toutes les 2 s, 500 seeks précis séquentiels au pas de
   0,5 s ; mesurer `seeked` → rVFC (p50/p95) par moteur ; `fastSeek` sous WKWebView (absent de
   Chromium, cassé en MSE sous GStreamer).
4. Banc `playbackRate` 2×–16× : rappels rVFC/s, `droppedVideoFrames`, monotonie de `mediaTime`.
5. WebCodecs : `typeof VideoDecoder` et `isConfigSupported({codec:'avc1.640028'})` sur Ubuntu
   22.04/24.04, Fedora, Flatpak GNOME.
6. Coût de 10 000 cycles `drawImage` 1080p → 320×180 + `getImageData`, avec et sans
   `willReadFrequently`, sur les trois moteurs.
7. Énergie : une heure à 1× + 10 img/s contre une pré-passe de 5 min, sur batterie.

**Conséquence pour les repères (ADR-0082).** `mediaTime` de rVFC est l'instant de l'image composée ;
c'est lui qui doit alimenter `roll_tick_ms` / `tick_ms` quand la détection pose un repère, à la
place de `currentTime` qui avance entre deux images.

## 02 — État de l'art : dés et plateau de backgammon

**Confirme le cadre.** Discriminer entre candidats (phase 3) plutôt que reconnaître une position est
le consensus des systèmes vidéo d'échecs qui marchent, 2008–2025 : Piškorec (N = 8 images
statiques), Wang & Green (hystérésis, coups fournis par un moteur), VICE (paires avant/après), Roman
(Brno 2025, YOLOv8, cases changées filtrées par les coups légaux) [1][2][3][4]. L'absolu ne
généralise pas : Chesscog passe de 93,86 % de plateaux exacts en synthétique à 2,30 % sur ChessReD ;
CVChess, de 63,96 % sur ChessReD à 29,8 % sur de nouvelles photos ; seul ChessQueries (ViT-L,
préprint du 31/08/2026, trop lourd pour une webview) atteint 87,1 % sur des retransmissions ; les LLM
multimodaux, 0–2,6 % [58]. En backgammon, homographie sur coins cliqués + HoughCircles : 90 % de
points justes, 40 % de plateaux exacts [42]. Le backgammon est plus favorable (deux couleurs, 2 à 4
points touchés, liste fermée) mais les piles se chevauchent et dés et videau masquent des pointes.
`<video>` + rVFC + canvas suffit, quelques images stables par coup : cohérent avec la passe dense
ciblée de la section 01.

**Change le cadre.**
- *Aucune mesure publiée sur du backgammon filmé.* Quatre à six dépôts amateurs en vue de dessus
  (nitzankoll : YOLOv5, 30 images au départ, HoughCircles abandonné [40][41] ; Roboflow : 49 images,
  dés détectés mais non lus [43]) ; Marom (arXiv 2501.05030) lit des diagrammes de livres [6] ; les
  applications (GammonSnap, Backgammon Eye, Sage Snap) ne publient rien et corrigent au tapotement
  [7][8][11]. Toute précision promise par la phase 2 viendra de notre banc (expérience 1).
- *Non trouvé, et ce sont nos cas durs* : piles de plus de 5 pions, pions sortis, dé cocké, reflets,
  videau et ses gestes. Distinguer 6 de 7 pions est annoncé comme le cas le plus dur ; seule piste :
  classer le *compte* par tranche de pointe (Marom) plutôt que détecter chaque pion [6]. Le videau
  passe entier au prompt 09.
- *Les chiffres de la réponse sont des estimations* : ~95–97 % par dé en blobs seuls, ~98–99,5 %
  avec un CNN < 100 k paramètres (vignettes 32–64 px, vote sur 5–10 images) ; ~95–99 % top-1,
  > 99 % top-3 entre 2 et 30 candidats sans main. Bornes : 99,76 % en validation intra-distribution
  (WKU 2020) [26] ; nell-byler (250 photos, SSD-MobileNet) « does not really work well on new data »
  [24][25]. Hypothèses du banc, pas des cibles.
- *Aucun jeu public ne transfère* (aucun dé de backgammon filmé sur plateau). Seul koryakinp « D6
  Dice » (11 000 images, CC0, angle unique) est libre de toute question [31] ; Roboflow CC BY 4.0
  (441 et 132 images) exige l'attribution [36][37] ; le reste, licence non vérifiée. Le corpus maison
  (fichiers licites, section 07) reste la source ; le public sert au plus au préentraînement.
- *Pannes à intégrer au prompt 03* : main immobile prise pour un état stable [3] ; aucune image libre
  de main quand un joueur enchaîne avant que l'autre ait lâché [4] ; rangée proche du joueur peu
  couverte par la main, corrigée en élargissant le masque [61] ; le brevet US 10828556 lit les dés
  10–350 ms après leur arrêt et annule si une main entre avant [22].

**Non retenu.**
- *« Chaque coup légal classé par le réseau… présenter le top-3 classé »* (recommandation 2b) : le
  top-3 se classe au **seul score visuel** ; l'équité ne trie ni ne départage, sinon les blunders sont
  blanchis (README, ADR-0044). Reste permis : « aucun coup légal ne correspond au plateau observé →
  lancer suspect », qui repose sur la légalité.
- *Désactiver WebCodecs sous WebKitGTK* : la source du « plantage au décodage H.264 » [13] traite d'un
  plantage au partage d'écran ; le support WebCodecs s'appuie en outre sur une source secondaire [15].
  La règle de la section 01 (détection de fonctionnalité) tient ; un test de stabilité du décodage
  s'ajoute à son expérience 5.
- *ONNX Runtime Web (WASM) pour le classifieur* : affirmé sans banc, c'est le prompt 05. YOLO11-N
  (2,6 M paramètres, dépôt tiers [28]) est surdimensionné pour une vignette ; aucune précision
  YOLO-nano sur des dés trouvée.
- *Banc des applications mobiles sur 20 photos* : outils fermés, photo de dessus [11] ; hors sujet.

**Expériences imposées.**
1. Banc sur 5 vidéos locales licites (matériels, angles, lumières variés ; ~300 lancers, ~600 coups),
   vérité terrain tirée de la base (`roll_tick_ms` / `tick_ms`, dés, `board_after`) : précision par
   dé (blobs ; blobs + CNN), top-1/top-3 par nombre de candidats (2–5, 6–15, 16–30) et par hauteur de
   pile touchée (≤ 5, > 5).
2. Contrôle d'équité : top-1 visuel calculé à part sur les coups joués qui ne sont pas le premier du
   réseau ; il ne doit pas s'effondrer (sinon le score visuel s'appuie sur la plausibilité).
3. Piles hautes et sortie : sur images redressées, le profil le long de la pointe distingue-t-il 6 de
   7 pions, jusqu'à quel angle ; comptage 0–15 de la zone de sortie.
4. Dés : polarité auto (dés sombres à points clairs), circularité 0,60 sur dés en contact [18] ; part
   de dés cockés ou masqués dans le corpus et part **signalée** plutôt que lue.
5. CNN de dé sur vignettes maison seules contre préentraîné sur koryakinp ; écart sur une vidéo écartée.
6. Faux états stables (main immobile) et coups sans image libre de main sur le corpus ; apport d'un
   masque de main élargi.

**Conséquence pour la phase 3.** Tant que les expériences 1 et 3 n'ont pas mesuré le top-1 sur piles
hautes et pions sortis, le mode automatique propose une liste à confirmer, jamais un coup d'office.

## 03 — Détection d'événements : lancer, repos, mains, remise à zéro

**Confirme le cadre.** Phase 1 sans modèle appris : quatre descripteurs par image sur `Uint8Array`
(énergie de différence normalisée E_d en ROI dés, fraction d'avant-plan F_b en ROI plateau,
changement global G, distance census D_raz au gabarit de départ), puis trois automates couplés
(mains / dés / partie) avec hystérésis et durées minimales. C'est le schéma des suiveurs d'échecs :
portillon « hysteretic n-frame thresholding », puis comparaison aux coups **légaux** fournis par le
moteur (Wang & Green 2013 [S1]), ce qui valide la phase 3 sans équité. Rien de publié sur le
backgammon (Digigammon, TrackGammon : commerciaux, invérifiables). Flux optique dense et TCN
écartés : coût, pas de JS pur, des centaines de séquences annotées (MS-TCN F1@10 76,3 [S38]).

**Change le cadre.**
- *À 1–2 img/s, la passe grossière ne repère que des fenêtres.* Les durées sont en images à 10 i/s :
  *lancer* ≥ 1, *repos* 5 sous E_lo (0,5 s), *mains dans* 2, *mains hors* 3 ; à 5 i/s, diviser par
  deux, plancher 2. TCG-AR à ~4 i/s rate des poses (« can fall entirely between two recognition
  frames » [S44]) : 5–10 i/s est un minimum. Le « plusieurs secondes » de 01 vaut pour l'épisode
  entier, pas pour le roulement ni pour un *repos* suivi d'un ramassage immédiat. **Arbitrage** : 01
  l'emporte sur ce que le décodeur livre, 03 sur la résolution. La passe grossière compare des
  **états** (dés déplacés, plateau changé) : un ramassage rapide laisse quand même les dés
  ailleurs. Elle ouvre des fenêtres où les automates tournent à 10 i/s.
- *±5 s autour de chaque événement (01), c'est trop large.* Au backgammon, les événements se
  suivent à quelques secondes : la passe dense couvrirait presque toute la vidéo. Un pas de 0,5 s
  justifie [−1 ; +2] s, à chiffrer (exp. 1). Si la couverture reste haute, l'analyse opportuniste
  à 1× devient le chemin principal.
- *Census, pas de différence brute.* L'auto-exposition allume toute la ROI ; census/LBSP est
  invariant aux transformations monotones de l'intensité [S4][S5][S6]. Normaliser chaque ROI par sa
  moyenne, ajouter une **ROI témoin** hors plateau ; > 60 % de pixels changés à la fois → *incertain*
  et réinitialisation des fonds. Test d'ombre de MOG2 : I/B dans [0,5 ; 0,95] [S15].
- *Calibration et valeurs de départ.* 4 coins, ROI dés gauche/droite + une ROI large par moitié
  (dés sur la barre ou un pion), ROI témoin, masques d'overlay (auto : pixel actif > 30 % de la
  1re minute). Seuil pixel 3σ (MAD sur les périodes calmes), E_hi = 3 × la médiane calme,
  E_lo = 1,5 ×, fond α = 0,02 hors mains. *Repos* : deux blobs de la taille d'un dé (blob > 3× un dé :
  main) et centroïde à < 1 px. Garde de 4 s entre lancer et repos, mains > 20 s → *incertain*.
  **Rien n'est sourcé** : transpositions de ViBe (R = 20, φ = 16 [S16]), MOG2, DCASE, SoccerNet.
  « Non trouvé » : un critère chiffré de repos des dés et les paramètres de Wang & Green.
- *Évaluation.* F1 par événement, appariement un à un, tolérance ±0,5 s (lancer, repos, mains),
  ±3 s (remise à zéro) ; collar DCASE 200 ms [S39][S40], SoccerNet 1–5 s [S41]. Plus : décalage
  médian signé et IQR, score Edit, taux de coups sans correction ; découpage val/test **par
  match**. Cible proposée, non sourcée : F1 ≥ 0,95 à ±0,5 s, < 5 % de coups à corriger.

**Non retenu.**
- *« Qui a lancé » par moitié + appariement de blocs (SAD 8×8, ±4 px)* : le transcript connaît le
  trait. La réponse oublie le **jet d'ouverture** (un dé chacun, dans sa moitié) : à modéliser à part.
- *Remise à zéro de premier rang.* La fin de partie est connue du transcript (dernier coup,
  abandon, passe) ; la remise à zéro ne sert qu'à sauter au jet suivant, donc phase 3. Un gabarit
  « appris sur la première position de la vidéo » est faux si la vidéo commence en cours de
  partie : le prendre sur une image confirmée par l'utilisateur.

**Expériences imposées.**
1. Couverture, en SQL seul : intervalles `roll_tick_ms` → `tick_ms` → jet suivant sur les matchs à
   fichier local ; part de la vidéo sous des fenêtres de ±5 s et de [−1 ; +2] s. Décide la hiérarchie.
2. Journal (quelques Ko/min ; cache du fichier, hors base, jamais exporté, ADR-0007) : un match
   lu à 1×, worker à 10 i/s, {E_d, F_b, G, D_raz} en JSON par image. Rejouer les automates (pure,
   vitest) : N_repos ∈ {3, 5, 8}, E_hi/E_lo ∈ {2/1,2 ; 3/1,5 ; 4/2}, census ou non → F1@0,5 s, décalage.
3. Sous-échantillonner ce journal à 5, 2, 1 i/s : rappel des événements à ±0,5 et ±1 s, rappel
   des fenêtres à 1–2 i/s. Teste ensemble TCG-AR et la passe grossière de 01.
4. Annoter 2 matchs (≈ 1 h ; à terme 3–5, ≈ 2 h, 300–500 coups) au pas d'image, dont 10 % deux
   fois : l'accord à ±0,2 s fixe δ. Mesurer au passage l'écart entre Repères tapés et événements.
5. Stress global (auto-exposition, coupure, overlay, replay) : taux de *lancer* sans *repos* ni coup.
6. Temps par image en worker sur les trois moteurs (avec l'exp. 6 de 01).

**Conséquence pour les Repères (ADR-0082).** Un Repère tapé n'est pas un événement. L'instant du
jet = première touche de dé, donc *repos* + lecture + frappe ; l'instant de l'action = validation,
pas *mains hors*. (a) Le pré-remplissage de l'annotation et les couples (image, dés) du README
doivent corriger cet écart. Prendre l'image médiane du *repos* : à `roll_tick_ms`, les dés sont
peut-être déjà ramassés. (b) Un Repère posé par la détection ferait des mesures des durées de
pions estimées. Mais il change la sémantique d'une colonne jusqu'ici tapée : l'ADR doit trancher,
proposition confirmée (ADR-0044) ou marque de provenance. **`transcriptionKeys.js`** : les automates
forment une machine à part, pure, nourrie de descripteurs, jamais logée dans `pressKey`. Ils
émettent des propositions (saut, dés via `enterDicePair`) que le clavier reçoit comme des gestes.

## 04 — Calibration du plateau et robustesse géométrique

**Confirme le cadre.** La géométrie est un problème résolu et tient en quelques centaines de lignes de
JS sans bibliothèque : DLT à 4 points (système 8×8, h₃₃ = 1, Gauss à pivot partiel, ~60 lignes),
normalisée à la Hartley (centroïde à l'origine, distance moyenne √2) [S1] ; moindres carrés au-delà
de 4 points. Avec 4 clics, l'erreur de clic domine, pas l'arithmétique. Deux refus à coder : un
quadrilatère non convexe, un déterminant 2×2 de H qui change de signe. La discrimination entre
candidats (phase 3) est le bon choix : à 99 % par case, une lecture libre de 361 cases n'est juste
que dans 0,99³⁶¹ = 2,7 % des cas [S2] ; le seul dépôt backgammon chiffré fait 90 % de flèches pour
40 % de positions justes [S22]. On ne compare que les zones touchées par un coup légal, entre
dernière image calme avant le lancer et première image calme après le coup. Backgammon en vidéo :
aucune publication (« non trouvé ») ; modèles tirés du go (VideoKifu [S3]) et des échecs.

**Change le cadre.**
- *Le modèle du plateau est en unités de diamètre de pion d, jamais en dimensions absolues* : les
  « dimensions officielles WBF » des sites marchands se contredisent (21″×13″ contre 53×44 cm, pion
  de 2,8 cm contre 1,75″) [S12][S16][S17]. Repère rectifié ≈ 15×13 d, 28 zones (24 flèches, 2 barres,
  2 sorties), case k de la flèche à (k+0,5)·d du bord. On compte par **profil 1D** dans une bande de
  0,8 d le long de l'axe, débordant de 0,3 d et décalée vers la caméra (épaisseur ≈ 0,2 d × obliquité)
  pour la parallaxe [S2] — pas de recherche de centres de pions. Au-delà de 5 pions : convention
  **non trouvée** ; deux hypothèses (2ᵉ rangée décalée de d/2, ou pile allongée au-delà de la
  pointe), donc une longueur occupée L(n) monotone pour n ∈ [0, 15], pas des positions fixes.
- *Le critère de lisibilité est une taille en pixels, pas un angle* (seuil d'obliquité « non
  trouvé » ; Kinovea fiable de 90° à 45° [S6]). Jacobien de H au centre de chaque flèche :
  « fiable » si un pion lointain fait ≥ 16 px et une anisotropie ≤ 2,5 (heuristiques à valider),
  flèche en orange sinon. **Conséquence sur 01** (notre inférence) : le canvas 320×180 de 01
  suffit aux événements, pas à la lecture — ~13 d sur ~250 px donne d ≈ 19 px de dessus, moins en
  oblique. Lire le coup exige de rectifier la zone du plateau en pleine résolution (WebGL2 ou
  `drawImage` d'une région), sur les seules images calmes.
- *Changement de plan = chute d'un score de présence du plateau calibré* (~2 Hz, hystérésis
  > 0,7 / < 0,5 ; histogramme de luminance 32 classes en accélérateur), pas un détecteur de coupes
  (3D-CNN à F1 0,75–0,82 [S31][S33]). Touche 03 : « vue perdue » est un état de la machine.
- *Plusieurs calibrations nommées par vidéo* (« plein écran », « incrustation »), testées après
  chaque coupe — contre « une calibration par vidéo » du prompt. Orientation explicite : 1er clic =
  coin du 1-point du joueur du bas, vérifié sur la position initiale (piège multi-tables).
- *Gobelet et videau posés échappent à la différence d'images* (immobiles) : un changement
  persistant hors des zones d'un coup légal les marque « incertaines ». Occlusion : 65/269 images
  (jusqu'à 6 d'affilée) sur une partie de go [S2] ; on filtre, on ne lit jamais à travers. Calme =
  aucune activité sur l'aire rectifiée 64×48 pendant 0,7–1,0 s ; la peau ne fait que *prolonger*.

**Non retenu.** Le pas « WebCodecs n'est pas nécessaire ; décodage GStreamer depuis WebKitGTK 2.42 »
contredit 01 (2.46 selon Igalia) et cite un plantage de `VideoDecoder::create` sous WebKitGTK 4.1
[S45] : 01 l'emporte (détection de fonctionnalité). Pose 3D, Hough complète, dimensions absolues,
recherche globale par gabarit (saut sur la table voisine : fenêtre ±25 %). Le « top-3 » proposé est
classé par score image seul, jamais par équité (README).

**Expériences imposées** ([Sn] = liste finale « Sources » de la réponse, pas celle qui la précède).
1. Obliquité : position initiale + 5 positions à 90°, 70°, 55°, 40°, en 720p/1080p ; flèches justes
   selon la taille de pion lointain calculée par H ; seuils 16 px / 2,5 tirés de la courbe.
2. L(n) réel sur 50 images de vidéos BMAB à piles de 6 à 10 pions.
3. Dérive : translations et zooms synthétiques (1–5 %) injectés dans 10 vidéos ; faux positifs et
   délai. Paramètres à valider : NCC de 4–8 imagettes 32×32 à ~1 Hz dans ±16 px ; < 1 px rien ;
   1–16 px cohérents → nouvelle DLT après 3 images concordantes (seule règle publiée : 2–3 images,
   VideoKifu [S3]) ; birapport complexe des 4 coins modifié → zoom/recadrage → corrélation de phase
   sur miniature rectifiée 128×96 (log-polaire de Reddy-Chatterji pour l'échelle) ou KLT sur 30–50
   coins + RANSAC [S9][S25] ; présence basse → bandeau non bloquant « Plateau déplacé ? », manuel.
4. Raffinement par les bords : 20 profils de ±8 px par bord, max de gradient, droite robuste,
   intersections → coins sous-pixel ; gain sur clics bruités, plateau sans mains [S3].
5. Banc 6 de 01 : vérifier que 200 images lues diffèrent (bug 159405) et un changement de codec en
   cours de lecture (PR 78459) [S46][S47].
6. Échantillonner les angles réels : ni statistique ni guide de cadrage BMAB trouvés [S7].

**Conséquence pour la calibration et son stockage.** Parcours ≈ 10 s : pause sur une position
initiale, 4 clics guidés, loupe ×4–6 (Kinovea : 600 % [S6]), poignées hors cadre (canevas élargi
~20 %) ; surimpression des 24 triangles, barre, sorties, **30 pions fantômes** 2-5-3-5 et carte de
résolution ; espace = clignotement 2 Hz ; raffinement affiché en px, accepté ou annulé. Stockage
comme `frontend/src/services/videoRotation.js` (localStorage par `draftId`, jamais en base,
ADR-0007) : H, clics, imagettes (~8 Ko) et miniature (~12 Ko) par calibration nommée. La réponse
propose une clé par hash du fichier : on garde `draftId` (hacher 1 Go dans la webview coûte). Clics
en coordonnées intrinsèques de la vidéo, avant la rotation d'affichage.

## 05 — Inférence légère dans la webview

**Confirme le cadre.** Inférence dans un Web Worker monothread, sans dépendance : poids float16 au
transport, calcul float32 (la discipline du moteur Go, BGN6), imagettes par `postMessage` avec
`ArrayBuffer` transférable. Pas de threads Wasm même après septembre 2026 (01) : sous schéma
personnalisé, `crossOriginIsolated` est faux (WebKitGTK 2.50.4, `WKURLSchemeHandler`) [15][16]. Poids dans
`frontend/dist`, déjà dans le binaire par `embed.FS`, lus par `fetch()` GET [55] : 100 Ko (dés) et
1 Mo (plateau) en float16 sont invisibles ; la gêne commencerait au-delà de 10–20 Mo.

**Change le cadre.**
- *Plancher WebKitGTK réel : 2.50, pas 2.42.* Ubuntu 24.04 en 2.52.6, 22.04 en 2.50.4, Debian 13 en
  2.52.5, Debian 12 en 2.50.6, Arch en 2.54.1 (oct. 2026) [17]–[22] : les mises à jour de sécurité
  font suivre WebKitGTK. Le chemin « confortable » de 01 (≥ 2.46.4) est le cas normal ; le repli
  2.42/2.44 reste par détection mais sort des priorités de test.
- *Wasm SIMD (« non trouvé » en 01) est tranché.* JSC l'active sur x86-64 seulement si le CPU a
  l'AVX (PR #8049, janv. 2023) [9] ; WKWebView depuis Safari 16.4 (mars 2023) [10] ; WebView2 depuis
  longtemps. Sans AVX (Celeron, Pentium N, Atom) il est désactivé de force [11] : **repli JS
  obligatoire** pour tout noyau Wasm. Relaxed SIMD encore sous flag dans Safari [14].
- *Les coûts sont des estimations de la réponse, pas des mesures.* Dés (3→16→32→64) ≈ 2,8 M MAC, soit
  2–6 ms en JS à 1–3 GFLOPS (extrapolé ; seul repère publié : 0,6 GFLOPS, Firefox 28, 2014 [34]).
  Bande : 5–20 M MAC ; plateau entier (24 bandes + barre + sorties) 120–500 M, soit 100–500 ms.
- *Ces 100–500 ms ne gênent pas la phase 3 telle que cadrée.* Le plateau se lit une fois par coup,
  au repos après l'événement (01), et discriminer entre candidats ne demande que les points où les
  `board_after` diffèrent : 2 à 8 bandes, 10–160 M MAC, ≈ 10–150 ms en JS par décision. Le GEMM Wasm
  SIMD que la réponse impose devient un **accélérateur conditionnel**, pas un prérequis. L'int8
  n'apporte rien en JS (pas d'entiers vectoriels).

**Recommandation ferme.**
- *Phase 2 (dés)* : module JS ES2020 dans un Worker — conv 3×3 directe, ReLU, max-pool, dense, sur
  `Float32Array` préalloués. Ni wasm ni COOP/COEP ; même code sous V8 et JSC.
- *Phase 3 (plateau)* : même runtime, appelé sur les seules bandes discriminantes. Un GEMM Wasm SIMD
  128 bits à la main (WAT ou `clang -msimd128`, < 50 Ko, `im2col` par blocs, détecté par
  `WebAssembly.validate`, repli JS) ne s'ajoute que si le banc 1 dépasse ~50 ms par décision. Gain
  attendu 1,5–3× et seulement par blocs : Wasm SIMD naïf 0,90× du JS, par blocs 1,64× [36][37].
- *Seuil* (estimation de l'auteur, non publiée) : « sans dépendance » tant que le modèle tient sous
  ~1–2 M paramètres et 50–100 M MAC **par inférence**, avec conv, pool, dense, BN fusionnée et
  activations simples. Au-delà, ou pour un détecteur à ancres et NMS (le cube, réponse 09 ?) : ORT
  Web par `ort.wasm.min.js` (jamais `ort.min.js`), version épinglée, mémoire mesurée sous WebKitGTK [1].
- *Entraînement* : PyTorch → BN fusionnée → tenseurs float16 dans un fichier « à la safetensors »
  (en-tête JSON préfixé par sa longueur + données little-endian), ~50 lignes de JS. Parité **PyTorch
  contre JS** à 1e-4. La « parité au bit près avec le moteur Go » de la réponse est sans objet (pas
  de moteur de vision côté Go) et inatteignable (le JS calcule en float64 sans `Math.fround`).

**Non retenu.**
- *ONNX Runtime Web* : wasm minimal 13,58 Mo (JSEP 27 Mo, Asyncify 25,54 Mo) [2], plus de build sans
  threads depuis 1.19 [23] ; depuis 1.21 `ort.min.js` charge JSEP, 5–7 Go sous WebKit en 1.30.0 [1].
- *TensorFlow.js* : wasm légers (222/320/327 Ko en 3.9.0 [26]) mais dernière stable 4.22.0 du
  21 oct. 2024 [3][25] — projet en sommeil.
- *WebNN* : sous flag, origin trial close le 27 mars 2026 [4][27]. *WebGPU* : « UNSTABLE » en
  WebKitGTK 2.52 [5], macOS Tahoe 26 exigé [6], WebView2 non trouvé. Prématurés, inutiles à ces tailles.
- *Go/TinyGo en wasm* : ≥ ~2 Mo (~500 Ko gzip), SIMD expérimental (Go 1.27, `GOEXPERIMENT=simd`, bug
  #82310 ouvert) [32][33], interop `syscall/js` limitée aux `Uint8Array` [40]–[42].
- *Inférence côté Go, moteur réutilisé* (vérifié) : `gammonnet` ne fait que du MLP (`denseFunc`,
  dense sur 8 positions par voie ; ni conv ni `im2col`) et ses invariants (port fidèle, identité bit à
  bit, ADR-0024/0106) interdisent d'y loger la vision. Pont Wails : 230–400 ms par liaison de
  plusieurs Mo, pauses de 250–760 ms [7][8], promesses perdues sous WebView2 [48], POST multipart →
  SIGSEGV sous WebKitGTK [51] ; et des pixels vers Go contredisent le cadre.

**Risque découvert.** Sur CPU sans AVX, WebKitGTK 2.52 plante dès que du Wasm tourne [12][13]
(GeoLibre : `JSC_useWasmOSR=false`) ; `internal/gui/run.go` ne pose aucune variable `JSC_*`.

**Expériences imposées.**
1. Banc Worker : CNN de dés et bande à 300 k paramètres, 1 000 inférences, p50/p95, JS seul puis JS +
   GEMM Wasm SIMD ; WebKitGTK 2.50/2.52, WebView2, WKWebView (macOS 14 et 26) — remplace les estimations.
2. Worker lancé depuis `wails://` sous WebKitGTK (non résolu, Meld #625 [56]) ; repli URL Blob. **Bloquant.**
3. Parité PyTorch/JS sur ~200 imagettes golden (1e-4, `vitest`) ; machine sans AVX, avec et sans
   `JSC_useWasmOSR=false`.

**Conséquence pour l'ADR de la phase 2.** Runtime JS maison + format de poids et son export Python ;
Wasm SIMD puis ORT Web sont deux crans d'escalade déclenchés par une mesure, jamais par principe.

## 06 — Amorçage des données et apprentissage faiblement supervisé

**Confirme le cadre.** La base suffit à amorcer lecteur de dés et discriminateur de coups si l'on
traite `roll_tick_ms` / `tick_ms` comme des **fenêtres**, jamais comme l'image étiquetée : c'est la
*timestamp supervision* (Moltisanti et al., CVPR 2019, distributions recentrées par le classifieur,
jusqu'à +5,4 pts top-1 [S26] ; Li, Abu Farha, Gall, CVPR 2021, confiance décroissante autour de
l'horodatage [S1] ; les étiquettes dures « s'effondrent » au moindre écart, ECCV 2022). Décision de
coup par discrimination, sans équité : prédire le plateau après, le comparer au résultat de chaque
coup légal, argmax. Seuls des **poids** sont distribués.

**Chaîne d'amorçage recommandée.** (1) *Audit* déterministe : coup légal pour les dés, après = avant
+ coup, après(n) = avant(n+1) ; échecs exclus ou en revue. (2) *Homographie* une fois par vidéo
(4 clics, réponse 04), zones fixes (24 points, barre, sorties, dés, videau). (3) *Extraction dans la
webview* sur `roll_tick_ms − 2 s` … `tick_ms + 2 s` à 5–10 img/s (passe dense de 01) : plus long
plateau de faible différence inter-images, puis top-K = 5 par variance du laplacien ; recadrage
immédiat, jamais de plein cadre. (4) *Modèle v0 chez le développeur* (Python permis) : CNN 6 classes
+ « illisible », classifieur de case, réel + synthétique procédural, augmentations (perspective,
couleur, flou, ré-encodage bas débit, overlays, main). (5) *EM* 2–3 tours (re-choisir l'image la
plus confiante pour l'étiquette connue), puis Confident Learning [S2] pour lister les suspects.
(6) ONNX → onnxruntime-web WASM ; WebGPU en bonus (absent des notes WebKitGTK 2.48–2.54 [S20]).

**Change le cadre.**
- *07 prime sur l'éthique de la réponse.* Elle raisonne sur des « streams publics » de tiers, invoque
  la fouille L122-5-3 III (copies détruites à l'issue [S22][S41]) et propose d'entraîner sur les
  vidéos dont la chaîne YouTube a activé « third-party training » [S24]. **Rejeté** : une imagette
  tirée de YouTube reste la « derived data » de III.E.4.h quel que soit ce réglage, et la fouille
  suppose un accès licite. Corpus = fichiers locaux licites seulement. L'application ne distingue
  qu'URL / chemin local : la licéité reste une déclaration (la réponse l'admet [S25][S42]).
- *Le bruit est temporel avant d'être sémantique* : l'oracle du jeu filtre mieux qu'une méthode
  générique ; Co-teaching [S29] est superflu à quelques pourcents de bruit.
- *Pas de ré-entraînement chez l'utilisateur.* Entraînement ONNX Runtime déprécié en v1.20.0
  (1er nov. 2024, `ort-training-wasm` retiré [S11][S12]) ; GoMLX pur Go « very portable but slow »,
  backend rapide sous CGO [S14]. **Amélioration continue la plus simple** : prototypes / k-NN sur
  embeddings d'un encodeur figé (SimpleShot [S33]), par classe et par caméra, combinés à la tête.
  Garder chaque correction, les confirmations en zone grise et un échantillon des sûres (TypiClust :
  à petit budget, typicité avant incertitude [S6]) ; jamais une auto-validation non confirmée.
- *Seuil d'autonomie par caméra* : courbe risque–couverture (Geifman, El-Yaniv, NeurIPS 2017, hors
  liste numérotée) sur streams tenus à l'écart, cible indicative ≤ 0,5 % d'erreur par coup ;
  **auto-validation coupée sur une caméra nouvelle** tant que la courbe locale n'y est pas. Les
  erreurs se composent : 0,23 % par case = 93,86 % de plateaux justes (chesscog [S36]).

**Tailles de corpus (estimations d'ingénieur, non sourcées).** Phase 0 : 1 500–3 000 lancers propres
sur ≥ 3 streams (≈ 15–30 k imagettes corrélées), quelques centaines de positions par stream ; bas de
fourchette avec encodeur pré-entraîné. Phase 1 : 20–50 exemples par caméra, chaque face 3–5 fois.
Phase 2 : ré-entraînement central dès 5–10 streams. Appuis : 99,76 % sur d6 recadrés (WKU 2021, non
relu, fuite probable [S4]) ; 0,959 → 0,776 sur dés inconnus après 4 498 scènes Blender (PR GitHub
[S5]) ; chesscog, 2 photos → 99,83 % par case sur jeu inconnu [S8]. **Non trouvé** : un chiffre
primaire pour > 99 % par dé sur stream ou > 95 % de discrimination de coup ; l'apport du synthétique
pour des pions.

**Non retenu.** Co-teaching, ONNX Runtime Web Training, WebGPU comme socle, plein cadre, réglage YouTube.

**Expériences imposées.**
1. Histogramme `t_lisible − roll_tick_ms`, 50 coups, 3 streams : il fixe la fenêtre (±2 s supposé).
2. Courbe d'apprentissage 250 / 500 / 1 000 / 2 000 lancers, leave-one-stream-out ; pré-entraîné
   contre de zéro, avec et sans synthétique.
3. k-NN (5, 20, 50 exemples) contre `fit()` TensorFlow.js de la dernière couche [S13], stream neuf.
4. `navigator.gpu` dans la webview Wails ; précision du seek à l'image (rejoint le banc 3 de 01).

**Conséquence pour les repères, l'export et ADR-0007.**
- *Le corpus est un manifeste, pas des images* : export CLI (parité GUI/serveur) des repères —
  `video_source`, `roll_tick_ms`, `tick_ms`, dés, coup, plateaux, verdict d'audit — limité aux sources
  **locales**, URL refusées. Les imagettes se régénèrent depuis le fichier et se purgent après
  entraînement, registre des sources tenu. L'export de bibliothèque réduit un `video_source` local
  à son nom de fichier (ADR-0082 règle 4, `issuance.CarriedVideoSource`) : une base reçue n'apporte
  que des Repères sans image, un corpus central ne peut **pas** naître de bases reçues ; il se
  construit chez qui détient les fichiers.
- *Stockage local pour l'apprentissage actif* : embeddings et étiquettes, jamais d'imagette. Ce n'est
  pas un enregistrement côté destinataire (rien sur qui ouvre ni quand), mais pas non plus un dérivé
  des lignes : hors `issuance.CarriedMetadataKeys`, jamais exporté, de préférence hors du fichier de
  base, par caméra. Un embedding reste une donnée dérivée (mains, visages) : partage en opt-in.

## 07 — YouTube : accès légitime aux images

**Confirme le cadre, et le durcit.** La réponse est un **non** sans réserve : aucune voie documentée
ne permet d'analyser les images d'une vidéo YouTube dans les conditions d'utilisation. Techniquement,
seul `getDisplayMedia` sur sa propre fenêtre donnerait des pixels (fiable sous WebView2 seulement ;
PipeWire + portail + `enable-media-stream` sous WebKitGTK ; API privée + CGO sous WKWebView), et ce
serait un usage non autorisé du service (restrictions n° 1 et n° 3 des CGU US du 15 déc. 2023) et une
création de « derived data » à partir des API Data (Developer Policies III.E.4.h, qui couvrent le
contenu audiovisuel depuis janvier 2017). Les storyboards (160×90, une vignette par ~10 s, API non
documentée) sont inutilisables et interdits (III.D.7). La seule clause ouverte est le « (c) as
permitted by applicable law » de la version EEE de 2022 — trop incertaine pour bâtir dessus.

**Change le cadre.**
- *Séparation stricte des deux modes, dans le code comme dans la doc* : le pipeline d'analyse ne
  s'active que pour un `File` local ; YouTube garde `postMessage` (temps, état, vitesse) et le clavier.
  Pas de `getDisplayMedia`, même sous WebView2.
- *Le corpus d'entraînement aussi.* Le README dit « la base est déjà un jeu d'entraînement » : cela ne
  vaut que pour les matchs dont `video_source` est un **fichier local licite**. Extraire des imagettes
  d'une vidéo YouTube (même pour entraîner, même sans les distribuer) est la « derived data » que
  III.E.4.h interdit. La chaîne d'amorçage (prompt 06) doit filtrer sur la nature de la source et
  s'appuyer sur des fichiers transmis par leurs auteurs ou organisateurs, ou sur ses propres
  enregistrements. Un enregistrement OBS d'une vidéo YouTube d'un tiers reste une reproduction
  interdite : ne pas le recommander.
- *Interdiction de nommer un outil de téléchargement* (III.E.1 « encourage, enable » ; OLG Hamburg
  5 U 54/23, 21 nov. 2024 : l'hébergeur du site de youtube-dl jugé complice). Le dépôt en contient
  deux mentions, antérieures à ce chantier : `doc/source/manuel_matchs.rst` l.906 (« C'est le cas de
  fichiers téléchargés par yt-dlp ») et `pkg/blunderdb/videomux/doc.go` l.9 (« Lavf as yt-dlp runs
  it »). Les deux décrivent un multiplexage, pas un conseil, mais la formulation « fichiers
  téléchargés par yt-dlp » dans une doc utilisateur est exactement ce que la réponse déconseille.
  **Décision à prendre par Kevin** : reformuler en « fichiers WebM dont les clusters ne commencent pas
  sur une image clé (certains outils de téléchargement les écrivent ainsi) » et retirer le nom.
- *Conformité III.A.1* : un client de l'IFrame Player API doit afficher un lien vers les CGU YouTube
  et une politique de confidentialité mentionnant l'usage des YouTube API Services. Aucun lien vers
  `youtube.com/t/terms` n'existe dans `doc/source` ni dans `frontend/src` : manquement déjà présent,
  indépendant de la vision. À corriger dans le volet vidéo (mode YouTube) et la doc.

**Texte retenu pour la documentation** (manuel du panneau de transcription, section « Mode assisté ») :

> L'analyse automatique des images (détection des lancers, lecture des dés, reconnaissance des
> coups et du videau) fonctionne uniquement sur des fichiers vidéo locaux que vous avez le droit
> d'utiliser : vos propres enregistrements, ou des fichiers fournis par leur auteur ou l'organisateur
> du tournoi. Pour les vidéos YouTube, blunderDB affiche le lecteur officiel de YouTube et ne lit que
> la position de lecture, l'état et la vitesse ; il n'accède pas aux images, ne télécharge rien, ne
> conserve aucune copie et ne recommande aucun outil de téléchargement. L'utilisation des vidéos
> YouTube est soumise aux Conditions d'utilisation de YouTube (https://www.youtube.com/t/terms).

**Message dans le volet** quand le mode assisté est demandé sur une vidéo YouTube : trois lignes
(pourquoi, ce qui marche quand même, comment obtenir un fichier) et deux actions, « Continuer avec
YouTube » / « Ouvrir un fichier local… ». Une page d'aide « Demander le fichier à l'organisateur »
avec un modèle de message est l'alternative honnête la plus utile.

**Expériences imposées.** Aucune technique : les expériences 3–5 de la réponse (capture d'écran par
webview, images noires sous DRM) sont caduques puisqu'on renonce à la capture. Restent des
vérifications documentaires : (1) version EEE des CGU en vigueur depuis une IP française (date,
texte des restrictions 1–3) ; (2) numérotation et date de révision des Developer Policies ;
(3) relecture de L.331-5, L.335-2-1, L.335-3-1/-2 et L.122-5-3 sur Légifrance avant publication de
la doc ; (4) recenser les vidéos de backgammon CC BY et écrire aux organisateurs (USBGF, WBF,
Backgammon Galaxy) pour des fichiers sources ou une licence — c'est aussi la source du corpus
d'entraînement.

## 08 — UX des outils d'annotation « humain dans la boucle »

[S#] = liste « Sources » commentée (1–49) ; les appels du corps suivent une autre liste, discordante.

**Confirme le cadre.** Les deux niveaux existent tels quels dans Label Studio : « Auto accept
annotation suggestions » ou approbation région par région [S28] ; CVAT nomme manuel / semi-auto /
auto [S26]. KLM : assisté M + K ≈ 1,6 s par tour contre ≈ 4,4 s en manuel (K = 0,28 s, M = 1,35 s,
erreur du modèle ~21 %) [S43, S44]. Corriger une pré-annotation est ~1,7× plus rapide que saisir
[S24]. Refus et correction bon marché (Amershi G8/G9) [S7], undo plutôt que confirmation [S36].

**Change le cadre.**
- *Validation par le lancer suivant, pas par délai.* Le délai visible à chaque tour est une tâche
  de vigilance [S40] qui s'habitue : 50/106 participants lisent l'avertissement modifié à la 1re
  exposition, 24/114 à la 20e (chances divisées par plus de 3), médiane 10 s → 1,2 s [S37] ;
  chute de l'attention dès la 2e exposition, « obligatory neurobiological response » [S39]. Il
  dégénère en validation sans regard. Le jet suivant, lui, est gratuit : il doit être saisi de
  toute façon — c'est déjà ADR-0092 décision 2 (le chiffre valide la ligne présélectionnée et
  ouvre le jet suivant). **Le lancer suivant l'emporte.** Mais aucune étude ne compare les deux
  (non trouvé) ; un délai fixe court (≈ 2–3 s) ne survit que comme filet en lecture continue,
  jamais proportionnel à la confiance (aucun appui trouvé). Précédent : Gmail 5 s, 30 s max [S41].
- *Pas de seuil transférable ; l'utilisateur ne voit pas la dégradation.* Croisement à 0,70 de
  fiabilité pour une aide diagnostique (méta-régression, 20 études) [S1], contredit par 0,6 encore
  utile [S3]. En ASR : au-delà de 30 % WER la saisie redevient plus rapide, mais les transcripteurs
  ne retapent qu'à ~45 % [S16, S17] ; 25 % ailleurs [S20]. L'abandon de l'auto n'est donc pas un
  signal de rentabilité, seul le chronomètre l'est ; la complaisance résiste à la pratique [S4].
- *Précision avant rappel.* Les fausses alarmes nuisent plus que les omissions (compliance et
  reliance) [S2] ; Kocielnik trouve l'inverse pour un assistant e-mail [S8]. **Dixon l'emporte**
  ici par asymétrie de coût : une fausse validation corrompt la base en silence, une pause coûte
  une touche. Horvitz : agir / dialoguer / s'abstenir selon l'utilité espérée [S12].
- *L'auto se mérite par vidéo* : assisté sur 10 tours, puis auto si précision ≥ 95 % (sans source).
- *Trois états, pas de pourcentage* : sûr / à vérifier / incertain, couleur + forme + texte [S11] ;
  « à vérifier » jamais auto-validé. Corriger doit coûter autant qu'accepter, sinon on corrige moins
  [S25]. Pause : image gelée, zoom sur les dés, indice flagrant [S15] (reprise 3,8 s [S14]).

**Non retenu.**
- *Filtre de légalité à la DGT* (« illegal moves will not be accepted ») [S33] : contraire à
  ADR-0044 règle 2. Reconnaître parmi les candidats légaux, oui ; si aucun ne correspond, état
  incertain et pause — jamais le légal le plus proche à la place d'un coup illégal joué.
- *N-best sur 1/2/3* [S11] : les chiffres tapent les dés (ADR-0092 décision 1). En pause, taper
  les dés vus *est* départager ; pour le coup, `j`/`k` + Entrée.
- *Retour arrière comme undo* : il efface le jet en cours (ADR-0048 déc. 1) ; undo = `Ctrl+Z`.
  *Tab/Maj+Tab* (Sportscode [S30], CVAT R/E [S26]) : à confronter aux raccourcis existants.
- *Explication « j'ai lu 6-3 car… »* : sans effet [S9, S10] ; la surimpression du cadre lu sur
  l'image, preuve vérifiable, est préférée (inférence, non mesurée).

**Expériences imposées.** Journal local horodaté hors base, jamais exporté : temps et gestes par
tour, corrections, undo, pauses par match, abandons de l'auto, erreurs résiduelles (référence).
1. Intra-sujet alterné manuel/assisté/auto (ABCACB…), segments appariés de 10 tours, soi puis 3–5
   utilisateurs ; NASA-TLX brut et SUS par bloc.
2. Temps par tour × fiabilité injectée (100, 95, 85, 70 %) : le croisement est *notre* seuil.
3. Précision contre rappel ; 4. lancer suivant contre délai (3 sessions, 5 % d'erreurs injectées) ;
   5. surimpression contre sans. Erreurs injectées sur segments de référence, jamais en brouillon.
Critère d'avance (proposé) : assisté si temps médian par tour −30 % sans plus d'erreurs résiduelles
que le manuel ; auto si, en plus, ≥ 80 % des erreurs injectées rattrapées. Le KLM ignore la
recherche dans la vidéo, coût sans doute dominant : mesurer à part le gain de la phase 1 seule.

**Cinq mécanismes.** (1) Pré-remplissage + validation par le jet suivant + `Ctrl+Z` [S36, S37, S7] ;
(2) abstention réglée sur la précision [S2, S1, S12] ; (3) trois états, sans pourcentage [S11, S9] ;
(4) surimpression, image gelée, zoom [S14, S15] ; (5) marqueurs, saut au clavier, incertains de
forme distincte, revue des seuls « à vérifier » en fin de match [S30, S26].

**Conséquence pour les ADR.** L'ADR de la phase 3 amende ADR-0044 règle 1 ainsi : l'utilisateur
reste l'auteur ; l'automatique n'est qu'une acceptation déléguée, déclenchée par un fait observé
(le lancer suivant), jamais par l'équité ni par l'écoulement du temps seul, toujours tracée et
réversible. Il doit dire si cette trace (« lu, non confirmé » / « confirmé ») est stockée par
Action — ce serait un changement de schéma, hors du hash Zobrist. Pour ne pas casser ADR-0092 :
aucune frappe ne change de sens sans que l'indicateur permanent le dise (« Auto : 63 24/18 13/10,
validé au prochain jet », « Pause : dés illisibles — tapez les dés ») ; une frappe de
l'utilisateur prime toujours sur une validation de l'assistant arrivée au même instant ; une
validation automatique ne déplace jamais le Cursor qu'il a posé (cf. ADR-0092 décision 7).

## 09 — Petits objets : videau et score

**Confirme le cadre.** La vision n'a qu'à détecter un *changement* : la machine connaît la valeur du
cube, son propriétaire et le score. Une détection rare et fautive coûte plus qu'elle ne rapporte :
les fausses alertes dégradent plus que les oublis (Dixon, Wickens & McCarley 2007 [S1]), point de
bascule vers une fiabilité de 0,70 (Wickens & Dixon 2007, 20 études [S44]), 0,6 selon la charge
[S45]. Tout ce qui est retenu tourne en JS pur (canvas 2D, typed arrays) : ni opencv.js (~8–12 Mo,
blog [S16]), ni modèle, ni WebGPU (non trouvé actif par défaut en WebKitGTK 2.48–2.52).

**Change le cadre.**
- *Le videau sort de la phase 3.* Détection du double/prise **différée**, lecture de la face
  **abandonnée** : valeur = 2^doubles (× beavers si `has_beaver`, que la réponse omet).
- *L'overlay du stream est la seule sous-détection à faire.* Zone du score tracée une fois ; toutes
  les 0,5–1 s, imagette 32 × 16 gris, dHash 64 bits (plus tolérant aux fondus qu'une différence
  absolue), seuil de Hamming, hystérésis > 1 s — gratuit dans la passe grossière à 1–2 img/s (01).
  Confronté à la machine : attendu (passe, gammon) → confirme ; inattendu → alerte douce ; **jamais**
  d'écriture du score. C'est un contrôle de dérive de la machine. Précédent : brevets *score bug*
  (fixe, statique, présent pendant le jeu — US 7 653 131, US 12 217 506 [S26–S28]). L'overlay peut
  suivre l'action de **plusieurs secondes** (ESPN, courtvision [S30]) : tolérance de décalage. Les
  streams de club en sont souvent dépourvus : le mode doit s'en passer.
- *Pas de tracker.* Caméra fixe, trois emplacements (centre, côté A, côté B) : NCC par zone aux
  seules transitions de la machine (coup validé → dés saisis). MOSSE/KCF/CSRT jamais mesurés sur
  10–40 px pris en main et reposé (**non trouvé**) ; en HoG à 4 px un cube de 10 px fait 2–3 cellules
  et le modèle apprend la main pendant l'occlusion (inférence) ; CSRT absent d'opencv.js même avec
  `opencv_contrib` [S13] et en OpenCV 5.0.0 [S15].

**Non retenu.**
- Le doute de la réponse sur rVFC dans WebKitGTK : la réponse 01 l'établit dès 2.36.
- Lecture de la face : SVHN 97,84 %/chiffre sur 54 × 54 px [S17] ; dés 95,10–99,76 % (rapport WKU
  non daté [S19]) ou 86,46 % sur 96 images [S20]. Rien à 10–40 px, traits de 1–3 px pour 16/32/64 ;
  **non trouvé : aucun jeu de données ni article sur le videau**. Utile seulement si la machine
  perd l'état (transcription prise en cours de match).
- MediaPipe Hand Landmarker : 17,12 ms CPU sur Pixel 6 [S25], mais 14,45 Mo (6,75 + 7,70 Mo, portage
  Qualcomm, secondaire) pour quelques événements par partie.
- OCR généraliste : Tesseract.js 15,34 Mo [S31] (tesseract-wasm ~2,1 Mo Brotli [S33]), PaddleOCR
  mobile 7,5 + 4,7 Mo [S35] — démesuré pour 2–6 chiffres de police fixe.

**Non trouvé, et décisif.** Aucune étude du gain d'automatiser un événement rare, ni de « confirmer »
contre « taper une touche ». Les seuils (> 1 s, 32 × 16, < 1 alerte/h, rappel > 90 %) sont des choix
d'ingénierie. [Sn] = liste « Sources » finale de la réponse ; celle qui la précède est numérotée autrement.

**Expériences imposées.**
1. Overlay : 10 streams, compter les changements détectés hors score (animations, pubs, fondus) ;
   régler zone, seuil de Hamming, hystérésis et décalage.
2. Avant toute reprise du videau : 15–20 vidéos variées, actions de videau et cubes bousculés
   annotés ; NCC par zone ; « faire » seulement si rappel > 90 % **et** < 1 fausse alerte/h.
3. Si (2) passe : intra-sujet sur 5 matchs, touche contre suggestion « double détecté ? [d] » ;
   temps, erreurs résiduelles, NASA-TLX court ; écart dans le bruit → abandon définitif.

| Sous-détection | Verdict (faire / différer / renoncer) | Technique la plus simple | Phase |
|---|---|---|---|
| Overlay : changement | Faire | Zone tracée, dHash 64 bits sur 32 × 16 gris, 0,5–1 s, hystérésis > 1 s, confronté à la machine | 1 |
| Overlay : lecture du score | Différer (peu coûteux) | Glyphes auto-étiquetés par le score connu, NCC avec rejet ; vérification seulement (non publié) | 2 |
| Overlay : valeur du cube | Renoncer | — (redondant) | — |
| Présence du cube par zone | Différer | NCC sur 3 zones fixes aux transitions de la machine | 3 si reprise |
| Geste de double / prise | Différer | Zone A → zone B entre coup et lancer, persistant N images ; suggestion non bloquante | 3 si exp. 2 passe |
| Passe | Renoncer (en propre) | Remise à zéro des pions + changement d'overlay | 3 |
| Lecture de la face | Renoncer | Valeur = 2^doubles (× beavers) | — |
| Suivi continu (MOSSE/KCF/CSRT) | Renoncer | — | — |
| Détection de mains (MediaPipe) | Renoncer | — | — |

## Conclusions transversales

Neuf réponses lues, sept d'entre elles synthétisées par un relecteur distinct puis confrontées entre
elles. Ce qui suit est le cadre tel qu'il sort de la bibliographie ; il remplace celui du `README.md`
là où ils divergent.

### Ce que la bibliographie confirme

- **Faisable dans la contrainte** : binaire unique, tout le pixel dans la webview, Go inchangé dans
  son rôle (règles, candidats, machine). Lecture des pixels sûre sur les trois moteurs (01) ;
  géométrie résolue en quelques centaines de lignes de JS (04) ; inférence en JS écrit à la main
  dans un Worker, sans dépendance (05) ; descripteurs simples et automates à hystérésis pour les
  événements (03).
- **Discriminer entre candidats, jamais reconnaître une position** : consensus des systèmes vidéo
  d'échecs qui marchent ; la reconnaissance absolue s'effondre hors domaine (93,9 % → 2,3 %) et
  0,99³⁶¹ = 2,7 % de plateaux justes à 99 % par case (02, 04).
- **L'équité n'est pas un a priori.** Trois réponses (02, 04, 08) ont proposé de classer ou de
  filtrer par le réseau ou par la légalité « à la DGT » ; les trois sont écartées. Le top-K est
  classé au seul score visuel ; si aucun candidat ne correspond, c'est un état *incertain*, jamais
  « le légal le plus proche ». Une expérience le contrôle (02, exp. 2 : le top-1 visuel ne doit pas
  s'effondrer sur les coups qui ne sont pas le premier du réseau).
- **Fichier local seulement, et strictement** : aucun pixel YouTube, pas de capture d'écran, pas de
  nom d'outil de téléchargement, corpus d'entraînement tiré des seules sources locales licites (07,
  06).

### Ce que la bibliographie change

1. **Il n'y a pas de « pré-passe rapide ».** Aucun moteur ne tient 10 img/s sur une heure en moins
   de 5 min (01). À 1–2 img/s on ne détecte pas des événements, on ouvre des *fenêtres* en
   comparant des **états** (dés déplacés, plateau changé) ; les automates tournent à 10 img/s dans
   ces fenêtres, réduites à [−1 ; +2] s (03). Si la couverture reste élevée — au backgammon les
   événements se suivent à quelques secondes — l'analyse **opportuniste pendant la lecture à 1×**
   devient le chemin principal. Une requête SQL sur `roll_tick_ms`/`tick_ms` le décide avant toute
   ligne de code (03, exp. 1).
2. **Deux résolutions.** 320×180 suffit aux événements ; lire le coup exige ≥ 16 px par pion, donc
   une rectification de la zone du plateau en pleine résolution, sur les seules images calmes (04).
3. **Validation par le lancer suivant, pas par délai.** Un compte à rebours à chaque tour devient un
   réflexe en moins de vingt expositions (08) ; le jet suivant est gratuit et c'est déjà ADR-0092
   décision 2. Le mode « automatique » devient une *acceptation déléguée* déclenchée par un fait
   observé, tracée, réversible par `Ctrl+Z` ; le délai fixe court n'est qu'un filet en lecture
   continue.
4. **Trois états, pas de pourcentage** (sûr / à vérifier / incertain), **précision avant rappel**
   (une fausse validation corrompt la base en silence, une pause coûte une touche), et
   **l'automatique se mérite par caméra** : coupé sur une caméra nouvelle tant que sa courbe
   risque–couverture locale n'atteint pas la cible (08, 06).
5. **Un Repère tapé n'est pas un événement** : le jet est tapé après le repos + lecture + frappe,
   l'action est validée après la sortie des mains. Les horodatages de la base sont des fenêtres,
   jamais l'image étiquetée (*timestamp supervision*, 06) ; un Repère posé par la détection change
   la sémantique de la colonne et l'ADR doit le trancher (03). L'instant à écrire est
   `metadata.mediaTime` de rVFC (01).
6. **Le videau sort des phases 1–3.** Lecture de la face abandonnée (valeur = 2^doubles × beavers),
   geste de double différé derrière un banc (rappel > 90 % **et** < 1 fausse alerte/h), aucun
   suivi continu. Seule la **détection de changement de l'overlay** du stream se fait, dès la phase
   1, comme contrôle de dérive de la machine — jamais d'écriture du score (09).
7. **Rien d'appris chez l'utilisateur.** Pas de ré-entraînement local (ORT training déprécié,
   WebGPU absent de WebKitGTK) ; l'amélioration continue est un k-NN de prototypes par caméra,
   vecteurs stockés localement sans image, hors `CarriedMetadataKeys`, jamais exportés (06).
8. **Le corpus est un manifeste, construit chez qui détient les fichiers.** L'export de
   bibliothèque réduit un `video_source` local à son nom de fichier (ADR-0082 règle 4) : une base
   reçue n'apporte que des Repères sans image, un corpus central ne peut pas naître de bases reçues. Un export CLI des repères, limité aux sources locales, les imagettes régénérées
   puis purgées (06).
9. **Plancher réel WebKitGTK : 2.50** (même Ubuntu 22.04 l'a reçu par les mises à jour de
   sécurité) : le chemin confortable de 01 est le cas normal (05). Toujours par détection de
   fonctionnalité. Pas de mémoire partagée ; Wasm SIMD seulement si le CPU a l'AVX, repli JS
   obligatoire (05).
10. **Aucune mesure publiée sur du backgammon filmé** et aucun jeu public de dés sur plateau (02).
    Toutes les précisions avancées par les réponses sont des estimations ; nos cas durs (piles > 5,
    pions sortis, dé cocké, reflets) sont « non trouvé ». Le banc maison précède toute promesse.

### Risques découverts, indépendants de la vision

- **Local Network Access de Chromium ≥ 142** : la lecture vidéo actuelle par loopback sous WebView2
  (`http://wails.localhost` → `127.0.0.1`) peut déjà déclencher une permission ou un blocage (01).
- **JavaScriptCore sans AVX** : WebKitGTK 2.52 plante dès que du Wasm tourne ; `internal/gui/run.go`
  ne pose aucune variable `JSC_*` (05).
- **Conformité YouTube** : `yt-dlp` nommé dans `doc/source/manuel_matchs.rst` et
  `pkg/blunderdb/videomux/doc.go` ; aucun lien vers les CGU YouTube alors que III.A.1 l'exige d'un
  client de l'IFrame Player API (07).
- **Worker depuis `wails://` sous WebKitGTK** : non résolu ailleurs (Meld #625) ; bloquant pour tout
  le chantier, à tester en premier (05).

### Phases révisées

**Phase 0 — bancs, avant toute ligne de produit.** Dans l'ordre : Worker depuis `wails://` (05.2) ;
couverture des fenêtres en SQL (03.1) ; CORS sur 206 et redirection (01.1) ; LNA WebView2 (01.2) ;
coût de `drawImage` 1080p → 320×180 (01.6) ; banc de seeks et `playbackRate` (01.3, 01.4) ;
WebCodecs par distribution (01.5) ; histogramme `t_lisible − roll_tick_ms` sur 50 coups (06.1). Un
seul banc rouge parmi les deux premiers arrête le chantier.

**Phase 1 — événements et ligne de temps.** Calibration 4 clics (loupe, pions fantômes, calibrations
nommées, 1er clic = 1-point du joueur du bas), stockée en localStorage par `draftId` ; score de
présence du plateau (« vue perdue ») ; descripteurs census + ROI témoin et trois automates
(mains / dés / partie) à 10 img/s dans des fenêtres ouvertes par comparaison d'états ; ligne de
temps à marqueurs, saut au clavier, incertains de forme distincte ; dHash de l'overlay ; journal
local de descripteurs (hors base, jamais exporté) qui permet de rejouer les automates en `vitest`.
Mesure : F1 à ±0,5 s, et le gain de temps de la seule navigation. **ADR-1** : amende ADR-0082
règle 6 (« la webview décode, l'application lit des images ») ; sémantique d'un Repère posé par
détection ; stockage de la calibration.

**Phase 2 — dés et mode assisté.** Export CLI des repères (sources locales) ; chaîne d'amorçage
(audit déterministe → fenêtres → images calmes → top-K par netteté → modèle v0 en Python → EM) ;
CNN < 100 k paramètres, 6 classes + « illisible », vote sur 5–10 images, poids float16 dans
`frontend/dist`, parité PyTorch/JS à 1e-4 ; pré-remplissage des dés, validation par le jet suivant,
trois états, pause sur incertain avec image gelée et zoom. Mesure : précision par dé en
leave-one-stream-out ; intra-sujet manuel/assisté, −30 % de temps médian par tour sans plus
d'erreurs. **ADR-2** : runtime JS maison et format de poids ; corpus-manifeste ; embeddings locaux.

**Phase 3 — coup et acceptation déléguée.** Rectification pleine résolution des bandes où les
`board_after` des candidats diffèrent (profil 1D, L(n) mesurée pour n > 5) ; top-K au score visuel,
abstention réglée sur la précision ; automatique par caméra après courbe risque–couverture ;
lecture du score de l'overlay en vérification ; videau seulement si le banc 09.2 passe. **ADR-3** :
amende ADR-0044 règle 1 (acceptation déléguée, déclenchée par le lancer suivant, tracée,
réversible ; trace par Action = changement de schéma hors hash) sans toucher ADR-0092 (l'indicateur
dit l'effet ; une frappe de l'utilisateur prime ; une validation automatique ne déplace jamais le
curseur).

### Fiabilité des réponses

Cinq réponses sur neuf ont deux listes de sources numérotées différemment ; les sections citent la
liste finale, celle des appels du texte. La plupart des seuils, durées et tailles de corpus sont des
choix d'ingénierie non sourcés, signalés comme tels. Deux contradictions entre réponses ont été
tranchées en faveur de 01 (WebCodecs et rVFC sous WebKitGTK : détection de fonctionnalité, jamais
une version). Les chiffres de précision sur les dés viennent de sources faibles (rapport non relu,
test sur 96 images) ; ceux de Chesscog sont repris de seconde main. Les préprints arXiv de 2026 ne
sont pas relus par des pairs.
