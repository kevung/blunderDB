Recherche du 2026-10-30 — claude.ai, recherche approfondie.
Prompt : ../prompts/01-webviews-acces-images.md

---

# Accès aux images d'un `<video>` dans WebKitGTK, WebView2 et WKWebView : revue technique pour blunderDB (état au 30 octobre 2026)

On peut lire les pixels d'un `<video>` sur les trois moteurs avec `drawImage` + `getImageData`, à condition d'utiliser `crossorigin="anonymous"` et de renvoyer `Access-Control-Allow-Origin` sur **toutes** les réponses, 206 comprises. En revanche, l'objectif « 10 img/s sur un match de 60 min analysé en moins de 5 min » ne tient pas avec `<video>` seul. Il impose 36 000 images en 300 s, soit environ 120 images traitées par seconde de temps réel. Aucune source ne montre qu'un moteur tienne ce rythme par `seek` ou par `playbackRate`. La seule voie plausible est une analyse hiérarchique : pré-passe grossière, puis passe dense autour des événements, avec WebCodecs là où il existe.

## TL;DR

- **Lecture des pixels : faisable partout.** Le canvas reste *origin-clean* si chaque réponse (200 et 206) est CORS-same-origin. Les pièges sont les redirections et les réponses partielles d'origines différentes. L'alternative la plus sûre est de servir la vidéo par l'AssetServer de Wails (même origine). Elle supprime aussi le risque Local Network Access de Chromium (Chrome 142+, octobre 2025).
- **Le chemin portable est `requestVideoFrameCallback` + canvas réduit à 320×180 + `getImageData`, avec un worker pour l'analyse.** rVFC existe dans WebKitGTK depuis 2.36 (mars 2022), dans Chromium et dans WebKit/Safari. `OffscreenCanvas` n'est activé par défaut dans WebKitGTK qu'à partir de 2.46 (septembre 2024). SharedArrayBuffer est absent sous WebKitGTK 2.42–2.48 : il n'a été activé dans les ports GLib qu'en septembre 2026, et seulement en contexte isolé.
- **Le budget de 5 min impose une stratégie en deux temps.** D'abord une pré-passe peu dense : images-clés via WebCodecs `VideoDecoder` (Chromium, Safari 16.4+, WebKitGTK 2.44/2.46+), ou `seek` successifs. Ensuite une passe dense locale. La lecture accélérée de WebKitGTK/GStreamer est documentée comme instable : désynchronisation et gels sur les bugs 245850/245852, encore observés en 2.42.4. Elle ne doit pas être le chemin principal sous Linux.

## Key Findings

### Matrice de compatibilité (moteur × fonction × version × source)

| Fonction | WebKitGTK 2.42–2.48 (webkit2gtk-4.0/4.1) | WebView2 Evergreen (Chromium récent) | WKWebView macOS 13–15 |
|---|---|---|---|
| `drawImage(video)` + `getImageData` sans tainting (CORS) | Oui, règle HTML commune ; pièges 206 et redirections non documentés spécifiquement (non trouvé) | Oui | Oui |
| `requestVideoFrameCallback` | Oui depuis 2.36.0 (21 mars 2022) [S3]\[1\] ; implémentation WebKit d'octobre 2021, d'abord derrière une préférence expérimentale désactivée [S2]\[2\] | Oui : Chrome 83 (19/05/2020) et Edge 83 (21/05/2020), selon le Web Platform Features Explorer (web-platform-dx, données BCD) | Oui : Safari 15.4, confirmé par caniuse (« 15.4 - 26.3: Supported ») et par l'Intent to ship de Mozilla (« It has already shipped in Safari 15.4 (2022) and Chrome 83 (2020) ») |
| `OffscreenCanvas` 2D en worker | **Non par défaut en 2.42/2.44** ; activé par défaut en 2.46.0 (17 sept. 2024) [S5][S6]\[3\] | Oui | Safari 16.4 (2D, 28 mars 2023) ; WebGL en Safari 17.0 [S9][S10]\[4\]\[5\] |
| `createImageBitmap(video)` | non trouvé (non vérifié) | Oui (non revérifié ici) | non trouvé (non vérifié) |
| WebCodecs `VideoDecoder` / `VideoFrame` | 2.44 selon l'annonce (27 mars 2024) [S7], mais « activé par défaut en 2.46 » selon Igalia [S8] → **contradiction**\[6\]\[7\] | Oui (Chrome 94+, secondaire [S12])\[8\] | Safari 16.4 (vidéo seulement) [S9] ; H.264 L1T2 en 17.2 [S11] ; audio en Safari 26 (secondaire [S12])\[4\]\[8\]\[9\] |
| `MediaStreamTrackProcessor` | non trouvé | Version ancienne, non conforme à la spécification actuelle [S14]\[10\] | Safari 18.0 (16 sept. 2024), en *dedicated worker* [S13]\[11\] |
| `HTMLMediaElement.captureStream()` | Non : « we do not support captureStream for the moment » (canvas, 2017) [S15]\[12\] ; aucune trace pour `<video>` (non trouvé) | Oui depuis Chrome 62 (17/10/2017) et Edge 79, selon le Web Platform Features Explorer (capture-stream-audio-video) | Non : « Safari ❌ … Baseline availability blocked … by Safari » (Web Platform Features Explorer) ; caniuse indique Safari « 3.1 - 26.6: Not supported » |
| SharedArrayBuffer / threads Wasm | **Non** : activé dans les ports GLib par le commit 321889@main (bug 317847, ~25 sept. 2026), seulement si l'origine est isolée [S16]\[13\] ; mesure sur 2.50.4 : `crossOriginIsolated` = faux sous schéma personnalisé [S17]\[14\] | Possible si COOP/COEP servis sur `http://wails.localhost` (non vérifié pour WebView2) | Sur iPad (iOS), COOP/COEP via `WKURLSchemeHandler` donnent `crossOriginIsolated=false`, mais `true` depuis `http://127.0.0.1` [S18]\[15\] ; macOS non vérifié |
| Wasm SIMD | non trouvé (non vérifié ici) | Oui depuis Chrome/Edge 91 (mai 2021), selon caniuse.com/wasm-simd et la doc Emscripten (« Chrome ≥ 91 (May 2021) ») | Safari 16.4 (27/03/2023), selon caniuse.com/wasm-simd et la doc Emscripten (« Safari ≥ 16.4 (March 2023) ») : présent sous macOS 13 avec Safari ≥ 16.4 |
| `playbackRate` max | Pas de plafond documenté ; garde-fou contre les valeurs extrêmes depuis 2014 [S20]\[16\]\[17\] ; lecture ≠ 1× instable [S21][S22]\[18\]\[19\] | [0,0625 ; 16], sinon `NotSupportedError` (Chrome 63, 2017) [S23]\[20\]\[21\]\[22\]\[23\] | Plage 0,07–16 rapportée (secondaire, 21 oct. 2026) [S25]\[24\] ; non confirmé par Apple |
| `fastSeek()` | Présent dans WebKit ; « [GStreamer][MSE] FastSeek is broken » (2023) [S26]\[25\] ; comportement en lecture progressive non trouvé | **Non** (crbug 41276303) [S27] | Oui depuis Safari 8 [S27]\[26\]\[27\] |
| Latence d'un `seek` | non trouvé ; un cas aval WPE montre un passage à PAUSED de ~2 s [S28]\[28\] | 10,7–18,7 ms (720p H.264, macOS, Chrome 149, 2026) [S29]\[29\] | non trouvé |

