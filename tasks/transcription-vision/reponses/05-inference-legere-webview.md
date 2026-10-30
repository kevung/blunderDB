Recherche du 2026-10-30 — claude.ai, recherche approfondie.
Prompt : ../prompts/05-inference-legere-webview.md

---

# Inférence légère dans la webview de blunderDB : revue technique 2015–2026 (état au 30 octobre 2026)

**Réponse courte :** pour la phase 2 (dés), écrivez l'inférence à la main en JavaScript sur `Float32Array`, dans un Web Worker, avec des poids float16 au même format que le moteur Go. Pour la phase 3 (plateau), gardez la même architecture et ajoutez un petit noyau GEMM en WebAssembly SIMD écrit à la main, avec un repli JS. ONNX Runtime Web, TensorFlow.js, WebGPU et WebNN n'apportent rien à ces tailles de modèles, et chacun ajoute un risque de compatibilité réel sur WebKitGTK ou WKWebView.

## TL;DR

- **Dés (phase 2) :** un CNN d'environ 50 k paramètres sur une imagette 32×32 coûte de l'ordre de 3 M multiplications-additions, soit quelques millisecondes en JS pur. Le JS écrit à la main suffit largement : aucun fichier wasm, aucun en-tête COOP/COEP, il tourne pareil dans V8 et JavaScriptCore, et il réutilise le format float16 du moteur Go existant.
- **Plateau (phase 3) :** mêmes conventions, plus un noyau GEMM en Wasm SIMD (quelques dizaines de Ko) pour la pré-passe. Pas de threads Wasm : `crossOriginIsolated` est faux sur les schémas personnalisés de WebKitGTK et WKWebView. L'option « sans dépendance » cesse d'être raisonnable au-delà d'environ 1–2 M paramètres ou 50–100 M MAC par inférence, ou dès qu'il faut des opérateurs exotiques (détecteur avec NMS, attention). Ce seuil est mon estimation, pas une mesure publiée.
- **Écarter pour l'instant :**
  - ONNX Runtime Web : son wasm le plus léger pèse 13,58 Mo, et la variante JSEP consomme 5–7 Go sous WebKit.\[1\]\[2\]
  - TensorFlow.js : dernière version stable 4.22.0 du 21 octobre 2024.\[3\]
  - WebNN : derrière un flag, origin trial désactivée le 27 mars 2026.\[4\]
  - WebGPU : statut « UNSTABLE » dans WebKitGTK 2.52, et seulement à partir de macOS Tahoe 26 côté WKWebView.\[5\]\[6\]
  - Inférence côté Go : intéressante pour avoir un seul moteur, mais elle oblige à faire passer les imagettes par le pont Wails, que des projets mesurent à des centaines de millisecondes pour des charges de plusieurs Mo.\[7\]\[8\]

## Key Findings

1. **Wasm SIMD fonctionne partout, avec deux exceptions.**
   - JavaScriptCore active Wasm SIMD par défaut sur x86-64 seulement si le CPU a l'AVX (WebKit PR #8049, commit 258309@main, janvier 2023).\[9\] Safari l'a livré en version 16.4 (mars 2023).\[10\]
   - Sur les CPU sans AVX (Celeron, Pentium N, Atom), JSC désactive SIMD, et des projets Tauri ou WebKitGTK signalent en 2026 des plantages du processus web.\[11\]\[12\]\[13\] Le repli JS est donc obligatoire.
   - Relaxed SIMD était encore derrière un flag dans Safari fin 2025.\[14\]
2. **Threads Wasm : indisponibles en pratique sur Linux et macOS.**
   - Mesure sur WebKitGTK 2.50.4 servi par un schéma personnalisé (Tauri ou Wails) : `crossOriginIsolated = false`, SharedArrayBuffer absent.\[15\]
   - Même résultat sur WKWebView avec `WKURLSchemeHandler`. En revanche, avec les mêmes en-têtes servis depuis `http://127.0.0.1`, l'isolation fonctionne.\[16\]
   - Toute bibliothèque qui compte sur les threads tournera donc en monothread.
3. **WebKitGTK réellement distribué (octobre 2026) :**
   - Ubuntu 24.04 noble-updates : 2.52.6.\[17\]
   - Ubuntu 22.04 : 2.50.x (2.50.2 en décembre 2025, puis 2.50.4).\[18\]\[19\]
   - Debian 13 trixie : 2.52.5 (DSA 6398-1, 23 juillet 2026).\[20\]
   - Debian 12 bookworm : 2.50.6 (DSA 6172-1, mars 2026).\[21\]
   - Arch : 2.54.1.\[22\]
   - Le plancher réaliste est donc 2.50, ce qui est récent. Le facteur limitant est l'origine personnalisée, pas la version.
4. **ONNX Runtime Web est lourd et a régressé sous WebKit.**
   - Taille des fichiers wasm (jsDelivr, dernière version) : `ort-wasm-simd-threaded.wasm` 13,58 Mo, `.jsep.wasm` 27 Mo, `.asyncify.wasm` 25,54 Mo, `.jspi.wasm` 15,98 Mo.\[2\]
   - Il n'existe plus de build sans threads depuis la 1.19.\[23\]
   - Depuis la 1.21, `ort.min.js` charge le build JSEP, qui occupe 5–7 Go sous WebKit, y compris en version 1.30.0.\[1\]
5. **TensorFlow.js est en quasi-sommeil.**
   - Dernière version stable : 4.22.0 (21 octobre 2024).\[3\] Une 4.23.0-rc.0 est sortie en janvier 2025.\[24\]
   - En décembre 2025, une issue réclamait encore une nouvelle version.\[25\]
   - Ses fichiers wasm sont en revanche petits : 222 / 320 / 327 Ko en version 3.9.0.\[26\]
