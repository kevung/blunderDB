# 06 — Amorçage des données et apprentissage faiblement supervisé

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

La base de données contient déjà des matchs transcrits à la main avec, pour chaque coup, l'instant
du lancer (`roll_tick_ms`) et l'instant de la décision (`tick_ms`) dans la vidéo, les dés, le coup
joué et le plateau avant/après. C'est un jeu étiqueté gratuit mais **bruité** : l'instant posé par
l'utilisateur peut précéder ou suivre de une à plusieurs secondes le moment où les dés sont lisibles
ou le plateau stable, et quelques coups sont mal transcrits. Les vidéos viennent de quelques streams
(peu de caméras différentes au départ). Chaque utilisateur qui corrige l'assistant produit de
nouvelles étiquettes.

1. **Extraction d'imagettes à partir d'horodatages approximatifs** : techniques pour choisir, dans
   une fenêtre autour de l'instant annoté, l'image où les dés sont posés et lisibles (critère de
   stabilité, score de netteté, accord entre images voisines) ; littérature sur l'alignement
   temporel faible (weak temporal alignment, « learning from timestamps », supervision ponctuelle en
   segmentation d'actions).
2. **Étiquettes bruitées** : méthodes robustes pour de petits classifieurs (co-teaching, pertes
   robustes, filtrage par confiance, « confident learning ») ; sont-elles utiles à l'échelle de
   quelques milliers d'exemples ?
3. **Peu de caméras** : augmentation de données pertinente pour des dés et des pions (perspective,
   couleur, flou, compression vidéo, overlays), et risque de sur-apprentissage à un stream ; rendu
   synthétique de dés et de plateaux (Blender, rendus procéduraux) : apport documenté pour ce genre
   d'objets simples.
4. **Apprentissage actif et auto-amélioration locale** : l'assistant propose, l'utilisateur corrige ;
   comment choisir ce qu'on garde comme nouvel exemple, et peut-on affiner un petit modèle sur la
   machine de l'utilisateur (dans la webview ou côté Go) sans chaîne Python ? Travaux sur le
   fine-tuning en navigateur et sur l'adaptation par calibration légère (prototypes, k-NN sur
   embeddings) comme alternative au ré-entraînement.
5. **Protocole d'évaluation** : séparation par vidéo (jamais par image), métriques par dé et par
   coup, courbe précision–couverture pour choisir le seuil de confiance au-delà duquel l'assistant
   valide seul.
6. **Taille de corpus** : ordres de grandeur rapportés pour atteindre > 99 % par dé et > 95 % de
   bonne discrimination de coup, pour des objets aussi simples, avec et sans pré-entraînement.
7. **Éthique et données** : les vidéos sont celles de tiers (streams publics) ; l'entraînement
   ne doit produire que des poids distribués, jamais des images ; précautions usuelles.

Attendu : la chaîne d'amorçage recommandée (de la base aux imagettes étiquetées), la stratégie
d'amélioration continue la plus simple qui tienne, et les tailles de corpus à viser pour chaque
phase.
