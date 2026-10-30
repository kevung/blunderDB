# Assistant de transcription par la vidéo — plan des chantiers

Plan de découpage, aucun code. Il tire de `synthese.md` (« Conclusions transversales ») et des
ADR-0108 à 0111 la liste des issues à ouvrir, rangées par **zone de code** pour que des ouvriers
tournent en parallèle sans se marcher dessus (`/traiter-lot`, ADR-0055). Les issues ne sont
ouvertes sur GitHub qu'une fois ce découpage validé.

## Décisions qui gouvernent

| ADR | Objet | Statut |
|---|---|---|
| 0108 | Cadre : fichier local, pixel dans la webview, discrimination entre candidats, équité jamais a priori, rien d'appris chez l'utilisateur | acceptée |
| 0109 | Phase 1 : fenêtres par états, automates, calibration, ligne de temps, Repère = instant du média | proposée — bancs V0.1 et V0.2 |
| 0110 | Phase 2 : réseau de dés en JS, corpus-manifeste CLI, mode assisté | proposée — ADR-0109 livrée, banc V0.8 |
| 0111 | Phase 3 : discrimination du coup, acceptation déléguée, trace en schéma, videau | proposée — ADR-0110 livrée et mesurée |

Une phase n'ouvre pas tant que son ADR n'est pas acceptée ; accepter l'ADR est la première issue
de la phase (réécriture en place des règles qu'elle touche : ADR-0082 règle 3 pour 0109,
ADR-0044 règle 1 pour 0111).

## Zones de code

Une zone = un ensemble de fichiers qu'un seul ouvrier tient à la fois. Deux issues de zones
différentes se font en parallèle ; deux issues de la même zone vont au même ouvrier ou se
suivent.

| Zone | Fichiers | Modèle par défaut |
|---|---|---|
| **Z-vision** | `frontend/src/vision/` (nouveau ; JS pur, aucun import `.svelte`, tests vitest sur journaux) | Opus (seuils et géométrie : une erreur est silencieuse) |
| **Z-panneau** | `frontend/src/components/TranscriptionPanel.svelte`, `VideoPane.svelte`, `services/transcriptionKeys.js`, indicateur | Sonnet (tests rouges) |
| **Z-go** | `internal/cli/`, `pkg/blunderdb/database/`, `storage/` (requêtes, export, schéma) | Sonnet ; **Opus** dès que `DatabaseVersion` bouge |
| **Z-media** | `internal/gui/media.go`, `internal/gui/run.go` | Sonnet |
| **Z-tools** | `tools/vision/` (nouveau ; Python, ffmpeg autorisé ici, hors de `go build` et `npm run build`) | Sonnet |
| **Z-doc** | `doc/source/*.rst` + huit `.po`, `CLI_USAGE.md`, `README` | Sonnet |

Règle d'architecture qui rend le parallélisme possible : **Z-vision ne connaît pas Svelte, et
Z-panneau ne connaît pas les pixels**. Le contrat entre les deux est un flux d'événements
`{ type, mediaTime, état ∈ {sûr, à vérifier, incertain}, charge }` et une poignée de commandes
(démarrer, arrêter, calibrer). Ce contrat est fixé par la première issue de la phase 1 et ne
bouge plus qu'avec un ADR.

## Lot V0 — bancs (avant toute ligne de produit)

Chaque banc rend une note `tasks/transcription-vision/bancs/NN-<sujet>.md` : protocole,
mesures, verdict **vert / rouge**, ce qui changerait au plan. Aucun banc ne fusionne de code
dans le produit ; une page d'essai vit derrière un drapeau de développement et part avec la
branche.

