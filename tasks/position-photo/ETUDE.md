# Position depuis une photo — étude de conception

Étude du 2026-11-22, sans code de production. `C:` = `~/bkgm/bancs/campagne-materiels/coulisses/`.

## Synthèse

- Le lecteur N1 est un petit CNN par pointe (441 066 paramètres, plus 636 642 pour la barre) qui
  se porte en Go pur sans difficulté, environ 2,6 G multiplications par photo.
- « Une erreur de pions par match » ne s'applique pas à une photo. Ce chiffre sort du décodeur, qui
  connaît les dés et les coups légaux. Lu seul, image par image, N1 donne la position exacte sur
  47 à 79 % des images vidéo (mesure ci-dessous). Une photo n'a pas d'a priori : c'est ce chiffre
  qu'elle hérite, avant tout écart de domaine.
- blunderDB a déjà toute la chaîne d'une photo, sauf le lecteur : `capture` lit une capture d'écran
  XG en Go, publiée en GUI, CLI et serveur, et pose le résultat sur le plateau EDIT avec ses doutes.
  La photo s'y branche comme deuxième lecteur.
- Recommandation : mesurer N1 tel quel, en Python, sur 30 photos de développement (tranche 0) avant
  tout portage. Le portage, le parcours et la doc coûtent ensuite 8 à 10 jours d'agent, plus 3 à 5
  jours si un réentraînement est nécessaire.

## 1. Le lecteur N1

Architecture (`C:pions/reseau.py`, repris tel quel par `C:t1/reseau_t1.py:2-5`, seules les données
changent) :
- `bloc` = conv 3×3 → BN → ReLU → conv 3×3 → BN → ReLU → max-pool 2 (`reseau.py:51-52`).
- `Case` = 4 blocs (3→24→48→64→96), Flatten, Dropout 0,3, Linear(96·8·2 → 128), ReLU, Dropout,
  deux têtes (`reseau.py:55-63`).
- Têtes d'une pointe : couleur en 3 classes (vide, foncée, claire) et nombre ordinal en 15 logits
  cumulés « ≥ k ». Elles se combinent en log p conjointe (3×16) par sigmoïde, `cummin`, différences
  et log_softmax (`reseau.py:66-75`).
- La barre est un second `Case((5, 5), 256)` qui donne le nombre de pions par couleur, 0 à 4
  (`reseau.py:134`).
- Normalisation de l'entrée : (x/255 − 0,45)/0,25, ordre BGR d'OpenCV (`reseau.py:46-48`).
  Entraînement : AdamW, OneCycle, 14 époques, augmentations de teinte complète, luminosité, flou,
  bruit, décalage ±4 px, échelle ±10 % et miroir (`reseau.py:22`, `reseau.py:29-43`).
- Les augmentations ne contiennent aucune déformation perspective.

Poids : `C:t1/cache/modele_cnn_tous.pt` (1,78 Mo) et `modele_barre_tous.pt` (2,57 Mo), 62 tenseurs
chacun. Comptés sur les fichiers : 441 066 et 636 642 paramètres. En float16, BN repliée, cela fait
environ 2,2 Mo à embarquer, contre 3,3 Mo pour les trois `.bin16` du moteur
(`pkg/blunderdb/engine/gammonnet/network.go:58-62`).

Entrées (`C:pions/geometrie.py`, `C:pions/bandes.py`) :
- Il faut 8 coins, 4 par table (HG, HD, BD, BG), et non 4. Chaque table est redressée par sa
  propre homographie, la barre par le quadrilatère entre les tables (`geometrie.py:26-41`).
- Le plateau canonique fait 660×520 : deux tables de 300×520 et une barre de 60, soit 1,2 largeur
  de pointe (`geometrie.py:14`).
- Vignette d'une pointe : colonne de 50 px élargie de 15 px de chaque côté, demi-hauteur, rangée
  basse retournée pour mettre le bord en haut, puis 80×260 → 40×128 en INTER_AREA (`bandes.py:4-6`,
  `bandes.py:18-33`). Vignette de la barre : 40×256.
