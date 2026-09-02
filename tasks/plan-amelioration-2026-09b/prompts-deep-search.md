<!-- Annexe du plan tasks/plan-amelioration-2026-09b/README.md : les questions
telles qu'elles sont posées. Les rapports se versent sous docs/recherche/
(P5 et suivants, la numérotation P1-P4 étant prise par la campagne gammonNet). -->

# Prompts de deep search

Chaque prompt est autonome et se donne tel quel à un moteur de recherche
approfondie. Convention des rapports (`docs/recherche/README.md`) : chaque
affirmation porte son marqueur `[MESURE]`, `[ÉDITEUR/DÉCLARÉ]`, `[EXTRAPOLÉ]`,
`[NON TROUVÉ]` ; un rapport documente, il ne décide pas — la décision passe
par une ADR ou une fiche. Numérotation : **P5 à P16** (P1-P4 existent déjà).

| Prompt | Alimente | Urgence |
|---|---|---|
| P5 classification de position | J.1a, I.9 | avant J.1 |
| P6 modèle de videau : efficacité par propriétaire, inversion en forme close | C.5, C.7 | avant C.5 |
| P7 similarité et kNN pur Go | J.3 | avant J.3 |
| P8 rollouts | J.2 | avant J.2 |
| P9 formats de fichiers concurrents | I.4, I.5, B.7 | avant I.4 |
| P10 pédagogie et FSRS à note dérivée | J.4, I.17-I.20 | avant J.4 |
| P11 compression de petits blobs JSON en base | B.12 | avant B.12 étape 2 |
| P12 diagrammes de backgammon en SVG | I.22, I.23 | avant I.22 |
| P13 multi-tenant PostgreSQL | A.1, G.3, G.11 | avant G.11 |
| P14 supply chain GitHub | A.4, E.7 | avant E.7 |
| P15 Svelte 5 : i18n paresseuse, tokens, a11y des tables | D.7, D.9, D.5 | avant D.9 |
| P16 distribution desktop : Flathub, mise à jour, associations | H.3, G.13 | avant H.3 |
| P17 filigrane lié au contenu, conteneur AEAD | A.11, J.6 | avant A.11 (v2) |
| P18 taxonomie des erreurs dans les outils d'analyse d'échecs et de poker | J.1b, J.8 | avant J.1b |

---

### P5 — Classer une position de backgammon par type de jeu, à partir du seul plateau

> Je maintiens blunderDB, une base de positions de backgammon (Go + SQLite)
> qui stocke pour chaque position : les 26 points (pions par point et par
> camp, barre, sortis), le score, le videau, les dés, le pipcount, un drapeau
> contact/no-contact, et souvent l'analyse d'un moteur (gagne/gammon/backgammon
> des deux camps, meilleurs coups). Je veux attribuer automatiquement à chaque
> position une ou deux étiquettes de **type de jeu** — course, holding game,
> mutual holding, backgame (1-2, 1-3, 2-3…), blitz, prime vs prime, jeu de
> containment, ace-point game, crunch, bear-in avec contact — de façon
> **déterministe, explicable et recalculable** (jamais éditée à la main).
>
> Fais une recherche approfondie et sourcée sur :
> 1. Les **définitions** données par la littérature (Magriel *Backgammon*,
>    Robertie, Woolsey, Trice *Backgammon Boot Camp*, Ballard/Weaver *Backgammon
>    Openings*, les articles de bkgm.com et du GammonVillage) : critères
>    positionnels explicites (points d'ancrage tenus, longueur de l'amorce la
>    plus longue, nombre de pions arriérés, différentiel de course, pions dans
>    le jan adverse, timing), avec les seuils numériques quand ils existent.
> 2. Ce que font les **logiciels** : la fonction `ClassifyPosition` de GNU
>    Backgammon (ses classes CLASS_RACE, CLASS_CRASHED, CLASS_CONTACT et leurs
>    critères exacts dans le code source), la classification interne de
>    eXtreme Gammon si elle est documentée, BGBlitz, et les « position
>    categories » de sites d'entraînement (Backgammon Galaxy, bgblitz, Heroes,
>    OpenGammon).
> 3. Les travaux publiés (académiques ou blogs techniques) qui **mesurent** un
>    taux d'accord entre classifieurs heuristiques, ou entre heuristiques et
>    jugement humain, sur des corpus de positions.
> 4. Une proposition concrète : un arbre de décision de 8-10 règles ordonnées
>    (avec les seuils), applicable en O(1) sur le vecteur des 26 points, et la
>    liste des cas ambigus qu'il faut assumer (par exemple holding vs backgame
>    à deux ancres).
>
> Livrable : synthèse, tableau des définitions par source, l'arbre de règles
> proposé avec ses seuils sourcés, et les positions-tests canoniques
> (XGID) qui illustrent chaque classe.

