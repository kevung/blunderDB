# Plan « grosses bases » — diagnostic du 2026-10-03

Audit mené le 2026-10-03 sur `main` @ 1fa74f61f par trois passes indépendantes
(stockage/import, frontend/UX, fonctionnel/parité), chacune vérifiant le code avant de
signaler, plus des mesures faites ce jour sur ce poste (Ryzen 6850U, 16 threads, 14 Go).
Cible : un corpus BMAB de 21 Go de fichiers `.xg`, avec doublons. Les chiffres sont à
re-mesurer avant d'être cités ailleurs. Ce plan ne redit pas `tasks/BACKLOG.md` : il dit
quels items y deviennent bloquants.

Format : `[S]` ≤ ½ journée, `[M]` ≤ 2-3 jours, `[L]` chantier. Une fiche = une branche =
un worktree ; méthode `traiter-lot` (ADR-0055) ; toute fiche visible par l'utilisateur
embarque sa doc et ses `.po`.

## 1. Mesures du jour

| Mesure | Valeur | Source |
|---|---|---|
| Import CLI, 154 `.xg` (20 Mo, base neuve) | 477 s, 58 428 positions, **≈ 120 positions/s**, ≈ 3 s/match | `blunderdb import --type batch` |
| Import d'un seul fichier (465 positions) | 1,8 s, dont démarrage du processus | idem, `--type match` |
| Réimport d'un fichier déjà présent | 0,06 s (empreinte de match avant toute écriture) | idem |
| Pic mémoire du processus d'import | 313 Mo | `/proc/<pid>/status` |
| Taille de base | 45,7 Mo, **≈ 780 o/position** : `analysis` 310 o, `position` 135 o, `move` 40 o, index ≈ 35 % | `dbstat` |
| Profil CPU de `BenchmarkImport_SingleXG` | **81 % dans `engine.EncodeAnalysisForStorage`**, dont 77 % de `memmove` dans `zstd.Reset` (dictionnaire recopié à chaque `EncodeAll`) ; SQL ≈ 10 % | pprof |
| Codec zstd avec dictionnaire, 300 blobs réels (JSON moyen 1 993 o) | niveau 19 (actuel) : 277 o, **5,97 ms** ; niveau 7 : 304 o, 0,31 ms ; niveau 3 : 328 o, 0,08 ms ; décodage 0,018 ms | `engine/analysiscodec.go:64` |
| Fichier `.xg` du corpus local | médiane 118 Ko, moyenne 110 Ko, ≈ 380 positions par fichier | `testdata/`, `~/bkgm` |

## 2. Extrapolation à 21 Go

| Grandeur | Estimation | Hypothèse |
|---|---|---|
| Fichiers | ≈ 190 000 | 110 Ko par fichier |
| Positions brutes | ≈ 72 M | 380 par fichier |
| Positions uniques | **40 à 70 M** | 30 à 50 % de fichiers en doublon ; la dédup Zobrist entre matchs distincts ne joue que sur les ouvertures |
| Durée d'import au débit actuel | **90 à 170 h** (4 à 7 jours), un seul cœur | 120 positions/s |
| Taille de base | **30 à 55 Go**, dont 10 à 20 Go d'index | 780 o/position |
| Ouverture de la GUI | 40 à 70 M d'ids (≈ 500 Mo de JSON) traversent Wails ; la session en resérialise autant 500 ms après **chaque** navigation | `positionService.js:273`, `sessionService.js:21-31`, `App.svelte:164-168` |

Verdict : le moteur de stockage (SQLite, WAL) tient ce volume ; c'est le **chemin d'import**
(CPU) et le **modèle de données de la GUI** (tout en mémoire) qui ne tiennent pas. Aucun des
deux ne demande de changer de base : PostgreSQL ne résout ni l'un ni l'autre (même
`WriteMatch` ligne à ligne ; la GUI est typée `*sqlite.Storage`, `database/db.go:30`, 142
appels SQL directs).

## 3. Diagnostic

### 3.1 Import — pourquoi 120 positions/s

