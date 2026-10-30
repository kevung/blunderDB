# ADR-0110 — Les dés se lisent par un réseau minuscule exécuté en JS, appris hors du binaire sur un corpus-manifeste

Statut : proposée ; acceptation après l'ADR-0109 livrée et le banc V0.8 (histogramme
t_lisible − roll_tick_ms sur 50 coups).
Voir aussi : ADR-0108 (le cadre), ADR-0109 (événements, calibration, images calmes), ADR-0104
(poids `.bin16` du moteur), ADR-0007 (rien chez le destinataire), ADR-0092 (aucune frappe
n'est avalée), ADR-0044 (transcrire n'est pas jouer). L'acceptation déléguée : ADR-0111.

## Contexte

La phase 2 de l'assistant lit les dés et pré-remplit le jet ; le coup reste à l'utilisateur.
ADR-0109 fournit l'événement « dés au repos » (avec son `mediaTime`), la calibration du
plateau et les images calmes de la fenêtre. Six faces sur un dé est la plus petite
discrimination du chantier : un réseau convolutif de quelques dizaines de milliers de
paramètres y suffit, quelques millions de multiplications par imagette, 2 à 6 ms en JS
(estimation, non mesurée). Aucun jeu public de dés sur plateau n'existe et aucune précision
publiée ne porte sur du backgammon filmé : le corpus est à construire.

Le corpus ne peut naître que chez qui détient les vidéos. L'export d'une bibliothèque ne garde
d'une source locale que le nom du fichier (`issuance.CarriedVideoSource`,
`pkg/blunderdb/issuance/issuance.go:97-138`, appelé par `pkg/blunderdb/ingest/export_sqlite.go:1264` ;
ADR-0082 règle 4) ; seule une URL voyage entière, et une URL est YouTube, exclue par ADR-0108
règle 1. Une base reçue n'apporte donc que des Repères sans image ; ses sources ne redeviennent
lisibles que relocalisées sur une copie que le destinataire détient lui-même.

Le moteur embarque déjà des poids float16 (ADR-0104), mais dans BGN6, le format amont de
gammonNet : un en-tête de perceptron (couches cachées, activation, sortie `prob5`) que le
chargeur Go refuse de compléter par défaut (`engine/gammonnet/network.go`).

## Décision

1. **Le modèle est un CNN de moins de 100 k paramètres, six faces plus « illisible ».** Une
   seule tête lit chacun des deux dés, sur une imagette par dé découpée dans le plan rectifié
   par la calibration d'ADR-0109. Choix d'ingénierie, à revoir sur le banc : 32×32×3 en
   entrée, trois blocs conv 3×3 + ReLU + max-pool (16, 32, 64 canaux), moyenne globale, dense à
   7 sorties, ≈ 24 k paramètres et ≈ 2,8 M de multiplications. Un jet se lit par vote sur 5 à
   10 images calmes de la fenêtre « dés au repos » (somme des log-probabilités par dé). La
   sortie est l'un des trois états d'ADR-0108 règle 5, par deux seuils de marge réglés sur la
   précision : « sûr » exige les deux dés au-dessus du seuil haut ; une classe « illisible »
   majoritaire ou un vote partagé donne « incertain » ; le reste est « à vérifier ». Les seuils
   voyagent avec les poids.
2. **L'inférence est du JS écrit à la main dans le Worker** (`frontend/src/vision/`) :
   convolution directe, ReLU, max-pool, dense, normalisation par lots fusionnée à l'export,
   décodage float16 → float32, sur des `Float32Array` préalloués ; même code sous V8 et JSC.
   La phase 2 ne livre aucun noyau Wasm : à cette taille le JS suffit. Un GEMM Wasm SIMD
   n'entre que si un banc le demande, alors par détection avec repli JS (ADR-0108 règle 2) ;
   JSC le désactive sans AVX. La parité se prouve contre PyTorch à 1e-4 sur les sorties, sur
   des vecteurs d'or committés ; ce n'est pas l'identité au bit du moteur (ADR-0106 vaut pour
   `gammonnet`, pas ici : JS calcule en float64 entre deux écritures float32). Les vecteurs
   d'or ont des entrées synthétiques (bruit à graine fixe, dés rendus procéduralement),
   jamais une imagette du corpus.