### P6 — Modèle de Janowski : efficacité du videau par propriétaire, et inversion en forme close

> Je porte en Go le module de décision de videau de gammonNet (modèle de
> Janowski, MET Kazaross-XG2, recherche expectiminimax 0→2-ply). Deux
> questions précises de modèle :
>
> 1. **L'efficacité du videau (`cube_x`, la fraction « live » de Janowski)
>    doit-elle être miroitée avec le propriétaire du videau à chaque ply ?**
>    Dans mon port, l'efficacité est fixée à la racine selon le propriétaire
>    (centré ≈ 0,68, possédé ≈ 0,57, adverse ≈ 0,69 — valeurs mesurées en
>    amont), puis chaque feuille de l'arbre est valorisée avec cette même
>    efficacité alors que le propriétaire est miroité à chaque ply. De même,
>    la branche « double pris » (`eDT`, videau désormais détenu par
>    l'adversaire) est tarifée avec l'efficacité du propriétaire courant.
>    Que font exactement GNU Backgammon (`eq2mwc`/`mwc2eq`, `Cl2CfMoney`,
>    `Cl2CfMatch`, les `cubeinfo` et le traitement de `fCubeOwner` dans
>    `eval.c`) et eXtreme Gammon (documentation publique, forum) ? Existe-t-il
>    une formulation publiée (Janowski 1993, Zare, Keith, Trice) qui fixe
>    l'efficacité **par état du videau après le coup** plutôt qu'à la racine ?
> 2. **Inversion en forme close.** Le prix « live » de Janowski est une
>    fonction linéaire par morceaux et monotone de la probabilité de gain,
>    avec 3 ou 4 segments dont les extrémités sont connues (points de prise,
>    de cash, de trop-bon selon l'état du videau et la MET). Mon port trouve
>    la probabilité seuil par 60 bissections, ce qui pèse 39 % d'une décision
>    au score. Recense les implémentations qui inversent ces segments en
>    forme close (gnubg, bgblitz, articles), les pièges numériques
>    (segments dégénérés, pentes nulles, égalité aux jonctions, arrondi
>    float32 vs float64 quand le résultat doit rester bit-identique à une
>    référence), et la façon de prouver l'équivalence (test de propriété
>    contre la bissection sur une grille).
>
> Livrable : réponse argumentée aux deux questions avec citations du code
> gnubg (fichier, fonction), formules, et un pseudo-code de l'inversion en
> forme close avec ses cas limites.

### P7 — Similarité de positions et recherche des k plus proches voisins en pur Go

> Je veux offrir « positions comme celle-ci » dans une base de 10 000 à
> 1 000 000 de positions de backgammon (Go, SQLite, sans dépendance native,
> binaire unique multi-plateforme). Deux représentations candidates : le
> vecteur brut de 26 points signés (ou 52 avec barre/sortis), et l'avant-
> dernière couche (128 float32) d'un réseau d'évaluation type TD-Gammon/gnubg
> déjà embarqué.
>
> Recherche approfondie et sourcée sur :
> 1. Les travaux qui utilisent l'**espace latent** d'un réseau d'évaluation de
>    jeu (backgammon, échecs NNUE, go) comme embedding pour du clustering ou
>    de la recherche de similarité, et comment ils évaluent que « proche
>    selon la métrique » = « proche selon un joueur » (protocoles, tailles
>    d'échantillon, accords mesurés).
> 2. Les métriques adaptées à un plateau discret : distance L1/L2 sur les
>    points, distance de transport (EMD/Wasserstein 1-D sur l'axe des points,
>    calculable en O(n)), similarité par motifs (points tenus, amorces), et
>    leurs invariances utiles (miroir des camps, symétries).
> 3. Les approches kNN en **pur Go** pour ces tailles : scan linéaire
>    (éventuellement SIMD), LSH, IVF, HNSW ; bibliothèques Go maintenues en
>    2026 (nom, licence, dernière release, état), et à partir de quelle
>    taille de corpus un index bat le scan.
> 4. Le stockage : garder les vecteurs dans SQLite (blob par ligne, extension
>    sqlite-vec — utilisable depuis modernc.org/sqlite sans cgo ?) ou dans un
>    fichier annexe reconstructible.
>
> Livrable : recommandation (métrique, index, stockage) par palier de taille,
> avec les chiffres de latence/mémoire trouvés et un protocole de validation
> humaine à 50 positions.

### P8 — Rollouts de backgammon : réduction de variance, troncature, critères d'arrêt

> Je vais ajouter des rollouts à un moteur de backgammon en Go (réseau
> prob5, recherche 0→2-ply, videau Janowski + MET, évaluation groupée SIMD,
> parallélisme sur les cœurs). Objectif : départager deux coups ou une
> décision de videau avec un intervalle de confiance affiché, sur un
> ordinateur portable, en quelques secondes à quelques minutes.
>
> Recherche approfondie et sourcée sur :
> 1. Les techniques de **réduction de variance** employées par GNU Backgammon
>    (`rollout.c` : variance reduction par « luck adjustment », dés en miroir/
>    stratifiés, quasi-aléatoire, appariement des lancers entre candidats),
>    eXtreme Gammon (documentation, forum), BGBlitz, et ce qui est publié
>    (Tesauro, Zadeh, articles bkgm.com) — avec les gains mesurés.
> 2. La **troncature** : nombre de coups avant évaluation terminale (XG et
>    gnubg utilisent quoi par défaut ?), effet sur le biais, cas de la course
>    (troncature vers une table de bearoff exacte).
> 3. Les **critères d'arrêt** : arrêt sur intervalle de confiance de la
>    différence entre les deux meilleurs candidats, nombre de parties
>    minimum/maximum, valeurs publiées (36, 108, 1296…), et la statistique
>    utilisée (écart-type de l'équité cubeful, intervalle joint).
> 4. La **reproductibilité** : graine, ordre de sommation, déterminisme sous
>    parallélisme (comment gnubg rend un rollout reproductible avec plusieurs
>    threads), et comment étiqueter un résultat de rollout (paramètres,
>    version du réseau) pour qu'il reste comparable.
> 5. Les pièges connus : rollouts cubeful vs cubeless, prise en compte du
>    videau pendant le rollout, Jacoby/beaver en money, Crawford.
>
> Livrable : synthèse avec les paramètres par défaut de gnubg et XG (sourcés),
> les formules d'intervalle de confiance, et une recommandation de
> paramètres pour trois profils (rapide 2 s / standard 30 s / précis 5 min).

### P9 — Formats de fichiers de match et de position du backgammon en 2026

> blunderDB importe XG (`.xg`, `.xgp`), GnuBG (`.sgf`), BGBlitz (`.bgf`),
> Jellyfish (`.mat`/`.txt`), et les positions collées (XGID, GNUbg ID, texte
> XG/GnuBG/BGBlitz). Je veux recenser ce qui manque et prioriser.
>
> Pour chacun des formats suivants — **OGXM / OGXM-JSON et OGID** (HedgeHog,
> OpenGammon), **Heroes** (heroesbackgammon), **Backgammon Galaxy**,
> **Backgammon Studio**, **GridGammon**, **BGRoom / Backgammon Room**,
> **Bgammon.org**, **GammonSpace**, le format « Backgammon NJ », et tout
> autre format ou export courant que tu trouverais (y compris les exports
> texte des sites en ligne) — donne :
> - la disponibilité et le lieu d'une **spécification** (URL, version,
>   licence), sa stabilité, la dernière modification ;
> - le **contenu** : coups seuls ou analyses (quel moteur, quelle
>   profondeur), chance/luck, marques d'étude, commentaires, horloges,
>   métadonnées de tournoi ; formats binaires vs texte ;
> - les **implémentations** de référence (langage, licence) et s'il existe un
>   parseur en Go ;
> - la **volumétrie d'usage** réelle (nombre de joueurs de la plateforme,
>   fréquence des demandes d'import dans les forums de gnubg/XG/bgblitz,
>   Discord, Reddit r/backgammon) ;
> - pour les exports **texte d'analyse XG** dans les langues autres que
>   FR/EN/JA/DE : les marqueurs de langue (ES, IT, RU, EL, FI, NL, PT) tels
>   qu'ils apparaissent dans un copier-coller depuis XG, avec un échantillon
>   par langue si tu en trouves.
>
> Livrable : tableau comparatif, ordre de priorité argumenté (valeur ×
> faisabilité), et pour OGXM/OGID un résumé de la spec (TLV, blocs
> d'analyse, signatures Ed25519, codec OGID) assez précis pour écrire un
> parseur.

### P10 — Pédagogie du backgammon et répétition espacée à note dérivée d'une mesure

> blunderDB propose la répétition espacée (FSRS) sur les positions où le
> joueur s'est trompé, et va ajouter des quiz où la réponse n'est pas
> « je me souviens / je ne me souviens pas » mais **un coup joué sur le
> plateau, dont l'erreur en équité est mesurée** contre l'analyse d'un moteur.
>
> Recherche approfondie et sourcée sur :
> 1. Les **méthodes d'entraînement** recommandées par les entraîneurs et
>    joueurs de haut niveau (revue de match, drills de positions de
>    référence, quiz chronométrés, comptage de pips, points de prise par
>    score, rollouts « à l'aveugle ») : livres, cours (Backgammon Galaxy,
>    bkgm.com, Backgammon Learning Center, Heroes Academy), podcasts ;
>    quelles preuves d'efficacité existent (études, témoignages chiffrés).
> 2. Les **indicateurs de progression** utilisés au-delà du PR : taux d'erreur
>    Snowie, PR par phase de partie, par score away, par type de décision,
>    « luck-adjusted result », et comment les outils (XG, gnubg, Galaxy,
>    Heroes) les présentent.
> 3. Les adaptations documentées de **FSRS/SM-2/Anki** pour des cartes à
>    réponse **graduée** (une décision plus ou moins coûteuse) : dériver la
>    note (Again/Hard/Good/Easy) d'une mesure continue (erreur en équité, ou
>    en millièmes de MWC), implémentations existantes (Chessable MoveTrainer,
>    Lichess Puzzle Storm/Streak, chess.com Puzzle Rush, apps de poker GTO),
>    et les pièges (cartes trop faciles jamais revues, biais de sélection).
> 4. Le **quiz** comme mesure : comment construire un « PR d'entraînement »
>    comparable au PR réel (même unité, même pondération checker/videau),
>    et les biais (positions choisies parce que ratées).
>
> Livrable : synthèse, recommandation d'un barème erreur → note FSRS avec
> justification, et une routine hebdomadaire type que la documentation
> pourrait proposer (avec sources).

### P11 — Compresser des dizaines de milliers de petits blobs JSON dans SQLite

> blunderDB stocke une analyse par position sous forme d'un JSON de 2 à 20 ko
> compressé en zlib niveau 9, un blob par ligne, dans SQLite (Go,
> modernc.org/sqlite, sans cgo) et dans PostgreSQL. Les blobs se ressemblent
> beaucoup (mêmes clés, mêmes structures, valeurs numériques). Une base réelle
> a 50 000 à 500 000 analyses ; la table `analysis` est la plus grosse.
>
> Recherche approfondie et sourcée sur :
> 1. Le gain mesuré d'un **dictionnaire partagé** (zlib `SetDictionary`,
>    zstd dictionary training) sur de petits JSON répétitifs : ordres de
>    grandeur publiés, taille de dictionnaire optimale, coût de la formation,
>    versionnage du dictionnaire (que faire quand on veut le changer sans
>    recompresser toute la base).
> 2. Les bibliothèques Go **pures** (sans cgo) pour zstd (klauspost/compress),
>    brotli, lz4, et leur support des dictionnaires ; vitesse de compression/
>    décompression en Go pur vs zlib standard ; stabilité de l'API en 2026.
> 3. Les alternatives structurelles : format binaire compact (CBOR, MessagePack,
>    flatbuffers), colonnes scalaires + blob réduit, compression par page
>    SQLite (extensions), et ce que ça change pour la recherche (les colonnes
>    filtrées sont déjà dénormalisées).
> 4. La **sécurité** : bombes de décompression avec dictionnaire, bornes à
>    poser, fuzzing des décodeurs.
> 5. Le chemin de migration : format de blob versionné (octet de tête),
>    recompression progressive en tâche de fond, `VACUUM`.
>
> Livrable : recommandation chiffrée (format, bibliothèque, dictionnaire ou
> non), protocole de mesure sur un échantillon de 10 000 blobs, et plan de
> migration compatible avec des bases ouvertes par d'anciennes versions.