1. **zstd niveau 19 à chaque écriture** (`engine/analysiscodec.go:64`), 1 à 2 fois par
   position (un encodage par fragment d'analyse, `ingest/match.go:238-280`), et
   `mergeAnalysis` pose `LastModifiedDate = time.Now()` (`ingest/merge.go:133`) : une
   analyse identique est relue, décompressée, fusionnée, recompressée, réécrite. ADR-0030
   supposait un chemin « écrit une fois » : c'est faux dès qu'une position revient.
2. **6 à 9 instructions SQL par position, aucune préparée** (`positions_sqlite.go:122-169`,
   `analyses_sqlite.go:52-100`) ; `PlayedActionsFor` lance presque toujours deux
   sous-requêtes sur `move` (`:206-208`), ≈ 7 000 lignes pour une ouverture partagée.
3. **Séquentiel** : un fichier à la fois, sous `d.mu.Lock` (`db_import_xg.go:17-35`), en
   GUI (`importService.js:781`) comme en CLI (`cli_import.go:450`). 15 cœurs dorment.
4. **≈ 30 B-trees mis à jour par position** : 16 index sur `position`, 10 sur `analysis`
   (`schema_sqlite.go:513-553`), index Zobrist à clés aléatoires, `cache_size` 64 Mo pour
   des index qui pèseront 10 Go. Une fois le CPU réglé, c'est la limite suivante (E/S).
5. **`ANALYZE` complet après chaque lot** (`cli_import.go:506`, `importService.js:813`),
   sans `analysis_limit`.
6. **Bug annexe** : sur un doublon exact, `applyFlags` relève les marques XG puis
   `writeImportedMatch` fait `Rollback` (`db_import_common.go:50-54`) ; ADR-0006 promet
   l'inverse. Le manuel (`manuel.rst:2393`) dit de « réimporter » pour la chance : impossible
   sans supprimer le match.

### 3.2 GUI — pourquoi elle ne s'ouvrira pas

1. `ListPositionIDs` sans borne à l'ouverture, après chaque import, à la sortie d'une
   collection, sur Ctrl+R (`positions_sqlite.go:283-315`, `modeMachine.js:785`) ;
   `indexOf` construit une `Map` sur tous les ids (`positionList.js:241-249`).
2. **Session** : `lastPositionIds = positionsStore.ids` sérialisé à chaque changement de
   position (`sessionService.js:21-31`) ; `viewStore.serialize` y ajoute les ids de chaque
   vue. Premier bloquant, et le plus simple.
3. **Recherche** : `Find` avec `ListOpts{}` vide (`db_search.go:72-85`) ; `scanRows`
   matérialise toutes les lignes en `domain.Position` reconstruites, blob décodé si un
   filtre Go l'exige (`sqlshared/search.go:449-505`), pour ne renvoyer que des ids ; les
   filtres `MoveErrorFilter`/`DateFilter` sont réglés en Go après chargement (`:79-84`,
   `:619-641`). Aucun retour visuel, aucune annulation côté GUI alors que
   `LoadPositionsByFiltersCoreCtx` existe.
4. **Listes** : `PanelTable` rend `{#each rows}` complet (`PanelTable.svelte:170`) pour six
   panneaux ; `GetAllMatches` sans LIMIT, rechargé à chaque édition en ligne et à chaque
   affichage (`MatchPanel.svelte:90,105,678`), doublé par `TournamentPanel.svelte:186` ;
   `$state([])` profond, aucun `$state.raw` dans `src/`. 50 000 matchs = 500 000 nœuds DOM.
5. **Collections et decks Anki en positions complètes** (`modeMachine.js:768`,
   `ankiService.js:324,375`), hors du modèle par fenêtre.
6. **Trois `COUNT(*)`** à chaque mutation (`StatusBar.svelte:80`, `db_match.go:459-475`),
   sous `RLock` : 1 à 3 s sur 30 M de lignes.
7. **Stats** : ≈ 10 agrégats avec `COUNT(DISTINCT)` sur la jointure complète
   (`stats_compute.go:75-90`) ; `stats_recurring` et `move_grades` décodent les blobs.
8. **Maintenance** : `positionIDsWithStaleGammonNet` fait un `LoadAnalysis` par ligne
   (`db_gammonnet_batch.go:80-102`) ; `VACUUM` avec `temp_store=MEMORY` risque de copier la
   base en RAM (`vacuum_sqlite.go:45-90`, à vérifier) ; le contrôle préalable ne regarde que
   le disque.

### 3.3 Fonctionnel — ce qui manque pour un corpus BMAB

1. **Une réanalyse plus profonde du même match est jetée** : `MatchHash` couvre noms,
   longueur, dés, coups, videau, jamais l'analyse (`ingest/xg.go:568-590`) ; la fusion par
   `AnalysisDepthRank` ne joue que sur le chemin inter-formats (`merge.go:48`). Entre une
   version 3-ply et une Roller++, l'ordre d'import décide.
2. **Doublons sous deux graphies** (« Unger K. » / « Kevin Unger ») : les deux empreintes
   incluent les noms (`xg.go:571-573`, `canonical.go:24-29`) ; le match compte deux fois
   dans la table Joueurs. `MergePlayers` réécrit une fois (`matches_sqlite.go:506-535`) et
   l'import suivant ramène l'ancienne graphie ; `StatsFilter.PlayerAliases` existe mais
   personne ne le renseigne (`stats.go:8-12`). Même problème pour les tournois (`Event`
   brut, `ingest/match.go:168`).
3. **Pas de journal fichier → match** : `MatchesSkipped` est un compteur
   (`importbatch.go:40`) ; une reprise re-parse tout.
4. **Recherche** : pas de longueur de match, pas de date du match (`T` lit
   `analysis.CreationDate`, posé à `time.Now()` à l'import, `xgmap.go:688` ; `cmd_mode.rst:216`
   dit autre chose), tournoi par id seulement, ni ronde, ni adversaire, ni « décisions du
   seul joueur X » (`pl` prend les deux sièges, `search.go:843-850`), ni PR du match, ni
   moteur/profondeur (ADR-0013 le reconnaît).
5. **PR mélangeant moteurs et profondeurs** sans filtre de provenance (`statsErrExpr`).
6. **Métadonnées XG ignorées** : Elo, expérience, transcripteur, commentaires d'en-tête,
   horloge, Jacoby/Beaver, version XG/moteur (`xg.go:41-48`, `xglight.go:41-43`).
7. **Parité** : rollout en CLI seulement (pris par J.2 du plan 2026-10b) ; fusion/inversion
   de joueurs absente de la CLI ; import de dossier absent du serveur ; rapport HTML des
   stats en GUI seulement.
8. **Doc** : `manuel.rst:1031` dit que blunderDB ne sait pas produire de rollout ;
   `CLI_USAGE.md:31-55` omet `stats`, `rollout`, `tournament`, `bearoff`, `cubematrix`,
   `trash`, `repair`.

### 3.4 UX, hors échelle

- L'import surveillé **chasse l'utilisateur de son contexte** : `importWatchedFiles` passe
  par `reloadPositions()` (`importService.js:822`), qui remet NORMAL, vide le contexte de
  match, efface la recherche et saute à la dernière position (`positionService.js:280-306`).
- **Course sur l'analyse** : `showPosition` écrit `analysisStore` sans vérifier que la
  position est encore la même (`positionService.js:175-222`) ; touche maintenue en MATCH =
  analyse de N−1 sur le plateau N. À confirmer par un test.
- Import long : modale bloquante, progression en fichiers seulement, pas d'ETA, annulation
  entre deux fichiers, doublon déduit d'une sous-chaîne anglaise (`importService.js:793`),
  liste d'erreurs copiée à chaque erreur (O(n²), `:797-801`).
- 50 000 matchs sans champ de filtre dans MatchPanel ; PageUp/Down = première/dernière.
- `window.confirm` (8 occurrences) à côté de `confirmAction` ; dialogue anglais en dur
  (`importService.js:1034`) ; 13 écouteurs `keydown` sur `document`/`window`.
- Bundle : chunk principal 1,0 Mo, aide `en` et `DirectionView` importées statiquement.
- Tests : MatchPanel 20 %, CollectionPanel 16 %, positionService 30 %, importService 40 % ;
  aucune fixture d'échelle ; e2e sans import de dossier, collections, tournois, Anki complet.

### 3.5 Parcours d'étude, club, serveur (seconde passe fonctionnelle)

- **La boucle d'apprentissage ne se ferme pas** : `training_item` n'a pas de `position_id`
  (`schema_sqlite.go:322-332`), une question ratée au quiz ne laisse aucune trace exploitable ;
  les erreurs récurrentes n'ont pas de « m'entraîner sur ce groupe » ; la file d'étude ne couvre
  qu'un lot d'import et n'a pas de mémoire ; l'explication n'est ni au dos de la carte Anki ni
  dans le verdict du quiz ; le journal d'entraînement n'entre pas dans Stats. Les outils de quiz
  du marché (Backgammon Mastery, OpenGammon Insights, Backgammon Studio) font cette boucle
  automatiquement ; blunderDB a toutes les briques.
- **Un commentaire n'a pas d'auteur** et la GUI n'en montre qu'un par position.
- **Club** : `Ranking` ne couvre qu'un tournoi ; pas de classement de saison ni d'export fédéral ;
  l'écran joueur et les deux postes sont portés par #380/#457 et gammonGo.
- **Serveur** : pas de route `/v1/training`, pas d'outil MCP d'évaluation, de révision Anki ou de
  transcription ; quotas limités au débit ; aucune recette de proxy authentifiant.
- **À préserver** : base locale sur son propre corpus, multi-moteurs avec bande de désaccord,
  grammaire de recherche et `like`, FSRS honnête, PR de quiz sur l'échelle réelle,
  transcription et direction sans équivalent, rien n'est envoyé ailleurs (ADR-0007).

### 3.6 UX en situation (captures Playwright sur le mock Wails, 2026-10-03)

Vu à l'écran : libellés du plateau rognés et non traduits (« ip: 158 », « crawforc »,
`boardScene.js:386-525`) ; changer d'onglet remplace le plateau par un plateau vide sans le dire ;
le panneau Transcription mange le bas du plateau ; une recherche sans résultat ne se voit pas ;
états vides sans action ; chaînes anglaises (« No double, take », « New », « Dashboard ») ;
vocabulaire pions/videau vs coup/cube incohérent entre panneaux ; 25 icônes sans libellé ;
modale Paramètres qui déborde ; Stats étouffé en 280 px ; contrastes du thème sombre. Bien :
l'accueil, la table des coups, la barre d'info, le menu contextuel, la modale d'import.
Non vérifié : le rendu WebKitGTK réel.

