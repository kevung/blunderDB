# Bloc commun à coller en tête de chaque prompt

Ce bloc est déjà inclus dans chaque fichier `NN-*.md` ; il est reproduit ici pour référence.

---

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
plutôt que d'extrapoler. Termine par :

1. **Verdict** — trois à cinq phrases qui tranchent la question posée.
2. **Recommandation technique** — ce que tu implémenterais en premier, et pourquoi.
3. **Incertitudes** — ce qui reste à vérifier par une expérience, avec l'expérience à mener.
4. **Sources** — liste numérotée : titre, auteurs ou éditeur, année, lien, une ligne sur ce qu'on y
   trouve et une mention de fiabilité (primaire / secondaire / blog).

Réponds en français ; les termes techniques peuvent rester en anglais.
