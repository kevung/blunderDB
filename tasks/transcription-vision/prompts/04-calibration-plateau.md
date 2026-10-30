# 04 — Calibration du plateau et robustesse géométrique

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

L'utilisateur calibre une fois par vidéo : il clique les quatre coins du plateau (et peut-être les
deux zones de dés), l'application en déduit une homographie et la position attendue des 24 points,
de la barre et des zones de sortie. Cette calibration est une préférence locale, jamais exportée.

1. **Homographie à quatre points** en JavaScript sans bibliothèque : estimation (DLT), stabilité
   numérique, raffinement par les bords du plateau (détection de lignes), gestion d'un plateau dont
   le cadre déborde du champ.
2. **Modèle géométrique du plateau** : où tombent les pions sur un point en fonction de la hauteur
   de pile (1 à 15 pions, chevauchement au-delà de 5, piles qui penchent) ; comment les travaux sur
   le backgammon ou le go modélisent la position attendue de chaque pion pour compter.
3. **Dérive de caméra** : détection d'un déplacement de la caméra ou d'un changement de zoom en
   cours de stream (recadrage de la régie), recalibration automatique par suivi des coins ou
   relocalisation sur gabarit, alerte à l'utilisateur. Techniques légères (suivi KLT de quelques
   coins, corrélation de phase).
4. **Perspective forte** : angles typiques des streams de backgammon (vue de dessus, vue oblique
   derrière un joueur, double caméra en incrustation) ; à partir de quelle obliquité la lecture des
   points lointains devient-elle non fiable ; stratégie quand la vidéo alterne entre plusieurs
   angles (plans de coupe, visages, publicité).
5. **Occlusions** : mains, bras, gobelet, cube posés sur le plateau ; comment ne comparer que les
   images sans occlusion (masque d'occlusion par différence au fond, couleur de peau, délai après la
   dernière activité).
6. **Multi-plateaux** : certaines retransmissions montrent deux tables ; l'utilisateur choisit la
   sienne au cadrage — y a-t-il des pièges connus ?
7. **Ergonomie de la calibration** : outils d'annotation vidéo qui demandent une calibration par
   l'utilisateur (quatre coins d'un terrain de sport, d'un échiquier) : comment la rendent-ils
   rapide et vérifiable (surimpression du modèle sur la vidéo, poignées déplaçables, validation sur
   une image à position connue comme la position initiale) ?

Attendu : le modèle géométrique minimal à implémenter, la stratégie de recalibration, et la
surimpression qui permet à l'utilisateur de voir en une seconde que la calibration est juste.