- Les coins sont posés à la main, une fois par vidéo, sur une image de référence. Chaque image est
  recalée par SIFT + RANSAC (`geometrie.py:44-62`), ce qu'une photo n'utilise pas.
- La sortie se lit en couleurs relatives (foncée, claire). Il faut deux données : `h`, le coin de la
  pointe 1, et `c1`, la couleur du joueur 1 (`C:pions/lecture.py:7-10`, `lecture.py:37-44`). En
  vidéo, `h` vient de la position de départ et `c1` des Repères. Une photo n'a ni l'un ni l'autre :
  l'utilisateur les donne.

Dépendances : torch et opencv (`reseau.py:15`), utilisés hors de blunderDB.

Exactitude image par image (mesurée pour cette étude, lecture seule, par un script dérivé de
`C:t1/exactitude.py` ; images de repos de R1-R3, chaque modèle laissant son match de côté) :

| match | pointe juste | image exacte | ≤ 1 erreur | ≥ 4 erreurs | erreurs/image | erreurs « marge < 2 » |
|---|---|---|---|---|---|---|
| R1 (213 im.) | 0,988 | 0,789 | 0,948 | 0,005 | 0,29 | 28 % |
| R2 (313 im.) | 0,965 | 0,473 | 0,802 | 0,029 | 0,84 | 56 % |
| R3 (180 im.) | 0,982 | 0,756 | 0,906 | 0,028 | 0,44 | 66 % |

Sans N1 (témoin) : image exacte 0,211, 0,045 et 0,267. La plupart des erreurs de nombre sont à ±1
(204 sur 221).

Le reste du chantier vision reste faible. Les dés lus seuls, sur dés cliqués, donnent 87 % de
paires justes en neutre (`~/bkgm/bancs/campagne-materiels/BILAN.md:15`, ligne « Lecteur seul ») et
55 à 92 % en détection selon le plateau. Aucun lecteur de videau n'est retenu (`ETAT.md:9-17` :
« restent l'événement, les dés et le videau »).

Portage en Go pur : faisable, coût faible.
- Couches à écrire :
  - conv 3×3 avec padding 1, BN repliée dans la conv à l'export ;
  - ReLU, max-pool 2, dense ;
  - log_softmax, sigmoïde, `cummin`.
  Le Dropout disparaît à l'inférence. Il n'y a ni attention ni normalisation exotique.
- Calcul, à vérifier par un banc :
  - environ 101 M multiplications par vignette, 24 vignettes, plus environ 0,2 G pour la barre,
    soit environ 2,6 G par photo ;
  - en Go scalaire réparti sur les cœurs, l'estimation est sous la seconde. C'est acceptable pour
    une photo, et le serveur borne déjà les lectures concurrentes (`captureSlots`,
    `internal/server/handlers_positions.go:404-424`).
- Précédents à reprendre :
  - le décodage float16 du moteur (`network.go:282`, `halfToFloat32`) ;
  - la parité contre PyTorch à 1e-4 sur des vecteurs d'or committés (ADR-0110 règle 2,
    `docs/adr/0110-…:42-51`). Ce n'est pas l'identité au bit d'ADR-0106, réservée à `gammonnet`.
- Pas d'ONNX ni de CGO. `cmd/serve` reste pur Go, et le frontend refuse tout runtime d'inférence
  (`frontend/src/__tests__/vision.noInferenceDependency.test.js:5`).
- Les noyaux AVX2 du moteur sont de l'assembleur Go (sans CGO). Ils ne servent qu'en dernier
  recours.