## 4. Objectifs chiffrés

| Indicateur | Aujourd'hui | Cible |
|---|---|---|
| Débit d'import (base de plusieurs millions) | 120 pos/s | **≥ 2 000 pos/s** (21 Go en moins d'une nuit) |
| Ouverture d'une base de 50 M de positions | impossible | < 2 s, indépendant de la taille |
| Première page d'une recherche indexée | O(N) mémoire | < 1 s, mémoire bornée |
| Taille par position | 780 o | ≤ 700 o après élagage d'index (le blob +10 % est accepté) |
| Réanalyse plus profonde d'un match déjà présent | perdue | fusionnée par profondeur |

## 5. Lots, dans l'ordre

### Lot 0 — mesurer avant de toucher `[S]`
- **Échantillon BMAB** : 2 000 fichiers pris au hasard (≈ 250 Mo) suffisent à mesurer le taux
  de doublons de fichiers, la dédup Zobrist réelle, la variété des graphies de noms et
  d'événements, et le débit à 1 M de positions. Le corpus entier sert à la validation finale.
- **Générateur de base synthétique** (`cmd/blunderdb-loadtest` ou nouveau `cmd/`) : N matchs
  tirés des fixtures avec noms et dés variés → base de 1 M puis 10 M de positions, pour les
  benchmarks de recherche, stats et GUI. Publié dans `nightly.yml` avec seuils.
