# 02 — Conversion EMG → MWC loss

**Goal:** Ajouter la fonction `ConvertEMGLossToMWCLoss` dans `db_met.go` qui convertit une perte d'équité EMG en perte de MWC, en réutilisant les primitives existantes (`gnuBGGetME`, MET Kazaross-XG2). Intégrer la MWC dans `StatsResult` pour exposer les paires PR/MWC côté backend.

**Depends on:** 01

**Impact:** Ajoute la métrique MWC au panneau sans aucune modification de schéma ni migration.

## Context

- `db_met.go:367` contient déjà la conversion **forward** `convertGnuBGCubeMWCToEMG(analysis, score0, score1, fMove, cubeValue, matchLength)` qui applique la formule NEMG :
  ```
  EMG = (2·MWC − (mwcWin + mwcLose)) / (mwcWin − mwcLose)
  ```
- Pour une **perte** d'équité (différence best − played), l'inverse est linéaire et s'applique identiquement aux équités cubeless et cubeful :
  ```
  ΔMWC = ΔEMG · (mwcWin − mwcLose) / 2
  ```
- Ne nécessite **pas** de calcul de MWC absolu (celui-ci étant non publié par gnuBG pour le cubeful) — cf. plan §« Calcul MWC ».
- Colonnes `position` utilisées : `score_1`, `score_2`, `match_length`, `cube_value`, `player_on_roll` (toujours 0 après normalisation — le fMove réel est `move.player`), `has_jacoby`.

## Tasks

### 1. Implémenter la conversion

- [ ] Ajouter à `db_met.go` :
  ```go
  // ConvertEMGLossToMWCLoss convertit une perte d'équité EMG (millipoints ×1000)
  // en perte de MWC (fraction, ex. 0.015 = 1.5 points de % de MWC).
  // Retourne NaN si money-game (matchLength <= 0) ou cube dégénéré.
  //
  // Inverse linéaire de convertGnuBGCubeMWCToEMG. Applicable aux cube ET checker
  // errors car la transformation NEMG est un simple changement d'unité.
  func ConvertEMGLossToMWCLoss(emgMillipoints, score0, score1, fMove, cubeValue, matchLength int) float64 {
      if matchLength <= 0 {
          return math.NaN()
      }
      mwcWin := float32(gnuBGGetME(score0, score1, matchLength, fMove, cubeValue, fMove, false))
      mwcLose := float32(gnuBGGetME(score0, score1, matchLength, fMove, cubeValue, 1-fMove, false))
      denom := mwcWin - mwcLose
      if denom < 1e-7 && denom > -1e-7 {
          return math.NaN()
      }
      return (float64(emgMillipoints) / 1000.0) * float64(denom) / 2.0
  }
  ```
- [ ] Exportée (majuscule) car consommée depuis `db_stats.go` et potentiellement utile ailleurs.

### 2. Tests unitaires

- [ ] Créer `db_met_mwc_test.go` (ou étendre `db_met_test.go` s'il existe).
- [ ] **Test round-trip** : pour une set de scores variés (DMP 1pt, Crawford, 3-away 5-away, 7-pt, 11-pt, 21-pt) et plusieurs valeurs de cube (1, 2, 4) :
  - Générer une MWC loss arbitraire ΔMWC_in.
  - La convertir en ΔEMG avec la formule forward (`2·ΔMWC / (mwcWin − mwcLose)`).
  - Appliquer `ConvertEMGLossToMWCLoss` sur ce ΔEMG.
  - Vérifier égalité à ΔMWC_in à 1e-5 près (précision float32).
- [ ] **Test money-game** : `matchLength = 0` → retour `NaN` (utiliser `math.IsNaN`).
- [ ] **Test cube dégénéré** : DMP (1-away/1-away, cube=1) → `mwcWin=1.0, mwcLose=0.0`, `denom=1.0`, ΔMWC = ΔEMG × 0.5. Valeur testée explicitement.
- [ ] **Test Crawford** : score 4-away / Crawford (1pt trailer), cube=1. Vérifier que la valeur retournée reste plausible (non NaN, signe cohérent).
- [ ] **Test plausibilité vs XG** (optionnel, si une fixture de référence est disponible) : comparer à une valeur XG connue pour une position de référence.

### 3. Étendre `StatsResult`

- [ ] Ajouter les champs MWC dans `StatsResult` (fiche 01) :
  ```go
  type StatsResult struct {
      // … champs existants de la fiche 01 …
      MWCGlobal            float64
      MWCChecker, MWCCube  float64
      MWCRolling           map[int]float64
  }
  type TournamentStats struct { …, MWC float64 }
  type MatchStats      struct { …, MWC float64 }
  type CubeActionStats struct { …, MWC float64 }
  type BlunderEntry    struct { …, MWCLoss float64 }
  ```
- [ ] La MWC **agrégée** d'un tournoi/match/breakdown est la **somme** des MWC loss par décision (cumul), **pas** `(mwcWin − mwcLose) / 2 × PR_moyen` — ce serait faux car `(mwcWin − mwcLose)` varie par position.

### 4. Adapter `ComputeStats`

- [ ] Les requêtes 3–7 de la fiche 01 doivent désormais ramener aussi les colonnes nécessaires à la conversion MWC par ligne : `p.score_1, p.score_2, mv.player as fMove, p.cube_value, p.match_length`.
- [ ] Boucle Go post-SQL (approche 1 du plan §Architecture backend) :
  - Pour chaque ligne agrégée au niveau tournoi/match/cube_action, **ne pas** agréger MWC côté SQL ; à la place, streamer les positions individuelles avec leur `err` et leur contexte, et sommer en Go avec `ConvertEMGLossToMWCLoss(err, score0, score1, fMove, cubeValue, matchLength)`.
  - Un seul passage Go par groupe. Complexité : O(n_décisions).
- [ ] Alternative perf (si benchmark le justifie) : changer ces requêtes en `SELECT per-row (err, score_1, score_2, mv.player, cube_value, match_length, tournament_id, match_id, best_cube_action)` sans GROUP BY, puis agréger en Go (PR + MWC simultanément). **Recommandé** car simplifie le code et ne dégrade pas la perf sur les bases usuelles.
- [ ] Positions money-game (`match_length = 0`) : leur contribution MWC est `NaN` ; en Go, les exclure de la somme MWC mais pas de la somme PR. Flagger chaque agrégat avec un bool `MWCAvailable bool` (true si au moins une position du groupe n'est pas money-game).

### 5. Tests d'intégration

- [ ] Étendre `db_stats_test.go` de la fiche 01 :
  - Fixture avec au moins 1 tournoi en match play + 1 match en money-game.
  - Vérifier que `result.MWCGlobal > 0` (tournoi match-play contribue) et que `MWCAvailable` est true.
  - Fixture 100% money-game : `MWCAvailable = false`, `MWCGlobal` non significatif (NaN ou 0 selon la convention).

## Acceptance criteria

- [ ] `go test -run TestConvertEMGLossToMWCLoss ./...` vert, incluant le test round-trip sur ≥ 5 scores différents.
- [ ] `go test -run TestComputeStats ./...` vert (tests MWC inclus).
- [ ] Pas de modif de schéma, pas de migration.
- [ ] `go vet ./...` clean.
- [ ] Aucune utilisation de CGO ou de nouvelle dépendance.

## Rollback

Revert = `git revert`. La fonction `ConvertEMGLossToMWCLoss` est pure, les champs MWC de `StatsResult` seront ignorés si absents. Aucune donnée persistée.
