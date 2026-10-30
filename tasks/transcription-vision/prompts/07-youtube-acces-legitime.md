# 07 — YouTube : existe-t-il un accès légitime aux images ?

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

Aujourd'hui l'application lit les vidéos YouTube par l'IFrame Player API dans une page hébergée en
loopback (le lecteur exige un référent HTTP) ; seuls l'instant courant, l'état et la vitesse
traversent `postMessage`. Les pixels restent dans l'iframe d'une autre origine.

1. **Conditions d'utilisation** (YouTube Terms of Service, API Services Terms, Developer Policies,
   versions 2024–2026) : que disent-elles précisément de l'accès au contenu vidéo hors lecteur, de
   l'analyse automatique des images d'une vidéo en lecture, et du téléchargement ? Cite les clauses.
2. **IFrame Player API et Data API** : exposent-elles quoi que ce soit d'exploitable (miniatures
   `storyboard`, images à intervalle, flux) ? Les « storyboards » de prévisualisation sont-ils
   accessibles et à quelle résolution/cadence ?
3. **Capture d'écran de la fenêtre** (`getDisplayMedia`, capture native de la fenêtre par l'OS) :
   faisabilité dans WebKitGTK, WebView2, WKWebView ; dialogues système, résolution, cadence ;
   position des CGU sur cette voie ; expérience utilisateur réaliste.
4. **Téléchargement par un outil tiers** (yt-dlp et équivalents) : position juridique en 2025–2026
   (UE, France, États-Unis ; affaires récentes), risques pour une application qui *suggère* à
   l'utilisateur d'utiliser un tel outil sans l'embarquer, et pratique des outils d'annotation
   sportive ou échiquéenne confrontés au même problème.
5. **Alternatives honnêtes** : l'utilisateur fournit son propre fichier (enregistrement d'écran par
   OBS, fichier source du streamer) ; sites qui publient les matchs de backgammon sous licence
   permettant le téléchargement ; Twitch et autres plateformes ont-elles une position différente ?
6. **Prior art** : des projets open source ont-ils trouvé une voie légitime pour analyser des
   images YouTube dans une application de bureau ? Lesquels, et par quel moyen ?

Attendu : une réponse franche oui/non, ce que l'application peut annoncer dans sa documentation
sans exposer son auteur, et la formulation recommandée du message à l'utilisateur qui ouvre une
vidéo YouTube en mode assisté.