### P12 — Diagrammes de backgammon en SVG/PNG : bibliothèques, conventions, impression

> Je veux un rendu unique du plateau de backgammon (position, dés, videau,
> score, pipcount, éventuellement flèches de coup) en SVG, réutilisé pour
> l'écran (aujourd'hui two.js), le presse-papier (PNG), l'export fichier, un
> rapport HTML imprimable et, plus tard, un client web.
>
> Recherche approfondie et sourcée sur :
> 1. Les **générateurs existants** de diagrammes à partir d'un XGID/GNUbg ID :
>    services web, scripts Python/JS, paquets LaTeX (bkgm, `backgammon` de
>    CTAN…), polices de diagramme, plugins ; licences et qualité.
> 2. Les **conventions** des livres et magazines (orientation, numérotation
>    des points, notation du score « X away », dés à côté du plateau, videau
>    dans le bord, coup joué en flèches) — avec exemples visuels décrits ;
>    et ce que font XG, gnubg, Galaxy pour leurs exports d'image.
> 3. Le **SVG** en pratique : polices embarquées vs converties en chemins,
>    dimensions pour l'impression (300 dpi), conversion SVG → PNG en Go pur
>    (bibliothèques maintenues en 2026, sans cgo, gestion du texte), et dans
>    le navigateur (canvas depuis SVG, `OffscreenCanvas`, presse-papier image).
> 4. L'accessibilité d'un diagramme (texte alternatif structuré, description
>    de la position lisible par un lecteur d'écran).
>
> Livrable : recommandation d'architecture (un module SVG source, dérivés
> PNG/écran), conventions retenues avec sources, et bibliothèques Go/JS
> retenues avec versions.

