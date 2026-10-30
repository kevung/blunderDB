# Banc V0.1 — la webview Linux (WebKitGTK) porte-t-elle le chantier vision ?

Issue #843. Cadre : ADR-0108, ADR-0109 ; questions : `synthese.md` §01 (expériences 3, 4, 5, 6)
et §05 (expérience 2, **bloquante**).

## Protocole

**Poste.** Arch Linux, noyau 7.1.9, X11 ; Intel Xeon E5-2683 v3 (Haswell, 28 fils, AVX2),
NVIDIA GTX TITAN X ; WebKitGTK 2.54.2 (`webkit2gtk-4.1`), GStreamer 1.28.8 avec `gst-libav`.
Politique GPU de l'application inchangée (`WebviewGpuPolicyOnDemand`, soit `ALWAYS` côté WebKit).
Deux modes mesurés : `wails dev` (origine `wails://wails.localhost:34115`, ressources servies
par Vite à travers le schéma) et le binaire de `wails build` (origine `wails://wails`, ressources
embarquées par `embed.FS`). Les chiffres ci-dessous sont ceux du binaire ; les quatre passes
en `wails dev` donnent les mêmes à ±5 %.

**Vidéo d'essai** (générée, non versionnée) : 120 s, 1920×1080, 30 img/s, H.264 High, CRF 20,
image clé toutes les 2 s exactement (`-g 60 -keyint_min 60 -sc_threshold 0`), `+faststart`,
114 Mo. Les 64 px du haut portent le **numéro de l'image** en 12 blocs binaires de 160 px :
chaque mesure lit dans les pixels l'image réellement composée et la compare à `mediaTime`
(et non à une horloge supposée).

```sh
ffmpeg -f lavfi -i testsrc2=s=1920x1080:r=30:d=120 \
  -f lavfi -i "nullsrc=s=1920x64:r=30:d=120,format=gray,geq=lum='255*mod(floor(N/pow(2\,floor(X/160)))\,2)'" \
  -filter_complex "[0][1]overlay=0:0:shortest=1,format=yuv420p" \
  -c:v libx264 -profile:v high -preset veryfast -crf 20 -g 60 -keyint_min 60 -sc_threshold 0 \
  -movflags +faststart bench1080.mp4
```

**Page d'essai** (dans l'historique de la branche `feat/v01-webview-bench`, retirée par son
dernier commit) : `frontend/public/bench/` (`v01.html`, `v01.js`, workers `w-*.js`) et
`internal/gui/bench_v01.go`. Rien ne s'active sans la variable `BLUNDERDB_BENCH_V01` : elle
nomme la vidéo, que le serveur de bouclage (`internal/gui/media.go`) sert alors **avec CORS**
(`Access-Control-Allow-Origin: *` sur toutes les réponses, OPTIONS compris) ; trois secondes
après le démarrage, la fenêtre navigue vers `wails://…/bench/v01.html`, la page poste son
rapport JSON au bouclage après chaque section, Go l'écrit dans `BLUNDERDB_BENCH_V01_OUT` et
quitte l'application. Reproduire :

```sh
git checkout d28ee8f37                 # la page d'essai, voir « Historique »
cd frontend && npm ci && cd ..
BLUNDERDB_BENCH_V01=/chemin/bench1080.mp4 BLUNDERDB_BENCH_V01_OUT=/tmp/v01.json \
  wails dev -tags webkit2_41           # ou : build, puis le binaire avec les mêmes variables
```

Sections de la page, dans l'ordre :

1. **Workers** : `fetch` du script, puis `new Worker` classique (absolu, relatif,
   `importScripts`), module (avec et sans `import` statique, et par
   `new URL('./w.js', import.meta.url)` — la forme que Vite réécrit), puis le repli par URL
   Blob (classique, module, module important une URL `wails://` absolue). Chaque essai attend
   un `pong` (délai 5 s). Puis 1 000 allers-retours d'un `ArrayBuffer` de 320×180×4 transféré
   (vérifie que la source est détachée).
2. **rVFC** pendant 12 s de lecture à 1× : `mediaTime`, `currentTime` lu dans le rappel,
   `presentedFrames`, et l'image décodée des pixels ; 6 s de plus sans traitement.
