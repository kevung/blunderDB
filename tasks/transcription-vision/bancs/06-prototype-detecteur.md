# Banc V1.8 — prototype d'un détecteur de dés appris (jetable, hors application)

## Question

V1 (descripteurs census + automates, ADR-0109) a échoué sur la vidéo réelle du match 285 :
`dice.rest` contre les Repères de jet, F1 0,046 (`04-mesure.md`, § Première mesure réelle). Un
détecteur appris trouve-t-il et lit-il les dés nettement mieux sur nos deux vidéos, en
restant dans le cadre d'ADR-0108/0110 : Worker de webview WebKitGTK, pas de
SharedArrayBuffer, poids hors du binaire, corpus-manifeste, rien d'écrit chez le destinataire ?

Réponse courte : **oui, d'un ordre de grandeur**, entraîné sur une vidéo et testé sur l'autre
(cadrage paysage ↔ portrait).

## Modèle et licence

Deux petits réseaux écrits à la main en PyTorch et entraînés à partir de zéro. Pas de poids
pré-entraînés, pas d'Ultralytics. Dépendances d'entraînement : torch et torchvision (BSD-3),
OpenCV (Apache-2.0). Inférence : onnxruntime-web (MIT).

- **Détecteur** : réseau entièrement convolutif à carte de chaleur, façon CenterNet, sans
  régression de boîte, parce qu'un dé a une taille quasi fixe (environ 16 px à 640×480).
  - Entrée : l'image entière en demi-résolution (320×240).
  - Six couches conv 3×3 + BN + ReLU (16-24-32-48-48-48 canaux, deux à pas 2, deux dilatées).
  - Sortie : deux canaux au pas de 8 px en pleine résolution, un par dé (noir, blanc).
  - 66 626 paramètres.
  - Décodage : le maximum de chaque canal ; le jeu n'a qu'un dé noir et un dé blanc.
- **Lecteur** : exactement l'architecture d'ADR-0110 règle 1. Imagette 32×32 découpée en
  pleine résolution autour du pic, trois blocs conv (16, 32, 64) + max-pool, moyenne globale,
  dense à 7 sorties (faces 1-6, puis « pas un dé »), 24 151 paramètres. La classe 7 tient ici
  le rôle de la classe « illisible » de l'ADR et sert de veto contre les fausses détections.

Les deux réseaux s'exportent en ONNX opset 17. L'écart avec PyTorch est inférieur à 5e-6.

## Données et annotation

- Vidéo 1 : match 285, `.webm` 640×480 paysage, 97 min, 383 Repères.
- Vidéo 2 : ronde 4, `.mp4` 480×640 portrait, 81 min, 271 Repères.
- Décodage à 8 img/s. Les images à 4 img/s sont gardées en JPEG (3,5 Go pour les deux vidéos,
  hors dépôt).

**Jeu public** : la seule source trouvée est Roboflow Universe, dont le téléchargement passe
vraisemblablement par un compte ; je ne l'ai pas vérifié et n'ai créé aucun compte. Ces jeux
montrent de toute façon de gros dés photographiés de près, loin de nos dés de 16 px vus de
dessus.

**Amorce automatique, sans annotation manuelle** :

1. **Générateur classique** (OpenCV) : composantes connexes blanches (V > 205, S < 45) ou
   noires (V < 70), de 6 à 24 px, presque carrées et pleines. On exige au moins un point de
   couleur contraire à l'intérieur, une position stable à 3 px près sur trois images
   consécutives, et une zone de jeu rectangulaire fixée à la main par vidéo.
2. **Image entièrement étiquetée** : on ne garde que les images où l'on voit exactement un dé
   noir et un dé blanc stables. Les deux joueurs partagent une seule paire, un dé noir et un
   dé blanc ; une telle image n'a donc aucun dé non étiqueté. On obtient 1533 images sur 23 302
   pour v1 et 1124 sur 19 570 pour v2.
3. **Valeurs** : la paire transcrite est attachée aux images de [R − 1,5 s ; R + 0,5 s], puis
   propagée aux images de ±8 s où les deux dés n'ont pas bougé de plus de 4 px.
   - L'ordre est inconnu : le lecteur apprend avec une perte qui ignore quel dé porte quelle
     valeur, −log(p_noir(a)·p_blanc(b) + p_noir(b)·p_blanc(a)).
   - Les négatifs (« pas un dé ») sont des imagettes tirées au hasard à au moins 14 px des dés.