## Details

### 1. Lecture des pixels : règles de *tainted canvas* et pièges CORS

**Règle normative (WHATWG HTML).** Un `HTMLVideoElement` n'est pas *origin-clean* si ses *media data* sont CORS-cross-origin [S30].\[30\] La ressource média porte une origine qui peut valoir « none », « multiple », « rewritten » ou une origine précise. À chaque nouvelle réponse, la spécification compare l'origine à la précédente. Si la réponse est CORS-cross-origin, le chargement échoue [S30].\[31\] En pratique, **chaque** réponse partielle (206) d'une suite de requêtes `Range` doit porter un `Access-Control-Allow-Origin` valide. Un seul 206 sans en-tête suffit à rendre la ressource cross-origin et donc à teinter le canvas. Une issue WHATWG ouverte (#6783) relève que « CORS-cross-origin » n'est défini formellement que pour une *response*, pas pour des *media data* [S31].\[30\] La spécification a donc une zone floue que chaque moteur comble à sa façon.

**Ce qui teinte, ce qui lève une erreur.** Sur un canvas teinté, `getImageData()`, `toBlob()`, `toDataURL()` et `captureStream()` lèvent `SecurityError` (MDN) [S32].\[32\] `drawImage` lui-même réussit. L'échec n'apparaît qu'à la lecture, ce qui peut masquer le problème jusqu'en production.

**Requêtes `Range` et pré-vol.** Selon MDN, `Range` ne compte comme en-tête « simple » qu'avec **une seule** plage, par exemple `bytes=256-` ou `bytes=127-255` [S33].\[33\] Les requêtes du lecteur ne déclenchent donc pas de pré-vol OPTIONS. Le serveur loopback doit quand même répondre aux OPTIONS par précaution. Côté serveur, il faut sur 200 **et** 206 :
- `Access-Control-Allow-Origin` : l'origine exacte de la page (`wails://wails` sous macOS/Linux, `http://wails.localhost` sous Windows ; voir plus bas) ou `*` ;
- `Accept-Ranges: bytes`, ainsi que des `Content-Range` et `Content-Length` cohérents ;
- `Vary: Origin` si l'origine est renvoyée en écho.

Avec `crossorigin="anonymous"`, aucun cookie n'est envoyé.\[34\] Le jeton doit donc rester dans l'URL, ce qui est déjà le cas.

**Redirections.** MDN note que « not all browsers currently support following redirects after a preflighted request » [S33].\[33\] Une redirection vers une autre origine en cours de lecture changerait l'origine de la ressource. Recommandation : **aucune redirection** sur l'URL tokenisée.

**Origine de la page Wails.** La documentation Wails indique que `wails dev` utilise le schéma `wails://` sous macOS et Linux depuis v2.5.1 (16 mai 2023) [S34].\[35\] Sous Windows, l'application est servie sur `http://wails.localhost` (issue Wails #4386, qui concerne v3 mais décrit le comportement Windows) [S35].\[36\] Le serveur loopback doit accepter les deux origines.

**Alternative même origine (fortement recommandée).** L'AssetServer de Wails v2 gère les requêtes `Range` si le `fs.FS` fournit un `io.ReadSeeker`, depuis v2.3.0 (29 déc. 2022). Cette version a aussi ajouté le support des méthodes et en-têtes HTTP pour WebKit2GTK 2.36+ et le streaming requête/réponse sous macOS [S36].\[37\] Servir la vidéo par un `AssetsHandler` ou un middleware sur le schéma de la page rend le canvas *origin-clean* par construction. Cela évite aussi le Local Network Access de Chromium. Ce choix reste à valider : la lecture média GStreamer via un schéma personnalisé (seek et `Range` côté WebKitGTK) n'est pas documentée (non trouvé).

**Local Network Access (Chromium).** Chrome 142 (28 oct. 2025) soumet à permission toute requête d'un site public vers une adresse locale ou loopback, et d'un site local vers loopback [S37][S38].\[38\]\[39\] Chrome 145/146 a séparé la permission en `local-network` et `loopback-network`, et Chrome 147 (avril 2026) l'a étendue aux WebSockets [S38][S39].\[39\]\[40\]\[41\] La permission est réservée aux contextes sécurisés [S40].\[38\] Le comportement exact de WebView2 n'est **pas trouvé** : classement de `http://wails.localhost` comme public, local ou loopback, présence d'une invite, et éventuel passage par l'événement `PermissionRequested`. C'est un risque réel pour la conception loopback sous Windows, et une raison de plus pour préférer la même origine.

**Côté WebKitGTK.** L'API `WebKitSecurityManager` permet d'enregistrer un schéma comme *CORS enabled*, *secure*, *local*, etc. [S41].\[42\] Les drapeaux qu'applique Wails v2 au schéma `wails` ne sont pas documentés (non trouvé). Ils déterminent si la page est un contexte sécurisé et comment elle traite les sous-ressources `http://127.0.0.1`.

### 2. Cadence : `requestVideoFrameCallback`