3. **Coût par image**, vidéo en pause : 2 000 cycles `drawImage` 1080p → 320×180 puis
   `getImageData`, avec et sans `willReadFrequently` ; et pendant la lecture à 1× de (2), le
   pipeline complet par rappel (dessin, lecture, décodage du numéro, `postMessage` transféré
   au Worker).
4. **Seeks** : 50 vers une image clé, 50 au milieu du GOP (+31 images), 50 en fin de GOP
   (+59), en ordre mélangé, puis 200 séquentiels au pas de 0,5 s ; mesure `currentTime =` →
   `seeked` et → rappel rVFC, et exactitude de l'image affichée. **`playbackRate`** 2× et 4×,
   6 s chacun, avec le pipeline de (3).
5. **Détection** (sans mesure) : `VideoFrame`, `VideoDecoder` et `isConfigSupported`,
   `OffscreenCanvas` (page et Worker), Wasm SIMD par `WebAssembly.validate`,
   `crossOriginIsolated`, `SharedArrayBuffer` ; en plus, canvas teinté sans l'attribut
   `crossorigin`.

L'horloge `performance.now()` de WebKitGTK est grossière (pas de 1 ms) : les coûts unitaires
viennent de moyennes sur un lot, les percentiles par itération ne sont que des bornes.

## Mesures

### 1. Worker depuis `wails://` — **vert**

| Essai | dev | binaire |
|---|---|---|
| `fetch('/bench/w-classic.js')` | 200, `text/javascript` | 200, `text/javascript; charset=utf-8` |
| `new Worker('/bench/w-classic.js')` (classique, absolu) | oui, 6 ms | oui, 6 ms |
| `new Worker('w-classic.js')` (relatif) | oui | oui |
| classique + `importScripts` | oui | oui |
| `{type: 'module'}` avec `import` statique relatif | oui | oui |
| `{type: 'module'}` sans import | oui | oui |
| `new Worker(new URL('./w-module.js', import.meta.url), {type: 'module'})` | oui | oui |
| repli URL Blob, classique | oui | oui |
| repli URL Blob, module | oui | oui |
| URL Blob module important une URL `wails://` absolue | oui | oui |

Le Worker voit `self.location` en `wails://…/bench/w-module.js` : le schéma personnalisé est
une origine à part entière pour les Workers, `import` compris. Transfert d'un `ArrayBuffer`
de 230 400 octets (320×180×4) et retour : **0,10 ms** de moyenne sur 1 000 allers-retours,
source détachée (`byteLength === 0`), taille conservée au retour.

### 2. `requestVideoFrameCallback` et `mediaTime` — **présent mais muet**

`HTMLVideoElement.prototype.requestVideoFrameCallback` est une fonction, mais **aucun rappel
n'arrive**, ni pendant la lecture (0 rappel en 12 s à 1×, en 6 s à 2× et 4×, cinq passes) ni
après un `seek` en pause (0 sur 250). Le rappel était demandé avant `play()` et de nouveau à
l'événement `playing`, sur un `<video>` visible de 480 px. `metadata.mediaTime` n'a donc pas pu
être comparé. `getVideoPlaybackQuality()` existe mais rend `totalVideoFrames = 0` et
`droppedVideoFrames = 0` en toute circonstance : inutilisable sous GStreamer.

Repli mesuré à sa place — échantillonneur `requestAnimationFrame` qui lit `currentTime` puis
dessine et décode le numéro d'image :

| Lecture | échantillons/s | images distinctes vues | images du média sur la durée | `floor(currentTime×30)` = image affichée |
|---|---|---|---|---|
| 1×, 12 s, pipeline complet | 60 | 361 | 361 (**aucune perdue**) | 717/722 (99,3 %), sinon ±1 |
| 1×, 6 s | 60 | 179 | ~180 | 356/360, sinon ±1 (une seule autre : l'image d'avant le seek) |
| 2×, 6 s | 46 | 270 | 474 (57 %) | 57/275 ; **+1 image** pour 217 |
| 4×, 6 s | 46 | 263 | 947 (28 %) | 19/277 ; +1 image pour 249 |