### P13 — Multi-tenant PostgreSQL avec RLS : identifiants, coût, migrations concurrentes

> Un démon Go (pgx/v5, pgxpool) sert plusieurs tenants derrière un reverse
> proxy qui injecte un en-tête `X-Tenant-ID`. Isolation par colonne
> `tenant_id BIGINT` sur toutes les tables + Row-Level Security avec
> `set_config('app.tenant_id', …)` à l'acquisition de la connexion et `RESET`
> à la libération. Migrations SQL numérotées appliquées au démarrage.
>
> Recherche approfondie et sourcée sur :
> 1. **Identifiants de tenant** : pratiques pour accepter un nom (chaîne)
>    côté API et le résoudre en entier côté base (table de correspondance,
>    hachage stable, UUID), et les pièges des conversions silencieuses
>    (`ParseInt` sans erreur ⇒ tenant 0).
> 2. **Coût du RLS** avec `set_config` à chaque acquisition : mesures
>    publiées, alternative `SET LOCAL` par transaction, `BeforeAcquire` vs
>    `PrepareConn` vs `AfterRelease` dans pgxpool v5 (sémantique exacte),
>    pooling par tenant, et l'impact de `RESET` sur les prepared statements.
> 3. **Migrations concurrentes** : `pg_advisory_lock` (clé, portée session
>    vs transaction), golang-migrate/tern/goose et leur gestion du verrou,
>    migrations transactionnelles et `CREATE INDEX CONCURRENTLY`, version de
>    schéma posée une fois en fin de chaîne.
> 4. **Clés étrangères composites** `(tenant_id, id)` : coût, bénéfice en
>    défense en profondeur, pratiques recommandées.
> 5. **Sauvegarde et restauration par tenant** : `pg_dump` filtré, export
>    logique, PITR ; et comment tester l'isolation (tests d'accès croisé,
>    fuzzing des en-têtes).
>
> Livrable : recommandations concrètes pour chaque point avec citations
> (docs PostgreSQL, pgx, articles), et un protocole de mesure du surcoût RLS.