- Coût du portage :
  - export (repli BN, `.bin16`, vecteurs d'or) : 0,5 jour ;
  - couches, décodage et tests de parité : 1,5 à 2 jours ;
  - rectification en Go (homographies, échantillonnage par aire, orientation EXIF) : 1 jour.

## 2. Écart de domaine photo / vidéo

Écarts constatés :
- (a) Perspective oblique. L'homographie redresse le plan du plateau, mais pas le flanc ni
  l'épaisseur des pions : ils débordent sur la pointe voisine ou vers le photographe. Aucune
  augmentation ne couvre ce cas.
- (b) Résolution. Une table de 3 000 px doit descendre à 300 px. Il faut un ré-échantillonnage par
  aire (ou une pyramide) avant le bilinéaire de `geometrie.py:39`, sinon on crée du repliement.
- (c) Orientation EXIF. Go l'ignore (`pkg/blunderdb/capture/capture.go:106-118` : `image.Decode`,
  aucune lecture EXIF), alors que la webview l'applique. Sans correction, les coins cliqués ne
  tombent pas sur les pixels lus.
- (d) Formats. HEIC (défaut de l'iPhone) n'a aucun décodeur Go pur dans la bibliothèque standard.
  Seuls PNG, JPEG et GIF passent (`capture.go:8-10`). Plafond : 50 Mpx (`capture.go:88`), suffisant
  hors capteurs de 200 Mpx.
- (e) Matériels et éclairages. L'entraînement couvre Sèvres/FFBG, EZE gris et HSBT, avec des pions
  rouges, bleus, blancs, jaunes, noirs et marine (`C:t1/JOURNAL.md:131-163`). La teinte est
  augmentée (`reseau.py:32`), mais pas les reflets de flash ni les ombres portées.
- (f) Pas de recalage ni de vote sur plusieurs images. Une photo est un seul essai.
- (g) Barre : 1,2 pointe dans le canonique. Avec 4 coins seulement, il faudrait un modèle de largeur
  de barre (`frontend/src/vision/geometry/boardModel.js` pose 1 pointe), et l'erreur au bord de la
  barre peut dépasser le décalage augmenté (±4 px).

Ce qu'il faut, sans réentraînement : 8 clics (ou 4, si la mesure montre qu'ils suffisent),
rectification par table au canonique de N1, réduction par aire, EXIF appliqué, vignettes
identiques à `bandes.py`.

Réentraînement seulement si la tranche 0 échoue :
- ajuster N1 sur des vignettes de photos étiquetées par leur XGID ;
- ajouter une augmentation de perspective (homographie perturbée avant découpe) ;
- l'entraînement reste hors du binaire (ADR-0108 règle 6, `docs/adr/0108-…:51-56`).

Mesure, à figer avant d'ouvrir une seule photo de test :
- Corpus :
  - photos de l'utilisateur, au téléphone, de positions montées depuis une liste tirée de sa base
    (contact, course, sortie, pions à la barre, piles ≥ 6) ;
  - vérité = XGID de la position montée, consigné avant la prise de vue (nom du fichier =
    identifiant) ;
  - 8 coins cliqués par photo, enregistrés une fois. Les 4 coins extérieurs s'en déduisent : la
    variante à 4 clics se mesure sur les mêmes photos.
- Répartition :
  - développement : 30 photos, réglages permis ;
  - test : 60 photos jamais ouvertes ;
  - au moins 3 matériels, 2 téléphones, lumière du jour et lumière artificielle ;
  - deux strates : « cadrage conseillé » (téléphone au-dessus, moins de 30° de la verticale, coins
    visibles) et « oblique » (30 à 60°).
- Métrique d'une photo : E = nombre d'emplacements faux (24 pointes + 2 barres, couleur et
  nombre), c'est-à-dire le nombre de corrections en EDIT.
- Critère d'acceptation, sur la strate « cadrage conseillé » du test, proposé d'après la référence
  vidéo ci-dessus :
  - (i) au moins 70 % des photos avec E ≤ 1 ;
  - (ii) au plus 10 % avec E ≥ 4 ;
  - (iii) E moyen ≤ 1,0.
- Pour information seulement :
  - la strate oblique ;
  - la part des erreurs signalées en doute ;
  - les doutes par photo ;
  - le temps de lecture.
- Un échec sur l'oblique ne bloque rien : la doc conseille alors le cadrage d'en haut.

## 3. Parcours dans blunderDB

