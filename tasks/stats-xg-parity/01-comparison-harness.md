# 01 — Comparison harness

**Goal:** Construire un test Go `TestStatsParity` qui charge chaque JSON de référence (fiche 00), importe le `.xg` et le `.sgf` du match dans deux DB en mémoire, calcule les stats blunderDB, et émet un tableau diff aligné `XG | gnuBG-ref | blunderDB-from-XG | blunderDB-from-SGF` pour chaque métrique. Il `t.Errorf` quand l'écart sort des tolérances courantes.

**Depends on:** 00.

**Does NOT touch:** schéma, formules. Lecture seule côté blunderDB.

## Context

- Les tolérances évoluent fiche par fiche. Avant la fiche 02, on accepte ±10 décisions / ±0.5 PR / ±2 pp MWC. La fiche 07 les resserre à ±2 / ±0.1 / ±1 pp.
- Le test `xg_stats_reference_test.go` actuel ne couvre qu'Aachen et compare blunderDB à des baselines blunderDB (pas à XG). Il sera étendu / remplacé par `TestStatsParity` puis retiré (fiche 07).
- Helpers existants : `newTestDB(t)` (test helper déjà utilisé), `db.ImportXGMatch(file)`, `db.ImportGNUBGMatch(file)` (vérifier le nom exact dans `db_import_gnubg.go` ou wrapper équivalent), `db.GetMatchDetailStats(matchID)`.

## Files touched

- `stats_parity_test.go` (créé, à la racine, package `main`)
- `testdata/stats_reference/*.json` (lus seuls)

## Tasks

### 1. Définir les types de chargement

- [ ] Dans `stats_parity_test.go`, structs miroirs du JSON (fiche 00) :
  ```go
  type referenceMatch struct {
      MatchFile   string                       `json:"match_file"`
      MatchLength int                          `json:"match_length"`
      Players     [2]string                    `json:"players"`
      XG          map[string]referencePlayer   `json:"xg"`
      GNUBG       map[string]referencePlayer   `json:"gnubg"`
      Notes       string                       `json:"notes"`
  }
  type referencePlayer struct {
      PR                     *float64 `json:"pr"`
      SnowieErrorRate        *float64 `json:"snowie_error_rate"`
      TotalDecisions         *int     `json:"total_decisions"`
      CheckerUnforced        *int     `json:"checker_unforced"`
      DoubleDecisions        *int     `json:"double_decisions"`
      TakeDecisions          *int     `json:"take_decisions"`
      PassDecisions          *int     `json:"pass_decisions"`
      TotalEquityErrorEMG    *float64 `json:"total_equity_error_emg"`
      TotalMWCLossPct        *float64 `json:"total_mwc_loss_pct"`
      // … cf. SCHEMA.md
  }
  ```
- [ ] Loader : `loadReference(t, path string) referenceMatch`.

### 2. Définir la grille de tolérances

- [ ] Constantes paramétrables (top du fichier) :
  ```go
  type tolerances struct {
      Decisions int
      PR        float64
      MWC       float64 // points de pourcentage
      Equity    float64 // EMG
      SnowieER  float64
  }
  // valeurs initiales (avant fiche 02)
  var tolPhase01 = tolerances{Decisions: 10, PR: 0.5, MWC: 2.0, Equity: 0.1, SnowieER: 0.5}
  ```
- [ ] Le test final (fiche 07) basculera sur `tolPhaseFinal = {2, 0.1, 1.0, 0.05, 0.1}`.

### 3. Fonction utilitaire de diff

- [ ] `diffRow(t, label, want *float64, got float64, tol float64)` :
  - skip si `want == nil` (champ absent dans la référence)
  - log la ligne dans tous les cas (`t.Logf`)
  - `t.Errorf` si `|got − *want| > tol`
- [ ] Variante `diffRowInt` pour les compteurs.

### 4. Subtest par fixture

- [ ] Boucler sur les 3 JSON :
  ```go
  fixtures := []string{
      "testdata/stats_reference/aachen-double-7pt.json",
      "testdata/stats_reference/test.json",
      "testdata/stats_reference/charlot1-charlot2.json",
  }
  for _, f := range fixtures { t.Run(filepath.Base(f), func(t *testing.T) { … }) }
  ```
- [ ] Chaque sous-test :
  1. `loadReference`.
  2. Importer le `.xg` dans une DB neuve → `statsXG := GetMatchDetailStats`.
  3. Importer le `.sgf` (s'il existe) dans une autre DB neuve → `statsSGF := GetMatchDetailStats`.
  4. Pour chaque joueur (mapping joueurs depuis `Players`) :
     - Émettre la ligne `t.Logf("%-25s | XG=%v | gnuBG-ref=%v | bDB-XG=%v | bDB-SGF=%v", …)` pour chaque métrique.
     - `diffRow` blunderDB-XG vs ref XG.
     - `diffRow` blunderDB-SGF vs ref gnuBG.

### 5. Tableau récapitulatif final

- [ ] À la fin du test, dumper un tableau global au format Markdown via `t.Logf` (lisible dans `go test -v`). C'est cette sortie que l'utilisateur lira pour décider quand passer à la fiche suivante.

### 6. Garde-fou : Snowie ER croisé

- [ ] Tant que blunderDB ne calcule pas Snowie ER (avant fiche 05), comparer la valeur reconstruite manuellement (`total_equity_error / total_moves_les_deux`) à `gnubg.snowie_error_rate` pour vérifier que la formule est bien comprise. Logger en T.Logf, pas d'erreur.

## Acceptance criteria

- [ ] `go test -run TestStatsParity ./...` passe avec les tolérances `tolPhase01`.
- [ ] Tous les `t.Logf` du diff sont lisibles en `-v` ; structure claire.
- [ ] Le test couvre les 3 fixtures (skip propre si `.sgf` manquant).
- [ ] Le fichier reste ≤ 250 lignes hors helpers.

## Risks

- **Mapping joueurs.** L'ordre player1/player2 peut différer entre le JSON ref et l'objet `MatchDetailStats`. Mitigation : le JSON nomme explicitement les joueurs ; lire `match.player1_name`/`player2_name` après import et construire le mapping ; échouer explicitement si un nom manque.
- **No-Double sur-comptés.** Avant la fiche 03, blunderDB sur-compte les `double_decisions` d'un facteur 2-6×. Le test va `t.Errorf` ; c'est attendu — il faut faire passer la baseline en élargissant temporairement la tolérance `Decisions` à 100 sur la métrique « double_decisions », ou skip cette ligne. **Décider : tolérance par métrique** (la struct `tolerances` doit contenir un champ par métrique, pas une seule valeur globale).
- **Ordre d'import.** `ImportXGMatch` et l'équivalent gnuBG peuvent fixer un ID match différent ; passer cet ID au `GetMatchDetailStats`.