| # | Banc | Zone | Dépend de | Bloquant | Modèle |
|---|---|---|---|---|---|
| V0.1 | **Worker depuis `wails://`** sous WebKitGTK (Linux), puis WebView2 et WKWebView si un poste existe ; dans le même banc : `requestVideoFrameCallback` + `mediaTime`, coût `drawImage` 1080p → 320×180 + `getImageData` par image, seeks et `playbackRate` (01.3, 01.4), détection WebCodecs par distribution (01.5). Un seul spike, une seule page. | Z-vision | — | **oui** | Opus |
| V0.2 | **Couverture des fenêtres en SQL** : sur `~/dwn/unger.db` (1061 coups horodatés, 3 matchs), part des coups dont `roll_tick_ms`/`tick_ms` tombent à moins de 3 s du coup précédent (analyse opportuniste suffisante) et histogramme des écarts. Requête seule, aucune écriture. | Z-go | — | **oui** | Sonnet |
| V0.3 | **CORS sur 200/206/OPTIONS** du serveur de bouclage et redirection : le canvas n'est pas « tainted » (`getImageData` ne lève pas). | Z-media | — | non (le correctif est connu) | Sonnet |
| V0.4 | **LNA Chromium ≥ 142 sous WebView2** : la lecture vidéo actuelle par bouclage déclenche-t-elle permission ou blocage ? Poste Windows requis (décidé : Linux seul disponible ; issue ouverte, en attente). Risque **hors vision** : touche le produit tel qu'il est. | Z-media | poste Windows | non pour le chantier, **oui pour le produit** | Sonnet |
| V0.5 | **Histogramme `t_lisible − roll_tick_ms`** sur 50 coups de deux vidéos : à la main (Kevin note l'instant où les dés sont lisibles), un script Z-tools trace l'écart. Fixe la fenêtre de l'ADR-0109 et dit si les Repères actuels valent comme fenêtres. | Z-tools | — | non (règle 0110) | Sonnet |

Un rouge sur V0.1 **arrête le chantier** ; le plan se réécrit alors (Go pur avec un
décodeur ? un outil externe ? renoncer ?) et c'est une décision de Kevin, pas d'un ouvrier.

V0.2 rendu (`bancs/02-couverture-sql.md`) : fenêtres [−1 ; +2] s = 32 % de la vidéo, 94 % des
jets à moins de 10 s de l'action précédente, Repères utilisables comme fenêtres (médiane 4 s,
26 % d'actions manquantes). Rouge sur « l'opportuniste seul suffit » : la hiérarchie par états
devient la façon normale d'analyser, pendant la lecture comme en parcours ; ADR-0109 règle 1
réécrite. Pas un arrêt : les Repères valent comme fenêtres et les événements sont denses.

Parallélisme : V0.1 ∥ V0.2 ∥ V0.3 ∥ V0.5 (quatre zones). V0.4 quand un poste Windows est là.

## Hors vision — à ouvrir maintenant, indépendamment du chantier

| # | Sujet | Zone | Modèle |
|---|---|---|---|
| H1 | **`JSC_*` sans AVX** : WebKitGTK 2.52 plante dès que du Wasm tourne sur un CPU sans AVX ; `internal/gui/run.go` ne pose aucune variable. Décider : poser `JSC_useWebAssembly=0` (ou équivalent) quand `/proc/cpuinfo` n'a pas `avx`, ou documenter. Aujourd'hui aucun Wasm ne tourne dans blunderDB : vérifier avant de corriger. | Z-media | Sonnet |
| H2 | **Conformité YouTube** : reformuler `doc/source/manuel_matchs.rst` (l. ~906) et `pkg/blunderdb/videomux/doc.go` (l. ~9) qui nomment un outil de téléchargement ; ajouter le lien vers les CGU YouTube (Developer Policies III.A.1) dans le manuel seulement (décidé : rien dans l'interface). Huit `.po`. | Z-doc | Sonnet |
| H3 | Dependabot #112 (deepmerge) : **traité sans issue** — tiré par le seul `@sveltejs/vite-plugin-svelte` (outillage de build), alerte close avec motif. | — | — |

H1 et H2 ne dépendent de rien et se font en parallèle du lot V0.

## Lot V1 — événements et ligne de temps (ADR-0109)

| # | Issue | Zone | Dépend de | Modèle |
|---|---|---|---|---|
| V1.0 | Accepter l'ADR-0109 : statut, réécriture en place de l'ADR-0082 règle 3, index. | docs | V0.1 et V0.2 verts | Sonnet |
| V1.1 | **Noyau Z-vision** : contrat d'événements, chargement du Worker (chemin trouvé par V0.1), échantillonneur rVFC à 320×180, descripteurs census + ROI témoin, **journal local** de descripteurs (IndexedDB, effaçable, jamais exporté) et son rejeu en vitest. | Z-vision | V1.0 | Opus |
| V1.2 | **Géométrie** : homographie DLT normalisée à 4 points, modèle du plateau en diamètres de pion, rectification d'une zone, score de présence avec hystérésis. Module pur, tests sur images de synthèse. | Z-vision (fichiers distincts de V1.1) | V1.0 | Opus |
| V1.3 | **Automates** mains / dés / partie et fenêtres par comparaison d'états (1–2 img/s → 10 img/s sur [−1 ; +2] s), chemin opportuniste à 1×. Seuils réglés sur les journaux de V1.1. | Z-vision | V1.1 | Opus |
| V1.4 | **Calibration à 4 clics** dans le volet vidéo : loupe, pions fantômes, calibrations nommées, localStorage par `draftId`, « vue perdue ». | Z-panneau | V1.2 | Sonnet |
| V1.5 | **Ligne de temps à marqueurs** sous la vidéo, saut au clavier, formes des incertains, indicateur ; **Repère = `mediaTime` de l'événement** quand la première touche de dé tombe dans sa fenêtre (ADR-0082 règle 3 réécrite). | Z-panneau + `transcript` si besoin | V1.3, V1.4 | Sonnet |
| V1.6 | **dHash de l'overlay** : contrôle de dérive, indicateur seulement. | Z-vision | V1.1 | Sonnet |
| V1.7 | **Mesure** : F1 à ±0,5 s contre les Repères pris comme fenêtres, gain de temps de navigation ; note dans `bancs/`. | Z-tools + Z-vision (lecture) | V1.5 | Sonnet |
| V1.8 | **Doc** : `manuel_matchs.rst`, `raccourcis.rst`, huit `.po` ; historique des versions. | Z-doc | V1.5 | Sonnet |

Parallélisme : V1.1 ∥ V1.2 (fichiers distincts, même zone — deux ouvriers acceptés parce que
le contrat est fixé par V1.0) ; puis V1.3 ∥ V1.4 ∥ V1.6 ; puis V1.5 ; puis V1.7 ∥ V1.8. Z-panneau
est la zone chaude du dépôt (le panneau est gros et bouge pour d'autres raisons) : **une seule
issue Z-panneau ouverte à la fois**, courte, fusionnée vite.

## Lot V2 — dés et mode assisté (ADR-0110)

| # | Issue | Zone | Dépend de | Modèle |
|---|---|---|---|---|
| V2.0 | Accepter l'ADR-0110 (banc V0.5 lu, V1 livré). | docs | V1.7 | Sonnet |
| V2.1 | **Export CLI des repères** (commande dans `handlers()`, sources locales seulement, aucune imagette), `CLI_USAGE.md`, `cli.rst`. **Indépendant de V1** : peut se faire pendant le lot V1. | Z-go | V0.2 (même requête) | Sonnet |
| V2.2 | **Runtime d'inférence JS** : conv, pool, dense, décodage float16, Wasm SIMD si détecté + repli ; test de parité à 1e-4 sur vecteurs d'or (réseau jouet committé). | Z-vision | V1.1 | Sonnet (le test est rouge) |
| V2.3 | **Chaîne d'amorçage** `tools/vision/` : manifeste → fenêtres → images calmes → top-K netteté → modèle v0 → EM ; export des poids et des vecteurs d'or au format tranché par 0110. | Z-tools | V2.1, format de V2.2 | Sonnet |
| V2.4 | **Modèle de dés v1** : apprentissage sur le corpus de Kevin, leave-one-stream-out, poids dans `frontend/dist`. Rend une mesure, pas du code produit. | Z-tools | V2.3 | Sonnet |
| V2.5 | **Mode assisté** : pré-remplissage, trois états, pause sur incertain avec image gelée et zoom, validation par le lancer suivant, la frappe prime. | Z-panneau | V2.2, V2.4, V1.5 | Sonnet |
| V2.6 | **Prototypes k-NN par caméra**, locaux, sans image, jamais exportés. | Z-vision | V2.2 | Opus |
| V2.7 | **Mesure intra-sujet** manuel/assisté (−30 % de temps médian sans plus d'erreurs) ; note dans `bancs/`. | — (Kevin) | V2.5 | — |
| V2.8 | **Doc** : manuel, raccourcis, `.po`, historique. | Z-doc | V2.5 | Sonnet |

Parallélisme : V2.1 dès maintenant ; V2.2 ∥ V2.3 (le format de poids est dans l'ADR) ; V2.4 ;
V2.5 ∥ V2.6 ; V2.7 ∥ V2.8.

## Lot V3 — coup et acceptation déléguée (ADR-0111)

| # | Issue | Zone | Dépend de | Modèle |
|---|---|---|---|---|
| V3.0 | Accepter l'ADR-0111 : réécriture en place de l'ADR-0044 règle 1. | docs | V2.7 | Sonnet |
| V3.1 | **Bandes et profils 1D** : rectification pleine résolution des seuls points où les `board_after` diffèrent, comptage par pile, L(n) pour n > 5. | Z-vision | V1.2 | Opus |
| V3.2 | **Score visuel et abstention** : top-K, seuil sur la précision, mesure séparée sur les coups qui ne sont pas le premier du moteur. | Z-vision | V3.1 | Opus |
| V3.3 | **Trace en schéma** : colonne sur `move`, hors hachage, `DatabaseVersion`, migration SQLite + PostgreSQL, suite de contrat, export. | Z-go | V3.0 | **Opus** |
| V3.4 | **Acceptation déléguée** : déclenchée par le lancer suivant observé, tracée, `Ctrl+Z`, ne déplace pas le curseur, la frappe prime ; activée par caméra sur la courbe risque–couverture ; indicateur. | Z-panneau | V3.2, V3.3 | Sonnet |
| V3.5 | **Score de l'overlay en vérification** (incohérence signalée, jamais écrite). | Z-vision | V1.6 | Sonnet |
| V3.6 | **Banc 09.2** geste de double (rappel > 90 % et < 1 fausse alerte/h) ; le produit ne suit que s'il est vert. | Z-vision | V3.1 | Opus |
| V3.7 | **Filtre de recherche** « acceptation déléguée » si 0111 le retient ; doc. | Z-go + Z-doc | V3.3 | Sonnet |
| V3.8 | **Doc** et historique. | Z-doc | V3.4 | Sonnet |

Parallélisme : V3.1 ∥ V3.3 ∥ V3.5 ; puis V3.2 ∥ V3.6 ; puis V3.4 ; puis V3.7 ∥ V3.8.

## Ce que ce découpage optimise

- **Conflits** : six zones, une zone chaude (Z-panneau) tenue par une seule issue à la fois ;
  tout le pixel dans un module neuf que personne d'autre ne touche.
- **Jetons** : les ouvriers Z-vision reçoivent des journaux rejouables, jamais des vidéos ; les
  seuils se règlent en vitest. Opus seulement là où l'erreur serait silencieuse (géométrie,
  automates, schéma) ; Sonnet partout où un test rougit.
- **Fusions** : issues courtes (≤ 3 jours d'ouvrier), fusionnées dans l'ordre des dépendances ;
  aucune branche longue. L'ADR de phase est acceptée **avant** le code de la phase, pas après.
- **Arrêt propre** : deux bancs bloquants en tête ; un rouge coûte deux issues, pas un lot.

## Ordre d'ouverture proposé

1. Ouvertes : H1 #849, H2 #850 ; V0.1 #843, V0.2 #844, V0.3 #845, V0.4 #846 (poste Windows), V0.5 #847 ; V2.1 #848. H3 traité directement.
2. V0.1 et V0.2 verts → V1.0, puis le lot V1 dans l'ordre ci-dessus.
3. V1.7 lu → V2.0 et le lot V2.
4. V2.7 lu → V3.0 et le lot V3.

Étiquettes GitHub : `transcription`, `lot:V0` … `lot:V3`, `effort:S/M/L`, plus la zone
(`frontend`, `backend`, `docs`).