L'existant à réutiliser :
- `capture.Reading` (`capture.go:39-50`) : position, `Doubts` (champ `point`/`bar`/`dice`/`cube`/
  `turn`/`orientation`/`checkers`) et `Unread`.
- `Database.ReadCaptureFile` et `ReadCaptureData` (`pkg/blunderdb/database/db_capture.go:14-30`).
  Elles sont pures : rien n'est stocké, et la sauvegarde passe par `SaveIndividualPosition`
  (`db_position.go:30`).
- CLI `capture` (`internal/cli/cli.go:133`, `cli_capture.go:21-28`), avec `--score`, `--turn`,
  `--save` et `--force`.
- Serveur `/v1/positions.fromImage` (`handlers_positions.go:279`, `405-431`).
- Parité déjà déclarée (`internal/cli/parity_test.go:509-510`).
- Frontend : `showCaptureOnBoard` passe en EDIT, applique la position et entoure les doutes
  (`frontend/src/services/captureService.js:37-55`). Le dépôt de fichier et le collage y mènent
  déjà (`importService.js:529`, `1447` ; `captureService.js:108-115`).
- 4 clics avec loupe et pions fantômes : `frontend/src/components/VideoCalibration.svelte:1-6`
  (ordre 1-12-13-24 du joueur du bas, ligne 24) et `frontend/src/vision/geometry/`
  (`homography.js`, `calibration.js`, `rectify.js`). Aujourd'hui, ce n'est monté que dans
  `VideoPane.svelte:639`, derrière l'interrupteur.

Parcours proposé :
1. L'utilisateur dépose ou colle une image. `capture` est essayée d'abord. Si aucun thème n'est
   trouvé (`MinFit`, `capture.go:68`), un panneau « photo » s'ouvre sur l'image au lieu du message
   d'échec.
2. Il clique les coins dans l'ordre 1, 12, 13, 24 du joueur du bas (8 coins en v1, voir la
   décision D3). Ce sens fixe `h`. Les pions fantômes confirment l'ajustement.
3. Il choisit d'un clic la couleur du joueur du bas, foncée ou claire. Cela donne `c1`.
4. Go lit la photo et rend un `capture.Reading`. La barre est lue, les pions sortis sont le reste
   jusqu'à 15. Les pointes à faible marge deviennent des `Doubt{point}`. Dés, videau, trait et
   score vont dans `Unread`.
5. `showCaptureOnBoard`, sans changement. L'utilisateur corrige et sauve par le geste habituel
   (`SaveIndividualPosition`).

Point d'entrée, avec parité selon CLAUDE.md :
- `Database.ReadPhotoData(b64, corners [8][2]float64, bottomDark bool) (capture.Reading, error)`,
  plus `ReadPhotoFile`. La logique vit dans un paquet neuf `pkg/blunderdb/photo`, avec un
  sous-paquet `cnn`. `capture` reste « sans modèle appris » (`capture/doc.go:1-3`).
- CLI : `capture --photo --corners "x,y;…" --bottom dark|light <image>`, avec les mêmes `--score`,
  `--turn`, `--save` et `--force`. Le refus de sauver avec des doutes s'applique tel quel.
- Serveur : `/v1/positions.fromPhoto`, sous le même sémaphore `captureSlots`, avec la borne
  `DecodeBounded`.
- Rien n'est écrit chez le lecteur (ADR-0007). Les coins ne sont pas stockés en base.

À trancher par un ADR neuf (0115). ADR-0108 règle 3, « Discriminer, jamais reconnaître »
(`docs/adr/0108-…:40-43`), et règle 2, « Go ne voit aucune image » (ligne 31), visent l'assistant
vidéo. La photo est une reconnaissance absolue, comme `capture`, rendue acceptable par la
correction obligatoire en EDIT et les doutes visibles. L'ADR doit dire pourquoi : une image unique
traverse le pont une fois (le précédent est `ReadCaptureData`), et la parité CLI/serveur exige Go.

## 4. Interrupteur de dev et publication

