# 06 — MWC cross-check

**Goal:** Re-valider le calcul de MWC loss après le resserrage du dénominateur PR (fiche 04) et l'ajout de Snowie ER (fiche 05). Si l'écart résiduel avec XG dépasse 1 pp sur les 3 fixtures, investiguer la cause et corriger.

**Depends on:** 04 + 05.

**Does NOT touch:** PR (fixé en 04). Idéalement aucun changement requis si l'écart MWC résiduel s'absorbe avec le filtre `statsCountedExpr`.

## Context

- État actuel de MWC (`db_met.go:435-447`) : conversion EMG→MWC post-hoc par décision, en s'appuyant sur le score de la position et le cube value au moment de la décision.
- État actuel de la précision : ~ 1 pp d'écart vs XG sur Aachen (P1 : -19.85 vs -20.83 ; P2 : -14.39 vs -15.48). L'écart pourrait venir :
  - des No Double sur-comptés (résolu par fiche 03) ;
  - des coups forcés à err=0 qui contribuent 0 mais inflatent le compteur (résolu par fiche 04, sans effet sur MWC car la somme est inchangée) ;
  - d'un décalage de score (l'erreur d'un coup est encourue *avant* de le jouer ; le score utilisé pour la conversion doit être celui *avant* le coup) ;
  - d'une convention de signe sur le « player favori » (gnuBG `eq2mwc` est pondéré par la perspective du joueur).
- Référence gnuBG : `gnubg/analysis.c:1449-1464` calcule `rCost = eq2mwc(rSkill, &ci) - eq2mwc(0.0f, &ci)` *par décision* puis somme — pas de conversion post-hoc.

## Files touched

- `db_met.go` — refonte éventuelle de `ConvertEMGLossToMWCLoss` ou ajout d'une voie alternative.
- `db_analysis.go` — si nécessaire, matérialiser `mwc_error` à l'import (nouvelle colonne).
- `db_stats.go` — adapter les requêtes MWC pour SUM-er la colonne directement plutôt que reconvertir en Go.
- Migration éventuelle (nouveau bump `DatabaseVersion` si on ajoute une colonne).

## Tasks

### 1. Mesure post-fix

- [x] Lancer `go test -run TestStatsParity -v ./...` et noter les écarts MWC sur les 3 fixtures, par joueur, après les fiches 04+05.
- [x] **Si l'écart est ≤ 1 pp partout :** la fiche se conclut par un test resserré (`MWC: 1.0`) et un mini-changelog. Stop ici.
- [x] **Si l'écart > 1 pp :** continuer en tâche 2.

> **Résultats pré-fix :** Aachen P1=0.98 pp, Aachen P2=1.09 pp. Le P2 dépasse 1 pp → passage en tâche 2.

### 2. Diagnostic

- [x] Ajouter un mode debug à `ConvertEMGLossToMWCLoss` (drapeau `verbose`) qui logge par décision :
- [x] Identifier la divergence : **bug cube_error pour le doubleur**.
  - Pour `Double/Pass`, `populateAnalysisColumns` utilisait `CubefulDoublePassError = DP − best`.
    Si le doubleur joue correctement (DT est le meilleur coup) et l'adversaire passe mal, le doubleur
    se voit imputer l'erreur du passer. Ex : DT=best, DP=1.0, DP−best > 0 → faux positif.
  - **Fix :** Pour `Double`, `Double/Take`, `Double/Pass`, utiliser `min(CubefulDoubleTakeError, CubefulDoublePassError)`.
    Le doubleur est responsable de sa décision de doubler ; l'adversaire répondra toujours de façon optimale
    (take si DT<DP, pass sinon). Cette formule correspond exactement à `rSkill` dans `gnubg/analysis.c:480−490`.

### 3. Si la cause est un score décalé

- [x] *Non applicable* (la cause était un bug de formule, pas un décalage de score).

### 4. Si la cause est plus structurelle (par-décision MWC requise)

- [x] *Non applicable* (correction dans la formule de cube_error suffit, pas besoin de colonne `mwc_error`).

### 5. Tests

- [x] Resserrer `tolerances.MWC` à `1.0` pp (objectif fiche 07 : 0.5 si possible).
- [x] Sur les 3 fixtures, MWC blunderDB ≈ MWC XG à 1 pp.

## Acceptance criteria

- [x] `go test -run TestStatsParity ./...` passe avec `MWC ≤ 1.0 pp`.
- [x] Aucune colonne `mwc_error` ajoutée (correction dans la formule suffit).
- [x] Aucune régression PR / Snowie ER (les fiches précédentes restent vertes).

## Risks

- **Nouveau bump de schéma.** Si on matérialise `mwc_error`, c'est un troisième bump après 02+03. Assumable, mais grouper si possible avec 02/03 si on ré-exécute les phases.
- **Sensibilité au crawford / post-crawford.** `gnuBGGetME` doit recevoir le bon flag crawford. Vérifier dans `populateAnalysisColumns` le passage de `pos.HasJacoby`/contexte crawford. À documenter.
- **MWC = `eq2mwc(skill) − eq2mwc(0)`.** Le code blunderDB fait `(emg/1000) × (mwcWin − mwcLose) / 2` qui suppose une linéarité approximative. Pour les positions « compressées » (proches de 0% ou 100% MWC), la linéarité casse. Si l'écart résiduel se concentre sur ces positions, c'est la signature du problème ; remplacer par la formule par-décision.
