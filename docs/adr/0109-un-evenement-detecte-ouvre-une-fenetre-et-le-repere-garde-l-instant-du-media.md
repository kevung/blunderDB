# ADR-0109 — Un événement détecté ouvre une fenêtre, et le Repère garde l'instant du média

Statut : proposée ; acceptation après le banc V0.1 (Worker depuis `wails://` sous WebKitGTK) —
un rouge arrête le chantier. Le banc V0.2 (couverture des fenêtres en SQL,
`tasks/transcription-vision/bancs/02-couverture-sql.md`) est rendu : la règle 1 en tient compte.
Voir aussi : ADR-0108 (le cadre de l'assistant : fichier local, pixel dans un Worker, trois
états, commodités du poste), ADR-0082 (le Repère et sa pose), ADR-0092 (aucune frappe n'est
avalée), ADR-0044 (transcrire n'est pas jouer), ADR-0111 (la lecture du coup, à venir).

## Contexte

Sur un match filmé, l'essentiel du temps de transcription passe à chercher dans la vidéo : le
jet suivant, le moment où les dés se posent. Les images portent ces événements, et des
descripteurs simples suivis d'automates à hystérésis les trouvent sans modèle appris. Mais
aucun moteur ne décode une heure à 10 images/s en quelques minutes : par `seek`, 10 à 19 ms
par image au mieux ; par `playbackRate`, un plafond d'environ 60 rappels/s, instable sous
GStreamer. Et le Repère du jet n'est pas l'instant de l'événement : la première touche de dé
(ADR-0082 règle 3) tombe après le repos, la lecture et la frappe, alors que la colonne dit
« l'instant où les dés tombent » (ADR-0082 règle 1).

## Décision

1. **Pas de pré-passe rapide : l'analyse suit la lecture, et elle est hiérarchique.** Le
   Worker reçoit chaque image composée pendant la lecture, à la vitesse que le décodeur tient
   (1× sûrement ; au-delà selon le banc V0.1). Il ne fait pas tourner les automates partout : à
   1–2 img/s il compare des **états** — dés déplacés, plateau changé — et n'ouvre une fenêtre
   de [−1 ; +2] s, où les automates tournent à 10 img/s, qu'autour d'un changement. Sur trois
   matchs horodatés, ces fenêtres couvrent un tiers de la vidéo (banc V0.2) : la hiérarchie
   divise par trois le travail dense, et c'est elle qui rend la lecture accélérée analysable.
   La même hiérarchie, par seeks séquentiels (un seul en vol), sert à parcourir une plage sans
   la regarder. La largeur de la fenêtre est un choix d'ingénierie : [−1 ; +2] s est la valeur
   de départ, à régler sur l'histogramme du banc V0.5 (à ±5 s la couverture passe à deux
   tiers et la hiérarchie ne vaut plus la peine).
2. **Deux résolutions.** Les événements se calculent sur une image réduite à 320×180,
   transférée au Worker. La rectification du plateau en pleine résolution, nécessaire pour
   lire un coup, relève de l'ADR-0111.
3. **Une calibration à quatre clics, et un plateau présent ou perdu.** Le premier clic est le
   coin du 1-point du joueur du bas ; une loupe guide chaque clic, des pions fantômes de la
   position initiale vérifient l'orientation, et une vidéo peut porter plusieurs calibrations
   nommées (« plein écran », « incrustation »), essayées après chaque coupe ; le nom d'une
   calibration est la « caméra » que les ADR-0110 et ADR-0111 retrouvent d'une vidéo à l'autre.
   L'homographie est
   une DLT à quatre points normalisée à la Hartley ; un quadrilatère non convexe est refusé.
   Le modèle du plateau s'exprime en diamètres de pion, jamais en dimensions absolues. Clics,
   homographie et imagettes vivent en `localStorage` par `draftId`, comme la rotation de la
   vidéo (ADR-0108 règle 6). Un score de présence du plateau calibré, à environ 2 Hz, bascule
   avec hystérésis (perdu sous 0,5, retrouvé au-dessus de 0,7) : « vue perdue » s'affiche, et
   l'analyse se tait jusqu'au retour.
4. **Trois automates couplés — mains, dés, partie — sur des descripteurs census.** Chaque ROI
   est normalisée par sa moyenne et comparée en census, invariant à l'auto-exposition ; une ROI
   témoin hors plateau qui change à plus de 60 % rend l'image incertaine et réinitialise les
   fonds. Les transitions ont une hystérésis (E_hi = 3 × la médiane calme, E_lo = 1,5 ×) et des
   durées minimales à 10 img/s (lancer ≥ 1 image, repos 5 images sous E_lo, mains dans 2, mains
   hors 3) ; le jet d'ouverture, un dé par moitié, est un cas de l'automate des dés. Ces
   valeurs sont des choix d'ingénierie, réglés sur le banc (règle 8). Un automate n'émet que
   des événements `{ type, mediaTime, état }` : `mediaTime` est celui de
   `requestVideoFrameCallback` pour la première image de l'épisode — la première image calme
   pour un repos, pas celle qui le confirme —, l'état l'un des trois de l'ADR-0108 règle 5.
   Les automates sont une machine pure, hors de `transcriptionKeys.js`.