À 1×, `currentTime` lu dans rAF désigne l'image composée à une image près, et rAF voit toutes
les images. À 2× et 4×, `currentTime` devance l'image affichée d'une image dans quatre cas sur
cinq. L'index décodé ne recule jamais. Canvas sans l'attribut `crossorigin` : `getImageData`
lève `SecurityError` (attendu) ; avec `crossorigin="anonymous"` et CORS sur le bouclage, lisible.

### 3. Coût d'une image

Vidéo en pause, 2 000 cycles, moyenne du lot :

| Canvas 320×180 | `drawImage` 1080p → 320×180 | `getImageData` | total |
|---|---|---|---|
| défaut | 4,2 ms | 0,37 ms (pics à 47 ms) | 4,5–4,8 ms |
| `willReadFrequently: true` | 4,3 ms | 0,37 ms (max 2 ms) | 4,6–5,0 ms |

Le coût est dans `drawImage` (la lecture de l'image 1080p décodée), pas dans `getImageData` ;
`willReadFrequently` ne coûte rien et supprime les pics de `getImageData`. Pendant la lecture
à 1×, pipeline complet par image (dessin, lecture, décodage, `postMessage` transféré au
Worker) : **4,7 ms** de moyenne, p95 6 ms, max 6 ms ; 30 images/s tenues sans perte, le média
avance à 1,000× (aucune saccade visible : 361 images consécutives).

### 4. Seeks et `playbackRate`

`currentTime =` → `seeked`, vidéo en pause, image affichée relue dans les pixels :

| Cible | n | p50 | p95 | max | image exacte |
|---|---|---|---|---|---|
| image clé | 50 | 46 ms | 49 ms | 52 ms | 50/50 |
| milieu de GOP (+31) | 50 | 111 ms | 124 ms | 137 ms | 50/50 |
| fin de GOP (+59) | 50 | 165 ms | 167 ms | 175 ms | 50/50 |
| séquentiel, pas de 0,5 s | 100 | 97 ms | 140 ms | 142 ms | 100/100 |

Le seek est **toujours exact** à l'image ; son coût croît d'environ 2 ms par image à décoder
depuis l'image clé. `fastSeek` existe (non mesuré). À 2× et 4×, la vidéo suit
(`currentTime` avance à 1,96× et 3,89×) ; l'échantillonneur rAF y voit 45 images/s, soit
≈ 23 et ≈ 11 images **du média** par seconde.

Deux pièges observés :
- **`play()` ne se résout jamais** quand `playbackRate ≠ 1` (2× et 4×, trois passes) ; à 1×
  la promesse se résout en 17 ms. Attendre `play()` à 2× a figé la section jusqu'à `ended`.
- **Canvas teinté après `ended`** : dans deux passes sur trois où la lecture à 2× a atteint la
  fin du fichier puis a été ramenée par `seek`, tout `getImageData` suivant a levé
  `SecurityError`, sur un canvas jusque-là lisible. Cause non isolée.

### 5. Détection (sans mesure)

| | page | Worker |
|---|---|---|
| `VideoFrame`, `VideoDecoder` | présents | présents |
| `VideoDecoder.isConfigSupported(...)` | **fige la page** | **fige la page** |
| `OffscreenCanvas` (+ contexte 2d) | présent | présent, 2d obtenu |
| Wasm SIMD (`WebAssembly.validate`) | vrai (CPU AVX2) | vrai |
| `crossOriginIsolated` | faux | faux |
| `SharedArrayBuffer` | absent | absent |
| `fastSeek`, `createImageBitmap` | présents | — |

`VideoDecoder.isConfigSupported` ne rend jamais la main et gèle tout le processus web :
minuteries comprises, plus aucun rapport après l'appel (codecs `avc1.640028` puis
`vp09.00.10.08` dans la page, la liste entière dans un Worker ; cinq passes, plus de 3 min
chacune). Le constructeur `new VideoFrame(video)` n'a pas pu être atteint pour cette raison.

## Verdict

**Point 1 : vert.** Un Worker se charge depuis `wails://` sous WebKitGTK 2.54, en script
classique comme en module ES avec imports, en `wails dev` comme dans le binaire ; le repli par
URL Blob marche aussi mais n'est pas nécessaire. Le chantier continue.