### P14 — Chaîne d'approvisionnement d'un projet GitHub Actions en 2026

> Un projet open source Go + Node publie des binaires pour 4 plateformes, une
> image conteneur sur GHCR, des paquets `.deb`/`.rpm`/AUR/Flatpak, via GitHub
> Actions. Aujourd'hui : checksums `.sha256`, `govulncheck`, Dependabot
> (version updates), actions tierces épinglées par SHA mais pas les actions
> `actions/*`, token par défaut en écriture, pas de SBOM ni d'attestation.
>
> Recherche approfondie et sourcée sur l'état 2026 de :
> 1. **Attestations de provenance** (`actions/attest-build-provenance`, SLSA
>    niveau atteignable sur runners hébergés), SBOM (`syft`, `docker/build-push-action`
>    `sbom: true`), signature keyless **cosign/sigstore** des binaires et de
>    l'image, et vérification côté utilisateur (`gh attestation verify`,
>    `cosign verify`) — coût réel pour un mainteneur seul.
> 2. **Épinglage par SHA** : outils qui maintiennent les pins (Dependabot,
>    Renovate, `pin-github-action`, `ratchet`), et le réglage
>    `sha_pinning_required` du dépôt.
> 3. **Permissions** : `default_workflow_permissions: read`, `permissions:`
>    par job, séparation d'un job de release tag-only, secrets dans les
>    workflows déclenchés par tag, et les attaques récentes documentées
>    (tj-actions, changed-files, `pull_request_target`).
> 4. **Signature des tags et des checksums** sans certificat EV : `git tag -s`
>    (SSH ou GPG), minisign vs cosign, où publier la clé publique.
> 5. Ce qui est **spécifique aux applications non signées** Windows/macOS :
>    la valeur d'une attestation de provenance pour rassurer un utilisateur
>    face à SmartScreen/Gatekeeper, et les canaux (winget, Homebrew) qui
>    contournent le mieux ces avertissements.
>
> Livrable : liste ordonnée par rapport bénéfice/effort, avec les extraits
> YAML nécessaires et les commandes de vérification à documenter.

