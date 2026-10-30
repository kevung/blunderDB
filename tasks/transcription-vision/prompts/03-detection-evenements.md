# 03 — Détection d'événements : lancer, repos des dés, mains, remise à zéro

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

Le premier livrable est une **pré-passe** qui parcourt la vidéo et produit une ligne de temps
d'événements : *lancer* (les dés bougent dans leur zone), *repos* (les dés sont posés et immobiles),
*mains dans le plateau* et *mains hors du plateau* (début et fin du déplacement des pions),
*remise à zéro* (position initiale : nouvelle partie), et un signal *incertain*. Les images sont
échantillonnées à 5–10 par seconde, réduites à ~320×180 ou à des régions d'intérêt (zone des dés,
plateau) calibrées une fois par vidéo. Tout tourne en JavaScript sur des `TypedArray` dans un Web
Worker, sans bibliothèque.

1. **Détection de mouvement sans modèle** : différence d'images, moyenne glissante de fond
   (running average, MOG/KNN en version simplifiée), énergie de mouvement par région, seuils
   adaptatifs, hystérésis et durées minimales ; comparaison documentée de leur robustesse aux
   changements d'exposition automatique de la caméra, au scintillement, aux overlays animés du
   stream, aux ombres de mains.
2. **Flux optique** : Lucas-Kanade par blocs ou Farnebäck sont-ils utiles ici ou trop coûteux pour
   ce qu'ils apportent ? Variantes « block matching » légères ; coût en JS à 320×180.
3. **Segmentation temporelle d'activité** dans les vidéos de jeux de table et de sport (« table-top
   game video analysis », « event detection board game video », travaux sur le poker, le go, les
   échecs, les jeux de cartes) : architectures d'états (machine à états avec durées minimales,
   HMM, modèles légers de type TCN sur descripteurs simples), précisions, latence de décision.
4. **Repos des dés** : critère de stabilité (N images consécutives sous un seuil dans la ROI dés),
   pièges : dé qui roule lentement, main qui reste au-dessus, dés ramassés tout de suite par le
   joueur, dés jetés hors de la ROI, dés dans un gobelet.
5. **Remise à zéro** : reconnaître la position initiale par comparaison à un gabarit après
   homographie ; robustesse quand les joueurs remettent les pions en désordre pendant plusieurs
   secondes.
6. **Qui a lancé** : les dés tombent du côté du lanceur ; fiabilité d'une règle « ROI gauche /
   ROI droite » ; cas des deux joueurs qui lancent dans la même moitié.
7. **Évaluation** : métriques usuelles pour une ligne de temps d'événements (précision/rappel par
   événement avec tolérance temporelle, décalage médian), et un protocole d'annotation d'un jeu de
   test à partir de quelques matchs déjà transcrits et horodatés.

Attendu : l'architecture de détection recommandée (descripteurs + machine à états), les seuils et
durées de départ justifiés par la littérature, et les trois pannes les plus probables à surveiller.