- `BenchmarkImport_SingleXG` et `BenchmarkSearch_*` sur cette base, chiffres dans
  `tasks/bench/`.

### Lot 1 — le CPU de l'import `[S à M]`, gain attendu ×8 à ×10
1. `[S]` zstd niveau 7 (`SpeedBetterCompression`) avec le même dictionnaire ; recompression
   en 19 déplacée dans `vacuum` (qui recompresse déjà les anciens blobs). Pas de bump : le
   blob dit son codec. Amender ADR-0030.
2. `[S-M]` fusionner les fragments d'une position en mémoire : un seul encodage, un seul
   UPSERT ; sauter l'écriture si l'analyse fusionnée est identique hors `LastModifiedDate`.
   Garder l'ordre « arrondi puis recalcul » de `savePositionWithAnalyses`.
3. `[M]` instructions préparées par transaction (insert position, upsert analyse, insert
   move) ; `PlayedActionsFor` nourri par le graphe quand il connaît déjà le coup joué.
   Parité SQLite/PostgreSQL via le contrat.
4. `[S]` `PRAGMA analysis_limit=1000` + `optimize` au lieu de l'`ANALYZE` complet.
5. `[S]` corriger le `Rollback` qui annule `applyFlags` (ADR-0006) ; corriger
   `manuel.rst:2393` et `CLI_USAGE.md:582` en conséquence.

