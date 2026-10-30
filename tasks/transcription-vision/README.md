# Assistant de transcription par la vidéo — phase de spécification

Chantier ouvert le 2026-10-30. Objectif : un mode du panneau de transcription où l'analyse des
images de la vidéo détecte les lancers, lit les dés, reconnaît le coup joué et les gestes de
videau ; l'utilisateur n'intervient que pour corriger ou quand la détection échoue.

Cadre fixé avant la bibliographie (voir la cartographie du panneau dans la discussion du 30 oct.) :

- **Fichier vidéo seulement.** YouTube est un iframe d'une autre origine : aucun pixel n'en sort.
- **Tout le pixel vit dans la webview** (canvas, Web Worker). Go ne décode rien (ADR-0082 règle 6,
  pur Go, pas de ffmpeg) ; il garde les règles, les candidats et la machine de transcription.
- **La base est déjà un jeu d'entraînement** : `video_source` + `roll_tick_ms` / `tick_ms` donnent
  des couples (image, dés) et (image, `board_after`).
- **L'équité n'est pas un a priori** : l'utiliser pour deviner le coup blanchirait les blunders
  (prolonge ADR-0044 : le moteur propose, personne ne décide à la place de l'utilisateur).
- Trois phases : (1) pré-passe des lancers et saut d'événement en événement ; (2) lecture des dés
  et pré-remplissage ; (3) reconnaissance du coup par discrimination entre candidats, videau, mode
  automatique. Chaque phase a son ADR.

## Bibliographie

Les prompts de `prompts/` sont à lancer un par un sur claude.ai (recherche approfondie). Chaque
réponse se colle dans `reponses/NN-<sujet>.md`, nom identique au prompt. Les prompts demandent un
format de réponse commun (sources numérotées, verdict, incertitudes) pour que la synthèse puisse se
faire mécaniquement.

| N° | Sujet | Ce que la réponse doit trancher |
|---|---|---|
| 01 | Accès aux images dans les trois webviews | Le chemin portable pour échantillonner ~10 img/s sans bloquer l'UI ; la vitesse maximale d'une pré-passe |
| 02 | Dés et plateau de backgammon : état de l'art | Techniques, précisions, jeux de données ; ce qui se transfère des échecs |
| 03 | Détection d'événements : lancer, repos, mains, remise à zéro | Différence d'images vs flux optique vs petit modèle ; seuils, hystérésis |
| 04 | Calibration du plateau et robustesse géométrique | Homographie à 4 coins, dérive de caméra, zoom des streams, occlusions |
| 05 | Inférence légère dans la webview | JS à la main vs ONNX Runtime Web vs TF.js vs Go pur ; taille, latence, compatibilité |
| 06 | Amorçage des données et apprentissage faiblement supervisé | Tirer un modèle de ses propres transcriptions horodatées ; étiquettes bruitées ; apprentissage actif |
| 07 | YouTube : accès légitime aux images | Réponse franche oui/non, CGU, ce qu'on peut honnêtement proposer |
| 08 | UX des outils d'annotation « humain dans la boucle » | Mécanismes qui rendent la correction moins chère que la vérification ; seuils de confiance acceptés |
| 09 | Petits objets : videau et score | Suivi du cube, lecture de la face, OCR de l'overlay ; quand renoncer |

Après les réponses : synthèse dans `synthese.md`, puis ADR de la phase 1 et feuille de route.