3. **Les poids sont un fichier safetensors float16 servi par `frontend/dist`, pas un `.bin16`.**
   `frontend/public/vision/dice-v1.safetensors` : en-tête JSON préfixé par sa longueur, tenseurs
   `F16` little-endian, et dans `__metadata__` l'architecture, la version du modèle et les
   seuils de la règle 1 ; lu par `fetch`, refusé entier sur tout tenseur, type ou architecture
   inattendu. On garde d'ADR-0104 la discipline (float16 au transport, calcul float32,
   règles 1-2), pas le conteneur : BGN6 est la spécification d'un autre projet, décrit un
   perceptron et ne sait pas dire une convolution ; l'étendre forkerait l'amont, et aucun code
   ne serait partagé (chargeur Go d'un côté, JS de l'autre). Safetensors s'écrit depuis
   PyTorch sans écrivain maison et se lit en une cinquantaine de lignes.
4. **Le corpus est un manifeste, exporté par `blunderdb transcribe --db <base> --marks
   <fichier.jsonl> [--match <id>]`.** Une option de la commande `transcribe` déjà dans
   `handlers()`, pas une commande nouvelle. Une ligne par coup qui porte un Repère : source,
   match, partie, numéro du coup, joueur, dés, coup, plateaux avant et après, `roll_tick_ms`,
   `tick_ms`, et les incohérences du rejeu (`--check`) qui le concernent. Seules entrent les
   sources qui sont des chemins locaux, nommées par le chemin complet de la base : une URL
   est exclue, une source introuvable sur le disque est signalée sur la sortie d'erreur et
   omise. La commande ne lit aucune image et n'en écrit aucune. Le manifeste n'est pas un
   export de bibliothèque : ni Watermark, ni import, il reste chez qui détient les fichiers et
   n'écrit rien dans la base (ADR-0007 règle 3).
5. **La logique est sur `Database`, exposée au CLI seul.** `Database.TranscriptionMarks`
   (contrat de stockage, les deux backends) rend les lignes ; le filtre « chemin local » est
   celui de `issuance.CarriedVideoSource`, une seule définition de ce qui est une URL. Le
   démon `serve`, donc `call`, ne l'expose pas : les chemins d'un tenant désignent les disques
   de ses clients, rien que le démon puisse vérifier, et le démon ne lit aucun fichier vidéo
   (ADR-0082 règle 5). Le bureau ne l'expose pas non plus : seul l'outillage d'amorçage
   (règle 6) la consomme. Les deux absences sont nommées dans `databaseParity`
   (`internal/cli/parity_test.go`) avec leur raison.
6. **L'amorçage vit hors du binaire, dans `tools/vision/`.** Python et ffmpeg y sont permis,
   et là seulement : rien n'entre dans `go build` ni dans `npm run build`. Chaîne : audit
   déterministe du manifeste (coup légal pour ses dés, plateau après = plateau avant + coup,
   après(n) = avant(n+1) ; un échec exclut le coup) ; fenêtres tirées des Repères (largeur
   fixée par le banc V0.8, ±2 s supposé) ; images calmes ; top-K par netteté (variance du
   laplacien) ; recadrage immédiat, jamais de plein cadre ; modèle v0 ; deux ou trois tours
   d'EM sur les étiquettes bruitées, puis la liste des suspects. Les imagettes se régénèrent
   depuis les fichiers et se purgent après l'entraînement. Sorties, seules committées : le
   fichier de poids de la règle 3 et les vecteurs d'or de la règle 2.