### Lot 2 — le pipeline d'import `[M]`, gain attendu ×3 à ×5 supplémentaire
1. Parse + mappage + compression sur N workers, **un seul écrivain** ; plusieurs fichiers par
   transaction ; annulation par contexte à l'intérieur d'un fichier. `d.mu` reste exclusif
   pendant l'écriture seulement.
2. **Mode « import en masse »** quand le lot dépasse un seuil : `cache_size` ≥ 512 Mo,
   `wal_autocheckpoint` élevé, `synchronous=OFF` le temps du lot (base neuve seulement),
   index secondaires supprimés puis recréés à la fin. Une coupure laisse des index absents :
   réparation à l'ouverture (comme `repairPositionsWithoutScalars`).
3. Progression par événement Wails (matchs, positions, octets, débit, ETA), modale
   réductible en barre de statut, doublon signalé par une erreur typée.
4. Journal d'import : pour chaque fichier, le match qui le couvre (neuf, doublon, enrichi,
   erreur) ; reprise d'un lot qui saute ce qui a déjà été vu. La colonne d'empreinte de
   fichier attend le bump du lot 4.
5. Même pipeline en CLI (`import --type batch`) et sur le serveur (`imports.batch` absent
   aujourd'hui).

### Lot 3 — la GUI à l'échelle `[L]`, bloquant avant tout import réel
1. `[S]` **Session** : persister un descripteur (commande de recherche, vue, id courant) et
   rejouer à la restauration, jamais la liste d'ids. Test : payload de
   `SaveSessionState` borné.
2. `[S]` jeton de génération dans `showPosition` ; test de réponses désordonnées.
3. `[L]` **Liste paginée** : contrat `CountPositions`, `ListPositionIDs(ListOpts)`,
   `IndexOfPosition(id)` ; `positionList` devient « longueur + fenêtres d'ids » ; même
   contrat pour la recherche (BACKLOG B.10) qui s'arrête dès que la page est pleine et ne
   reconstruit plus de `domain.Position` pour rendre des ids. Les filtres Go restants
   (`MoveErrorFilter`, `DateFilter`) descendent en SQL. Parité CLI/serveur (`ListOpts`
   existe déjà).
4. `[M]` indicateur « recherche en cours », Échap annule via `…CoreCtx`, une seule recherche
   à la fois.
5. `[M]` `PanelTable` virtualisé (hauteur fixe, fenêtre + tampon), six panneaux (BACKLOG D.8).
6. `[M]` MatchPanel/TournamentPanel : un seul magasin de matchs, `$state.raw`, mise à jour
   ciblée de la ligne éditée, **filtre texte (joueur, tournoi, date) et tri en SQL**, pagination.
7. `[M]` collections et decks Anki en ids et fenêtres.
8. `[M]` compteurs de la barre de statut tenus à jour à l'insert (table de compteurs) ou
   estimés (`sqlite_stat1`), plus jamais trois `COUNT(*)`.

### Lot 4 — un seul bump de schéma, groupé `[M, Opus]`
Un bump = `DatabaseVersion`, `migrate_X_to_Y`, DDL triple, `migrations/` PostgreSQL, test,
base de démo. Grouper tout ce qui en a besoin :
1. moteur et profondeur sortis du blob dans `analysis` (BACKLOG B.12) → filtre de
   provenance des stats, recherche par profondeur, `positionIDsWithStaleGammonNet` par
   index ;
2. `match_date` dénormalisée sur `position` (ou `analysis`) pour que `T` filtre la date du
   match, et `creation_date` promue + index (BACKLOG) ;
3. empreinte de fichier (SHA-256, taille, mtime) et journal d'import (lot 2.4) ;
4. table `player_alias` (lot 5.2) ;
5. élagage des index peu sélectifs (`game_phase`, `game_type`, `dice`, `off`,
   `back_checkers_*`, `pip_1`, `*_2` d'`analysis`), décidé sur EXPLAIN contre la base
   synthétique de 10 M du lot 0 ; `EnsureSchema` recrée par nom, les supprimer explicitement.

### Lot 5 — le corpus comme objet d'étude `[S à M chacun]`
1. `[S-M]` **réanalyse plus profonde** : sur doublon exact, passer par la branche `enrich`
   avec fusion par `AnalysisDepthRank` (option `--merge-analysis`, défaut à trancher).
   `mergeCheckerMoves` existe.
2. `[M]` **alias de joueurs** : table appliquée à l'import et aux stats (`PlayerAliases`
   est déjà câblé), GUI de fusion persistante, CLI `players merge`. Même mécanisme pour les
   événements.
3. `[M]` **empreinte sans les noms** (longueur + suite de dés) : signaler un match déjà
   présent sous une autre graphie, proposer sans fusionner.
4. `[S chacun]` jetons de recherche : longueur de match, date du match, tournoi par nom,
   ronde, adversaire, décisions du seul joueur X (avec joker), PR du match ; `[M]`
   moteur/profondeur (après lot 4).
5. `[S-M]` métadonnées XG : Elo, expérience, transcripteur, commentaires d'en-tête, dans
   `xgparser` d'abord (amont, même auteur).
6. `[M]` **stats de corpus** : face-à-face, PR par fenêtre calendaire glissante, classement
   (Elo ou rang par PR), erreurs récurrentes croisées entre joueurs ; **stats par match
   matérialisées** (PR, nombre de décisions, chance) dans une table tenue à l'import, pour
   que l'onglet Stats ne rescanne plus 50 M de lignes.
7. `[S]` doc : `manuel.rst:1031` (rollout), `cmd_mode.rst:216` (sémantique de `T`), liste
   des commandes de `CLI_USAGE.md`, collections vivantes dans la bonne section.

### Lot 6 — UX hors échelle `[S chacun]`
Import surveillé qui ne touche ni mode, ni recherche, ni plateau ; `confirmAction` partout ;
dialogue `importService.js:1034` traduit ; Home/End et PageUp/Down par pas de N ; aide `en`
et `DirectionView` en `import()` dynamique ; un seul dispatch clavier ; `raccourcis.rst`
alignés. Tests : import surveillé, MatchPanel à 50 000 lignes sous budget de temps, e2e
import de dossier et collections.

### Lot 7 — maintenance à pleine échelle `[S à M]`
`VACUUM` : vérifier `temp_store=MEMORY`, sinon `temp_store=FILE` le temps du VACUUM ou
`VACUUM INTO` ; contrôle préalable sur la RAM. `positionIDsWithStaleGammonNet` par lots
puis par colonne (lot 4). Parité : fusion/inversion de joueurs en CLI, rapport HTML des
stats en CLI/serveur, import de dossier sur le serveur.

### Lot 8 — UX en situation `[M]`
Chantiers C13 (plateau, mise en page, thème) et C14 (libellés, vocabulaire, états vides,
gestes) ; détail dans les issues.

### Lot 9 — le parcours d'étude, le club, le serveur `[M]`
C11 : explication au dos des cartes et au quiz, quiz depuis les erreurs récurrentes,
`training_item.position_id` (bump C6), file d'étude transversale, journal d'entraînement dans
Stats, carte Anki jouée au damier en option, commentaires avec auteur. C12 : classement de
saison et export, routes `/v1/training` et outils MCP manquants, quotas et recette de proxy.

### Hors plan, explicitement
- **PostgreSQL pour la GUI** : `[L]`, sans gain sur les deux goulots. À reconsidérer
  seulement si un corpus partagé entre plusieurs postes devient le besoin.
- **Import à seuil** (ne garder que les décisions dont l'erreur dépasse X, ou seulement les
  décisions de videau) : diviserait le volume par 5 à 10, mais rompt « une position par
  hash » comme matière d'étude. Décision produit, pas technique ; à trancher avant le lot 2.

## 6. Ordre et dépendances

```
Lot 0 ─┬─ Lot 1 ─ Lot 2 ──┐
       └─ Lot 3 (3.1, 3.2 immédiats) ─┤
                                      ├─ Lot 4 (un bump) ─ Lot 5 ─ Lot 7
Lot 6 (indépendant, en parallèle) ────┘
```

Jalon 1 (lots 1 + 3.1 + 3.2) : importer l'échantillon de 2 000 fichiers en moins de
10 minutes et rouvrir la base sans gel. Jalon 2 (lots 2 + 3) : le corpus entier en une
nuit, GUI indifférente à la taille. Jalon 3 (lots 4 + 5) : le corpus devient un objet
d'étude (alias, profondeur, stats de corpus).

Articulation avec `tasks/plan-2026-10b.md` : ce plan démarre après la 0.37.0 ; ses lots 1 et
3.1-3.2 peuvent y entrer s'ils sont prêts, le bump du lot 4 doit se coordonner avec celui
que J.2 étape 2 pourrait demander.

## 7. Issues GitHub

Jalon « Grosses bases — corpus BMAB », label `chantier` + `lot:GBn`. Le 2026-10-03, les 39 fiches ont d'abord été ouvertes une par une (#478-#516), puis **regroupées en 14 chantiers** : une zone de code, des étapes qui s'enchaînent dans un seul worktree, une revue Opus à la fin. Les fiches fines sont fermées et renvoient vers leur chantier. Suivi : #517.

| Chantier | Issue | Étapes (fiches) |
|---|---|---|
| GB-C1 — Mesurer avant de toucher [S] | #518 | GB0.1 (#478), GB0.2 (#479), GB0.3 (#480) |
| GB-C2 — Import : le CPU [M] | #519 | GB1.1 (#481), GB1.2 (#482), GB1.3 (#483), GB1.4 (#484), GB1.5 (#485) |
| GB-C3 — Import : le pipeline [M] | #520 | GB2.1 (#486), GB2.2 (#487), GB2.3 (#488), GB2.4 (#489), GB2.5 (#490) |
| GB-C4 — GUI à l'échelle : liste et recherche [L] | #521 | GB3.1 (#491), GB3.2 (#492), GB3.3 (#493), GB3.4 (#494), GB3.5 (#495) |
| GB-C5 — GUI à l'échelle : panneaux et compteurs [M] | #522 | GB3.6 (#496), GB3.7 (#497), GB3.8 (#498), GB3.9 (#499) |
| GB-C6 — Un seul bump de schéma, groupé [M] | #523 | GB4.2 (#501), GB4.1 (#500) |
| GB-C7 — Le corpus : doublons, alias, métadonnées [M] | #524 | GB5.1 (#502), GB5.3 (#504), GB5.2 (#503), GB5.5 (#506) |
| GB-C8 — Le corpus : recherche, stats, doc [M] | #525 | GB5.7 (#508), GB5.4 (#505), GB5.6 (#507) |
| GB-C9 — UX hors échelle [M] | #526 | GB6.1 (#509), GB6.2 (#510), GB6.4 (#512), GB6.3 (#511), GB6.5 (#513) |
| GB-C10 — Maintenance et parité à pleine échelle [M] | #527 | GB7.1 (#514), GB7.2 (#515), GB7.3 (#516) |
| GB-C11 — Le parcours d'étude : du quiz aux cartes [M] | #528 | GB9.4, GB9.2, GB9.1, GB9.3, GB9.5, GB9.6, GB9.7 |
| GB-C12 — Club et serveur [M] | #529 | GB9.8, GB9.9, GB9.10 |
| GB-C13 — UX en situation : plateau, mise en page, thème [M] | #530 | 7 étapes |
| GB-C14 — UX en situation : libellés, vocabulaire, états vides, gestes [M] | #531 | 8 étapes |

```
C1 ─┬─ C2 ─ C3 ─────────────┐
    └─ C4 (étapes 1-2 immédiates) ─┤
       C5 ─────────────────────────┼─ C6 (un seul bump) ─ C7 ─ C8 ─ C10
       C9, C13, C14 (indépendants) ┘            └─ C11 ─ C12
```