L'interrupteur est `VISION_AVAILABLE = import.meta.env.DEV` (`frontend/src/vision/available.js:3`),
frontend seulement. Une méthode `Database` exportée est liée par Wails et contrôlée par le test de
parité, sans interrupteur côté Go.
- Restent derrière l'interrupteur : l'assistant vidéo (#870, #858, #846), les dés et le videau en
  photo, et la strate oblique tant qu'elle n'est pas mesurée.
- Publiable après la tranche 3 seulement : la lecture des pions avec 8 (ou 4) clics, dans le
  parcours ci-dessus, avec la doc (`manuel.rst`, `cmd_mode.rst`, `cli.rst`, `CLI_USAGE.md`, huit
  `.po`).
- Pendant les tranches 1-2 : paquet Go et tests seulement, sans méthode `Database`. Rien n'est
  visible et la parité n'est pas touchée.

## 5. Tranches

| # | Tranche | Test | Coût |
|---|---|---|---|
| 0 | Hors dépôt (`~/bkgm/bancs/position-photo/`) : outil de clic des 8 coins ; N1 tel quel sur les photos de développement, rectification et réduction par aire en Python ; puis la mesure de test figée (§2) | critère (i)-(iii), un seul passage | 1 j + photos |
| 0b | Seulement si 0 échoue : réentraînement (vignettes de photos et augmentation de perspective), nouvelle mesure sur un **nouveau** jeu de test | même critère | 3-5 j |
| 1 | `pkg/blunderdb/photo/cnn` : export `.bin16` (BN repliée), couches Go, `logp_case` | parité à 1e-4 contre PyTorch sur vecteurs d'or committés ; banc du temps par photo | 2 j |
| 2 | `pkg/blunderdb/photo` : EXIF, homographies par table, vignettes, barre, doutes, sortie `capture.Reading` | parité des vignettes avec `bandes.py` sur images d'or ; `photo_measure_test.go` sur le corpus (passé si la variable d'environnement est absente) qui redonne E de la tranche 0 | 2 j |
| 3 | `Database.ReadPhoto*`, `capture --photo`, `/v1/positions.fromPhoto`, entrée de parité, doc FR et `.po`, `cli-doc-gen` | tests CLI, serveur et parité, `scripts/doc-inventory.sh` | 1,5 j |
| 4 | Panneau photo (repli de `capture`, clics réutilisant `VideoCalibration` sur `<img>`, choix de couleur) | vitest qui monte le composant ; e2e Playwright dépôt → clics → EDIT avec doutes (mock Wails, comme `capture-paste.spec.js`) | 2-3 j |
| 5 | Plus tard, derrière l'interrupteur : dés par clic sur chaque dé (lecteur d'ADR-0110), videau | mesure propre, critère fixé avant | à chiffrer |

Total des tranches 1 à 4 : 8 à 10 jours d'agent. Les tranches 1 et 2 se lancent en parallèle (Opus :
arithmétique silencieuse) ; les tranches 3 et 4 en Sonnet.

## Décisions de l'utilisateur

- D1. Lancer la tranche 0 avant tout code ? Fournir 90 photos (30 de développement, 60 de test)
  avec leur XGID et cliquer les 8 coins, environ 2-3 h. Préciser quels matériels et quels
  téléphones.
- D2. Fixer ou modifier les critères (i)-(iii) et les deux strates **avant** la mesure.
- D3. 8 clics (fidèles à N1) ou 4 (plus simples, si la mesure le permet).
- D4. Accepter un ADR 0115 qui fait de la photo une reconnaissance absolue corrigée, en écart à
  ADR-0108 règles 2 et 3.
- D5. HEIC : refus avec un message (« exporter en JPEG ») ou un décodeur tiers en Go pur (dépendance
  nouvelle).
- D6. Dés et videau en v1 : saisis à la main (recommandé) ou tranche 5 derrière l'interrupteur.
- D7. Un réentraînement (0b) emploie les photos de l'utilisateur et les vidéos de la campagne pour
  des poids publiés (ADR-0108 règle 7, corpus manifeste) : accord à donner.
