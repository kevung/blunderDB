# Fiche 11 — Tests unitaires sur les fonctions pures

Branche : `test/fonctions-pures`

## Objectif

Couvrir les gisements « valeur/effort maximal » : fonctions pures sans
dépendance, aujourd'hui à 0 % ou testées seulement au travers de tests SQL
lents.

## Tâches

- [ ] **`domain/position_match.go` (~700 lignes de prédicats purs, 0 % en
      direct)** : tables de cas pour `MatchesMirrorPosition`,
      `MatchesScorePosition`, `MatchesDiceRoll(Mode)`, les paires symétriques
      P1/P2 (`CheckerOff`, `BackChecker`, `CheckerInZone`, `OutfieldBlot`,
      `JanBlot` — le copié-collé-inversé est là où vivent les bugs de
      symétrie), `MatchesNoContact`, `Mirror`, `NormalizeForStorage`,
      `ComputePipCounts`, `PipCountDifference`. Compléter par
      `SearchOrderByClause`/`MatchOrderByClause` et
      `EffectiveIncludeFilter`/`ContainsAnyCheckerOf` (domain.go).
- [ ] **`config.go` racine** : round-trip complet avec `XDG_CONFIG_HOME` sur
      `t.TempDir()` — Load/Save, dimensions, langue, couleurs, uiScale,
      positions de panneaux, tourSeen, bearoffTsPath, epcChallenge,
      statsFilter, `sanitizePanelPosition`.
- [ ] **`internal/gui` helpers purs** : `IsDirectory`, `PathExists`,
      `CollectImportableFiles`, `ReadFileContent`, `DeleteFile`,
      `BearoffStatus` avec `t.TempDir()` ; `identity.go` hors dialogue
      (`GetIssuerIdentity`, `SetIssuerName`, export/import/regenerate avec
      XDG temporaire).
- [ ] **Frontend utilitaires purs** : `utils/dragReorder.js` (152 l.) et
      `utils/focusTrap.js` (34 l.) ; `stores/viewStore.js` (orchestration
      snapshot/restore). Au passage vérifier la forme de la position par
      défaut de `viewStore.js:8-21` (tableau de nombres vs objets
      `{checkers, color}` attendus par Board — bug latent signalé par
      l'audit) ; si bug confirmé, fabrique `emptyPosition()` unique exportée
      par `positionStore` et corrective.
- [ ] **Fixtures BGF cube** : câbler `testdata/bgf_positions/02_NDT_FR.txt`,
      `04_DP_EN.txt`, `06_RT_FR.txt` (aujourd'hui orphelines) sur des tests
      d'ingestion qui exécutent `classifyBGFCubeAction` et
      `bgfADoubleResponse` (0 % sur tout le dépôt).
- [ ] **Faux tests** : `tests/analysis_merge_test.go` (4 fonctions sans
      assertion, `t.Log` uniquement) — convertir en vrais tests (dédup
      Zobrist, tri par équité) ou supprimer.
- [ ] **Fixtures orphelines** : supprimer `testdata/xgid_corpus.json` si
      réellement sans référence (vérifier `xgid_contract_test.go` d'abord —
      mémoire projet dit qu'il garde la dérive !), `testdata/
      Tournoi_videau_Passy_2026/` ; ignorer/supprimer le dossier untracked
      « Match Report… BMAB_files ».

## Critères de fin

- `go test ./...` vert ; couverture de `domain` et de la racine nettement
  accrue (chiffre avant/après dans le commit).
- Les 3 fixtures BGF cube sont exécutées par la suite.

## Risques & garde-fous

- **`xgid_corpus.json`** : la mémoire projet indique qu'il verrouille le
  contrat de parsing XGID double (Go/GUI) — vérifier les références réelles
  avant toute suppression ; dans le doute, le garder.
- Tests de caractérisation : décrire le comportement ACTUEL ; toute
  divergence découverte (ex. symétrie P1/P2) se signale en commentaire et se
  corrige dans un commit séparé, pas silencieusement.