5. **La détection ne pose aucun Repère ; elle cale celui que l'utilisateur pose.** Quand la
   première touche de dé d'une Action nouvelle pose l'instant du jet (ADR-0082 règle 3) et que
   l'instant courant tombe dans la fenêtre d'un repos des dés détecté **sûr** — de son
   `mediaTime` au premier ramassage ou lancer suivant, au plus 60 s —, le Repère prend le
   `mediaTime` de ce repos (le plus récent) au lieu de l'instant courant. Sinon, dont un repos
   à vérifier ou incertain, une source YouTube ou une vue perdue : l'instant courant, comme
   aujourd'hui. L'instant de l'action, une validation implicite et le geste explicite `v` /
   `MAJ-V` ne changent pas : celui qui désigne un instant l'obtient. La colonne garde son sens,
   l'instant où les dés tombent, mieux mesuré ; le schéma ne change pas, et le Replay (ADR-0082
   règle 2) non plus. Un Repère calé n'est pas distingué en base : c'est le même instant du
   média, pas une autre donnée, et rien ne le lirait autrement. L'indicateur du panneau dit
   l'instant retenu (ADR-0092 règle 3) ; `Ctrl+Z` défait le geste entier.
6. **Une ligne de temps à marqueurs sous la vidéo.** Chaque événement y est un marqueur ;
   `g` saute au suivant et `MAJ-G` au précédent, la lecture ou la pause étant gardée, le clic
   sur un marqueur fait de même. Un événement à vérifier ou incertain a une forme distincte,
   pas seulement une couleur, et reste un arrêt du saut. Aucun marqueur n'écrit rien : sauter
   sur un repos puis `MAJ-V` est la voie explicite pour poser son instant.
7. **L'overlay du stream sert au contrôle de dérive, jamais au score.** Sur une zone tracée
   une fois, toutes les 0,5–1 s, une imagette 32×16 en gris donne un dHash de 64 bits ; un
   changement au-delà d'une distance de Hamming, tenu plus d'une seconde, est confronté à la
   machine de transcription, avec une tolérance de quelques secondes de retard de l'overlay.
   Attendu (fin de partie, passe), il ne dit rien ; inattendu, un indicateur discret dit
   « l'overlay a changé sans action ». Il ne lit ni n'écrit jamais le score : le transcripteur
   rapporte les faits (ADR-0044 règle 1). Sans overlay, rien ne change.
8. **Un journal local de descripteurs rejoue les automates.** Le Worker consigne par image
   ses descripteurs et `mediaTime` (quelques Ko/min), hors base, jamais exporté, effaçable
   depuis le volet (ADR-0108 règle 6). Rejoué en vitest, il nourrit les automates sans vidéo :
   les seuils se règlent sur des journaux, pas sur des vidéos.

## Conséquences

- Mesures de la phase : le F1 des événements à ±0,5 s, appariés un à un, contre les Repères
  existants pris comme fenêtres (le repos précède le Repère du jet) puis contre deux matchs
  annotés à l'image ; et le gain de temps de la seule navigation, mesuré à part. La borne de
  60 s de la règle 5 se règle sur l'histogramme `t_lisible − roll_tick_ms` de la phase 0.
- Un match transcrit avec l'assistant porte des durées de décision de videau plus courtes et
  de décision de pions plus longues qu'avant, d'autant que la frappe était lente : la mesure
  devient plus juste, le filtre et l'affichage (ADR-0073) ne changent pas.
- `g` et `MAJ-G` entrent dans `shortcutMap.js` avec les touches vidéo du panneau, et dans
  `doc/source/raccourcis.rst` avec ses huit `.po`.
- À l'acceptation, l'ADR-0082 règle 3 est réécrite en place ; sa phrase « Sur une Action
  *nouvelle*, la première touche de dé pose l'instant du jet et la validation pose l'instant de
  l'action, à l'instant courant de la vidéo » devient « Sur une Action *nouvelle*, la première
  touche de dé pose l'instant du jet — celui où les dés se sont posés si un repos détecté sûr
  couvre l'instant courant (ADR-0109 règle 5), l'instant courant sinon — et la validation pose
  l'instant de l'action, à l'instant courant de la vidéo ».
- Rejeté : une pré-passe accélérée (aucun moteur ne tient 10 img/s sur une heure en 5 min) ;
  un Repère écrit par la détection seule (la colonne resterait un fait rapporté sans que
  personne l'ait rapporté) ; une colonne « origine du Repère » (un changement de schéma pour un
  instant de même nature) ; le flux optique dense et un modèle appris pour les événements
  (coût, pas de JS pur, des centaines de séquences annotées) ; une validation par délai (un
  compte à rebours s'habitue en quelques expositions ; ADR-0092 règle 7) ; caler l'instant de
  l'action sur « mains hors » avant que le F1 l'ait mesuré.

## Garde

À écrire avec le code de la phase : `frontend/src/__tests__/vision.automata.replay.test.js`
(journaux rejoués sous `frontend/src/__tests__/fixtures/vision/`, F1 et décalage médian
attendus), `vision.overlayDrift.replay.test.js`, `vision.homography.test.js` (DLT normalisée,
refus du non convexe), `vision.timeline.test.js` (sauts, aucun geste émis),
`TranscriptionPanel.rollSnap.test.js` (calage sur un repos sûr, instant courant sur un repos à
vérifier, hors fenêtre et pour `MAJ-V`) ; `vision.noSvelteImport.test.js` de l'ADR-0108 couvre
`frontend/src/vision/`.
