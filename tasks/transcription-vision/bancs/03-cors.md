# Banc V0.3 : CORS du serveur de bouclage

Fait : les réponses 200, 206 et OPTIONS de `/media/<jeton>` portent `Access-Control-Allow-Origin`
égal à l'origine de la requête (jamais `*`), `Vary: Origin`; le `<video>` des fichiers locaux
porte `crossorigin="anonymous"`. Couvert par `TestMediaHostCORS` (internal/gui/media_test.go).

Vérifié sur WebKitGTK 2.54.2 (2026-11-07), sous Xvfb : une webview servie depuis l'origine
`wails://wails` (schéma enregistré comme le fait Wails) lit une vidéo du vrai `mediaHost`
(`register`), saute à 12,5 s (requête Range), puis `drawImage` + `getImageData` passent ; la
lecture reprend ensuite (12,5 → 14,0 s). Témoin : le même `<video>` sans `crossorigin` lève
`SecurityError`, le banc détecte donc bien un canvas tainted. Même résultat en MP4/H.264 et en
MKV (vidéo de match 09-73). WebView2 (Windows) reste à voir avec #846.

Procédure à la main, pour revérifier dans l'app :
1. `make dev`, ouvrir un fichier vidéo local dans le panneau vidéo.
2. Console du webview (clic droit > Inspecter), avec la vidéo en pause sur une image :
   ```js
   const v = document.querySelector('[data-testid="video-pane"] video');
   const c = document.createElement('canvas'); c.width = v.videoWidth; c.height = v.videoHeight;
   const g = c.getContext('2d'); g.drawImage(v, 0, 0);
   g.getImageData(0, 0, 1, 1);
   ```
3. Attendu : aucune `SecurityError`. Vérifier aussi qu'un saut (Range) et la lecture marchent.

Aucune redirection à traiter : `ServeContent` ne redirige pas.
OPTIONS répond 204 avant le contrôle du jeton, sans fuite : aucun corps, aucun en-tête propre au fichier.

Banc automatique : `03-cors/` (sonde `probe.py`, serveur `zz_manual_serve_test.go.txt` à copier sans `.txt`
temporairement dans `internal/gui/`, lancé avec `CORS_URL_FILE`, `CORS_VIDEO`, `CORS_STOP`).