- **Support.** Le bug WebKit 211945 a été ouvert le 15 mai 2020 [S1].\[43\] Le changeset r284528 (octobre 2021) ajoute l'IDL et une préférence `RequestVideoFrameCallbackEnabled`, d'abord par défaut à `false` [S2].\[2\] WebKitGTK 2.36.0 (21 mars 2022) annonce « Add support for requestVideoFrameCallback » [S3].\[1\] Toute la plage 2.42–2.48 est donc couverte. Il reste à vérifier à l'exécution que la préférence est active dans la webview Wails (`'requestVideoFrameCallback' in HTMLVideoElement.prototype`).
- **Précision.** web.dev rappelle que rVFC s'exécute sur le thread principal alors que la composition vidéo se fait sur le thread du compositeur. Tout est donc « best effort », sans garantie stricte [S4].\[44\] MDN précise que la cadence est plafonnée par le rafraîchissement de l'écran : une vidéo 120 img/s sur un écran 60 Hz donne 60 rappels/s [S4b].\[45\] `metadata.mediaTime` est l'horodatage de présentation de l'image composée. C'est la valeur à stocker, et non `currentTime`, qui avance entre deux images. Aucune mesure publiée de l'écart `mediaTime`/PTS réel par moteur n'a été trouvée.
- **Pause et `seek`.** En pause, aucun rappel ne part tant qu'aucune nouvelle image n'est composée. Après un `seek`, un rappel part en général quand l'image cible est présentée, et c'est l'usage courant pour « attendre l'image après seek ». Aucune source primaire par moteur n'a été trouvée sur ce comportement exact : à tester.
- **Repli.** `requestAnimationFrame` + `currentTime`, en dédoublonnant par `getVideoPlaybackQuality().totalVideoFrames`, comme le fait le polyfill cité par web.dev [S4].\[44\] Côté GStreamer, `getVideoPlaybackQuality` a été corrigé récemment pour compter correctement les images rendues et perdues (PR WebKit #77183, date non trouvée) [S42].\[46\]

### 3. Hors du thread principal

- **`OffscreenCanvas`** : Safari 16.4 (2D) et Safari 17.0 (WebGL) [S9][S10].\[4\]\[5\] Côté WebKitGTK, activation par défaut en **2.46.0** seulement [S5][S6].\[3\]\[47\] En 2.42/2.44, il faut l'activer explicitement par un réglage, ce que Wails v2 ne fait pas à notre connaissance (non trouvé), ou prévoir un repli.
- **Où réduire l'image.** `drawImage(video)` doit se faire sur le thread principal : l'élément `<video>` n'est pas transférable. Le chemin portable est donc : rVFC → `drawImage(video, 0,0,320,180)` sur un petit canvas → `getImageData` (environ 230 Ko) → `postMessage` avec transfert de l'`ArrayBuffer` au worker. L'inférence lourde (dés, pions, videau) tourne dans le worker. Pour éviter même `getImageData` sur le thread principal, on peut utiliser `createImageBitmap(video, {resizeWidth:320, resizeHeight:180})` ou `new VideoFrame(video)` (WebCodecs), puis transférer au worker vers un `OffscreenCanvas`. Cette variante exige `OffscreenCanvas` (WebKitGTK ≥ 2.46) et, pour `VideoFrame`, WebCodecs.
- **Coût de la copie 1920×1080 → 320×180.** Aucune mesure publiée par moteur n'a été trouvée. Ordres de grandeur, par calcul : une image 1080p RGBA pèse 8,3 Mo et une image 320×180 en pèse 0,23 Mo. Si la réduction se fait sur le GPU, la relecture vers le CPU ne porte que sur 0,23 Mo, soit 2,3 Mo/s à 10 img/s, ce qui est négligeable. Point propre à WebKitGTK 2.46 : `willReadFrequently: true` force un canvas **non accéléré** [S6].\[47\] La réduction se fait alors probablement sur le CPU, après rapatriement de l'image complète (inférence, à mesurer). Les versions 2.46.2 et 2.46.4 corrigent la consommation mémoire de `putImageData` puis de `getImageData` [S43][S44].\[48\]\[49\] Il faut donc viser **≥ 2.46.4** sous Linux.

### 4. WebCodecs, `MediaStreamTrackProcessor`, `captureStream()`

- **`VideoDecoder`** est disponible dans Chromium, dans Safari 16.4+ (vidéo seulement) [S9][S11] et dans WebKitGTK. Pour ce dernier, les sources se contredisent : l'annonce officielle de 2.44 (27 mars 2024) dit « WebKitGTK now supports the WebCodecs API » [S7], tandis que le bilan Igalia 2024 dit « this is now enabled by default in 2.46 » [S8].\[6\]\[7\] Une lecture prudente : présent en 2.44, fiable et actif par défaut à partir de 2.46. À tester par détection de fonctionnalité.
- **Utilité ici : forte pour la pré-passe.** WebCodecs n'est pas du « décodage hors webview » : le décodeur appartient à la webview (GStreamer, VideoToolbox, Media Foundation). Il suppose un démultiplexage MP4 en JavaScript (bibliothèque JS pure, sans dépendance native) et un `fetch` par plages depuis un worker, soumis aux mêmes règles CORS. Son intérêt est de ne décoder que les **images-clés**, soit environ une toutes les 2 s sur un enregistrement de stream courant (hypothèse à vérifier sur vos fichiers). On obtient ainsi un échantillonnage grossier de l'heure entière, sans horloge de lecture ni compositeur. Les `VideoFrame` doivent être fermées (`close()`) immédiatement. Une proposition Interop 2026 signale des fuites mémoire de `VideoFrame` vers WebGL et `OffscreenCanvas` sur WebKit mobile [S45].\[50\]
- **`MediaStreamTrackProcessor`** n'est utile que pour un `MediaStreamTrack`, donc à partir de `captureStream()`. Safari 18.0 le fournit en *dedicated worker* [S13].\[11\] Chromium expose une ancienne version incompatible [S14].\[10\] Rien n'a été trouvé pour WebKitGTK. **Superflu ici.**
- **`captureStream()` sur `<video>`** : absent de Safari selon le Web Platform Features Explorer (« Safari ❌ ») et caniuse (Safari « 3.1 - 26.6: Not supported »), alors que Chromium l'a depuis Chrome 62 (17/10/2017) ; rien de trouvé côté WebKitGTK [S15]. Il serait de toute façon soumis au rythme de lecture réel. **Superflu ici**, et non portable.

### 5. WebAssembly : SIMD, threads, COOP/COEP sous schéma personnalisé

- **Threads (SharedArrayBuffer)** : à exclure du chemin portable. Sous WebKitGTK, le commit 321889@main (bug 317847, ~25 sept. 2026) active SAB dans les ports GLib seulement « when the origin is isolated ».\[13\] Avant ce commit, les TestExpectations indiquaient : « Cross-origin isolation is not fully implemented (see webkit.org/b/268068) » [S16].\[13\]\[51\] Les versions 2.42–2.48 n'ont donc pas de SAB. Une mesure sur WebKitGTK 2.50.4 (via Tauri/Wails) donne `crossOriginIsolated` faux sous schéma personnalisé [S17]\[14\] ; la source est une PR générée par un bot, donc secondaire. Sur WKWebView iOS, COOP/COEP via `WKURLSchemeHandler` ne suffisent pas, alors qu'une origine `http://127.0.0.1` avec les mêmes en-têtes fonctionne [S18]\[15\] ; macOS n'est pas vérifié. Pour WebView2, l'isolation sur `http://wails.localhost` n'est pas trouvée.
- **SIMD** : Chrome/Edge 91 (mai 2021) et Safari 16.4 (27/03/2023), selon caniuse.com/wasm-simd et la doc Emscripten (« Chrome ≥ 91 (May 2021) … Safari ≥ 16.4 (March 2023) ») ; WKWebView sous macOS 13 avec Safari ≥ 16.4 a donc le SIMD. Le support WebKitGTK n'a pas été trouvé. Il ne dépend pas de COOP/COEP.
- **Conséquence** : un modèle léger (CNN de quelques centaines de Ko sur 320×180 ou sur des zones recadrées) en Wasm monothread, ou WebGL/WebGPU, dans **un** worker suffit à 10 img/s. Le parallélisme passe par plusieurs workers et des `postMessage` avec transfert, pas par de la mémoire partagée.

### 6. Pré-passe accélérée : `playbackRate` contre `seek`

**Arithmétique de l'objectif.** 60 min à 10 img/s font 36 000 images. Les traiter en 300 s demande environ 120 images/s, et une vitesse de 12× si l'on passe par la lecture.

- **`playbackRate` dans Chromium/WebView2** : plage [0,0625 ; 16], `NotSupportedError` au-delà (Chrome 63, 2017) [S23].\[22\]\[23\] Depuis Chrome 64, l'audio n'est plus coupé aux vitesses extrêmes (WSOLA) [S23].\[20\] 12× est donc accepté. Mais rVFC reste plafonné par l'affichage (~60 rappels/s à 60 Hz [S4b]), soit au mieux ~5 images par seconde de vidéo à 12×, et non 10.\[45\] Le décodeur doit en outre traiter 360 img/s pour une source à 30 img/s. Aucune mesure publiée du taux d'images réellement rendues à 8–16× n'a été trouvée.
- **WKWebView** : une plage 0,07–16 est rapportée par un auteur d'extension, sur simulateur, sans généralisation (secondaire, 21 oct. 2026) [S25].\[24\] Sous AVPlayer natif, au-delà de 2×, seules les images-clés étaient affichées, avant un correctif signalé « fixed in macOS 13 » par un utilisateur (FB9343129, juillet 2021) [S24].\[52\] Rien de confirmé par Apple.
- **WebKitGTK/GStreamer** : un garde-fou contre les vitesses extrêmes existe depuis 2014 (r163871). Le relecteur suggérait de limiter à (−10, 10), mais la borne effectivement retenue n'est pas vérifiée [S20].\[16\]\[17\] Dans l'ancien code, l'audio est coupé si `!preservesPitch && (rate < 0.8 || rate > 2)` [S20b].\[53\] Des bugs ouverts décrivent qu'à une vitesse ≠ 1× la vidéo « play rapidly through the video stream, while the audio is not in sync », puis des gels après `seek`. Ces bugs sont encore observés avec WebKitGTK 2.42.4 et GStreamer 1.22.8 [S21][S22].\[18\]\[19\]
- **`seek` successifs** : `fastSeek()` existe dans Safari depuis la version 8 mais **pas dans Chromium** [S27].\[26\]\[27\] Il est cassé en MSE sous GStreamer (2023) [S26].\[25\] `currentTime =` fait un seek précis : vidage du décodeur, puis décodage depuis l'image-clé précédente jusqu'à la cible [S29]. Mesure la plus solide trouvée : Chrome 149 sur macOS, MP4 H.264 **720p**, 10,7–18,7 ms par seek [S29],\[29\] soit environ 50 à 90 seeks/s. Pour du 1080p60 H.264 en défilement dans Chrome, une source personnelle peu fiable donne 22–29 img/s avec un p95 de 67–275 ms quand on enchaîne sans attendre [S46].\[54\] **Non trouvé** pour Safari/WKWebView et WebKitGTK. Un cas aval WPE montre un passage à PAUSED d'environ 2 s après seek [S28].\[28\]
- **Conclusion chiffrée.** Même au meilleur cas Chromium, 36 000 seeks précis prennent environ 400 à 700 s, au-delà des 5 min. 7 200 seeks (2 img/s) prennent environ 80 à 140 s. Une pré-passe à 1–2 img/s suffit à repérer les phases (dés posés, main au-dessus du plateau, videau). Il suffit ensuite de densifier à 10 img/s sur des fenêtres de quelques secondes autour de chaque événement.

### 7. GStreamer sous WebKitGTK : limites connues

- **Stabilité** : lecture à vitesse ≠ 1× désynchronisée et gels après `seek` (bugs 245850/245852, 2022 → 2024, encore vus en 2.42.4) [S21][S22].\[18\]\[19\] Reprise ratée après seek dans un lecteur aval (PR WebKit #51698, octobre 2025) [S28].\[28\]
- **Accélération** : avec GStreamer ≥ 1.24, WebKitGTK 2.44 utilise le *sink* DMA-BUF avec modificateurs DRM, ce qui permet une négociation zéro copie [S7].\[6\] 2.46.4 corrige les dimensions vidéo cassées depuis GStreamer 1.24.9 [S44].\[49\]
- **Codecs** : H.264 dépend des greffons installés (gst-libav, ou openh264/VA-API selon la distribution). Le détail Ubuntu, la version de webkit2gtk-4.0 sur 22.04, et le runtime Flatpak GNOME/Freedesktop avec son extension ffmpeg-full n'ont **pas été trouvés** dans cette recherche. À vérifier par `canPlayType('video/mp4; codecs="avc1.640028"')` et `VideoDecoder.isConfigSupported` au démarrage.

### 8. Mémoire et énergie (10 img/s pendant une heure)

Aucune mesure publiée n'a été trouvée. Estimations par calcul :
- **Mémoire** : 36 000 images 320×180 RGBA représenteraient 8,3 Go si on les conservait. Il ne faut **rien conserver** d'autre que des descripteurs (scores, recadrages des dés en 64×64, quelques Ko) dans un tampon circulaire. Les tampons doivent être réutilisés (un canvas, un `ImageData` recyclé) et les `VideoFrame`/`ImageBitmap` fermées sans délai [S45]. Sous WebKitGTK, il faut ≥ 2.46.4 pour les correctifs mémoire de `getImageData` [S44].\[49\]
- **Énergie** : à 1×, le coût est proche d'une lecture vidéo matérielle d'une heure, plus 10 inférences/s d'un petit modèle. La pré-passe accélérée concentre la charge sur quelques minutes : décodeur au maximum et ventilateur probable. Il faut mesurer avec `powermetrics` (macOS), le moniteur de ressources (Windows) et `powertop` (Linux).

## Recommendations

### Chemin portable recommandé (~10 img/s sans bloquer l'interface)
1. **Servir la vidéo en même origine** via l'AssetServer Wails (`Range` + `io.ReadSeeker`). À défaut, garder le serveur loopback avec `Access-Control-Allow-Origin` sur 200/206/OPTIONS, sans redirection, et `crossorigin="anonymous"` posé **avant** `src`.
2. **Thread principal** : `requestVideoFrameCallback` → horodatage `metadata.mediaTime` → `drawImage` sur un canvas 320×180 → `getImageData` → transfert de l'`ArrayBuffer` au worker. Limiter à 10 img/s en comparant les `mediaTime`.
3. **Worker** : inférence Wasm monothread ou WebGL. Pas de SharedArrayBuffer.
4. **Analyse hiérarchique** : pré-passe à 1–2 img/s (WebCodecs images-clés si disponible, sinon `seek` séquentiels attendus par rVFC ou `seeked`), puis passe dense à 10 img/s autour des événements.

### Chemins de repli
- **Pas d'`OffscreenCanvas`** (WebKitGTK < 2.46) : réduction sur le thread principal, worker classique sur `ArrayBuffer`.
- **Pas de rVFC** : `requestAnimationFrame` + `currentTime`, dédoublonnage par `getVideoPlaybackQuality()`.
- **Pas de WebCodecs** (WebKitGTK 2.42, ou H.264 indisponible) : pré-passe par `seek` (`fastSeek` sous WKWebView, `currentTime` ailleurs), avec un seek en vol à la fois.
- **Canvas teinté malgré CORS** : basculer automatiquement sur la même origine.
- **`playbackRate` instable (WebKitGTK)** : ne jamais dépasser 1–2× sous Linux ; préférer les seeks.

## Caveats

Les versions d'apparition de rVFC (Safari 15.4, Chrome 83), du Wasm SIMD (Safari 16.4, Chrome 91) et de `captureStream` (Chrome 62, absent de Safari) sont désormais recoupées avec les données BCD de caniuse et du Web Platform Features Explorer ; seul `createImageBitmap(video)` par moteur n'est toujours pas vérifié, de même que le Wasm SIMD sous WebKitGTK. Les données WebKitGTK propres à Ubuntu 22.04 et au Flatpak n'ont pas été trouvées. Les plafonds de `playbackRate` de WebKit (AVFoundation et GStreamer) reposent sur des sources secondaires ou anciennes.

## 1. Verdict

La lecture des pixels est **sûre sur les trois moteurs** dès lors que CORS est appliqué à chaque réponse, ou mieux, que la vidéo est servie en même origine. L'échantillonnage à 10 img/s **en lecture 1×** sans bloquer l'interface est **portable** : rVFC partout dans les versions cibles, canvas réduit, analyse en worker. L'objectif **« 1 h à 10 img/s en moins de 5 min » n'est pas atteignable de façon portable avec `<video>`**. Les seeks Chromium les plus rapides mesurés donnent 400–700 s pour 36 000 images. `playbackRate` est plafonné par le compositeur, et instable sous GStreamer. Il faut le remplacer par une analyse hiérarchique.

## 2. Recommandation technique

- Exiger **WebKitGTK ≥ 2.46.4** sous Linux, pour `OffscreenCanvas` par défaut, WebCodecs actif par défaut et les correctifs mémoire de `getImageData`. Garder 2.42/2.44 en mode dégradé.
- Viser **macOS 13 avec Safari ≥ 16.4**, ce qui apporte `OffscreenCanvas` et `VideoDecoder`.
- Code par **détection de fonctionnalité**, jamais par version.
- Pré-passe : **WebCodecs sur images-clés** (worker, `fetch` par plages, démultiplexeur JS) là où `VideoDecoder.isConfigSupported` accepte le H.264 du fichier. Sinon, seeks séquentiels à 1–2 img/s.
- Passe dense : lecture 1× (ou 2× sous Chromium/WKWebView) avec rVFC à 10 img/s sur des fenêtres de ±5 s autour des événements.
- Abandonner SharedArrayBuffer, `captureStream` et `MediaStreamTrackProcessor`.

## 3. Incertitudes (et expériences à mener)

1. **CORS sur 206** : servir un MP4 de 1 Go en loopback et retirer l'en-tête sur un 206 sur deux. Vérifier si `getImageData` lève `SecurityError` sur les 3 moteurs, puis tester une redirection 302 en cours de lecture.
2. **LNA dans WebView2** : depuis `http://wails.localhost`, mesurer si une requête vers `http://127.0.0.1:port` déclenche une invite ou un blocage selon la version d'Edge (≥ 142, ≥ 146).
3. **Même origine sous WebKitGTK** : vérifier que GStreamer lit et seek correctement une vidéo servie par le schéma `wails://` (`Range`, durée, seek à la fin).
4. **Banc de seeks** : un MP4 H.264 1080p30 avec une image-clé toutes les 2 s, et 500 seeks précis aléatoires puis séquentiels (pas de 0,5 s). Mesurer `seeked` → rVFC (p50/p95) par moteur, et `fastSeek` sous WKWebView.
5. **Banc `playbackRate`** : 2×, 4×, 8×, 12×, 16×. Compter les rappels rVFC/s et `getVideoPlaybackQuality().droppedVideoFrames`, et vérifier la monotonie de `mediaTime`.
6. **WebCodecs WebKitGTK 2.44 contre 2.46** : `typeof VideoDecoder` et `isConfigSupported({codec:'avc1.640028'})` sous Ubuntu 22.04/24.04, Fedora et Flatpak GNOME.
7. **Coût de la copie** : 10 000 cycles `drawImage` 1080p → 320×180 + `getImageData`, avec et sans `willReadFrequently`, mesurés via `performance.now()` sur les trois moteurs.
8. **Énergie** : une heure à 1× + 10 img/s contre une pré-passe de 5 min, sur portable sur batterie (Wh consommés).

## 4. Sources

1. « Implement HTMLVideoElement.requestVideoFrameCallback() », bug 211945 — WebKit Bugzilla, 2020. https://bugs.webkit.org/show_bug.cgi?id=211945 — Ouverture de la demande rVFC dans WebKit. Primaire.
2. Changeset r284528 — WebKit Trac, 2021. https://trac.webkit.org/changeset/284528/webkit — Ajout de l'IDL rVFC et de la préférence expérimentale (défaut `false`). Primaire.
3. « WebKitGTK 2.36.0 released! » — WebKitGTK Project, 21/03/2022. https://webkitgtk.org/2022/03/21/webkitgtk2.36.0-released.html — « Add support for requestVideoFrameCallback ». Primaire.
4. « Perform efficient per-video-frame operations » — web.dev (Google). https://web.dev/articles/requestvideoframecallback-rvfc — rVFC « best effort », polyfill rAF + `getVideoPlaybackQuality`. Secondaire.
4b. « HTMLVideoElement: requestVideoFrameCallback() » — MDN. https://developer.mozilla.org/en-US/docs/Web/API/HTMLVideoElement/requestVideoFrameCallback — Cadence plafonnée par le rafraîchissement de l'écran. Secondaire.
5. « WebKitGTK 2.46.0 released! » — WebKitGTK Project, 17/09/2024. https://webkitgtk.org/2024/09/17/webkitgtk2.46.0-released.html — Skia, rendu GPU, « Enable offscreen canvas by default ». Primaire.
6. « Graphics improvements in WebKitGTK and WPEWebKit 2.46 » — Carlos García Campos (Igalia), 27/09/2024. https://blogs.igalia.com/carlosgc/2024/09/27/graphics-improvements-in-webkitgtk-and-wpewebkit-2-46/ — Canvas accéléré, `willReadFrequently` force le non-accéléré. Blog (mainteneur).
7. « What's new in WebKitGTK 2.44? » — WebKitGTK Project, 27/03/2024. https://webkitgtk.org/2024/03/27/webkigit-2.44.html — WebCodecs pris en charge, sink DMA-BUF avec GStreamer 1.24. Primaire.
8. « Igalia and WebKit: status update and plans (2024) » — Igalia, via mariospr.org, 2024. https://mariospr.org/category/planet-webkitgtk/ — « WebCodecs… enabled by default in 2.46 » (contredit 7). Blog (mainteneur).
9. « WebKit Features in Safari 16.4 » — WebKit (Apple), 28/03/2023. https://webkit.org/blog/13966/webkit-features-in-safari-16-4/ — OffscreenCanvas 2D, partie vidéo de WebCodecs. Primaire.
10. « WebKit Features in Safari 17.0 » — WebKit (Apple), 2023. https://webkit.org/blog/14445/webkit-features-in-safari-17-0/ — WebGL dans OffscreenCanvas. Primaire.
11. « WebKit Features in Safari 17.2 » — WebKit (Apple), 2023. https://webkit.org/blog/14787/webkit-features-in-safari-17-2/ — H.264 L1T2 dans WebCodecs, correctifs des décodeurs. Primaire.
12. « WebCodecs API » — openpwa.net (miroir BCD). https://openpwa.net/reference/capabilities/webcodecs/ — Chrome 94, Safari 16.4 vidéo, Safari 26 audio. Secondaire.
13. « WebKit Features in Safari 18.0 » — WebKit (Apple), 16/09/2024. https://webkit.org/blog/15865/webkit-features-in-safari-18-0/ — Traitement MediaStreamTrack en dedicated worker. Primaire.
14. « Unbundling MediaStreamTrackProcessor and VideoTrackGenerator » — Mozilla WebRTC blog. https://blog.mozilla.org/webrtc/unbundling-mediastreamtrackprocessor-and-videotrackgenerator/ — API worker-only ; Chrome expose une version ancienne incompatible. Blog (éditeur).
15. « [GStreamer] Add support for canvas captureStream », bug 169811 — WebKit Bugzilla, 2017. https://bugs.webkit.org/show_bug.cgi?id=169811 — « we do not support captureStream for the moment ». Primaire.
16. « [GLIB] Enable SharedArrayBuffer in GLib-backed WebKit (WPE, GTK) », commit 104506c / 321889@main, bug 317847 — WebKit (GitHub), ~25/09/2026. https://github.com/WebKit/WebKit/commit/104506cdb2c45aa547db1eb50313abc15473b815 — SAB seulement si l'origine est isolée ; isolation auparavant incomplète (b/268068). Primaire.\[13\]
17. PR #10 « Tauri v2, Wails v3, Electron native shells » — uist1idrju3i/web-playground (GitHub), 2026. https://github.com/uist1idrju3i/web-playground/pull/10 — WebKitGTK 2.50.4 : `crossOriginIsolated` faux sous schéma personnalisé. Secondaire (bot).
18. Issue #287 « Spike: wss dial, secure context, and SharedArrayBuffer » — logos-fleet (GitHub), 2026. https://github.com/logos-fleet/logos-workspace/issues/287 — iPad : COOP/COEP via WKURLSchemeHandler → non isolé, loopback → isolé. Secondaire.
19. (réservé)
20. « [GStreamer] High playback rate causes crash », bug 128453 — WebKit Bugzilla, 2014. https://bugs.webkit.org/show_bug.cgi?id=128453 — Garde-fou dans `setRate` ; suggestion de borne (−10, 10). Primaire.
20b. `MediaPlayerPrivateGStreamer.cpp` (copie historique) — Chromium/Blink git. https://chromium.googlesource.com/chromium/blink/+/09ce30e2ac620c37b731386dd7dea149d99ea270/Source/WebCore/platform/graphics/gstreamer/MediaPlayerPrivateGStreamer.cpp — Coupure audio hors [0,8 ; 2] sans préservation de hauteur. Primaire (ancien).
21. « [GStreamer] Video playback is highly unreliable, slow, and buggy », bug 245850 — WebKit Bugzilla, 2022. https://bugs.webkit.org/show_bug.cgi?id=245850 — Gels au défilement, vitesse ≠ 1× désynchronisée. Primaire.
22. Bug 245852 — WebKit Bugzilla, 2022–2024. https://bugs.webkit.org/show_bug.cgi?id=245852 — Avance rapide incontrôlée et gels encore en 2.42.4 / GStreamer 1.22.8. Primaire.
23. « Media updates in Chrome 63/64 » — Chrome for Developers, 2017. https://developer.chrome.com/blog/media-updates-in-chrome-63-64 — `playbackRate` hors [0,0625 ; 16] → `NotSupportedError` ; fin de la coupure audio. Primaire.\[20\]
24. FB9343129 (miroir) — feedback-assistant/reports #229 (GitHub), 16/07/2021. https://github.com/feedback-assistant/reports/issues/229 — AVPlayer > 2× n'affiche que les images-clés ; « fixed in macOS 13 » selon un utilisateur. Secondaire.
25. PR #60 — KyleStay/video-speed-controller (GitHub), 21/10/2026. https://github.com/KyleStay/video-speed-controller/pull/60 — Safari : plage 0,07–16 (simulateur). Secondaire.
26. « [GStreamer][MSE] FastSeek is broken », bug 260607 / PR #17541 — WebKit, 07/09/2023. https://github.com/WebKit/WebKit/pull/17541 — fastSeek cassé en MSE sous GStreamer. Primaire.
27. « HTMLMediaElement API: fastSeek » — caniuse / web-features. https://caniuse.com/mdn-api_htmlmediaelement_fastseek — Safari 8+, Firefox 31+, Chrome non (crbug 41276303). Secondaire (données BCD).\[26\]\[55\]
28. « [GStreamer] Sometimes progressive playback is not resumed after seek », PR #51698 — WebKit (GitHub), 02/10/2025. https://github.com/WebKit/WebKit/pull/51698 — Seek de ~2 s avant PAUSED dans un cas aval. Primaire.
29. Bug 2049301 « Higher video seek-to-first-frame latency than Chrome on MacOS » — Mozilla Bugzilla, 2026. https://bugzilla.mozilla.org/show_bug.cgi?id=2049301 — Chrome 149 : 10,7–18,7 ms par seek (H.264 720p). Primaire.
30. HTML Standard, « Media elements » — WHATWG. https://html.spec.whatwg.org/multipage/media.html — Origine de la ressource média, CORS-cross-origin, `fastSeek`, `playbackRate`. Primaire.\[31\]
31. Issue #6783 « CORS-cross-origin is used for image data and media data but appears undefined » — whatwg/html (GitHub). https://github.com/whatwg/html/issues/6783 — Zone floue de la spécification. Primaire.
32. « Use cross-origin images in a canvas » — MDN. https://developer.mozilla.org/en-US/docs/Web/HTML/How_to/CORS_enabled_image — Méthodes qui lèvent `SecurityError` sur un canvas teinté. Secondaire.
33. « Cross-Origin Resource Sharing (CORS) » — MDN. https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS — `Range` simple seulement à plage unique ; redirections après pré-vol. Secondaire.
34. Changelog Wails — wails.io. https://wails.io/changelog/ — v2.5.1 (16/05/2023) : schéma `wails://` sous macOS/Linux ; v2.17.0 (18/10/2026). Primaire.
35. Issue #4386 « wails.localhost » — wailsapp/wails (GitHub). https://github.com/wailsapp/wails/issues/4386 — `wails.localhost` sous Windows, absent sous macOS. Secondaire.
36. Release v2.3.0 — wailsapp/wails (GitHub), 29/12/2022. https://github.com/wailsapp/wails/releases/tag/v2.3.0 — AssetServer : requêtes Range, en-têtes sous WebKit2GTK 2.36+, streaming macOS. Primaire.
37. « Google Chrome 142 restricts local network access » — gHacks, 29/10/2025. https://www.ghacks.net/2025/10/29/google-chrome-142-restricts-local-network-access-and-changes-sync-on-desktop/ — Définition des requêtes locales et loopback. Secondaire.
38. « Local Network Access » — Chrome for Developers. https://developer.chrome.com/docs/modern-web-guidance/guides/security/local-network-access — Permissions `local-network` et `loopback-network`, Permissions-Policy. Primaire.
39. « Chrome Local Network Access for APIs and WebSockets » — Localtonet. https://localtonet.com/blog/chrome-local-network-access-apis-websockets — Chrome 147 (avril 2026) étend LNA aux WebSockets. Blog.
40. « Error message – Local network access permission denied » — Dynamsoft docs (GitHub). https://github.com/dynamsoft-docs/web-twain-docs/blob/master/_articles/faq/chromium-142-local-network-access-issue.md — Chrome 142 (28/10/2025), séparation en Chrome 145. Secondaire.
41. « WebKit2.SecurityManager » — WebKitGTK reference. https://webkitgtk.org/reference/webkit2gtk/stable/class.SecurityManager.html — Schémas CORS-enabled, secure, local. Primaire.
42. PR #77183 — WebKit (GitHub). https://github.com/WebKit/WebKit/pull/77183 — `getVideoPlaybackQuality` GStreamer conforme (images rendues et perdues). Primaire (date non trouvée).
43. « WebKitGTK 2.46.2 released! » — WebKitGTK Project, 21/10/2024. https://webkitgtk.org/2024/10/21/webkitgtk2.46.2-released.html — Mémoire de `putImageData` sur canvas accéléré. Primaire.
44. « WebKitGTK 2.46.4 released! » — WebKitGTK Project, 27/11/2024. https://webkitgtk.org/2024/11/27/webkitgtk2.46.4-released.html — Mémoire et performance de `getImageData` ; dimensions vidéo GStreamer 1.24.9. Primaire.
45. Issue #1378 « WebCodecs API Interoperability » — web-platform-tests/interop (GitHub), 2026. https://github.com/web-platform-tests/interop/issues/1378 — Fuites de `VideoFrame` vers WebGL/OffscreenCanvas sur WebKit mobile. Secondaire.
46. PR #28 — andresaclan/airtime (GitHub). https://github.com/andresaclan/airtime/pull/28 — Chrome, défilement 1080p60 H.264 : 22–29 img/s, p95 67–275 ms. Blog/projet personnel.

## Sources

1. [WebKitGTK 2.36.0 released! - The WebKitGTK Project](https://webkitgtk.org/2022/03/21/webkitgtk2.36.0-released.html)
2. [Changeset 284528](https://trac.webkit.org/changeset/284528/webkit)
3. [WebKitGTK 2.46.0 released! - The WebKitGTK Project](https://webkitgtk.org/2024/09/17/webkitgtk2.46.0-released.html)
4. [WebKit Features in Safari 16.4](https://webkit.org/blog/13966/webkit-features-in-safari-16-4/)
5. [WebKit Features in Safari 17.0](https://webkit.org/blog/14445/webkit-features-in-safari-17-0/)
6. [What's new in WebKitGTK 2.44? - The WebKitGTK Project](https://webkitgtk.org/2024/03/27/webkigit-2.44.html)
7. [Planet WebKitGTK](https://mariospr.org/category/planet-webkitgtk/)
8. [WebCodecs API](https://openpwa.net/reference/capabilities/webcodecs/)
9. [WebKit Features in Safari 17.2](https://webkit.org/blog/14787/webkit-features-in-safari-17-2/)
10. [Unbundling MediaStreamTrackProcessor and VideoTrackGenerator - Advancing WebRTC](https://blog.mozilla.org/webrtc/unbundling-mediastreamtrackprocessor-and-videotrackgenerator/)
11. [WebKit Features in Safari 18.0](https://webkit.org/blog/15865/webkit-features-in-safari-18-0/)
12. [169811](https://bugs.webkit.org/show_bug.cgi?id=169811)
13. [\[GLIB\] Enable SharedArrayBuffer in GLib-backed WebKit (WPE, GTK) · WebKit/WebKit@104506c · GitHub](https://github.com/WebKit/WebKit/commit/104506cdb2c45aa547db1eb50313abc15473b815)
14. [feat: add Tauri v2, Wails v3, and Electron native shells for BrowserFeatures by devin-ai-integration\[bot\] · Pull Request #10 · uist1idrju3i/web-playground · GitHub](https://github.com/uist1idrju3i/web-playground/pull/10)
15. [Spike: wss dial, secure context, and SharedArrayBuffer from the Worker on device · Issue #287 · logos-fleet/logos-workspace · GitHub](https://github.com/logos-fleet/logos-workspace/issues/287)
16. [github.com](https://github.com/WebKit/WebKit/commit/b6f736948251f4c1c30b3e6d3db738c03909ef54)
17. [128453](https://bugs.webkit.org/show_bug.cgi?id=128453)
18. [245850](https://bugs.webkit.org/show_bug.cgi?id=245850)
19. [245852](https://bugs.webkit.org/show_bug.cgi?id=245852)
20. [Media updates in Chrome 63/64 | Blog | Chrome for Developers](https://developer.chrome.com/blog/media-updates-in-chrome-63-64)
21. [How to Play YouTube Videos at Custom Speed](https://www.labnol.org/youtube-custom-speed-201125)
22. [Fix: clamp Smart Speed playback rate to 16x in \_setRate by AryaXDG · Pull Request #4371 · code-charity/youtube · GitHub](https://github.com/code-charity/youtube/pull/4371)
23. [video playbackrate](https://chromium.googlesource.com/chromium/src/+/80bb4f04ec77b8cba2067f8dab8121cb0b6be9b8/third_party/blink/web_tests/media/video-playbackrate.html)
24. [Improve Safari playback efficiency and cap speeds at 16× by KyleStay · Pull Request #60 · KyleStay/video-speed-controller · GitHub](https://github.com/KyleStay/video-speed-controller/pull/60)
25. [github.com](https://github.com/WebKit/WebKit/pull/17541)
26. [fast seek](https://web-platform-dx.github.io/web-features-explorer/features/fast-seek)
27. [HTMLMediaElement.fastSeek - Web APIs - W3cubDocs](https://docs.w3cub.com/dom/htmlmediaelement/fastseek)
28. [github.com](https://github.com/WebKit/WebKit/pull/51698)
29. [2049301 - Higher video seek-to-first-frame latency than Chrome on MacOS hardware decoding](https://bugzilla.mozilla.org/show_bug.cgi?id=2049301)
30. [CORS-cross-origin is used for "image data" and "media data" but appears undefined · Issue #6783 · whatwg/html · GitHub](https://github.com/whatwg/html/issues/6783)
31. [4.8.8 The video element](https://html.spec.whatwg.org/multipage/media.html)
32. [Use cross-origin images in a canvas - HTML - MDN Web Docs](https://developer.mozilla.org/en-US/docs/Web/HTML/How_to/CORS_enabled_image)
33. [Cross-Origin Resource Sharing (CORS) - MDN Web Docs](https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS)
34. [HTMLMediaElement: crossOrigin property - Web APIs](https://developer.mozilla.org/en-US/docs/Web/API/HTMLMediaElement/crossOrigin)
35. [Changelog](https://wails.io/changelog/)
36. [wails.localhost · Issue #4386 · wailsapp/wails · GitHub](https://github.com/wailsapp/wails/issues/4386)
37. [github.com](https://github.com/wailsapp/wails/releases/tag/v2.3.0)
38. [Google Chrome 142 restricts local network access and changes Sync on desktop - gHacks Tech News](https://www.ghacks.net/2025/10/29/google-chrome-142-restricts-local-network-access-and-changes-sync-on-desktop/)
39. [web-twain-docs/\_articles/faq/chromium-142-local-network-access-issue.md at master · dynamsoft-docs/web-twain-docs · GitHub](https://github.com/dynamsoft-docs/web-twain-docs/blob/master/_articles/faq/chromium-142-local-network-access-issue.md)
40. [Configure Chrome to Suppress the Local Network Access Prompt for Okta FastPass](https://support.okta.com/help/s/article/configure-chrome-to-suppress-the-local-network-access-prompt-for-okta-fastpass?language=en_US)
41. [Chrome Local Network Access for APIs and WebSockets](https://localtonet.com/blog/chrome-local-network-access-apis-websockets)
42. [webkitgtk.org](https://webkitgtk.org/reference/webkit2gtk/stable/class.SecurityManager.html)
43. [211945](https://bugs.webkit.org/show_bug.cgi?id=211945)
44. [Perform efficient per-video-frame operations | Articles | web.dev](https://web.dev/articles/requestvideoframecallback-rvfc)
45. [HTMLVideoElement: requestVideoFrameCallback() method](https://developer.mozilla.org/en-US/docs/Web/API/HTMLVideoElement/requestVideoFrameCallback)
46. [\[GStreamer\] Media Playback Quality support isn't spec conformant by eocanha · Pull Request #77183 · WebKit/WebKit · GitHub](https://github.com/WebKit/WebKit/pull/77183)
47. [Graphics improvements in WebKitGTK and WPEWebKit 2.46](https://blogs.igalia.com/carlosgc/2024/09/27/graphics-improvements-in-webkitgtk-and-wpewebkit-2-46/)
48. [WebKitGTK 2.46.2 released! - The WebKitGTK Project](https://webkitgtk.org/2024/10/21/webkitgtk2.46.2-released.html)
49. [WebKitGTK 2.46.4 released! - The WebKitGTK Project](https://webkitgtk.org/2024/11/27/webkitgtk2.46.4-released.html)
50. [WebCodecs API Interoperability · Issue #1378 · web-platform-tests/interop · GitHub](https://github.com/web-platform-tests/interop/issues/1378)
51. [\[pull\] main from WebKit:main by pull\[bot\] · Pull Request #2401 · AKJUS/WebKit · GitHub](https://github.com/AKJUS/WebKit/pull/2401)
52. [FB9343129: AVPlayer should support playing videos smoothly at a rate higher than 2x](https://github.com/feedback-assistant/reports/issues/229)
53. [Source/WebCore/platform/graphics/gstreamer/MediaPlayerPrivateGStreamer.cpp - chromium/blink - Git at Google](https://chromium.googlesource.com/chromium/blink/+/09ce30e2ac620c37b731386dd7dea149d99ea270/Source/WebCore/platform/graphics/gstreamer/MediaPlayerPrivateGStreamer.cpp)
54. [Scrub like QuickTime: cached frames while seeking, one seek in flight by andresaclan · Pull Request #28 · andresaclan/airtime · GitHub](https://github.com/andresaclan/airtime/pull/28)
55. [HTMLMediaElement API: fastSeek](https://caniuse.com/mdn-api_htmlmediaelement_fastseek)