### P15 — Svelte 5 en production : locales paresseuses, tokens de couleur, thème sombre, tables et onglets accessibles

> Application de bureau Svelte 5 (runes) + Vite 7 dans une WebView (Wails).
> Aujourd'hui : 9 langues importées statiquement (65 % du bundle), 108
> couleurs hexadécimales en dur sans token, pas de thème sombre, des tableaux
> triables et des onglets faits maison, un `focusTrap` maison, Tab confisqué
> globalement pour un raccourci.
>
> Recherche approfondie et sourcée sur :
> 1. **Chargement paresseux des locales** : `import.meta.glob` + `await import`,
>    découpage par `manualChunks`, repli synchrone pour la langue par défaut,
>    et les pièges avec un store de langue Svelte 5 (`$state` global, effet au
>    changement, chargement de l'aide HTML à la demande).
> 2. **Tokens de couleur et thème sombre** : conventions de nommage
>    (sémantiques vs primitives), contraste WCAG AA pour le texte secondaire
>    à 11 px, `prefers-color-scheme` + préférence utilisateur, thèmes « contraste
>    élevé » et « imprimable », gestion d'un canvas (two.js) qui doit lire les
>    tokens, et outils qui **vérifient** qu'aucune couleur en dur ne subsiste
>    (stylelint, tests).
> 3. **Accessibilité** : patterns ARIA APG pour *tabs* (avec `aria-controls`),
>    *grid/table* triable avec sélection de ligne au clavier (`aria-selected`,
>    tabindex roving), *combobox* d'autocomplétion, `aria-live` bien délimité,
>    focus trap qui ignore les éléments invisibles ; ce que `svelte-check` et
>    `eslint-plugin-svelte` savent détecter en 2026, et comment faire échouer
>    un build sur les warnings a11y du compilateur Svelte.
> 4. **Raccourcis clavier vs navigation** : bonnes pratiques quand une
>    application confisque Tab (limiter à une zone, offrir Ctrl+Tab), et
>    exemples d'applications de bureau web qui le font proprement.
>
> Livrable : recommandations avec extraits Svelte 5 idiomatiques, liste
> d'outils/versions, et un budget de warnings à mettre en CI.

### P16 — Distribuer une application de bureau Wails non signée : Flathub, mise à jour, associations de fichiers

> Application Wails v2 (Go + WebView) publiée pour Linux (`.deb`, `.rpm`, AUR,
> tarball, Flatpak bundle), macOS (`.app` universel non notarisé) et Windows
> (`.exe` non signé). Manifestes winget et cask Homebrew rendus à chaque
> release mais pas encore soumis ; pas de tap Homebrew ; pas de vérification
> de version dans l'application ; pas d'association de fichiers.
>
> Recherche approfondie et sourcée sur :
> 1. **Flathub** : exigences 2026 (build from source hors ligne, sources
>    vendored Go et npm avec `flatpak-go-mod`/`flatpak-node-generator`,
>    `<screenshots>` et `<branding>` obligatoires dans le metainfo, runtime
>    GNOME/KDE et webkit2gtk-4.1), procédure de soumission, délai, et les
>    exemples de projets Wails déjà sur Flathub.
> 2. **winget** et **Homebrew** : soumission à `microsoft/winget-pkgs`
>    (validation, `wingetcreate`, mise à jour automatique par action), tap
>    personnel vs `homebrew/cask` (seuil de notoriété), et la gestion de
>    l'avertissement Gatekeeper dans un cask (`xattr -d com.apple.quarantine`
>    est-il accepté ?).
> 3. **Vérification de version** dans une application de bureau : appel à
>    l'API GitHub Releases (limites, cache, opt-in), détection du canal
>    d'installation pour ne pas proposer une mise à jour manuelle à un
>    utilisateur de paquet, et frameworks de mise à jour sans signature de
>    code (ce qui est raisonnable et ce qui ne l'est pas).
> 4. **Associations de fichiers** : `MimeType`/`shared-mime-info` sur Linux,
>    `CFBundleDocumentTypes` sur macOS, registre Windows via l'installeur
>    NSIS de Wails ; et comment Wails v2 transmet le fichier ouvert
>    (argument, événement `OnFileOpen` sur macOS).
> 5. **Binaires Linux arm64** : runners `ubuntu-24.04-arm`, webkit2gtk-4.1
>    disponible, coût.
>
> Livrable : plan de soumission par canal (étapes, fichiers, pièges), et
> extraits de configuration.

### P17 — Lier un filigrane signé au contenu d'un fichier SQLite, et authentifier l'en-tête d'un conteneur AEAD

> blunderDB marque une base SQLite exportée avec un filigrane signé (Ed25519 :
> version, origine, nom d'émetteur, note, date) stocké dans une table
> `metadata`, et peut envelopper l'export dans un conteneur chiffré
> (AES-256-GCM, clé dérivée par Argon2id, en-tête clair : version, filigrane,
> sel, nonce ; l'en-tête n'est pas passé en AAD). Problème : la signature ne
> couvre aucun digest du contenu, donc un filigrane valide est transférable
> sur une autre base.
>
> Recherche approfondie et sourcée sur :
> 1. **Digest canonique d'une base SQLite** : comment calculer un hash du
>    contenu logique (tables, lignes ordonnées) indépendant de la
>    représentation physique (pages libres, ordre d'insertion, `VACUUM`) ;
>    outils existants (`sqlite3 .sha3sum`, `dbhash`), coût sur 500 Mo, et
>    l'alternative « signer l'export au moment où il est produit » (digest du
>    fichier tel qu'écrit, réputé immuable).
> 2. Le **schéma de signature** : signer (digest ‖ métadonnées) en un seul
>    document canonique (JSON canonique RFC 8785, ou COSE/PASETO), gestion
>    des versions de format, et vérification quand le destinataire modifie
>    la base ensuite (le filigrane reste-t-il « valide pour la version
>    d'origine » ?).
> 3. **AEAD et en-tête** : passer l'en-tête en `additionalData`, migration de
>    format (v1 sans AAD lue, v2 écrite), stockage des paramètres Argon2id
>    dans l'en-tête, bornes de taille avant lecture, déchiffrement en flux
>    (chunking AEAD, par exemple STREAM/Tink) pour ne pas charger 2 Go en
>    mémoire.
> 4. Les précédents : Anki shared decks, Chessable, paquets signés Sigstore
>    bundle, minisign, age — ce qui est réutilisable tel quel.
>
> Livrable : conception recommandée (format, canonicalisation, versions),
> risques résiduels, et pseudo-code Go des chemins signer/vérifier/
> envelopper/ouvrir.

### P18 — Comment les outils d'analyse d'échecs et de poker catégorisent les erreurs d'un joueur

> Pour un outil d'analyse d'erreurs au backgammon, je veux savoir ce que
> l'écosystème des échecs et du poker a appris sur la **catégorisation
> automatique des erreurs** et sa présentation à l'utilisateur.
>
> Recherche approfondie et sourcée sur :
> 1. **Lichess** (Insights, analyse de partie, « Learn from your mistakes »),
>    **Chess.com** (Game Review : Brilliant/Great/Best/Mistake/Blunder/Miss,
>    « Game Report » par phase, Insights), **Chessable/Aimchess/DecodeChess**
>    (explications textuelles), **ChessBase** : quelles catégories, quels
>    seuils (centipawns, probabilité de gain), quelle taxonomie par phase
>    (ouverture/milieu/finale) et par thème tactique/stratégique, et comment
>    c'est calculé (règles, modèles).
> 2. **Poker** : GTO Wizard, PokerSnowie, PioSolver, Poker Copilot — « EV
>    loss » par rue, par type de décision, par position ; « leak finder » et
>    regroupement d'erreurs récurrentes ; présentation (heatmaps, ranges).
> 3. Les **explications en une phrase** générées automatiquement (sans LLM) :
>    quelles règles produisent des phrases fiables, quel taux d'erreur est
>    toléré, comment on se tait quand la règle n'est pas confiante.
> 4. Les leçons d'**interface** : vocabulaire, échelle de gravité, drill-down
>    vers les positions, ce qui a été abandonné et pourquoi (retours
>    utilisateurs, billets de blog des équipes).
>
> Livrable : synthèse comparative, taxonomie transposable au backgammon
> (5-6 thèmes d'erreur défendables avec leur critère mesurable), et les
> anti-patterns à éviter.