4. **Auto-étiquetage par le détecteur** : le lecteur entraîné sur la seule amorce classique
   est faible (voir les chiffres). On fait donc tourner le détecteur entraîné sur l'autre
   vidéo sur les images de la vidéo d'entraînement, et on prend les images de [R − 3 s ; R + 1 s]
   où les deux dés sont détectés (score ≥ 0,5) à la même place qu'à l'image la plus proche
   de R. On obtient 226 coups (2067 images) sur v1 et 130 coups (1183 images) sur v2, contre
   une quarantaine avec la seule amorce.
   - Les positions viennent alors d'un détecteur qui a vu l'autre vidéo, mais le lecteur ne
     voit jamais un pixel de la vidéo de test.

**Limites des étiquettes** (à garder en tête pour lire les chiffres) :
- **Bruit d'étiquette d'environ 8 %** dans l'amorce classique, d'après un contrôle visuel de
  64 imagettes : creux sombres entre deux pions noirs, bords de pions blancs.
- **Peu de coups porteurs de valeurs** par la seule amorce : 44 coups sur v1, 33 sur v2.
- Le dé blanc est souvent centré trop bas de quelques pixels, parce que la perspective montre
  une face latérale. On compense par une gigue de ±4 px à l'entraînement.

## Protocole de test

- **Test sur la vidéo jamais vue** : on entraîne sur une vidéo et on teste sur l'autre.
- **Coups annotés à la main exclus** : les 47 + 49 coups annotés (instant lisible connu
  dans `annot.csv` ; quatre coups sans instant sont écartés) sortent de tout entraînement,
  avec leurs fenêtres [−10 s ; +5 s] autour de l'instant lisible et du Repère, dans les deux
  sens.
