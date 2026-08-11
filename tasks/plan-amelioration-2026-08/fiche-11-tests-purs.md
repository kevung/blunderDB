# Fiche 11 — Tests unitaires sur les fonctions pures

Branche : `test/fonctions-pures`

## Objectif

Couvrir les gisements « valeur/effort maximal » : fonctions pures sans
dépendance, aujourd'hui à 0 % ou testées seulement au travers de tests SQL
lents.

## Tâches

- [x] **`domain/position_match.go` (~700 lignes de prédicats purs, 0 % en
      direct)** : tables de cas pour `MatchesMirrorPosition`,
      `MatchesScorePosition`, `MatchesDiceRoll(Mode)`, les paires symétriques
      P1/P2 (`CheckerOff`, `BackChecker`, `CheckerInZone`, `OutfieldBlot`,
      `JanBlot` — le copié-collé-inversé est là où vivent les bugs de
      symétrie), `MatchesNoContact`, `Mirror`, `NormalizeForStorage`,
      `ComputePipCounts`, `PipCountDifference`. Compléter par
      `SearchOrderByClause`/`MatchOrderByClause` et
      `EffectiveIncludeFilter`/`ContainsAnyCheckerOf` (domain.go).
- [x] **`config.go` racine** : round-trip complet avec `XDG_CONFIG_HOME` sur
      `t.TempDir()` — Load/Save, dimensions, langue, couleurs, uiScale,
      positions de panneaux, tourSeen, bearoffTsPath, epcChallenge,
      statsFilter, `sanitizePanelPosition`.
- [x] **`internal/gui` helpers purs** : `IsDirectory`, `PathExists`,
      `CollectImportableFiles`, `ReadFileContent`, `DeleteFile`,
      `BearoffStatus` avec `t.TempDir()` ; `identity.go` hors dialogue
      (`GetIssuerIdentity`, `SetIssuerName`, export/import/regenerate avec
      XDG temporaire).
