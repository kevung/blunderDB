# 01 — Accès aux images d'un `<video>` dans WebKitGTK, WebView2 et WKWebView

Contexte. Je conçois une fonction pour blunderDB, une application de bureau de backgammon
(Go + Svelte dans une webview via Wails v2, binaire unique pour Linux/WebKitGTK, Windows/WebView2,
macOS/WKWebView). Le panneau de transcription lit une vidéo de match (fichier local) ; l'utilisateur
saisit les dés au clavier, l'application liste les coups légaux classés par un réseau de neurones de
backgammon, l'utilisateur choisit le coup joué. Je veux qu'une analyse des images de la vidéo
détecte les lancers, lise les dés, reconnaisse le coup joué et les gestes de videau, pour que
l'utilisateur ne fasse plus que confirmer ou corriger. Contraintes : aucun décodage vidéo hors de la
webview, pas de dépendance native (pas de CGO, pas de ffmpeg, pas d'OpenCV natif), une caméra de
stream généralement fixe mais d'angle, de lumière et de matériel variables.

Exigences sur la réponse. Travaille comme pour une revue de littérature : privilégie les sources
primaires (articles avec DOI ou arXiv, dépôts avec code, documentation des éditeurs de navigateurs,
bugzilla/chromium-status, caniuse) et date chaque fait. Couvre 2015–2026, en insistant sur
2023–2026. Quand deux sources se contredisent, dis-le. Quand tu ne trouves pas, dis « non trouvé »
plutôt que d'extrapoler. Termine par : 1. **Verdict** ; 2. **Recommandation technique** ;
3. **Incertitudes** (avec l'expérience à mener) ; 4. **Sources** numérotées (titre, auteurs ou
éditeur, année, lien, une ligne de contenu, fiabilité primaire / secondaire / blog). Réponds en
français.

---

## Question

Le fichier vidéo est servi par un serveur HTTP loopback (`127.0.0.1`, port aléatoire, URL sous
jeton) à un `<video>` dont la page a une autre origine (schéma Wails). Les versions cibles :
WebKitGTK 2.42 à 2.48 (webkit2gtk-4.1 ; aussi 4.0 pour Ubuntu 22.04), WebView2 Evergreen (Chromium
récent), WKWebView de macOS 13 à 15.

Pour chacun des trois moteurs, avec version d'apparition et source primaire :

1. **Lecture des pixels** : `canvas.drawImage(video)` puis `getImageData` avec
   `crossorigin="anonymous"` sur la balise et `Access-Control-Allow-Origin` côté loopback. Règles
   exactes de *tainted canvas* ; pièges connus (redirections, requêtes `Range` et CORS, en-têtes à
   renvoyer sur les 206). Le serveur loopback doit déjà gérer `Range` pour la lecture.
2. **Cadence** : `requestVideoFrameCallback` (support, précision des métadonnées `mediaTime`,
   comportement en pause et en `seek`), sinon `requestAnimationFrame` + `currentTime`.
3. **Hors du thread principal** : `OffscreenCanvas` dans un Web Worker, `createImageBitmap(video)`,
   transfert de `VideoFrame`/`ImageBitmap` au worker, coût d'une copie 1920×1080 → 320×180.
4. **WebCodecs** (`VideoDecoder`, `MediaStreamTrackProcessor`, `captureStream()` sur `<video>`) :
   disponibles ? utiles ici ou superflus ?
5. **WebAssembly** : SIMD et threads (`SharedArrayBuffer`, en-têtes COOP/COEP nécessaires sous un
   schéma personnalisé Wails ?).
6. **Pré-passe accélérée** : `playbackRate` maximal réellement honoré *avec rendu des images* (pas
   seulement l'audio coupé), par moteur ; alternative par `seek` successifs : précision du
   positionnement (`fastSeek` vs `currentTime =`), latence par seek pour un H.264 1080p, nombre de
   seeks par seconde atteignable. Objectif : analyser un match de 60 min en moins de 5 min.
7. **GStreamer sous WebKitGTK** : limites connues du pipeline média (seek lent, `playbackRate`
   plafonné, frames non rendues en accéléré, codecs selon la distribution et le Flatpak).
8. **Mémoire et énergie** : coût d'un échantillonnage à 10 img/s pendant une heure sur un portable.

Attendu : une **matrice de compatibilité** (moteur × fonction × version × source), puis le chemin le
plus portable pour échantillonner ~10 images/s sans bloquer l'interface, et le chemin de repli si
une fonction manque sur un moteur.
