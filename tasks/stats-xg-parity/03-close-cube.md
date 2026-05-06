# 03 — Close cube classification

**Goal:** Persister un drapeau `is_close_cube` pour distinguer les décisions cube *proches* (XG / gnuBG les comptent) des décisions cube triviales (No Double évidents que XG ignore). Aucune formule modifiée à cette phase.

**Depends on:** 01 (et idéalement 02 pour grouper les bumps de schéma).

**Does NOT touch:** `db_stats.go` formules. Pas de modif UI.

## Context

- Référence : `gnubg/eval.c:5088-5100` :
  ```c
  const float rThr = 0.16f;
  float rDouble = MIN(arDouble[OUTPUT_TAKE], 1.0f);
  if (arDouble[OUTPUT_OPTIMAL] - rDouble < rThr) return 1;  // close
  ```
  → une décision est « close » si l'écart entre l'équité optimale (cube ou no-cube selon ce qui est meilleur) et l'équité prise (capée à +1.0) est **< 0.16**.
- Cas particuliers :
  - **Take / Pass** : XG les compte toujours (seuil non appliqué). `is_close_cube = 1` systématiquement quand `cube_action ∈ {"Take","Pass"}`.
  - **No Double** : c'est sur celles-ci que la majorité des décisions sont rejetées. Sur Aachen, blunderDB compte 42+78=120 No Double, XG 16+12=28 → ratio ~ 4×.
- Champs disponibles dans `model.go:198 DoublingCubeAnalysis` : `CubefulNoDoubleEquity`, `CubefulDoubleTakeEquity`, `CubefulDoublePassEquity`, `BestCubeAction`, `WrongPassPercentage`, `WrongTakePercentage`. Mapping exact à valider en relisant `gnubg/eval.c` autour de la ligne 5088 pour les indices `OUTPUT_OPTIMAL` / `OUTPUT_TAKE` / `OUTPUT_DROP`.

## Files touched

- `model.go` — bump `DatabaseVersion` (suite à fiche 02, ex. 2.1.0 → 2.2.0 ou regrouper en un seul bump).
- `db.go` — colonne `analysis.is_close_cube` + index + migration.
- `db_analysis.go` — calcul à l'import.
- `migration_test.go` — couverture du backfill.
- `stats_parity_test.go` — nouvelle assertion `cube_close_count`.

## Tasks

### 1. Décrypter le mapping gnuBG → blunderDB

- [ ] Ouvrir `gnubg/eval.c:5088` dans le contexte (lire ±50 lignes autour) pour fixer définitivement :
  - `arDouble[OUTPUT_OPTIMAL]` = max(no-double-cubeful, double-take-cubeful, double-pass-cubeful) ?
  - `arDouble[OUTPUT_TAKE]` = équité après double + take depuis la perspective du joueur cube ?
- [ ] Documenter dans la fiche le mapping final, par ex. :
  ```
  rOptimal = max(CubefulNoDoubleEquity, CubefulDoubleTakeEquity, CubefulDoublePassEquity)
  rTakeCapped = min(CubefulDoubleTakeEquity, 1.0)
  isClose = (rOptimal - rTakeCapped) < 0.16
  ```
  (À valider en lisant gnuBG, pas à présumer.)
- [ ] Définir le comportement quand `BestCubeAction == "Take"` ou `"Pass"` : `is_close_cube = 1` directement, sans appliquer le seuil.
- [ ] Quand `DoublingCubeAnalysis == nil` (analyse incomplète) : `is_close_cube = 0` par défaut, à logger en debug.

### 2. Schéma + migration

- [ ] Ajouter colonne :
  ```sql
  is_close_cube INTEGER NOT NULL DEFAULT 0
  ```
- [ ] Index partiel :
  ```sql
  CREATE INDEX IF NOT EXISTS idx_analysis_is_close_cube ON analysis(is_close_cube) WHERE is_close_cube = 1
  ```
- [ ] Étape de migration : décompresser `data`, parser, appliquer la formule de la tâche 1, UPDATE.

### 3. Population à l'import

- [ ] Dans `populateAnalysisColumns` (ou équivalent), ajouter le calcul après celui de `is_forced`.
- [ ] Réutiliser un helper `func computeIsCloseCube(pa *PositionAnalysis, decisionType int) int` (testable unitairement avec une grille d'inputs).

### 4. Tests unitaires de la formule

- [ ] Dans un `db_analysis_test.go` (ou nouveau fichier), créer 5-6 cas :
  | Cas | NoDouble eq | Take eq | Pass eq | BestAction | isClose attendu |
  |---|---|---|---|---|---|
  | Trivial pas de cube | 0.20 | 0.50 | 1.00 | NoDouble | 0 |
  | Limite take/no-double | 0.30 | 0.32 | 1.00 | Double | 1 |
  | Take mandaté | -0.10 | 0.05 | 1.00 | DoubleTake | 1 |
  | Beaver/etc. | … | … | … | Take | 1 (toujours) |
  | Analyse manquante | nil | — | — | — | 0 |

### 5. Étendre le harnais

- [ ] Nouvelle ligne `cube_close_count` dans le tableau diff. Tolérance ±5 initialement (les seuils peuvent diverger légèrement entre XG et gnuBG).
- [ ] Vérifier sur Aachen que `count(is_close_cube=1, decision_type=1, P1) ≈ 16+2 = 18` et `P2 ≈ 12+5 = 17` — c'est-à-dire `xg.double_decisions + xg.take_decisions + xg.pass_decisions`.

## Acceptance criteria

- [ ] `go test ./...` vert.
- [ ] Migration v2.1.0 → v2.2.0 testée.
- [ ] Sur les 3 fixtures, `count(is_close_cube=1, decision_type=1, player)` ≈ XG total cube decisions à ±5 (avant resserrage en fiche 07).
- [ ] Tests unitaires de `computeIsCloseCube` couvrent les 5 cas listés.

## Risks

- **Mauvais mapping `OUTPUT_OPTIMAL`.** Si on confond cubeful no-double vs cubeless, le seuil 0.16 sera appliqué sur la mauvaise valeur. Mitigation : la tâche 1 demande de relire le source gnuBG, pas de deviner ; commit séparé du test unitaire pour pouvoir revenir dessus si le mapping est mauvais.
- **Sensibilité du seuil 0.16.** XG peut utiliser un seuil légèrement différent (jamais documenté publiquement). Mitigation : si l'écart sur la métrique « double decisions » reste > ±2 après tous les fixes, ajuster le seuil dans une plage [0.14, 0.20] et choisir empiriquement la meilleure valeur sur les 3 fixtures. Documenter en `notes` du JSON de référence et dans la fiche 07.
- **Take/Pass non analysés.** Avant le fix `ce025484`, les positions Take/Pass n'avaient pas d'analyse → `DoublingCubeAnalysis = nil`. Vérifier que le fix a bien rempli ces analyses pour les imports récents. Sinon, ajouter une heuristique : `cube_action ∈ {"Take","Pass"} ⇒ is_close_cube = 1` même sans analyse.