- [x] **Frontend utilitaires purs** : `utils/dragReorder.js` (152 l.) et
      `utils/focusTrap.js` (34 l.) ; `stores/viewStore.js` (orchestration
      snapshot/restore). Au passage vérifier la forme de la position par
      défaut de `viewStore.js:8-21` (tableau de nombres vs objets
      `{checkers, color}` attendus par Board — bug latent signalé par
      l'audit) ; si bug confirmé, fabrique `emptyPosition()` unique exportée
      par `positionStore` et corrective.
- [x] **Fixtures BGF cube** : câbler `testdata/bgf_positions/02_NDT_FR.txt`,
      `04_DP_EN.txt`, `06_RT_FR.txt` (aujourd'hui orphelines) sur des tests
      d'ingestion qui exécutent `classifyBGFCubeAction` et
      `bgfADoubleResponse` (0 % sur tout le dépôt).
- [x] **Faux tests** : `tests/analysis_merge_test.go` (4 fonctions sans
      assertion, `t.Log` uniquement) — convertir en vrais tests (dédup
      Zobrist, tri par équité) ou supprimer.
- [x] **Fixtures orphelines** : supprimer `testdata/xgid_corpus.json` si
      réellement sans référence (vérifier `xgid_contract_test.go` d'abord —
      mémoire projet dit qu'il garde la dérive !), `testdata/
      Tournoi_videau_Passy_2026/` ; ignorer/supprimer le dossier untracked
      « Match Report… BMAB_files ».

## Critères de fin

- [x] `go test ./...` vert ; couverture de `domain` et de la racine nettement
  accrue (chiffre avant/après dans le commit).
- [x] Les 3 fixtures BGF cube sont exécutées par la suite.

## Risques & garde-fous

- **`xgid_corpus.json`** : la mémoire projet indique qu'il verrouille le
  contrat de parsing XGID double (Go/GUI) — vérifier les références réelles
  avant toute suppression ; dans le doute, le garder.
- Tests de caractérisation : décrire le comportement ACTUEL ; toute
  divergence découverte (ex. symétrie P1/P2) se signale en commentaire et se
  corrige dans un commit séparé, pas silencieusement.

## Notes d'exécution

### Couverture avant/après (`go test -cover`)

| Paquet                        | Avant | Après |
| ------------------------------ | ----- | ----- |
| `pkg/blunderdb/domain`        | 33.6% | 68.3% |
| `.` (racine, `main`)          | 22.5% | 47.4% |
| `internal/gui`                | 24.3% | 45.2% |
| `pkg/blunderdb/ingest`        | 58.4% | 63.0% |

### Bugs réels trouvés et corrigés (commits séparés, documentés)

1. **`config.go` `LoadConfig` n'exposait pas `BearoffTsPath`/`EpcChallenge`
   au récepteur lié à Wails.** Chaque champ relu depuis le disque était
   recopié sur `c` (l'instance que le frontend appelle via les méthodes
   `Get*`), sauf ces deux-là — trouvé en écrivant le round-trip Load/Save.
   `GetBearoffTsPath()`/`GetEpcChallenge()` retournaient donc toujours la
   valeur zéro après un redémarrage, alors que la valeur était bien
   persistée et même correctement appliquée au moteur bearoff (mais via un
   autre chemin, la variable locale `config` de `main.go`, jamais via le
   récepteur). Corrigé dans un commit séparé
   (`fix(config): propager BearoffTsPath et EpcChallenge au chargement`),
   avec `TestLoadConfigPropagatesBearoffAndEpcChallenge` en non-régression.
2. **`stores/viewStore.js:8-21` avait sa propre position par défaut au
   mauvais format** — un tableau de 26 nombres signés (notation "-2 0 0 5
   ...") au lieu du tableau d'objets `{checkers, color}` que Board.svelte et
   `positionStore.js` attendent partout ailleurs. Confirmé comme un vrai bug
   atteignable en usage normal : ce défaut alimente le repli de
   `deserialize()` quand une position sauvegardée en session ne se retrouve
   plus côté base (positions supprimées entre deux lancements via
   `sessionService.deserialize()`) — pas seulement la toute première vue au
   démarrage. `App.svelte` avait une copie inline de la MÊME position vide,
   au bon format cette fois (~l.172-183). Corrigé dans un commit séparé
   (`fix(stores): unifier la position vide autour de emptyPosition()`) : une
   fabrique `emptyPosition()` unique, exportée par `positionStore.js`,
   remplace les deux endroits — et alloue un objet `Point` frais par case
   plutôt que `Array(26).fill({...})` (qui partageait la même référence sur
   les 26 cases ; inoffensif tant que rien ne mute un point en place, mais
   un piège pour la prochaine fois). `viewStore.test.js` verrouille le
   format sur la vue initiale et sur le repli de `deserialize()`, en rouge
   avant le correctif (vérifié : `git stash` du correctif → 3 tests
   échouent avec le message exact attendu → `git stash pop`).

### Asymétries P1/P2 examinées — aucune trouvée

Les cinq paires ciblées par la fiche (`CheckerOff`, `BackChecker`,
`CheckerInZone`, `OutfieldBlot`, `JanBlot`) ont chacune été testées avec des
cas miroirs identiques (mêmes checkers, position reflétée par `Mirror()` :
point `i` → `25-i`, couleur inversée). Dans les cinq cas, la plage de points
P2 est l'exacte image miroir de la plage P1 — pas de copié-collé-inversé.

La seule asymétrie relevée n'est pas un prédicat mal reflété : il n'existe
pas de `MatchesPlayer2AbsolutePipCount` (ni de
`SearchFilters.Player2AbsolutePipCountFilter`) en face de
`MatchesPlayer1AbsolutePipCount` — une lacune d'API pré-existante et
délibérée (le différentiel de pip count, lui symétrique, est déjà couvert
par le filtre `p`/`PipCountDifference`), pas une correction à faire ici.

### Faux tests `tests/analysis_merge_test.go`

Les 4 fonctions (`TestMergePlayedMoves`, `TestMergeCheckerAnalysis`,
`TestPositionDeduplication`, `TestEquitySorting`) documentaient un
comportement avec `t.Log` seul, aucune assertion. La logique réelle
(`mergeCheckerMoves`, `mergePlayedMoves`, `sortCheckerMovesByEquity`,
`mergeAnalysis`) est non exportée dans `pkg/blunderdb/ingest` : le paquet
externe `tests` ne peut structurellement pas l'atteindre. Fichier supprimé ;
remplacé par `pkg/blunderdb/ingest/merge_test.go` (mêmes package, donc
accès direct aux fonctions non exportées) avec de vraies assertions : dédup
par chaîne de coup avec conservation de l'analyse la plus profonde en cas de
conflit, tri par équité décroissante avec recalcul de `EquityError`, union
normalisée des coups/actions joués (`engine.NormalizeMove`), et les deux
chemins d'insertion/mise à jour de `mergeAnalysis` (promotion des champs
historiques `PlayedMove`/`PlayedCubeAction`, fusion multi-moteur
d'`AllCubeAnalyses` avec XG prioritaire, conservation de `CreationDate`). Le
volet dédup-par-hash-Zobrist que l'un des 4 faux tests décrivait est couvert
ailleurs (`domain/position_match_test.go` `TestNormalizeForStorage`,
`engine/zobrist_test.go`), donc pas dupliqué dans `merge_test.go`.

### Fixtures orphelines

- `testdata/xgid_corpus.json` : **gardé** — `xgid_contract_test.go` le
  charge pour verrouiller le contrat de parsing XGID double (Go
  `DecodeXGID` / frontend), conformément à la mémoire projet.
- `testdata/Tournoi_videau_Passy_2026/` : **supprimé** (720 Ko, 0 référence
  dans tout le dépôt — vérifié par recherche texte sur `.go`, `.js`, `.rst`,
  `.sh`).
- Dossier untracked « Match Report… BMAB_files » et le `.mat` untracked
  signalés dans le contexte de mission : **absents de ce worktree**
  (`blunderDB-fiche11`) — ils n'existaient que dans le checkout principal,
  jamais touché. Ajouté `testdata/Match Report*` au `.gitignore` par
  précaution, pour que ce cas ne revienne pas untracked si quelqu'un
  reproduit la manipulation ici.

### Fixtures BGF cube (02_NDT_FR, 04_DP_EN, 06_RT_FR)

Câblées via `ingest.MapBGFTextPosition`, en asserttant les valeurs d'équité
EMG exactes lues dans le tableau "Videau:"/"Cube Action:" de chaque
fixture — une mauvaise classification par `classifyBGFCubeAction` laisserait
le champ correspondant (`CubefulNoDoubleEquity`/`DoubleTakeEquity`/
`DoublePassEquity`) à sa valeur zéro plutôt que la vraie valeur du fixture,
donc ces tests prouvent la classification (FR "Pas de double"/"Prendre"/
"Refuser" et EN "No Double"/"Take"/"Pass"), pas seulement l'absence d'erreur
de parsing. `bgfADoubleResponse` (aucune fixture `.bgf` de match ne contient
d'`adouble` explicite — BGBlitz encode le plus souvent un double comme
`amove` synthétique) est testée séparément avec le `[]interface{}` minimal
qu'elle parcourt.

### Commits de cette fiche

1. `fix(config): propager BearoffTsPath et EpcChallenge au chargement`
2. `test(domain): tables de cas pour les prédicats purs de position_match.go`
3. `test(config): round-trip complet Load/Save avec XDG_CONFIG_HOME isolé`
4. `test(gui): couverture des helpers purs et de l'identité d'émission`
5. `test(ingest): câbler les fixtures BGF orphelines et bgfADoubleResponse`
6. `test(ingest): remplacer les 4 faux tests de tests/analysis_merge_test.go`
7. `chore(testdata): supprimer le fixture orphelin Tournoi_videau_Passy_2026`
8. `test(frontend): couvrir dragReorder.js et focusTrap.js`
9. `fix(stores): unifier la position vide autour de emptyPosition()`
10. cette mise à jour de la fiche