Le reste du cadre de l'ADR-0109 tient, sauf l'horloge : **rVFC est muet** ici, et
**WebCodecs est inutilisable** (gel).

## Conséquences pour l'ADR-0109

- *Chargement du Worker (V1.1)* : `new Worker(new URL('./x.js', import.meta.url), {type:
  'module'})`, la forme que Vite sait empaqueter ; aucun repli Blob à écrire.
- *L'instant du média* : la règle « `mediaTime` de rVFC » devient « `mediaTime` si un premier
  rappel rVFC arrive, sinon `currentTime` lu dans `requestAnimationFrame`, une image retenue à
  chaque changement ». rVFC se détecte **par un rappel reçu**, jamais par `typeof`. À 1× l'écart
  est d'au plus une image (33 ms à 30 img/s) ; en lecture accélérée, retrancher une image ou
  tolérer ±1 image dans `roll_tick_ms` / `tick_ms`.
- *Chemin opportuniste* : confirmé et moins cher que prévu. À 1×, chaque image est accessible
  pour ~5 ms du thread principal ; 10 img/s coûtent ≈ 5 % du thread, 30 img/s ≈ 15 %. Une
  lecture à 2× fournit encore ≈ 23 img/s du média, à 4× ≈ 11 : l'utilisateur qui regarde en
  accéléré nourrit toujours des automates à 10 img/s.
- *Fenêtre [−1 ; +2] s* : rien ne la remet en cause ; à 10 img/s, une fenêtre de 3 s coûte
  ≈ 30 × 5 ms.
- *Résolution 320×180* : confirmée ; le coût est la lecture de l'image 1080p (`drawImage`),
  pas la taille de la cible — réduire davantage ne ferait presque rien gagner. Créer le canvas
  avec `willReadFrequently: true`.
- *Passe hiérarchique par seeks* : faisable en arrière-plan, exacte à l'image. Une heure à
  1 img/s ≈ 3 600 seeks × ~100 ms ≈ 6 min ; à 2 img/s ≈ 11 min. Le coût dépend du GOP du
  fichier (≈ 2 ms par image décodée depuis l'image clé) : mesurer le GOP réel des vidéos du
  corpus avant de chiffrer. Le chemin WebCodecs sur images clés est **exclu sous Linux** tant
  que `isConfigSupported` gèle la page ; ne jamais l'appeler sans le savoir sûr.
- *Garde-fous du code de la phase 1* : ne jamais `await video.play()` quand `playbackRate ≠ 1` ;
  surveiller `SecurityError` sur `getImageData` après `ended` (recharger la source si besoin) ;
  ne pas compter sur `getVideoPlaybackQuality`.
- *Inférence (ADR-0110)* : Worker monothread, transfert d'`ArrayBuffer`, Wasm SIMD validé,
  `OffscreenCanvas` disponible dans le Worker ; ni `SharedArrayBuffer` ni origine isolée —
  conforme à la synthèse §05.

## À reprendre

- **rVFC** : muet sur ce poste (NVIDIA, pilote propriétaire, X11). Refaire la mesure sur une
  carte Intel ou AMD et sous Wayland avant de conclure que WebKitGTK ne le livre jamais.
- **`isConfigSupported`** : reproduire hors Wails (MiniBrowser de WebKitGTK) avant de signaler
  le gel en amont.
- **Canvas teinté après `ended`** : isoler (CORS des 206 rejouées ? nouveau pipeline GStreamer ?).
- Banc d'énergie (synthèse 01, expérience 7) et banc CORS sur 206 alternés (expérience 1) :
  hors de ce banc.

## WebView2 (Windows) — poste requis

Même page, même protocole ; en plus, la permission *Local Network Access* de Chromium ≥ 142
vers `127.0.0.1` (synthèse 01, expérience 2). Non mesuré : aucun poste Windows.

## WKWebView (macOS) — poste requis

Même page, même protocole ; en plus `fastSeek` (synthèse 01, expérience 3). Non mesuré : aucun
poste macOS.

## Historique

La page d'essai est dans `d28ee8f37` (branche `feat/v01-webview-bench`) ; le commit suivant la
retire. `git checkout d28ee8f37` puis la commande du protocole la remet en service.
