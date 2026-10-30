# 09 — Petits objets : videau et score

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

Le videau (cube de doublement) est un petit cube à six faces numérotées 2, 4, 8, 16, 32, 64, posé
au centre quand personne ne le possède, puis du côté du joueur qui a pris, face visible = valeur.
Un *double* se voit par le cube déplacé vers l'adversaire avant un lancer ; une *prise* par le cube
qui reste chez l'adversaire et la partie qui continue ; une *passe* par une remise à zéro des pions.
Beaucoup de streams affichent aussi un overlay graphique avec score, noms et parfois la valeur du
cube. La machine de transcription connaît déjà le score et le propriétaire du cube ; la vision n'a
qu'à détecter un **changement**.

1. **Suivi d'un petit objet** (10–40 pixels de côté) dans une vidéo compressée : template matching
   avec mise à jour, filtres de corrélation (KCF, MOSSE, CSRT en version légère), détecteurs
   minuscules ; robustesse quand le cube est pris en main et reposé ailleurs ; coût en JS.
2. **Lecture de la face** : classification 6 classes sur une imagette de cube en perspective ;
   précision rapportée pour des chiffres imprimés de cette taille ; alternative : ne pas lire la
   face et déduire la valeur du nombre de doubles (déjà connu par la machine).
3. **Détection du geste de double** : signature temporelle (main vers le cube avant tout lancer,
   cube déplacé de plus d'une distance seuil, pause) ; distinguer d'un cube simplement bousculé ;
   faux positifs typiques.
4. **Overlay du stream** : OCR léger d'un score ou d'une valeur de cube dans une zone fixe (Tesseract
   en wasm est trop lourd ; alternatives : classification par gabarit sur des chiffres de police
   fixe, modèles d'OCR minuscules) ; détection d'un *changement* de l'overlay comme signal de fin de
   partie sans lire la valeur.
5. **Priorité** : à quel point ces détections valent-elles l'effort, sachant que les gestes de
   videau sont rares (quelques-uns par partie) et déjà saisissables d'une touche (`d`, `t`, `p`) ?
   Y a-t-il des travaux qui mesurent le gain d'automatiser des événements rares dans un flux par
   ailleurs automatisé ?

Attendu : une recommandation « faire / différer / renoncer » par sous-détection, avec la technique
la plus simple qui tienne pour celles à faire.
