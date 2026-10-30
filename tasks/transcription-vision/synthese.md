# Synthèse de la bibliographie — assistant de transcription par la vidéo

Une section par réponse reçue, dans l'ordre d'arrivée : ce qu'elle confirme, ce qu'elle change au
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
