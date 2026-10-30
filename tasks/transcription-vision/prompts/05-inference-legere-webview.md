# 05 — Inférence légère dans la webview, sans dépendance lourde

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

Les modèles envisagés sont petits : un classifieur de dés sur imagette 32×32 → 6 classes
(~50 k paramètres), un classifieur par point du plateau (imagette d'une bande → couleur × nombre
de pions, ~100–500 k paramètres), peut-être un détecteur de cube. Ils tournent dans la webview,
dans un Web Worker, à 5–10 inférences par seconde en régime de croisière et bien plus pendant une
pré-passe. L'application embarque déjà, côté Go, un réseau de backgammon stocké en float16 avec son
propre moteur d'inférence écrit à la main (pas de dépendance) ; l'équipe sait écrire ce genre de
code.

Compare, chiffres à l'appui :

1. **JavaScript écrit à la main** sur `Float32Array` (convolutions, GEMM, quantification int8) :
   latence attendue pour les tailles ci-dessus dans V8 et JavaScriptCore ; projets qui ont fait ce
   choix et leur retour.
2. **WebAssembly écrit à la main ou compilé depuis Go** (`GOOS=js GOARCH=wasm`, TinyGo) : gain
   réel sur JS, taille du module, support SIMD dans WebKitGTK/JavaScriptCore et WebView2 ; coût de
   l'aller-retour des données.
3. **ONNX Runtime Web** : taille des artefacts wasm (avec/sans SIMD, threads, WebGPU), compatibilité
   documentée ou rapportée avec WebKitGTK et WKWebView, licence, fréquence de ruptures.
4. **TensorFlow.js** (backends wasm, webgl, webgpu) : même grille.
5. **WebNN** et **WebGPU** : état réel en 2026 dans les trois moteurs ; prématuré ?
6. **Inférence côté Go pur** (sans CGO) : bibliothèques maintenues en 2026 (ONNX en Go pur, gorgonia,
   spago, ou code maison comme le moteur existant) ; coût de transporter des imagettes du frontend
   vers Go par le pont Wails (JSON/base64) à 10 img/s ; intérêt : un seul moteur d'inférence pour le
   backgammon et la vision.
7. **Distribution des poids** : embarquer dans le bundle frontend ou dans le binaire Go
   (`go:embed`), formats compacts (float16, int8), ordre de grandeur acceptable (le binaire fait
   quelques dizaines de Mo).
8. **Entraînement** : chaîne Python usuelle (PyTorch → export ONNX ou poids bruts) et outils pour
   exporter vers un format maison trivial à lire ; quantification post-entraînement pour des modèles
   de cette taille.

Attendu : une recommandation ferme pour la phase 2 (dés) et une pour la phase 3 (plateau), avec le
seuil de taille de modèle à partir duquel l'option « sans dépendance » cesse d'être raisonnable.