7. **Le mode assisté pré-remplit les dés dans le panneau, sans jamais valider un coup.**
   « Sûr » : les dés sont tapés comme si l'utilisateur l'avait fait, et l'indicateur d'ADR-0092
   règle 3 le dit (« dés lus : 63 ») ; ils ne se tapent que sur un jet ouvert, jamais comme le
   chiffre qui validerait le coup en attente (ADR-0092 règle 2) — cette validation-là est
   l'objet d'ADR-0111. « À vérifier » : les dés sont proposés, non tapés ; une touche les
   accepte. « Incertain » : pause sur l'image gelée, zoom sur les dés, l'utilisateur tape. Des
   dés pré-remplis deviennent ceux du coup quand l'utilisateur valide ce coup, par le jet
   suivant ou Entrée ; jamais par un délai. Une frappe de l'utilisateur sur le jet prime et
   efface la proposition (ADR-0108 règle 5), `Ctrl+Z` défait un pré-remplissage.
8. **L'amélioration continue est un k-NN de prototypes, sans apprentissage.** Les vecteurs de
   l'avant-dernière couche des lectures corrigées ou confirmées par l'utilisateur, rangés par
   caméra, sans image ; ce sont des commodités du poste (ADR-0108 règle 6). Une caméra est un
   nom que l'utilisateur donne à une calibration nommée d'ADR-0109 pour la retrouver d'une
   vidéo à l'autre ; une calibration sans ce nom est une caméra nouvelle. Un désaccord du k-NN avec la tête abaisse l'état d'un cran ; un accord
   ne l'élève qu'une fois la courbe risque–couverture de cette caméra à la cible. Cette courbe
   est aussi ce par quoi l'automatique se mérite par caméra (ADR-0111) ; en phase 2 seul le mode
   assisté existe.

## Conséquences

- Mesures, avant toute promesse dans la doc : précision par dé en leave-one-stream-out sur au
  moins trois sources ; étude intra-sujet alternée manuel/assisté par segments appariés de dix
  tours. Critère d'avance (choix d'ingénierie) : −30 % de temps médian par tour sans plus
  d'erreurs résiduelles que le manuel. Un critère manqué laisse l'assistant absent.
- Un nouveau modèle est un nouveau fichier de poids et de nouveaux vecteurs d'or, produits par
  `tools/vision/` ; aucune base ne change, aucun `DatabaseVersion`, aucun Repère réécrit.
- `CLI_USAGE.md` (`go run ./cmd/cli-doc-gen`) et `cli.rst` décrivent `--marks` ; le manuel
  décrit le mode assisté et ses trois états.
- Le manifeste porte des chemins complets : partagé, il dit l'arborescence de son auteur. C'est
  son geste, comme tout fichier qu'il produit.
- Rejeté : ONNX Runtime Web, TensorFlow.js, OpenCV.js (ADR-0108 ; taille, Wasm sans SIMD sous
  JSC, dépendance) ; le conteneur BGN6 pour la vision (règle 3) ; ré-entraîner sur le poste
  (ORT training déprécié, WebGPU absent de WebKitGTK ; ADR-0108 règle 6) ; des imagettes dans le
  manifeste, dans un export ou dans les vecteurs d'or ; un corpus central tiré de bases reçues ;
  un pourcentage de confiance ; une validation par délai ; l'équité comme prior (ADR-0108
  règle 4) ; apprendre sur des pixels YouTube, même d'une chaîne qui l'autorise ; exposer
  `--marks` par `serve`.

## Garde

À écrire avec le premier code de la phase 2 :
`frontend/src/__tests__/vision.diceParity.test.js` (vecteurs d'or de
`frontend/src/__tests__/fixtures/vision/`, écart maximal 1e-4 sur le chemin JS, et sur le chemin
Wasm s'il existe) ; `frontend/src/__tests__/vision.weights.test.js` (le chargeur refuse un type,
un tenseur ou une architecture inattendus) ;
`frontend/src/__tests__/TranscriptionPanel.assist.test.js` (« sûr » ne valide aucun coup en
attente ; une frappe efface la proposition ; des minuteurs avancés ne valident rien) ;
`TestTranscriptionMarks_LocalSourcesOnly` dans la suite de contrat
(`storage/storagetest/`) et `TestTranscribeMarks_ExcludesURLSources` dans `internal/cli/` ;
l'entrée `TranscriptionMarks` de `databaseParity`. L'absence d'import Svelte et de dépendance
d'inférence : gardes d'ADR-0108.