6. **WebGPU :**
   - Safari 26 l'active, mais `navigator.gpu` exige macOS Tahoe 26 (WebKit bug 299237, septembre 2025), ce que confirme web.dev (25 novembre 2025) : « available in macOS Tahoe 26, iOS 26, iPadOS 26, and visionOS 26 ».
   - WebKitGTK 2.52 le classe « UNSTABLE ».\[5\]
   - Côté Chromium, web.dev (25 novembre 2025) le donne disponible sous Windows (Direct3D 12), macOS et ChromeOS depuis Chrome et Edge 113, Linux étant « in progress ». Je n'ai pas trouvé de source primaire sur WebView2 en particulier.
7. **WebNN n'est pas livré.**
   - Origin trial Chrome reportée de M146, réactivée en M147, puis désactivée le 27 mars 2026.\[27\]
   - Le flag est à `false` par défaut dans M155. Statut chromestatus au 20 octobre 2026 : « Proposed ».\[4\]
   - WebKit et Gecko : aucun signe de livraison trouvé.
8. **Pont Wails v2 : volumes faibles pour la phase 2, mais coûteux pour de gros transferts.**
   - Le projet Meld mesure 230–400 ms pour un seul appel de liaison transportant une carte entière en base64.\[8\]
   - Découpé en parties de 4 Mo, avec un pont « chaud » et `toBase64` natif, chaque appel tombe à 33–44 ms.\[7\]\[8\]
   - `LoadRegion` (36,5 Mo) prend 370–430 ms, dont 32 ms d'encodage JSON côté Go.\[28\]
9. **Go pur pour ONNX :**
   - gonnx : opset 13, licence MPL-2.0, dernière mise à jour le 30 mars 2025.\[29\]\[30\]
   - Le README d'onnx-gomlx estime les moteurs Go purs environ 8× plus lents qu'ONNX Runtime sur un BERT.\[31\]
   - Go 1.27 (août 2026) ajoute un SIMD Wasm 128 bits expérimental via `simd/archsimd`, derrière `GOEXPERIMENT=simd`, avec un bug ouvert sur `v128.store`.\[32\]\[33\]

## Details

### 1. JavaScript écrit à la main (Float32Array)

**Charge de calcul (mes estimations, pas des mesures publiées).**

- **Classifieur de dés :** une architecture typique 3→16→32→64 canaux, avec des convolutions 3×3 et un pooling à chaque étage, demande environ 0,44 + 1,18 + 1,18 ≈ 2,8 M MAC.
- **Classifieur de bande (100–500 k paramètres) :** 5–20 M MAC par bande. Avec 24 bandes plus la barre et les zones de sortie, on arrive à 120–500 M MAC par image complète, si l'on analyse tout le plateau.

**Débit attendu.**

- Peu de chiffres publiés pour V8 et JSC modernes sur des noyaux GEMM en JS scalaire. Un repère ancien : 0,6 GFLOPS en JS sur Core i7 (Firefox 28, 2014, dépôt webgl-matrix-demo).\[34\]
- Pour des boucles bien écrites aujourd'hui, comptez de l'ordre de 1–3 GFLOPS sur un cœur. C'est une extrapolation, à mesurer.
- **Conséquences :**
  - Dés : environ 2–6 ms par inférence. À 10 inférences/s, moins de 10 % d'un cœur.
  - Plateau complet : environ 100–500 ms par image en JS. C'est trop pour une pré-passe rapide, mais acceptable en régime de croisière si l'on ne réanalyse que les bandes qui ont changé.

**Retours de projets.** Le pattern « petits réseaux codés à la main » est courant : le moteur Go de blunderDB en est un exemple, tout comme les réseaux de backgammon. En revanche, je n'ai pas trouvé de retour d'expérience publié et chiffré sur un CNN de vision écrit en JS pur dans une webview : **non trouvé**.

**Quantification int8 en JS.** Elle n'apporte rien en vitesse, car JS n'a pas d'instructions entières vectorielles. Elle ne sert qu'à réduire la taille des poids, ce qui est inutile à 50 k paramètres (100 Ko en float16).

### 2. WebAssembly à la main, Go ou TinyGo

**Gain réel par rapport à JS.**

- USENIX ATC 2019 (Jangda et al.) : Wasm va plus vite que JS sur SPEC, mais une multiplication de matrices reste 2–3,4× plus lente que le natif, faute d'auto-vectorisation à l'époque.\[35\]
- Une étude récente (ASRJETS) mesure, pour une multiplication 1024×1024 :
  - Wasm SIMD naïf : **0,90×** par rapport à JS, donc plus lent.\[36\]
  - Wasm SIMD avec accès mémoire par blocs ou matrice transposée : **1,64×**.\[36\]
- Le dépôt Chankse mesure 1,6–1,8× pour Wasm (Rust) face à JS en GEMM.\[37\]
- Nikhil Thorat (équipe TF.js) annonçait des gains de 3× avec Wasm SIMD sur des modèles réels (InfoQ, 5 février 2020), et le blog TensorFlow (Ann Yuan et Marat Dukhan, 2 septembre 2020) donne une fourchette de 1,7 à 4,5× par rapport au Wasm simple.

En résumé : **Wasm SIMD n'apporte de gain que si le noyau est écrit pour la vectorisation** (disposition des données, blocs). Le gain réaliste va de 1,5 à 3× sur des GEMM.

**SIMD dans les moteurs.**

- **JSC sur x86-64 :** activé si le CPU a l'AVX (PR #8049, 2023). Sans AVX, il est désactivé de force (notes de CI de Bun).\[9\]\[11\] GeoLibre et opencode-tauri documentent en 2026 des plantages du rendu WebKitGTK sur CPU sans AVX dès que Wasm est sollicité, et les contournent avec `JSC_useWasmOSR=false` ou `JSC_useJIT=0`.\[12\]\[13\]
- **WebView2 :** Chromium, donc SIMD disponible depuis plusieurs années.
- **WKWebView :** depuis Safari 16.4 (mars 2023).\[10\]

