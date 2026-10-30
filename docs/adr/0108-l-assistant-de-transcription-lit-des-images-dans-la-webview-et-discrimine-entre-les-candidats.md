# ADR-0108 — L'assistant de transcription lit des images d'un fichier vidéo dans la webview et discrimine entre les candidats du moteur

Statut : acceptée (décision de l'utilisateur).
Voir aussi : ADR-0044 (transcrire n'est pas jouer), ADR-0082 (la vidéo horodate les décisions),
ADR-0092 (aucune frappe n'est avalée), ADR-0004 (capacités détectées), ADR-0007 (rien chez le
destinataire), ADR-0057 (l'API de transcription). Les phases citent cet ADR : ADR-0109
(événements), ADR-0110 (dés), ADR-0111 (coup), à venir.

## Contexte

Transcrire un match filmé, c'est guetter la vidéo pour taper un jet et choisir un coup parmi
ceux que le moteur propose. Les images contiennent ces faits ; un assistant peut les lire pour
naviguer, pré-remplir et valider plus vite. Mais une erreur de lecture validée sans être vue
corrompt la base en silence, alors qu'une pause ne coûte qu'une touche. La reconnaissance
absolue d'un plateau s'effondre hors de son domaine (à 99 % par case, 2,7 % de plateaux justes),
alors que choisir entre quelques candidats tient. Le binaire est unique, sans CGO de vision ; le
pont Wails ne porte pas des mégaoctets d'images. Les conditions de YouTube interdisent d'en
analyser les images (restrictions 1 et 3 des CGU : accès par un autre moyen que le service et
reproduction ; Developer Policies III.D.7 : les storyboards ; III.E.4.h : toute « derived
data » tirée du contenu audiovisuel) et de pousser vers un outil de téléchargement (III.E.1).

## Décision

1. **L'assistant ne lit que les images d'un fichier vidéo local.** Une source YouTube se lit
   dans son lecteur intégré (ADR-0082 règle 5) et aucun pixel n'en sort : ni capture
   d'écran, ni enregistrement de l'écran, ni `getDisplayMedia`, sur aucune plateforme. Aucun
   outil de téléchargement n'est nommé ni suggéré dans le code, la doc ou l'interface. Le volet
   vidéo en mode YouTube et la doc lient les CGU YouTube (Developer Policies III.A.1) ;
   demander l'assistant sur une vidéo YouTube dit pourquoi il ne marche pas et propose
   d'ouvrir un fichier local.
2. **Tout le pixel vit dans la webview ; Go ne voit aucune image.** Go garde les règles, les
   candidats (`TranscriptionCandidates`) et la machine de transcription (ADR-0057) ; il ne
   décode ni ne reçoit rien. Le code de vision vit dans `frontend/src/vision/`, sans import
   Svelte, testé en vitest ; le calcul tourne dans un Web Worker qui reçoit des tampons
   transférés. Le panneau n'en reçoit que des événements : un instant (`metadata.mediaTime`
   de `requestVideoFrameCallback`), un état, une confiance (règle 5). `requestVideoFrameCallback`,
   WebCodecs, Wasm SIMD et `OffscreenCanvas` se détectent, jamais par version (ADR-0004) ; pas
   de mémoire partagée ; tout chemin accéléré a un repli en JS. L'inférence est du code écrit à
   la main, sans dépendance d'inférence.
3. **Discriminer, jamais reconnaître.** La vision choisit entre les candidats que le moteur
   fournit — le `board_after` des coups légaux, les six faces d'un dé — et ne lit jamais une
   position absolue. Si aucun candidat ne correspond, l'état est incertain, jamais « le légal
   le plus proche ».
4. **L'équité n'est jamais un a priori.** Le classement visuel ignore l'ordre du moteur et
   l'équité : un prior les ferait gagner sur l'image et blanchirait les blunders, que la
   transcription doit consigner (prolonge ADR-0044 règle 1). Le top-1 visuel ne doit pas
   s'effondrer sur les coups qui ne sont pas le premier du moteur.
5. **Trois états, jamais un pourcentage : sûr, à vérifier, incertain.** La précision passe
   avant le rappel : dans le doute l'assistant s'abstient. Une frappe de l'utilisateur prime
   toujours sur ce que l'assistant propose (ADR-0092 règle 1).
6. **Rien n'est appris ni enregistré chez l'utilisateur.** Aucun ré-entraînement sur le poste.
   Calibration, journal de descripteurs, prototypes par caméra sont des commodités du poste
   (`localStorage`, IndexedDB) : hors base, hors `issuance.CarriedMetadataKeys`, jamais
   exportés (ADR-0007). Les Repères de la base (`roll_tick_ms`, `tick_ms`) sont des fenêtres
   où chercher un événement, jamais l'étiquette d'une image : un jet se tape après que les dés
   se sont posés et ont été lus.
7. **Les modèles s'apprennent hors du binaire, sur un corpus de sources locales licites, et
   sont livrés en poids.** Le corpus est un manifeste construit chez qui détient les fichiers ;
   une source YouTube n'y entre jamais, même sans distribution. Les poids sont embarqués dans
   `frontend/dist` et lus par `fetch`. Format, chaîne d'amorçage et parité : ADR-0110.

## Conséquences

- `internal/gui/media.go` pose `Access-Control-Allow-Origin` sur toutes ses réponses de
  `/media/` — 200, 206 et `OPTIONS` — sans redirection, et le `<video>` de `VideoPane.svelte`
  porte `crossorigin="anonymous"` posé avant `src` : un seul 206 sans en-tête teinte le canvas,
  et l'erreur ne paraît qu'à `getImageData`. La page `/yt/` n'en reçoit pas.
- La doc reformule ses mentions d'un outil de téléchargement : `doc/source/manuel_matchs.rst`
  (et ses huit `.po`) et `pkg/blunderdb/videomux/doc.go` décrivent le format du fichier
  (« des WebM dont les clusters ne commencent pas sur une image clé »), sans nom d'outil. Le
  lien vers `https://www.youtube.com/t/terms` entre au manuel et au volet YouTube.
- Une Transcription sans fichier local ne change pas : l'assistant est absent, pas dégradé.
- Rejeté : lire les pixels de YouTube par capture d'écran ou de fenêtre (CGU, III.E.4.h) ;
  décoder en Go ou par ffmpeg (CGO, codecs à embarquer, images à travers le pont) ; le serveur
  d'assets de Wails comme source (ADR-0082 règle 5) ; ONNX Runtime Web (13 à 27 Mo de Wasm, plus
  de build sans threads), TensorFlow.js (en sommeil), OpenCV.js ; la vision dans `gammonnet`
  (un MLP dont l'identité au bit, ADR-0106, interdit d'y loger autre chose) ; apprendre sur le
  poste ; l'équité ou l'ordre du moteur comme prior ; la reconnaissance absolue d'une position
  ou son recalage sur le légal le plus proche ; un score en pourcentage ; un Repère tapé pris
  pour l'instant exact de l'événement.

## Garde

À écrire avec le premier code de la phase 1 :
`frontend/src/__tests__/vision.noInferenceDependency.test.js` (aucune de `onnxruntime-web`,
`@tensorflow/*`, `opencv*` dans `frontend/package.json`) ;
`frontend/src/__tests__/vision.noSvelteImport.test.js` (aucun fichier de `frontend/src/vision/`
n'importe `svelte` ni un `.svelte`) ; une étape `grep -rniE 'yt-dlp|youtube-dl'` sur `doc/`,
`frontend/src` et `pkg/` dans `build.yml`, qui échoue sur toute occurrence ;
`TestMediaHostCORS` dans `internal/gui/` (l'en-tête sur 200, 206 et `OPTIONS`).
