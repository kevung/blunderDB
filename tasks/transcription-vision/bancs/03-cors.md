# Banc V0.3 : CORS du serveur de bouclage

Fait : les réponses 200, 206 et OPTIONS de `/media/<jeton>` portent `Access-Control-Allow-Origin`
égal à l'origine de la requête (jamais `*`), `Vary: Origin`; le `<video>` des fichiers locaux
porte `crossorigin="anonymous"`. Couvert par `TestMediaHostCORS` (internal/gui/media_test.go).

Non vérifié : le rendu sous `make dev` (pas d'affichage dans la session de l'ouvrier).

Procédure à la main :
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
