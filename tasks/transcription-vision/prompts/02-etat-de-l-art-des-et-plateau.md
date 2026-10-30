# 02 — État de l'art : dés et plateau de backgammon depuis une image ou une vidéo

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

Recense la littérature et les projets avec code sur trois sujets, en donnant pour chaque source :
méthode, précision rapportée et conditions de mesure, contraintes (caméra fixe ? calibration ?
taille des objets en pixels ?), jeu de données (public ? taille ? licence ?), licence du code.

1. **Lecture de la valeur de dés** dans des images ou vidéos : comptage de points par seuillage et
   composantes connexes, transformée de Hough, détecteurs de blobs ; réseaux convolutifs légers
   (nombre de paramètres, taille d'entrée, précision) ; détecteurs d'objets (YOLO-nano et
   équivalents) ; cas difficiles : dés en perspective, dés de couleur sombre à points clairs,
   deux dés qui se touchent, dé cocké, dé partiellement hors cadre, reflets. Jeux de données publics
   de dés (Kaggle, Roboflow, articles).
2. **État d'un plateau de backgammon** depuis une photo ou une vidéo : localisation du plateau,
   homographie, segmentation des pions par couleur, **comptage de piles** (où les pions se
   chevauchent au-delà de 5), détection de la barre et des pions sortis, estimation de la position
   complète. Projets GitHub, mémoires, articles ; applications mobiles qui prétendent le faire
   (quelle qualité ?).
3. **Transfert depuis les échecs**, bien plus étudiés : reconnaissance d'un échiquier et de ses
   pièces depuis une vidéo de partie réelle (« chess move recognition from video », « chessboard
   recognition », travaux type ChessVision / Chesscog / LiveChess / DGT) et surtout la stratégie
   **différentielle** : comparer deux états stables successifs et ne retenir que le coup *légal*
   compatible avec la différence, au lieu de reconnaître la position en absolu. Quelles précisions
   ces systèmes atteignent-ils, et quelles sont leurs pannes typiques (mains, ombres, pièces
   déplacées puis reposées) ?

Attendu : un tableau comparatif par sujet ; une estimation argumentée de la précision atteignable
pour les dés (par dé) et pour la discrimination entre 2 à 30 coups candidats dont les plateaux
d'arrivée ne diffèrent que sur 2 à 4 points ; les deux ou trois techniques à prototyper en premier.