- **À l'instant lisible ±0,5 s** (images à 8 img/s) :
  - Un dé est présent si le score du détecteur est ≥ 0,3 et que P(« pas un dé ») < 0,5.
  - « Trouvés » : au moins une image de la fenêtre a les deux dés présents.
  - Valeur : somme des log-probabilités par dé sur ces images (vote d'ADR-0110), comparée à la
    transcription sans tenir compte de l'ordre.
- **Événement `dice.rest`** (4 img/s) : les deux dés sont présents et immobiles à 4 px près
  pendant 3 images (0,75 s). Un nouvel événement n'est émis que si l'un des dés a bougé de plus
  de 8 px depuis le repos précédent.
- **Deux mesures de l'événement** :
  - `tools/vision/mesure.py`, la même que pour V1. Elle n'est évaluable que sur v1 : le
    `marks.csv` de v2 n'a pas de `prev_tick_ms`, ses 271 fenêtres sont toutes non évaluables.
  - Une fenêtre fixe [R − 7 s ; R + 1 s] tirée de `05-histogramme.md`, avec appariement un à un
    (`fenetre.py`).
- **Référence** : le générateur classique seul passe par la même règle d'événement. C'est une
  base « sans apprentissage ».

## Chiffres

À l'instant lisible ±0,5 s, sur la vidéo de test :

| test (entraîné sur) | dés trouvés | valeur exacte, lecteur amorce classique | valeur exacte, lecteur auto-étiqueté | classique seul, trouvés |
|---|---|---|---|---|
| v2 (v1) | 47/49 (96 %) | 22/49 | **46/49 (94 %)** | 19/49 |
| v1 (v2) | 46/47 (98 %) | 10/47 | **38/47 (81 %)** | 16/47 |

Les erreurs de valeur restantes sur v1 n'ont pas été examinées une à une. Sur v2, avant
l'auto-étiquetage, les imagettes en erreur montraient des dés bien localisés et lisibles à
l'œil, avec une fois le videau pris pour un dé blanc. Le maillon faible était donc le lecteur,
faute d'exemples, pas le détecteur.

Événement `dice.rest` contre les Repères de jet :

| vidéo de test | mesure | TP | FP | FN | précision | rappel | F1 | FP/min |
|---|---|---|---|---|---|---|---|---|
| v1 | `mesure.py` ±0,5 s (88 fenêtres non évaluables) | 224 | 258 | 71 | 0,465 | 0,759 | **0,577** | 2,7 |
| v1, 55 premières minutes (couverture de V1) | `mesure.py` | 100 | 154 | 37 | 0,394 | 0,730 | **0,512** | 2,8 |
| v1 | fenêtre [R − 7 ; R + 1] s | 336 | 146 | 47 | 0,697 | 0,877 | 0,777 | 1,5 |
| v2 | fenêtre [R − 7 ; R + 1] s | 244 | 236 | 27 | 0,508 | 0,900 | 0,650 | 2,9 |
| v1, classique seul | `mesure.py` | 46 | 65 | 249 | 0,414 | 0,156 | 0,227 | 0,7 |
| v2, classique seul | fenêtre | 49 | 68 | 222 | 0,419 | 0,181 | 0,253 | 0,8 |

- **Comparaison à V1** : sur le même fichier de Repères et avec la même mesure, F1 0,577
  contre 0,046 pour V1.
- **Le rappel est le point fort** : 76 à 90 % des jets ont un `dice.rest` dans leur fenêtre.
- **D'où viennent les faux positifs** : ce sont surtout de vrais dés qu'on déplace sans jet
  (dé poussé, ramassé puis reposé), et le videau parfois pris pour un dé blanc.
- **Ce qui les réduirait, non essayé ici** :
  - n'émettre que si la valeur lue change ou si le dé a quitté le champ entre deux repos ;
  - une classe « videau » ;
  - le seuil haut des états d'ADR-0108.
- **Les FP/min ne sont qu'un majorant** : ils comptent tout événement non apparié, y compris
  ceux qui tombent dans une fenêtre non évaluable.

## Latence et poids

| réseau | paramètres | ONNX float32 | float16 (estimé) | onnxruntime-web WASM, 1 fil (Node 26) | onnxruntime natif, 1 fil |
|---|---|---|---|---|---|
| détecteur, entrée 1×3×240×320 | 66 626 | 262 Kio | ≈ 130 Kio | 53 ms | 15 ms |
| lecteur, 2 imagettes 32×32 | 24 151 | 96 Kio | ≈ 48 Kio | 0,9 ms | 0,2 ms |

- **Conditions de mesure** : machine chargée par d'autres tâches (charge 20 à 30 sur 16 fils),
  donc des majorants.
- **Coût en régime** : à 4 img/s, environ 22 % d'un cœur dans un Worker, sans
  SharedArrayBuffer (WASM mono-fil).
- **JS écrit à la main** : le détecteur fait environ 0,37 G multiplications par image. En JS à
  la main, à la manière d'ADR-0110 règle 2, il faudrait compter 200 à 400 ms par image ;
  c'est tenable à 2 img/s, mais c'est le cas où un noyau WASM SIMD se justifie.

Coût d'entraînement sur CPU (16 fils, partagés) :
- décodage et générateur classique : environ 15 min par vidéo ;
- détecteur : 25 époques de 20 s, soit environ 8 min ;
- lecteur : environ 2 min.

Le prototype entier tient en moins d'une heure de calcul.

## Verdict

**Le détecteur appris justifie de réviser ADR-0109 sur la détection des dés et de fusionner
V1 et V2.** Le même petit réseau localise les dés dans 96 à 98 % des coups annotés, sur une
vidéo qu'il n'a jamais vue et dans un autre cadrage. Ses `dice.rest` atteignent F1 0,58 contre
0,046 pour V1 sur la même mesure. Le lecteur d'ADR-0110, découpé autour du pic du détecteur
plutôt que dans le plan rectifié, lit 81 à 94 % des jets.

**Il rend inutiles** la calibration de présence et le témoin hors plateau, causes 1 et 2 de
l'échec de V1 : il travaille sur l'image brute, sans référence prise à la calibration.

Ce que la révision devrait trancher :

1. **ADR-0109** : `dice.rest` provient du détecteur, deux dés immobiles pendant N images.
   - La fenêtre et le `mediaTime` du Repère restent.
   - Les automates census restent au plus pour les mains, si elles servent encore.
   - La calibration se réduit, au mieux, à une zone de jeu.
2. **ADR-0110** :
   - L'imagette se découpe autour de la détection, plus dans le plan rectifié.
   - La classe « pas un dé » sert de veto.
   - Le second réseau (environ 67 k paramètres) dépasse le budget implicite du JS écrit à la
     main. Deux voies : un noyau WASM SIMD avec repli JS (règle 2), ou onnxruntime-web en
     WASM mono-fil. Cette dernière ajoute une dépendance d'environ 10 Mo de `.wasm` servie par
     `frontend/dist` : à décider.
   - Poids : safetensors float16, environ 180 Kio pour les deux réseaux.
3. **Corpus** :
   - L'amorce classique et l'auto-étiquetage par le détecteur suffisent à partir des Repères
     et des dés transcrits. Le manifeste d'ADR-0110 règle 4 fournit exactement ces entrées,
     sans aucune annotation manuelle.
   - Il faudrait plus de deux vidéos, d'autres jeux de dés et d'autres éclairages avant de
     croire à la généralisation. Deux tournois filmés par le même dispositif ne prouvent pas
     qu'on lira n'importe quelle vidéo.
4. **Avant d'intégrer** :
   - réduire les faux positifs (valeur qui change, classe « videau ») ;
   - régler les seuils des trois états sur la précision ;
   - mesurer le JS ou le WASM sous JSC.

Les scripts du prototype restent hors dépôt, dans `~/src/vision-proto/scripts/`, avec les
poids et les données. Rien n'a été écrit dans les vidéos, les fichiers de Repères ni
`~/bkgm/unger.db`.