**Taille des modules.**

- Un noyau GEMM et convolution écrit en WAT ou en C (`clang -msimd128`) pèse quelques Ko à quelques dizaines de Ko. C'est une estimation.
- Go standard : le Go Wiki indique un minimum d'environ 2 Mo (« 10MB+ is common ») et environ 500 Ko après gzip ; dans son exemple 2, 2,3 Mo tombent à 668 Ko avec gzip et 496 Ko avec brotli.
- TinyGo : de quelques dizaines à quelques centaines de Ko selon les mesures, qui se contredisent : le « Hello world » du Go Wiki fait 575 octets (408 octets après gzip -6), contre 64 Ko dans l'issue #376 et 702 Ko sur un blog.

**SIMD en Go et TinyGo.**

- Go 1.27 (août 2026) émet du v128 uniquement avec `GOEXPERIMENT=simd`, via `simd/archsimd`. Le bug #82310 sur les offsets de `v128.store` est encore ouvert.\[32\]\[33\]
- TinyGo accepte `-llvm-features +simd128`, mais n'émet pas forcément d'arithmétique flottante vectorisée : une PR tierce rapporte « no packed floating-point arithmetic » avec TinyGo 0.41.1.\[38\]\[39\]

**Aller-retour des données.**

- `syscall/js.CopyBytesToGo` et `CopyBytesToJS` n'acceptent que des `Uint8Array`. Il n'existe aucune vue `Float32Array` directe (go#31980, go#30210).\[40\]\[41\]\[42\]
- Pour un module Wasm écrit à la main, en revanche, le JS écrit directement dans `memory.buffer` à travers une vue `Float32Array`. La copie d'une imagette de 32×32×3 flottants (12 Ko) est négligeable.

**Verdict.** Go et TinyGo en Wasm cumulent le surcoût de taille, un SIMD expérimental et une interop limitée aux octets. À éviter. Un noyau Wasm écrit à la main et très petit est la bonne façon d'accélérer.

### 3. ONNX Runtime Web

**Tailles des artefacts (jsDelivr, dernière version, octobre 2026).**

| Fichier | Taille |
|---|---|
| `ort-wasm-simd-threaded.wasm` | 13,58 Mo |
| `ort-wasm-simd-threaded.jspi.wasm` | 15,98 Mo |
| `ort-wasm-simd-threaded.asyncify.wasm` (WebGPU) | 25,54 Mo |
| `ort-wasm-simd-threaded.jsep.wasm` | 27 Mo |

- Évolution : 11 Mo en 1.20.1, 14 Mo en 1.30.0 (BirdBox PR #31).\[1\]
- Les builds sans SIMD et sans threads ont été supprimés à partir de la 1.19. L'équipe ORT explique que le build threadé fonctionne en monothread sans worker, et cite la limite de taille du paquet npm (issue #25666).\[23\]

**Compatibilité WebKit.** Depuis la 1.21, `ort.min.js` charge le build JSEP même pour le backend wasm. Sous WebKit, cela consomme 5–7 Go, toujours en 1.30.0 (microsoft/onnxruntime#33516, cité par BirdBox). Le contournement consiste à importer `ort.wasm.min.js`.\[1\] Transformers.js mesure par ailleurs un pic mémoire de 2 273 Mo avec Asyncify contre 1 543 Mo avec JSPI sous Safari.\[43\]

**Licence et rythme.** Licence MIT. Le rythme de versions est soutenu : de la 1.19 en 2024 à la 1.30 et à la 1.31-dev en septembre 2026. Les ruptures portent surtout sur les noms et le choix des fichiers wasm (1.19, 1.21).

**Verdict.** Ajouter au moins 13,6 Mo au binaire, plus des pièges mémoire sous WebKit, pour exécuter un modèle de 100 Ko, n'a pas de sens.

### 4. TensorFlow.js

- **Dernière version stable :** 4.22.0, du 21 octobre 2024. Une 4.23.0-rc.0 est sortie le 8 janvier 2025.\[24\] En décembre 2025, l'issue #8609 constate qu'aucune version n'a suivi le correctif d'avril.\[25\]
- **Taille des fichiers wasm (3.9.0) :** `tfjs-backend-wasm.wasm` 222 Ko, `-simd` 320 Ko, `-threaded-simd` 327 Ko.\[26\] La copie dans JetStream fait 311 et 425 Ko.\[44\]\[45\] Le paquet npm 4.22.0 fait 13,2 Mo décompressé, mais le runtime téléchargé reste petit.\[3\]
- **Licence :** Apache-2.0.
- **Backend webgl :** dépend de WebGL2 dans WebKitGTK, qui varie selon le pilote (nombreux contournements DMA-BUF documentés par GeoLibre).\[13\]

**Verdict.** Techniquement léger, mais c'est une dépendance quasi abandonnée. Mieux vaut ne pas lier un produit de 2026 à ce projet.

### 5. WebNN et WebGPU

**WebNN.**

- Intent to Experiment : M145–147, puis reporté à M146–148 (5 février 2026), réactivé en M147, puis désactivé le 27 mars 2026 « due to more release blocking issues ».\[27\]
- Le flag `kWebMachineLearningNeuralNetwork` est à `false` par défaut dans M155. Statut « Proposed » (issue #964, 20 octobre 2026). webnn.io annonce « Chrome 160 (TBD) to 164 (TBD) ».\[4\]
- Aucune implémentation livrée dans WebKit ou Gecko trouvée : **non trouvé**.
- **Prématuré.**

**WebGPU.**

- Chrome et Edge : depuis la version 113 sous Windows (Direct3D 12), macOS et ChromeOS, Linux étant « in progress » (web.dev, 25 novembre 2025).
- Safari 26 (15 septembre 2025) : sur macOS Tahoe 26 uniquement, d'après WebKit bug 299237 (« navigator.gpu requires macOS Tahoe ») et web.dev (25 novembre 2025). **Contradiction :** un guide tiers (cinevva) l'annonce aussi pour Sequoia et Sonoma.\[46\] Je retiens la source WebKit.
- WebKitGTK : fonction « UNSTABLE » en 2.52 (tinyjsapp #78). Une issue de projet cite la liste webkit-gtk d'octobre 2023 (« not supported and nobody is working on it »).\[5\]\[47\]
- WebView2 : non confirmé par une source primaire.
- **Prématuré** pour un binaire unique sur trois plateformes, et inutile à ces tailles : le temps de lancement d'une passe GPU dépasse le calcul lui-même.

### 6. Inférence côté Go pur

**Bibliothèques.**

| Bibliothèque | État | Remarque |
|---|---|---|
| gonnx (AdvancedClimateSystems) | Dernière mise à jour 30 mars 2025, opset 13, MPL-2.0 | Opsets incomplets, maintenance faible |
| onnx-go (oramasearch) | Cité comme option Go pure par onnx-gomlx | État de maintenance 2026 non vérifié |
| GoMLX, backend `go` | Actif, Go pur | Qualifié par ses auteurs de « very portable but slow » |
| onnxruntime-purego | Sans CGO, mais charge `libonnxruntime.so` | **Exclu** : dépendance native |
| gorgonia, spago | Non vérifiés dans cette revue | Pas d'activité 2026 trouvée |

**Coût du pont Wails v2.**

- Imagettes de dés : 32×32×3 octets ≈ 3 Ko, soit environ 4 Ko en base64. À 10 images/s, environ 40 Ko/s : négligeable en volume.
- Plateau complet : environ 300 Ko par image, soit 3–4 Mo/s en base64, et bien davantage en pré-passe.
- Ce que mesure Meld (Wails v2, 2026) :
  - une liaison de plusieurs Mo bloque la page 230–400 ms ;\[8\]
  - en parties de 4 Mo, 33–44 ms par appel ;\[7\]
  - des pauses de 250–760 ms sans aucun script en cours (#639).\[7\]
- L'issue wails#4418 signale que sous WebView2, des promesses de liaison ne se résolvent jamais en cas de forte charge.\[48\]

**Alternative : AssetServer `Handler`.** Wails v2 transmet les requêtes non GET au `Handler` Go.\[49\]\[50\] Ce qui fonctionne et ce qui casse :

- Un corps brut `application/octet-stream` de 2 Mio arrive intact sur les trois plateformes.\[51\]
- Un corps multipart provoque un SIGSEGV sous WebKitGTK et arrive vide sous WKWebView (wails#6068, 2026).\[51\]
- Le support des corps de requête sous Linux exige WebKit2GTK 2.36 ou 2.40 et des tags de build.\[52\]\[53\]
- Meld a d'ailleurs adopté la règle « jamais de POST vers l'asset server » sous WebKitGTK.\[8\]
- Le streaming de corps de réponse n'est pas pris en charge sous Windows.\[54\]

**Verdict.** Pour la phase 2, le coût du pont est tolérable. Mais un aller-retour par inférence ajoute une latence et un risque de blocage que l'inférence locale n'a pas. L'argument « un seul moteur » vaut pour le **format de poids et les tests de parité**, pas pour le runtime.

### 7. Distribution des poids

**Volumes en jeu.**

- Dés : 50 k paramètres × 2 octets (float16) = 100 Ko.
- Plateau : 0,5 M × 2 = 1 Mo.
- Avec int8 par canal : 50–500 Ko.

**Embarquement.** Wails v2 sert déjà le frontend depuis un `embed.FS`.\[55\] Mettre les poids dans le bundle frontend revient donc à les mettre dans le binaire. Il suffit de les placer dans `frontend/dist`, et le worker les charge avec `fetch()` (requête GET, sans les pièges des POST). Je n'ai pas vérifié qu'un Worker démarre bien depuis le schéma `wails://` sous Linux : Meld le signale comme point à tester (#625).\[56\]

**Taille acceptable.** Quelques Mo restent invisibles dans un binaire de plusieurs dizaines de Mo. Le seuil de gêne se situerait au-delà de 10–20 Mo, c'est-à-dire la taille du seul wasm d'ORT.

### 8. Entraînement et export

**Chaîne recommandée.** PyTorch, puis `state_dict`, puis export des tenseurs en float16 dans un fichier simple : un en-tête JSON de longueur préfixée (nom, forme, dtype, offset) suivi des données brutes en little-endian. C'est la structure de safetensors, lisible en environ 50 lignes de Go ou de JS. Il est plus simple encore de fusionner la BatchNorm dans les convolutions avant l'export. L'export ONNX n'est utile que comme référence de parité, ou si vous dépassez le seuil.

**Quantification post-entraînement.** À 50–500 k paramètres, stocker en float16 et calculer en float32 suffit. L'int8 (par canal, symétrique, calibré sur quelques centaines d'imagettes) ne se justifie qu'avec un noyau Wasm SIMD entier. Ce point n'est pas sourcé ici : recommandation d'ingénierie.

## Recommendations

### 1. Verdict

L'option « sans dépendance » est la bonne pour les deux phases. Les alternatives clés en main sont soit lourdes (ORT Web : 13,6 Mo et plus, plus un risque mémoire sous WebKit), soit en sommeil (TF.js), soit pas encore livrées (WebNN, WebGPU sous Linux). Les threads Wasm sont indisponibles dans les webviews à schéma personnalisé, ce qui annule l'avantage principal d'ORT.

### 2. Recommandation technique

**Phase 2 (dés), décision ferme.**

- Un module JS (ES2020) dans un Web Worker : convolution 3×3 directe, ReLU, max-pool et couche dense, en boucles sur `Float32Array` avec des tampons préalloués. Les poids sont en float16, convertis en float32 au chargement.
- Transfert des imagettes par `postMessage` avec un `ArrayBuffer` transférable.
- Mêmes fichiers de poids et mêmes vecteurs de test « golden » que le moteur Go, pour vérifier la parité au bit près avec une tolérance de 1e-4.

**Phase 3 (plateau), décision ferme.**

- Même runtime JS, avec un chemin rapide : un GEMM Wasm SIMD 128 bits écrit à la main (WAT ou C/`clang -msimd128`, moins de 50 Ko), précédé d'un `im2col` et organisé par blocs.
- Détection à l'exécution avec `WebAssembly.validate` d'un module v128, et repli JS automatique : CPU sans AVX sous JSC, avec en plus la recommandation d'exporter `JSC_useWasmOSR=false` côté Go sur ces machines, comme GeoLibre.\[13\]
- Monothread. Lots de bandes. Réanalyse uniquement des bandes dont la différence d'image dépasse un seuil.

**Seuil.** Restez « sans dépendance » tant que le modèle tient sous environ 1–2 M paramètres et 50–100 M MAC par inférence, et n'utilise que conv, pool, dense, BN fusionnée et activations simples. Au-delà, ou pour un détecteur avec ancres et NMS, passez à ORT Web en important `ort.wasm.min.js` (pas `ort.min.js`), en épinglant la version et en mesurant la mémoire sous WebKitGTK.\[1\]

### 3. Incertitudes et expériences à mener

1. **Débit réel JS et Wasm SIMD sur vos trois webviews.** Banc de test : le même CNN de dés et un modèle de bande à 300 k paramètres, 1 000 inférences dans un Worker, médiane et p95, sur WebKitGTK 2.50 et 2.52 (Ubuntu 22.04 et 24.04), WebView2 et WKWebView (macOS 14 et 26), plus un CPU sans AVX.
2. **Démarrage d'un Worker depuis `wails://` sous WebKitGTK** (Meld #625 non résolu). Si cela échoue, construire le worker depuis une URL Blob.\[56\]
3. **`crossOriginIsolated` sous WebView2 dans Wails v2 :** non vérifié. C'est sans importance si vous n'utilisez pas les threads.
4. **Latence d'un appel de liaison Wails v2 de 4 Ko à 10 Hz**, si vous gardez une option Go : à mesurer, car Meld ne mesure que des charges de plusieurs Mo.
5. **Estimations de FLOPs et de GFLOPS JS :** ce sont mes calculs. À confirmer par l'expérience 1.

## Caveats

- Plusieurs faits proviennent d'issues GitHub de projets tiers (Meld, BirdBox, web-playground, GeoLibre). Ce sont des mesures de terrain, primaires pour ces projets, mais dans des configurations particulières.
- Les tailles ORT viennent du listing jsDelivr et de dépôts Hugging Face, pas des notes de version officielles.
- Les estimations de débit et de seuil sont des extrapolations et sont signalées comme telles.

## Sources

1. WebKit PR #8049 « [JSC] Enable Wasm SIMD on x64 », WebKit, janvier 2023, https://github.com/WebKit/WebKit/pull/8049. Wasm SIMD activé sur x64 seulement si le CPU a l'AVX. Primaire.
2. WebKit Bugzilla 222382 « WebAssembly: SIMD support », WebKit, 2021–2023, https://bugs.webkit.org/show_bug.cgi?id=222382. Suivi du support SIMD dans JSC. Primaire.
3. « The State of WebAssembly 2023–2024 » et « 2025–2026 », Uno Platform (G. Gallant), 2024 et 2026, https://platform.uno/blog/the-state-of-webassembly-2025-2026/. Safari 16.4 en mars 2023 ; Relaxed SIMD encore derrière un flag dans Safari. Secondaire.
4. oven-sh/bun PR #34877, Bun, 2026, https://github.com/oven-sh/bun/pull/34877. JSC force `useWasmSIMD=false` sans AVX. Primaire (CI).
5. GeoLibre « Getting Started », 2026, https://geolibre.app/getting-started/. Plantages WebKitGTK sur CPU sans AVX et variables `JSC_*` de contournement. Primaire (projet).
6. tahayerd/opencode-tauri release v1.1.6, 2026, https://github.com/tahayerd/opencode-tauri/releases/tag/v1.1.6. JIT de WebKitGTK 2.52 émettant de l'AVX sur des CPU sans AVX. Primaire (projet).
7. uist1idrju3i/web-playground PR #10, 2026, https://github.com/uist1idrju3i/web-playground/pull/10. WebKitGTK 2.50.4 sur schéma personnalisé : `crossOriginIsolated` faux, pas de SharedArrayBuffer. Primaire (mesure).
8. logos-fleet/logos-workspace issue #287, 2026, https://github.com/logos-fleet/logos-workspace/issues/287. WKURLSchemeHandler : non isolé ; loopback http : isolé. Primaire (mesure).
9. WebKit commit 104506c « [GLIB] Enable SharedArrayBuffer in GLib-backed WebKit », WebKit, date non relevée, https://github.com/WebKit/WebKit/commit/104506cdb2c45aa547db1eb50313abc15473b815. SharedArrayBuffer activé dans les ports GTK et WPE. Primaire.
10. Ubuntu packages « webkit », Canonical, octobre 2026, https://packages.ubuntu.com/webkit. noble-updates en 2.52.6. Primaire.
11. USN-7914-1, Canonical, décembre 2025, https://ubuntu.com/security/notices/USN-7914-1. 2.50.2 pour 22.04, 24.04 et 25.x. Primaire.
12. DSA 6398-1 webkit2gtk, Debian, 23 juillet 2026, https://lists.debian.org/debian-security-announce/2026/msg00309.html. trixie en 2.52.5. Primaire.
13. DSA 6172-1 webkit2gtk, Debian, 21 mars 2026, https://lists.debian.org/debian-security-announce/2026/msg00081.html. bookworm en 2.50.6, trixie en 2.50.6 à cette date. Primaire.
14. onnxruntime-web, listing CDN, jsDelivr, consulté en octobre 2026, https://cdn.jsdelivr.net/npm/onnxruntime-web/dist/. Tailles des fichiers wasm (13,58 / 15,98 / 25,54 / 27 Mo). Primaire (artefacts).
15. microsoft/onnxruntime issue #25666, Microsoft, 2025, https://github.com/microsoft/onnxruntime/issues/25666. Suppression des builds sans threads depuis la 1.19, et justification. Primaire.
16. birdnet-team/BirdBox PR #31, 2026, https://github.com/birdnet-team/BirdBox/pull/31. JSEP à 5–7 Go sous WebKit ; wasm de 11 Mo (1.20.1) contre 14 Mo (1.30.0). Primaire (mesure).
17. huggingface/transformers.js issue #1835, 2026, https://github.com/huggingface/transformers.js/issues/1835. Asyncify contre JSPI : mémoire Safari 2 273 contre 1 543 Mo, wasm 26,9 contre 16,8 Mo. Primaire (mesure).
18. @tensorflow/tfjs-backend-wasm 3.9.0, listing CDN, jsDelivr, 2021, https://cdn.jsdelivr.net/npm/@tensorflow/tfjs-backend-wasm@3.9.0/dist/. Tailles 222 / 320 / 327 Ko. Primaire (artefacts).
19. @tensorflow/tfjs sur npm, Google, 2024–2026, https://www.npmjs.com/package/@tensorflow/tfjs. Dernière version 4.22.0. Primaire.
20. tensorflow/tfjs issue #8609, décembre 2025, https://github.com/tensorflow/tfjs/issues/8609. Aucune version depuis la 4.22.0. Primaire.
21. webmachinelearning/webnn issue #964, W3C WebML CG (M. Salnikov), 20 octobre 2026, https://github.com/webmachinelearning/webnn/issues/964. État réel de WebNN : flag, OT fermée. Primaire.
22. « Intent to Experiment: WebNN », blink-dev, 2026, https://groups.google.com/a/chromium.org/g/blink-dev/c/5CWKSChYo98. Reports et désactivation de l'origin trial. Primaire.
23. « WebGPU is now supported in major browsers », web.dev (Google), novembre 2025, https://web.dev/blog/webgpu-supported-major-browsers. Matrice de disponibilité de WebGPU. Primaire (éditeur).
24. WebKit Bugzilla 299237, WebKit, septembre 2025, https://bugs.webkit.org/show_bug.cgi?id=299237. `navigator.gpu` exige macOS Tahoe. Primaire.
25. tarwin/tinyjsapp issue #78, 2026, https://github.com/tarwin/tinyjsapp/issues/78. WebGPU « UNSTABLE » dans WebKitGTK 2.52. Primaire (projet).
26. xjmusic issue #576, 2026, https://github.com/xjmusic/xjmusic/issues/576. Cite la liste webkit-gtk d'octobre 2023 sur l'absence de WebGPU. Secondaire.
27. Coldensjo/Meld issues #637, #639, #636, #546 et #625, 2026, https://github.com/Coldensjo/Meld/issues/637. Coût mesuré du pont Wails v2 et de base64 ; Workers sur `wails://`. Primaire (mesure).
28. wailsapp/wails issue #4418, 2025, https://github.com/wailsapp/wails/issues/4418. Promesses de liaison perdues sous WebView2 en cas de forte charge. Primaire.
29. wailsapp/wails issue #6068, 2026, https://github.com/wailsapp/wails/issues/6068. Corps POST multipart : SIGSEGV sous WebKitGTK, vide sous WKWebView ; octet-stream de 2 Mio correct. Primaire.
30. Wails v2.15.0 Options et pkg.go.dev assetserver, Wails, 2026, https://pkg.go.dev/github.com/wailsapp/wails/v2/pkg/options/assetserver. Handler et Middleware ; requêtes non GET transmises au Handler. Primaire.
31. « Streams — internals », Wails (v3), 2026, https://wails.io/guides/advanced/streams-internals/. Surcoût base64 de 33 % ; WebView2 limite les corps à environ 2 Mo. Primaire (extrait seulement).
32. AdvancedClimateSystems/gonnx, 2025, https://github.com/AdvancedClimateSystems/gonnx. ONNX en Go pur, opset 13, MPL-2.0. Primaire.
33. gomlx/onnx-gomlx, 2026, https://github.com/gomlx/onnx-gomlx. Go pur environ 8× plus lent qu'ORT sur BERT. Primaire (affirmation des auteurs).
34. Notes de version Go 1.27 et billet « Arch-specific SIMD in Go », Go team, août et octobre 2026, https://go.dev/doc/go1.27. SIMD Wasm 128 bits expérimental derrière `GOEXPERIMENT=simd`. Primaire.
35. golang/go issues #82310, #31980 et #30210, 2019–2026, https://github.com/golang/go/issues/82310. Bug `v128.store` ; interop limitée aux octets. Primaire.
36. Go Wiki « WebAssembly », Go team, non daté, https://go.dev/wiki/WebAssembly. Environ 2 Mo au minimum, environ 500 Ko après gzip ; tailles TinyGo. Primaire.
37. tinygo-org/tinygo PR #1844 et issue #2196, 2021 et 2024, https://github.com/tinygo-org/tinygo/pull/1844. Option `-llvm-features +simd128`. Primaire.
38. Jangda et al., « Not So Fast: Analyzing the Performance of WebAssembly vs. Native Code », USENIX ATC 2019, https://www.usenix.org/system/files/atc19-jangda.pdf. Wasm comparé au natif et à JS ; étude de cas GEMM. Primaire (article).
39. « Integration of WebAssembly in Performance-critical Web… », ASRJETS, 2025 environ, https://asrjetsjournal.org/American_Scientific_Journal/article/download/11676/2837/27853. Wasm SIMD naïf 0,90×, par blocs 1,64× face à JS. Secondaire (revue peu sélective).
40. Chankse/JavaScript-vs-WebAssembly-Comparative-Analysis, non daté, https://github.com/Chankse/JavaScript-vs-WebAssembly-Comparative-Analysis. GEMM Wasm 1,6–1,8× plus rapide que JS. Blog/dépôt.
41. « Google's V8 Engine Adds Support for WebAssembly SIMD », InfoQ, février 2020, https://www.infoq.com/news/2020/02/v8-webassembly-simd/. Gain d'environ 3× annoncé par l'équipe TF.js. Secondaire.
42. watmough/webgl-matrix-demo, 2014, https://github.com/watmough/webgl-matrix-demo. Environ 0,6 GFLOPS en JS (Firefox 28). Blog/dépôt, ancien.

## Sources

1. [fix: WASM-only ONNX Runtime bundle for the browser demo (WebKit memory) by El1as42 · Pull Request #31 · birdnet-team/BirdBox · GitHub](https://github.com/birdnet-team/BirdBox/pull/31)
2. [onnxruntime-web CDN by jsDelivr - A free, fast, and reliable Open Source CDN](https://cdn.jsdelivr.net/npm/onnxruntime-web/dist/)
3. [@tensorflow/tfjs-backend-wasm - npm](https://www.npmjs.com/package/@tensorflow/tfjs-backend-wasm)
4. [What is the current testing/shipping status of WebNN in Chrome and Edge, and what should developers use today? · Issue #964 · webmachinelearning/webnn](https://github.com/webmachinelearning/webnn/issues/964)
5. [Linux/Wayland: app dies within a second — WebKit processes die, document never loads; no working GPU-accelerated path · Issue #78 · tarwin/tinyjsapp · GitHub](https://github.com/tarwin/tinyjsapp/issues/78)
6. [299237](https://bugs.webkit.org/show_bug.cgi?id=299237)
7. [Find what pauses the probe page 250-760 ms with no script running while large binding calls cross the Wails bridge · Issue #639 · Coldensjo/Meld · GitHub](https://github.com/Coldensjo/Meld/issues/639)
8. [Send the autosave's recovery copy across the Wails bridge in parts, since one SaveRecovery call stalls the page 230-400 ms on a 4x4-tile map · Issue #637 · Coldensjo/Meld · GitHub](https://github.com/Coldensjo/Meld/issues/637)
9. [\[JSC\] Enable Wasm SIMD on x64 by Constellation · Pull Request #8049 · WebKit/WebKit · GitHub](https://github.com/WebKit/WebKit/pull/8049)
10. [The State of WebAssembly](https://platform.uno/blog/state-of-webassembly-2023-2024/)
11. [ci(verify-baseline): parse //@ JSC flags, skip v128 wasm fixtures on x64, re-enable --jit-stress on WebKit changes by robobun · Pull Request #34877 · oven-sh/bun · GitHub](https://github.com/oven-sh/bun/pull/34877)
12. [Release v1.1.6 · tahayerd/opencode-tauri · GitHub](https://github.com/tahayerd/opencode-tauri/releases/tag/v1.1.6)
13. [Getting Started - GeoLibre](https://geolibre.app/getting-started/)
14. [The State of WebAssembly](https://platform.uno/blog/the-state-of-webassembly-2025-2026/)
15. [feat: add Tauri v2, Wails v3, and Electron native shells for BrowserFeatures by devin-ai-integration\[bot\] · Pull Request #10 · uist1idrju3i/web-playground · GitHub](https://github.com/uist1idrju3i/web-playground/pull/10)
16. [Spike: wss dial, secure context, and SharedArrayBuffer from the Worker on device · Issue #287 · logos-fleet/logos-workspace · GitHub](https://github.com/logos-fleet/logos-workspace/issues/287)
17. [Ubuntu](https://packages.ubuntu.com/webkit)
18. [USN-7914-1: WebKitGTK vulnerabilities](https://ubuntu.com/security/notices/USN-7914-1)
19. [USN-7957-1: WebKitGTK vulnerabilities](https://ubuntu.com/security/notices/USN-7957-1)
20. [\[SECURITY\] \[DSA 6398-1\] webkit2gtk security update](https://lists.debian.org/debian-security-announce/2026/msg00309.html)
21. [\[SECURITY\] \[DSA 6172-1\] webkit2gtk security update](https://lists.debian.org/debian-security-announce/2026/msg00081.html)
22. [Arch Linux - webkit2gtk-4.1 2.54.1-1 (x86\_64) - File List](https://archlinux.org/packages/extra/x86_64/webkit2gtk-4.1/files/)
23. [\[Web\] Clarification on wasm/simd vs wasm/simd/threaded default in onnxruntime-web v1.19.0+ · Issue #25666 · microsoft/onnxruntime · GitHub](https://github.com/microsoft/onnxruntime/issues/25666)
24. [@tensorflow/tfjs 4.23.0-rc.0 on npm - Libraries.io - security & maintenance data for open source software](https://libraries.io/npm/@tensorflow%2Ftfjs)
25. [latest release (4.22.0) tfjs-node is broken with node 24 · Issue #8609 · tensorflow/tfjs · GitHub](https://github.com/tensorflow/tfjs/issues/8609)
26. [@tensorflow/tfjs-backend-wasm CDN by jsDelivr - A free, fast, and reliable Open Source CDN](https://cdn.jsdelivr.net/npm/@tensorflow/tfjs-backend-wasm@3.9.0/dist/)
27. [Intent to Experiment: WebNN](https://groups.google.com/a/chromium.org/g/blink-dev/c/5CWKSChYo98)
28. [Measure LoadRegion's bridge cost and send a region's alpha maps without 6,000 JSON base64 strings, since they are 33 of a 3x3 view's 36.5 MB · Issue #636 · Coldensjo/Meld · GitHub](https://github.com/Coldensjo/Meld/issues/636)
29. [GitHub - AdvancedClimateSystems/gonnx: Run ONNX models using Go · GitHub](https://github.com/AdvancedClimateSystems/gonnx)
30. [AdvancedClimateSystems repositories · GitHub](https://github.com/orgs/AdvancedClimateSystems/repositories?type=all)
31. [GitHub - gomlx/onnx-gomlx: ONNX / GoMLX Conversion · GitHub](https://github.com/gomlx/onnx-gomlx)
32. [cmd/internal/obj/wasm: V128Store offset is always encoded as 0 · Issue #82310 · golang/go · GitHub](https://github.com/golang/go/issues/82310)
33. [Go 1.27 Release Notes - The Go Programming Language](https://go.dev/doc/go1.27)
34. [GitHub - watmough/webgl-matrix-demo: High performance matrix multiplication in Javascript vs WebGL · GitHub](https://github.com/watmough/webgl-matrix-demo)
35. [Analyzing the Performance of WebAssembly vs. Native Code](https://www.usenix.org/system/files/atc19-jangda.pdf)
36. [Integration of WebAssembly in Performance-critical Web ...](https://asrjetsjournal.org/American_Scientific_Journal/article/download/11676/2837/27853)
37. [GitHub - Chankse/JavaScript-vs-WebAssembly-Comparative-Analysis · GitHub](https://github.com/Chankse/JavaScript-vs-WebAssembly-Comparative-Analysis)
38. [Add -llvm-features parameter by fgsch · Pull Request #1844 · tinygo-org/tinygo · GitHub](https://github.com/tinygo-org/tinygo/pull/1844)
39. [Add opt-in WASM SIMD builds and comparison tools by odvcencio · Pull Request #158 · odvcencio/cicada · GitHub](https://github.com/odvcencio/cicada/pull/158)
40. [js package - syscall/js - Go Packages](https://pkg.go.dev/syscall/js)
41. [misc/wasm: js.TypedArrayOf is impossible to use correctly · Issue #31980 · golang/go · GitHub](https://github.com/golang/go/issues/31980)
42. [syscall/js: no method to convert TypedArrays into slices · Issue #30210 · golang/go · GitHub](https://github.com/golang/go/issues/30210)
43. [Use onnxruntime-web's JSPI build when the browser supports JSPI (much lower memory in Safari) · Issue #1835 · huggingface/transformers.js · GitHub](https://github.com/huggingface/transformers.js/issues/1835)
44. [wasm/tfjs-backend-wasm-simd.wasm - external/github.com/WebKit/JetStream - Git at Google](https://chromium.googlesource.com/external/github.com/WebKit/JetStream/+/e71ad6a0b43f309d7aa20f08a6def9421fd41445/wasm/tfjs-backend-wasm-simd.wasm)
45. [wasm/tfjs-backend-wasm.wasm - external/github.com/WebKit/JetStream - Git at Google](https://chromium.googlesource.com/external/github.com/WebKit/JetStream/+/e71ad6a0b43f309d7aa20f08a6def9421fd41445/wasm/tfjs-backend-wasm.wasm)
46. [WebGL 2 or WebGPU Not Supported? How to Fix It in Every Browser (2026)](https://app.cinevva.com/guides/webgl-webgpu-not-supported-fix)
47. [Build the Tauri 2 + Svelte slice with a Canvas2D surface on Windows, macOS and Linux · Issue #576 · xjmusic/xjmusic · GitHub](https://github.com/xjmusic/xjmusic/issues/576)
48. [\[v2\] WebView2 Bridge Loses JavaScript Promises Under Heavy Load / Promises Never Resolve · Issue #4418 · wailsapp/wails · GitHub](https://github.com/wailsapp/wails/issues/4418)
49. [assetserver package - github.com/wailsapp/wails/v2/pkg/options/assetserver - Go Packages](https://pkg.go.dev/github.com/wailsapp/wails/v2/pkg/options/assetserver)
50. [Application Development](https://wails.io/docs/next/guides/application-development/)
51. [v2.14.0: multipart/form-data request body SIGSEGVs the Linux asset server (webkit2\_41) and arrives empty on macOS](https://github.com/wailsapp/wails/issues/6068)
52. [Options](https://wails.io/ru/docs/reference/options/)
53. [wails/website/docs/reference/options.mdx at master · wailsapp/wails · GitHub](https://github.com/wailsapp/wails/blob/master/website/docs/reference/options.mdx)
54. [Options](https://wails.io/docs/v2.15.0/reference/options/)
55. [Migrating from v1](https://wails.io/docs/next/guides/migrating/)
56. [Check that the model workers start in the Linux build, where WebKitGTK serves the app from the wails:// scheme · Issue #625 · Coldensjo/Meld · GitHub](https://github.com/Coldensjo/Meld/issues/625)
